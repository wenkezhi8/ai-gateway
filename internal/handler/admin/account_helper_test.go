package admin

import (
	"testing"
)

func TestGenerateAccountID(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantLen  int
	}{
		{
			name:     "with provider name",
			provider: "openai",
			wantLen:  len("openai-") + 19,
		},
		{
			name:     "with uppercase provider",
			provider: "DeepSeek",
			wantLen:  len("deepseek-") + 19,
		},
		{
			name:     "with spaces",
			provider: "Deep Seek",
			wantLen:  len("deep-seek-") + 19,
		},
		{
			name:     "empty provider",
			provider: "",
			wantLen:  len("account-") + 19,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateAccountID(tt.provider)
			if len(result) != tt.wantLen {
				t.Errorf("generateAccountID() len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestMapProviderToBackend(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{
			name:     "openai",
			provider: "openai",
			want:     "openai",
		},
		{
			name:     "deepseek",
			provider: "deepseek",
			want:     "openai",
		},
		{
			name:     "moonshot",
			provider: "moonshot",
			want:     "openai",
		},
		{
			name:     "qwen",
			provider: "qwen",
			want:     "openai",
		},
		{
			name:     "zhipu",
			provider: "zhipu",
			want:     "openai",
		},
		{
			name:     "baichuan",
			provider: "baichuan",
			want:     "openai",
		},
		{
			name:     "minimax",
			provider: "minimax",
			want:     "openai",
		},
		{
			name:     "volcengine",
			provider: "volcengine",
			want:     "openai",
		},
		{
			name:     "yi",
			provider: "yi",
			want:     "openai",
		},
		{
			name:     "azure-openai",
			provider: "azure-openai",
			want:     "openai",
		},
		{
			name:     "anthropic",
			provider: "anthropic",
			want:     "anthropic",
		},
		{
			name:     "unknown provider",
			provider: "unknown",
			want:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapProviderToBackend(tt.provider)
			if result != tt.want {
				t.Errorf("mapProviderToBackend() = %v, want %v", result, tt.want)
			}
		})
	}
}
