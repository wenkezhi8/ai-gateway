package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/constants"
	"ai-gateway/internal/routing"

	"github.com/gin-gonic/gin"
)

// RouterHandler handles smart router configuration requests.
type RouterHandler struct {
	router          *routing.SmartRouter
	mu              sync.RWMutex
	switchTaskStore *classifierSwitchTaskStore
	nowFn           func() time.Time
	sleepFn         func(time.Duration)
	probeSwitchFn   func(targetModel, originalModel string) error
	intentEngineCfg IntentEngineConfig
}

// NewRouterHandler creates a new router handler.
func NewRouterHandler(router *routing.SmartRouter) *RouterHandler {
	taskStore, err := newClassifierSwitchTaskStore(constants.ClassifierSwitchTaskDBPath)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize classifier switch task store: %v", err))
	}

	h := &RouterHandler{
		router:          router,
		switchTaskStore: taskStore,
		intentEngineCfg: defaultIntentEngineConfig(),
	}
	h.nowFn = time.Now
	h.sleepFn = time.Sleep
	h.probeSwitchFn = h.probeAndApplyClassifierSwitch
	h.loadConfig()
	h.loadIntentEngineConfig()
	return h
}

// RouterConfigResponse represents the router configuration response.
type RouterConfigResponse struct {
	UseAutoMode     string                   `json:"use_auto_mode"` // "auto", "default", "fixed", "latest"
	DefaultStrategy string                   `json:"default_strategy"`
	DefaultModel    string                   `json:"default_model"`
	Classifier      routing.ClassifierConfig `json:"classifier"`
	Strategies      []StrategyOption         `json:"strategies"`
}

// StrategyOption represents a strategy option.
type StrategyOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ModelScoreResponse represents model score in response.
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

// UpdateRouterConfigRequest represents update request.
type UpdateRouterConfigRequest struct {
	UseAutoMode     json.RawMessage           `json:"use_auto_mode,omitempty"` // "auto", "default", "fixed", "latest" or bool
	DefaultStrategy *string                   `json:"default_strategy,omitempty"`
	DefaultModel    *string                   `json:"default_model,omitempty"`
	Classifier      *routing.ClassifierConfig `json:"classifier,omitempty"`
}

// UpdateModelScoreRequest represents model score update request.
type UpdateModelScoreRequest struct {
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	DisplayName  string `json:"display_name,omitempty"`
	QualityScore int    `json:"quality_score"`
	SpeedScore   int    `json:"speed_score"`
	CostScore    int    `json:"cost_score"`
	Enabled      bool   `json:"enabled"`
}

// IntentEngineConfig represents local intent-engine runtime config.
type IntentEngineConfig struct {
	Enabled           bool   `json:"enabled"`
	BaseURL           string `json:"base_url"`
	TimeoutMs         int    `json:"timeout_ms"`
	Language          string `json:"language"`
	ExpectedDimension int    `json:"expected_dimension"`
}

// PersistedRouterConfig is the structure stored for UI routing mode selection.
type PersistedRouterConfig struct {
	UseAutoMode     string                   `json:"use_auto_mode"`
	DefaultStrategy string                   `json:"default_strategy"`
	DefaultModel    string                   `json:"default_model"`
	Classifier      routing.ClassifierConfig `json:"classifier"`
}

const routerUIConfigFile = constants.RouterUIConfigFilePath
const routerConfigFile = constants.RouterConfigFilePath
const intentEngineConfigFile = constants.IntentEngineConfigFilePath
const autoModeAuto = "auto"
const autoModeFixed = "fixed"

var persistedConfig *PersistedRouterConfig

