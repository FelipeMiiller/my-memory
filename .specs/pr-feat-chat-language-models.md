# feat(viewer-chat): VS Code-style chatLanguageModels.json (ADR-052)

Adota o padrao VS Code `chatLanguageModels.json` no chat LLM do viewer Electron. Substitui o hardcoded `CHAT_MODELS` por um arquivo JSON-driven, editavel pelo usuario, com descoberta automatica de modelos.

Ref: https://code.visualstudio.com/docs/agent-customization/language-models

## Schema

```typescript
type ApiType = 'chat-completions' | 'responses' | 'messages';
type ReasoningEffort = 'low' | 'medium' | 'high';

interface LanguageModelConfig {
  id: string;
  name: string;
  url?: string;
  apiType?: ApiType;
  toolCalling?: boolean;
  vision?: boolean;
  thinking?: boolean;
  streaming?: boolean;
  maxInputTokens?: number;
  maxOutputTokens?: number;
  supportsReasoningEffort?: ReasoningEffort[];
  zeroDataRetentionEnabled?: boolean;
  requestHeaders?: Record<string, string>;
}

interface ChatProviderConfig {
  vendor: string;
  name: string;
  apiType?: ApiType;
  baseUrl?: string;
  models: LanguageModelConfig[];
  autoDiscover?: boolean;
}
```

## Bundled defaults

- **Anthropic** (messages API) - Opus 4.1, Sonnet 4.5, Haiku 4.5
- **OpenAI** (responses API) - GPT-4o, GPT-4o mini, o3, o3-mini
- **OpenRouter** (chat-completions, autoDiscover)
- **Groq** (chat-completions, autoDiscover)
- **Ollama local** (chat-completions, autoDiscover)

## Mudancas

### Main process

- `electron/services/chat/models-config.ts` (NEW) - load/save + bundled fallback
- `electron/services/chat/models-discover.ts` (NEW) - GET /v1/models (Anthropic + OpenAI-compat) com SSRF guard, 5s timeout
- `electron/services/chat/llm.ts` - dispatch dinamico por vendor (anthropic | openai | openai-compatible via `@ai-sdk/openai-compatible`)
- `electron/services/chat/config.ts` - schema 2 com `apiKeys: Record<vendor, key>` + migration do schema 1
- `electron/ipc/chat.ts` - novos channels `mem:chat:providers:get` e `mem:chat:models:discover`
- `electron/preload.ts` - expoe `memAPI.chat.getProviders` + `discoverModels`
- `electron/types.d.ts` - declaracoes ambient dos novos tipos
- `electron/constants/index.ts` - novos `IPC_CHANNELS.MEM_CHAT_PROVIDERS_GET` + `MEM_CHAT_MODELS_DISCOVER`

### Renderer

- `src/lib/chat/types.ts` - re-exports renderer-side
- `src/lib/chat/models-config.ts` (NEW) - parser + validator de schema
- `src/lib/chat/ipc.ts` - wrappers type-safe
- `src/lib/chat/store.ts` - tracka `activeVendor` + `activeModelId` + lista de providers
- `src/components/chat/chat-settings-modal.tsx` - reescrita: lista todos os providers, capabilities por model, botao Discover
- `src/components/chat/chat-view.tsx` - send usa `{ vendor, modelId }` da config ativa

### Bundled resource

- `resources/chatLanguageModels.default.json` (NEW) - schemaVersion 1 com 5 providers

### Dependencies

- `+ @ai-sdk/openai-compatible` (Groq, OpenRouter, Ollama, custom)

### Spec

- `.specs/features/feat-chat-language-models/spec.md` (NEW)

## Quality Gate

- gofmt -l . clean
- go build ./... exit 0
- tsc --noEmit exit 0
- vitest 1/1 PASS
- electron-forge package exit 0

## Seguranca

- SSRF guard em `models-discover.ts` recusa IPs privados/loopback/link-local (excecao: localhost/127.0.0.1 pra Ollama)
- API keys nunca cruzam IPC - renderer ve so `hasApiKey: boolean` por vendor
- Schema validado no boot
- Sanitizacao de error messages (regex para redact sk-ant-..., sk-..., ghp_...)

## Como adicionar custom endpoint

Editar `<userData>/chatLanguageModels.json`:

```json
{
  "vendor": "lmstudio",
  "name": "LM Studio",
  "apiType": "chat-completions",
  "baseUrl": "http://localhost:1234/v1",
  "models": [],
  "autoDiscover": true
}
```
