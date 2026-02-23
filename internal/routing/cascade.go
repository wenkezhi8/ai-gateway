// Package routing provides cascade routing strategy for cost optimization
// 改动点: 新增级联路由模块，支持小模型→大模型逐级升路
package routing

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CascadeLevel defines the level in cascade routing
type CascadeLevel string

const (
	CascadeLevelSmall  CascadeLevel = "small"  // 小模型：快速、低成本
	CascadeLevelMedium CascadeLevel = "medium" // 中等模型：平衡
	CascadeLevelLarge  CascadeLevel = "large"  // 大模型：高质量、高成本
)

// CascadeRule defines rules for cascade routing
type CascadeRule struct {
	TaskType        TaskType        `json:"task_type"`
	Difficulty      DifficultyLevel `json:"difficulty"`
	StartLevel      CascadeLevel    `json:"start_level"`       // 起始级别
	MaxLevel        CascadeLevel    `json:"max_level"`         // 最大级别
	FallbackEnabled bool            `json:"fallback_enabled"`  // 是否启用降级
	MaxRetries      int             `json:"max_retries"`       // 最大重试次数
	TimeoutPerLevel time.Duration   `json:"timeout_per_level"` // 每级别超时
}

// CascadeResult represents the result of cascade routing
type CascadeResult struct {
	SelectedModel string       `json:"selected_model"`
	Level         CascadeLevel `json:"level"`
	Attempts      int          `json:"attempts"`
	Success       bool         `json:"success"`
	FallbackUsed  bool         `json:"fallback_used"`
}

// CascadeRouter handles cascade routing logic
type CascadeRouter struct {
	mu          sync.RWMutex
	smartRouter *SmartRouter
	assessor    *DifficultyAssessor
	rules       map[string]*CascadeRule // key: "taskType:difficulty"
	modelLevels map[CascadeLevel][]string
	stats       map[string]*CascadeStats // key: model
}

// CascadeStats tracks statistics for cascade routing
type CascadeStats struct {
	TotalRequests   int64 `json:"total_requests"`
	SuccessAtSmall  int64 `json:"success_at_small"`
	SuccessAtMedium int64 `json:"success_at_medium"`
	SuccessAtLarge  int64 `json:"success_at_large"`
	Failures        int64 `json:"failures"`
	AvgLatencyMs    int64 `json:"avg_latency_ms"`
}

var cascadeLogger = logrus.WithField("component", "cascade_router")

// DefaultCascadeRules returns default cascade routing rules
func DefaultCascadeRules() map[string]*CascadeRule {
	return map[string]*CascadeRule{
		"chat:low": {
			TaskType:        TaskTypeChat,
			Difficulty:      DifficultyLow,
			StartLevel:      CascadeLevelSmall,
			MaxLevel:        CascadeLevelMedium,
			FallbackEnabled: true,
			MaxRetries:      2,
			TimeoutPerLevel: 10 * time.Second,
		},
		"chat:medium": {
			TaskType:        TaskTypeChat,
			Difficulty:      DifficultyMedium,
			StartLevel:      CascadeLevelMedium,
			MaxLevel:        CascadeLevelLarge,
			FallbackEnabled: true,
			MaxRetries:      2,
			TimeoutPerLevel: 15 * time.Second,
		},
		"chat:high": {
			TaskType:        TaskTypeChat,
			Difficulty:      DifficultyHigh,
			StartLevel:      CascadeLevelLarge,
			MaxLevel:        CascadeLevelLarge,
			FallbackEnabled: false,
			MaxRetries:      1,
			TimeoutPerLevel: 30 * time.Second,
		},
		"code:low": {
			TaskType:        TaskTypeCode,
			Difficulty:      DifficultyLow,
			StartLevel:      CascadeLevelSmall,
			MaxLevel:        CascadeLevelMedium,
			FallbackEnabled: true,
			MaxRetries:      2,
			TimeoutPerLevel: 15 * time.Second,
		},
		"code:medium": {
			TaskType:        TaskTypeCode,
			Difficulty:      DifficultyMedium,
			StartLevel:      CascadeLevelMedium,
			MaxLevel:        CascadeLevelLarge,
			FallbackEnabled: true,
			MaxRetries:      2,
			TimeoutPerLevel: 20 * time.Second,
		},
		"code:high": {
			TaskType:        TaskTypeCode,
			Difficulty:      DifficultyHigh,
			StartLevel:      CascadeLevelLarge,
			MaxLevel:        CascadeLevelLarge,
			FallbackEnabled: false,
			MaxRetries:      1,
			TimeoutPerLevel: 60 * time.Second,
		},
		"reasoning:low": {
			TaskType:        TaskTypeReasoning,
			Difficulty:      DifficultyLow,
			StartLevel:      CascadeLevelMedium,
			MaxLevel:        CascadeLevelLarge,
			FallbackEnabled: true,
			MaxRetries:      2,
			TimeoutPerLevel: 20 * time.Second,
		},
		"reasoning:medium": {
			TaskType:        TaskTypeReasoning,
			Difficulty:      DifficultyMedium,
			StartLevel:      CascadeLevelMedium,
			MaxLevel:        CascadeLevelLarge,
			FallbackEnabled: true,
			MaxRetries:      2,
			TimeoutPerLevel: 30 * time.Second,
		},
		"reasoning:high": {
			TaskType:        TaskTypeReasoning,
			Difficulty:      DifficultyHigh,
			StartLevel:      CascadeLevelLarge,
			MaxLevel:        CascadeLevelLarge,
			FallbackEnabled: false,
			MaxRetries:      1,
			TimeoutPerLevel: 120 * time.Second,
		},
	}
}

