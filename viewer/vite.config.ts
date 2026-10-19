import { resolve } from "node:path"
import tailwindcss from "@tailwindcss/vite"
import { tanstackRouter } from "@tanstack/router-plugin/vite"
import react from "@vitejs/plugin-react"
import { defineConfig, loadEnv } from "vite"

/**
 * Base URL do MCP usada pelo proxy do dev server.
 *
 * A porta nunca é hardcoded no código da aplicação: quem decide é a variável
 * de ambiente `VITE_MCP_BASE_URL`. O default aponta para o servidor local do
 * MyMemory (`mem mcp --http`).
 */
const DEFAULT_MCP_BASE_URL = "http://127.0.0.1:38400"

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "VITE_")
  const mcpBaseUrl = env.VITE_MCP_BASE_URL || DEFAULT_MCP_BASE_URL

  return {
    plugins: [
      tanstackRouter({
        target: "react",
        routesDirectory: "./src/routes",
        // A árvore de rotas gerada fica versionada, junto das rotas.
        generatedRouteTree: "./src/routes/routeTree.gen.ts",
        // ...e é ignorada pela varredura de arquivos de rota, para não virar
        // uma rota fantasma.
        routeFileIgnorePattern: "routeTree\\.gen\\.ts",
        autoCodeSplitting: false,
      }),
      react(),
      tailwindcss(),
    ],
    resolve: {
      alias: {
        "@": resolve(import.meta.dirname, "./src"),
      },
    },
    server: {
      // Em desenvolvimento a SPA roda em outra porta que o MCP: o proxy
      // encaminha /mcp e /health para o servidor real.
      proxy: {
        "/mcp": { target: mcpBaseUrl, changeOrigin: true },
        "/health": { target: mcpBaseUrl, changeOrigin: true },
      },
    },
    preview: {
      // O mesmo encaminhamento vale para `npm run preview`, que serve o build
      // de produção.
      proxy: {
        "/mcp": { target: mcpBaseUrl, changeOrigin: true },
        "/health": { target: mcpBaseUrl, changeOrigin: true },
      },
    },
  }
})
