package policy

import (
	"testing"
)

func TestEngine_BalancedDefaultsAllowOwner(t *testing.T) {
	eng := NewEngine(defaults())
	if got := eng.Decide("user:owner", "/tmp/foo.md", OpWrite); got != Allow {
		t.Errorf("balanced owner write = %v, want Allow", got)
	}
}

func TestEngine_BalancedDefaultsAllowSystem(t *testing.T) {
	eng := NewEngine(defaults())
	if got := eng.Decide("system:writer", "/tmp/foo.md", OpWrite); got != Allow {
		t.Errorf("balanced system write = %v, want Allow", got)
	}
}

func TestEngine_BalancedRequiresApprovalForAgent(t *testing.T) {
	eng := NewEngine(defaults())
	if got := eng.Decide("agent:external", "/tmp/foo.md", OpWrite); got != RequireApproval {
		t.Errorf("balanced agent write = %v, want RequireApproval", got)
	}
}

func TestEngine_EmptyActorDeny(t *testing.T) {
	eng := NewEngine(defaults())
	if got := eng.Decide("", "/tmp/foo.md", OpWrite); got != Deny {
		t.Errorf("empty actor = %v, want Deny (fail-closed)", got)
	}
}

func TestEngine_StrictProfileAllowsOnlyOwner(t *testing.T) {
	p := Profile{
		SchemaVersion: 1,
		DefaultOwner:  "user:owner",
		Rules: []Rule{
			{MatchActor: "user:owner", Decision: "allow"},
			// catch-all forces RequireApproval for everything else
			{MatchActor: "*", Decision: "require_approval"},
		},
	}
	eng := NewEngine(p)

	cases := []struct {
		actor string
		want  Decision
	}{
		{"user:owner", Allow},
		{"user:other", RequireApproval},
		{"agent:external", RequireApproval},
		{"system:foo", RequireApproval},
	}
	for _, tc := range cases {
		if got := eng.Decide(tc.actor, "/x", OpWrite); got != tc.want {
			t.Errorf("%s → %v, want %v", tc.actor, got, tc.want)
		}
	}
}

func TestEngine_PermissiveDevAllowsEverything(t *testing.T) {
	p := Profile{
		SchemaVersion: 1,
		DefaultOwner:  "user:owner",
		Rules: []Rule{
			{MatchActor: "*", Decision: "allow"},
		},
	}
	eng := NewEngine(p)

	for _, actor := range []string{"user:owner", "agent:external", "system:writer", ""} {
		if actor == "" {
			continue // empty actor is hard-deny per fail-closed
		}
		if got := eng.Decide(actor, "/x", OpWrite); got != Allow {
			t.Errorf("%s → %v, want Allow", actor, got)
		}
	}
}

func TestEngine_VaultScopeForcesApproval(t *testing.T) {
	p := Profile{
		SchemaVersion: 1,
		DefaultOwner:  "user:owner",
		VaultScope:    "docs/**",
		Rules: []Rule{
			{MatchActor: "user:owner", Decision: "allow"},
		},
	}
	eng := NewEngine(p)

	if got := eng.Decide("user:owner", "docs/foo.md", OpWrite); got != Allow {
		t.Errorf("in-scope write = %v, want Allow", got)
	}
	if got := eng.Decide("user:owner", "/etc/passwd", OpWrite); got != RequireApproval {
		t.Errorf("out-of-scope write = %v, want RequireApproval", got)
	}
}

func TestEngine_PathRuleOverride(t *testing.T) {
	p := Profile{
		SchemaVersion: 1,
		DefaultOwner:  "user:owner",
		Rules: []Rule{
			{MatchActor: "user:owner", Decision: "allow"},
			{MatchActor: "*", MatchPath: "secret/*", Decision: "deny"},
		},
	}
	eng := NewEngine(p)

	if got := eng.Decide("agent:external", "docs/foo.md", OpWrite); got != Deny {
		t.Errorf("non-secret non-owner = %v, want Deny (no rule, fail-closed)", got)
	}
	if got := eng.Decide("agent:external", "secret/key.pem", OpWrite); got != Deny {
		t.Errorf("secret path non-owner = %v, want Deny", got)
	}
}

func TestEngine_UnknownDecisionStringFailsClosed(t *testing.T) {
	p := Profile{
		SchemaVersion: 1,
		DefaultOwner:  "user:owner",
		Rules: []Rule{
			{MatchActor: "user:owner", Decision: "maybe"}, // typo
		},
	}
	eng := NewEngine(p)
	if got := eng.Decide("user:owner", "/x", OpWrite); got != Deny {
		t.Errorf("unknown decision = %v, want Deny (fail-closed)", got)
	}
}

func TestEngine_OpFilter(t *testing.T) {
	p := Profile{
		SchemaVersion: 1,
		DefaultOwner:  "user:owner",
		Rules: []Rule{
			{MatchActor: "user:owner", MatchOp: "write", Decision: "allow"},
			{MatchActor: "*", MatchOp: "read", Decision: "deny"},
		},
	}
	eng := NewEngine(p)

	if got := eng.Decide("user:owner", "/x", OpWrite); got != Allow {
		t.Errorf("owner write = %v, want Allow", got)
	}
	if got := eng.Decide("user:owner", "/x", OpRead); got != Deny {
		t.Errorf("owner read = %v, want Deny (read rule matched)", got)
	}
}

func TestActorMatch(t *testing.T) {
	cases := []struct {
		pattern, actor string
		want           bool
	}{
		{"", "user:owner", true},
		{"*", "user:owner", true},
		{"user:owner", "user:owner", true},
		{"user:owner", "user:other", false},
		{"agent:*", "agent:external", true},
		{"agent:*", "user:owner", false},
		{"user:*", "user:owner", true},
	}
	for _, tc := range cases {
		if got := actorMatch(tc.pattern, tc.actor); got != tc.want {
			t.Errorf("actorMatch(%q, %q) = %v, want %v", tc.pattern, tc.actor, got, tc.want)
		}
	}
}

func TestGlobMatch(t *testing.T) {
	cases := []struct {
		pattern, target string
		want            bool
	}{
		{"", "anything", true},
		{"docs/*", "docs/foo.md", true},
		{"docs/*", "etc/foo.md", false},
		{"docs/**", "docs/sub/bar.md", true},
		{"secret/*", "secret/key.pem", true},
		{"secret/*", "docs/key.pem", false},
	}
	for _, tc := range cases {
		if got := globMatch(tc.pattern, tc.target); got != tc.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", tc.pattern, tc.target, got, tc.want)
		}
	}
}

func TestDecision_String(t *testing.T) {
	cases := map[Decision]string{
		Allow:            "allow",
		RequireApproval:  "require_approval",
		Deny:             "deny",
	}
	for d, want := range cases {
		if got := d.String(); got != want {
			t.Errorf("Decision(%d).String() = %q, want %q", int(d), got, want)
		}
	}
}
