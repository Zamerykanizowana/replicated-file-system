package logging

import (
	"go.uber.org/zap"
)

func Configure() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "severity"
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	var (
		logger *zap.Logger
		err    error
	)
	if logger, err = config.Build(); err != nil {
		logger = zap.NewNop()
	} else {
		logger = logger.Named("default")
	}

	zap.ReplaceGlobals(logger)
}
