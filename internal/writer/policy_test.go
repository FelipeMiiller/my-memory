package writer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
	"github.com/FelipeMiiller/my-memory/internal/policy"
	_ "modernc.org/sqlite"
)

// newWriterDBWithPolicy is the same minimal schema as newWriterDB
// but kept separate so future schema additions don't break policy
// tests (the policy suite doesn't need anything beyond documents,
// event_log, and projection_cursor).
func newWriterDBWithPolicy(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, stmt := range []string{
		`CREATE TABLE documents (
			id TEXT PRIMARY KEY,
			path TEXT NOT NULL UNIQUE,
			title TEXT,
			updated_at INTEGER NOT NULL,
			content_hash TEXT,
			abstract TEXT,
			category TEXT DEFAULT 'resource',
			revision INTEGER NOT NULL DEFAULT 0,
			last_event_id TEXT
		)`,
		`CREATE TABLE event_log (
			sequence INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id TEXT UNIQUE NOT NULL,
			schema_version INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			aggregate_id TEXT NOT NULL,
			revision INTEGER,
			payload BLOB NOT NULL,
			headers BLOB NOT NULL,
			created_at TEXT NOT NULL,
			acked_at TEXT
		)`,
		`CREATE TABLE projection_cursor (
			projection_name TEXT PRIMARY KEY,
			last_sequence INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}
	return db
}

// strictProfile returns an engine that requires approval for
// non-owner writes (the canonical "strict" semantic from ADR-050).
func strictProfile() *policy.Engine {
	return policy.NewEngine(policy.Profile{
		SchemaVersion: 1,
		DefaultOwner:  "user:owner",
		Rules: []policy.Rule{
			{MatchActor: "user:owner", Decision: "allow"},
			{MatchActor: "*", Decision: "require_approval"},
		},
	})
}

// denyAllProfile returns an engine that denies every non-owner write
// outright (used to exercise the Deny path specifically).
func denyAllProfile() *policy.Engine {
	return policy.NewEngine(policy.Profile{
		SchemaVersion: 1,
		DefaultOwner:  "user:owner",
		Rules: []policy.Rule{
			{MatchActor: "user:owner", Decision: "allow"},
			{MatchActor: "*", Decision: "deny"},
		},
	})
}

func TestWrite_PolicyDeniesAgent(t *testing.T) {
	db := newWriterDBWithPolicy(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.md")

	_, err := w.Write(context.Background(), WriteRequest{
		Path:         path,
		Content:      []byte("denied"),
		Actor:        "agent:external",
		PolicyEngine: denyAllProfile(),
	})
	if err == nil {
		t.Fatal("expected policy denial")
	}
	if !errIsPolicyBlocked(err) {
		t.Errorf("expected ErrPolicyDenied, got %v", err)
	}

	// No .md written
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf(".md should not exist after policy denial, stat err=%v", err)
	}
	// No event_log entry (Deny path emits no audit event)
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM event_log`).Scan(&n)
	if n != 0 {
		t.Errorf("event_log count=%d after denied write, want 0", n)
	}
}

func TestWrite_PolicyRecordsDecisionIDInPayload(t *testing.T) {
	db := newWriterDBWithPolicy(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := t.TempDir()
	path := filepath.Join(dir, "owned.md")

	res, err := w.Write(context.Background(), WriteRequest{
		Path:         path,
		Content:      []byte("v1"),
		Actor:        "user:owner",
		PolicyEngine: strictProfile(),
	})
	if err != nil {
		t.Fatalf("Write owner: %v", err)
	}

	var raw []byte
	if err := db.QueryRow(`SELECT payload FROM event_log WHERE event_id = ?`, res.EventID).Scan(&raw); err != nil {
		t.Fatalf("payload lookup: %v", err)
	}
	var env event_runtime.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	var p committedPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if p.PolicyDecisionID != "allow" {
		t.Errorf("PolicyDecisionID=%q, want 'allow'", p.PolicyDecisionID)
	}
}

func TestWrite_PolicyApprovalRequiredEmitsEvent(t *testing.T) {
	db := newWriterDBWithPolicy(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := t.TempDir()
	path := filepath.Join(dir, "external.md")

	_, err := w.Write(context.Background(), WriteRequest{
		Path:         path,
		Content:      []byte("needs approval"),
		Actor:        "agent:external",
		PolicyEngine: strictProfile(),
	})
	if !errIsPolicyBlocked(err) {
		t.Fatalf("expected policy block, got %v", err)
	}

	// approval.requested event MUST be present (for audit trail).
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM event_log WHERE event_type = 'approval.requested'`).Scan(&n)
	if n != 1 {
		t.Errorf("approval.requested count=%d, want 1", n)
	}
}

func TestWrite_NoPolicyIsBackwardCompatible(t *testing.T) {
	db := newWriterDBWithPolicy(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.md")

	// No PolicyEngine on WriteRequest → no policy check, no
	// policy_decision_id in payload.
	res, err := w.Write(context.Background(), WriteRequest{
		Path:    path,
		Content: []byte("v1"),
		Actor:   "user:owner",
	})
	if err != nil {
		t.Fatalf("Write without policy: %v", err)
	}
	var raw []byte
	_ = db.QueryRow(`SELECT payload FROM event_log WHERE event_id = ?`, res.EventID).Scan(&raw)
	var env event_runtime.Envelope
	_ = json.Unmarshal(raw, &env)
	var p committedPayload
	_ = json.Unmarshal(env.Payload, &p)
	if p.PolicyDecisionID != "" {
		t.Errorf("PolicyDecisionID=%q, want empty (backward-compat path)", p.PolicyDecisionID)
	}
}

// errIsPolicyBlocked reports true if err wraps either policy block
// sentinel — keeps tests resilient if the writer ever changes which
// sentinel a given path returns.
func errIsPolicyBlocked(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrPolicyDenied) || errors.Is(err, ErrApprovalRequired)
}

func containsString(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
