package skailar

// Mime is the MIME type of an audio clip submitted for transcription.
type Mime string

const (
	// MimeWav is audio/wav.
	MimeWav Mime = "audio/wav"
	// MimeWebm is audio/webm.
	MimeWebm Mime = "audio/webm"
	// MimeMp4 is audio/mp4.
	MimeMp4 Mime = "audio/mp4"
	// MimeM4a is audio/m4a.
	MimeM4a Mime = "audio/m4a"
	// MimeMpeg is audio/mpeg.
	MimeMpeg Mime = "audio/mpeg"
	// MimeMp3 is audio/mp3.
	MimeMp3 Mime = "audio/mp3"
)

// TranscriptionRequest is a request to transcribe audio to text.
type TranscriptionRequest struct {
	// Base64 is the base64-encoded audio bytes, without a data: prefix.
	Base64 string `json:"base64"`
	// Mime is the audio MIME type; defaults to audio/wav server-side when empty.
	Mime Mime `json:"mime,omitempty"`
}

// TranscriptionResponse is the response of [TranscriptionsService.Create].
type TranscriptionResponse struct {
	// Text is the transcribed text.
	Text string `json:"text"`
}

// Voice is a synthesis voice for [SpeechService.Create].
type Voice string

const (
	// VoiceAlloy is the "alloy" voice.
	VoiceAlloy Voice = "alloy"
	// VoiceAsh is the "ash" voice.
	VoiceAsh Voice = "ash"
	// VoiceBallad is the "ballad" voice.
	VoiceBallad Voice = "ballad"
	// VoiceCoral is the "coral" voice.
	VoiceCoral Voice = "coral"
	// VoiceEcho is the "echo" voice.
	VoiceEcho Voice = "echo"
	// VoiceFable is the "fable" voice.
	VoiceFable Voice = "fable"
	// VoiceNova is the "nova" voice (the default).
	VoiceNova Voice = "nova"
	// VoiceOnyx is the "onyx" voice.
	VoiceOnyx Voice = "onyx"
	// VoiceSage is the "sage" voice.
	VoiceSage Voice = "sage"
	// VoiceShimmer is the "shimmer" voice.
	VoiceShimmer Voice = "shimmer"
)

// SpeechRequest is a request to synthesize speech from text.
type SpeechRequest struct {
	// Input is the text to synthesize, up to 4000 characters.
	Input string `json:"input"`
	// Voice is the synthesis voice; defaults to nova server-side when empty.
	Voice Voice `json:"voice,omitempty"`
}
