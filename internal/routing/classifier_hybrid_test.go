package routing

import (
	"context"
	"testing"
)

type stubTaskClassifier struct {
	result *AssessmentResult
	err    error
	calls  int

	lastPrompt  string
	lastContext string
}

func (s *stubTaskClassifier) Classify(_ context.Context, prompt, contextText string) (*AssessmentResult, error) {
	s.calls++
	s.lastPrompt = prompt
	s.lastContext = contextText
	return s.result, s.err
}

func (s *stubTaskClassifier) Health(_ context.Context) *ClassifierHealth {
	return &ClassifierHealth{Healthy: true}
}

//nolint:gocritic // method signature follows TaskClassifier interface.
func (s *stubTaskClassifier) UpdateConfig(_ ClassifierConfig) {}

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

func TestHybridClassifier_ShortGreetingShortCircuit_NoLLMCall(t *testing.T) {
	assessor := NewDifficultyAssessor()
	cfg := DefaultClassifierConfig()
	cfg.Enabled = true

	hybrid := NewHybridTaskClassifier(assessor, cfg)
	stub := &stubTaskClassifier{
		result: &AssessmentResult{
			TaskType:   TaskTypeCode,
			Difficulty: DifficultyHigh,
			Confidence: 0.99,
		},
	}
	hybrid.classifier = stub

	result := hybrid.Classify(context.Background(), "[request_id=req-1] hi", "[session_id=s-1]")
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.TaskType != TaskTypeChat {
		t.Fatalf("expected chat task type, got %s", result.TaskType)
	}
	if result.Difficulty != DifficultyLow {
		t.Fatalf("expected low difficulty, got %s", result.Difficulty)
	}
	if result.Source != ClassificationSourceHeuristic {
		t.Fatalf("expected heuristic source, got %s", result.Source)
	}
	if result.FallbackReason != "greeting_short_circuit" {
		t.Fatalf("expected greeting short-circuit reason, got %s", result.FallbackReason)
	}
	if result.Confidence < 0.90 {
		t.Fatalf("expected confidence >= 0.90, got %.2f", result.Confidence)
	}
	if stub.calls != 0 {
		t.Fatalf("expected llm classifier not to be called, got %d calls", stub.calls)
	}
}

func TestHybridClassifier_SanitizesInputBeforeLLMCall(t *testing.T) {
	assessor := NewDifficultyAssessor()
	cfg := DefaultClassifierConfig()
	cfg.Enabled = true
	cfg.ConfidenceThreshold = 0.1

	hybrid := NewHybridTaskClassifier(assessor, cfg)
	stub := &stubTaskClassifier{
		result: &AssessmentResult{
			TaskType:   TaskTypeCode,
			Difficulty: DifficultyLow,
			Confidence: 0.99,
		},
	}
	hybrid.classifier = stub

	result := hybrid.Classify(context.Background(), "[request_id=req-1] write a unit test", "[conversation_id=c-1] context")
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if stub.calls != 1 {
		t.Fatalf("expected llm classifier called once, got %d", stub.calls)
	}
	if stub.lastPrompt != "write a unit test" {
		t.Fatalf("expected sanitized prompt, got %q", stub.lastPrompt)
	}
	if stub.lastContext != "context" {
		t.Fatalf("expected sanitized context, got %q", stub.lastContext)
	}
}
