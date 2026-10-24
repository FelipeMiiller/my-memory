import * as z from "zod/mini"

/**
 * Contrato de schemas da viewer.
 *
 * Cada schema espelha o `structuredContent` que o handler Go já tinha antes de
 * formatar a saída em Markdown — os JSON tags dos tipos em `internal/mcp/`.
 * A fonte de verdade é o tipo Go, não este arquivo: se o lado Go mudar, o
 * schema é que precisa ser reconciliado.
 *
 * Regra de ouro (ADR-060 D-2): importar `zod/mini`, nunca o barrel `zod`.
 * `zod` completo = 92,9 kB gzip; `zod/mini` = 3,0 kB.
 */

/** Envelope mínimo de um schema zod usado pela viewer. */
export interface MiniSchema<T> {
  safeParse(
    data: unknown
  ): { success: true; data: T } | { success: false; error: unknown }
}

/* -------------------------------------------------------------------------- */
/* health — GET {base}/health                                                  */
/* -------------------------------------------------------------------------- */

/**
 * mcp.HealthResponse (internal/mcp/http_server.go:50)
 * {status, name, version, protocol_version, uptime_seconds, default_repo?, tools_count, active_sessions}
 */
export interface HealthPayload {
  status: string
  name: string
  version: string
  protocol_version: string
  uptime_seconds: number
  default_repo?: string
  tools_count: number
  active_sessions: number
}

export const healthSchema: MiniSchema<HealthPayload> = z.object({
  status: z.string(),
  name: z.string(),
  version: z.string(),
  protocol_version: z.string(),
  uptime_seconds: z.number(),
  default_repo: z.optional(z.string()),
  tools_count: z.number(),
  active_sessions: z.number(),
})

/* -------------------------------------------------------------------------- */
/* memory_doctor — mcp.DoctorReport (internal/mcp/handlers.go:143)             */
/* -------------------------------------------------------------------------- */

/** mcp.DeadLink — {source_id, target_id, relation} */
export interface DeadLink {
  source_id: string
  target_id: string
  relation: string
}

/** mcp.OrphanNote — {id, title} */
export interface OrphanNote {
  id: string
  title: string
}

/** mcp.SelfLoop — {node_id, relation} */
export interface SelfLoop {
  node_id: string
  relation: string
}

/** mcp.DesyncedChunk — {chunk_id, document_id, issue} */
export interface DesyncedChunk {
  chunk_id: string
  document_id: string
  issue: string
}

/**
 * mcp.DoctorReport
 * {total_documents, total_chunks, total_edges, total_nodes, health_score,
 *  missing_mtime_count, dead_links[], orphan_notes[], self_loops[], desynced_chunks[]}
 */
export interface DoctorReport {
  total_documents: number
  total_chunks: number
  total_edges: number
  total_nodes: number
  health_score: number
  missing_mtime_count?: number
  dead_links?: DeadLink[]
  orphan_notes?: OrphanNote[]
  self_loops?: SelfLoop[]
  desynced_chunks?: DesyncedChunk[]
}

export const doctorReportSchema: MiniSchema<DoctorReport> = z.object({
  total_documents: z.number(),
  total_chunks: z.number(),
  total_edges: z.number(),
  total_nodes: z.number(),
  health_score: z.number(),
  missing_mtime_count: z.optional(z.number()),
  dead_links: z.optional(
    z.array(
      z.object({
        source_id: z.string(),
        target_id: z.string(),
        relation: z.string(),
      })
    )
  ),
  orphan_notes: z.optional(
    z.array(
      z.object({
        id: z.string(),
        title: z.string(),
      })
    )
  ),
  self_loops: z.optional(
    z.array(
      z.object({
        node_id: z.string(),
        relation: z.string(),
      })
    )
  ),
  desynced_chunks: z.optional(
    z.array(
      z.object({
        chunk_id: z.string(),
        document_id: z.string(),
        issue: z.string(),
      })
    )
  ),
})

/* -------------------------------------------------------------------------- */
/* memory_get_hubs — mcp.GodNode / mcp.PageRankNode (internal/mcp/handlers.go)  */
/* -------------------------------------------------------------------------- */

/** mcp.GodNode (algorithm=degree) — {id, name, in_degree, out_degree, total_degree} */
export interface HubNode {
  id: string
  name: string
  in_degree: number
  out_degree: number
  total_degree: number
}

/** mcp.PageRankNode (algorithm=pagerank) — {id, name, score, rank, in_degree, out_degree} */
export interface PageRankNode {
  id: string
  name: string
  score: number
  rank: number
  in_degree: number
  out_degree: number
}

export const hubsDegreeSchema: MiniSchema<HubNode[]> = z.array(
  z.object({
    id: z.string(),
    name: z.string(),
    in_degree: z.number(),
    out_degree: z.number(),
    total_degree: z.number(),
  })
)

