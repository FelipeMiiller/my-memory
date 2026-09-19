# ADR-045: ASR Streaming Engine — Nemotron 3.5 (0.6B) Embutido via ONNX Runtime

- **Date**: 2026-09-19
- **Status**: Proposed
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: asr, voice, streaming, fastconformer, rnnt, nemotron, onnx-runtime, frugal, devx, language-pt-br

## Context and Problem Statement

A pesquisa autoral de 2026-09-18 (`pesquisa-infraestrutura-autoral-mymemory.md` §3.3) define transcrição streaming como parte do **F2 — Voz**. O ADR-042 (mymemoryd com workers sidecar isolados) atualmente pressupõe que ASR roda em **worker Python sidecar** (Whisper.cpp, Vosk, sherpa-onnx). Esse caminho tem dois problemas materiais para o caso de uso do my-memory:

1. **Latência de IPC in-process → subprocess** — workers Python exigem serialização JSON por chunk de áudio (16 kHz mono, ~512 bytes a cada 16ms). Para streaming sub-100ms, cada round-trip Go↔Python adiciona ~5-15ms mesmo em localhost.
2. **Topologia pesada pra MVP** — manter Python sidecar significa lifecycle manager adicional, manifestos de workers, perfis `.memory/profiles/*.yaml`, e graceful shutdown coordination. Tudo isso só pra rodar UM modelo.
3. **A pesquisa autoral §6 fixa Python como worker sidecar isolado, mas com escopo aberto** — não é prescritivo; a decisão entre in-process Go vs subprocess Python é de implementação.

Simultaneamente, o ADR-035 (Embedder Embutido com Fallback ONNX MiniLM) já estabelece o **padrão canônico do my-memory para inferência local in-process**: modelo quantizado carregado via `github.com/yalue/onnxruntime_go` (binding CGO para ONNX Runtime), tokenização BPE in-process, vetor exportado direto pra Go. Esse padrão foi escolhido especificamente por ser frugality-first (≤100 MB RAM adicional), zero-dependência externa opcional e licença permissive.

