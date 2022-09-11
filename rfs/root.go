package rfs

import (
	"context"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/rs/zerolog/log"
)

type root struct {
	fs.LoopbackNode
	rep    Replicator
	mirror Mirror
}

func (r *root) newRfsRoot(lr *fs.LoopbackRoot, _ *fs.Inode, _ string, _ *syscall.Stat_t) fs.InodeEmbedder {
	return &root{
		rep:          r.rep,
		mirror:       r.mirror,
		LoopbackNode: fs.LoopbackNode{RootData: lr}}
}

const (
	NoError          = 0
	PermissionDenied = syscall.EPERM
)

func (r *root) Replicate(ctx context.Context, req *protobuf.Request) syscall.Errno {
	return replicate(ctx, req, r.rep, r.mirror)
}

func (r *root) Create(ctx context.Context, name string, flags uint32, mode uint32,
	out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	p := r.physicalPath(name)

	log.Info().
		Uint32("mode", mode).
		Str("path", p).
		Msg("creating a file")

	if errno := r.Replicate(ctx, &protobuf.Request{
		Type: protobuf.Request_CREATE,
		Metadata: &protobuf.Request_Metadata{
			RelativePath: r.relativePath(name),
			Mode:         mode,
		},
	}); errno != NoError {
		return nil, nil, 0, errno
	}

	flags = flags &^ syscall.O_APPEND
	fd, err := syscall.Open(p, int(flags)|os.O_CREATE, mode)
	if err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}

	st := syscall.Stat_t{}
	if err = syscall.Fstat(fd, &st); err != nil {
		_ = syscall.Close(fd)
		return nil, nil, 0, fs.ToErrno(err)
	}

	node := r.RootData.NewNode(r.RootData, r.EmbeddedInode(), name, &st)
	ch := r.NewInode(ctx, node, r.idFromStat(&st))
	lf := NewRfsFile(fd, r.relativePath(name), r.rep, r.mirror)

	out.FromStat(&st)
	return ch, lf, 0, 0
}

func (r *root) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (
	*fs.Inode, syscall.Errno,
) {
	if errno := r.Replicate(ctx, &protobuf.Request{
		Type: protobuf.Request_MKDIR,
		Metadata: &protobuf.Request_Metadata{
			RelativePath: r.relativePath(name),
			Mode:         mode,
		},
	}); errno != NoError {
		return nil, errno
	}

	return r.LoopbackNode.Mkdir(ctx, name, mode, out)
}

func (r *root) Open(_ context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	flags = flags &^ syscall.O_APPEND
	p := filepath.Join(r.RootData.Path, r.Path(r.Root()))
	f, err := syscall.Open(p, int(flags), 0)
	if err != nil {
		return nil, 0, fs.ToErrno(err)
	}
	return NewRfsFile(f, r.Path(r.Root()), r.rep, r.mirror), 0, 0
}

func (r *root) Rename(
	ctx context.Context,
	name string, newParent fs.InodeEmbedder,
	newName string, flags uint32,
) syscall.Errno {
	if errno := r.Replicate(ctx, &protobuf.Request{
		Type: protobuf.Request_RENAME,
		Metadata: &protobuf.Request_Metadata{
			RelativePath: r.relativePath(name),
			NewRelativePath: filepath.Join(
				newParent.EmbeddedInode().Path(nil),
				r.relativePath(newName)),
		},
	}); errno != NoError {
		return errno
	}

	return r.LoopbackNode.Rename(ctx, name, newParent, newName, flags)
}

func (r *root) Rmdir(ctx context.Context, name string) syscall.Errno {
	if errno := r.Replicate(ctx, &protobuf.Request{
		Type: protobuf.Request_RMDIR,
		Metadata: &protobuf.Request_Metadata{
			RelativePath: r.relativePath(name),
		},
	}); errno != NoError {
		return errno
	}

	return r.LoopbackNode.Rmdir(ctx, name)
}

