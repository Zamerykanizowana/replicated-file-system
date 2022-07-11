package connection

import (
	"encoding/binary"
	"io"
	"net"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func recv(conn net.Conn) ([]byte, error) {
	var size int64
	if err := binary.Read(conn, binary.BigEndian, &size); err != nil {
		return nil, errors.Wrap(err, "failed to read size header")
	}
	if size < 0 {
		return nil, errors.New("invalid message size, might be too long")
	}

	buf := make([]byte, size)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, errors.Wrap(err, "failed to read protobuf.Response")
	}
	return buf, nil
}

func send(conn net.Conn, data []byte) error {
	// Serialize the length header.
	lb := make([]byte, 8)
	binary.BigEndian.PutUint64(lb, uint64(len(data)))

	// Attach the length header along with body.
	buff := net.Buffers{lb, data}

	if _, err := buff.WriteTo(conn); err != nil {
		return errors.Wrap(err, "failed to send protobuf.Request")
	}
	return nil
}

func closeConn(conn net.Conn) {
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
