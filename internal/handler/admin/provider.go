package admin

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"ai-gateway/internal/limiter"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"

	"github.com/gin-gonic/gin"
)

// ProviderHandler handles provider management requests.
type ProviderHandler struct {
	registry   *provider.Registry
	manager    *limiter.AccountManager
	router     *routing.SmartRouter
	configPath string
}

// NewProviderHandler creates a new provider handler.
func NewProviderHandler(registry *provider.Registry, manager *limiter.AccountManager, router *routing.SmartRouter, configPath string) *ProviderHandler {
	if strings.TrimSpace(configPath) == "" {
		configPath = defaultRuntimeConfigPath
	}
	return &ProviderHandler{
		registry:   registry,
		manager:    manager,
		router:     router,
		configPath: configPath,
	}
}

func normalizeProviderKey(input string) string {
	return strings.TrimSpace(strings.ToLower(input))
}

func sameProvider(left, right string) bool {
	return normalizeProviderKey(left) != "" && normalizeProviderKey(left) == normalizeProviderKey(right)
}

const (
	providerCategoryInternational = "international"
	providerCategoryChinese       = "chinese"
	providerCategoryLocal         = "local"
	providerCategoryCustom        = "custom"

	providerIDOpenAI     = "openai"
	providerIDAnthropic  = "anthropic"
	providerIDVolcengine = "volcengine"
	providerIDQwen       = "qwen"
	providerIDZhipu      = "zhipu"
	providerIDMoonshot   = "moonshot"
	providerIDMinimax    = "minimax"
	providerIDBaichuan   = "baichuan"
	providerIDGoogle     = "google"
	providerIDOllama     = "ollama"
	providerIDLMStudio   = "lmstudio"

	providerErrCodeInvalidRequest = "invalid_request"
	providerErrCodeNotFound       = "not_found"
	providerErrMsgNotFound        = "Provider not found"
)

