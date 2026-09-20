package writer

import (
	"context"
	"database/sql"
	"errors"
)

// Check verifies that documents.revision matches expectedRevision for the
// given documentID. It is the precondition gate for every write into the
// vault (ADR-044 §P3) and is what makes concurrent edits safe — two
// callers racing on the same document will see different revisions and
// only one will pass Check.
//
// Special cases:
//   - expectedRevision < 0: returns ErrInvalidRevision
//   - expectedRevision == 0 and the document does NOT exist: returns nil
//     (creation path; caller is creating a new document)
//   - expectedRevision == 0 and the document already exists: returns
//     *PreconditionError with Expected=0 (creation conflict; caller
//     must re-read the document first)
//   - expectedRevision > 0 and the document does NOT exist: returns
//     *PreconditionError with Current=0 (the caller asked for a
//     specific revision but the document is gone)
//
// The function reads under the caller's transaction context (or the
// implicit transaction on *sql.DB). The actual atomicity guarantee
// comes from the writer wrapping Check in BEGIN IMMEDIATE — see
// writer.Write in T3. Check itself is the comparison primitive only.
//
// Check returns nil, *PreconditionError, or ErrInvalidRevision. All
// other errors (database connection lost, etc.) are propagated from the
// underlying *sql.DB.
func Check(ctx context.Context, db *sql.DB, documentID string, expectedRevision int64) error {
	if expectedRevision < 0 {
		return ErrInvalidRevision
	}

	var current int64
	err := db.QueryRowContext(ctx,
		`SELECT revision FROM documents WHERE id = ?`, documentID,
	).Scan(&current)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Document does not exist. Two valid cases:
		//   expected == 0 → creation path is fine, no precondition violation.
		//   expected >  0 → caller expected a specific revision on a missing
		//                  document; report Current=0 so the caller can
		//                  format the 409 with the right number.
		if expectedRevision == 0 {
			return nil
		}
		return &PreconditionError{
			Expected:   expectedRevision,
			Current:    0,
			DocumentID: documentID,
		}

	case err != nil:
		return err // propagate database errors verbatim
	}

	// Document exists. Compare revisions.
	if expectedRevision == 0 {
		// Creation requested on an existing document → conflict.
		return &PreconditionError{
			Expected:   0,
			Current:    current,
			DocumentID: documentID,
		}
	}
	if current != expectedRevision {
		return &PreconditionError{
			Expected:   expectedRevision,
			Current:    current,
			DocumentID: documentID,
		}
	}
	return nil
}