**O gatilho decisório:** descoberta em 2026-09-19 do modelo [`nvidia/nemotron-3.5-asr-streaming-0.6b`](https://huggingface.co/nvidia/nemotron-3.5-asr-streaming-0.6b) — 600M parâmetros, **Cache-Aware FastConformer-RNNT**, 40 language-locales com detecção automática (incluindo **pt-BR e pt-PT**), latência sub-100ms (24ms median time-to-final), chunk sizes configuráveis (80/160/320/560/1120ms), licença **NVIDIA Open Model License** (NeMo original) e **openmdw-1.1** (GGUF). Versões disponíveis: `.nemo` original, GGUF (`cstr/nemotron-3.5-asr-streaming-0.6b-GGUF` via CrispASR), MLX (Apple Silicon), e **ONNX reimplementado** descrito no paper arxiv 2604.14493 ("Pushing the Limits of On-Device Streaming ASR") — int4 quantization cabe em **~670 MB** com WER dentro de 1% absoluto do baseline PyTorch em CPU.

A pergunta que este ADR responde: **deve o my-memory implementar transcrição ASR in-process via ONNX Runtime (mesmo padrão do embedder builtin), mantendo o ADR-042 com workers Python reservados para casos que realmente exigem (LLM, TTS)?**

## Decision Drivers

- **Frugality**: alinhar com o padrão do embedder builtin (ADR-035) — ≤100 MB RAM adicional, CPU-friendly, binário self-contained.
- **Latência streaming**: tempo-to-final < 100ms é requisito implícito pra barge-in e feedback de usuário. Subprocess adiciona 5-15ms de overhead por chunk.
- **Multilingual out-of-the-box**: 40 locales incluindo pt-BR + pt-PT — atende requirement da pesquisa autoral §3.3 sobre "perfis separados para pt-BR e pt-PT" sem precisar de 2 modelos.
- **Zero dependência externa opcional**: worker Python sidecar ainda pode ser usado se usuário quiser trocar de engine; este ADR adiciona in-process como **provider primário** ou **fallback automático**.
- **Licença permissive**: NVIDIA Open Model License + openmdw-1.1 — uso comercial OK, sem copyleft.
- **Padrão arquitetural consistente**: ADR-035 já provou que ONNX Runtime CGO funciona no my-memory; repetir o pattern reduz risco de implementação.
- **Compatibilidade com event_runtime**: transcrições precisam emitir `stt.partial` e `stt.final` events (ADR-043 §4.2); in-process elimina complicação de correlação cross-process.

## Considered Options

1. **Manter ADR-042 com worker Python sidecar Whisper.cpp/Vosk/sherpa-onnx (status quo)** — descartado: overhead IPC, complexidade de lifecycle, e ignora a existência do Nemotron 3.5 que é melhor fit técnico.
2. **CrispASR subprocess (Rust via JSON-RPC)** — viável mas duplica o padrão "subprocess + IPC" que estamos tentando evitar; adiciona dependência operacional (CrispASR binary separado).
3. **NeMo Python sidecar com checkpoint `.nemo`** — preserva ADR-042 mas exige Python obrigatório pra ASR, mesmo com LLM/TTS opcionais.
4. **`nvidia/nemotron-3.5-asr-streaming-0.6b` via ONNX Runtime Go (in-process, padrão ADR-035)** — *Opção Escolhida*. Segue o pattern do embedder builtin, elimina IPC, mantém mymemoryd self-contained.
5. **Whisper.cpp via CGO in-process** — descartado: Whisper não tem streaming cache-aware nativo (precisa de buffer fixo), WER pior que Nemotron em latência sub-200ms, modelo monolíngue (precisa de 2 pra pt-BR + pt-PT).

## Pros and Cons of the Options

### Manter ADR-042 com worker Python sidecar (status quo)

- ✅ Zero código novo; segue o planejado.
- ❌ Overhead IPC Go↔Python (~5-15ms por chunk) prejudica streaming sub-100ms.
- ❌ Lifecycle Python sidecar adiciona manifestos, perfis, supervisor coordination.
- ❌ Requer Python + dependências ML no path crítico de voz.
- ❌ Ignora existência do Nemotron 3.5 (jun/2026) que é melhor fit técnico.
- ❌ Modelos multilíngues (Whisper large-v3) têm latência maior que FastConformer streaming.

### CrispASR subprocess (Rust via JSON-RPC)

- ✅ Nemotron 3.5 disponível em formato GGUF pronto pra CrispASR (openmdw-1.1, permissive).
- ✅ Streaming real (cache-aware attention preservado).
- ❌ Subprocess separado: lifecycle, manifest, supervisão.
- ❌ IPC: cada chunk de áudio (~16 kHz mono = 512 bytes a cada 16ms) atravessa boundary de processo.
- ❌ CrispASR é projeto separado (não tem releases muito maduros em 2026); adotar dependência menos testada.
- ❌ Duplica o pattern "subprocess" que ADR-035 explícita evitou pro embedder.

### NeMo Python sidecar com checkpoint `.nemo` original

- ✅ Pipeline NeMo completo (tokenização, mel spectrogram, encoder, decoder, decoder joint, beam search opcional).
- ✅ Latência sub-100ms documentada pela NVIDIA.
- ❌ Python obrigatório no path crítico de voz.
- ❌ CGO Python ou subprocess: ambas adicionam overhead ou complexidade.
- ❌ Mantém topologia pesada do ADR-042 original.

### `nvidia/nemotron-3.5-asr-streaming-0.6b` via ONNX Runtime Go (in-process) ✅ Escolhida

- ✅ In-process, zero IPC, latência mínima (apenas o compute do modelo).
- ✅ Multilingual 40 locales com detecção automática — pt-BR + pt-PT cobertos sem 2 modelos.
- ✅ Streaming cache-aware preservado via ONNX Runtime (paper 2604.14493 prova viabilidade).
- ✅ Padrão consistente com ADR-035 (embedder builtin) — mesmo `github.com/yalue/onnxruntime_go`, mesma frugalidade, mesma licença permissive.
- ✅ Elimina dependência de Python sidecar pra ASR — workers Python ficam reservados pra casos onde realmente precisar (LLM inferência via llama.cpp Python bindings, TTS Piper).
- ✅ Tamanho de download aceitável: ~670 MB int4 ou ~458 MB GGUF q4_k.
- ❌ Binário aumenta ~30-50 MB (lib onnxruntime compartilhada + modelo opcional).
- ❌ Requer portabilidade da implementação ONNX do paper 2604.14493 (ou usar GGUF + llama.cpp ASR, se compatível) — **esforço de implementação não trivial**.
- ❌ Cold start ~2-5s na primeira inferência (warm-up na inicialização do mymemoryd mitiga).
- ❌ Quality ~equal ao NeMo PyTorch (paper provou), mas benchmark local com corpus pt-BR é mandatório antes de aceitar como default.

### Whisper.cpp via CGO in-process

- ✅ Whisper.cpp é C++ maduro; bindings Go via CGO viáveis.
- ✅ Multilingual nativo (99 idiomas).
- ❌ Whisper não tem streaming cache-aware — usa buffer fixo de 30s, latência alta em primeira inferência.
- ❌ WER significativamente pior que Nemotron em latência sub-200ms (Whisper foi otimizado pra batch, não streaming).
- ❌ Para pt-BR + pt-PT monolíngue exige 2 modelos (Whisper large-v3 é o melhor multilíngue mas pesado).

## Decision Outcome

Adota-se a **Opção 4**: incluir `nvidia/nemotron-3.5-asr-streaming-0.6b` como **provider ASR único** do mymemoryd, executando **in-process via ONNX Runtime Go**, seguindo o padrão canônico do embedder builtin (ADR-035).

O worker Python sidecar ASR previsto no ADR-042 fica **explicitamente deferred** (ver `## Deferral: Plano B (Python Sidecar)` abaixo) até que uma das condições-gatilho seja atendida com evidência. Não construir `internal/asr/python_sidecar.go`, não incluir bloco `asr.python_sidecar` em `config.yaml`, não documentar como caminho de fallback em `docs/CLI_GUIDE.md` enquanto a seção estiver deferred.

### Topologia

```text
                    ┌─────────────────────────────────────────────┐
                    │      config.yaml (asr.provider)             │
                    │                                             │
                    │   "onnx-nemotron" ──► ONNXRuntimeClient    │
                    │   (vazio)        ──► default fallback       │
                    └────────────────────────┬────────────────────┘
                                             │
                                  provider selection
                                             │
                    ┌────────────────────────▼────────────────────┐
                    │              AsrSelector                     │
                    │                                              │
                    │  1. Hoje só existe 1 provider ativo.         │
                    │  2. Se o ONNX falhar ao carregar, mymemoryd │
                    │     falha fechado com hint ("baixe o modelo │
                    │     via `mem asr download` ou configure     │
                    │     --asr-model-path").                     │
                    │  3. Plano B (Python sidecar) está deferred  │
                    │     — ver seção "Deferral: Plano B".        │
                    └──────────────────────────────────────────────┘
                                    │
                                    ▼ emits events
                    ┌──────────────────────────────────────────────┐
                    │            event_runtime (ADR-043)           │
                    │                                              │
                    │  stt.partial (cada chunk decodificado)       │
                    │  stt.final   (ao detectar end-of-utterance) │
                    └──────────────────────────────────────────────┘
```

### Componentes (paralelos ao ADR-035)

#### `internal/asr/nemotron_onnx.go` (NOVO)

- Carrega `nemotron-3.5-asr-streaming-0.6b` (formato a decidir: ONNX reimplementado do paper 2604.14493 ou GGUF via llama.cpp ASR se compatível) quantizado int4 (~670 MB ou ~458 MB) do caminho configurado (default: `.memory/models/nemotron-asr-int4.onnx` ou `.gguf`).
- Streaming inference via `github.com/yalue/onnxruntime_go` (binding para ONNX Runtime C API) com cache-aware attention state preservado entre chunks.
- Sample rate: 16 kHz mono PCM (mesmo padrão de Wyoming/Wyoming-spec).
- Configurable chunk size: 80/160/320/560/1120ms (parâmetro `att_context_size` do modelo).
- Auto-detecção de idioma via `target_lang=auto` quando configurado.
- Saída: `Transcript{Text string, Language string, Confidence float64, Tokens []Token, LatencyMS int64}`.

#### `internal/asr/selector.go` (NOVO)

- Decide provider único com base em `asr.provider`. Default = `"onnx-nemotron"` (única opção ativa).
- Fallback: hoje só há um provider ativo. Se o carregamento do modelo ONNX falhar, `mymemoryd` falha fechado com hint claro ("baixe o modelo via `mem asr download` ou configure `--asr-model-path`"). Plano B Python fica deferred (ver seção abaixo).
- Loga a decisão via `Asr: <provider> falhou; usando <próximo>` (placeholder; hoje o próximo provider nunca é atingido enquanto o sidecar Python estiver deferred).

#### `internal/asr/python_sidecar.go` **(DEFERRED)**

Não criar este arquivo até a seção `## Deferral: Plano B (Python Sidecar)` ser promovida. Stub será adicionado apenas quando uma das condições-gatilho documentadas for atendida.

#### `internal/asr/model_downloader.go` (NOVO)

- `mem asr download --provider onnx-nemotron` baixa o modelo int4 do HuggingFace Hub (`nvidia/nemotron-3.5-asr-streaming-0.6b` ou versão GGUF/CrispASR via `cstr/nemotron-3.5-asr-streaming-0.6b-GGUF`) para `.memory/models/`.
- Sem internet? Usuário pode colocar o modelo manualmente via flag `--model-path`.

#### Configuração

```yaml
# .memory/config.yaml — exemplo com provider ONNX primário
asr:
  provider: "onnx-nemotron"  # único provider ativo hoje; "python-sidecar" deferred (ver abaixo)
  onnx_nemotron:
    model_path: ""            # default: .memory/models/nemotron-asr-int4.onnx
    vocab_path: ""            # default: .memory/models/nemotron-asr-vocab.txt
    target_lang: "auto"       # "auto" | "pt-BR" | "pt-PT" | "en-US" | ...
    chunk_ms: 160             # 80 | 160 | 320 | 560 | 1120
    num_threads: 0            # 0 = auto-detect
# asr.python_sidecar: deferred — ver "Deferral: Plano B (Python Sidecar)" abaixo
```

#### CLI

```bash
# Subcomandos novos
mem asr download --provider onnx-nemotron   # baixa o modelo int4 (~670 MB) ou GGUF q4_k (~458 MB)
mem asr doctor                              # mostra provider ativo, health, idioma detectado
mem asr test --audio ./sample.wav           # transcrição rápida de teste
mem asr emit --session <id>                 # emite eventos stt.partial/final pra event_runtime
```

#### MCP (proposto pra ADR-048 planejado)

- `asr_transcribe` — recebe audio base64 ou path, retorna texto + metadata.
- `asr_get_provider` — reporta provider ativo.
- `asr_set_provider` — troca provider em runtime.

### Compatibilidade com ADR-042

- **ADR-042 não muda estruturalmente.** Workers Python sidecar continuam existindo como contrato (json-rpc, manifest, supervisor) pra TTS e LLM. Pra ASR, in-process Go é o caminho único até a seção "Deferral" ser promovida. TTS Piper e LLM (Ollama HTTP / Python binding) permanecem como sidecars Python sob o ADR-042.
- **`internal/voice/asr_worker.py`** (planejado no ADR-042) vira **opcional** e atualmente não construído (`DEFERRED`). Será criado apenas quando a promoção da seção "Deferral: Plano B (Python Sidecar)" acontecer com evidência documentada.

### Integração com event_runtime (ADR-043)

- ASR client emite eventos `stt.partial` a cada chunk decodificado (~80-320ms) e `stt.final` ao detectar end-of-utterance.
- Correlation ID propagado: cada `stt.partial` carrega `correlation_id` do turno de conversa; replay de eventos via `mem replay --since <seq>` reproduz a transcrição completa.
- Schema version 1 do envelope (ADR-043 §4.1) comporta sem mudança.

### Threat Model (ADR-050)

- **LLM02 — PII**: ASR é o ponto de entrada de PII (áudio contém conversas, ditados, conteúdo sensível). Transcript bruto NUNCA é logado; o audit subscriber (T9) aplica redaction LLM02 ao campo `payload.transcript` antes de persistir.
- **LLM06 — Excessive Agency**: ASR client é puramente read-only no vault. Tool calls só vêm depois do `stt.final` ser consumido pelo agente runtime (F3).
- **LLM10 — Unbounded Consumption**: MaxAckPending por ASR subscriber enforça backpressure; chunk_size configurável permite tuning CPU/RAM.

## Consequences

### Positive

- **Latência de streaming sub-100ms** sem IPC overhead — feedback de usuário em tempo real.
- **Zero dependência externa obrigatória pra ASR** — workers Python opcionais. mymemoryd fica self-contained pra voz.
- **Multilingual out-of-the-box** — 40 locales incluindo pt-BR + pt-PT cobrem requirement da pesquisa autoral.
- **Padrão consistente** — mesmo ONNX Runtime Go, mesma frugalidade, mesma licença permissive do embedder builtin (ADR-035).
- **Event-driven natural** — `stt.partial`/`stt.final` eventos fluem direto pro event_runtime, replay nativo, audit subscriber com redaction LLM02.
- **Tamanho de download aceitável** — ~670 MB int4 ou ~458 MB GGUF q4_k cabe em SSDs modernos; lazy download opcional.
- **Compatível com cluster mode futuro** — provider é trocável via `asr.provider`; cluster pode rodar só `onnx-nemotron` em worker Python ou in-process.

### Negative

- **Esforço de implementação não trivial** — portar pipeline ONNX do paper 2604.14493 ou integrar GGUF/CrispASR via llama.cpp ASR (compatibilidade FastConformer-RNNT a verificar). Estimativa: 2-3 sprints.
- **Tamanho do binário aumenta** — `onnxruntime_go` ~30 MB; modelo int4 ~670 MB (lazy download opcional, não vai pro binário base).
- **Cold start ~2-5s** na primeira inferência — warm-up na inicialização do mymemoryd mitiga.
- **Benchmark local obrigatório** — qualidade do int4 vs fp16 vs q4_k vs NeMo PyTorch precisa ser medida em corpus pt-BR antes de aceitar como default. Não confiar só nos benchmarks do paper.
- **Manutenção de modelo** — atualizações upstream do Nemotron 3.5 (releases novos) exigem rebuild do quantizado + revalidação de regressão.
- **Comunidade menor que Whisper/Vosk** — Nemotron 3.5 é novíssimo (jun/2026); exemplos Go podem ser escassos.
- **Sem Plano B implementado até o gatilho disparar.** Se a binding `yalue/onnxruntime_go` regredir ou a quantização int4 não convergir numa plataforma suportada, my-memory fica sem ASR até o sidecar Python ser ativado pela via deferral. Mitigação: ADR-050 LLM10 fail-closed + workflow `mem asr doctor` que aponta claramente a falha + docs `docs/CLI_GUIDE.md` indicando o fallback manual até a promoção.

### Neutral

- **My-memory continua compatível com ADR-042** — Python sidecar opcional, não obrigatório.
- **ADR-035 e ADR-045 compartilham a infraestrutura** — `onnxruntime_go` já estará no go.mod, downloads já existirão em `.memory/models/`.
- **F2 (voz) continua sendo roadmap pós-Foundation** — ADR-044 (writer) + ADR-047 (agente) ainda vem antes de F2 atacar voz de fato.

## Deferral: Plano B (Python Sidecar)

A construção do sidecar Python ASR (Whisper.cpp / Vosk / sherpa-onnx) está **explicitamente diferida** sob o princípio da ADR-038 (decisões adiadas custam menos que código pago à toa). Esta cláusula é parte vinculante deste ADR — promover `internal/asr/python_sidecar.go` sem cumprir os gatilhos abaixo é uma violação arquitetural e exige ADR próprio.

### Condições-gatilho para promoção

A seção sai do estado deferred **somente** quando **uma** das condições abaixo for atendida com evidência documental anexada (issue, ADR de reversão, ou RFC datada e linkada):

1. **Incidente de produção** — a binding `github.com/yalue/onnxruntime_go` regredir ou falhar ao carregar em qualquer plataforma suportada (Linux x86_64, Linux arm64, macOS arm64, Windows amd64). Comprovação: log/issue de campo com data, versão do mymemoryd, traceback, e binário da plataforma afetada.
2. **Modelo upstream sem reimplementação ONNX** — sair uma nova versão principal do Nemotron ASR (ex: 4.0) cuja reimplementação ONNX do paper 2604.14493 não esteja disponível em ≤90 dias após release, **e** houver issue/prod requirement de A/B testing antes da comunidade portar.
3. **Restrição de cadeia de suprimentos** — cliente corporativo exigir apenas bindings ONNX Runtime **oficiais** (NVIDIA), sem dependência de binding mantida pela comunidade. Justificativa: cláusula contratual redigida + data de exigência.

**Não é gatilho válido:** "Vai que precisa" / "Por via das dúvidas" / especulação sobre plataformas sem dados reais. Cada gatilho precisa de pelo menos um caso real documentado, não inferido.

### O que fica proibido até a promoção

- ❌ Criar `internal/asr/python_sidecar.go` (qualquer arquivo com esse nome em qualquer pacote).
- ❌ Adicionar bloco `asr.python_sidecar` em `config.yaml` schemas (`internal/config/*.go`) ou no template `.memory/config.yaml`.
- ❌ Documentar `--asr-provider=python-sidecar` ou `--asr-engine=python` em `docs/CLI_GUIDE.md`, `docs/AGENT_INTEGRATION_GUIDE.md`, `README.md`, ou help text do CLI.
- ❌ Adicionar dependência Python (`onnxruntime`, `pywhispercpp`, `vosk`, `sherpa-onnx`) em `pyproject.toml`, `requirements.txt`, ou `setup.py` do my-memory.
- ❌ Promover ADR-042 a incorporar ASR Python como caminho ativo antes desta seção ser movida para "Promoted".

### Como promover (quando um gatilho dispara)

1. Abrir issue rastreável no GitHub referenciando este ADR e o gatilho que disparou.
2. Criar ADR-046 "ASR Python Sidecar Fallback" com análise do provider específico escolhido (Whisper.cpp? sherpa-onnx?), estimativa de esforço, e trade-offs de licença/tamanho.
3. Atualizar este ADR-045 movendo a subseção "Status" abaixo de **Deferred (→ ADR-046)** para **Promoted** com link cruzado.
4. Implementar a promotion via spec `tlc-spec-driven` (`spec.md` → `tasks.md` → execução em batches com quality gate).
5. Atualizar `docs/REFERENCES.md` entry 1.8 (Voz) com a nova referência.

### Status atual

- **Última revisão**: 2026-09-19
- **Gatilhos disparados**: 0
- **Evidence log**: nenhum ainda
- **Próxima revisão**: quando F2 (voz) entrar na trilha de implementação ativa (post-Foundation per Trilha D do ADR-036).

## Implementation Plan (futuro, fora do escopo deste ADR)

1. **Decisão técnica pré-implementação** — Prototipar 2 caminhos em branches separados: (a) portar ONNX do paper 2604.14493; (b) usar GGUF/CrispASR se llama.cpp suportar FastConformer-RNNT (verificar). Escolher o de menor esforço que preserve streaming.
2. `internal/asr/nemotron_onnx.go` — client ONNX + integração com `onnxruntime_go`.
3. `internal/asr/selector.go` — seletor de provider único (sem fallback ativo enquanto Plano B deferred).
4. `internal/asr/python_sidecar.go` — **(DEFERRED)** stub opcional pra compatibilidade com ADR-042. **Não construir até promoção da seção "Deferral: Plano B (Python Sidecar)"**.
5. `internal/asr/model_downloader.go` + CLI `mem asr {download, doctor, test, emit}`.
6. Integração com event_runtime: emitir `stt.partial`/`stt.final` envelopes com correlation_id.
7. Audit subscriber (já existente) com redaction LLM02 aplicada em `payload.transcript`.
8. Testes: corpus pt-BR com ground-truth (`prova-md` data), comparar WER entre ONNX int4 vs NeMo PyTorch.
9. Benchmark de latência: `time-to-final` vs `chunk_size` vs `num_threads` em CPU.
10. Atualizar ADR-042 — marcar Python sidecar ASR como deferred (não retirar dos contratos do ADR-042; apenas marcar que hoje não é instanciado pra ASR).
11. Atualizar `docs/REFERENCES.md` entry 1.8 — adicionar Nemotron 3.5 + CrispASR + paper arxiv 2604.14493.

## Alternatives Considered

- **Whisper.cpp in-process via CGO** — descartado por latência de buffer fixo; Whisper é otimizado pra batch.
- **Vosk in-process** — Vosk é C++ com Kaldi; bindings Go existem mas qualidade menor que Nemotron 3.5 em streaming.
- **sherpa-onnx in-process** — viável; sherpa-onnx tem suporte oficial a múltiplos modelos incluindo Parakeet/TDT. Mas Nemotron 3.5 é state-of-art em jun/2026; sherpa-onnx pode demorar pra incorporar.
- **Wait for upstream llama.cpp ASR support** — llama.cpp adicionou suporte parcial a ASR em 2025; maturidade ainda baixa pra FastConformer-RNNT streaming.

## References

- [`nvidia/nemotron-3.5-asr-streaming-0.6b`](https://huggingface.co/nvidia/nemotron-3.5-asr-streaming-0.6b) — modelo base, NVIDIA Open Model License
- [`cstr/nemotron-3.5-asr-streaming-0.6b-GGUF`](https://huggingface.co/cstr/nemotron-3.5-asr-streaming-0.6b-GGUF) — versão GGUF (openmdw-1.1) via CrispASR
- [`mlx-community/nemotron-3.5-asr-streaming-0.6b`](https://huggingface.co/mlx-community/nemotron-3.5-asr-streaming-0.6b) — versão MLX (Apple Silicon)
- [`build.nvidia.com/nvidia/nemotron-asr-streaming/modelcard`](https://build.nvidia.com/nvidia/nemotron-asr-streaming/modelcard) — NIM card com specs de deployment
- [arxiv 2604.14493 "Pushing the Limits of On-Device Streaming ASR" — reimplementação ONNX do Nemotron com int4 quantization](https://arxiv.org/html/2604.14493v1)
- [NVIDIA NeMo](https://github.com/NVIDIA-NeMo/Speech) — framework original com checkpoints `.nemo`
- [`yalue/onnxruntime_go`](https://github.com/yalue/onnxruntime_go) — binding Go para ONNX Runtime (mesmo usado pelo embedder builtin)
- ADR-035 — Embedder Embutido com Fallback ONNX MiniLM (mesmo padrão)
- ADR-042 — mymemoryd com Workers Sidecar Isolados (Python opcional pra ASR via fallback)
- ADR-043 — Envelope de Eventos Canônico (`stt.partial` / `stt.final` eventos)
- ADR-050 — Threat Model OWASP LLM (LLM02 redaction do transcript; LLM10 unbounded consumption)
- Pesquisa autoral `pesquisa-infraestrutura-autoral-mymemory.md` §3.3 (ASR streaming), §3.5 (TTS), §6 (escolhas tecnológicas)
- [`docs/REFERENCES.md` — entry 1.8 — Voz (Wyoming, Whisper.cpp, Vosk, sherpa-onnx)](../REFERENCES.md) — referências anteriores que continuam válidas
