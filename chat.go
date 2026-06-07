package skailar

import "context"

// ChatService is the chat resource, accessed as Client.Chat.
type ChatService struct {
	// Completions is the chat-completions sub-resource.
	Completions *ChatCompletionsService

	client *Client
}

// ChatCompletionsService is the chat-completions resource, accessed as
// Client.Chat.Completions.
type ChatCompletionsService struct {
	client *Client
}

// Create generates a non-streamed chat completion.
//
// For streaming, use [ChatCompletionsService.CreateStream] instead. This is a
// billable, side-effecting call and is not retried on 5xx responses.
func (s *ChatCompletionsService) Create(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	var out ChatCompletionResponse
	if err := s.client.postJSON(ctx, "v1/chat/completions", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateStream generates a streamed chat completion, returning a
// [*ChatCompletionStream] of incremental chunks. It forces stream mode on the
// wire regardless of the request's Stream field.
//
// The caller must call [ChatCompletionStream.Close] to release the underlying
// connection. This is a billable, side-effecting call and is not retried.
func (s *ChatCompletionsService) CreateStream(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionStream, error) {
	req.Stream = true
	return s.client.postStream(ctx, "v1/chat/completions", req)
}
