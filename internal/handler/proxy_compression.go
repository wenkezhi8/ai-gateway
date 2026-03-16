package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"ai-gateway/internal/config"
)

const compressionSummaryPrefix = "[ctx]"

type compressionDecision struct {
	Apply        bool
	Reason       string
	PromptTokens int
}

type compressionSummaryInput struct {
	History        []ChatMessage
	Prompt         string
	TargetMaxChars int
}

type compressionSummarizer interface {
	Summarize(ctx context.Context, input compressionSummaryInput) (string, error)
}

type compressionRAGRetriever interface {
	Retrieve(ctx context.Context, query string, topK int) ([]string, error)
}

type compressionResult struct {
	Applied            bool
	Reason             string
	PromptTokensBefore int
	PromptTokensAfter  int
	CompressionRatio   float64
	RAGRequested       bool
	RAGUsed            bool
	RAGFailed          bool
	FallbackInvoked    bool
	FallbackSaved      bool
}

func evaluateCompressionDecision(cfg config.CompressionConfig, messages []ChatMessage, cacheHit bool) compressionDecision {
	if !cfg.Enabled {
		return compressionDecision{Apply: false, Reason: "disabled"}
	}
	if cacheHit {
		return compressionDecision{Apply: false, Reason: "cache_hit"}
	}
	promptTokens := estimatePromptTokens(messages)
	if promptTokens <= cfg.MaxPromptTokens {
		return compressionDecision{
			Apply:        false,
			Reason:       "within_budget",
			PromptTokens: promptTokens,
		}
	}
	return compressionDecision{
		Apply:        true,
		Reason:       "over_budget",
		PromptTokens: promptTokens,
	}
}

func applyCompressionByBudget(cfg config.CompressionConfig, messages []ChatMessage) ([]ChatMessage, bool) {
	compressed, result := reduceMessagesForBudget(context.Background(), cfg, messages, "", false, nil, nil)
	return compressed, result.Applied
}

func reduceMessagesForBudget(
	ctx context.Context,
	cfg config.CompressionConfig,
	messages []ChatMessage,
	prompt string,
	cacheHit bool,
	summarizer compressionSummarizer,
	retriever compressionRAGRetriever,
) ([]ChatMessage, compressionResult) {
	decision := evaluateCompressionDecision(cfg, messages, cacheHit)
	result := compressionResult{
		Applied:            false,
		Reason:             decision.Reason,
		PromptTokensBefore: decision.PromptTokens,
		PromptTokensAfter:  decision.PromptTokens,
	}
	if !decision.Apply {
		return cloneChatMessages(messages), result
	}

	if summarizer == nil {
		summarizer = &historyCompressionSummarizer{}
	}

	systemMessages, historyMessages, preservedTail := splitMessagesForCompression(messages, cfg.PreserveRecentTurns)

	summaryText := ""
	if cfg.SummaryInjectEnable {
		var err error
		summaryText, err = summarizer.Summarize(ctx, compressionSummaryInput{
			History:        historyMessages,
			Prompt:         prompt,
			TargetMaxChars: cfg.SummaryMaxChars,
		})
		if err != nil || strings.TrimSpace(summaryText) == "" {
			result.Reason = "summary_failed"
			result.FallbackInvoked = true
			result.FallbackSaved = true
			return cloneChatMessages(messages), result
		}
	}

	var ragSnippets []string
	if cfg.RAGDependencyEnable {
		result.RAGRequested = true
		if retriever == nil {
			result.RAGFailed = true
		} else {
			snippets, err := retriever.Retrieve(ctx, prompt, cfg.RAGTopK)
			if err != nil {
				result.RAGFailed = true
			} else if len(snippets) > 0 {
				result.RAGUsed = true
				ragSnippets = snippets
			}
		}
	}

	compressed := make([]ChatMessage, 0, len(systemMessages)+len(preservedTail)+2)
	compressed = append(compressed, cloneChatMessages(systemMessages)...)
	if cfg.SummaryInjectEnable {
		summaryMessage := buildCompressionSummaryMessage(summaryText, ragSnippets)
		compressed = append(compressed, summaryMessage)
	}
	compressed = append(compressed, cloneChatMessages(preservedTail)...)

	for estimatePromptTokens(compressed) > cfg.TargetPromptTokens {
		if !dropOldestNonSystemMessage(&compressed, 2) {
			break
		}
	}

	result.PromptTokensAfter = estimatePromptTokens(compressed)
	if decision.PromptTokens > 0 {
		result.CompressionRatio = 1.0 - float64(result.PromptTokensAfter)/float64(decision.PromptTokens)
		if result.CompressionRatio < 0 {
			result.CompressionRatio = 0
		}
	}

	if result.PromptTokensAfter >= decision.PromptTokens {
		result.Reason = "no_gain"
		return cloneChatMessages(messages), result
	}

	result.Applied = true
	result.Reason = "applied"
	return compressed, result
}

func splitMessagesForCompression(messages []ChatMessage, preserveRecentTurns int) (systems []ChatMessage, history []ChatMessage, tail []ChatMessage) {
	if preserveRecentTurns <= 0 {
		preserveRecentTurns = 1
	}
	preserveTailMessages := preserveRecentTurns * 2
	if preserveTailMessages < 2 {
		preserveTailMessages = 2
	}

	nonSystem := make([]ChatMessage, 0, len(messages))
	for _, msg := range messages {
		if strings.EqualFold(strings.TrimSpace(msg.Role), "system") {
			systems = append(systems, msg)
			continue
		}
		nonSystem = append(nonSystem, msg)
	}

	if len(nonSystem) <= preserveTailMessages {
		return systems, nil, nonSystem
	}

	split := len(nonSystem) - preserveTailMessages
	history = nonSystem[:split]
	tail = nonSystem[split:]
	return systems, history, tail
}

