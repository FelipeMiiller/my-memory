package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DefaultLogTail is the default number of log lines `mem logs`
// returns when --limit is omitted.
const DefaultLogTail = 50

// runLogsCommand is the entry point for `mem logs <worker>`
// (tasks.md T8 done-when #5). Behavior:
//   - Opens <memory-dir>/logs/supervisor.jsonl.
//   - Returns the last <limit> entries whose event payload mentions
//     the requested worker_id (default 50).
//   - When no filter is given, prints every entry (used by
//     operators to skim audit history across all workers).
func runLogsCommand(ctx context.Context, args []string, memoryDir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.SetOutput(stderr)
	memoryDirFlag := fs.String("memory-dir", memoryDir, "vault root")
	limit := fs.Int("limit", DefaultLogTail, "max number of entries to return")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *memoryDirFlag == "" {
		*memoryDirFlag = ".memory"
	}

	logPath := filepath.Join(*memoryDirFlag, "logs", "supervisor.jsonl")
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(stdout, "(no log file)")
			return 0
		}
		fmt.Fprintf(stderr, "mem logs: open %s: %v\n", logPath, err)
		return 1
	}
	defer f.Close()

	workerFilter := ""
	if fs.NArg() > 0 {
		workerFilter = fs.Arg(0)
	}

	// Ring buffer of the last <limit> matching entries. We read
	// the whole file (small in practice) into memory — the audit
	// log is rotated at 100 MiB by T9 so an in-memory tail is
	// fine.
	type logEntry struct {
		eventType  string
		workerID   string
		timestamp  string
		rawPayload json.RawMessage
	}
	var kept []logEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1<<16), 1<<24)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var env struct {
			EventType   string          `json:"event_type"`
			AggregateID string          `json:"aggregate_id"`
			CreatedAt   string          `json:"created_at"`
			Payload     json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(line, &env); err != nil {
			continue
		}
		if workerFilter != "" && env.AggregateID != workerFilter {
			continue
		}
		kept = append(kept, logEntry{
			eventType:  env.EventType,
			workerID:   env.AggregateID,
			timestamp:  env.CreatedAt,
			rawPayload: env.Payload,
		})
		if len(kept) > *limit {
			// Drop oldest — we want the LAST `limit` entries.
			kept = kept[len(kept)-*limit:]
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "mem logs: scan: %v\n", err)
		return 1
	}

	for _, e := range kept {
		summary := summarizePayload(e.rawPayload)
		fmt.Fprintf(stdout, "%s %s worker=%s %s\n",
			e.timestamp, e.eventType, e.workerID, summary)
	}
	return 0
}

// summarizePayload renders the payload blob as a one-line key=value
// summary so operators can grep without jq. Falls back to the raw
// blob when the payload isn't a JSON object.
func summarizePayload(payload json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		return string(payload)
	}
	parts := make([]string, 0, len(obj))
	for k, v := range obj {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, " ")
}