func normalizeAutoMode(value string) string {
	switch value {
	case autoModeAuto, "default", autoModeFixed, "latest":
		return value
	default:
		return constants.RoutingDefaultStrategy
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
			return constants.RoutingDefaultStrategy
		}
		return autoModeFixed
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

func resolveClassifierModels(ctx context.Context, cfg *routing.ClassifierConfig) []string {
	if cfg == nil {
		def := routing.DefaultClassifierConfig()
		cfg = &def
	}

	timeout := constants.AdminResolveClassifierBaseTimeout
	if cfg.TimeoutMs > 0 {
		candidate := time.Duration(cfg.TimeoutMs) * time.Millisecond
		if candidate > timeout {
			timeout = candidate
		}
	}
	if timeout > constants.AdminResolveClassifierMaxTimeout {
		timeout = constants.AdminResolveClassifierMaxTimeout
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	models, err := routing.ListOllamaModels(ctxWithTimeout, cfg.BaseURL, timeout)
	if err != nil {
		return mergeClassifierCandidateModels(cfg.ActiveModel, cfg.CandidateModels)
	}

	return mergeClassifierCandidateModels(cfg.ActiveModel, models, cfg.CandidateModels)
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runShellCommand(timeout time.Duration, command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		if output == "" {
			return "", err
		}
		return output, fmt.Errorf("%w: %s", err, output)
	}
	return output, nil
}

func getOllamaStopCommand(goos string) (string, error) {
	switch goos {
	case goosDarwin:
		return `osascript -e 'quit app "Ollama"' >/dev/null 2>&1 || true; pkill -f "ollama serve" >/dev/null 2>&1 || true`, nil
	case goosLinux:
		return `pkill -f "ollama serve" >/dev/null 2>&1 || true`, nil
	default:
		return "", fmt.Errorf("current OS is not supported for auto stop")
	}
}

func checkOllamaRunning(ctx context.Context, cfg *routing.ClassifierConfig) (running bool, models []string, detail string) {
	if cfg == nil {
		def := routing.DefaultClassifierConfig()
		cfg = &def
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = constants.ClassifierDefaultBaseURL
	}

	timeout := 2 * time.Second
	if cfg.TimeoutMs > 0 {
		candidate := time.Duration(cfg.TimeoutMs) * time.Millisecond
		if candidate < timeout {
			timeout = candidate
		}
	}

	models, err := routing.ListOllamaModels(ctx, baseURL, timeout)
	if err != nil {
		return false, nil, err.Error()
	}
	return true, models, "ok"
}

func containsModel(models []string, model string) bool {
	target := strings.TrimSpace(model)
	if target == "" {
		return false
	}
	for _, m := range models {
		if strings.TrimSpace(m) == target {
			return true
		}
	}
	return false
}

func defaultIntentEngineConfig() IntentEngineConfig {
	return IntentEngineConfig{
		Enabled:           false,
		BaseURL:           "http://127.0.0.1:18566",
		TimeoutMs:         1500,
		Language:          "zh-CN",
		ExpectedDimension: 1024,
	}
}

func (h *RouterHandler) loadIntentEngineConfig() {
	h.mu.Lock()
	defer h.mu.Unlock()

	cfg := defaultIntentEngineConfig()
	data, err := os.ReadFile(intentEngineConfigFile)
	if err == nil && len(data) > 0 {
		var persisted IntentEngineConfig
		if json.Unmarshal(data, &persisted) == nil {
			if strings.TrimSpace(persisted.BaseURL) != "" {
				cfg.BaseURL = strings.TrimSpace(persisted.BaseURL)
			}
			if persisted.TimeoutMs > 0 {
				cfg.TimeoutMs = persisted.TimeoutMs
			}
			if strings.TrimSpace(persisted.Language) != "" {
				cfg.Language = strings.TrimSpace(persisted.Language)
			}
			if persisted.ExpectedDimension > 0 {
				cfg.ExpectedDimension = persisted.ExpectedDimension
			}
			cfg.Enabled = persisted.Enabled
		}
	}
	h.intentEngineCfg = cfg
}

func (h *RouterHandler) saveIntentEngineConfig() error {
	h.mu.RLock()
	cfg := h.intentEngineCfg
	h.mu.RUnlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(intentEngineConfigFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(intentEngineConfigFile, data, 0640)
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
	mode := parseAutoModeJSON(raw.UseAutoMode, constants.RoutingDefaultStrategy)
	if raw.DefaultStrategy != "" {
		h.router.SetStrategy(routing.StrategyType(raw.DefaultStrategy))
	}
	if raw.DefaultModel != "" {
		h.router.SetDefaultModel(raw.DefaultModel)
	}
	h.router.SetUseAutoMode(mode == autoModeAuto)
}

// loadConfig loads persisted config from file.
func (h *RouterHandler) loadConfig() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if persistedConfig != nil {
		return
	}

	h.migrateLegacyRouterConfig()

	config := h.router.GetConfig()
	mode := autoModeFixed
	if config.UseAutoMode {
		mode = constants.RoutingDefaultStrategy
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
		cfg.DefaultStrategy = constants.RoutingDefaultStrategy
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = constants.RoutingDefaultModel
	}
	cfg.Classifier = routing.ClampClassifierConfig(cfg.Classifier)

	persistedConfig = &cfg
}

// saveConfig saves config to file.
func (h *RouterHandler) saveConfig() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := json.MarshalIndent(persistedConfig, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(routerUIConfigFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(routerUIConfigFile, data, 0640)
}

// GET /api/admin/router/config.
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
	cfg.Classifier.CandidateModels = resolveClassifierModels(c.Request.Context(), &cfg.Classifier)

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

// GET /api/admin/router/classifier/models.
func (h *RouterHandler) GetClassifierModels(c *gin.Context) {
	cfg := h.router.GetClassifierConfig()
	models := resolveClassifierModels(c.Request.Context(), &cfg)

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"active_model": cfg.ActiveModel,
			"models":       models,
		},
	})
}

