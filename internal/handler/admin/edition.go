package admin

import (
	"net/http"

	"ai-gateway/internal/config"

	"github.com/gin-gonic/gin"
)

type EditionHandler struct {
	configPath string
}

var dependencyStatusProvider = checkAllDependencies

func NewEditionHandler() *EditionHandler {
	return &EditionHandler{configPath: config.ResolveConfigPath()}
}

func (h *EditionHandler) GetEdition(c *gin.Context) {
	cfg, err := config.LoadFromPath(h.configPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "load_config_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cfg.GetEditionConfig(),
	})
}

func (h *EditionHandler) GetEditionDefinitions(c *gin.Context) {
	defs := make([]config.EditionDefinition, 0, len(config.EditionDefinitions))
	ordered := []config.EditionType{config.EditionBasic, config.EditionStandard, config.EditionEnterprise}
	for _, key := range ordered {
		defs = append(defs, config.EditionDefinitions[key])
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    defs,
	})
}

func (h *EditionHandler) CheckDependencies(c *gin.Context) {
	cfg, err := config.LoadFromPath(h.configPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "load_config_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dependencyStatusProvider(cfg),
	})
}

func (h *EditionHandler) UpdateEdition(c *gin.Context) {
	var req struct {
		Type config.EditionType `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if !config.IsValidEditionType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_edition",
			"message": "edition must be basic/standard/enterprise",
		})
		return
	}

	currentCfg, err := config.LoadFromPath(h.configPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "load_config_failed",
			"message": err.Error(),
		})
		return
	}

	def := config.EditionDefinitions[req.Type]
	missing := collectMissingDependencies(&def, dependencyStatusProvider(currentCfg))
	if len(missing) > 0 {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"success": false,
			"error":   "missing_dependencies",
			"message": "缺少必需依赖服务",
			"data": gin.H{
				"missing": missing,
			},
		})
		return
	}

	updatedCfg, err := config.UpdateEditionInFile(h.configPath, req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "update_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "版本配置已更新，重启后可确保全量生效",
		"data": gin.H{
			"restart_required": true,
			"edition":          updatedCfg.GetEditionConfig(),
		},
	})
}

func collectMissingDependencies(def *config.EditionDefinition, status map[string]DependencyStatus) []string {
	missing := make([]string, 0)
	for _, dep := range def.Dependencies {
		d, ok := status[dep]
		if !ok || !d.Healthy {
			missing = append(missing, dep)
		}
	}
	return missing
}
