package routing

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
)

const modelScoresFile = "data/model_scores.json"
const providerDefaultsFile = "data/provider_defaults.json"
const routerConfigFile = "data/router_config.json"

var routerLogger = logrus.New()

// StrategyType defines the routing strategy
type StrategyType string

const (
	StrategyAuto    StrategyType = "auto"    // Smart balance
	StrategyQuality StrategyType = "quality" // Best quality
	StrategySpeed   StrategyType = "speed"   // Fastest response
	StrategyCost    StrategyType = "cost"    // Lowest cost
	StrategyCustom  StrategyType = "custom"  // Custom rules
)

// ModelScore represents scoring for a model
type ModelScore struct {
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	QualityScore int    `json:"quality_score"` // 0-100
	SpeedScore   int    `json:"speed_score"`   // 0-100
	CostScore    int    `json:"cost_score"`    // 0-100 (higher = cheaper)
	Enabled      bool   `json:"enabled"`
}

// TaskRule defines model selection for specific task types
type TaskRule struct {
	TaskType        string   `json:"task_type"`
	Keywords        []string `json:"keywords"`
	PreferredModels []string `json:"preferred_models"`
}

// RouterConfig holds the smart router configuration
type RouterConfig struct {
	DefaultStrategy  StrategyType           `json:"default_strategy"`
	DefaultModel     string                 `json:"default_model"`
	UseAutoMode      bool                   `json:"use_auto_mode"`
	ModelScores      map[string]*ModelScore `json:"model_scores"`
	TaskRules        []TaskRule             `json:"task_rules"`
	ProviderDefaults map[string]string      `json:"provider_defaults"` // provider -> default model
}

// SmartRouter handles intelligent model selection
type SmartRouter struct {
	mu     sync.RWMutex
	config *RouterConfig
}

