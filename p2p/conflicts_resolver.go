package p2p

import (
	"sync/atomic"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

var conflictingOperations = []protobuf.Request_Type{
	protobuf.Request_CREATE,
	protobuf.Request_LINK,
	protobuf.Request_MKDIR,
	protobuf.Request_RENAME,
	protobuf.Request_RMDIR,
	protobuf.Request_SETATTR,
	protobuf.Request_SYMLINK,
	protobuf.Request_UNLINK,
	protobuf.Request_WRITE,
}

func newConflictsResolver() *conflictsResolver {
	conflicts := make(map[protobuf.Request_Type]atomic.Uint64, len(conflictingOperations))
	for _, op := range conflictingOperations {
		conflicts[op] = atomic.Uint64{}
	}
	return &conflictsResolver{conflicts: conflicts}
}

type conflictsResolver struct {
	conflicts map[protobuf.Request_Type]atomic.Uint64
}

func (c *conflictsResolver) DetectAndResolveConflict(trans *transactions, req *protobuf.Request) bool {
	return false
}
