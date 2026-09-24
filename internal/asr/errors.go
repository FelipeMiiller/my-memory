// Package asr provides streaming ASR (Automatic Speech Recognition) for the
// my-memory CLI. Implements ADR-045: Nemotron 3.5 ASR via ONNX Runtime Go
// in-process. See docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md
// for the architectural decision.
package asr

import (
	"errors"
	"fmt"
)

// Sentinel errors used across internal/asr and surfaced to callers via
// %w wrapping. Mirrors the pattern in internal/event_runtime/errors.go.
var (
	// ErrModelNotFound is returned when the configured model file is
	// missing from disk. Hint: run `mem asr download --provider onnx-nemotron`.
	ErrModelNotFound = errors.New("asr: model file not found on disk")

	// ErrInvalidChunkSize is returned when chunk_ms is not in the allowed
	// set {80, 160, 320, 560, 1120}. Mirrors Nemotron's att_context_size
	// supported values.
	ErrInvalidChunkSize = errors.New("asr: chunk_ms must be one of 80, 160, 320, 560, 1120")

	// ErrInvalidLocale is returned when target_lang is set to a locale not
	// supported by Nemotron 3.5 (40 locales + "auto"). See Locale.Valid.
	ErrInvalidLocale = errors.New("asr: target_lang not in supported set")

	// ErrUnsupportedSampleRate is returned when input audio sample rate is
	// not 16 kHz mono PCM (the Wyoming-spec convention that Nemotron 3.5
	// expects; we intentionally do NOT resample).
	ErrUnsupportedSampleRate = errors.New("asr: only 16 kHz mono PCM supported")

	// ErrModelIntegrity is returned when the SHA-256 of a downloaded model
	// does not match the expected hash in `.sha256` sidecar.
	ErrModelIntegrity = errors.New("asr: model SHA-256 mismatch")

	// ErrInvalidConfig is returned at startup when `.memory/config.yaml`
	// has invalid ASR settings. Wraps a list-like detail via Errors() below.
	ErrInvalidConfig = errors.New("asr: invalid configuration")

	// ErrProviderFailed is returned when the active provider fails to
	// initialize (e.g., onnxruntime_go binding link failure on macOS arm64).
	// Wraps the underlying ONNX Runtime error.
	ErrProviderFailed = errors.New("asr: provider initialization failed")
)

// HintFor returns the user-facing remediation hint for an ASR error, when
// one is defined. Returns "" if the error has no specific hint.
func HintFor(err error) string {
	switch {
	case errors.Is(err, ErrModelNotFound):
		return "Run 'mem asr download --provider onnx-nemotron' or place model manually at .memory/models/."
	case errors.Is(err, ErrInvalidChunkSize):
		return "Set asr.onnx_nemotron.chunk_ms to one of: 80, 160, 320, 560, 1120."
	case errors.Is(err, ErrInvalidLocale):
		return "Set asr.onnx_nemotron.target_lang to 'auto' or one of the supported locales."
	case errors.Is(err, ErrUnsupportedSampleRate):
		return "Provide 16 kHz mono PCM (s16le); resampling is not done in-process."
	case errors.Is(err, ErrModelIntegrity):
		return "Delete the model file at .memory/models/ and re-run 'mem asr download'."
	case errors.Is(err, ErrProviderFailed):
		return "Verify the onnxruntime native library is installed for your platform."
	default:
		return ""
	}
}

// ConfigErrors aggregates multiple config validation violations so callers
// can report all problems at once instead of failing on the first. Use
// `var ce ConfigErrors; ce.Append(...); return &ce` from validators.
type ConfigErrors struct {
	Violations []string
}

// Error implements the error interface.
func (c *ConfigErrors) Error() string {
	if c == nil || len(c.Violations) == 0 {
		return ErrInvalidConfig.Error()
	}
	msg := ErrInvalidConfig.Error() + ":"
	for _, v := range c.Violations {
		msg += "\n  - " + v
	}
	return msg
}

// Append adds a violation to the list.
func (c *ConfigErrors) Append(format string, args ...any) {
	c.Violations = append(c.Violations, fmt.Sprintf(format, args...))
}

// Unwrap exposes the sentinel so errors.Is(err, ErrInvalidConfig) works.
func (c *ConfigErrors) Unwrap() error {
	return ErrInvalidConfig
}

// Is reports the error as ErrInvalidConfig when the slice is non-nil,
// satisfying errors.As expectations.
func (c *ConfigErrors) Is(target error) bool {
	return errors.Is(ErrInvalidConfig, target)
}
