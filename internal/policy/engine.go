// Package policy implements the ADR-050 §DR-3 policy engine: a
// configuration-driven gate that decides whether a write should be
// allowed, denied, or require explicit approval before it lands.
//
// The engine is intentionally narrow — it has no knowledge of SQLite,
// the writer, or the event_runtime envelope. It receives the
// caller's actor identity, the target path, and the operation kind,
// and returns a Decision. The writer (T10) wires the engine into
// the write flow and translates the Decision into either a normal
// commit, an error, or an approval.requested event.
//
// Three profiles ship by default (T9):
//   - strict:             every non-owner actor → RequireApproval
//   - balanced:           writes within the vault scope proceed;
//     writes outside the scope require approval
//   - permissive-dev:     every write is allowed (development only)
//
// The default profile (used when no policy file is configured) is
// balanced — fail-safe posture per ADR-050 §DR-2.
package policy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Decision is the outcome of Engine.Decide.
type Decision int

const (
	// Allow: the write may proceed without further checks.
	Allow Decision = iota
	// RequireApproval: the write is potentially risky; emit
	// approval.requested and block until approval.granted arrives.
	RequireApproval
	// Deny: the write MUST NOT happen; surface ErrPolicyDenied.
	Deny
)

// String returns a stable lowercase name for log output and audit
// payloads. The string is part of the wire format.
func (d Decision) String() string {
	switch d {
	case Allow:
		return "allow"
	case RequireApproval:
		return "require_approval"
	case Deny:
		return "deny"
	default:
		return fmt.Sprintf("unknown(%d)", int(d))
	}
}

// Op is the operation kind the policy decides on. Kept narrow so
// future ops (read, delete, exec) can be added without breaking
// existing profiles.
type Op string

const (
	OpWrite Op = "write"
	OpRead  Op = "read"
)

// Profile is the YAML shape stored in `.memory/policy/<name>.yaml`.
// Field names match the policy spec exactly so reviewers can grep
// between docs and config without translation.
type Profile struct {
	SchemaVersion int    `yaml:"schema_version"`
	DefaultOwner  string `yaml:"default_owner"`
	VaultScope    string `yaml:"vault_scope"` // glob; paths matching are "in scope"
	Rules         []Rule `yaml:"rules"`
}

// Rule is one entry in a profile's decision table. First match wins.
type Rule struct {
	// MatchActor is a prefix match against the actor string
	// ("user:owner", "agent:*", "*"). Empty means "any actor".
	MatchActor string `yaml:"match_actor"`
	// MatchPath is a glob against the target path. Empty means
	// "any path". Vault-scope checks live in the engine itself;
	// Rule paths are independent of the vault scope.
	MatchPath string `yaml:"match_path"`
	// MatchOp is one of the Op constants as a string. Empty means
	// "any op".
	MatchOp string `yaml:"match_op"`
	// Decision is "allow" | "require_approval" | "deny".
	Decision string `yaml:"decision"`
}

// Engine holds the parsed profile and is safe for concurrent use
// (Decide is read-only after Load).
type Engine struct {
	mu      sync.RWMutex
	profile Profile
	path    string
}

// LoadDefault loads the policy file at the default location. If the
// file is missing or unreadable, returns an Engine with the balanced
// profile loaded from defaults() and the error (callers decide
// whether to log + proceed or fail-closed per ADR-050 §DR-2).
//
// The default location is `.memory/policy/active.yaml` — a small
// symlink or copy of the actual profile yaml. CLI tools manage
// the symlink via `mem profiles use <name>` (out of scope here).
func LoadDefault() (*Engine, error) {
	eng, err := Load(".memory/policy/active.yaml")
	if err == nil {
		return eng, nil
	}
	// Fallback to balanced defaults; surface the load error so
	// operators see "policy file missing" instead of "policy OK".
	return NewEngine(defaults()), err
}

