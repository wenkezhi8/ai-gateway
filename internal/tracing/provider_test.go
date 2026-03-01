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
	defer tp.Shutdown(context.Background())
}

func TestGetTracer(t *testing.T) {
	tp, _ := NewTracerProvider()
	defer tp.Shutdown(context.Background())

	tracer := tp.GetTracer("test-tracer")
	if tracer == nil {
		t.Fatal("GetTracer() returned nil")
	}
}
