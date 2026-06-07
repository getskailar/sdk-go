# Changelog

All notable changes to this project are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.1] - 2026-06-07

Initial pre-release of the official Skailar SDK for Go.

### Added

- `Client` with functional options: `WithAPIKey`, `WithBaseURL`, `WithTimeout`,
  `WithMaxRetries`, `WithHTTPClient`, `WithDefaultHeader`.
- Chat completions: `client.Chat.Completions.Create` (JSON) and
  `CreateStream` (Server-Sent Events).
- Model catalog: `client.Models.List` and `client.Models.Retrieve`.
- Image generation: `client.Images.Generate`.
- Audio: `client.Audio.Transcriptions.Create` and `client.Audio.Speech.Create`
  (returns an `io.ReadCloser` of `audio/mpeg` bytes).
- Storage uploads: `client.Uploads.Images.Create` and
  `client.Uploads.Files.Create`.
- Key verification: `client.Ping`.
- Single concrete `*Error` type with a `Kind` discriminant and sentinel values
  (`ErrAuth`, `ErrRateLimit`, `ErrUpstream`, `ErrNetwork`, `ErrTimeout`,
  `ErrNotFound`, `ErrBadRequest`) for `errors.Is` / `errors.As`.
- `ChatCompletionStream` with the idiomatic `Next` / `Current` / `Err` /
  `Close` iterator shape and a dependency-free SSE parser.
- Known model-id constants in `modelids.go` and a `Ptr` helper for optional
  fields.

### Fixed

Shipped already-corrected for bugs fixed in the TypeScript SDK across
0.0.1–0.0.5, so the Go SDK never regressed through them:

- The retry loop honours `ctx.Done()` at every iteration and waits on an
  interruptible timer; a cancelled context returns immediately as
  `&Error{Kind: KindAborted}` without leaking goroutines or timers.
- Internal timeouts are reported as `KindTimeout`, distinct from `KindNetwork`
  for other transport failures.
- Early exit from a stream (`stream.Close()`) closes the underlying HTTP body
  instead of leaking the connection.
- The `Authorization` header cannot be overridden by `WithDefaultHeader` or
  per-call headers; conflicting keys are dropped case-insensitively before the
  bearer token is applied.
- Side-effecting `POST` requests (chat completions, image generation, speech,
  transcription, uploads) are never retried on `5xx`, to avoid double billing.
  Only idempotent `GET` requests are retried on `5xx`.
- `Retry-After` is capped at 60 seconds; the uncapped server value is still
  exposed on `Error.RetryAfter`.
- The SSE parser accepts all three line terminators (`\n`, `\r\n`, `\r`).

[Unreleased]: https://github.com/getskailar/sdk-go/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/getskailar/sdk-go/releases/tag/v0.0.1
