package rfs

import (
	"context"
	"sync"
	"syscall"
	"time"
	"unsafe"

	//	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

// NewLoopbackFile creates a FileHandle out of a file descriptor. All
// operations are implemented. When using the Fd from a *os.File, call
// syscall.Dup() on the fd, to avoid os.File's finalizer from closing
// the file descriptor.
func NewRfsFile(fd int) fs.FileHandle {
	return &loopbackFile{fd: fd}
}

type loopbackFile struct {
	mu sync.Mutex
	fd int
}

func (f *loopbackFile) Allocate(ctx context.Context, off uint64, sz uint64, mode uint32) syscall.Errno {
	f.mu.Lock()
	defer f.mu.Unlock()
	err := syscall.Fallocate(f.fd, mode, int64(off), int64(sz))
	if err != nil {
		return syscall.Errno(fuse.ToStatus(err))
	}
	return syscall.Errno(0)
}

// Utimens - file handle based version of loopbackFileSystem.Utimens()
func (f *loopbackFile) utimens(a *time.Time, m *time.Time) syscall.Errno {
	var ts [2]syscall.Timespec
	ts[0] = fuse.UtimeToTimespec(a)
	ts[1] = fuse.UtimeToTimespec(m)
	err := futimens(int(f.fd), &ts)
	return syscall.Errno(fuse.ToStatus(err))
}

// futimens - futimens(3) calls utimensat(2) with "pathname" set to null and
// "flags" set to zero
func futimens(fd int, times *[2]syscall.Timespec) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_UTIMENSAT, uintptr(fd), 0, uintptr(unsafe.Pointer(times)), uintptr(0), 0, 0)
	if e1 != 0 {
		err = syscall.Errno(e1)
	}
	return
}

var _ = (fs.FileHandle)((*loopbackFile)(nil))
var _ = (fs.FileReleaser)((*loopbackFile)(nil))
var _ = (fs.FileGetattrer)((*loopbackFile)(nil))
var _ = (fs.FileReader)((*loopbackFile)(nil))
var _ = (fs.FileWriter)((*loopbackFile)(nil))
var _ = (fs.FileGetlker)((*loopbackFile)(nil))
var _ = (fs.FileSetlker)((*loopbackFile)(nil))
var _ = (fs.FileSetlkwer)((*loopbackFile)(nil))
var _ = (fs.FileLseeker)((*loopbackFile)(nil))
var _ = (fs.FileFlusher)((*loopbackFile)(nil))
var _ = (fs.FileFsyncer)((*loopbackFile)(nil))
var _ = (fs.FileSetattrer)((*loopbackFile)(nil))
var _ = (fs.FileAllocater)((*loopbackFile)(nil))

func (f *loopbackFile) Read(ctx context.Context, buf []byte, off int64) (res fuse.ReadResult, errno syscall.Errno) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := fuse.ReadResultFd(uintptr(f.fd), off, len(buf))
	return r, syscall.Errno(0)
}

func (f *loopbackFile) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	log.Info().Msg("Hello from Write")
	f.mu.Lock()
	defer f.mu.Unlock()
	n, err := syscall.Pwrite(f.fd, data, off)
	return uint32(n), syscall.Errno(fuse.ToStatus(err))
}

func (f *loopbackFile) Release(ctx context.Context) syscall.Errno {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fd != -1 {
		err := syscall.Close(f.fd)
		f.fd = -1
		return syscall.Errno(fuse.ToStatus(err))
	}
	return syscall.EBADF
}

func (f *loopbackFile) Flush(ctx context.Context) syscall.Errno {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Since Flush() may be called for each dup'd fd, we don't
	// want to really close the file, we just want to flush. This
	// is achieved by closing a dup'd fd.
	newFd, err := syscall.Dup(f.fd)

	if err != nil {
		return syscall.Errno(fuse.ToStatus(err))
	}
	err = syscall.Close(newFd)
	return syscall.Errno(fuse.ToStatus(err))
}

func (f *loopbackFile) Fsync(ctx context.Context, flags uint32) (errno syscall.Errno) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := syscall.Errno(fuse.ToStatus(syscall.Fsync(f.fd)))

	return r
}

const (
	_OFD_GETLK  = 36
	_OFD_SETLK  = 37
	_OFD_SETLKW = 38
)

