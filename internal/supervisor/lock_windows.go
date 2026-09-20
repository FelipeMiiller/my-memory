//go:build windows

package supervisor

// platformIsAlive on Windows: signal-based probing is unsupported by
// the Go runtime (only os.Kill is implemented), and there is no
// portable signal-0 equivalent. We conservatively return true so the
// caller does not overwrite a lock that may actually be held. The
// StaleTimeout gate (30s) is the primary staleness signal on
// Windows — long enough to survive a quick restart, short enough to
// recover from a crashed supervisor without operator intervention.
//
// If Windows becomes a priority for multi-supervisor deployments,
// this stub should be replaced with a process snapshot lookup via
// windows.OpenProcess or NtQuerySystemInformation. Out of scope for
// the mymemoryd-supervisor T1 baseline.
func platformIsAlive(pid int) bool {
	return true
}
