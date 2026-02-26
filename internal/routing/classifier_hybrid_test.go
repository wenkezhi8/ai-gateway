package routing

import (
	"context"
	"testing"
)

type stubTaskClassifier struct {
	result *AssessmentResult
	err    error
}

func (s *stubTaskClassifier) Classify(ctx context.Context, prompt, contextText string) (*AssessmentResult, error) {
	return s.result, s.err
}

func (s *stubTaskClassifier) Health(ctx context.Context) *ClassifierHealth {
	return &ClassifierHealth{Healthy: true}
}

func (s *stubTaskClassifier) UpdateConfig(cfg ClassifierConfig) {}

func (s *stubTaskClassifier) GetConfig() ClassifierConfig { return DefaultClassifierConfig() }

func TestHybridClassifier_FallbackOnUnknownTaskType(t *testing.T) {
	assessor := NewDifficultyAssessor()
	cfg := DefaultClassifierConfig()
	cfg.Enabled = true
	cfg.ConfidenceThreshold = 0.1

	hybrid := NewHybridTaskClassifier(assessor, cfg)
	hybrid.classifier = &stubTaskClassifier{
		result: &AssessmentResult{
			TaskType:   TaskTypeUnknown,
			Difficulty: DifficultyLow,
			Confidence: 0.99,
		},
	}

	result := hybrid.Classify(context.Background(), "你好啊", "")
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.TaskType != TaskTypeChat {
		t.Fatalf("expected chat task type, got %s", result.TaskType)
	}
}
