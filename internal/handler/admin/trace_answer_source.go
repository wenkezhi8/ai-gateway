package admin

import (
	"fmt"
	"strings"
)

const (
	traceAnswerSourceV2           = "v2"
	traceAnswerSourceSemantic     = "semantic"
	traceAnswerSourceExactRaw     = "exact_raw"
	traceAnswerSourceExactPrompt  = "exact_prompt"
	traceAnswerSourceProviderChat = "provider_chat"
)

func canonicalTraceAnswerSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case traceAnswerSourceV2, "cache_v2", "vector-exact", "vector-semantic":
		return traceAnswerSourceV2
	case traceAnswerSourceSemantic, "cache_semantic":
		return traceAnswerSourceSemantic
	case traceAnswerSourceExactRaw:
		return traceAnswerSourceExactRaw
	case traceAnswerSourceExactPrompt, "cache_exact", "exact":
		return traceAnswerSourceExactPrompt
	case traceAnswerSourceProviderChat:
		return traceAnswerSourceProviderChat
	default:
		return ""
	}
}

func normalizedTraceAnswerSourceSQL(expr string) string {
	return fmt.Sprintf(`
		CASE LOWER(TRIM(COALESCE(%s, '')))
			WHEN 'v2' THEN 'v2'
			WHEN 'cache_v2' THEN 'v2'
			WHEN 'vector-exact' THEN 'v2'
			WHEN 'vector-semantic' THEN 'v2'
			WHEN 'semantic' THEN 'semantic'
			WHEN 'cache_semantic' THEN 'semantic'
			WHEN 'exact_raw' THEN 'exact_raw'
			WHEN 'exact_prompt' THEN 'exact_prompt'
			WHEN 'cache_exact' THEN 'exact_prompt'
			WHEN 'exact' THEN 'exact_prompt'
			WHEN 'provider_chat' THEN 'provider_chat'
			ELSE ''
		END
	`, expr)
}

func requestAggAnswerSourceSQL() string {
	responseLayer := normalizedTraceAnswerSourceSQL("json_extract(rt.attributes, '$.cache_layer')")
	exactLayer := normalizedTraceAnswerSourceSQL("json_extract(rt.attributes, '$.cache_layer')")
	v2Layer := normalizedTraceAnswerSourceSQL("json_extract(rt.attributes, '$.layer')")

	return fmt.Sprintf(`
		CASE
			WHEN MAX(CASE WHEN rt.operation = 'http.response' AND json_valid(rt.attributes) = 1 AND %s = 'exact_raw' THEN 1 ELSE 0 END) = 1 THEN 'exact_raw'
			WHEN MAX(CASE WHEN rt.operation = 'http.response' AND json_valid(rt.attributes) = 1 AND %s = 'exact_prompt' THEN 1 ELSE 0 END) = 1 THEN 'exact_prompt'
			WHEN MAX(CASE WHEN rt.operation = 'http.response' AND json_valid(rt.attributes) = 1 AND %s = 'semantic' THEN 1 ELSE 0 END) = 1 THEN 'semantic'
			WHEN MAX(CASE WHEN rt.operation = 'http.response' AND json_valid(rt.attributes) = 1 AND %s = 'v2' THEN 1 ELSE 0 END) = 1 THEN 'v2'
			WHEN MAX(CASE WHEN rt.operation = 'http.response' AND json_valid(rt.attributes) = 1 AND %s = 'provider_chat' THEN 1 ELSE 0 END) = 1 THEN 'provider_chat'
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-exact' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' AND %s = 'exact_raw' THEN 1 ELSE 0 END) = 1 THEN 'exact_raw'
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-exact' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' AND %s = 'exact_prompt' THEN 1 ELSE 0 END) = 1 THEN 'exact_prompt'
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-semantic' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN 'semantic'
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-v2' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' AND %s = 'v2' THEN 1 ELSE 0 END) = 1 THEN 'v2'
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-v2' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN 'v2'
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-exact' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN 'exact_prompt'
			WHEN MAX(CASE WHEN rt.operation = 'provider.chat' AND rt.status = 'success' THEN 1 ELSE 0 END) = 1 THEN 'provider_chat'
			ELSE 'provider_chat'
		END AS answer_source
	`, responseLayer, responseLayer, responseLayer, responseLayer, responseLayer, exactLayer, exactLayer, v2Layer)
}
