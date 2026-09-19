package event_runtime

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// canonicalFieldOrder mirrors the ADR-043 §4.1 envelope schema. If a field is
// added, removed or reordered in the Envelope struct, update this list AND
// fix the struct — they must stay in sync.
var canonicalFieldOrder = []string{
	"event_id",
	"schema_version",
	"event_type",
	"producer",
	"created_at",
	"monotonic_ns",
	"session_id",
	"conversation_id",
	"aggregate_id",
	"epoch",
	"sequence",
	"correlation_id",
	"causation_id",
	"actor",
	"locale",
	"priority",
	"content_type",
	"payload",
	"provenance",
	"trace",
}

func TestEnvelope_NewEnvelope_AssignsRequiredFields(t *testing.T) {
	e := NewEnvelope("memory.committed", "documents/foo.md")
	if e == nil {
		t.Fatal("NewEnvelope retornou nil")
	}
	if e.EventID == "" {
		t.Error("NewEnvelope não atribuiu EventID")
	}
	if got, want := e.SchemaVersion, CurrentSchemaVersion; got != want {
		t.Errorf("SchemaVersion = %d; esperado %d", got, want)
	}
	if got, want := e.Priority, PriorityNormal; got != want {
		t.Errorf("Priority = %q; esperado %q", got, want)
	}
	if e.EventType != "memory.committed" {
		t.Errorf("EventType = %q; esperado %q", e.EventType, "memory.committed")
	}
	if e.AggregateID != "documents/foo.md" {
		t.Errorf("AggregateID = %q; esperado %q", e.AggregateID, "documents/foo.md")
	}
	if e.CreatedAt == "" {
		t.Error("NewEnvelope não atribuiu CreatedAt")
	}
	if e.MonotonicNS == 0 {
		t.Error("NewEnvelope não atribuiu MonotonicNS")
	}
}

func TestEnvelope_MarshalJSON_FieldOrderIsCanonical(t *testing.T) {
	e := NewEnvelope("audio.started", "session-abc")
	// Populate every field so `omitempty` doesn't strip them and the order is
	// observable.
	e.Producer = "mymemory-stt"
	e.SessionID = "sess-1"
	e.ConversationID = "conv-1"
	e.Epoch = 2
	e.Sequence = 17
	e.CorrelationID = "corr-1"
	e.CausationID = "cause-1"
	e.Actor = "user:felipe"
	e.Locale = "pt-BR"
	e.Priority = PriorityHigh
	e.ContentType = "application/json"
	e.Payload = json.RawMessage(`{"chunk":1}`)
	e.Provenance = &Provenance{ToolCallID: "tool-1"}
	e.Trace = &Trace{SpanID: "span-1"}

	data, err := e.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON falhou: %v", err)
	}

	gotOrder := keysInJSONOrder(data)
	if len(gotOrder) != len(canonicalFieldOrder) {
		t.Fatalf("número de chaves = %d; esperado %d (json=%s)", len(gotOrder), len(canonicalFieldOrder), data)
	}
	for i, want := range canonicalFieldOrder {
		if gotOrder[i] != want {
			t.Errorf("ordem da chave %d = %q; esperado %q (json=%s)", i, gotOrder[i], want, data)
		}
	}
}

func TestEnvelope_RoundTrip_DeepEqual(t *testing.T) {
	original := NewEnvelope("tool.finished", "documents/foo.md")
	original.Producer = "mymemory-agent"
	original.SessionID = "sess-1"
	original.ConversationID = "conv-1"
	original.CorrelationID = "corr-1"
	original.CausationID = "cause-1"
	original.Actor = "user:felipe"
	original.Locale = "pt-BR"
	original.Priority = PriorityHigh
	original.ContentType = "application/json"
	original.Payload = json.RawMessage(`{"tool":"search","ok":true,"count":42}`)
	original.Provenance = &Provenance{
		ToolCallID:          "tool-uuid",
		ModelRunID:          "run-uuid",
		RedactedPayloadHash: "abc123",
	}
	original.Trace = &Trace{SpanID: "span-1", ParentSpanID: "span-0"}

	data, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON falhou: %v", err)
	}

	var decoded Envelope
	if err := decoded.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON falhou: %v", err)
	}

	if decoded.EventID != original.EventID {
		t.Errorf("EventID divergente: %q vs %q", decoded.EventID, original.EventID)
	}
	if decoded.EventType != original.EventType {
		t.Errorf("EventType divergente: %q vs %q", decoded.EventType, original.EventType)
	}
	if decoded.AggregateID != original.AggregateID {
		t.Errorf("AggregateID divergente: %q vs %q", decoded.AggregateID, original.AggregateID)
	}
	if decoded.SchemaVersion != original.SchemaVersion {
		t.Errorf("SchemaVersion divergente: %d vs %d", decoded.SchemaVersion, original.SchemaVersion)
	}
	if decoded.MonotonicNS != original.MonotonicNS {
		t.Errorf("MonotonicNS divergente: %d vs %d", decoded.MonotonicNS, original.MonotonicNS)
	}
	if string(decoded.Payload) != string(original.Payload) {
		t.Errorf("Payload divergente: %s vs %s", decoded.Payload, original.Payload)
	}
	if decoded.Provenance == nil || original.Provenance == nil {
		t.Fatalf("Provenance é nil após round-trip: got=%v want=%v", decoded.Provenance, original.Provenance)
	}
	if decoded.Provenance.ToolCallID != original.Provenance.ToolCallID {
		t.Errorf("Provenance.ToolCallID divergente")
	}
	if decoded.Provenance.RedactedPayloadHash != original.Provenance.RedactedPayloadHash {
		t.Errorf("Provenance.RedactedPayloadHash divergente")
	}
	if decoded.Trace == nil || original.Trace == nil {
		t.Fatalf("Trace é nil após round-trip")
	}
	if decoded.Trace.SpanID != original.Trace.SpanID {
		t.Errorf("Trace.SpanID divergente")
	}
}

