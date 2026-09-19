# ADR-050: Threat Model — OWASP LLM Top 10 Aplicado ao MyMemory

- **Date**: 2026-09-19
- **Status**: Proposed (transversal — âncora para ADRs de voz/agente/MCP/embedder)
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: security, threat-model, owasp, llm, sandbox, egress, policy, red-team

## Context and Problem Statement

A pesquisa autoral de 2026-09-18 (`pesquisa-infraestrutura-autoral-mymemory.md`, §5 e §8) classifica o **threat model OWASP LLM Top 10** como **Prioridade P0**, ao lado de MCP, Whisper.cpp, Wyoming, Assist e do próprio Obsidian Vault API. A justificativa é simples: a partir do momento em que MyMemory ganhar voz, agente runtime, MCP server autoral e workers Python sidecar (ADR-042), a superfície de ataque deixa de ser "arquivo Markdown local" e passa a incluir áudio hostil, transcrições parciais, saídas de LLM, tool results, embeddings externos e plugins de terceiros.

Os ADRs foundation já entregues — **042** (mymemoryd com workers sidecar), **043** (envelope de eventos canônico), **044** (writer atômico + outbox) — cada um já cita controles de segurança específicos (egress deny-by-default, auditoria via event log, sandbox por processo). Mas esses controles estão **espalhados** e referenciam um ADR-050 que ainda não existia. Sem um threat model formal e transversal:

1. **F2 (voz)** vai reinventar controles de áudio/injeção de transcrição a cada worker Python.
2. **F3 (agente)** vai decidir policy/admission caso a caso em vez de aplicar um framework.
3. **F4 (MCP server autoral)** vai precisar decidir scopes, allowlist e auditoria sem âncora.
4. **Red-teaming** (garak, §5 do documento) não tem alvo concreto.

Este ADR **mapeia o OWASP LLM Top 10 (versão 2026)** para controles concretos no MyMemory, define uma **postura de segurança por padrão** (deny-by-default, fail-closed, audit-tudo) e estabelece **threat vectors específicos** das camadas voz/agente/MCP/viewer/embedder.

## Decision Drivers

- **DR-1**: Threat model é **transversal**, não cabe em F5 nem num ADR-045 de visão. Cada ADR novo (voz, agente, MCP, embedder, viewer contínuo) deve referenciar este.
- **DR-2**: Postura **deny-by-default, fail-closed**. Tudo que não é explicitamente permitido é negado; toda falha aborta o efeito em vez de degradar.
- **DR-3**: **Audit-tudo** — toda ação autorizada (tool call, write, projection, session event) emite um `memory.audit.*` com `actor`, `provenance`, `correlation_id`, `causation_id`, `redacted_payload_hash`.
- **DR-4**: **Redaction-first** — logs nunca contêm áudio bruto, transcrições completas, tokens, segredos, prompts inteiros ou conteúdo do usuário acima do necessário pra forense.
- **DR-5**: **Sandbox por processo** — workers Python (ASR/TTS) não compartilham memória com o core; IPC via JSON-RPC + framing binário sobre stdio/Unix socket; egress deny-by-default no nível do processo.
- **DR-6**: **Provenance antes de effect** — nenhuma transcrição parcial, tool result, embedding ou aresta de grafo vira memória canônica sem passar pelo writer (ADR-044) com outbox.
- **DR-7**: **Testes adversariais como gate** — quality gate inclui probes gerados por garak/red-team contra os vetores LLM01-LLM10 antes de promote `Proposed → Accepted`.

## Considered Options

- **Opção A: ADR-050 transversal agora, com mapeamento LLM01-LLM10 → controles concretos + tabela de threat vectors por camada**
- **Opção B: Embutir controles em cada ADR (042/043/044 + futuros)**
- **Opção C: Defer pra F5 (hardening), aceitar risco de voice/agent/MCP entrarem sem âncora**
- **Opção D: Adotar framework externo (e.g. NVIDIA garak runtime, OWASP ASVS) como dependência**

## Decision Outcome

Chosen option: **"Opção A: ADR-050 transversal agora"**, porque (1) controla a dívida antes que ela se espalhe, (2) ancora F2/F3/F4 com controles reusáveis, (3) cria alvo concreto pra red-team, (4) o documento de pesquisa (§5) já define 90% do conteúdo — falta só formatar como decisão.

### Mapeamento OWASP LLM Top 10 → Controles no MyMemory

