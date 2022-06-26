package config

import (
	_ "embed"
	"encoding/json"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

//go:embed config.json
var rawConfig []byte


func Default() *Config {
	return mustUnmarshalConfig(rawConfig)
}

type Config struct {
	Replicas []struct {
		Address string `json:"address"`
	} `json:"replicas"`
	Paths struct {
		FuseDir   string `json:"fuse_dir"`
		MirrorDir string `json:"mirror_dir"`
	} `json:"paths"`
}

func Read(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		zap.L().
			With(zap.String("filepath", path)).
			Fatal("failed to open config file", zap.Error(err))
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		zap.L().
			With(zap.String("filepath", path)).
			Fatal("failed to read config file contents", zap.Error(err))
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

func Read(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		zap.L().
			With(zap.String("filepath", path)).
			Fatal("failed to open config file", zap.Error(err))
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		zap.L().
			With(zap.String("filepath", path)).
			Fatal("failed to read config file contents", zap.Error(err))
	}
	return mustUnmarshalConfig(raw)
}

type (
	Config struct {
		Peers []*PeerConfig
	}
	PeerConfig struct {
		Host string
		Name string
		Port uint
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
	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		zap.L().Fatal("failed to read config.json", zap.Error(err))
	}
	var err error
	for _, path := range []*string{&config.Paths.FuseDir, &config.Paths.MirrorDir} {
		*path, err = expandHome(*path)
		if err != nil {
			zap.L().Fatal("failed to expand home", zap.Error(err))
		}
	}
	return &config
}
