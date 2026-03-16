package handler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ai-gateway/internal/config"
)

func TestEvaluateCompressionDecision_CacheHitSkipsCompression(t *testing.T) {
	cfg := config.CompressionConfig{
		Enabled:             true,
		MaxPromptTokens:     10,
		TargetPromptTokens:  6,
		PreserveRecentTurns: 2,
	}
	messages := []ChatMessage{
		{Role: "user", Content: "这是一个很长很长的问题，用于触发预算判断。"},
		{Role: "assistant", Content: "这是一个很长很长的回答，用于触发预算判断。"},
	}

	decision := evaluateCompressionDecision(cfg, messages, true)
	if decision.Apply {
		t.Fatalf("expected no compression on cache hit")
	}
	if decision.Reason != "cache_hit" {
		t.Fatalf("expected reason cache_hit, got %q", decision.Reason)
	}
}

func TestEvaluateCompressionDecision_OverBudgetTriggersCompression(t *testing.T) {
	cfg := config.CompressionConfig{
		Enabled:             true,
		MaxPromptTokens:     6,
		TargetPromptTokens:  3,
		PreserveRecentTurns: 1,
	}
	messages := []ChatMessage{
		{Role: "system", Content: "你是助手"},
		{Role: "user", Content: "第一段很长很长的上下文，超过预算"},
		{Role: "assistant", Content: "第一段回答"},
		{Role: "user", Content: "第二段很长很长的上下文，超过预算"},
		{Role: "assistant", Content: "第二段回答"},
	}

	decision := evaluateCompressionDecision(cfg, messages, false)
	if !decision.Apply {
		t.Fatalf("expected compression for over budget request")
	}
	if decision.Reason != "over_budget" {
		t.Fatalf("expected reason over_budget, got %q", decision.Reason)
	}
}

func TestApplyCompressionByBudget_PreservesSystemAndRecentTurns(t *testing.T) {
	cfg := config.CompressionConfig{
		Enabled:             true,
		MaxPromptTokens:     10,
		TargetPromptTokens:  6,
		PreserveRecentTurns: 1,
	}
	messages := []ChatMessage{
		{Role: "system", Content: "全局约束"},
		{Role: "user", Content: "用户问题 1 很长很长很长"},
		{Role: "assistant", Content: "助手回答 1 很长很长很长"},
		{Role: "user", Content: "用户问题 2 很长很长很长"},
		{Role: "assistant", Content: "助手回答 2 很长很长很长"},
	}

	compressed, applied := applyCompressionByBudget(cfg, messages)
	if !applied {
		t.Fatalf("expected compression to be applied")
	}
	if len(compressed) >= len(messages) {
		t.Fatalf("expected compressed messages fewer than original")
	}
	if compressed[0].Role != "system" {
		t.Fatalf("expected system message preserved at head")
	}
	if compressed[len(compressed)-1].Role != "assistant" {
		t.Fatalf("expected latest assistant message preserved")
	}
}

type fakeCompressionSummarizer struct {
	summary string
	err     error
}

func (f *fakeCompressionSummarizer) Summarize(_ context.Context, _ compressionSummaryInput) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.summary, nil
}

type fakeCompressionRetriever struct {
	snippets []string
	err      error
}

func (f *fakeCompressionRetriever) Retrieve(_ context.Context, _ string, _ int) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.snippets, nil
}

