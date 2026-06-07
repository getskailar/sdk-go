package skailar

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatCompletionReturnsMessage(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer "+testKey, r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		writeJSON(t, w, sampleCompletion("Hi!"))
	})
	client := newTestClient(t, server.URL)

	res, err := client.Chat.Completions.Create(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{userMessage("hi")},
	})
	require.NoError(t, err)
	require.Equal(t, "Hi!", res.Choices[0].Message.Content.Text())
	require.Equal(t, FinishStop, res.Choices[0].FinishReason)
	require.Equal(t, 3, res.Usage.TotalTokens)
}

func TestChatRequestSerializesBody(t *testing.T) {
	var body map[string]any
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(t, w, sampleCompletion("ok"))
	})
	client := newTestClient(t, server.URL)

	_, err := client.Chat.Completions.Create(context.Background(), ChatCompletionRequest{
		Model:       "claude-sonnet-4-6",
		Messages:    []ChatMessage{userMessage("hi")},
		Temperature: Ptr(0.7),
		MaxTokens:   Ptr(256),
	})
	require.NoError(t, err)
	require.Equal(t, "claude-sonnet-4-6", body["model"])
	require.InDelta(t, 0.7, body["temperature"], 1e-9)
	require.InDelta(t, 256, body["max_tokens"], 1e-9)
}

func TestChatRequestOmitsUnsetOptionalFields(t *testing.T) {
	var raw map[string]json.RawMessage
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
		writeJSON(t, w, sampleCompletion("ok"))
	})
	client := newTestClient(t, server.URL)

	_, err := client.Chat.Completions.Create(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{userMessage("hi")},
	})
	require.NoError(t, err)
	require.NotContains(t, raw, "temperature")
	require.NotContains(t, raw, "max_tokens")
	require.NotContains(t, raw, "stream")
	require.NotContains(t, raw, "tools")
	require.Contains(t, raw, "model")
	require.Contains(t, raw, "messages")
}

func TestChatRequestSerializesToolChoiceModes(t *testing.T) {
	cases := map[string]struct {
		choice *ToolChoice
		want   string
	}{
		"auto":     {ToolChoiceAuto(), `"auto"`},
		"none":     {ToolChoiceNone(), `"none"`},
		"required": {ToolChoiceRequired(), `"required"`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b, err := json.Marshal(tc.choice)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(b))
		})
	}
}

