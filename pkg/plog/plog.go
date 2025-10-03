package plog

// Import & call this to hook slog to capture a history of logs

import (
	"io"
	"log/slog"
	"os"
)

var logg *slog.Logger

var LogHistory []string

type ioWrapper struct {
	writer io.Writer
}

func (w *ioWrapper) Write(p []byte) (n int, err error) {
	LogHistory = append(LogHistory, string(p))
	if len(LogHistory) > 1000 {
		LogHistory = LogHistory[:1000]
	}
	return w.writer.Write(p)
}

func Init(level slog.Level) {
	wrapped := &ioWrapper{writer: os.Stderr}
	if os.Getenv("TREASURY_LOG_FORMAT") == "json" {
		logg = slog.New(slog.NewJSONHandler(wrapped, &slog.HandlerOptions{Level: level}))
	} else {
		logg = slog.New(slog.NewTextHandler(wrapped, &slog.HandlerOptions{Level: level}))
	}
	slog.SetDefault(logg)
}
