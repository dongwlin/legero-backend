package logger

import (
	"os"
	"path/filepath"
	"time"

	"github.com/dongwlin/legero-backend/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

func New(conf *config.Config) *zerolog.Logger {

	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldUnit = time.Nanosecond

	level, err := zerolog.ParseLevel(conf.Log.Level)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to parse log level")
	}

	logPath := filepath.Join("logs", "app.log")

	writer := zerolog.MultiLevelWriter(
		&lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    128, // megabytes
			MaxBackups: 3,   // files
			MaxAge:     90,  // days
			Compress:   true,
		},
		zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339Nano,
		},
	)

	log.Logger = zerolog.New(writer).
		With().
		Timestamp().
		Caller().
		Logger().
		Level(level)

	return &log.Logger
}
