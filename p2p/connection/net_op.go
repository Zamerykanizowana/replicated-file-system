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
		stream.CancelRead(quic.StreamErrorCode(streamErrReadHeader))
		return nil, errors.Wrap(streamErrReadHeader, err.Error())
	}
	if size < 0 {
		stream.CancelRead(quic.StreamErrorCode(streamErrInvalidSizeHeader))
		return nil, streamErrInvalidSizeHeader
	}

	ctx, cancel := contextWithOptionalTimeout(ctx, timeout)
	defer cancel()

	buf := make([]byte, size)
	done := make(chan struct{})
	errCh := make(chan error)

	go func() {
		if _, err = io.ReadFull(stream, buf); err != nil {
			errCh <- errors.Wrap(streamErrReadBody, err.Error())
			return
		}
		done <- struct{}{}
	}()

	if err = selectResult(ctx, errCh, done); err != nil {
		if sErr, ok := errors.Cause(err).(streamErr); ok {
			stream.CancelRead(quic.StreamErrorCode(sErr))
		}
		return nil, err
	}

	return buf, nil
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
			errCh <- errors.Wrap(streamErrWrite, err.Error())
			return
		}
		done <- struct{}{}
	}()

	if err := selectResult(ctx, errCh, done); err != nil {
		if sErr, ok := errors.Cause(err).(streamErr); ok && stream != nil {
			stream.CancelWrite(quic.StreamErrorCode(sErr))
		}
		return err
	}

	return nil
}

// selectResult selects from context, error and done channels,
// if any error was detected it returns the error.
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

// streamContextDone handles scenarios when stream context was either:
// - cancelled
// - deadline was exceeded
// It maps these two errors to related streamErr.
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

// closeConn closes the QUIC connection with the error which is converted to
// quic.ApplicationErrorCode and passed to the other side of the connection.
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
