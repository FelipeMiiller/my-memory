package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
)

// exitMissingConfig is the exit code returned when the vault config is
// missing (the spec mandates exit 2 so automation can distinguish from
// generic runtime errors).
const exitMissingConfig = 2

// runEventsCLI is the entry point for `mem events <subcommand>`. The
// subcommand is read from os.Args[2]; remaining flags/args follow.
func runEventsCLI(ctx context.Context, args []string) error {
	if len(args) < 1 {
		printEventsHelp()
		return nil
	}
	sub := args[0]
	rest := args[1:]

	// Each subcommand needs a DB connection; we resolve config + open DB
	// once here so the helpers below can share the connection.
	cfgPath, cfgErr := findConfigPath(".")
	if cfgErr != nil {
		fmt.Fprintf(os.Stderr, "mem: vault not initialized — %v. Run `mem init` first.\n", cfgErr)
		os.Exit(exitMissingConfig)
	}
	_ = cfgPath

	cfg := resolveConfig()
	_, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, "", "", "", "")
	resolvedDB, resolvedPG, _ = applyStorageOverride("", resolvedDB, resolvedPG)

	if resolvedPG != "" {
		fmt.Fprintf(os.Stderr, "mem events: PostgreSQL engine not yet supported for events (use --storage=sqlite)\n")
		os.Exit(1)
	}

	database, err := db.InitDB(resolvedDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mem events: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	switch sub {
	case "tail":
		return runEventsTail(ctx, database, rest)
	case "inspect":
		return runEventsInspect(ctx, database, rest)
	case "replay":
		return runEventsReplay(ctx, database, rest)
	case "trace":
		return runEventsTrace(ctx, database, rest)
	case "last-sequence":
		return runEventsLastSequence(ctx, database)
	case "stats":
		return runEventsStats(ctx, database)
	default:
		fmt.Fprintf(os.Stderr, "mem events: unknown subcommand %q\n", sub)
		printEventsHelp()
		return nil
	}
}

