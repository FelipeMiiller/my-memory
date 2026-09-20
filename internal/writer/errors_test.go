package writer

import (
	"errors"
	"testing"
)

func TestSentinelErrors_AreDistinct(t *testing.T) {
	// Each sentinel must be a distinct value so errors.Is distinguishes
	// the failure mode the caller needs to handle.
	sentinels := []error{
		ErrWriteFailed,
		ErrDatabaseUnavailable,
		ErrPreconditionFailed,
		ErrPolicyDenied,
		ErrInvalidRevision,
		ErrApprovalRequired,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("sentinel %d matches sentinel %d via errors.Is (want distinct)", i, j)
			}
		}
	}
}

func TestPreconditionError_ImplementsErrorInterface(t *testing.T) {
	err := &PreconditionError{Expected: 5, Current: 7, DocumentID: "doc-abc"}
	if err.Error() == "" {
		t.Fatal("Error() must produce a non-empty message")
	}
	if !errors.Is(err, ErrPreconditionFailed) {
		t.Error("PreconditionError must satisfy errors.Is(err, ErrPreconditionFailed)")
	}
}

func TestPreconditionError_CreationConflictMessage(t *testing.T) {
	// Expected == 0 means "create new document"; conflict path produces
	// a distinct message so the caller can render the right 409 body.
	err := &PreconditionError{Expected: 0, Current: 3, DocumentID: "doc-xyz"}
	msg := err.Error()
	if msg == "" {
		t.Fatal("creation conflict message must not be empty")
	}
	// Should mention creation + document id for actionable diagnosis.
	if !contains(msg, "creation conflict") {
		t.Errorf("creation conflict message should mention 'creation conflict', got: %s", msg)
	}
}

func TestItoa(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{-7, "-7"},
		{9223372036854775807, "9223372036854775807"}, // math.MaxInt64
	}
	for _, tc := range cases {
		if got := itoa(tc.in); got != tc.want {
			t.Errorf("itoa(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
