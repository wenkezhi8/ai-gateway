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

type APIKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Key         string     `json:"key"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	Enabled     bool       `json:"enabled"`
}

type APIKeyCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type APIKeyUpdateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
}

type APIKeyHandler struct {
	mu       sync.RWMutex
	store    map[string]*APIKey
	dataFile string
}

func NewAPIKeyHandler() *APIKeyHandler {
	h := &APIKeyHandler{
		store:    make(map[string]*APIKey),
		dataFile: "./data/api_keys.json",
	}
	h.loadFromFile()
	return h
}

func (h *APIKeyHandler) loadFromFile() {
	data, err := os.ReadFile(h.dataFile)
	if err != nil {
		return
	}

	var keys []*APIKey
	if err := json.Unmarshal(data, &keys); err != nil {
		return
	}

	for _, k := range keys {
		h.store[k.ID] = k
	}
}

func (h *APIKeyHandler) saveToFile() error {
	keys := make([]*APIKey, 0, len(h.store))
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

	return os.WriteFile(h.dataFile, data, 0640)
}

func generateAPIKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk-" + hex.EncodeToString(b), nil
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}

func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	keys := make([]*APIKey, 0, len(h.store))
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
func (h *APIKeyHandler) FindNameByKey(apiKey string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, k := range h.store {
		if k.Key == apiKey {
			return k.Name
		}
	}
	return ""
}

func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
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

func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req APIKeyCreateRequest
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
	randomSuffix, err := randomString(4)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "generate_failed", "message": "Failed to generate key id"},
		})
		return
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "generate_failed", "message": "Failed to generate API key"},
		})
		return
	}

	key := &APIKey{
		ID:          "key_" + now.Format("20060102150405") + "_" + randomSuffix,
		Name:        req.Name,
		Key:         apiKey,
		Description: req.Description,
		CreatedAt:   now,
		Enabled:     true,
	}

	h.store[key.ID] = key
	if err := h.saveToFile(); err != nil {
		delete(h.store, key.ID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "persist_failed", "message": "Failed to persist API key"},
		})
		return
	}

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

func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
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

	var req APIKeyUpdateRequest
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

	if err := h.saveToFile(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "persist_failed", "message": "Failed to persist API key"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":      key.ID,
			"name":    key.Name,
			"enabled": key.Enabled,
		},
	})
}

func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
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

	key := h.store[id]
	delete(h.store, id)
	if err := h.saveToFile(); err != nil {
		h.store[id] = key
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "persist_failed", "message": "Failed to persist API key"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"id": id},
	})
}

func (h *APIKeyHandler) ValidateAPIKey(apiKey string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, k := range h.store {
		if k.Key == apiKey && k.Enabled {
			now := time.Now()
			k.LastUsed = &now
			if err := h.saveToFile(); err != nil {
				return false
			}
			return true
		}
	}
	return false
}