// PUT /api/admin/router/config.
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

	if err := h.saveConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "save_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Router configuration updated",
	})
}

type UpdateIntentEngineConfigRequest struct {
	Enabled           *bool   `json:"enabled"`
	BaseURL           *string `json:"base_url"`
	TimeoutMs         *int    `json:"timeout_ms"`
	Language          *string `json:"language"`
	ExpectedDimension *int    `json:"expected_dimension"`
}

// GET /api/admin/router/intent-engine/config.
func (h *RouterHandler) GetIntentEngineConfig(c *gin.Context) {
	h.mu.RLock()
	cfg := h.intentEngineCfg
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cfg,
	})
}

// PUT /api/admin/router/intent-engine/config.
func (h *RouterHandler) UpdateIntentEngineConfig(c *gin.Context) {
	var req UpdateIntentEngineConfigRequest
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
	if req.Enabled != nil {
		h.intentEngineCfg.Enabled = *req.Enabled
	}
	if req.BaseURL != nil && strings.TrimSpace(*req.BaseURL) != "" {
		h.intentEngineCfg.BaseURL = strings.TrimSpace(*req.BaseURL)
	}
	if req.TimeoutMs != nil && *req.TimeoutMs > 0 {
		h.intentEngineCfg.TimeoutMs = *req.TimeoutMs
	}
	if req.Language != nil && strings.TrimSpace(*req.Language) != "" {
		h.intentEngineCfg.Language = strings.TrimSpace(*req.Language)
	}
	if req.ExpectedDimension != nil && *req.ExpectedDimension > 0 {
		h.intentEngineCfg.ExpectedDimension = *req.ExpectedDimension
	}
	h.mu.Unlock()

	if err := h.saveIntentEngineConfig(); err != nil {
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
		"message": "intent engine config updated",
	})
}

// GET /api/admin/router/intent-engine/health.
func (h *RouterHandler) GetIntentEngineHealth(c *gin.Context) {
	h.mu.RLock()
	cfg := h.intentEngineCfg
	h.mu.RUnlock()

	if !cfg.Enabled {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"enabled": false,
				"healthy": false,
				"message": "intent engine disabled",
			},
		})
		return
	}

	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.BaseURL, "/")+"/health", http.NoBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "request_build_failed", "message": err.Error()},
		})
		return
	}

	start := time.Now()
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"enabled":    true,
				"healthy":    false,
				"latency_ms": time.Since(start).Milliseconds(),
				"message":    err.Error(),
			},
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "read_response_failed", "message": err.Error()},
		})
		return
	}
	payload := gin.H{
		"enabled":    true,
		"healthy":    resp.StatusCode >= 200 && resp.StatusCode < 300,
		"status":     resp.StatusCode,
		"latency_ms": time.Since(start).Milliseconds(),
	}
	if len(body) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err == nil {
			for k, v := range parsed {
				payload[k] = v
			}
		} else {
			payload["raw"] = strings.TrimSpace(string(body))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    payload,
	})
}

