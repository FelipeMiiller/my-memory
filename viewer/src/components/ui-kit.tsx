import type { ReactNode } from "react"

/** Painel com título — a unidade visual de todas as rotas. */
export function Panel({
  title,
  description,
  actions,
  children,
}: {
  title: string
  description?: string
  actions?: ReactNode
  children: ReactNode
}) {
  return (
    <section className="rounded-xl border border-border bg-card text-card-foreground">
      <header className="flex flex-wrap items-start justify-between gap-3 border-b border-border px-4 py-3">
        <div>
          <h2 className="text-sm font-semibold tracking-tight">{title}</h2>
          {description ? (
            <p className="mt-0.5 text-xs text-muted-foreground">
              {description}
            </p>
          ) : null}
        </div>
        {actions ? (
          <div className="flex items-center gap-2">{actions}</div>
        ) : null}
      </header>
      <div className="px-4 py-3">{children}</div>
    </section>
  )
}

/** Tile de métrica: rótulo em cima, valor em baixo. */
export function Stat({
  label,
  value,
  hint,
}: {
  label: string
  value: ReactNode
  hint?: string
}) {
  return (
    <div className="rounded-lg border border-border bg-background px-3 py-2">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-0.5 text-xl font-semibold tabular-nums">{value}</p>
      {hint ? (
        <p className="mt-0.5 text-[11px] text-muted-foreground">{hint}</p>
      ) : null}
    </div>
  )
}

/** Par rótulo/valor, para metadados. */
export function KeyValue({
  label,
  value,
}: {
  label: string
  value: ReactNode
}) {
  return (
    <div className="flex items-baseline justify-between gap-4 py-1 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-medium break-all">{value}</span>
    </div>
  )
}

/** Estado vazio — o dado veio, mas não tem o que mostrar. */
export function EmptyState({ message }: { message: string }) {
  return (
    <p className="rounded-lg border border-dashed border-border px-3 py-6 text-center text-sm text-muted-foreground">
      {message}
    </p>
  )
}

/** Estado de carregamento. */
export function Loading({ message = "Carregando…" }: { message?: string }) {
  return <p className="px-1 py-4 text-sm text-muted-foreground">{message}</p>
}

/** Estado de erro — o MCP pode estar fora do ar. */
export function ErrorState({
  title = "Falha ao consultar o MCP",
  message,
  hint,
}: {
  title?: string
  message: string
  hint?: string
}) {
  return (
    <div className="rounded-lg border border-destructive/40 bg-destructive/5 px-3 py-3">
      <p className="text-sm font-semibold text-destructive">{title}</p>
      <p className="mt-1 text-sm break-words">{message}</p>
      {hint ? (
        <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
      ) : null}
    </div>
  )
}

/**
 * Texto cru da tool, no formato que o MCP devolveu.
 *
 * É o fallback do painel: quando não há `structuredContent`, ou quando o
 * payload não valida, o que o usuário vê é exatamente o que o MCP respondeu.
 */
export function RawText({ text }: { text: string }) {
  if (text.trim() === "") {
    return <EmptyState message="A tool não devolveu nenhum conteúdo." />
  }
  return (
    <pre className="max-h-[32rem] overflow-auto rounded-lg border border-border bg-muted/40 p-3 text-xs leading-relaxed whitespace-pre-wrap">
      {text}
    </pre>
  )
}

/** Tabela simples, sem dependência extra. */
export function DataTable({
  headers,
  rows,
}: {
  headers: string[]
  rows: ReactNode[][]
}) {
  if (rows.length === 0) {
    return <EmptyState message="Nada para exibir." />
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-border text-xs text-muted-foreground">
            {headers.map((header) => (
              <th key={header} className="py-2 pr-4 font-medium">
                {header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => (
            <tr key={index} className="border-b border-border/60 last:border-0">
              {row.map((cell, cellIndex) => (
                <td key={cellIndex} className="py-2 pr-4 align-top">
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
