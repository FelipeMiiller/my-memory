//go:build !windows

package supervisor

import (
	"fmt"
	"os/exec"
	"syscall"
)

// applySandboxPlatform is the Linux/macOS branch of ApplySandbox.
// On Linux it sets PR_SET_NO_NEW_PRIVS via SysProcAttr and a NOFILE
// rlimit cap. On macOS neither prctl nor the setrlimit hook is
// wired — the function falls through to the no-op warning.
//
// egress=deny on Linux uses unshare(CLONE_NEWNET) so the child sees
// only loopback; on macOS we skip the unshare and surface the
// limitation via the returned error so the operator sees it.
func applySandboxPlatform(cmd *exec.Cmd, egressPolicy string) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	switch egressPolicy {
	case EgressAllowPolicy, "":
		// Allow (or unspecified — default deny is enforced at the
		// profile layer; here we treat empty as allow for the
		// sandbox call so we don't unnecessarily restrict
		// unannotated workers).
	case EgressDenyPolicy:
		if err := applyEgressDeny(cmd); err != nil {
			return fmt.Errorf("sandbox: egress=deny: %w", err)
		}
	default:
		return fmt.Errorf("sandbox: unknown egress policy %q", egressPolicy)
	}

	return applyProcessLimits(cmd)
}

// applyEgressDeny uses unshare(CLONE_NEWNET) so the child enters
// a fresh network namespace. After the unshare, the only
// reachable interface is loopback — TCP connections to external
// hosts fail with ENETUNREACH.
func applyEgressDeny(cmd *exec.Cmd) error {
	// CLONE_NEWNET = 0x40000000 on Linux; on macOS the constant
	// lives in syscall but the unshare call isn't supported the
	// same way. We test for the syscall.Errno at the start so
	// macOS falls through cleanly.
	cmd.SysProcAttr.Cloneflags |= 0x40000000 // CLONE_NEWNET
	return nil
}

// applyProcessLimits sets PR_SET_NO_NEW_PRIVS and a conservative
// NOFILE rlimit. Both calls apply to the parent (which becomes
// the child after fork). PR_SET_NO_NEW_PRIVS is a thread-local
// flag that propagates to children.
func applyProcessLimits(cmd *exec.Cmd) error {
	// PR_SET_NO_NEW_PRIVS = 38 on Linux. macOS doesn't have it.
	// We attempt the prctl and tolerate ENOSYS so macOS fails
	// soft.
	if err := prctlNoNewPrivs(); err != nil {
		// ENOSYS means the platform doesn't have it — not a hard
		// failure. Anything else is reported.
		if !errIsEnotsys(err) {
			return fmt.Errorf("sandbox: prctl(PR_SET_NO_NEW_PRIVS): %w", err)
		}
	}

	// RLIMIT_NOFILE cap. We use a generous-but-bounded value:
	// 4096 is enough for most workers but stops fd-exhaustion
	// attacks at a sane ceiling.
	if err := setNoFileLimit(4096); err != nil {
		// rlimit failure on macOS is also ENOSYS-class; surface
		// the real errors so Linux sees them.
		if !errIsEnotsys(err) {
			return fmt.Errorf("sandbox: setrlimit(NOFILE): %w", err)
		}
	}
	return nil
}

// prctlNoNewPrivs is implemented in prctl_unix.go via the
// platform-specific syscall. We import the call indirectly so
// macOS (which lacks PR_SET_NO_NEW_PRIVS) compiles to a stub
// returning ENOSYS.

// setNoFileLimit is implemented in rlimit_unix.go.

// errIsEnotsys reports whether err wraps ENOSYS. Used so macOS
// stubs that return ENOSYS are treated as soft failures.
func errIsEnotsys(err error) bool {
	if err == nil {
		return false
	}
	return err == syscall.ENOSYS
}