// DefaultModelScores returns default scoring for common models
func DefaultModelScores() map[string]*ModelScore {
	return map[string]*ModelScore{
		// DeepSeek
		"deepseek-chat":     {Model: "deepseek-chat", Provider: "deepseek", QualityScore: 85, SpeedScore: 90, CostScore: 95, Enabled: true},
		"deepseek-reasoner": {Model: "deepseek-reasoner", Provider: "deepseek", QualityScore: 95, SpeedScore: 60, CostScore: 90, Enabled: true},
		"deepseek-coder":    {Model: "deepseek-coder", Provider: "deepseek", QualityScore: 90, SpeedScore: 85, CostScore: 95, Enabled: true},

		// OpenAI
		"gpt-4o":      {Model: "gpt-4o", Provider: "openai", QualityScore: 95, SpeedScore: 75, CostScore: 60, Enabled: true},
		"gpt-4o-mini": {Model: "gpt-4o-mini", Provider: "openai", QualityScore: 80, SpeedScore: 95, CostScore: 85, Enabled: true},
		"gpt-4-turbo": {Model: "gpt-4-turbo", Provider: "openai", QualityScore: 92, SpeedScore: 70, CostScore: 55, Enabled: true},
		"o1":          {Model: "o1", Provider: "openai", QualityScore: 98, SpeedScore: 40, CostScore: 30, Enabled: true},
		"o1-mini":     {Model: "o1-mini", Provider: "openai", QualityScore: 90, SpeedScore: 60, CostScore: 50, Enabled: true},

		// Anthropic
		"claude-3-5-sonnet-20241022": {Model: "claude-3-5-sonnet-20241022", Provider: "anthropic", QualityScore: 96, SpeedScore: 70, CostScore: 55, Enabled: true},
		"claude-3-5-haiku-20241022":  {Model: "claude-3-5-haiku-20241022", Provider: "anthropic", QualityScore: 82, SpeedScore: 95, CostScore: 80, Enabled: true},
		"claude-3-opus-20240229":     {Model: "claude-3-opus-20240229", Provider: "anthropic", QualityScore: 97, SpeedScore: 50, CostScore: 40, Enabled: true},

		// Qwen
		"qwen-max":   {Model: "qwen-max", Provider: "qwen", QualityScore: 90, SpeedScore: 80, CostScore: 75, Enabled: true},
		"qwen-plus":  {Model: "qwen-plus", Provider: "qwen", QualityScore: 85, SpeedScore: 85, CostScore: 85, Enabled: true},
		"qwen-turbo": {Model: "qwen-turbo", Provider: "qwen", QualityScore: 75, SpeedScore: 95, CostScore: 95, Enabled: true},
		"qwen-long":  {Model: "qwen-long", Provider: "qwen", QualityScore: 80, SpeedScore: 70, CostScore: 70, Enabled: true},

		// Zhipu
		"glm-4-plus":  {Model: "glm-4-plus", Provider: "zhipu", QualityScore: 88, SpeedScore: 80, CostScore: 80, Enabled: true},
		"glm-4":       {Model: "glm-4", Provider: "zhipu", QualityScore: 85, SpeedScore: 85, CostScore: 85, Enabled: true},
		"glm-4-flash": {Model: "glm-4-flash", Provider: "zhipu", QualityScore: 70, SpeedScore: 98, CostScore: 98, Enabled: true},
		"glm-4-long":  {Model: "glm-4-long", Provider: "zhipu", QualityScore: 80, SpeedScore: 70, CostScore: 75, Enabled: true},

		// Moonshot
		"moonshot-v1-8k":   {Model: "moonshot-v1-8k", Provider: "moonshot", QualityScore: 85, SpeedScore: 85, CostScore: 85, Enabled: true},
		"moonshot-v1-32k":  {Model: "moonshot-v1-32k", Provider: "moonshot", QualityScore: 85, SpeedScore: 80, CostScore: 80, Enabled: true},
		"moonshot-v1-128k": {Model: "moonshot-v1-128k", Provider: "moonshot", QualityScore: 85, SpeedScore: 75, CostScore: 75, Enabled: true},

		// MiniMax
		"abab6.5s-chat": {Model: "abab6.5s-chat", Provider: "minimax", QualityScore: 85, SpeedScore: 90, CostScore: 85, Enabled: true},
		"abab6.5g-chat": {Model: "abab6.5g-chat", Provider: "minimax", QualityScore: 88, SpeedScore: 80, CostScore: 80, Enabled: true},
		"abab5.5-chat":  {Model: "abab5.5-chat", Provider: "minimax", QualityScore: 80, SpeedScore: 90, CostScore: 90, Enabled: true},

		// Baichuan
		"Baichuan4":       {Model: "Baichuan4", Provider: "baichuan", QualityScore: 88, SpeedScore: 80, CostScore: 80, Enabled: true},
		"Baichuan3-Turbo": {Model: "Baichuan3-Turbo", Provider: "baichuan", QualityScore: 82, SpeedScore: 90, CostScore: 90, Enabled: true},

		// Volcengine
		"doubao-pro-128k": {Model: "doubao-pro-128k", Provider: "volcengine", QualityScore: 85, SpeedScore: 85, CostScore: 85, Enabled: true},
		"doubao-lite-32k": {Model: "doubao-lite-32k", Provider: "volcengine", QualityScore: 75, SpeedScore: 95, CostScore: 95, Enabled: true},

		// Google
		"gemini-2.0-flash": {Model: "gemini-2.0-flash", Provider: "google", QualityScore: 88, SpeedScore: 90, CostScore: 80, Enabled: true},
		"gemini-1.5-pro":   {Model: "gemini-1.5-pro", Provider: "google", QualityScore: 92, SpeedScore: 75, CostScore: 70, Enabled: true},
	}
}

