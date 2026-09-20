package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
	_ "modernc.org/sqlite"
)

// countingSub is a minimal Subscriber for replay tests. Records the
// count of envelopes Handle was invoked with and, on apply mode, the
// last sequence it saw.
type countingSub struct {
	name        string
	patterns    []string
	count       int64
	lastSeq     int64
	failOnApply bool
}

func (c *countingSub) Name() string         { return c.name }
func (c *countingSub) EventTypes() []string { return c.patterns }
func (c *countingSub) MaxAckPending() int   { return 16 }
func (c *countingSub) Handle(_ context.Context, env *event_runtime.Envelope) error {
	atomic.AddInt64(&c.count, 1)
	atomic.StoreInt64(&c.lastSeq, env.Sequence)
	if c.failOnApply {
		return os.ErrInvalid
	}
	return nil
}

func newReplayDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, stmt := range []string{
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

// seedEvents appends n memory.committed envelopes so replayRange has
// something to operate on. Returns the highest sequence written.
func seedEvents(t *testing.T, db *sql.DB, log *event_runtime.Log, n int) int64 {
	t.Helper()
	var lastSeq int64
	for i := 0; i < n; i++ {
		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		env := event_runtime.NewEnvelope("memory.committed", "doc-foo")
		env.Revision = i + 1
		env.Payload = []byte(`{"path":"/tmp/foo.md","content_hash":"x","size_bytes":1}`)
		if err := log.Append(context.Background(), tx, env); err != nil {
			t.Fatalf("append: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit: %v", err)
		}
		lastSeq = env.Sequence
	}
	return lastSeq
}

func TestRunReplayCommand_DryRunReportsSummary(t *testing.T) {
	db := newReplayDB(t)
	log := event_runtime.NewLog(db)
	last := seedEvents(t, db, log, 5)

	// Use replayRange directly (CLI integration is separate).
	subs := []event_runtime.Subscriber{&countingSub{name: "test", patterns: []string{"*"}}}
	res, err := replayRange(context.Background(), log, subs, 0, false)
	if err != nil {
		t.Fatalf("replayRange: %v", err)
	}

	if res.From != 1 {
		t.Errorf("From=%d, want 1", res.From)
	}
	if res.To != last {
		t.Errorf("To=%d, want %d", res.To, last)
	}
	if res.EventsReplayed != 5 {
		t.Errorf("EventsReplayed=%d, want 5", res.EventsReplayed)
	}
	if res.Apply {
		t.Error("Apply should be false in dry-run")
	}
}

func TestRunReplayCommand_ApplyInvokesSubscribers(t *testing.T) {
	db := newReplayDB(t)
	log := event_runtime.NewLog(db)
	_ = seedEvents(t, db, log, 3)

	sub := &countingSub{name: "apply-test", patterns: []string{"memory.committed"}}
	res, err := replayRange(context.Background(), log, []event_runtime.Subscriber{sub}, 0, true)
	if err != nil {
		t.Fatalf("replayRange: %v", err)
	}
	if sub.count != 3 {
		t.Errorf("subscriber count=%d, want 3", sub.count)
	}
	if res.EventsReplayed != 3 {
		t.Errorf("EventsReplayed=%d, want 3", res.EventsReplayed)
	}
}

func TestRunReplayCommand_EmptyRange(t *testing.T) {
	db := newReplayDB(t)
	log := event_runtime.NewLog(db)
	last := seedEvents(t, db, log, 2)

	subs := []event_runtime.Subscriber{&countingSub{name: "empty", patterns: []string{"*"}}}
	// since >= last → empty range, no error
	res, err := replayRange(context.Background(), log, subs, last, false)
	if err != nil {
		t.Fatalf("replayRange empty: %v", err)
	}
	if res.EventsReplayed != 0 {
		t.Errorf("EventsReplayed=%d on empty range, want 0", res.EventsReplayed)
	}
	if res.To != last {
		t.Errorf("To=%d, want %d", res.To, last)
	}
}

func TestRunReplayCommand_EventTypeFilter(t *testing.T) {
	db := newReplayDB(t)
	log := event_runtime.NewLog(db)

	// Append 2 memory.committed + 1 conflict.detected
	for i, et := range []string{"memory.committed", "conflict.detected", "memory.committed"} {
		tx, _ := db.BeginTx(context.Background(), nil)
		env := event_runtime.NewEnvelope(et, "doc-foo")
		env.Revision = i + 1
		_ = log.Append(context.Background(), tx, env)
		_ = tx.Commit()
	}

	sub := &countingSub{name: "only-committed", patterns: []string{"memory.committed"}}
	res, _ := replayRange(context.Background(), log, []event_runtime.Subscriber{sub}, 0, true)
	if sub.count != 2 {
		t.Errorf("subscriber count=%d, want 2 (filtered)", sub.count)
	}
	if res.EventsReplayed != 2 {
		t.Errorf("EventsReplayed=%d, want 2", res.EventsReplayed)
	}
}

func TestParseReplayFlags(t *testing.T) {
	cases := []struct {
		args    []string
		wantSeq int64
		wantApp bool
		wantErr bool
	}{
		{[]string{"--since", "0"}, 0, false, false},
		{[]string{"--since", "42", "--apply"}, 42, true, false},
		{[]string{"--apply", "--since", "7"}, 7, true, false},
		{[]string{}, 0, false, true},
		{[]string{"--since"}, 0, false, true},
		{[]string{"--since", "not-a-number"}, 0, false, true},
		{[]string{"--bogus"}, 0, false, true},
	}
	for _, tc := range cases {
		got, err := parseReplayFlags(tc.args)
		if (err != nil) != tc.wantErr {
			t.Errorf("parseReplayFlags(%v) err=%v, wantErr=%v", tc.args, err, tc.wantErr)
		}
		if !tc.wantErr && (got.since != tc.wantSeq || got.apply != tc.wantApp) {
			t.Errorf("parseReplayFlags(%v) = (%d,%v), want (%d,%v)",
				tc.args, got.since, got.apply, tc.wantSeq, tc.wantApp)
		}
	}
}

func TestEventTypeMatches(t *testing.T) {
	cases := []struct {
		patterns  []string
		eventType string
		want      bool
	}{
		{[]string{"*"}, "anything", true},
		{[]string{"memory.committed"}, "memory.committed", true},
		{[]string{"memory.committed"}, "conflict.detected", false},
		{[]string{}, "memory.committed", false},
	}
	for _, tc := range cases {
		if got := eventTypeMatches(tc.patterns, tc.eventType); got != tc.want {
			t.Errorf("eventTypeMatches(%v, %q) = %v, want %v", tc.patterns, tc.eventType, got, tc.want)
		}
	}
}

func TestReplayResultJSONShape(t *testing.T) {
	// Verify the JSON output matches the spec contract.
	res := ReplayResult{
		From:           1,
		To:             10,
		EventsReplayed: 5,
		Subscribers:    []string{"projection.sqlite"},
		Apply:          false,
	}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(res); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Confirm all expected fields appear
	s := buf.String()
	for _, key := range []string{`"from":1`, `"to":10`, `"events_replayed":5`, `"subscribers":`, `"apply":false`} {
		if !bytes.Contains([]byte(s), []byte(key)) {
			t.Errorf("JSON missing key %q in: %s", key, s)
		}
	}
}

// Suppress unused-import lints on platforms where path/filepath may
// be referenced indirectly; ensures the test compiles cleanly across
// Windows + Linux.
var _ = filepath.Join
