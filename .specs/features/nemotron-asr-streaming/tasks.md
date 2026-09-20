# nemotron-asr-streaming Tasks

## Execution Protocol (MANDATORY — do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path.

**If the skill cannot be activated, STOP and tell the user — do not proceed without it.**

---

**Spec**: `.specs/features/nemotron-asr-streaming/spec.md`
**Design**: [ADR-045](../../../docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md)
**Security**: [ADR-050 (Threat Model OWASP LLM)](../../../docs/adr/050-threat-model-owasp-llm-aplicado-ao-mymemory.md) — controls LLM02 (transcript redaction) apply.
**Status**: Draft → Approved → In Progress → Done

---

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| --- | --- | --- | --- | --- |
| Service (Nemotron client, selector) | unit + integration | All branches; latency p95 ≤100ms; redaction applied | `internal/asr/*_test.go` | `go test -count=1 ./internal/asr/...` |
| Domain (Transcript, Locale, ChunkSize) | unit | Type validation; enum parsing | `internal/asr/types_test.go` | `go test -count=1 ./internal/asr/...` |
| Model downloader (HuggingFace fetch) | integration | Range header resume; SHA-256 verification | `internal/asr/downloader_test.go` | `go test -count=1 ./internal/asr/...` |
| CLI (mem asr download/doctor/test/emit) | integration | Happy path + every listed edge case | `cmd/mem/asr_test.go` | `go test -count=1 ./cmd/mem/...` |
| Config (yaml schema + defaults) | unit | YAML parse; defaults applied; chunk_size validation | `internal/asr/config_test.go` | `go test -count=1 ./internal/asr/...` |

---

## Gate Check Commands

| Gate Level | When to Use | Command |
| --- | --- | --- |
| Quick | After unit-only tasks (T2, T3, T8) | `go test -count=1 -short ./internal/asr/...` |
| Full | After integration tasks (T1, T4, T5, T6, T7, T9) | `go test -count=1 ./internal/asr/... ./cmd/mem/...` |
| Build | After phase completion | `gofmt -l . && go build ./... && go test -count=1 ./...` |

---

## Execution Plan

### Phase 1: Types + selector skeleton

Domain types, single-provider selector, fail-closed on missing model.

```
T1 -> T2
```

### Phase 2: Nemotron ONNX client (streaming)

Model loading + streaming inference via `onnxruntime_go`.

```
T2 -> T4
T4 -> T5
```

### Phase 3: event_runtime integration + redaction

Emit `stt.partial` / `stt.final` envelopes with redaction.

```
T5 -> T6
T6 -> T7
```

### Phase 4: CLI + model downloader + config

CLI subcommands + HuggingFace downloader + config validation.

```
T3 -> T9
T6 -> T9
```

Total: **9 tasks across 4 phases**. Fits 2 batches at Execute time.

---

## Task Breakdown

### Phase 1

### T1: Domain types + errors

**What**: Create `internal/asr/types.go` with `Transcript`, `Locale` (enum: `auto|pt-BR|pt-PT|en-US|...`), `ChunkSize` (enum: 80|160|320|560|1120), `Confidence`, `Token`, `LatencyMS`. Create `internal/asr/errors.go` with sentinels: `ErrModelNotFound`, `ErrInvalidChunkSize`, `ErrInvalidLocale`, `ErrUnsupportedSampleRate`, `ErrModelIntegrity`.
**Where**: `internal/asr/types.go` (new), `internal/asr/errors.go` (new)
**Depends on**: None
**Reuses**: Existing sentinel pattern from `internal/event_runtime/errors.go`.
**Requirement**: ASR-07
**Status**: Pending

**Done when**:

- [ ] All types exported with JSON tags
- [ ] All sentinels exported with descriptive docstrings
- [ ] `ChunkSize` validates against allowed set {80, 160, 320, 560, 1120}
- [ ] `Locale` enum supports `auto` + 40 documented locales
- [ ] `Transcript.MarshalJSON` round-trips cleanly
- [ ] Tests in `internal/asr/types_test.go` cover validation

