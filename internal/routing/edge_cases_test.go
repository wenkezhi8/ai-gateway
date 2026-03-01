package routing

import (
	"context"
	"testing"
	"time"
)

func TestCascadeRouter_EdgeCases(t *testing.T) {
	router := NewSmartRouter()
	cascade := router.GetCascadeRouter()

	t.Run("empty available models", func(t *testing.T) {
		ctx := context.Background()
		result := cascade.SelectCascadeModel(ctx, "test prompt", "", []string{})
		// Should fallback to smart router
		if result.SelectedModel == "" {
			t.Error("should have a fallback model")
		}
	})

	t.Run("long prompt", func(t *testing.T) {
		longPrompt := string(make([]byte, 10000))
		ctx := context.Background()
		result := cascade.SelectCascadeModel(ctx, longPrompt, "", []string{"gpt-4o", "deepseek-chat"})
		if !result.Success && result.SelectedModel == "" {
			t.Error("should handle long prompts")
		}
	})

	t.Run("context timeout", func(_ *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(1 * time.Millisecond)

		result := cascade.SelectCascadeModel(ctx, "test", "", []string{"gpt-4o"})
		// Should still work with background context internally
		_ = result
	})
}

func TestDifficultyAssessor_EdgeCases(t *testing.T) {
	assessor := NewDifficultyAssessor()

	t.Run("empty prompt", func(t *testing.T) {
		level := assessor.Assess("", "")
		if level == "" {
			t.Error("should return a difficulty level")
		}
	})

	t.Run("very long prompt", func(t *testing.T) {
		longPrompt := string(make([]byte, 10000))
		level := assessor.Assess(longPrompt, "")
		if level == "" {
			t.Error("should return a difficulty level")
		}
	})

	t.Run("unicode content", func(t *testing.T) {
		level := assessor.Assess("你好世界这是一个测试🎉🎊", "")
		if level == "" {
			t.Error("should handle unicode")
		}
	})

	t.Run("code with special chars", func(t *testing.T) {
		level := assessor.Assess("func main() { fmt.Println(\"Hello\\nWorld\") }", "")
		if level == "" {
			t.Error("should handle code")
		}
	})
}

func TestSmartRouter_EdgeCases(t *testing.T) {
	router := NewSmartRouter()

	t.Run("select with empty available models", func(t *testing.T) {
		model := router.SelectModel("auto", "test prompt", []string{})
		if model == "" {
			t.Error("should return a model")
		}
	})

	t.Run("select with nil available models", func(t *testing.T) {
		model := router.SelectModel("auto", "test prompt", nil)
		if model == "" {
			t.Error("should return a model")
		}
	})

	t.Run("select with disabled models only", func(t *testing.T) {
		scores := router.GetAllModelScores()
		for _, score := range scores {
			score.Enabled = false
		}

		model := router.SelectModel("auto", "test", []string{})
		// Should fallback to default
		if model == "" {
			t.Error("should return a fallback model")
		}

		// Re-enable for other tests
		for _, score := range scores {
			score.Enabled = true
		}
	})

	t.Run("provider not found", func(t *testing.T) {
		provider := router.GetProviderForModel("nonexistent-model-xyz")
		if provider != "" {
			t.Error("should return empty for unknown model")
		}
	})
}

func TestCascadeRouter_SetModelLevel(t *testing.T) {
	router := NewSmartRouter()
	cascade := router.GetCascadeRouter()

	cascade.SetModelLevel("new-model", CascadeLevelLarge)
	level := cascade.GetModelLevel("new-model")
	if level != CascadeLevelLarge {
		t.Errorf("expected large level, got %s", level)
	}

	// Change level
	cascade.SetModelLevel("new-model", CascadeLevelSmall)
	level = cascade.GetModelLevel("new-model")
	if level != CascadeLevelSmall {
		t.Errorf("expected small level, got %s", level)
	}
}

func TestDifficultyAssessor_UpdateSuccessRateConcurrent(t *testing.T) {
	assessor := NewDifficultyAssessor()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				success := (id+j)%2 == 0
				assessor.UpdateSuccessRate("concurrent-model", TaskTypeCode, success)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	rate := assessor.GetSuccessRate("concurrent-model", TaskTypeCode)
	if rate < 0 || rate > 1 {
		t.Errorf("invalid success rate: %f", rate)
	}
}

func TestSmartRouter_ProviderDefaults(t *testing.T) {
	router := NewSmartRouter()

	defaults := router.GetProviderDefaults()
	if len(defaults) == 0 {
		t.Error("should have default provider mappings")
	}

	router.SetProviderDefault("test-provider", "test-model")
	defaultModel := router.GetProviderDefault("test-provider")
	if defaultModel != "test-model" {
		t.Errorf("expected test-model, got %s", defaultModel)
	}

	// Override existing
	router.SetProviderDefault("openai", "gpt-4o-mini")
	defaultModel = router.GetProviderDefault("openai")
	if defaultModel != "gpt-4o-mini" {
		t.Errorf("expected gpt-4o-mini, got %s", defaultModel)
	}
}

func TestSmartRouter_TopModels(t *testing.T) {
	router := NewSmartRouter()

	top3 := router.GetTopModels(StrategyQuality, 3)
	if len(top3) == 0 {
		t.Error("should return top models")
	}
	if len(top3) > 3 {
		t.Errorf("should return at most 3 models, got %d", len(top3))
	}

	top10 := router.GetTopModels(StrategySpeed, 10)
	if len(top10) == 0 {
		t.Error("should return top models for speed strategy")
	}
}

func TestSmartRouter_AvailableModels(t *testing.T) {
	router := NewSmartRouter()

	models := router.GetAvailableModels()
	if len(models) == 0 {
		t.Error("should have available models")
	}

	// All returned models should be enabled
	scores := router.GetAllModelScores()
	for _, model := range models {
		if score, ok := scores[model]; ok {
			if !score.Enabled {
				t.Errorf("returned model %s is not enabled", model)
			}
		}
	}
}

func TestCascadeRouter_CascadeRuleGet(t *testing.T) {
	router := NewSmartRouter()
	cascade := router.GetCascadeRouter()

	rule := cascade.GetRule(TaskTypeCode, DifficultyHigh)
	if rule == nil {
		t.Fatal("should have rule for code:high")
	}
	if rule.StartLevel != CascadeLevelLarge {
		t.Errorf("expected large start level for high difficulty, got %s", rule.StartLevel)
	}

	// Non-existent rule
	rule = cascade.GetRule(TaskType("unknown"), DifficultyLow)
	if rule != nil {
		t.Error("should return nil for unknown task type")
	}
}
