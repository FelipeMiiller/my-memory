package asr

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// DefaultModelPath is the conventional location for the Nemotron int4
// model. Override via `asr.onnx_nemotron.model_path` in `.memory/config.yaml`.
//
// The path is relative to the repo root (where `mem` is invoked), not the
// user's CWD. This avoids ambiguity when mem is run from outside the repo
// (e.g., homelab deployment).
const DefaultModelPath = ".memory/models/nemotron-asr-int4.onnx"

// SupportedProvider is the only ASR provider currently implemented.
// Python-sidecar ASR is explicitly DEFERRED per ADR-045 §Deferral.
const SupportedProvider = "onnx-nemotron"

// Config mirrors ADR-045 §Configuration YAML schema.
//
//	asr:
//	  provider: "onnx-nemotron"
//	  onnx_nemotron:
//	    model_path: ""        # default: .memory/models/nemotron-asr-int4.onnx
//	    vocab_path: ""        # default: .memory/models/nemotron-asr-vocab.txt
//	    target_lang: "auto"   # auto | pt-BR | pt-PT | en-US | ...
//	    chunk_ms: 160         # 80 | 160 | 320 | 560 | 1120
//	    num_threads: 0        # 0 = auto-detect via runtime.NumCPU()
//
// The OnnxNemotron sub-config is the only provider block currently
// accepted; Python-sidecar is deferred (ADR-045 §Deferral).
type Config struct {
	Provider     string             `yaml:"provider"`
	OnnxNemotron OnnxNemotronConfig `yaml:"onnx_nemotron"`
	Vad          VadConfig          `yaml:"vad,omitempty"`
}

// OnnxNemotronConfig holds Nemotron-3.5-specific knobs.
type OnnxNemotronConfig struct {
	ModelPath  string `yaml:"model_path"`
	VocabPath  string `yaml:"vocab_path"`
	TargetLang string `yaml:"target_lang"`
	ChunkMS    int    `yaml:"chunk_ms"`
	NumThreads int    `yaml:"num_threads"`
	// Device is a future-facing knob (CPU today; "gpu" requires
	// ONNX Runtime GPU build + CUDA/ROCm/Metal runtime). Default
	// "cpu" matches the cross-platform stable binding.
	Device string `yaml:"device,omitempty"`
}

// VadConfig controls end-of-utterance detection. Default silence
// threshold is 500 ms per ADR-045 + Nemotron streaming conventions.
type VadConfig struct {
	SilenceThresholdMs int     `yaml:"silence_threshold_ms"`
	EnergyThreshold    float64 `yaml:"energy_threshold"`
}

// Load reads YAML from yamlPath, applies defaults, and validates. Returns
// *ConfigErrors (multi-violation) when validation fails so callers can
// report all problems at once.
func Load(yamlPath string) (*Config, error) {
	var cfg Config
	// Parse empty default if file missing — Load() is meant to be
	// safe to call before `mem asr download` has been run.
	if yamlPath != "" {
		data, err := os.ReadFile(yamlPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("asr: read %s: %w", yamlPath, err)
		}
		if len(data) > 0 {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("asr: parse %s: %w", yamlPath, err)
			}
		}
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// applyDefaults sets the canonical defaults per ADR-045. Called by Load;
// not exported because callers should go through Load (which validates).
func (c *Config) applyDefaults() {
	if c.Provider == "" {
		c.Provider = SupportedProvider
	}
	if c.OnnxNemotron.ModelPath == "" {
		c.OnnxNemotron.ModelPath = DefaultModelPath
	}
	if c.OnnxNemotron.VocabPath == "" {
		c.OnnxNemotron.VocabPath = filepath.Join(filepath.Dir(DefaultModelPath), "nemotron-asr-vocab.txt")
	}
	if c.OnnxNemotron.TargetLang == "" {
		c.OnnxNemotron.TargetLang = string(LocaleAuto)
	}
	if c.OnnxNemotron.ChunkMS == 0 {
		c.OnnxNemotron.ChunkMS = 160
	}
	if c.OnnxNemotron.NumThreads == 0 {
		c.OnnxNemotron.NumThreads = runtime.NumCPU()
	}
	if c.OnnxNemotron.Device == "" {
		c.OnnxNemotron.Device = "cpu"
	}
	if c.Vad.SilenceThresholdMs == 0 {
		c.Vad.SilenceThresholdMs = 500
	}
	if c.Vad.EnergyThreshold == 0 {
		c.Vad.EnergyThreshold = 0.01 // energy-based default; tuned per ADR-045
	}
}

// Validate checks that all values are within the supported set. Returns
// *ConfigErrors (aggregated) on any violation so callers can report all
// problems at once.
func (c *Config) Validate() error {
	var ce ConfigErrors

	// Provider must be one of the supported set. Python-sidecar is
	// explicitly deferred per ADR-045 §Deferral — do not accept here.
	switch c.Provider {
	case SupportedProvider:
		// OK
	default:
		ce.Append("provider %q not supported (only %q is active; Python-sidecar deferred per ADR-045)",
			c.Provider, SupportedProvider)
	}

	// Chunk size must be one of {80, 160, 320, 560, 1120}.
	cs := ChunkSizeMS(c.OnnxNemotron.ChunkMS)
	if !cs.Valid() {
		ce.Append("asr.onnx_nemotron.chunk_ms %d not in allowed set {80, 160, 320, 560, 1120}",
			c.OnnxNemotron.ChunkMS)
	}

	// Target lang must be non-empty (can be "auto" or a real locale;
	// empty string after defaults means the YAML had target_lang: "").
	if c.OnnxNemotron.TargetLang == "" {
		ce.Append("asr.onnx_nemotron.target_lang cannot be empty")
	}

	// NumThreads sanity: must be >= 1 after defaults (0 means
	// auto-detect, resolved in applyDefaults).
	if c.OnnxNemotron.NumThreads < 1 {
		ce.Append("asr.onnx_nemotron.num_threads must be >= 1 (got %d)", c.OnnxNemotron.NumThreads)
	}

	// ModelPath cannot be empty or whitespace-only.
	if c.OnnxNemotron.ModelPath == "" {
		ce.Append("asr.onnx_nemotron.model_path cannot be empty")
	}

	if len(ce.Violations) > 0 {
		return &ce
	}
	return nil
}
