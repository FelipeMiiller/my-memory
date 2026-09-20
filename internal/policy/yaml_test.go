package policy

import (
	"path/filepath"
	"testing"
)

// TestLoadBundledProfiles validates the three profile YAMLs that ship
// in `.memory/policy/`. Catches typos / schema drift at quality-gate
// time without requiring a running vault.
func TestLoadBundledProfiles(t *testing.T) {
	for _, name := range []string{"strict", "balanced", "permissive-dev"} {
		t.Run(name, func(t *testing.T) {
			eng, err := Load(filepath.Join("..", "..", ".memory", "policy", name+".yaml"))
			if err != nil {
				t.Fatalf("Load %s: %v", name, err)
			}
			if eng == nil {
				t.Fatalf("nil engine for %s", name)
			}
			// Every profile must define at least one rule.
			if len(eng.profile.Rules) == 0 {
				t.Errorf("profile %s has no rules", name)
			}
		})
	}
}