export const hubsPagerankSchema: MiniSchema<PageRankNode[]> = z.array(
  z.object({
    id: z.string(),
    name: z.string(),
    score: z.number(),
    rank: z.number(),
    in_degree: z.number(),
    out_degree: z.number(),
  })
)

/* -------------------------------------------------------------------------- */
/* memory_get_clusters — graph.CommunityResult (internal/graph/community.go)    */
/* -------------------------------------------------------------------------- */

/** graph.Community — {id, label, lead_node, members[], size, dominant_type?} */
export interface Community {
  id: number
  label: string
  lead_node: string
  members: string[]
  size: number
  dominant_type?: string
}

/** graph.CommunityResult — {communities[], modularity, total_nodes, total_edges} */
export interface CommunityResult {
  communities: Community[]
  modularity: number
  total_nodes?: number
  total_edges?: number
}

export const clustersSchema: MiniSchema<CommunityResult> = z.object({
  communities: z.array(
    z.object({
      id: z.number(),
      label: z.string(),
      lead_node: z.string(),
      members: z.array(z.string()),
      size: z.number(),
      dominant_type: z.optional(z.string()),
    })
  ),
  modularity: z.number(),
  total_nodes: z.optional(z.number()),
  total_edges: z.optional(z.number()),
})

/* -------------------------------------------------------------------------- */
/* memory_get_insights — mcp.SurprisingConnection (internal/mcp/handlers.go)   */
/* -------------------------------------------------------------------------- */

/** mcp.SurprisingConnection — {source_id, source_name, target_id, target_name, similarity, reason} */
export interface SurprisingConnection {
  source_id: string
  source_name: string
  target_id: string
  target_name: string
  similarity: number
  reason: string
}

export const insightsSchema: MiniSchema<SurprisingConnection[]> = z.array(
  z.object({
    source_id: z.string(),
    source_name: z.string(),
    target_id: z.string(),
    target_name: z.string(),
    similarity: z.number(),
    reason: z.string(),
  })
)

/* -------------------------------------------------------------------------- */
/* memory_list_files — []store.VaultDocument (internal/store/vault.go)         */
/* -------------------------------------------------------------------------- */

/** store.VaultDocument — {id, path, title, mtime, size} */
export interface VaultDocument {
  id: string
  path: string
  title: string
  mtime: number
  size: number
}

export const listFilesSchema: MiniSchema<VaultDocument[]> = z.array(
  z.object({
    id: z.string(),
    path: z.string(),
    title: z.string(),
    mtime: z.number(),
    size: z.number(),
  })
)

/* -------------------------------------------------------------------------- */
/* memory_read_note — mcp.ReadNoteResult (internal/mcp/vault_handlers.go)      */
/* -------------------------------------------------------------------------- */

/** mcp.ReadNoteResult — {path, title, content} */
export interface ReadNoteResult {
  path: string
  title: string
  content: string
}

export const readNoteSchema: MiniSchema<ReadNoteResult> = z.object({
  path: z.string(),
  title: z.string(),
  content: z.string(),
})

/* -------------------------------------------------------------------------- */
/* memory_note_outline — []mcp.OutlineHeading (internal/mcp/vault_handlers.go) */
/* -------------------------------------------------------------------------- */

/** mcp.OutlineHeading — {level, text} */
export interface OutlineHeading {
  level: number
  text: string
}

export const noteOutlineSchema: MiniSchema<OutlineHeading[]> = z.array(
  z.object({
    level: z.number(),
    text: z.string(),
  })
)

/* -------------------------------------------------------------------------- */
/* memory_file_metadata — mcp.FileMetadataResult (internal/mcp/vault_handlers.go) */
/* -------------------------------------------------------------------------- */

/** mcp.FileMetadataResult — {path, title, size, mtime} */
export interface FileMetadataResult {
  path: string
  title: string
  size: number
  mtime: number
}

export const fileMetadataSchema: MiniSchema<FileMetadataResult> = z.object({
  path: z.string(),
  title: z.string(),
  size: z.number(),
  mtime: z.number(),
})

/* -------------------------------------------------------------------------- */
/* memory_get_drift — drift.DriftReport (internal/drift/drift.go)              */
/* -------------------------------------------------------------------------- */

/** drift.Severity — "LOW" | "MEDIUM" | "HIGH" | "CRITICAL" */
export type DriftSeverity = "LOW" | "MEDIUM" | "HIGH" | "CRITICAL"

/** drift.NoteDrift — {note_id, title, path, drift_score, severity, commits_behind, lines_changed, affected_files[], reason, pagerank, links?} */
export interface NoteDrift {
  note_id: string
  title: string
  path: string
  drift_score: number
  severity: DriftSeverity
  commits_behind: number
  lines_changed: number
  affected_files: string[]
  reason: string
  pagerank: number
  links?: { obsidian?: string; vscode?: string }
}

