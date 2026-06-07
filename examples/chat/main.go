// Command chat sends a single chat completion and prints the reply.
//
// Run it with a valid key:
//
//	SKAILAR_API_KEY=skl_live_... go run ./examples/chat
package main

import (
	"context"
	"fmt"
	"log"

	skailar "github.com/getskailar/sdk-go"
)

func main() {
	client, err := skailar.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	res, err := client.Chat.Completions.Create(ctx, skailar.ChatCompletionRequest{
		Model: skailar.ModelClaudeSonnet4_6,
		Messages: []skailar.ChatMessage{
			{Role: skailar.RoleSystem, Content: skailar.TextContent("You are concise.")},
			{Role: skailar.RoleUser, Content: skailar.TextContent("Say hello in one word.")},
		},
		MaxTokens: skailar.Ptr(64),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res.Choices[0].Message.Content.Text())
	fmt.Printf("tokens: %d prompt + %d completion = %d total\n",
		res.Usage.PromptTokens, res.Usage.CompletionTokens, res.Usage.TotalTokens)
}
