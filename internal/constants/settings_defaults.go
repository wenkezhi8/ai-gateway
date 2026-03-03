package constants

var SettingsDefaults = map[string]any{
	"gateway": map[string]any{
		"host":            "0.0.0.0",
		"port":            8080,
		"timeout":         30,
		"max_connections": 1000,
		"enable_cors":     true,
		"cors_origins":    "*",
	},
	"cache": map[string]any{
		"enabled":     true,
		"type":        "memory",
		"default_ttl": 3600,
		"max_size":    1024,
		"redis": map[string]any{
			"host":     "localhost:6379",
			"password": "",
			"db":       0,
		},
	},
	"logging": map[string]any{
		"level":         "info",
		"format":        "json",
		"outputs":       []string{"console"},
		"file_path":     "/var/log/ai-gateway",
		"max_file_size": 100,
		"max_backups":   7,
	},
	"security": map[string]any{
		"enabled":        true,
		"type":           "apikey",
		"rate_limit":     true,
		"rate_limit_rpm": 100,
		"ip_whitelist":   "",
	},
}