var providerTypeCatalog = map[string]ProviderTypeResponse{
	"openai": {
		ID:                 "openai",
		Label:              "OpenAI",
		Category:           providerCategoryInternational,
		Color:              "#10A37F",
		Logo:               "/logos/openai.svg",
		DefaultEndpoint:    "https://api.openai.com/v1",
		CodingEndpoint:     "https://api.openai.com/v1",
		SupportsCodingPlan: true,
		Models:             []string{"gpt-4o", "gpt-4o-mini"},
	},
	"anthropic": {
		ID:                 "anthropic",
		Label:              "Anthropic Claude",
		Category:           providerCategoryInternational,
		Color:              "#CC785C",
		Logo:               "/logos/anthropic.svg",
		DefaultEndpoint:    "https://api.anthropic.com/v1",
		CodingEndpoint:     "https://api.anthropic.com/v1",
		SupportsCodingPlan: true,
		Models:             []string{"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022"},
	},
	"azure-openai": {
		ID:                 "azure-openai",
		Label:              "Azure OpenAI",
		Category:           providerCategoryInternational,
		Color:              "#0078D4",
		Logo:               "/logos/azure.svg",
		DefaultEndpoint:    "https://your-resource.openai.azure.com",
		CodingEndpoint:     "https://your-resource.openai.azure.com",
		SupportsCodingPlan: true,
		Models:             []string{"gpt-4o", "gpt-4o-mini"},
	},
	"google": {
		ID:                 "google",
		Label:              "Google Gemini",
		Category:           providerCategoryInternational,
		Color:              "#4285F4",
		Logo:               "/logos/google.svg",
		DefaultEndpoint:    "https://generativelanguage.googleapis.com/v1beta",
		CodingEndpoint:     "https://generativelanguage.googleapis.com/v1beta/openai",
		SupportsCodingPlan: true,
		Models:             []string{"gemini-2.0-flash", "gemini-1.5-pro"},
	},
	"mistral": {
		ID:                 "mistral",
		Label:              "Mistral AI",
		Category:           providerCategoryInternational,
		Color:              "#FF7000",
		Logo:               "/logos/mistral.svg",
		DefaultEndpoint:    "https://api.mistral.ai/v1",
		CodingEndpoint:     "https://api.mistral.ai/v1",
		SupportsCodingPlan: true,
		Models:             []string{"mistral-large-latest", "mistral-medium-latest"},
	},
	"deepseek": {
		ID:                 "deepseek",
		Label:              "DeepSeek",
		Category:           providerCategoryChinese,
		Color:              "#4D6BFE",
		Logo:               "/logos/deepseek.svg",
		DefaultEndpoint:    "https://api.deepseek.com/v1",
		CodingEndpoint:     "https://api.deepseek.com/v1",
		SupportsCodingPlan: true,
		Models:             []string{"deepseek-chat", "deepseek-reasoner"},
	},
	"qwen": {
		ID:                 "qwen",
		Label:              "阿里云通义千问",
		Category:           providerCategoryChinese,
		Color:              "#FF6A00",
		Logo:               "/logos/qwen.svg",
		DefaultEndpoint:    "https://dashscope.aliyuncs.com/compatible-mode/v1",
		CodingEndpoint:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
		SupportsCodingPlan: true,
		Models:             []string{"qwen-max", "qwen-plus"},
	},
	"zhipu": {
		ID:                 "zhipu",
		Label:              "智谱AI",
		Category:           providerCategoryChinese,
		Color:              "#3657ED",
		Logo:               "/logos/zhipu.svg",
		DefaultEndpoint:    "https://open.bigmodel.cn/api/paas/v4",
		CodingEndpoint:     "https://open.bigmodel.cn/api/paas/v4",
		SupportsCodingPlan: true,
		Models:             []string{"glm-4-plus", "glm-4-air"},
	},
	"moonshot": {
		ID:                 "moonshot",
		Label:              "月之暗面 (Kimi)",
		Category:           providerCategoryChinese,
		Color:              "#1A1A1A",
		Logo:               "/logos/moonshot.svg",
		DefaultEndpoint:    "https://api.moonshot.cn/v1",
		CodingEndpoint:     "https://api.moonshot.cn/v1",
		SupportsCodingPlan: true,
		Models:             []string{"kimi-k2.5", "moonshot-v1-8k"},
	},
	"minimax": {
		ID:                 "minimax",
		Label:              "MiniMax",
		Category:           providerCategoryChinese,
		Color:              "#615CED",
		Logo:               "/logos/minimax.svg",
		DefaultEndpoint:    "https://api.minimax.chat/v1",
		CodingEndpoint:     "https://api.minimax.chat/v1",
		SupportsCodingPlan: true,
		Models:             []string{"abab6.5s-chat", "abab6.5g-chat"},
	},
	"baichuan": {
		ID:                 "baichuan",
		Label:              "百川智能",
		Category:           providerCategoryChinese,
		Color:              "#0066FF",
		Logo:               "/logos/baichuan.svg",
		DefaultEndpoint:    "https://api.baichuan-ai.com/v1",
		CodingEndpoint:     "https://api.baichuan-ai.com/v1",
		SupportsCodingPlan: true,
		Models:             []string{"Baichuan4", "Baichuan3-Turbo"},
	},
	"volcengine": {
		ID:                 "volcengine",
		Label:              "火山方舟 (豆包)",
		Category:           providerCategoryChinese,
		Color:              "#FF4D4F",
		Logo:               "/logos/volcengine.svg",
		DefaultEndpoint:    "https://ark.cn-beijing.volces.com/api/v3",
		CodingEndpoint:     "https://ark.cn-beijing.volces.com/api/v3",
		SupportsCodingPlan: true,
		Models:             []string{"doubao-pro-32k", "doubao-lite-32k"},
	},
	"ernie": {
		ID:                 "ernie",
		Label:              "百度文心一言",
		Category:           providerCategoryChinese,
		Color:              "#2932E1",
		Logo:               "/logos/ernie.svg",
		DefaultEndpoint:    "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat",
		CodingEndpoint:     "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat",
		SupportsCodingPlan: true,
		Models:             []string{"ernie-4.0", "ernie-3.5-8k"},
	},
	"hunyuan": {
		ID:                 "hunyuan",
		Label:              "腾讯混元",
		Category:           providerCategoryChinese,
		Color:              "#00A3FF",
		Logo:               "/logos/hunyuan.svg",
		DefaultEndpoint:    "https://api.hunyuan.cloud.tencent.com/v1",
		CodingEndpoint:     "https://api.hunyuan.cloud.tencent.com/v1",
		SupportsCodingPlan: true,
		Models:             []string{"hunyuan-turbo", "hunyuan-pro"},
	},
	"spark": {
		ID:                 "spark",
		Label:              "讯飞星火",
		Category:           providerCategoryChinese,
		Color:              "#E60012",
		Logo:               "/logos/spark.svg",
		DefaultEndpoint:    "https://spark-api-open.xf-yun.com/v1",
		CodingEndpoint:     "https://spark-api-open.xf-yun.com/v1",
		SupportsCodingPlan: true,
		Models:             []string{"spark-4.0-ultra", "spark-3.5-max"},
	},
	"yi": {
		ID:                 "yi",
		Label:              "零一万物",
		Category:           providerCategoryChinese,
		Color:              "#00D4AA",
		Logo:               "/logos/yi.svg",
		DefaultEndpoint:    "https://api.lingyiwanwu.com/v1",
		CodingEndpoint:     "https://api.lingyiwanwu.com/v1",
		SupportsCodingPlan: true,
		Models:             []string{"yi-large", "yi-medium"},
	},
	providerIDOllama: {
		ID:                 providerIDOllama,
		Label:              "Ollama",
		Category:           providerCategoryLocal,
		Color:              "#10B981",
		Logo:               "/logos/ollama.svg",
		DefaultEndpoint:    "http://localhost:11434/v1",
		CodingEndpoint:     "http://localhost:11434/v1",
		SupportsCodingPlan: true,
		Models:             []string{"qwen2.5-coder", "llama3.1"},
	},
	"lmstudio": {
		ID:                 "lmstudio",
		Label:              "LM Studio",
		Category:           providerCategoryLocal,
		Color:              "#3B82F6",
		Logo:               "/logos/lmstudio.svg",
		DefaultEndpoint:    "http://localhost:1234/v1",
		CodingEndpoint:     "http://localhost:1234/v1",
		SupportsCodingPlan: true,
		Models:             []string{"local-model"},
	},
	"local": {
		ID:                 "local",
		Label:              "本地模型",
		Category:           providerCategoryLocal,
		Color:              "#6B7280",
		Logo:               "/logos/local.svg",
		DefaultEndpoint:    "http://localhost:11434/v1",
		CodingEndpoint:     "http://localhost:11434/v1",
		SupportsCodingPlan: true,
		Models:             []string{"local-model"},
	},
}

