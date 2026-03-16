package logger

import (
	"context"
	"io"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
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
	writer  io.Writer // multi-writer (stdout + file)
	rotator *RotatingWriter
}

// traceIDHandler wraps an inner slog.Handler and injects a "trace_id"
// attribute when the context carries an active OTel span (FR-008).
type traceIDHandler struct {
	inner slog.Handler
}

func (h *traceIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *traceIDHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		r.AddAttrs(slog.String("trace_id", sc.TraceID().String()))
	}
	return h.inner.Handle(ctx, r)
}

func (h *traceIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceIDHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *traceIDHandler) WithGroup(name string) slog.Handler {
	return &traceIDHandler{inner: h.inner.WithGroup(name)}
}

// multiHandler fans out log records to multiple slog.Handler implementations.
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

// New creates a Logger that writes to both stdout and a rotating log file.
// When logProvider is non-nil, log records are also exported via the OTel
// log bridge (FR-005, FR-008).
func New(cfg Config, logProvider otellog.LoggerProvider) (*Logger, error) {
	rotator, err := NewRotatingWriter(cfg.Dir, cfg.Prefix, cfg.MaxBytes, cfg.MaxFiles)
	if err != nil {
		return nil, err
	}

	multi := io.MultiWriter(os.Stdout, rotator)

	textHandler := slog.NewTextHandler(multi, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Wrap text handler to inject trace_id on every record.
	traceHandler := &traceIDHandler{inner: textHandler}

	// When an OTel log provider is configured, fan out to both the text
	// handler and the OTel bridge handler.
	var rootHandler slog.Handler = traceHandler
	if logProvider != nil {
		otelHandler := otelslog.NewHandler("gatheryourdeals", otelslog.WithLoggerProvider(logProvider))
		rootHandler = &multiHandler{handlers: []slog.Handler{traceHandler, otelHandler}}
	}

	l := &Logger{
		Logger:  slog.New(rootHandler),
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
