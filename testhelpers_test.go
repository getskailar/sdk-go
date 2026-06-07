package skailar

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const testKey = "skl_live_testtesttesttesttesttesttesttesttest"

// newMockServer starts an httptest server with the given handler and registers
// its cleanup with the test.
func newMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// newTestClient builds a client pointed at baseURL with the test key and any
// extra options applied after the defaults.
func newTestClient(t *testing.T, baseURL string, opts ...Option) *Client {
	t.Helper()
	all := append([]Option{WithAPIKey(testKey), WithBaseURL(baseURL)}, opts...)
	client, err := NewClient(all...)
	require.NoError(t, err)
	return client
}

// writeJSON serializes v as a JSON response body.
func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(v))
}

// sampleCompletion returns a minimal chat-completion payload whose assistant
// message carries the given text.
func sampleCompletion(text string) map[string]any {
	return map[string]any{
		"id":      "cmpl_test",
		"object":  "chat.completion",
		"created": 1,
		"model":   "m",
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": text,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     1,
			"completion_tokens": 2,
			"total_tokens":      3,
		},
	}
}

// errorBody returns an OpenAI-style error envelope.
func errorBody(code, message string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"type":    code,
			"message": message,
		},
	}
}

// userMessage is a convenience for a single-text user message.
func userMessage(text string) ChatMessage {
	return ChatMessage{Role: RoleUser, Content: TextContent(text)}
}
