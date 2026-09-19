# ADR-042: Núcleo Local `mymemoryd` com Workers Sidecar Isolados

- **Date**: 2026-09-19
- **Status**: Proposed
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: runtime, supervision, architecture, go, python, sidecar, sandbox, mymemoryd

## Context and Problem Statement

O MyMemory hoje opera como **CLI pontual** (`mem index`, `mem search`, `mem graph`) e como **servidor MCP sob demanda** (ADR-021). Cada invocação abre SQLite, lê do disco, escreve e fecha. Não há processo de longa duração, não há canal de eventos, não há workers residentes.

A pesquisa de 2026-09-18 (documento autoral) demanda uma camada nova: **voz + agente runtime + viewer/editor contínuo + replay de eventos + projeções assíncronas**. Esses componentes precisam de:

1. **Loop de longa duração** — capturar áudio contínuo, manter sessão conversacional, segurar conexões WebSocket do viewer.
2. **Workers residentes** — ASR streaming, TTS incremental, embedder embutido (ADR-035), drift watcher (ADR-031).
3. **Barramento de eventos** — correlação entre voz, agente, writer e projeções; suporte a replay e auditoria.
4. **Sandbox explícito** — processar áudio, transcrições, saídas de LLM e tool results sem que conteúdo hostil vire memória canônica (OWASP LLM Top 10 [P0]).
5. **Recovery pós-crash** — se `mymemoryd` cair durante uma fala ou uma tool call, retomar a sessão sem perder contexto.

O **ADR-002** estabelece Go como linguagem principal e o **ADR-001** estabelece SQLite como storage unificado. Mas nenhum dos dois formaliza **onde Python entra** (necessário pra ASR/TTS/ML) nem **como supervisionar processos**. O documento de pesquisa cita 12 componentes próprios, mas não está escrito qual é a topologia de runtime.

Sem essa decisão, qualquer trabalho em voz/agente vai reinventar topologia em código, espalhando Python pelo núcleo Go.

## Decision Drivers

- **DR-1**: CLI continua respondendo em <15ms (ADR-002) — processos longos não podem bloquear `mem index`.
- **DR-2**: Workers Python (ASR/TTS/ML) são necessários, mas Python **não pode** virar dependência do core domain.
- **DR-3**: Markdown permanece source-of-truth (ADR-005) — nenhum sidecar pode escrever direto no SQLite bypassando o writer.
- **DR-4**: Crash-safety — supervisor reinicia workers automaticamente; estado durável em SQLite WAL.
- **DR-5**: Egress deny-by-default (OWASP LLM Top 10 LLM05/LLM06) — workers sem rede por padrão.
- **DR-6**: Replay nativo — todas as ações autorizadas devem ser eventos reproduzíveis (audit + debugging).
- **DR-7**: Compatibilidade retroativa — `mem <subcomando>` CLI deve continuar funcionando sem `mymemoryd` rodando (modo standalone).

## Considered Options

- **Opção A: `mymemoryd` em Go como núcleo + workers Python sidecar via stdin/stdout JSON-RPC**
- **Opção B: Monolito Go único (ML in-process via CGO/ONNX)**
- **Opção C: Monolito Python (FastAPI + workers)**
- **Opção D: Adotar framework existente (Ollama, LangGraph Server, etc.) como supervisor**

## Decision Outcome

Chosen option: **"Opção A: `mymemoryd` em Go + workers Python sidecar"**, porque (1) preserva ADR-002 (cold-start Go), (2) isola o ecossistema ML sem contaminar o core domain, (3) dá topologia clara de supervisor/workers com sandbox por processo, (4) mantém o CLI standalone intacto, (5) encaixa no envelope de eventos do ADR-043.

### Topologia

```text
                  ┌──────────────────────────┐
                  │  mymemoryd (Go core)     │
                  │  - event_runtime         │
                  │  - session-manager       │
                  │  - policy-engine         │
                  │  - memory-writer         │
                  │  - projection-workers    │
                  │  - mcp-server (stdio)    │
                  └────────┬─────────────────┘
                           │ JSON-RPC via stdio / unix socket
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
  ┌───────────┐      ┌───────────┐      ┌───────────┐
  │ asr-worker│      │ tts-worker│      │ embedder  │
  │ (Python)  │      │ (Python)  │      │ (ONNX/CGO)│
  └───────────┘      └───────────┘      └───────────┘
```

### Componentes do núcleo (Go)

| Componente | Responsabilidade |
|---|---|
| `event-runtime` | Envelope canônico, sequência, correlação, ACK, retry, outbox, replay (ADR-043). |
| `session-manager` | Sessão, conversa, epoch, turnos, cancelamento, barge-in. |
| `policy-engine` | Escopos, permissões, risco, confirmação, egress. |
| `memory-writer` | Markdown atômico + precondition + outbox (ADR-044). |
| `projection-workers` | SQLite/FTS5/grafo/vetor/stats/drift — idempotentes. |
| `mcp-server` | Resources e tools MCP stdio (default); HTTP só F5. |
| `supervisor` | Health checks, manifestos, hashes, quotas, restart. |

