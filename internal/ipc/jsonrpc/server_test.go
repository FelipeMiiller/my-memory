package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// drainOutput returns the bytes written to stdout by Serve as a
// string. We collect via the bytes.Buffer returned from the test
// helper so concurrent writes don't race (Serve is single-threaded
// per Server, so a single Buffer is safe).
func runServerOnce(t *testing.T, srv *Server, input string) string {
	t.Helper()
	_, err := srv.stdin.(*bytes.Buffer).WriteString(input)
	if err != nil {
		t.Fatalf("seed stdin: %v", err)
	}
	if err := srv.Serve(context.Background()); err != nil {
		// Serve returns nil on EOF (bufio.Scanner returns false
		// when the underlying reader is empty). Anything else is
		// a real failure.
		t.Fatalf("Serve: %v", err)
	}
	return srv.stdout.(*bytes.Buffer).String()
}

func newTestServer(t *testing.T, opts ...Option) *Server {
	t.Helper()
	s, err := New(bytes.NewBuffer(nil), bytes.NewBuffer(nil), opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

// ---------------- Tests ----------------

func TestJsonRpc_InitializeHandshake(t *testing.T) {
	s := newTestServer(t)
	s.Register("ping", func(ctx context.Context, params json.RawMessage) (any, error) {
		return "pong", nil
	})

	input := `{"jsonrpc":"2.0","method":"initialize","id":1}` + "\n" +
		`{"jsonrpc":"2.0","method":"ping","id":2}` + "\n"

	out := runServerOnce(t, s, input)

	// Two responses: initialize + ping.
	lines := splitNonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("got %d response lines, want 2:\n%s", len(lines), out)
	}

	var initResp, pingResp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("parse initialize response: %v", err)
	}
	if initResp["id"].(float64) != 1 {
		t.Errorf("initialize id = %v, want 1", initResp["id"])
	}
	caps, ok := initResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("initialize result missing: %v", initResp)
	}
	if caps["protocol"] != "jsonrpc" {
		t.Errorf("protocol = %v, want jsonrpc", caps["protocol"])
	}

	if err := json.Unmarshal([]byte(lines[1]), &pingResp); err != nil {
		t.Fatalf("parse ping response: %v", err)
	}
	if pingResp["result"] != "pong" {
		t.Errorf("ping result = %v, want pong", pingResp["result"])
	}

	if !s.Initialized() {
		t.Error("server should be initialized after handshake")
	}
}

func TestJsonRpc_RejectsMalformedFrame(t *testing.T) {
	s := newTestServer(t)

	// Garbage that isn't valid JSON at all, followed by a valid
	// request that lands as method-not-found AFTER the handshake.
	// Initialize first so the second frame isn't bounced as
	// "handshake required" — the test is about parse-error
	// isolation, not handshake gating.
	input := `{"jsonrpc":"2.0","method":"initialize","id":1}` + "\n" +
		"not json\n" +
		`{"jsonrpc":"2.0","method":"foo","id":7}` + "\n"

	out := runServerOnce(t, s, input)
	lines := splitNonEmptyLines(out)
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), out)
	}

	// First line: initialize ok.
	var initResp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("parse init response: %v", err)
	}
	if initResp["error"] != nil {
		t.Errorf("init should succeed, got error: %v", initResp["error"])
	}

	// Second line: parse error response.
	var parseErr map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &parseErr); err != nil {
		t.Fatalf("parse error envelope: %v", err)
	}
	errObj := parseErr["error"].(map[string]any)
	if errObj["code"].(float64) != float64(CodeParseError) {
		t.Errorf("parse error code = %v, want %d", errObj["code"], CodeParseError)
	}

	// Third line: foo response — parse error must not pollute
	// the subsequent frame's id.
	var fooResp map[string]any
	if err := json.Unmarshal([]byte(lines[2]), &fooResp); err != nil {
		t.Fatalf("parse foo response: %v", err)
	}
	if fooResp["id"].(float64) != 7 {
		t.Errorf("foo id = %v, want 7", fooResp["id"])
	}
	errObj = fooResp["error"].(map[string]any)
	if errObj["code"].(float64) != float64(CodeMethodNotFound) {
		t.Errorf("foo error code = %v, want %d", errObj["code"], CodeMethodNotFound)
	}
}

