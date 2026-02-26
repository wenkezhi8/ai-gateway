package admin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/routing"

	"github.com/gin-gonic/gin"
)

// RouterHandler handles smart router configuration requests
type RouterHandler struct {
	router *routing.SmartRouter
	mu     sync.RWMutex
}

// NewRouterHandler creates a new router handler
func NewRouterHandler(router *routing.SmartRouter) *RouterHandler {
	h := &RouterHandler{
		router: router,
	}
	h.loadConfig()
	return h
}

// RouterConfigResponse represents the router configuration response
type RouterConfigResponse struct {
	UseAutoMode     string                   `json:"use_auto_mode"` // "auto", "default", "fixed", "latest"
	DefaultStrategy string                   `json:"default_strategy"`
	DefaultModel    string                   `json:"default_model"`
	Classifier      routing.ClassifierConfig `json:"classifier"`
	Strategies      []StrategyOption         `json:"strategies"`
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
	DisplayName    string  `json:"display_name,omitempty"`
	QualityScore   int     `json:"quality_score"`
	SpeedScore     int     `json:"speed_score"`
	CostScore      int     `json:"cost_score"`
	CompositeScore float64 `json:"composite_score"`
	Enabled        bool    `json:"enabled"`
}

// UpdateRouterConfigRequest represents update request
type UpdateRouterConfigRequest struct {
	UseAutoMode     json.RawMessage           `json:"use_auto_mode,omitempty"` // "auto", "default", "fixed", "latest" or bool
	DefaultStrategy *string                   `json:"default_strategy,omitempty"`
	DefaultModel    *string                   `json:"default_model,omitempty"`
	Classifier      *routing.ClassifierConfig `json:"classifier,omitempty"`
}

// UpdateModelScoreRequest represents model score update request
type UpdateModelScoreRequest struct {
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	DisplayName  string `json:"display_name,omitempty"`
	QualityScore int    `json:"quality_score"`
	SpeedScore   int    `json:"speed_score"`
	CostScore    int    `json:"cost_score"`
	Enabled      bool   `json:"enabled"`
}

// PersistedRouterConfig is the structure stored for UI routing mode selection
type PersistedRouterConfig struct {
	UseAutoMode     string                   `json:"use_auto_mode"`
	DefaultStrategy string                   `json:"default_strategy"`
	DefaultModel    string                   `json:"default_model"`
	Classifier      routing.ClassifierConfig `json:"classifier"`
}

const routerUIConfigFile = "data/router_ui_config.json"
const routerConfigFile = "data/router_config.json"

var persistedConfig *PersistedRouterConfig

func normalizeAutoMode(value string) string {
	switch value {
	case "auto", "default", "fixed", "latest":
		return value
	default:
		return "auto"
	}
}

func parseAutoModeJSON(raw json.RawMessage, fallback string) string {
	if len(raw) == 0 || string(raw) == "null" {
		return normalizeAutoMode(fallback)
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return normalizeAutoMode(s)
		}
		return normalizeAutoMode(fallback)
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		if b {
			return "auto"
		}
		return "fixed"
	}
	return normalizeAutoMode(fallback)
}

func mergeClassifierCandidateModels(activeModel string, groups ...[]string) []string {
	merged := make([]string, 0)
	seen := make(map[string]struct{})
	appendUnique := func(model string) {
		model = strings.TrimSpace(model)
		if model == "" {
			return
		}
		if _, ok := seen[model]; ok {
			return
		}
		seen[model] = struct{}{}
		merged = append(merged, model)
	}

	appendUnique(activeModel)
	for _, group := range groups {
		for _, model := range group {
			appendUnique(model)
		}
	}

	if len(merged) == 0 {
		def := routing.DefaultClassifierConfig()
		return append([]string{}, def.CandidateModels...)
	}
	return merged
}

