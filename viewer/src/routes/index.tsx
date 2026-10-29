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
import { formatUptime } from "@/lib/format"
import { describeUnknownError } from "@/lib/errors"
import { useDoctor, useHealth } from "@/lib/queries"

/**
 * Overview: a saúde do servidor MCP e o diagnóstico do grafo.
 *
 * Consome `GET /health` (readiness) e `memory_doctor` (health score, dead
 * links, notas órfãs).
 */
export const Route = createFileRoute("/")({
  component: OverviewRoute,
  errorComponent: OverviewError,
})

function OverviewRoute() {
  const health = useHealth()
  const doctor = useDoctor()

  return (
    <div className="space-y-4">
      <Panel
        title="Servidor MCP"
        description="Readiness do transporte HTTP que serve a própria viewer."
      >
        {health.isPending ? (
          <Loading />
        ) : health.isError ? (
          <ErrorState
            title="MCP fora do ar"
            message={health.error.message}
            hint="Confirme se o servidor MCP está no ar e se a viewer aponta para a URL certa (VITE_MCP_BASE_URL)."
          />
        ) : (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              <Stat
                label="Status"
                value={health.data.status}
                hint={health.data.name}
              />
              <Stat label="Versão" value={health.data.version} />
              <Stat
                label="Tools"
                value={health.data.tools_count}
                hint={`${health.data.active_sessions} sessões ativas`}
              />
              <Stat
                label="Uptime"
                value={formatUptime(health.data.uptime_seconds)}
                hint={`protocolo ${health.data.protocol_version}`}
              />
            </div>
            <div className="divide-y divide-border">
              <KeyValue
                label="Repositório padrão"
                value={health.data.default_repo || "—"}
              />
              <KeyValue
                label="Protocolo MCP"
                value={health.data.protocol_version}
              />
            </div>
          </div>
        )}
      </Panel>

      <Panel
        title="Diagnóstico da memória"
        description="memory_doctor — score de saúde, links quebrados e notas órfãs."
      >
        {doctor.isPending ? (
          <Loading />
        ) : doctor.isError ? (
          <ErrorState message={doctor.error.message} />
        ) : (
          <ToolPayload result={doctor.data}>
            {(report) => (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
                  <Stat
                    label="Health score"
                    value={
                      report.health_score === undefined
                        ? "—"
                        : `${report.health_score}/100`
                    }
                    hint={
                      report.health_score === undefined
                        ? "vault sem documentos"
                        : report.health_score >= 80
                          ? "saudável"
                          : "atenção"
                    }
                  />
                  <Stat label="Documentos" value={report.total_documents} />
                  <Stat label="Chunks" value={report.total_chunks} />
                  <Stat label="Nós" value={report.total_nodes} />
                  <Stat label="Arestas" value={report.total_edges} />
                </div>

                {report.missing_mtime_count ? (
                  <p className="rounded-lg border border-amber-500/40 bg-amber-500/5 px-3 py-2 text-xs">
                    {report.missing_mtime_count} documento(s) indexados antes do
                    metadado existir — rode `mem index --force` para capturar o
                    mtime.
                  </p>
                ) : null}

                <div>
                  <h3 className="mb-2 text-xs font-semibold text-muted-foreground">
                    Links quebrados
                  </h3>
                  {report.dead_links && report.dead_links.length > 0 ? (
                    <DataTable
                      headers={["Origem", "Destino", "Relação"]}
                      rows={report.dead_links.map((link) => [
                        link.source_id,
                        link.target_id,
                        link.relation,
                      ])}
                    />
                  ) : (
                    <EmptyState message="Nenhum dead link: toda conexão aponta para uma nota existente." />
                  )}
                </div>

                <div>
                  <h3 className="mb-2 text-xs font-semibold text-muted-foreground">
                    Notas órfãs
                  </h3>
                  {report.orphan_notes && report.orphan_notes.length > 0 ? (
                    <DataTable
                      headers={["Nota", "ID"]}
                      rows={report.orphan_notes.map((note) => [
                        note.title || note.id,
                        note.id,
                      ])}
                    />
                  ) : (
                    <EmptyState message="Nenhuma nota órfã: todo o conhecimento está interligado." />
                  )}
                </div>
              </div>
            )}
          </ToolPayload>
        )}
      </Panel>
    </div>
  )
}

function OverviewError({ error }: ErrorComponentProps) {
  return (
    <ErrorState
      title="Erro no Overview"
      message={describeUnknownError(error)}
    />
  )
}
