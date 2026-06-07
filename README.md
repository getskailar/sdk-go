# Skailar SDK for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/getskailar/sdk-go.svg)](https://pkg.go.dev/github.com/getskailar/sdk-go)

The official Go SDK for [Skailar](https://skailar.com), a multi-provider LLM
gateway with an OpenAI-compatible surface. It covers chat completions (including
streaming), model discovery, image generation, speech synthesis and
transcription, and storage uploads — using only the Go standard library at
runtime.

## Installation

```sh
go get github.com/getskailar/sdk-go
```

Requires Go 1.22 or newer.

## Quickstart

`NewClient` reads the API key from the `SKAILAR_API_KEY` environment variable.
Every method that performs I/O takes a `context.Context` first.

```go
package main

import (
	"context"
	"fmt"
	"log"

	skailar "github.com/getskailar/sdk-go"
)

func main() {
	client, err := skailar.NewClient() // reads SKAILAR_API_KEY
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	res, err := client.Chat.Completions.Create(ctx, skailar.ChatCompletionRequest{
		Model: skailar.ModelClaudeSonnet4_6,
		Messages: []skailar.ChatMessage{
			{Role: skailar.RoleUser, Content: skailar.TextContent("Hello!")},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res.Choices[0].Message.Content.Text())
}
```

## Streaming

`CreateStream` returns a `*ChatCompletionStream` that follows the standard Go
iterator shape used by `bufio.Scanner` and `database/sql.Rows`. Always
`Close` the stream to release the connection.

```go
stream, err := client.Chat.Completions.CreateStream(ctx, skailar.ChatCompletionRequest{
	Model: skailar.ModelClaudeSonnet4_6,
	Messages: []skailar.ChatMessage{
		{Role: skailar.RoleUser, Content: skailar.TextContent("Count to 5")},
	},
})
if err != nil {
	log.Fatal(err)
}
defer stream.Close()

for stream.Next() {
	chunk := stream.Current()
	if text, ok := chunk.ContentDelta(); ok {
		fmt.Print(text)
	}
}
if err := stream.Err(); err != nil {
	log.Fatal(err)
}
```

The stream surfaces an error from `Err()` (not `Next()`); a clean end of stream
leaves `Err()` nil.

## Configuration

The client is configured with functional options, applied in order. It is safe
for concurrent use once constructed.

```go
client, err := skailar.NewClient(
	skailar.WithAPIKey("skl_live_..."),                 // overrides SKAILAR_API_KEY
	skailar.WithBaseURL("http://localhost:8080"),       // overrides SKAILAR_BASE_URL
	skailar.WithTimeout(30*time.Second),                // per-request timeout (default 60s)
	skailar.WithMaxRetries(2),                          // default 2 (three attempts total)
	skailar.WithHTTPClient(myExistingHTTPClient),       // reuse a connection pool
	skailar.WithDefaultHeader("x-trace-id", "abc123"),  // sent on every request
)
```

The `Authorization` header is always managed by the SDK; a value passed to
`WithDefaultHeader` (in any case) is ignored.

## Optional fields

Wire-optional fields use pointers so "unset" is distinct from a zero value. Use
the `Ptr` helper:

```go
req := skailar.ChatCompletionRequest{
	Model:       skailar.ModelClaudeSonnet4_6,
	Messages:    messages,
	Temperature: skailar.Ptr(0.7),
	MaxTokens:   skailar.Ptr(1024),
}
```

## Error handling

Every call returns a `*skailar.Error` with a `Kind`. Match the package sentinels
with `errors.Is`, and read the detail with `errors.As`:

```go
_, err := client.Chat.Completions.Create(ctx, req)
if errors.Is(err, skailar.ErrRateLimit) {
	var apiErr *skailar.Error
	if errors.As(err, &apiErr) {
		time.Sleep(time.Duration(apiErr.RetryAfter) * time.Second)
	}
}
```

Sentinels: `ErrAuth`, `ErrBadRequest`, `ErrNotFound`, `ErrRateLimit`,
`ErrUpstream`, `ErrNetwork`, `ErrTimeout`, `ErrAborted`. The retry policy retries
`429` (always) and `5xx` (only for idempotent `GET`s), with exponential backoff
and full jitter; side-effecting `POST`s are never retried on `5xx`, to avoid
double billing.

## Context cancellation

Cancellation and deadlines flow through the `context.Context`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

res, err := client.Chat.Completions.Create(ctx, req)
if errors.Is(err, skailar.ErrAborted) {
	// the context was cancelled
}
```

A context cancelled mid-retry interrupts the backoff immediately and returns
`ErrAborted`.

## Drop-in OpenAI alternative

The wire format mirrors the OpenAI API, so these request and response types are
drop-in compatible for the covered endpoints. Go has no official OpenAI SDK to
replace; if you currently use a third-party client such as
[`sashabaranov/go-openai`](https://github.com/sashabaranov/go-openai), you can
migrate by pointing its base URL at `https://api.skailar.com/v1` with your
Skailar key, or adopt this SDK's types directly.

## Local development

```sh
go build ./...
go test ./... -race
go vet ./...
golangci-lint run      # if installed; otherwise: staticcheck ./...
go doc -all .
```

Run an example against a local gateway:

```sh
SKAILAR_API_KEY=skl_live_... SKAILAR_BASE_URL=http://localhost:8080 go run ./examples/chat
```

## Status

Pre-release `v0.0.1`. The API may change before `v1.0.0`.

## License

[MIT](./LICENSE)
