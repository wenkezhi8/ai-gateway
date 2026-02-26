// Package routing provides intelligent routing based on task type and difficulty
// 改动点: 新增任务难度评估模块
package routing

import (
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

var mathExpressionPattern = regexp.MustCompile(`(?i)(\d+\s*[+\-*/x×÷＋－＊／]\s*\d+|\d+\s*\^\s*\d+|\d+\s*=\s*\d+|\b(sqrt|sin|cos|tan|log|ln)\s*\()`)

// DifficultyLevel represents the difficulty level of a task
type DifficultyLevel string

const (
	DifficultyLow    DifficultyLevel = "low"
	DifficultyMedium DifficultyLevel = "medium"
	DifficultyHigh   DifficultyLevel = "high"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeChat      TaskType = "chat"
	TaskTypeCode      TaskType = "code"
	TaskTypeReasoning TaskType = "reasoning"
	TaskTypeCreative  TaskType = "creative"
	TaskTypeFact      TaskType = "fact"
	TaskTypeLongText  TaskType = "long_text"
	TaskTypeMath      TaskType = "math"
	TaskTypeTranslate TaskType = "translate"
	TaskTypeUnknown   TaskType = "unknown"
)

// DifficultyAssessor assesses the difficulty of a prompt
type DifficultyAssessor struct {
	mu sync.RWMutex

	// 评估参数
	lengthThresholds map[DifficultyLevel]int
	complexPatterns  map[DifficultyLevel][]*regexp.Regexp
	taskKeywords     map[TaskType][]string

	// 历史数据
	historySuccessRate map[string]float64 // key: "model:taskType"

	// TTL配置
	ttlConfig *TTLConfig
}

// TTLConfig represents TTL configuration for different task types
type TTLConfig struct {
	TaskTypeDefaults      map[TaskType]time.Duration  `json:"task_type_defaults"`
	DifficultyMultipliers map[DifficultyLevel]float64 `json:"difficulty_multipliers"`
}

// DefaultTTLConfig returns the default TTL configuration
func DefaultTTLConfig() *TTLConfig {
	return &TTLConfig{
		TaskTypeDefaults: map[TaskType]time.Duration{
			TaskTypeFact:      24 * time.Hour,  // 事实查询缓存1天
			TaskTypeCode:      168 * time.Hour, // 代码缓存7天
			TaskTypeMath:      720 * time.Hour, // 数学缓存30天
			TaskTypeChat:      1 * time.Hour,   // 闲聊缓存1小时
			TaskTypeCreative:  0,               // 创意类不缓存
			TaskTypeReasoning: 168 * time.Hour, // 推理缓存7天
			TaskTypeLongText:  360 * time.Hour, // 长文本缓存15天
			TaskTypeTranslate: 72 * time.Hour,  // 翻译缓存3天
			TaskTypeUnknown:   24 * time.Hour,  // 未知类型缓存1天
		},
		DifficultyMultipliers: map[DifficultyLevel]float64{
			DifficultyLow:    0.5, // 低难度 TTL 减半
			DifficultyMedium: 1.0, // 中难度保持默认
			DifficultyHigh:   2.0, // 高难度 TTL 翻倍
		},
	}
}

// GetTTLConfig returns the current TTL configuration
func (a *DifficultyAssessor) GetTTLConfig() *TTLConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.ttlConfig == nil {
		return DefaultTTLConfig()
	}
	return a.ttlConfig
}

// SetTTLConfig sets the TTL configuration
func (a *DifficultyAssessor) SetTTLConfig(config *TTLConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ttlConfig = config
}

// ParseDuration parses hours to time.Duration
func ParseDuration(hours int) time.Duration {
	return time.Duration(hours) * time.Hour
}

// NewDifficultyAssessor creates a new difficulty assessor
func NewDifficultyAssessor() *DifficultyAssessor {
	a := &DifficultyAssessor{
		lengthThresholds: map[DifficultyLevel]int{
			DifficultyLow:    500,
			DifficultyMedium: 2000,
			DifficultyHigh:   8000,
		},
		historySuccessRate: make(map[string]float64),
	}

	a.initComplexPatterns()
	a.initTaskKeywords()

	return a
}

