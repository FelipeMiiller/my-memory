import { create } from "zustand"

/**
 * Estado de **cliente** da viewer (ADR-060).
 *
 * Aqui mora só o que o usuário escolheu: a nota selecionada, os filtros da
 * listagem e os dois nós do caminho. O dado do servidor vive no cache do
 * TanStack Query — misturar os dois é o que produz cache inconsistente.
 */
export interface ViewerState {
  /** Path do vault da nota selecionada. Vazio = nenhuma nota selecionada. */
  selectedPath: string
  /** Filtro de extensão ("" = todas). */
  extFilter: string
  /** Filtro de pasta ("" = todas). */
  folderFilter: string
  /** Nó de origem do caminho entre dois nós. */
  pathSource: string
  /** Nó de destino do caminho entre dois nós. */
  pathTarget: string
  /** Nó alvo da análise de impacto (blast radius). */
  impactNode: string

  selectPath: (path: string) => void
  setExtFilter: (ext: string) => void
  setFolderFilter: (folder: string) => void
  setPathSource: (node: string) => void
  setPathTarget: (node: string) => void
  setImpactNode: (node: string) => void
  clearPathSearch: () => void
}

export const useViewerStore = create<ViewerState>((set) => ({
  selectedPath: "",
  extFilter: "",
  folderFilter: "",
  pathSource: "",
  pathTarget: "",
  impactNode: "",

  selectPath: (path) => set({ selectedPath: path }),
  setExtFilter: (ext) => set({ extFilter: ext }),
  setFolderFilter: (folder) => set({ folderFilter: folder }),
  setPathSource: (node) => set({ pathSource: node }),
  setPathTarget: (node) => set({ pathTarget: node }),
  setImpactNode: (node) => set({ impactNode: node }),
  clearPathSearch: () => set({ pathSource: "", pathTarget: "" }),
}))
