package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/hanwen/go-fuse/v2/fs"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
)

var appConfig config.Config

var (
	branch = "unknown-branch"
	commit = "unknown-commit"
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

	fakeError := syscall.Errno(syscall.EBUSY)

	return fh, flags, fakeError
}

func main() {
	logging.Configure()
	appConfig = config.ReadConfig()

	zap.L().Info("Hello!", zap.String("commit", commit), zap.String("branch", branch))
	zap.L().Info("Initializing FS", zap.String("local_path", appConfig.Paths.FuseDir))

	if err := os.MkdirAll(appConfig.Paths.FuseDir, 0777); err != nil {
		zap.L().Fatal("unable to create local directory", zap.Error(err))
	}

	root := &fs.LoopbackRoot{
		NewNode: newRfsRoot,
		Path:    appConfig.Paths.MirrorDir,
	}

	sec := time.Second

	server, err := fs.Mount(appConfig.Paths.FuseDir, newRfsRoot(root, nil, "", nil), &fs.Options{
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	})

	if err != nil {
		zap.L().Fatal("unable to mount fuse filesystem", zap.Error(err))
	}

	zap.L().Info("unmount by calling", zap.String("cmd", fmt.Sprintf("fusermount -u %s", appConfig.Paths.FuseDir)))

	// Wait until user unmounts FS
	server.Wait()
}
