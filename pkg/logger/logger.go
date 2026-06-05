package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New creates a configured zerolog.Logger.
// In development it pretty-prints; in production it emits JSON.
func New(service, env string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	var w io.Writer
	if env == "production" {
		w = os.Stdout
	} else {
		w = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}
	}

	return zerolog.New(w).
		With().
		Timestamp().
		Str("service", service).
		Str("env", env).
		Logger()
}