func (r *root) Setattr(ctx context.Context, _ fs.FileHandle, in *fuse.SetAttrIn, _ *fuse.AttrOut) syscall.Errno {
	mode, ok := in.GetMode()
	// The caller did not request mode change.
	// Considering that it's the only thing that we handle, we can call it quits at this point.
	if !ok {
		return fs.OK
	}

	if errno := r.Replicate(ctx, &protobuf.Request{
		Type: protobuf.Request_SETATTR,
		Metadata: &protobuf.Request_Metadata{
			RelativePath: r.Path(r.Root()),
			Mode:         mode,
		},
	}); errno != NoError {
		return errno
	}

	// No name as we're operating on the existing node (can't go deeper).
	p := r.physicalPath("")

	if err := syscall.Chmod(p, mode); err != nil {
		return fs.ToErrno(err)
	}

	return fs.OK
}

func (r *root) Symlink(ctx context.Context, target, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	if errno := r.Replicate(ctx, &protobuf.Request{
		Type: protobuf.Request_SYMLINK,
		Metadata: &protobuf.Request_Metadata{
			RelativePath:    r.relativePath(target),
			NewRelativePath: r.relativePath(name),
		},
	}); errno != NoError {
		return nil, errno
	}

	return r.LoopbackNode.Symlink(ctx, target, name, out)
}

// Link is for hard link, not for symlink.
// It behaves just like Symlink and it literally calls it under the hood.
func (r *root) Link(
	ctx context.Context, target fs.InodeEmbedder,
	name string, out *fuse.EntryOut,
) (*fs.Inode, syscall.Errno) {
	targetPath := target.EmbeddedInode().Path(nil)
	if errno := r.Replicate(ctx, &protobuf.Request{
		Type: protobuf.Request_LINK,
		Metadata: &protobuf.Request_Metadata{
			RelativePath:    r.relativePath(targetPath),
			NewRelativePath: r.relativePath(name),
		},
	}); errno != NoError {
		return nil, errno
	}

	return r.LoopbackNode.Symlink(ctx, targetPath, name, out)
}

func (r *root) Unlink(ctx context.Context, name string) syscall.Errno {
	if errno := r.Replicate(ctx, &protobuf.Request{
		Type: protobuf.Request_UNLINK,
		Metadata: &protobuf.Request_Metadata{
			RelativePath: r.relativePath(name),
		},
	}); errno != NoError {
		return errno
	}

	return r.LoopbackNode.Unlink(ctx, name)
}

func (r *root) CopyFileRange(
	ctx context.Context, fhIn fs.FileHandle,
	offIn uint64, _ *fs.Inode, fhOut fs.FileHandle,
	offOut, len, flags uint64,
) (uint32, syscall.Errno) {
	rfsInFile, _ := fhIn.(*rootFile)
	rfsOutFile, _ := fhOut.(*rootFile)

	if errno := r.Replicate(ctx, &protobuf.Request{
		Type: protobuf.Request_COPY_FILE_RANGE,
		Metadata: &protobuf.Request_Metadata{
			RelativePath:    rfsInFile.path,
			NewRelativePath: rfsOutFile.path,
			WriteOffset:     int64(offOut),
			ReadOffset:      int64(offIn),
			WriteLen:        int64(len),
		},
	}); errno != NoError {
		return 0, errno
	}

	signedOffIn := int64(offIn)
	signedOffOut := int64(offOut)
	count, err := unix.CopyFileRange(rfsInFile.fd, &signedOffIn, rfsOutFile.fd, &signedOffOut, int(len), int(flags))
	return uint32(count), fs.ToErrno(err)
}

// physicalPath returns the full path to the file in the underlying file system.
// It is therefore an equivalent to *LoopbackNode.path.
// This path is only relevant in the context of local changes (i.e. it shouldn't
// be sent over the wire).
func (r *root) physicalPath(name string) string {
	return filepath.Join(r.RootData.Path, r.relativePath(name))
}

// relativePath returns the full file path without the component derived
// from the underlying file system.
func (r *root) relativePath(name string) string {
	return filepath.Join(r.Path(r.Root()), name)
}

// idFromStat is a rip-off from go-fuse's loopback.go.
func (r *root) idFromStat(st *syscall.Stat_t) fs.StableAttr {
	dev := r.LoopbackNode.RootData.Dev
	swapped := (st.Dev << 32) | (st.Dev >> 32)
	swappedRootDev := (dev << 32) | (dev >> 32)
	return fs.StableAttr{
		Mode: st.Mode,
		Gen:  1,
		Ino:  (swapped ^ swappedRootDev) ^ st.Ino,
	}
}
