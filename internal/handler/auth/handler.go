package auth

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ai-gateway/internal/audit"
	"ai-gateway/internal/middleware"
	"ai-gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const usersDataFile = "data/users.json"

var authLogger = logger.WithField("component", "auth")

//nolint:revive // keep exported name for compatibility with existing callers.
type AuthHandler struct {
	jwtConfig middleware.JWTConfig
	users     map[string]*middleware.User
	mu        sync.RWMutex
	auditLog  *audit.Logger
}

type UserPersist struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at,omitempty"`
}

func NewAuthHandler(jwtConfig middleware.JWTConfig, auditLog *audit.Logger) *AuthHandler {
	h := &AuthHandler{
		jwtConfig: jwtConfig,
		users:     make(map[string]*middleware.User),
		auditLog:  auditLog,
	}

	h.loadFromFile()

	if len(h.users) == 0 {
		h.users["admin"] = &middleware.User{
			ID:           "1",
			Username:     "admin",
			PasswordHash: mustHashPassword("admin123"),
			Role:         "admin",
			CreatedAt:    time.Now().Unix(),
		}
		if err := h.saveToFile(); err != nil {
			authLogger.WithError(err).Warn("Failed to persist default admin user")
		}
		authLogger.Info("Created default admin user")
	}

	return h
}

func (h *AuthHandler) loadFromFile() {
	data, err := os.ReadFile(usersDataFile)
	if err != nil {
		authLogger.WithError(err).Debug("No saved users file, will use defaults")
		return
	}

	var savedUsers map[string]*UserPersist
	if err := json.Unmarshal(data, &savedUsers); err != nil {
		authLogger.WithError(err).Warn("Failed to parse users file")
		return
	}

	for username, u := range savedUsers {
		if u != nil {
			h.users[username] = &middleware.User{
				ID:           u.ID,
				Username:     u.Username,
				PasswordHash: u.PasswordHash,
				Role:         u.Role,
				CreatedAt:    u.CreatedAt,
			}
		}
	}

	authLogger.Infof("Loaded %d users from file", len(h.users))
}

func (h *AuthHandler) saveToFile() error {
	dir := filepath.Dir(usersDataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	savedUsers := make(map[string]*UserPersist)
	for username, u := range h.users {
		savedUsers[username] = &UserPersist{
			ID:           u.ID,
			Username:     u.Username,
			PasswordHash: u.PasswordHash,
			Role:         u.Role,
			CreatedAt:    u.CreatedAt,
		}
	}

	data, err := json.MarshalIndent(savedUsers, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(usersDataFile, data, 0600)
}

func mustHashPassword(password string) string {
	hash, err := middleware.HashPassword(password)
	if err != nil {
		panic(err)
	}
	return hash
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string    `json:"token"`
	User  *UserInfo `json:"user"`
}

type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin viewer"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": "Invalid request body"}})
		return
	}

	h.mu.RLock()
	user, exists := h.users[req.Username]
	h.mu.RUnlock()

	if !exists || !middleware.CheckPasswordHash(req.Password, user.PasswordHash) {
		h.logAudit("", req.Username, c.ClientIP(), c.Request.UserAgent(), audit.ActionLogin, audit.ResourceAuth, "", "Login failed", "failed", "Invalid credentials")
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "Invalid username or password"}})
		return
	}

	token, err := middleware.GenerateToken(user, h.jwtConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": "Failed to generate token"}})
		return
	}

	h.logAudit(user.ID, user.Username, c.ClientIP(), c.Request.UserAgent(), audit.ActionLogin, audit.ResourceAuth, "", "Login successful", "success", "")

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, username, _ := middleware.GetCurrentUser(c)

	h.logAudit(userID, username, c.ClientIP(), c.Request.UserAgent(), audit.ActionLogout, audit.ResourceAuth, "", "Logout", "success", "")

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, username, role := middleware.GetCurrentUser(c)

	c.JSON(http.StatusOK, gin.H{
		"user": &UserInfo{
			ID:       userID,
			Username: username,
			Role:     role,
		},
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	userID, username, _ := middleware.GetCurrentUser(c)

	h.mu.Lock()
	defer h.mu.Unlock()

	user, exists := h.users[username]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "User not found"}})
		return
	}

	if !middleware.CheckPasswordHash(req.OldPassword, user.PasswordHash) {
		h.logAudit(userID, username, c.ClientIP(), c.Request.UserAgent(), audit.ActionUpdate, audit.ResourceAuth, userID, "Password change failed", "failed", "Invalid old password")
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_password", "message": "Invalid old password"}})
		return
	}

	newHash, err := middleware.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": "Failed to hash password"}})
		return
	}

	user.PasswordHash = newHash

	if err := h.saveToFile(); err != nil {
		authLogger.WithError(err).Warn("Failed to save users to file after password change")
	}

	h.logAudit(userID, username, c.ClientIP(), c.Request.UserAgent(), audit.ActionUpdate, audit.ResourceAuth, userID, "Password changed", "success", "")

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// UpdateProfileRequest represents the request for updating user profile.
type UpdateProfileRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// UpdateProfile updates the current user's profile.
// 改动点: 新增用户资料更新接口，支持修改用户名.
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	userID, currentUsername, _ := middleware.GetCurrentUser(c)

	if req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": "Username cannot be empty"}})
		return
	}

	if len(req.Username) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": "Username must be at least 3 characters"}})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	user, exists := h.users[currentUsername]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "User not found"}})
		return
	}

	// Check if new username is already taken by another user
	if req.Username != currentUsername {
		if _, exists := h.users[req.Username]; exists {
			c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "conflict", "message": "Username already exists"}})
			return
		}

		// Remove old key and add new one
		delete(h.users, currentUsername)
		user.Username = req.Username
		h.users[req.Username] = user
	}

	if err := h.saveToFile(); err != nil {
		authLogger.WithError(err).Warn("Failed to save users to file after profile update")
	}

	h.logAudit(userID, currentUsername, c.ClientIP(), c.Request.UserAgent(), audit.ActionUpdate, audit.ResourceAuth, userID, "Profile updated", "success", "New username: "+req.Username)

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]*UserInfo, 0, len(h.users))
	for _, u := range h.users {
		users = append(users, &UserInfo{
			ID:       u.ID,
			Username: u.Username,
			Role:     u.Role,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	userID, username, _ := middleware.GetCurrentUser(c)

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.users[req.Username]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "conflict", "message": "Username already exists"}})
		return
	}

	hash, err := middleware.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": "Failed to hash password"}})
		return
	}

	newUser := &middleware.User{
		ID:           time.Now().Format("20060102150405"),
		Username:     req.Username,
		PasswordHash: hash,
		Role:         req.Role,
		CreatedAt:    time.Now().Unix(),
	}

	h.users[req.Username] = newUser

	if err := h.saveToFile(); err != nil {
		authLogger.WithError(err).Warn("Failed to save users to file after creating user")
	}

	h.logAudit(userID, username, c.ClientIP(), c.Request.UserAgent(), audit.ActionCreate, audit.ResourceAuth, newUser.ID, "Created user: "+req.Username, "success", "")

	c.JSON(http.StatusCreated, gin.H{
		"user": &UserInfo{
			ID:       newUser.ID,
			Username: newUser.Username,
			Role:     newUser.Role,
		},
	})
}

