package connection

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

func NewConnection(
	host *config.Peer,
	peer *config.Peer,
	timeout time.Duration,
	sink chan<- Message,
	activeConnections *atomic.Int64,
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
		netOpTimeout:        timeout,
		sink:                sink,
		activeConnections:   activeConnections,
	}
}

type (
	// Connection is used to encapsulate all required logic and parameters
	// for a single net.Conn uniquely associated with a single peer.
	Connection struct {
		// host describes the p2p.Host we're running.
		host *config.Peer
		// peer describes who are we connecting with.
		peer *config.Peer
		// conn hols the underlying QUIC connection to the peer.
		conn quic.Connection
		// status describes the Status of the connection.
		status Status
		// perspective informs about the QUIC perspective we assumed either Client or Server.
		perspective Perspective
		// perspectiveResolver is used to resolve the perspective,
		// so that we can discern whether we're acting as a Client or Server.
		perspectiveResolver *perspectiveResolver
		// mu is a mutex used to block operations which change the Connection state.
		mu *sync.Mutex
		// openNotify is a channel onto which Connection publishes
		// when it changes it's status to StatusAlive.
		openNotify chan struct{}
		// netOpTimeout is the timeout on all network operations, both send and recv.
		netOpTimeout time.Duration
		// sink is used to forward received messages further down the pipeline.
		sink chan<- Message
		// activeConnections should be incremented when Connection is StatusAlive and decremented
		// when it goes to StatusDead.
		activeConnections *atomic.Int64
	}
	// Status informs about the connection state, If the net.Conn is established and running
	// it will hold StatusAlive, otherwise StatusDead.
	// StatusShutdown is only set during shutdown, to prevent accepting new connections.
	Status uint8
)

const (
	StatusDead Status = iota
	StatusAlive
	StatusShutdown
)

func (c Status) String() string {
	switch c {
	case StatusAlive:
		return "alive"
	case StatusDead:
		return "dead"
	case StatusShutdown:
		return "shutdown"
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
	log.Debug().EmbedObject(c).Msg("handling new connection request")

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

	c.conn = conn
	c.perspective = perspective
	c.status = StatusAlive
	c.activeConnections.Add(1)
	go func() { c.openNotify <- struct{}{} }()

	go c.watchConnection(conn)

	log.Info().EmbedObject(c).Msg("connection established")
	return nil
}

// watchConnection watches the quic.Connection and block until the context.Context associated with
// this connection reruns. It then resets the Connection and notifies the routines blocked at WaitForClosed.
func (c *Connection) watchConnection(conn quic.Connection) {
	ctx := conn.Context()
	<-ctx.Done()
	if err := c.Close(ctx.Err()); err != nil {
		log.Err(err).Send()
	}
}

// Listen runs receiver loop, waiting for new messages.
// If the Connection.Status is StatusDead it will block until WaitForOpen returns.
// The received data along with any errors is wrapped by Message struct and sent
// to the sink channel.
func (c *Connection) Listen(ctx context.Context) {
	for {
		if c.status == StatusDead {
			if err := c.WaitForOpen(ctx); err != nil {
				return
			}
		}
		data, err := c.Recv(ctx)
		if err != nil {
			c.sink <- Message{Err: err}
			continue
		}
		c.sink <- Message{Data: data}
	}
}

// Recv receives data with timeout.
func (c *Connection) Recv(ctx context.Context) (data []byte, err error) {
	if c.status == StatusDead {
		return nil, connErrClosed
	}
	data, err = recv(ctx, c.conn, c.netOpTimeout)
	if err = c.handleErrors(err); err != nil {
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
	if err = c.handleErrors(send(ctx, c.conn, data)); err != nil {
		return errors.Wrap(err, "failed to send data")
	}
	return
}

// Status returns the connections' Status.
func (c *Connection) Status() Status {
	return c.status
}

// WaitForOpen blocks until the Connection Status changes to StatusAlive.
func (c *Connection) WaitForOpen(ctx context.Context) error {
	select {
	case <-c.openNotify:
		return nil
	case <-ctx.Done():
		return errPoolClosed
	}
}

// Close closes the underlying net.Conn and sets Connection Status to StatusDead.
func (c *Connection) Close(err error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Either watchConnection or handleErrors gets here first.
	if c.status == StatusDead {
		return nil
	}

	c.status = StatusDead

	log.Err(err).EmbedObject(c).Msg("connection to the peer was lost")

	// It should only be called when the remote connection was not closed yet, so it won't
	// attempt to close a connection that is already dead, this will result in blocking here
	// potentially forever...
	if c.conn.Context().Err() == nil {
		err = closeConn(c.conn, err)
	}

	c.perspective = Unknown
	c.perspectiveResolver.Reset()
	c.activeConnections.Add(-1)
	return err
}

func (c *Connection) MarshalZerologObject(e *zerolog.Event) {
	e.Object("peer", c.peer).
		Stringer("status", c.status).
		Stringer("perspective", c.perspective)
}

// handleErrors discerns temporary errors from permanent and closes the Connection for the latter.
func (c *Connection) handleErrors(err error) error {
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
	case context.Canceled:
		closed = true
	default:
		switch e := cause.(type) {
		case *net.OpError:
			if e.Temporary() == false {
				closed = true
			}
		case *quic.StreamError:
			err = streamErr(e.ErrorCode)
		case *quic.ApplicationError:
			err = connErr(e.ErrorCode)
			switch err {
			case connErrAlreadyEstablished:
				return nil
			case connErrClosed:
				closed = true
				break
			}
		}
	}
	if closed {
		if closeErr := c.Close(err); closeErr != nil {
			log.Err(closeErr).Send()
		}
		return errors.Wrap(err, connErrClosed.Error())
	}
	return err
}
