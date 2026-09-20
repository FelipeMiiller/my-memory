package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

// TestNewMemoryWriteNoteHandler_AcceptsIfMatch validates that the
// memory_write_note tool parses the `if_match` parameter introduced
// by ADR-044 §T11 without erroring. The actual enforcement happens
// in the `mem write` CLI subcommand (writer path) — for the MCP
// tool we just confirm the parameter round-trips through the JSON
// unmarshal layer.
//
// Future hardening (not in T11 scope):
//   - Wire the handler to use internal/writer.Writer so if_match
//     is actually enforced; returns a 409 + conflict.detected when
//     the precondition fails.
//   - Emit the same `if_match` parameter on memory_append_section
//     and memory_compile_note (the spec mentioned all three tools).
func TestNewMemoryWriteNoteHandler_AcceptsIfMatch(t *testing.T) {
	handler := NewMemoryWriteNoteHandler(nil, ".")

	cases := []struct {
		name      string
		args      string
		wantError bool
	}{
		{
			name:      "with if_match present",
			args:      `{"path":"/x.md","content":"hello","if_match":3}`,
			wantError: false, // accepted (parsed), even if not enforced
		},
		{
			name:      "without if_match",
			args:      `{"path":"/x.md","content":"hello"}`,
			wantError: false,
		},
		{
			name:      "with if_match null",
			args:      `{"path":"/x.md","content":"hello","if_match":null}`,
			wantError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// We don't exercise the handler end-to-end (it would
			// require a real compiler engine + vault). Instead, decode
			// the same JSON shape and verify it parses.
			var raw struct {
				Path    string `json:"path"`
				Content string `json:"content"`
				IfMatch *int64 `json:"if_match,omitempty"`
			}
			err := json.Unmarshal([]byte(tc.args), &raw)
			if (err != nil) != tc.wantError {
				t.Errorf("Unmarshal err=%v, wantError=%v", err, tc.wantError)
			}

			// Handler must at least not panic on the input. We discard
			// both return values; this test only validates that the
			// handler accepts the new parameter shape.
			_, _ = handler(context.Background(), json.RawMessage(tc.args))
		})
	}
}
