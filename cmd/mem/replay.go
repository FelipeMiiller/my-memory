package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
)

// runReplayCommand is the entry point for `mem replay --since <seq>
// [--apply]`. It re-emits events from the event_log to the registered
// subscribers so operators can recover from a corrupted read-side state
// or replay a fresh database from a backup.
//
// Default is dry-run (prints the JSON summary without invoking
// subscriber handlers); --apply re-executes the subscribers with
// idempotency provided by the projections (INSERT OR IGNORE on
// chunks, primary-key on graph_edges, etc.).
func runReplayCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	flags, err := parseReplayFlags(args)
	if err != nil {
		return err
	}

	cfg := resolveConfig()
	_, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, "", "", "", "")
	resolvedDB, resolvedPG, _ = applyStorageOverride("", resolvedDB, resolvedPG)
	if resolvedPG != "" {
		fmt.Fprintln(out, `{"error":"postgres not supported for replay"}`)
		return errReplayUnsupported
	}

	database, err := db.InitDB(resolvedDB)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer database.Close()

	// T8 ships with the audit subscriber always wired; projection
	// subscribers are registered separately by the calling binary so
	// replay works against whatever is actually registered in the
	// dispatcher's registry. We don't load subscribers here — the
	// replay tool reads events directly from event_log and dispatches
	// through the public Subscriber.Handle contract.
	subs := registeredSubscribers()
	if len(subs) == 0 {
		return errors.New("no subscribers registered — replay would be a no-op")
	}

	log := event_runtime.NewLog(database)
	result, err := replayRange(ctx, log, subs, flags.since, flags.apply)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// replayFlags captures --since and --apply. We avoid cobra/flag here
// to keep the binary footprint small; flag.FlagSet is plenty.
type replayFlags struct {
	since int64
	apply bool
}

func parseReplayFlags(args []string) (replayFlags, error) {
	var f replayFlags
	if len(args) == 0 {
		return f, errors.New("missing --since <seq>")
	}
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--since":
			if i+1 >= len(args) {
				return f, errors.New("--since requires a value")
			}
			v, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				return f, fmt.Errorf("--since must be an integer: %w", err)
			}
			f.since = v
			i += 2
		case "--apply":
			f.apply = true
			i++
		default:
			return f, fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	return f, nil
}

// ReplayResult is the JSON payload returned on success.
type ReplayResult struct {
	From           int64    `json:"from"`
	To             int64    `json:"to"`
	EventsReplayed int      `json:"events_replayed"`
	Subscribers    []string `json:"subscribers"`
	Apply          bool     `json:"apply"`
	Errors         []string `json:"errors,omitempty"`
}

// replayRange reads events from event_log starting at (since+1) up to
// the highest sequence, and dispatches each to every subscriber whose
// EventTypes() matches. When apply=false (dry-run) the subscribers are
// not invoked — only the summary is built.
//
// Idempotency: when apply=true, each subscriber is responsible for
// dedup (e.g. projection.sqlite uses INSERT OR IGNORE, projection.graph
// uses PRIMARY KEY). This is the contract — replay must be safe to
// run repeatedly.
//
// Returns 0 events with no error when since >= MAX(sequence).
func replayRange(ctx context.Context, log *event_runtime.Log, subs []event_runtime.Subscriber, since int64, apply bool) (ReplayResult, error) {
	if log == nil {
		return ReplayResult{}, errors.New("replay: nil log")
	}

	last, err := log.LastSequence(ctx)
	if err != nil {
		return ReplayResult{}, fmt.Errorf("replay: last sequence: %w", err)
	}

	result := ReplayResult{
		From:        since + 1,
		To:          last,
		Apply:       apply,
		Subscribers: subscriberNames(subs),
	}

	if last <= since {
		// stream Unix-like: empty range is not an error
		return result, nil
	}

	// Pull events in ASC order. We re-pull in chunks because the
	// event_log can grow large; chunked reads are gentler on memory
	// and let us short-circuit on the first error without holding the
	// full backlog.
	const chunkSize = 256
	for from := since + 1; from <= last; {
		budget := int64(chunkSize)
		if last-from+1 < budget {
			budget = last - from + 1
		}
		envs, err := readReplayWindow(ctx, log, from, budget)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			return result, err
		}
		for i := range envs {
			env := &envs[i]
			for _, sub := range subs {
				if !eventTypeMatches(sub.EventTypes(), env.EventType) {
					continue
				}
				if apply {
					if err := sub.Handle(ctx, env); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("sub=%s seq=%d: %v", sub.Name(), env.Sequence, err))
						// We continue so partial-replay errors don't
						// leave the operator blind to which other
						// subscribers succeeded. The dispatcher would
						// retry here, but for a one-shot CLI we surface
						// the partial state and let the operator decide.
					}
				}
				result.EventsReplayed++
			}
			from = env.Sequence + 1
		}
		if len(envs) == 0 {
			break
		}
	}
	return result, nil
}

// readReplayWindow is a tiny helper that reads events directly from
// event_log using a raw query — ReadUnacked applies the projection
// cursor filter which we want to bypass for replay.
func readReplayWindow(ctx context.Context, log *event_runtime.Log, fromSeq, limit int64) ([]event_runtime.Envelope, error) {
	return log.ReadRange(ctx, fromSeq, int(limit))
}

// eventTypeMatches uses path.Match semantics (same as dispatcher.go) so
// behavior is consistent with the in-process dispatcher.
func eventTypeMatches(patterns []string, eventType string) bool {
	for _, p := range patterns {
		if p == "*" {
			return true
		}
		if p == eventType {
			return true
		}
		// Glob match — the dispatcher uses path.Match too. We keep it
		// minimal here (no third-party globber); "*foo*" isn't a
		// production pattern for event_types.
		if matched, _ := filepath.Match(p, eventType); matched {
			return true
		}
	}
	return false
}

// subscriberNames extracts the names from a slice of subscribers for
// the JSON summary.
func subscriberNames(subs []event_runtime.Subscriber) []string {
	names := make([]string, 0, len(subs))
	for _, s := range subs {
		if s != nil {
			names = append(names, s.Name())
		}
	}
	return names
}

// errReplayUnsupported is returned when a non-sqlite engine is
// requested. Mirrors the pattern in events.go.
var errReplayUnsupported = errors.New("replay: only sqlite storage supported")

// registeredSubscribers is a placeholder hook so the rest of the CLI
// can wire real projection subscribers. The current implementation
// returns the empty slice; production wiring lives in the mymemoryd
// supervisor spec (ADR-042) which is out of scope for this ADR-044
// P1 implementation. Until that wiring lands, replay surfaces an
// explicit error so operators don't think a no-op was a success.
func registeredSubscribers() []event_runtime.Subscriber {
	return nil
}
