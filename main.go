package main

import (
	"errors"

	"go.uber.org/zap"

	"github.com/Zamerykanizowana/replicated-file-system/logging"
)

func main() {
	logging.Configure()
	zap.L().With(zap.Error(errors.New("lkol")), zap.Int("dupa", 1)).Info("o matko")
}
