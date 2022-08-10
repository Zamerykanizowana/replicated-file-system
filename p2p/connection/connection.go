package connection

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

func NewConnection(
	peer *config.Peer,
	timeout time.Duration,
	sink chan<- message,
) *Connection {
	return &Connection{
		peer:        peer,
		conn:        nil,
		status:      StatusDead,
		mu:          new(sync.Mutex),
		openNotify:  make(chan struct{}, 1),
		closeNotify: make(chan struct{}, 1),
		timeout:     timeout,
		sink:        sink,
	}
}

type (
	// Connection is used to encapsulate all required logic and parameters
	// for a single net.Conn uniquely associated with a single peer.
	Connection struct {
		peer       *config.Peer
		conn       quic.Connection
		status     Status
		mu         *sync.Mutex
		openNotify chan struct{}
		// closeNotify should only be called when we would like to reestablish the connection.
		// Right now there's no such case, but If it's ever the case we should make sure the
		// goroutine responsible for dialing is shutdown too.
		closeNotify chan struct{}
		timeout     time.Duration
		sink        chan<- message
	}
	// Status informs about the connection state, If the net.Conn is established and running
	// it will hold StatusAlive, otherwise StatusDead.
	Status uint8
)

const (
	StatusDead Status = iota
	StatusAlive
)

// Establish attempts to set StatusAlive for Connection and assign net.Conn to it.
// It also releases WaitForOpen if no errors were generated.
// If the Connection is already alive it will return an error.
func (c *Connection) Establish(ctx context.Context, conn quic.Connection) error {
	// Fast path, no need to use mutex if we're already alive.
	if c.status == StatusAlive {
		return connErrAlreadyEstablished
	}

	// Slow path.
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == StatusAlive {
		return connErrAlreadyEstablished
	}

	c.conn = conn
	c.status = StatusAlive
	c.openNotify <- struct{}{}

	log.Ctx(ctx).Info().Msg("connection established")
	return nil
}

// Listen runs receiver loop, waiting for new messages.
// If the Connection.Status is StatusDead it will block until WaitForOpen returns.
// The received data along with any errors is wrapped by message struct and sent
// to the sink channel.
func (c *Connection) Listen(ctx context.Context) {
	for {
		if c.status == StatusDead {
			c.WaitForOpen()
		}
		data, err := c.Recv(ctx)
		if err != nil {

			c.sink <- message{err: err}
			continue
		}
		c.sink <- message{data: data}
	}
}

// Recv receives data with timeout.
func (c *Connection) Recv(ctx context.Context) (data []byte, err error) {
	if c.status == StatusDead {
		return nil, connErrClosed
	}
	data, err = c.recv(ctx)
	if err = c.handleErrors(ctx, err); err != nil {
		return nil, errors.Wrap(err, "failed to receive data")
	}
	return
}

// recv handles timeout and closes the quic.ReceiveStream with an appropriate quic.StreamErrorCode.
// It blocks until the peer opens a new unidirectional QUIC stream.
// It checks for the size header first before reading the data.
func (c *Connection) recv(ctx context.Context) ([]byte, error) {
	stream, err := c.conn.AcceptUniStream(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to accept unidirectional QUIC stream")
	}

	var size int64
	if err = binary.Read(stream, binary.BigEndian, &size); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to read size header")
		return nil, streamErrReadHeader
	}
	if size < 0 {
		log.Ctx(ctx).Error().
			Int64("size", size).
			Msg("invalid header size, might be too long")
		return nil, streamErrInvalidSizeHeader
	}

	buf := make([]byte, size)
	done := make(chan struct{})
	errCh := make(chan error)

	go func() {
		if _, err = io.ReadFull(stream, buf); err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to read body")
			errCh <- streamErrReadBody
			return
		}
		done <- struct{}{}
	}()

	return buf, c.selectResult(ctx, errCh, done)
}