// GET /api/admin/router/models.
func (h *RouterHandler) GetModelScores(c *gin.Context) {
	scores := h.router.GetAllModelScores()
	config := h.router.GetConfig()

	response := make([]ModelScoreResponse, 0, len(scores))
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

// PUT /api/admin/router/models/:model.
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

// DELETE /api/admin/router/models/:model.
func (h *RouterHandler) DeleteModelScore(c *gin.Context) {
	model := c.Param("model")

	h.router.DeleteModelScore(model)

	c.JSON(200, gin.H{
		"success": true,
		"message": "Model score deleted",
	})
}

// GET /api/admin/router/available-models?format=object.
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

// GET /api/admin/router/top-models.
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

// SelectModelRequest represents model selection request.
type SelectModelRequest struct {
	RequestedModel string `json:"requested_model"`
	Prompt         string `json:"prompt"`
}

// POST /api/admin/router/select.
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

// GET /api/admin/router/provider-defaults.
func (h *RouterHandler) GetProviderDefaults(c *gin.Context) {
	defaults := h.router.GetProviderDefaults()

	c.JSON(200, gin.H{
		"success": true,
		"data":    defaults,
	})
}

// PUT /api/admin/router/provider-defaults.
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

// TTLConfigResponse represents TTL configuration response.
type TTLConfigResponse struct {
	TaskTypeDefaults      map[string]int     `json:"task_type_defaults"` // TTL in hours
	DifficultyMultipliers map[string]float64 `json:"difficulty_multipliers"`
}

// GET /api/admin/router/ttl-config.
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

// UpdateTTLConfigRequest represents TTL config update request.
type UpdateTTLConfigRequest struct {
	TaskTypeDefaults      map[string]int     `json:"task_type_defaults"` // TTL in hours
	DifficultyMultipliers map[string]float64 `json:"difficulty_multipliers"`
}

// PUT /api/admin/router/ttl-config.
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

// CascadeRuleResponse represents cascade rule in response.
type CascadeRuleResponse struct {
	TaskType        string `json:"task_type"`
	Difficulty      string `json:"difficulty"`
	StartLevel      string `json:"start_level"`
	MaxLevel        string `json:"max_level"`
	FallbackEnabled bool   `json:"fallback_enabled"`
	MaxRetries      int    `json:"max_retries"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
}

// GET /api/admin/router/cascade-rules.
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

// GET /api/admin/router/cascade-rules/:taskType/:difficulty.
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

// UpdateCascadeRuleRequest represents cascade rule update request.
type UpdateCascadeRuleRequest struct {
	TaskType        string `json:"task_type"`
	Difficulty      string `json:"difficulty"`
	StartLevel      string `json:"start_level"`
	MaxLevel        string `json:"max_level"`
	FallbackEnabled bool   `json:"fallback_enabled"`
	MaxRetries      int    `json:"max_retries"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
}

// PUT /api/admin/router/cascade-rules.
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

// DELETE /api/admin/router/cascade-rules/:taskType/:difficulty.
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

// POST /api/admin/router/cascade-rules/reset.
func (h *RouterHandler) ResetCascadeRules(c *gin.Context) {
	h.router.ResetCascadeRules()

	c.JSON(200, gin.H{
		"success": true,
		"message": "Cascade rules reset to defaults",
	})
}

// GET /api/admin/router/task-model-mapping.
func (h *RouterHandler) GetTaskModelMapping(c *gin.Context) {
	mapping := h.router.GetTaskModelMapping()

	c.JSON(200, gin.H{
		"success": true,
		"data":    mapping,
	})
}

// UpdateTaskModelMappingRequest represents task model mapping update request.
type UpdateTaskModelMappingRequest map[string]string

type SwitchClassifierModelRequest struct {
	Model string `json:"model"`
}

type switchClassifierTaskResponse struct {
	TaskID string `json:"task_id"`
}

// PUT /api/admin/router/task-model-mapping.
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

// POST /api/admin/router/classifier/switch.
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.AdminClassifierHealthTimeout)
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
	if err := h.saveConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "save_failed", "message": err.Error()},
		})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "Classifier model switched", "data": health})
}

// POST /api/admin/router/classifier/switch-async.
func (h *RouterHandler) SwitchClassifierModelAsync(c *gin.Context) {
	var req SwitchClassifierModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}
	if req.Model == "" {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "invalid_model", "message": "model is required"}})
		return
	}
	if h.switchTaskStore == nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "task_store_unavailable", "message": "switch task store unavailable"}})
		return
	}

	taskID := generateID()
	now := time.Now().UnixMilli()
	originalModel := ""
	if h.router != nil {
		originalModel = h.router.GetClassifierConfig().ActiveModel
	}
	deadline := h.nowFn().Add(constants.AdminClassifierSwitchAsyncMaxWait).UnixMilli()
	task := &ClassifierSwitchTask{
		TaskID:        taskID,
		TargetModel:   req.Model,
		OriginalModel: originalModel,
		Status:        ClassifierSwitchTaskStatusPending,
		StartedAt:     now,
		UpdatedAt:     now,
		DeadlineAt:    deadline,
		Attempts:      0,
	}
	if err := h.switchTaskStore.Create(task); err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "task_create_failed", "message": err.Error()}})
		return
	}

	go h.executeSwitchTask(taskID)

	c.JSON(202, gin.H{
		"success": true,
		"data": switchClassifierTaskResponse{
			TaskID: taskID,
		},
	})
}

