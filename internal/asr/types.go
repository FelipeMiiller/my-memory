// Package asr provides streaming ASR (Automatic Speech Recognition) for the
// my-memory CLI. Implements ADR-045: Nemotron 3.5 ASR via ONNX Runtime Go
// in-process.
package asr

import (
	"encoding/json"
	"fmt"
)

// ChunkSizeMS is the duration of a single audio chunk fed to the ASR
// streaming engine. Nemotron 3.5 supports att_context_size values matching
// these durations (80, 160, 320, 560, 1120 ms).
type ChunkSizeMS int

const (
	// Chunk80MS is the lowest-latency configuration: 80 ms of audio per
	// chunk. Recommended for fast back-and-forth dialog.
	Chunk80MS ChunkSizeMS = 80
	// Chunk160MS is the default per ADR-045 — balance of latency and
	// throughput.
	Chunk160MS ChunkSizeMS = 160
	// Chunk320MS is a mid-tier configuration.
	Chunk320MS ChunkSizeMS = 320
	// Chunk560MS is a longer-context configuration for noisy audio.
	Chunk560MS ChunkSizeMS = 560
	// Chunk1120MS is the maximum supported context size.
	Chunk1120MS ChunkSizeMS = 1120
)

// Valid reports whether c is one of the chunk sizes Nemotron 3.5 supports.
func (c ChunkSizeMS) Valid() bool {
	switch c {
	case Chunk80MS, Chunk160MS, Chunk320MS, Chunk560MS, Chunk1120MS:
		return true
	}
	return false
}

// SamplesPerChunk returns the number of 16 kHz mono PCM samples in a chunk
// of this size. Use to validate input buffer length.
func (c ChunkSizeMS) SamplesPerChunk() int {
	// 16 kHz × seconds.
	return int(c) * 16
}

// Locale identifies the target language for transcription. "auto" lets
// Nemotron 3.5 detect the language from the audio (40 locales including
// pt-BR, pt-PT, en-US).
type Locale string

const (
	LocaleAuto Locale = "auto"
	LocalePtBR Locale = "pt-BR"
	LocalePtPT Locale = "pt-PT"
	LocaleEnUS Locale = "en-US"
)

// supportedLocales is the subset we list explicitly. The full set is 40,
// see ADR-045 §"Decision Outcome". We allow any string value through
// Valid() — runtime validation happens against the model at inference
// time (Nemotron 3.5 supports a wider set; we just don't enumerate them
// to keep this file bounded).
var supportedLocales = map[Locale]struct{}{
	LocaleAuto: {},
	LocalePtBR: {},
	LocalePtPT: {},
	LocaleEnUS: {},
}

// Valid reports whether l is one of the explicitly-supported locales
// (auto + a small starter set) OR an arbitrary locale string that the
// model may accept at inference time. To enforce strict enumeration,
// use Known().
func (l Locale) Valid() bool {
	return l != ""
}

// Known reports whether l is in the explicitly-enumerated set. Use this
// when validating `.memory/config.yaml` to surface typos early.
func (l Locale) Known() bool {
	_, ok := supportedLocales[l]
	return ok
}

// Token is a single decoded token from the ASR output. Confidence is the
// posterior probability assigned by the decoder joint at that step.
type Token struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	StartMS    int64   `json:"start_ms,omitempty"`
	EndMS      int64   `json:"end_ms,omitempty"`
}

// Transcript is the streaming output of one ASR inference. Returned by
// StreamChunk for partial transcripts and by Finalize for the terminal
// utterance result.
type Transcript struct {
	Text       string  `json:"text"`
	Language   Locale  `json:"language"`
	Confidence float64 `json:"confidence"`
	Tokens     []Token `json:"tokens,omitempty"`
	LatencyMS  int64   `json:"latency_ms"`
	IsFinal    bool    `json:"is_final"`
}

// MarshalJSON customizes JSON encoding to ensure Tokens is never null
// (omit-only slice to keep consumers from checking for nil).
func (t Transcript) MarshalJSON() ([]byte, error) {
	type alias Transcript
	// If Tokens is nil, replace with empty slice for clean JSON round-trip.
	if t.Tokens == nil {
		t.Tokens = []Token{}
	}
	return json.Marshal(alias(t))
}

// String returns a redacted-safe rendering of the transcript. Use this in
// logs and CLI output per ADR-050 LLM02 (PII redaction before persistence).
func (t Transcript) String() string {
	return fmt.Sprintf("Transcript{lang=%s confidence=%.2f latency=%dms tokens=%d text-len=%d}",
		t.Language, t.Confidence, t.LatencyMS, len(t.Tokens), len(t.Text))
}

// Validate enforces minimum sanity: LatencyMS non-negative, Confidence in
// [0,1]. Called by StreamChunk implementations before returning.
func (t *Transcript) Validate() error {
	if t.LatencyMS < 0 {
		return fmt.Errorf("asr: Transcript.LatencyMS must be >= 0 (got %d)", t.LatencyMS)
	}
	if t.Confidence < 0 || t.Confidence > 1 {
		return fmt.Errorf("asr: Transcript.Confidence must be in [0,1] (got %f)", t.Confidence)
	}
	if t.Text == "" && t.IsFinal {
		// Final transcripts with empty text are suspicious; partials
		// during silence can legitimately be empty.
		return fmt.Errorf("asr: final Transcript must contain non-empty Text")
	}
	return nil
}
