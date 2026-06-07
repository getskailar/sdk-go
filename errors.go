package skailar

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Kind classifies an [Error] so callers can branch without matching on strings.
type Kind int

const (
	// KindAPI is a non-2xx response that does not map to a more specific kind.
	KindAPI Kind = iota
	// KindAuth is a 401 response: missing, invalid, or revoked API key.
	KindAuth
	// KindBadRequest is a 400 response: the request was malformed.
	KindBadRequest
	// KindNotFound is a 404 response: the resource does not exist.
	KindNotFound
	// KindRateLimit is a 429 response: the rate limit was exceeded.
	KindRateLimit
	// KindUpstream is a 5xx response: the upstream provider failed or timed out.
	KindUpstream
	// KindNetwork is a transport failure (DNS, TLS, connection reset).
	KindNetwork
	// KindTimeout is a request that exceeded the configured timeout.
	KindTimeout
	// KindAborted is a request cancelled via its context.
	KindAborted
	// KindDecode is a successful response whose body could not be decoded.
	KindDecode
	// KindConfig is a client misconfiguration, such as a missing API key.
	KindConfig
)

// String returns the lowercase name of the kind.
func (k Kind) String() string {
	switch k {
	case KindAPI:
		return "api"
	case KindAuth:
		return "auth"
	case KindBadRequest:
		return "bad_request"
	case KindNotFound:
		return "not_found"
	case KindRateLimit:
		return "rate_limit"
	case KindUpstream:
		return "upstream"
	case KindNetwork:
		return "network"
	case KindTimeout:
		return "timeout"
	case KindAborted:
		return "aborted"
	case KindDecode:
		return "decode"
	case KindConfig:
		return "config"
	default:
		return "unknown"
	}
}

// Error is the single error type returned by every fallible operation in this
// package. Discriminate with [Kind] via the package sentinels and [errors.Is],
// and recover the full detail with [errors.As].
type Error struct {
	// Kind classifies the failure.
	Kind Kind
	// Status is the HTTP status code, or 0 when not applicable.
	Status int
	// Code is the machine-readable error code from the response body, if any.
	Code string
	// Message is the human-readable error message.
	Message string
	// RequestID is the server-assigned request identifier, if present.
	RequestID string
	// Raw is the raw response body, if one was read.
	Raw []byte
	// RetryAfter is the uncapped Retry-After value in seconds, set on a 429.
	//
	// The retry loop caps the delay it actually waits at 60 seconds; this field
	// reports the original server value.
	RetryAfter int

	cause error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("skailar: %s error (status %d): %s", e.Kind, e.Status, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("skailar: %s error: %s", e.Kind, e.Message)
	}
	return fmt.Sprintf("skailar: %s error", e.Kind)
}

// Unwrap returns the underlying cause, if any, for use with [errors.Unwrap].
func (e *Error) Unwrap() error { return e.cause }

// Is reports whether target is an [*Error] of the same [Kind]. It lets callers
// match against the package sentinels (which carry only a Kind) with
// [errors.Is].
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Kind == t.Kind
}

// Sentinel errors for use with [errors.Is]. Each carries only a [Kind]; match
// against them and use [errors.As] to read status, code, and Retry-After.
var (
	// ErrAuth matches any 401 failure.
	ErrAuth = &Error{Kind: KindAuth}
	// ErrBadRequest matches any 400 failure.
	ErrBadRequest = &Error{Kind: KindBadRequest}
	// ErrNotFound matches any 404 failure.
	ErrNotFound = &Error{Kind: KindNotFound}
	// ErrRateLimit matches any 429 failure.
	ErrRateLimit = &Error{Kind: KindRateLimit}
	// ErrUpstream matches any 5xx failure.
	ErrUpstream = &Error{Kind: KindUpstream}
	// ErrNetwork matches any transport failure.
	ErrNetwork = &Error{Kind: KindNetwork}
	// ErrTimeout matches any timeout.
	ErrTimeout = &Error{Kind: KindTimeout}
	// ErrAborted matches any context cancellation.
	ErrAborted = &Error{Kind: KindAborted}
)

// kindForStatus maps an HTTP status code to the corresponding [Kind].
func kindForStatus(status int) Kind {
	switch {
	case status == 401:
		return KindAuth
	case status == 400:
		return KindBadRequest
	case status == 404:
		return KindNotFound
	case status == 429:
		return KindRateLimit
	case status >= 500:
		return KindUpstream
	default:
		return KindAPI
	}
}

// newConfigError builds a [KindConfig] error with the given message.
func newConfigError(message string) *Error {
	return &Error{Kind: KindConfig, Message: message}
}

// apiError builds an [Error] from a non-2xx response.
func apiError(status int, requestID string, retryAfter int, body []byte) *Error {
	code, message := parseErrorFields(body)
	if message == "" {
		if trimmed := strings.TrimSpace(string(body)); trimmed != "" {
			message = trimmed
		} else {
			message = fmt.Sprintf("HTTP %d", status)
		}
	}
	return &Error{
		Kind:       kindForStatus(status),
		Status:     status,
		Code:       code,
		Message:    message,
		RequestID:  requestID,
		Raw:        body,
		RetryAfter: retryAfter,
	}
}

// parseErrorFields extracts (code, message) from a Skailar or OpenAI error
// body, tolerating three layouts so the SDK keeps working if the wire shape
// shifts:
//
//   - flat: {"error": "code", "message": "msg"}
//   - nested object: {"error": {"type"|"code": "...", "message": "..."}}
//   - OpenAI-style: {"error": {"code": "...", "message": "..."}}
func parseErrorFields(body []byte) (code, message string) {
	var envelope struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", ""
	}

	if len(envelope.Error) == 0 {
		return "", envelope.Message
	}

	var flat string
	if err := json.Unmarshal(envelope.Error, &flat); err == nil {
		return flat, envelope.Message
	}

	var nested struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(envelope.Error, &nested); err == nil {
		code = nested.Type
		if code == "" {
			code = nested.Code
		}
		return code, nested.Message
	}

	return "", envelope.Message
}