func normalizeProviderID(input string) string {
	normalized := strings.TrimSpace(strings.ToLower(input))
	if normalized == "claude" {
		return providerIDAnthropic
	}
	return normalized
}

type providerURLRule struct {
	providerID string
	keywords   []string
}

var providerURLRules = []providerURLRule{
	{providerID: "deepseek", keywords: []string{"deepseek.com"}},
	{providerID: providerIDOpenAI, keywords: []string{"openai.com"}},
	{providerID: providerIDAnthropic, keywords: []string{"anthropic.com"}},
	{providerID: providerIDVolcengine, keywords: []string{"volces.com", "volcengine"}},
	{providerID: providerIDQwen, keywords: []string{"dashscope.aliyuncs.com", "aliyun"}},
	{providerID: providerIDZhipu, keywords: []string{"zhipuai.cn", "bigmodel.cn"}},
	{providerID: providerIDMoonshot, keywords: []string{"moonshot.cn", "kimi.ai"}},
	{providerID: providerIDMinimax, keywords: []string{"minimax"}},
	{providerID: providerIDBaichuan, keywords: []string{"baichuan"}},
	{providerID: providerIDGoogle, keywords: []string{"googleapis.com"}},
	{providerID: providerIDOllama, keywords: []string{"localhost:11434", "127.0.0.1:11434", providerIDOllama}},
	{providerID: providerIDLMStudio, keywords: []string{"localhost:1234", "127.0.0.1:1234", providerIDLMStudio}},
}

func containsAny(text string, keywords []string) bool {
	for i := range keywords {
		if strings.Contains(text, keywords[i]) {
			return true
		}
	}

	return false
}

func inferProviderFromBaseURL(rawURL string) string {
	baseURL := strings.ToLower(strings.TrimSpace(rawURL))
	if baseURL == "" {
		return ""
	}

	for i := range providerURLRules {
		rule := providerURLRules[i]
		if containsAny(baseURL, rule.keywords) {
			return rule.providerID
		}
	}

	return ""
}

func inferCategory(providerID string) string {
	normalized := normalizeProviderID(providerID)
	if normalized == "" {
		return providerCategoryCustom
	}
	if item, ok := providerTypeCatalog[normalized]; ok && item.Category != "" {
		return item.Category
	}

	switch normalized {
	case providerIDOpenAI, providerIDAnthropic, "azure-openai", providerIDGoogle, "mistral":
		return providerCategoryInternational
	case "deepseek", providerIDQwen, providerIDZhipu, providerIDMoonshot, providerIDMinimax, providerIDBaichuan, providerIDVolcengine, "ernie", "hunyuan", "spark", "yi":
		return providerCategoryChinese
	case providerIDOllama, providerIDLMStudio, "local":
		return providerCategoryLocal
	default:
		return providerCategoryCustom
	}
}

