package cache

import (
	"fmt"
	"sort"
	"strings"
)

// BuildStandardKey generates a stable key in form:
// intent:<intent>:k1=v1,k2=v2
func BuildStandardKey(intent string, slots map[string]string) string {
	normalizedIntent := strings.TrimSpace(intent)
	if normalizedIntent == "" {
		normalizedIntent = "unknown"
	}

	if len(slots) == 0 {
		return fmt.Sprintf("intent:%s", normalizedIntent)
	}

	keys := make([]string, 0, len(slots))
	for k, v := range slots {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" || val == "" {
			continue
		}
		keys = append(keys, key)
	}

	if len(keys) == 0 {
		return fmt.Sprintf("intent:%s", normalizedIntent)
	}

	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.TrimSpace(slots[key])))
	}

	return fmt.Sprintf("intent:%s:%s", normalizedIntent, strings.Join(parts, ","))
}

