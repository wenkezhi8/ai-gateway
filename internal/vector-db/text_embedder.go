package vectordb

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
)

type TextEmbedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type deterministicTextEmbedder struct{}

func newDeterministicTextEmbedder() TextEmbedder {
	return &deterministicTextEmbedder{}
}

func (e *deterministicTextEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return nil, fmt.Errorf("text is required")
	}
	hash := sha256.Sum256([]byte(normalized))
	vector := make([]float32, 8)
	for i := 0; i < 8; i++ {
		chunk := binary.BigEndian.Uint32(hash[i*4 : (i+1)*4])
		vector[i] = float32(chunk%10000) / 10000
	}
	return vector, nil
}