func buildCompressionSummaryMessage(historySummary string, ragSnippets []string) ChatMessage {
	sections := make([]string, 0, 2)
	if strings.TrimSpace(historySummary) != "" {
		sections = append(sections, "历史:"+strings.TrimSpace(historySummary))
	}
	if len(ragSnippets) > 0 {
		lines := make([]string, 0, len(ragSnippets))
		for _, snippet := range ragSnippets {
			trimmed := strings.TrimSpace(snippet)
			if trimmed == "" {
				continue
			}
			lines = append(lines, trimmed)
		}
		if len(lines) > 0 {
			sections = append(sections, "RAG:"+strings.Join(lines, " | "))
		}
	}
	content := compressionSummaryPrefix + "\n" + strings.Join(sections, "\n\n")
	return ChatMessage{Role: "system", Content: content}
}

func dropOldestNonSystemMessage(messages *[]ChatMessage, preserveTail int) bool {
	msgs := *messages
	if len(msgs) == 0 {
		return false
	}

	nonSystemIdx := make([]int, 0, len(msgs))
	for i := range msgs {
		if strings.EqualFold(strings.TrimSpace(msgs[i].Role), "system") {
			continue
		}
		nonSystemIdx = append(nonSystemIdx, i)
	}
	if len(nonSystemIdx) <= preserveTail {
		return false
	}

	removeIdx := nonSystemIdx[0]
	*messages = append(msgs[:removeIdx], msgs[removeIdx+1:]...)
	return true
}

func estimatePromptTokens(messages []ChatMessage) int {
	total := 0
	for _, msg := range messages {
		text := strings.TrimSpace(getTextContent(msg.Content))
		if text == "" {
			continue
		}
		runes := utf8.RuneCountInString(text)
		estimated := runes / 4
		if estimated < 1 {
			estimated = 1
		}
		total += estimated
	}
	return total
}

type historyCompressionSummarizer struct{}

func (s *historyCompressionSummarizer) Summarize(_ context.Context, input compressionSummaryInput) (string, error) {
	if len(input.History) == 0 {
		return "", errors.New("empty history")
	}
	maxChars := input.TargetMaxChars
	if maxChars <= 0 {
		maxChars = 1200
	}

	lines := make([]string, 0, len(input.History)+1)
	if strings.TrimSpace(input.Prompt) != "" {
		lines = append(lines, "当前目标: "+truncateByRune(strings.TrimSpace(input.Prompt), 40))
	}
	for _, msg := range input.History {
		text := strings.TrimSpace(getTextContent(msg.Content))
		if text == "" {
			continue
		}
		line := fmt.Sprintf("- [%s] %s", strings.ToLower(strings.TrimSpace(msg.Role)), truncateByRune(text, 24))
		lines = append(lines, line)
		if len(lines) >= 10 {
			break
		}
	}
	if len(lines) == 0 {
		return "", errors.New("empty summary payload")
	}
	summary := strings.Join(lines, "\n")
	return truncateByRune(summary, maxChars), nil
}

func truncateByRune(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

type sqliteKnowledgeRetriever struct {
	db *sql.DB
}

func newSQLiteKnowledgeRetriever(db *sql.DB) compressionRAGRetriever {
	if db == nil {
		return nil
	}
	return &sqliteKnowledgeRetriever{db: db}
}

func (r *sqliteKnowledgeRetriever) Retrieve(ctx context.Context, query string, topK int) ([]string, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("rag db unavailable")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if topK <= 0 {
		topK = 3
	}

	keywords := buildCompressionKeywords(query)
	if len(keywords) == 0 {
		keywords = []string{query}
	}

	rows, err := r.db.QueryContext(ctx, `SELECT c.content
		FROM kb_chunks c
		JOIN kb_documents d ON d.id = c.document_id
		WHERE d.status = 'completed'
		ORDER BY c.created_at DESC
		LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snippets := make([]string, 0, topK)
	for rows.Next() {
		var content string
		if scanErr := rows.Scan(&content); scanErr != nil {
			return nil, scanErr
		}
		if !matchesCompressionKeyword(content, keywords) {
			continue
		}
		snippets = append(snippets, truncateByRune(strings.TrimSpace(content), 180))
		if len(snippets) >= topK {
			break
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}
	return snippets, nil
}

func buildCompressionKeywords(query string) []string {
	cleaned := strings.NewReplacer("？", " ", "?", " ", "，", " ", ",", " ", "。", " ", ".", " ", "！", " ", "!", " ", "：", " ", ":", " ", "；", " ", ";", " ").Replace(strings.TrimSpace(query))
	parts := strings.Fields(cleaned)
	if len(parts) > 1 {
		keywords := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if utf8.RuneCountInString(trimmed) >= 2 {
				keywords = append(keywords, strings.ToLower(trimmed))
			}
		}
		return keywords
	}
	return []string{strings.ToLower(strings.TrimSpace(query))}
}

func matchesCompressionKeyword(content string, keywords []string) bool {
	lower := strings.ToLower(content)
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}
