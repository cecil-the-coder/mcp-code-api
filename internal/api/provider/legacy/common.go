package legacy

import "github.com/cecil-the-coder/mcp-code-api/internal/api/provider"

// MockStream implements ChatCompletionStream for testing
type MockStream struct {
	chunks []provider.ChatCompletionChunk
	index  int
}

func NewMockStream(chunks []provider.ChatCompletionChunk) provider.ChatCompletionStream {
	return &MockStream{chunks: chunks, index: 0}
}

func (ms *MockStream) Next() (provider.ChatCompletionChunk, error) {
	if ms.index >= len(ms.chunks) {
		return provider.ChatCompletionChunk{}, nil
	}
	chunk := ms.chunks[ms.index]
	ms.index++
	return chunk, nil
}

func (ms *MockStream) Close() error {
	ms.index = 0
	return nil
}
