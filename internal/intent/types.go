package intent

// IntentEmbeddingRequest is the payload sent to local intent-engine.
type IntentEmbeddingRequest struct {
	Query   string `json:"query"`
	Context string `json:"context,omitempty"`
	Lang    string `json:"lang,omitempty"`
}

// IntentEmbeddingResult is the inference output from intent-engine.
type IntentEmbeddingResult struct {
	Intent         string            `json:"intent"`
	Slots          map[string]string `json:"slots"`
	StandardKey    string            `json:"standard_key"`
	Embedding      []float64         `json:"embedding"`
	EmbeddingDim   int               `json:"embedding_dim"`
	Confidence     float64           `json:"confidence"`
	EngineVersion  string            `json:"engine_version"`
	NormalizedText string            `json:"normalized_text,omitempty"`
}