func (f *loopbackFile) Getlk(ctx context.Context, owner uint64, lk *fuse.FileLock, flags uint32, out *fuse.FileLock) (errno syscall.Errno) {
	f.mu.Lock()
	defer f.mu.Unlock()
	flk := syscall.Flock_t{}
	lk.ToFlockT(&flk)
	errno = syscall.Errno(fuse.ToStatus(syscall.FcntlFlock(uintptr(f.fd), _OFD_GETLK, &flk)))
	out.FromFlockT(&flk)
	return
}

func (f *loopbackFile) Setlk(ctx context.Context, owner uint64, lk *fuse.FileLock, flags uint32) (errno syscall.Errno) {
	return f.setLock(ctx, owner, lk, flags, false)
}

func (f *loopbackFile) Setlkw(ctx context.Context, owner uint64, lk *fuse.FileLock, flags uint32) (errno syscall.Errno) {
	return f.setLock(ctx, owner, lk, flags, true)
}

func (f *loopbackFile) setLock(ctx context.Context, owner uint64, lk *fuse.FileLock, flags uint32, blocking bool) (errno syscall.Errno) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if (flags & fuse.FUSE_LK_FLOCK) != 0 {
		var op int
		switch lk.Typ {
		case syscall.F_RDLCK:
			op = syscall.LOCK_SH
		case syscall.F_WRLCK:
			op = syscall.LOCK_EX
		case syscall.F_UNLCK:
			op = syscall.LOCK_UN
		default:
			return syscall.EINVAL
		}
		if !blocking {
			op |= syscall.LOCK_NB
		}
		return syscall.Errno(fuse.ToStatus(syscall.Flock(f.fd, op)))
	} else {
		flk := syscall.Flock_t{}
		lk.ToFlockT(&flk)
		var op int
		if blocking {
			op = _OFD_SETLKW
		} else {
			op = _OFD_SETLK
		}
		return syscall.Errno(fuse.ToStatus(syscall.FcntlFlock(uintptr(f.fd), op, &flk)))
	}
}

func (f *loopbackFile) Setattr(ctx context.Context, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	if errno := f.setAttr(ctx, in); errno != 0 {
		return errno
	}

	return f.Getattr(ctx, out)
}

func (f *loopbackFile) setAttr(ctx context.Context, in *fuse.SetAttrIn) syscall.Errno {
	f.mu.Lock()
	defer f.mu.Unlock()
	var errno syscall.Errno
	if mode, ok := in.GetMode(); ok {
		errno = syscall.Errno(fuse.ToStatus(syscall.Fchmod(f.fd, mode)))
		if errno != 0 {
			return errno
		}
	}

	uid32, uOk := in.GetUID()
	gid32, gOk := in.GetGID()
	if uOk || gOk {
		uid := -1
		gid := -1

		if uOk {
			uid = int(uid32)
		}
		if gOk {
			gid = int(gid32)
		}
		errno = syscall.Errno(fuse.ToStatus(syscall.Fchown(f.fd, uid, gid)))
		if errno != 0 {
			return errno
		}
	}

	mtime, mok := in.GetMTime()
	atime, aok := in.GetATime()

	if mok || aok {
		ap := &atime
		mp := &mtime
		if !aok {
			ap = nil
		}
		if !mok {
			mp = nil
		}
		errno = f.utimens(ap, mp)
		if errno != 0 {
			return errno
		}
	}

	if sz, ok := in.GetSize(); ok {
		errno = syscall.Errno(fuse.ToStatus(syscall.Ftruncate(f.fd, int64(sz))))
		if errno != 0 {
			return errno
		}
	}
	return syscall.Errno(0)
}

func (f *loopbackFile) Getattr(ctx context.Context, a *fuse.AttrOut) syscall.Errno {
	f.mu.Lock()
	defer f.mu.Unlock()
	st := syscall.Stat_t{}
	err := syscall.Fstat(f.fd, &st)
	if err != nil {
		return syscall.Errno(fuse.ToStatus(err))
	}
	a.FromStat(&st)

	return syscall.Errno(0)
}

func (f *loopbackFile) Lseek(ctx context.Context, off uint64, whence uint32) (uint64, syscall.Errno) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, err := unix.Seek(f.fd, int64(off), int(whence))
	return uint64(n), syscall.Errno(fuse.ToStatus(err))
}
