package routing

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/constants"
	"ai-gateway/pkg/logger"

	"github.com/sirupsen/logrus"
)

const modelScoresFile = constants.ModelScoresFilePath
const providerDefaultsFile = constants.ProviderDefaultsFilePath
const routerConfigFile = constants.RouterConfigFilePath

var routerLogger = logger.WithField("component", "routing")

func normalizeProviderDefaults(defaults map[string]string) map[string]string {
	normalized := make(map[string]string)
	for provider, model := range defaults {
		normalizedProvider := strings.TrimSpace(strings.ToLower(provider))
		normalizedModel := strings.TrimSpace(model)
		if normalizedProvider == "" || normalizedModel == "" {
			continue
		}
		normalized[normalizedProvider] = normalizedModel
	}
	return normalized
}

func providerDefaultsNeedRewrite(raw map[string]string, normalized map[string]string) bool {
	if len(raw) != len(normalized) {
		return true
	}
	for provider, model := range raw {
		normalizedProvider := strings.TrimSpace(strings.ToLower(provider))
		normalizedModel := strings.TrimSpace(model)
		if normalizedProvider == "" || normalizedModel == "" {
			return true
		}
		if normalized[normalizedProvider] != normalizedModel {
			return true
		}
	}
	return false
}

// StrategyType defines the routing strategy.
type StrategyType string

const (
	StrategyAuto    StrategyType = "auto"    // Smart balance
	StrategyQuality StrategyType = "quality" // Best quality
	StrategySpeed   StrategyType = "speed"   // Fastest response
	StrategyCost    StrategyType = "cost"    // Lowest cost
	StrategyCustom  StrategyType = "custom"  // Custom rules
)

// ModelScore represents scoring for a model.
type ModelScore struct {
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	DisplayName  string `json:"display_name,omitempty"`
	QualityScore int    `json:"quality_score"` // 0-100
	SpeedScore   int    `json:"speed_score"`   // 0-100
	CostScore    int    `json:"cost_score"`    // 0-100 (higher = cheaper)
	Enabled      bool   `json:"enabled"`
}

// TaskRule defines model selection for specific task types.
type TaskRule struct {
	TaskType        string   `json:"task_type"`
	Keywords        []string `json:"keywords"`
	PreferredModels []string `json:"preferred_models"`
}

// RouterConfig holds the smart router configuration.
type RouterConfig struct {
	DefaultStrategy  StrategyType           `json:"default_strategy"`
	DefaultModel     string                 `json:"default_model"`
	UseAutoMode      bool                   `json:"use_auto_mode"`
	Classifier       ClassifierConfig       `json:"classifier"`
	ModelScores      map[string]*ModelScore `json:"model_scores"`
	TaskRules        []TaskRule             `json:"task_rules"`
	ProviderDefaults map[string]string      `json:"provider_defaults"` // provider -> default model
}

// SmartRouter handles intelligent model selection.
type SmartRouter struct {
	mu               sync.RWMutex
	config           *RouterConfig
	assessor         *DifficultyAssessor
	hybridClassifier *HybridTaskClassifier
	cascade          *CascadeRouter
	taskModelMapping map[string]string // task type -> model
}

func normalizeTaskMappingKey(taskType string) string {
	switch taskType {
	case "other":
		return string(TaskTypeUnknown)
	default:
		return taskType
	}
}

// DefaultModelScores returns default scoring for common models.
func DefaultModelScores() map[string]*ModelScore {
	defaults := make(map[string]*ModelScore, len(constants.RoutingDefaultModelScores))
	for key, preset := range constants.RoutingDefaultModelScores {
		defaults[key] = &ModelScore{
			Model:        preset.Model,
			Provider:     preset.Provider,
			QualityScore: preset.QualityScore,
			SpeedScore:   preset.SpeedScore,
			CostScore:    preset.CostScore,
			Enabled:      preset.Enabled,
		}
	}
	return defaults
}

