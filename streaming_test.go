package skailar

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// splitSSE runs the SSE split function over the whole input and returns the
// lines, exercising the buffer-boundary logic at EOF.
func splitSSE(input string) []string {
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(scanSSELines)
	var out []string
	for scanner.Scan() {
		out = append(out, scanner.Text())
	}
	return out
}

func TestSSESplitOnLF(t *testing.T) {
	require.Equal(t, []string{"a", "b", "c"}, splitSSE("a\nb\nc\n"))
}

func TestSSESplitOnCRLF(t *testing.T) {
	require.Equal(t, []string{"a", "b", "c"}, splitSSE("a\r\nb\r\nc\r\n"))
}

func TestSSESplitOnCR(t *testing.T) {
	require.Equal(t, []string{"a", "b", "c"}, splitSSE("a\rb\rc\r"))
}

func TestSSESplitFinalUnterminatedLine(t *testing.T) {
	require.Equal(t, []string{"a", "b"}, splitSSE("a\nb"))
}

func TestSSESplitCRLFAcrossBoundary(t *testing.T) {
	// Feed a lone \r then \n in a second write via a chunked reader.
	r := &chunkedReader{chunks: [][]byte{[]byte("a\r"), []byte("\nb\n")}}
	scanner := bufio.NewScanner(r)
	scanner.Split(scanSSELines)
	var out []string
	for scanner.Scan() {
		out = append(out, scanner.Text())
	}
	require.NoError(t, scanner.Err())
	require.Equal(t, []string{"a", "b"}, out)
}

func TestDataPayloadStripsPrefixAndSpace(t *testing.T) {
	p, ok := dataPayload([]byte("data: hello"))
	require.True(t, ok)
	require.Equal(t, "hello", string(p))

	p, ok = dataPayload([]byte("data:hello"))
	require.True(t, ok)
	require.Equal(t, "hello", string(p))
}

func TestDataPayloadIgnoresCommentsAndBlanks(t *testing.T) {
	_, ok := dataPayload([]byte(": keep-alive"))
	require.False(t, ok)
	_, ok = dataPayload([]byte(""))
	require.False(t, ok)
	_, ok = dataPayload([]byte("event: message"))
	require.False(t, ok)
}

func TestDecodeChunkParsesDelta(t *testing.T) {
	payload := []byte(`{"id":"1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":"hi"}}]}`)
	chunk, err := decodeChunk(payload)
	require.NoError(t, err)
	text, ok := chunk.ContentDelta()
	require.True(t, ok)
	require.Equal(t, "hi", text)
}

func TestDecodeChunkMalformedIsDecodeError(t *testing.T) {
	_, err := decodeChunk([]byte("{not json"))
	require.Error(t, err)
	var e *Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, KindDecode, e.Kind)
}

func TestDecodeChunkInBandErrorBecomesAPIError(t *testing.T) {
	_, err := decodeChunk([]byte(`{"error":{"type":"upstream_error","message":"boom"}}`))
	require.Error(t, err)
	var e *Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, KindUpstream, e.Kind)
	require.Equal(t, "upstream_error", e.Code)
	require.Equal(t, "boom", e.Message)
}

func TestStreamRoundtrip(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "text/event-stream", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + chunkJSON("He") + "\n\n"))
		_, _ = w.Write([]byte(": keep-alive\n\n"))
		_, _ = w.Write([]byte("data: " + chunkJSON("llo") + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	})
	client := newTestClient(t, server.URL)

	stream, err := client.Chat.Completions.CreateStream(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{userMessage("hi")},
	})
	require.NoError(t, err)
	defer stream.Close()

	var got strings.Builder
	for stream.Next() {
		if text, ok := stream.Current().ContentDelta(); ok {
			got.WriteString(text)
		}
	}
	require.NoError(t, stream.Err())
	require.Equal(t, "Hello", got.String())
}

func TestStreamForcesStreamFlag(t *testing.T) {
	var body map[string]any
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, decodeBody(r, &body))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	})
	client := newTestClient(t, server.URL)

	stream, err := client.Chat.Completions.CreateStream(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{userMessage("hi")},
		Stream:   false,
	})
	require.NoError(t, err)
	defer stream.Close()
	for stream.Next() {
	}
	require.NoError(t, stream.Err())
	require.Equal(t, true, body["stream"])
}

func TestStreamInBandErrorSurfaces(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + chunkJSON("partial") + "\n\n"))
		_, _ = w.Write([]byte(`data: {"error":{"type":"upstream_error","message":"mid-stream boom"}}` + "\n\n"))
	})
	client := newTestClient(t, server.URL)

	stream, err := client.Chat.Completions.CreateStream(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{userMessage("hi")},
	})
	require.NoError(t, err)
	defer stream.Close()

	count := 0
	for stream.Next() {
		count++
	}
	require.Equal(t, 1, count)
	require.Error(t, stream.Err())
	require.ErrorIs(t, stream.Err(), ErrUpstream)
}

func TestStreamCloseIsIdempotent(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + chunkJSON("x") + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	})
	client := newTestClient(t, server.URL)

	stream, err := client.Chat.Completions.CreateStream(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{userMessage("hi")},
	})
	require.NoError(t, err)
	require.True(t, stream.Next())
	require.NoError(t, stream.Close())
	require.NoError(t, stream.Close())
	require.False(t, stream.Next())
}

func TestStreamErrorBeforeStreamStarts(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(t, w, errorBody("invalid_api_key", "bad key"))
	})
	client := newTestClient(t, server.URL)

	_, err := client.Chat.Completions.CreateStream(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{userMessage("hi")},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAuth)
}

// chunkJSON builds a minimal chunk payload carrying a content delta.
func chunkJSON(content string) string {
	return `{"id":"1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":"` + content + `"}}]}`
}

// decodeBody decodes a JSON request body in a test handler.
func decodeBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// chunkedReader yields its chunks across successive Read calls, to simulate
// network packet boundaries.
type chunkedReader struct {
	chunks [][]byte
	idx    int
}

func (c *chunkedReader) Read(p []byte) (int, error) {
	if c.idx >= len(c.chunks) {
		return 0, io.EOF
	}
	n := copy(p, c.chunks[c.idx])
	if n < len(c.chunks[c.idx]) {
		c.chunks[c.idx] = c.chunks[c.idx][n:]
		return n, nil
	}
	c.idx++
	return n, nil
}
