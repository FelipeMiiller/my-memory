import { describe, expect, it } from "vitest"
import {
  clustersSchema,
  doctorReportSchema,
  driftReportSchema,
  fileMetadataSchema,
  healthSchema,
  hubsDegreeSchema,
  hubsPagerankSchema,
  impactSchema,
  insightsSchema,
  listFilesSchema,
  noteOutlineSchema,
  pathSchema,
  readNoteSchema,
  visualizeGraphSchema,
} from "./schemas"

/**
 * Cada schema tem que **validar** o payload que o Go devolve e **rejeitar**
 * um payload quebrado. A rejeição é o que protege a tela: sem ela, um
 * `structuredContent` com shape errado entraria no painel e quebraria a
 * renderização.
 */
describe("schemas zod do contrato MCP", () => {
  const doctorValido = {
    total_documents: 12,
    total_chunks: 340,
    total_edges: 87,
    total_nodes: 12,
    health_score: 92,
    missing_mtime_count: 0,
    dead_links: [{ source_id: "a", target_id: "b", relation: "links_to" }],
    orphan_notes: [{ id: "c", title: "Nota solta" }],
    self_loops: [],
    desynced_chunks: [],
  }

  const clustersValido = {
    communities: [
      {
        id: 0,
        label: "comunidade 0",
        lead_node: "hub",
        members: ["hub", "a"],
        size: 2,
        dominant_type: "note",
      },
    ],
    modularity: 0.4231,
    total_nodes: 12,
    total_edges: 87,
  }

  const driftValido = {
    range: "HEAD~5..HEAD",
    total_commits: 5,
    total_changed_files: 3,
    drifted_notes: [
      {
        note_id: "n1",
        title: "ADR antigo",
        path: "docs/adr/001.md",
        drift_score: 80,
        severity: "HIGH",
        commits_behind: 5,
        lines_changed: 42,
        affected_files: ["internal/x.go"],
        reason: "o código mudou",
        pagerank: 0.1,
      },
    ],
    uncovered_code: [
      {
        file_path: "internal/y.go",
        package: "internal",
        status: "added",
        additions: 10,
        deletions: 0,
        suggested_action: "criar nota",
      },
    ],
    overall_score: 72.5,
    critical_count: 0,
    high_count: 1,
    medium_count: 0,
    low_count: 0,
    analyzed_at: "2026-10-19T12:00:00Z",
  }

  const impactValido = {
    target_node: "ADR-060",
    total_impacted: 2,
    max_depth_reached: 2,
    risk_score: 30.5,
    risk_level: "BAIXO",
    direct_dependents: 1,
    indirect_dependents: 1,
    severity_counts: { HIGH: 1, LOW: 1 },
    affected_clusters: [0],
    nodes: [
      {
        id: "a",
        depth: 1,
        relation: "implements",
        via_node: "ADR-060",
        severity: "HIGH",
      },
    ],
  }

  const pathValido = {
    found: true,
    source: "A",
    target: "B",
    directed: true,
    cost_mode: "epistemic",
    hops: 2,
    total_cost: 1.5,
    nodes: ["A", "X", "B"],
    edges: [
      {
        from: "A",
        to: "X",
        relation: "links_to",
        epistemic_status: "EXTRACTED",
        weight: 1,
        cost: 1,
        direction: "forward",
      },
    ],
    summary: "caminho encontrado",
  }

  const casos: Array<{
    nome: string
    schema: { safeParse: (data: unknown) => { success: boolean } }
    valido: unknown
    invalido: unknown
    invalidoDescricao: string
  }> = [
    {
      nome: "health",
      schema: healthSchema,
      valido: {
        status: "healthy",
        name: "my-memory",
        version: "1.4.0",
        protocol_version: "2025-06-18",
        uptime_seconds: 12.5,
        default_repo: "central",
        tools_count: 24,
        active_sessions: 0,
      },
      invalido: { status: "healthy", name: "my-memory" },
      invalidoDescricao: "faltam campos obrigatórios",
    },
    {
      nome: "memory_doctor",
      schema: doctorReportSchema,
      valido: doctorValido,
      invalido: { ...doctorValido, health_score: "alta" },
      invalidoDescricao: "health_score com o tipo errado",
    },
    {
      nome: "memory_get_hubs (degree)",
      schema: hubsDegreeSchema,
      valido: [
        {
          id: "a",
          name: "hub",
          in_degree: 3,
          out_degree: 2,
          total_degree: 5,
        },
      ],
      invalido: [{ id: "a", name: "hub", in_degree: "3" }],
      invalidoDescricao: "in_degree com o tipo errado",
    },
    {
      nome: "memory_get_hubs (pagerank)",
      schema: hubsPagerankSchema,
      valido: [
        {
          id: "a",
          name: "hub",
          score: 0.12,
          rank: 1,
          in_degree: 3,
          out_degree: 2,
        },
      ],
      invalido: [{ id: "a", name: "hub", score: 0.12, rank: 1 }],
      invalidoDescricao: "faltam in_degree e out_degree",
    },
    {
      nome: "memory_get_clusters",
      schema: clustersSchema,
      valido: clustersValido,
      invalido: { ...clustersValido, communities: [{ id: "x" }] },
      invalidoDescricao: "comunidade com shape errado",
    },
    {
      nome: "memory_get_insights",
      schema: insightsSchema,
      valido: [
        {
          source_id: "a",
          source_name: "A",
          target_id: "b",
          target_name: "B",
          similarity: 0.82,
          reason: "semelhança semântica",
        },
      ],
      invalido: [{ source_id: "a" }],
      invalidoDescricao: "faltam campos obrigatórios",
    },
    {
      nome: "memory_list_files",
      schema: listFilesSchema,
      valido: [
        {
          id: "1",
          path: "docs/a.md",
          title: "A",
          mtime: 1700000000,
          size: 120,
        },
      ],
      invalido: [{ id: "1", path: "docs/a.md" }],
      invalidoDescricao: "faltam title, mtime e size",
    },
    {
      nome: "memory_read_note",
      schema: readNoteSchema,
      valido: { path: "docs/a.md", title: "A", content: "# A" },
      invalido: { path: "docs/a.md", title: "A" },
      invalidoDescricao: "falta o content",
    },
    {
      nome: "memory_note_outline",
      schema: noteOutlineSchema,
      valido: [{ level: 1, text: "A" }],
      invalido: [{ level: "1", text: "A" }],
      invalidoDescricao: "level com o tipo errado",
    },
    {
      nome: "memory_file_metadata",
      schema: fileMetadataSchema,
      valido: { path: "docs/a.md", title: "A", size: 120, mtime: 1700000000 },
      invalido: { path: "docs/a.md", title: "A", size: 120 },
      invalidoDescricao: "falta o mtime",
    },
    {
      nome: "memory_get_drift",
      schema: driftReportSchema,
      valido: driftValido,
      invalido: { ...driftValido, drifted_notes: [{ note_id: "n1" }] },
      invalidoDescricao: "nota com drift com shape errado",
    },
    {
      nome: "memory_get_impact",
      schema: impactSchema,
      valido: impactValido,
      invalido: { ...impactValido, nodes: [{ id: "a" }] },
      invalidoDescricao: "nó impactado com shape errado",
    },
    {
      nome: "memory_find_path",
      schema: pathSchema,
      valido: pathValido,
      invalido: { ...pathValido, edges: [{ from: "A" }] },
      invalidoDescricao: "aresta com shape errado",
    },
    {
      nome: "memory_visualize_graph",
      schema: visualizeGraphSchema,
      valido: {
        url: "http://127.0.0.1:38400/graph/graph.html",
        path: "C:/vault/.mymemory/www/graph/graph.html",
        total_nodes: 12,
        total_edges: 87,
        hub_count: 3,
        density: 0.11,
      },
      invalido: { path: "x", total_nodes: 1 },
      invalidoDescricao: "faltam campos obrigatórios",
    },
  ]

  for (const caso of casos) {
    it(`${caso.nome} aceita o payload válido`, () => {
      const resultado = caso.schema.safeParse(caso.valido)
      expect(
        resultado.success,
        `payload válido de ${caso.nome} foi rejeitado`
      ).toBe(true)
    })

    it(`${caso.nome} rejeita o payload inválido (${caso.invalidoDescricao})`, () => {
      const resultado = caso.schema.safeParse(caso.invalido)
      expect(
        resultado.success,
        `payload inválido de ${caso.nome} foi aceito`
      ).toBe(false)
    })
  }

  it("memory_visualize_graph aceita payload sem url (transporte sem HTTP)", () => {
    const resultado = visualizeGraphSchema.safeParse({
      path: "C:/vault/graph.html",
      total_nodes: 1,
      total_edges: 0,
      hub_count: 0,
      density: 0,
    })

    expect(resultado.success).toBe(true)
  })
})
