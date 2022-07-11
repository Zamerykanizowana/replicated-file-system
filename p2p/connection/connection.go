package connection

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func NewConnection(peerName, addr string, sink chan<- message) *Connection {
	return &Connection{
		peerName:   peerName,
		addr:       addr,
		conn:       nil,
		status:     StatusDead,
		mu:         new(sync.Mutex),
		openNotify: make(chan struct{}, 1),
		log: log.With().Dict("peer", zerolog.Dict().
			Str("name", peerName).
			Str("address", addr),
		).Logger(),
		// TODO make it configurable.
		timeout: 1 * time.Minute,
		sink:    sink,
	}
}

type (
	Connection struct {
		peerName   string
		addr       string
		conn       net.Conn
		status     Status
		mu         *sync.Mutex
		openNotify chan struct{}
		log        zerolog.Logger
		timeout    time.Duration
		sink       chan<- message
	}
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

func (c *Connection) Recv() (data []byte, err error) {
	if c.status == StatusDead {
		return nil, ErrClosed
	}
	done := make(chan struct{}, 1)
	go func() {
		data, err = recv(c.conn)
		done <- struct{}{}
	}()

	select {
	case <-time.After(c.timeout):
		return nil, errors.Wrap(ErrTimedOut, "recv timed out")
	case <-done:
		return data, c.handleError(err)
	}
}

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

func (c *Connection) WaitForOpen() {
	<-c.openNotify
	return
}

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

func (c *Connection) handleError(err error) error {
	cause := errors.Cause(err)
	switch cause {
	case nil:
		return nil
	case io.EOF:
		c.Close()
		return ErrClosed
	default:
		switch v := cause.(type) {
		case *net.OpError:
			if v.Temporary() == false {
				c.Close()
				return ErrClosed
			}
		default:
			return err
		}
	}
	return err
}
