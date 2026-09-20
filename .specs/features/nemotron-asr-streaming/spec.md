# Feature: nemotron-asr-streaming

> **Status**: Specify phase — design document is [ADR-045](../../../docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md); this spec captures WHAT (testable requirements + acceptance criteria), not the architectural discussion (already done).

## Problem Statement

A pesquisa autoral de 2026-09-18 define **F2 — Voz** como prioridade P1 do my-memory pós-Foundation. O ADR-043 (envelope de eventos canônico + event_runtime) já está merged e estabelece o substrato sobre o qual `stt.partial` / `stt.final` eventos fluem. O ADR-045 (Nemotron 3.5 ASR via ONNX Runtime Go in-process, Accepted pattern) decide **qual modelo** e **como rodar** (in-process Go, sem worker Python sidecar). Falta o **WHAT testável**: como exatamente o cliente ASR emite eventos, em que ordem, com que garantias, e como verificar latência sub-100ms em produção.

Sem este spec, F2 (voz) vira implementação ad-hoc que reinventa contratos a cada subsistema (caller do mic, projection subscriber, audit subscriber com redaction LLM02, MCP tool `asr_transcribe`). Exatamente o que ADR-045 + ADR-043 querem evitar.

## Goals

- [ ] Implementar `internal/asr/nemotron_onnx.go` carregando modelo int4 (`nemotron-3.5-asr-streaming-0.6b`) via `github.com/yalue/onnxruntime_go`, com cache-aware attention state preservado entre chunks.
- [ ] Implementar `internal/asr/selector.go` selecionando provider único (`onnx-nemotron`); fail-closed com hint claro se modelo ausente.
- [ ] Implementar `internal/asr/model_downloader.go` baixando de HuggingFace (`nvidia/nemotron-3.5-asr-streaming-0.6b` ou variante GGUF/CrispASR se ONNX oficial ainda indisponível) para `.memory/models/nemotron-asr-int4.onnx`.
- [ ] Expor CLI: `mem asr download`, `mem asr doctor`, `mem asr test --audio <path>`, `mem asr emit --session <id>`.
- [ ] Integrar com event_runtime (ADR-043): emitir `stt.partial` a cada chunk decodificado e `stt.final` ao detectar end-of-utterance, com `correlation_id` propagado do turno de conversa.
- [ ] Aplicar redaction LLM02 (ADR-050) ao `payload.transcript` antes de `event_log.Append` — transcript bruto NUNCA persiste.
- [ ] Suportar `target_lang: "auto"` com detecção automática entre os 40 locales do modelo (pt-BR + pt-PT inclusos).
- [ ] Configurar chunk_size em 80/160/320/560/1120ms (`att_context_size`) via `.memory/config.yaml`.
- [ ] Cobrir invariantes com testes: `TestNemotron_StreamingLatency_<100ms`, `TestNemotron_PtBrWER_Acceptable`, `TestNemotron_LocaleAutoDetect`, `TestNemotron_RedactionApplied`, `TestSelector_FailsClosedOnMissingModel`.

## Out of Scope

