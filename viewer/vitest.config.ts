import { resolve } from "node:path"
import { defineConfig } from "vitest/config"

/**
 * Configuração do vitest.
 *
 * Os testes da viewer são de unidade e de contrato (parser de wikilinks,
 * schemas, cliente MCP, store) — nada precisa de DOM, então o ambiente é
 * `node`. O config é separado do `vite.config.ts` para que o plugin de
 * geração de rotas não rode durante a suíte.
 */
export default defineConfig({
  resolve: {
    alias: {
      "@": resolve(import.meta.dirname, "./src"),
    },
  },
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
  },
})
