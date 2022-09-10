package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/Zamerykanizowana/replicated-file-system/connection"
	"github.com/Zamerykanizowana/replicated-file-system/connection/tlsconf"
	"github.com/Zamerykanizowana/replicated-file-system/mirror"
	"github.com/Zamerykanizowana/replicated-file-system/p2p"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
	"github.com/Zamerykanizowana/replicated-file-system/rfs"
)

var (
	// These are set during build through ldflags.
	branch, commit, name string

	// flagValues is a value holder for app flags.
	flagValues = struct {
		Name string
		Path cli.Path
	}{}
)

func main() {
	app := &cli.App{
		Name:  "rfs",
		Usage: "Replicated file system using FUSE bindings and peer-2-peer architecture",
		//HelpName: "Replicated file system using FUSE bindings and peer-2-peer architecture",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:        "config",
				DefaultText: "By default embedded config.json is loaded",
				Usage:       "Load configuration from 'FILE'",
				Destination: &flagValues.Path,
				Aliases:     []string{"c"},
				TakesFile:   true,
			},
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Provide peer name, It must be also present in the config, linked to an address",
				Value:       name,
				Destination: &flagValues.Name,
				Aliases:     []string{"n"},
			},
		},
		Action: func(context *cli.Context) error {
			conf := mustReadConfig()
			logging.Configure(conf.Logging)
			log.Debug().Object("config", conf).Msg("loaded config")
			protobuf.SetCompression(conf.Connection.Compression)

			return run(context.Context, conf)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func run(ctx context.Context, conf *config.Config) error {
	sig := newSignalHandler()

	log.Info().
		Str("commit", commit).
		Str("branch", branch).
		Str("host", flagValues.Name).
		Str("local_path", conf.Paths.FuseDir).
		Msg("Initializing Replicated File System")

	mir := mirror.NewMirror(conf.Paths)

	host, peers := conf.Peers.Pop(flagValues.Name)
	if len(host.Name) == 0 {
		log.Fatal().
			Str("name", flagValues.Name).
			Interface("peers_config", conf.Peers).
			Msg("invalid peer name provided, peer must be listed in the peers config")
	}
	conn := connection.NewPool(host, peers, conf.Connection, tlsconf.Default(conf.Connection.GetTLSVersion()))

	p2pHost := p2p.NewHost(host, peers, conn, mir, conf.ReplicationTimeout)
	p2pHost.Run(ctx)

	server := rfs.NewServer(*conf, p2pHost, mir)
	if err := server.Mount(); err != nil {
		return errors.Wrap(err, "unable to mount fuse filesystem")
	}

	// Unmount filesystem and close connections upon receiving signal.
	sig.closeOnSignal(p2pHost, server)
	return nil
}

func newSignalHandler() *signalHandler {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	return &signalHandler{sigs: sigs}
}

type signalHandler struct {
	sigs chan os.Signal
}

func (s signalHandler) closeOnSignal(closers ...io.Closer) {
	sig := <-s.sigs
	log.Info().Stringer("signal", sig).Msg("Signal received!")
	for _, c := range closers {
		if err := c.Close(); err != nil {
			log.Err(err).Send()
		}
	}
}

func mustReadConfig() *config.Config {
	var conf *config.Config
	switch len(flagValues.Path) {
	case 0:
		conf = config.Default()
	default:
		conf = config.Read(flagValues.Path)
	}
	return conf
}
