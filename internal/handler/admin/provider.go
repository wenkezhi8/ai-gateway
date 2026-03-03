package admin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ai-gateway/internal/limiter"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"

	"github.com/gin-gonic/gin"
)

// ProviderHandler handles provider management requests.
type ProviderHandler struct {
	registry   *provider.Registry
	manager    *limiter.AccountManager
	router     *routing.SmartRouter
	configPath string
}

// NewProviderHandler creates a new provider handler.
func NewProviderHandler(registry *provider.Registry, manager *limiter.AccountManager, router *routing.SmartRouter, configPath string) *ProviderHandler {
	if strings.TrimSpace(configPath) == "" {
		configPath = defaultRuntimeConfigPath
	}
	return &ProviderHandler{
		registry:   registry,
		manager:    manager,
		router:     router,
		configPath: configPath,
	}
}

func normalizeProviderKey(input string) string {
	return strings.TrimSpace(strings.ToLower(input))
}

func sameProvider(left, right string) bool {
	return normalizeProviderKey(left) != "" && normalizeProviderKey(left) == normalizeProviderKey(right)
}

func (h *ProviderHandler) removeProviderFromConfig(providerName string) (bool, error) {
	root, err := loadConfigMap(h.configPath)
	if err != nil {
		return false, err
	}

	rawProviders, ok := root["providers"]
	if !ok {
		return false, nil
	}

	providers, ok := rawProviders.([]any)
	if !ok {
		return false, nil
	}

	changed := false
	filtered := make([]any, 0, len(providers))
	for _, item := range providers {
		providerConfig, castOK := item.(map[string]any)
		if !castOK {
			filtered = append(filtered, item)
			continue
		}
		name, _ := providerConfig["name"].(string)
		if sameProvider(name, providerName) {
			changed = true
			continue
		}
		filtered = append(filtered, item)
	}

	if !changed {
		return false, nil
	}

	root["providers"] = filtered
	if err := writeConfigMapAtomic(h.configPath, root); err != nil {
		return false, err
	}

	return true, nil
}

func (h *ProviderHandler) getAccountCount(providerName string) int {
	if h.manager == nil {
		return 0
	}
	accounts := h.manager.GetAllAccounts()
	count := 0
	for _, acc := range accounts {
		providerType := acc.ProviderType
		if providerType == "" {
			providerType = acc.Provider
		}
		if providerType == providerName || acc.Provider == providerName {
			count++
		}
	}
	return count
}

func (h *ProviderHandler) checkProviderHealth(ctx context.Context, p provider.Provider) bool {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return p.ValidateKey(checkCtx)
}

// GET /api/admin/providers.
func (h *ProviderHandler) ListProviders(c *gin.Context) {
	providers := h.registry.List()

	response := make([]ProviderResponse, 0, len(providers))
	for _, p := range providers {
		healthy := h.checkProviderHealth(c.Request.Context(), p)
		providerResp := ProviderResponse{
			Name:         p.Name(),
			Models:       p.Models(),
			Enabled:      p.IsEnabled(),
			Healthy:      healthy,
			AccountCount: h.getAccountCount(p.Name()),
			LastCheck:    time.Now(),
		}

		// Get base URL if available
		if baseProvider, ok := p.(interface{ BaseURL() string }); ok {
			providerResp.BaseURL = baseProvider.BaseURL()
		}

		response = append(response, providerResp)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GET /api/admin/providers/:id.
func (h *ProviderHandler) GetProvider(c *gin.Context) {
	providerName := c.Param("id")

	p, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Provider not found",
			},
		})
		return
	}

	response := ProviderResponse{
		Name:         p.Name(),
		Models:       p.Models(),
		Enabled:      p.IsEnabled(),
		Healthy:      h.checkProviderHealth(c.Request.Context(), p),
		AccountCount: h.getAccountCount(p.Name()),
		LastCheck:    time.Now(),
	}

	if baseProvider, ok := p.(interface{ BaseURL() string }); ok {
		response.BaseURL = baseProvider.BaseURL()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// POST /api/admin/providers.
func (h *ProviderHandler) CreateProvider(c *gin.Context) {
	var req ProviderRequest
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

	// Convert to provider config
	config := &provider.ProviderConfig{
		Name:    req.Name,
		APIKey:  req.APIKey,
		BaseURL: req.BaseURL,
		Models:  req.Models,
		Enabled: req.Enabled,
		Extra:   req.Extra,
	}

	// Create and register provider
	p, err := h.registry.CreateAndRegister(config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "create_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"name":    p.Name(),
			"message": "Provider created successfully",
		},
	})
}

