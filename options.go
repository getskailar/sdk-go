package skailar

import (
	"net/http"
	"time"
)

// Option configures a [Client] during [NewClient]. Options are applied in
// order; later options override earlier ones.
type Option func(*clientConfig)

// clientConfig holds the resolved configuration for a [Client]. It is internal
// and immutable once [NewClient] returns.
type clientConfig struct {
	apiKey         string
	baseURL        string
	timeout        time.Duration
	maxRetries     int
	httpClient     *http.Client
	defaultHeaders http.Header
}

// WithAPIKey sets the API key, overriding the SKAILAR_API_KEY environment
// variable.
func WithAPIKey(key string) Option {
	return func(c *clientConfig) { c.apiKey = key }
}

// WithBaseURL sets the base URL, overriding the SKAILAR_BASE_URL environment
// variable and the default of https://api.skailar.com.
func WithBaseURL(url string) Option {
	return func(c *clientConfig) { c.baseURL = url }
}

// WithTimeout sets the per-request timeout. The default is 60 seconds. A
// non-positive duration disables the SDK-managed timeout, deferring entirely to
// the request context.
func WithTimeout(timeout time.Duration) Option {
	return func(c *clientConfig) { c.timeout = timeout }
}

// WithMaxRetries sets the maximum number of retries for eligible requests. The
// default is 2 (three attempts total). A value of 0 disables retries.
func WithMaxRetries(n int) Option {
	return func(c *clientConfig) {
		if n < 0 {
			n = 0
		}
		c.maxRetries = n
	}
}

// WithHTTPClient supplies an existing [*http.Client] whose connection pool and
// transport settings the SDK reuses. When set, [WithTimeout] still governs the
// per-request deadline via context.
func WithHTTPClient(client *http.Client) Option {
	return func(c *clientConfig) { c.httpClient = client }
}

// WithDefaultHeader adds a header sent on every request. An Authorization
// header set here is ignored; the SDK always applies its own bearer token.
func WithDefaultHeader(name, value string) Option {
	return func(c *clientConfig) {
		if c.defaultHeaders == nil {
			c.defaultHeaders = make(http.Header)
		}
		c.defaultHeaders.Set(name, value)
	}
}