// DefaultModelLevels returns default model classifications by level
func DefaultModelLevels() map[CascadeLevel][]string {
	return map[CascadeLevel][]string{
		CascadeLevelSmall: {
			"gpt-4o-mini", "glm-4-flash", "qwen-turbo", "doubao-lite-32k",
			"deepseek-chat", "abab5.5-chat", "Baichuan3-Turbo",
		},
		CascadeLevelMedium: {
			"deepseek-coder", "gpt-4o", "claude-3-5-haiku-20241022",
			"qwen-plus", "glm-4", "moonshot-v1-8k", "abab6.5s-chat",
			"doubao-pro-128k", "gemini-2.0-flash",
		},
		CascadeLevelLarge: {
			"deepseek-reasoner", "o1", "o1-mini", "claude-3-5-sonnet-20241022",
			"claude-3-opus-20240229", "qwen-max", "glm-4-plus", "Baichuan4",
			"gemini-1.5-pro", "gpt-4-turbo",
		},
	}
}

// NewCascadeRouter creates a new cascade router
func NewCascadeRouter(smartRouter *SmartRouter, assessor *DifficultyAssessor) *CascadeRouter {
	return &CascadeRouter{
		smartRouter: smartRouter,
		assessor:    assessor,
		rules:       DefaultCascadeRules(),
		modelLevels: DefaultModelLevels(),
		stats:       make(map[string]*CascadeStats),
	}
}

// SelectCascadeModel selects a model using cascade strategy
// 改动点: 基于任务类型和难度选择级联路由策略
func (c *CascadeRouter) SelectCascadeModel(ctx context.Context, prompt string, context_ string, availableModels []string) *CascadeResult {
	assessment := c.assessor.AssessWithResult(prompt, context_)

	taskType := assessment.TaskType
	difficulty := assessment.Difficulty

	ruleKey := string(taskType) + ":" + string(difficulty)

	c.mu.RLock()
	rule, ok := c.rules[ruleKey]
	if !ok {
		rule = &CascadeRule{
			TaskType:        taskType,
			Difficulty:      difficulty,
			StartLevel:      CascadeLevelMedium,
			MaxLevel:        CascadeLevelLarge,
			FallbackEnabled: true,
			MaxRetries:      2,
			TimeoutPerLevel: 20 * time.Second,
		}
	}
	c.mu.RUnlock()

	result := &CascadeResult{
		Level:    rule.StartLevel,
		Attempts: 0,
	}

	availableSet := make(map[string]bool)
	for _, m := range availableModels {
		availableSet[m] = true
	}

	levels := []CascadeLevel{CascadeLevelSmall, CascadeLevelMedium, CascadeLevelLarge}
	startIdx := 0
	maxIdx := 2

	for i, l := range levels {
		if l == rule.StartLevel {
			startIdx = i
		}
		if l == rule.MaxLevel {
			maxIdx = i
		}
	}

	for i := startIdx; i <= maxIdx; i++ {
		level := levels[i]
		result.Level = level
		result.Attempts++

		model := c.selectBestModelForLevel(level, availableSet, taskType)
		if model != "" {
			result.SelectedModel = model
			result.FallbackUsed = i > startIdx
			result.Success = true

			cascadeLogger.WithFields(logrus.Fields{
				"model":      model,
				"level":      level,
				"task_type":  taskType,
				"difficulty": difficulty,
				"attempt":    result.Attempts,
			}).Info("Cascade routing selected model")

			return result
		}
	}

	result.Success = false
	result.SelectedModel = c.smartRouter.SelectModel("auto", prompt, availableModels)

	cascadeLogger.WithFields(logrus.Fields{
		"task_type":  taskType,
		"difficulty": difficulty,
		"fallback":   result.SelectedModel,
	}).Warn("Cascade routing fallback to smart router")

	return result
}

