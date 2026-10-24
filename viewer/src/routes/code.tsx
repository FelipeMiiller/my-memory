import { useState } from "react"
import type { FormEvent } from "react"
import { createFileRoute } from "@tanstack/react-router"
import type { ErrorComponentProps } from "@tanstack/react-router"
import {
  DataTable,
  EmptyState,
  ErrorState,
  KeyValue,
  Loading,
  Panel,
  RawText,
  Stat,
} from "@/components/ui-kit"
import {
  codeSymbols,
  describeNeighborDirection,
  distinctSymbolValues,
  filterNeighborsByKind,
  filterSymbols,
  formatCallSite,
  formatDefinitionSite,
  groupNeighborsByKind,
  normalizeCodePayload,
  symbolLabel,
} from "@/lib/code-view"
import { describeUnknownError } from "@/lib/errors"
import { useCodeNeighbors, useCodeSearch } from "@/lib/queries"
import { codeNeighborsSchema, codeSearchSchema } from "@/lib/schemas"
import type { CodeSymbol } from "@/lib/schemas"

/**
 * Código: os símbolos de um repositório e seus vizinhos estruturais.
 *
 * Consome `memory_code_search` (lista de símbolos) e `memory_code_neighbors`
 * (vizinhos de um símbolo, com o **tipo da aresta** e o **call-site**).
 *
 * A tela é fina de propósito: a normalização da payload (hoje o MCP devolve
 * JSON no texto, amanhã devolve `structuredContent`), o agrupamento por tipo
 * de aresta e a formatação do call-site vivem em `src/lib/code-view.ts`.
 */
export const Route = createFileRoute("/code")({
  component: CodeRoute,
  errorComponent: CodeError,
})

const DIRECOES = [
  { value: "both", label: "Entrada e saída" },
  { value: "out", label: "Saída (o que o símbolo chama)" },
  { value: "in", label: "Entrada (quem chama o símbolo)" },
]