func TestReduceMessagesForBudget_RAGEnabledAndHit_ShouldInjectRAGSummary(t *testing.T) {
	cfg := config.CompressionConfig{
		Enabled:             true,
		MaxPromptTokens:     6,
		TargetPromptTokens:  3,
		PreserveRecentTurns: 1,
		SummaryInjectEnable: true,
		RAGDependencyEnable: true,
		RAGTopK:             2,
	}
	original := []ChatMessage{
		{Role: "system", Content: "你是助手"},
		{Role: "user", Content: "第一轮问题很长很长很长很长很长很长很长很长"},
		{Role: "assistant", Content: "第一轮回答很长很长很长很长很长很长很长很长"},
		{Role: "user", Content: "Q2"},
		{Role: "assistant", Content: "A2"},
	}

	compressed, result := reduceMessagesForBudget(
		context.Background(),
		cfg,
		original,
		"第二轮问题很长很长很长",
		false,
		&fakeCompressionSummarizer{summary: "历史摘要"},
		&fakeCompressionRetriever{snippets: []string{"知识片段A", "知识片段B"}},
	)

	if !result.Applied {
		t.Fatalf("expected compression applied")
	}
	if !result.RAGUsed {
		t.Fatalf("expected rag snippets used")
	}
	if len(compressed) >= len(original) {
		t.Fatalf("expected compressed conversation")
	}
	if !containsSystemSummary(compressed, "历史摘要") {
		t.Fatalf("expected injected history summary")
	}
	if !containsSystemSummary(compressed, "知识片段A") {
		t.Fatalf("expected injected rag snippet")
	}
}

func TestReduceMessagesForBudget_RAGRetrieveFail_ShouldFallbackHistoryOnly(t *testing.T) {
	cfg := config.CompressionConfig{
		Enabled:             true,
		MaxPromptTokens:     6,
		TargetPromptTokens:  3,
		PreserveRecentTurns: 1,
		SummaryInjectEnable: true,
		RAGDependencyEnable: true,
		RAGTopK:             2,
	}
	original := []ChatMessage{
		{Role: "system", Content: "你是助手"},
		{Role: "user", Content: "第一轮问题很长很长很长很长很长很长很长很长"},
		{Role: "assistant", Content: "第一轮回答很长很长很长很长很长很长很长很长"},
		{Role: "user", Content: "Q2"},
		{Role: "assistant", Content: "A2"},
	}

	compressed, result := reduceMessagesForBudget(
		context.Background(),
		cfg,
		original,
		"第二轮问题很长很长很长",
		false,
		&fakeCompressionSummarizer{summary: "历史摘要"},
		&fakeCompressionRetriever{err: errors.New("rag down")},
	)

	if !result.Applied {
		t.Fatalf("expected compression applied")
	}
	if !result.RAGFailed {
		t.Fatalf("expected rag failure recorded")
	}
	if containsSystemSummary(compressed, "知识片段A") {
		t.Fatalf("expected no rag snippets when retrieval failed")
	}
}

func TestReduceMessagesForBudget_SummaryFail_ShouldFallbackToOriginal(t *testing.T) {
	cfg := config.CompressionConfig{
		Enabled:             true,
		MaxPromptTokens:     6,
		TargetPromptTokens:  3,
		PreserveRecentTurns: 1,
		SummaryInjectEnable: true,
	}
	original := []ChatMessage{
		{Role: "system", Content: "你是助手"},
		{Role: "user", Content: "第一轮问题很长很长很长很长很长很长很长很长"},
		{Role: "assistant", Content: "第一轮回答很长很长很长很长很长很长很长很长"},
		{Role: "user", Content: "Q2"},
		{Role: "assistant", Content: "A2"},
	}

	compressed, result := reduceMessagesForBudget(
		context.Background(),
		cfg,
		original,
		"第二轮问题很长很长很长",
		false,
		&fakeCompressionSummarizer{err: errors.New("local model timeout")},
		nil,
	)

	if !result.FallbackInvoked {
		t.Fatalf("expected fallback invoked")
	}
	if !result.FallbackSaved {
		t.Fatalf("expected fallback saved")
	}
	if len(compressed) != len(original) {
		t.Fatalf("expected original messages after fallback")
	}
}

func containsSystemSummary(messages []ChatMessage, keyword string) bool {
	for _, msg := range messages {
		if msg.Role != "system" {
			continue
		}
		if text, ok := msg.Content.(string); ok && text != "" && keyword != "" && strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}
