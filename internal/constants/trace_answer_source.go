package constants

const (
	TraceAnswerSourceV2           = "v2"
	TraceAnswerSourceSemantic     = "semantic"
	TraceAnswerSourceExactRaw     = "exact_raw"
	TraceAnswerSourceExactPrompt  = "exact_prompt"
	TraceAnswerSourceProviderChat = "provider_chat"
)

var TraceAnswerSources = []string{
	TraceAnswerSourceExactRaw,
	TraceAnswerSourceExactPrompt,
	TraceAnswerSourceSemantic,
	TraceAnswerSourceV2,
	TraceAnswerSourceProviderChat,
}