// DefaultTaskRules returns default task-based routing rules
func DefaultTaskRules() []TaskRule {
	return []TaskRule{
		{
			TaskType:        "code",
			Keywords:        []string{"代码", "code", "编程", "bug", "debug", "function", "class", "实现"},
			PreferredModels: []string{"deepseek-coder", "claude-3-5-sonnet-20241022", "gpt-4o"},
		},
		{
			TaskType:        "reasoning",
			Keywords:        []string{"推理", "reasoning", "分析", "逻辑", "证明", "数学", "math"},
			PreferredModels: []string{"deepseek-reasoner", "o1", "o1-mini", "claude-3-opus-20240229"},
		},
		{
			TaskType:        "long_context",
			Keywords:        []string{"长文本", "总结", "摘要", "文档", "分析报告"},
			PreferredModels: []string{"qwen-long", "glm-4-long", "moonshot-v1-128k", "claude-3-5-sonnet-20241022"},
		},
		{
			TaskType:        "creative",
			Keywords:        []string{"写作", "创意", "故事", "文案", "创作"},
			PreferredModels: []string{"claude-3-5-sonnet-20241022", "gpt-4o", "qwen-max"},
		},
		{
			TaskType:        "chat",
			Keywords:        []string{},
			PreferredModels: []string{"deepseek-chat", "gpt-4o-mini", "glm-4-flash", "qwen-turbo"},
		},
	}
}

// NewSmartRouter creates a new smart router with default config
func NewSmartRouter() *SmartRouter {
	router := &SmartRouter{
		config: &RouterConfig{
			DefaultStrategy:  StrategyAuto,
			DefaultModel:     "deepseek-chat",
			UseAutoMode:      true,
			ModelScores:      DefaultModelScores(),
			TaskRules:        DefaultTaskRules(),
			ProviderDefaults: DefaultProviderDefaults(),
		},
	}

	// Load persisted model scores
	router.loadFromFile()

	return router
}

// SaveToFile saves the model scores to a file
func (r *SmartRouter) SaveToFile() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Ensure data directory exists
	dir := filepath.Dir(modelScoresFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r.config.ModelScores, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(modelScoresFile, data, 0644)
}

// SaveProviderDefaultsToFile saves provider defaults to a file
func (r *SmartRouter) SaveProviderDefaultsToFile() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Ensure data directory exists
	dir := filepath.Dir(providerDefaultsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r.config.ProviderDefaults, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(providerDefaultsFile, data, 0644)
}

// RouterConfigPersist is the structure saved to file for router config
type RouterConfigPersist struct {
	DefaultStrategy string `json:"default_strategy"`
	DefaultModel    string `json:"default_model"`
	UseAutoMode     bool   `json:"use_auto_mode"`
}

