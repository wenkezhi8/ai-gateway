package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Format   string              `json:"format,omitempty"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message    *ollamaChatMessage `json:"message,omitempty"`
	Response   string             `json:"response,omitempty"`
	Done       bool               `json:"done,omitempty"`
	DoneReason string             `json:"done_reason,omitempty"`
	Error      string             `json:"error,omitempty"`
}

type OllamaTaskClassifier struct {
	mu     sync.RWMutex
	cfg    ClassifierConfig
	client *http.Client
}

func NewOllamaTaskClassifier(cfg ClassifierConfig) *OllamaTaskClassifier {
	cfg = clampClassifierConfig(cfg)
	return &OllamaTaskClassifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: classifierTimeout(cfg),
		},
	}
}

func (o *OllamaTaskClassifier) UpdateConfig(cfg ClassifierConfig) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.cfg = clampClassifierConfig(cfg)
	o.client.Timeout = classifierTimeout(o.cfg)
}

func (o *OllamaTaskClassifier) GetConfig() ClassifierConfig {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.cfg
}

func (o *OllamaTaskClassifier) Health(ctx context.Context) *ClassifierHealth {
	cfg := o.GetConfig()
	start := time.Now()

	body, status, err := o.chat(ctx, cfg.ActiveModel, "请仅输出JSON: {\"task_type\":\"chat\",\"difficulty\":\"low\",\"confidence\":0.99,\"semantic_signature\":\"问候\",\"route_hint\":\"speed\"}")
	latency := time.Since(start).Milliseconds()
	health := &ClassifierHealth{
		Healthy:    err == nil,
		Model:      cfg.ActiveModel,
		Provider:   cfg.Provider,
		LatencyMs:  latency,
		CheckedAt:  time.Now().UnixMilli(),
		StatusCode: status,
	}
	if err != nil {
		health.Message = err.Error()
		return health
	}
	if body == "" {
		health.Healthy = false
		health.Message = "empty classifier response"
		return health
	}
	health.Message = "ok"
	return health
}

func (o *OllamaTaskClassifier) Classify(ctx context.Context, prompt, contextText string) (*AssessmentResult, error) {
	cfg := o.GetConfig()
	inputPrompt := truncateForClassifier(prompt, cfg.MaxInputChars)
	inputContext := truncateForClassifier(contextText, cfg.MaxInputChars)

	content := buildClassifierPrompt(inputPrompt, inputContext)
	raw, _, err := o.chat(ctx, cfg.ActiveModel, content)
	if err != nil {
		return nil, err
	}

	parsed, err := parseClassifierOutput(raw)
	if err != nil {
		return nil, err
	}
	parsed.Source = ClassificationSourceOllama
	if parsed.SemanticSignature == "" {
		parsed.SemanticSignature = buildFallbackSignature(parsed.TaskType, inputPrompt)
	}
	return parsed, nil
}

func (o *OllamaTaskClassifier) chat(ctx context.Context, model, content string) (string, int, error) {
	cfg := o.GetConfig()
	endpoint := strings.TrimRight(cfg.BaseURL, "/") + "/api/chat"
	reqBody := ollamaChatRequest{
		Model: model,
		Messages: []ollamaChatMessage{
			{Role: "system", Content: "你是任务分类器。仅返回 JSON，不要任何额外文本。"},
			{Role: "user", Content: content},
		},
		Stream: false,
		Format: "json",
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("marshal classifier request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", 0, fmt.Errorf("create classifier request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	var chatResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", resp.StatusCode, fmt.Errorf("decode classifier response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := chatResp.Error
		if msg == "" {
			msg = "classifier http error"
		}
		return "", resp.StatusCode, fmt.Errorf("%s: %d", msg, resp.StatusCode)
	}

	if chatResp.Error != "" {
		return "", resp.StatusCode, fmt.Errorf("classifier error: %s", chatResp.Error)
	}

	if chatResp.Message != nil && strings.TrimSpace(chatResp.Message.Content) != "" {
		return chatResp.Message.Content, resp.StatusCode, nil
	}
	if strings.TrimSpace(chatResp.Response) != "" {
		return chatResp.Response, resp.StatusCode, nil
	}

	return "", resp.StatusCode, nil
}

func buildClassifierPrompt(prompt, contextText string) string {
	return fmt.Sprintf(`请进行任务分类，严格返回 JSON，不要 markdown，不要解释。
可选 task_type: chat, code, reasoning, creative, fact, long_text, math, translate, unknown
可选 difficulty: low, medium, high
可选 route_hint: speed, balanced, quality, reasoning_first
可选 ttl_band: short, medium, long

输出 JSON 字段:
{
  "task_type":"...",
  "difficulty":"...",
  "confidence":0.0,
  "semantic_signature":"意图归一化短句",
  "route_hint":"...",
  "control_version":"v1",
  "normalized_query":"可缓存查询短句",
  "query_stability_score":0.0,
  "cacheable":true,
  "cache_reason":"",
  "ttl_band":"medium"
}

