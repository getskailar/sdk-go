// Command tool_calling demonstrates a two-turn function-calling exchange: the
// model requests a tool call, the program answers it, and the model produces a
// final reply.
//
//	SKAILAR_API_KEY=skl_live_... go run ./examples/tool_calling
package main

import (
	"context"
	"encoding/json"
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

	weatherTool := skailar.NewFunctionTool(
		"get_weather",
		"Get the current temperature for a city.",
		json.RawMessage(`{
			"type": "object",
			"properties": {"city": {"type": "string"}},
			"required": ["city"]
		}`),
	)

	messages := []skailar.ChatMessage{
		{Role: skailar.RoleUser, Content: skailar.TextContent("What's the weather in Rome?")},
	}

	first, err := client.Chat.Completions.Create(ctx, skailar.ChatCompletionRequest{
		Model:      skailar.ModelClaudeSonnet4_6,
		Messages:   messages,
		Tools:      []skailar.Tool{weatherTool},
		ToolChoice: skailar.ToolChoiceAuto(),
	})
	if err != nil {
		log.Fatal(err)
	}

	calls := first.Choices[0].Message.ToolCalls
	if len(calls) == 0 {
		fmt.Println(first.Choices[0].Message.Content.Text())
		return
	}

	// Echo the assistant's tool-call message back, then answer each call.
	messages = append(messages, skailar.ChatMessage{
		Role:      skailar.RoleAssistant,
		ToolCalls: calls,
	})
	for _, call := range calls {
		fmt.Printf("model called %s(%s)\n", call.Function.Name, call.Function.Arguments)
		messages = append(messages, skailar.ChatMessage{
			Role:       skailar.RoleTool,
			ToolCallID: call.ID,
			Content:    skailar.TextContent(`{"temperature_c": 24}`),
		})
	}

	second, err := client.Chat.Completions.Create(ctx, skailar.ChatCompletionRequest{
		Model:    skailar.ModelClaudeSonnet4_6,
		Messages: messages,
		Tools:    []skailar.Tool{weatherTool},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(second.Choices[0].Message.Content.Text())
}
