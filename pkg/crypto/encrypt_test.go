package crypto

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	testCases := []string{
		"sk-1234567890abcdef",
		"simple-api-key",
		"",
		"very-long-api-key-with-special-chars-!@#$%^&*()",
	}

	for _, tc := range testCases {
		encrypted, err := Encrypt(tc)
		if err != nil {
			t.Errorf("Encrypt(%q) error: %v", tc, err)
			continue
		}

		if tc == "" {
			if encrypted != "" {
				t.Errorf("Encrypt('') = %q, want ''", encrypted)
			}
			continue
		}

		decrypted, err := Decrypt(encrypted)
		if err != nil {
			t.Errorf("Decrypt error: %v", err)
			continue
		}

		if decrypted != tc {
			t.Errorf("Decrypt(Encrypt(%q)) = %q, want %q", tc, decrypted, tc)
		}
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"sk-1234567890abcdef", "sk-1****cdef"},
		{"short", "****"},
		{"ab", "****"},
		{"", "****"},
		{"sk-proj-12345678901234567890", "sk-p****7890"},
	}

	for _, tc := range tests {
		result := MaskAPIKey(tc.key)
		if result != tc.expected {
			t.Errorf("MaskAPIKey(%q) = %q, want %q", tc.key, result, tc.expected)
		}
	}
}

func TestIsEncrypted(t *testing.T) {
	plaintext := "my-secret-key"
	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt(%q) error: %v", plaintext, err)
	}

	if !IsEncrypted(encrypted) {
		t.Errorf("IsEncrypted(encrypted) = false, want true")
	}

	if IsEncrypted(plaintext) {
		t.Errorf("IsEncrypted(plaintext) = true, want false")
	}

	if IsEncrypted("") {
		t.Errorf("IsEncrypted('') = true, want false")
	}
}