用户输入:
%s

系统上下文:
%s
`, prompt, contextText)
}

func truncateForClassifier(input string, maxLen int) string {
	trimmed := strings.TrimSpace(input)
	if maxLen <= 0 || len(trimmed) <= maxLen {
		return trimmed
	}
	return trimmed[:maxLen]
}

func parseClassifierOutput(raw string) (*AssessmentResult, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var data struct {
		TaskType            string  `json:"task_type"`
		Difficulty          string  `json:"difficulty"`
		Confidence          float64 `json:"confidence"`
		SemanticSignature   string  `json:"semantic_signature"`
		RouteHint           string  `json:"route_hint"`
		ControlVersion      string  `json:"control_version"`
		NormalizedQuery     string  `json:"normalized_query"`
		QueryStabilityScore float64 `json:"query_stability_score"`
		Cacheable           *bool   `json:"cacheable"`
		CacheReason         string  `json:"cache_reason"`
		TTLBand             string  `json:"ttl_band"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, fmt.Errorf("%w: parse classifier json: %v", ErrClassifierParseOutput, err)
	}

	taskType := TaskType(data.TaskType)
	if !isSupportedTaskType(taskType) {
		return nil, fmt.Errorf("%w: invalid task_type: %s", ErrClassifierParseOutput, data.TaskType)
	}
	difficulty := DifficultyLevel(data.Difficulty)
	if !isSupportedDifficulty(difficulty) {
		return nil, fmt.Errorf("%w: invalid difficulty: %s", ErrClassifierParseOutput, data.Difficulty)
	}
	if data.Confidence < 0 || data.Confidence > 1 {
		return nil, fmt.Errorf("%w: invalid confidence: %.2f", ErrClassifierParseOutput, data.Confidence)
	}

	controlSignals := parseControlSignals(
		data.ControlVersion,
		data.NormalizedQuery,
		data.QueryStabilityScore,
		data.Cacheable,
		data.CacheReason,
		data.TTLBand,
	)

	result := &AssessmentResult{
		TaskType:          taskType,
		Difficulty:        difficulty,
		Confidence:        data.Confidence,
		SemanticSignature: strings.TrimSpace(data.SemanticSignature),
		ControlSignals:    controlSignals,
		RouteHint:         strings.TrimSpace(data.RouteHint),
	}
	return result, nil
}

func parseControlSignals(controlVersion, normalizedQuery string, queryStabilityScore float64, cacheable *bool, cacheReason, ttlBand string) *ControlSignals {
	controlVersion = strings.TrimSpace(controlVersion)
	normalizedQuery = strings.TrimSpace(normalizedQuery)
	cacheReason = strings.TrimSpace(cacheReason)
	ttlBand = strings.TrimSpace(ttlBand)

	if queryStabilityScore < 0 || queryStabilityScore > 1 {
		queryStabilityScore = 0
	}
	switch ttlBand {
	case "short", "medium", "long", "":
	default:
		ttlBand = ""
	}

	if controlVersion == "" && normalizedQuery == "" && queryStabilityScore == 0 && cacheable == nil && cacheReason == "" && ttlBand == "" {
		return nil
	}

	return &ControlSignals{
		ControlVersion:      controlVersion,
		NormalizedQuery:     normalizedQuery,
		QueryStabilityScore: queryStabilityScore,
		Cacheable:           cacheable,
		CacheReason:         cacheReason,
		TTLBand:             ttlBand,
	}
}

func isSupportedTaskType(taskType TaskType) bool {
	switch taskType {
	case TaskTypeChat, TaskTypeCode, TaskTypeReasoning, TaskTypeCreative, TaskTypeFact, TaskTypeLongText, TaskTypeMath, TaskTypeTranslate, TaskTypeUnknown:
		return true
	default:
		return false
	}
}

func isSupportedDifficulty(d DifficultyLevel) bool {
	switch d {
	case DifficultyLow, DifficultyMedium, DifficultyHigh:
		return true
	default:
		return false
	}
}

func buildFallbackSignature(taskType TaskType, prompt string) string {
	p := strings.TrimSpace(prompt)
	if len(p) > 120 {
		p = p[:120]
	}
	if p == "" {
		return string(taskType)
	}
	return string(taskType) + ":" + p
}

func ListOllamaModels(ctx context.Context, baseURL string, timeout time.Duration) ([]string, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://127.0.0.1:11434"
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create ollama tags request: %w", err)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request ollama tags failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ollama tags http error: %d", resp.StatusCode)
	}

	var tagsResp ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("decode ollama tags response: %w", err)
	}

	models := make([]string, 0, len(tagsResp.Models))
	seen := make(map[string]struct{}, len(tagsResp.Models))
	for _, model := range tagsResp.Models {
		name := strings.TrimSpace(model.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		models = append(models, name)
	}
	sort.Strings(models)

	return models, nil
}