// Load reads + parses the YAML at path and returns a fresh Engine.
// Returns an error on malformed YAML or unreadable file.
func Load(path string) (*Engine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("policy: read %s: %w", path, err)
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("policy: parse %s: %w", path, err)
	}
	if p.SchemaVersion != 1 {
		return nil, fmt.Errorf("policy: unsupported schema_version %d (want 1)", p.SchemaVersion)
	}
	return &Engine{profile: p, path: path}, nil
}

// NewEngine wraps an already-parsed Profile (useful for tests).
func NewEngine(p Profile) *Engine {
	return &Engine{profile: p, path: "<inline>"}
}

// defaults returns the balanced profile used when no policy file is
// configured. Mirrors `.memory/policy/balanced.yaml`.
func defaults() Profile {
	return Profile{
		SchemaVersion: 1,
		DefaultOwner:  "user:owner",
		VaultScope:    "",
		Rules: []Rule{
			{MatchActor: "user:owner", Decision: "allow"},
			{MatchActor: "system:*", Decision: "allow"},
			{MatchActor: "*", Decision: "require_approval"},
		},
	}
}

// Path returns the file the engine was loaded from (or "<inline>").
func (e *Engine) Path() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.path
}

// Decide evaluates (actor, path, op) against the engine's rules.
// First rule that matches all three fields wins. The engine also
// checks vault scope: a write OUTSIDE the vault scope (when one is
// configured) always returns RequireApproval even if a rule said
// Allow — the rule's MatchPath is independent of the vault scope
// (rules are the operator's fine-grained exceptions).
func (e *Engine) Decide(actor, targetPath string, op Op) Decision {
	e.mu.RLock()
	p := e.profile
	e.mu.RUnlock()

	if actor == "" {
		return Deny
	}

	// 1. Vault scope check — out-of-scope paths always need approval
	// (per spec P4-AC: "writes outside scope SHALL require approval").
	if p.VaultScope != "" {
		if !globMatch(p.VaultScope, targetPath) {
			return RequireApproval
		}
	}

	// 2. Rule table — first match wins.
	for _, r := range p.Rules {
		if !actorMatch(r.MatchActor, actor) {
			continue
		}
		if r.MatchPath != "" && !globMatch(r.MatchPath, targetPath) {
			continue
		}
		if r.MatchOp != "" && string(op) != r.MatchOp {
			continue
		}
		switch r.Decision {
		case "allow":
			return Allow
		case "require_approval":
			return RequireApproval
		case "deny":
			return Deny
		default:
			// Unknown decision string — fail-closed per ADR-050 §DR-2.
			return Deny
		}
	}

	// 3. No rule matched. Default: deny (fail-closed). Operators who
	// want a permissive default put a catch-all "*" rule first.
	return Deny
}

// actorMatch is a simple prefix-or-exact match. "agent:*" matches
// "agent:external" but not "user:owner"; "" or "*" matches any.
func actorMatch(pattern, actor string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(actor, prefix+":")
	}
	return pattern == actor
}

// globMatch is path.Match semantics for the vault scope and rule
// match paths. The project owns the path layout so we don't need
// anything more sophisticated than filepath.Match.
func globMatch(pattern, target string) bool {
	if pattern == "" {
		return true
	}
	// Normalize to forward slashes for cross-platform consistency.
	p := filepath.ToSlash(pattern)
	t := filepath.ToSlash(target)
	if matched, err := filepath.Match(p, t); err == nil && matched {
		return true
	}
	// Prefix match for "vault/*" style patterns where the path may
	// be deeper than the glob can express.
	if strings.HasSuffix(p, "/*") {
		prefix := strings.TrimSuffix(p, "/*")
		return strings.HasPrefix(t, prefix+"/") || t == strings.TrimSuffix(p, "/*")
	}
	return false
}

// ErrInvalidProfile is exported for callers that want to surface a
// distinct error type when YAML schema validation fails.
var ErrInvalidProfile = errors.New("policy: invalid profile")
