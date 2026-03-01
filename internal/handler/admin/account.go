package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/limiter"
	"ai-gateway/internal/routing"
	"ai-gateway/pkg/logger"

	"github.com/gin-gonic/gin"
)

const accountsFile = "data/accounts.json"
const switchHistoryFile = "data/switch_history.json"

const (
	planTypeLite     = "Lite"
	planTypePro      = "Pro"
	planTypeMax      = "Max"
	planTypeBasic    = "Basic"
	planTypeStandard = "Standard"
)

var (
	accountsMu     sync.Mutex
	accountsLogger = logger.WithField("component", "admin_account")
	globalRouter   *routing.SmartRouter
)

// SetGlobalRouter sets the global smart router for model syncing.
func SetGlobalRouter(router *routing.SmartRouter) {
	globalRouter = router
}

// AccountHandler handles account management requests.
type AccountHandler struct {
	manager *limiter.AccountManager
}

// NewAccountHandler creates a new account handler.
func NewAccountHandler(manager *limiter.AccountManager) *AccountHandler {
	return &AccountHandler{
		manager: manager,
	}
}

// PersistedAccount represents an account for JSON persistence.
type PersistedAccount struct {
	ID                string                    `json:"id"`
	Name              string                    `json:"name"`
	Provider          string                    `json:"provider"`
	ProviderType      string                    `json:"provider_type,omitempty"`
	APIKey            string                    `json:"api_key"`
	BaseURL           string                    `json:"base_url"`
	Enabled           bool                      `json:"enabled"`
	Priority          int                       `json:"priority"`
	Concurrency       int                       `json:"concurrency,omitempty"`
	HealthStatus      string                    `json:"health_status,omitempty"`
	Limits            map[string]PersistedLimit `json:"limits,omitempty"`
	CodingPlanEnabled bool                      `json:"coding_plan_enabled,omitempty"`
}

type PersistedLimit struct {
	Type    string  `json:"type"`
	Period  string  `json:"period"`
	Limit   int64   `json:"limit"`
	Warning float64 `json:"warning"`
}