// DefaultTaskRules returns default task-based routing rules.
func DefaultTaskRules() []TaskRule {
	rules := make([]TaskRule, 0, len(constants.RoutingDefaultTaskRules))
	for _, preset := range constants.RoutingDefaultTaskRules {
		rules = append(rules, TaskRule{
			TaskType:        preset.TaskType,
			Keywords:        append([]string{}, preset.Keywords...),
			PreferredModels: append([]string{}, preset.PreferredModels...),
		})
	}
	return rules
}

// NewSmartRouter creates a new smart router with default config.
func NewSmartRouter() *SmartRouter {
	router := &SmartRouter{
		config: &RouterConfig{
			DefaultStrategy:  StrategyType(constants.RoutingDefaultStrategy),
			DefaultModel:     constants.RoutingDefaultModel,
			UseAutoMode:      true,
			Classifier:       DefaultClassifierConfig(),
			ModelScores:      DefaultModelScores(),
			TaskRules:        DefaultTaskRules(),
			ProviderDefaults: DefaultProviderDefaults(),
		},
		assessor:         NewDifficultyAssessor(),
		taskModelMapping: make(map[string]string),
	}

	router.hybridClassifier = NewHybridTaskClassifier(router.assessor, router.config.Classifier)

	router.cascade = NewCascadeRouter(router, router.assessor)

	router.loadFromFile()

	if _, err := os.Stat(modelScoresFile); os.IsNotExist(err) {
		if saveErr := router.SaveToFile(); saveErr != nil {
			routerLogger.WithError(saveErr).Warn("Failed to save default model scores to file")
		}
		routerLogger.Info("Saved default model scores to file")
	}

	return router
}

// SaveToFile saves ALL model scores to a file (complete snapshot).
func (r *SmartRouter) SaveToFile() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Ensure data directory exists
	dir := filepath.Dir(modelScoresFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Save complete model scores, not just modified ones
	// This ensures deleted models stay deleted on next load
	data, err := json.MarshalIndent(r.config.ModelScores, "", "  ")
	if err != nil {
		return err
	}

	routerLogger.Infof("Saving %d model scores to file", len(r.config.ModelScores))
	return os.WriteFile(modelScoresFile, data, 0640)
}

// SaveProviderDefaultsToFile saves provider defaults to a file.
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

	return os.WriteFile(providerDefaultsFile, data, 0640)
}

// RouterConfigPersist is the structure saved to file for router config.
type RouterConfigPersist struct {
	DefaultStrategy string           `json:"default_strategy"`
	DefaultModel    string           `json:"default_model"`
	UseAutoMode     bool             `json:"use_auto_mode"`
	Classifier      ClassifierConfig `json:"classifier"`
}

// SaveRouterConfigToFile saves router config to a file.
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
		Classifier:      r.config.Classifier,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(routerConfigFile, data, 0640)
}

// loadFromFile loads model scores from the persistence file.
func (r *SmartRouter) loadFromFile() {
	// Load router config (strategy, default model, auto mode)
	configData, err := os.ReadFile(routerConfigFile)
	if err == nil {
		var savedConfig RouterConfigPersist
		if unmarshalErr := json.Unmarshal(configData, &savedConfig); unmarshalErr == nil {
			r.mu.Lock()
			if savedConfig.DefaultStrategy != "" {
				r.config.DefaultStrategy = StrategyType(savedConfig.DefaultStrategy)
			}
			if savedConfig.DefaultModel != "" {
				r.config.DefaultModel = savedConfig.DefaultModel
			}
			r.config.UseAutoMode = savedConfig.UseAutoMode
			r.config.Classifier = clampClassifierConfig(savedConfig.Classifier)
			r.mu.Unlock()
			if r.hybridClassifier != nil {
				r.hybridClassifier.UpdateConfig(r.config.Classifier)
			}
			routerLogger.Info("Loaded router config from persistence file")
		} else {
			routerLogger.WithError(unmarshalErr).Warn("Failed to parse router config file")
		}
	}

	// Load model scores - if file exists, completely replace defaults
	data, err := os.ReadFile(modelScoresFile)
	if err == nil && len(data) > 2 {
		var savedScores map[string]*ModelScore
		if unmarshalErr := json.Unmarshal(data, &savedScores); unmarshalErr == nil && len(savedScores) > 0 {
			r.mu.Lock()
			// Completely replace model scores with saved data
			// This ensures deleted models stay deleted
			r.config.ModelScores = savedScores
			r.mu.Unlock()
			routerLogger.Infof("Loaded %d model scores from persistence file (replaced defaults)", len(savedScores))
		} else {
			routerLogger.WithError(unmarshalErr).Warn("Failed to parse model scores file")
		}
	} else {
		routerLogger.Info("No saved model scores found, using defaults")
	}

	// Load provider defaults
	defaultsData, err := os.ReadFile(providerDefaultsFile)
	if err == nil {
		var savedDefaults map[string]string
		if err := json.Unmarshal(defaultsData, &savedDefaults); err == nil {
			normalizedDefaults := normalizeProviderDefaults(savedDefaults)
			needRewrite := providerDefaultsNeedRewrite(savedDefaults, normalizedDefaults)
			r.mu.Lock()
			r.config.ProviderDefaults = normalizedDefaults
			r.mu.Unlock()
			if needRewrite {
				if saveErr := r.SaveProviderDefaultsToFile(); saveErr != nil {
					routerLogger.WithError(saveErr).Warn("Failed to rewrite normalized provider defaults")
				}
			}
			routerLogger.Infof("Loaded %d provider defaults from persistence file", len(normalizedDefaults))
		} else {
			routerLogger.WithError(err).Warn("Failed to parse provider defaults file")
		}
	}
}

