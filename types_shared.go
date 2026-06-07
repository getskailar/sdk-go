package skailar

import "encoding/json"

// Ptr returns a pointer to v. It is a convenience for setting optional,
// pointer-typed request fields:
//
//	req.Temperature = skailar.Ptr(0.7)
func Ptr[T any](v T) *T { return &v }

// Usage reports token accounting for a completion.
type Usage struct {
	// PromptTokens is the number of tokens in the prompt.
	PromptTokens int `json:"prompt_tokens"`
	// CompletionTokens is the number of generated tokens.
	CompletionTokens int `json:"completion_tokens"`
	// TotalTokens is the sum of prompt and completion tokens.
	TotalTokens int `json:"total_tokens"`
}

// Tool is a function-calling tool definition, OpenAI-compatible.
type Tool struct {
	// Type is the tool type; always "function".
	Type string `json:"type"`
	// Function describes the callable function.
	Function FunctionDefinition `json:"function"`
}

// NewFunctionTool builds a [Tool] of type "function" from a name, description,
// and JSON Schema parameter object.
func NewFunctionTool(name, description string, parameters json.RawMessage) Tool {
	return Tool{
		Type: "function",
		Function: FunctionDefinition{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

// FunctionDefinition describes a callable function in a [Tool].
type FunctionDefinition struct {
	// Name is the function name.
	Name string `json:"name"`
	// Description is an optional human-readable description.
	Description string `json:"description,omitempty"`
	// Parameters is a JSON Schema object describing the arguments.
	Parameters json.RawMessage `json:"parameters,omitempty"`
}

// ToolCall is a function call requested by the model.
type ToolCall struct {
	// ID identifies this tool call, for pairing with a tool-result message.
	ID string `json:"id"`
	// Type is the tool type; always "function".
	Type string `json:"type"`
	// Function carries the called function name and arguments.
	Function FunctionCall `json:"function"`
}

// FunctionCall is the function name and arguments within a [ToolCall].
type FunctionCall struct {
	// Name is the called function name.
	Name string `json:"name"`
	// Arguments is the JSON-encoded arguments string, as produced by the model.
	Arguments string `json:"arguments"`
}

// ToolChoice constrains how the model may call tools. Use [ToolChoiceAuto],
// [ToolChoiceNone], [ToolChoiceRequired], or [NamedToolChoice].
//
// It marshals either as a bare string ("auto", "none", "required") or as a
// named-function object.
type ToolChoice struct {
	// Mode is the string form: "auto", "none", or "required". When empty, Name
	// selects a specific function instead.
	Mode string
	// Name, when set, forces the model to call the named function.
	Name string
}

// ToolChoiceAuto lets the model decide whether to call a tool.
func ToolChoiceAuto() *ToolChoice { return &ToolChoice{Mode: "auto"} }

// ToolChoiceNone forbids tool calls.
func ToolChoiceNone() *ToolChoice { return &ToolChoice{Mode: "none"} }

// ToolChoiceRequired forces the model to call at least one tool.
func ToolChoiceRequired() *ToolChoice { return &ToolChoice{Mode: "required"} }

// NamedToolChoice forces the model to call the named function.
func NamedToolChoice(name string) *ToolChoice { return &ToolChoice{Name: name} }

// MarshalJSON implements [json.Marshaler].
func (t ToolChoice) MarshalJSON() ([]byte, error) {
	if t.Name != "" {
		return json.Marshal(map[string]any{
			"type":     "function",
			"function": map[string]string{"name": t.Name},
		})
	}
	mode := t.Mode
	if mode == "" {
		mode = "auto"
	}
	return json.Marshal(mode)
}

// UnmarshalJSON implements [json.Unmarshaler].
func (t *ToolChoice) UnmarshalJSON(data []byte) error {
	var mode string
	if err := json.Unmarshal(data, &mode); err == nil {
		t.Mode = mode
		t.Name = ""
		return nil
	}
	var named struct {
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := json.Unmarshal(data, &named); err != nil {
		return err
	}
	t.Mode = ""
	t.Name = named.Function.Name
	return nil
}
