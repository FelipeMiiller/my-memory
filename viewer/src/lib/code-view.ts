import { describeUnknownError } from "./errors"
import { describeValidationError } from "./mcp-client"
import type { McpToolResult } from "./mcp-client"
import type {
  CodeNeighbor,
  CodeSearchResponse,
  CodeSymbol,
  MiniSchema,
} from "./schemas"

/**
 * Lógica da rota `/code` — a parte que não depende de React.
 *
 * Tudo aqui é função pura: dá para testar no ambiente `node` do Vitest, sem
 * DOM e sem biblioteca nova. O componente (`src/routes/code.tsx`) só consome
 * estas funções.
 *
 * ## O contrato das tools de código
 *
 * As tools `memory_code_search` e `memory_code_neighbors` devolvem hoje o
 * JSON **no texto** (`internal/mcp/code_handlers.go` usa `NewTextResult`), sem
 * `structuredContent`. A rota precisa funcionar nos dois mundos:
 *
 *   - `structuredContent` presente e válido → usa o payload estruturado;
 *   - só texto → faz `JSON.parse` e valida com o **mesmo** schema zod.
 *
 * `normalizeCodePayload` é o ponto único onde essa bifurcação acontece.
 */

/** Tipos de aresta estrutural do code graph (ADR-061). */
export const CODE_EDGE_KINDS = [
  "calls",
  "imports",
  "inherits",
  "uses_type",
  "references",
] as const

export type CodeEdgeKind = (typeof CODE_EDGE_KINDS)[number]

/** De onde veio o payload normalizado. */
export type CodePayloadSource = "structured" | "text" | "none"

/** Resultado da normalização de uma tool de código. */
export interface NormalizedCodePayload<T> {
  /** Payload validado, ou `null` quando não deu para interpretar. */
  data: T | null
  /** `structured` = veio em `structuredContent`; `text` = veio no texto. */
  source: CodePayloadSource
  /** Mensagem do erro, quando houve. */
  error: string | null
}

/**
 * Normaliza o resultado de uma tool de código.
 *
 * Usa o `structuredContent` quando ele existe e valida; senão faz
 * `JSON.parse` do texto e valida com o mesmo schema. Nunca lança: a tela
 * precisa mostrar o que o MCP devolveu, mesmo quando o payload é garbage.
 */
export function normalizeCodePayload<T>(
  result: McpToolResult<T>,
  schema: MiniSchema<T>
): NormalizedCodePayload<T> {
  if (result.hasStructured && result.structured !== null) {
    return { data: result.structured, source: "structured", error: null }
  }

  const text = result.text.trim()
  if (text === "") {
    return {
      data: null,
      source: "none",
      error:
        result.validationError ??
        "A tool de código não devolveu nenhum conteúdo.",
    }
  }

  let raw: unknown
  try {
    raw = JSON.parse(text)
  } catch (cause) {
    return {
      data: null,
      source: "text",
      error: `A resposta da tool de código não é JSON: ${describeUnknownError(
        cause
      )}`,
    }
  }

  const parsed = schema.safeParse(raw)
  if (!parsed.success) {
    return {
      data: null,
      source: "text",
      error: `Payload de código inválido: ${describeValidationError(
        parsed.error
      )}`,
    }
  }

  return { data: parsed.data, source: "text", error: null }
}

/** Normaliza um rótulo de tipo (aresta ou kind de símbolo). */
export function normalizeKind(kind: string): string {
  return kind.trim().toLowerCase()
}

/** Lista de símbolos de uma resposta de busca (tolera `null`). */
export function codeSymbols(response: CodeSearchResponse): CodeSymbol[] {
  return response.results ?? []
}

/** Rótulo de exibição de um símbolo: qualified_name quando existe. */
export function symbolLabel(symbol: CodeSymbol): string {
  return symbol.qualified_name.trim() !== ""
    ? symbol.qualified_name
    : symbol.name
}

/**
 * Call-site de uma aresta: `arquivo:linha` da CHAMADA.
 *
 * Sem arquivo, devolve `linha N`; sem linha, devolve só o arquivo; sem os
 * dois, `—`.
 */