**Tests**: unit
**Gate**: quick
**Commit**: `feat(asr): domain types + sentinel errors`

---

### T2: Selector (single-provider, fail-closed)

**What**: Create `internal/asr/selector.go` with `Selector` struct holding provider config. `New(ctx, cfg)` checks `.memory/models/nemotron-asr-int4.onnx` exists; returns `ErrModelNotFound` with hint if missing. `Transcript(ctx, audioChunk) (Transcript, error)` delegates to the active provider (currently only `onnx-nemotron`).
**Where**: `internal/asr/selector.go` (new), `internal/asr/selector_test.go` (new)
**Depends on**: T1
**Reuses**: `os.Stat` for file existence check.
**Requirement**: ASR-01, ASR-05, ASR-06
**Status**: Pending

**Done when**:

- [ ] `Selector` validates model file on construction
- [ ] Returns `ErrModelNotFound` with hint `Run 'mem asr download --provider onnx-nemotron'` if missing
- [ ] Logs single warning on ONNX init failure (no per-chunk retry spam)
- [ ] `Transcript(ctx, []byte)` delegates to provider, returns `Transcript` or wrapped error
- [ ] Test `TestSelector_MissingModel_ReturnsErrModelNotFound`
- [ ] Test `TestSelector_ValidModel_InitializesProvider`

**Tests**: integration
**Gate**: full
**Commit**: `feat(asr): selector + fail-closed on missing model`

---

### T3: Config schema + validation

**What**: Create `internal/asr/config.go` with `Config` struct matching ADR-045 §Configuration. `Load(yamlPath) (*Config, error)` parses YAML. Apply defaults when fields empty. Reject invalid `chunk_ms` with `ErrInvalidChunkSize`. Reject unknown `provider` (only `onnx-nemotron` accepted).
**Where**: `internal/asr/config.go` (new), `internal/asr/config_test.go` (new)
**Depends on**: None
**Reuses**: `gopkg.in/yaml.v3`.
**Requirement**: ASR-18, ASR-19, ASR-20, ASR-21, ASR-22, ASR-23
**Status**: Pending

**Done when**:

- [ ] `Config` struct matches ADR-045 §Configuration fields
- [ ] Defaults applied: `provider="onnx-nemotron"`, `model_path=".memory/models/nemotron-asr-int4.onnx"`, `target_lang="auto"`, `chunk_ms=160`, `num_threads=0`
- [ ] `num_threads=0` → auto-detect via `runtime.NumCPU()`
- [ ] Invalid `chunk_ms` → `ErrInvalidConfig` listing all violations
- [ ] Unknown `provider` → `ErrInvalidConfig`
- [ ] Test `TestConfig_ValidatesChunkSize` rejects 300
- [ ] Test `TestConfig_AppliesDefaults` validates all defaults

**Tests**: unit
**Gate**: quick
**Commit**: `feat(asr): config schema + defaults + validation`

---

### Phase 2

### T4: Nemotron ONNX client (model load + warm-up)

**What**: Create `internal/asr/nemotron_onnx.go` with `NemotronClient` struct loading model via `github.com/yalue/onnxruntime_go`. `Load(ctx, modelPath) (*NemotronClient, error)` initializes ORT environment, loads session. `Warmup(ctx)` runs inference on dummy input to mitigate cold start.
**Where**: `internal/asr/nemotron_onnx.go` (new), `internal/asr/nemotron_onnx_test.go` (new)
**Depends on**: T1, T2
**Reuses**: `github.com/yalue/onnxruntime_go` binding.
**Requirement**: ASR-01, ASR-02
**Status**: Pending

**Done when**:

- [ ] `NemotronClient` initializes ORT environment via `onnxruntime_go.NewEnvironment()`
- [ ] Loads session from `.onnx` file path
- [ ] `Warmup` runs 1 inference cycle on zero-filled input
- [ ] Returns `ErrModelNotFound` on file missing
- [ ] Returns wrapped error on ORT init failure
- [ ] Test `TestNemotronLoad_ValidModel` (uses tiny test fixture ~5 MB)
- [ ] Test `TestNemotronLoad_MissingFile_ReturnsErrModelNotFound`

