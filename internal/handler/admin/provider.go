package admin

import (
	"ai-gateway/internal/provider"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ProviderHandler handles provider management requests
type ProviderHandler struct {
	registry *provider.Registry
}

// NewProviderHandler creates a new provider handler
func NewProviderHandler(registry *provider.Registry) *ProviderHandler {
	return &ProviderHandler{
		registry: registry,
	}
}

// ListProviders returns all providers
// GET /api/admin/providers
func (h *ProviderHandler) ListProviders(c *gin.Context) {
	providers := h.registry.List()

	response := make([]ProviderResponse, 0, len(providers))
	for _, p := range providers {
		providerResp := ProviderResponse{
			Name:         p.Name(),
			Models:       p.Models(),
			Enabled:      p.IsEnabled(),
			Healthy:      true, // TODO: implement health check
			AccountCount: 0,    // TODO: count accounts per provider
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

// GetProvider returns a single provider by name
// GET /api/admin/providers/:id
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
		Healthy:      true,
		AccountCount: 0,
	}

	if baseProvider, ok := p.(interface{ BaseURL() string }); ok {
		response.BaseURL = baseProvider.BaseURL()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// CreateProvider adds a new provider
// POST /api/admin/providers
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

// UpdateProvider updates provider configuration
// PUT /api/admin/providers/:id
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

// DeleteProvider removes a provider
// DELETE /api/admin/providers/:id
func (h *ProviderHandler) DeleteProvider(c *gin.Context) {
	providerName := c.Param("id")

	// Check if provider exists
	if _, ok := h.registry.Get(providerName); !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Provider not found",
			},
		})
		return
	}

	h.registry.Remove(providerName)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":    providerName,
			"message": "Provider deleted successfully",
		},
	})
}

// TestProvider tests provider connectivity
// POST /api/admin/providers/:id/test
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

// EnableProvider enables a provider
// POST /api/admin/providers/:id/enable
func (h *ProviderHandler) EnableProvider(c *gin.Context) {
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

	p.SetEnabled(true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":    providerName,
			"enabled": true,
			"message": "Provider enabled successfully",
		},
	})
}

// DisableProvider disables a provider
// POST /api/admin/providers/:id/disable
func (h *ProviderHandler) DisableProvider(c *gin.Context) {
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

	p.SetEnabled(false)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":    providerName,
			"enabled": false,
			"message": "Provider disabled successfully",
		},
	})
}

// GetProviderModels returns models supported by a provider
// GET /api/admin/providers/:id/models
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
