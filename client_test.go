package skailar

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	t.Setenv(envAPIKey, "")
	_, err := NewClient()
	require.Error(t, err)
	require.ErrorIs(t, err, &Error{Kind: KindConfig})
}

func TestNewClientReadsAPIKeyFromEnv(t *testing.T) {
	t.Setenv(envAPIKey, "skl_live_fromenv")
	client, err := NewClient()
	require.NoError(t, err)
	require.Equal(t, "skl_live_fromenv", client.apiKey)
}

func TestNewClientExplicitKeyOverridesEnv(t *testing.T) {
	t.Setenv(envAPIKey, "skl_live_fromenv")
	client, err := NewClient(WithAPIKey("skl_live_explicit"))
	require.NoError(t, err)
	require.Equal(t, "skl_live_explicit", client.apiKey)
}

func TestNewClientDefaultBaseURL(t *testing.T) {
	t.Setenv(envBaseURL, "")
	client, err := NewClient(WithAPIKey(testKey))
	require.NoError(t, err)
	require.Equal(t, defaultBaseURL, client.baseURL)
}

func TestNewClientBaseURLFromEnv(t *testing.T) {
	t.Setenv(envBaseURL, "http://env.example")
	client, err := NewClient(WithAPIKey(testKey))
	require.NoError(t, err)
	require.Equal(t, "http://env.example", client.baseURL)
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	client, err := NewClient(WithAPIKey(testKey), WithBaseURL("http://h.example/"))
	require.NoError(t, err)
	require.Equal(t, "http://h.example", client.baseURL)
}

func TestNewClientDefaults(t *testing.T) {
	client, err := NewClient(WithAPIKey(testKey))
	require.NoError(t, err)
	require.Equal(t, defaultTimeout, client.timeout)
	require.Equal(t, defaultRetries, client.maxRetries)
	require.NotNil(t, client.httpClient)
}

func TestNewClientOptionsApplied(t *testing.T) {
	hc := &http.Client{}
	client, err := NewClient(
		WithAPIKey(testKey),
		WithTimeout(5*time.Second),
		WithMaxRetries(7),
		WithHTTPClient(hc),
	)
	require.NoError(t, err)
	require.Equal(t, 5*time.Second, client.timeout)
	require.Equal(t, 7, client.maxRetries)
	require.Same(t, hc, client.httpClient)
}

func TestEndpointJoinsWithoutDoubleSlash(t *testing.T) {
	client := newTestClient(t, "http://h.example")
	require.Equal(t, "http://h.example/v1/models", client.endpoint("v1/models"))
	require.Equal(t, "http://h.example/v1/models", client.endpoint("/v1/models"))
}

func TestServicesWired(t *testing.T) {
	client := newTestClient(t, "http://h.example")
	require.NotNil(t, client.Chat)
	require.NotNil(t, client.Chat.Completions)
	require.NotNil(t, client.Models)
	require.NotNil(t, client.Images)
	require.NotNil(t, client.Audio)
	require.NotNil(t, client.Audio.Transcriptions)
	require.NotNil(t, client.Audio.Speech)
	require.NotNil(t, client.Uploads)
	require.NotNil(t, client.Uploads.Images)
	require.NotNil(t, client.Uploads.Files)
}

func TestPingSendsBearerAndUserAgent(t *testing.T) {
	var gotAuth, gotUA, gotAccept string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v1/ping-key", r.URL.Path)
		writeJSON(t, w, PingKeyResponse{Status: "ok", UserID: "u-1"})
	})
	client := newTestClient(t, server.URL)

	pong, err := client.Ping(context.Background())
	require.NoError(t, err)
	require.Equal(t, "ok", pong.Status)
	require.Equal(t, "u-1", pong.UserID)
	require.Equal(t, "Bearer "+testKey, gotAuth)
	require.Equal(t, userAgent, gotUA)
	require.Equal(t, "application/json", gotAccept)
}

func TestDefaultHeaderSent(t *testing.T) {
	var gotTrace string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotTrace = r.Header.Get("X-Trace-Id")
		writeJSON(t, w, PingKeyResponse{Status: "ok", UserID: "u"})
	})
	client := newTestClient(t, server.URL, WithDefaultHeader("X-Trace-Id", "abc123"))

	_, err := client.Ping(context.Background())
	require.NoError(t, err)
	require.Equal(t, "abc123", gotTrace)
}

func TestAuthorizationCannotBeOverriddenByDefaultHeader(t *testing.T) {
	var gotAuth string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSON(t, w, PingKeyResponse{Status: "ok", UserID: "u"})
	})
	// Mixed-case header name must still be dropped before the SDK's bearer.
	client := newTestClient(t, server.URL, WithDefaultHeader("AuThOrIzAtIoN", "Bearer attacker"))

	_, err := client.Ping(context.Background())
	require.NoError(t, err)
	require.Equal(t, "Bearer "+testKey, gotAuth)
}

func TestModelsList(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v1/models", r.URL.Path)
		writeJSON(t, w, map[string]any{
			"object": "list",
			"data": []map[string]any{
				{
					"id": "claude-sonnet-4-6", "object": "model", "created": 1,
					"owned_by": "anthropic", "display_name": "Claude Sonnet 4.6",
					"context_window": 200000, "max_output_tokens": 8192,
					"capabilities": map[string]any{
						"streaming": true, "tool_calls": true, "vision": true, "json_mode": true,
					},
					"pricing": map[string]any{
						"input_per_mtok": 3.0, "output_per_mtok": 15.0, "currency": "USD",
					},
					"status": "active",
				},
			},
		})
	})
	client := newTestClient(t, server.URL)

	list, err := client.Models.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "list", list.Object)
	require.Len(t, list.Data, 1)
	require.Equal(t, "claude-sonnet-4-6", list.Data[0].ID)
	require.True(t, list.Data[0].Capabilities.Vision)
	require.Equal(t, "USD", list.Data[0].Pricing.Currency)
}

func TestModelsRetrieveEscapesSlashes(t *testing.T) {
	var gotPath string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		writeJSON(t, w, map[string]any{
			"id": "google/gemini-2.5-pro", "object": "model", "created": 1,
			"owned_by": "google", "display_name": "Gemini 2.5 Pro",
			"context_window": 1000000, "max_output_tokens": 8192,
			"capabilities": map[string]any{
				"streaming": true, "tool_calls": true, "vision": true, "json_mode": true,
			},
			"pricing": map[string]any{
				"input_per_mtok": 1.0, "output_per_mtok": 2.0, "currency": "USD",
			},
			"status":      "active",
			"aliases":     []string{"gemini-pro"},
			"released_at": "2025-01-01",
		})
	})
	client := newTestClient(t, server.URL)

	model, err := client.Models.Retrieve(context.Background(), "google/gemini-2.5-pro")
	require.NoError(t, err)
	require.Equal(t, "/v1/models/google/gemini-2.5-pro", gotPath)
	require.Equal(t, "google/gemini-2.5-pro", model.ID)
	require.Equal(t, []string{"gemini-pro"}, model.Aliases)
}
