package asr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// Provider is the minimal interface a streaming ASR provider must implement.
// Implemented by *NemotronClient (T4-T5 — future) and any test stub today.
//
// StreamChunk consumes one PCM chunk of `cfg.ChunkMS` duration at 16 kHz
// mono, returns the partial Transcript, or an error. Providers MUST
// preserve cache-aware attention state across calls when stateful.
type Provider interface {
	// StreamChunk runs one inference pass over the given PCM samples
	// (int16, 16 kHz, mono) and returns the partial transcript. MUST
	// preserve state across calls within the same session.
	StreamChunk(ctx context.Context, pcm []byte) (Transcript, error)

	// Locale returns the provider's currently active locale setting.
	// Useful for `mem asr doctor` reporting.
	Locale() Locale

	// Close releases any underlying resources (ONNX session, file
	// handles). Safe to call multiple times.
	Close() error
}

// Selector chooses an ASR provider based on Config.Provider and surfaces
// it to callers. Today only `onnx-nemotron` is active (Python-sidecar
// is DEFERRED per ADR-045 §Deferral); the routing logic is in place so
// future providers slot in without breaking call sites.
//
// Selector is safe for concurrent use. The active provider is set once
// at construction time (or zero value) and never changes.
type Selector struct {
	cfg     *Config
	active  Provider
	warning atomic.Pointer[slog.Record] // single warning emitted at init
	log     *slog.Logger
}

// New constructs a Selector from cfg. Performs fail-closed checks per
// ADR-045 §Decision Outcome:
//
//   - If cfg.OnnxNemotron.ModelPath is missing on disk, returns
//     ErrModelNotFound (the most common onboarding failure).
//   - If provider construction fails (e.g., ONNX Runtime link error on
//     macOS arm64), wraps the error with ErrProviderFailed and logs a
//     single warning (no per-chunk retry spam).
//   - mem asr download provides the workaround hint via HintFor.
func New(ctx context.Context, cfg *Config, log *slog.Logger) (*Selector, error) {
	if cfg == nil {
		return nil, fmt.Errorf("asr: Selector.New: cfg is nil")
	}
	if log == nil {
		log = slog.Default()
	}

	// Fail-closed: model file must exist on disk.
	if _, err := os.Stat(cfg.OnnxNemotron.ModelPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w (looked at %q)",
				ErrModelNotFound, cfg.OnnxNemotron.ModelPath)
		}
		return nil, fmt.Errorf("asr: stat model file: %w", err)
	}

	sel := &Selector{cfg: cfg, log: log}

	// Wire up the active provider. Today only `onnx-nemotron`; T4 will
	// construct the real NemotronClient. For now we use a noop provider
	// stub so the selector is exercised end-to-end without the ONNX
	// runtime dependency, which is gated behind a build tag in T4.
	provider, err := sel.newProvider(ctx)
	if err != nil {
		// Log single warning per ADR-045 (no per-chunk retry spam).
		log.Warn("asr: provider init failed",
			"provider", cfg.Provider,
			"err", err.Error(),
		)
		return nil, fmt.Errorf("%w: %w", ErrProviderFailed, err)
	}
	sel.active = provider
	return sel, nil
}

// newProvider is overridable by tests to inject a stub Provider. The
// production version (replaced in T4) constructs the NemotronClient.
var newProviderFunc func(ctx context.Context, cfg *Config) (Provider, error)

// newProvider returns the active provider per cfg.Provider. Today this
// always returns a noop stub; T4 will wire in the real NemotronClient
// behind the same interface.
func (s *Selector) newProvider(ctx context.Context) (Provider, error) {
	if newProviderFunc != nil {
		return newProviderFunc(ctx, s.cfg)
	}
	// Production stub: noop provider until T4 lands the ONNX client.
	// Selector logic itself (file-existence check, error wrapping, hint
	// surfacing) is exercised by tests.
	return newNoopProvider(s.cfg), nil
}

// Transcript delegates to the active provider. Returns ErrProviderFailed
// if no provider was constructed (e.g., constructor returned a Selector
// with nil active due to a recoverable init failure).
func (s *Selector) Transcript(ctx context.Context, pcm []byte) (Transcript, error) {
	if s.active == nil {
		return Transcript{}, ErrProviderFailed
	}
	return s.active.StreamChunk(ctx, pcm)
}

// Active returns the underlying Provider, useful for advanced callers
// (e.g., VAD hookup in T7 that needs direct Locale() lookup) and tests.
func (s *Selector) Active() Provider {
	return s.active
}

// Config returns the Config the Selector was constructed with. Useful
// for CLI subcommands (`mem asr doctor`) that need to report settings.
func (s *Selector) Config() *Config {
	return s.cfg
}

// Close releases the provider's resources.
func (s *Selector) Close() error {
	if s.active == nil {
		return nil
	}
	return s.active.Close()
}

// StreamUntilCancel runs the selector over chunks read from `in` until the
// caller cancels ctx or `in` is closed. Convenience wrapper used by the
// `mem asr emit` CLI subcommand (T9).
func (s *Selector) StreamUntilCancel(ctx context.Context, in <-chan []byte) (<-chan Transcript, <-chan error) {
	out := make(chan Transcript)
	errCh := make(chan error, 1)
	if s.active == nil {
		errCh <- ErrProviderFailed
		close(out)
		close(errCh)
		return out, errCh
	}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		defer close(out)
		for {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case pcm, ok := <-in:
				if !ok {
					return nil
				}
				t, err := s.Transcript(gctx, pcm)
				if err != nil {
					return err
				}
				select {
				case <-gctx.Done():
					return gctx.Err()
				case out <- t:
				}
			}
		}
	})
	go func() {
		err := g.Wait()
		errCh <- err
		close(errCh)
	}()
	return out, errCh
}

// newNoopProvider returns a deterministic Provider that fakes streaming
// by returning an empty Transcript. Lets Selector tests run before
// NemotronClient lands in T4.
func newNoopProvider(cfg *Config) Provider {
	return &noopProvider{cfg: cfg, locale: Locale(cfg.OnnxNemotron.TargetLang)}
}

type noopProvider struct {
	cfg    *Config
	locale Locale
	closed atomic.Bool
}

func (n *noopProvider) StreamChunk(ctx context.Context, pcm []byte) (Transcript, error) {
	if err := ctx.Err(); err != nil {
		return Transcript{}, err
	}
	// Validate input dimensions per ADR-045: 16 kHz mono PCM.
	expected := ChunkSizeMS(n.cfg.OnnxNemotron.ChunkMS).SamplesPerChunk() * 2 // int16 = 2 bytes per sample
	if len(pcm) != expected {
		return Transcript{}, fmt.Errorf("%w: got %d bytes, want %d (chunk_ms=%d @ 16 kHz s16le)",
			ErrUnsupportedSampleRate, len(pcm), expected, n.cfg.OnnxNemotron.ChunkMS)
	}
	return Transcript{
		Text:       "",
		Language:   n.locale,
		Confidence: 0.0,
		Tokens:     []Token{},
		LatencyMS:  0, // T4 will measure real latency
		IsFinal:    false,
	}, nil
}

func (n *noopProvider) Locale() Locale { return n.locale }

func (n *noopProvider) Close() error {
	if n.closed.Swap(true) {
		return nil
	}
	return nil
}
