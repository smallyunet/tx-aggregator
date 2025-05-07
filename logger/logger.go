// Package logger provides a single global zerolog.Logger that can write to both
// the console and a daily‑rotated log file, each with its own format.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Log is the application‑wide logger. Use it directly:
//
//	logger.Log.Info().Msg("hello world")
var Log zerolog.Logger

// -----------------------------------------------------------------------------
// Fallback console logger (usable before Init is called).
// -----------------------------------------------------------------------------
func init() {
	console := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339, // ISO‑8601
	}

	Log = zerolog.New(console).
		With().
		Timestamp().
		Caller().
		Logger()

	zerolog.SetGlobalLevel(zerolog.InfoLevel) // default level
}

// Init configures the global logger.
//
// Parameters
//
//	level         Log level applied to *both* sinks (cast zerolog.<Level> to int8).
//	logDir        Directory where log files are stored (created if missing).
//	consoleFormat Format for console output: "text" or "json" (case‑insensitive).
//	fileFormat    Format for file output   : "text" or "json".
//
// Example
//
//	logger.Init(int8(zerolog.DebugLevel), "/var/log/myapp", "text", "json")
func Init(level int8, logDir, consoleFormat, fileFormat string) {
	// 1) Ensure the log directory exists.
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		fmt.Println("Failed to create log directory:", err)
	}

	// 2) Create / open today’s logfile: app‑YYYY‑MM‑DD.log.
	date := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("app-%s.log", date))

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		// Cannot write to file ➜ degrade gracefully to console‑only logging.
		fallback := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		Log = zerolog.New(fallback).With().Timestamp().Caller().Logger()
		Log.Error().Err(err).Msg("Failed to open log file (console‑only mode)")
		return
	}

	// 3) Build writers for console and file.
	consoleOut := buildWriter(strings.ToLower(consoleFormat), os.Stdout, true)
	fileOut := buildWriter(strings.ToLower(fileFormat), logFile, false)

	// 4) Fan‑out every log entry to both destinations.
	multi := io.MultiWriter(consoleOut, fileOut)

	// 5) Replace global logger.
	Log = zerolog.New(multi).
		With().
		Timestamp().
		Caller().
		Logger()

	zerolog.SetGlobalLevel(zerolog.Level(level))

	Log.Info().
		Int("level", int(level)).
		Str("console_format", strings.ToLower(consoleFormat)).
		Str("file_format", strings.ToLower(fileFormat)).
		Str("log_file", logPath).
		Msg("Logger initialized")
}

// buildWriter returns an io.Writer that writes in either text (ConsoleWriter) or
// raw JSON format. `forConsole == true` enables ANSI colors.
func buildWriter(format string, target io.Writer, forConsole bool) io.Writer {
	switch format {
	case "json":
		return target
	default: // "text"
		return zerolog.ConsoleWriter{
			Out:        target,
			TimeFormat: time.RFC3339,
			NoColor:    !forConsole, // disable colors for file output
		}
	}
}
