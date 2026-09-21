package supervisor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SupportedProfileSchemaVersion is the schema_version this build
// understands. Profiles declaring a higher value MUST be rejected
// with ErrInvalidProfile so operators see a clear failure rather
// than silent data loss on load. Profiles without a
// schema_version field default to 1 (back-compat for early T1
// hand-written manifests).
const SupportedProfileSchemaVersion = 1

// Egress policy constants. The default is deny-by-default
// (ADR-042 §DR-5); allow is opt-in and audit-flagged (ADR-050
// LLM05).
const (
	EgressDeny  = "deny"
	EgressAllow = "allow"
)

// Profile is the YAML manifest declaring the worker pool the
// supervisor should bring up for a given deployment scenario.
// See ADR-042 §profile-yaml and the mymemoryd-supervisor feature
// spec (P3) for the user stories this struct satisfies.
//
// Extends is parsed by LoadProfile but only merged by ResolveProfile
// (T5). T4 captures the raw field so the YAML contract is stable
// ahead of inheritance logic.
type Profile struct {
	SchemaVersion int          `yaml:"schema_version"`
	Extends       string       `yaml:"extends,omitempty"`
	Workers       []WorkerSpec `yaml:"workers"`
}

// Validate enforces structural rules after YAML decode. It does
// NOT resolve extends (that's T5) or start any worker; the goal is
// to surface the most common authoring mistakes before the
// supervisor tries to launch a worker with the bad config.
//
// Returns nil when the profile is structurally sound; otherwise
// wraps ErrInvalidProfile so callers can errors.Is the failure.
func (p *Profile) Validate() error {
	if p == nil {
		return fmt.Errorf("%w: profile is nil", ErrInvalidProfile)
	}
	if p.SchemaVersion == 0 {
		// Implicit default to 1 (back-compat with early hand-written
		// manifests). ResolveProfile may upgrade later.
		p.SchemaVersion = 1
	}
	if p.SchemaVersion > SupportedProfileSchemaVersion {
		return fmt.Errorf("%w: schema_version=%d > supported=%d",
			ErrInvalidProfile, p.SchemaVersion, SupportedProfileSchemaVersion)
	}
	if len(p.Workers) == 0 {
		// Empty profile is structurally valid (the spec allows it
		// — mymemoryd starts with an empty worker set and exits).
		return nil
	}
	seen := make(map[string]struct{}, len(p.Workers))
	for i := range p.Workers {
		w := &p.Workers[i]
		if w.Name == "" {
			return fmt.Errorf("%w: workers[%d].name is empty", ErrInvalidProfile, i)
		}
		if _, dup := seen[w.Name]; dup {
			return fmt.Errorf("%w: duplicate worker name %q", ErrInvalidProfile, w.Name)
		}
		seen[w.Name] = struct{}{}

		if w.Command == "" {
			return fmt.Errorf("%w: workers[%q].command is empty", ErrInvalidProfile, w.Name)
		}
		if w.Egress != "" && w.Egress != EgressDeny && w.Egress != EgressAllow {
			return fmt.Errorf("%w: workers[%q].egress=%q (must be %q or %q)",
				ErrInvalidProfile, w.Name, w.Egress, EgressDeny, EgressAllow)
		}
	}
	return nil
}

// EffectiveWorkers returns the workers slice after applying the
// T4 baseline (no inheritance yet — T5 owns the merge). Exposed
// as a method so call sites stay stable when ResolveProfile
// lands.
func (p *Profile) EffectiveWorkers() []WorkerSpec {
	if p == nil {
		return nil
	}
	return p.Workers
}

// LoadProfile reads and validates the YAML profile at path. The
// decoder runs with KnownFields(true) so unknown top-level keys
// surface as a load error rather than being silently ignored —
// authoring mistakes like a typo'd `workes:` instead of `workers:`
// must fail loud.
//
// Returns:
//   - (*Profile, nil) on success.
//   - (nil, *fs.PathError wrapping ErrInvalidProfile) when path
//     does not exist.
//   - (nil, fmt.Errorf wrapping ErrInvalidProfile) when YAML is
//     malformed, contains unknown fields, or fails Validate.
func LoadProfile(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s: %v", ErrInvalidProfile, path, err)
		}
		return nil, fmt.Errorf("supervisor: read profile %s: %w", path, err)
	}

	var p Profile
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrInvalidProfile, path, err)
	}

	// yaml.v3 leaves the cursor past EOF after a single Decode; a
	// second call would return io.EOF which is the canonical
	// "single-document stream" sentinel — we don't need to check it
	// for a profile but we keep the decoder for forward compat
	// (multi-doc profiles in the future).

	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// ResolveProfile applies T5 inheritance semantics: when p.Extends
