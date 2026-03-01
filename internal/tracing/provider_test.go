package tracing

import (
	"context"
	"testing"
)

func TestNewTracerProvider(t *testing.T) {
	tp, err := NewTracerProvider()
	if err != nil {
		t.Fatalf("NewTracerProvider() error = %v", err)
	}
	if tp == nil {
		t.Fatal("NewTracerProvider() returned nil")
	}
	if shutdownErr := tp.Shutdown(context.Background()); shutdownErr != nil {
		t.Errorf("Shutdown() error = %v", shutdownErr)
	}
}

func TestGetTracer(t *testing.T) {
	tp, err := NewTracerProvider()
	if err != nil {
		t.Fatalf("NewTracerProvider() error = %v", err)
	}
	tracer := tp.GetTracer("test-tracer")
	if tracer == nil {
		t.Fatal("GetTracer() returned nil")
	}

	if shutdownErr := tp.Shutdown(context.Background()); shutdownErr != nil {
		t.Errorf("Shutdown() error = %v", shutdownErr)
	}
}
