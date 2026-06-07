package skailar

// ModelCapabilities reports which features a model supports.
type ModelCapabilities struct {
	// Streaming reports whether the model supports SSE streaming.
	Streaming bool `json:"streaming"`
	// ToolCalls reports whether the model supports function calling.
	ToolCalls bool `json:"tool_calls"`
	// Vision reports whether the model accepts image input.
	Vision bool `json:"vision"`
	// JSONMode reports whether the model supports JSON-constrained output.
	JSONMode bool `json:"json_mode"`
	// Reasoning reports whether the model exposes a reasoning trace; nil when
	// unknown.
	Reasoning *bool `json:"reasoning,omitempty"`
}

// ModelPricing reports per-million-token pricing for a model.
type ModelPricing struct {
	// InputPerMTok is the price per million input tokens.
	InputPerMTok float64 `json:"input_per_mtok"`
	// OutputPerMTok is the price per million output tokens.
	OutputPerMTok float64 `json:"output_per_mtok"`
	// Currency is the ISO 4217 currency code, for example "USD".
	Currency string `json:"currency"`
}

// ModelSummary is the catalog entry for a model, as returned by
// [ModelsService.List].
type ModelSummary struct {
	// ID is the model identifier.
	ID string `json:"id"`
	// Object is the object type; always "model".
	Object string `json:"object"`
	// Created is the Unix epoch seconds at registration.
	Created int64 `json:"created"`
	// OwnedBy is the provider: skailar, anthropic, openai, google, deepseek, xai.
	OwnedBy string `json:"owned_by"`
	// DisplayName is the human-readable model name.
	DisplayName string `json:"display_name"`
	// ContextWindow is the maximum context length in tokens.
	ContextWindow int `json:"context_window"`
	// MaxOutputTokens is the maximum number of tokens the model can generate.
	MaxOutputTokens int `json:"max_output_tokens"`
	// Capabilities reports supported features.
	Capabilities ModelCapabilities `json:"capabilities"`
	// Pricing reports per-million-token pricing.
	Pricing ModelPricing `json:"pricing"`
	// Status is the lifecycle state: "active", "preview", or "deprecated".
	Status string `json:"status"`
}

// Modalities reports the input and output modalities of a model.
type Modalities struct {
	// Input lists accepted input modalities, for example "text", "image".
	Input []string `json:"input,omitempty"`
	// Output lists produced output modalities.
	Output []string `json:"output,omitempty"`
}

// Model is the full detail card for a model, as returned by
// [ModelsService.Retrieve]. It embeds [ModelSummary] and adds detail fields.
type Model struct {
	ModelSummary
	// Description is a human-readable description.
	Description string `json:"description,omitempty"`
	// Modalities reports input and output modalities.
	Modalities *Modalities `json:"modalities,omitempty"`
	// SupportedParameters lists request parameters the model honours.
	SupportedParameters []string `json:"supported_parameters,omitempty"`
	// KnowledgeCutoff is the training-data cutoff, as a date string.
	KnowledgeCutoff string `json:"knowledge_cutoff,omitempty"`
	// ReleasedAt is the release date, as a date string.
	ReleasedAt string `json:"released_at,omitempty"`
	// DocumentationURL links to the model's documentation.
	DocumentationURL string `json:"documentation_url,omitempty"`
	// Aliases lists alternative identifiers that resolve to this model.
	Aliases []string `json:"aliases,omitempty"`
}

// ModelList is the response of [ModelsService.List].
type ModelList struct {
	// Object is the object type; always "list".
	Object string `json:"object"`
	// Data is the flat list of models.
	Data []ModelSummary `json:"data"`
}
