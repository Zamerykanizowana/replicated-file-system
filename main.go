package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
	"github.com/Zamerykanizowana/replicated-file-system/p2p"
)

var (
	branch = "unknown-branch"
	commit = "unknown-commit"
)

func main() {
	logging.Configure()

	app := &cli.App{
		Name:     "rfs",
		HelpName: "Replicated file system using FUSE bindings and peer-2-peer architecture",
		Action:   run,
		Flags:    flags(),
	}
	if err := app.Run(os.Args); err != nil {
		zap.L().Fatal(err.Error())
	}
}

func run(_ *cli.Context) error {
	var cfg *config.Config
	switch len(flagValues.Path) {
	case 0:
		cfg = config.Default()
	default:
		cfg = config.Read(flagValues.Path)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

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

func runRFS() {
	appConfig = config.ReadConfig()

	zap.L().Info("Hello!", zap.String("commit", commit), zap.String("branch", branch))
	zap.L().Info("Initializing FS", zap.String("local_path", appConfig.Paths.FuseDir))

	server := rfs.NewRfsFuseServer(appConfig)

	if err := server.Mount(); err != nil {
		zap.L().Fatal("unable to mount fuse filesystem", zap.Error(err))
	}

	server.Wait()
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
