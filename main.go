package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
	"github.com/Zamerykanizowana/replicated-file-system/p2p"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
	"github.com/Zamerykanizowana/replicated-file-system/rfs"
	"github.com/Zamerykanizowana/replicated-file-system/test"
)

var (
	// These are set during build through ldflags.
	branch, commit, peer string

	// flagValues is a value holder for app flags.
	flagValues = struct {
		Name string
		Path cli.Path
		Test bool
	}{}
)

func main() {
	logging.Configure()
	conf := mustReadConfig()

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
		},
		Action: func(context *cli.Context) error { return run() },
		Commands: cli.Commands{
			{
				Name:   "p2p",
				Action: func(context *cli.Context) error { return runP2P(conf) },
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "name",
						Usage:       "Provide peer name, It must be also present in the config, linked to an address",
						Required:    true,
						Destination: &flagValues.Name,
						Aliases:     []string{"n"},
					},
					&cli.BoolFlag{
						Name:        "test",
						Usage:       "Run P2P test.",
						Destination: &flagValues.Test,
					},
				},
			},
			{
				Name:   "rfs",
				Action: func(context *cli.Context) error { return runRFS(conf) },
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func run() error {
	conf := mustReadConfig()
	return runP2P(conf)
}

func runP2P(conf *config.Config) error {
	protobuf.SetCompression("DefaultCompression")

	if peer == "" {
		peer = flagValues.Name
	}

	var selfConfig *config.Peer
	peersConfig := make([]*config.Peer, 0, len(conf.Peers)-1)
	for _, p := range conf.Peers {
		if p.Name == peer {
			selfConfig = p
			continue
		}
		peersConfig = append(peersConfig, p)
	}
	if selfConfig == nil {
		return fmt.Errorf("peer with name %s was not found", peer)
	}
	node, err := p2p.NewNode(selfConfig, peersConfig, &conf.Connection)
	if err != nil {
		return err
	}
	return test.P2P(node)
}

func runRFS(conf *config.Config) error {
	log.Info().
		Str("commit", commit).
		Str("branch", branch).
		Str("peer", peer).
		Str("local_path", conf.Paths.FuseDir).
		Msg("Initializing FS")

	server := rfs.NewRfsFuseServer(*conf)

	if err := server.Mount(); err != nil {
		return errors.Wrap(err, "unable to mount fuse filesystem")
	}

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
