package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ApiKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Key         string     `json:"key"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	Enabled     bool       `json:"enabled"`
}

type ApiKeyCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type ApiKeyUpdateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
}

type ApiKeyHandler struct {
	mu       sync.RWMutex
	store    map[string]*ApiKey
	dataFile string
}

func NewApiKeyHandler() *ApiKeyHandler {
	h := &ApiKeyHandler{
		store:    make(map[string]*ApiKey),
		dataFile: "./data/api_keys.json",
	}
	h.loadFromFile()
	return h
}

func (h *ApiKeyHandler) loadFromFile() {
	data, err := os.ReadFile(h.dataFile)
	if err != nil {
		return
	}

	var keys []*ApiKey
	if err := json.Unmarshal(data, &keys); err != nil {
		return
	}

	for _, k := range keys {
		h.store[k.ID] = k
	}
}

func (h *ApiKeyHandler) saveToFile() error {
	keys := make([]*ApiKey, 0, len(h.store))
	for _, k := range h.store {
		keys = append(keys, k)
	}

	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(h.dataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(h.dataFile, data, 0644)
}

func generateApiKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "sk-" + hex.EncodeToString(b)
}

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func maskKey(key string) string {
	if len(key) <= 12 {
		return "****"
	}
	return key[:8] + "..." + key[len(key)-4:]
}

func (h *ApiKeyHandler) ListApiKeys(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	keys := make([]*ApiKey, 0, len(h.store))
	for _, k := range h.store {
		keyCopy := *k
		keys = append(keys, &keyCopy)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    keys,
	})
}

// FindNameByKey returns the API key name for display purposes.
func (h *ApiKeyHandler) FindNameByKey(apiKey string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, k := range h.store {
		if k.Key == apiKey {
			return k.Name
		}
	}
	return ""
}

func (h *ApiKeyHandler) GetApiKey(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	id := c.Param("id")
	key, exists := h.store[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   gin.H{"code": "not_found", "message": "API Key not found"},
		})
		return
	}

	keyCopy := *key
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &keyCopy,
	})
}

func (h *ApiKeyHandler) CreateApiKey(c *gin.Context) {
	var req ApiKeyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "invalid_request", "message": err.Error()},
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	key := &ApiKey{
		ID:          "key_" + now.Format("20060102150405") + "_" + randomString(4),
		Name:        req.Name,
		Key:         generateApiKey(),
		Description: req.Description,
		CreatedAt:   now,
		Enabled:     true,
	}

	h.store[key.ID] = key
	h.saveToFile()

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":         key.ID,
			"key":        key.Key,
			"name":       key.Name,
			"created_at": key.CreatedAt,
		},
	})
}

func (h *ApiKeyHandler) UpdateApiKey(c *gin.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := c.Param("id")

	key, exists := h.store[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   gin.H{"code": "not_found", "message": "API Key not found"},
		})
		return
	}

	var req ApiKeyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "invalid_request", "message": err.Error()},
		})
		return
	}

	if req.Name != "" {
		key.Name = req.Name
	}
	if req.Description != "" {
		key.Description = req.Description
	}
	if req.Enabled != nil {
		key.Enabled = *req.Enabled
	}

	h.saveToFile()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":      key.ID,
			"name":    key.Name,
			"enabled": key.Enabled,
		},
	})
}

func (h *ApiKeyHandler) DeleteApiKey(c *gin.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := c.Param("id")

	if _, exists := h.store[id]; !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   gin.H{"code": "not_found", "message": "API Key not found"},
		})
		return
	}

	delete(h.store, id)
	h.saveToFile()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"id": id},
	})
}

func (h *ApiKeyHandler) ValidateApiKey(apiKey string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, k := range h.store {
		if k.Key == apiKey && k.Enabled {
			now := time.Now()
			k.LastUsed = &now
			h.saveToFile()
			return true
		}
	}
	return false
}