func saveAccountsToFile(accounts []*limiter.AccountConfig) error {
	accountsMu.Lock()
	defer accountsMu.Unlock()

	dir := filepath.Dir(accountsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	persisted := make([]PersistedAccount, 0, len(accounts))
	for _, acc := range accounts {
		pa := PersistedAccount{
			ID:                acc.ID,
			Name:              acc.Name,
			Provider:          acc.Provider,
			ProviderType:      acc.ProviderType,
			APIKey:            acc.APIKey,
			BaseURL:           acc.BaseURL,
			Enabled:           acc.Enabled,
			Priority:          acc.Priority,
			Concurrency:       acc.Concurrency,
			HealthStatus:      acc.HealthStatus,
			Limits:            make(map[string]PersistedLimit),
			CodingPlanEnabled: acc.CodingPlanEnabled,
		}
		for k, v := range acc.Limits {
			pa.Limits[string(k)] = PersistedLimit{
				Type:    string(v.Type),
				Period:  string(v.Period),
				Limit:   v.Limit,
				Warning: v.Warning,
			}
		}
		persisted = append(persisted, pa)
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(accountsFile, data, 0640)
}

// LoadPersistedAccounts loads accounts from the persistence file.
func LoadPersistedAccounts() ([]*limiter.AccountConfig, error) {
	accountsMu.Lock()
	defer accountsMu.Unlock()

	data, err := os.ReadFile(accountsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var persisted []PersistedAccount
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, err
	}

	accounts := make([]*limiter.AccountConfig, 0, len(persisted))
	for i := range persisted {
		pa := &persisted[i]
		config := &limiter.AccountConfig{
			ID:                pa.ID,
			Name:              pa.Name,
			Provider:          pa.Provider,
			ProviderType:      pa.ProviderType,
			APIKey:            pa.APIKey,
			BaseURL:           pa.BaseURL,
			Enabled:           pa.Enabled,
			Priority:          pa.Priority,
			Concurrency:       pa.Concurrency,
			HealthStatus:      pa.HealthStatus,
			Limits:            make(map[limiter.LimitType]*limiter.LimitConfig),
			CodingPlanEnabled: pa.CodingPlanEnabled,
		}
		for k, v := range pa.Limits {
			config.Limits[limiter.LimitType(k)] = &limiter.LimitConfig{
				Type:    limiter.LimitType(v.Type),
				Period:  limiter.Period(v.Period),
				Limit:   v.Limit,
				Warning: v.Warning,
			}
		}
		accounts = append(accounts, config)
	}

	return accounts, nil
}

func SaveSwitchHistoryToFile(history []limiter.SwitchEvent) error {
	accountsMu.Lock()
	defer accountsMu.Unlock()

	dir := filepath.Dir(switchHistoryFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if history == nil {
		history = []limiter.SwitchEvent{}
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(switchHistoryFile, data, 0640)
}

func LoadPersistedSwitchHistory() ([]limiter.SwitchEvent, error) {
	accountsMu.Lock()
	defer accountsMu.Unlock()

	data, err := os.ReadFile(switchHistoryFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var history []limiter.SwitchEvent
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return history, nil
}

// GET /api/admin/accounts.
func (h *AccountHandler) ListAccounts(c *gin.Context) {
	accounts := h.manager.GetAllAccounts()

	response := make([]AccountResponse, 0, len(accounts))
	for _, acc := range accounts {
		accountResp := convertAccountToResponse(acc, h.manager)
		response = append(response, accountResp)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GET /api/admin/accounts/:id.
func (h *AccountHandler) GetAccount(c *gin.Context) {
	accountID := c.Param("id")

	accounts := h.manager.GetAllAccounts()
	var found *limiter.AccountConfig
	for _, acc := range accounts {
		if acc.ID == accountID {
			found = acc
			break
		}
	}

	if found == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Account not found",
			},
		})
		return
	}

	response := convertAccountToResponse(found, h.manager)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// POST /api/admin/accounts.
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req AccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	config := convertRequestToAccountConfig(&req)
	if strings.TrimSpace(config.ID) == "" {
		config.ID = generateAccountID(config.Provider)
	}

	// Map frontend provider names to backend provider types
	// Keep original provider name for display, use ProviderType for routing
	config.ProviderType = mapProviderToBackend(config.Provider)

	if err := h.manager.AddAccount(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "create_failed",
				"message": err.Error(),
			},
		})
		return
	}

	// Persist accounts to file
	if err := saveAccountsToFile(h.manager.GetAllAccounts()); err != nil {
		accountsLogger.WithError(err).Warn("Failed to persist accounts to file")
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":      config.ID,
			"message": "Account created successfully",
		},
	})
}

func generateAccountID(provider string) string {
	p := strings.TrimSpace(strings.ToLower(provider))
	if p == "" {
		p = "account"
	}
	p = strings.ReplaceAll(p, " ", "-")
	return fmt.Sprintf("%s-%d", p, time.Now().UnixNano())
}

// mapProviderToBackend maps frontend provider names to backend provider types.
func mapProviderToBackend(frontendProvider string) string {
	// Providers that use OpenAI-compatible API
	openaiCompatible := map[string]bool{
		"openai":       true,
		"deepseek":     true,
		"moonshot":     true,
		"qwen":         true,
		"zhipu":        true,
		"baichuan":     true,
		"minimax":      true,
		"volcengine":   true,
		"yi":           true,
		"azure-openai": true,
	}

	if openaiCompatible[frontendProvider] {
		return "openai"
	}
	return frontendProvider
}

// PUT /api/admin/accounts/:id.
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	accountID := c.Param("id")

	var req AccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	// Get existing account for partial update
	existingConfig, err := h.manager.GetAccount(accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Account not found",
			},
		})
		return
	}

	// Merge updates - only update fields that are provided
	config := existingConfig
	config.ID = accountID
	if req.Name != "" {
		config.Name = req.Name
	}
	if req.Provider != "" {
		config.Provider = req.Provider
		config.ProviderType = mapProviderToBackend(req.Provider)
	}
	if req.APIKey != "" {
		config.APIKey = req.APIKey
	}
	if req.BaseURL != "" {
		config.BaseURL = req.BaseURL
	}
	// Only update enabled status if it was explicitly provided
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.Priority != 0 {
		config.Priority = req.Priority
	}
	if req.Limits != nil {
		config.Limits = make(map[limiter.LimitType]*limiter.LimitConfig)
		for k, v := range req.Limits {
			config.Limits[limiter.LimitType(k)] = &limiter.LimitConfig{
				Type:    limiter.LimitType(v.Type),
				Period:  limiter.Period(v.Period),
				Limit:   v.Limit,
				Warning: v.Warning,
			}
		}
	}
	if req.CodingPlanEnabled != nil {
		config.CodingPlanEnabled = *req.CodingPlanEnabled
	}

	if err := h.manager.UpdateAccount(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "update_failed",
				"message": err.Error(),
			},
		})
		return
	}

	// Persist accounts to file
	if err := saveAccountsToFile(h.manager.GetAllAccounts()); err != nil {
		accountsLogger.WithError(err).Warn("Failed to persist accounts to file")
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":      accountID,
			"message": "Account updated successfully",
		},
	})
}

