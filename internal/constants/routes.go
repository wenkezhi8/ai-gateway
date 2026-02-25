package constants

// API 路径常量 - 统一管理所有 API 路径
// 修改接口时只需改这里
const (
	// API v1 前缀
	ApiV1Prefix = "/api/v1"

	// Anthropic API 前缀
	ApiAnthropicBasePrefix = "/api/anthropic"

	// Anthropic API v1 前缀
	ApiAnthropicPrefix = "/api/anthropic/v1"

	// Chat Completions
	ChatCompletions = ApiV1Prefix + "/chat/completions"

	// Completions
	Completions = ApiV1Prefix + "/completions"

	// Embeddings
	Embeddings = ApiV1Prefix + "/embeddings"

	// Providers
	Providers = ApiV1Prefix + "/providers"

	// Models
	Models = ApiV1Prefix + "/models"

	// Config Providers
	ConfigProviders = ApiV1Prefix + "/config/providers"
)

// Admin API 路径
const (
	AdminPrefix = "/api/admin"

	// Accounts
	Accounts = AdminPrefix + "/accounts"

	// Providers
	AdminProviders = AdminPrefix + "/providers"

	// Router
	Router = AdminPrefix + "/router"

	// Dashboard
	Dashboard = AdminPrefix + "/dashboard"

	// Cache
	Cache = AdminPrefix + "/cache"

	// API Keys
	ApiKeys = AdminPrefix + "/api-keys"

	// Upload
	UploadLogo = AdminPrefix + "/upload/logo"
)

// Auth API 路径
const (
	AuthPrefix = "/api/auth"

	Login          = AuthPrefix + "/login"
	Logout         = AuthPrefix + "/logout"
	Me             = AuthPrefix + "/me"
	Refresh        = AuthPrefix + "/refresh"
	ChangePassword = AuthPrefix + "/change-password"
	Validate       = AuthPrefix + "/validate"
)
