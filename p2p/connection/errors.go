package connection

import (
	"strings"
	"sync"

	"github.com/lucas-clemente/quic-go"
)

// BroadcastMultiErr aggregates errors from Broadcast.
type BroadcastMultiErr struct {
	errs map[peerName]error
	mu   sync.Mutex
}

func (e *BroadcastMultiErr) Errors() map[peerName]error {
	return e.errs
}

func (e *BroadcastMultiErr) Error() string {
	b := strings.Builder{}
	b.WriteString("sending errors: ")
	for peerAddr, err := range e.errs {
		b.WriteString(peerAddr)
		b.WriteString(" > ")
		b.WriteString(err.Error())
		b.WriteString(", ")
	}
	return b.String()[:b.Len()-2]
}

func (e *BroadcastMultiErr) Append(peer string, err error) {
	e.mu.Lock()
	if e.errs == nil {
		e.errs = make(map[peerName]error)
	}
	e.errs[peer] = err
	e.mu.Unlock()
}

func (e *BroadcastMultiErr) Empty() bool {
	return len(e.errs) == 0
}

// streamErr represents common quic.Stream errors with a meaningful message.
type streamErr quic.StreamErrorCode

const (
	streamErrTimeout streamErr = iota
	streamErrCancelled
	streamErrInvalidSizeHeader
	streamErrReadHeader
	streamErrReadBody
	streamErrWrite
)

func (e streamErr) Error() string {
	switch e {
	case streamErrTimeout:
		return "timed out"
	case streamErrCancelled:
		return "context was cancelled"
	case streamErrInvalidSizeHeader:
		return "invalid header size"
	case streamErrReadHeader:
		return "failed to read size header"
	case streamErrReadBody:
		return "failed to read body"
	case streamErrWrite:
		return "failed to write data"
	default:
		return "unknown stream error"
	}
}

// connErr represents common quic.Connection errors with a meaningful message.
type connErr quic.ApplicationErrorCode

const (
	connErrUnspecified connErr = iota
	connErrAlreadyEstablished
	connErrClosed
	connErrCancelled
	connErrNotResolved
)

func (e connErr) Error() string {
	switch e {
	case connErrAlreadyEstablished:
		return "connection was already established"
	case connErrClosed:
		return "connection is closed"
	case connErrCancelled:
		return "context was cancelled"
	case connErrNotResolved:
		return "connection was not resolved and both peers have agreed to close it"
	default:
		return "unspecified connection error"
	}
}