func (h *AuthHandler) DeleteUser(c *gin.Context) {
	targetUsername := c.Param("username")

	userID, username, _ := middleware.GetCurrentUser(c)

	if targetUsername == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "forbidden", "message": "Cannot delete admin user"}})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.users[targetUsername]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "User not found"}})
		return
	}

	delete(h.users, targetUsername)

	if err := h.saveToFile(); err != nil {
		authLogger.WithError(err).Warn("Failed to save users to file after deleting user")
	}

	h.logAudit(userID, username, c.ClientIP(), c.Request.UserAgent(), audit.ActionDelete, audit.ResourceAuth, targetUsername, "Deleted user: "+targetUsername, "success", "")

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	userID, username, role := middleware.GetCurrentUser(c)

	h.mu.RLock()
	user, exists := h.users[username]
	h.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "User not found"}})
		return
	}

	token, err := middleware.GenerateToken(user, h.jwtConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": "Failed to generate token"}})
		return
	}

	h.logAudit(userID, username, c.ClientIP(), c.Request.UserAgent(), audit.ActionLogin, audit.ResourceAuth, "", "Token refreshed", "success", "")

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Role:     role,
		},
	})
}

func (h *AuthHandler) ValidateToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "error": "No token provided"})
		return
	}

	tokenString := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenString = authHeader[7:]
	}

	claims, err := middleware.ParseToken(tokenString, h.jwtConfig.Secret)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user": &UserInfo{
			ID:       claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
		},
	})
}

//nolint:unparam // keep parameter for future non-auth resource audit events.
func (h *AuthHandler) logAudit(userID, username, ip, userAgent string, action audit.ActionType, resource audit.ResourceType, resourceID, detail, status, errMsg string) {
	if h.auditLog != nil {
		h.auditLog.Log(audit.LogEntry{
			UserID:     userID,
			Username:   username,
			IP:         ip,
			UserAgent:  userAgent,
			Action:     action,
			Resource:   resource,
			ResourceID: resourceID,
			Detail:     detail,
			Status:     status,
			Error:      errMsg,
		})
	}
}

func ExtractClaims(c *gin.Context) *middleware.Claims {
	if v, exists := c.Get("jwt_claims"); exists {
		if claims, ok := v.(*middleware.Claims); ok {
			return claims
		}
	}
	return nil
}

func ParseTokenMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		claims, err := jwt.ParseWithClaims(tokenString, &middleware.Claims{}, func(_ *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err == nil && claims != nil {
			if parsedClaims, ok := claims.Claims.(*middleware.Claims); ok {
				c.Set("jwt_claims", parsedClaims)
			}
		}

		c.Next()
	}
}
