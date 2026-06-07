package skailar

import (
	"context"
	"io"
)

// AudioService is the audio resource, accessed as Client.Audio.
type AudioService struct {
	// Transcriptions is the speech-to-text sub-resource.
	Transcriptions *TranscriptionsService
	// Speech is the text-to-speech sub-resource.
	Speech *SpeechService

	client *Client
}

// TranscriptionsService is the transcription resource, accessed as
// Client.Audio.Transcriptions.
type TranscriptionsService struct {
	client *Client
}

// Create transcribes base64-encoded audio to text. This is a billable,
// side-effecting call and is not retried on 5xx responses.
func (s *TranscriptionsService) Create(ctx context.Context, req TranscriptionRequest) (*TranscriptionResponse, error) {
	var out TranscriptionResponse
	if err := s.client.postJSON(ctx, "v1/audio/transcriptions", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SpeechService is the speech-synthesis resource, accessed as
// Client.Audio.Speech.
type SpeechService struct {
	client *Client
}

// Create synthesizes speech and returns a stream of MP3 (audio/mpeg) bytes. The
// caller owns closing the returned [io.ReadCloser].
//
// Use [SpeechService.CreateBytes] to collect the whole clip into memory
// instead. This is a billable, side-effecting call and is not retried on 5xx
// responses.
func (s *SpeechService) Create(ctx context.Context, req SpeechRequest) (io.ReadCloser, error) {
	return s.client.postBinary(ctx, "v1/audio/speech", req, "audio/mpeg")
}

// CreateBytes synthesizes speech and collects the full MP3 clip into a byte
// slice. It closes the underlying stream before returning.
func (s *SpeechService) CreateBytes(ctx context.Context, req SpeechRequest) ([]byte, error) {
	body, err := s.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, transportError(err)
	}
	return data, nil
}