func TestChatRequestSerializesNamedToolChoice(t *testing.T) {
	b, err := json.Marshal(NamedToolChoice("get_weather"))
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"function","function":{"name":"get_weather"}}`, string(b))
}

func TestMultimodalContentSerializes(t *testing.T) {
	msg := ChatMessage{
		Role: RoleUser,
		Content: PartsContent(
			TextPart("look:"),
			ImagePart("https://example.com/a.png"),
		),
	}
	b, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded struct {
		Role    string `json:"role"`
		Content []struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			ImageURL *struct {
				URL string `json:"url"`
			} `json:"image_url"`
		} `json:"content"`
	}
	require.NoError(t, json.Unmarshal(b, &decoded))
	require.Equal(t, "user", decoded.Role)
	require.Len(t, decoded.Content, 2)
	require.Equal(t, "text", decoded.Content[0].Type)
	require.Equal(t, "look:", decoded.Content[0].Text)
	require.Equal(t, "image_url", decoded.Content[1].Type)
	require.Equal(t, "https://example.com/a.png", decoded.Content[1].ImageURL.URL)
}

func TestStringContentTextHelper(t *testing.T) {
	require.Equal(t, "hello", TextContent("hello").Text())
	require.Equal(t, "ab", PartsContent(TextPart("a"), ImagePart("x"), TextPart("b")).Text())
}

func TestAssistantMessageWithoutContentOmitsField(t *testing.T) {
	msg := ChatMessage{
		Role:      RoleAssistant,
		ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "f", Arguments: "{}"}}},
	}
	b, err := json.Marshal(msg)
	require.NoError(t, err)
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &raw))
	require.NotContains(t, raw, "content")
	require.Contains(t, raw, "tool_calls")
}

func TestToolMessageCarriesCallID(t *testing.T) {
	msg := ChatMessage{Role: RoleTool, ToolCallID: "call_1", Content: TextContent("42")}
	b, err := json.Marshal(msg)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(b, &raw))
	require.Equal(t, "tool", raw["role"])
	require.Equal(t, "call_1", raw["tool_call_id"])
}

func TestChatResponseDecodesToolCalls(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"id": "x", "object": "chat.completion", "created": 1, "model": "m",
			"choices": []map[string]any{{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": "",
					"tool_calls": []map[string]any{{
						"id": "call_1", "type": "function",
						"function": map[string]any{"name": "get_weather", "arguments": `{"city":"Rome"}`},
					}},
				},
				"finish_reason": "tool_calls",
			}},
			"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	})
	client := newTestClient(t, server.URL)

	res, err := client.Chat.Completions.Create(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []ChatMessage{userMessage("weather?")},
	})
	require.NoError(t, err)
	require.Equal(t, FinishToolCalls, res.Choices[0].FinishReason)
	require.Len(t, res.Choices[0].Message.ToolCalls, 1)
	require.Equal(t, "get_weather", res.Choices[0].Message.ToolCalls[0].Function.Name)
	require.JSONEq(t, `{"city":"Rome"}`, res.Choices[0].Message.ToolCalls[0].Function.Arguments)
}

func TestImagesGenerate(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/images/generations", r.URL.Path)
		writeJSON(t, w, map[string]any{
			"created": 1,
			"data":    []map[string]any{{"url": "https://img.example/1.png", "revised_prompt": "a cat"}},
		})
	})
	client := newTestClient(t, server.URL)

	res, err := client.Images.Generate(context.Background(), ImageGenerationRequest{
		Model:  "gpt-image-1",
		Prompt: "a cat",
		N:      Ptr(1),
	})
	require.NoError(t, err)
	require.Len(t, res.Data, 1)
	require.Equal(t, "https://img.example/1.png", res.Data[0].URL)
	require.Equal(t, "a cat", res.Data[0].RevisedPrompt)
}

func TestTranscriptionCreate(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/audio/transcriptions", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "AAAA", body["base64"])
		require.Equal(t, "audio/mp3", body["mime"])
		writeJSON(t, w, TranscriptionResponse{Text: "hello world"})
	})
	client := newTestClient(t, server.URL)

	res, err := client.Audio.Transcriptions.Create(context.Background(), TranscriptionRequest{
		Base64: "AAAA",
		Mime:   MimeMp3,
	})
	require.NoError(t, err)
	require.Equal(t, "hello world", res.Text)
}

func TestSpeechCreateReturnsBytes(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/audio/speech", r.URL.Path)
		require.Equal(t, "audio/mpeg", r.Header.Get("Accept"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "hello", body["input"])
		require.Equal(t, "nova", body["voice"])
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3-mp3-bytes"))
	})
	client := newTestClient(t, server.URL)

	rc, err := client.Audio.Speech.Create(context.Background(), SpeechRequest{Input: "hello", Voice: VoiceNova})
	require.NoError(t, err)
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, "ID3-mp3-bytes", string(data))
}

func TestSpeechCreateBytes(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("clip"))
	})
	client := newTestClient(t, server.URL)

	data, err := client.Audio.Speech.CreateBytes(context.Background(), SpeechRequest{Input: "hi"})
	require.NoError(t, err)
	require.Equal(t, "clip", string(data))
}

func TestUploadImage(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/uploads/images", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "image/png", body["content_type"])
		writeJSON(t, w, UploadResponse{URL: "/assets/abc.png", ContentType: "image/png"})
	})
	client := newTestClient(t, server.URL)

	res, err := client.Uploads.Images.Create(context.Background(), "AAAA", ImagePNG)
	require.NoError(t, err)
	require.Equal(t, "/assets/abc.png", res.URL)
	require.Equal(t, "image/png", res.ContentType)
}

func TestUploadFile(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/uploads/files", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "application/pdf", body["content_type"])
		writeJSON(t, w, UploadResponse{URL: "/assets/doc.pdf", ContentType: "application/pdf"})
	})
	client := newTestClient(t, server.URL)

	res, err := client.Uploads.Files.Create(context.Background(), "AAAA", FilePDF)
	require.NoError(t, err)
	require.Equal(t, "/assets/doc.pdf", res.URL)
}
