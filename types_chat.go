package skailar

import (
	"bytes"
	"encoding/json"
)

// Role is the author role of a [ChatMessage].
type Role string

const (
	// RoleSystem carries system or developer instructions.
	RoleSystem Role = "system"
	// RoleUser carries end-user input.
	RoleUser Role = "user"
	// RoleAssistant carries model output.
	RoleAssistant Role = "assistant"
	// RoleTool carries the result of a tool call, paired with a tool-call id.
	RoleTool Role = "tool"
)

// ReasoningEffort is the reasoning budget for reasoning-capable models.
type ReasoningEffort string

const (
	// ReasoningLow requests minimal reasoning.
	ReasoningLow ReasoningEffort = "low"
	// ReasoningMedium requests balanced reasoning.
	ReasoningMedium ReasoningEffort = "medium"
	// ReasoningHigh requests maximum reasoning.
	ReasoningHigh ReasoningEffort = "high"
)

// FinishReason is why a completion stopped generating.
type FinishReason string

const (
	// FinishStop is a natural stop or a stop sequence was hit.
	FinishStop FinishReason = "stop"
	// FinishLength is the token limit was reached.
	FinishLength FinishReason = "length"
	// FinishToolCalls is the model emitted tool calls.
	FinishToolCalls FinishReason = "tool_calls"
	// FinishContentFilter is the output was filtered.
	FinishContentFilter FinishReason = "content_filter"
)

// MessageContent is the content of a [ChatMessage]: either a single text
// string, an ordered list of multimodal parts, or absent.
//
// On the wire it is a string, an array of [ContentPart], or null. Construct it
// with [TextContent] or [PartsContent], and read the plain text with
// [MessageContent.Text].
type MessageContent struct {
	// Value holds either a string or a []ContentPart. It is nil when the content
	// is absent.
	text  string
	parts []ContentPart
	isSet bool
}

// TextContent builds a [MessageContent] holding a single text string.
func TextContent(text string) MessageContent {
	return MessageContent{text: text, isSet: true}
}

// PartsContent builds a [MessageContent] holding multimodal parts.
func PartsContent(parts ...ContentPart) MessageContent {
	return MessageContent{parts: parts, isSet: true}
}

