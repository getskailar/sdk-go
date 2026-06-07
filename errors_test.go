package skailar

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseErrorFieldsNestedType(t *testing.T) {
	code, msg := parseErrorFields([]byte(`{"error":{"type":"invalid_api_key","message":"bad key"}}`))
	require.Equal(t, "invalid_api_key", code)
	require.Equal(t, "bad key", msg)
}

func TestParseErrorFieldsOpenAICode(t *testing.T) {
	code, msg := parseErrorFields([]byte(`{"error":{"code":"rate_limited","message":"slow down"}}`))
	require.Equal(t, "rate_limited", code)
	require.Equal(t, "slow down", msg)
}

func TestParseErrorFieldsFlatString(t *testing.T) {
	code, msg := parseErrorFields([]byte(`{"error":"bad_request","message":"nope"}`))
	require.Equal(t, "bad_request", code)
	require.Equal(t, "nope", msg)
}

func TestParseErrorFieldsTopLevelMessage(t *testing.T) {
	code, msg := parseErrorFields([]byte(`{"message":"plain"}`))
	require.Equal(t, "", code)
	require.Equal(t, "plain", msg)
}

func TestApiErrorFallsBackToStatusWhenEmpty(t *testing.T) {
	e := apiError(500, "", 0, []byte(""))
	require.Equal(t, "HTTP 500", e.Message)
	require.Equal(t, KindUpstream, e.Kind)
}

func TestKindForStatus(t *testing.T) {
	require.Equal(t, KindAuth, kindForStatus(401))
	require.Equal(t, KindBadRequest, kindForStatus(400))
	require.Equal(t, KindNotFound, kindForStatus(404))
	require.Equal(t, KindRateLimit, kindForStatus(429))
	require.Equal(t, KindUpstream, kindForStatus(503))
	require.Equal(t, KindAPI, kindForStatus(418))
}

func TestErrorIsMatchesSentinelByKind(t *testing.T) {
	err := error(&Error{Kind: KindRateLimit, Status: 429})
	require.ErrorIs(t, err, ErrRateLimit)
	require.NotErrorIs(t, err, ErrAuth)
}

func TestErrorAsRecoversFields(t *testing.T) {
	err := error(apiError(429, "req-7", 12, []byte(`{"error":{"type":"rate_limited","message":"slow"}}`)))
	var e *Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, 429, e.Status)
	require.Equal(t, "req-7", e.RequestID)
	require.Equal(t, 12, e.RetryAfter)
	require.Equal(t, "rate_limited", e.Code)
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("root")
	e := &Error{Kind: KindNetwork, cause: cause}
	require.ErrorIs(t, e, cause)
}

func TestAPIErrorEndToEnd(t *testing.T) {
	cases := []struct {
		status   int
		sentinel error
		kind     Kind
	}{
		{400, ErrBadRequest, KindBadRequest},
		{401, ErrAuth, KindAuth},
		{404, ErrNotFound, KindNotFound},
		{429, ErrRateLimit, KindRateLimit},
		{503, ErrUpstream, KindUpstream},
	}
	for _, tc := range cases {
		t.Run(tc.kind.String(), func(t *testing.T) {
			server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.status == 429 {
					w.Header().Set("Retry-After", "5")
				}
				w.Header().Set("X-Request-Id", "rid-1")
				w.WriteHeader(tc.status)
				writeJSON(t, w, errorBody("some_code", "some message"))
			})
			// No retries so the terminal error surfaces immediately.
			client := newTestClient(t, server.URL, WithMaxRetries(0))

			_, err := client.Models.List(context.Background())
			require.Error(t, err)
			require.ErrorIs(t, err, tc.sentinel)

			var e *Error
			require.ErrorAs(t, err, &e)
			require.Equal(t, tc.status, e.Status)
			require.Equal(t, "rid-1", e.RequestID)
			require.Equal(t, "some message", e.Message)
			if tc.status == 429 {
				require.Equal(t, 5, e.RetryAfter)
			}
		})
	}
}

func TestRequestIDPriority(t *testing.T) {
	h := http.Header{}
	h.Set("Request-Id", "c")
	h.Set("X-Request-Id", "a")
	require.Equal(t, "a", extractRequestID(h))
}

func TestDecodeErrorOnMalformedSuccessBody(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{not json"))
	})
	client := newTestClient(t, server.URL)

	_, err := client.Models.List(context.Background())
	require.Error(t, err)
	var e *Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, KindDecode, e.Kind)
}

func TestErrorMessageFormat(t *testing.T) {
	withStatus := &Error{Kind: KindAuth, Status: 401, Message: "bad key"}
	require.Contains(t, withStatus.Error(), "status 401")
	require.Contains(t, withStatus.Error(), "bad key")

	noStatus := &Error{Kind: KindNetwork, Message: "boom"}
	require.Contains(t, noStatus.Error(), "network")
	require.Contains(t, noStatus.Error(), "boom")
}

func TestConfigErrorIsKindConfig(t *testing.T) {
	t.Setenv(envAPIKey, "")
	_, err := NewClient()
	require.ErrorIs(t, err, &Error{Kind: KindConfig})
}
