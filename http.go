package skailar

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// idempotency records whether a request may be safely replayed after a 5xx or
// transport failure.
type idempotency int

const (
	// idempotent requests (GET) may be retried on 5xx and transport errors.
	idempotent idempotency = iota
	// sideEffect requests (billable POSTs) are only retried on 429.
	sideEffect
)

// maxResponseBytes caps how much of a response body the SDK buffers, guarding
// against a runaway body on the error and JSON paths.
const maxResponseBytes = 32 << 20 // 32 MiB

// getJSON performs a GET and decodes the JSON response into out.
func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	body, err := c.doBuffered(ctx, http.MethodGet, path, nil, nil, idempotent)
	if err != nil {
		return err
	}
	return decodeInto(body, out)
}

// postJSON performs a POST with a JSON body and decodes the JSON response into
// out. POSTs are side-effecting and are not retried on 5xx.
func (c *Client) postJSON(ctx context.Context, path string, body, out any) error {
	encoded, err := encodeJSON(body)
	if err != nil {
		return err
	}
	respBody, err := c.doBuffered(ctx, http.MethodPost, path, encoded, nil, sideEffect)
	if err != nil {
		return err
	}
	return decodeInto(respBody, out)
}

// postBinary performs a POST with a JSON body and returns the raw response body
// for a non-JSON content type (such as audio/mpeg). The caller owns closing the
// returned [io.ReadCloser]. The configured timeout governs establishing the
// response, not consuming its body.
func (c *Client) postBinary(ctx context.Context, path string, body any, accept string) (io.ReadCloser, error) {
	encoded, err := encodeJSON(body)
	if err != nil {
		return nil, err
	}
	header := http.Header{"Accept": []string{accept}}
	// Body ownership transfers to the caller of postBinary.
	resp, err := c.doStreaming(ctx, http.MethodPost, path, encoded, header, sideEffect) //nolint:bodyclose
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// postStream performs a POST that returns a Server-Sent Events stream. The
// configured timeout governs establishing the response, not consuming the
// stream; the caller's context controls the stream's lifetime.
func (c *Client) postStream(ctx context.Context, path string, body any) (*ChatCompletionStream, error) {
	encoded, err := encodeJSON(body)
	if err != nil {
		return nil, err
	}
	header := http.Header{"Accept": []string{"text/event-stream"}}
	// Body ownership transfers to ChatCompletionStream, which closes it via
	// Close(); bodyclose can't follow it across the assignment.
	resp, err := c.doStreaming(ctx, http.MethodPost, path, encoded, header, sideEffect) //nolint:bodyclose
	if err != nil {
		return nil, err
	}
	return newChatCompletionStream(resp.Body), nil
}

// doBuffered executes a request with retries and returns the full success body.
// Each attempt, including reading the body, is bounded by the configured
// timeout, so a slow body read surfaces as [KindTimeout] rather than hanging.
func (c *Client) doBuffered(ctx context.Context, method, path string, body []byte, perCall http.Header, idem idempotency) ([]byte, error) {
	url := c.endpoint(path)
	maxAttempts := c.maxRetries + 1

	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, abortedError(err)
		}

		attemptCtx, cancel := c.attemptContext(ctx)
		res, err := c.attempt(attemptCtx, method, url, body, perCall)
		cancel()

		if err != nil {
			if isRetryableTransport(err) && idem == idempotent && attempt+1 < maxAttempts {
				if waitErr := sleepBackoff(ctx, attempt, 0); waitErr != nil {
					return nil, waitErr
				}
				continue
			}
			return nil, err
		}

		if res.status >= 200 && res.status < 300 {
			return res.body, nil
		}

		if shouldRetry(res.status, idem, attempt, maxAttempts) {
			if waitErr := sleepBackoff(ctx, attempt, res.retryAfter); waitErr != nil {
				return nil, waitErr
			}
			continue
		}

		return nil, apiError(res.status, res.requestID, res.retryAfter, res.body)
	}
}

// bufferedResult holds the outcome of a single buffered request attempt.
type bufferedResult struct {
	body       []byte
	status     int
	retryAfter int
	requestID  string
}

// attempt performs a single buffered request: it sends, reads the full body,
// and returns it along with the status, Retry-After, and request id. A non-2xx
// status is not an error here; the caller decides whether to retry.
func (c *Client) attempt(ctx context.Context, method, url string, body []byte, perCall http.Header) (*bufferedResult, error) {
	req, err := c.buildRequest(ctx, method, url, body, perCall)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, transportError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	res := &bufferedResult{
		status:     resp.StatusCode,
		retryAfter: parseRetryAfter(resp.Header),
		requestID:  extractRequestID(resp.Header),
	}
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if readErr != nil {
		return nil, transportError(readErr)
	}
	res.body = data
	return res, nil
}

