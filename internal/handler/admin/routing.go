package admin

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// Strategy types.
const (
	StrategyFailover      = "failover"
	StrategyRoundRobin    = "roundrobin"
	StrategyWeighted      = "weighted"
	StrategyCostOptimized = "cost"
)

// RoutingHandler handles routing strategy management.
type RoutingHandler struct {
	config *RoutingConfig
	mu     sync.RWMutex
}

// NewRoutingHandler creates a new routing handler.
func NewRoutingHandler() *RoutingHandler {
	return &RoutingHandler{
		config: &RoutingConfig{
			DefaultStrategy: "roundrobin",
			ModelStrategies: make(map[string]string),
			ProviderWeights: make(map[string]int),
			FailoverConfig: &FailoverConfig{
				MaxRetries:       3,
				RetryDelayMs:     100,
				HealthCheckSec:   30,
				CircuitBreaker:   true,
				FailureThreshold: 5,
			},
		},
	}
}

// GET /api/admin/routing.
func (h *RoutingHandler) GetRouting(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.config,
	})
}

// PUT /api/admin/routing.
func (h *RoutingHandler) UpdateRouting(c *gin.Context) {
	var req RoutingConfig
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

	// Validate strategy
	validStrategies := map[string]bool{
		StrategyFailover:      true,
		StrategyRoundRobin:    true,
		StrategyWeighted:      true,
		StrategyCostOptimized: true,
	}

	if !validStrategies[req.DefaultStrategy] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_strategy",
				"message": "Invalid routing strategy. Valid options: failover, roundrobin, weighted, cost",
			},
		})
		return
	}

	// Update config
	h.config = &req

	// Initialize maps if nil
	if h.config.ModelStrategies == nil {
		h.config.ModelStrategies = make(map[string]string)
	}
	if h.config.ProviderWeights == nil {
		h.config.ProviderWeights = make(map[string]int)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Routing configuration updated successfully",
		},
	})
}

// GET /api/admin/routing/strategies.
func (h *RoutingHandler) GetStrategies(c *gin.Context) {
	strategies := []gin.H{
		{
			"name":        "failover",
			"description": "Use primary provider, fail over to backups on error",
			"use_case":    "High availability with preferred provider",
		},
		{
			"name":        "roundrobin",
			"description": "Distribute requests evenly across all providers",
			"use_case":    "Load balancing across equivalent providers",
		},
		{
			"name":        "weighted",
			"description": "Distribute requests based on provider weights",
			"use_case":    "Traffic splitting with custom ratios",
		},
		{
			"name":        "cost",
			"description": "Route to provider with lowest cost for the model",
			"use_case":    "Cost optimization across providers",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    strategies,
	})
}

// PUT /api/admin/routing/models/:model/strategy.
func (h *RoutingHandler) SetModelStrategy(c *gin.Context) {
	model := c.Param("model")

	var req struct {
		Strategy string `json:"strategy" binding:"required"`
	}
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

	if h.config.ModelStrategies == nil {
		h.config.ModelStrategies = make(map[string]string)
	}
	h.config.ModelStrategies[model] = req.Strategy

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"model":    model,
			"strategy": req.Strategy,
			"message":  "Model strategy updated successfully",
		},
	})
}

// PUT /api/admin/routing/providers/:provider/weight.
func (h *RoutingHandler) SetProviderWeight(c *gin.Context) {
	providerName := c.Param("provider")

	var req struct {
		Weight int `json:"weight" binding:"required,min=0,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": "Weight must be between 0 and 100",
			},
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.config.ProviderWeights == nil {
		h.config.ProviderWeights = make(map[string]int)
	}
	h.config.ProviderWeights[providerName] = req.Weight

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"provider": providerName,
			"weight":   req.Weight,
			"message":  "Provider weight updated successfully",
		},
	})
}

// POST /api/admin/routing/reset.
func (h *RoutingHandler) ResetRouting(c *gin.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.config = &RoutingConfig{
		DefaultStrategy: "roundrobin",
		ModelStrategies: make(map[string]string),
		ProviderWeights: make(map[string]int),
		FailoverConfig: &FailoverConfig{
			MaxRetries:       3,
			RetryDelayMs:     100,
			HealthCheckSec:   30,
			CircuitBreaker:   true,
			FailureThreshold: 5,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Routing configuration reset to defaults",
		},
	})
}
