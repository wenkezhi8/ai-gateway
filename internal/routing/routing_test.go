package routing

import (
	"context"
	"testing"
	"time"
)

func TestDifficultyAssessor_Assess(t *testing.T) {
	assessor := NewDifficultyAssessor()

	tests := []struct {
		name     string
		prompt   string
		context  string
		expected DifficultyLevel
	}{
		{
			name:     "simple greeting",
			prompt:   "你好",
			context:  "",
			expected: DifficultyLow,
		},
		{
			name:     "complex reasoning",
			prompt:   "请证明费马大定理，并详细推导每一步",
			context:  "",
			expected: DifficultyHigh,
		},
		{
			name:     "code task",
			prompt:   "写一个Python函数实现快速排序",
			context:  "",
			expected: DifficultyMedium,
		},
		{
			name:     "long context",
			prompt:   "总结这段文本",
			context:  string(make([]byte, 5000)),
			expected: DifficultyMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assessor.Assess(tt.prompt, tt.context)
			if result == "" {
				t.Error("result should not be empty")
			}
		})
	}
}

func TestDifficultyAssessor_DetectTaskType(t *testing.T) {
	assessor := NewDifficultyAssessor()

	tests := []struct {
		name     string
		prompt   string
		expected TaskType
	}{
		{
			name:     "code task",
			prompt:   "帮我写一个Python函数",
			expected: TaskTypeCode,
		},
		{
			name:     "math task",
			prompt:   "计算 123 * 456",
			expected: TaskTypeMath,
		},
		{
			name:     "translate task",
			prompt:   "请翻译这段文字",
			expected: TaskTypeTranslate,
		},
		{
			name:     "reasoning task",
			prompt:   "请推理这个逻辑问题",
			expected: TaskTypeReasoning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assessor.DetectTaskType(tt.prompt)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDifficultyAssessor_AssessWithResult(t *testing.T) {
	assessor := NewDifficultyAssessor()

	result := assessor.AssessWithResult("写一个复杂的分布式系统架构设计", "")

	if result.TaskType == TaskTypeUnknown {
		t.Error("task type should be detected")
	}
	if result.Confidence <= 0 {
		t.Error("confidence should be positive")
	}
}

func TestCascadeRouter_SelectCascadeModel(t *testing.T) {
	router := NewSmartRouter()
	cascade := router.GetCascadeRouter()

	tests := []struct {
		name            string
		prompt          string
		availableModels []string
		wantLevel       CascadeLevel
	}{
		{
			name:            "simple chat - should use small model",
			prompt:          "你好",
			availableModels: []string{"deepseek-chat", "gpt-4o-mini", "glm-4-flash"},
			wantLevel:       CascadeLevelSmall,
		},
		{
			name:            "complex reasoning - should use large model",
			prompt:          "请详细证明哥德巴赫猜想，并给出完整的数学推导过程",
			availableModels: []string{"deepseek-reasoner", "o1", "gpt-4o"},
			wantLevel:       CascadeLevelLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result := cascade.SelectCascadeModel(ctx, tt.prompt, "", tt.availableModels)

			if !result.Success {
				t.Error("selection should succeed")
			}
			if result.SelectedModel == "" {
				t.Error("selected model should not be empty")
			}
			if result.Attempts < 1 {
				t.Error("should have at least 1 attempt")
			}
		})
	}
}

func TestCascadeRouter_GetModelLevel(t *testing.T) {
	router := NewSmartRouter()
	cascade := router.GetCascadeRouter()

	tests := []struct {
		model    string
		expected CascadeLevel
	}{
		{"deepseek-chat", CascadeLevelSmall},
		{"gpt-4o", CascadeLevelMedium},
		{"deepseek-reasoner", CascadeLevelLarge},
		{"glm-4-flash", CascadeLevelSmall},
		{"claude-3-5-sonnet-20241022", CascadeLevelLarge},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			level := cascade.GetModelLevel(tt.model)
			if level != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, level)
			}
		})
	}
}

func TestSmartRouter_SelectModelWithAssessment(t *testing.T) {
	router := NewSmartRouter()

	model, assessment := router.SelectModelWithAssessment(
		"auto",
		"写一个Python快速排序算法",
		"",
		[]string{"deepseek-coder", "gpt-4o", "claude-3-5-sonnet-20241022"},
	)

	if model == "" {
		t.Error("model should not be empty")
	}
	if assessment == nil {
		t.Error("assessment should not be nil")
	}
	if assessment.TaskType != TaskTypeCode {
		t.Errorf("expected task type code, got %s", assessment.TaskType)
	}
}

