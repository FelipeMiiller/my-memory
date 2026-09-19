// Package event_runtime — audit_subscriber.go
//
// AuditSubscriber is the canonical Subscriber implementation (ADR-043 §5 +
// ADR-050 LLM02). It writes a JSONL line per delivered event to a file on
// disk. The line contains:
//
//   - event_id
//   - sequence
//   - event_type
//   - actor
//   - redacted_payload_hash (SHA-256 of pre-redaction payload, NEVER the raw payload)
//   - created_at
//
// All payloads are redacted before write (PII gate, ADR-050 LLM02). The
// audit log is fail-closed: write errors are returned to the dispatcher so
// they retry with exponential backoff — we never silently drop events
// (ADR-043 edge case "audit log file is on a read-only filesystem").
package event_runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// auditLine is the JSON shape persisted to the audit log. Kept minimal so
// the file is grep-able with standard Unix tools.
type auditLine struct {
	EventID             string `json:"event_id"`
	Sequence            int64  `json:"sequence"`
	EventType           string `json:"event_type"`
	Actor               string `json:"actor,omitempty"`
	RedactedPayloadHash string `json:"redacted_payload_hash"`
	CreatedAt           string `json:"created_at"`
}

// AuditSubscriber is the production audit implementation of Subscriber.
// It serializes writes through a sync.Mutex because the dispatcher calls
// Handle from a delivery goroutine.
type AuditSubscriber struct {
	path string
	mu   sync.Mutex
	out  *os.File
}

// NewAuditSubscriber opens (or creates) the audit log file in append mode.
// Returns an error if the file cannot be opened — for example, if the
// directory is read-only or the file path is invalid.
func NewAuditSubscriber(path string) (*AuditSubscriber, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("event_runtime: open audit log: %w", err)
	}
	return &AuditSubscriber{path: path, out: f}, nil
}

// Name returns the subscriber name used as projection_cursor key.
// Constant value is part of the audit log semantics; renaming it would
// reset the audit cursor and replay every event from sequence 0.
func (a *AuditSubscriber) Name() string { return "audit" }

// EventTypes subscribes to every event_type (audit sees ALL deliveries).
func (a *AuditSubscriber) EventTypes() []string { return []string{"*"} }

// MaxAckPending is 256 — audit is fast (single fsync + small JSON line)
// and we want it to drain quickly so the cursor advances.
func (a *AuditSubscriber) MaxAckPending() int { return 256 }

// Handle redacts the envelope, writes one JSONL line to the audit log,
// fsyncs for durability, and returns nil. On error (write failure, fsync
// failure, file closed), returns the error so the dispatcher retries with
// backoff (fail-closed).
func (a *AuditSubscriber) Handle(ctx context.Context, env *Envelope) error {
	if env == nil {
		return fmt.Errorf("event_runtime: audit handle nil envelope")
	}

	// 1. Redact PII (LLM02 gate). This populates RedactedPayloadHash.
	redacted := Redact(*env)

	// 2. Build the audit line. NO raw payload is ever written.
	line := auditLine{
		EventID:             redacted.EventID,
		Sequence:            redacted.Sequence,
		EventType:           redacted.EventType,
		Actor:               redacted.Actor,
		RedactedPayloadHash: redacted.Provenance.RedactedPayloadHash,
		CreatedAt:           redacted.CreatedAt,
	}

	data, err := json.Marshal(line)
	if err != nil {
		return fmt.Errorf("event_runtime: audit marshal: %w", err)
	}
	data = append(data, '\n')

	// 3. Write under mutex so concurrent Handle calls don't interleave.
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.out == nil {
		return fmt.Errorf("event_runtime: audit log file is closed")
	}
	if _, err := a.out.Write(data); err != nil {
		return fmt.Errorf("event_runtime: audit write: %w", err)
	}
	if err := a.out.Sync(); err != nil {
		return fmt.Errorf("event_runtime: audit fsync: %w", err)
	}
	return nil
}

// Close releases the file handle. Safe to call multiple times.
func (a *AuditSubscriber) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.out == nil {
		return nil
	}
	err := a.out.Close()
	a.out = nil
	return err
}

// auditLogPathForTesting is a tiny helper used by tests to build a unique
// path under t.TempDir().
func auditLogPathForTesting(prefix string) string {
	return fmt.Sprintf("%s/audit-%d.log", prefix, time.Now().UnixNano())
}

// Compile-time interface conformance check.
var _ Subscriber = (*AuditSubscriber)(nil)

// bufio.Writer is intentionally NOT used — every audit line is followed by
// an explicit fsync, so a buffered writer would delay durability. The
// direct *os.File gives us "write then fsync" semantics without an extra
// layer to flush.
var _ = bufio.NewWriter // keep import for future buffered variant
