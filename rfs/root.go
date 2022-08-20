package rfs

import (
	"context"
	"path"
	"path/filepath"
	"syscall"

	"github.com/Zamerykanizowana/replicated-file-system/p2p"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/rs/zerolog/log"
)

type rfsRoot struct {
	fs.LoopbackNode
	peer *p2p.Host
}

func (r *rfsRoot) newRfsRoot(lr *fs.LoopbackRoot, p *fs.Inode, n string, st *syscall.Stat_t) fs.InodeEmbedder {
	return &rfsRoot{
		peer:         r.peer,
		LoopbackNode: fs.LoopbackNode{RootData: lr}}
}

const PermissionDenied = syscall.EPERM

func (r *rfsRoot) Create(ctx context.Context, name string, flags uint32, mode uint32,
	out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {

	req := &protobuf.Request{
		Type: protobuf.Request_CREATE,
		Metadata: &protobuf.Request_Metadata{
			RelativePath: r.relativePath(name),
			Mode:         mode,
		},
	}
	if err := r.peer.Replicate(ctx, req); err != nil {
		log.Err(err).
			Object("request", req).
			Msg("failed to create the file")
		return nil, nil, 0, PermissionDenied
	}

	return r.LoopbackNode.Create(ctx, name, flags, mode, out)
}

// Link is for hard link, not for symlink
func (r *rfsRoot) Link(ctx context.Context, target fs.InodeEmbedder, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	inode, _ := r.LoopbackNode.Link(ctx, target, name, out)

	fakeError := syscall.EXDEV

	log.Info().Msg("error for link: EXDEV: Cross-device link")

	return inode, fakeError
}

func (r *rfsRoot) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode,
	syscall.Errno) {
	inode, _ := r.LoopbackNode.Mkdir(ctx, name, mode, out)

	fakeError := syscall.EAGAIN

	log.Info().Msg("error for mkdir: EAGAIN: Resource temporarily unavailable / Try again")

	return inode, fakeError
}

func (r *rfsRoot) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	flags = flags &^ syscall.O_APPEND
	p := filepath.Join(r.RootData.Path, r.Path(r.Root()))
	f, err := syscall.Open(p, int(flags), 0)
	if err != nil {
		return nil, 0, fs.ToErrno(err)
	}
	lf := NewRfsFile(f)

	log.Info().Msg("Hello from custom Open func")

	return lf, 0, 0
}

func (r *rfsRoot) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string,
	flags uint32) syscall.Errno {
	_ = r.LoopbackNode.Rename(ctx, name, newParent, newName, flags)

	fakeError := syscall.EBADF

	log.Info().Msg("error for rename: EBADF: File descriptor in bad state")

	return fakeError
}

func (r *rfsRoot) Rmdir(ctx context.Context, name string) syscall.Errno {
	_ = r.LoopbackNode.Rmdir(ctx, name)

	fakeError := syscall.EISDIR

	log.Info().Msg("error for rmdir: EISDIR: Is a directory")

	return fakeError
}

func (r *rfsRoot) Setattr(ctx context.Context, fh fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	_ = r.LoopbackNode.Setattr(ctx, fh, in, out)

	fakeError := syscall.ENOSYS

	log.Info().Msg("error for setattr: ENOSYS: Function not implemented")

	return fakeError
}

func (r *rfsRoot) Symlink(ctx context.Context, target, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	inode, _ := r.LoopbackNode.Symlink(ctx, target, name, out)

	fakeError := syscall.ENFILE

	log.Info().Msg("error for symlink: ENFILE: Too many open files in system")

	return inode, fakeError
}

func (r *rfsRoot) Unlink(ctx context.Context, name string) syscall.Errno {
	_ = r.LoopbackNode.Unlink(ctx, name)

	fakeError := syscall.ENOSPC

	log.Info().Msg("error for unlink: ENOSPC: No space left on device")

	return fakeError
}

func (r *rfsRoot) CopyFileRange(ctx context.Context, fhIn fs.FileHandle,
	offIn uint64, out *fs.Inode, fhOut fs.FileHandle, offOut uint64,
	len uint64, flags uint64) (uint32, syscall.Errno) {
	fflags, _ := r.LoopbackNode.CopyFileRange(ctx, fhIn, offIn, out, fhOut, offOut, len, flags)

	fakeError := syscall.ETXTBSY

	log.Info().Msg("error for copyfilerange: ETXTBSY: ")

	return fflags, fakeError
}

func (r *rfsRoot) relativePath(name string) string {
	return path.Join(r.Path(r.Root()), name)
}
