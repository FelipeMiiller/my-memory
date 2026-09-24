package asr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_AppliesDefaults_OnEmptyFile(t *testing.T) {
	// Empty YAML — Load applies all defaults.
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(yamlPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Provider != SupportedProvider {
		t.Errorf("default provider = %q, want %q", cfg.Provider, SupportedProvider)
	}
	if cfg.OnnxNemotron.ModelPath != DefaultModelPath {
		t.Errorf("default model_path = %q, want %q", cfg.OnnxNemotron.ModelPath, DefaultModelPath)
	}
	if cfg.OnnxNemotron.TargetLang != string(LocaleAuto) {
		t.Errorf("default target_lang = %q, want %q", cfg.OnnxNemotron.TargetLang, LocaleAuto)
	}
	if cfg.OnnxNemotron.ChunkMS != 160 {
		t.Errorf("default chunk_ms = %d, want 160", cfg.OnnxNemotron.ChunkMS)
	}
	if cfg.OnnxNemotron.NumThreads < 1 {
		t.Errorf("default num_threads = %d, want >= 1 (auto-detect)", cfg.OnnxNemotron.NumThreads)
	}
	if cfg.OnnxNemotron.Device != "cpu" {
		t.Errorf("default device = %q, want cpu", cfg.OnnxNemotron.Device)
	}
	if cfg.Vad.SilenceThresholdMs != 500 {
		t.Errorf("default VAD silence = %d, want 500", cfg.Vad.SilenceThresholdMs)
	}
}

func TestConfig_AppliesDefaults_OnMissingFile(t *testing.T) {
	// YAML file doesn't exist — Load still returns defaults without
	// erroring. Useful when `mem asr doctor` runs before first download.
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist.yaml")

	cfg, err := Load(missing)
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if cfg.Provider != SupportedProvider {
		t.Errorf("default provider not applied when file missing: %q", cfg.Provider)
	}
	if cfg.OnnxNemotron.ChunkMS != 160 {
		t.Errorf("default chunk_ms not applied when file missing: %d", cfg.OnnxNemotron.ChunkMS)
	}
}

func TestConfig_AppliesDefaults_OnEmptyPath(t *testing.T) {
	// Empty path means "use defaults only"; same semantics as missing file.
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load on empty path: %v", err)
	}
	if cfg.Provider != SupportedProvider {
		t.Errorf("default provider not applied: %q", cfg.Provider)
	}
}