func (a *DifficultyAssessor) initComplexPatterns() {
	// 高难度模式
	highPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(证明|推导|分析|论证|比较|评估|设计|架构|优化)`),
		regexp.MustCompile(`(?i)(多步|复杂|嵌套|递归|算法)`),
		regexp.MustCompile(`(?i)(prove|derive|analyze|design|architecture|optimize)`),
		regexp.MustCompile(`(?i)(step.{0,5}by.{0,5}step|multi.{0,5}step)`),
	}

	// 中等难度模式
	mediumPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(解释|说明|描述|总结|概括)`),
		regexp.MustCompile(`(?i)(写一个|编写|实现|生成)`),
		regexp.MustCompile(`(?i)(explain|describe|implement|create)`),
		regexp.MustCompile(`(?i)(如何|怎么|怎样|why|how|what)`),
	}

	a.complexPatterns = map[DifficultyLevel][]*regexp.Regexp{
		DifficultyHigh:   highPatterns,
		DifficultyMedium: mediumPatterns,
	}
}

func (a *DifficultyAssessor) initTaskKeywords() {
	a.taskKeywords = map[TaskType][]string{
		TaskTypeCode: {
			"代码", "函数", "程序", "bug", "调试", "编程", "实现",
			"code", "function", "program", "debug", "implement", "python", "javascript", "golang",
		},
		TaskTypeReasoning: {
			"推理", "证明", "逻辑", "分析", "思考", "判断",
			"reasoning", "prove", "logic", "analyze", "think",
		},
		TaskTypeCreative: {
			"创作", "写", "故事", "文章", "文案", "创意", "想象",
			"write", "create", "story", "article", "creative", "imagine",
		},
		TaskTypeFact: {
			"是什么", "多少", "什么时候", "在哪里", "谁", "事实",
			"what is", "how many", "when", "where", "who", "fact",
		},
		TaskTypeMath: {
			"计算", "数学", "加", "减", "乘", "除", "方程", "公式", "求解", "运算", "等于",
			"calculate", "math", "equation", "formula", "solve", "arithmetic",
		},
		TaskTypeTranslate: {
			"翻译", "译", "translate", "translation",
		},
		TaskTypeChat: {
			"你好", "hello", "hi", "早上好", "晚安", "谢谢",
		},
	}
}

// Assess evaluates the difficulty of a prompt
// 改动点: 基于多维度评估难度
func (a *DifficultyAssessor) Assess(prompt string, context string) DifficultyLevel {
	score := 0.0

	// 1. 长度评估 (0-30分)
	lengthScore := a.assessLength(prompt, context)
	score += lengthScore * 0.3

	// 2. 复杂度模式评估 (0-40分)
	complexityScore := a.assessComplexity(prompt)
	score += complexityScore * 0.4

	// 3. 任务类型评估 (0-20分)
	taskScore := a.assessByTaskType(prompt)
	score += taskScore * 0.2

	// 4. 历史成功率调整 (±10分)
	historyAdjust := a.assessByHistory(prompt)
	score += historyAdjust * 0.1

	// 转换为难度等级
	if score >= 70 {
		return DifficultyHigh
	} else if score >= 40 {
		return DifficultyMedium
	}
	return DifficultyLow
}

func (a *DifficultyAssessor) assessLength(prompt string, context string) float64 {
	totalLen := len(prompt) + len(context)

	if totalLen > a.lengthThresholds[DifficultyHigh] {
		return 100
	} else if totalLen > a.lengthThresholds[DifficultyMedium] {
		return 60 + float64(totalLen-a.lengthThresholds[DifficultyMedium])/float64(a.lengthThresholds[DifficultyHigh]-a.lengthThresholds[DifficultyMedium])*40
	} else if totalLen > a.lengthThresholds[DifficultyLow] {
		return 20 + float64(totalLen-a.lengthThresholds[DifficultyLow])/float64(a.lengthThresholds[DifficultyMedium]-a.lengthThresholds[DifficultyLow])*40
	}
	return 20
}

