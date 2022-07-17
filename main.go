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
	conf := mustReadConfig()
	logging.Configure(&conf.Logging)
	protobuf.SetCompression(conf.Connection.Compression)

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
				Required:    true,
				Destination: &flagValues.Name,
				Aliases:     []string{"n"},
			},
		},
		Action: func(context *cli.Context) error { return run(conf) },
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func run(conf *config.Config) error {
	log.Info().
		Str("commit", commit).
		Str("branch", branch).
		Str("peer", peer).
		Str("local_path", conf.Filesystem.FuseDir).
		Msg("Initializing FS")

	node, err := p2pNode(conf)
	if err != nil {
		return err
	}

	server := rfs.NewReplicatedFuseServer(conf, node)

	if err = server.Mount(); err != nil {
		return errors.Wrap(err, "unable to mount fuse filesystem")
	}

	server.Wait()
	return nil
}

func p2pNode(conf *config.Config) (*p2p.Node, error) {
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
		return nil, fmt.Errorf("peer with name %s was not found", peer)
	}
	return p2p.NewNode(selfConfig, peersConfig, &conf.Connection), nil
}

func mustReadConfig() *config.Config {
	var conf *config.Config
	switch len(flagValues.Path) {
	case 0:
		conf = config.Default()
	default:
		conf = config.Read(flagValues.Path)
	}
	log.Debug().Object("config", conf).Msg("loaded config")
	return conf
}
