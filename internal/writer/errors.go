package writer

import "errors"

// Package-level error sentinels for the writer (ADR-044). Callers use
// errors.Is to match against these. Each sentinel corresponds to a
// documented failure mode in ADR-044 §Considered Options + §Decision
// Outcome; the comments cite the spec section so reviewers can trace each
// error back to a concrete requirement.
var (
	// ErrWriteFailed is returned when the on-disk `.md` write fails
	// (disk full, permission denied, rename race on Windows). The SQLite
	// transaction is rolled back and no event is emitted. See
	// ADR-044 §Decision Outcome step 4.
	ErrWriteFailed = errors.New("writer: markdown write failed")

	// ErrDatabaseUnavailable is returned when the SQLite transaction
	// cannot begin or commit. The compensating action removes the
	// `.md` file written just before the transaction attempt. See
	// ADR-044 §Consequences (compensating action).
	ErrDatabaseUnavailable = errors.New("writer: database unavailable")

	// ErrPreconditionFailed is returned when the caller-supplied
	// expected_revision does not match documents.revision (or when
	// expected=0 conflicts with an existing document). The error wraps
	// the current revision via errors.As (see *PreconditionError). See
	// ADR-044 §Decision Outcome "Precondition".
	ErrPreconditionFailed = errors.New("writer: precondition failed (revision mismatch)")

	// ErrPolicyDenied is returned when the policy engine returns a
	// Deny decision (e.g., strict profile + non-owner actor). No
	// event is emitted, no transaction is started. See
	// ADR-044 §P4 (policy engine).
	ErrPolicyDenied = errors.New("writer: denied by policy")

	// ErrInvalidRevision is returned when ExpectedRevision is
	// negative or otherwise outside the int64 range accepted by the
	// precondition checker. See ADR-044 §Edge Cases.
	ErrInvalidRevision = errors.New("writer: invalid revision value")
)

// PreconditionError enriches ErrPreconditionFailed with the current
// documents.revision so callers (CLI, MCP tools) can format a useful 409
// response without an extra database round-trip.
type PreconditionError struct {
	Expected   int64  // the revision the caller expected (or 0 for creation)
	Current    int64  // the actual revision observed under BEGIN IMMEDIATE
	DocumentID string // document the precondition was checked against
}

func (e *PreconditionError) Error() string {
	if e.Expected == 0 {
		return "writer: precondition failed (creation conflict on " + e.DocumentID + ")"
	}
	return "writer: precondition failed (expected revision " +
		itoa(e.Expected) + ", current is " + itoa(e.Current) +
		" for document " + e.DocumentID + ")"
}

// Unwrap returns ErrPreconditionFailed so errors.Is works.
func (e *PreconditionError) Unwrap() error { return ErrPreconditionFailed }

// itoa is a tiny allocation-free integer formatter used in the error
// string. Errors are rare; pulling in strconv just for this would inflate
// the writer's dependency footprint.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