func normalizeProviderTypeItem(id string, item *ProviderTypeResponse) (string, ProviderTypeResponse, bool) {
	normalized := normalizeProviderID(id)
	if normalized == "" {
		return "", ProviderTypeResponse{}, false
	}

	normalizedItem := *item
	normalizedItem.ID = normalized
	if normalizedItem.Label == "" {
		normalizedItem.Label = normalized
	}
	if normalizedItem.Category == "" {
		normalizedItem.Category = inferCategory(normalized)
	}
	if normalizedItem.CodingEndpoint == "" {
		normalizedItem.CodingEndpoint = normalizedItem.DefaultEndpoint
	}
	normalizedItem.Models = uniqueSortedStrings(normalizedItem.Models)

	return normalized, normalizedItem, true
}

func defaultProviderTypeItem(normalized string) ProviderTypeResponse {
	return ProviderTypeResponse{
		ID:                 normalized,
		Label:              normalized,
		Category:           inferCategory(normalized),
		Color:              "#6B7280",
		Logo:               "",
		DefaultEndpoint:    "",
		CodingEndpoint:     "",
		SupportsCodingPlan: false,
		Models:             []string{},
	}
}

func mergeOneProviderType(merged map[string]ProviderTypeResponse, providerID string, models []string) {
	normalized := normalizeProviderID(providerID)
	if normalized == "" {
		return
	}

	item, ok := merged[normalized]
	if !ok {
		item = defaultProviderTypeItem(normalized)
	}

	if item.Category == "" {
		item.Category = inferCategory(normalized)
	}
	if item.CodingEndpoint == "" {
		item.CodingEndpoint = item.DefaultEndpoint
	}
	item.Models = uniqueSortedStrings(append(item.Models, models...))
	merged[normalized] = item
}

func accountProviderID(account *limiter.AccountConfig) string {
	providerID := strings.TrimSpace(account.Provider)
	if providerID == "" {
		providerID = strings.TrimSpace(account.ProviderType)
	}
	if inferred := inferProviderFromBaseURL(account.BaseURL); inferred != "" {
		providerID = inferred
	}

	return providerID
}

func finalizeProviderTypeItem(item *ProviderTypeResponse) ProviderTypeResponse {
	finalized := *item
	if finalized.Models == nil {
		finalized.Models = []string{}
	}
	if finalized.Category == "" {
		finalized.Category = inferCategory(finalized.ID)
	}
	if finalized.CodingEndpoint == "" {
		finalized.CodingEndpoint = finalized.DefaultEndpoint
	}

	return finalized
}

func providerTypeSortLess(result []ProviderTypeResponse, i, j int, categoryOrder map[string]int) bool {
	leftOrder, leftOK := categoryOrder[result[i].Category]
	rightOrder, rightOK := categoryOrder[result[j].Category]
	if !leftOK {
		leftOrder = 99
	}
	if !rightOK {
		rightOrder = 99
	}
	if leftOrder != rightOrder {
		return leftOrder < rightOrder
	}

	leftLabel := strings.ToLower(strings.TrimSpace(result[i].Label))
	rightLabel := strings.ToLower(strings.TrimSpace(result[j].Label))
	if leftLabel != rightLabel {
		return leftLabel < rightLabel
	}

	return result[i].ID < result[j].ID
}

func mergeProviderTypes(base map[string]ProviderTypeResponse, registry *provider.Registry, manager *limiter.AccountManager) []ProviderTypeResponse {
	merged := make(map[string]ProviderTypeResponse, len(base))

	for id := range base {
		baseItem := base[id]
		normalized, item, ok := normalizeProviderTypeItem(id, &baseItem)
		if !ok {
			continue
		}
		merged[normalized] = item
	}

	if registry != nil {
		providers := registry.List()
		for i := range providers {
			p := providers[i]
			mergeOneProviderType(merged, p.Name(), p.Models())
		}
	}

	if manager != nil {
		accounts := manager.GetAllAccounts()
		for i := range accounts {
			account := accounts[i]
			mergeOneProviderType(merged, accountProviderID(account), nil)
		}
	}

	result := make([]ProviderTypeResponse, 0, len(merged))
	for id := range merged {
		mergedItem := merged[id]
		item := finalizeProviderTypeItem(&mergedItem)
		result = append(result, item)
	}

	categoryOrder := map[string]int{
		providerCategoryInternational: 0,
		providerCategoryChinese:       1,
		providerCategoryLocal:         2,
		providerCategoryCustom:        3,
	}

	sort.Slice(result, func(i, j int) bool {
		return providerTypeSortLess(result, i, j, categoryOrder)
	})

	return result
}

func uniqueSortedStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	sort.Strings(result)
	return result
}

func (h *ProviderHandler) removeProviderFromConfig(providerName string) (bool, error) {
	root, err := loadConfigMap(h.configPath)
	if err != nil {
		return false, err
	}

	rawProviders, ok := root["providers"]
	if !ok {
		return false, nil
	}

	providers, ok := rawProviders.([]any)
	if !ok {
		return false, nil
	}

	changed := false
	filtered := make([]any, 0, len(providers))
	for _, item := range providers {
		providerConfig, castOK := item.(map[string]any)
		if !castOK {
			filtered = append(filtered, item)
			continue
		}
		nameValue, hasName := providerConfig["name"]
		name, nameOK := nameValue.(string)
		if !hasName || !nameOK {
			filtered = append(filtered, item)
			continue
		}
		if sameProvider(name, providerName) {
			changed = true
			continue
		}
		filtered = append(filtered, item)
	}

	if !changed {
		return false, nil
	}

	root["providers"] = filtered
	if err := writeConfigMapAtomic(h.configPath, root); err != nil {
		return false, err
	}

	return true, nil
}

func (h *ProviderHandler) getAccountCount(providerName string) int {
	if h.manager == nil {
		return 0
	}
	accounts := h.manager.GetAllAccounts()
	count := 0
	for _, acc := range accounts {
		providerType := acc.ProviderType
		if providerType == "" {
			providerType = acc.Provider
		}
		if providerType == providerName || acc.Provider == providerName {
			count++
		}
	}
	return count
}

func (h *ProviderHandler) checkProviderHealth(ctx context.Context, p provider.Provider) bool {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return p.ValidateKey(checkCtx)
}

func (h *ProviderHandler) removeProviderAccounts(providerName string) (int, error) {
	if h.manager == nil {
		return 0, nil
	}

	removedAccounts := 0
	accounts := h.manager.GetAllAccounts()
	for _, account := range accounts {
		providerType := account.ProviderType
		if providerType == "" {
			providerType = account.Provider
		}
		if !sameProvider(account.Provider, providerName) && !sameProvider(providerType, providerName) {
			continue
		}

		if err := h.manager.RemoveAccount(account.ID); err != nil {
			return 0, err
		}
		removedAccounts++
	}

	if removedAccounts == 0 {
		return 0, nil
	}

	if err := saveAccountsToFile(h.manager.GetAllAccounts()); err != nil {
		return 0, err
	}

	return removedAccounts, nil
}

// GET /api/admin/providers.
func (h *ProviderHandler) ListProviders(c *gin.Context) {
	providers := h.registry.List()

	response := make([]ProviderResponse, 0, len(providers))
	for _, p := range providers {
		healthy := h.checkProviderHealth(c.Request.Context(), p)
		providerResp := ProviderResponse{
			Name:         p.Name(),
			Models:       p.Models(),
			Enabled:      p.IsEnabled(),
			Healthy:      healthy,
			AccountCount: h.getAccountCount(p.Name()),
			LastCheck:    time.Now(),
		}

		// Get base URL if available
		if baseProvider, ok := p.(interface{ BaseURL() string }); ok {
			providerResp.BaseURL = baseProvider.BaseURL()
		}

		response = append(response, providerResp)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GET /api/admin/providers/types.
func (h *ProviderHandler) GetProviderTypes(c *gin.Context) {
	providerTypes := mergeProviderTypes(providerTypeCatalog, h.registry, h.manager)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    providerTypes,
	})
}

