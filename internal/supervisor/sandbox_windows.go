//go:build windows

package supervisor

import (
	"os/exec"
)

// applySandboxPlatform on Windows is a no-op + silent warning
// (logged at the call site via the cmd's stderr). Windows lacks
// portable unshare/prctl/setrlimit hooks for restricting
// subprocesses from a Go binary. The shipped mymemoryd is
// single-host on Windows; operators who need a hard sandbox
// should run the worker under WSL2 or a container runtime, both
// of which are out of scope for T9.
func applySandboxPlatform(cmd *exec.Cmd, egressPolicy string) error {
	_ = cmd
	_ = egressPolicy
	return nil
}
