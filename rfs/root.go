package rfs

import (
	"context"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
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

	return inode, fh, fflags, fakeError
}

func (n *rfsRoot) Link(ctx context.Context, target fs.InodeEmbedder, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	inode, _ := n.LoopbackNode.Link(ctx, target, name, out)

	fakeError := syscall.EXDEV

	return inode, fakeError
}

func (n *rfsRoot) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode,
	syscall.Errno) {
	inode, _ := n.LoopbackNode.Mkdir(ctx, name, mode, out)

	fakeError := syscall.EAGAIN

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

func (n *rfsRoot) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	fh, flags, _ := n.LoopbackNode.Open(ctx, flags)

	fakeError := syscall.EBUSY

	return fh, flags, fakeError
}