/** drift.UncoveredCode — {file_path, package, status, additions, deletions, suggested_action} */
export interface UncoveredCode {
  file_path: string
  package: string
  status: string
  additions: number
  deletions: number
  suggested_action: string
}

/**
 * drift.DriftReport
 * {range, total_commits, total_changed_files, drifted_notes[], uncovered_code[],
 *  overall_score, critical_count, high_count, medium_count, low_count, analyzed_at}
 */
export interface DriftReport {
  range: string
  total_commits: number
  total_changed_files: number
  drifted_notes: NoteDrift[]
  uncovered_code: UncoveredCode[]
  overall_score: number
  critical_count: number
  high_count: number
  medium_count: number
  low_count: number
  analyzed_at?: string
}

export const driftReportSchema: MiniSchema<DriftReport> = z.object({
  range: z.string(),
  total_commits: z.number(),
  total_changed_files: z.number(),
  drifted_notes: z.array(
    z.object({
      note_id: z.string(),
      title: z.string(),
      path: z.string(),
      drift_score: z.number(),
      severity: z.enum(["LOW", "MEDIUM", "HIGH", "CRITICAL"]),
      commits_behind: z.number(),
      lines_changed: z.number(),
      affected_files: z.array(z.string()),
      reason: z.string(),
      pagerank: z.number(),
      links: z.optional(
        z.object({
          obsidian: z.optional(z.string()),
          vscode: z.optional(z.string()),
        })
      ),
    })
  ),
  uncovered_code: z.array(
    z.object({
      file_path: z.string(),
      package: z.string(),
      status: z.string(),
      additions: z.number(),
      deletions: z.number(),
      suggested_action: z.string(),
    })
  ),
  overall_score: z.number(),
  critical_count: z.number(),
  high_count: z.number(),
  medium_count: z.number(),
  low_count: z.number(),
  analyzed_at: z.optional(z.string()),
})

/* -------------------------------------------------------------------------- */
/* memory_get_impact — graph.ImpactResult (internal/graph/impact.go)           */
/* -------------------------------------------------------------------------- */

/** graph.ImpactedNode — {id, depth, relation, via_node, severity, pagerank?, node_type?, community_id?, community_tag?} */
export interface ImpactedNode {
  id: string
  depth: number
  relation: string
  via_node: string
  severity: string
  pagerank?: number
  node_type?: string
  community_id?: number
  community_tag?: string
}

/**
 * graph.ImpactResult
 * {target_node, total_impacted, max_depth_reached, risk_score, risk_level,
 *  direct_dependents, indirect_dependents, severity_counts{}, affected_clusters[], nodes[]}
 */
export interface ImpactResult {
  target_node: string
  total_impacted: number
  max_depth_reached: number
  risk_score: number
  risk_level: string
  direct_dependents: number
  indirect_dependents: number
  severity_counts?: Record<string, number>
  affected_clusters?: number[]
  nodes: ImpactedNode[]
}

export const impactSchema: MiniSchema<ImpactResult> = z.object({
  target_node: z.string(),
  total_impacted: z.number(),
  max_depth_reached: z.number(),
  risk_score: z.number(),
  risk_level: z.string(),
  direct_dependents: z.number(),
  indirect_dependents: z.number(),
  severity_counts: z.optional(z.record(z.string(), z.number())),
  affected_clusters: z.optional(z.array(z.number())),
  nodes: z.array(
    z.object({
      id: z.string(),
      depth: z.number(),
      relation: z.string(),
      via_node: z.string(),
      severity: z.string(),
      pagerank: z.optional(z.number()),
      node_type: z.optional(z.string()),
      community_id: z.optional(z.number()),
      community_tag: z.optional(z.string()),
    })
  ),
})

/* -------------------------------------------------------------------------- */
/* memory_find_path — graph.PathResult (internal/graph/path.go)                */
/* -------------------------------------------------------------------------- */

/** graph.PathEdge — {from, to, relation, epistemic_status, weight, cost, direction} */
export interface PathEdge {
  from: string
  to: string
  relation: string
  epistemic_status: string
  weight: number
  cost: number
  direction: string
}

/**
 * graph.PathResult
 * {found, source, target, directed, cost_mode, hops, total_cost, nodes[], edges[], summary}
 */
export interface PathResult {
  found: boolean
  source: string
  target: string
  directed: boolean
  cost_mode: string
  hops: number
  total_cost: number
  nodes: string[]
  edges: PathEdge[]
  summary: string
}