// GET /api/admin/router/classifier/switch-tasks/:taskId.
func (h *RouterHandler) GetSwitchClassifierTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if strings.TrimSpace(taskID) == "" {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "invalid_task", "message": "task id is required"}})
		return
	}

	if h.switchTaskStore == nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "task_store_unavailable", "message": "switch task store unavailable"}})
		return
	}

	task, err := h.switchTaskStore.Get(taskID)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "task_query_failed", "message": err.Error()}})
		return
	}
	if task == nil {
		c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "task_not_found", "message": "switch task not found"}})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": task})
}

// GET /api/admin/router/classifier/health.
func (h *RouterHandler) GetClassifierHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.AdminClassifierHealthTimeout)
	defer cancel()
	health := h.router.GetClassifierHealth(ctx)
	if health == nil {
		c.JSON(503, gin.H{"success": false, "error": gin.H{"code": "classifier_unavailable", "message": "classifier unavailable"}})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": health})
}

// GET /api/admin/router/classifier/stats.
func (h *RouterHandler) GetClassifierStats(c *gin.Context) {
	stats := h.router.GetClassifierStats()
	c.JSON(200, gin.H{"success": true, "data": stats})
}

type ollamaSetupRequest struct {
	Model string `json:"model"`
}

func (h *RouterHandler) GetOllamaSetupStatus(c *gin.Context) {
	cfg := h.router.GetClassifierConfig()
	model := strings.TrimSpace(c.Query("model"))
	if model == "" {
		model = cfg.ActiveModel
	}
	if model == "" {
		model = constants.ClassifierDefaultModel
	}

	installed := commandExists("ollama")
	running := false
	modelInstalled := false
	models := make([]string, 0)
	runningModels := make([]string, 0)
	runningModelDetails := make([]gin.H, 0)
	runningVramBytesTotal := int64(0)
	runningModel := ""
	message := "ollama not installed"

	if installed {
		runCtx, cancel := context.WithTimeout(c.Request.Context(), constants.AdminOllamaCheckTimeout)
		defer cancel()
		var detail string
		running, models, detail = checkOllamaRunning(runCtx, &cfg)
		if running {
			modelInstalled = containsModel(models, model)
			psCtx, psCancel := context.WithTimeout(c.Request.Context(), constants.AdminOllamaCheckTimeout)
			details, err := routing.ListOllamaRunningModelDetails(psCtx, cfg.BaseURL, constants.AdminOllamaCheckTimeout)
			psCancel()
			if err == nil {
				runningModels = make([]string, 0, len(details))
				for _, detail := range details {
					runningModels = append(runningModels, detail.Name)
					runningModelDetails = append(runningModelDetails, gin.H{"name": detail.Name, "size_vram": detail.SizeVRAM})
					if detail.SizeVRAM > 0 {
						runningVramBytesTotal += detail.SizeVRAM
					}
				}
			} else {
				message = err.Error()
			}
			if containsModel(runningModels, model) {
				runningModel = model
			} else if len(runningModels) > 0 {
				runningModel = runningModels[0]
			}
			if err == nil {
				message = "ok"
			}
		} else {
			message = detail
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"installed":                installed,
			"running":                  running,
			"model":                    model,
			"model_installed":          modelInstalled,
			"models":                   models,
			"running_models":           runningModels,
			"running_model_details":    runningModelDetails,
			"running_vram_bytes_total": runningVramBytesTotal,
			"running_model":            runningModel,
			"keep_alive_disabled":      true,
			"message":                  message,
			"os":                       runtime.GOOS,
		},
	})
}

