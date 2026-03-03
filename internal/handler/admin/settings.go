package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"ai-gateway/internal/constants"

	"github.com/gin-gonic/gin"
)

type UISettingsRouting struct {
	AutoSaveEnabled bool   `json:"auto_save_enabled"`
	LastSavedAt     string `json:"last_saved_at"`
}

type UISettingsModelManagement struct {
	LastSavedAt string `json:"last_saved_at"`
}

type UISettings struct {
	Routing         UISettingsRouting         `json:"routing"`
	ModelManagement UISettingsModelManagement `json:"model_management"`
	Settings        map[string]any            `json:"settings"`
}

type updateUISettingsRequest struct {
	Routing         *UISettingsRouting         `json:"routing"`
	ModelManagement *UISettingsModelManagement `json:"model_management"`
	Settings        map[string]any             `json:"settings"`
}

type SettingsHandler struct {
	mu       sync.Mutex
	filePath string
	data     UISettings
}

func NewSettingsHandler(filePath string) *SettingsHandler {
	if filePath == "" {
		filePath = constants.UISettingsFilePath
	}
	handler := &SettingsHandler{
		filePath: filePath,
		data: UISettings{
			Settings: map[string]any{},
		},
	}
	handler.ensureDefaultsLocked()
	return handler
}

func (h *SettingsHandler) ensureDefaultsLocked() {
	if h.data.Settings == nil {
		h.data.Settings = map[string]any{}
	}
}

func (h *SettingsHandler) loadLocked() error {
	data, err := os.ReadFile(h.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			h.data = UISettings{
				Settings: map[string]any{},
			}
			h.ensureDefaultsLocked()
			return nil
		}
		return err
	}

	if len(data) == 0 {
		h.data = UISettings{
			Settings: map[string]any{},
		}
		h.ensureDefaultsLocked()
		return nil
	}

	var decoded UISettings
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	h.data = decoded
	h.ensureDefaultsLocked()
	return nil
}

func (h *SettingsHandler) saveLocked() error {
	h.ensureDefaultsLocked()
	raw, err := json.MarshalIndent(h.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(h.filePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(h.filePath, raw, 0640)
}

// GetUISettings handles GET /api/admin/settings/ui.
func (h *SettingsHandler) GetUISettings(c *gin.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.loadLocked(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "load_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.data,
	})
}

func (h *SettingsHandler) GetSettingsDefaults(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    constants.SettingsDefaults,
	})
}

// UpdateUISettings handles PUT /api/admin/settings/ui.
func (h *SettingsHandler) UpdateUISettings(c *gin.Context) {
	var req updateUISettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.loadLocked(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "load_failed",
				"message": err.Error(),
			},
		})
		return
	}

	if req.Routing != nil {
		h.data.Routing = *req.Routing
	}
	if req.ModelManagement != nil {
		h.data.ModelManagement = *req.ModelManagement
	}
	if req.Settings != nil {
		h.data.Settings = req.Settings
	}

	if err := h.saveLocked(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "save_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.data,
	})
}
