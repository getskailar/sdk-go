// Command vision sends a multimodal message containing text and an image URL.
//
//	SKAILAR_API_KEY=skl_live_... go run ./examples/vision
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
			{
				Role: skailar.RoleUser,
				Content: skailar.PartsContent(
					skailar.TextPart("What is in this image?"),
					skailar.ImagePart("https://upload.wikimedia.org/wikipedia/commons/3/3a/Cat03.jpg"),
				),
			},
		},
		MaxTokens: skailar.Ptr(300),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res.Choices[0].Message.Content.Text())
}
