package connection

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
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
		log:         log.With().Object("peer", peer).Logger(),
		timeout:     timeout,
		sink:        sink,
	}
}

type (
	// Connection is used to encapsulate all required logic and parameters
	// for a single net.Conn uniquely associated with a single peer.
	Connection struct {
		peer       *config.Peer
		conn       net.Conn
		status     Status
		mu         *sync.Mutex
		openNotify chan struct{}
		// closeNotify should only be called when we would like to reestablish the connection.
		// Right now there's no such case, but If it's ever the case we should make sure the
		// goroutine responsible for dialing is shutdown too.
		closeNotify chan struct{}
		log         zerolog.Logger
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

var (
	ErrClosed           = errors.New("connection is closed")
	ErrAlreadyConnected = errors.New("connection was already established")
	ErrTimedOut         = errors.New("operation timed out")
)

// Establish attempts to set StatusAlive for Connection and assign net.Conn to it.
// It also releases WaitForOpen if no errors were generated.
// If the Connection is already alive it will return an error.
func (c *Connection) Establish(conn net.Conn) error {
	// Fast path, no need to use mutex if we're already alive.
	if c.status == StatusAlive {
		return ErrAlreadyConnected
	}

	// Slow path.
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == StatusAlive {
		return ErrAlreadyConnected
	}

	c.conn = conn
	c.status = StatusAlive
	c.openNotify <- struct{}{}

	c.log.Info().Msg("connection established")
	return nil
}

// Listen runs receiver loop, waiting for new messages.
// If the Connection.Status is StatusDead it will block until WaitForOpen returns.
// The received data along with any errors is wrapped by message struct and sent
// to the sink channel.
func (c *Connection) Listen() {
	for {
		if c.status == StatusDead {
			c.WaitForOpen()
		}
		data, err := c.Recv()
		if err != nil {
			c.sink <- message{err: err}
			continue
		}
		c.sink <- message{data: data}
	}
}

// Recv is not structured like Send is due to the fact that
// we're only able to measure timeout on the lowest level, just
// after receiving size header we know the peer is sending the message.
// If we'd try to do the timeout here, we'd time out on waiting
// for file descriptor to wake up. This would result in timeouts
// for simply not receiving any traffic from the peer.
func (c *Connection) Recv() (data []byte, err error) {
	if c.status == StatusDead {
		return nil, ErrClosed
	}
	data, err = recv(c.conn, c.timeout)
	return data, c.handleError(err)
}

// Send sends data with timeout.
func (c *Connection) Send(data []byte) (err error) {
	if c.status == StatusDead {
		return ErrClosed
	}
	done := make(chan struct{}, 1)
	go func() {
		err = send(c.conn, data)
		done <- struct{}{}
	}()

	select {
	case <-time.After(c.timeout):
		return errors.Wrap(ErrTimedOut, "send timed out")
	case <-done:
		return c.handleError(err)
	}
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
func (c *Connection) Close() {
	c.mu.Lock()
	closeConn(c.conn)
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

// handleError discerns temporary errors from permanent and closes the Connection for the latter.
func (c *Connection) handleError(err error) error {
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
		switch v := cause.(type) {
		case *net.OpError:
			if v.Temporary() == false {
				closed = true
			}
		}
	}
	if closed {
		c.Close()
		c.closeNotify <- struct{}{}
		return errors.Wrap(err, ErrClosed.Error())
	}
	return err
}
