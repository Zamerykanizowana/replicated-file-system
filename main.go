package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
)

var appConfig config.Config

var (
	branch = "unknown-branch"
	commit = "unknown-commit"
)

type rfsRoot struct {
	fs.Inode
}

var staticFiles = map[string]string{
	"junkie": "this is some contents",
	"groovy": "and something else!",
}

func (root *rfsRoot) OnAdd(ctx context.Context) {
	for name, content := range staticFiles {
		p := &root.Inode

		memFile := &fs.MemRegularFile{
			Data: []byte(content),
		}

		inode := p.NewPersistentInode(ctx, memFile, fs.StableAttr{})

		p.AddChild(name, inode, true)
	}
}

func main() {
	logging.Configure()
	appConfig = config.ReadConfig()

	zap.L().Info("Hello!", zap.String("commit", commit), zap.String("branch", branch))
	zap.L().Info("Initializing FS", zap.String("local_path", appConfig.LocalDir))

	if err := os.MkdirAll(appConfig.LocalDir, 0777); err != nil {
		zap.L().Fatal("unable to create local directory", zap.Error(err))
	}

	root := &rfsRoot{}

	server, err := fs.Mount(appConfig.LocalDir, root, &fs.Options{
		MountOptions: fuse.MountOptions{Debug: true},
	})

	if err != nil {
		zap.L().Fatal("unable to mount fuse filesystem", zap.Error(err))
	}

	zap.L().Info("unmount by calling", zap.String("cmd", fmt.Sprintf("fusermount -u %s", appConfig.LocalDir)))

	// Wait until user unmounts FS
	server.Wait()
}