func (h *RouterHandler) InstallOllama(c *gin.Context) {
	if commandExists("ollama") {
		c.JSON(200, gin.H{"success": true, "message": "ollama already installed"})
		return
	}

	var command string
	switch runtime.GOOS {
	case goosDarwin:
		if !commandExists("brew") {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "brew_not_found", "message": "Homebrew not found, please install Homebrew first"}})
			return
		}
		command = "brew install ollama"
	case goosLinux:
		command = "curl -fsSL https://ollama.com/install.sh | sh"
	default:
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "unsupported_os", "message": "current OS is not supported for auto install, please install Ollama manually"}})
		return
	}

	output, err := runShellCommand(constants.AdminOllamaInstallTimeout, command)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "install_failed", "message": err.Error()}, "data": gin.H{"output": output}})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "ollama installed", "data": gin.H{"output": output}})
}

func (h *RouterHandler) StartOllama(c *gin.Context) {
	cfg := h.router.GetClassifierConfig()
	runCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	running, _, _ := checkOllamaRunning(runCtx, &cfg)
	cancel()
	if running {
		c.JSON(200, gin.H{"success": true, "message": "ollama already running"})
		return
	}

	if !commandExists("ollama") {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "ollama_not_installed", "message": "ollama not installed"}})
		return
	}

	var command string
	switch runtime.GOOS {
	case goosDarwin:
		command = "open -a Ollama"
	case goosLinux:
		command = "nohup ollama serve >/tmp/ollama.log 2>&1 &"
	default:
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "unsupported_os", "message": "current OS is not supported for auto start"}})
		return
	}

	output, err := runShellCommand(constants.AdminOllamaStartCommandTimeout, command)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "start_failed", "message": err.Error()}, "data": gin.H{"output": output}})
		return
	}

	deadline := time.Now().Add(constants.AdminOllamaStartReadyDeadline)
	for time.Now().Before(deadline) {
		checkCtx, stop := context.WithTimeout(c.Request.Context(), constants.AdminOllamaStartProbeTimeout)
		alive, _, _ := checkOllamaRunning(checkCtx, &cfg)
		stop()
		if alive {
			c.JSON(200, gin.H{"success": true, "message": "ollama started", "data": gin.H{"output": output}})
			return
		}
		time.Sleep(constants.AdminOllamaStartProbeInterval)
	}

	c.JSON(503, gin.H{"success": false, "error": gin.H{"code": "start_timeout", "message": "ollama did not become ready in time"}, "data": gin.H{"output": output}})
}

func (h *RouterHandler) StopOllama(c *gin.Context) {
	if !commandExists("ollama") {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "ollama_not_installed", "message": "ollama not installed"}})
		return
	}

	command, err := getOllamaStopCommand(runtime.GOOS)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "unsupported_os", "message": err.Error()}})
		return
	}

	output, runErr := runShellCommand(constants.AdminOllamaStartCommandTimeout, command)
	if runErr != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "stop_failed", "message": runErr.Error()}, "data": gin.H{"output": output}})
		return
	}

	cfg := h.router.GetClassifierConfig()
	deadline := time.Now().Add(constants.AdminOllamaStartReadyDeadline)
	for time.Now().Before(deadline) {
		checkCtx, cancel := context.WithTimeout(c.Request.Context(), constants.AdminOllamaStartProbeTimeout)
		running, _, _ := checkOllamaRunning(checkCtx, &cfg)
		cancel()
		if !running {
			c.JSON(200, gin.H{"success": true, "message": "ollama stopped", "data": gin.H{"output": output}})
			return
		}
		time.Sleep(constants.AdminOllamaStartProbeInterval)
	}

	c.JSON(503, gin.H{"success": false, "error": gin.H{"code": "stop_timeout", "message": "ollama did not stop in time"}, "data": gin.H{"output": output}})
}

func (h *RouterHandler) PullOllamaModel(c *gin.Context) {
	var req ollamaSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	cfg := h.router.GetClassifierConfig()
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = cfg.ActiveModel
	}
	if model == "" {
		model = constants.ClassifierDefaultModel
	}

	if !commandExists("ollama") {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "ollama_not_installed", "message": "ollama not installed"}})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.AdminOllamaCheckTimeout)
	running, _, msg := checkOllamaRunning(ctx, &cfg)
	cancel()
	if !running {
		c.JSON(503, gin.H{"success": false, "error": gin.H{"code": "ollama_not_running", "message": msg}})
		return
	}

	output, err := runShellCommand(constants.AdminOllamaPullTimeout, fmt.Sprintf("ollama pull %s", model))
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "pull_failed", "message": err.Error()}, "data": gin.H{"output": output}})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "model installed", "data": gin.H{"model": model, "output": output}})
}