// is non-empty, the parent's profile is loaded from
// baseDir/<extends>.yaml and merged with p.Workers.
//
// Merge rules (spec P3 AC 2):
//  1. Parent workers appear first, in declared order.
//  2. Child workers with a Name matching a parent entry OVERRIDE
//     that parent entry (Command, Args, Env, Required, Egress,
//     Config, RestartPolicy all replaced). The override does NOT
//     preserve parent fields the child omits — the child must
//     re-declare everything it wants to keep.
//  3. Child-only workers (Name not in parent) are appended in
//     declared order after the parent list.
//  4. Worker configs are merged at the Config map level: child
//     keys override parent keys for the same worker.
//
// A non-empty extends field that points at a missing or malformed
// file is an error — inheritance failures must surface loudly so
// operators don't ship a profile that silently drops half the
// worker pool.
func ResolveProfile(p *Profile, baseDir string) (*Profile, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: profile is nil", ErrInvalidProfile)
	}
	if p.Extends == "" {
		return p, nil
	}

	parentPath := filepath.Join(baseDir, p.Extends+".yaml")
	parent, err := LoadProfile(parentPath)
	if err != nil {
		return nil, fmt.Errorf("%w: extends %q: %v", ErrInvalidProfile, p.Extends, err)
	}

	// Recursive resolve in case parent also extends something.
	resolvedParent, err := ResolveProfile(parent, baseDir)
	if err != nil {
		return nil, err
	}

	merged := mergeWorkers(resolvedParent.Workers, p.Workers)

	// Build a fresh Profile so callers can't mutate the parent
	// via the returned pointer.
	out := *p
	out.Workers = merged
	out.Extends = "" // fully resolved; downstream code shouldn't see extends
	return &out, nil
}

// mergeWorkers combines parent and child worker lists per the rules
// in ResolveProfile. Pure function — exposed at package level so
// future tests can exercise the merge independently of file IO.
func mergeWorkers(parent, child []WorkerSpec) []WorkerSpec {
	if len(parent) == 0 {
		return append([]WorkerSpec(nil), child...)
	}
	if len(child) == 0 {
		return append([]WorkerSpec(nil), parent...)
	}

	// Index parent by name so we can detect overrides in O(1) and
	// preserve the parent's declared order. Override entries
	// replace the parent entry at the same position so the merged
	// list keeps the parent's order for overridden workers; new
	// child entries go to the end.
	merged := make([]WorkerSpec, 0, len(parent)+len(child))
	overridden := make(map[string]bool, len(child))
	for _, pw := range parent {
		replaced := false
		for _, cw := range child {
			if cw.Name == pw.Name {
				merged = append(merged, mergeWorker(pw, cw))
				overridden[cw.Name] = true
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, pw)
		}
	}
	// Append child-only workers in their declared order.
	for _, cw := range child {
		if overridden[cw.Name] {
			continue
		}
		merged = append(merged, cw)
	}
	return merged
}

// mergeWorker combines a parent and child worker entry where the
// Name matches. The child's scalar fields (Command, Args, Env,
// Required, Egress) win outright; the child's Config map is
// merged into the parent's (child keys override parent keys).
func mergeWorker(parent, child WorkerSpec) WorkerSpec {
	out := child
	if len(parent.Config) > 0 {
		out.Config = make(map[string]any, len(parent.Config)+len(child.Config))
		for k, v := range parent.Config {
			out.Config[k] = v
		}
		for k, v := range child.Config {
			out.Config[k] = v
		}
	}
	return out
}

// writeProfileAtomic serializes p to YAML atomically (tmp + rename).
// Used by tests to set up round-trip fixtures without leaking
// partial files when the marshal fails partway.
func writeProfileAtomic(path string, p *Profile) error {
	tmp := path + ".tmp"
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("supervisor: marshal profile: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("supervisor: mkdir profile dir: %w", err)
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("supervisor: write tmp profile: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("supervisor: rename profile: %w", err)
	}
	return nil
}
