package supervisor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// validProfileYAML is the canonical default profile fixture used
// across tests. Mirrors the planned .memory/profiles/default.yaml
// shape so round-trip tests exercise the same on-disk form
// operators will eventually author by hand.
const validProfileYAML = `schema_version: 1
workers:
  - name: embedder
    command: .memory/workers/embedder
    args:
      - --port
      - "49156"
    required: true
    egress: deny
    config:
      model: minilm
      batch_size: 32
  - name: events
    command: .memory/workers/events
`

// malformedYAML is intentionally broken YAML so the loader must
// fail with ErrInvalidProfile. Indentation is dropped on the
// second worker to trip the parser.
const malformedYAML = `schema_version: 1
workers:
  - name: embedder
    command: .memory/workers/embedder
 - name: events
    command: .memory/workers/events
`

// unknownFieldsYAML declares a top-level key the spec doesn't
// recognize. The strict decoder must reject this.
const unknownFieldsYAML = `schema_version: 1
bogus_field: nope
workers:
  - name: embedder
    command: .memory/workers/embedder
`

// schemaVersionTooHighYAML declares schema_version=2, which
// this build doesn't understand. Profile.Validate must refuse.
const schemaVersionTooHighYAML = `schema_version: 2
workers:
  - name: embedder
    command: .memory/workers/embedder
`

// duplicateNamesYAML declares two workers with the same name.
// Profile.Validate must refuse.
const duplicateNamesYAML = `schema_version: 1
workers:
  - name: embedder
    command: .memory/workers/embedder
  - name: embedder
    command: .memory/workers/embedder-2
`

// emptyCommandYAML declares a worker with empty command.
// Profile.Validate must refuse.
const emptyCommandYAML = `schema_version: 1
workers:
  - name: orphan
    command: ""
`

// invalidEgressYAML declares a worker with an egress value that
// isn't deny/allow. Profile.Validate must refuse.
const invalidEgressYAML = `schema_version: 1
workers:
  - name: embedder
    command: .memory/workers/embedder
    egress: maybe
`

// emptyWorkersYAML is structurally valid — the spec explicitly
// allows a profile with no workers (mymemoryd starts and exits).
const emptyWorkersYAML = `schema_version: 1
workers: []
`

func writeProfileFixture(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "default.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestProfile_LoadValid(t *testing.T) {
	path := writeProfileFixture(t, validProfileYAML)

	p, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if p.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", p.SchemaVersion)
	}
	if got := len(p.Workers); got != 2 {
		t.Fatalf("Workers count = %d, want 2", got)
	}

	embedder := p.Workers[0]
	if embedder.Name != "embedder" {
		t.Fatalf("Workers[0].Name = %q, want embedder", embedder.Name)
	}
	if !embedder.Required {
		t.Fatal("Workers[0].Required should be true")
	}
	if embedder.Egress != EgressDeny {
		t.Fatalf("Workers[0].Egress = %q, want %q", embedder.Egress, EgressDeny)
	}
	if len(embedder.Args) != 2 || embedder.Args[0] != "--port" || embedder.Args[1] != "49156" {
		t.Fatalf("Workers[0].Args = %v", embedder.Args)
	}
	if model, _ := embedder.Config["model"].(string); model != "minilm" {
		t.Fatalf("Workers[0].Config[model] = %v, want minilm", embedder.Config["model"])
	}

	events := p.Workers[1]
	if events.Required {
		t.Fatal("Workers[1].Required should be false (default)")
	}
	if events.Egress != "" {
		t.Fatalf("Workers[1].Egress = %q, want empty (default deny)", events.Egress)
	}
	if events.Name != "events" {
		t.Fatalf("Workers[1].Name = %q, want events", events.Name)
	}
}

func TestProfile_LoadMalformed_ReturnsErrInvalidProfile(t *testing.T) {
	path := writeProfileFixture(t, malformedYAML)
	_, err := LoadProfile(path)
	if err == nil {
		t.Fatal("expected error on malformed YAML, got nil")
	}
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
}

func TestProfile_LoadUnknownFieldsRejected(t *testing.T) {
	path := writeProfileFixture(t, unknownFieldsYAML)
	_, err := LoadProfile(path)
	if err == nil {
		t.Fatal("expected error on unknown top-level field, got nil")
	}
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
	if !strings.Contains(err.Error(), "bogus_field") &&
		!strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected error to mention bogus_field or unknown field, got %v", err)
	}
}

func TestProfile_SchemaVersionTooHighRejected(t *testing.T) {
	path := writeProfileFixture(t, schemaVersionTooHighYAML)
	_, err := LoadProfile(path)
	if err == nil {
		t.Fatal("expected error on schema_version=2, got nil")
	}
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
	if !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("expected error to mention schema_version, got %v", err)
	}
}

func TestProfile_DuplicateNamesRejected(t *testing.T) {
	path := writeProfileFixture(t, duplicateNamesYAML)
	_, err := LoadProfile(path)
	if err == nil {
		t.Fatal("expected error on duplicate worker names, got nil")
	}
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected error to mention duplicate, got %v", err)
	}
}

func TestProfile_EmptyCommandRejected(t *testing.T) {
	path := writeProfileFixture(t, emptyCommandYAML)
	_, err := LoadProfile(path)
	if err == nil {
		t.Fatal("expected error on empty command, got nil")
	}
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
}

func TestProfile_InvalidEgressRejected(t *testing.T) {
	path := writeProfileFixture(t, invalidEgressYAML)
	_, err := LoadProfile(path)
	if err == nil {
		t.Fatal("expected error on invalid egress, got nil")
	}
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
	if !strings.Contains(err.Error(), "egress") {
		t.Fatalf("expected error to mention egress, got %v", err)
	}
}