// selectBestModelForLevel selects the best available model for a given level
func (c *CascadeRouter) selectBestModelForLevel(level CascadeLevel, availableSet map[string]bool, taskType TaskType) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	models := c.modelLevels[level]

	taskModels := c.getModelsForTaskType(taskType, level)
	if len(taskModels) > 0 {
		for _, model := range taskModels {
			if len(availableSet) == 0 || availableSet[model] {
				scores := c.smartRouter.GetAllModelScores()
				if score, ok := scores[model]; ok && score.Enabled {
					return model
				}
			}
		}
	}

	for _, model := range models {
		if len(availableSet) == 0 || availableSet[model] {
			scores := c.smartRouter.GetAllModelScores()
			if score, ok := scores[model]; ok && score.Enabled {
				return model
			}
		}
	}

	return ""
}

// getModelsForTaskType returns preferred models for a task type at a given level
func (c *CascadeRouter) getModelsForTaskType(taskType TaskType, level CascadeLevel) []string {
	switch taskType {
	case TaskTypeCode:
		switch level {
		case CascadeLevelSmall:
			return []string{"deepseek-chat", "gpt-4o-mini"}
		case CascadeLevelMedium:
			return []string{"deepseek-coder", "claude-3-5-haiku-20241022"}
		case CascadeLevelLarge:
			return []string{"claude-3-5-sonnet-20241022", "gpt-4o", "o1-mini"}
		}
	case TaskTypeReasoning:
		switch level {
		case CascadeLevelSmall:
			return []string{"gpt-4o-mini", "glm-4-flash"}
		case CascadeLevelMedium:
			return []string{"gpt-4o", "qwen-plus"}
		case CascadeLevelLarge:
			return []string{"deepseek-reasoner", "o1", "claude-3-opus-20240229"}
		}
	case TaskTypeMath:
		switch level {
		case CascadeLevelSmall:
			return []string{"gpt-4o-mini", "deepseek-chat"}
		case CascadeLevelMedium:
			return []string{"gpt-4o", "qwen-plus"}
		case CascadeLevelLarge:
			return []string{"deepseek-reasoner", "o1"}
		}
	case TaskTypeCreative:
		switch level {
		case CascadeLevelSmall:
			return []string{"gpt-4o-mini", "qwen-turbo"}
		case CascadeLevelMedium:
			return []string{"gpt-4o", "claude-3-5-haiku-20241022"}
		case CascadeLevelLarge:
			return []string{"claude-3-5-sonnet-20241022", "claude-3-opus-20240229"}
		}
	}
	return nil
}

// RecordResult records the result of a cascade routing for statistics
func (c *CascadeRouter) RecordResult(model string, level CascadeLevel, success bool, latencyMs int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats, ok := c.stats[model]
	if !ok {
		stats = &CascadeStats{}
		c.stats[model] = stats
	}

	stats.TotalRequests++
	if success {
		switch level {
		case CascadeLevelSmall:
			stats.SuccessAtSmall++
		case CascadeLevelMedium:
			stats.SuccessAtMedium++
		case CascadeLevelLarge:
			stats.SuccessAtLarge++
		}
	} else {
		stats.Failures++
	}

	if stats.AvgLatencyMs == 0 {
		stats.AvgLatencyMs = latencyMs
	} else {
		stats.AvgLatencyMs = (stats.AvgLatencyMs + latencyMs) / 2
	}
}

