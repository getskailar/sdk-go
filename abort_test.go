package skailar

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContextCancelledBeforeRequest(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"object": "list", "data": []any{}})
	})
	client := newTestClient(t, server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Models.List(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAborted)
}

func TestContextCancelledDuringBackoff(t *testing.T) {
	// Server always 429s with a long Retry-After so the client would sleep; the
	// cancelled context must interrupt the wait promptly.
	var calls atomic.Int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		writeJSON(t, w, errorBody("rate_limited", "slow"))
	})
	client := newTestClient(t, server.URL, WithMaxRetries(5))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := client.Models.List(ctx)
	elapsed := time.Since(start)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrAborted)
	require.Less(t, elapsed, 5*time.Second, "cancellation must interrupt the backoff sleep")
	// One request made, then it began the (interrupted) backoff.
	require.Equal(t, int32(1), calls.Load())
}

func TestSleepBackoffReturnsAbortedOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// retryAfter forces a non-zero delay so the select hits ctx.Done.
	err := sleepBackoff(ctx, 0, 30)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAborted)
}

func TestDeadlineExceededIsTimeout(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		writeJSON(t, w, map[string]any{"object": "list", "data": []any{}})
	})
	client := newTestClient(t, server.URL, WithMaxRetries(0))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := client.Models.List(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTimeout)

	var e *Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, KindTimeout, e.Kind)
}

func TestTransportErrorIsNetwork(t *testing.T) {
	// Point at a closed port so the dial fails with a connection error.
	client := newTestClient(t, "http://127.0.0.1:1", WithMaxRetries(0))

	_, err := client.Models.List(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNetwork)
}

func TestWithTimeoutEnforcesDeadline(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		writeJSON(t, w, map[string]any{"object": "list", "data": []any{}})
	})
	// No context deadline: the configured per-request timeout must fire.
	client := newTestClient(t, server.URL, WithMaxRetries(0), WithTimeout(20*time.Millisecond))

	_, err := client.Models.List(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTimeout)
}
