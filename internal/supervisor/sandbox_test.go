package supervisor

import (
	"os/exec"
	"testing"
)

// TestSandbox_ApplyOnNilCmd is a no-op smoke test: passing a nil
// cmd must not panic, must return nil. Production callers may
// invoke ApplySandbox defensively before the cmd is constructed.
func TestSandbox_ApplyOnNilCmd(t *testing.T) {
	if err := ApplySandbox(nil, ""); err != nil {
		t.Fatalf("ApplySandbox(nil): %v", err)
	}
}

// TestSandbox_ApplyUnknownEgressReturnsError asserts the
// fail-closed contract: unknown egress policies are an error,
// not silently coerced to deny/allow.
func TestSandbox_ApplyUnknownEgressReturnsError(t *testing.T) {
	cmd := exec.Command("/bin/true")
	if err := ApplySandbox(cmd, "maybe"); err == nil {
		t.Fatal("expected error on unknown egress policy")
	}
}

// TestSandbox_ApplyEgressAllowDoesNotError is the contract test
// for "allow": on every supported platform ApplySandbox must
// return nil (it's a no-op for the egress side; the rlimit+prctl
// best-effort is asserted in the platform-specific test files).
func TestSandbox_ApplyEgressAllowDoesNotError(t *testing.T) {
	cmd := exec.Command("/bin/true")
	if err := ApplySandbox(cmd, EgressAllowPolicy); err != nil {
		t.Fatalf("ApplySandbox: %v", err)
	}
}
