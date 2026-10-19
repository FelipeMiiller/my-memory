import { createFileRoute } from "@tanstack/react-router"
import type { ErrorComponentProps } from "@tanstack/react-router"
import { ToolPayload } from "@/components/tool-payload"
import {
  DataTable,
  EmptyState,
  ErrorState,
  KeyValue,
  Loading,
  Panel,
  Stat,
} from "@/components/ui-kit"
import { formatScore } from "@/lib/format"
import { describeUnknownError } from "@/lib/errors"
import { useDrift, useFindPath, useImpact } from "@/lib/queries"
import { useViewerStore } from "@/store/viewer-store"

/**
 * Drift: o quanto a documentação ficou para trás do código.
 *
 * Consome `memory_get_drift` (notas defasadas e código órfão),
 * `memory_get_impact` (blast radius de um nó) e `memory_find_path` (caminho
 * entre dois nós).
 */
export const Route = createFileRoute("/drift")({
  component: DriftRoute,
  errorComponent: DriftError,
})

function DriftRoute() {
  const drift = useDrift()

  const impactNode = useViewerStore((state) => state.impactNode)
  const setImpactNode = useViewerStore((state) => state.setImpactNode)
  const pathSource = useViewerStore((state) => state.pathSource)
  const pathTarget = useViewerStore((state) => state.pathTarget)
  const setPathSource = useViewerStore((state) => state.setPathSource)
  const setPathTarget = useViewerStore((state) => state.setPathTarget)

  const impact = useImpact(impactNode)
  const path = useFindPath(pathSource, pathTarget)

  return (
    <div className="space-y-4">
      <Panel
        title="Desvio semântico"
        description="memory_get_drift — documentação defasada e código sem nota."
      >
        {drift.isPending ? (
          <Loading />
        ) : drift.isError ? (
          <ErrorState message={drift.error.message} />
        ) : (
          <ToolPayload result={drift.data}>
            {(report) => (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
                  <Stat
                    label="Score geral"
                    value={`${report.overall_score.toFixed(1).replace(".", ",")}/100`}
                  />
                  <Stat label="Crítico" value={report.critical_count} />
                  <Stat label="Alto" value={report.high_count} />
                  <Stat label="Médio" value={report.medium_count} />
                  <Stat label="Baixo" value={report.low_count} />
                </div>

                <div className="divide-y divide-border">
                  <KeyValue label="Faixa Git" value={report.range} />
                  <KeyValue
                    label="Commits"
                    value={String(report.total_commits)}
                  />
                  <KeyValue
                    label="Arquivos modificados"
                    value={String(report.total_changed_files)}
                  />
                  {report.analyzed_at ? (
                    <KeyValue label="Analisado em" value={report.analyzed_at} />
                  ) : null}
                </div>

                <div>
                  <h3 className="mb-2 text-xs font-semibold text-muted-foreground">
                    Notas e ADRs defasados
                  </h3>
                  {report.drifted_notes.length === 0 ? (
                    <EmptyState message="Nenhuma nota com desvio significativo: a documentação reflete fielmente o código." />
                  ) : (
                    <DataTable
                      headers={[
                        "Nota",
                        "Score",
                        "Severidade",
                        "Commits atrás",
                        "Diagnóstico",
                      ]}
                      rows={report.drifted_notes.map((note) => [
                        note.title || note.path,
                        note.drift_score.toFixed(1).replace(".", ","),
                        note.severity,
                        String(note.commits_behind),
                        note.reason,
                      ])}
                    />
                  )}
                </div>

                <div>
                  <h3 className="mb-2 text-xs font-semibold text-muted-foreground">
                    Código órfão
                  </h3>
                  {report.uncovered_code.length === 0 ? (
                    <EmptyState message="Nenhum arquivo de código sem nota vinculada." />
                  ) : (
                    <DataTable
                      headers={["Arquivo", "Status", "Linhas", "Ação sugerida"]}
                      rows={report.uncovered_code.map((code) => [
                        code.file_path,
                        code.status,
                        `+${code.additions}/-${code.deletions}`,
                        code.suggested_action,
                      ])}
                    />
                  )}
                </div>
              </div>
            )}
          </ToolPayload>
        )}
      </Panel>

      <Panel
        title="Raio de impacto"
        description="memory_get_impact — blast radius de um nó do grafo."
      >
        <label className="block text-xs font-medium text-muted-foreground">
          Nó alvo
          <input
            value={impactNode}
            onChange={(event) => setImpactNode(event.target.value.trim())}
            placeholder="ADR-060"
            className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
          />
        </label>

        <div className="mt-3">
          {impactNode === "" ? (
            <EmptyState message="Informe um nó para medir o raio de impacto." />
          ) : impact.isPending ? (
            <Loading />
          ) : impact.isError ? (
            <ErrorState message={impact.error.message} />
          ) : (
            <ToolPayload result={impact.data}>
              {(result) => (
                <div className="space-y-3">
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                    <Stat label="Risco" value={result.risk_level} />
                    <Stat
                      label="Score"
                      value={result.risk_score.toFixed(1).replace(".", ",")}
                    />
                    <Stat label="Diretos" value={result.direct_dependents} />
                    <Stat
                      label="Indiretos"
                      value={result.indirect_dependents}
                    />
                  </div>
                  {result.nodes.length === 0 ? (
                    <EmptyState message="Nenhum nó dependente dentro da profundidade configurada." />
                  ) : (
                    <DataTable
                      headers={[
                        "Profundidade",
                        "Severidade",
                        "Relação",
                        "Via",
                        "ID afetado",
                      ]}
                      rows={result.nodes.map((node) => [
                        String(node.depth),
                        node.severity,
                        node.relation,
                        node.via_node,
                        node.id,
                      ])}
                    />
                  )}
                </div>
              )}
            </ToolPayload>
          )}
        </div>
      </Panel>

      <Panel
        title="Caminho entre dois nós"
        description="memory_find_path — a rota mais barata (custo epistêmico) no grafo."
      >
        <div className="grid gap-2 sm:grid-cols-2">
          <label className="block text-xs font-medium text-muted-foreground">
            Origem
            <input
              value={pathSource}
              onChange={(event) => setPathSource(event.target.value.trim())}
              placeholder="ADR-060"
              className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
            />
          </label>
          <label className="block text-xs font-medium text-muted-foreground">
            Destino
            <input
              value={pathTarget}
              onChange={(event) => setPathTarget(event.target.value.trim())}
              placeholder="ADR-059"
              className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
            />
          </label>
        </div>

        <div className="mt-3">
          {pathSource === "" || pathTarget === "" ? (
            <EmptyState message="Informe origem e destino para buscar um caminho." />
          ) : path.isPending ? (
            <Loading />
          ) : path.isError ? (
            <ErrorState message={path.error.message} />
          ) : (
            <ToolPayload result={path.data}>
              {(result) => (
                <div className="space-y-3">
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                    <Stat
                      label="Encontrado"
                      value={result.found ? "sim" : "não"}
                    />
                    <Stat label="Saltos" value={result.hops} />
                    <Stat
                      label="Custo"
                      value={formatScore(result.total_cost, 3)}
                    />
                    <Stat label="Modo" value={result.cost_mode} />
                  </div>
                  {!result.found ? (
                    <EmptyState message={result.summary} />
                  ) : result.edges.length === 0 ? (
                    <EmptyState message="Origem e destino são o mesmo nó." />
                  ) : (
                    <DataTable
                      headers={[
                        "De",
                        "Para",
                        "Relação",
                        "Status epistêmico",
                        "Direção",
                        "Custo",
                      ]}
                      rows={result.edges.map((edge) => [
                        edge.from,
                        edge.to,
                        edge.relation,
                        edge.epistemic_status,
                        edge.direction,
                        formatScore(edge.cost, 3),
                      ])}
                    />
                  )}
                </div>
              )}
            </ToolPayload>
          )}
        </div>
      </Panel>
    </div>
  )
}

function DriftError({ error }: ErrorComponentProps) {
  return (
    <ErrorState
      title="Erro na rota de Drift"
      message={describeUnknownError(error)}
    />
  )
}