func TestConfig_RespectsExplicitValues(t *testing.T) {
	// asr.Config is a self-contained sub-section. Top-level YAML keys
	// map directly to struct fields; the wrapping `asr:` key is added
	// by the parent GlobalConfig (internal/config/config.go).
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	yamlContent := `
provider: onnx-nemotron
onnx_nemotron:
  model_path: /opt/custom/nemotron.onnx
  target_lang: pt-BR
  chunk_ms: 320
  num_threads: 8
  device: cpu
vad:
  silence_threshold_ms: 750
  energy_threshold: 0.05
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OnnxNemotron.ModelPath != "/opt/custom/nemotron.onnx" {
		t.Errorf("explicit model_path lost: %q", cfg.OnnxNemotron.ModelPath)
	}
	if cfg.OnnxNemotron.TargetLang != "pt-BR" {
		t.Errorf("explicit target_lang lost: %q", cfg.OnnxNemotron.TargetLang)
	}
	if cfg.OnnxNemotron.ChunkMS != 320 {
		t.Errorf("explicit chunk_ms lost: %d", cfg.OnnxNemotron.ChunkMS)
	}
	if cfg.OnnxNemotron.NumThreads != 8 {
		t.Errorf("explicit num_threads lost: %d", cfg.OnnxNemotron.NumThreads)
	}
	if cfg.Vad.SilenceThresholdMs != 750 {
		t.Errorf("explicit VAD silence lost: %d", cfg.Vad.SilenceThresholdMs)
	}
}

func TestConfig_ValidatesChunkSize(t *testing.T) {
	cases := []struct {
		name     string
		chunkMS  int
		wantPass bool
	}{
		{"80", 80, true},
		{"160", 160, true},
		{"320", 320, true},
		{"560", 560, true},
		{"1120", 1120, true},
		{"300", 300, false},
		{"1000", 1000, false},
		{"0_default_will_become_160", 0, true}, // 0 → default 160
		{"negative", -160, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &Config{
				Provider: SupportedProvider,
				OnnxNemotron: OnnxNemotronConfig{
					ModelPath: DefaultModelPath,
					ChunkMS:   c.chunkMS,
				},
			}
			cfg.applyDefaults() // 0 → 160 before validation
			err := cfg.Validate()
			if (err == nil) != c.wantPass {
				t.Errorf("chunk_ms=%d Validate err=%v, want pass=%v", c.chunkMS, err, c.wantPass)
			}
		})
	}
}

func TestConfig_RejectsUnknownProvider(t *testing.T) {
	cfg := &Config{
		Provider: "python-sidecar", // DEFERRED per ADR-045 §Deferral
		OnnxNemotron: OnnxNemotronConfig{
			ModelPath:  DefaultModelPath,
			TargetLang: "auto",
			ChunkMS:    160,
			NumThreads: 4,
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected ErrInvalidConfig for python-sidecar provider")
	}
	if !contains(err.Error(), "python-sidecar") {
		t.Errorf("expected error to mention python-sidecar; got %q", err.Error())
	}
	if !contains(err.Error(), "deferred") {
		t.Errorf("expected error to mention 'deferred' rationale; got %q", err.Error())
	}
}

func TestConfig_RejectsEmptyTargetLang(t *testing.T) {
	cfg := &Config{
		Provider: SupportedProvider,
		OnnxNemotron: OnnxNemotronConfig{
			ModelPath:  DefaultModelPath,
			TargetLang: "", // forced empty (skipped default to test validation)
			ChunkMS:    160,
			NumThreads: 4,
		},
	}
	// applyDefaults would fix empty → "auto". Skip it; test direct
	// validation by clearing after defaults.
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected ErrInvalidConfig for empty target_lang after applyDefaults")
	}
}

func TestConfig_AggregatesMultipleViolations(t *testing.T) {
	cfg := &Config{
		Provider: "python-sidecar", // invalid provider
		OnnxNemotron: OnnxNemotronConfig{
			ModelPath:  "",  // invalid (empty after defaults would catch this)
			ChunkMS:    300, // invalid chunk size
			NumThreads: 0,   // invalid (zero)
			TargetLang: "auto",
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	// Should mention all 3 violations
	msg := err.Error()
	if !contains(msg, "provider") {
		t.Errorf("error should mention provider; got %q", msg)
	}
	if !contains(msg, "chunk_ms") {
		t.Errorf("error should mention chunk_ms; got %q", msg)
	}
	if !contains(msg, "num_threads") {
		t.Errorf("error should mention num_threads; got %q", msg)
	}
}

func TestConfig_NumThreadsAutoDetect(t *testing.T) {
	// YAML explicitly says num_threads: 0 → auto-detect → runtime.NumCPU().
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	yamlContent := `
asr:
  provider: onnx-nemotron
  onnx_nemotron:
    num_threads: 0
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OnnxNemotron.NumThreads < 1 {
		t.Errorf("num_threads auto-detect failed: got %d", cfg.OnnxNemotron.NumThreads)
	}
}

func TestConfig_RoundTrip_FromYAML(t *testing.T) {
	// A YAML that explicitly sets every field must round-trip through
	// Load without losing values.
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	yamlContent := `
provider: onnx-nemotron
onnx_nemotron:
  model_path: /opt/nemotron/0.6b.onnx
  vocab_path: /opt/nemotron/vocab.txt
  target_lang: pt-PT
  chunk_ms: 560
  num_threads: 16
  device: cpu
vad:
  silence_threshold_ms: 600
  energy_threshold: 0.02
`
	// The root struct has a single `provider` key; embed under `asr`:
	full := "asr:\n" + yamlContent
	if err := os.WriteFile(yamlPath, []byte(full), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OnnxNemotron.ModelPath != "/opt/nemotron/0.6b.onnx" {
		t.Errorf("model_path: %q", cfg.OnnxNemotron.ModelPath)
	}
	if cfg.OnnxNemotron.VocabPath != "/opt/nemotron/vocab.txt" {
		t.Errorf("vocab_path: %q", cfg.OnnxNemotron.VocabPath)
	}
	if cfg.OnnxNemotron.TargetLang != "pt-PT" {
		t.Errorf("target_lang: %q", cfg.OnnxNemotron.TargetLang)
	}
	if cfg.OnnxNemotron.ChunkMS != 560 {
		t.Errorf("chunk_ms: %d", cfg.OnnxNemotron.ChunkMS)
	}
	if cfg.OnnxNemotron.NumThreads != 16 {
		t.Errorf("num_threads: %d", cfg.OnnxNemotron.NumThreads)
	}
	if cfg.Vad.SilenceThresholdMs != 600 {
		t.Errorf("VAD silence: %d", cfg.Vad.SilenceThresholdMs)
	}
	if cfg.Vad.EnergyThreshold != 0.02 {
		t.Errorf("VAD energy: %f", cfg.Vad.EnergyThreshold)
	}
}
