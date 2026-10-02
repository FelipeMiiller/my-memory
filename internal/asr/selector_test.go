package asr

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// stubProvider implements Provider for Selector unit tests. Lets us
// exercise delegation, latency reporting, and Close semantics without
// pulling in the real ONNX binding (T4).
type stubProvider struct {
	locale     Locale
	chunkFn    func(ctx context.Context, pcm []byte) (Transcript, error)
	closed     bool
	streamCall int
}

func (s *stubProvider) StreamChunk(ctx context.Context, pcm []byte) (Transcript, error) {
	s.streamCall++
	if s.chunkFn != nil {
		return s.chunkFn(ctx, pcm)
	}
	return Transcript{Text: "stub", Language: s.locale, Confidence: 0.5, LatencyMS: 10}, nil
}

func (s *stubProvider) Locale() Locale { return s.locale }
func (s *stubProvider) Close() error   { s.closed = true; return nil }

// silentLogger discards log output so test runs stay quiet.
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// withStubProvider replaces newProviderFunc for the duration of a test so
// the Selector uses our stub instead of the noop default. Restores on
// cleanup.
func withStubProvider(t *testing.T, p Provider) {
	t.Helper()
	prev := newProviderFunc
	newProviderFunc = func(ctx context.Context, cfg *Config) (Provider, error) {
		return p, nil
	}
	t.Cleanup(func() { newProviderFunc = prev })
}

// tempModelFile creates a file at path/to/dir/model.onnx so the
// Selector's `os.Stat` check passes. Contents don't matter; the
// Selector only verifies existence.
func tempModelFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "nemotron-asr-int4.onnx")
	if err := os.WriteFile(path, []byte("fake-model-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSelector_MissingModel_ReturnsErrModelNotFound(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "no-such-model.onnx")
	cfg := &Config{
		Provider: SupportedProvider,
		OnnxNemotron: OnnxNemotronConfig{
			ModelPath:  missing,
			TargetLang: "auto",
			ChunkMS:    160,
			NumThreads: 2,
		},
	}

	sel, err := New(context.Background(), cfg, silentLogger())
	if sel != nil {
		t.Errorf("expected nil Selector when model missing; got %v", sel)
	}
	if err == nil {
		t.Fatal("expected ErrModelNotFound; got nil")
	}
	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("expected ErrModelNotFound; got %v", err)
	}
	if HintFor(err) == "" {
		t.Errorf("expected HintFor to return remediation hint for missing model")
	}
}

func TestSelector_ValidModel_InitializesProvider(t *testing.T) {
	stub := &stubProvider{locale: LocalePtBR}
	withStubProvider(t, stub)

	cfg := defaultConfigForTest(t, tempModelFile(t))
	sel, err := New(context.Background(), cfg, silentLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if sel == nil {
		t.Fatal("expected non-nil Selector")
	}
	if sel.Active() == nil {
		t.Error("expected active Provider after successful init")
	}
	if sel.Config() == nil {
		t.Error("Selector.Config() should return the input Config")
	}
}

func TestSelector_Transcript_DelegatesToProvider(t *testing.T) {
	wantLatency := int64(42)
	stub := &stubProvider{locale: LocalePtBR,
		chunkFn: func(ctx context.Context, pcm []byte) (Transcript, error) {
			return Transcript{
				Text:       "olá memória",
				Language:   LocalePtBR,
				Confidence: 0.9,
				Tokens:     []Token{{Text: "olá", Confidence: 0.9}},
				LatencyMS:  wantLatency,
				IsFinal:    false,
			}, nil
		}}
	withStubProvider(t, stub)

	cfg := defaultConfigForTest(t, tempModelFile(t))
	sel, err := New(context.Background(), cfg, silentLogger())
	if err != nil {
		t.Fatal(err)
	}

	pcm := make([]byte, cfg.OnnxNemotron.ChunkMS*32)
	tr, err := sel.Transcript(context.Background(), pcm)
	if err != nil {
		t.Fatalf("Transcript: %v", err)
	}
	if tr.Text != "olá memória" {
		t.Errorf("Text round-trip: got %q want %q", tr.Text, "olá memória")
	}
	if tr.LatencyMS != wantLatency {
		t.Errorf("latency: got %d want %d", tr.LatencyMS, wantLatency)
	}
	if stub.streamCall != 1 {
		t.Errorf("provider call count: got %d want 1", stub.streamCall)
	}
}

func TestSelector_Close_ReleasesProvider(t *testing.T) {
	stub := &stubProvider{locale: LocaleEnUS}
	withStubProvider(t, stub)

	cfg := defaultConfigForTest(t, tempModelFile(t))
	sel, err := New(context.Background(), cfg, silentLogger())
	if err != nil {
		t.Fatal(err)
	}
	if err := sel.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if !stub.closed {
		t.Error("expected provider Close to be called")
	}
}

func TestSelector_StreamUntilCancel_ConveysContextCancel(t *testing.T) {
	stub := &stubProvider{locale: LocaleEnUS}
	withStubProvider(t, stub)

	cfg := defaultConfigForTest(t, tempModelFile(t))
	sel, err := New(context.Background(), cfg, silentLogger())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := make(chan []byte)
	go func() {
		<-ctx.Done()
		close(in)
	}()
	cancel() // cancel before sending any chunks

	out, errCh := sel.StreamUntilCancel(ctx, in)
	// Both channels must close; errCh receives the cancellation error.
	select {
	case _, ok := <-out:
		if ok {
			// drain remaining
		}
	default:
	}
	if err := <-errCh; err == nil || !errors.Is(err, context.Canceled) {
		t.Errorf("StreamUntilCancel err = %v, want context.Canceled", err)
	}
}

func TestSelector_StreamUntilCancel_PropagatesProviderError(t *testing.T) {
	provErr := errors.New("mock provider failure")
	stub := &stubProvider{
		locale: LocalePtBR,
		chunkFn: func(ctx context.Context, pcm []byte) (Transcript, error) {
			return Transcript{}, provErr
		},
	}
	withStubProvider(t, stub)

	cfg := defaultConfigForTest(t, tempModelFile(t))
	sel, err := New(context.Background(), cfg, silentLogger())
	if err != nil {
		t.Fatal(err)
	}

	in := make(chan []byte, 1)
	in <- make([]byte, cfg.OnnxNemotron.ChunkMS*32)
	close(in)

	out, errCh := sel.StreamUntilCancel(context.Background(), in)
	for range out {
		// drain
	}
	if err := <-errCh; err == nil || err.Error() != provErr.Error() {
		t.Errorf("StreamUntilCancel err = %v, want %v", err, provErr)
	}
}

// defaultConfigForTest returns a Config with sensible defaults pointing
// at modelPath. Saves repetitive boilerplate in tests.
func defaultConfigForTest(t *testing.T, modelPath string) *Config {
	t.Helper()
	cfg := &Config{
		Provider: SupportedProvider,
		OnnxNemotron: OnnxNemotronConfig{
			ModelPath:  modelPath,
			TargetLang: "auto",
			ChunkMS:    160,
			NumThreads: 2,
		},
	}
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("defaultConfigForTest validation: %v", err)
	}
	return cfg
}
