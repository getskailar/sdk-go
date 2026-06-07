package skailar

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
)

const doneSentinel = "[DONE]"

// ChatCompletionStream is an iterator over the [ChatCompletionChunk]s of a
// streamed completion. It follows the standard Go iterator shape used by
// bufio.Scanner and database/sql.Rows:
//
//	defer stream.Close()
//	for stream.Next() {
//		chunk := stream.Current()
//		// ...
//	}
//	if err := stream.Err(); err != nil {
//		// ...
//	}
//
// Next reports whether another chunk is available, Current returns it, and Err
// reports a terminal error (nil on clean end-of-stream). Close releases the
// underlying connection and must be called.
//
// A ChatCompletionStream is not safe for concurrent use.
type ChatCompletionStream struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	current *ChatCompletionChunk
	err     error
	done    bool
	closed  bool
}

// newChatCompletionStream wraps a streaming response body.
func newChatCompletionStream(body io.ReadCloser) *ChatCompletionStream {
	scanner := bufio.NewScanner(body)
	scanner.Split(scanSSELines)
	// SSE data lines (a chunk's JSON) can be large; raise the token cap well
	// above the default 64 KiB.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &ChatCompletionStream{body: body, scanner: scanner}
}

// Next advances to the next chunk. It returns false at end of stream or on
// error; check [ChatCompletionStream.Err] afterwards to distinguish the two.
func (s *ChatCompletionStream) Next() bool {
	if s.done {
		return false
	}
	for s.scanner.Scan() {
		payload, ok := dataPayload(s.scanner.Bytes())
		if !ok {
			continue
		}
		if string(payload) == doneSentinel {
			s.finish(nil)
			return false
		}
		chunk, err := decodeChunk(payload)
		if err != nil {
			s.finish(err)
			return false
		}
		s.current = chunk
		return true
	}
	if err := s.scanner.Err(); err != nil {
		s.finish(transportError(err))
		return false
	}
	s.finish(nil)
	return false
}

// Current returns the chunk produced by the most recent [ChatCompletionStream.Next]
// that returned true. It is nil before the first such call.
func (s *ChatCompletionStream) Current() *ChatCompletionChunk { return s.current }

// Err returns the terminal error, if any. It is nil on a clean end of stream.
func (s *ChatCompletionStream) Err() error { return s.err }

// Close releases the underlying HTTP connection. It is safe to call more than
// once and safe to call before the stream is fully consumed, which cancels the
// in-flight body.
func (s *ChatCompletionStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	s.done = true
	return s.body.Close()
}

// finish records a terminal state and marks the stream done.
func (s *ChatCompletionStream) finish(err error) {
	s.err = err
	s.done = true
	s.current = nil
}

// decodeChunk parses a data-line payload into a chunk, surfacing an in-band
// error frame as a [*Error] rather than a chunk.
func decodeChunk(payload []byte) (*ChatCompletionChunk, error) {
	var probe struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(payload, &probe); err != nil {
		return nil, &Error{Kind: KindDecode, Message: "malformed streaming event", Raw: append([]byte(nil), payload...), cause: err}
	}
	if len(probe.Error) > 0 {
		code, message := parseErrorFields(payload)
		if message == "" {
			message = "streaming error"
		}
		return nil, &Error{
			Kind:    KindUpstream,
			Status:  500,
			Code:    code,
			Message: message,
			Raw:     append([]byte(nil), payload...),
		}
	}
	var chunk ChatCompletionChunk
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return nil, &Error{Kind: KindDecode, Message: "malformed streaming event", Raw: append([]byte(nil), payload...), cause: err}
	}
	return &chunk, nil
}

// dataPayload extracts the payload of a "data:" SSE line, returning false for
// blank lines, comments, and other SSE fields.
func dataPayload(line []byte) ([]byte, bool) {
	trimmed := bytes.TrimRight(line, " \t")
	if len(trimmed) == 0 || trimmed[0] == ':' {
		return nil, false
	}
	rest, ok := bytes.CutPrefix(trimmed, []byte("data:"))
	if !ok {
		return nil, false
	}
	rest = bytes.TrimPrefix(rest, []byte(" "))
	return rest, true
}

// scanSSELines is a [bufio.SplitFunc] that splits on any of the three SSE line
// terminators (\n, \r\n, \r), returning lines without their terminator.
func scanSSELines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		switch data[i] {
		case '\n':
			return i + 1, data[:i], nil
		case '\r':
			// A lone trailing \r at a non-EOF boundary might be the first half
			// of a \r\n straddling the buffer; ask for more unless at EOF.
			if i+1 < len(data) {
				if data[i+1] == '\n' {
					return i + 2, data[:i], nil
				}
				return i + 1, data[:i], nil
			}
			if atEOF {
				return i + 1, data[:i], nil
			}
			return 0, nil, nil
		}
	}

	if atEOF {
		return len(data), dropTrailingCR(data), nil
	}
	return 0, nil, nil
}

// dropTrailingCR removes a single trailing carriage return.
func dropTrailingCR(data []byte) []byte {
	if n := len(data); n > 0 && data[n-1] == '\r' {
		return data[:n-1]
	}
	return data
}