// Send sends data with timeout.
func (c *Connection) Send(ctx context.Context, data []byte) (err error) {
	if c.status == StatusDead {
		return connErrClosed
	}
	if err = c.handleErrors(ctx, c.send(ctx, data)); err != nil {
		return errors.Wrap(err, "failed to send data")
	}
	return
}

// send handles timeout and closes the quic.SendStream with an appropriate quic.StreamErrorCode.
// It writes the size header before sending the data.
func (c *Connection) send(ctx context.Context, data []byte) error {
	var stream quic.SendStream
	defer func() {
		if stream != nil {
			// The only error we can get here is if the Close was called on a cancelled stream.
			// We don't really care, just want to make sure, the stream is closed.
			_ = stream.Close()
		}
	}()

	done := make(chan struct{})
	errCh := make(chan error)

	go func() {
		var err error
		stream, err = c.conn.OpenUniStreamSync(ctx)
		if err != nil {
			errCh <- errors.Wrap(err, "failed to open unidirectional QUIC stream")
			return
		}

		// Serialize the length header.
		lb := make([]byte, 8)
		binary.BigEndian.PutUint64(lb, uint64(len(data)))

		// Attach the length header along with body.
		buff := net.Buffers{lb, data}

		if _, err = buff.WriteTo(stream); err != nil {
			errCh <- errors.Wrap(err, "failed to send protobuf.Request")
			return
		}
		done <- struct{}{}
	}()

	return c.selectResult(ctx, errCh, done)
}

func (c *Connection) Status() Status {
	return c.status
}

// WaitForOpen blocks until the Connection.Status changes to StatusAlive.
func (c *Connection) WaitForOpen() {
	<-c.openNotify
	return
}

// WaitForClosed blocks until the Connection.Status changes from StatusAlive to StatusDead.
func (c *Connection) WaitForClosed() {
	<-c.closeNotify
	return
}

// Close closes the underlying net.Conn and sets Connection.Status to StatusDead.
func (c *Connection) Close(err error) {
	c.mu.Lock()
	closeConn(c.conn, err)
	c.status = StatusDead
	c.mu.Unlock()
}

func (c Status) String() string {
	switch c {
	case StatusAlive:
		return "alive"
	case StatusDead:
		return "dead"
	default:
		return "unspecified"
	}
}

// handleErrors discerns temporary errors from permanent and closes the Connection for the latter.
func (c *Connection) handleErrors(ctx context.Context, err error) error {
	cause := errors.Cause(err)
	// Unwrap if we can, this helps reveal net.OpError from
	// tls.permanentError (which is private for whatever reason...).
	if unw := errors.Unwrap(cause); unw != nil {
		cause = unw
	}
	var closed bool
	switch cause {
	case nil:
		return nil
	case io.EOF:
		closed = true
	default:
		switch e := cause.(type) {
		case *net.OpError:
			if e.Temporary() == false {
				closed = true
			}
		case *quic.StreamError:
			return streamErr(e.ErrorCode)
		case *quic.ApplicationError:
			err = connErr(e.ErrorCode)
			switch err {
			case connErrAlreadyEstablished:
				log.Ctx(ctx).Err(err).Send()
				return nil
			default:
				return err
			}
		}
	}
	if closed {
		c.Close(err)
		c.closeNotify <- struct{}{}
		return errors.Wrap(err, connErrClosed.Error())
	}
	return err
}

func (c *Connection) selectResult(
	ctx context.Context,
	errCh <-chan error,
	done <-chan struct{},
) error {
	select {
	case <-ctx.Done():
		return c.streamContextDone(ctx)
	case <-time.After(c.timeout):
		return streamErrTimeout
	case err := <-errCh:
		return err
	case <-done:
		return nil
	}
}

func (c *Connection) streamContextDone(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return streamErrCancelled
	case context.DeadlineExceeded:
		return streamErrTimeout
	default:
		log.Ctx(ctx).Error().Msg("context was neither subject to any deadline nor was it cancellable")
	}
	return nil
}
