package skailar

import (
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// Version is the SDK version, reported in the User-Agent header.
	Version = "0.0.1"

	defaultBaseURL = "https://api.skailar.com"
	defaultTimeout = 60 * time.Second
	defaultRetries = 2

	envAPIKey  = "SKAILAR_API_KEY"
	envBaseURL = "SKAILAR_BASE_URL"

	userAgent = "skailar-go/" + Version
)

// Client is the entry point to the Skailar API. Construct it once with
// [NewClient] and reuse it; it is safe for concurrent use by multiple
// goroutines.
//
// The resource handles ([Client.Chat], [Client.Models], [Client.Images],
// [Client.Audio], [Client.Uploads]) are accessed as fields.
type Client struct {
	apiKey         string
	baseURL        string
	httpClient     *http.Client
	timeout        time.Duration
	maxRetries     int
	defaultHeaders http.Header

	// Chat is the chat-completions resource.
	Chat *ChatService
	// Models is the model-catalog resource.
	Models *ModelsService
	// Images is the image-generation resource.
	Images *ImagesService
	// Audio is the speech and transcription resource.
	Audio *AudioService
	// Uploads is the storage-uploads resource.
	Uploads *UploadsService
}

// NewClient constructs a [Client] from the given options.
//
// With no options it reads the API key from SKAILAR_API_KEY and the base URL
// from SKAILAR_BASE_URL (falling back to https://api.skailar.com). It returns a
// [*Error] of [KindConfig] if no API key is available.
func NewClient(opts ...Option) (*Client, error) {
	cfg := clientConfig{
		timeout:    defaultTimeout,
		maxRetries: defaultRetries,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	apiKey := cfg.apiKey
	if apiKey == "" {
		apiKey = os.Getenv(envAPIKey)
	}
	if apiKey == "" {
		return nil, newConfigError("missing API key (pass WithAPIKey or set " + envAPIKey + ")")
	}

	baseURL := cfg.baseURL
	if baseURL == "" {
		baseURL = os.Getenv(envBaseURL)
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	c := &Client{
		apiKey:         apiKey,
		baseURL:        baseURL,
		httpClient:     httpClient,
		timeout:        cfg.timeout,
		maxRetries:     cfg.maxRetries,
		defaultHeaders: cfg.defaultHeaders,
	}

	c.Chat = &ChatService{client: c}
	c.Chat.Completions = &ChatCompletionsService{client: c}
	c.Models = &ModelsService{client: c}
	c.Images = &ImagesService{client: c}
	c.Audio = &AudioService{client: c}
	c.Audio.Transcriptions = &TranscriptionsService{client: c}
	c.Audio.Speech = &SpeechService{client: c}
	c.Uploads = &UploadsService{client: c}
	c.Uploads.Images = &ImageUploadsService{client: c}
	c.Uploads.Files = &FileUploadsService{client: c}

	return c, nil
}

// endpoint joins the base URL with a request path, avoiding a double slash.
func (c *Client) endpoint(path string) string {
	return c.baseURL + "/" + strings.TrimLeft(path, "/")
}
