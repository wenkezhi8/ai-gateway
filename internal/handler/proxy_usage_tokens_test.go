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

func TestExtractUsageTokensFromBody_WithCachedTokensDetails(t *testing.T) {
	body := []byte(`{
		"usage":{
			"prompt_tokens":120,
			"completion_tokens":40,
			"total_tokens":160,
			"prompt_tokens_details":{"cached_tokens":33}
		}
	}`)

	tokens := extractUsageTokensFromBody(body)
	if tokens.CachedRead != 33 {
		t.Fatalf("expected cached read tokens 33, got %d", tokens.CachedRead)
	}
}

func TestExtractUsageTokensFromBody_WithInputTokensDetailsCachedTokens(t *testing.T) {
	body := []byte(`{
		"usage":{
			"input_tokens":100,
			"output_tokens":20,
			"total_tokens":120,
			"input_tokens_details":{"cached_tokens":11}
		}
	}`)

	tokens := extractUsageTokensFromBody(body)
	if tokens.Prompt != 100 || tokens.Completion != 20 || tokens.Total != 120 {
		t.Fatalf("unexpected tokens parsed from input/output usage: %+v", tokens)
	}
	if tokens.CachedRead != 11 {
		t.Fatalf("expected cached read tokens 11, got %d", tokens.CachedRead)
	}
}

func TestEstimateTokensByText_MixedAsciiAndCJK(t *testing.T) {
	// ascii=4 -> 1 token, non-ascii=2 -> 2 tokens, total ceil(2.333)=3
	got := estimateTokensByText("abcd你好")
	if got != 3 {
		t.Fatalf("expected estimated tokens 3, got %d", got)
	}
}

func TestEstimateTokensByText_EmptyText_ReturnsZero(t *testing.T) {
	got := estimateTokensByText("")
	if got != 0 {
		t.Fatalf("expected estimated tokens 0, got %d", got)
	}
}

func TestBuildCachedUsage_UsesPromptTokensFromInput(t *testing.T) {
	usage := buildCachedUsage(12, 8, 3, 20)
	if usage["prompt_tokens"] != 12 {
		t.Fatalf("expected prompt_tokens 12, got %d", usage["prompt_tokens"])
	}
	if usage["completion_tokens"] != 8 {
		t.Fatalf("expected completion_tokens 8, got %d", usage["completion_tokens"])
	}
	if usage["total_tokens"] != 20 {
		t.Fatalf("expected total_tokens 20, got %d", usage["total_tokens"])
	}
	if usage["cached_read_tokens"] != 3 {
		t.Fatalf("expected cached_read_tokens 3, got %d", usage["cached_read_tokens"])
	}
}

func TestResolveUsageWithFallback_WhenActualUsageExists_ReturnsActualSource(t *testing.T) {
	provided := usageTokens{Prompt: 10, Completion: 5, Total: 15}
	resolved, source := resolveUsageWithFallback("1+1", "2", provided)

	if source != "actual" {
		t.Fatalf("expected usage source actual, got %s", source)
	}
	if resolved.Total != 15 || resolved.Prompt != 10 || resolved.Completion != 5 {
		t.Fatalf("unexpected resolved usage: %+v", resolved)
	}
}

func TestResolveUsageWithFallback_WhenUsageMissing_ReturnsEstimatedSource(t *testing.T) {
	provided := usageTokens{}
	resolved, source := resolveUsageWithFallback("请计算1+1", "1+1=2", provided)

	if source != "estimated" {
		t.Fatalf("expected usage source estimated, got %s", source)
	}
	if resolved.Total <= 0 {
		t.Fatalf("expected positive estimated total tokens, got %d", resolved.Total)
	}
	if resolved.Prompt <= 0 || resolved.Completion <= 0 {
		t.Fatalf("expected positive prompt/completion tokens, got prompt=%d completion=%d", resolved.Prompt, resolved.Completion)
	}
}
