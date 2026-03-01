package handler

import "testing"

func TestExtractUsageTokensFromBody_WithStandardUsageFields(t *testing.T) {
	body := []byte(`{
		"id":"chatcmpl-test",
		"usage":{
			"prompt_tokens":120,
			"completion_tokens":80,
			"total_tokens":200
		}
	}`)

	tokens := extractUsageTokensFromBody(body)

	if tokens.Prompt != 120 {
		t.Fatalf("expected prompt tokens 120, got %d", tokens.Prompt)
	}
	if tokens.Completion != 80 {
		t.Fatalf("expected completion tokens 80, got %d", tokens.Completion)
	}
	if tokens.Total != 200 {
		t.Fatalf("expected total tokens 200, got %d", tokens.Total)
	}
}

func TestExtractUsageTokensFromBody_WithoutTotalTokens_ComputesFromPromptAndCompletion(t *testing.T) {
	body := []byte(`{
		"usage":{
			"prompt_tokens":31,
			"completion_tokens":9
		}
	}`)

	tokens := extractUsageTokensFromBody(body)

	if tokens.Prompt != 31 {
		t.Fatalf("expected prompt tokens 31, got %d", tokens.Prompt)
	}
	if tokens.Completion != 9 {
		t.Fatalf("expected completion tokens 9, got %d", tokens.Completion)
	}
	if tokens.Total != 40 {
		t.Fatalf("expected total tokens 40, got %d", tokens.Total)
	}
}

func TestExtractUsageTokensFromBody_WithoutUsage_ReturnsZero(t *testing.T) {
	body := []byte(`{"id":"chatcmpl-no-usage","choices":[]}`)

	tokens := extractUsageTokensFromBody(body)

	if tokens.Prompt != 0 || tokens.Completion != 0 || tokens.Total != 0 {
		t.Fatalf("expected all zero tokens, got prompt=%d completion=%d total=%d", tokens.Prompt, tokens.Completion, tokens.Total)
	}
}
