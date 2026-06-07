package skailar

import "context"

// PingKeyResponse is the response of [Client.Ping].
type PingKeyResponse struct {
	// Status is the health status; always "ok" on success.
	Status string `json:"status"`
	// UserID is the identifier of the account that owns the API key.
	UserID string `json:"user_id"`
}

// Ping verifies the API key against GET /v1/ping-key. It returns a [*Error] of
// [KindAuth] if the key is invalid. This is an idempotent call and is retried
// on 5xx responses.
func (c *Client) Ping(ctx context.Context) (*PingKeyResponse, error) {
	var out PingKeyResponse
	if err := c.getJSON(ctx, "v1/ping-key", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
