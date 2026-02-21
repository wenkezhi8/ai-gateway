package docs

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SwaggerInfo struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Host        string   `json:"host"`
	BasePath    string   `json:"basePath"`
	Schemes     []string `json:"schemes"`
}

var swaggerDoc = map[string]interface{}{
	"openapi": "3.0.3",
	"info": map[string]interface{}{
		"title":       "AI Gateway API",
		"description": "AI多服务商智能中转网关 - 统一管理多个AI服务商的API网关",
		"version":     "1.0.0",
		"contact": map[string]string{
			"name":  "AI Gateway Team",
			"email": "support@example.com",
		},
	},
	"servers": []map[string]interface{}{
		{
			"url":         "http://localhost:8566",
			"description": "开发服务器",
		},
	},
	"tags": []map[string]interface{}{
		{"name": "health", "description": "健康检查"},
		{"name": "chat", "description": "聊天补全接口"},
		{"name": "admin-accounts", "description": "账号管理"},
		{"name": "admin-providers", "description": "服务商管理"},
		{"name": "admin-routing", "description": "路由策略管理"},
		{"name": "admin-cache", "description": "缓存管理"},
		{"name": "admin-dashboard", "description": "仪表盘"},
		{"name": "auth", "description": "认证接口"},
		{"name": "audit", "description": "审计日志"},
	},
	"paths": map[string]interface{}{
		"/health": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":        []string{"health"},
				"summary":     "健康检查",
				"description": "检查服务是否正常运行",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "服务正常",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/HealthResponse",
								},
							},
						},
					},
				},
			},
		},
		"/api/v1/chat/completions": map[string]interface{}{
			"post": map[string]interface{}{
				"tags":        []string{"chat"},
				"summary":     "聊天补全",
				"description": "OpenAI兼容的聊天补全接口，支持流式响应",
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/ChatCompletionRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "成功响应",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ChatCompletionResponse",
								},
							},
						},
					},
				},
			},
		},
		"/api/admin/accounts": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":    []string{"admin-accounts"},
				"summary": "获取账号列表",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "账号列表",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/AccountListResponse",
								},
							},
						},
					},
				},
			},
			"post": map[string]interface{}{
				"tags":    []string{"admin-accounts"},
				"summary": "创建账号",
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/AccountCreateRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"201": map[string]interface{}{
						"description": "创建成功",
					},
				},
			},
		},
		"/api/admin/providers": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":    []string{"admin-providers"},
				"summary": "获取服务商列表",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "服务商列表",
					},
				},
			},
		},
		"/api/admin/routing": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":    []string{"admin-routing"},
				"summary": "获取路由配置",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "路由配置",
					},
				},
			},
		},
		"/api/admin/cache/stats": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":    []string{"admin-cache"},
				"summary": "获取缓存统计",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "缓存统计",
					},
				},
			},
		},
		"/api/admin/dashboard/stats": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":    []string{"admin-dashboard"},
				"summary": "获取仪表盘统计",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "统计数据",
					},
				},
			},
		},
		"/api/auth/login": map[string]interface{}{
			"post": map[string]interface{}{
				"tags":    []string{"auth"},
				"summary": "用户登录",
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/LoginRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "登录成功",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/LoginResponse",
								},
							},
						},
					},
					"401": map[string]interface{}{
						"description": "认证失败",
					},
				},
			},
		},
		"/api/audit/logs": map[string]interface{}{
			"get": map[string]interface{}{
				"tags":    []string{"audit"},
				"summary": "获取审计日志",
				"parameters": []map[string]interface{}{
					{"name": "limit", "in": "query", "schema": map[string]interface{}{"type": "integer", "default": 100}},
					{"name": "offset", "in": "query", "schema": map[string]interface{}{"type": "integer", "default": 0}},
					{"name": "user_id", "in": "query", "schema": map[string]interface{}{"type": "string"}},
					{"name": "action", "in": "query", "schema": map[string]interface{}{"type": "string"}},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "审计日志列表",
					},
				},
			},
		},
	},
	"components": map[string]interface{}{
		"schemas": map[string]interface{}{
			"HealthResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status":    map[string]interface{}{"type": "string", "example": "healthy"},
					"timestamp": map[string]interface{}{"type": "string", "format": "date-time"},
					"service":   map[string]interface{}{"type": "string", "example": "ai-gateway"},
				},
			},
			"ChatCompletionRequest": map[string]interface{}{
				"type":     "object",
				"required": []string{"model", "messages"},
				"properties": map[string]interface{}{
					"model":       map[string]interface{}{"type": "string", "example": "gpt-4"},
					"messages":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"$ref": "#/components/schemas/Message"}},
					"temperature": map[string]interface{}{"type": "number", "example": 0.7},
					"max_tokens":  map[string]interface{}{"type": "integer", "example": 1000},
					"stream":      map[string]interface{}{"type": "boolean", "default": false},
				},
			},
			"Message": map[string]interface{}{
				"type":     "object",
				"required": []string{"role", "content"},
				"properties": map[string]interface{}{
					"role":    map[string]interface{}{"type": "string", "enum": []string{"system", "user", "assistant"}},
					"content": map[string]interface{}{"type": "string"},
				},
			},
			"ChatCompletionResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":      map[string]interface{}{"type": "string"},
					"object":  map[string]interface{}{"type": "string"},
					"created": map[string]interface{}{"type": "integer"},
					"model":   map[string]interface{}{"type": "string"},
					"choices": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object"}},
					"usage":   map[string]interface{}{"$ref": "#/components/schemas/Usage"},
				},
			},
			"Usage": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt_tokens":     map[string]interface{}{"type": "integer"},
					"completion_tokens": map[string]interface{}{"type": "integer"},
					"total_tokens":      map[string]interface{}{"type": "integer"},
				},
			},
			"AccountListResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{"type": "array", "items": map[string]interface{}{"$ref": "#/components/schemas/Account"}},
				},
			},
			"Account": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":       map[string]interface{}{"type": "string"},
					"name":     map[string]interface{}{"type": "string"},
					"provider": map[string]interface{}{"type": "string"},
					"enabled":  map[string]interface{}{"type": "boolean"},
					"priority": map[string]interface{}{"type": "integer"},
				},
			},
			"AccountCreateRequest": map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "provider", "api_key"},
				"properties": map[string]interface{}{
					"id":       map[string]interface{}{"type": "string"},
					"name":     map[string]interface{}{"type": "string"},
					"provider": map[string]interface{}{"type": "string"},
					"api_key":  map[string]interface{}{"type": "string"},
					"base_url": map[string]interface{}{"type": "string"},
					"enabled":  map[string]interface{}{"type": "boolean"},
					"priority": map[string]interface{}{"type": "integer"},
				},
			},
			"LoginRequest": map[string]interface{}{
				"type":     "object",
				"required": []string{"username", "password"},
				"properties": map[string]interface{}{
					"username": map[string]interface{}{"type": "string"},
					"password": map[string]interface{}{"type": "string"},
				},
			},
			"LoginResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"token": map[string]interface{}{"type": "string"},
					"user":  map[string]interface{}{"$ref": "#/components/schemas/UserInfo"},
				},
			},
			"UserInfo": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":       map[string]interface{}{"type": "string"},
					"username": map[string]interface{}{"type": "string"},
					"role":     map[string]interface{}{"type": "string"},
				},
			},
		},
		"securitySchemes": map[string]interface{}{
			"bearerAuth": map[string]interface{}{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
			},
			"apiKeyAuth": map[string]interface{}{
				"type": "apiKey",
				"in":   "header",
				"name": "X-API-Key",
			},
		},
	},
}

func SwaggerJSON(c *gin.Context) {
	c.JSON(http.StatusOK, swaggerDoc)
}

func SetupSwaggerRoutes(r *gin.Engine) {
	r.GET("/swagger/doc.json", SwaggerJSON)

	r.GET("/swagger/index.html", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, swaggerUIHTML)
	})
}

const swaggerUIHTML = `<!DOCTYPE html>
<html>
<head>
    <title>AI Gateway API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
    window.onload = function() {
        const ui = SwaggerUIBundle({
            url: "/swagger/doc.json",
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIStandalonePreset
            ],
            layout: "StandaloneLayout",
            deepLinking: true,
            displayOperationId: false,
            defaultModelsExpandDepth: 1,
            defaultModelExpandDepth: 1,
        })
        window.ui = ui
    }
    </script>
</body>
</html>`
