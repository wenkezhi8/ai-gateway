package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-gateway/internal/audit"
	"ai-gateway/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestHandler(t *testing.T) (*AuthHandler, *gin.Engine) {
	jwtConfig := middleware.JWTConfig{
		Secret:     "test-secret-key-for-testing",
		ExpireTime: 3600,
	}

	auditLogger := &audit.Logger{}

	handler := NewAuthHandler(jwtConfig, auditLogger)

	router := gin.New()
	return handler, router
}

func TestNewAuthHandler(t *testing.T) {
	handler, _ := setupTestHandler(t)

	require.NotNil(t, handler)
	assert.NotNil(t, handler.users)
	assert.Contains(t, handler.users, "admin")
}

func TestLogin_Success(t *testing.T) {
	handler, router := setupTestHandler(t)

	handler.users["admin"].PasswordHash = mustHashPassword("admin123")

	router.POST("/login", handler.Login)

	body := LoginRequest{Username: "admin", Password: "admin123"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "admin", resp.User.Username)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.POST("/login", handler.Login)

	body := LoginRequest{Username: "admin", Password: "wrongpassword"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_InvalidRequest(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.POST("/login", handler.Login)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_UserNotFound(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.POST("/login", handler.Login)

	body := LoginRequest{Username: "nonexistent", Password: "password"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChangePassword_Success(t *testing.T) {
	handler, router := setupTestHandler(t)

	handler.users["admin"].PasswordHash = mustHashPassword("admin123")

	router.PUT("/password", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.ChangePassword(c)
	})

	body := ChangePasswordRequest{OldPassword: "admin123", NewPassword: "newpassword123"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChangePassword_InvalidOldPassword(t *testing.T) {
	handler, router := setupTestHandler(t)

	handler.users["admin"].PasswordHash = mustHashPassword("admin123")

	router.PUT("/password", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.ChangePassword(c)
	})

	body := ChangePasswordRequest{OldPassword: "wrongpassword", NewPassword: "newpassword123"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListUsers(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.GET("/users", handler.ListUsers)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp, "data")
}

func TestCreateUser_Success(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.POST("/users", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.CreateUser(c)
	})

	body := CreateUserRequest{Username: "testuser", Password: "password123", Role: "viewer"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateUser_Duplicate(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.POST("/users", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.CreateUser(c)
	})

	body := CreateUserRequest{Username: "admin", Password: "password123", Role: "viewer"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestDeleteUser_Success(t *testing.T) {
	handler, router := setupTestHandler(t)

	handler.users["testuser"] = &middleware.User{
		ID:           "2",
		Username:     "testuser",
		PasswordHash: "hash",
		Role:         "viewer",
	}

	router.DELETE("/users/:username", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.DeleteUser(c)
	})

	req := httptest.NewRequest("DELETE", "/users/testuser", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteUser_CannotDeleteAdmin(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.DELETE("/users/:username", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.DeleteUser(c)
	})

	req := httptest.NewRequest("DELETE", "/users/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteUser_NotFound(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.DELETE("/users/:username", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.DeleteUser(c)
	})

	req := httptest.NewRequest("DELETE", "/users/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateProfile_Success(t *testing.T) {
	handler, router := setupTestHandler(t)

	handler.users["testuser"] = &middleware.User{
		ID:           "2",
		Username:     "testuser",
		PasswordHash: "hash",
		Role:         "viewer",
	}

	router.PUT("/profile", func(c *gin.Context) {
		c.Set("user_id", "2")
		c.Set("username", "testuser")
		c.Set("role", "viewer")
		handler.UpdateProfile(c)
	})

	newUsername := "updated_" + time.Now().Format("20060102150405")
	body := UpdateProfileRequest{Username: newUsername}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateProfile_EmptyUsername(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.PUT("/profile", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.UpdateProfile(c)
	})

	body := UpdateProfileRequest{Username: ""}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateProfile_ShortUsername(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.PUT("/profile", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.UpdateProfile(c)
	})

	body := UpdateProfileRequest{Username: "ab"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateProfile_Conflict(t *testing.T) {
	handler, router := setupTestHandler(t)

	handler.users["testuser"] = &middleware.User{
		ID:           "2",
		Username:     "testuser",
		PasswordHash: "hash",
		Role:         "viewer",
	}

	router.PUT("/profile", func(c *gin.Context) {
		c.Set("user_id", "2")
		c.Set("username", "testuser")
		c.Set("role", "viewer")
		handler.UpdateProfile(c)
	})

	body := UpdateProfileRequest{Username: "admin"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestGetCurrentUser(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.GET("/me", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.GetCurrentUser(c)
	})

	req := httptest.NewRequest("GET", "/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogout(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.POST("/logout", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.Logout(c)
	})

	req := httptest.NewRequest("POST", "/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateToken(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.GET("/validate", handler.ValidateToken)

	token, _ := middleware.GenerateToken(handler.users["admin"], handler.jwtConfig)

	req := httptest.NewRequest("GET", "/validate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateToken_NoToken(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.GET("/validate", handler.ValidateToken)

	req := httptest.NewRequest("GET", "/validate", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.GET("/validate", handler.ValidateToken)

	req := httptest.NewRequest("GET", "/validate", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, false, resp["valid"])
}

func TestRefreshToken(t *testing.T) {
	handler, router := setupTestHandler(t)

	router.POST("/refresh", func(c *gin.Context) {
		c.Set("user_id", "1")
		c.Set("username", "admin")
		c.Set("role", "admin")
		handler.RefreshToken(c)
	})

	req := httptest.NewRequest("POST", "/refresh", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
}

func TestMustHashPassword(t *testing.T) {
	hash := mustHashPassword("password123")
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "password123", hash)
}

func TestExtractClaims(t *testing.T) {
	_, router := setupTestHandler(t)

	router.GET("/extract", func(c *gin.Context) {
		claims := &middleware.Claims{
			UserID:   "1",
			Username: "admin",
			Role:     "admin",
		}
		c.Set("jwt_claims", claims)
		extracted := ExtractClaims(c)
		require.NotNil(t, extracted)
		c.JSON(200, extracted)
	})

	req := httptest.NewRequest("GET", "/extract", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExtractClaims_NoClaims(t *testing.T) {
	_, router := setupTestHandler(t)

	router.GET("/extract", func(c *gin.Context) {
		extracted := ExtractClaims(c)
		if extracted == nil {
			c.Status(200)
		}
	})

	req := httptest.NewRequest("GET", "/extract", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
