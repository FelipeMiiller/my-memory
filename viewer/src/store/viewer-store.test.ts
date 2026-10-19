import { beforeEach, describe, expect, it } from "vitest"
import { useViewerStore } from "./viewer-store"

describe("store Zustand da viewer", () => {
  beforeEach(() => {
    useViewerStore.setState({
      selectedPath: "",
      extFilter: "",
      folderFilter: "",
      pathSource: "",
      pathTarget: "",
      impactNode: "",
    })
  })

  it("começa vazio", () => {
    const state = useViewerStore.getState()

    expect(state.selectedPath).toBe("")
    expect(state.extFilter).toBe("")
    expect(state.folderFilter).toBe("")
    expect(state.pathSource).toBe("")
    expect(state.pathTarget).toBe("")
    expect(state.impactNode).toBe("")
  })

  it("seleciona a nota", () => {
    useViewerStore.getState().selectPath("docs/adr/060-stack.md")

    expect(useViewerStore.getState().selectedPath).toBe("docs/adr/060-stack.md")
  })

  it("muda o filtro de extensão", () => {
    useViewerStore.getState().setExtFilter("md")

    expect(useViewerStore.getState().extFilter).toBe("md")
  })

  it("muda o filtro de pasta", () => {
    useViewerStore.getState().setFolderFilter("docs/adr")

    expect(useViewerStore.getState().folderFilter).toBe("docs/adr")
  })

  it("muda os nós de origem e destino do caminho", () => {
    useViewerStore.getState().setPathSource("A")
    useViewerStore.getState().setPathTarget("B")

    expect(useViewerStore.getState().pathSource).toBe("A")
    expect(useViewerStore.getState().pathTarget).toBe("B")
  })

  it("muda o nó alvo do impacto", () => {
    useViewerStore.getState().setImpactNode("ADR-060")

    expect(useViewerStore.getState().impactNode).toBe("ADR-060")
  })

  it("limpa a busca de caminho sem tocar nos demais campos", () => {
    useViewerStore.getState().setPathSource("A")
    useViewerStore.getState().setPathTarget("B")
    useViewerStore.getState().selectPath("docs/a.md")

    useViewerStore.getState().clearPathSearch()

    const state = useViewerStore.getState()
    expect(state.pathSource).toBe("")
    expect(state.pathTarget).toBe("")
    expect(state.selectedPath).toBe("docs/a.md")
  })
})
