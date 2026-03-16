package telemetry_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gatheryourdeals/data/internal/telemetry"
)

func TestSetup_NoopWhenEndpointUnset(t *testing.T) {
	// Ensure the endpoint env var is not set.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	ctx := context.Background()
	prov, err := telemetry.Setup(ctx)
	if err != nil {
		t.Fatalf("Setup() error = %v, want nil", err)
	}
	if prov == nil {
		t.Fatal("Setup() returned nil provider")
	}
	if prov.TracerProvider() == nil {
		t.Error("TracerProvider() is nil")
	}
	if prov.LoggerProvider() == nil {
		t.Error("LoggerProvider() is nil")
	}
}

func TestSetup_ShutdownCompletesWithinTimeout(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	ctx := context.Background()
	prov, err := telemetry.Setup(ctx)
	if err != nil {
		t.Fatalf("Setup() error = %v, want nil", err)
	}

	// Shutdown should complete well within 10 seconds for noop providers.
	done := make(chan error, 1)
	go func() {
		done <- prov.Shutdown(context.Background())
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Shutdown() error = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Shutdown() did not complete within 5 seconds")
	}
}

func TestSetup_NoErrorOnValidEndpoint(t *testing.T) {
	// Set a reachable-looking endpoint; the exporter will fail lazily on first
	// export attempt, not at construction time. Setup() must succeed.
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prov, err := telemetry.Setup(ctx)
	if err != nil {
		t.Fatalf("Setup() error = %v, want nil", err)
	}

	// Shutdown with a short timeout — export will fail (no real backend) but
	// Shutdown must not block indefinitely.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	// Ignore export errors; we're only verifying it doesn't block.
	_ = prov.Shutdown(shutdownCtx)
}
