package skailar

// Known model identifiers, provided for autocomplete. Any string is a valid
// model; these constants are not exhaustive and the catalog may change. Call
// [ModelsService.List] for the live list.
const (
	// Claude.
	ModelClaudeOpus4_8   = "claude-opus-4-8"
	ModelClaudeOpus4_7   = "claude-opus-4-7"
	ModelClaudeOpus4_6   = "claude-opus-4-6"
	ModelClaudeSonnet4_6 = "claude-sonnet-4-6"
	ModelClaudeSonnet4_5 = "claude-sonnet-4-5"
	ModelClaudeHaiku4_5  = "claude-haiku-4-5"

	// GPT.
	ModelGPT5_5     = "gpt-5.5"
	ModelGPT5_4     = "gpt-5.4"
	ModelGPT5_4Mini = "gpt-5.4-mini"
	ModelGPT5_4Nano = "gpt-5.4-nano"
	ModelGPT5_1     = "gpt-5.1"
	ModelGPT5       = "gpt-5"
	ModelGPT5Mini   = "gpt-5-mini"

	// OpenAI reasoning.
	ModelO3     = "o3"
	ModelO4Mini = "o4-mini"

	// Gemini.
	ModelGemini3_5Flash      = "gemini-3.5-flash"
	ModelGemini3_1ProPreview = "gemini-3.1-pro-preview"
	ModelGemini3FlashPreview = "gemini-3-flash-preview"
	ModelGemini2_5Pro        = "gemini-2.5-pro"
	ModelGemini2_5Flash      = "gemini-2.5-flash"
	ModelGemini2_5FlashLite  = "gemini-2.5-flash-lite"

	// DeepSeek.
	ModelDeepSeekV4Pro   = "deepseek-v4-pro"
	ModelDeepSeekV4Flash = "deepseek-v4-flash"

	// Grok.
	ModelGrok4_3           = "grok-4.3"
	ModelGrok4_20Reasoning = "grok-4.20-reasoning"
	ModelGrok4_20NonReason = "grok-4.20-non-reasoning"
	ModelGrokBuild0_1      = "grok-build-0.1"

	// Image.
	ModelGPTImage1 = "gpt-image-1"
)
