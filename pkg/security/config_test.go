package security

import (
	"os"
	"testing"
)

func TestGetSecurityConfig(t *testing.T) {
	config := GetSecurityConfig()
	if config == nil {
		t.Fatal("GetSecurityConfig() returned nil")
	}
}

func TestSecurityConfig_Validate(t *testing.T) {
	tests := []struct {
		name          string
		jwtSecret     string
		allowInsecure bool
		wantErr       bool
	}{
		{
			name:          "valid secret",
			jwtSecret:     "this-is-a-valid-secret-key",
			allowInsecure: false,
			wantErr:       false,
		},
		{
			name:          "short secret",
			jwtSecret:     "short",
			allowInsecure: false,
			wantErr:       true,
		},
		{
			name:          "empty secret with insecure",
			jwtSecret:     "",
			allowInsecure: true,
			wantErr:       false,
		},
		{
			name:          "empty secret without insecure",
			jwtSecret:     "",
			allowInsecure: false,
			wantErr:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{
				JWTSecret:     tc.jwtSecret,
				AllowInsecure: tc.allowInsecure,
			}

			err := config.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestGetEnvString(t *testing.T) {
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	if got := GetEnvString("TEST_KEY", "default"); got != "test_value" {
		t.Errorf("GetEnvString() = %q, want %q", got, "test_value")
	}

	if got := GetEnvString("NONEXISTENT_KEY", "default"); got != "default" {
		t.Errorf("GetEnvString() = %q, want %q", got, "default")
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		envValue   string
		defaultVal bool
		expected   bool
	}{
		{"true", false, true},
		{"false", true, false},
		{"1", false, true},
		{"0", true, false},
		{"", true, true},
		{"", false, false},
	}

	for _, tc := range tests {
		t.Setenv("TEST_BOOL", tc.envValue)

		if got := GetEnvBool("TEST_BOOL", tc.defaultVal); got != tc.expected {
			t.Errorf("GetEnvBool(%q, %v) = %v, want %v", tc.envValue, tc.defaultVal, got, tc.expected)
		}
	}
}

func TestSecureString(t *testing.T) {
	original := "my-secret-api-key"
	ss := NewSecureString(original)

	if ss.Masked() != "my-s****-key" {
		t.Errorf("Masked() = %q, want %q", ss.Masked(), "my-s****-key")
	}

	retrieved := ss.Get()
	if retrieved != original {
		t.Errorf("Get() = %q, want %q", retrieved, original)
	}

	emptySS := NewSecureString("")
	if emptySS.Get() != "" {
		t.Errorf("NewSecureString('').Get() = %q, want ''", emptySS.Get())
	}
}
