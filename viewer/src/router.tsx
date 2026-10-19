import { createRouter } from "@tanstack/react-router"
import type { ErrorComponentProps } from "@tanstack/react-router"
import { describeUnknownError } from "./lib/errors"
import { routeTree } from "./routes/routeTree.gen"

/**
 * Roteador da viewer.
 *
 * A árvore é gerada por `@tanstack/router-plugin` a partir de `src/routes/`
 * e **versionada** (`src/routes/routeTree.gen.ts`): um clone do repositório
 * não pode depender de rodar o dev server para ter as rotas (ADR-060 D-5).
 */
export const router = createRouter({
  routeTree,
  defaultPreload: "intent",
  defaultErrorComponent: RootFallbackError,
})

function RootFallbackError({ error }: ErrorComponentProps) {
  return (
    <div className="p-6">
      <p className="text-sm font-semibold text-destructive">Erro na rota</p>
      <p className="mt-1 text-sm">{describeUnknownError(error)}</p>
    </div>
  )
}

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}