// doStreaming executes a request with retries and returns the live response for
// the caller to stream. Only the establishment of the response is bounded by
// the configured timeout; the body is left open for the caller.
func (c *Client) doStreaming(ctx context.Context, method, path string, body []byte, perCall http.Header, idem idempotency) (*http.Response, error) {
	url := c.endpoint(path)
	maxAttempts := c.maxRetries + 1

	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, abortedError(err)
		}

		req, err := c.buildRequest(ctx, method, url, body, perCall)
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			mapped := transportError(err)
			if mapped.Kind != KindAborted && idem == idempotent && attempt+1 < maxAttempts {
				if waitErr := sleepBackoff(ctx, attempt, 0); waitErr != nil {
					return nil, waitErr
				}
				continue
			}
			return nil, mapped
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		retryAfter := parseRetryAfter(resp.Header)
		if shouldRetry(resp.StatusCode, idem, attempt, maxAttempts) {
			drainAndClose(resp.Body)
			if waitErr := sleepBackoff(ctx, attempt, retryAfter); waitErr != nil {
				return nil, waitErr
			}
			continue
		}

		return nil, c.apiErrorFromResponse(resp, retryAfter)
	}
}

// attemptContext derives a per-attempt context bounded by the configured
// timeout, or the parent context unchanged when no timeout is set.
func (c *Client) attemptContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.timeout)
}

// isRetryableTransport reports whether err is a transport failure eligible for
// retry (anything but a caller-initiated cancellation).
func isRetryableTransport(err error) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Kind == KindNetwork || e.Kind == KindTimeout
}

// buildRequest assembles a single HTTP request, applying default headers,
// per-call headers, the JSON content type, and the bearer token last so no
// caller header can shadow it.
func (c *Client) buildRequest(ctx context.Context, method, url string, body []byte, perCall http.Header) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, &Error{Kind: KindConfig, Message: "invalid request: " + err.Error(), cause: err}
	}

	for name, values := range c.defaultHeaders {
		for _, v := range values {
			req.Header.Add(name, v)
		}
	}
	for name, values := range perCall {
		req.Header[http.CanonicalHeaderKey(name)] = values
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("User-Agent", userAgent)

	// The bearer token is owned by the SDK; drop any caller-supplied
	// Authorization (case-insensitively) before applying it.
	req.Header.Del("Authorization")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	return req, nil
}

// shouldRetry reports whether a non-2xx status warrants another attempt.
func shouldRetry(status int, idem idempotency, attempt, maxAttempts int) bool {
	if attempt+1 >= maxAttempts {
		return false
	}
	if status == http.StatusTooManyRequests {
		return true
	}
	return status >= 500 && idem == idempotent
}

// apiErrorFromResponse drains a non-2xx response and converts it to a [*Error].
func (c *Client) apiErrorFromResponse(resp *http.Response, retryAfter int) error {
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	requestID := extractRequestID(resp.Header)
	return apiError(resp.StatusCode, requestID, retryAfter, body)
}

// encodeJSON marshals a request body, wrapping failures as a config error.
func encodeJSON(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, &Error{Kind: KindConfig, Message: "failed to encode request body: " + err.Error(), cause: err}
	}
	return data, nil
}

// decodeInto unmarshals a buffered success body into out.
func decodeInto(body []byte, out any) error {
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return &Error{Kind: KindDecode, Message: "malformed response body", Raw: body, cause: err}
	}
	return nil
}

// transportError maps a transport-level failure to a [*Error], distinguishing
// context cancellation, deadline/timeout, and generic network failures.
func transportError(err error) *Error {
	switch {
	case errors.Is(err, context.Canceled):
		return abortedError(err)
	case errors.Is(err, context.DeadlineExceeded):
		return &Error{Kind: KindTimeout, Message: "request timed out", cause: err}
	}
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &Error{Kind: KindTimeout, Message: "request timed out", cause: err}
	}
	return &Error{Kind: KindNetwork, Message: "network error: " + err.Error(), cause: err}
}

// abortedError wraps a context cancellation cause as a [KindAborted] error.
func abortedError(cause error) *Error {
	return &Error{Kind: KindAborted, Message: "request aborted", cause: cause}
}

// parseRetryAfter reads the Retry-After header as an integer number of seconds.
func parseRetryAfter(header http.Header) int {
	v := strings.TrimSpace(header.Get("Retry-After"))
	if v == "" {
		return 0
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		return 0
	}
	return secs
}

// extractRequestID returns the first request-id header present, in priority
// order.
func extractRequestID(header http.Header) string {
	for _, name := range []string{"X-Request-Id", "X-Skailar-Request-Id", "Request-Id"} {
		if v := header.Get(name); v != "" {
			return v
		}
	}
	return ""
}

// drainAndClose discards and closes a response body so the connection can be
// reused.
func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 1<<20))
	_ = body.Close()
}
