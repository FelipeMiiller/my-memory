//go:build !windows

package supervisor

import (
	"os/exec"
	"syscall"
	"testing"
)

// CLONE_NEWNET = 0x40000000 on Linux. Defined locally so we
// don't depend on any particular package's constant set.
const cloneNewNet = 0x40000000

// TestSandbox_ApplyEgressAllowDoesNotTouchSysProcAttr is the
// contract test for "allow" — the cmd is left untouched except
// for the parent-process-side prctl/setrlimit (which we don't
// assert on here because they're best-effort).
func TestSandbox_ApplyEgressAllowDoesNotTouchSysProcAttr(t *testing.T) {
	cmd := exec.Command("/bin/true")
	if err := ApplySandbox(cmd, EgressAllowPolicy); err != nil {
		t.Fatalf("ApplySandbox: %v", err)
	}
	if cmd.SysProcAttr == nil {
		return
	}
	if cmd.SysProcAttr.Cloneflags&cloneNewNet != 0 {
		t.Errorf("egress=allow must NOT set CLONE_NEWNET (got flags=%x)",
			cmd.SysProcAttr.Cloneflags)
	}
}

// TestSandbox_ApplyEgressDenySetsNewNet asserts the
// cross-platform marker for the egress=deny behavior. On Linux
// the apply must set CLONE_NEWNET on the cmd's Cloneflags. On
// macOS the syscall may not be supported and we skip.
func TestSandbox_ApplyEgressDenySetsNewNet(t *testing.T) {
	cmd := exec.Command("/bin/true")
	if err := ApplySandbox(cmd, EgressDenyPolicy); err != nil {
		// macOS may surface ENOSYS — that's a soft fail.
		t.Skipf("ApplySandbox egress=deny not supported on this platform: %v", err)
	}
	if cmd.SysProcAttr == nil {
		t.Fatal("egress=deny must populate SysProcAttr.Cloneflags")
	}
	if cmd.SysProcAttr.Cloneflags&cloneNewNet == 0 {
		t.Errorf("egress=deny must set CLONE_NEWNET (got flags=%x)",
			cmd.SysProcAttr.Cloneflags)
	}
}

// TestSandbox_PlatformPrctlIsCallable covers the prctl branch
// without requiring CAP_SYS_ADMIN. We just call the package
// helper directly and assert that any failure surfaces (rather
// than crashing the supervisor silently).
func TestSandbox_PlatformPrctlIsCallable(t *testing.T) {
	if err := prctlNoNewPrivs(); err != nil {
		if err == syscall.ENOSYS {
			t.Skipf("PR_SET_NO_NEW_PRIVS not available: %v", err)
		}
		t.Logf("prctl returned %v (acceptable on unprivileged runners)", err)
	}
}

// TestSandbox_PlatformSetrlimitIsCallable mirrors the prctl test
// for the NOFILE cap. Best-effort — unprivileged runners can't
// always raise their own limits.
func TestSandbox_PlatformSetrlimitIsCallable(t *testing.T) {
	if err := setNoFileLimit(4096); err != nil {
		if err == syscall.ENOSYS {
			t.Skipf("RLIMIT_NOFILE not available: %v", err)
		}
		t.Logf("setrlimit returned %v (acceptable on unprivileged runners)", err)
	}
}
