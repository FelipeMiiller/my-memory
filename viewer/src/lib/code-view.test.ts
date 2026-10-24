import { describe, expect, it } from "vitest"
import type { McpToolResult } from "./mcp-client"
import {
  CODE_EDGE_KINDS,
  describeNeighborDirection,
  distinctSymbolValues,
  filterNeighborsByKind,
  filterSymbols,
  formatCallSite,
  formatDefinitionSite,
  groupNeighborsByKind,
  normalizeCodePayload,
  normalizeKind,
  symbolLabel,
} from "./code-view"
import { codeSearchSchema } from "./schemas"
import type { CodeNeighbor, CodeSearchResponse, CodeSymbol } from "./schemas"

/**
 * A rota `/code` depende de duas garantias:
 *
 *   1. a payload das tools de código é normalizada nos DOIS worlds — com
 *      `structuredContent` e só no texto (o estado atual do servidor Go);
 *   2. o call-site e o tipo da aresta chegam até a tela sem perder a origem.
 */

const SIMBOLO: CodeSymbol = {
  symbol_id: 1,
  qualified_name: "internal/store.Store.Save",
  name: "Save",
  kind: "method",
  language: "go",
  file: "internal/store/vault.go",
  start_line: 42,
  end_line: 60,
  signature: "func (s *Store) Save(doc *Document) error",
  score: 2,
  boosted: true,
}

const BUSCA_VALIDA = {
  query: "Save",
  count: 1,
  results: [SIMBOLO],
}

function resultadoTexto<T>(
  text: string,
  extras: Partial<McpToolResult<T>> = {}
): McpToolResult<T> {
  return {
    text,
    structured: null,
    hasStructured: false,
    validationError: null,
    ...extras,
  }
}

function vizinho(
  kind: string,
  symbol: string,
  file: string,
  line: number,
  direction = "out"
): CodeNeighbor {
  return { direction, kind, symbol, file, line, hop: 1 }
}

describe("normalização da payload de código", () => {
  it("usa o structuredContent quando ele existe e valida", () => {
    const resultado = normalizeCodePayload(
      {
        text: "resumo em texto",
        structured: BUSCA_VALIDA,
        hasStructured: true,
        validationError: null,
      },
      codeSearchSchema
    )

    expect(resultado.source).toBe("structured")
    expect(resultado.error).toBeNull()
    expect(resultado.data?.count).toBe(1)
    expect(resultado.data?.results[0]?.name).toBe("Save")
  })

  it("faz JSON.parse do texto quando a tool não devolve structuredContent", () => {
    const resultado = normalizeCodePayload<CodeSearchResponse>(
      resultadoTexto<CodeSearchResponse>(JSON.stringify(BUSCA_VALIDA)),
      codeSearchSchema
    )

    expect(resultado.source).toBe("text")
    expect(resultado.error).toBeNull()
    expect(resultado.data?.query).toBe("Save")
    expect(resultado.data?.results).toHaveLength(1)
    expect(resultado.data?.results[0]?.file).toBe("internal/store/vault.go")
  })

  it("rejeita um texto que não é JSON", () => {
    const resultado = normalizeCodePayload(
      resultadoTexto("memory_code_search indisponível: code pipeline não configurado."),
      codeSearchSchema
    )

    expect(resultado.data).toBeNull()
    expect(resultado.source).toBe("text")
    expect(resultado.error).toContain("não é JSON")
  })

  it("rejeita um JSON que não bate com o schema", () => {
    const resultado = normalizeCodePayload(
      resultadoTexto(JSON.stringify({ query: "Save", count: "um" })),
      codeSearchSchema
    )

    expect(resultado.data).toBeNull()
    expect(resultado.error).toContain("Payload de código inválido")
  })

  it("aceita results vazio vindo como null do Go", () => {
    const resultado = normalizeCodePayload<CodeSearchResponse>(
      resultadoTexto<CodeSearchResponse>(
        JSON.stringify({ query: "nada", count: 0, results: null })
      ),
      codeSearchSchema
    )

    expect(resultado.error).toBeNull()
    expect(resultado.data?.results).toEqual([])
  })

  it("reporta texto vazio sem quebrar a tela", () => {
    const resultado = normalizeCodePayload(
      resultadoTexto("   "),
      codeSearchSchema
    )

    expect(resultado.data).toBeNull()
    expect(resultado.source).toBe("none")
    expect(resultado.error).toBeTruthy()
  })
})