// DELETE /api/admin/accounts/:id.
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	accountID := c.Param("id")

	if err := h.manager.RemoveAccount(accountID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": err.Error(),
			},
		})
		return
	}

	// Persist accounts to file
	if err := saveAccountsToFile(h.manager.GetAllAccounts()); err != nil {
		accountsLogger.WithError(err).Warn("Failed to persist accounts to file")
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":      accountID,
			"message": "Account deleted successfully",
		},
	})
}

// PUT /api/admin/accounts/:id/status.
func (h *AccountHandler) UpdateAccountStatus(c *gin.Context) {
	accountID := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	// Get existing account
	config, err := h.manager.GetAccount(accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Account not found",
			},
		})
		return
	}

	config.Enabled = req.Enabled

	if err := h.manager.UpdateAccount(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "update_failed",
				"message": err.Error(),
			},
		})
		return
	}

	// Persist accounts to file
	if err := saveAccountsToFile(h.manager.GetAllAccounts()); err != nil {
		accountsLogger.WithError(err).Warn("Failed to persist accounts to file")
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":      accountID,
			"enabled": req.Enabled,
		},
	})
}

// GET /api/admin/accounts/:id/usage.
func (h *AccountHandler) GetAccountUsage(c *gin.Context) {
	accountID := c.Param("id")

	status, err := h.manager.GetAccountStatus(accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Account not found",
			},
		})
		return
	}

	usage := &AccountUsageResponse{}
	for limitType, usageData := range status.CurrentUsage {
		switch limitType {
		case limiter.LimitTypeToken:
			usage.TokensUsed = usageData.Used
			usage.TokenLimit = usageData.Limit
			usage.TokenPercent = usageData.PercentUsed
			usage.WarningLevel = usageData.WarningLevel
		case limiter.LimitTypeRPM:
			usage.RequestsCount = usageData.Used
			usage.RPM = int(usageData.Used)
			usage.RPMLimit = int(usageData.Limit)
		case limiter.LimitTypeHour5:
			usage.Hour5Used = usageData.Used
			usage.Hour5Limit = usageData.Limit
			usage.Hour5Percent = usageData.PercentUsed
		case limiter.LimitTypeWeek:
			usage.WeekUsed = usageData.Used
			usage.WeekLimit = usageData.Limit
			usage.WeekPercent = usageData.PercentUsed
		case limiter.LimitTypeMonth:
			usage.MonthUsed = usageData.Used
			usage.MonthLimit = usageData.Limit
			usage.MonthPercent = usageData.PercentUsed
		case limiter.LimitTypeConcurrent, limiter.LimitTypeRequest:
			continue
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    usage,
	})
}

// getDefaultModelsForProvider returns default models for providers that don't support /models API.
func getDefaultModelsForProvider(provider string) []string {
	defaults := map[string][]string{
		"minimax": {
			"abab6.5s-chat",
			"abab6.5g-chat",
			"abab6.5t-chat",
			"abab6.5-chat",
			"abab5.5-chat",
			"abab5.5s-chat",
		},
		"ernie": {
			"ernie-4.0-8k",
			"ernie-4.0",
			"ernie-3.5-8k",
			"ernie-3.5",
			"ernie-speed-128k",
			"ernie-speed-8k",
		},
		"hunyuan": {
			"hunyuan-turbo",
			"hunyuan-pro",
			"hunyuan-standard",
			"hunyuan-lite",
			"hunyuan-code",
		},
		"spark": {
			"spark-4.0-ultra",
			"spark-3.5-max",
			"spark-3.0",
			"spark-2.0",
			"spark-lite",
		},
		"anthropic": {
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
		},
		"google": {
			"gemini-2.0-flash",
			"gemini-1.5-pro",
			"gemini-1.5-flash",
			"gemini-1.0-pro",
		},
	}

	if models, ok := defaults[provider]; ok {
		return models
	}
	return []string{}
}

