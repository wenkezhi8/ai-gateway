package constants

// 修改接口时只需改这里.
const (
	// API v1 前缀.
	ApiV1Prefix = "/api/v1" //nolint:revive // Keep legacy exported name for compatibility.

	// Anthropic API 前缀.
	ApiAnthropicBasePrefix = "/api/anthropic" //nolint:revive // Keep legacy exported name for compatibility.

	// Anthropic API v1 前缀.
	ApiAnthropicPrefix = "/api/anthropic/v1" //nolint:revive // Keep legacy exported name for compatibility.

	// Chat Completions.
	ChatCompletions = ApiV1Prefix + "/chat/completions"

	// Completions.
	Completions = ApiV1Prefix + "/completions"

	// Embeddings.
	Embeddings = ApiV1Prefix + "/embeddings"

	// Providers.
	Providers = ApiV1Prefix + "/providers"

	// Models.
	Models = ApiV1Prefix + "/models"

	// Config Providers.
	ConfigProviders = ApiV1Prefix + "/config/providers"
)

// Admin API 路径.
const (
	AdminPrefix = "/api/admin"

	// Accounts.
	Accounts = AdminPrefix + "/accounts"

	// Providers.
	AdminProviders = AdminPrefix + "/providers"

	// Router.
	Router = AdminPrefix + "/router"

	// Dashboard.
	Dashboard = AdminPrefix + "/dashboard"

	// Cache.
	Cache = AdminPrefix + "/cache"

	// API Keys.
	ApiKeys = AdminPrefix + "/api-keys" //nolint:revive // Keep legacy exported name for compatibility.

	// Upload.
	UploadLogo = AdminPrefix + "/upload/logo"

	// UI settings.
	AdminSettingsUI = AdminPrefix + "/settings/ui"
)

// Auth API 路径.
const (
	AuthPrefix = "/api/auth"

	Login          = AuthPrefix + "/login"
	Logout         = AuthPrefix + "/logout"
	Me             = AuthPrefix + "/me"
	Refresh        = AuthPrefix + "/refresh"
	ChangePassword = AuthPrefix + "/change-password"
	Validate       = AuthPrefix + "/validate"
)
