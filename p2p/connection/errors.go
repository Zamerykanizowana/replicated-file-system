package connection

import (
	"strings"
	"sync"

	"github.com/lucas-clemente/quic-go"
)

type SendMultiErr struct {
	errs map[peerName]error
	mu   sync.Mutex
}

func (s *SendMultiErr) Error() string {
	b := strings.Builder{}
	b.WriteString("sending errors: ")
	for peerAddr, err := range s.errs {
		b.WriteString(peerAddr)
		b.WriteString(" > ")
		b.WriteString(err.Error())
		b.WriteString(", ")
	}
	return b.String()[:b.Len()-2]
}

func (s *SendMultiErr) Append(pname string, err error) {
	s.mu.Lock()
	if s.errs == nil {
		s.errs = make(map[peerName]error)
	}
	s.errs[pname] = err
	s.mu.Unlock()
}

func (s *SendMultiErr) Empty() bool {
	return len(s.errs) == 0
}

// Iota starts with an arbitrary number, so we won't get in the way of builtin errors.
const (
	StreamErrTimeout quic.StreamErrorCode = iota + 99
	StreamErrInvalidSizeHeader
	StreamErrCancelled
	StreamErrRead
	StreamErrRecv
)

const (
	ConnErrGeneric quic.ApplicationErrorCode = iota + 199
)