**Tests**: integration (uses tiny model fixture)
**Gate**: full
**Commit**: `feat(asr-nemotron): ONNX client with warmup`

---

### T5: Streaming inference (chunk in → partial transcript)

**What**: Extend `NemotronClient` with `StreamChunk(ctx, pcm []byte) (Transcript, error)`. Preserves cache-aware attention state across consecutive chunks within same session. Sample rate: 16 kHz mono PCM. Configurable chunk size (80/160/320/560/1120 ms). Auto-detect locale when `target_lang="auto"`.
**Where**: `internal/asr/nemotron_onnx.go` (modify), `internal/asr/streaming_test.go` (new)
**Depends on**: T4
**Reuses**: Existing ONNX session + tensor APIs.
**Requirement**: ASR-02, ASR-03, ASR-04
**Status**: Pending

**Done when**:

- [ ] `StreamChunk` accepts 16 kHz mono PCM of `chunk_ms * 16` samples
- [ ] Rejects other sample rates with `ErrUnsupportedSampleRate`
- [ ] Preserves attention state across calls (cache-aware)
- [ ] Returns `Transcript{Text, Language, Confidence, Tokens, LatencyMS}`
- [ ] `target_lang="auto"` runs locale detection, returns detected code
- [ ] Test `TestNemotron_StreamingLatency_Pt95Lte100ms` measures 100 chunks
- [ ] Test `TestNemotron_LocaleAutoDetect` validates `pt-BR` detection on 5 samples
- [ ] Test `TestNemotron_PreservesStateAcrossChunks` validates cache-aware behavior

**Tests**: integration
**Gate**: full
**Commit**: `feat(asr-nemotron): streaming inference with cache-aware state`

---

### Phase 3

### T6: event_runtime integration (stt.partial / stt.final envelopes)

**What**: Extend `NemotronClient.StreamChunk` (or wrap in `Selector`) to emit `stt.partial` envelope via `event_runtime.Log.Append` on each partial. Add `Finalize(ctx, correlationID)` method emitting `stt.final` envelope with `payload.duration_ms` and `payload.locale`. Propagate `correlation_id` from constructor.
**Where**: `internal/asr/selector.go` (modify), `internal/asr/event_integration_test.go` (new)
**Depends on**: T5
**Reuses**: `event_runtime.Log.Append` from event-runtime-event-log.
**Requirement**: ASR-08, ASR-09, ASR-10
**Status**: Pending

**Done when**:

- [ ] Each `StreamChunk` emits `stt.partial` with `correlation_id`, `aggregate_id=<session_id>`, `payload.transcript` (redacted)
- [ ] `Finalize` emits `stt.final` with `payload.duration_ms`, `payload.locale`
- [ ] `correlation_id` propagated from constructor argument
- [ ] `payload.transcript` NEVER contains raw PII (redaction applied before Append)
- [ ] Backpressure: if `Log.Append` fails, buffer up to 64 envelopes, retry with backoff
- [ ] Test `TestNemotron_EmitsPartialEnvelope` validates 100 envelopes with same correlation_id
- [ ] Test `TestNemotron_EmitsFinalEnvelope` validates end-of-utterance transition
- [ ] Test `TestNemotron_RedactionApplied` validates no raw transcript in `event_log.payload`

**Tests**: integration
**Gate**: full
**Commit**: `feat(asr): event_runtime integration with stt.partial/stt.final`

---

### T7: End-of-utterance detection (silence threshold)

**What**: Add `internal/asr/vad.go` with energy-based VAD. Detects silence >500ms → triggers `Finalize(ctx, correlationID)`. Configurable silence threshold via `Config.VadSilenceMs` (default 500).
**Where**: `internal/asr/vad.go` (new), `internal/asr/vad_test.go` (new)
**Depends on**: T6
**Reuses**: Energy computation: `sum(abs(samples)) / N > threshold`.
**Requirement**: ASR-09 (end-of-utterance side)
**Status**: Pending

