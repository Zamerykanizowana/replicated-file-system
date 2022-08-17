package connection

import (
	"context"
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
	host *config.Peer,
	peer *config.Peer,
	timeout time.Duration,
	sink chan<- message,
) *Connection {
	return &Connection{
		host:                host,
		peer:                peer,
		conn:                nil,
		status:              StatusDead,
		perspective:         Unknown,
		perspectiveResolver: newPerspectiveResolver(),
		mu:                  new(sync.Mutex),
		openNotify:          make(chan struct{}),
		closeNotify:         make(chan struct{}),
		netOpTimeout:        timeout,
		sink:                sink,
	}
}

type (
	// Connection is used to encapsulate all required logic and parameters
	// for a single net.Conn uniquely associated with a single peer.
	Connection struct {
		host                *config.Peer
		peer                *config.Peer
		conn                quic.Connection
		status              Status
		perspective         Perspective
		perspectiveResolver *perspectiveResolver
		mu                  *sync.Mutex
		openNotify          chan struct{}
		// closeNotify should only be called when we would like to reestablish the connection.
		// Right now there's no such case, but If it's ever the case we should make sure the
		// goroutine responsible for dialing is shutdown too.
		closeNotify  chan struct{}
		netOpTimeout time.Duration
		sink         chan<- message
	}
	// Status informs about the connection state, If the net.Conn is established and running
	// it will hold StatusAlive, otherwise StatusDead.
	Status uint8
)

const (
	StatusDead Status = iota
	StatusAlive
)

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

// Establish attempts to set StatusAlive for Connection and assign net.Conn to it.
// It also releases WaitForOpen if no errors were generated.
// If the Connection is already alive it will return connErrAlreadyEstablished.
// Unlike standard TCP implementation, If we end up "establishing" two connections through
// dialing and listening, we should not close the latter connection as QUIC be design operates on a single
// connection with multiple streams and the go-quic library complies having just a single connection
// manager per peer address.
func (c *Connection) Establish(
	ctx context.Context,
	perspective Perspective,
	conn quic.Connection,
) error {
	// Fast path, no need to use mutex if we're already alive.
	if c.status == StatusAlive && c.perspective != Unknown {
		return connErrAlreadyEstablished
	}

	// Naming takes precedence, we may want to change this logic to a pseudo Lamport's clock of sorts.
	fallback := func() (accept bool) {
		// Complement logic has to be applied on both sides,
		// otherwise both peers would arrive at different conclusions.
		switch perspective {
		case Client:
			accept = c.host.Name < c.peer.Name
		case Server:
			accept = c.host.Name > c.peer.Name
		}
		return
	}
	if err := c.perspectiveResolver.Resolve(ctx, perspective, conn, fallback); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == StatusAlive {
		return connErrAlreadyEstablished
	}

	c.perspectiveResolver.Reset()
	c.conn = conn
	c.perspective = perspective
	c.status = StatusAlive
	c.openNotify <- struct{}{}

	log.Ctx(ctx).Info().Stringer("perspective", perspective).Msg("connection established")
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
	data, err = recv(ctx, c.conn, c.netOpTimeout)
	if err = c.handleErrors(ctx, err); err != nil {
		return nil, errors.Wrap(err, "failed to receive data")
	}
	return
}

// Send sends data with timeout.
func (c *Connection) Send(ctx context.Context, data []byte) (err error) {
	if c.status == StatusDead {
		return connErrClosed
	}
	ctx, cancel := contextWithOptionalTimeout(ctx, c.netOpTimeout)
	defer cancel()
	if err = c.handleErrors(ctx, send(ctx, c.conn, data)); err != nil {
		return errors.Wrap(err, "failed to send data")
	}
	return
}

// Status returns the connections' Status.
func (c *Connection) Status() Status {
	return c.status
}

// WaitForOpen blocks until the Connection Status changes to StatusAlive.
func (c *Connection) WaitForOpen() {
	<-c.openNotify
	return
}

// WaitForClosed blocks until the Connection Status changes from StatusAlive to StatusDead.
func (c *Connection) WaitForClosed() {
	<-c.closeNotify
	return
}

// Close closes the underlying net.Conn and sets Connection Status to StatusDead.
func (c *Connection) Close(err error) {
	c.mu.Lock()
	closeConn(c.conn, err)
	c.status = StatusDead
	c.mu.Unlock()
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
				log.Ctx(ctx).Debug().Err(err).Send()
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