func (a *DifficultyAssessor) assessComplexity(prompt string) float64 {
	maxScore := 0.0

	// 检查高难度模式
	for _, pattern := range a.complexPatterns[DifficultyHigh] {
		if pattern.MatchString(prompt) {
			maxScore = math.Max(maxScore, 90)
			break
		}
	}

	// 检查中等难度模式
	if maxScore < 90 {
		for _, pattern := range a.complexPatterns[DifficultyMedium] {
			if pattern.MatchString(prompt) {
				maxScore = math.Max(maxScore, 50)
				break
			}
		}
	}

	// 检查问题数量
	questionCount := strings.Count(prompt, "?") + strings.Count(prompt, "？")
	if questionCount >= 3 {
		maxScore = math.Max(maxScore, 70)
	} else if questionCount >= 2 {
		maxScore = math.Max(maxScore, 50)
	}

	// 检查代码块
	if strings.Contains(prompt, "```") || strings.Contains(prompt, "code") {
		maxScore = math.Max(maxScore, 60)
	}

	return maxScore
}

func (a *DifficultyAssessor) assessByTaskType(prompt string) float64 {
	taskType := a.DetectTaskType(prompt)

	switch taskType {
	case TaskTypeReasoning, TaskTypeMath:
		return 80
	case TaskTypeCode:
		return 60
	case TaskTypeCreative:
		return 50
	case TaskTypeLongText:
		return 70
	case TaskTypeTranslate:
		return 40
	case TaskTypeChat, TaskTypeFact:
		return 20
	default:
		return 40
	}
}

func (a *DifficultyAssessor) assessByHistory(prompt string) float64 {
	taskType := a.DetectTaskType(prompt)

	// 检查历史成功率
	key := "default:" + string(taskType)
	if rate, ok := a.historySuccessRate[key]; ok {
		if rate < 0.7 {
			return 20 // 历史成功率低，增加难度
		} else if rate > 0.95 {
			return -10 // 历史成功率高，降低难度
		}
	}
	return 0
}

// DetectTaskType detects the type of task from prompt
func (a *DifficultyAssessor) DetectTaskType(prompt string) TaskType {
	promptLower := strings.ToLower(prompt)

	if looksLikeMathExpression(prompt) {
		return TaskTypeMath
	}

	// 按优先级检测
	priorityOrder := []TaskType{
		TaskTypeCode,
		TaskTypeMath,
		TaskTypeReasoning,
		TaskTypeTranslate,
		TaskTypeCreative,
		TaskTypeFact,
		TaskTypeChat,
	}

	for _, taskType := range priorityOrder {
		keywords := a.taskKeywords[taskType]
		for _, keyword := range keywords {
			if strings.Contains(promptLower, strings.ToLower(keyword)) {
				return taskType
			}
		}
	}

	// 长文本检测
	if len(prompt) > 4000 {
		return TaskTypeLongText
	}

	return TaskTypeUnknown
}

func looksLikeMathExpression(prompt string) bool {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return false
	}

	return mathExpressionPattern.MatchString(trimmed)
}

// UpdateSuccessRate updates the historical success rate for a model and task type
func (a *DifficultyAssessor) UpdateSuccessRate(model string, taskType TaskType, success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := model + ":" + string(taskType)

	currentRate, ok := a.historySuccessRate[key]
	if !ok {
		if success {
			a.historySuccessRate[key] = 1.0
		} else {
			a.historySuccessRate[key] = 0.0
		}
		return
	}

	// 滑动窗口更新
	alpha := 0.1
	if success {
		a.historySuccessRate[key] = currentRate*(1-alpha) + 1.0*alpha
	} else {
		a.historySuccessRate[key] = currentRate*(1-alpha) + 0.0*alpha
	}
}

