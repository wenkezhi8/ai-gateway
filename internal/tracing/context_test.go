package tracing

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	if id1 == id2 {
		t.Error("GenerateRequestID() should generate unique IDs")
	}

	if _, err := uuid.Parse(id1); err != nil {
		t.Errorf("GenerateRequestID() returned invalid UUID: %v", err)
	}
}

func TestGetRequestIDFromContext(t *testing.T) {
	ctx := context.Background()

	// Without request ID
	id := GetRequestIDFromContext(ctx)
	if id != "" {
		t.Errorf("GetRequestIDFromContext() should return empty string for context without request ID, got %s", id)
	}

	// With request ID
	expectedID := "test-request-id"
	ctx = SetRequestIDToContext(ctx, expectedID)
	id = GetRequestIDFromContext(ctx)
	if id != expectedID {
		t.Errorf("GetRequestIDFromContext() = %s, want %s", id, expectedID)
	}
}
