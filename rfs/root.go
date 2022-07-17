package rfs

import (
	"context"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/rs/zerolog/log"
)

type root struct {
	fs.LoopbackNode
}

func newRoot(r *fs.LoopbackRoot, p *fs.Inode, n string, st *syscall.Stat_t) fs.InodeEmbedder {
	node := &root{
		LoopbackNode: fs.LoopbackNode{
			RootData: r,
		},
	}
	return node
}

func (r *root) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	fh, flags, _ := r.LoopbackNode.Open(ctx, flags)

	fakeError := syscall.EBUSY

	return fh, flags, fakeError
}

func (r *root) Create(
	ctx context.Context,
	name string,
	flags, mode uint32,
	out *fuse.EntryOut,
) (
	node *fs.Inode,
	fh fs.FileHandle,
	fuseFlags uint32,
	errno syscall.Errno,
) {
	node, fh, fuseFlags, errno = r.LoopbackNode.Create(ctx, name, flags, mode, out)
	log.Info().Msgf("created: %s", name)
	return
}
