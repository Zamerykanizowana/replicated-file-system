package rfs

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

type (
	Mirror interface {
		Consult(request *protobuf.Request) *protobuf.Response
	}
	Replicator interface {
		Replicate(ctx context.Context, request *protobuf.Request) error
	}
)

func NewServer(c config.Config, replicator Replicator, mirror Mirror) *Server {
	if err := os.MkdirAll(c.Paths.FuseDir, 0777); err != nil {
		log.Fatal().Err(err).Msg("unable to create local directory")
	}

	r := &root{
		rep:    replicator,
		mirror: mirror,
	}

	loopbackRoot := &fs.LoopbackRoot{
		NewNode: r.newRfsRoot,
		Path:    c.Paths.MirrorDir,
	}

	return &Server{
		LoopbackRoot: loopbackRoot,
		RfsRoot:      r.newRfsRoot(loopbackRoot, nil, "", nil),
		Config:       c,
		Replicator:   replicator,
	}
}

type Server struct {
	LoopbackRoot *fs.LoopbackRoot
	RfsRoot      fs.InodeEmbedder
	Config       config.Config
	Server       *fuse.Server
	Replicator   Replicator
}

func (s *Server) Mount() (err error) {
	sec := time.Second

	s.Server, err = fs.Mount(s.Config.Paths.FuseDir, s.RfsRoot, &fs.Options{
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	})
	return
}

func (s *Server) Close() error {
	log.Debug().Msg("closing FUSE server")
	if err := s.Server.Unmount(); err != nil {
		return errors.Wrapf(err,
			"failed to unmount fuse, unmount manually by calling: fusermount -u %s",
			s.Config.Paths.FuseDir)
	}
	return nil
}