| Feature | Reason |
| --- | --- |
| Python sidecar ASR (Whisper.cpp / Vosk / sherpa-onnx) | **DEFERRED** per [ADR-045 §Deferral](../../../../docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md). Ativação exige gatilho documentado + ADR-047. |
| NeMo PyTorch sidecar com `.nemo` checkpoint | Mesma justificativa do item anterior. |
| CrispASR subprocess (Rust via JSON-RPC) | Avaliado e descartado em ADR-045 §Considered Options #2. |
| Whisper-large-v3 batch transcription | Streaming é requisito (latência sub-100ms); batch é caminho diferente e cobre caso de uso distinto (transcrição de arquivo) que ADR-045 não endereça. |
| TTS (Text-to-Speech) | F2 inclui voz mas TTS é escopo separado; ADR-042 mantém Python sidecar pra TTS Piper. |
| Wake-word detection ("Hey my-memory") | Camada adicional fora do escopo desta spec; ADR-047+ ou roadmap posterior. |
| Voice activity detection (VAD) custom | Reutilizar Silero VAD via `internal/asr/vad.go` é enhancement posterior; começar com energy-based VAD in-process. |
| Streaming via WebRTC / RTP / SIP | Transporte físico de áudio é camada do MCP server autoral (F4); esta spec recebe bytes já no `mymemoryd`. |
| Quantização dinâmica (fp16, q4_k) do modelo | Benchmark local primeiro; int4 é o default documentado no ADR-045. Outras quantizações viriam em spec separada após medição de WER/latência. |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --- | --- | --- | --- |
| Source do modelo ONNX | Reimplementação do paper 2604.14493 ("Pushing the Limits of On-Device Streaming ASR") — int4 quantization | Único caminho confirmado publicamente com preservação de streaming cache-aware; GGUF/CrispASR é alternativa se a reimplementação ONNX demorar >90d (gatilho 2 do ADR-045 §Deferral) | n (verificar disponibilidade antes da T1) |
| Sample rate de áudio | 16 kHz mono PCM | Wyoming-spec convention; padrão da indústria pra ASR streaming | y |
| Chunk size default | 160ms | Equilíbrio entre latência e throughput; ADR-045 §Configuração sugere 160 como default | y |
| `target_lang` default | `"auto"` | 40 locales do modelo cobrem requirement de pt-BR + pt-PT sem precisar de 2 modelos; detecção automática é trivial | y |
| Modelo int4 tamanho | ~670 MB | ADR-045 §Decision Outcome; lazy download (não vai no binário base) | y |
| Modelo storage path | `.memory/models/nemotron-asr-int4.onnx` | Convencionado em ADR-045 §Configuração | y |
| Path de download HuggingFace | `https://huggingface.co/nvidia/nemotron-3.5-asr-streaming-0.6b` (ou variante GGUF) | Source canônico NVIDIA; openmdw-1.1 license compatível | y |
| Forma de detecção de end-of-utterance | Threshold de silence + buffer 320ms; emitir `stt.final` quando silence detectado >500ms | Heurística conservadora; tuning viria com corpus real pt-BR | y |
| Backpressure no producer de eventos | Reusar `MaxAckPending = 64` do dispatcher (ADR-043) | Sem novo código de backpressure; reuso de contrato existente | y |
| Política de cold start | Warm-up na inicialização do `mymemoryd` carrega modelo em goroutine separada e bloqueia `mem asr doctor` até ready | Mitiga cold start 2-5s; ADR-045 §Negative já flagra | y |
| Storage do modelo durante testes | `:memory:` SQLite + diretório temporário pra `.onnx` | Testes não devem baixar 670 MB em CI; usar fixture menor (5-10 MB) ou mock do binding | y |
| Modelo de fixture para testes | Whisper-tiny int4 ou encoder-decoder mock | Trade-off: cobertura real vs velocidade CI; começar com mock + smoke test no modelo real em job separado | n (decidir antes da T2) |
| Compatibilidade macOS arm64 | Verificar `onnxruntime_go` linkage com `onnxruntime-osx-arm64-*.dylib` antes de T1 | Plataforma suportada mas binding é community-maintained | n (smoke test antes da T1) |

**Open questions:** two — modelo fonte ONNX disponibilidade + fixture strategy para testes. Resolver antes da T1.

---

## User Stories

### P1: Cliente Nemotron 3.5 in-process ⭐ MVP

**User Story**: As a `mymemoryd`, I want carregar o modelo Nemotron 3.5 ASR streaming em-process via ONNX Runtime Go so that transcrições parciais cheguem com latência <100ms sem IPC overhead.

**Why P1**: É o substrato sobre o qual todos os outros P-stories constroem. Sem cliente funcionando e com streaming real, o resto é placebo.

**Acceptance Criteria**:

1. The system SHALL load the model from `.memory/models/nemotron-asr-int4.onnx` via `github.com/yalue/onnxruntime_go` at `mymemoryd` startup. (ubiquitous)
2. The system SHALL preserve cache-aware attention state across consecutive audio chunks within the same session. (ubiquitous)
3. WHEN a chunk of 160ms (configurable 80/160/320/560/1120ms) of 16 kHz mono PCM is fed THEN the system SHALL emit a partial transcript via the configured callback within 100ms (p95 measured on Linux x86_64, 4 cores). (event-driven)
4. The system SHALL support `target_lang: "auto"` detecting locale automatically across the model's 40 supported locales, including pt-BR and pt-PT. (ubiquitous)
5. IF the model file is missing from disk THEN the system SHALL return `ErrModelNotFound` with hint `Run 'mem asr download --provider onnx-nemotron' to fetch`. (unwanted-behavior)
6. IF ONNX Runtime fails to initialize (linking error, version mismatch) THEN the system SHALL fail-closed with the same hint above and log a single warning (no per-chunk retry spam). (unwanted-behavior)
7. The system SHALL expose `Transcript{Text string, Language string, Confidence float64, Tokens []Token, LatencyMS int64}` as the Go return type from the streaming callback. (ubiquitous)

