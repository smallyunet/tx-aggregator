package logger

import (
	"os"
	"time"
	"tx-aggregator/config"

	"github.com/rs/zerolog"
)

// Log is the global logger instance that can be used throughout the application
var Log zerolog.Logger

// Init initializes the global logger with console output configuration
// It sets up the logger with:
// - Console output writer
// - RFC3339 timestamp format
// - Caller information
// - Log level from application configuration
func Init() {
	// Configure console output writer with custom time format
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	// Initialize logger with timestamp and caller information
	Log = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	// Set global log level based on application configuration
	zerolog.SetGlobalLevel(zerolog.Level(config.AppConfig.Log.Level))

	// Log initialization completion
	Log.Info().Msg("Logger initialized successfully")
}
