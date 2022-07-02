package logging

import (
	stdLog "log"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"golang.org/x/term"
)

func Configure() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.TimestampFieldName = "ts"
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldInteger = false
	zerolog.ErrorFieldName = "error.message"
	zerolog.ErrorStackFieldName = "error.stack"

	log.Logger = log.With().Caller().Stack().Logger()
	if term.IsTerminal(int(os.Stdout.Fd())) {
		log.Logger = log.Logger.Output(zerolog.NewConsoleWriter())
	}

	// Capture all logs, might be useful when a third party logs somewhere else.
	stdLog.SetFlags(0)
	stdLog.SetOutput(log.Logger)
}
