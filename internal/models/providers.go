package models

var ProviderModels = map[string][]string{
	"openai": {
		"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4", "gpt-3.5-turbo",
		"o1", "o1-mini", "o1-preview", "o3-mini",
	},
	"anthropic": {
		"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022",
		"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307",
	},
	"deepseek": {
		"deepseek-chat", "deepseek-reasoner", "deepseek-coder",
	},
	"qwen": {
		"qwen-max", "qwen-max-longcontext", "qwen-plus", "qwen-turbo",
		"qwen-long", "qwen-vl-max", "qwen-vl-plus",
	},
	"zhipu": {
		"glm-4-plus", "glm-4-0520", "glm-4", "glm-4-air", "glm-4-airx",
		"glm-4-flash", "glm-4v", "glm-4v-plus",
	},
	"moonshot":   {"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"},
	"minimax":    {"abab6.5s-chat", "abab6.5g-chat", "abab6.5-chat", "abab5.5-chat"},
	"baichuan":   {"Baichuan4", "Baichuan3-Turbo", "Baichuan3-Turbo-128k"},
	"volcengine": {"doubao-pro-4k", "doubao-pro-32k", "doubao-pro-128k"},
	"ernie":      {"ernie-4.0-8k", "ernie-3.5-8k", "ernie-speed-8k"},
	"yi":         {"yi-lightning", "yi-large", "yi-medium", "yi-spark"},
	"google":     {"gemini-1.5-pro", "gemini-1.5-flash", "gemini-pro"},
	"mistral":    {"mistral-large-latest", "mistral-medium", "codestral-latest"},
	"hunyuan":    {"hunyuan-lite", "hunyuan-standard", "hunyuan-pro"},
	"spark":      {"spark-v3.5", "spark-v3.0"},
}

var ProviderBaseURLs = map[string]string{
	"openai":     "https://api.openai.com/v1",
	"anthropic":  "https://api.anthropic.com/v1",
	"deepseek":   "https://api.deepseek.com",
	"qwen":       "https://dashscope.aliyuncs.com/compatible-mode/v1",
	"zhipu":      "https://open.bigmodel.cn/api/paas/v4",
	"moonshot":   "https://api.moonshot.cn/v1",
	"minimax":    "https://api.minimax.chat/v1",
	"baichuan":   "https://api.baichuan-ai.com/v1",
	"volcengine": "https://ark.cn-beijing.volces.com/api/v3",
	"ernie":      "https://aip.baidubce.com/rpc/2.0/ai_custom/v1",
	"yi":         "https://api.lingyiwanwu.com/v1",
	"google":     "https://generativelanguage.googleapis.com/v1beta",
	"mistral":    "https://api.mistral.ai/v1",
	"hunyuan":    "https://hunyuan.tencentcloudapi.com",
	"spark":      "https://spark-api-open.xf-yun.com/v1",
}

func GetProviderModels(provider string) []string {
	return ProviderModels[provider]
}

func GetProviderBaseURL(provider string) string {
	return ProviderBaseURLs[provider]
}

func IsProviderSupported(provider string) bool {
	_, ok := ProviderModels[provider]
	return ok
}

func GetAllProviders() []string {
	providers := make([]string, 0, len(ProviderModels))
	for p := range ProviderModels {
		providers = append(providers, p)
	}
	return providers
}

type AccountRecord struct {
	ID           string `json:"id"`
	Provider     string `json:"provider"`
	APIKey       string `json:"api_key"`
	Priority     int    `json:"priority"`
	Enabled      bool   `json:"enabled"`
	QuotaLimit   int64  `json:"quota_limit"`
	QuotaUsed    int64  `json:"quota_used"`
	QuotaResetAt string `json:"quota_reset_at,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type ModelScoreRecord struct {
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	QualityScore int    `json:"quality_score"`
	SpeedScore   int    `json:"speed_score"`
	CostScore    int    `json:"cost_score"`
	Enabled      bool   `json:"enabled"`
	IsCustom     bool   `json:"is_custom"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type RouterConfigRecord struct {
	DefaultStrategy string `json:"default_strategy"`
	DefaultModel    string `json:"default_model"`
	UseAutoMode     bool   `json:"use_auto_mode"`
}

type APIKeyRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Permissions string `json:"permissions"`
	Enabled     bool   `json:"enabled"`
	LastUsedAt  string `json:"last_used_at,omitempty"`
	CreatedAt   string `json:"created_at"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type UserRecord struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	Email        string `json:"email,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type FeedbackRecord struct {
	ID         string `json:"id"`
	RequestID  string `json:"request_id"`
	Model      string `json:"model"`
	Provider   string `json:"provider"`
	TaskType   string `json:"task_type"`
	Rating     int    `json:"rating"`
	Comment    string `json:"comment"`
	LatencyMs  int64  `json:"latency_ms"`
	TokensUsed int64  `json:"tokens_used"`
	CacheHit   bool   `json:"cache_hit"`
	CreatedAt  string `json:"created_at"`
}