export function formatCallSite(file: string, line: number): string {
  const path = file.trim()
  const hasLine = Number.isFinite(line) && line > 0
  if (path === "") return hasLine ? `linha ${line}` : "—"
  if (!hasLine) return path
  return `${path}:${line}`
}

/** Local da DEFINIÇÃO de um símbolo: `arquivo:linha` onde ele nasce. */
export function formatDefinitionSite(symbol: CodeSymbol): string {
  return formatCallSite(symbol.file, symbol.start_line)
}

/** Grupo de vizinhos que compartilham o mesmo tipo de aresta. */
export interface NeighborGroup {
  kind: string
  neighbors: CodeNeighbor[]
}

/**
 * Agrupa vizinhos por tipo de aresta.
 *
 * A ordem é a canônica do code graph (`calls`, `imports`, `inherits`,
 * `uses_type`, `references`); tipos desconhecidos vão para o fim, em ordem
 * alfabética.
 */
export function groupNeighborsByKind(
  neighbors: CodeNeighbor[]
): NeighborGroup[] {
  const buckets = new Map<string, CodeNeighbor[]>()
  for (const neighbor of neighbors) {
    const key = normalizeKind(neighbor.kind)
    const bucket = buckets.get(key)
    if (bucket) {
      bucket.push(neighbor)
    } else {
      buckets.set(key, [neighbor])
    }
  }

  const known = CODE_EDGE_KINDS.filter((kind) => buckets.has(kind))
  const unknown = [...buckets.keys()]
    .filter((kind) => !(CODE_EDGE_KINDS as readonly string[]).includes(kind))
    .sort()

  return [...known, ...unknown].map((kind) => ({
    kind,
    neighbors: buckets.get(kind) ?? [],
  }))
}

/** Filtra vizinhos pelos tipos de aresta marcados. Lista vazia = tudo. */
export function filterNeighborsByKind(
  neighbors: CodeNeighbor[],
  kinds: string[]
): CodeNeighbor[] {
  const wanted = kinds.map(normalizeKind).filter((kind) => kind !== "")
  if (wanted.length === 0) return neighbors
  return neighbors.filter((neighbor) =>
    wanted.includes(normalizeKind(neighbor.kind))
  )
}

/** Filtros de cliente da lista de símbolos. */
export interface SymbolFilter {
  /** Texto livre: casa com nome, qualified_name, arquivo e assinatura. */
  query?: string
  /** Tipo do símbolo (`function`, `method`, `class`, ...). */
  kind?: string
  /** Linguagem do símbolo (`go`, `python`, ...). */
  language?: string
}

/** Filtra a lista de símbolos por texto, tipo e linguagem. */
export function filterSymbols(
  symbols: CodeSymbol[],
  filter: SymbolFilter
): CodeSymbol[] {
  const query = (filter.query ?? "").trim().toLowerCase()
  const kind = normalizeKind(filter.kind ?? "")
  const language = normalizeKind(filter.language ?? "")

  return symbols.filter((symbol) => {
    if (kind !== "" && normalizeKind(symbol.kind) !== kind) return false
    if (language !== "" && normalizeKind(symbol.language) !== language) {
      return false
    }
    if (query !== "") {
      const haystack = [
        symbol.name,
        symbol.qualified_name,
        symbol.file,
        symbol.signature ?? "",
      ]
        .join(" ")
        .toLowerCase()
      if (!haystack.includes(query)) return false
    }
    return true
  })
}

/** Valores distintos de uma propriedade dos símbolos, em ordem alfabética. */
export function distinctSymbolValues(
  symbols: CodeSymbol[],
  pick: (symbol: CodeSymbol) => string
): string[] {
  const values = new Set<string>()
  for (const symbol of symbols) {
    const value = pick(symbol).trim()
    if (value !== "") values.add(value)
  }
  return [...values].sort()
}

/** Rótulo legível da direção de uma aresta. */
export function describeNeighborDirection(direction: string): string {
  switch (normalizeKind(direction)) {
    case "out":
      return "chamada (saída)"
    case "in":
      return "chamado por (entrada)"
    default:
      return direction.trim() === "" ? "—" : direction.trim()
  }
}
