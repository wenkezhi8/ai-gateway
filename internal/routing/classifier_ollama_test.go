package routing

import (
	"errors"
	"testing"
)

func TestParseClassifierOutput_WithControlSignals(t *testing.T) {
	raw := `{
  "task_type":"code",
  "difficulty":"medium",
  "confidence":0.88,
  "semantic_signature":"code:sort",
  "route_hint":"quality",
  "control_version":"v1",
  "normalized_query":"write quick sort in go",
  "query_stability_score":0.92,
  "cacheable":false,
  "cache_reason":"sensitive",
  "ttl_band":"short",
  "risk_level":"medium",
  "risk_tags":["prompt_injection","prompt_injection"],
  "tool_needed":true,
  "rag_needed":false,
  "context_load":"high",
  "model_fit":{"gpt-4o-mini":0.72,"invalid":1.8},
  "recommended_temperature":0.2,
  "recommended_top_p":0.9,
  "recommended_max_tokens":1024,
  "experiment_tag":"ctrl-exp-a",
  "domain_tag":"coding"
}`

	result, err := parseClassifierOutput(raw)
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}

	if result.TaskType != TaskTypeCode {
		t.Fatalf("expected task type code, got %s", result.TaskType)
	}
	if result.ControlSignals == nil {
		t.Fatal("expected control signals")
	}
	if result.ControlSignals.NormalizedQuery != "write quick sort in go" {
		t.Fatalf("unexpected normalized query: %s", result.ControlSignals.NormalizedQuery)
	}
	if result.ControlSignals.TTLBand != "short" {
		t.Fatalf("expected ttl band short, got %s", result.ControlSignals.TTLBand)
	}
	if result.ControlSignals.RiskLevel != "medium" {
		t.Fatalf("expected risk level medium, got %s", result.ControlSignals.RiskLevel)
	}
	if len(result.ControlSignals.RiskTags) != 1 || result.ControlSignals.RiskTags[0] != "prompt_injection" {
		t.Fatalf("expected deduped risk tags, got %#v", result.ControlSignals.RiskTags)
	}
	if result.ControlSignals.ToolNeeded == nil || !*result.ControlSignals.ToolNeeded {
		t.Fatal("expected tool_needed=true")
	}
	if result.ControlSignals.RAGNeeded == nil || *result.ControlSignals.RAGNeeded {
		t.Fatal("expected rag_needed=false")
	}
	if result.ControlSignals.ContextLoad != "high" {
		t.Fatalf("expected context_load high, got %s", result.ControlSignals.ContextLoad)
	}
	if len(result.ControlSignals.ModelFit) != 1 {
		t.Fatalf("expected cleaned model_fit size 1, got %d", len(result.ControlSignals.ModelFit))
	}
	if result.ControlSignals.RecommendedTemperature == nil || *result.ControlSignals.RecommendedTemperature != 0.2 {
		t.Fatal("expected recommended temperature 0.2")
	}
	if result.ControlSignals.RecommendedTopP == nil || *result.ControlSignals.RecommendedTopP != 0.9 {
		t.Fatal("expected recommended top_p 0.9")
	}
	if result.ControlSignals.RecommendedMaxTokens == nil || *result.ControlSignals.RecommendedMaxTokens != 1024 {
		t.Fatal("expected recommended max_tokens 1024")
	}
	if result.ControlSignals.ExperimentTag != "ctrl-exp-a" {
		t.Fatalf("expected experiment tag ctrl-exp-a, got %s", result.ControlSignals.ExperimentTag)
	}
	if result.ControlSignals.DomainTag != "coding" {
		t.Fatalf("expected domain tag coding, got %s", result.ControlSignals.DomainTag)
	}
}

func TestParseClassifierOutput_InvalidControlValuesAreClamped(t *testing.T) {
	raw := `{
  "task_type":"chat",
  "difficulty":"low",
  "confidence":0.77,
  "semantic_signature":"chat:greeting",
  "route_hint":"speed",
  "query_stability_score":2.5,
  "ttl_band":"never",
  "risk_level":"critical",
  "context_load":"extreme",
  "risk_tags":["  ","A"],
  "model_fit":{"x":-1},
  "recommended_temperature":3,
  "recommended_top_p":2,
  "recommended_max_tokens":99999,
  "experiment_tag":"@@@",
  "domain_tag":"Domain*Tag"
}`

	result, err := parseClassifierOutput(raw)
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}

	if result.ControlSignals == nil {
		t.Fatal("expected control signals to be present")
	}
	if result.ControlSignals.QueryStabilityScore != 0 {
		t.Fatalf("expected clamped stability score 0, got %v", result.ControlSignals.QueryStabilityScore)
	}
	if result.ControlSignals.TTLBand != "" {
		t.Fatalf("expected invalid ttl band to be cleared, got %s", result.ControlSignals.TTLBand)
	}
	if result.ControlSignals.RiskLevel != "" {
		t.Fatalf("expected invalid risk level cleared, got %s", result.ControlSignals.RiskLevel)
	}
	if result.ControlSignals.ContextLoad != "" {
		t.Fatalf("expected invalid context load cleared, got %s", result.ControlSignals.ContextLoad)
	}
	if len(result.ControlSignals.ModelFit) != 0 {
		t.Fatalf("expected invalid model_fit removed, got %#v", result.ControlSignals.ModelFit)
	}
	if result.ControlSignals.RecommendedTemperature != nil {
		t.Fatal("expected invalid recommended_temperature dropped")
	}
	if result.ControlSignals.RecommendedTopP != nil {
		t.Fatal("expected invalid recommended_top_p dropped")
	}
	if result.ControlSignals.RecommendedMaxTokens != nil {
		t.Fatal("expected invalid recommended_max_tokens dropped")
	}
	if result.ControlSignals.ExperimentTag != "" {
		t.Fatal("expected invalid experiment tag dropped")
	}
	if result.ControlSignals.DomainTag != "domaintag" {
		t.Fatalf("expected normalized domain tag domaintag, got %s", result.ControlSignals.DomainTag)
	}
}

func TestParseClassifierOutput_InvalidTaskReturnsParseError(t *testing.T) {
	raw := `{"task_type":"bad","difficulty":"low","confidence":0.5}`

	_, err := parseClassifierOutput(raw)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !errors.Is(err, ErrClassifierParseOutput) {
		t.Fatalf("expected parse error wrapper, got %v", err)
	}
}