function CodeRoute() {
  const [queryInput, setQueryInput] = useState("")
  const [appliedQuery, setAppliedQuery] = useState("")
  const [limit, setLimit] = useState(20)
  const [textFilter, setTextFilter] = useState("")
  const [kindFilter, setKindFilter] = useState("")
  const [languageFilter, setLanguageFilter] = useState("")
  const [selected, setSelected] = useState<CodeSymbol | null>(null)
  const [depth, setDepth] = useState(1)
  const [direction, setDirection] = useState("both")
  const [kinds, setKinds] = useState<string[]>([])

  const search = useCodeSearch(appliedQuery, limit)
  const searchPayload = search.data
    ? normalizeCodePayload(search.data, codeSearchSchema)
    : null

  const symbols = searchPayload?.data ? codeSymbols(searchPayload.data) : []
  const visible = filterSymbols(symbols, {
    query: textFilter,
    kind: kindFilter,
    language: languageFilter,
  })

  const selectedLabel = selected ? symbolLabel(selected) : ""
  const neighbors = useCodeNeighbors(selectedLabel, depth, direction)
  const neighborsPayload = neighbors.data
    ? normalizeCodePayload(neighbors.data, codeNeighborsSchema)
    : null

  const neighborList = neighborsPayload?.data?.neighbors ?? []
  const groups = groupNeighborsByKind(
    filterNeighborsByKind(neighborList, kinds)
  )

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setAppliedQuery(queryInput.trim())
  }

  function toggleKind(kind: string) {
    setKinds((atuais) =>
      atuais.includes(kind)
        ? atuais.filter((item) => item !== kind)
        : [...atuais, kind]
    )
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[22rem_1fr]">
      <div className="space-y-4">
        <Panel
          title="Busca de símbolos"
          description="memory_code_search — símbolos indexados do code graph."
        >
          <form onSubmit={submitSearch} className="space-y-2">
            <label className="block text-xs font-medium text-muted-foreground">
              Query
              <input
                value={queryInput}
                onChange={(event) => setQueryInput(event.target.value)}
                placeholder="Save ou internal/store.Store.Save"
                className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
              />
            </label>
            <label className="block text-xs font-medium text-muted-foreground">
              Limite
              <input
                type="number"
                min={1}
                max={100}
                value={limit}
                onChange={(event) =>
                  setLimit(Math.max(1, Number(event.target.value) || 1))
                }
                className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
              />
            </label>
            <button
              type="submit"
              className="w-full rounded-md border border-border px-3 py-1.5 text-sm font-medium hover:bg-accent"
            >
              Buscar
            </button>
          </form>
        </Panel>

        <Panel
          title="Filtros"
          description="Aplicados no cliente, sobre o resultado da busca."
        >
          <div className="space-y-2">
            <label className="block text-xs font-medium text-muted-foreground">
              Texto
              <input
                value={textFilter}
                onChange={(event) => setTextFilter(event.target.value)}
                placeholder="nome, arquivo ou assinatura"
                className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
              />
            </label>
            <label className="block text-xs font-medium text-muted-foreground">
              Tipo
              <select
                value={kindFilter}
                onChange={(event) => setKindFilter(event.target.value)}
                className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
              >
                <option value="">Todos</option>
                {distinctSymbolValues(symbols, (symbol) => symbol.kind).map(
                  (kind) => (
                    <option key={kind} value={kind}>
                      {kind}
                    </option>
                  )
                )}
              </select>
            </label>
            <label className="block text-xs font-medium text-muted-foreground">
              Linguagem
              <select
                value={languageFilter}
                onChange={(event) => setLanguageFilter(event.target.value)}
                className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
              >
                <option value="">Todas</option>
                {distinctSymbolValues(symbols, (symbol) => symbol.language).map(
                  (language) => (
                    <option key={language} value={language}>
                      {language}
                    </option>
                  )
                )}
              </select>
            </label>
          </div>
        </Panel>

        <Panel title="Símbolos">
          {appliedQuery === "" ? (
            <EmptyState message="Busque um símbolo para listar o code graph." />
          ) : search.isPending ? (
            <Loading />
          ) : search.isError ? (
            <ErrorState
              title="MCP indisponível"
              message={search.error.message}
              hint="O code graph precisa do code pipeline habilitado (ADR-047)."
            />
          ) : searchPayload?.error ? (
            <ErrorState
              title="Payload de código inválido"
              message={searchPayload.error}
            />
          ) : visible.length === 0 ? (
            <EmptyState message="Nenhum símbolo corresponde à busca e aos filtros." />
          ) : (
            <ul className="max-h-[28rem] space-y-0.5 overflow-auto">
              {visible.map((symbol) => (
                <li key={`${symbol.symbol_id}-${symbol.qualified_name}`}>
                  <button
                    type="button"
                    onClick={() => setSelected(symbol)}
                    className={
                      symbolLabel(symbol) === selectedLabel
                        ? "w-full rounded-md bg-secondary px-2 py-1.5 text-left text-sm font-medium text-secondary-foreground"
                        : "w-full rounded-md px-2 py-1.5 text-left text-sm hover:bg-accent"
                    }
                  >
                    <span className="block truncate">{symbolLabel(symbol)}</span>
                    <span className="block truncate text-[11px] text-muted-foreground">
                      {symbol.kind} · {symbol.language} ·{" "}
                      {formatDefinitionSite(symbol)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      </div>

      <div className="space-y-4">
        <Panel
          title="Símbolo selecionado"
          description="memory_code_neighbors — vizinhos estruturais, tipo da aresta e call-site."
        >
          {selected === null ? (
            <EmptyState message="Escolha um símbolo na lista para ver seus vizinhos." />
          ) : (
            <div className="space-y-3">
              <div className="divide-y divide-border">
                <KeyValue label="Símbolo" value={symbolLabel(selected)} />
                <KeyValue label="Tipo" value={selected.kind} />
                <KeyValue label="Linguagem" value={selected.language} />
                <KeyValue
                  label="Definição"
                  value={formatDefinitionSite(selected)}
                />
                {selected.signature ? (
                  <KeyValue label="Assinatura" value={selected.signature} />
                ) : null}
              </div>

              <div className="flex flex-wrap items-end gap-2">
                <label className="block text-xs font-medium text-muted-foreground">
                  Profundidade
                  <input
                    type="number"
                    min={1}
                    max={5}
                    value={depth}
                    onChange={(event) =>
                      setDepth(Math.min(5, Math.max(1, Number(event.target.value) || 1)))
                    }
                    className="mt-1 w-20 rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
                  />
                </label>
                <label className="block text-xs font-medium text-muted-foreground">
                  Direção
                  <select
                    value={direction}
                    onChange={(event) => setDirection(event.target.value)}
                    className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring sm:w-56"
                  >
                    {DIRECOES.map((item) => (
                      <option key={item.value} value={item.value}>
                        {item.label}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
            </div>
          )}
        </Panel>

        {selected !== null ? (
          <Panel
            title="Vizinhos estruturais"
            description="Agrupados por tipo de aresta, com o call-site da chamada."
            actions={
              groups.length > 0 ? (
                <div className="flex flex-wrap gap-1">
                  {groups.map((group) => (
                    <button
                      key={group.kind}
                      type="button"
                      onClick={() => toggleKind(group.kind)}
                      className={
                        kinds.includes(group.kind)
                          ? "rounded-md bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground"
                          : "rounded-md border border-border px-2 py-0.5 text-xs hover:bg-accent"
                      }
                    >
                      {group.kind} ({group.neighbors.length})
                    </button>
                  ))}
                </div>
              ) : null
            }
          >
            {neighbors.isPending ? (
              <Loading />
            ) : neighbors.isError ? (
              <ErrorState
                title="MCP indisponível"
                message={neighbors.error.message}
                hint="O código precisa estar indexado no code graph (ADR-047)."
              />
            ) : neighborsPayload?.error ? (
              <div className="space-y-2">
                <ErrorState
                  title="Payload de código inválido"
                  message={neighborsPayload.error}
                />
                <RawText text={neighbors.data?.text ?? ""} />
              </div>
            ) : groups.length === 0 ? (
              <EmptyState message="Nenhum vizinho estrutural para este símbolo." />
            ) : (
              <div className="space-y-4">
                {groups.map((group) => (
                  <div key={group.kind}>
                    <h3 className="mb-2 text-xs font-semibold text-muted-foreground">
                      {group.kind} ({group.neighbors.length})
                    </h3>
                    <DataTable
                      headers={["Vizinho", "Direção", "Call-site", "Hop"]}
                      rows={group.neighbors.map((neighbor) => [
                        neighbor.symbol,
                        describeNeighborDirection(neighbor.direction),
                        formatCallSite(neighbor.file, neighbor.line),
                        String(neighbor.hop),
                      ])}
                    />
                  </div>
                ))}
              </div>
            )}
          </Panel>
        ) : null}

        {searchPayload?.data ? (
          <Panel title="Resumo da busca">
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
              <Stat label="Query" value={searchPayload.data.query} />
              <Stat label="Símbolos" value={searchPayload.data.count} />
              <Stat
                label="Fonte"
                value={
                  searchPayload.source === "structured"
                    ? "structuredContent"
                    : "texto JSON"
                }
              />
            </div>
          </Panel>
        ) : null}
      </div>
    </div>
  )
}

function CodeError({ error }: ErrorComponentProps) {
  return (
    <ErrorState
      title="Erro na rota de Código"
      message={describeUnknownError(error)}
    />
  )
}
