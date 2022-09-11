package p2p

import (
	"sync/atomic"

	"github.com/pkg/errors"
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

// IncrementClock atomically increments the conflicts clock used for resolving conflicting transactions.
func (c *conflictsResolver) IncrementClock() {
	c.clock.Add(1)
}

var ErrTransactionConflict = errors.New("transaction conflict detected and resolved in favor of other peer")

// DetectAndResolveConflict goes through all existing transactions and checks if any of them
// are conflicting with the incoming message, it detects conflict if all these conditions are met:
//   - transaction initiator is not the same peer which sent the message. We don't want to
//     check conflicts for the other peers, they should've done it themselves.
//   - RelativePath and NewRelativePath are the same.
//
// It then proceeds to resolve the conflict based on these premises:
//
// IF
//
//	Incoming request's clock is smaller than existing transaction.
//	This prevents starvation problems, when one peer looses the conflicts resolution too often.
//	We always want to let the peer which had the most transaction conflicts
//	not resolved in his favor proceed with his request.
//
// OR
//
//	Clocks are equal but incoming request's peer name is smaller than
//	existing transaction initiator (peer name)
//	We only slightly favor some peers here, as once the clock was incremented
//	for the resolution winner the other peer will win next time when comparing clocks.
//
// THEN
//
//	Return ErrTransactionConflict
//
// It goes through all transactions unless it meets a conflict which is resolved in favor
// of existing transcation, then it returns the error and breaks the loop.
//
// If no conflicts are detected or detected conflicts are resolved in favor of the
// incoming request, then it returns no error.
func (c *conflictsResolver) DetectAndResolveConflict(
	trans *transactions,
	msg *protobuf.Message,
) error {
	req := msg.GetRequest()
	if req == nil {
		log.Panic().Msg("BUG: expected protobuf.Request in the message, fix the code!")
	}
	for _, t := range trans.ts {
		// If the transactions has been initialized through protobuf.Response, skip detection.
		if t.Request == nil {
			continue
		}
		// Let us not decide for the other peer. Don't check conflicts for our own transactions either.
		if t.InitiatorName == msg.PeerName {
			continue
		}
		// All conflicting operations. If NewRelativePath is empty it will still work correct.
		if conflict :=
			t.Request.Metadata.RelativePath == req.Metadata.RelativePath &&
				t.Request.Metadata.NewRelativePath == req.Metadata.NewRelativePath; !conflict {
			continue
		}
		// If the request clock is smaller than the transactions, return conflict.
		if req.Clock < t.Request.Clock {
			return ErrTransactionConflict
		}
		// If the clocks are equal, compare peer names as a fallback.
		if req.Clock == t.Request.Clock && msg.PeerName < t.InitiatorName {
			return ErrTransactionConflict
		}
	}
	return nil
}
