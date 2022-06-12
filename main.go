package main

import (
	"os"

	"go.uber.org/zap"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
)

var appConfig config.Config

var (
	branch = "unknown-branch"
	commit = "unknown-commit"
)

func main() {
	logging.Configure()
	appConfig = config.ReadConfig()

	zap.L().Info("Hello!", zap.String("commit", commit), zap.String("branch", branch))
	zap.L().Info("Initializing FS", zap.String("local_path", appConfig.LocalDir))

	if err := os.MkdirAll(appConfig.LocalDir, 0660); err != nil {
		zap.L().Fatal("unable to create local directory", zap.Error(err))
	}
}