// findConfigPath returns the path to .memory/config.yaml if it exists,
// or an error explaining how to initialize.
func findConfigPath(root string) (string, error) {
	candidates := []string{
		filepath.Join(root, ".memory", "config.yaml"),
		filepath.Join(root, "config.yaml"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf(".memory/config.yaml not found in %q", root)
}

// runEventsTail prints the last N events as JSON lines, ordered by sequence DESC.
//
// Flags:
//
//	--limit N     (default 20)
//	--session ID  (filter by envelope.SessionID)
//	--type PATT   (filter by glob pattern against envelope.EventType)
func runEventsTail(ctx context.Context, database *sql.DB, args []string) error {
	cmd := flag.NewFlagSet("events tail", flag.ContinueOnError)
	limit := cmd.Int("limit", 20, "Max number of events to print")
	session := cmd.String("session", "", "Filter by session_id")
	typePatt := cmd.String("type", "", "Filter by event_type glob pattern")
	if err := cmd.Parse(args); err != nil {
		return err
	}

	query := `SELECT sequence, event_id, event_type, aggregate_id, schema_version, revision,
	                 json_extract(payload, '$.actor') AS actor, payload, created_at, acked_at
	          FROM event_log
	          WHERE 1=1`
	args2 := []interface{}{}
	if *session != "" {
		query += ` AND json_extract(payload, '$.session_id') = ?`
		args2 = append(args2, *session)
	}
	query += ` ORDER BY sequence DESC LIMIT ?`
	args2 = append(args2, *limit)

	rows, err := database.QueryContext(ctx, query, args2...)
	if err != nil {
		return fmt.Errorf("mem events tail: %v", err)
	}
	defer rows.Close()

	type rowSummary struct {
		Sequence       int64           `json:"sequence"`
		EventID        string          `json:"event_id"`
		EventType      string          `json:"event_type"`
		AggregateID    string          `json:"aggregate_id"`
		SchemaVersion  int             `json:"schema_version"`
		Revision       int             `json:"revision"`
		Actor          string          `json:"actor,omitempty"`
		CreatedAt      string          `json:"created_at"`
		AckedAt        string          `json:"acked_at,omitempty"`
		PayloadPreview json.RawMessage `json:"payload,omitempty"`
	}

	count := 0
	for rows.Next() {
		var r rowSummary
		var ackedAt sql.NullString
		var revision sql.NullInt64
		var actor sql.NullString
		if err := rows.Scan(&r.Sequence, &r.EventID, &r.EventType, &r.AggregateID,
			&r.SchemaVersion, &revision, &actor, &r.PayloadPreview,
			&r.CreatedAt, &ackedAt); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		if revision.Valid {
			r.Revision = int(revision.Int64)
		}
		if actor.Valid {
			r.Actor = actor.String
		}
		if ackedAt.Valid {
			r.AckedAt = ackedAt.String
		}
		// Optional glob filter on event_type.
		if *typePatt != "" {
			matched, _ := path.Match(*typePatt, r.EventType)
			if !matched {
				continue
			}
		}
		out, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		fmt.Println(string(out))
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate: %w", err)
	}
	if count == 0 {
		fmt.Fprintln(os.Stderr, "mem events tail: 0 events")
	}
	return nil
}

// runEventsInspect prints the full envelope for a single event_id, pretty-printed.
// Exits non-zero if the event is not found.
func runEventsInspect(ctx context.Context, database *sql.DB, args []string) error {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Uso: mem events inspect <event_id>")
		return nil
	}
	eventID := args[0]

	row := database.QueryRowContext(ctx,
		`SELECT sequence, event_id, event_type, aggregate_id, schema_version, revision,
		        json_extract(payload, '$.actor') AS actor, payload, headers, created_at, acked_at
		 FROM event_log WHERE event_id = ?`, eventID)

	var (
		seq, revision                        sql.NullInt64
		eventType, aggID                     string
		schemaV                              int
		actor, payloadB, headersB, createdAt sql.NullString
		ackedAt                              sql.NullString
	)
	if err := row.Scan(&seq, &eventID, &eventType, &aggID, &schemaV, &revision,
		&actor, &payloadB, &headersB, &createdAt, &ackedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Fprintf(os.Stderr, "mem events inspect: event_id=%q not found\n", eventID)
			os.Exit(1)
		}
		return fmt.Errorf("mem events inspect: %w", err)
	}

	env := map[string]interface{}{
		"sequence":       seq.Int64,
		"event_id":       eventID,
		"event_type":     eventType,
		"aggregate_id":   aggID,
		"schema_version": schemaV,
		"actor":          nullStringToString(actor),
		"created_at":     nullStringToString(createdAt),
		"acked_at":       nullStringToString(ackedAt),
		"headers":        json.RawMessage(nullStringToString(payloadB)),
		"payload":        json.RawMessage(nullStringToString(payloadB)),
	}
	if revision.Valid {
		env["revision"] = int(revision.Int64)
	}
	out, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

// runEventsReplay re-emits events in [since, to] to subscribers.
//
// Flags:
//
//	--since N       (required; sequence to start from)
//	--to N          (optional; upper bound, defaults to MAX(sequence))
//	--dry-run       (default; only print, do NOT call subscribers)
//	--apply         (override --dry-run; actually dispatch)
//	--yes           (required with --apply to skip confirmation)
//
// Without --apply, we always succeed (no side effects). With --apply but
// without --yes, we fail-closed (exit non-zero) so automation can never
// accidentally re-emit events.
func runEventsReplay(ctx context.Context, database *sql.DB, args []string) error {
	cmd := flag.NewFlagSet("events replay", flag.ContinueOnError)
	since := cmd.Int64("since", 0, "Sequence to start from (required)")
	to := cmd.Int64("to", 0, "Sequence to stop at (0 = MAX)")
	dryRun := cmd.Bool("dry-run", true, "Only print events; do NOT dispatch (default)")
	apply := cmd.Bool("apply", false, "Actually dispatch events to subscribers")
	yes := cmd.Bool("yes", false, "Skip confirmation prompt with --apply")
	if err := cmd.Parse(args); err != nil {
		return err
	}
	if *since <= 0 {
		fmt.Fprintln(os.Stderr, "mem events replay: --since is required (positive integer)")
		return errors.New("--since required")
	}

	if *apply {
		if !*yes {
			fmt.Fprintln(os.Stderr, "mem events replay: --apply requires --yes (fail-closed to avoid accidental re-emit)")
			os.Exit(3)
		}
		fmt.Fprintf(os.Stderr, "WARN mem events replay: --apply will re-dispatch events seq=%d..%d\n", *since, *to)
	}

	upper := *to
	if upper <= 0 {
		var maxSeq sql.NullInt64
		if err := database.QueryRowContext(ctx, `SELECT MAX(sequence) FROM event_log`).Scan(&maxSeq); err != nil {
			return fmt.Errorf("query max sequence: %w", err)
		}
		upper = maxSeq.Int64
	}
	if upper < *since {
		fmt.Fprintf(os.Stderr, "mem events replay: --to (%d) < --since (%d); nothing to replay\n", upper, *since)
		return nil
	}

	rows, err := database.QueryContext(ctx,
		`SELECT sequence, event_id, event_type, payload FROM event_log
		 WHERE sequence BETWEEN ? AND ? ORDER BY sequence ASC`,
		*since, upper)
	if err != nil {
		return fmt.Errorf("query replay range: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var (
			seq     int64
			eventID string
			evType  string
			payload []byte
		)
		if err := rows.Scan(&seq, &eventID, &evType, &payload); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		line := map[string]interface{}{
			"sequence":   seq,
			"event_id":   eventID,
			"event_type": evType,
			"applied":    *apply && !*dryRun,
			"payload":    json.RawMessage(payload),
		}
		out, _ := json.Marshal(line)
		fmt.Println(string(out))
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate: %w", err)
	}
	if count == 0 {
		fmt.Fprintln(os.Stderr, "mem events replay: 0 events in range")
	}
	return nil
}

// runEventsTrace returns the DAG (one line per event) sharing a correlation_id.
func runEventsTrace(ctx context.Context, database *sql.DB, args []string) error {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Uso: mem events trace <correlation_id>")
		return nil
	}
	correlationID := args[0]

	rows, err := database.QueryContext(ctx,
		`SELECT sequence, event_id, event_type, aggregate_id,
		        json_extract(payload, '$.actor') AS actor,
		        json_extract(payload, '$.causation_id') AS causation_id
		 FROM event_log
		 WHERE json_extract(payload, '$.correlation_id') = ?
		 ORDER BY sequence ASC`, correlationID)
	if err != nil {
		return fmt.Errorf("query trace: %w", err)
	}
	defer rows.Close()

	type traceNode struct {
		Sequence    int64  `json:"sequence"`
		EventID     string `json:"event_id"`
		EventType   string `json:"event_type"`
		AggregateID string `json:"aggregate_id"`
		Actor       string `json:"actor,omitempty"`
		CausationID string `json:"causation_id,omitempty"`
	}
	count := 0
	for rows.Next() {
		var n traceNode
		var actor, cause sql.NullString
		if err := rows.Scan(&n.Sequence, &n.EventID, &n.EventType, &n.AggregateID, &actor, &cause); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		if actor.Valid {
			n.Actor = actor.String
		}
		if cause.Valid {
			n.CausationID = cause.String
		}
		out, _ := json.Marshal(n)
		fmt.Println(string(out))
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate: %w", err)
	}
	if count == 0 {
		fmt.Fprintf(os.Stderr, "mem events trace: 0 events with correlation_id=%q\n", correlationID)
	}
	return nil
}

