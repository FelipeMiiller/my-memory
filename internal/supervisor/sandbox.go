package supervisor

import (
	"fmt"
	"os/exec"
)

// Egress policy strings. The supervisor refuses to start any
// worker whose egress declaration doesn't match one of these —
// see Profile.Validate. The default (empty string) is deny so
// operators must opt in to network access explicitly.
const (
	EgressDenyPolicy  = "deny"
	EgressAllowPolicy = "allow"
)

// ApplySandbox attaches the best-effort sandbox surface to cmd.
// On Linux it sets PR_SET_NO_NEW_PRIVS (so the worker can't
// re-acquire privileges via setuid binaries) and the RLIMIT_NOFILE
// cap (so a buggy worker can't exhaust the supervisor's fd pool).
// On macOS and Windows ApplySandbox is a no-op that returns nil
// and surfaces the limitation via stderr-wired logging at the
// call site.
//
// egressPolicy controls the network surface:
//   - EgressDenyPolicy (default): Linux uses unshare(CLONE_NEWNET)
//     so the worker sees only loopback. macOS/Windows: no-op +
//     warning (no portable equivalent).
//   - EgressAllowPolicy: no-op — the worker shares the host's
//     network namespace.
//
// Errors from the underlying syscalls are returned so the caller
// (StartAll) can decide whether to fail-closed (default) or
// degrade gracefully.
func ApplySandbox(cmd *exec.Cmd, egressPolicy string) error {
	if cmd == nil {
		return nil
	}
	// Validate the egress policy here so the contract is the
	// same on every platform — Windows must reject unknown
	// policies too, even though it has no platform-specific
	// sandbox surface to apply.
	switch egressPolicy {
	case "", EgressAllowPolicy, EgressDenyPolicy:
		// ok
	default:
		return fmt.Errorf("sandbox: unknown egress policy %q", egressPolicy)
	}
	return applySandboxPlatform(cmd, egressPolicy)
}
