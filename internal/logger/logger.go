package logger

import (
	"io"
	"log/slog"
	"os"
)

// Config holds settings for the logging system.
type Config struct {
	Dir      string // directory for log files (default: "logs")
	Prefix   string // filename prefix (default: "gatheryourdeals")
	MaxBytes int64  // max file size in bytes (default: 10 MB)
	MaxFiles int    // max files to keep (default: 2)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Dir:      "logs",
		Prefix:   "gatheryourdeals",
		MaxBytes: 10 * 1024 * 1024, // 10 MB
		MaxFiles: 2,
	}
}

// Logger wraps the rotating writer and provides both slog and io.Writer
// interfaces so Gin and application code share the same output.
type Logger struct {
	*slog.Logger
	writer  io.Writer        // multi-writer (stdout + file)
	rotator *RotatingWriter
}

// New creates a Logger that writes to both stdout and a rotating log file.
func New(cfg Config) (*Logger, error) {
	rotator, err := NewRotatingWriter(cfg.Dir, cfg.Prefix, cfg.MaxBytes, cfg.MaxFiles)
	if err != nil {
		return nil, err
	}

	multi := io.MultiWriter(os.Stdout, rotator)

	handler := slog.NewTextHandler(multi, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	l := &Logger{
		Logger:  slog.New(handler),
		writer:  multi,
		rotator: rotator,
	}
	return l, nil
}

// Writer returns the underlying io.Writer that writes to both stdout
// and the rotating log file. Pass this to gin.DefaultWriter so Gin
// logs end up in the same destination.
func (l *Logger) Writer() io.Writer {
	return l.writer
}

// Close flushes and closes the log file.
func (l *Logger) Close() error {
	return l.rotator.Close()
}
