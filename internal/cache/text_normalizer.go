package cache

import (
	"strings"
	"unicode"
)

// TextNormalizer provides deterministic query normalization for cache keys.
type TextNormalizer struct {
	fillerWords []string
}

// NewTextNormalizer creates a text normalizer with production-safe defaults.
func NewTextNormalizer() *TextNormalizer {
	return &TextNormalizer{
		fillerWords: []string{
			"请问", "帮我", "请", "能不能", "啊", "吧", "呀", "呢",
		},
	}
}

// Normalize converts noisy user input into a stable normalized query string.
func (n *TextNormalizer) Normalize(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}

	text := toHalfWidth(strings.TrimSpace(input))
	text = normalizeCommonSymbols(text)
	text = normalizeChineseDigits(text)

	for _, word := range n.fillerWords {
		text = strings.ReplaceAll(text, word, "")
	}

	text = deduplicatePunctuation(text)
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

func normalizeCommonSymbols(s string) string {
	replacer := strings.NewReplacer(
		"。", ".",
		"，", ",",
		"？", "?",
		"！", "!",
		"；", ";",
		"：", ":",
		"（", "(",
		"）", ")",
		"【", "[",
		"】", "]",
		"＋", "+",
		"－", "-",
		"＊", "*",
		"／", "/",
	)
	return replacer.Replace(s)
}

func normalizeChineseDigits(s string) string {
	digitMap := map[rune]rune{
		'零': '0',
		'一': '1',
		'二': '2',
		'三': '3',
		'四': '4',
		'五': '5',
		'六': '6',
		'七': '7',
		'八': '8',
		'九': '9',
	}
	runes := []rune(s)
	for i, r := range runes {
		mapped, ok := digitMap[r]
		if !ok {
			continue
		}
		prevBoundary := i == 0 || isDigitBoundary(runes[i-1]) || isChineseDigit(runes[i-1])
		nextBoundary := i == len(runes)-1 || isDigitBoundary(runes[i+1]) || isChineseDigit(runes[i+1])
		if prevBoundary || nextBoundary {
			runes[i] = mapped
		}
	}
	return string(runes)
}

func deduplicatePunctuation(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	var prev rune
	for _, r := range s {
		if isDedupPunctuation(r) && prev == r {
			continue
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}

func isDedupPunctuation(r rune) bool {
	switch r {
	case '.', ',', '?', '!', ';', ':':
		return true
	default:
		return false
	}
}

func toHalfWidth(s string) string {
	var out []rune
	out = make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r == 0x3000:
			out = append(out, ' ')
		case r >= 0xFF01 && r <= 0xFF5E:
			out = append(out, r-0xFEE0)
		case unicode.IsSpace(r):
			out = append(out, ' ')
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

func isDigitBoundary(r rune) bool {
	if unicode.IsSpace(r) || unicode.IsPunct(r) {
		return true
	}
	switch r {
	case '+', '-', '*', '/', 'x', 'X', '×', '÷', '=', '>', '<':
		return true
	default:
		return false
	}
}

func isChineseDigit(r rune) bool {
	switch r {
	case '零', '一', '二', '三', '四', '五', '六', '七', '八', '九':
		return true
	default:
		return false
	}
}
