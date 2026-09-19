package event_runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

// TestRedact_TranscriptIsReplaced: emit envelope with payload.transcript;
// assert output contains the redaction tag and NOT the original transcript.
func TestRedact_TranscriptIsReplaced(t *testing.T) {
	env := NewEnvelope("voice.transcribed", "audio-123")
	env.Payload = json.RawMessage(`{"transcript":"hello world","lang":"en"}`)

	out := Redact(*env)

	if strings.Contains(string(out.Payload), "hello world") {
		t.Errorf("payload ainda contém 'hello world'; payload=%s", out.Payload)
	}
	if !strings.Contains(string(out.Payload), "[REDACTED:transcript]") {
		t.Errorf("payload não contém tag de redação; payload=%s", out.Payload)
	}
	if !strings.Contains(string(out.Payload), `"lang":"en"`) {
		t.Errorf("campo não-PII foi removido indevidamente; payload=%s", out.Payload)
	}
	if got := out.Provenance.RedactedFields; len(got) == 0 || got[0] != "transcript" {
		t.Errorf("RedactedFields=%v; esperado ['transcript']", got)
	}
}

// TestRedact_AudioURLIsReplaced: same for payload.audio_url.
func TestRedact_AudioURLIsReplaced(t *testing.T) {
	env := NewEnvelope("voice.uploaded", "audio-123")
	env.Payload = json.RawMessage(`{"audio_url":"https://secret.example.com/audio/abc","size":12345}`)

	out := Redact(*env)

	if strings.Contains(string(out.Payload), "secret.example.com") {
		t.Errorf("payload ainda contém a URL privada; payload=%s", out.Payload)
	}
	if !strings.Contains(string(out.Payload), "[REDACTED:audio_url]") {
		t.Errorf("payload não contém tag de redação; payload=%s", out.Payload)
	}
	if !strings.Contains(string(out.Payload), `"size":12345`) {
		t.Errorf("campo não-PII foi removido indevidamente; payload=%s", out.Payload)
	}
}

// TestRedact_NoMatch_ReturnsOriginalPayload: envelope with no PII fields
// passes through with provenance populated.
func TestRedact_NoMatch_ReturnsOriginalPayload(t *testing.T) {
	env := NewEnvelope("memory.committed", "agg-1")
	env.Payload = json.RawMessage(`{"path":"docs/foo.md","sha256":"abc123"}`)

	out := Redact(*env)

	if string(out.Payload) != string(env.Payload) {
		t.Errorf("payload foi modificado indevidamente: in=%s out=%s", env.Payload, out.Payload)
	}
	if out.Provenance == nil {
		t.Fatal("Provenance não foi populado")
	}
	if out.Provenance.RedactedPayloadHash == "" {
		t.Error("RedactedPayloadHash não foi preenchido")
	}
	if len(out.Provenance.RedactedFields) != 0 {
		t.Errorf("RedactedFields=%v; esperado vazio", out.Provenance.RedactedFields)
	}
}

// TestRedact_HashIsStable: same input produces same SHA-256.
func TestRedact_HashIsStable(t *testing.T) {
	env := NewEnvelope("voice.transcribed", "audio-1")
	env.Payload = json.RawMessage(`{"transcript":"hello world"}`)

	out1 := Redact(*env)
	out2 := Redact(*env)

	if out1.Provenance.RedactedPayloadHash != out2.Provenance.RedactedPayloadHash {
		t.Errorf("hash instável: %s vs %s",
			out1.Provenance.RedactedPayloadHash, out2.Provenance.RedactedPayloadHash)
	}

	// And the hash must match sha256 of the original payload bytes.
	expected := sha256.Sum256(env.Payload)
	if out1.Provenance.RedactedPayloadHash != hex.EncodeToString(expected[:]) {
		t.Errorf("hash divergente: got=%s want=%s",
			out1.Provenance.RedactedPayloadHash, hex.EncodeToString(expected[:]))
	}
}

// TestRedact_DoesNotMutateInput: input envelope must be unchanged.
func TestRedact_DoesNotMutateInput(t *testing.T) {
	env := NewEnvelope("voice.transcribed", "audio-1")
	env.Payload = json.RawMessage(`{"transcript":"hello world"}`)
	original := string(env.Payload)

	_ = Redact(*env)

	if string(env.Payload) != original {
		t.Errorf("input foi mutado: was=%s now=%s", original, env.Payload)
	}
}

// TestRedact_MultipleRulesMatch: all three default rules apply in one envelope.
func TestRedact_MultipleRulesMatch(t *testing.T) {
	env := NewEnvelope("voice.transcribed", "audio-1")
	env.Payload = json.RawMessage(`{"transcript":"hi","audio_url":"https://x","prompt":"explain"}`)

	out := Redact(*env)

	if !strings.Contains(string(out.Payload), "[REDACTED:transcript]") {
		t.Error("transcript não redactado")
	}
	if !strings.Contains(string(out.Payload), "[REDACTED:audio_url]") {
		t.Error("audio_url não redactado")
	}
	if !strings.Contains(string(out.Payload), "[REDACTED:prompt]") {
		t.Error("prompt não redactado")
	}
	if len(out.Provenance.RedactedFields) != 3 {
		t.Errorf("RedactedFields=%v; esperado 3 entradas", out.Provenance.RedactedFields)
	}
}

// TestRedact_EmptyPayload: edge case — empty payload should not crash.
func TestRedact_EmptyPayload(t *testing.T) {
	env := NewEnvelope("memory.committed", "agg-1")
	// Payload is nil/empty by default.

	out := Redact(*env)

	if out.Provenance == nil || out.Provenance.RedactedPayloadHash == "" {
		t.Error("Provenance ou hash não preenchidos para payload vazio")
	}
}

// TestRedact_NonObjectPayload: payload that's not a JSON object passes
// through with hash populated.
func TestRedact_NonObjectPayload(t *testing.T) {
	env := NewEnvelope("memory.committed", "agg-1")
	env.Payload = json.RawMessage(`"just a string"`)

	out := Redact(*env)

	if string(out.Payload) != `"just a string"` {
		t.Errorf("payload não-string foi modificado: %s", out.Payload)
	}
	if out.Provenance == nil || out.Provenance.RedactedPayloadHash == "" {
		t.Error("hash não preenchido")
	}
}