**Independent Test**: `TestNemotron_StreamingLatency_<100ms` mede p95 de latência entre `chunk_in` e `partial_out` em 100 chunks consecutivos; `TestNemotron_PtBrWER_Acceptable` mede WER em corpus pt-BR com ground-truth (`prova-md` data) — falha se WER > 12% (limiar conservador pra int4); `TestNemotron_LocaleAutoDetect` valida detecção `pt-BR` em 5 amostras.

---

### P2: Integração com event_runtime

**User Story**: As a `mymemoryd`, I want cada transcrição parcial virar um envelope `stt.partial` e cada fim de utterance virar `stt.final` so that replay, audit, e projeções conversacionais funcionem sem canal paralelo.

**Why P2**: Sem este hook, o ASR vira ilha. Audit (ADR-050 LLM02 redaction) e replay (memória reconstruída do event log) só funcionam se o ASR passa pelo envelope canônico.

**Acceptance Criteria**:

1. WHEN a partial transcript is ready THEN the system SHALL emit `stt.partial` envelope via `event_runtime.Log.Append` with `event_type="stt.partial"`, `aggregate_id=<session_id>`, `correlation_id=<conversation_turn_id>`, `payload.transcript` redacted per ADR-050 LLM02. (event-driven)
2. WHEN end-of-utterance is detected (>500ms silence) THEN the system SHALL emit `stt.final` envelope with `event_type="stt.final"`, `payload.transcript` (still redacted), `payload.duration_ms`, `payload.locale`. (event-driven)
3. The system SHALL propagate the `correlation_id` from the upstream conversation turn (passed in constructor) into both `stt.partial` and `stt.final` envelopes. (ubiquitous)
4. IF `event_runtime.Log.Append` fails (DB locked, disk full) THEN the system SHALL buffer the event up to 64 envelopes in memory and retry with backoff (100ms base, 30s cap, jitter) — matching ADR-043 §6 retry contract. (unwanted-behavior)
5. The system SHALL NEVER log raw `payload.transcript` to stdout/stderr; audit log receives only `redacted_payload_hash` per ADR-050 LLM02. (ubiquitous)

**Independent Test**: `TestNemotron_EmitsPartialEnvelope` valida 100 envelopes `stt.partial` consecutivos para 100 chunks, todos com mesmo `correlation_id` e `sequence` monotônico; `TestNemotron_EmitsFinalEnvelope` valida transição para `stt.final` após silence; `TestNemotron_RedactionApplied` valida que nenhum transcript bruto vaza em `event_log.payload`.

---

### P3: CLI + Model Downloader + Doctor

**User Story**: As a operador, I want baixar o modelo, diagnosticar o provider ativo, e testar com um sample sem precisar escrever código Go so that onboarding e troubleshooting sejam self-service.

**Why P3**: Sem CLI o usuário precisa escrever um teste Go pra validar setup. Doctor é mandatório pra `--strict` mode do CI (ADR-036 quality gate).

**Acceptance Criteria**:

1. WHEN `mem asr download --provider onnx-nemotron` is invoked THEN the system SHALL download the int4 model from HuggingFace and save to `.memory/models/nemotron-asr-int4.onnx` with SHA-256 verification. (event-driven)
2. WHEN `mem asr download` is invoked without internet THEN the system SHALL return exit 3 with hint `Place model manually at .memory/models/ and pass --model-path`. (unwanted-behavior)
3. WHEN `mem asr doctor` is invoked THEN the system SHALL report: provider ativo, modelo path, sha256, tamanho, locale default, p95 latência medida (rolling window de 100 chunks), erros recentes. (event-driven)
4. WHEN `mem asr test --audio ./sample.wav` is invoked THEN the system SHALL transcrever o arquivo, imprimir o texto redacted, e exit 0 (sucesso) ou 2 (erro de provider/modelo). (event-driven)
5. WHEN `mem asr emit --session <id>` is invoked THEN the system SHALL capturar áudio do microfone (loopback ou device configurado em `.memory/config.yaml`) e emitir envelopes `stt.partial`/`stt.final` em modo interativo (não bloqueia CLI forever — `Ctrl+C` = flush final + exit). (event-driven)

**Independent Test**: `TestCliDownload_ResumesPartial` valida resume de download parcial via Range header; `TestCliDoctor_ReportsAllFields` valida JSON output contém todos os 7 campos; `TestCliTest_RunOnFixture` valida transcrição de fixture pt-BR (5s, ~30 palavras).

---

### P4: Configuração Declarativa + Auto-Scoping

**User Story**: As a operador, I want configurar provider, modelo path, chunk size, locale, e num_threads via `.memory/config.yaml` so that mudanças não exijam rebuild do binário.

