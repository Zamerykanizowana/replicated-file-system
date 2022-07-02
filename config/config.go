package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

//go:embed config.json
var rawConfig []byte

func Default() *Config {
	return mustUnmarshalConfig(rawConfig)
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

type (
	Config struct {
		Peers []*PeerConfig
		Paths struct {
			FuseDir   string `json:"fuse_dir"`
			MirrorDir string `json:"mirror_dir"`
		} `json:"paths"`
	}
	PeerConfig struct {
		Host string `json:"host"`
		Name string `json:"name"`
		Port uint   `json:"port"`
	}
)

func (c Config) Validate() (err error) {
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
	return
}

func mustUnmarshalConfig(raw []byte) *Config {
	var (
		config Config
		err    error
	)
	if err = json.Unmarshal(raw, &config); err != nil {
		log.Fatal().Err(err).Msg("failed to read config.json")
	}
	for _, path := range []*string{&config.Paths.FuseDir, &config.Paths.MirrorDir} {
		*path, err = expandHome(*path)
		if err != nil {
			log.Fatal().Err(err).
				Str("path", *path).
				Msg("failed to expand home")
		}
	}
	return &config
}
