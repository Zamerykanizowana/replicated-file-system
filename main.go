package main

import (
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
	"github.com/Zamerykanizowana/replicated-file-system/p2p"
	"github.com/Zamerykanizowana/replicated-file-system/rfs"
)

var (
	branch = "unknown-branch"
	commit = "unknown-commit"
)

func main() {
	logging.Configure()
	cfg := mustReadConfig()

	app := &cli.App{
		Name:     "rfs",
		HelpName: "Replicated file system using FUSE bindings and peer-2-peer architecture",
		Flags:    flags(),
		Commands: cli.Commands{
			{
				Name:   "p2p",
				Action: func(context *cli.Context) error { return runP2P(cfg) },
			},
			{
				Name:   "rfs",
				Action: func(context *cli.Context) error { return runRFS(cfg) },
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		zap.L().Fatal(err.Error())
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
	return p2p.Run(selfConfig, peersConfig)
}

func runRFS(cfg *config.Config) error {
	zap.L().Info("Hello!", zap.String("commit", commit), zap.String("branch", branch))
	zap.L().Info("Initializing FS", zap.String("local_path", cfg.Paths.FuseDir))

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
		zap.L().Panic(err.Error())
	}
	return cfg
}

var flagValues = struct {
	Name string
	Path cli.Path
}{}

func flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "name",
			Usage:       "Provide peer name, It must be also present in the config, linked to an address",
			Required:    true,
			Destination: &flagValues.Name,
			Aliases:     []string{"n"},
		},
		&cli.PathFlag{
			Name:        "config",
			DefaultText: "By default embedded config.json is loaded",
			Usage:       "Load configuration from 'FILE'",
			Destination: &flagValues.Path,
			Aliases:     []string{"c"},
			TakesFile:   true,
		},
	}
}
