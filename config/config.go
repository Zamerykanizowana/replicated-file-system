package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

//go:embed config.json
var defaultConfig []byte

func Default() *Config {
	return mustUnmarshalConfig(defaultConfig)
}

func Read(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal().Err(err).
			Str("filepath", path).
			Msg("failed to open config file")
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		log.Fatal().Err(err).
			Str("filepath", path).
			Msg("failed to read config file contents")
	}
	return mustUnmarshalConfig(raw)
}

type (
	Config struct {
		Peers           []*Peer
		Paths           Paths
		TransportScheme string
	}
	Peer struct {
		Name    string
		Address string
	}
	Paths struct {
		FuseDir   string
		MirrorDir string
	}
)

type rawConfig struct {
	Peers []struct {
		Host string `json:"host"`
		Port uint   `json:"port"`
		Name string `json:"name"`
	} `json:"peers"`
	Paths struct {
		FuseDir   string `json:"fuse_dir"`
		MirrorDir string `json:"mirror_dir"`
	} `json:"paths"`
	TransportScheme string `json:"transport_scheme"`
}

func (c rawConfig) Validate() (err error) {
	if len(c.Peers) == 0 {
		err = multierr.Append(err, errors.New("provide at least one peer config"))
	}
	for _, p := range c.Peers {
		if len(p.Host) == 0 {
			err = multierr.Append(err, fmt.Errorf("host must not be empty for peer: %v", p))
		}
		if len(p.Name) == 0 {
			err = multierr.Append(err, fmt.Errorf("name must not be empty for peer: %v", p))
		}
		if p.Port == 0 {
			err = multierr.Append(err, fmt.Errorf("port must be greater than 0 for peer: %v", p))
		}
	}
	if c.TransportScheme != "tcp" {
		err = multierr.Append(err, errors.New("only 'tcp' scheme is supported now"))
	}
	return
}

func mustUnmarshalConfig(raw []byte) *Config {
	var (
		rc  rawConfig
		err error
	)
	if err = json.Unmarshal(raw, &rc); err != nil {
		log.Fatal().Err(err).Msg("failed to read config.json")
	}
	for _, path := range []*string{&rc.Paths.FuseDir, &rc.Paths.MirrorDir} {
		*path, err = expandHome(*path)
		if err != nil {
			log.Fatal().Err(err).
				Str("path", *path).
				Msg("failed to expand home")
		}
	}
	if err = rc.Validate(); err != nil {
		log.Fatal().Err(err).Msg("validation failed for config")
	}

	peers := make([]*Peer, 0, len(rc.Peers))
	for _, p := range rc.Peers {
		peers = append(peers, &Peer{
			Name:    p.Name,
			Address: mustBuildAddress(rc.TransportScheme, p.Host, p.Port),
		})
	}
	return &Config{
		Peers:           peers,
		Paths:           Paths(rc.Paths),
		TransportScheme: rc.TransportScheme,
	}
}

func mustBuildAddress(network, host string, port uint) string {
	address := fmt.Sprintf("%s:%d", host, port)
	tcpAddr, err := net.ResolveTCPAddr(network, address)
	if err != nil {
		log.Panic().
			Err(err).
			Str("network", network).
			Str("host", host).
			Uint("port", port).
			Msg("failed to resolve TCP address")
	}
	return tcpAddr.String()
}

func expandHome(path string) (string, error) {
	usr, _ := user.Current()

	if path == "~" {
		return "", errors.New("Mirroring home directory directly is not allowed!")
	} else if strings.HasPrefix(path, "~/") {
		return filepath.Join(usr.HomeDir, path[2:]), nil
	} else {
		return path, nil
	}
}