// GetSuccessRate returns the success rate for a model and task type
func (a *DifficultyAssessor) GetSuccessRate(model string, taskType TaskType) float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	key := model + ":" + string(taskType)
	if rate, ok := a.historySuccessRate[key]; ok {
		return rate
	}
	return 0.8 // 默认80%
}

// AssessmentResult represents the result of difficulty assessment
type AssessmentResult struct {
	TaskType          TaskType             `json:"task_type"`
	Difficulty        DifficultyLevel      `json:"difficulty"`
	Confidence        float64              `json:"confidence"`
	Dimensions        map[string]float64   `json:"dimensions"`
	SuggestedTTL      time.Duration        `json:"suggested_ttl"`
	SemanticSignature string               `json:"semantic_signature,omitempty"`
	ControlSignals    *ControlSignals      `json:"control_signals,omitempty"`
	RouteHint         string               `json:"route_hint,omitempty"`
	Source            ClassificationSource `json:"source,omitempty"`
	FallbackReason    string               `json:"fallback_reason,omitempty"`
}

type ControlSignals struct {
	ControlVersion         string             `json:"control_version,omitempty"`
	NormalizedQuery        string             `json:"normalized_query,omitempty"`
	QueryStabilityScore    float64            `json:"query_stability_score,omitempty"`
	Cacheable              *bool              `json:"cacheable,omitempty"`
	CacheReason            string             `json:"cache_reason,omitempty"`
	TTLBand                string             `json:"ttl_band,omitempty"`
	RiskLevel              string             `json:"risk_level,omitempty"`
	RiskTags               []string           `json:"risk_tags,omitempty"`
	ToolNeeded             *bool              `json:"tool_needed,omitempty"`
	RAGNeeded              *bool              `json:"rag_needed,omitempty"`
	ModelFit               map[string]float64 `json:"model_fit,omitempty"`
	ContextLoad            string             `json:"context_load,omitempty"`
	RecommendedTemperature *float64           `json:"recommended_temperature,omitempty"`
	RecommendedTopP        *float64           `json:"recommended_top_p,omitempty"`
	RecommendedMaxTokens   *int               `json:"recommended_max_tokens,omitempty"`
	ExperimentTag          string             `json:"experiment_tag,omitempty"`
	DomainTag              string             `json:"domain_tag,omitempty"`
}

// AssessWithResult returns detailed assessment result
func (a *DifficultyAssessor) AssessWithResult(prompt string, context string) *AssessmentResult {
	taskType := a.DetectTaskType(prompt)
	difficulty := a.Assess(prompt, context)

	// 计算置信度
	confidence := 0.7
	if taskType != TaskTypeUnknown {
		confidence += 0.1
	}
	if len(prompt) > 100 {
		confidence += 0.1
	}
	if len(context) > 0 {
		confidence += 0.1
	}

	// 建议的缓存 TTL
	suggestedTTL := a.getSuggestedTTL(taskType, difficulty)

	// 各维度得分
	dimensions := map[string]float64{
		"length":     a.assessLength(prompt, context),
		"complexity": a.assessComplexity(prompt),
		"task_type":  a.assessByTaskType(prompt),
	}

	return &AssessmentResult{
		TaskType:          taskType,
		Difficulty:        difficulty,
		Confidence:        confidence,
		Dimensions:        dimensions,
		SuggestedTTL:      suggestedTTL,
		SemanticSignature: buildFallbackSignature(taskType, prompt),
		Source:            ClassificationSourceHeuristic,
	}
}

func (a *DifficultyAssessor) getSuggestedTTL(taskType TaskType, difficulty DifficultyLevel) time.Duration {
	config := a.GetTTLConfig()

	baseTTL, ok := config.TaskTypeDefaults[taskType]
	if !ok {
		baseTTL = config.TaskTypeDefaults[TaskTypeUnknown]
	}

	// 创意类不缓存
	if taskType == TaskTypeCreative {
		return 0
	}

	// 根据难度调整
	multiplier, ok := config.DifficultyMultipliers[difficulty]
	if !ok {
		multiplier = 1.0
	}

	return time.Duration(float64(baseTTL) * multiplier)
}
