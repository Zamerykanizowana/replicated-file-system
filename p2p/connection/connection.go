package connection

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func New() *Connection {
	return &Connection{
		nc:     nil,
		status: StatusDead,
		mu:     new(sync.Mutex),
	}
}

type (
	Connection struct {
		nc     net.Conn
		status Status
		mu     *sync.Mutex
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
)

func (c *Connection) Establish(conn net.Conn, self string) (peer string, err error) {
	closeIfAlive := func() (closed bool) {
		if c.status == StatusAlive {
			log.Debug().
				Stringer("remote_addr", conn.RemoteAddr()).
				Msg("closing duplicated connection")
			Close(conn)
			closed = true
		}
		return
	}

	// Fast path, no need to use mutex if we're already alive.
	if closeIfAlive() {
		return "", ErrAlreadyConnected
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Slow path.
	if closeIfAlive() {
		return "", ErrAlreadyConnected
	}

	c.nc = conn
	peer, err = c.handshake(self)
	if err != nil {
		c.Close()
		return
	}
	c.status = StatusAlive

	log.Info().Str("peer", peer).Msg("connected with peer")
	return
}

func (c *Connection) Recv() ([]byte, error) {
	data, err := c.recv()
	return data, c.handleError(err)
}

func (c *Connection) recv() ([]byte, error) {
	var size int64
	if err := binary.Read(c.nc, binary.BigEndian, &size); err != nil {
		return nil, errors.Wrap(err, "failed to read size header")
	}
	if size < 0 {
		return nil, errors.New("invalid message size, might be too long")
	}

	buf := make([]byte, size)
	if _, err := io.ReadFull(c.nc, buf); err != nil {
		return nil, errors.Wrap(err, "failed to read protobuf.Response")
	}
	return buf, nil
}

func (c *Connection) Send(data []byte) error {
	return c.handleError(c.send(data))
}

func (c *Connection) send(data []byte) error {
	// Serialize the length header.
	lb := make([]byte, 8)
	binary.BigEndian.PutUint64(lb, uint64(len(data)))

	// Attach the length header along with body.
	buff := net.Buffers{lb, data}

	if _, err := buff.WriteTo(c.nc); err != nil {
		return errors.Wrap(err, "failed to send protobuf.Request")
	}
	return nil
}

func (c *Connection) Status() Status {
	return c.status
}

func (c *Connection) Close() {
	c.mu.Lock()
	Close(c.nc)
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

func Close(conn net.Conn) {
	if conn == nil {
		log.Warn().Msg("connection was already closed")
		return
	}
	if err := conn.Close(); err != nil {
		if err == syscall.EINVAL {
			log.Warn().
				Stringer("remote_addr", conn.RemoteAddr()).
				Msg("connection was already closed")
		}
		log.Err(err).
			Stringer("remote_addr", conn.RemoteAddr()).
			Msg("failed to closed connection")
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

func (c *Connection) handshake(self string) (peer string, err error) {
	h := &protobuf.Handshake{PeerName: self}
	data, err := proto.Marshal(h)
	if err != nil {
		return "", errors.Wrap(err, "failed to proto marshal handshake")
	}
	if err = c.Send(data); err != nil {
		return "", errors.Wrap(err, "failed to send handshake")
	}

	data, err = c.Recv()
	if err != nil {
		return "", errors.Wrap(err, "failed to read handshake header")
	}
	if err = proto.Unmarshal(data, h); err != nil {
		return "", errors.Errorf("invalid handshake received: %v", data)
	}

	if _, whitelisted := whitelist[h.PeerName]; !whitelisted {
		return "", errors.Errorf("remote peer: %s is not whitelisted", h.PeerName)
	}
	return h.PeerName, nil
}
