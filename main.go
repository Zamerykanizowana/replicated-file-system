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
	"github.com/Zamerykanizowana/replicated-file-system/rfs"
)

var (
	branch = "unknown-branch"
	commit = "unknown-commit"

	flagValues = struct {
		Name string
		Path cli.Path
	}{}
)

func main() {
	logging.Configure()
	cfg := mustReadConfig()

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
		Commands: cli.Commands{
			{
				Name:   "p2p",
				Action: func(context *cli.Context) error { return runP2P(cfg) },
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "name",
						Usage:       "Provide peer name, It must be also present in the config, linked to an address",
						Required:    true,
						Destination: &flagValues.Name,
						Aliases:     []string{"n"},
					},
				},
			},
			{
				Name:   "rfs",
				Action: func(context *cli.Context) error { return runRFS(cfg) },
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func runP2P(cfg *config.Config) error {
	var selfConfig *config.PeerConfig
	peersConfig := make([]*config.PeerConfig, 0, len(cfg.Peers)-1)
	for _, p := range cfg.Peers {
		if p.Name == flagValues.Name {
			selfConfig = p
			continue
		}
		peersConfig = append(peersConfig, p)
	}
	if selfConfig == nil {
		return fmt.Errorf("peer with name %s was not found", flagValues.Name)
	}
	return p2p.NewPeer(selfConfig, peersConfig).Run()
}

func runRFS(cfg *config.Config) error {
	log.Info().
		Str("commit", commit).
		Str("branch", branch).
		Str("local_path", cfg.Paths.FuseDir).
		Msg("Initializing FS")

	server := rfs.NewRfsFuseServer(*cfg)

	if err := server.Mount(); err != nil {
		return errors.Wrap(err, "unable to mount fuse filesystem")
	}

	server.Wait()
	return nil
}

func mustReadConfig() *config.Config {
	var cfg *config.Config
	switch len(flagValues.Path) {
	case 0:
		cfg = config.Default()
	default:
		cfg = config.Read(flagValues.Path)
	}
	if err := cfg.Validate(); err != nil {
		log.Panic().Err(err).Send()
	}
	return cfg
}
