//go:build !windows

package supervisor

import "syscall"

// setNoFileLimit raises RLIMIT_NOFILE to cap on the supervisor
// process. The cap propagates to children (fork inherits the
// limit) so any worker subprocess can't open more than cap fds.
//
// On macOS the syscall exists but the operation is a no-op for
// the sandbox goal (macOS workers can still raise the limit
// themselves via setrlimit inside the child). T9 documents this
// limitation; the Windows path is a no-op stub in
// sandbox_windows.go.
func setNoFileLimit(cap uint64) error {
	var lim syscall.Rlimit
	lim.Cur = cap
	lim.Max = cap
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim)
}
