//go:build !windows

package supervisor

import "syscall"

// prctlNoNewPrivs calls prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) on
// Linux. PR_SET_NO_NEW_PRIVS = 38. On macOS the prctl syscall is
// not present so the call returns ENOSYS — callers treat that as
// a soft failure via errIsEnotsys.
//
// The call is made in the supervisor's goroutine (parent), not
// inside the child, because cmd.SysProcAttr doesn't expose a
// per-syscall hook for PR_SET_NO_NEW_PRIVS. The flag is inherited
// across fork() on Linux, so the child gets the same restriction.
func prctlNoNewPrivs() error {
	const prSetNoNewPrivs = 38
	_, _, errno := syscall.Syscall6(
		syscall.SYS_PRCTL,
		prSetNoNewPrivs,
		1, 0, 0, 0, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
