package rfs

import (
	"context"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
)

type rfsRoot struct {
	fs.LoopbackNode
}

func newRfsRoot(r *fs.LoopbackRoot, p *fs.Inode, n string, st *syscall.Stat_t) fs.InodeEmbedder {
	node := &rfsRoot{
		LoopbackNode: fs.LoopbackNode{
			RootData: r,
		},
	}
	return node
}

func (n *rfsRoot) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	fh, flags, _ := n.LoopbackNode.Open(ctx, flags)

	fakeError := syscall.EBUSY

	return fh, flags, fakeError
}
