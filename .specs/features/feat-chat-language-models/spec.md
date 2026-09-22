---
title: feat-chat-language-models — Schema VS Code-style para múltiplos providers/modelos
category: resource
summary: Adota o padrão VS Code `chatLanguageModels.json` para o my-memory: arquivo JSON com providers (vendor, name, models) e capabilities por model (id, url, vision, maxInputTokens, maxOutputTokens, streaming, supportsReasoningEffort, toolCalling). Substitui o hardcoded CHAT_MODELS atual.
status: draft
tags: [chat, llm, ai-sdk, ipc, electron, config]
---

# feat-chat-language-models — Schema VS Code-style

## Contexto

O chat LLM no viewer Electron do my-memory (ADR-051) tem providers hardcoded em `src/lib/chat/types.ts`:

```typescript
export const CHAT_MODELS: Record<ChatProvider, readonly string[]> = {
  anthropic: ["claude-opus-4-1", "claude-sonnet-4-5", "claude-haiku-4-5"],
  openai: ["gpt-4o", "gpt-4o-mini", "o3", "o3-mini"],
} as const;
```

E o `services/chat/llm.ts` faz `if (provider === "anthropic") ... else createOpenAI(...)` — só 2 vendors.

Em 2026-09-22, Felipe pediu para **adotar o padrão VS Code `chatLanguageModels.json`** ([docs](https://code.visualstudio.com/docs/agent-customization/language-models)) pra permitir adicionar mais providers/modelos sem recompilar.

## Objetivo

Substituir o hardcoded CHAT_MODELS por um arquivo JSON carregável, com:

- **Provider-level:** `vendor` (string), `name` (display), `models` (array)
- **Model-level:** `id`, `name`, `url`, `toolCalling`, `vision`, `maxInputTokens`, `maxOutputTokens`, `streaming`, `supportsReasoningEffort[]`, `requestHeaders{}`, `zeroDataRetentionEnabled`
- **Auto-discovery** de modelos via `/v1/models` (OpenAI-compat) e `/v1/models` (Anthropic)
- **UI:** settings modal mostra todos os providers configurados + seus models, agrupados por vendor

## Out of Scope

- Tool calling (function calling) — apenas capability flag por enquanto
- Thinking mode (extended thinking) — flag exposta, não implementada em Phase 1
- Reasoning effort picker — flag exposta, não implementada em Phase 1
- Image/vision inputs — flag exposta, Image content type não enviado em Phase 1
- Per-model API keys (mantém per-provider key, como VS Code)
- Marketplace-style community models

## Decisões arquiteturais

| Decisão | Escolha | Por quê |
|---|---|---|
| File location | `app.getPath('userData')/chatLanguageModels.json` | Editável pelo usuário, fora do bundle, não-commita |
| Bundled defaults | `resources/chatLanguageModels.default.json` (no bundle) | Primeiro boot tem Anthropic + OpenAI + Groq + Ollama + OpenRouter pré-configurados |
| API key storage | Continua em `chat-config.json` (write-only) | Não muda; `byok` por provider id |
| Schema versioning | `schemaVersion: 1` | Permite migração futura |
| Auto-discovery | OpenAI-compatible `/v1/models` + Anthropic `/v1/models` | Cobre 90% dos providers |
| Custom endpoint | `vendor: "customendpoint"` com `url` explícito | Para Llamafile, vLLM, LM Studio, etc. |
| Vercel AI SDK | Continua provider abstraction (`@ai-sdk/openai-compatible`) | Llamafile/Ollama/OpenRouter usam OpenAI-compatible |

## Bundled defaults (`resources/chatLanguageModels.default.json`)

```jsonc
{
  "schemaVersion": 1,
  "providers": [
    {
      "vendor": "anthropic",
      "name": "Anthropic Claude",
      "apiType": "messages",
      "baseUrl": "https://api.anthropic.com",
      "models": [
        { "id": "claude-opus-4-1",      "name": "Claude Opus 4.1",     "maxInputTokens": 200000, "maxOutputTokens": 32000, "vision": true, "toolCalling": true, "supportsReasoningEffort": ["low","medium","high"] },
        { "id": "claude-sonnet-4-5",   "name": "Claude Sonnet 4.5",  "maxInputTokens": 200000, "maxOutputTokens": 64000, "vision": true, "toolCalling": true, "supportsReasoningEffort": ["low","medium","high"] },
        { "id": "claude-haiku-4-5",    "name": "Claude Haiku 4.5",   "maxInputTokens": 200000, "maxOutputTokens": 64000, "vision": true, "toolCalling": true }
      ]
    },
    {
      "vendor": "openai",
      "name": "OpenAI",
      "apiType": "responses",
      "baseUrl": "https://api.openai.com/v1",
      "models": [
        { "id": "gpt-4o",       "name": "GPT-4o",       "maxInputTokens": 128000, "maxOutputTokens": 16384, "vision": true, "toolCalling": true },
        { "id": "gpt-4o-mini",  "name": "GPT-4o mini",  "maxInputTokens": 128000, "maxOutputTokens": 16384, "vision": true, "toolCalling": true },
        { "id": "o3",          "name": "o3",          "maxInputTokens": 200000, "maxOutputTokens": 100000, "toolCalling": true, "supportsReasoningEffort": ["low","medium","high"] },
        { "id": "o3-mini",     "name": "o3-mini",     "maxInputTokens": 200000, "maxOutputTokens": 100000, "toolCalling": true, "supportsReasoningEffort": ["low","medium","high"] }
      ]
    },
    {
      "vendor": "openrouter",
      "name": "OpenRouter",
      "apiType": "chat-completions",
      "baseUrl": "https://openrouter.ai/api/v1",
      "autoDiscover": true
    },
    {
      "vendor": "groq",
      "name": "Groq",
      "apiType": "chat-completions",
      "baseUrl": "https://api.groq.com/openai/v1",
      "autoDiscover": true
    },
    {
      "vendor": "ollama",
      "name": "Ollama (local)",
      "apiType": "chat-completions",
      "baseUrl": "http://localhost:11434/v1",
      "autoDiscover": true,
      "requestHeaders": {}
    }
  ]
}
```

## Acceptance Criteria (EARS)

- **AC-1 (EARS-U):** Quando o usuário abre a aba Chat pela primeira vez, vê o provider `Anthropic` com 3 models default + Ollama (local) na sidebar de settings.
- **AC-2 (EARS-U):** Quando o usuário clica num provider e depois num model, a seleção é salva e o próximo envio usa esse provider+model.
- **AC-3 (EARS-U):** Quando o usuário configura um provider sem API key e tenta enviar, vê o mesmo erro `"auth"` claro de hoje (sem mudança comportamental).
- **AC-4 (EARS-U):** Quando o usuário edita `chatLanguageModels.json` no `userData` e adiciona um provider customizado (ex.: `lmstudio`), o próximo restart do app mostra o novo provider em settings.
- **AC-5 (EARS-U):** Quando o usuário ativa um provider com `autoDiscover: true`, o app busca os models via `/v1/models` e popula a lista (substituindo o `models: []` no JSON).
- **AC-6 (EARS-S):** API keys nunca saem do main process; renderer só vê `hasApiKey: boolean` por provider.
- **AC-7 (EARS-S):** IPC `chat:providers-get` retorna providers sanitizados (sem keys, sem `apiKey` field).
- **AC-8 (EARS-S):** IPC `chat:models-discover` valida URL (no SSRF), timeout 5s, fail-soft em erros.
- **AC-9 (EARS-S):** Validação de schema no boot: provider sem `vendor` é rejeitado; model sem `id` é rejeitado.
- **AC-10 (EARS-S):** `npm run typecheck` exit 0; vitest verde (incluindo 3 testes novos: loadConfig defaults, schema validation, URL sanitization).
- **AC-11 (EARS-S):** `biome lint` não cresce além do baseline.

## Estrutura de arquivos

```
src/lib/chat/
  models-config.ts        # parser + validator + types
  types.ts                # + adicionar LanguageModelConfig, ChatProviderConfig, ChatLanguageModelsConfig

electron/services/chat/
  models-config.ts        # reader/writer do JSON em userData (espelha config.ts)
  llm.ts                  # dispatch dinâmico por vendor (substitui switch anthropic/openai)

electron/ipc/
  chat.ts                 # + IPC channels: chat:providers-get, chat:models-discover

electron/constants/
  index.ts                # + IPC_CHANNELS.MEM_CHAT_PROVIDERS_GET, MEM_CHAT_MODELS_DISCOVER

src/components/chat/
  chat-settings-modal.tsx # refatora para consumir lista dinâmica de providers

resources/
  chatLanguageModels.default.json   # bundled defaults (commitado)
```

## Critérios de quality gate

- gofmt clean, go build exit 0 (sem mudanças Go)
- npm run typecheck exit 0
- npm test (vitest) PASS — incluindo 3 testes novos
- electron-forge package exit 0
- biome lint não cresce além do baseline
- ADR-052 (Proposed → Accepted) documenta a decisão

## Plano de execução (Phase 1 close)

19 → ~12 tasks: T1 (spec já feito) → T2 (schema + bundled defaults) → T3-T5 (services) → T6 (IPC) → T7-T8 (UI) → T9 (discover) → T10-T12 (docs/PR).
