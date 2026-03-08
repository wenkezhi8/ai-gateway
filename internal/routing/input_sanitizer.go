package routing

import (
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	metadataBracketTokenPattern = regexp.MustCompile(`^\[([^\]]+)\]`)
	senderMetadataBlockPattern  = regexp.MustCompile("(?is)^sender\\s*\\([^)]*metadata[^)]*\\):\\s*```[a-z0-9_-]*\\s*.*?```\\s*")
	metadataKeyValuePattern     = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*\s*=\s*\S+$`)
	humanTimestampPattern       = regexp.MustCompile(`(?i)^(mon|tue|wed|thu|fri|sat|sun)\s+\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}(:\d{2})?\s+(gmt|utc)[+-]\d{1,2}(:\d{2})?$`)
	metadataKeyWhitelist        = map[string]struct{}{
		"request_id":      {},
		"message_id":      {},
		"conversation_id": {},
		"session_id":      {},
		"trace_id":        {},
		"parent_id":       {},
		"user_id":         {},
		"account_id":      {},
		"tenant_id":       {},
		"ts":              {},
		"time":            {},
		"timestamp":       {},
	}
	shortGreetingSet = map[string]struct{}{
		"hi":    {},
		"hello": {},
		"hey":   {},
		"yo":    {},
		"你好":    {},
		"您好":    {},
		"嗨":     {},
		"哈喽":    {},
		"早上好":   {},
		"下午好":   {},
		"晚上好":   {},
		"在吗":    {},
		"在嗎":    {},
	}
)

func SanitizeIntentInput(raw string) string {
	remaining := strings.TrimSpace(raw)
	if remaining == "" {
		return ""
	}

	for {
		strippedSender := strings.TrimSpace(senderMetadataBlockPattern.ReplaceAllString(remaining, ""))
		if strippedSender != remaining {
			remaining = strippedSender
			continue
		}

		remaining = strings.TrimLeftFunc(remaining, unicode.IsSpace)
		match := metadataBracketTokenPattern.FindStringSubmatchIndex(remaining)
		if match == nil || match[0] != 0 {
			break
		}

		token := strings.TrimSpace(remaining[match[2]:match[3]])
		if !isMetadataBracketToken(token) {
			break
		}

		remaining = strings.TrimSpace(remaining[match[1]:])
	}

	return strings.TrimSpace(remaining)
}

func IsShortGreetingIntent(raw string) bool {
	sanitized := SanitizeIntentInput(raw)
	if sanitized == "" {
		return false
	}

	normalized := normalizeGreetingCandidate(sanitized)
	if normalized == "" {
		return false
	}

	if _, ok := shortGreetingSet[normalized]; ok {
		return true
	}

	runeLen := utf8.RuneCountInString(normalized)
	if runeLen <= 4 && strings.HasPrefix(normalized, "你好") {
		return true
	}
	if runeLen <= 4 && strings.HasPrefix(normalized, "您好") {
		return true
	}

	return false
}

func normalizeGreetingCandidate(input string) string {
	raw := strings.ToLower(strings.TrimSpace(input))
	if raw == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		b.WriteRune(r)
	}

	return strings.TrimSpace(b.String())
}

func isMetadataBracketToken(token string) bool {
	if token == "" {
		return false
	}

	if _, err := time.Parse(time.RFC3339, token); err == nil {
		return true
	}

	if humanTimestampPattern.MatchString(token) {
		return true
	}

	if !metadataKeyValuePattern.MatchString(token) {
		return false
	}

	parts := strings.SplitN(token, "=", 2)
	if len(parts) != 2 {
		return false
	}

	key := strings.ToLower(strings.TrimSpace(parts[0]))
	if key == "" {
		return false
	}
	if _, ok := metadataKeyWhitelist[key]; ok {
		return true
	}

	if strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "-id") {
		return true
	}
	if strings.HasSuffix(key, "_ts") || strings.HasSuffix(key, "-ts") || strings.HasSuffix(key, "timestamp") {
		return true
	}

	return false
}