### Workers sidecar (Python ou ONNX/CGO)

| Worker | Linguagem | Protocolo | Função |
|---|---|---|---|
| `asr-worker` | Python (Whisper/Vosk/sherpa) | JSON-RPC + binário PCM | Transcrição streaming parcial/final. |
| `tts-worker` | Python (Piper/Kokoro) | JSON-RPC + chunks PCM | Síntese incremental. |
| `embedder` | CGO/ONNX (ADR-035) | in-process | Embeddings MiniLM-L6 INT8. |
| `drift-watcher` | Go in-process | n/a | ADR-031 (já é Go). |

### Negative Consequences

- **Mais processos**: `mymemoryd` + N workers. Mitigação: supervisor centraliza lifecycle; profiles definem workers obrigatórios vs opcionais.
- **Overhead de IPC**: JSON-RPC via stdio/socket. Mitigação: framing binário p/ áudio (Wyoming-inspired), JSON apenas p/ controle.
- **Complexidade operacional**: usuário precisa entender supervisor. Mitigação: `mem up` / `mem down` envolvem o supervisor; modo standalone (CLI sem `mymemoryd`) continua válido.
- **Fase 0 (F0) é fundação, não diferencial**: trabalho inicial não entrega feature visível. Mitigação: já entregado `mymemoryd --help`, `mem up`, `mem status` no fim da F1.

## Pros and Cons of the Options

### Opção A: `mymemoryd` Go + Python sidecar ✅ Chosen

- ✅ Mantém ADR-002 (cold-start Go) e CLI standalone.
- ✅ Sandbox por processo (Python não compartilha memória do core).
- ✅ Topologia clara: core/edge separa contrato de implementação.
- ✅ Workers trocáveis (Whisper ↔ Vosk ↔ sherpa-onnx) sem recompilar core.
- ❌ Mais processos → precisa supervisor decente.
- ❌ IPC adiciona latência (mitigável com framing binário).

### Opção B: Monolito Go único (ONNX in-process)

- ✅ Menos processos.
- ✅ IPC zero.
- ❌ CGO/ONNX dentro do core → bleed de licenças (alguns modelos ONNX são CC BY-NC-SA).
- ❌ Workers Python (ASR/TTS) continuam precisando de sidecar → híbrido pior que A.
- ❌ Quebra ADR-002 indiretamente (cold-start fica mais lento com ONNX carregado).

### Opção C: Monolito Python

- ❌ Viola ADR-002 (cold-start lento, dependência de venv).
- ❌ Não aproveita concorrência nativa do Go.
- ❌ Mantém Go só como CLI thin, desperdiçando o core domain.

### Opção D: Adotar framework externo (Ollama/LangGraph/etc.)

- ❌ Introduz dependência autoral de terceiro no core.
- ❌ Documento de pesquisa já descarta explicitamente Ollama como obrigatório: "É um serviço/orquestrador próprio; não deve virar dependência obrigatória do núcleo."
- ❌ Dificulta replay/outbox (frameworks externos têm seus próprios modelos de evento).

## Implementation Notes (Fase 1 do roadmap)

1. `cmd/mymemoryd/main.go` — entrypoint, lê manifesto de workers, sobe supervisor.
2. `internal/supervisor/` — gerencia processos (start/stop/health/restart).
3. `internal/event_runtime/` — barramento in-process (ADR-043).
4. `internal/ipc/jsonrpc/` — framing JSON-RPC sobre stdio/socket.
5. Perfis de workers: `.memory/profiles/{default,voice,headless}.yaml`.
6. CLI `mem up`, `mem down`, `mem status`, `mem logs <worker>`.
7. Modo standalone: `mem <subcomando>` continua funcionando sem `mymemoryd`.

## Links

- **ADR-001**: SQLite como Camada Unificada — storage do core e do event log.
- **ADR-002**: Go como Linguagem Principal — núcleo e CLI.
- **ADR-018**: Compile-not-Retrieve — escrita atômica via MCP/CLI (refinado por ADR-044).
- **ADR-021**: Servidor MCP HTTP/SSE — base do `mcp-server` (transporte default = stdio).
- **ADR-035**: Embedder Embutido — primeiro worker CGO/ONNX.
- **ADR-043**: Envelope de Eventos Canônico — barramento interno.
- **ADR-044**: Memory Writer Atômico + Outbox — escrita durável.
- **ADR-050**: Threat Model OWASP LLM — egress sandbox e policy.
- **Documento de pesquisa** `pesquisa-infraestrutura-autoral-mymemory.md` (2026-09-18).
- Referências externas (documento §3.1, §3.7): [Wyoming](https://github.com/OHF-Voice/wyoming), [Home Assistant voice pipelines](https://developers.home-assistant.io/docs/voice/pipelines/), [MCP Specification](https://modelcontextprotocol.io/specification/2026-07-28).
