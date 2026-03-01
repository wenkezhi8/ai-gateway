package qdrant

import (
	"context"
	"testing"
)

func TestNewQdrantClient(t *testing.T) {
	client, err := NewQdrantClient("", "", "test-collection")
	if err != nil {
		t.Fatalf("Failed to create Qdrant client: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
}

func TestParseHTTPAddr(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "empty address",
			addr:    "",
			wantErr: false,
		},
		{
			name:    "localhost with port",
			addr:    "localhost:6334",
			wantErr: false,
		},
		{
			name:    "http:// address",
			addr:    "http://localhost:6334",
			wantErr: false,
		},
		{
			name:    "https:// address",
			addr:    "https://qdrant.example.com:6334",
			wantErr: false,
		},
		{name: "invalid port", addr: "localhost:abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, _, _, err := parseHTTPAddr(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHTTPAddr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && host == "" {
				t.Errorf("parseHTTPAddr() returned empty host")
			}
		})
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "empty key",
			key:  "",
			want: "none",
		},
		{
			name: "short key",
			key:  "abc123",
			want: "***",
		},
		{
			name: "long key",
			key:  "my-secret-api-key-12345",
			want: "my-s***2345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskAPIKey(tt.key)
			if got != tt.want {
				t.Errorf("maskAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Health(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := NewQdrantClient("", "", "test-collection")
	if err != nil {
		t.Fatalf("Failed to create Qdrant client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	err = client.Health(ctx)
	if err != nil {
		t.Logf("Health check failed (expected if Qdrant is not running): %v", err)
	}
}

func TestClient_GetCollections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := NewQdrantClient("", "", "test-collection")
	if err != nil {
		t.Fatalf("Failed to create Qdrant client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	collections, err := client.GetCollections(ctx)
	if err != nil {
		t.Logf("GetCollections failed (expected if Qdrant is not running): %v", err)
	} else {
		t.Logf("Found %d collections: %v", len(collections), collections)
	}
}
