package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Configure() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "severity"
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	config.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	config.OutputPaths = []string{"stdout"}

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