| OWASP | Vetor | Controle no MyMemory | ADR responsável |
|---|---|---|---|
| **LLM01 — Prompt Injection** | System prompt, tool descriptions, conteúdo importado | `policy-engine` valida tools/payloads ANTES de chegar ao modelo; nunca montar prompt do usuário com conteúdo não-confiável; separar system prompt de dados | ADR-042 + ADR-047 (planejado) |
| **LLM02 — Sensitive Information Disclosure** | Logs, telemetry, transcripts | **Redaction-first**: logs nunca contêm áudio bruto, transcrições completas, tokens, prompts; `redacted_payload_hash` em audit events; retenção configurável | este ADR + ADR-043 |
| **LLM03 — Supply Chain** | Modelos, vozes, datasets, plugins | **SBOM + hash** de modelos (Whisper/Vosk/Piper/Kokoro), vozes, datasets, dependências Go/Python; assinatura digital opcional; auditoria periódica via `mem security audit` | ADR-042 + este ADR |
| **LLM04 — Data and Model Poisoning** | Markdown importado, embeddings externos | Markdown importado entra em **quarantine** (`memory.quarantined`) até validação; embeddings só de fontes com `provenance.trust_level` ≥ medium; `mem doctor` detecta drift semântico suspeito | ADR-044 + ADR-031 (semantic drift) |
| **LLM05 — Improper Output Handling** | Tool calls, file writes, shell exec | **Structured output + validação em Go** (nunca confiar no JSON do modelo); `additionalProperties: false` em schemas; `tool-gateway` valida paths/URIs/size/destino antes de cada exec | ADR-042 + ADR-047 |
| **LLM06 — Excessive Agency** | Tools invocadas pelo agente | **Allowlist por tarefa** + scopes de curta duração + **confirmação obrigatória** para mutações sensíveis (write, exec, network); policy engine fora do LLM | ADR-042 (policy-engine) |
| **LLM07 — System Prompt Leakage** | Debug, logs, error messages | System prompt nunca vai pra log de usuário; mensagens de erro sanitizadas; `mem trace` mostra apenas `provenance` (tool_call_id, run_id), não prompts | este ADR |
| **LLM08 — Vector and Embedding Weaknesses** | Inserção de texto adversarial via vault | **ACL antes de FTS/vetor/grafo** (não buscar conteúdo que o usuário não tem acesso); embeddings validados por `content_hash`; cache incremental (ADR-010) detecta mudanças suspeitas | ADR-001 + ADR-010 |
| **LLM09 — Misinformation** | Saída do modelo usada como fato | **Citations obrigatórias** em respostas do agente; tool results citam `source_document`; nada vira memória canônica sem passar pelo writer | ADR-044 + ADR-047 |
| **LLM10 — Unbounded Consumption** | Loops infinitos, requests excessivos, DoS | **Loop limitado** no agent-runtime (max steps, max tokens, max wall-clock); rate-limit por session_id; `MaxAckPending` no event_runtime | ADR-042 + ADR-043 |

### Threat Vectors por Camada

| Camada | Vetor específico | Mitigação primária |
|---|---|---|
| **Voz — captura** | Áudio adversarial (ultrasons, ruído malicioso, dobra de wake word) | VAD + wake word com **double-check** (openWakeWord + segundo verificador leve); rejeitar chunks fora do perfil acústico esperado |
| **Voz — ASR** | Injeção de transcrição via ruído (homoglyphs em PT-BR/PT-PT, comandos DTMF-like) | Worker Python não confia em hipótese parcial; só `stt.final` chega ao core; **transcrição final** carrega confidence, timestamps, backend, modelo, hash |
| **Voz — TTS** | Sintese de áudio malicioso para confirmação do usuário | TTS é só output; confirmação de ações críticas vem por canal visual (viewer) ou push notification, **nunca** por TTS |
| **Agente — prompt** | System prompt injection via conteúdo importado | `policy-engine` separa system prompt de dados; tool inputs validados contra schema; `policy.strip_untrusted_data` antes de montar contexto |
| **Agente — tools** | Tool chamada com argumentos adversariais | `tool-gateway` valida args com JSON Schema + `additionalProperties: false`; sandbox de shell/filesystem/network; egress deny-by-default |
| **Agente — memoria** | Conteúdo importado virando memória canônica | `memory-writer` exige `provenance` + `actor`; `quarantine` para novos sources; rollback disponível |
| **MCP — transporte** | Origin spoofing, token passthrough | stdio-first (default); HTTP só F5 com `Origin` allowlist + auth + audience binding; **nenhum token passthrough** |
| **MCP — tools** | Path traversal, size overflow, namespace collision | `SafeResolvePath` (ADR-018); paginação obrigatória; namespaces determinísticos (`memory.*`, `session.*`, `agent.*`, `approval.*`) |
| **Embedder** | Modelo malicioso ou poisoned weights | SBOM + hash do modelo; fallback FTS5 quando embedder indisponível; `provenance.trust_level` por modelo |
| **Viewer** | Markdown hostil (imagens remotas, iframes, scripts) | **CSP strict** (`default-src 'none'; img-src 'self' data:`); Markdown sanitizado; sem `<script>`, sem `<iframe>`, sem `<img src="http...">`; **nenhuma imagem remota** |

### Controles obrigatórios transversais

