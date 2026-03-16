package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	lognoop "go.opentelemetry.io/otel/log/noop"
	oteltrace "go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	defaultServiceName    = "gatheryourdeals"
	shutdownTimeout       = 10 * time.Second
)

// Provider holds the OTel SDK providers and manages their lifecycle.
// When OTel export is disabled (no endpoint configured) the providers are
// noop implementations that impose zero overhead (FR-003).
type Provider struct {
	tracerProvider oteltrace.TracerProvider
	loggerProvider otellog.LoggerProvider
	shutdownFuncs  []func(context.Context) error
}

// TracerProvider returns the trace.TracerProvider to pass to instrumentation.
func (p *Provider) TracerProvider() oteltrace.TracerProvider {
	return p.tracerProvider
}

// LoggerProvider returns the log.LoggerProvider to pass to the slog bridge.
func (p *Provider) LoggerProvider() otellog.LoggerProvider {
	return p.loggerProvider
}

// Shutdown flushes all pending telemetry and releases resources.
// Uses a 10-second timeout (FR-017). Safe to call multiple times.
func (p *Provider) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	var errs []error
	for _, fn := range p.shutdownFuncs {
		if err := fn(shutdownCtx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Setup initialises the OTel SDK from standard environment variables.
//
// If OTEL_EXPORTER_OTLP_ENDPOINT is not set, noop providers are returned and
// the service operates identically to before this feature was added (FR-003).
//
// When the endpoint is set, OTLP/HTTP exporters are created for both traces
// and logs. The resource is built from OTEL_SERVICE_NAME and
// OTEL_RESOURCE_ATTRIBUTES (FR-008, FR-010).
func Setup(ctx context.Context) (*Provider, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return &Provider{
			tracerProvider: tracenoop.NewTracerProvider(),
			loggerProvider: lognoop.NewLoggerProvider(),
		}, nil
	}

	res, err := buildResource(ctx)
	if err != nil {
		// Non-fatal: use default resource if detectors fail.
		res = resource.Default()
	}

	tp, err := buildTracerProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	lp, err := buildLoggerProvider(ctx, res)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, err
	}

	// Register as global providers so instrumentation libraries (otelgin,
	// otelsql) can pick them up without needing explicit provider injection.
	otel.SetTracerProvider(tp)
	global.SetLoggerProvider(lp)

	return &Provider{
		tracerProvider: tp,
		loggerProvider: lp,
		shutdownFuncs: []func(context.Context) error{
			tp.Shutdown,
			lp.Shutdown,
		},
	}, nil
}

// buildResource creates the OTel resource using environment variables.
// OTEL_SERVICE_NAME overrides the default service name (FR-010).
// OTEL_RESOURCE_ATTRIBUTES adds arbitrary deployment context (FR-008).
func buildResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		// Default service name — overridden if OTEL_SERVICE_NAME is set.
		resource.WithAttributes(semconv.ServiceName(defaultServiceName)),
		// Reads OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES from env.
		// Takes precedence over the default above.
		resource.WithFromEnv(),
	)
}

func buildTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	return tp, nil
}

func buildLoggerProvider(ctx context.Context, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	exp, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create log exporter: %w", err)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
		sdklog.WithResource(res),
	)
	return lp, nil
}
