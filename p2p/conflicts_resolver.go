package p2p

import (
	"sync/atomic"

	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func newConflictsResolver() *conflictsResolver {
	conflicts := make(map[protobuf.Request_Type]atomic.Uint64, len(protobuf.Request_Type_name))
	for op := range protobuf.Request_Type_name {
		conflicts[protobuf.Request_Type(op)] = atomic.Uint64{}
	}
	return &conflictsResolver{conflicts: conflicts}
}

type conflictsResolver struct {
	clock     atomic.Uint64
	conflicts map[protobuf.Request_Type]atomic.Uint64
}

func (c *conflictsResolver) IncrementClock() {
	c.clock.Add(1)
}

func (c *conflictsResolver) DetectAndResolveConflict(
	trans *Transactions,
	msg *protobuf.Message,
) (detected, greenLight bool) {
	req := msg.GetRequest()
	if req == nil {
		log.Panic().Msg("BUG: expected protobuf.Request in the message, fix the code!")
	}
	for _, t := range trans.ts {
		// If the Transactions has been initialized through protobuf.Response, skip detection.
		if t.Request == nil {
			continue
		}
		// Let us not decide for the other peer. Don't check conflicts for our own Transactions either.
		if t.InitiatorName == msg.PeerName {
			continue
		}
		// All conflicting operations
		if conflict := t.Request.Metadata.RelativePath == req.Metadata.RelativePath; !conflict {
			continue
		}
		detected = true
		// If the request clock is smaller than the Transactions, give a green light.
		// This prevents starvation problems, when one peer looses the conflicts resolution too often.
		if req.Clock < t.Request.Clock {
			return true, false
		}
		// If the clocks are equal, compare peer names as a fallback.
		// We only slightly favor some of the peers here, as once the clock was incremented
		// for the resolution winner the other peer will win next time when comparing clocks.
		if req.Clock == t.Request.Clock && msg.PeerName < t.InitiatorName {
			return true, false
		}
	}
	// At this point we would've already returned red light,
	// so we're ready to give the green one out.
	if detected {
		return true, true
	}
	return
}
