package skailar

import (
	"context"
	"net/url"
	"strings"
)

// ModelsService is the model-catalog resource, accessed as Client.Models.
type ModelsService struct {
	client *Client
}

// List returns every model the gateway can route to, across all providers, as a
// flat list. This is an idempotent call and is retried on 5xx responses.
func (s *ModelsService) List(ctx context.Context) (*ModelList, error) {
	var out ModelList
	if err := s.client.getJSON(ctx, "v1/models", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Retrieve returns the full detail card for one model. The id may contain
// slashes (for example, "google/gemini-2.5-pro"); each segment is escaped. This
// is an idempotent call and is retried on 5xx responses.
func (s *ModelsService) Retrieve(ctx context.Context, id string) (*Model, error) {
	var out Model
	if err := s.client.getJSON(ctx, "v1/models/"+escapePathSegments(id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// escapePathSegments percent-encodes each slash-separated segment of id while
// preserving the separators, so model ids like "google/gemini-2.5-pro" are
// routed correctly.
func escapePathSegments(id string) string {
	segments := strings.Split(id, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}
