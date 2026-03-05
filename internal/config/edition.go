package config

import "strings"

type EditionType string

const (
	EditionBasic      EditionType = "basic"
	EditionStandard   EditionType = "standard"
	EditionEnterprise EditionType = "enterprise"
)

type EditionRuntime string

const (
	EditionRuntimeDocker EditionRuntime = "docker"
	EditionRuntimeNative EditionRuntime = "native"
)

const (
	DependencyRedis  = "redis"
	DependencyOllama = "ollama"
	DependencyQdrant = "qdrant"
)

const (
	DefaultRedisVersion  = "7.2.0-v18"
	DefaultOllamaVersion = "latest"
	DefaultQdrantVersion = "latest"
)

type EditionFeatures struct {
	VectorCache        bool `json:"vector_cache"`
	VectorDBManagement bool `json:"vector_db_management"`
	KnowledgeBase      bool `json:"knowledge_base"`
	ColdHotTiering     bool `json:"cold_hot_tiering"`
}

type EditionDefinition struct {
	Type         EditionType     `json:"type"`
	Features     EditionFeatures `json:"features"`
	DisplayName  string          `json:"display_name"`
	Description  string          `json:"description"`
	Dependencies []string        `json:"dependencies"`
}

type EditionConfig struct {
	Type               string            `json:"type"`
	Runtime            string            `json:"runtime"`
	DependencyVersions map[string]string `json:"dependency_versions"`
}

func DefaultEditionDependencyVersions() map[string]string {
	return map[string]string{
		DependencyRedis:  DefaultRedisVersion,
		DependencyOllama: DefaultOllamaVersion,
		DependencyQdrant: DefaultQdrantVersion,
	}
}

func CloneEditionDependencyVersions(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func IsValidEditionRuntime(runtime string) bool {
	switch EditionRuntime(strings.TrimSpace(strings.ToLower(runtime))) {
	case EditionRuntimeDocker, EditionRuntimeNative:
		return true
	default:
		return false
	}
}

func normalizeEditionRuntime(runtime string) string {
	v := strings.TrimSpace(strings.ToLower(runtime))
	if !IsValidEditionRuntime(v) {
		return string(EditionRuntimeDocker)
	}
	return v
}

func normalizeEditionDependencyVersions(raw map[string]string) map[string]string {
	defaults := DefaultEditionDependencyVersions()
	if len(raw) == 0 {
		return defaults
	}
	normalized := make(map[string]string, len(defaults))
	for key, fallback := range defaults {
		value := strings.TrimSpace(raw[key])
		if value == "" {
			value = fallback
		}
		normalized[key] = value
	}
	return normalized
}

func normalizeEditionConfig(ec *EditionConfig) {
	if ec == nil {
		return
	}
	ec.Runtime = normalizeEditionRuntime(ec.Runtime)
	ec.DependencyVersions = normalizeEditionDependencyVersions(ec.DependencyVersions)
}

var EditionDefinitions = map[EditionType]EditionDefinition{
	EditionBasic: {
		Type:        EditionBasic,
		DisplayName: "基础版",
		Description: "纯AI网关功能，轻量级部署",
		Features: EditionFeatures{
			VectorCache:        false,
			VectorDBManagement: false,
			KnowledgeBase:      false,
			ColdHotTiering:     false,
		},
		Dependencies: []string{},
	},
	EditionStandard: {
		Type:        EditionStandard,
		DisplayName: "标准版",
		Description: "网关 + 语义缓存，中大规模场景",
		Features: EditionFeatures{
			VectorCache:        true,
			VectorDBManagement: false,
			KnowledgeBase:      false,
			ColdHotTiering:     false,
		},
		Dependencies: []string{"redis", "ollama"},
	},
	EditionEnterprise: {
		Type:        EditionEnterprise,
		DisplayName: "企业版",
		Description: "完整功能，企业级生产环境",
		Features: EditionFeatures{
			VectorCache:        true,
			VectorDBManagement: true,
			KnowledgeBase:      true,
			ColdHotTiering:     true,
		},
		Dependencies: []string{"redis", "ollama", "qdrant"},
	},
}

func (c *Config) GetEditionConfig() EditionDefinition {
	editionType := EditionType(strings.TrimSpace(strings.ToLower(c.Edition.Type)))
	if _, ok := EditionDefinitions[editionType]; !ok {
		editionType = EditionStandard
	}
	return EditionDefinitions[editionType]
}