// ProviderConfigResponse represents a provider configuration for frontend.
type ProviderConfigResponse struct {
	Value    string   `json:"value"`
	Label    string   `json:"label"`
	Color    string   `json:"color"`
	BaseURL  string   `json:"base_url"`
	Models   []string `json:"models,omitempty"`
	IsOpenAI bool     `json:"is_openai_compatible"`
}

// GET /api/admin/providers/configs.
func (h *AccountHandler) GetProviderConfigs(c *gin.Context) {
	providers := []ProviderConfigResponse{
		// 国际服务商
		{Value: "openai", Label: "OpenAI", Color: "#10A37F", BaseURL: "https://api.openai.com/v1", IsOpenAI: true},
		{Value: "anthropic", Label: "Anthropic Claude", Color: "#CC785C", BaseURL: "https://api.anthropic.com/v1", IsOpenAI: false},
		{Value: "azure-openai", Label: "Azure OpenAI", Color: "#0078D4", BaseURL: "https://your-resource.openai.azure.com", IsOpenAI: true},
		{Value: "google", Label: "Google Gemini", Color: "#4285F4", BaseURL: "https://generativelanguage.googleapis.com/v1beta", IsOpenAI: false},

		// 国内服务商
		{Value: "deepseek", Label: "DeepSeek", Color: "#4D6BFE", BaseURL: "https://api.deepseek.com/v1", IsOpenAI: true},
		{Value: "qwen", Label: "阿里云通义千问", Color: "#FF6A00", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", IsOpenAI: true},
		{Value: "zhipu", Label: "智谱AI", Color: "#3657ED", BaseURL: "https://open.bigmodel.cn/api/paas/v4", IsOpenAI: true},
		{Value: "moonshot", Label: "月之暗面 (Kimi)", Color: "#1A1A1A", BaseURL: "https://api.moonshot.cn/v1", IsOpenAI: true},
		{Value: "kimi", Label: "月之暗面 (Kimi)", Color: "#1A1A1A", BaseURL: "https://api.moonshot.cn/v1", IsOpenAI: true},
		{Value: "minimax", Label: "MiniMax", Color: "#615CED", BaseURL: "https://api.minimax.chat/v1", IsOpenAI: true},
		{Value: "baichuan", Label: "百川智能", Color: "#0066FF", BaseURL: "https://api.baichuan-ai.com/v1", IsOpenAI: true},
		{Value: "volcengine", Label: "火山方舟 (豆包)", Color: "#FF4D4F", BaseURL: "https://ark.cn-beijing.volces.com/api/v3", IsOpenAI: true},
		{Value: "ernie", Label: "百度文心一言", Color: "#2932E1", BaseURL: "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat", IsOpenAI: true},
		{Value: "hunyuan", Label: "腾讯混元", Color: "#00A3FF", BaseURL: "https://api.hunyuan.cloud.tencent.com/v1", IsOpenAI: true},
		{Value: "spark", Label: "讯飞星火", Color: "#E60012", BaseURL: "https://spark-api-open.xf-yun.com/v1", IsOpenAI: true},
		{Value: "yi", Label: "零一万物", Color: "#00D4AA", BaseURL: "https://api.lingyiwanwu.com/v1", IsOpenAI: true},
		{Value: "mistral", Label: "Mistral AI", Color: "#FF7000", BaseURL: "https://api.mistral.ai/v1", IsOpenAI: true},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    providers,
	})
}

// POST /api/admin/accounts/:id/switch.
func (h *AccountHandler) ForceSwitchAccount(c *gin.Context) {
	accountID := c.Param("id")

	// Get account to find provider
	accounts := h.manager.GetAllAccounts()
	var provider string
	for _, acc := range accounts {
		if acc.ID == accountID {
			// Use ProviderType if set, otherwise use Provider
			if acc.ProviderType != "" {
				provider = acc.ProviderType
			} else {
				provider = acc.Provider
			}
			break
		}
	}

	if provider == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Account not found",
			},
		})
		return
	}

	if err := h.manager.ForceSwitch(provider, accountID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "switch_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":      accountID,
			"message": "Account switched successfully",
		},
	})
}

// GET /api/admin/accounts/switch-history.
func (h *AccountHandler) GetSwitchHistory(c *gin.Context) {
	limit := 50 // default limit

	history := h.manager.GetSwitchHistory(limit)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    history,
	})
}

