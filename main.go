package main

import (
	"go.uber.org/zap"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
)

var appConfig config.Config

func main() {
	logging.Configure()
	appConfig = config.ReadConfig()

	zap.L().Info("Initializing FS", zap.String("local_path", appConfig.LocalDir))
}