func TestSmartRouter_SelectModelWithAssessment_TaskMappingPreferred(t *testing.T) {
	router := NewSmartRouter()
	router.SetTaskModelMapping(map[string]string{
		"code": "deepseek-coder",
	})

	model, assessment := router.SelectModelWithAssessment(
		"auto",
		"请写一个 Go 快速排序函数",
		"",
		[]string{"deepseek-coder", "gpt-4o"},
	)

	if assessment.TaskType != TaskTypeCode {
		t.Errorf("expected task type code, got %s", assessment.TaskType)
	}
	if model != "deepseek-coder" {
		t.Errorf("expected mapped model deepseek-coder, got %s", model)
	}
}

func TestSmartRouter_SelectModelWithAssessment_TaskMappingFallbackWhenUnavailable(t *testing.T) {
	router := NewSmartRouter()
	router.SetTaskModelMapping(map[string]string{
		"code": "deepseek-coder",
	})

	model, _ := router.SelectModelWithAssessment(
		"auto",
		"请写一个 Go 快速排序函数",
		"",
		[]string{"gpt-4o"},
	)

	if model == "deepseek-coder" {
		t.Errorf("expected fallback model when mapped model unavailable, got %s", model)
	}
}

func TestSmartRouter_SetTaskModelMapping_OtherAlias(t *testing.T) {
	router := NewSmartRouter()
	router.SetTaskModelMapping(map[string]string{
		"other": "gpt-4o-mini",
	})

	model := router.GetModelForTaskType(TaskTypeUnknown)
	if model != "gpt-4o-mini" {
		t.Errorf("expected alias model gpt-4o-mini, got %s", model)
	}
}

func TestSmartRouter_GetRecommendedTTL(t *testing.T) {
	router := NewSmartRouter()

	tests := []struct {
		name           string
		prompt         string
		minExpectedTTL time.Duration
	}{
		{
			name:           "fact query should have long TTL",
			prompt:         "中国的首都是哪里？",
			minExpectedTTL: 1 * time.Hour,
		},
		{
			name:           "creative writing should have zero TTL",
			prompt:         "请创作一篇关于春天的诗歌",
			minExpectedTTL: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttl := router.GetRecommendedTTL(tt.prompt, "")
			if ttl < tt.minExpectedTTL {
				t.Errorf("expected TTL >= %v, got %v", tt.minExpectedTTL, ttl)
			}
		})
	}
}

func TestCascadeRouter_RecordResult(t *testing.T) {
	router := NewSmartRouter()
	cascade := router.GetCascadeRouter()

	cascade.RecordResult("deepseek-chat", CascadeLevelSmall, true, 100)
	cascade.RecordResult("deepseek-chat", CascadeLevelSmall, true, 150)
	cascade.RecordResult("deepseek-chat", CascadeLevelSmall, false, 200)

	stats := cascade.GetStats()
	if stats["deepseek-chat"] == nil {
		t.Error("stats should exist for model")
	}
	if stats["deepseek-chat"].TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", stats["deepseek-chat"].TotalRequests)
	}
}

func TestDifficultyAssessor_UpdateSuccessRate(t *testing.T) {
	assessor := NewDifficultyAssessor()

	assessor.UpdateSuccessRate("test-model", TaskTypeCode, true)
	assessor.UpdateSuccessRate("test-model", TaskTypeCode, true)
	assessor.UpdateSuccessRate("test-model", TaskTypeCode, false)

	rate := assessor.GetSuccessRate("test-model", TaskTypeCode)
	if rate <= 0 || rate > 1 {
		t.Errorf("success rate should be between 0 and 1, got %f", rate)
	}
}

func TestCascadeRouter_ShouldCascadeUp(t *testing.T) {
	router := NewSmartRouter()
	cascade := router.GetCascadeRouter()

	tests := []struct {
		name     string
		response string
		err      error
		level    CascadeLevel
		shouldUp bool
	}{
		{
			name:     "error should cascade up",
			response: "",
			err:      context.DeadlineExceeded,
			level:    CascadeLevelSmall,
			shouldUp: true,
		},
		{
			name:     "low quality response should cascade up",
			response: "我无法回答这个问题",
			err:      nil,
			level:    CascadeLevelSmall,
			shouldUp: true,
		},
		{
			name:     "good response should not cascade up",
			response: "这是一个很好的答案",
			err:      nil,
			level:    CascadeLevelMedium,
			shouldUp: false,
		},
		{
			name:     "large level should not cascade up",
			response: "",
			err:      nil,
			level:    CascadeLevelLarge,
			shouldUp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cascade.ShouldCascadeUp(tt.response, tt.err, tt.level)
			if result != tt.shouldUp {
				t.Errorf("expected %v, got %v", tt.shouldUp, result)
			}
		})
	}
}
