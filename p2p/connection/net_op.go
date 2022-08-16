package connection

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type (
	recvFuncDef = func(ctx context.Context, ac uniStreamAcceptor, timeout time.Duration) ([]byte, error)
	sendFuncDef = func(ctx context.Context, op uniStreamOpener, data []byte) error

	uniStreamAcceptor interface {
		AcceptUniStream(ctx context.Context) (quic.ReceiveStream, error)
	}

	uniStreamOpener interface {
		OpenUniStreamSync(ctx context.Context) (quic.SendStream, error)
	}
)

// recv handles timeout and closes the quic.ReceiveStream with an appropriate quic.StreamErrorCode.
// It blocks until the peer opens a new unidirectional QUIC stream.
// It checks for the size header first before reading the data.
func recv(ctx context.Context, ac uniStreamAcceptor, timeout time.Duration) ([]byte, error) {
	stream, err := ac.AcceptUniStream(ctx)
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

	ctx, cancel := contextWithOptionalTimeout(ctx, timeout)
	defer cancel()

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

	return buf, selectResult(ctx, errCh, done)
}

// send handles timeout and closes the quic.SendStream with an appropriate quic.StreamErrorCode.
// It writes the size header before sending the data.
func send(ctx context.Context, op uniStreamOpener, data []byte) error {
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
		stream, err = op.OpenUniStreamSync(ctx)
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

	return selectResult(ctx, errCh, done)
}

// TODO handle stream errors here somehow.
func selectResult(
	ctx context.Context,
	errCh <-chan error,
	done <-chan struct{},
) error {
	select {
	case <-ctx.Done():
		return streamContextDone(ctx)
	case err := <-errCh:
		return err
	case <-done:
		return nil
	}
}

func streamContextDone(ctx context.Context) error {
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

func closeConn(conn quic.Connection, err error) {
	if conn == nil {
		return
	}
	log.Debug().Err(err).Msg("closing connection")
	var cErr connErr
	if !errors.As(err, &cErr) {
		cErr = connErrUnspecified
	}
	if err = conn.CloseWithError(quic.ApplicationErrorCode(cErr), err.Error()); err != nil {
		log.Err(err).
			Stringer("remote_addr", conn.RemoteAddr()).
			Msg("failed to close connection")
	}
}
