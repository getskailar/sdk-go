package skailar

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBackoffRespectsRetryAfterCap(t *testing.T) {
	require.Equal(t, maxRetryAfter, backoffDelay(0, 120))
	require.Equal(t, 5*time.Second, backoffDelay(0, 5))
}

func TestBackoffStaysWithinExponentialWindow(t *testing.T) {
	for attempt := range 6 {
		window := min(backoffBase<<attempt, backoffCap)
		for range 50 {
			d := backoffDelay(attempt, 0)
			require.LessOrEqual(t, d, window, "attempt %d", attempt)
			require.GreaterOrEqual(t, d, time.Duration(0))
		}
	}
}

func TestShouldRetryRules(t *testing.T) {
	// 429 retried for both idempotency classes.
	require.True(t, shouldRetry(http.StatusTooManyRequests, sideEffect, 0, 3))
	require.True(t, shouldRetry(http.StatusTooManyRequests, idempotent, 0, 3))
	// 5xx retried only for idempotent.
	require.True(t, shouldRetry(http.StatusBadGateway, idempotent, 0, 3))
	require.False(t, shouldRetry(http.StatusInternalServerError, sideEffect, 0, 3))
	// Exhausted attempts never retry.
	require.False(t, shouldRetry(http.StatusTooManyRequests, idempotent, 2, 3))
	// 4xx (other than 429) never retried.
	require.False(t, shouldRetry(http.StatusBadRequest, idempotent, 0, 3))
}

func TestGetRetriesOn5xxThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		writeJSON(t, w, map[string]any{"object": "list", "data": []any{}})
	})
	client := newTestClient(t, server.URL, WithMaxRetries(2))

	_, err := client.Models.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load())
}

func TestPostNotRetriedOn5xx(t *testing.T) {
	var calls atomic.Int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(t, w, errorBody("upstream_error", "boom"))
	})
	client := newTestClient(t, server.URL, WithMaxRetries(3))

	_, err := client.Chat.Completions.Create(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{userMessage("hi")},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUpstream)
	require.Equal(t, int32(1), calls.Load(), "side-effect POST must not retry on 5xx")
}

func TestPostRetriedOn429(t *testing.T) {
	var calls atomic.Int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			writeJSON(t, w, errorBody("rate_limited", "slow"))
			return
		}
		writeJSON(t, w, sampleCompletion("ok"))
	})
	client := newTestClient(t, server.URL, WithMaxRetries(2))

	res, err := client.Chat.Completions.Create(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{userMessage("hi")},
	})
	require.NoError(t, err)
	require.Equal(t, "ok", res.Choices[0].Message.Content.Text())
	require.Equal(t, int32(2), calls.Load())
}

func TestRetriesExhaustReturnLastError(t *testing.T) {
	var calls atomic.Int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		writeJSON(t, w, errorBody("rate_limited", "slow"))
	})
	client := newTestClient(t, server.URL, WithMaxRetries(2))

	_, err := client.Models.List(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRateLimit)
	// maxRetries=2 => 3 total attempts.
	require.Equal(t, int32(3), calls.Load())
}

func TestZeroRetriesMeansSingleAttempt(t *testing.T) {
	var calls atomic.Int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	})
	client := newTestClient(t, server.URL, WithMaxRetries(0))

	_, err := client.Models.List(context.Background())
	require.Error(t, err)
	require.Equal(t, int32(1), calls.Load())
}
