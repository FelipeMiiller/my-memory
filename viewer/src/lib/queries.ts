import { useMutation, useQuery } from "@tanstack/react-query"
import { callTool, getHealth } from "./mcp-client"
import type {
  CodeNeighborsResponse,
  CodeSearchResponse,
  CommunityResult,
  DoctorReport,
  DriftReport,
  FileMetadataResult,
  HealthPayload,
  HubNode,
  ImpactResult,
  OutlineHeading,
  PageRankNode,
  PathResult,
  ReadNoteResult,
  SurprisingConnection,
  VaultDocument,
  VisualizeGraphResult,
} from "./schemas"
import {
  clustersSchema,
  codeNeighborsSchema,
  codeSearchSchema,
  doctorReportSchema,
  driftReportSchema,
  fileMetadataSchema,
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
 * Hooks de leitura do MCP.
 *
 * Todo dado do servidor vive no cache do TanStack Query. Cada hook devolve o
 * `McpToolResult` completo, porque a tela precisa saber se o que chegou foi o
 * `structuredContent` validado ou o texto cru (fallback).
 */

export const queryKeys = {
  health: ["health"] as const,
  doctor: ["doctor"] as const,
  hubs: (algorithm: string) => ["hubs", algorithm] as const,
  clusters: ["clusters"] as const,
  insights: ["insights"] as const,
  files: (ext: string, folder: string) => ["files", ext, folder] as const,
  note: (path: string) => ["note", path] as const,
  outline: (path: string) => ["outline", path] as const,
  metadata: (path: string) => ["metadata", path] as const,
  drift: ["drift"] as const,
  impact: (node: string) => ["impact", node] as const,
  path: (source: string, target: string) => ["path", source, target] as const,
  graph: ["graph"] as const,
  codeSearch: (query: string, limit: number) =>
    ["code", "search", query, limit] as const,
  codeNeighbors: (symbol: string, depth: number, direction: string) =>
    ["code", "neighbors", symbol, depth, direction] as const,
}

export function useHealth() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: (): Promise<HealthPayload> => getHealth(),
    staleTime: 15_000,
    retry: 1,
  })
}

export function useDoctor() {
  return useQuery({
    queryKey: queryKeys.doctor,
    queryFn: () =>
      callTool<DoctorReport>("memory_doctor", {}, doctorReportSchema),
    staleTime: 30_000,
  })
}

export function useDegreeHubs() {
  return useQuery({
    queryKey: queryKeys.hubs("degree"),
    queryFn: () =>
      callTool<HubNode[]>(
        "memory_get_hubs",
        { algorithm: "degree" },
        hubsDegreeSchema
      ),
    staleTime: 30_000,
  })
}

export function usePagerankHubs() {
  return useQuery({
    queryKey: queryKeys.hubs("pagerank"),
    queryFn: () =>
      callTool<PageRankNode[]>(
        "memory_get_hubs",
        { algorithm: "pagerank" },
        hubsPagerankSchema
      ),
    staleTime: 30_000,
  })
}

export function useClusters() {
  return useQuery({
    queryKey: queryKeys.clusters,
    queryFn: () =>
      callTool<CommunityResult>("memory_get_clusters", {}, clustersSchema),
    staleTime: 30_000,
  })
}

export function useInsights() {
  return useQuery({
    queryKey: queryKeys.insights,
    queryFn: () =>
      callTool<SurprisingConnection[]>(
        "memory_get_insights",
        {},
        insightsSchema
      ),
    staleTime: 60_000,
  })
}

export function useVaultFiles(ext: string, folder: string) {
  return useQuery({
    queryKey: queryKeys.files(ext, folder),
    queryFn: () =>
      callTool<VaultDocument[]>(
        "memory_list_files",
        { ext, folder },
        listFilesSchema
      ),
    staleTime: 30_000,
  })
}

export function useNoteContent(path: string) {
  return useQuery({
    queryKey: queryKeys.note(path),
    queryFn: () =>
      callTool<ReadNoteResult>("memory_read_note", { path }, readNoteSchema),
    enabled: path !== "",
    staleTime: 30_000,
  })
}

export function useNoteOutline(path: string) {
  return useQuery({
    queryKey: queryKeys.outline(path),
    queryFn: () =>
      callTool<OutlineHeading[]>(
        "memory_note_outline",
        { path },
        noteOutlineSchema
      ),
    enabled: path !== "",
    staleTime: 30_000,
  })
}

export function useFileMetadata(path: string) {
  return useQuery({
    queryKey: queryKeys.metadata(path),
    queryFn: () =>
      callTool<FileMetadataResult>(
        "memory_file_metadata",
        { path },
        fileMetadataSchema
      ),
    enabled: path !== "",
    staleTime: 30_000,
  })
}

export function useDrift() {
  return useQuery({
    queryKey: queryKeys.drift,
    queryFn: () =>
      callTool<DriftReport>("memory_get_drift", {}, driftReportSchema),
    staleTime: 60_000,
  })
}

export function useImpact(node: string) {
  return useQuery({
    queryKey: queryKeys.impact(node),
    queryFn: () =>
      callTool<ImpactResult>(
        "memory_get_impact",
        { node_id: node },
        impactSchema
      ),
    enabled: node !== "",
    staleTime: 30_000,
  })
}

export function useFindPath(source: string, target: string) {
  return useQuery({
    queryKey: queryKeys.path(source, target),
    queryFn: () =>
      callTool<PathResult>("memory_find_path", { source, target }, pathSchema),
    enabled: source !== "" && target !== "",
    staleTime: 30_000,
  })
}

/**
 * Gerar o artefato do grafo é uma **ação**, não uma leitura: a tool escreve o
 * HTML no diretório de artefatos servidos (ADR-059 D-5). Por isso é mutation.
 */
export function useVisualizeGraph() {
  return useMutation({
    mutationKey: queryKeys.graph,
    mutationFn: () =>
      callTool<VisualizeGraphResult>(
        "memory_visualize_graph",
        {},
        visualizeGraphSchema
      ),
  })
}

/**
 * Busca estrutural de símbolos do code graph (ADR-061).
 *
 * `memory_code_search` exige `query` — por isso a query só roda com texto
 * não-vazio. Os filtros de tipo e de linguagem são aplicados no cliente
 * (`filterSymbols`), para que a lista não dependa do servidor ter indexado o
 * repositório inteiro.
 */
export function useCodeSearch(query: string, limit: number) {
  return useQuery({
    queryKey: queryKeys.codeSearch(query, limit),
    queryFn: () =>
      callTool<CodeSearchResponse>(
        "memory_code_search",
        { query, limit },
        codeSearchSchema
      ),
    enabled: query.trim() !== "",
    staleTime: 30_000,
  })
}

/**
 * Vizinhos estruturais de um símbolo: o tipo da aresta e o call-site.
 *
 * A tool devolve hoje o JSON no texto (sem `structuredContent`); a
 * normalização para os dois formatos vive em `src/lib/code-view.ts`.
 */
export function useCodeNeighbors(
  symbol: string,
  depth: number,
  direction: string
) {
  return useQuery({
    queryKey: queryKeys.codeNeighbors(symbol, depth, direction),
    queryFn: () =>
      callTool<CodeNeighborsResponse>(
        "memory_code_neighbors",
        { symbol, depth, direction },
        codeNeighborsSchema
      ),
    enabled: symbol.trim() !== "",
    staleTime: 30_000,
  })
}