describe("call-site dos vizinhos", () => {
  it("formata arquivo:linha", () => {
    expect(formatCallSite("internal/store/vault.go", 42)).toBe(
      "internal/store/vault.go:42"
    )
  })

  it("formata o call-site sem arquivo e sem linha", () => {
    expect(formatCallSite("", 42)).toBe("linha 42")
    expect(formatCallSite("", 0)).toBe("—")
    expect(formatCallSite("internal/store/vault.go", 0)).toBe(
      "internal/store/vault.go"
    )
  })

  it("formata o local da definição do símbolo", () => {
    expect(formatDefinitionSite(SIMBOLO)).toBe("internal/store/vault.go:42")
  })
})

describe("agrupamento por tipo de aresta", () => {
  it("agrupa e ordena pelas arestas canônicas", () => {
    const grupos = groupNeighborsByKind([
      vizinho("references", "A", "a.go", 1),
      vizinho("calls", "B", "b.go", 2),
      vizinho("imports", "C", "c.go", 3),
    ])

    expect(grupos.map((grupo) => grupo.kind)).toEqual([
      "calls",
      "imports",
      "references",
    ])
    expect(grupos[0]?.neighbors[0]?.symbol).toBe("B")
  })

  it("joga tipos desconhecidos para o fim, em ordem alfabética", () => {
    const grupos = groupNeighborsByKind([
      vizinho("zeta", "Z", "z.go", 1),
      vizinho("calls", "B", "b.go", 2),
      vizinho("alpha", "A", "a.go", 3),
    ])

    expect(grupos.map((grupo) => grupo.kind)).toEqual([
      "calls",
      "alpha",
      "zeta",
    ])
  })

  it("mantém o call-site dentro do grupo", () => {
    const grupos = groupNeighborsByKind([
      vizinho("calls", "B", "internal/b.go", 12),
    ])

    expect(formatCallSite(grupos[0]!.neighbors[0]!.file, grupos[0]!.neighbors[0]!.line)).toBe(
      "internal/b.go:12"
    )
  })

  it("filtra vizinhos por tipo de aresta", () => {
    const vizinhos = [
      vizinho("calls", "B", "b.go", 2),
      vizinho("imports", "C", "c.go", 3),
    ]

    expect(filterNeighborsByKind(vizinhos, ["calls"])).toHaveLength(1)
    expect(filterNeighborsByKind(vizinhos, [])).toHaveLength(2)
    expect(filterNeighborsByKind(vizinhos, ["CALLS"])[0]?.symbol).toBe("B")
  })
})

describe("filtro da lista de símbolos", () => {
  const OUTRO: CodeSymbol = {
    ...SIMBOLO,
    symbol_id: 2,
    qualified_name: "internal/parser.Parse",
    name: "Parse",
    kind: "function",
    language: "python",
    file: "internal/parser/parse.py",
    start_line: 10,
    end_line: 20,
    signature: "",
    score: 1,
    boosted: false,
  }

  it("filtra por texto livre", () => {
    const resultado = filterSymbols([SIMBOLO, OUTRO], { query: "parse" })

    expect(resultado).toHaveLength(1)
    expect(resultado[0]?.name).toBe("Parse")
  })

  it("filtra por tipo e linguagem", () => {
    expect(filterSymbols([SIMBOLO, OUTRO], { kind: "method" })).toHaveLength(1)
    expect(filterSymbols([SIMBOLO, OUTRO], { language: "go" })).toHaveLength(1)
    expect(filterSymbols([SIMBOLO, OUTRO], { kind: "function", language: "go" })).toHaveLength(0)
  })

  it("devolve tudo quando não há filtro", () => {
    expect(filterSymbols([SIMBOLO, OUTRO], {})).toHaveLength(2)
  })

  it("lista os valores distintos para os filtros da tela", () => {
    expect(distinctSymbolValues([SIMBOLO, OUTRO], (s) => s.kind)).toEqual([
      "function",
      "method",
    ])
    expect(distinctSymbolValues([SIMBOLO, OUTRO], (s) => s.language)).toEqual([
      "go",
      "python",
    ])
  })
})

describe("rótulos", () => {
  it("usa o qualified_name quando existe", () => {
    expect(symbolLabel(SIMBOLO)).toBe("internal/store.Store.Save")
    expect(symbolLabel({ ...SIMBOLO, qualified_name: "" })).toBe("Save")
  })

  it("descreve a direção da aresta", () => {
    expect(describeNeighborDirection("out")).toBe("chamada (saída)")
    expect(describeNeighborDirection("in")).toBe("chamado por (entrada)")
    expect(describeNeighborDirection("")).toBe("—")
  })

  it("normaliza rótulos de tipo", () => {
    expect(normalizeKind("  Calls ")).toBe("calls")
  })

  it("expõe os tipos de aresta canônicos", () => {
    expect(CODE_EDGE_KINDS).toEqual([
      "calls",
      "imports",
      "inherits",
      "uses_type",
      "references",
    ])
  })
})