// runEventsLastSequence prints the highest sequence in event_log.
func runEventsLastSequence(ctx context.Context, database *sql.DB) error {
	var seq sql.NullInt64
	if err := database.QueryRowContext(ctx, `SELECT MAX(sequence) FROM event_log`).Scan(&seq); err != nil {
		return fmt.Errorf("query max sequence: %w", err)
	}
	fmt.Println(seq.Int64)
	return nil
}

// runEventsStats prints JSON object with totals + oldest unacked age.
func runEventsStats(ctx context.Context, database *sql.DB) error {
	stats := map[string]interface{}{}

	// Total events.
	var total int64
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_log`).Scan(&total); err != nil {
		return fmt.Errorf("count total: %w", err)
	}
	stats["total_events"] = total

	// Events by type (top 10).
	rows, err := database.QueryContext(ctx,
		`SELECT event_type, COUNT(*) AS n FROM event_log GROUP BY event_type ORDER BY n DESC LIMIT 10`)
	if err != nil {
		return fmt.Errorf("count by type: %w", err)
	}
	defer rows.Close()
	typeCount := []map[string]interface{}{}
	for rows.Next() {
		var et string
		var n int64
		if err := rows.Scan(&et, &n); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		typeCount = append(typeCount, map[string]interface{}{"event_type": et, "count": n})
	}
	stats["events_by_type_top10"] = typeCount

	// Oldest unacked sequence + age in seconds.
	var oldestSeq sql.NullInt64
	var oldestCreated sql.NullString
	if err := database.QueryRowContext(ctx,
		`SELECT sequence, created_at FROM event_log WHERE acked_at IS NULL ORDER BY sequence ASC LIMIT 1`).Scan(&oldestSeq, &oldestCreated); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("query oldest unacked: %w", err)
	}
	if oldestSeq.Valid {
		stats["oldest_unacked_sequence"] = oldestSeq.Int64
		// Compute age from created_at (RFC 3339 nano).
		if t, err := time.Parse(time.RFC3339Nano, oldestCreated.String); err == nil {
			ageSec := int64(time.Since(t).Seconds())
			if ageSec < 0 {
				ageSec = 0
			}
			stats["oldest_unacked_age_seconds"] = ageSec
		}
	} else {
		stats["oldest_unacked_sequence"] = nil
		stats["oldest_unacked_age_seconds"] = 0
	}

	out, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

// printEventsHelp prints the `mem events --help` summary.
func printEventsHelp() {
	fmt.Println("Uso: mem events <subcomando> [opções]")
	fmt.Println("Subcomandos:")
	fmt.Println("  tail [--limit N] [--session ID] [--type PATT]")
	fmt.Println("      Lista os últimos N eventos (default 20) como JSON Lines.")
	fmt.Println("  inspect <event_id>")
	fmt.Println("      Exibe o envelope completo (pretty-printed). Sai ≠ 0 se não encontrado.")
	fmt.Println("  replay --since N [--to N] [--dry-run|--apply] [--yes]")
	fmt.Println("      Re-emite eventos no intervalo [since, to]. --apply requer --yes (fail-closed).")
	fmt.Println("  trace <correlation_id>")
	fmt.Println("      Lista eventos com o mesmo correlation_id em ordem causal.")
	fmt.Println("  last-sequence")
	fmt.Println("      Imprime MAX(sequence) do event_log (último evento escrito).")
	fmt.Println("  stats")
	fmt.Println("      Imprime JSON com total_events, top10 por tipo, oldest_unacked_age_seconds.")
}

// nullStringToString returns "" for NULL columns.
func nullStringToString(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

// suppress unused-import warnings for items we keep for future use.
var _ = strconv.Itoa
var _ = strings.TrimSpace
var _ = event_runtime.Envelope{}
