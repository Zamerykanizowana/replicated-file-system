package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"os/user"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

//go:embed config.json
var rawConfig []byte

type Config struct {
	Replicas []struct {
		Address string `json:"address"`
	} `json:"replicas"`
	Paths struct {
		FuseDir   string `json:"fuse_dir"`
		MirrorDir string `json:"mirror_dir"`
	} `json:"paths"`
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

func ReadConfig() Config {
	var config Config
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		zap.L().Fatal("failed to read config.json", zap.Error(err))
	}

	var err error
	for _, path := range []*string{&config.Paths.FuseDir, &config.Paths.MirrorDir} {
		*path, err = expandHome(*path)
		if err != nil {
			zap.L().Fatal("failed to expand home", zap.Error(err))
		}
	}

	return config
}