// Helper functions

func convertAccountToResponse(acc *limiter.AccountConfig, manager *limiter.AccountManager) AccountResponse {
	response := AccountResponse{
		ID:                acc.ID,
		Name:              acc.Name,
		Provider:          acc.Provider,
		APIKey:            acc.APIKey,
		BaseURL:           acc.BaseURL,
		Enabled:           acc.Enabled,
		Priority:          acc.Priority,
		Limits:            make(map[string]LimitConfig),
		CodingPlanEnabled: acc.CodingPlanEnabled,
	}

	for limitType, limitCfg := range acc.Limits {
		response.Limits[string(limitType)] = LimitConfig{
			Type:    string(limitCfg.Type),
			Period:  string(limitCfg.Period),
			Limit:   limitCfg.Limit,
			Warning: limitCfg.Warning,
		}
		if string(limitType) == "hour5" || string(limitType) == "week" || string(limitType) == "month" {
			response.PlanType = detectPlanType(acc.Provider, acc.Limits)
		}
	}

	if status, err := manager.GetAccountStatus(acc.ID); err == nil {
		response.IsActive = status.IsActive
		response.LastSwitch = status.LastSwitched

		if len(status.CurrentUsage) > 0 {
			usage := &AccountUsageResponse{}
			for limitType, usageData := range status.CurrentUsage {
				switch string(limitType) {
				case "token":
					usage.TokensUsed = usageData.Used
					usage.TokenLimit = usageData.Limit
					usage.TokenPercent = usageData.PercentUsed
					usage.WarningLevel = usageData.WarningLevel
				case "rpm":
					usage.RequestsCount = usageData.Used
					usage.RPM = int(usageData.Used)
					usage.RPMLimit = int(usageData.Limit)
				case "hour5":
					usage.Hour5Used = usageData.Used
					usage.Hour5Limit = usageData.Limit
					usage.Hour5Percent = usageData.PercentUsed
				case "week":
					usage.WeekUsed = usageData.Used
					usage.WeekLimit = usageData.Limit
					usage.WeekPercent = usageData.PercentUsed
				case "month":
					usage.MonthUsed = usageData.Used
					usage.MonthLimit = usageData.Limit
					usage.MonthPercent = usageData.PercentUsed
				}
			}
			response.Usage = usage
		}
	}

	return response
}

//nolint:gocyclo
func detectPlanType(provider string, limits map[limiter.LimitType]*limiter.LimitConfig) string {
	if limits == nil {
		return ""
	}

	var hour5Limit, weekLimit int64
	if l, ok := limits[limiter.LimitType("hour5")]; ok {
		hour5Limit = l.Limit
	}
	if l, ok := limits[limiter.LimitType("week")]; ok {
		weekLimit = l.Limit
	}

	switch provider {
	case "zhipu":
		if hour5Limit <= 80 && weekLimit <= 400 {
			return planTypeLite
		}
		if hour5Limit <= 400 && weekLimit <= 2000 {
			return planTypePro
		}
		if hour5Limit > 400 {
			return planTypeMax
		}
	case "bailian", "qwen":
		if hour5Limit <= 1200 && weekLimit <= 9000 {
			return planTypeLite
		}
		if hour5Limit > 1200 {
			return planTypePro
		}
	case "volcengine":
		if hour5Limit <= 500 {
			return planTypeBasic
		}
		if hour5Limit <= 2000 {
			return planTypeStandard
		}
		return planTypePro
	}
	return ""
}

func convertRequestToAccountConfig(req *AccountRequest) *limiter.AccountConfig {
	enabled := true // default to true for new accounts
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	config := &limiter.AccountConfig{
		ID:       req.ID,
		Name:     req.Name,
		Provider: req.Provider,
		APIKey:   req.APIKey,
		BaseURL:  req.BaseURL,
		Enabled:  enabled,
		Priority: req.Priority,
		Limits:   make(map[limiter.LimitType]*limiter.LimitConfig),
	}

	for k, v := range req.Limits {
		config.Limits[limiter.LimitType(k)] = &limiter.LimitConfig{
			Type:    limiter.LimitType(v.Type),
			Period:  limiter.Period(v.Period),
			Limit:   v.Limit,
			Warning: v.Warning,
		}
	}

	return config
}

