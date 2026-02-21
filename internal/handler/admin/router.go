package admin

import (
	"ai-gateway/internal/routing"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RouterHandler handles smart router configuration requests
type RouterHandler struct {
	router *routing.SmartRouter
}

// NewRouterHandler creates a new router handler
func NewRouterHandler(router *routing.SmartRouter) *RouterHandler {
	return &RouterHandler{
		router: router,
	}
}

// RouterConfigResponse represents the router configuration response
type RouterConfigResponse struct {
	UseAutoMode     bool             `json:"use_auto_mode"`
	DefaultStrategy string           `json:"default_strategy"`
	DefaultModel    string           `json:"default_model"`
	Strategies      []StrategyOption `json:"strategies"`
}

// StrategyOption represents a strategy option
type StrategyOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ModelScoreResponse represents model score in response
type ModelScoreResponse struct {
	Model          string  `json:"model"`
	Provider       string  `json:"provider"`
	QualityScore   int     `json:"quality_score"`
	SpeedScore     int     `json:"speed_score"`
	CostScore      int     `json:"cost_score"`
	CompositeScore float64 `json:"composite_score"`
	Enabled        bool    `json:"enabled"`
}

// UpdateRouterConfigRequest represents update request
type UpdateRouterConfigRequest struct {
	UseAutoMode     *bool   `json:"use_auto_mode,omitempty"`
	DefaultStrategy *string `json:"default_strategy,omitempty"`
	DefaultModel    *string `json:"default_model,omitempty"`
}

// UpdateModelScoreRequest represents model score update request
type UpdateModelScoreRequest struct {
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	QualityScore int    `json:"quality_score"`
	SpeedScore   int    `json:"speed_score"`
	CostScore    int    `json:"cost_score"`
	Enabled      bool   `json:"enabled"`
}

// GetRouterConfig returns current router configuration
// GET /api/admin/router/config
func (h *RouterHandler) GetRouterConfig(c *gin.Context) {
	config := h.router.GetConfig()

	strategies := []StrategyOption{
		{Value: "auto", Label: "智能平衡", Description: "综合效果 + 速度 + 成本，自动选择最优模型"},
		{Value: "quality", Label: "效果优先", Description: "优先选择效果最好的模型"},
		{Value: "speed", Label: "速度优先", Description: "优先选择响应最快的模型"},
		{Value: "cost", Label: "成本优先", Description: "优先选择成本最低的模型"},
		{Value: "custom", Label: "自定义规则", Description: "根据任务类型自动选择模型"},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": RouterConfigResponse{
			UseAutoMode:     config.UseAutoMode,
			DefaultStrategy: string(config.DefaultStrategy),
			DefaultModel:    config.DefaultModel,
			Strategies:      strategies,
		},
	})
}

// UpdateRouterConfig updates router configuration
// PUT /api/admin/router/config
func (h *RouterHandler) UpdateRouterConfig(c *gin.Context) {
	var req UpdateRouterConfigRequest
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

	if req.UseAutoMode != nil {
		h.router.SetUseAutoMode(*req.UseAutoMode)
	}
	if req.DefaultStrategy != nil {
		h.router.SetStrategy(routing.StrategyType(*req.DefaultStrategy))
	}
	if req.DefaultModel != nil {
		h.router.SetDefaultModel(*req.DefaultModel)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Router configuration updated",
	})
}

// GetModelScores returns all model scores
// GET /api/admin/router/models
func (h *RouterHandler) GetModelScores(c *gin.Context) {
	scores := h.router.GetAllModelScores()
	config := h.router.GetConfig()

	var response []ModelScoreResponse
	for model, score := range scores {
		composite := float64(score.QualityScore)*0.4 + float64(score.SpeedScore)*0.35 + float64(score.CostScore)*0.25
		if config.DefaultStrategy == routing.StrategyQuality {
			composite = float64(score.QualityScore)
		} else if config.DefaultStrategy == routing.StrategySpeed {
			composite = float64(score.SpeedScore)
		} else if config.DefaultStrategy == routing.StrategyCost {
			composite = float64(score.CostScore)
		}

		response = append(response, ModelScoreResponse{
			Model:          model,
			Provider:       score.Provider,
			QualityScore:   score.QualityScore,
			SpeedScore:     score.SpeedScore,
			CostScore:      score.CostScore,
			CompositeScore: composite,
			Enabled:        score.Enabled,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// UpdateModelScore updates score for a specific model
// PUT /api/admin/router/models/:model
func (h *RouterHandler) UpdateModelScore(c *gin.Context) {
	model := c.Param("model")

	var req UpdateModelScoreRequest
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

	score := &routing.ModelScore{
		Model:        model,
		Provider:     req.Provider,
		QualityScore: req.QualityScore,
		SpeedScore:   req.SpeedScore,
		CostScore:    req.CostScore,
		Enabled:      req.Enabled,
	}

	h.router.UpdateModelScore(model, score)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Model score updated",
	})
}

// DeleteModelScore deletes a model score
// DELETE /api/admin/router/models/:model
func (h *RouterHandler) DeleteModelScore(c *gin.Context) {
	model := c.Param("model")

	h.router.DeleteModelScore(model)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Model score deleted",
	})
}

// GetAvailableModels returns list of enabled models
// GET /api/admin/router/available-models
func (h *RouterHandler) GetAvailableModels(c *gin.Context) {
	models := h.router.GetAvailableModels()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    models,
	})
}

// GetTopModels returns top N models for current strategy
// GET /api/admin/router/top-models
func (h *RouterHandler) GetTopModels(c *gin.Context) {
	config := h.router.GetConfig()
	topModels := h.router.GetTopModels(config.DefaultStrategy, 5)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    topModels,
	})
}

// SelectModelRequest represents model selection request
type SelectModelRequest struct {
	RequestedModel string `json:"requested_model"`
	Prompt         string `json:"prompt"`
}

// SelectModel selects best model based on configuration
// POST /api/admin/router/select
func (h *RouterHandler) SelectModel(c *gin.Context) {
	var req SelectModelRequest
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

	selectedModel := h.router.SelectModel(req.RequestedModel, req.Prompt, nil)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"selected_model": selectedModel,
		},
	})
}

// GetProviderDefaults returns default models for all providers
// GET /api/admin/router/provider-defaults
func (h *RouterHandler) GetProviderDefaults(c *gin.Context) {
	defaults := h.router.GetProviderDefaults()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    defaults,
	})
}

// UpdateProviderDefaults updates default models for providers
// PUT /api/admin/router/provider-defaults
func (h *RouterHandler) UpdateProviderDefaults(c *gin.Context) {
	var req map[string]string
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

	h.router.SetProviderDefaults(req)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Provider defaults updated",
	})
}
