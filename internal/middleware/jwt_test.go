package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("password123")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "password123", hash)
}

func TestCheckPasswordHash(t *testing.T) {
	hash, _ := HashPassword("password123")

	assert.True(t, CheckPasswordHash("password123", hash))
	assert.False(t, CheckPasswordHash("wrongpassword", hash))
}

func TestGenerateAndParseToken(t *testing.T) {
	user := &User{
		ID:       "1",
		Username: "testuser",
		Role:     "admin",
	}

	config := JWTConfig{
		Secret:     "test-secret-key",
		ExpireTime: time.Hour,
		Issuer:     "test",
	}

	token, err := GenerateToken(user, config)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ParseToken(token, config.Secret)
	require.NoError(t, err)
	assert.Equal(t, "1", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "admin", claims.Role)
}

func TestParseToken_Invalid(t *testing.T) {
	_, err := ParseToken("invalid-token", "secret")
	assert.Error(t, err)
}

func TestParseToken_WrongSecret(t *testing.T) {
	user := &User{ID: "1", Username: "test", Role: "admin"}
	config := JWTConfig{Secret: "secret1", ExpireTime: time.Hour}

	token, _ := GenerateToken(user, config)

	_, err := ParseToken(token, "secret2")
	assert.Error(t, err)
}

func TestJWTAuth_NoHeader(t *testing.T) {
	config := JWTConfig{Secret: "test-secret", ExpireTime: time.Hour}

	router := gin.New()
	router.Use(JWTAuth(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	config := JWTConfig{Secret: "test-secret", ExpireTime: time.Hour}

	router := gin.New()
	router.Use(JWTAuth(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_Success(t *testing.T) {
	config := JWTConfig{Secret: "test-secret", ExpireTime: time.Hour}

	user := &User{ID: "1", Username: "testuser", Role: "admin"}
	token, _ := GenerateToken(user, config)

	router := gin.New()
	router.Use(JWTAuth(config))
	router.GET("/test", func(c *gin.Context) {
		userID, username, role := GetCurrentUser(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
			"role":     role,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	config := JWTConfig{Secret: "test-secret", ExpireTime: time.Hour}

	router := gin.New()
	router.Use(JWTAuth(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOptionalJWT_NoHeader(t *testing.T) {
	config := JWTConfig{Secret: "test-secret", ExpireTime: time.Hour}

	router := gin.New()
	router.Use(OptionalJWT(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOptionalJWT_WithToken(t *testing.T) {
	config := JWTConfig{Secret: "test-secret", ExpireTime: time.Hour}

	user := &User{ID: "1", Username: "testuser", Role: "admin"}
	token, _ := GenerateToken(user, config)

	router := gin.New()
	router.Use(OptionalJWT(config))
	router.GET("/test", func(c *gin.Context) {
		userID, username, role := GetCurrentUser(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
			"role":     role,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_NoRole(t *testing.T) {
	router := gin.New()
	router.Use(RequireRole("admin"))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireRole_InsufficientPermissions(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "test")
		c.Set("role", "viewer")
		c.Next()
	})
	router.Use(RequireRole("admin"))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireRole_Success(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Next()
	})
	router.Use(RequireRole("admin", "superadmin"))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetCurrentUser_Empty(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		userID, username, role := GetCurrentUser(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
			"role":     role,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuth_BearerCaseInsensitive(t *testing.T) {
	config := JWTConfig{Secret: "test-secret", ExpireTime: time.Hour}

	user := &User{ID: "1", Username: "testuser", Role: "admin"}
	token, _ := GenerateToken(user, config)

	router := gin.New()
	router.Use(JWTAuth(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOptionalJWT_InvalidFormat(t *testing.T) {
	config := JWTConfig{Secret: "test-secret", ExpireTime: time.Hour}

	router := gin.New()
	router.Use(OptionalJWT(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