// Text returns the plain text of the content. For multimodal content it
// concatenates the text of every text part; image parts contribute nothing.
func (m MessageContent) Text() string {
	if m.parts == nil {
		return m.text
	}
	var b bytes.Buffer
	for _, p := range m.parts {
		if p.Type == ContentPartText {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

// Parts returns the multimodal parts, or nil if the content is plain text.
func (m MessageContent) Parts() []ContentPart { return m.parts }

// IsZero reports whether the content is absent (and so omitted from JSON).
func (m MessageContent) IsZero() bool { return !m.isSet }

// MarshalJSON implements [json.Marshaler].
func (m MessageContent) MarshalJSON() ([]byte, error) {
	if !m.isSet {
		return []byte("null"), nil
	}
	if m.parts != nil {
		return json.Marshal(m.parts)
	}
	return json.Marshal(m.text)
}

// UnmarshalJSON implements [json.Unmarshaler].
func (m *MessageContent) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*m = MessageContent{}
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*m = MessageContent{text: s, isSet: true}
		return nil
	}
	var parts []ContentPart
	if err := json.Unmarshal(data, &parts); err != nil {
		return err
	}
	*m = MessageContent{parts: parts, isSet: true}
	return nil
}

// ContentPartType discriminates a [ContentPart].
type ContentPartType string

const (
	// ContentPartText is a run of text.
	ContentPartText ContentPartType = "text"
	// ContentPartImageURL is an image reference.
	ContentPartImageURL ContentPartType = "image_url"
)

// ContentPart is one part of a multimodal message.
type ContentPart struct {
	// Type discriminates the part.
	Type ContentPartType `json:"type"`
	// Text is set when Type is [ContentPartText].
	Text string `json:"text,omitempty"`
	// ImageURL is set when Type is [ContentPartImageURL].
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// TextPart builds a text [ContentPart].
func TextPart(text string) ContentPart {
	return ContentPart{Type: ContentPartText, Text: text}
}

// ImagePart builds an image [ContentPart] from a data: URI or HTTPS URL.
func ImagePart(url string) ContentPart {
	return ContentPart{Type: ContentPartImageURL, ImageURL: &ImageURL{URL: url}}
}

// ImageURL is an image reference within a [ContentPart].
type ImageURL struct {
	// URL is a data: URI or an HTTPS URL (for example, from Uploads.Images).
	URL string `json:"url"`
	// Detail is an optional hint: "low", "high", or "auto".
	Detail string `json:"detail,omitempty"`
}

// ChatMessage is a single message in a chat conversation.
type ChatMessage struct {
	// Role is the author role.
	Role Role `json:"role"`
	// Content is the message content; omitted when an assistant message only
	// carries tool calls.
	Content MessageContent `json:"content,omitempty"`
	// ToolCalls are tool calls requested by an assistant message.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// ToolCallID identifies the tool call this message responds to; required
	// when Role is [RoleTool].
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// chatMessageWire is the JSON shape of [ChatMessage]. It is needed because
// encoding/json does not honour the omitempty tag for struct-typed fields, so
// absent content is encoded explicitly via a pointer.
type chatMessageWire struct {
	Role       Role            `json:"role"`
	Content    *MessageContent `json:"content,omitempty"`
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// MarshalJSON implements [json.Marshaler], omitting absent content.
func (m ChatMessage) MarshalJSON() ([]byte, error) {
	wire := chatMessageWire{
		Role:       m.Role,
		ToolCalls:  m.ToolCalls,
		ToolCallID: m.ToolCallID,
	}
	if !m.Content.IsZero() {
		wire.Content = &m.Content
	}
	return json.Marshal(wire)
}

// UnmarshalJSON implements [json.Unmarshaler].
func (m *ChatMessage) UnmarshalJSON(data []byte) error {
	var wire chatMessageWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	m.Role = wire.Role
	m.ToolCalls = wire.ToolCalls
	m.ToolCallID = wire.ToolCallID
	if wire.Content != nil {
		m.Content = *wire.Content
	} else {
		m.Content = MessageContent{}
	}
	return nil
}

// StopSequence is a stop condition: a single string or several.
type StopSequence struct {
	// One is a single stop string. Ignored when Many is non-empty.
	One string
	// Many is a list of stop strings.
	Many []string
}

// Stop builds a single-sequence [StopSequence].
func Stop(s string) *StopSequence { return &StopSequence{One: s} }

// StopSequences builds a multi-sequence [StopSequence].
func StopSequences(s ...string) *StopSequence { return &StopSequence{Many: s} }

// MarshalJSON implements [json.Marshaler].
func (s StopSequence) MarshalJSON() ([]byte, error) {
	if len(s.Many) > 0 {
		return json.Marshal(s.Many)
	}
	return json.Marshal(s.One)
}

// UnmarshalJSON implements [json.Unmarshaler].
func (s *StopSequence) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		s.One = one
		s.Many = nil
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	s.One = ""
	s.Many = many
	return nil
}

// ChatCompletionRequest is a request to create a chat completion. Optional
// fields use pointer types; set them with [Ptr].
type ChatCompletionRequest struct {
	// Model is the model identifier or alias. See the Model* constants.
	Model string `json:"model"`
	// Messages is the conversation so far.
	Messages []ChatMessage `json:"messages"`
	// Stream requests an SSE stream of chunks instead of a single response.
	// CreateStream sets this automatically; setting it for Create has no effect.
	Stream bool `json:"stream,omitempty"`
	// MaxTokens caps the number of generated tokens.
	MaxTokens *int `json:"max_tokens,omitempty"`
	// Temperature is the sampling temperature in [0, 2].
	Temperature *float64 `json:"temperature,omitempty"`
	// TopP is the nucleus sampling probability in [0, 1].
	TopP *float64 `json:"top_p,omitempty"`
	// ReasoningEffort sets the reasoning budget for reasoning-capable models.
	ReasoningEffort *ReasoningEffort `json:"reasoning_effort,omitempty"`
	// Tools are the tool definitions the model may call.
	Tools []Tool `json:"tools,omitempty"`
	// ToolChoice constrains tool calling.
	ToolChoice *ToolChoice `json:"tool_choice,omitempty"`
	// ResponseFormat is an OpenAI-compatible response format object.
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	// N is the number of completions to generate.
	N *int `json:"n,omitempty"`
	// PresencePenalty penalizes token presence.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`
	// FrequencyPenalty penalizes token frequency.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	// LogitBias is a per-token logit bias map.
	LogitBias json.RawMessage `json:"logit_bias,omitempty"`
	// User is an end-user identifier for abuse monitoring.
	User *string `json:"user,omitempty"`
	// Seed requests best-effort determinism.
	Seed *int64 `json:"seed,omitempty"`
	// Stop sets one or more stop sequences.
	Stop *StopSequence `json:"stop,omitempty"`
}

// ChatCompletionResponse is a non-streamed chat completion.
type ChatCompletionResponse struct {
	// ID is the unique completion identifier.
	ID string `json:"id"`
	// Object is the object type; always "chat.completion".
	Object string `json:"object"`
	// Created is the Unix epoch seconds at creation.
	Created int64 `json:"created"`
	// Model is the model that produced the completion.
	Model string `json:"model"`
	// Choices holds one entry per generated choice.
	Choices []Choice `json:"choices"`
	// Usage is the token accounting.
	Usage Usage `json:"usage"`
}

// Choice is one choice within a [ChatCompletionResponse].
type Choice struct {
	// Index is the position of this choice in the list.
	Index int `json:"index"`
	// Message is the generated message.
	Message ResponseMessage `json:"message"`
	// FinishReason is why generation stopped.
	FinishReason FinishReason `json:"finish_reason"`
}

// ResponseMessage is the assistant message inside a [Choice].
type ResponseMessage struct {
	// Role is the author role; always [RoleAssistant].
	Role Role `json:"role"`
	// Content is the generated text.
	Content MessageContent `json:"content"`
	// ReasoningContent is the reasoning trace, for reasoning-capable models.
	ReasoningContent string `json:"reasoning_content,omitempty"`
	// ToolCalls are tool calls requested by the model.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatCompletionChunk is one event in a streamed completion.
type ChatCompletionChunk struct {
	// ID is the completion identifier, stable across the stream.
	ID string `json:"id"`
	// Object is the object type; always "chat.completion.chunk".
	Object string `json:"object"`
	// Created is the Unix epoch seconds at creation.
	Created int64 `json:"created"`
	// Model is the model producing the stream.
	Model string `json:"model"`
	// Choices holds the incremental choices.
	Choices []ChunkChoice `json:"choices"`
	// Usage is the cumulative token accounting, present on the final chunk(s).
	Usage *Usage `json:"usage,omitempty"`
}

// ContentDelta returns the text fragment of the first choice and whether one
// was present. It is a convenience for the common single-choice streaming loop.
func (c *ChatCompletionChunk) ContentDelta() (string, bool) {
	if len(c.Choices) == 0 {
		return "", false
	}
	d := c.Choices[0].Delta.Content
	if d == "" {
		return "", false
	}
	return d, true
}

// ChunkChoice is one choice within a [ChatCompletionChunk].
type ChunkChoice struct {
	// Index is the position of this choice in the list.
	Index int `json:"index"`
	// Delta is the incremental payload for this choice.
	Delta Delta `json:"delta"`
	// FinishReason is why generation stopped, on the final chunk for this choice.
	FinishReason *FinishReason `json:"finish_reason,omitempty"`
}

// Delta is the incremental payload of a [ChunkChoice].
type Delta struct {
	// Role is the author role, present on the first delta.
	Role Role `json:"role,omitempty"`
	// Content is a text fragment.
	Content string `json:"content,omitempty"`
	// ReasoningContent is a reasoning-trace fragment.
	ReasoningContent string `json:"reasoning_content,omitempty"`
	// ToolCalls are incremental tool-call fragments.
	ToolCalls []ToolCallDelta `json:"tool_calls,omitempty"`
}

// ToolCallDelta is an incremental tool-call fragment within a [Delta].
type ToolCallDelta struct {
	// Index is the index of the tool call being assembled.
	Index int `json:"index"`
	// ID is the tool-call id, present on the first fragment.
	ID string `json:"id,omitempty"`
	// Type is the discriminator, present on the first fragment.
	Type string `json:"type,omitempty"`
	// Function carries function name and argument fragments.
	Function *FunctionCallDelta `json:"function,omitempty"`
}

// FunctionCallDelta is an incremental function name/arguments within a
// [ToolCallDelta].
type FunctionCallDelta struct {
	// Name is the function name, present on the first fragment.
	Name string `json:"name,omitempty"`
	// Arguments is an argument-string fragment to be concatenated.
	Arguments string `json:"arguments,omitempty"`
}