// SaveRouterConfigToFile saves router config to a file
func (r *SmartRouter) SaveRouterConfigToFile() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Ensure data directory exists
	dir := filepath.Dir(routerConfigFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	config := RouterConfigPersist{
		DefaultStrategy: string(r.config.DefaultStrategy),
		DefaultModel:    r.config.DefaultModel,
		UseAutoMode:     r.config.UseAutoMode,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(routerConfigFile, data, 0644)
}

// loadFromFile loads model scores from the persistence file
func (r *SmartRouter) loadFromFile() {
	// Load router config (strategy, default model, auto mode)
	configData, err := os.ReadFile(routerConfigFile)
	if err == nil {
		var savedConfig RouterConfigPersist
		if err := json.Unmarshal(configData, &savedConfig); err == nil {
			r.mu.Lock()
			if savedConfig.DefaultStrategy != "" {
				r.config.DefaultStrategy = StrategyType(savedConfig.DefaultStrategy)
			}
			if savedConfig.DefaultModel != "" {
				r.config.DefaultModel = savedConfig.DefaultModel
			}
			r.config.UseAutoMode = savedConfig.UseAutoMode
			r.mu.Unlock()
			routerLogger.Info("Loaded router config from persistence file")
		} else {
			routerLogger.WithError(err).Warn("Failed to parse router config file")
		}
	}

	// Load model scores
	data, err := os.ReadFile(modelScoresFile)
	if err == nil {
		var savedScores map[string]*ModelScore
		if err := json.Unmarshal(data, &savedScores); err == nil {
			r.mu.Lock()
			for model, score := range savedScores {
				r.config.ModelScores[model] = score
			}
			r.mu.Unlock()
			routerLogger.Infof("Loaded %d model scores from persistence file", len(savedScores))
		} else {
			routerLogger.WithError(err).Warn("Failed to parse model scores file")
		}
	}

	// Load provider defaults
	defaultsData, err := os.ReadFile(providerDefaultsFile)
	if err == nil {
		var savedDefaults map[string]string
		if err := json.Unmarshal(defaultsData, &savedDefaults); err == nil {
			r.mu.Lock()
			for provider, model := range savedDefaults {
				r.config.ProviderDefaults[provider] = model
			}
			r.mu.Unlock()
			routerLogger.Infof("Loaded %d provider defaults from persistence file", len(savedDefaults))
		} else {
			routerLogger.WithError(err).Warn("Failed to parse provider defaults file")
		}
	}
}

// DefaultProviderDefaults returns default models for each provider
func DefaultProviderDefaults() map[string]string {
	return map[string]string{
		"deepseek":   "deepseek-chat",
		"openai":     "gpt-4o",
		"anthropic":  "claude-3-5-sonnet-20241022",
		"qwen":       "qwen-max",
		"zhipu":      "glm-4-plus",
		"moonshot":   "moonshot-v1-8k",
		"minimax":    "abab6.5s-chat",
		"baichuan":   "Baichuan4",
		"volcengine": "doubao-pro-128k",
		"google":     "gemini-2.0-flash",
	}
}

// GetConfig returns current router configuration
func (r *SmartRouter) GetConfig() *RouterConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// SetConfig updates router configuration and persists to file
func (r *SmartRouter) SetConfig(config *RouterConfig) {
	r.mu.Lock()
	r.config = config
	r.mu.Unlock()

	if err := r.SaveRouterConfigToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save router config to file")
	}
}

// SetStrategy sets the default routing strategy and persists to file
func (r *SmartRouter) SetStrategy(strategy StrategyType) {
	r.mu.Lock()
	r.config.DefaultStrategy = strategy
	r.mu.Unlock()

	if err := r.SaveRouterConfigToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save router config to file")
	}
}

// SetDefaultModel sets the default model and persists to file
func (r *SmartRouter) SetDefaultModel(model string) {
	r.mu.Lock()
	r.config.DefaultModel = model
	r.mu.Unlock()

	if err := r.SaveRouterConfigToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save router config to file")
	}
}

// SetUseAutoMode enables or disables auto mode and persists to file
func (r *SmartRouter) SetUseAutoMode(useAuto bool) {
	r.mu.Lock()
	r.config.UseAutoMode = useAuto
	r.mu.Unlock()

	if err := r.SaveRouterConfigToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save router config to file")
	}
}

// UpdateModelScore updates score for a specific model and persists to file
func (r *SmartRouter) UpdateModelScore(model string, score *ModelScore) {
	r.mu.Lock()
	r.config.ModelScores[model] = score
	r.mu.Unlock()

	// Save to file after updating (outside lock to avoid deadlock)
	if err := r.SaveToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save model scores to file")
	}
}

// DeleteModelScore deletes a model score and persists to file
func (r *SmartRouter) DeleteModelScore(model string) {
	r.mu.Lock()
	delete(r.config.ModelScores, model)
	r.mu.Unlock()

	// Save to file after deleting (outside lock to avoid deadlock)
	if err := r.SaveToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save model scores to file")
	}
}

// SelectModel selects the best model based on strategy and context
// Returns the selected model name
func (r *SmartRouter) SelectModel(requestedModel string, prompt string, availableModels []string) string {
	return r.SelectModelWithStrategy(requestedModel, r.config.DefaultStrategy, prompt, availableModels)
}

