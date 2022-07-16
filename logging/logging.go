package logging

import (
	stdLog "log"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"golang.org/x/term"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

func Configure(conf *config.Logging) {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimestampFieldName = "ts"
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldInteger = false
	zerolog.ErrorFieldName = "error.message"
	zerolog.ErrorStackFieldName = "error.stack"

	log.Logger = log.With().Caller().Stack().Logger()
	if term.IsTerminal(int(os.Stdout.Fd())) {
		log.Logger = log.Logger.Output(zerolog.NewConsoleWriter())
	}

	level, err := zerolog.ParseLevel(conf.Level)
	if err != nil {
		log.Err(err).Str("level", conf.Level).Msg("invalid log level, defaulting to info")
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Capture all logs, might be useful when a third party logs somewhere else.
	stdLog.SetFlags(0)
	stdLog.SetOutput(log.Logger)
}
