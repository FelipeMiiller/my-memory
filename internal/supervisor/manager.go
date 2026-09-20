package supervisor

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"sync"

	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
)

// Manager is the long-lived supervisor that owns the worker pool.
// T1 ships the struct + lifecycle hooks as a skeleton; T2 wires the
// Start/Stop/HealthCheck methods, T3 adds restart with backoff,
// T4 wires the profile YAML manifest.
type Manager struct {
	mu sync.Mutex

	profile   string
	memoryDir string

	// workers holds the worker manifest declared in the active
	// profile. Populated by LoadManifest (T4); empty for T1.
	workers map[string]WorkerSpec

	// active holds live Worker handles (started subprocesses). Empty
	// until T2 ships Start().
	active map[string]*Worker

	// db is the SQLite handle used for the event_log (when SetLog
	// has been called). nil when the supervisor runs without an event
	// sink — useful for T1-style smoke tests.
	db *sql.DB

	// log is the event_runtime.Log bound to db. nil when db is nil.
	log *event_runtime.Log

	// done is closed when Run() returns. The entrypoint waits on
	// it to coordinate lock release and shutdown.
	done chan struct{}
}

// WorkerSpec describes a single worker declared in the active
// profile. T2 extends the T1 placeholder with Command/Args/Env so
// the lifecycle hooks can launch the subprocess. Required / Egress
// / Config land in T4 + T5 with the YAML parser.
type WorkerSpec struct {
	Name    string
	Command string
	Args    []string
	Env     []string
}

// Worker holds the live handle to a started worker subprocess.
// T2 implements the subprocess plumbing (start, watch, stop).
//
// expectedExit is set by Stop() before sending SIGTERM so the
// watcher can distinguish graceful shutdown (no heartbeat_lost)
// from a crash (heartbeat_lost emitted).
type Worker struct {
	ID           string
	PID          int
	spec         WorkerSpec
	cmd          *exec.Cmd
	doneCh       chan struct{}
	expectedExit bool
}

// NewManager constructs a Manager for the given memory directory and
// profile name. It does NOT acquire the lock — call AcquireLock
// before NewManager so the lock ownership is unambiguous and the
// deferred ReleaseLock in main() covers every exit path.
//
// db may be nil; when non-nil the manager will emit lifecycle events
// (worker.started, worker.stopped, worker.heartbeat_lost) into the
// event_log via log.
func NewManager(memoryDir, profile string, db *sql.DB) *Manager {
	m := &Manager{
		profile:   profile,
		memoryDir: memoryDir,
		workers:   make(map[string]WorkerSpec),
		active:    make(map[string]*Worker),
		db:        db,
		done:      make(chan struct{}),
	}
	if db != nil {
		m.log = event_runtime.NewLog(db)
	}
	return m
}

// Run blocks until ctx is cancelled. It does NOT install signal
// handlers — the binary's main() owns those and wires them via
// signal.NotifyContext. Returns ctx.Err().
//
// T1 only blocks on ctx; T2 wires in the worker goroutines and the
// heartbeat supervisor. The method is safe to call exactly once per
// Manager; subsequent calls return ctx.Err() immediately.
func (m *Manager) Run(ctx context.Context) error {
	defer close(m.done)
	<-ctx.Done()
	return ctx.Err()
}

// Done returns a channel that is closed when Run() returns. Used
// by the entrypoint to wait for the supervisor loop to wind down
// before releasing the lock and exiting.
func (m *Manager) Done() <-chan struct{} {
	return m.done
}

// ActiveWorkers returns the IDs of currently-running workers in
// arbitrary order. T1 returns an empty slice (no workers started
// yet); T2 populates this from the live subprocess map.
func (m *Manager) ActiveWorkers() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.active))
	for id := range m.active {
		out = append(out, id)
	}
	return out
}

// RegisterSpec adds a worker declaration to the manifest. T4
// replaces this with YAML-driven LoadManifest; T2 uses it directly
// in tests to inject fake workers without touching the filesystem.
func (m *Manager) RegisterSpec(spec WorkerSpec) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workers[spec.Name] = spec
}

// Spec returns the registered WorkerSpec by name. The bool is false
// when no such worker is declared (e.g., caller typo).
func (m *Manager) Spec(name string) (WorkerSpec, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.workers[name]
	return s, ok
}

// Profile returns the active profile name as passed to NewManager.
// Used by the entrypoint for log lines and by `mem status`.
func (m *Manager) Profile() string {
	return m.profile
}

// String returns a debug-friendly identifier for log lines.
func (m *Manager) String() string {
	return fmt.Sprintf("Manager{profile=%s dir=%s}", m.profile, m.memoryDir)
}