**Why P4**: Pra deployment em múltiplas máquinas (laptop + homelab), config externa é obrigatória. Hoje embedder builtin (ADR-035) já segue esse padrão; replicar evita inconsistência.

**Acceptance Criteria**:

1. WHERE `.memory/config.yaml` has `asr.provider: "onnx-nemotron"` THEN the system SHALL load that provider. (optional-feature)
2. WHERE `asr.onnx_nemotron.model_path` is empty THEN the system SHALL default to `.memory/models/nemotron-asr-int4.onnx`. (optional-feature)
3. WHERE `asr.onnx_nemotron.chunk_ms` is one of `80|160|320|560|1120` THEN the system SHALL use that value. (optional-feature)
4. WHERE `asr.onnx_nemotron.num_threads` is `0` THEN the system SHALL auto-detect via `runtime.NumCPU()`. (optional-feature)
5. IF `asr.onnx_nemotron.chunk_ms` is not in the allowed set THEN the system SHALL reject with `ErrInvalidChunkSize` at startup. (unwanted-behavior)
6. The system SHALL validate config at startup and exit with `ErrInvalidConfig` listing all violations before binding ONNX. (unwanted-behavior)

**Independent Test**: `TestConfig_ValidatesChunkSize` rejeita `300`; `TestConfig_AppliesDefaults` valida todos os defaults quando YAML omitido.

---

## Edge Cases

- IF ONNX binding fails to link on macOS arm64 (M1/M2/M3) THEN system SHALL emit warning com hint `brew install onnxruntime` e fail-closed. (unwanted-behavior)
- WHEN modelo corrompido (SHA-256 mismatch) is detected at startup THEN system SHALL refuse to load and suggest `mem asr download` again. (unwanted-behavior)
- WHILE model is being downloaded concurrently with `mymemoryd` startup, the system SHALL wait via mutex or exit with hint `re-run after download completes`. (state-driven)
- IF a chunk arrives with sample rate != 16 kHz THEN system SHALL reject with `ErrUnsupportedSampleRate` (sem resampling — ADR-045 §Decision Outcome fixa 16 kHz). (unwanted-behavior)
- IF GPU is available (CUDA/ROCm/Metal) THEN system SHALL use GPU only if `asr.onnx_nemotron.device: "gpu"` is explicit; default é CPU (binding CPU-only estável). (optional-feature)

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| --- | --- | --- | --- |
| ASR-01 | P1 | Design | Pending |
| ASR-02 | P1 | Design | Pending |
| ASR-03 | P1 | Design | Pending |
| ASR-04 | P1 | Design | Pending |
| ASR-05 | P1 | Design | Pending |
| ASR-06 | P1 | Design | Pending |
| ASR-07 | P1 | Design | Pending |
| ASR-08 | P2 | Design | Pending |
| ASR-09 | P2 | Design | Pending |
| ASR-10 | P2 | Design | Pending |
| ASR-11 | P2 | Design | Pending |
| ASR-12 | P2 | Design | Pending |
| ASR-13 | P3 | Design | Pending |
| ASR-14 | P3 | Design | Pending |
| ASR-15 | P3 | Design | Pending |
| ASR-16 | P3 | Design | Pending |
| ASR-17 | P3 | Design | Pending |
| ASR-18 | P4 | Design | Pending |
| ASR-19 | P4 | Design | Pending |
| ASR-20 | P4 | Design | Pending |
| ASR-21 | P4 | Design | Pending |
| ASR-22 | P4 | Design | Pending |
| ASR-23 | P4 | Design | Pending |

**Coverage:** 23 total, 0 mapped to tasks, 23 unmapped ⚠️ (tasks.md pending — auto-generated after Execute starts).

---

## Success Criteria

- [ ] `mymemoryd --asr-test sample.wav` retorna texto redacted em <500ms para arquivos de 5s.
- [ ] p95 latência streaming ≤100ms em CPU x86_64 4-core (medido em `mem asr doctor`).
- [ ] WER em corpus pt-BR ≤12% (int4) — confirmado em benchmark local antes de `Accepted`.
- [ ] Zero `transcript` bruto em `event_log.payload` (validado por `TestNemotron_RedactionApplied` + grep de regressão).
- [ ] Cold start ≤5s com modelo carregado (warm-up async).
- [ ] `go test -count=1 ./internal/asr/...` passa 100%.
- [ ] `mem asr doctor` retorna JSON parseável com os 7 campos documentados.
- [ ] ADR-046 §Deferral: Plano B Python NÃO é instanciado enquanto gatilhos não dispararem (validado por `TestNoPythonSidecar_InDefaultBuild`).
