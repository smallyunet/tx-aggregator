package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func init() {
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	Log = zerolog.New(consoleWriter).
		With().
		Timestamp().
		Caller().
		Logger()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func Init(level int8, logDir string) {
	// Ensure logs directory exists
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		fmt.Println("Failed to create logs directory:", err)
	}

	// Generate daily log file name
	today := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("app-%s.log", today))

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		Log = zerolog.New(consoleWriter).
			With().
			Timestamp().
			Caller().
			Logger()

		Log.Error().Err(err).Msg("Failed to open log file")
		return
	}

	// Combine outputs
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	multiWriter := io.MultiWriter(consoleWriter, logFile)

	Log = zerolog.New(multiWriter).
		With().
		Timestamp().
		Caller().
		Logger()

	zerolog.SetGlobalLevel(zerolog.Level(level))

	Log.Info().
		Int("level", int(level)).
		Str("log_file", logPath).
		Msg("Logger initialized successfully with file rotation")
}
