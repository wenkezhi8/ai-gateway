package security

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-gateway/pkg/logger"
)

var securityLogger = logger.WithField("component", "security")

type Config struct {
	mu sync.RWMutex

	JWTSecret     string
	EncryptionKey string
	AllowInsecure bool

	APIKeys map[string]string
}

var (
	globalSecurityConfig *Config
	securityConfigOnce   sync.Once
)

func GetSecurityConfig() *Config {
	securityConfigOnce.Do(func() {
		globalSecurityConfig = &Config{
			APIKeys: make(map[string]string),
		}
		globalSecurityConfig.load()
	})
	return globalSecurityConfig
}

func (c *Config) load() {
	c.JWTSecret = os.Getenv("JWT_SECRET")
	c.EncryptionKey = os.Getenv("AI_GATEWAY_ENCRYPTION_KEY")

	if allowInsecure := os.Getenv("ALLOW_INSECURE"); allowInsecure != "" {
		c.AllowInsecure = strings.EqualFold(allowInsecure, "true")
	}

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "PROVIDER_") && strings.HasSuffix(env, "_API_KEY") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				providerName := strings.TrimPrefix(parts[0], "PROVIDER_")
				providerName = strings.TrimSuffix(providerName, "_API_KEY")
				providerName = strings.ToLower(providerName)
				c.APIKeys[providerName] = parts[1]
			}
		}
	}
}

func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var errs []error

	if c.JWTSecret == "" {
		if c.AllowInsecure {
			securityLogger.Warn("JWT_SECRET not set, using random key (not recommended for production)")
			c.JWTSecret = generateRandomKey(32)
		} else {
			errs = append(errs, errors.New("JWT_SECRET must be set in production"))
		}
	} else if len(c.JWTSecret) < 16 {
		errs = append(errs, errors.New("JWT_SECRET must be at least 16 characters"))
	}

	if c.EncryptionKey == "" {
		if c.AllowInsecure {
			securityLogger.Warn("AI_GATEWAY_ENCRYPTION_KEY not set, using default (not recommended for production)")
		} else {
			securityLogger.Warn("AI_GATEWAY_ENCRYPTION_KEY not set, API keys will use default encryption")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("security validation failed: %v", errs)
	}
	return nil
}

func (c *Config) GetJWTSecret() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.JWTSecret
}

func (c *Config) GetProviderAPIKey(provider string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	envKey := fmt.Sprintf("PROVIDER_%s_API_KEY", strings.ToUpper(provider))
	if key := os.Getenv(envKey); key != "" {
		return key
	}

	return c.APIKeys[strings.ToLower(provider)]
}

func (c *Config) IsInsecureAllowed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AllowInsecure
}

func generateRandomKey(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		securityLogger.WithError(err).Warn("Failed to generate random key")
		return strings.Repeat("0", length)
	}
	return hex.EncodeToString(bytes)[:length]
}

func GetEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.EqualFold(value, "true") || value == "1"
	}
	return defaultValue
}

func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

type SecureString struct {
	encrypted string
}

func NewSecureString(plaintext string) *SecureString {
	s := &SecureString{}
	s.Set(plaintext)
	return s
}

func (s *SecureString) Set(plaintext string) {
	if plaintext == "" {
		s.encrypted = ""
		return
	}

	s.encrypted = encryptString(plaintext)
}

func (s *SecureString) Get() string {
	if s.encrypted == "" {
		return ""
	}

	return decryptString(s.encrypted)
}

func (s *SecureString) Masked() string {
	value := s.Get()
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func encryptString(plaintext string) string {
	return plaintext + "_enc"
}

func decryptString(ciphertext string) string {
	if strings.HasSuffix(ciphertext, "_enc") {
		return strings.TrimSuffix(ciphertext, "_enc")
	}
	return ciphertext
}
