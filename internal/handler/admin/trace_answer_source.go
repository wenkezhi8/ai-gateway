package admin

import (
	"fmt"
	"strings"

	"ai-gateway/internal/constants"
)

const (
	traceAnswerSourceV2           = constants.TraceAnswerSourceV2
	traceAnswerSourceSemantic     = constants.TraceAnswerSourceSemantic
	traceAnswerSourceExactRaw     = constants.TraceAnswerSourceExactRaw
	traceAnswerSourceExactPrompt  = constants.TraceAnswerSourceExactPrompt
	traceAnswerSourceProviderChat = constants.TraceAnswerSourceProviderChat
)

func quotedSQLString(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

func traceAnswerSourceAllowedValues() string {
	return "all|" + strings.Join(constants.TraceAnswerSources, "|")
}

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
			WHEN 'v2' THEN %s
			WHEN 'cache_v2' THEN %s
			WHEN 'vector-exact' THEN %s
			WHEN 'vector-semantic' THEN %s
			WHEN 'semantic' THEN %s
			WHEN 'cache_semantic' THEN %s
			WHEN 'exact_raw' THEN %s
			WHEN 'exact_prompt' THEN %s
			WHEN 'cache_exact' THEN %s
			WHEN 'exact' THEN %s
			WHEN 'provider_chat' THEN %s
			ELSE ''
		END
	`, expr,
		quotedSQLString(traceAnswerSourceV2),
		quotedSQLString(traceAnswerSourceV2),
		quotedSQLString(traceAnswerSourceV2),
		quotedSQLString(traceAnswerSourceV2),
		quotedSQLString(traceAnswerSourceSemantic),
		quotedSQLString(traceAnswerSourceSemantic),
		quotedSQLString(traceAnswerSourceExactRaw),
		quotedSQLString(traceAnswerSourceExactPrompt),
		quotedSQLString(traceAnswerSourceExactPrompt),
		quotedSQLString(traceAnswerSourceExactPrompt),
		quotedSQLString(traceAnswerSourceProviderChat),
	)
}

func requestAggAnswerSourceSQL() string {
	responseLayer := normalizedTraceAnswerSourceSQL("json_extract(rt.attributes, '$.cache_layer')")
	exactLayer := normalizedTraceAnswerSourceSQL("json_extract(rt.attributes, '$.cache_layer')")
	v2Layer := normalizedTraceAnswerSourceSQL("json_extract(rt.attributes, '$.layer')")

	return fmt.Sprintf(`
		CASE
			WHEN MAX(CASE WHEN rt.operation = 'http.response' AND json_valid(rt.attributes) = 1 AND %s = %s THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'http.response' AND json_valid(rt.attributes) = 1 AND %s = %s THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'http.response' AND json_valid(rt.attributes) = 1 AND %s = %s THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'http.response' AND json_valid(rt.attributes) = 1 AND %s = %s THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'http.response' AND json_valid(rt.attributes) = 1 AND %s = %s THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-exact' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' AND %s = %s THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-exact' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' AND %s = %s THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-semantic' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-v2' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' AND %s = %s THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-v2' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'cache.read-exact' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN %s
			WHEN MAX(CASE WHEN rt.operation = 'provider.chat' AND rt.status = 'success' THEN 1 ELSE 0 END) = 1 THEN %s
			ELSE %s
		END AS answer_source
	`,
		responseLayer, quotedSQLString(traceAnswerSourceExactRaw), quotedSQLString(traceAnswerSourceExactRaw),
		responseLayer, quotedSQLString(traceAnswerSourceExactPrompt), quotedSQLString(traceAnswerSourceExactPrompt),
		responseLayer, quotedSQLString(traceAnswerSourceSemantic), quotedSQLString(traceAnswerSourceSemantic),
		responseLayer, quotedSQLString(traceAnswerSourceV2), quotedSQLString(traceAnswerSourceV2),
		responseLayer, quotedSQLString(traceAnswerSourceProviderChat), quotedSQLString(traceAnswerSourceProviderChat),
		exactLayer, quotedSQLString(traceAnswerSourceExactRaw), quotedSQLString(traceAnswerSourceExactRaw),
		exactLayer, quotedSQLString(traceAnswerSourceExactPrompt), quotedSQLString(traceAnswerSourceExactPrompt),
		quotedSQLString(traceAnswerSourceSemantic),
		v2Layer, quotedSQLString(traceAnswerSourceV2), quotedSQLString(traceAnswerSourceV2),
		quotedSQLString(traceAnswerSourceV2),
		quotedSQLString(traceAnswerSourceExactPrompt),
		quotedSQLString(traceAnswerSourceProviderChat),
		quotedSQLString(traceAnswerSourceProviderChat),
	)
}
