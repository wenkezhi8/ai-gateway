package google

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key", "")
	require.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, defaultBaseURL, client.baseURL)
}

func TestBuildGeneratePath(t *testing.T) {
	client := NewClient("test-api-key", "")
	path := client.buildGeneratePath("gemini-3.1-pro-preview")
	assert.Equal(t, "/models/gemini-3.1-pro-preview:generateContent?key=test-api-key", path)
}

func TestBuildStreamPath(t *testing.T) {
	client := NewClient("test-api-key", "")
	path := client.buildStreamPath("gemini-3.1-pro-preview")
	assert.Equal(t, "/models/gemini-3.1-pro-preview:streamGenerateContent?alt=sse&key=test-api-key", path)
}
