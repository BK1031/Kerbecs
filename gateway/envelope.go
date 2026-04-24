package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"kerbecs/model"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const timestampFormat = "Mon Jan 02 15:04:05 MST 2006"

// envelopeFromBody wraps an upstream response body in the Kerbecs envelope.
// If body parses as JSON it is embedded as-is; otherwise it is wrapped as
// {"message": <body>}.
func envelopeFromBody(code int, gateway, service string, start time.Time, body []byte) ([]byte, error) {
	resp := model.Response{
		Status:    statusFromCode(code),
		Ping:      strconv.FormatInt(time.Since(start).Milliseconds(), 10) + "ms",
		Gateway:   gateway,
		Service:   service,
		Timestamp: time.Now().Format(timestampFormat),
	}
	if len(body) > 0 {
		if json.Valid(body) {
			resp.Data = json.RawMessage(body)
		} else {
			wrapped, err := json.Marshal(map[string]string{"message": string(body)})
			if err != nil {
				return nil, err
			}
			resp.Data = json.RawMessage(wrapped)
		}
	}
	return json.Marshal(resp)
}

// envelopeFromMessage wraps a plain string message in the Kerbecs envelope.
func envelopeFromMessage(code int, gateway, service string, start time.Time, message string) ([]byte, error) {
	wrapped, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		return nil, err
	}
	resp := model.Response{
		Status:    statusFromCode(code),
		Ping:      strconv.FormatInt(time.Since(start).Milliseconds(), 10) + "ms",
		Gateway:   gateway,
		Service:   service,
		Timestamp: time.Now().Format(timestampFormat),
		Data:      json.RawMessage(wrapped),
	}
	return json.Marshal(resp)
}

func statusFromCode(code int) string {
	switch {
	case code < 200:
		return "INFO"
	case code < 300:
		return "SUCCESS"
	case code < 400:
		return "REDIRECT"
	default:
		return "ERROR"
	}
}

// isBinaryContent returns true for content types that should not be buffered
// and rewritten by the envelope.
func isBinaryContent(ct string) bool {
	for _, p := range []string{
		"application/octet-stream",
		"application/pdf",
		"application/zip",
		"text/csv",
	} {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	return false
}

// isStreamingContent returns true for content types that must stream
// (event-by-event or frame-by-frame) and would break if buffered whole.
func isStreamingContent(ct string) bool {
	for _, p := range []string{
		"text/event-stream",
		"application/grpc",
	} {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	return false
}

// modifyResponseWithEnvelope returns an httputil.ReverseProxy ModifyResponse
// hook that buffers the upstream response and rewrites it as a Kerbecs
// envelope. WebSocket upgrades and binary content are passed through
// untouched.
func modifyResponseWithEnvelope(gateway, service string, start time.Time) func(*http.Response) error {
	return func(resp *http.Response) error {
		if resp.StatusCode == http.StatusSwitchingProtocols {
			return nil
		}
		ct := resp.Header.Get("Content-Type")
		if isBinaryContent(ct) || isStreamingContent(ct) {
			return nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if err := resp.Body.Close(); err != nil {
			return err
		}

		out, err := envelopeFromBody(resp.StatusCode, gateway, service, start, body)
		if err != nil {
			return err
		}

		resp.Body = io.NopCloser(bytes.NewReader(out))
		resp.ContentLength = int64(len(out))
		resp.Header.Set("Content-Length", strconv.Itoa(len(out)))
		resp.Header.Set("Content-Type", "application/json")
		return nil
	}
}
