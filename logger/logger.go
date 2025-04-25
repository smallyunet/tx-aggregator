package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Log is the global logger instance that can be used throughout the application
var Log zerolog.Logger

func init() {
	// Set up a temporary basic logger before full config is available
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	Log = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	// Optional: allow early debug logging
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// Init initializes the global logger with console output configuration
// It sets up the logger with:
// - Console output writer
// - RFC3339 timestamp format
// - Caller information
// - Log level from application configuration
func Init(level int8) {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	Log = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	zerolog.SetGlobalLevel(zerolog.Level(level))

	Log.Info().
		Int("level", int(level)).
		Msg("Logger initialized successfully")
}
