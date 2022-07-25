package rfs

import (
	"context"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/rs/zerolog/log"
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

func (n *rfsRoot) Create(ctx context.Context, name string, flags uint32, mode uint32,
	out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	inode, fh, fflags, _ := n.LoopbackNode.Create(ctx, name, flags, mode, out)

	fakeError := syscall.EEXIST

	log.Info().Msg("error for create: EEXIST: File exists")

	return inode, fh, fflags, fakeError
}

func (n *rfsRoot) Link(ctx context.Context, target fs.InodeEmbedder, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	inode, _ := n.LoopbackNode.Link(ctx, target, name, out)

	fakeError := syscall.EXDEV

	// TODO: something is wrong with this func
	// When try to ln -s then error is: EMFILE Too many open files in system
	// When try to ln then error is: Invalid cross-device link
	log.Info().Msg("error for link: EXDEV: ")

	return inode, fakeError
}

func (n *rfsRoot) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode,
	syscall.Errno) {
	inode, _ := n.LoopbackNode.Mkdir(ctx, name, mode, out)

	fakeError := syscall.EAGAIN

	log.Info().Msg("error for mkdir: EAGAIN: Resource temporarily unavailable")

	return inode, fakeError
}

func (n *rfsRoot) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string,
	flags uint32) syscall.Errno {
	_ = n.LoopbackNode.Rename(ctx, name, newParent, newName, flags)

	fakeError := syscall.EBADF

	return fakeError
}

func (n *rfsRoot) Rmdir(ctx context.Context, name string) syscall.Errno {
	_ = n.LoopbackNode.Rmdir(ctx, name)

	fakeError := syscall.EISDIR

	return fakeError
}

func (n *rfsRoot) Setattr(ctx context.Context, fh fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	_ = n.LoopbackNode.Setattr(ctx, fh, in, out)

	fakeError := syscall.ENOSYS

	return fakeError
}

func (n *rfsRoot) Symlink(ctx context.Context, target, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	inode, _ := n.LoopbackNode.Symlink(ctx, target, name, out)

	fakeError := syscall.ENFILE

	return inode, fakeError
}

func (n *rfsRoot) Unlink(ctx context.Context, name string) syscall.Errno {
	_ = n.LoopbackNode.Unlink(ctx, name)

	fakeError := syscall.ENOSPC

	return fakeError
}

func (n *rfsRoot) CopyFileRange(ctx context.Context, fhIn fs.FileHandle,
	offIn uint64, out *fs.Inode, fhOut fs.FileHandle, offOut uint64,
	len uint64, flags uint64) (uint32, syscall.Errno) {
	fflags, _ := n.LoopbackNode.CopyFileRange(ctx, fhIn, offIn, out, fhOut, offOut, len, flags)

	fakeError := syscall.ETXTBSY

	return fflags, fakeError
}
