// Package supervisor implements the mymemoryd core: lock acquisition,
// worker lifecycle, signal-driven shutdown, and IPC bootstrap.
//
// The package owns three responsibilities today (see ADR-042 and the
// mymemoryd-supervisor feature spec):
//
//   - Lock acquisition: AcquireLock / ReleaseLock over
//     .memory/supervisor.lock with PID + timestamp staleness checks.
//   - Lifecycle skeleton: Manager.Run blocks on context cancellation
//     and emits no events (T1). T2 wires Start/Stop/HealthCheck,
//     T3 adds restart with exponential backoff.
//   - Profile path: T4 introduces YAML parsing for .memory/profiles/<name>.yaml.
//
// All functions are safe for concurrent use unless explicitly stated.
// Tests live in `internal/supervisor/*_test.go` co-located with the
// implementation (see .agents/rules/always-test.md).
package supervisor
