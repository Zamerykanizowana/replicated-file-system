package connection

import (
	"context"
	"sync/atomic"

	"github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

type Perspective byte

const (
	Unknown Perspective = iota
	Server
	Client
)

func (p Perspective) String() string {
	switch p {
	case Server:
		return "listener"
	case Client:
		return "client"
	default:
		return "unknown"
	}
}

func newPerspectiveResolver() *perspectiveResolver {
	return &perspectiveResolver{
		counter: new(atomic.Uint32),
		recv:    recv,
		send:    send,
	}
}

type (
	perspectiveResolver struct {
		counter *atomic.Uint32
		recv    recvFuncDef
		send    sendFuncDef
	}

	resolveFallback = func() bool
)

func (p *perspectiveResolver) Resolve(
	ctx context.Context,
	perspective Perspective,
	conn quic.Connection,
	fallback resolveFallback,
) error {
	resolved, first, err := p.resolve(ctx, perspective, conn)
	switch {
	case err != nil:
		return errors.Wrapf(err, "failed to resolve connection perspective for %s", perspective)
	case !resolved && !fallback():
		return errors.Wrap(connErrResolved, "perspective was not resolved and fallback did not apply")
	case resolved && !first:
		return errors.Wrap(connErrResolved, "perspective was not the first one to have been resolved")
	}
	return nil
}

func (p *perspectiveResolver) resolve(
	ctx context.Context,
	perspective Perspective,
	conn quic.Connection,
) (resolved, first bool, err error) {
	rcv := make(chan message)
	go func() {
		data, err := p.recv(ctx, conn)
		rcv <- message{data: data, err: err}
	}()

	ctr := p.counter.Add(1)
	first = ctr == 1

	resolvent := perspectiveResolvent{Perspective: perspective, ctr: ctr}
	if err = p.send(ctx, conn, resolvent.Encode()); err != nil {
		return
	}

	msg := <-rcv
	if err = msg.err; err != nil {
		return
	}
	if err = resolvent.Decode(msg.data); err != nil {
		return
	}
	if resolvent.Perspective == perspective {
		err = errors.New("BUG: host and peer perspectives are the same across a single connection")
		return
	}

	if ctr != resolvent.ctr {
		return
	}

	resolved = true
	return
}

func (p *perspectiveResolver) Reset() {
	p.counter.Store(0)
}

type perspectiveResolvent struct {
	Perspective
	ctr uint32 // 1 or 2
}

func (p perspectiveResolvent) Encode() (b []byte) {
	return []byte{byte(p.Perspective), byte(p.ctr)}
}

func (p *perspectiveResolvent) Decode(b []byte) error {
	if len(b) != 2 {
		return errors.Errorf("invalid perspectiveResolvent received,"+
			" expected 2 bytes, got %d with %v content", len(b), b)
	}
	p.Perspective = Perspective(b[0])
	p.ctr = uint32(b[1])
	return nil
}
