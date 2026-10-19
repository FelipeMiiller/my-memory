import { describeUnknownError } from "./errors"
import type { HealthPayload, MiniSchema } from "./schemas"
import { healthSchema } from "./schemas"

/**
 * Cliente MCP da viewer.
 *
 * O contrato é o do transporte HTTP do MyMemory:
 *   - `POST {base}/mcp` — JSON-RPC 2.0, método `tools/call`
 *   - `GET  {base}/health` — readiness do servidor
 *
 * A base vem de `VITE_MCP_BASE_URL`. Vazio (o default) significa **mesma
 * origem**: em produção a SPA é servida pelo próprio MCP (ADR-059 D-11), e em
 * desenvolvimento quem encaminha é o proxy do Vite. A porta nunca é hardcoded
 * no código da aplicação.
 */

/** Erro de transporte ou de contrato do MCP. */
export class McpError extends Error {
  readonly code: number
  readonly data: unknown

  constructor(message: string, code = -32603, data: unknown = undefined) {
    super(message)
    this.name = "McpError"
    this.code = code
    this.data = data
  }
}

/** Resultado de uma chamada de tool. */
export interface McpToolResult<T> {
  /** Texto legível devolvido pela tool (`content[].text`). Sempre presente. */
  text: string
  /** Payload estruturado validado. `null` quando ausente ou inválido. */
  structured: T | null
  /** `true` somente quando havia `structuredContent` e ele validou. */
  hasStructured: boolean
  /** Mensagem do erro de validação, quando houve. */
  validationError: string | null
}

interface JsonRpcError {
  code?: number
  message?: string
  data?: unknown
}

interface JsonRpcResponse {
  jsonrpc?: string
  id?: unknown
  result?: {
    content?: unknown
    isError?: boolean
    structuredContent?: unknown
  }
  error?: JsonRpcError
}

let requestCounter = 0

function nextRequestId(): number {
  requestCounter += 1
  return requestCounter
}

/** Base URL do MCP, sem barra final. Vazio = mesma origem. */
export function mcpBaseUrl(): string {
  const raw = import.meta.env.VITE_MCP_BASE_URL
  const value = typeof raw === "string" ? raw.trim() : ""
  return value.replace(/\/+$/, "")
}

/** Monta a URL absoluta de um endpoint do MCP. */
export function mcpUrl(path: string): string {
  return `${mcpBaseUrl()}${path}`
}

function describeError(cause: unknown): string {
  return describeUnknownError(cause)
}

/** Extrai o texto de um bloco de conteúdo MCP. */
function extractText(content: unknown): string {
  if (!Array.isArray(content)) return ""
  const parts: string[] = []
  for (const block of content) {
    if (typeof block === "object" && block !== null) {
      const text = (block as { text?: unknown }).text
      if (typeof text === "string") parts.push(text)
    }
  }
  return parts.join("\n")
}

/** Converte o erro do zod em uma mensagem legível. */
export function describeValidationError(error: unknown): string {
  if (typeof error === "object" && error !== null) {
    const message = (error as { message?: unknown }).message
    if (typeof message === "string" && message.length > 0) return message
    const issues = (error as { issues?: unknown }).issues
    if (Array.isArray(issues) && issues.length > 0) {
      return issues
        .map((issue) => {
          if (typeof issue !== "object" || issue === null) return String(issue)
          const path = (issue as { path?: unknown }).path
          const text = (issue as { message?: unknown }).message
          const where = Array.isArray(path) ? path.join(".") : ""
          return where ? `${where}: ${String(text)}` : String(text)
        })
        .join("; ")
    }
  }
  return String(error)
}

/**
 * Chama uma tool do MCP.
 *
 * Robustness contract: a viewer precisa funcionar com um MCP **antigo** (sem
 * `structuredContent`) e com o novo. Por isso:
 *   - sem `structuredContent` → devolve só o texto, sem erro;
 *   - com `structuredContent` inválido → devolve o texto cru, registra o erro
 *     no console e **não** lança (AC-36 da spec).
 */
export async function callTool<T>(
  name: string,
  args: Record<string, unknown>,
  schema: MiniSchema<T>
): Promise<McpToolResult<T>> {
  const url = mcpUrl("/mcp")
  const payload = {
    jsonrpc: "2.0",
    id: nextRequestId(),
    method: "tools/call",
    params: { name, arguments: args },
  }

  let response: Response
  try {
    response = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    })
  } catch (cause) {
    throw new McpError(
      `Não foi possível falar com o MCP em ${url}: ${describeError(cause)}`
    )
  }

  if (response.status === 204) {
    throw new McpError(`O MCP não devolveu resposta para a tool "${name}".`)
  }
  if (!response.ok) {
    throw new McpError(
      `O MCP respondeu ${response.status} ${response.statusText} para a tool "${name}".`
    )
  }

  let body: JsonRpcResponse
  try {
    body = (await response.json()) as JsonRpcResponse
  } catch (cause) {
    throw new McpError(
      `Resposta do MCP não é JSON válido para a tool "${name}": ${describeError(cause)}`
    )
  }

  if (body.error) {
    throw new McpError(
      body.error.message ?? `Erro JSON-RPC na tool "${name}".`,
      body.error.code ?? -32603,
      body.error.data
    )
  }

  const result = body.result
  if (!result) {
    throw new McpError(`Resposta MCP sem campo "result" para a tool "${name}".`)
  }

  const text = extractText(result.content)
  if (result.isError) {
    throw new McpError(text || `A tool "${name}" retornou erro.`)
  }

  const raw = result.structuredContent
  if (raw === undefined || raw === null) {
    return {
      text,
      structured: null,
      hasStructured: false,
      validationError: null,
    }
  }

  const parsed = schema.safeParse(raw)
  if (parsed.success) {
    return {
      text,
      structured: parsed.data,
      hasStructured: true,
      validationError: null,
    }
  }

  const message = describeValidationError(parsed.error)
  console.warn(
    `[viewer] structuredContent inválido em "${name}": ${message}. Exibindo o texto cru.`
  )
  return {
    text,
    structured: null,
    hasStructured: false,
    validationError: message,
  }
}

/** Lê o payload de readiness do MCP (`GET {base}/health`). */
export async function getHealth(): Promise<HealthPayload> {
  const url = mcpUrl("/health")
  let response: Response
  try {
    response = await fetch(url, { headers: { Accept: "application/json" } })
  } catch (cause) {
    throw new McpError(
      `Não foi possível falar com o MCP em ${url}: ${describeError(cause)}`
    )
  }
  if (!response.ok) {
    throw new McpError(
      `O MCP respondeu ${response.status} ${response.statusText} em /health.`
    )
  }

  let body: unknown
  try {
    body = await response.json()
  } catch (cause) {
    throw new McpError(
      `Resposta de /health não é JSON válido: ${describeError(cause)}`
    )
  }

  const parsed = healthSchema.safeParse(body)
  if (!parsed.success) {
    const message = describeValidationError(parsed.error)
    console.warn(`[viewer] payload de /health inválido: ${message}`)
    throw new McpError(`Payload de /health inválido: ${message}`)
  }
  return parsed.data
}
