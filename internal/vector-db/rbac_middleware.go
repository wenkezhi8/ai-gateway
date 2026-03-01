package vectordb

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type RBACMiddleware struct {
	service *RBACService
}

func NewRBACMiddleware(service *RBACService) *RBACMiddleware {
	return &RBACMiddleware{service: service}
}

func (m *RBACMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m == nil || m.service == nil {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
			c.Abort()
			return
		}

		rawKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if rawKey == "" {
			authorization := strings.TrimSpace(c.GetHeader("Authorization"))
			if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
				rawKey = strings.TrimSpace(authorization[len("Bearer "):])
			}
		}
		permission := resolveVectorPermission(c.Request.Method, c.Request.URL.Path)
		allowed, err := m.service.CheckPermission(c.Request.Context(), rawKey, permission)
		if err != nil || !allowed {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func resolveVectorPermission(method, path string) VectorPermission {
	if method == http.MethodGet && strings.Contains(path, "/vectors/") {
		return VectorPermissionRead
	}
	if method == http.MethodPost && strings.HasSuffix(path, "/recommend") {
		return VectorPermissionRecommend
	}
	return VectorPermissionSearch
}