// DefaultProviderDefaults returns default models for each provider.
func DefaultProviderDefaults() map[string]string {
	defaults := make(map[string]string, len(constants.RoutingDefaultProviderDefaults))
	for provider, model := range constants.RoutingDefaultProviderDefaults {
		defaults[provider] = model
	}
	return defaults
}

// GetConfig returns current router configuration.
func (r *SmartRouter) GetConfig() *RouterConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// SetConfig updates router configuration and persists to file.
func (r *SmartRouter) SetConfig(config *RouterConfig) {
	r.mu.Lock()
	config.Classifier = clampClassifierConfig(config.Classifier)
	r.config = config
	r.mu.Unlock()
	if r.hybridClassifier != nil {
		r.hybridClassifier.UpdateConfig(config.Classifier)
	}

	if err := r.SaveRouterConfigToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save router config to file")
	}
}

// SetStrategy sets the default routing strategy and persists to file.
func (r *SmartRouter) SetStrategy(strategy StrategyType) {
	r.mu.Lock()
	r.config.DefaultStrategy = strategy
	r.mu.Unlock()

	if err := r.SaveRouterConfigToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save router config to file")
	}
}

// SetDefaultModel sets the default model and persists to file.
func (r *SmartRouter) SetDefaultModel(model string) {
	r.mu.Lock()
	r.config.DefaultModel = model
	r.mu.Unlock()

	if err := r.SaveRouterConfigToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save router config to file")
	}
}

// SetUseAutoMode enables or disables auto mode and persists to file.
func (r *SmartRouter) SetUseAutoMode(useAuto bool) {
	r.mu.Lock()
	r.config.UseAutoMode = useAuto
	r.mu.Unlock()

	if err := r.SaveRouterConfigToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save router config to file")
	}
}

func (r *SmartRouter) GetClassifierConfig() ClassifierConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.Classifier
}

//nolint:gocritic // keep value-type config API across router/classifier.
func (r *SmartRouter) SetClassifierConfig(cfg ClassifierConfig) {
	cfg = clampClassifierConfig(cfg)
	r.mu.Lock()
	r.config.Classifier = cfg
	r.mu.Unlock()
	if r.hybridClassifier != nil {
		r.hybridClassifier.UpdateConfig(cfg)
	}
	if err := r.SaveRouterConfigToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save router config to file")
	}
}

func (r *SmartRouter) SwitchClassifierModel(model string) {
	r.mu.Lock()
	r.config.Classifier.ActiveModel = model
	r.config.Classifier = clampClassifierConfig(r.config.Classifier)
	updated := r.config.Classifier
	r.mu.Unlock()
	if r.hybridClassifier != nil {
		r.hybridClassifier.UpdateConfig(updated)
	}
	if err := r.SaveRouterConfigToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save router config to file")
	}
}

