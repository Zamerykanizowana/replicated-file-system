package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
		Test bool
	}{}
)

func main() {
	app := &cli.App{
		Name:     "rfs",
		HelpName: "Replicated file system using FUSE bindings and peer-2-peer architecture",
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
			logging.Configure(&conf.Logging)
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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	log.Info().
		Str("commit", commit).
		Str("branch", branch).
		Str("peer", flagValues.Name).
		Str("local_path", conf.Paths.FuseDir).
		Msg("Initializing FS")

	peer := p2p.NewPeer(flagValues.Name, conf.Peers, &conf.Connection, &mirror.Mirror{
		Conf: &conf.Paths,
	})
	peer.Run(ctx)

	server := rfs.NewRfsFuseServer(*conf, peer)

	if err := server.Mount(); err != nil {
		return errors.Wrap(err, "unable to mount fuse filesystem")
	}

	// Unmount filesystem upon receiving signal.
	go func() {
		sig := <-sigs
		log.Info().Str("signal", fmt.Sprint(sig)).Msg("Signal received!")
		peer.Close()
		err := server.Server.Unmount()
		if err != nil {
			log.Err(err).Msg("unsuccessful unmount")
		}
	}()

	server.Wait()
	return nil
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