// SelectModelWithStrategy selects the best model using specified strategy
func (r *SmartRouter) SelectModelWithStrategy(requestedModel string, strategy StrategyType, prompt string, availableModels []string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If specific model requested and not "auto"/"latest", use it
	if requestedModel != "" && requestedModel != "auto" && requestedModel != "latest" {
		return requestedModel
	}

	// If auto mode is disabled and using auto request, use default model
	if !r.config.UseAutoMode && r.config.DefaultModel != "" && requestedModel == "auto" {
		return r.config.DefaultModel
	}

	// Filter to only available models
	availableSet := make(map[string]bool)
	for _, m := range availableModels {
		availableSet[m] = true
	}

	// Get candidates (enabled and available)
	var candidates []*ModelScore
	for model, score := range r.config.ModelScores {
		if score.Enabled && (len(availableSet) == 0 || availableSet[model]) {
			candidates = append(candidates, score)
		}
	}

	if len(candidates) == 0 {
		if r.config.DefaultModel != "" {
			return r.config.DefaultModel
		}
		return "deepseek-chat"
	}

	// Custom strategy: try to detect task type from prompt
	if strategy == StrategyCustom {
		detectedModel := r.detectTaskAndSelect(prompt, candidates)
		if detectedModel != "" {
			return detectedModel
		}
		strategy = StrategyAuto
	}

	// Sort candidates based on strategy
	sort.Slice(candidates, func(i, j int) bool {
		return r.calculateScore(candidates[i], strategy) > r.calculateScore(candidates[j], strategy)
	})

	return candidates[0].Model
}

// GetProviderForModel returns the provider for a given model
func (r *SmartRouter) GetProviderForModel(model string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if score, ok := r.config.ModelScores[model]; ok {
		return score.Provider
	}
	return ""
}

// GetProviderDefaults returns all provider default models
func (r *SmartRouter) GetProviderDefaults() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range r.config.ProviderDefaults {
		result[k] = v
	}
	return result
}

// SetProviderDefaults sets provider default models and persists to file
func (r *SmartRouter) SetProviderDefaults(defaults map[string]string) {
	r.mu.Lock()
	if r.config.ProviderDefaults == nil {
		r.config.ProviderDefaults = make(map[string]string)
	}
	for k, v := range defaults {
		r.config.ProviderDefaults[k] = v
	}
	r.mu.Unlock()

	// Save to file after updating (outside lock to avoid deadlock)
	if err := r.SaveProviderDefaultsToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save provider defaults to file")
	}
}

// GetProviderDefault returns the default model for a provider
func (r *SmartRouter) GetProviderDefault(provider string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.config.ProviderDefaults == nil {
		return ""
	}
	return r.config.ProviderDefaults[provider]
}

// SetProviderDefault sets the default model for a provider and persists to file
func (r *SmartRouter) SetProviderDefault(provider, model string) {
	r.mu.Lock()
	if r.config.ProviderDefaults == nil {
		r.config.ProviderDefaults = make(map[string]string)
	}
	r.config.ProviderDefaults[provider] = model
	r.mu.Unlock()

	// Save to file after updating (outside lock to avoid deadlock)
	if err := r.SaveProviderDefaultsToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save provider defaults to file")
	}
}

// SelectModelForProvider selects model for a specific provider
func (r *SmartRouter) SelectModelForProvider(requestedModel string, provider string, prompt string, availableModels []string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Handle special model names
	switch requestedModel {
	case "default":
		// Return provider's default model
		if r.config.ProviderDefaults != nil {
			if model, ok := r.config.ProviderDefaults[provider]; ok && model != "" {
				return model
			}
		}
		// Fallback to first available model for this provider
		return r.getFirstModelForProvider(provider, availableModels)

	case "latest":
		// Return highest quality model for this provider
		return r.getBestModelForProvider(provider, StrategyQuality, availableModels)

	case "auto":
		// Return best balanced model for this provider
		return r.getBestModelForProvider(provider, StrategyAuto, availableModels)

	default:
		// Return the specified model
		return requestedModel
	}
}