**Done when**:

- [ ] VAD computes energy per PCM chunk
- [ ] Tracks rolling silence duration
- [ ] Silence > `VadSilenceMs` (default 500) → `Finalize` called
- [ ] Configurable threshold via `Config.VadSilenceMs`
- [ ] Test `TestVad_DetectsSilenceAfterSpeech` simulates 1s speech + 600ms silence → Finalize called
- [ ] Test `TestVad_ContinuesOnShortPauses` simulates 200ms silence → no Finalize

**Tests**: unit + integration
**Gate**: full
**Commit**: `feat(asr): energy-based VAD with configurable silence threshold`

---

### Phase 4

### T8: Model downloader (HuggingFace fetch + SHA-256 verify)

**What**: Create `internal/asr/model_downloader.go` with `Download(ctx, modelURL, destPath) error` fetching from HuggingFace with Range header resume. Computes SHA-256 and verifies against `.memory/models/<name>.sha256`. If `.sha256` file missing, generates it on first download.
**Where**: `internal/asr/model_downloader.go` (new), `internal/asr/downloader_test.go` (new)
**Depends on**: T1
**Reuses**: `crypto/sha256`, `net/http` with Range header.
**Requirement**: ASR-13, ASR-14, ASR-22, ASR-23
**Status**: Pending

**Done when**:

- [ ] `Download` fetches model to `.memory/models/nemotron-asr-int4.onnx`
- [ ] Supports Range header resume for partial downloads
- [ ] Computes SHA-256, compares to `.sha256` if present, fails on mismatch with `ErrModelIntegrity`
- [ ] Generates `.sha256` on first successful download
- [ ] Returns `ErrModelNotFound` if HuggingFace returns 404
- [ ] Returns clear error if no internet + no local file
- [ ] Test `TestDownload_ResumesPartial` validates Range header behavior
- [ ] Test `TestDownload_VerifiesSHA256` validates mismatch rejection

**Tests**: integration
**Gate**: full
**Commit**: `feat(asr): model downloader with resume + SHA-256 verify`

---

### T9: CLI `mem asr {download, doctor, test, emit}`

**What**: Add CLI subcommands in `cmd/mem/asr.go`: `mem asr download --provider onnx-nemotron` (calls T8 downloader), `mem asr doctor` (reports provider, model path, sha256, size, locale, p95 latency, errors), `mem asr test --audio <path>` (transcribes file), `mem asr emit --session <id>` (interactive microphone capture).
**Where**: `cmd/mem/asr.go` (new), `cmd/mem/asr_test.go` (new)
**Depends on**: T3, T6
**Reuses**: `cobra` framework; `mem asr doctor --json` JSON output pattern from `cmd/mem/events.go`.
**Requirement**: ASR-13, ASR-14, ASR-15, ASR-16, ASR-17
**Status**: Pending

**Done when**:

- [ ] `mem asr download` downloads model with progress bar, verifies SHA-256, generates `.sha256`
- [ ] `mem asr download` without internet returns exit 3 with hint `Place model manually at .memory/models/`
- [ ] `mem asr doctor` prints 7 fields: provider, model_path, sha256, size, locale_default, p95_latency, errors_recent
- [ ] `mem asr doctor --json` outputs parseable JSON
- [ ] `mem asr test --audio ./sample.wav` transcribes, prints redacted text, exit 0/2
- [ ] `mem asr emit --session <id>` captures mic in interactive mode, Ctrl+C flushes
- [ ] Test `TestCliDownload_ResumesPartial` validates Range header
- [ ] Test `TestCliDoctor_ReportsAllFields` validates JSON output structure
- [ ] Test `TestCliTest_RunOnFixture` validates transcription of pt-BR fixture

**Tests**: integration
**Gate**: full
**Commit**: `feat(cli): mem asr download/doctor/test/emit`
