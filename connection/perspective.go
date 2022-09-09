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
		recv: recv,
		send: send,
	}
}

type (
	perspectiveResolver struct {
		counter atomic.Uint32
		recv    recvFuncDef
		send    sendFuncDef
	}

	resolveFallback = func() bool
)

// Resolve attempts to resolve our Perspective, which can be either:
//   - Server, when we're the ones accepting the connection
//   - Client, when we're dialing the other peer
//
// It achieves it's goal via a single round trip during which we're sending
// and receiving a message called perspective resolvent.
// Perspective resolvent contains the following details:
//   - the perspective for our connection (are we the Client or the Server?)
//   - counter, which helps us keep track of the order in which we're handling
//     client/server connections. It can be either 1 or 2.
//
// The resolvent carries a single byte which holds on positions
// 3,4 Perspective and 1,2 counter. Here are some examples:
//   - 00001001 --> { Perspective: Client, Counter: 1 }
//   - 00000110 --> { Perspective: Server, Counter: 2 }
//
// If the counter from the received resolvent is equal to the counter value
// we've sent ourselves
// OR
// the counters differed but the fallback returns true
// AND
// the counter is equal to 1 (first connection takes priority),
// the connection is successfully resolved.
// In any other case an error describing the reason for failed resolution is returned.
func (p *perspectiveResolver) Resolve(
	ctx context.Context,
	perspective Perspective,
	conn quic.Connection,
	fallback resolveFallback,
) error {
	resolved, first, err := p.resolve(ctx, perspective, conn)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve connection perspective for %s", perspective)
	}
	if !resolved && !fallback() {
		return errors.Wrap(connErrNotResolved, "perspective was not resolved and fallback did not apply")
	}
	if !first {
		return errors.Wrap(connErrNotResolved, "perspective was not the first one to have been resolved")
	}
	return nil
}

// resolve receives the peer's resolvent asynchronously and sends the host's resolvent.
// It returns resolved == true If both counters were equal and
// first == true If the counter is equal to 0.
func (p *perspectiveResolver) resolve(
	ctx context.Context,
	perspective Perspective,
	conn quic.Connection,
) (resolved, first bool, err error) {
	ctr := p.counter.Add(1)
	first = ctr == 1

	rcv := make(chan message)
	go func() {
		data, err := p.recv(ctx, conn, 0)
		rcv <- message{data: data, err: err}
	}()

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

// Reset should be called when we're closing the QUIC connection assigned to the Connection (to the peer).
// When we mark the Connection with StatusDead, we'll resume dialing and accepting new QUIC
// connections for this peer which requires perspectiveResolver's counter to be reset to 0.
func (p *perspectiveResolver) Reset() {
	p.counter.Store(0)
}

// perspectiveResolvent is the message sent during perspective resolving round trip.
// It holds Perspective and counter.
type perspectiveResolvent struct {
	Perspective
	ctr uint32 // 1 or 2
}

// Encode places the Perspective at the 3rd and 4th and counter at the 1st and 2nd bit position.
func (p *perspectiveResolvent) Encode() (b []byte) {
	return []byte{byte(uint32(p.Perspective)<<2 | p.ctr)}
}

// Decode decodes the resolvent reading Perspective and counter from the first 4 bits.
func (p *perspectiveResolvent) Decode(b []byte) error {
	if len(b) != 1 {
		return errors.Errorf("invalid perspectiveResolvent received,"+
			" expected exactly 1 byte, got %d with %v content", len(b), b)
	}
	// Right bit shift to get the 3rd and 4th bits.
	p.Perspective = Perspective(b[0] >> 2)
	// Apply 0000011 bitmask to get the 1st and 2nd bits.
	p.ctr = uint32(b[0] & 3)
	return nil
}
