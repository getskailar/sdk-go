// Package skailar is the official Go SDK for the Skailar API, a multi-provider
// LLM gateway with an OpenAI-compatible surface.
//
// The package talks to chat completions, model discovery, image generation,
// speech synthesis and transcription, and storage uploads through a single
// bearer-authenticated client. Calls are billed per request from the Skailar
// account that owns the API key.
//
// # Authentication
//
// The API key is a bearer token of the form skl_live_<43 url-safe base64
// characters>, created from the dashboard at https://skailar.com. NewClient
// reads it from the SKAILAR_API_KEY environment variable, or it can be passed
// explicitly with WithAPIKey.
//
// # Client
//
// Construct a client once and reuse it; it is safe for concurrent use by
// multiple goroutines.
//
//	client, err := skailar.NewClient()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	res, err := client.Chat.Completions.Create(ctx, skailar.ChatCompletionRequest{
//		Model: skailar.ModelClaudeSonnet4_6,
//		Messages: []skailar.ChatMessage{
//			{Role: skailar.RoleUser, Content: skailar.TextContent("Hello!")},
//		},
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(res.Choices[0].Message.Content)
//
// Every method that performs I/O takes a [context.Context] as its first
// argument, so callers control cancellation and deadlines.
//
// # Streaming
//
// CreateStream returns a [ChatCompletionStream] that follows the standard Go
// iterator shape used by bufio.Scanner and database/sql.Rows:
//
//	stream, err := client.Chat.Completions.CreateStream(ctx, req)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer stream.Close()
//
//	for stream.Next() {
//		chunk := stream.Current()
//		if text, ok := chunk.ContentDelta(); ok {
//			fmt.Print(text)
//		}
//	}
//	if err := stream.Err(); err != nil {
//		log.Fatal(err)
//	}
//
// # Errors
//
// Every fallible call returns a [*Error] with a [Kind] discriminant. Branch on
// it with the package sentinels and [errors.Is], and recover the full detail
// with [errors.As]:
//
//	if errors.Is(err, skailar.ErrRateLimit) {
//		var apiErr *skailar.Error
//		if errors.As(err, &apiErr) {
//			time.Sleep(time.Duration(apiErr.RetryAfter) * time.Second)
//		}
//	}
//
// # OpenAI compatibility
//
// The wire format mirrors the OpenAI API, so the request and response types
// here are drop-in compatible for the covered endpoints. Go has no official
// OpenAI SDK to replace; projects using a third-party client can migrate by
// pointing the base URL at Skailar and adopting these types.
package skailar