```text
1. policy-engine fora do LLM              (ADR-042)
2. capabilities + scopes curta duração   (ADR-042)
3. allowlist de tools por tarefa          (ADR-042)
4. schemas rigorosos + additionalProperties:false (ADR-042, ADR-047)
5. validação de URL/path/size/destino     (ADR-018 SafeResolvePath)
6. sandbox shell/filesystem/network       (ADR-042 supervisor)
7. egress deny-by-default em workers     (ADR-042)
8. confirmação para mutações sensíveis   (ADR-042 approval)
9. quarantine + rollback para memórias   (ADR-044)
10. ACL antes de FTS/vetor/grafo          (ADR-001)
11. CSP strict + Markdown sanitizado      (ADR-049 planejado viewer)
12. sem imagens remotas/iframes/scripts   (ADR-049)
13. logs sem áudio/tokens/segredos        (este ADR)
14. redaction + retenção configurável    (este ADR)
15. SBOM + hash de modelos/vozes/datasets (este ADR)
16. red-team com garak em staging         (este ADR)
```

### Negative Consequences

- **Mais ceremony** — todo ADR novo de voz/agente/MCP precisa referenciar este. Mitigação: linkagem é 1 linha por ADR.
- **Testes adversariais adicionam tempo** — `garak` probes viram parte do gate. Mitigação: probes são JSONL, podem rodar em CI nightly; só bloqueia promote, não bloqueia commit.
- **Restrições podem frustrar usuário** — "não posso baixar modelo X sem SBOM" pode parecer pedante. Mitigação: profiles de segurança (`strict`, `balanced`, `permissive-dev`) permitem ajustar por ambiente.
- **F5 (hardening) herda decisões que ainda não foram validadas em produção**. Mitigação: ADR-050 é `Proposed`; só vira `Accepted` após red-team com pelo menos 1 vetor LLM01-LLM10 explorado e mitigado.

## Pros and Cons of the Options

### Opção A: ADR-050 transversal agora ✅ Chosen

- ✅ Ancora F2/F3/F4 com controles reusáveis.
- ✅ Red-team (garak) tem alvo concreto.
- ✅ Documento de pesquisa §5 já tem 90% do conteúdo — falta formatar.
- ✅ Evita controles inconsistentes entre ADRs.
- ❌ Mais um ADR pra linkar em tudo (mitigável).

### Opção B: Embutir controles em cada ADR

- ✅ Menos ADRs.
- ❌ Inconsistência: voz trata injeção de transcrição diferente do agente trata injeção de prompt.
- ❌ Red-team precisa entender N controles diferentes.
- ❌ Dívida técnica cresce exponencialmente.

### Opção C: Defer pra F5

- ✅ Velocidade inicial.
- ❌ Voice/agent/MCP (F2/F3/F4) entram **sem âncora de segurança** — exatamente quando mais precisam.
- ❌ F5 vira "reescrever tudo com controles" — caro e arriscado.

### Opção D: Framework externo (garak runtime, OWASP ASVS)

- ✅ Madurez de terceiros.
- ❌ Dependência autoral de terceiro no core (viola ADR-042).
- ❌ Garak é scanner externo — não deve ser incorporado como runtime (própria recomendação do projeto garak).
- ❌ OWASP ASVS é voltado a apps web tradicionais; LLM Top 10 é separado.

## Implementation Notes

1. **F1** — adicionar coluna `provenance.trust_level` no envelope de eventos (ADR-043) — small refactor.
2. **F2** — quando entrar voz, primeiro teste adversarial é probe de injeção de transcrição (garak `promptinject` adapted).
3. **F3** — quando entrar agente, `policy-engine` ganha `policy/strict.yaml`, `policy/balanced.yaml`, `policy/permissive-dev.yaml`.
4. **F4** — MCP server autoral carrega allowlist por tool no startup; confirmação obrigatória para mutações.
5. **F5** — `mem security audit` (subcomando novo) roda SBOM + hash check + probes básicos; CI nightly roda garak contra staging.
6. **Quality gate** — testes adversariais viram parte do `always-quality-gate.md`: `make security-probe` (F5+).

## Links

- **ADR-001**: SQLite como Camada Unificada — ACL antes de FTS/vetor/grafo.
- **ADR-002**: Go como Linguagem Principal — core em Go, sandbox por processo.
- **ADR-010**: Cache Incremental com SHA-256 — `content_hash` detecta mudanças suspeitas.
- **ADR-018**: Compile-not-Retrieve — `SafeResolvePath` (exemplo LLM05/LLM06).
- **ADR-031**: Semantic Drift — detecção de desvio código-memória (apoia LLM04).
- **ADR-042**: `mymemoryd` com Workers Sidecar — sandbox, egress, policy-engine, approval.
- **ADR-043**: Envelope de Eventos Canônico — auditoria via event log (LLM02).
- **ADR-044**: Memory Writer Atômico + Outbox — quarantine, provenance, rollback (LLM04/LLM09).
- **ADR-049**: Viewer/Canvas Autoral — planejado, CSP strict, sem imagens remotas (LLM05).
- **Documento de pesquisa** `pesquisa-infraestrutura-autoral-mymemory.md` §5 (Security) e §8 (Prioridade P0).
- Referências externas: [OWASP LLM Top 10 2026](https://genai.owasp.org/llm-top-10/), [NVIDIA garak](https://github.com/NVIDIA/garak) (scanner externo), [Wyoming (sem auth por design — não usar em rede aberta)](https://github.com/OHF-Voice/wyoming).
