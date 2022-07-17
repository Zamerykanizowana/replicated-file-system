package rfs

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

func NewReplicatedFuseServer(conf *config.Config, relay ReplicasRelay) *ReplicatedFuseServer {
	if err := os.MkdirAll(conf.Filesystem.FuseDir, 0777); err != nil {
		log.Fatal().Err(err).Msg("unable to create local directory")
	}

	r := &fs.LoopbackRoot{
		NewNode: newRoot,
		Path:    conf.Filesystem.MirrorDir,
	}

	return &ReplicatedFuseServer{
		LoopbackRoot: r,
		Root:         newRoot(r, nil, "", nil),
		conf:         conf,
		relay:        relay,
	}
}

type ReplicatedFuseServer struct {
	LoopbackRoot *fs.LoopbackRoot
	Root         fs.InodeEmbedder
	Server       *fuse.Server
	conf         *config.Config
	relay        ReplicasRelay
}

func (r *ReplicatedFuseServer) Mount() error {
	s, err := fs.Mount(
		r.conf.Filesystem.FuseDir,
		newRoot(r.LoopbackRoot, nil, "", nil),
		&fs.Options{
			MountOptions: fuse.MountOptions{
				FsName: "rfs",
				Name:   "rfs",
				Debug:  func() bool { return r.conf.Logging.Level == zerolog.LevelDebugValue }(),
				// This might come in handy, when negotiating access to the resource with replicas.
				EnableLocks: false,
			},
			AttrTimeout:  &r.conf.Filesystem.AttrTimeout,
			EntryTimeout: &r.conf.Filesystem.EntryTimeout,
		})
	if err != nil {
		return errors.Wrap(err, "failed to mount fs")
	}
	r.Server = s

	return err
}

func (r *ReplicatedFuseServer) Wait() {
	log.Info().
		Str("cmd", fmt.Sprintf("fusermount -u %s", r.conf.Filesystem.FuseDir)).
		Msg("unmount by calling")

	// Trap syscall.SIGINT and syscall.SIGTERM and unmount.
	go func() {
		notify := make(chan os.Signal, 1)
		signal.Notify(notify, syscall.SIGINT, syscall.SIGTERM)
		<-notify
		if err := r.Server.Unmount(); err != nil {
			log.Err(err).Msg("failed to unmount fs, it might need to be unmounted manually")
		}
	}()

	// Wait until user unmounts FS.
	r.Server.Wait()
}
