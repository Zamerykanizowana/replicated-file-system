package rfs

import (
	"fmt"
	"github.com/Zamerykanizowana/replicated-file-system/p2p"
	"os"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

type RfsFuseServer struct {
	LoopbackRoot *fs.LoopbackRoot
	RfsRoot      fs.InodeEmbedder
	Config       config.Config
	Server       *fuse.Server
	Peer         *p2p.Peer
}

func (r *RfsFuseServer) Mount() error {
	sec := time.Second

	s, err := fs.Mount(r.Config.Paths.FuseDir, r.RfsRoot, &fs.Options{
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	})

	if err == nil {
		r.Server = s
	}

	return err
}

func (r *RfsFuseServer) Wait() {
	log.Info().
		Str("cmd", fmt.Sprintf("fusermount -u %s", r.Config.Paths.FuseDir)).
		Msg("unmount by calling")

	// Wait until user unmounts FS
	r.Server.Wait()
}

func NewRfsFuseServer(c config.Config, p *p2p.Peer) *RfsFuseServer {
	if err := os.MkdirAll(c.Paths.FuseDir, 0777); err != nil {
		log.Fatal().Err(err).Msg("unable to create local directory")
	}

	root := &rfsRoot{
		peer: p,
	}

	loopbackRoot := &fs.LoopbackRoot{
		NewNode: root.newRfsRoot,
		Path:    c.Paths.MirrorDir,
	}

	return &RfsFuseServer{
		LoopbackRoot: loopbackRoot,
		RfsRoot:      root,
		Config:       c,
		Peer:         p,
	}
}
