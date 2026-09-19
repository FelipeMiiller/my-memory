// Package event_runtime implements the canonical event envelope, durable
// SQLite event log, transactional outbox, in-process dispatcher and audit
// subscriber (see ADR-043 and ADR-050).
//
// This file defines the Envelope struct, its JSON contract and the validation
// rules used by every producer in the system. The struct field declaration
// order matches ADR-043 §4.1 so encoding/json emits fields in canonical order
// without relying on Go map iteration semantics.
package event_runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CurrentSchemaVersion is the schema_version this build emits and understands.
const CurrentSchemaVersion = 1

// Priority constants (ADR-043 §P1).
const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
)

// MaxPayloadBytes is the hard limit enforced on Envelope.Payload (ADR-043).
const MaxPayloadBytes = 1 << 20 // 1 MiB

// Package-level error sentinels. Tests and callers use errors.Is for matching.
var (
	// ErrInvalidEnvelope is returned when a required envelope field is empty
	// (event_type, aggregate_id) or the envelope pointer is nil.
	ErrInvalidEnvelope = errors.New("event_runtime: envelope is invalid")

	// ErrPayloadTooLarge is returned when Envelope.Payload exceeds 1 MiB.
	ErrPayloadTooLarge = errors.New("event_runtime: payload exceeds 1 MiB hard limit")

	// ErrTxRequired is returned by Outbox.Emit when the caller passes a nil *sql.Tx.
	ErrTxRequired = errors.New("event_runtime: transaction is required")

	// ErrDuplicateSubscriber is returned by RegisterSubscriber when a subscriber
	// with the same Name() was already registered on the dispatcher.
	ErrDuplicateSubscriber = errors.New("event_runtime: duplicate subscriber name")

	// ErrEventLogNotInitialized is returned by the dispatcher when the event_log
	// table is missing on startup (fail-closed per ADR-043 edge cases).
	ErrEventLogNotInitialized = errors.New("event_runtime: event log table is not initialized")

	// ErrDuplicateEventID is returned when the same event_id is appended twice
	// (UNIQUE constraint on event_log.event_id). The caller treats this as
	// idempotent retry success.
	ErrDuplicateEventID = errors.New("event_runtime: duplicate event_id")
)

// Provenance carries audit metadata for a single event (ADR-043 §4.1).
// All fields are optional and emitted only when populated.
type Provenance struct {
	ToolCallID          string   `json:"tool_call_id,omitempty"`
	ModelRunID          string   `json:"model_run_id,omitempty"`
	ApprovalID          string   `json:"approval_id,omitempty"`
	AuditSubscriber     bool     `json:"audit_subscriber,omitempty"`
	RedactedFields      []string `json:"redacted_fields,omitempty"`
	RedactedPayloadHash string   `json:"redacted_payload_hash,omitempty"`
}

// Trace carries distributed-tracing identifiers (ADR-043 §4.1).
type Trace struct {
	SpanID       string `json:"span_id,omitempty"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
}

// Envelope is the canonical event envelope (ADR-043 §4.1). Field declaration
// order is part of the JSON contract — encoding/json emits fields in this
// exact order, which keeps byte-level stability across producers and consumers.
type Envelope struct {
	EventID        string          `json:"event_id"`
	SchemaVersion  int             `json:"schema_version"`
	EventType      string          `json:"event_type"`
	Producer       string          `json:"producer,omitempty"`
	CreatedAt      string          `json:"created_at"`
	MonotonicNS    int64           `json:"monotonic_ns"`
	SessionID      string          `json:"session_id,omitempty"`
	ConversationID string          `json:"conversation_id,omitempty"`
	AggregateID    string          `json:"aggregate_id"`
	Epoch          int             `json:"epoch,omitempty"`
	Sequence       int64           `json:"sequence,omitempty"`
	CorrelationID  string          `json:"correlation_id,omitempty"`
	CausationID    string          `json:"causation_id,omitempty"`
	Actor          string          `json:"actor,omitempty"`
	Locale         string          `json:"locale,omitempty"`
	Priority       string          `json:"priority,omitempty"`
	ContentType    string          `json:"content_type,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	Provenance     *Provenance     `json:"provenance,omitempty"`
	Trace          *Trace          `json:"trace,omitempty"`
}

// NewEnvelope builds a fresh Envelope with sensible defaults:
//   - EventID: UUID v7 (timestamp-ordered, collision-safe)
//   - SchemaVersion: CurrentSchemaVersion
//   - CreatedAt: RFC 3339 nano UTC
//   - MonotonicNS: time.Now().UnixNano()
//   - Priority: PriorityNormal
//
// The caller is still expected to set Producer, Actor, Payload, etc. before
// passing the envelope to Log.Append.
func NewEnvelope(eventType, aggregateID string) *Envelope {
	now := time.Now().UTC()
	return &Envelope{
		EventID:       uuid.Must(uuid.NewV7()).String(),
		SchemaVersion: CurrentSchemaVersion,
		EventType:     eventType,
		CreatedAt:     now.Format(time.RFC3339Nano),
		MonotonicNS:   now.UnixNano(),
		AggregateID:   aggregateID,
		Priority:      PriorityNormal,
	}
}

// Validate enforces ADR-043 §P1 AC 4 (reject empty event_type / aggregate_id)
// and AC 6 (payload size ≤ 1 MiB). Returns nil when the envelope is valid.
//
// The function is safe to call on a nil receiver — returns ErrInvalidEnvelope.
func (e *Envelope) Validate() error {
	if e == nil {
		return ErrInvalidEnvelope
	}
	if e.EventType == "" {
		return fmt.Errorf("%w: event_type is empty", ErrInvalidEnvelope)
	}
	if e.AggregateID == "" {
		return fmt.Errorf("%w: aggregate_id is empty", ErrInvalidEnvelope)
	}
	if len(e.Payload) > MaxPayloadBytes {
		return fmt.Errorf("%w: payload is %d bytes, max %d", ErrPayloadTooLarge, len(e.Payload), MaxPayloadBytes)
	}
	return nil
}

// MarshalJSON emits the envelope in canonical ADR-043 §4.1 order. We use a
// local type alias to avoid infinite recursion while inheriting the field
// declaration order of Envelope (Go's encoding/json marshals struct fields in
// declaration order, which is deterministic across runs).
func (e *Envelope) MarshalJSON() ([]byte, error) {
	type envelopeAlias Envelope
	return json.Marshal((*envelopeAlias)(e))
}

// UnmarshalJSON accepts any well-formed envelope JSON. Unknown fields are
// preserved as-is (json.RawMessage on Payload handles nested unknown fields);
// the standard json.Unmarshal behaviour is sufficient for round-trip fidelity.
func (e *Envelope) UnmarshalJSON(data []byte) error {
	type envelopeAlias Envelope
	return json.Unmarshal(data, (*envelopeAlias)(e))
}