func TestJsonRpc_RejectsBatchOver100(t *testing.T) {
	s := newTestServer(t)
	s.Register("noop", func(ctx context.Context, params json.RawMessage) (any, error) {
		return nil, nil
	})

	// Build a batch of 101 requests.
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < 101; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"jsonrpc":"2.0","method":"noop","id":`)
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("}")
	}
	sb.WriteString("]\n")

	out := runServerOnce(t, s, sb.String())
	lines := splitNonEmptyLines(out)
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1 (single batch error):\n%s", len(lines), out)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"].(float64) != float64(CodeBatchLimitExceeded) {
		t.Errorf("error code = %v, want %d", errObj["code"], CodeBatchLimitExceeded)
	}
	if !strings.Contains(errObj["message"].(string), "101") {
		t.Errorf("error message should mention 101: %v", errObj["message"])
	}
}

func TestJsonRpc_HandshakeTimeout(t *testing.T) {
	// Shrink BOTH the per-request timeout and the handshake TTL so
	// the test runs in milliseconds rather than the 30s default.
	s := newTestServer(t,
		WithHandshakeTTL(50*time.Millisecond),
		WithRequestTimeout(100*time.Millisecond),
	)

	// Send initialize followed by a request that won't complete in
	// time. The handler blocks on ctx.Done() so the per-request
	// timeout fires deterministically.
	blockCh := make(chan struct{})
	defer close(blockCh) // unblock goroutine on test exit
	s.Register("slow", func(ctx context.Context, params json.RawMessage) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	input := `{"jsonrpc":"2.0","method":"initialize","id":1}` + "\n" +
		`{"jsonrpc":"2.0","method":"slow","id":2}` + "\n"
	out := runServerOnce(t, s, input)
	lines := splitNonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), out)
	}

	var slow map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &slow); err != nil {
		t.Fatalf("parse slow response: %v", err)
	}
	errObj := slow["error"].(map[string]any)
	if errObj["code"].(float64) != float64(CodeInternalError) {
		t.Errorf("slow error code = %v, want %d (timeout wraps as internal)",
			errObj["code"], CodeInternalError)
	}
	if !strings.Contains(errObj["message"].(string), "timeout") {
		t.Errorf("slow error message should mention timeout: %v", errObj["message"])
	}
}

func TestJsonRpc_NotificationHasNoResponse(t *testing.T) {
	s := newTestServer(t)
	s.Register("log", func(ctx context.Context, params json.RawMessage) (any, error) {
		// Notification handler — return value is ignored by the
		// caller, but we still must return something non-error
		// for the dispatch path to land cleanly.
		return nil, nil
	})

	// initialize + notification + real request
	input := `{"jsonrpc":"2.0","method":"initialize","id":1}` + "\n" +
		`{"jsonrpc":"2.0","method":"log"}` + "\n" +
		`{"jsonrpc":"2.0","method":"log","id":2}` + "\n"

	out := runServerOnce(t, s, input)
	lines := splitNonEmptyLines(out)

	// Two responses: initialize + the id=2 log request. The
	// notification produces no reply.
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (notification silent):\n%s", len(lines), out)
	}
}

func TestJsonRpc_TypedErrorPassesThrough(t *testing.T) {
	s := newTestServer(t)
	s.Register("validate", func(ctx context.Context, params json.RawMessage) (any, error) {
		return nil, &Error{Code: CodeInvalidParams, Message: "missing required field: name"}
	})

	input := `{"jsonrpc":"2.0","method":"initialize","id":1}` + "\n" +
		`{"jsonrpc":"2.0","method":"validate","id":7}` + "\n"

	out := runServerOnce(t, s, input)
	lines := splitNonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), out)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"].(float64) != float64(CodeInvalidParams) {
		t.Errorf("error code = %v, want %d", errObj["code"], CodeInvalidParams)
	}
	if !strings.Contains(errObj["message"].(string), "name") {
		t.Errorf("error message = %v, want it to mention 'name'", errObj["message"])
	}
}

func TestJsonRpc_RejectsNonV2Protocol(t *testing.T) {
	s := newTestServer(t)

	input := `{"jsonrpc":"1.0","method":"foo","id":1}` + "\n"
	out := runServerOnce(t, s, input)
	lines := splitNonEmptyLines(out)
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1:\n%s", len(lines), out)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"].(float64) != float64(CodeParseError) {
		t.Errorf("error code = %v, want %d", errObj["code"], CodeParseError)
	}
}

func TestJsonRpc_NewRejectsNilStreams(t *testing.T) {
	if _, err := New(nil, bytes.NewBuffer(nil)); err == nil {
		t.Fatal("expected error on nil stdin")
	}
	if _, err := New(bytes.NewBuffer(nil), nil); err == nil {
		t.Fatal("expected error on nil stdout")
	}
}

func TestJsonRpc_BatchEmptyRejected(t *testing.T) {
	s := newTestServer(t)

	out := runServerOnce(t, s, "[]\n")
	lines := splitNonEmptyLines(out)
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1:\n%s", len(lines), out)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"].(float64) != float64(CodeInvalidRequest) {
		t.Errorf("error code = %v, want %d", errObj["code"], CodeInvalidRequest)
	}
}

func TestJsonRpc_HandlerErrorIsInternal(t *testing.T) {
	s := newTestServer(t)
	s.Register("boom", func(ctx context.Context, params json.RawMessage) (any, error) {
		return nil, errors.New("kaboom")
	})

	input := `{"jsonrpc":"2.0","method":"initialize","id":1}` + "\n" +
		`{"jsonrpc":"2.0","method":"boom","id":2}` + "\n"
	out := runServerOnce(t, s, input)
	lines := splitNonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), out)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"].(float64) != float64(CodeInternalError) {
		t.Errorf("error code = %v, want %d", errObj["code"], CodeInternalError)
	}
	if !strings.Contains(errObj["message"].(string), "kaboom") {
		t.Errorf("error message = %v, want it to mention 'kaboom'", errObj["message"])
	}
}

// splitNonEmptyLines splits a multi-line string on newlines and
// drops blank lines so tests can assert on the response count
// without tripping over the trailing newline Serve always emits.
func splitNonEmptyLines(s string) []string {
	parts := strings.Split(s, "\n")
	out := parts[:0]
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// silence unused-import warnings when a test gets temporarily
// commented out (e.g. during refactors).
var _ = atomic.Bool{}
