package config

import "strings"

type EditionType string

const (
	EditionBasic      EditionType = "basic"
	EditionStandard   EditionType = "standard"
	EditionEnterprise EditionType = "enterprise"
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
	Type string `json:"type"`
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
		Dependencies: []string{"redis"},
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
		editionType = EditionBasic
	}
	return EditionDefinitions[editionType]
}
