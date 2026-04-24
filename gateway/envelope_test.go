package gateway

import (
	"encoding/json"
	"kerbecs/model"
	"strings"
	"testing"
	"time"
)

func TestEnvelopeFromBody_JSON(t *testing.T) {
	start := time.Now().Add(-10 * time.Millisecond)
	body := []byte(`{"id":42,"name":"alice"}`)
	out, err := envelopeFromBody(200, "kerbecs:v3.0.0", "users:v1.0.0", start, body)
	if err != nil {
		t.Fatal(err)
	}

	var r model.Response
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Status != "SUCCESS" {
		t.Errorf("status: want SUCCESS, got %q", r.Status)
	}
	if r.Gateway != "kerbecs:v3.0.0" {
		t.Errorf("gateway: %q", r.Gateway)
	}
	if r.Service != "users:v1.0.0" {
		t.Errorf("service: %q", r.Service)
	}

	var data map[string]any
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("data unmarshal: %v", err)
	}
	if data["name"] != "alice" {
		t.Errorf("data: %+v", data)
	}
}

func TestEnvelopeFromBody_NonJSON(t *testing.T) {
	body := []byte(`plain string with "quotes" and \backslash`)
	out, err := envelopeFromBody(500, "kerbecs", "users", time.Now(), body)
	if err != nil {
		t.Fatal(err)
	}

	var r model.Response
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Status != "ERROR" {
		t.Errorf("status: want ERROR, got %q", r.Status)
	}
	var wrapped map[string]string
	if err := json.Unmarshal(r.Data, &wrapped); err != nil {
		t.Fatalf("data unmarshal: %v", err)
	}
	if wrapped["message"] != string(body) {
		t.Errorf("message: want exact, got %q", wrapped["message"])
	}
}

func TestStatusFromCode(t *testing.T) {
	cases := map[int]string{
		100: "INFO", 199: "INFO",
		200: "SUCCESS", 299: "SUCCESS",
		301: "REDIRECT", 399: "REDIRECT",
		400: "ERROR", 404: "ERROR", 500: "ERROR", 503: "ERROR",
	}
	for code, want := range cases {
		if got := statusFromCode(code); got != want {
			t.Errorf("statusFromCode(%d) = %q, want %q", code, got, want)
		}
	}
}

func TestIsBinaryContent(t *testing.T) {
	yes := []string{
		"application/octet-stream",
		"application/octet-stream; name=foo",
		"application/pdf",
		"application/zip",
		"text/csv",
		"text/csv; charset=utf-8",
	}
	no := []string{
		"application/json",
		"text/html",
		"text/plain",
		"",
	}
	for _, ct := range yes {
		if !isBinaryContent(ct) {
			t.Errorf("want binary: %q", ct)
		}
	}
	for _, ct := range no {
		if isBinaryContent(ct) {
			t.Errorf("want non-binary: %q", ct)
		}
	}
}

func TestEnvelopeFromMessage_HandlesTrickyChars(t *testing.T) {
	msg := `broke at path: "/users/\n/123"`
	out, err := envelopeFromMessage(502, "kerbecs", "users", time.Now(), msg)
	if err != nil {
		t.Fatal(err)
	}
	var r model.Response
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("envelope should be valid JSON even with tricky chars: %v", err)
	}
	var wrapped map[string]string
	if err := json.Unmarshal(r.Data, &wrapped); err != nil {
		t.Fatal(err)
	}
	if wrapped["message"] != msg {
		t.Errorf("message round-trip: %q", wrapped["message"])
	}
	if !strings.Contains(string(out), `"status":"ERROR"`) {
		t.Errorf("expected ERROR status in %q", out)
	}
}