export const pathSchema: MiniSchema<PathResult> = z.object({
  found: z.boolean(),
  source: z.string(),
  target: z.string(),
  directed: z.boolean(),
  cost_mode: z.string(),
  hops: z.number(),
  total_cost: z.number(),
  nodes: z.array(z.string()),
  edges: z.array(
    z.object({
      from: z.string(),
      to: z.string(),
      relation: z.string(),
      epistemic_status: z.string(),
      weight: z.number(),
      cost: z.number(),
      direction: z.string(),
    })
  ),
  summary: z.string(),
})

/* -------------------------------------------------------------------------- */
/* memory_visualize_graph — mcp.VisualizeGraphResult (internal/mcp/visualizer_handlers.go) */
/* -------------------------------------------------------------------------- */

/**
 * mcp.VisualizeGraphResult
 * {url?, path, total_nodes, total_edges, hub_count, density}
 *
 * `url` só existe quando o transporte HTTP está ativo (ADR-059 D-5).
 */
export interface VisualizeGraphResult {
  url?: string
  path: string
  total_nodes: number
  total_edges: number
  hub_count: number
  density: number
}

export const visualizeGraphSchema: MiniSchema<VisualizeGraphResult> = z.object({
  url: z.optional(z.string()),
  path: z.string(),
  total_nodes: z.number(),
  total_edges: z.number(),
  hub_count: z.number(),
  density: z.number(),
})

/* -------------------------------------------------------------------------- */
/* memory_code_search — mcp.CodeSymbolResult (internal/mcp/code_handlers.go)   */
/* -------------------------------------------------------------------------- */

/**
 * mcp.CodeSymbolResult
 * {symbol_id, qualified_name, name, kind, language, file, start_line, end_line,
 *  signature?, doc_comment?, score, boosted}
 *
 * `signature` e `doc_comment` são `omitempty` no Go: somem quando vazios.
 */
export interface CodeSymbol {
  symbol_id: number
  qualified_name: string
  name: string
  kind: string
  language: string
  file: string
  start_line: number
  end_line: number
  signature?: string
  doc_comment?: string
  score: number
  boosted: boolean
}

/**
 * mcp.MemoryCodeSearchResponse — {query, count, results[]}
 *
 * `results` é uma lista no contrato, mas o Go serializa slice nil como
 * `null` (não `[]`); o schema tolera as duas formas.
 */
export interface CodeSearchResponse {
  query: string
  count: number
  results: CodeSymbol[]
}

const codeSymbolSchema = z.object({
  symbol_id: z.number(),
  qualified_name: z.string(),
  name: z.string(),
  kind: z.string(),
  language: z.string(),
  file: z.string(),
  start_line: z.number(),
  end_line: z.number(),
  signature: z.optional(z.string()),
  doc_comment: z.optional(z.string()),
  score: z.number(),
  boosted: z.boolean(),
})

/** Lista de símbolos que tolera o `null` do Go e devolve sempre uma lista. */
const codeSymbolListSchema = z.pipe(
  z.nullish(z.array(codeSymbolSchema)),
  z.transform<CodeSymbol[] | null | undefined, CodeSymbol[]>(
    (value) => value ?? []
  )
)

export const codeSearchSchema: MiniSchema<CodeSearchResponse> = z.object({
  query: z.string(),
  count: z.number(),
  results: codeSymbolListSchema,
})

/* -------------------------------------------------------------------------- */
/* memory_code_neighbors — mcp.CodeNeighborResult (internal/mcp/code_handlers.go) */
/* -------------------------------------------------------------------------- */

/**
 * mcp.CodeNeighborResult
 * {direction, kind, symbol, file, line, hop}
 *
 * `kind` é o **tipo da aresta** (`calls`, `imports`, `inherits`, `uses_type`,
 * `references`) e `file` + `line` são o **call-site** da aresta.
 */
export interface CodeNeighbor {
  direction: string
  kind: string
  symbol: string
  file: string
  line: number
  hop: number
}

/**
 * mcp.MemoryCodeNeighborsResponse
 * {root?, neighbors[], depth, direction}
 *
 * `root` é `omitempty`: some quando o símbolo não existe.
 */
export interface CodeNeighborsResponse {
  root?: CodeSymbol
  neighbors: CodeNeighbor[]
  depth: number
  direction: string
}

const codeNeighborSchema = z.object({
  direction: z.string(),
  kind: z.string(),
  symbol: z.string(),
  file: z.string(),
  line: z.number(),
  hop: z.number(),
})

/** Lista de vizinhos que tolera o `null` do Go e devolve sempre uma lista. */
const codeNeighborListSchema = z.pipe(
  z.nullish(z.array(codeNeighborSchema)),
  z.transform<CodeNeighbor[] | null | undefined, CodeNeighbor[]>(
    (value) => value ?? []
  )
)

export const codeNeighborsSchema: MiniSchema<CodeNeighborsResponse> = z.object({
  root: z.optional(codeSymbolSchema),
  neighbors: codeNeighborListSchema,
  depth: z.number(),
  direction: z.string(),
})