func (r *SmartRouter) GetClassifierHealth(ctx context.Context) *ClassifierHealth {
	if r.hybridClassifier == nil {
		cfg := r.GetClassifierConfig()
		return &ClassifierHealth{Healthy: false, Model: cfg.ActiveModel, Provider: cfg.Provider, Message: "classifier not initialized", CheckedAt: time.Now().UnixMilli()}
	}
	return r.hybridClassifier.Health(ctx)
}

func (r *SmartRouter) GetClassifierStats() ClassifierStats {
	if r.hybridClassifier == nil {
		return ClassifierStats{}
	}
	return r.hybridClassifier.GetStats()
}

// UpdateModelScore updates score for a specific model and persists to file.
func (r *SmartRouter) UpdateModelScore(model string, score *ModelScore) {
	r.mu.Lock()
	r.config.ModelScores[model] = score
	r.mu.Unlock()

	// Save to file after updating (outside lock to avoid deadlock)
	if err := r.SaveToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save model scores to file")
	}
}

// DeleteModelScore deletes a model score and persists to file.
func (r *SmartRouter) DeleteModelScore(model string) {
	r.mu.Lock()
	delete(r.config.ModelScores, model)
	r.mu.Unlock()

	// Save to file after deleting (outside lock to avoid deadlock)
	if err := r.SaveToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save model scores to file")
	}
}

// RemoveProviderData removes all routing data for a provider and persists changes.
// Returns removed model count and whether provider default was removed.
func (r *SmartRouter) RemoveProviderData(provider string) (int, bool) {
	target := strings.TrimSpace(strings.ToLower(provider))
	if target == "" {
		return 0, false
	}

	r.mu.Lock()
	removedModels := 0
	for model, score := range r.config.ModelScores {
		if score == nil {
			continue
		}
		if strings.TrimSpace(strings.ToLower(score.Provider)) == target {
			delete(r.config.ModelScores, model)
			removedModels++
		}
	}

	removedDefault := false
	if r.config.ProviderDefaults != nil {
		for providerKey := range r.config.ProviderDefaults {
			if strings.TrimSpace(strings.ToLower(providerKey)) == target {
				delete(r.config.ProviderDefaults, providerKey)
				removedDefault = true
			}
		}
	}
	r.mu.Unlock()

	if removedModels > 0 {
		if err := r.SaveToFile(); err != nil {
			routerLogger.WithError(err).Warn("Failed to save model scores to file")
		}
	}
	if removedDefault {
		if err := r.SaveProviderDefaultsToFile(); err != nil {
			routerLogger.WithError(err).Warn("Failed to save provider defaults to file")
		}
	}

	return removedModels, removedDefault
}

// Returns the selected model name.
func (r *SmartRouter) SelectModel(requestedModel, prompt string, availableModels []string) string {
	r.mu.RLock()
	strategy := r.config.DefaultStrategy
	r.mu.RUnlock()

	return r.SelectModelWithStrategy(requestedModel, strategy, prompt, availableModels)
}

