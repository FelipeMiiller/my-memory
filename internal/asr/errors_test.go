package asr

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinels_AreDistinct(t *testing.T) {
	sentinels := []error{
		ErrModelNotFound,
		ErrInvalidChunkSize,
		ErrInvalidLocale,
		ErrUnsupportedSampleRate,
		ErrModelIntegrity,
		ErrInvalidConfig,
		ErrProviderFailed,
	}
	seen := map[string]bool{}
	for _, e := range sentinels {
		if seen[e.Error()] {
			t.Errorf("duplicate sentinel message: %q", e.Error())
		}
		seen[e.Error()] = true
	}
}

func TestHintFor(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantHint bool
	}{
		{"ModelNotFound", ErrModelNotFound, true},
		{"InvalidChunkSize", ErrInvalidChunkSize, true},
		{"InvalidLocale", ErrInvalidLocale, true},
		{"UnsupportedSampleRate", ErrUnsupportedSampleRate, true},
		{"ModelIntegrity", ErrModelIntegrity, true},
		{"ProviderFailed", ErrProviderFailed, true},
		{"Generic", errors.New("some other error"), false},
	}
	for _, c := range cases {
		hint := HintFor(c.err)
		if (hint != "") != c.wantHint {
			t.Errorf("%s: HintFor non-empty = %v, want %v (hint=%q)", c.name, hint != "", c.wantHint, hint)
		}
	}
}

func TestHintFor_WrappedError(t *testing.T) {
	wrapped := fmt.Errorf("loading provider: %w", ErrModelNotFound)
	hint := HintFor(wrapped)
	if hint == "" {
		t.Error("HintFor must traverse wrapping via errors.Is")
	}
}

func TestConfigErrors_AggregatesAndFormats(t *testing.T) {
	var ce ConfigErrors
	ce.Append("chunk_ms %d not in allowed set", 300)
	ce.Append("target_lang %q not known", "klingon")
	ce.Append("provider %q not supported", "python-sidecar")

	msg := ce.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
	for _, v := range ce.Violations {
		if !contains(msg, v) {
			t.Errorf("expected message to contain %q; got %q", v, msg)
		}
	}

	if !errors.Is(&ce, ErrInvalidConfig) {
		t.Error("errors.Is should return true for ErrInvalidConfig")
	}
}

func TestConfigErrors_Empty(t *testing.T) {
	var ce ConfigErrors
	if ce.Error() != ErrInvalidConfig.Error() {
		t.Errorf("empty ConfigErrors should format as ErrInvalidConfig only; got %q", ce.Error())
	}
}

func TestConfigErrors_NilSafe(t *testing.T) {
	var ce *ConfigErrors
	// Calling Error() on nil pointer must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Error() panicked on nil receiver: %v", r)
		}
	}()
	_ = ce.Error()
}