func resolveClassifierModels(ctx context.Context, cfg routing.ClassifierConfig) []string {
	timeout := 2 * time.Second
	if cfg.TimeoutMs > 0 {
		candidate := time.Duration(cfg.TimeoutMs) * time.Millisecond
		if candidate > timeout {
			timeout = candidate
		}
	}
	if timeout > 5*time.Second {
		timeout = 5 * time.Second
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	models, err := routing.ListOllamaModels(ctxWithTimeout, cfg.BaseURL, timeout)
	if err != nil {
		return mergeClassifierCandidateModels(cfg.ActiveModel, cfg.CandidateModels)
	}

	return mergeClassifierCandidateModels(cfg.ActiveModel, models, cfg.CandidateModels)
}

func (h *RouterHandler) migrateLegacyRouterConfig() {
	data, err := os.ReadFile(routerConfigFile)
	if err != nil {
		return
	}
	var raw struct {
		UseAutoMode     json.RawMessage `json:"use_auto_mode"`
		DefaultStrategy string          `json:"default_strategy"`
		DefaultModel    string          `json:"default_model"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
	if len(raw.UseAutoMode) == 0 || raw.UseAutoMode[0] != '"' {
		return
	}
	mode := parseAutoModeJSON(raw.UseAutoMode, "auto")
	if raw.DefaultStrategy != "" {
		h.router.SetStrategy(routing.StrategyType(raw.DefaultStrategy))
	}
	if raw.DefaultModel != "" {
		h.router.SetDefaultModel(raw.DefaultModel)
	}
	h.router.SetUseAutoMode(mode == "auto")
}

// loadConfig loads persisted config from file
func (h *RouterHandler) loadConfig() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if persistedConfig != nil {
		return
	}

	h.migrateLegacyRouterConfig()

	config := h.router.GetConfig()
	mode := "fixed"
	if config.UseAutoMode {
		mode = "auto"
	}
	cfg := PersistedRouterConfig{
		UseAutoMode:     normalizeAutoMode(mode),
		DefaultStrategy: string(config.DefaultStrategy),
		DefaultModel:    config.DefaultModel,
		Classifier:      routing.DefaultClassifierConfig(),
	}
	if config.Classifier.ActiveModel != "" {
		cfg.Classifier = config.Classifier
	}

	if data, err := os.ReadFile(routerUIConfigFile); err == nil {
		var uiCfg PersistedRouterConfig
		if json.Unmarshal(data, &uiCfg) == nil {
			if uiCfg.UseAutoMode != "" {
				cfg.UseAutoMode = normalizeAutoMode(uiCfg.UseAutoMode)
			}
			if uiCfg.DefaultStrategy != "" {
				cfg.DefaultStrategy = uiCfg.DefaultStrategy
			}
			if uiCfg.DefaultModel != "" {
				cfg.DefaultModel = uiCfg.DefaultModel
			}
			if uiCfg.Classifier.ActiveModel != "" {
				cfg.Classifier = routing.ClampClassifierConfig(uiCfg.Classifier)
			}
		}
	}

	if cfg.DefaultStrategy == "" {
		cfg.DefaultStrategy = "auto"
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "deepseek-chat"
	}
	cfg.Classifier = routing.ClampClassifierConfig(cfg.Classifier)

	persistedConfig = &cfg
}

// saveConfig saves config to file
func (h *RouterHandler) saveConfig() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := json.MarshalIndent(persistedConfig, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(routerUIConfigFile), 0755)
	return os.WriteFile(routerUIConfigFile, data, 0644)
}

// GetRouterConfig returns current router configuration
// GET /api/admin/router/config
func (h *RouterHandler) GetRouterConfig(c *gin.Context) {
	h.loadConfig()

	strategies := []StrategyOption{
		{Value: "auto", Label: "智能平衡", Description: "综合效果 + 速度 + 成本，自动选择最优模型"},
		{Value: "quality", Label: "效果优先", Description: "优先选择效果最好的模型"},
		{Value: "speed", Label: "速度优先", Description: "优先选择响应最快的模型"},
		{Value: "cost", Label: "成本优先", Description: "优先选择成本最低的模型"},
		{Value: "custom", Label: "自定义规则", Description: "根据任务类型自动选择模型"},
	}

	h.mu.RLock()
	cfg := *persistedConfig
	h.mu.RUnlock()

	config := h.router.GetConfig()
	classifierCfg := h.router.GetClassifierConfig()
	if cfg.DefaultStrategy == "" {
		cfg.DefaultStrategy = string(config.DefaultStrategy)
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = config.DefaultModel
	}
	if cfg.Classifier.ActiveModel == "" {
		cfg.Classifier = classifierCfg
	}
	cfg.Classifier.CandidateModels = resolveClassifierModels(c.Request.Context(), cfg.Classifier)

	c.JSON(200, gin.H{
		"success": true,
		"data": RouterConfigResponse{
			UseAutoMode:     normalizeAutoMode(cfg.UseAutoMode),
			DefaultStrategy: cfg.DefaultStrategy,
			DefaultModel:    cfg.DefaultModel,
			Classifier:      cfg.Classifier,
			Strategies:      strategies,
		},
	})
}

// GetClassifierModels returns classifier model options from Ollama + persisted config
// GET /api/admin/router/classifier/models
func (h *RouterHandler) GetClassifierModels(c *gin.Context) {
	cfg := h.router.GetClassifierConfig()
	models := resolveClassifierModels(c.Request.Context(), cfg)

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"active_model": cfg.ActiveModel,
			"models":       models,
		},
	})
}

// UpdateRouterConfig updates router configuration
// PUT /api/admin/router/config
func (h *RouterHandler) UpdateRouterConfig(c *gin.Context) {
	h.loadConfig()

	var req UpdateRouterConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	h.mu.Lock()
	if len(req.UseAutoMode) != 0 {
		mode := parseAutoModeJSON(req.UseAutoMode, persistedConfig.UseAutoMode)
		persistedConfig.UseAutoMode = mode
		// 同时更新SmartRouter的UseAutoMode布尔值用于向后兼容
		h.router.SetUseAutoMode(mode == "auto")
	}
	if req.DefaultStrategy != nil {
		persistedConfig.DefaultStrategy = *req.DefaultStrategy
		h.router.SetStrategy(routing.StrategyType(*req.DefaultStrategy))
	}
	if req.DefaultModel != nil {
		persistedConfig.DefaultModel = *req.DefaultModel
		h.router.SetDefaultModel(*req.DefaultModel)
	}
	if req.Classifier != nil {
		persistedConfig.Classifier = routing.ClampClassifierConfig(*req.Classifier)
		h.router.SetClassifierConfig(persistedConfig.Classifier)
	}
	h.mu.Unlock()

	h.saveConfig()

	c.JSON(200, gin.H{
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
			DisplayName:    score.DisplayName,
			QualityScore:   score.QualityScore,
			SpeedScore:     score.SpeedScore,
			CostScore:      score.CostScore,
			CompositeScore: composite,
			Enabled:        score.Enabled,
		})
	}

	c.JSON(200, gin.H{
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
		c.JSON(400, gin.H{
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
		DisplayName:  req.DisplayName,
		QualityScore: req.QualityScore,
		SpeedScore:   req.SpeedScore,
		CostScore:    req.CostScore,
		Enabled:      req.Enabled,
	}

	h.router.UpdateModelScore(model, score)

	c.JSON(200, gin.H{
		"success": true,
		"message": "Model score updated",
	})
}

// DeleteModelScore deletes a model score
// DELETE /api/admin/router/models/:model
func (h *RouterHandler) DeleteModelScore(c *gin.Context) {
	model := c.Param("model")

	h.router.DeleteModelScore(model)

	c.JSON(200, gin.H{
		"success": true,
		"message": "Model score deleted",
	})
}

// GetAvailableModels returns list of enabled models
// GET /api/admin/router/available-models?format=object
func (h *RouterHandler) GetAvailableModels(c *gin.Context) {
	format := c.DefaultQuery("format", "string")
	models := h.router.GetAvailableModels()

	if format == "object" {
		type ModelWithDisplay struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name,omitempty"`
		}
		result := make([]ModelWithDisplay, 0, len(models))
		for _, model := range models {
			score := h.router.GetModelScore(model)
			displayName := ""
			if score != nil {
				displayName = score.DisplayName
			}
			result = append(result, ModelWithDisplay{
				ID:          model,
				DisplayName: displayName,
			})
		}
		c.JSON(200, gin.H{
			"success": true,
			"data":    result,
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    models,
	})
}

// GetTopModels returns top N models for current strategy
// GET /api/admin/router/top-models
func (h *RouterHandler) GetTopModels(c *gin.Context) {
	h.loadConfig()

	h.mu.RLock()
	strategy := routing.StrategyType(persistedConfig.DefaultStrategy)
	h.mu.RUnlock()

	topModels := h.router.GetTopModels(strategy, 5)

	c.JSON(200, gin.H{
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
		c.JSON(400, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	selectedModel := h.router.SelectModel(req.RequestedModel, req.Prompt, nil)

	c.JSON(200, gin.H{
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

	c.JSON(200, gin.H{
		"success": true,
		"data":    defaults,
	})
}

// UpdateProviderDefaults updates default models for providers
// PUT /api/admin/router/provider-defaults
func (h *RouterHandler) UpdateProviderDefaults(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	h.router.SetProviderDefaults(req)

	c.JSON(200, gin.H{
		"success": true,
		"message": "Provider defaults updated",
	})
}

// TTLConfigResponse represents TTL configuration response
type TTLConfigResponse struct {
	TaskTypeDefaults      map[string]int     `json:"task_type_defaults"` // TTL in hours
	DifficultyMultipliers map[string]float64 `json:"difficulty_multipliers"`
}

// GetTTLConfig returns TTL configuration
// GET /api/admin/router/ttl-config
func (h *RouterHandler) GetTTLConfig(c *gin.Context) {
	config := h.router.GetDifficultyAssessor().GetTTLConfig()

	// Convert duration to hours for easier consumption
	taskTypeDefaults := make(map[string]int)
	for k, v := range config.TaskTypeDefaults {
		taskTypeDefaults[string(k)] = int(v.Hours())
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": TTLConfigResponse{
			TaskTypeDefaults: taskTypeDefaults,
			DifficultyMultipliers: map[string]float64{
				"low":    config.DifficultyMultipliers[routing.DifficultyLow],
				"medium": config.DifficultyMultipliers[routing.DifficultyMedium],
				"high":   config.DifficultyMultipliers[routing.DifficultyHigh],
			},
		},
	})
}

// UpdateTTLConfigRequest represents TTL config update request
type UpdateTTLConfigRequest struct {
	TaskTypeDefaults      map[string]int     `json:"task_type_defaults"` // TTL in hours
	DifficultyMultipliers map[string]float64 `json:"difficulty_multipliers"`
}

// UpdateTTLConfig updates TTL configuration
// PUT /api/admin/router/ttl-config
func (h *RouterHandler) UpdateTTLConfig(c *gin.Context) {
	var req UpdateTTLConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	config := h.router.GetDifficultyAssessor().GetTTLConfig()

	// Update task type defaults
	if req.TaskTypeDefaults != nil {
		for k, v := range req.TaskTypeDefaults {
			config.TaskTypeDefaults[routing.TaskType(k)] = time.Duration(v) * time.Hour
		}
	}

	// Update difficulty multipliers
	if req.DifficultyMultipliers != nil {
		for k, v := range req.DifficultyMultipliers {
			switch k {
			case "low":
				config.DifficultyMultipliers[routing.DifficultyLow] = v
			case "medium":
				config.DifficultyMultipliers[routing.DifficultyMedium] = v
			case "high":
				config.DifficultyMultipliers[routing.DifficultyHigh] = v
			}
		}
	}

	h.router.GetDifficultyAssessor().SetTTLConfig(config)

	c.JSON(200, gin.H{
		"success": true,
		"message": "TTL configuration updated",
	})
}

// CascadeRuleResponse represents cascade rule in response
type CascadeRuleResponse struct {
	TaskType        string `json:"task_type"`
	Difficulty      string `json:"difficulty"`
	StartLevel      string `json:"start_level"`
	MaxLevel        string `json:"max_level"`
	FallbackEnabled bool   `json:"fallback_enabled"`
	MaxRetries      int    `json:"max_retries"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
}

// GetCascadeRules returns all cascade routing rules
// GET /api/admin/router/cascade-rules
func (h *RouterHandler) GetCascadeRules(c *gin.Context) {
	rules := h.router.GetCascadeRules()

	response := make([]CascadeRuleResponse, 0, len(rules))
	for _, rule := range rules {
		response = append(response, CascadeRuleResponse{
			TaskType:        string(rule.TaskType),
			Difficulty:      string(rule.Difficulty),
			StartLevel:      string(rule.StartLevel),
			MaxLevel:        string(rule.MaxLevel),
			FallbackEnabled: rule.FallbackEnabled,
			MaxRetries:      rule.MaxRetries,
			TimeoutSeconds:  int(rule.TimeoutPerLevel.Seconds()),
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetCascadeRule returns a specific cascade rule
// GET /api/admin/router/cascade-rules/:taskType/:difficulty
func (h *RouterHandler) GetCascadeRule(c *gin.Context) {
	taskType := c.Param("taskType")
	difficulty := c.Param("difficulty")

	rule := h.router.GetCascadeRule(routing.TaskType(taskType), routing.DifficultyLevel(difficulty))
	if rule == nil {
		c.JSON(404, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Cascade rule not found",
			},
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": CascadeRuleResponse{
			TaskType:        string(rule.TaskType),
			Difficulty:      string(rule.Difficulty),
			StartLevel:      string(rule.StartLevel),
			MaxLevel:        string(rule.MaxLevel),
			FallbackEnabled: rule.FallbackEnabled,
			MaxRetries:      rule.MaxRetries,
			TimeoutSeconds:  int(rule.TimeoutPerLevel.Seconds()),
		},
	})
}

// UpdateCascadeRuleRequest represents cascade rule update request
type UpdateCascadeRuleRequest struct {
	TaskType        string `json:"task_type"`
	Difficulty      string `json:"difficulty"`
	StartLevel      string `json:"start_level"`
	MaxLevel        string `json:"max_level"`
	FallbackEnabled bool   `json:"fallback_enabled"`
	MaxRetries      int    `json:"max_retries"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
}

// UpdateCascadeRule updates or creates a cascade rule
// PUT /api/admin/router/cascade-rules
func (h *RouterHandler) UpdateCascadeRule(c *gin.Context) {
	var req UpdateCascadeRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	rule := &routing.CascadeRule{
		TaskType:        routing.TaskType(req.TaskType),
		Difficulty:      routing.DifficultyLevel(req.Difficulty),
		StartLevel:      routing.CascadeLevel(req.StartLevel),
		MaxLevel:        routing.CascadeLevel(req.MaxLevel),
		FallbackEnabled: req.FallbackEnabled,
		MaxRetries:      req.MaxRetries,
		TimeoutPerLevel: time.Duration(req.TimeoutSeconds) * time.Second,
	}

	h.router.SetCascadeRule(rule)

	c.JSON(200, gin.H{
		"success": true,
		"message": "Cascade rule updated",
	})
}

// DeleteCascadeRule deletes a cascade rule
// DELETE /api/admin/router/cascade-rules/:taskType/:difficulty
func (h *RouterHandler) DeleteCascadeRule(c *gin.Context) {
	taskType := c.Param("taskType")
	difficulty := c.Param("difficulty")

	deleted := h.router.DeleteCascadeRule(routing.TaskType(taskType), routing.DifficultyLevel(difficulty))
	if !deleted {
		c.JSON(404, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Cascade rule not found",
			},
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Cascade rule deleted",
	})
}

// ResetCascadeRules resets all cascade rules to defaults
// POST /api/admin/router/cascade-rules/reset
func (h *RouterHandler) ResetCascadeRules(c *gin.Context) {
	h.router.ResetCascadeRules()

	c.JSON(200, gin.H{
		"success": true,
		"message": "Cascade rules reset to defaults",
	})
}

// GetTaskModelMapping returns task type to model mapping
// GET /api/admin/router/task-model-mapping
func (h *RouterHandler) GetTaskModelMapping(c *gin.Context) {
	mapping := h.router.GetTaskModelMapping()

	c.JSON(200, gin.H{
		"success": true,
		"data":    mapping,
	})
}

// UpdateTaskModelMappingRequest represents task model mapping update request
type UpdateTaskModelMappingRequest map[string]string

type SwitchClassifierModelRequest struct {
	Model string `json:"model"`
}

// UpdateTaskModelMapping updates task type to model mapping
// PUT /api/admin/router/task-model-mapping
func (h *RouterHandler) UpdateTaskModelMapping(c *gin.Context) {
	var req UpdateTaskModelMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	h.router.SetTaskModelMapping(req)

	c.JSON(200, gin.H{
		"success": true,
		"message": "Task model mapping updated",
	})
}

// SwitchClassifierModel switches classifier model after health verification
// POST /api/admin/router/classifier/switch
func (h *RouterHandler) SwitchClassifierModel(c *gin.Context) {
	var req SwitchClassifierModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}
	if req.Model == "" {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "invalid_model", "message": "model is required"}})
		return
	}

	cfg := h.router.GetClassifierConfig()
	originalModel := cfg.ActiveModel
	cfg.ActiveModel = req.Model
	h.router.SetClassifierConfig(cfg)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	health := h.router.GetClassifierHealth(ctx)
	if health == nil || !health.Healthy {
		cfg.ActiveModel = originalModel
		h.router.SetClassifierConfig(cfg)
		message := "classifier health check failed"
		if health != nil && health.Message != "" {
			message = health.Message
		}
		c.JSON(503, gin.H{"success": false, "error": gin.H{"code": "classifier_unhealthy", "message": message}})
		return
	}

	h.loadConfig()
	h.mu.Lock()
	persistedConfig.Classifier = cfg
	h.mu.Unlock()
	_ = h.saveConfig()

	c.JSON(200, gin.H{"success": true, "message": "Classifier model switched", "data": health})
}

// GetClassifierHealth returns classifier health
// GET /api/admin/router/classifier/health
func (h *RouterHandler) GetClassifierHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	health := h.router.GetClassifierHealth(ctx)
	if health == nil {
		c.JSON(503, gin.H{"success": false, "error": gin.H{"code": "classifier_unavailable", "message": "classifier unavailable"}})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": health})
}

// GetClassifierStats returns classifier runtime stats
// GET /api/admin/router/classifier/stats
func (h *RouterHandler) GetClassifierStats(c *gin.Context) {
	stats := h.router.GetClassifierStats()
	c.JSON(200, gin.H{"success": true, "data": stats})
}
