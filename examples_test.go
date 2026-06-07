package skailar_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	skailar "github.com/getskailar/sdk-go"
)

func ExampleClient_chat() {
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

func ExampleChatCompletionsService_CreateStream() {
	client, err := skailar.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	stream, err := client.Chat.Completions.CreateStream(context.Background(), skailar.ChatCompletionRequest{
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
		if text, ok := stream.Current().ContentDelta(); ok {
			fmt.Print(text)
		}
	}
	if err := stream.Err(); err != nil {
		log.Fatal(err)
	}
}

func ExampleNewClient_options() {
	client, err := skailar.NewClient(
		skailar.WithAPIKey("skl_live_..."),
		skailar.WithBaseURL("http://localhost:8080"),
		skailar.WithTimeout(30*time.Second),
		skailar.WithMaxRetries(2),
		skailar.WithDefaultHeader("x-trace-id", "abc123"),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

func ExampleError() {
	client, err := skailar.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.Chat.Completions.Create(context.Background(), skailar.ChatCompletionRequest{
		Model:    skailar.ModelClaudeSonnet4_6,
		Messages: []skailar.ChatMessage{{Role: skailar.RoleUser, Content: skailar.TextContent("hi")}},
	})
	if errors.Is(err, skailar.ErrRateLimit) {
		var apiErr *skailar.Error
		if errors.As(err, &apiErr) {
			time.Sleep(time.Duration(apiErr.RetryAfter) * time.Second)
		}
	}
}
