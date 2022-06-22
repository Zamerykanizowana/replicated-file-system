package main

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
	"github.com/Zamerykanizowana/replicated-file-system/rfs"
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
	zap.L().Info("Initializing FS", zap.String("local_path", appConfig.Paths.FuseDir))

	server := rfs.NewRfsFuseServer(appConfig)

	err := server.Mount()

	if err != nil {
		zap.L().Fatal("unable to mount fuse filesystem", zap.Error(err))
	}

	zap.L().Info("unmount by calling", zap.String("cmd", fmt.Sprintf("fusermount -u %s", appConfig.Paths.FuseDir)))

	// Wait until user unmounts FS
	server.Server.Wait()
}
