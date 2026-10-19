import { lazy, Suspense } from "react"

/**
 * Devtools do TanStack — só em desenvolvimento.
 *
 * O import é dinâmico de propósito: as duas ferramentas são pesadas, e um
 * import estático colocaria elas no bundle de produção. Com o carregamento
 * tardio, elas viram um chunk separado que o navegador nunca pede em produção
 * (ADR-060: o bundle de produção fica perto de 139 kB gzip).
 */
const ReactQueryDevtools = lazy(() =>
  import("@tanstack/react-query-devtools").then((mod) => ({
    default: mod.ReactQueryDevtools,
  }))
)

const TanStackRouterDevtools = lazy(() =>
  import("@tanstack/react-router-devtools").then((mod) => ({
    default: mod.TanStackRouterDevtools,
  }))
)

export function Devtools() {
  if (!import.meta.env.DEV) return null

  return (
    <Suspense fallback={null}>
      <ReactQueryDevtools initialIsOpen={false} />
      <TanStackRouterDevtools position="bottom-right" />
    </Suspense>
  )
}
