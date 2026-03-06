package admin

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyHandler_LoadFromFile_ShouldSupportLegacyLastUsedAt(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "api_keys.json")
	legacyLastUsed := "2026-03-06T01:02:03Z"

	err := os.WriteFile(dataFile, []byte(`[
  {
    "id": "legacy-key-1",
    "name": "legacy",
    "key": "sk-legacy",
    "created_at": "2026-03-05T01:02:03Z",
    "last_used_at": "`+legacyLastUsed+`",
    "enabled": true
  }
]`), 0o600)
	require.NoError(t, err)

	h := &APIKeyHandler{
		store:    make(map[string]*APIKey),
		dataFile: dataFile,
	}

	h.loadFromFile()

	loaded, exists := h.store["legacy-key-1"]
	require.True(t, exists)
	require.NotNil(t, loaded.LastUsed)
	assert.Equal(t, legacyLastUsed, loaded.LastUsed.UTC().Format(time.RFC3339))
}
