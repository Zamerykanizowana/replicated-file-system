package rfs

import (
	"context"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

// loopbackFile has to aggregate all of these functions so that we're able to preserve them
// when embedding the fs.loopbackFile (which is not exported...) inside rootFile.
type loopbackFile interface {
	Release(ctx context.Context) syscall.Errno
	Getattr(ctx context.Context, out *fuse.AttrOut) syscall.Errno
	Write(ctx context.Context, data []byte, off int64) (written uint32, errno syscall.Errno)
	Read(ctx context.Context, buf []byte, off int64) (res fuse.ReadResult, errno syscall.Errno)
	Getlk(ctx context.Context, owner uint64, lk *fuse.FileLock, flags uint32, out *fuse.FileLock) syscall.Errno
	Setlk(ctx context.Context, owner uint64, lk *fuse.FileLock, flags uint32) syscall.Errno
	Setlkw(ctx context.Context, owner uint64, lk *fuse.FileLock, flags uint32) syscall.Errno
	Lseek(ctx context.Context, off uint64, whence uint32) (uint64, syscall.Errno)
	Flush(ctx context.Context) syscall.Errno
	Fsync(ctx context.Context, flags uint32) syscall.Errno
	Allocate(ctx context.Context, off uint64, size uint64, mode uint32) syscall.Errno
}

func NewRfsFile(fd int, path string, replicator Replicator, mirror Mirror) fs.FileHandle {
	return &rootFile{
		loopbackFile: fs.NewLoopbackFile(fd).(loopbackFile),
		rep:          replicator,
		mirror:       mirror,
		path:         path,
		fd:           fd,
	}
}

type rootFile struct {
	loopbackFile
	rep    Replicator
	mirror Mirror
	path   string
	fd     int
}

func (f *rootFile) Write(ctx context.Context, data []byte, off int64) (written uint32, errno syscall.Errno) {
	if errno = replicate(ctx, &protobuf.Request{
		Type:    protobuf.Request_WRITE,
		Content: data,
		Metadata: &protobuf.Request_Metadata{
			RelativePath: f.path,
			WriteOffset:  off,
		},
	}, f.rep, f.mirror); errno != NoError {
		return 0, errno
	}

	return f.loopbackFile.Write(ctx, data, off)
}

// Lseek is here so that maybe one day we'll catch it in the wild! We were not able to observe when it's called
// that's why this log is still left here. Once we spot it and know when it's used the log can be removed.
func (f *rootFile) Lseek(ctx context.Context, off uint64, whence uint32) (uint64, syscall.Errno) {
	log.Info().
		Str("path", f.path).
		Uint64("offset", off).
		Uint32("whence", whence).
		Msg("running lseek")
	return f.loopbackFile.Lseek(ctx, off, whence)
}
