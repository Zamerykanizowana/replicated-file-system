package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os/user"
	"path/filepath"
	"reflect"
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
	expandedPath := path

	if path == "~" {
		return "", errors.New("Mirroring home directory directly is not allowed!")
	} else if strings.HasPrefix(path, "~/") {
		expandedPath = filepath.Join(usr.HomeDir, path[2:])
	}

	return expandedPath, nil
}

func ReadConfig() Config {
	var config Config
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		zap.L().Fatal("failed to read config.json", zap.Error(err))
	}

	configPaths := reflect.ValueOf(&config.Paths).Elem()

	for i := 0; i < configPaths.NumField(); i++ {
		pathField := configPaths.Field(i)
		expandedPath, err := expandHome(fmt.Sprintf("%s", pathField))

		// TODO: log error!
		if err != nil {
			continue
		}

		pathField.SetString(expandedPath)
	}

	return config
}
