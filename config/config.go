package config

import (
	_ "embed"
	"encoding/json"

	"go.uber.org/zap"
)

//go:embed config.json
var rawConfig []byte

type Config struct {
	Replicas []struct {
		Address string `json:"address"`
	} `json:"replicas"`
}

func ReadConfig() *Config {
	var config *Config
	if err := json.Unmarshal(rawConfig, config); err != nil {
		zap.L().Fatal("failed to read config.json", zap.Error(err))
	}
	return config
}
