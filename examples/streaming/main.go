// Command streaming prints a streamed chat completion token by token.
//
//	SKAILAR_API_KEY=skl_live_... go run ./examples/streaming
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	skailar "github.com/getskailar/sdk-go"
)

func main() {
	client, err := skailar.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	stream, err := client.Chat.Completions.CreateStream(context.Background(), skailar.ChatCompletionRequest{
		Model: skailar.ModelClaudeSonnet4_6,
		Messages: []skailar.ChatMessage{
			{Role: skailar.RoleUser, Content: skailar.TextContent("Count from 1 to 5, one number per line.")},
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
	fmt.Fprintln(os.Stderr, "\n[stream complete]")
}
