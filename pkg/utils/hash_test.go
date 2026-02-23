package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRandomString(t *testing.T) {
	str1, err := GenerateRandomString(16)
	require.NoError(t, err)
	assert.Len(t, str1, 32) // hex encoding doubles length

	str2, err := GenerateRandomString(16)
	require.NoError(t, err)
	assert.NotEqual(t, str1, str2)
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	assert.True(t, Contains(slice, "a"))
	assert.True(t, Contains(slice, "b"))
	assert.True(t, Contains(slice, "c"))
	assert.False(t, Contains(slice, "d"))
	assert.False(t, Contains([]string{}, "a"))
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "all same",
			input:    []string{"x", "x", "x"},
			expected: []string{"x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Unique(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