// ProviderModelsResponse represents the response from provider's /v1/models API.
type ProviderModelsResponse struct {
	Data []ProviderModel `json:"data"`
}

type ProviderModel struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Created     int    `json:"created,omitempty"`
	OwnedBy     string `json:"owned_by,omitempty"`
	Type        string `json:"type,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// GET /api/admin/accounts/:id/fetch-models?sync=true.
//
//nolint:gocyclo
func (h *AccountHandler) FetchModels(c *gin.Context) {
	accountID := c.Param("id")
	syncModels := c.Query("sync") == "true"

	account, err := h.manager.GetAccount(accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Account not found",
			},
		})
		return
	}

	if !account.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "account_disabled",
				"message": "Account is disabled",
			},
		})
		return
	}

	if account.APIKey == "" || account.BaseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "missing_credentials",
				"message": "Account missing API key or base URL",
			},
		})
		return
	}

	// Build the models endpoint URL
	baseURL := strings.TrimRight(account.BaseURL, "/")
	modelsURL := baseURL + "/models"

	// Create HTTP client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, http.NoBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "request_failed",
				"message": "Failed to create request: " + err.Error(),
			},
		})
		return
	}

	// Set authorization header
	req.Header.Set("Authorization", "Bearer "+account.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "request_failed",
				"message": "Failed to fetch models: " + err.Error(),
			},
		})
		return
	}
	defer resp.Body.Close()

	// Handle non-OK responses
	if resp.StatusCode == http.StatusNotFound {
		// Provider doesn't support /models endpoint, use default models
		models := getDefaultModelsForProvider(account.Provider)

		// Sync to smart router if requested
		syncedCount := 0
		if syncModels && globalRouter != nil {
			for _, modelID := range models {
				globalRouter.UpdateModelScore(modelID, &routing.ModelScore{
					Model:        modelID,
					Provider:     account.Provider,
					QualityScore: 80,
					SpeedScore:   80,
					CostScore:    80,
					Enabled:      true,
				})
				syncedCount++
			}
		}

		response := gin.H{
			"account_id":   accountID,
			"provider":     account.Provider,
			"models":       models,
			"total_models": len(models),
			"fallback":     true,
		}
		if syncModels {
			response["synced"] = true
			response["synced_count"] = syncedCount
		}

		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_supported",
				"message": "该服务商不支持自动获取模型列表，已使用默认模型",
			},
			"data": response,
		})
		return
	}

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = []byte(fmt.Sprintf("读取服务商错误响应失败: %v", readErr))
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "provider_error",
				"message": fmt.Sprintf("服务商返回错误 (状态码 %d): %s", resp.StatusCode, string(body)),
			},
		})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "read_failed",
				"message": "Failed to read response: " + err.Error(),
			},
		})
		return
	}

	var modelsResp map[string]interface{}
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "parse_failed",
				"message": "Failed to parse response: " + err.Error(),
			},
		})
		return
	}

	// Keep provider model format as-is, only extract id for sync
	rawData, ok := modelsResp["data"].([]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "parse_failed",
				"message": "Provider response missing data array",
			},
		})
		return
	}

	models := make([]map[string]interface{}, 0, len(rawData))
	modelIDs := make([]string, 0, len(rawData))
	modelDisplayNames := make(map[string]string, len(rawData))
	for _, item := range rawData {
		modelObj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		modelID, ok := modelObj["id"].(string)
		if !ok || modelID == "" {
			continue
		}

		models = append(models, modelObj)
		modelIDs = append(modelIDs, modelID)

		if displayName, ok := modelObj["display_name"].(string); ok && displayName != "" {
			modelDisplayNames[modelID] = displayName
		}
	}

	// Sync to smart router if requested
	syncedCount := 0
	if syncModels && globalRouter != nil {
		for _, modelID := range modelIDs {
			globalRouter.UpdateModelScore(modelID, &routing.ModelScore{
				Model:        modelID,
				Provider:     account.Provider,
				DisplayName:  modelDisplayNames[modelID],
				QualityScore: 80,
				SpeedScore:   80,
				CostScore:    80,
				Enabled:      true,
			})
			syncedCount++
		}
	}

	response := gin.H{
		"account_id":   accountID,
		"provider":     account.Provider,
		"models":       models,
		"total_models": len(models),
	}

	if syncModels {
		response["synced"] = true
		response["synced_count"] = syncedCount
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}
