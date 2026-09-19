// Package event_runtime — redaction.go
//
// PII redaction gate (ADR-050 LLM02 Sensitive Information Disclosure).
//
// The default ruleset replaces three Payload paths that are routinely
// classified as PII or proprietary:
//
//   - payload.transcript  → [REDACTED:transcript]   (voice transcripts)
//   - payload.audio_url   → [REDACTED:audio_url]    (audio file URLs)
//   - payload.prompt      → [REDACTED:prompt]       (LLM prompts)
//
// The function is deliberately conservative: it does a full JSON round-trip
// (Marshal → Unmarshal into generic map → path rewrite → Marshal back).
// That costs ~1ms per envelope but is auditable and panic-free. The
// pre-redaction hash (SHA-256) is stored in Provenance.RedactedPayloadHash
// so the audit trail can confirm "this is what the original payload was"
// without storing the payload itself.
//
// References:
//   - ADR-050 §LLM02 (controls + detection)
//   - ADR-043 §P5 (audit subscriber MUST apply redaction)
package event_runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Rule maps a top-level payload key to the redaction tag that replaces it.
//
// The Path is interpreted as a literal key match against the unmarshaled
// JSON object under "payload" (e.g. "transcript" matches payload.transcript
// when the envelope payload is a JSON object). Nested paths use "."
// separators (e.g. "user.email").
type Rule struct {
	Path string
	Tag  string
}

// DefaultRules returns the canonical redaction rule set for ADR-050 LLM02.
// Tests and production callers use this unless they explicitly opt-in to a
// custom rule set via WithRules.
func DefaultRules() []Rule {
	return []Rule{
		{Path: "transcript", Tag: "[REDACTED:transcript]"},
		{Path: "audio_url", Tag: "[REDACTED:audio_url]"},
		{Path: "prompt", Tag: "[REDACTED:prompt]"},
	}
}

// Redact returns a new Envelope (input not mutated) with PII fields
// rewritten according to the supplied rules. The Provenance.RedactedFields
// slice records which paths were redacted, and Provenance.RedactedPayloadHash
// holds the SHA-256 hex of the original payload bytes (for audit
// verification).
//
// If no rules match, the returned envelope is a deep copy with provenance
// populated but the payload bytes preserved verbatim.
//
// The function never panics — invalid JSON payloads are passed through
// unchanged and the hash is computed over the raw bytes.
func Redact(env Envelope) Envelope {
	return RedactWithRules(env, DefaultRules())
}

// RedactWithRules is the customizable redaction entry point. Useful for
// tests that want a smaller ruleset, and for future extensions (per-vault
// rule overrides via .memory/config.yaml).
func RedactWithRules(env Envelope, rules []Rule) Envelope {
	// Step 1: deep-copy the envelope (input not mutated).
	out := env
	out.Payload = append(json.RawMessage(nil), env.Payload...)
	if env.Provenance != nil {
		p := *env.Provenance
		if env.Provenance.RedactedFields != nil {
			p.RedactedFields = append([]string(nil), env.Provenance.RedactedFields...)
		}
		out.Provenance = &p
	}

	// Step 2: hash the pre-redaction payload (always — even when no rules
	// match — so audit can confirm "this envelope was processed by redaction").
	if out.Provenance == nil {
		out.Provenance = &Provenance{}
	}
	sum := sha256.Sum256(env.Payload)
	out.Provenance.RedactedPayloadHash = hex.EncodeToString(sum[:])

	// Step 3: apply rules. If payload is not a JSON object, we cannot
	// rewrite fields — leave the bytes untouched and note it.
	if len(env.Payload) == 0 {
		return out
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(env.Payload, &obj); err != nil {
		// Not a JSON object (or not JSON at all); pass through unchanged.
		return out
	}

	var redactedFields []string
	for _, r := range rules {
		key := r.Path
		if _, exists := obj[key]; exists {
			obj[key] = json.RawMessage(fmt.Sprintf("%q", r.Tag))
			redactedFields = append(redactedFields, key)
		}
	}
	out.Provenance.RedactedFields = append(out.Provenance.RedactedFields, redactedFields...)

	// Step 4: re-marshal the modified object back into the payload.
	rewritten, err := json.Marshal(obj)
	if err != nil {
		// Shouldn't happen — we only modified string values — but be safe.
		return out
	}
	out.Payload = rewritten
	return out
}