// PUT /api/admin/providers/:id.
func (h *ProviderHandler) UpdateProvider(c *gin.Context) {
	providerName := c.Param("id")

	var req ProviderRequest
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

	p, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Provider not found",
			},
		})
		return
	}

	// Update enabled status
	p.SetEnabled(req.Enabled)

	// Note: Other fields like APIKey, BaseURL require provider-specific implementation

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":    providerName,
			"message": "Provider updated successfully",
		},
	})
}

// DELETE /api/admin/providers/:id.
func (h *ProviderHandler) DeleteProvider(c *gin.Context) {
	providerName := c.Param("id")
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": "Provider ID is required",
			},
		})
		return
	}

	removedConfig, err := h.removeProviderFromConfig(providerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "config_update_failed",
				"message": err.Error(),
			},
		})
		return
	}

	removedRegistry := false
	if _, ok := h.registry.Get(providerName); ok {
		h.registry.Remove(providerName)
		removedRegistry = true
	}

	removedAccounts := 0
	if h.manager != nil {
		accounts := h.manager.GetAllAccounts()
		for _, account := range accounts {
			providerType := account.ProviderType
			if providerType == "" {
				providerType = account.Provider
			}
			if !sameProvider(account.Provider, providerName) && !sameProvider(providerType, providerName) {
				continue
			}
			if removeErr := h.manager.RemoveAccount(account.ID); removeErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "delete_account_failed",
						"message": removeErr.Error(),
					},
				})
				return
			}
			removedAccounts++
		}
		if removedAccounts > 0 {
			if err := saveAccountsToFile(h.manager.GetAllAccounts()); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "persist_accounts_failed",
						"message": err.Error(),
					},
				})
				return
			}
		}
	}

	removedModels := 0
	removedDefaults := false
	if h.router != nil {
		removedModels, removedDefaults = h.router.RemoveProviderData(providerName)
	}

	if !removedRegistry && removedAccounts == 0 && removedModels == 0 && !removedDefaults && !removedConfig {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Provider not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":             providerName,
			"message":          "Provider deleted successfully",
			"removed_registry": removedRegistry,
			"removed_accounts": removedAccounts,
			"removed_models":   removedModels,
			"removed_defaults": removedDefaults,
			"updated_config":   removedConfig,
		},
	})
}

// POST /api/admin/providers/:id/test.
func (h *ProviderHandler) TestProvider(c *gin.Context) {
	providerName := c.Param("id")

	p, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Provider not found",
			},
		})
		return
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	valid := p.ValidateKey(ctx)
	responseTime := time.Since(startTime).Milliseconds()

	result := ProviderTestResult{
		Success:      valid,
		ResponseTime: responseTime,
		Timestamp:    time.Now(),
	}

	if valid {
		result.Message = "Provider connection successful"
	} else {
		result.Message = "Provider connection failed - invalid API key or unreachable"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// POST /api/admin/providers/:id/enable.
func (h *ProviderHandler) EnableProvider(c *gin.Context) {
	h.setProviderEnabled(c, true)
}

// POST /api/admin/providers/:id/disable.
func (h *ProviderHandler) DisableProvider(c *gin.Context) {
	h.setProviderEnabled(c, false)
}

func (h *ProviderHandler) setProviderEnabled(c *gin.Context, enabled bool) {
	providerName := c.Param("id")

	p, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Provider not found",
			},
		})
		return
	}

	p.SetEnabled(enabled)

	message := "Provider disabled successfully"
	if enabled {
		message = "Provider enabled successfully"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":    providerName,
			"enabled": enabled,
			"message": message,
		},
	})
}

// GET /api/admin/providers/:id/models.
func (h *ProviderHandler) GetProviderModels(c *gin.Context) {
	providerName := c.Param("id")

	p, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Provider not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    p.Models(),
	})
}
