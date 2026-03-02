package constants

import "time"

const (
	ModelScoresFilePath         = "data/model_scores.json"
	ProviderDefaultsFilePath    = "data/provider_defaults.json"
	RouterConfigFilePath        = "data/router_config.json"
	RouterUIConfigFilePath      = "data/router_ui_config.json"
	UISettingsFilePath          = "data/ui-settings.json"
	ClassifierSwitchTaskDBPath  = "data/classifier_switch_tasks.db"
	OllamaRuntimeConfigFilePath = "data/ollama_runtime_config.json"
)

const (
	RoutingDefaultStrategy = "auto"
	RoutingDefaultModel    = "deepseek-chat"

	RoutingCascadeFallbackStartLevel             = "medium"
	RoutingCascadeFallbackMaxLevel               = "large"
	RoutingCascadeFallbackEnabled                = true
	RoutingCascadeFallbackMaxRetries             = 2
	RoutingCascadeFallbackTimeoutPerLevelSeconds = 20

	RoutingAutoQualityWeight = 0.4
	RoutingAutoSpeedWeight   = 0.35
	RoutingAutoCostWeight    = 0.25
)

const (
	ClassifierDefaultProvider            = "ollama"
	ClassifierDefaultBaseURL             = "http://127.0.0.1:11434"
	ClassifierDefaultModel               = "qwen2.5:0.5b-instruct"
	ClassifierDefaultTimeoutMs           = 5000
	ClassifierDefaultConfidenceThreshold = 0.65
	ClassifierDefaultMaxInputChars       = 4000
)

const (
	AdminResolveClassifierBaseTimeout  = 2 * time.Second
	AdminResolveClassifierMaxTimeout   = 5 * time.Second
	AdminClassifierHealthTimeout       = 5 * time.Second
	AdminClassifierSwitchAsyncMaxWait  = 180 * time.Second
	AdminClassifierSwitchProbeTimeout  = 5 * time.Second
	AdminClassifierSwitchProbeInterval = 2 * time.Second
	AdminOllamaCheckTimeout            = 3 * time.Second
	AdminOllamaInstallTimeout          = 20 * time.Minute
	AdminOllamaStartCommandTimeout     = 10 * time.Second
	AdminOllamaStartReadyDeadline      = 12 * time.Second
	AdminOllamaStartProbeTimeout       = 1500 * time.Millisecond
	AdminOllamaStartProbeInterval      = 500 * time.Millisecond
	AdminOllamaPullTimeout             = 30 * time.Minute
)