// SelectModelWithStrategy selects the best model using specified strategy.
//
//nolint:gocyclo // legacy branching; keep behavior stable in this lint cleanup.
func (r *SmartRouter) SelectModelWithStrategy(requestedModel string, strategy StrategyType, prompt string, availableModels []string) string {
	// If specific model requested and not "auto"/"latest", use it
	if requestedModel != "" && requestedModel != string(StrategyAuto) && requestedModel != "latest" {
		return requestedModel
	}

	// Filter to only available models
	availableSet := make(map[string]bool)
	for _, m := range availableModels {
		availableSet[m] = true
	}

	r.mu.RLock()
	defaultModel := r.config.DefaultModel
	useAutoMode := r.config.UseAutoMode

	// If auto mode is disabled and using auto request, use default model
	if !useAutoMode && defaultModel != "" && requestedModel == string(StrategyAuto) {
		r.mu.RUnlock()
		return defaultModel
	}

	// Get candidates (enabled and available)
	var candidates []*ModelScore
	for model, score := range r.config.ModelScores {
		if score != nil && score.Enabled && (len(availableSet) == 0 || availableSet[model]) {
			candidates = append(candidates, score)
		}
	}

	var taskRules []TaskRule
	if strategy == StrategyCustom {
		taskRules = append(taskRules, r.config.TaskRules...)
	}
	r.mu.RUnlock()

	if len(candidates) == 0 {
		if defaultModel != "" {
			return defaultModel
		}
		return constants.RoutingDefaultModel
	}

	// Custom strategy: try to detect task type from prompt
	if strategy == StrategyCustom {
		detectedModel := detectTaskAndSelect(prompt, candidates, taskRules)
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

// GetProviderForModel returns the provider for a given model.
func (r *SmartRouter) GetProviderForModel(model string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if score, ok := r.config.ModelScores[model]; ok {
		return score.Provider
	}
	return ""
}

// GetModelScore returns the model score for a given model.
func (r *SmartRouter) GetModelScore(model string) *ModelScore {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if score, ok := r.config.ModelScores[model]; ok {
		return score
	}
	return nil
}

// GetProviderDefaults returns all provider default models.
func (r *SmartRouter) GetProviderDefaults() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range r.config.ProviderDefaults {
		result[k] = v
	}
	return result
}

// SetProviderDefaults sets provider default models and persists to file.
func (r *SmartRouter) SetProviderDefaults(defaults map[string]string) {
	normalizedDefaults := normalizeProviderDefaults(defaults)
	r.mu.Lock()
	r.config.ProviderDefaults = normalizedDefaults
	r.mu.Unlock()

	// Save to file after updating (outside lock to avoid deadlock)
	if err := r.SaveProviderDefaultsToFile(); err != nil {
		routerLogger.WithError(err).Warn("Failed to save provider defaults to file")
	}
}

// GetProviderDefault returns the default model for a provider.
func (r *SmartRouter) GetProviderDefault(provider string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.config.ProviderDefaults == nil {
		return ""
	}
	return r.config.ProviderDefaults[provider]
}

// SetProviderDefault sets the default model for a provider and persists to file.
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

// SelectModelForProvider selects model for a specific provider.
func (r *SmartRouter) SelectModelForProvider(requestedModel, provider, _ string, availableModels []string) string {
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

// getFirstModelForProvider returns the first available model for a provider.
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

// getBestModelForProvider returns the best model for a provider based on strategy.
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

// calculateScore calculates weighted score based on strategy.
func (r *SmartRouter) calculateScore(score *ModelScore, strategy StrategyType) float64 {
	switch strategy {
	case StrategyQuality:
		return float64(score.QualityScore)
	case StrategySpeed:
		return float64(score.SpeedScore)
	case StrategyCost:
		return float64(score.CostScore)
	case StrategyAuto, StrategyCustom:
		return float64(score.QualityScore)*constants.RoutingAutoQualityWeight +
			float64(score.SpeedScore)*constants.RoutingAutoSpeedWeight +
			float64(score.CostScore)*constants.RoutingAutoCostWeight
	default:
		return float64(score.QualityScore)*constants.RoutingAutoQualityWeight +
			float64(score.SpeedScore)*constants.RoutingAutoSpeedWeight +
			float64(score.CostScore)*constants.RoutingAutoCostWeight
	}
}

// detectTaskAndSelect detects task type from prompt and selects appropriate model.
func detectTaskAndSelect(prompt string, candidates []*ModelScore, taskRules []TaskRule) string {
	promptLower := prompt

	for _, rule := range taskRules {
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

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || s != "" && containsSubstring(s, substr))
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

// GetAllModelScores returns all model scores.
func (r *SmartRouter) GetAllModelScores() map[string]*ModelScore {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*ModelScore)
	for k, v := range r.config.ModelScores {
		result[k] = v
	}
	return result
}

// GetAvailableModels returns list of enabled models.
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

// GetTopModels returns top N models for a strategy.
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

// GetDifficultyAssessor returns the difficulty assessor.
func (r *SmartRouter) GetDifficultyAssessor() *DifficultyAssessor {
	return r.assessor
}

// GetCascadeRouter returns the cascade router.
func (r *SmartRouter) GetCascadeRouter() *CascadeRouter {
	return r.cascade
}

// GetCascadeRules returns all cascade rules.
func (r *SmartRouter) GetCascadeRules() map[string]*CascadeRule {
	return r.cascade.GetCascadeRules()
}

// GetCascadeRule returns a specific cascade rule.
func (r *SmartRouter) GetCascadeRule(taskType TaskType, difficulty DifficultyLevel) *CascadeRule {
	return r.cascade.GetCascadeRule(taskType, difficulty)
}

// SetCascadeRule sets a cascade rule.
func (r *SmartRouter) SetCascadeRule(rule *CascadeRule) {
	r.cascade.SetCascadeRule(rule)
}

// DeleteCascadeRule deletes a cascade rule.
func (r *SmartRouter) DeleteCascadeRule(taskType TaskType, difficulty DifficultyLevel) bool {
	return r.cascade.DeleteCascadeRule(taskType, difficulty)
}

// ResetCascadeRules resets all cascade rules to defaults.
func (r *SmartRouter) ResetCascadeRules() {
	r.cascade.ResetCascadeRules()
}

// 改动点: 集成难度评估到模型选择.
func (r *SmartRouter) SelectModelWithAssessment(requestedModel, prompt, context string, availableModels []string) (string, *AssessmentResult) {
	assessment := r.classify(prompt, context)

	// 改动点: 任务类型映射优先于难度策略，确保 /routing 页面映射配置生效
	if requestedModel == "auto" || requestedModel == "" {
		if mappedModel := r.selectMappedModelForTask(assessment.TaskType, availableModels); mappedModel != "" {
			return mappedModel, assessment
		}
		if fitModel := r.selectModelByControlFit(assessment, availableModels); fitModel != "" {
			return fitModel, assessment
		}
	}

	difficultyStrategy := r.getStrategyForAssessment(assessment)

	selectedModel := r.SelectModelWithStrategy(requestedModel, difficultyStrategy, prompt, availableModels)

	return selectedModel, assessment
}

func (r *SmartRouter) selectMappedModelForTask(taskType TaskType, availableModels []string) string {
	mappedModel := r.GetModelForTaskType(taskType)
	if mappedModel == "" {
		return ""
	}

	availableSet := make(map[string]struct{}, len(availableModels))
	for _, m := range availableModels {
		availableSet[m] = struct{}{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	score, ok := r.config.ModelScores[mappedModel]
	if !ok || !score.Enabled {
		return ""
	}

	if len(availableSet) > 0 {
		if _, ok := availableSet[mappedModel]; !ok {
			return ""
		}
	}

	return mappedModel
}

//nolint:gocyclo // legacy branching; keep behavior stable in this lint cleanup.
func (r *SmartRouter) selectModelByControlFit(assessment *AssessmentResult, availableModels []string) string {
	if assessment == nil || assessment.ControlSignals == nil || len(assessment.ControlSignals.ModelFit) == 0 {
		return ""
	}

	cfg := r.GetClassifierConfig()
	if !cfg.Control.Enable || !cfg.Control.ModelFitEnable {
		return ""
	}
	applySelection := !cfg.Control.ShadowOnly

	availableSet := make(map[string]struct{}, len(availableModels))
	for _, model := range availableModels {
		availableSet[model] = struct{}{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	bestModel := ""
	bestScore := -1.0
	for model, score := range assessment.ControlSignals.ModelFit {
		if score < 0 || score > 1 {
			continue
		}
		if len(availableSet) > 0 {
			if _, ok := availableSet[model]; !ok {
				continue
			}
		}
		if ms, ok := r.config.ModelScores[model]; !ok || !ms.Enabled {
			continue
		}
		if score > bestScore {
			bestScore = score
			bestModel = model
		}
	}

	if bestModel != "" {
		routerLogger.WithFields(logrus.Fields{
			"model":       bestModel,
			"score":       bestScore,
			"task_type":   assessment.TaskType,
			"shadow_only": cfg.Control.ShadowOnly,
		}).Info("Control model-fit selected model")
	}
	if !applySelection {
		return ""
	}

	return bestModel
}

// 改动点: 使用级联路由策略选择模型.
func (r *SmartRouter) SelectModelCascade(ctx context.Context, prompt, context string, availableModels []string) *CascadeResult {
	return r.cascade.SelectCascadeModel(ctx, prompt, context, availableModels)
}

// getStrategyForDifficulty returns recommended strategy based on difficulty.
func (r *SmartRouter) getStrategyForDifficulty(difficulty DifficultyLevel) StrategyType {
	switch difficulty {
	case DifficultyLow:
		return StrategySpeed
	case DifficultyMedium:
		return StrategyAuto
	case DifficultyHigh:
		return StrategyQuality
	default:
		return StrategyAuto
	}
}

func (r *SmartRouter) getStrategyForAssessment(assessment *AssessmentResult) StrategyType {
	if assessment != nil {
		switch assessment.RouteHint {
		case "speed":
			return StrategySpeed
		case "quality", "reasoning_first":
			return StrategyQuality
		case "balanced":
			return StrategyAuto
		}
		if strategy, ok := r.getStrategyForControlSignals(assessment); ok {
			return strategy
		}
		return r.getStrategyForDifficulty(assessment.Difficulty)
	}
	return StrategyAuto
}

func (r *SmartRouter) getStrategyForControlSignals(assessment *AssessmentResult) (StrategyType, bool) {
	if assessment == nil || assessment.ControlSignals == nil {
		return StrategyAuto, false
	}
	cfg := r.GetClassifierConfig()
	if !cfg.Control.Enable || cfg.Control.ShadowOnly {
		return StrategyAuto, false
	}

	contextLoad := strings.TrimSpace(strings.ToLower(assessment.ControlSignals.ContextLoad))
	switch contextLoad {
	case string(DifficultyHigh):
		return StrategyQuality, true
	case string(DifficultyLow):
		return StrategySpeed, true
	case string(DifficultyMedium):
		return StrategyAuto, true
	default:
		return StrategyAuto, false
	}
}

// AssessDifficulty assesses the difficulty of a prompt.
func (r *SmartRouter) AssessDifficulty(prompt, context string) *AssessmentResult {
	return r.classify(prompt, context)
}

// UpdateModelSuccessRate updates the success rate for a model.
func (r *SmartRouter) UpdateModelSuccessRate(model string, taskType TaskType, success bool) {
	r.assessor.UpdateSuccessRate(model, taskType, success)
}

// GetRecommendedTTL returns recommended cache TTL for a request.
func (r *SmartRouter) GetRecommendedTTL(prompt, context string) time.Duration {
	assessment := r.classify(prompt, context)
	return assessment.SuggestedTTL
}

func (r *SmartRouter) classify(prompt, contextText string) *AssessmentResult {
	if r.hybridClassifier == nil {
		return r.assessor.AssessWithResult(prompt, contextText)
	}
	ctx, cancel := context.WithTimeout(context.Background(), classifierTimeout(r.GetClassifierConfig()))
	defer cancel()
	result := r.hybridClassifier.Classify(ctx, prompt, contextText)
	if result == nil {
		return r.assessor.AssessWithResult(prompt, contextText)
	}
	return result
}

// GetTaskModelMapping returns the task type to model mapping.
func (r *SmartRouter) GetTaskModelMapping() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range r.taskModelMapping {
		result[k] = v
	}
	return result
}

// SetTaskModelMapping sets the task type to model mapping.
func (r *SmartRouter) SetTaskModelMapping(mapping map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.taskModelMapping = make(map[string]string)
	for k, v := range mapping {
		normalizedKey := normalizeTaskMappingKey(k)
		r.taskModelMapping[normalizedKey] = v
	}

	routerLogger.WithField("mapping", mapping).Info("Task model mapping updated")
}

// GetModelForTaskType returns the model for a specific task type.
func (r *SmartRouter) GetModelForTaskType(taskType TaskType) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if model := r.taskModelMapping[string(taskType)]; model != "" {
		return model
	}

	if taskType == TaskTypeUnknown {
		// Backward compatibility: old frontend may persist key "other".
		return r.taskModelMapping["other"]
	}

	return ""
}