// getFirstModelForProvider returns the first available model for a provider
func (r *SmartRouter) getFirstModelForProvider(provider string, availableModels []string) string {
	availableSet := make(map[string]bool)
	for _, m := range availableModels {
		availableSet[m] = true
	}

	for model, score := range r.config.ModelScores {
		if score.Provider == provider && score.Enabled && (len(availableSet) == 0 || availableSet[model]) {
			return model
		}
	}
	return ""
}

// getBestModelForProvider returns the best model for a provider based on strategy
func (r *SmartRouter) getBestModelForProvider(provider string, strategy StrategyType, availableModels []string) string {
	availableSet := make(map[string]bool)
	for _, m := range availableModels {
		availableSet[m] = true
	}

	var candidates []*ModelScore
	for model, score := range r.config.ModelScores {
		if score.Provider == provider && score.Enabled && (len(availableSet) == 0 || availableSet[model]) {
			candidates = append(candidates, score)
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	sort.Slice(candidates, func(i, j int) bool {
		return r.calculateScore(candidates[i], strategy) > r.calculateScore(candidates[j], strategy)
	})

	return candidates[0].Model
}

// calculateScore calculates weighted score based on strategy
func (r *SmartRouter) calculateScore(score *ModelScore, strategy StrategyType) float64 {
	switch strategy {
	case StrategyQuality:
		return float64(score.QualityScore)
	case StrategySpeed:
		return float64(score.SpeedScore)
	case StrategyCost:
		return float64(score.CostScore)
	case StrategyAuto, StrategyCustom:
		// Balanced: 40% quality, 35% speed, 25% cost
		return float64(score.QualityScore)*0.4 + float64(score.SpeedScore)*0.35 + float64(score.CostScore)*0.25
	default:
		return float64(score.QualityScore)*0.4 + float64(score.SpeedScore)*0.35 + float64(score.CostScore)*0.25
	}
}

// detectTaskAndSelect detects task type from prompt and selects appropriate model
func (r *SmartRouter) detectTaskAndSelect(prompt string, candidates []*ModelScore) string {
	promptLower := prompt

	for _, rule := range r.config.TaskRules {
		for _, keyword := range rule.Keywords {
			if contains(promptLower, keyword) {
				// Find best model from preferred list that is available
				for _, preferredModel := range rule.PreferredModels {
					for _, c := range candidates {
						if c.Model == preferredModel {
							return c.Model
						}
					}
				}
			}
		}
	}

	return ""
}

// contains checks if s contains substr (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			subc := substr[j]
			// Simple lowercase comparison
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if subc >= 'A' && subc <= 'Z' {
				subc += 32
			}
			if sc != subc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// GetAllModelScores returns all model scores
func (r *SmartRouter) GetAllModelScores() map[string]*ModelScore {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*ModelScore)
	for k, v := range r.config.ModelScores {
		result[k] = v
	}
	return result
}

// GetAvailableModels returns list of enabled models
func (r *SmartRouter) GetAvailableModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []string
	for model, score := range r.config.ModelScores {
		if score.Enabled {
			models = append(models, model)
		}
	}

	// Sort by composite score
	sort.Slice(models, func(i, j int) bool {
		si, ok1 := r.config.ModelScores[models[i]]
		sj, ok2 := r.config.ModelScores[models[j]]
		if !ok1 || !ok2 {
			return models[i] < models[j]
		}
		return r.calculateScore(si, StrategyAuto) > r.calculateScore(sj, StrategyAuto)
	})

	return models
}

// GetTopModels returns top N models for a strategy
func (r *SmartRouter) GetTopModels(strategy StrategyType, n int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var candidates []*ModelScore
	for _, score := range r.config.ModelScores {
		if score.Enabled {
			candidates = append(candidates, score)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return r.calculateScore(candidates[i], strategy) > r.calculateScore(candidates[j], strategy)
	})

	var result []string
	for i := 0; i < int(math.Min(float64(n), float64(len(candidates)))); i++ {
		result = append(result, candidates[i].Model)
	}

	return result
}