// GetStats returns cascade routing statistics
func (c *CascadeRouter) GetStats() map[string]*CascadeStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*CascadeStats)
	for k, v := range c.stats {
		result[k] = v
	}
	return result
}

// SetRule sets a cascade routing rule
func (c *CascadeRouter) SetRule(rule *CascadeRule) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := string(rule.TaskType) + ":" + string(rule.Difficulty)
	c.rules[key] = rule
}

// GetRule returns a cascade routing rule
func (c *CascadeRouter) GetRule(taskType TaskType, difficulty DifficultyLevel) *CascadeRule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := string(taskType) + ":" + string(difficulty)
	if rule, ok := c.rules[key]; ok {
		return rule
	}
	return nil
}

// ShouldCascadeUp determines if should cascade to a larger model
// 改动点: 根据响应质量判断是否需要升级模型
func (c *CascadeRouter) ShouldCascadeUp(response string, err error, currentLevel CascadeLevel) bool {
	if err != nil {
		return true
	}

	if currentLevel == CascadeLevelLarge {
		return false
	}

	lowQualityIndicators := []string{
		"我无法", "I cannot", "无法回答",
		"不明确", "unclear", "需要更多信息",
		"超过", "exceeded", "limit",
	}

	for _, indicator := range lowQualityIndicators {
		if contains(response, indicator) {
			return true
		}
	}

	return false
}

// GetNextLevel returns the next cascade level
func (c *CascadeRouter) GetNextLevel(current CascadeLevel) CascadeLevel {
	switch current {
	case CascadeLevelSmall:
		return CascadeLevelMedium
	case CascadeLevelMedium:
		return CascadeLevelLarge
	default:
		return CascadeLevelLarge
	}
}

// GetModelLevel returns the cascade level for a model
func (c *CascadeRouter) GetModelLevel(model string) CascadeLevel {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for level, models := range c.modelLevels {
		for _, m := range models {
			if m == model {
				return level
			}
		}
	}
	return CascadeLevelMedium
}

// GetModelsForLevel returns all models for a level
func (c *CascadeRouter) GetModelsForLevel(level CascadeLevel) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	models := c.modelLevels[level]
	result := make([]string, len(models))
	copy(result, models)
	return result
}

// SetModelLevel sets the level for a model
func (c *CascadeRouter) SetModelLevel(model string, level CascadeLevel) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for l := range c.modelLevels {
		for i, m := range c.modelLevels[l] {
			if m == model {
				c.modelLevels[l] = append(c.modelLevels[l][:i], c.modelLevels[l][i+1:]...)
				break
			}
		}
	}

	c.modelLevels[level] = append(c.modelLevels[level], model)
}

// GetCascadeRules returns all cascade rules
func (c *CascadeRouter) GetCascadeRules() map[string]*CascadeRule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*CascadeRule)
	for k, v := range c.rules {
		ruleCopy := *v
		result[k] = &ruleCopy
	}
	return result
}

// GetCascadeRule returns a specific cascade rule
func (c *CascadeRouter) GetCascadeRule(taskType TaskType, difficulty DifficultyLevel) *CascadeRule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ruleKey := string(taskType) + ":" + string(difficulty)
	if rule, ok := c.rules[ruleKey]; ok {
		ruleCopy := *rule
		return &ruleCopy
	}
	return nil
}

// SetCascadeRule sets a cascade rule
func (c *CascadeRouter) SetCascadeRule(rule *CascadeRule) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ruleKey := string(rule.TaskType) + ":" + string(rule.Difficulty)
	c.rules[ruleKey] = rule

	cascadeLogger.WithFields(logrus.Fields{
		"task_type":   rule.TaskType,
		"difficulty":  rule.Difficulty,
		"start_level": rule.StartLevel,
		"max_level":   rule.MaxLevel,
	}).Info("Cascade rule updated")
}

// DeleteCascadeRule deletes a cascade rule
func (c *CascadeRouter) DeleteCascadeRule(taskType TaskType, difficulty DifficultyLevel) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	ruleKey := string(taskType) + ":" + string(difficulty)
	if _, ok := c.rules[ruleKey]; ok {
		delete(c.rules, ruleKey)
		return true
	}
	return false
}

// ResetCascadeRules resets all rules to default
func (c *CascadeRouter) ResetCascadeRules() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rules = DefaultCascadeRules()
	cascadeLogger.Info("Cascade rules reset to defaults")
}
