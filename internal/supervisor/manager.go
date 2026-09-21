package supervisor

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"sync"
	"time"

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
// the lifecycle hooks can launch the subprocess. T3 adds
// RestartPolicy so crashes trigger automatic restart with
// exponential backoff. T4 completes the YAML contract with
// Required / Egress / Config so profiles round-trip cleanly.
//
// YAML tags are explicit so the T4 strict decoder rejects
// unknown fields without ambiguity. Keep tag names snake_case
// to match the spec (ADR-042 §profile-yaml).
type WorkerSpec struct {
	Name          string         `yaml:"name"`
	Command       string         `yaml:"command"`
	Args          []string       `yaml:"args,omitempty"`
	Env           []string       `yaml:"env,omitempty"`
	Required      bool           `yaml:"required,omitempty"`
	Egress        string         `yaml:"egress,omitempty"`
	Config        map[string]any `yaml:"config,omitempty"`
	RestartPolicy RestartPolicy  `yaml:"restart_policy,omitempty"`
}

// RestartPolicy configures automatic restart behavior for a worker.
// T3 baseline: exponential backoff with jitter (ADR-043 §6 retry
// contract). MaxRetries=0 disables restart entirely; positive
// values bound the number of restart attempts before the supervisor
// emits supervisor.worker_failed and gives up.
//
// BaseDelay and MaxDelay are not strict across platforms — the
// jitter can push individual intervals slightly outside the [base,
// max] window, which is intentional (thundering-herd avoidance,
// ADR-043 §6.2).
//
// YAML tags follow the contract documented in ADR-042 §profile-yaml
// and are kept stable so future Profile.SchemaVersion > 1 bumps can
// land without a migration step.
type RestartPolicy struct {
	MaxRetries int           `yaml:"max_retries,omitempty"`
	BaseDelay  time.Duration `yaml:"base_delay,omitempty"`
	MaxDelay   time.Duration `yaml:"max_delay,omitempty"`
	Jitter     float64       `yaml:"jitter,omitempty"`
}

// DefaultRestartPolicy returns the T3 baseline used when a worker
// spec omits a policy. Matches ADR-043 §6 defaults.
func DefaultRestartPolicy() RestartPolicy {
	return RestartPolicy{
		MaxRetries: 5,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		Jitter:     0.2,
	}
}

// Worker holds the live handle to a started worker subprocess.
// T2 implements the subprocess plumbing (start, watch, stop).
// T3 adds restart bookkeeping so the watcher can schedule a
// restart on unexpected exit.
//
// expectedExit is set by Stop() before sending SIGTERM so the
// watcher can distinguish graceful shutdown (no heartbeat_lost,
// no restart) from a crash (heartbeat_lost + restart attempt).
type Worker struct {
	ID           string
	PID          int
	spec         WorkerSpec
	cmd          *exec.Cmd
	doneCh       chan struct{}
	expectedExit bool

	// restartCount tracks how many times this worker has been
	// auto-restarted since the initial Start(). Persisted across
	// restarts via the *Worker pointer that watchWorker carries;
	// if Start() ever re-allocates we lose the count, so the
	// restart path always reuses the same Worker struct.
	restartCount int

	// lastRestart is the wall-clock time of the most recent
	// successful restart attempt. Used by `mem status` and by
	// tests that want to measure backoff timing.
	lastRestart time.Time

	// failed is set to true once the restart policy is exhausted;
	// the watcher emits supervisor.worker_failed exactly once and
	// stops attempting further restarts.
	failed bool
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

// StartAll starts each worker in specs sequentially, honoring
// Required vs optional semantics (ADR-042 §DR-2 + spec P3 AC 3/4).
// Returns nil when every worker started successfully or when only
// optional workers failed. Returns an error wrapping
// ErrRequiredFailed when any required worker fails to start —
// callers should map that to exit code 2.
//
// Required workers are evaluated strictly: when one fails, the
// remaining workers are NOT started (fail-closed). Optional worker
// failures emit worker.optional_failed and StartAll continues with
// the next spec.
//
// Each successful Start emits worker.started; each optional
// failure emits worker.optional_failed; each required failure
// emits supervisor.required_failed. The 30s ready timeout
// referenced in the spec is enforced by the IPC handshake in T6/T7
// (not by StartAll — Start only validates the binary exists).
func (m *Manager) StartAll(ctx context.Context, specs []WorkerSpec) error {
	for _, spec := range specs {
		if _, err := m.Start(ctx, spec); err != nil {
			if spec.Required {
				m.emitEvent(ctx, EventSupervisorRequired, spec.Name, map[string]any{
					"command": spec.Command,
					"error":   err.Error(),
					"ts":      time.Now().UTC().Format(time.RFC3339Nano),
				})
				return fmt.Errorf("%w: id=%s command=%s: %v",
					ErrRequiredFailed, spec.Name, spec.Command, err)
			}
			// Optional failure: log and continue.
			m.emitEvent(ctx, EventWorkerOptionalFailed, spec.Name, map[string]any{
				"command": spec.Command,
				"error":   err.Error(),
				"ts":      time.Now().UTC().Format(time.RFC3339Nano),
			})
		}
	}
	return nil
}
