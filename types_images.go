package skailar

// ImageGenerationRequest is a request to generate images from a prompt.
type ImageGenerationRequest struct {
	// Model is the image model identifier, for example "gpt-image-1".
	Model string `json:"model"`
	// Prompt is the text description of the desired image.
	Prompt string `json:"prompt"`
	// N is the number of images to generate (1-10).
	N *int `json:"n,omitempty"`
	// Size is the image size, for example "1024x1024", "1024x1792", "1792x1024".
	Size string `json:"size,omitempty"`
	// Quality is a provider-specific quality hint, for example "standard", "hd".
	Quality string `json:"quality,omitempty"`
	// Background is a provider-specific background hint, for example
	// "transparent".
	Background string `json:"background,omitempty"`
}

// GeneratedImage is one image within an [ImageGenerationResponse].
type GeneratedImage struct {
	// URL is the hosted image URL, when the provider returns a URL.
	URL string `json:"url,omitempty"`
	// B64JSON is the base64-encoded image bytes, when the provider returns
	// inline data.
	B64JSON string `json:"b64_json,omitempty"`
	// RevisedPrompt is the prompt as revised by the provider, when present.
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImageGenerationResponse is the response of [ImagesService.Generate].
type ImageGenerationResponse struct {
	// Created is the Unix epoch seconds at generation.
	Created int64 `json:"created"`
	// Data holds the generated images.
	Data []GeneratedImage `json:"data"`
}
