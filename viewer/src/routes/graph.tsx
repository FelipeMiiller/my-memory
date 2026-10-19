import { createFileRoute } from "@tanstack/react-router"
import type { ErrorComponentProps } from "@tanstack/react-router"
import { Button } from "@/components/ui/button"
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
import { formatPercent, formatScore } from "@/lib/format"
import { describeUnknownError } from "@/lib/errors"
import {
  useClusters,
  useDegreeHubs,
  useInsights,
  usePagerankHubs,
  useVisualizeGraph,
} from "@/lib/queries"

/**
 * Grafo: os nós centrais, as comunidades e as conexões latentes.
 *
 * Consome `memory_get_hubs` (degree e pagerank), `memory_get_clusters` e
 * `memory_get_insights`. O grafo interativo não é desenhado aqui: a ADR-060
 * mantém o renderer em `internal/graphview`; a viewer **invoca** a tool e abre
 * a URL servida (ADR-059 D-5).
 */
export const Route = createFileRoute("/graph")({
  component: GraphRoute,
  errorComponent: GraphError,
})

function GraphRoute() {
  const degree = useDegreeHubs()
  const pagerank = usePagerankHubs()
  const clusters = useClusters()
  const insights = useInsights()
  const graph = useVisualizeGraph()

  function openInteractiveGraph() {
    graph.mutate(undefined, {
      onSuccess: (result) => {
        const url = result.structured?.url
        if (url) {
          window.open(url, "_blank", "noopener,noreferrer")
        }
      },
    })
  }

  return (
    <div className="space-y-4">
      <Panel
        title="Grafo interativo"
        description="memory_visualize_graph gera o artefato HTML servido pelo MCP."
        actions={
          <Button
            type="button"
            size="sm"
            disabled={graph.isPending}
            onClick={openInteractiveGraph}
          >
            {graph.isPending ? "Gerando…" : "Abrir grafo interativo"}
          </Button>
        }
      >
        {graph.isError ? (
          <ErrorState message={graph.error.message} />
        ) : graph.isSuccess ? (
          <ToolPayload result={graph.data}>
            {(result) => (
              <div className="space-y-2">
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                  <Stat label="Nós" value={result.total_nodes} />
                  <Stat label="Arestas" value={result.total_edges} />
                  <Stat label="Hubs" value={result.hub_count} />
                  <Stat label="Densidade" value={formatScore(result.density)} />
                </div>
                <div className="divide-y divide-border">
                  <KeyValue
                    label="URL servida"
                    value={
                      result.url ? (
                        <a
                          href={result.url}
                          target="_blank"
                          rel="noreferrer noopener"
                          className="underline"
                        >
                          {result.url}
                        </a>
                      ) : (
                        "não disponível neste transporte"
                      )
                    }
                  />
                  <KeyValue label="Arquivo" value={result.path} />
                </div>
              </div>
            )}
          </ToolPayload>
        ) : (
          <p className="text-sm text-muted-foreground">
            Clique em “Abrir grafo interativo” para gerar o artefato e abri-lo
            numa nova aba.
          </p>
        )}
      </Panel>

      <Panel
        title="Nós centrais — grau"
        description="memory_get_hubs (algorithm=degree): os hubs por grau total."
      >
        {degree.isPending ? (
          <Loading />
        ) : degree.isError ? (
          <ErrorState message={degree.error.message} />
        ) : (
          <ToolPayload result={degree.data}>
            {(hubs) =>
              hubs.length === 0 ? (
                <EmptyState message="Nenhum nó central encontrado no grafo." />
              ) : (
                <DataTable
                  headers={["#", "Nó", "Entrada", "Saída", "Grau total"]}
                  rows={hubs.map((hub, index) => [
                    String(index + 1),
                    hub.name || hub.id,
                    String(hub.in_degree),
                    String(hub.out_degree),
                    String(hub.total_degree),
                  ])}
                />
              )
            }
          </ToolPayload>
        )}
      </Panel>

      <Panel
        title="Nós centrais — PageRank"
        description="memory_get_hubs (algorithm=pagerank): a autoridade de cada nó."
      >
        {pagerank.isPending ? (
          <Loading />
        ) : pagerank.isError ? (
          <ErrorState message={pagerank.error.message} />
        ) : (
          <ToolPayload result={pagerank.data}>
            {(nodes) =>
              nodes.length === 0 ? (
                <EmptyState message="Nenhum PageRank calculado." />
              ) : (
                <DataTable
                  headers={["Rank", "Nó", "Score", "Entrada", "Saída"]}
                  rows={nodes.map((node) => [
                    String(node.rank),
                    node.name || node.id,
                    formatScore(node.score, 6),
                    String(node.in_degree),
                    String(node.out_degree),
                  ])}
                />
              )
            }
          </ToolPayload>
        )}
      </Panel>

      <Panel
        title="Comunidades"
        description="memory_get_clusters: label propagation e modularidade Newman-Girvan."
      >
        {clusters.isPending ? (
          <Loading />
        ) : clusters.isError ? (
          <ErrorState message={clusters.error.message} />
        ) : (
          <ToolPayload result={clusters.data}>
            {(result) => (
              <div className="space-y-3">
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                  <Stat
                    label="Modularidade (Q)"
                    value={formatScore(result.modularity)}
                  />
                  <Stat label="Nós" value={result.total_nodes ?? "—"} />
                  <Stat label="Arestas" value={result.total_edges ?? "—"} />
                  <Stat label="Comunidades" value={result.communities.length} />
                </div>
                {result.communities.length === 0 ? (
                  <EmptyState message="Nenhuma comunidade com o tamanho mínimo solicitado." />
                ) : (
                  <DataTable
                    headers={[
                      "ID",
                      "Líder",
                      "Tamanho",
                      "Tipo dominante",
                      "Membros",
                    ]}
                    rows={result.communities.map((community) => [
                      String(community.id),
                      community.lead_node,
                      String(community.size),
                      community.dominant_type ?? "—",
                      community.members.slice(0, 8).join(", ") +
                        (community.members.length > 8
                          ? ` +${community.members.length - 8}`
                          : ""),
                    ])}
                  />
                )}
              </div>
            )}
          </ToolPayload>
        )}
      </Panel>

      <Panel
        title="Conexões latentes"
        description="memory_get_insights: pares counterintuitive ligados por similaridade, sem link direto."
      >
        {insights.isPending ? (
          <Loading />
        ) : insights.isError ? (
          <ErrorState message={insights.error.message} />
        ) : (
          <ToolPayload result={insights.data}>
            {(connections) =>
              connections.length === 0 ? (
                <EmptyState message="Nenhuma conexão inesperada encontrada." />
              ) : (
                <DataTable
                  headers={["Origem", "Destino", "Similaridade", "Por quê"]}
                  rows={connections.map((connection) => [
                    connection.source_name || connection.source_id,
                    connection.target_name || connection.target_id,
                    formatPercent(connection.similarity),
                    connection.reason,
                  ])}
                />
              )
            }
          </ToolPayload>
        )}
      </Panel>
    </div>
  )
}

function GraphError({ error }: ErrorComponentProps) {
  return (
    <ErrorState
      title="Erro na rota do Grafo"
      message={describeUnknownError(error)}
    />
  )
}
