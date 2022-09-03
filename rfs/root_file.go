package rfs

import (
	"context"
	"os"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/p2p"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func NewRfsFile(fd int, path string, host *p2p.Host, mirror Mirror) fs.FileHandle {
	return &rfsFile{
		loopbackFile: fs.NewLoopbackFile(fd).(loopbackFile),
		host:         host,
		mirror:       mirror,
		path:         path,
	}
}

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

type rfsFile struct {
	loopbackFile
	host   *p2p.Host
	mirror Mirror
	path   string
	mode   os.FileMode
}

func (f *rfsFile) Write(ctx context.Context, data []byte, off int64) (written uint32, errno syscall.Errno) {
	req := &protobuf.Request{
		Type:    protobuf.Request_WRITE,
		Content: data,
		Metadata: &protobuf.Request_Metadata{
			RelativePath: f.path,
			WriteOffset:  off,
		},
	}
	if typ := f.mirror.Consult(req).GetType(); typ == protobuf.Response_NACK {
		return 0, PermissionDenied
	}
	if err := f.host.Replicate(ctx, req); err != nil {
		log.Err(err).
			Object("request", req).
			Str("file", f.path).
			Msg("failed to write to file")
		return 0, PermissionDenied
	}

	return f.loopbackFile.Write(ctx, data, off)
}

func (f *rfsFile) Lseek(ctx context.Context, off uint64, whence uint32) (uint64, syscall.Errno) {
	log.Info().
		Str("path", f.path).
		Uint64("offset", off).
		Uint32("whence", whence).
		Msg("running lseek")
	return f.loopbackFile.Lseek(ctx, off, whence)
}
