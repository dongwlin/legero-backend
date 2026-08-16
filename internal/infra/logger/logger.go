package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// New initializes the global zerolog logger with a ConsoleWriter (RFC3339
// timestamps) and returns it. It is called first thing in cmd.Execute() so all
// startup output — config loading, migrations, CLI bootstrap — uses the
// configured format rather than zerolog's default JSON logger.
func New() zerolog.Logger {
	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	logger := zerolog.New(writer).With().Timestamp().Logger()
	log.Logger = logger

	return logger
}