// GET /api/admin/providers/:id.
func (h *ProviderHandler) GetProvider(c *gin.Context) {
	providerName := c.Param("id")

	p, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    providerErrCodeNotFound,
				"message": providerErrMsgNotFound,
			},
		})
		return
	}

	response := ProviderResponse{
		Name:         p.Name(),
		Models:       p.Models(),
		Enabled:      p.IsEnabled(),
		Healthy:      h.checkProviderHealth(c.Request.Context(), p),
		AccountCount: h.getAccountCount(p.Name()),
		LastCheck:    time.Now(),
	}

	if baseProvider, ok := p.(interface{ BaseURL() string }); ok {
		response.BaseURL = baseProvider.BaseURL()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// POST /api/admin/providers.
func (h *ProviderHandler) CreateProvider(c *gin.Context) {
	var req ProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    providerErrCodeInvalidRequest,
				"message": err.Error(),
			},
		})
		return
	}

	// Convert to provider config
	config := &provider.ProviderConfig{
		Name:    req.Name,
		APIKey:  req.APIKey,
		BaseURL: req.BaseURL,
		Models:  req.Models,
		Enabled: req.Enabled,
		Extra:   req.Extra,
	}

	// Create and register provider
	p, err := h.registry.CreateAndRegister(config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "create_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"name":    p.Name(),
			"message": "Provider created successfully",
		},
	})
}

// PUT /api/admin/providers/:id.
func (h *ProviderHandler) UpdateProvider(c *gin.Context) {
	providerName := c.Param("id")

	var req ProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    providerErrCodeInvalidRequest,
				"message": err.Error(),
			},
		})
		return
	}

	p, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    providerErrCodeNotFound,
				"message": providerErrMsgNotFound,
			},
		})
		return
	}

	// Update enabled status
	p.SetEnabled(req.Enabled)

	// Note: Other fields like APIKey, BaseURL require provider-specific implementation

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":    providerName,
			"message": "Provider updated successfully",
		},
	})
}

// DELETE /api/admin/providers/:id.
func (h *ProviderHandler) DeleteProvider(c *gin.Context) {
	providerName := c.Param("id")
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    providerErrCodeInvalidRequest,
				"message": "Provider ID is required",
			},
		})
		return
	}

	removedConfig, err := h.removeProviderFromConfig(providerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "config_update_failed",
				"message": err.Error(),
			},
		})
		return
	}

	removedRegistry := false
	if _, ok := h.registry.Get(providerName); ok {
		h.registry.Remove(providerName)
		removedRegistry = true
	}

	removedAccounts, err := h.removeProviderAccounts(providerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "delete_account_failed",
				"message": err.Error(),
			},
		})
		return
	}

	removedModels := 0
	removedDefaults := false
	if h.router != nil {
		removedModels, removedDefaults = h.router.RemoveProviderData(providerName)
	}

	if !removedRegistry && removedAccounts == 0 && removedModels == 0 && !removedDefaults && !removedConfig {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    providerErrCodeNotFound,
				"message": providerErrMsgNotFound,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":             providerName,
			"message":          "Provider deleted successfully",
			"removed_registry": removedRegistry,
			"removed_accounts": removedAccounts,
			"removed_models":   removedModels,
			"removed_defaults": removedDefaults,
			"updated_config":   removedConfig,
		},
	})
}

// POST /api/admin/providers/:id/test.
func (h *ProviderHandler) TestProvider(c *gin.Context) {
	providerName := c.Param("id")

	p, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    providerErrCodeNotFound,
				"message": providerErrMsgNotFound,
			},
		})
		return
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	valid := p.ValidateKey(ctx)
	responseTime := time.Since(startTime).Milliseconds()

	result := ProviderTestResult{
		Success:      valid,
		ResponseTime: responseTime,
		Timestamp:    time.Now(),
	}

	if valid {
		result.Message = "Provider connection successful"
	} else {
		result.Message = "Provider connection failed - invalid API key or unreachable"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// POST /api/admin/providers/:id/enable.
func (h *ProviderHandler) EnableProvider(c *gin.Context) {
	h.setProviderEnabled(c, true)
}

// POST /api/admin/providers/:id/disable.
func (h *ProviderHandler) DisableProvider(c *gin.Context) {
	h.setProviderEnabled(c, false)
}

func (h *ProviderHandler) setProviderEnabled(c *gin.Context, enabled bool) {
	providerName := c.Param("id")

	p, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    providerErrCodeNotFound,
				"message": providerErrMsgNotFound,
			},
		})
		return
	}

	p.SetEnabled(enabled)

	message := "Provider disabled successfully"
	if enabled {
		message = "Provider enabled successfully"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":    providerName,
			"enabled": enabled,
			"message": message,
		},
	})
}

// GET /api/admin/providers/:id/models.
func (h *ProviderHandler) GetProviderModels(c *gin.Context) {
	providerName := c.Param("id")

	p, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    providerErrCodeNotFound,
				"message": providerErrMsgNotFound,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    p.Models(),
	})
}
