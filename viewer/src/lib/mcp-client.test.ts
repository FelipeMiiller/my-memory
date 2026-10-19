import { afterEach, describe, expect, it, vi } from "vitest"
import { McpError, callTool, getHealth, mcpUrl } from "./mcp-client"
import { doctorReportSchema } from "./schemas"

const RELATORIO_VALIDO = {
  total_documents: 12,
  total_chunks: 340,
  total_edges: 87,
  total_nodes: 12,
  health_score: 92,
  missing_mtime_count: 0,
  dead_links: [],
  orphan_notes: [],
  self_loops: [],
  desynced_chunks: [],
}

function respostaJson(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.unstubAllEnvs()
  vi.restoreAllMocks()
})

describe("cliente MCP", () => {
  it("envia uma requisição JSON-RPC 2.0 no formato do MCP", async () => {
    const fetchMock = vi.fn(async (_url: string, _init?: RequestInit) =>
      respostaJson({
        jsonrpc: "2.0",
        id: 1,
        result: { content: [{ type: "text", text: "ok" }], isError: false },
      })
    )
    vi.stubGlobal("fetch", fetchMock)

    await callTool(
      "memory_doctor",
      { repository: "central" },
      doctorReportSchema
    )

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe("/mcp")
    expect(init.method).toBe("POST")
    expect((init.headers as Record<string, string>)["Content-Type"]).toBe(
      "application/json"
    )

    const payload = JSON.parse(init.body as string)
    expect(payload.jsonrpc).toBe("2.0")
    expect(payload.method).toBe("tools/call")
    expect(payload.params.name).toBe("memory_doctor")
    expect(payload.params.arguments).toEqual({ repository: "central" })
    expect(typeof payload.id).toBe("number")
  })

  it("usa VITE_MCP_BASE_URL quando definido", async () => {
    vi.stubEnv("VITE_MCP_BASE_URL", "http://127.0.0.1:38400/")
    const fetchMock = vi.fn(async (_url: string, _init?: RequestInit) =>
      respostaJson({
        jsonrpc: "2.0",
        id: 1,
        result: { content: [{ type: "text", text: "ok" }] },
      })
    )
    vi.stubGlobal("fetch", fetchMock)

    await callTool("memory_doctor", {}, doctorReportSchema)

    expect(fetchMock.mock.calls[0]?.[0]).toBe("http://127.0.0.1:38400/mcp")
    expect(mcpUrl("/health")).toBe("http://127.0.0.1:38400/health")
  })

  it("devolve o texto do content", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        respostaJson({
          jsonrpc: "2.0",
          id: 1,
          result: {
            content: [
              { type: "text", text: "primeiro" },
              { type: "text", text: "segundo" },
            ],
            isError: false,
          },
        })
      )
    )

    const result = await callTool("memory_doctor", {}, doctorReportSchema)

    expect(result.text).toBe("primeiro\nsegundo")
    expect(result.hasStructured).toBe(false)
    expect(result.structured).toBeNull()
    expect(result.validationError).toBeNull()
  })

  it("devolve o structuredContent quando ele existe e valida", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        respostaJson({
          jsonrpc: "2.0",
          id: 1,
          result: {
            content: [{ type: "text", text: "resumo" }],
            isError: false,
            structuredContent: RELATORIO_VALIDO,
          },
        })
      )
    )

    const result = await callTool("memory_doctor", {}, doctorReportSchema)

    expect(result.hasStructured).toBe(true)
    expect(result.structured?.health_score).toBe(92)
    expect(result.text).toBe("resumo")
  })

  it("não quebra quando o structuredContent está ausente (MCP antigo)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        respostaJson({
          jsonrpc: "2.0",
          id: 1,
          result: { content: [{ type: "text", text: "sem estruturado" }] },
        })
      )
    )

    const result = await callTool("memory_doctor", {}, doctorReportSchema)

    expect(result.hasStructured).toBe(false)
    expect(result.structured).toBeNull()
    expect(result.text).toBe("sem estruturado")
  })

  it("não quebra quando o structuredContent é inválido", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {})

    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        respostaJson({
          jsonrpc: "2.0",
          id: 1,
          result: {
            content: [{ type: "text", text: "texto cru" }],
            structuredContent: { health_score: "muito alto" },
          },
        })
      )
    )

    const result = await callTool("memory_doctor", {}, doctorReportSchema)

    expect(result.hasStructured).toBe(false)
    expect(result.structured).toBeNull()
    expect(result.text).toBe("texto cru")
    expect(result.validationError).toBeTruthy()
    expect(warn).toHaveBeenCalled()
  })

  it("lança quando a resposta traz erro JSON-RPC", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        respostaJson({
          jsonrpc: "2.0",
          id: 1,
          error: { code: -32601, message: "Ferramenta não encontrada" },
        })
      )
    )

    await expect(
      callTool("memory_inexistente", {}, doctorReportSchema)
    ).rejects.toBeInstanceOf(McpError)
  })

  it("lança quando a tool responde isError", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        respostaJson({
          jsonrpc: "2.0",
          id: 1,
          result: {
            content: [{ type: "text", text: "Falha ao ler o vault" }],
            isError: true,
          },
        })
      )
    )

    await expect(
      callTool("memory_doctor", {}, doctorReportSchema)
    ).rejects.toThrow("Falha ao ler o vault")
  })

  it("lança quando o MCP está fora do ar", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("fetch failed")
      })
    )

    await expect(
      callTool("memory_doctor", {}, doctorReportSchema)
    ).rejects.toBeInstanceOf(McpError)
  })
})

describe("health do MCP", () => {
  it("valida o payload de /health", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        respostaJson({
          status: "healthy",
          name: "my-memory",
          version: "1.4.0",
          protocol_version: "2025-06-18",
          uptime_seconds: 42.5,
          default_repo: "central",
          tools_count: 24,
          active_sessions: 1,
        })
      )
    )

    const health = await getHealth()

    expect(health.status).toBe("healthy")
    expect(health.tools_count).toBe(24)
  })

  it("rejeita um payload de /health inválido", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => respostaJson({ status: "healthy" }))
    )

    await expect(getHealth()).rejects.toBeInstanceOf(McpError)
  })
})