func TestEnvelope_Validate_RejectsEmptyEventType(t *testing.T) {
	e := NewEnvelope("", "documents/foo.md")
	err := e.Validate()
	if err == nil {
		t.Fatal("Validate deveria rejeitar event_type vazio")
	}
	if !errors.Is(err, ErrInvalidEnvelope) {
		t.Errorf("Validate deveria retornar ErrInvalidEnvelope; obteve %v", err)
	}
	if !strings.Contains(err.Error(), "event_type") {
		t.Errorf("mensagem de erro deve mencionar event_type; obteve %q", err.Error())
	}
}

func TestEnvelope_Validate_RejectsEmptyAggregateID(t *testing.T) {
	e := NewEnvelope("memory.committed", "")
	err := e.Validate()
	if err == nil {
		t.Fatal("Validate deveria rejeitar aggregate_id vazio")
	}
	if !errors.Is(err, ErrInvalidEnvelope) {
		t.Errorf("Validate deveria retornar ErrInvalidEnvelope; obteve %v", err)
	}
	if !strings.Contains(err.Error(), "aggregate_id") {
		t.Errorf("mensagem de erro deve mencionar aggregate_id; obteve %q", err.Error())
	}
}

func TestEnvelope_Validate_RejectsPayloadTooLarge(t *testing.T) {
	e := NewEnvelope("memory.committed", "documents/foo.md")
	e.Payload = json.RawMessage(make([]byte, MaxPayloadBytes+1))
	// Make it valid JSON-looking bytes.
	for i := range e.Payload {
		e.Payload[i] = 'x'
	}
	err := e.Validate()
	if err == nil {
		t.Fatal("Validate deveria rejeitar payload > 1 MiB")
	}
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Errorf("Validate deveria retornar ErrPayloadTooLarge; obteve %v", err)
	}
}

func TestEnvelope_Validate_AcceptsValidEnvelope(t *testing.T) {
	e := NewEnvelope("memory.committed", "documents/foo.md")
	e.Payload = json.RawMessage(`{"x":1}`)
	if err := e.Validate(); err != nil {
		t.Errorf("Validate em envelope válido falhou: %v", err)
	}
}

func TestEnvelope_Validate_NilReceiver(t *testing.T) {
	var e *Envelope
	err := e.Validate()
	if !errors.Is(err, ErrInvalidEnvelope) {
		t.Errorf("Validate(nil) deveria retornar ErrInvalidEnvelope; obteve %v", err)
	}
}

func TestEnvelope_NewEnvelope_GeneratesUniqueEventIDs(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		e := NewEnvelope("memory.committed", "documents/foo.md")
		if seen[e.EventID] {
			t.Fatalf("EventID duplicado em 100 NewEnvelope calls: %s", e.EventID)
		}
		seen[e.EventID] = true
	}
}

// keysInJSONOrder walks the top-level object keys in emission order.
// Returns nil if data isn't a JSON object.
func keysInJSONOrder(data []byte) []string {
	var keys []string
	dec := json.NewDecoder(strings.NewReader(string(data)))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return keys
	}
	for dec.More() {
		kTok, err := dec.Token()
		if err != nil {
			return keys
		}
		k, ok := kTok.(string)
		if !ok {
			return keys
		}
		keys = append(keys, k)
		// Skip the value associated with k.
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return keys
		}
	}
	return keys
}
