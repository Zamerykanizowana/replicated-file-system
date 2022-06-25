package rfs

import (
	"fmt"
	"os"
	"time"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"go.uber.org/zap"
)

type RfsFuseServer struct {
	LoopbackRoot *fs.LoopbackRoot
	RfsRoot      fs.InodeEmbedder
	Config       config.Config
	Server       *fuse.Server
}

func (r *RfsFuseServer) Mount() error {
	sec := time.Second

	s, err := fs.Mount(r.Config.Paths.FuseDir, newRfsRoot(r.LoopbackRoot, nil, "", nil), &fs.Options{
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	})

	if err == nil {
		r.Server = s
	}

	return err
}

func (r *RfsFuseServer) Wait() {
	zap.L().Info(
		"unmount by calling",
		zap.String(
			"cmd",
			fmt.Sprintf("fusermount -u %s", r.Config.Paths.FuseDir),
		),
	)

	// Wait until user unmounts FS
	r.Server.Wait()
}

func NewRfsFuseServer(c config.Config) *RfsFuseServer {
	if err := os.MkdirAll(c.Paths.FuseDir, 0777); err != nil {
		zap.L().Fatal("unable to create local directory", zap.Error(err))
	}

	root := &fs.LoopbackRoot{
		NewNode: newRfsRoot,
		Path:    c.Paths.MirrorDir,
	}

	return &RfsFuseServer{
		LoopbackRoot: root,
		RfsRoot:      newRfsRoot(root, nil, "", nil),
		Config:       c,
	}
}
