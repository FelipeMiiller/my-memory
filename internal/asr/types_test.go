package asr

import (
	"encoding/json"
	"testing"
)

func TestChunkSize_Valid(t *testing.T) {
	cases := []struct {
		in   ChunkSizeMS
		want bool
	}{
		{Chunk80MS, true},
		{Chunk160MS, true},
		{Chunk320MS, true},
		{Chunk560MS, true},
		{Chunk1120MS, true},
		{ChunkSizeMS(300), false},
		{ChunkSizeMS(0), false},
		{ChunkSizeMS(-1), false},
	}
	for _, c := range cases {
		if got := c.in.Valid(); got != c.want {
			t.Errorf("ChunkSizeMS(%d).Valid() = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestChunkSize_SamplesPerChunk(t *testing.T) {
	cases := []struct {
		in   ChunkSizeMS
		want int
	}{
		{Chunk80MS, 80 * 16},
		{Chunk160MS, 160 * 16},
		{Chunk320MS, 320 * 16},
		{Chunk560MS, 560 * 16},
		{Chunk1120MS, 1120 * 16},
	}
	for _, c := range cases {
		if got := c.in.SamplesPerChunk(); got != c.want {
			t.Errorf("ChunkSizeMS(%d).SamplesPerChunk() = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestLocale_ValidAndKnown(t *testing.T) {
	known := []Locale{LocaleAuto, LocalePtBR, LocalePtPT, LocaleEnUS}
	for _, l := range known {
		if !l.Valid() {
			t.Errorf("%q should be Valid", l)
		}
		if !l.Known() {
			t.Errorf("%q should be Known", l)
		}
	}
	// Arbitrary locale string the model may accept at inference time.
	bare := Locale("zh-CN")
	if !bare.Valid() {
		t.Errorf("bare locale %q should still be Valid() (runtime model accepts)", bare)
	}
	if bare.Known() {
		t.Errorf("bare locale %q should NOT be Known (not in our enumerated set)", bare)
	}
	// Empty string invalid.
	if Locale("").Valid() {
		t.Error("empty locale should not be Valid")
	}
}

func TestTranscript_MarshalJSON_Roundtrip(t *testing.T) {
	in := Transcript{
		Text:       "olá memória",
		Language:   LocalePtBR,
		Confidence: 0.92,
		Tokens: []Token{
			{Text: "olá", Confidence: 0.95, StartMS: 0, EndMS: 240},
			{Text: "memória", Confidence: 0.89, StartMS: 240, EndMS: 720},
		},
		LatencyMS: 87,
		IsFinal:   false,
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var out Transcript
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if out.Text != in.Text {
		t.Errorf("text round-trip mismatch: got %q want %q", out.Text, in.Text)
	}
	if out.Language != in.Language {
		t.Errorf("language round-trip mismatch: got %q want %q", out.Language, in.Language)
	}
	if out.Confidence != in.Confidence {
		t.Errorf("confidence mismatch: got %f want %f", out.Confidence, in.Confidence)
	}
	if out.LatencyMS != in.LatencyMS {
		t.Errorf("latency mismatch: got %d want %d", out.LatencyMS, in.LatencyMS)
	}
	if len(out.Tokens) != len(in.Tokens) {
		t.Errorf("tokens length mismatch: got %d want %d", len(out.Tokens), len(in.Tokens))
	}
}

func TestTranscript_MarshalJSON_EmptyTokensOmittedAsArray(t *testing.T) {
	// Partial transcripts with no decoded tokens yet should omit the
	// tokens field entirely (json:",omitempty" on the slice). The
	// MarshalJSON method ensures a non-nil empty slice so consumers
	// see a deterministic "" for missing tokens.
	in := Transcript{Text: "hi", Language: LocaleEnUS, Confidence: 0.5, LatencyMS: 30}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	if contains(got, `"tokens"`) {
		t.Errorf("empty Tokens slice should be omitted from JSON output; got %s", got)
	}
}

func TestTranscript_Validate(t *testing.T) {
	goodPartial := Transcript{Text: "x", Language: LocaleAuto, Confidence: 0.5, LatencyMS: 30}
	if err := goodPartial.Validate(); err != nil {
		t.Errorf("partial with non-empty text should validate: %v", err)
	}

	// Empty partial is OK (silence).
	if err := (&Transcript{Text: "", Language: LocaleAuto, Confidence: 0.5, LatencyMS: 30}).Validate(); err != nil {
		t.Errorf("partial with empty text should validate (silence): %v", err)
	}

	// Final with empty text is suspicious.
	finalEmpty := Transcript{Text: "", Language: LocaleAuto, Confidence: 0.5, LatencyMS: 100, IsFinal: true}
	if err := finalEmpty.Validate(); err == nil {
		t.Error("final Transcript with empty Text should fail Validate")
	}

	// Negative latency.
	negLat := Transcript{Text: "y", Language: LocaleAuto, Confidence: 0.5, LatencyMS: -1}
	if err := negLat.Validate(); err == nil {
		t.Error("Transcript with negative LatencyMS should fail Validate")
	}

	// Out-of-range confidence.
	badConf := Transcript{Text: "z", Language: LocaleAuto, Confidence: 1.5, LatencyMS: 50}
	if err := badConf.Validate(); err == nil {
		t.Error("Transcript with confidence > 1 should fail Validate")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
