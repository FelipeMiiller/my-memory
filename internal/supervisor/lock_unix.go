//go:build !windows

package supervisor

import "syscall"

// platformIsAlive returns true when the OS reports the PID is held
// by some process (even a zombie). Uses signal 0 which is a no-op
// probe that succeeds iff the process exists.
func platformIsAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