func TestProfile_EmptyWorkersIsValid(t *testing.T) {
	path := writeProfileFixture(t, emptyWorkersYAML)
	p, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile on empty workers: %v", err)
	}
	if len(p.Workers) != 0 {
		t.Fatalf("expected 0 workers, got %d", len(p.Workers))
	}
}

func TestProfile_MissingFileReturnsErrInvalidProfile(t *testing.T) {
	_, err := LoadProfile(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil {
		t.Fatal("expected error on missing file")
	}
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
}

func TestProfile_RoundTrip(t *testing.T) {
	// Round-trip 1: load fixture, validate, marshal back, reload,
	// confirm structural equality.
	path := writeProfileFixture(t, validProfileYAML)
	p1, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}

	roundTripPath := filepath.Join(t.TempDir(), "roundtrip.yaml")
	if err := writeProfileAtomic(roundTripPath, p1); err != nil {
		t.Fatalf("writeProfileAtomic: %v", err)
	}
	p2, err := LoadProfile(roundTripPath)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	if p1.SchemaVersion != p2.SchemaVersion {
		t.Fatalf("SchemaVersion drift: %d != %d", p1.SchemaVersion, p2.SchemaVersion)
	}
	if len(p1.Workers) != len(p2.Workers) {
		t.Fatalf("Workers count drift: %d != %d", len(p1.Workers), len(p2.Workers))
	}
	for i := range p1.Workers {
		w1, w2 := p1.Workers[i], p2.Workers[i]
		if w1.Name != w2.Name {
			t.Errorf("Workers[%d].Name: %q != %q", i, w1.Name, w2.Name)
		}
		if w1.Command != w2.Command {
			t.Errorf("Workers[%d].Command: %q != %q", i, w1.Command, w2.Command)
		}
		if w1.Required != w2.Required {
			t.Errorf("Workers[%d].Required: %v != %v", i, w1.Required, w2.Required)
		}
		if w1.Egress != w2.Egress {
			t.Errorf("Workers[%d].Egress: %q != %q", i, w1.Egress, w2.Egress)
		}
		// Args comparison is order-sensitive (the YAML contract is
		// positional within a worker's args list).
		if len(w1.Args) != len(w2.Args) {
			t.Errorf("Workers[%d].Args length: %d != %d", i, len(w1.Args), len(w2.Args))
		} else {
			for j := range w1.Args {
				if w1.Args[j] != w2.Args[j] {
					t.Errorf("Workers[%d].Args[%d]: %q != %q",
						i, j, w1.Args[j], w2.Args[j])
				}
			}
		}
	}
}

func TestProfile_ExtendsParsedAndMerged(t *testing.T) {
	// T5 ships the merge. ResolveProfile must load the parent
	// from <baseDir>/<extends>.yaml and combine workers per the
	// spec rules. Extends must be cleared on the resolved profile
	// so downstream code never sees the inheritance pointer.
	dir := t.TempDir()
	writeProfileFile(t, dir, "default.yaml", `schema_version: 1
workers:
  - name: embedder
    command: .memory/workers/embedder
`)
	childPath := filepath.Join(dir, "voice.yaml")
	childYAML := `schema_version: 1
extends: default
workers:
  - name: tts
    command: .memory/workers/tts
`
	if err := os.WriteFile(childPath, []byte(childYAML), 0o644); err != nil {
		t.Fatalf("write child: %v", err)
	}

	p, err := LoadProfile(childPath)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if p.Extends != "default" {
		t.Fatalf("Extends = %q, want default", p.Extends)
	}

	resolved, err := ResolveProfile(p, dir)
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}
	if resolved.Extends != "" {
		t.Fatalf("ResolveProfile must clear Extends after merge; got %q", resolved.Extends)
	}
	if len(resolved.Workers) != 2 {
		t.Fatalf("workers count = %d, want 2 (parent + child)", len(resolved.Workers))
	}
	if resolved.Workers[0].Name != "embedder" {
		t.Errorf("workers[0].Name = %q, want embedder (parent first)", resolved.Workers[0].Name)
	}
	if resolved.Workers[1].Name != "tts" {
		t.Errorf("workers[1].Name = %q, want tts (child-only appended)", resolved.Workers[1].Name)
	}
}

func TestProfile_ValidateOnNil(t *testing.T) {
	var p *Profile
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error on nil profile")
	}
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
}

func TestProfile_DefaultSchemaVersionBackCompat(t *testing.T) {
	// Profiles without a schema_version default to 1 so early
	// hand-written manifests keep loading through the T4 strict
	// decoder.
	yaml := `workers:
  - name: embedder
    command: .memory/workers/embedder
`
	path := writeProfileFixture(t, yaml)
	p, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if p.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1 (default)", p.SchemaVersion)
	}
}

func TestProfile_RestartPolicySurvivesRoundTrip(t *testing.T) {
	yaml := `schema_version: 1
workers:
  - name: embedder
    command: .memory/workers/embedder
    restart_policy:
      max_retries: 3
      base_delay: 200ms
      max_delay: 10s
      jitter: 0.1
`
	path := writeProfileFixture(t, yaml)
	p, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	rp := p.Workers[0].RestartPolicy
	if rp.MaxRetries != 3 {
		t.Fatalf("MaxRetries = %d, want 3", rp.MaxRetries)
	}
	if rp.BaseDelay != 200*time.Millisecond {
		t.Fatalf("BaseDelay = %v, want 200ms", rp.BaseDelay)
	}
	if rp.MaxDelay != 10*time.Second {
		t.Fatalf("MaxDelay = %v, want 10s", rp.MaxDelay)
	}
	if rp.Jitter != 0.1 {
		t.Fatalf("Jitter = %v, want 0.1", rp.Jitter)
	}
}
