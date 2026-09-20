package supervisor

import (
	"context"
	"fmt"
	"sync"
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

	// done is closed when Run() returns. The entrypoint waits on
	// it to coordinate lock release and shutdown.
	done chan struct{}
}

// WorkerSpec describes a single worker declared in the active
// profile. T1 ships the placeholder; the full struct
// (Command, Args, Required, Egress, Config) lands in T4 with the
// YAML parser.
type WorkerSpec struct {
	Name string
}

// Worker holds the live handle to a started worker subprocess.
// T2 implements the subprocess plumbing (start, heartbeat, exit).
type Worker struct {
	ID  string
	PID int
}

// NewManager constructs a Manager for the given memory directory and
// profile name. It does NOT acquire the lock — call AcquireLock
// before NewManager so the lock ownership is unambiguous and the
// deferred ReleaseLock in main() covers every exit path.
func NewManager(memoryDir, profile string) *Manager {
	return &Manager{
		profile:   profile,
		memoryDir: memoryDir,
		workers:   make(map[string]WorkerSpec),
		active:    make(map[string]*Worker),
		done:      make(chan struct{}),
	}
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

// Profile returns the active profile name as passed to NewManager.
// Used by the entrypoint for log lines and by `mem status`.
func (m *Manager) Profile() string {
	return m.profile
}

// String returns a debug-friendly identifier for log lines.
func (m *Manager) String() string {
	return fmt.Sprintf("Manager{profile=%s dir=%s}", m.profile, m.memoryDir)
}
