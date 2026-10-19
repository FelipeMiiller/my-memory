import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  Link,
  Outlet,
  createRootRoute,
  useRouterState,
} from "@tanstack/react-router"
import type { ErrorComponentProps } from "@tanstack/react-router"
import { Devtools } from "@/components/devtools"
import { ThemeProvider } from "@/components/theme-provider"
import { ErrorState } from "@/components/ui-kit"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

const NAV_ITEMS = [
  { to: "/", label: "Overview" },
  { to: "/graph", label: "Grafo" },
  { to: "/notes", label: "Notas" },
  { to: "/drift", label: "Drift" },
] as const

export const Route = createRootRoute({
  component: RootComponent,
  errorComponent: RootErrorComponent,
  notFoundComponent: NotFoundComponent,
})

function RootComponent() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <div className="min-h-svh bg-background text-foreground">
          <header className="border-b border-border">
            <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-x-6 gap-y-2 px-4 py-3">
              <Link to="/" className="text-sm font-semibold tracking-tight">
                my-memory
              </Link>
              <nav className="flex flex-wrap items-center gap-1">
                {NAV_ITEMS.map((item) => {
                  const active =
                    item.to === "/"
                      ? pathname === "/"
                      : pathname.startsWith(item.to)
                  return (
                    <Link
                      key={item.to}
                      to={item.to}
                      className={
                        active
                          ? "rounded-md bg-secondary px-2.5 py-1 text-sm font-medium text-secondary-foreground"
                          : "rounded-md px-2.5 py-1 text-sm text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                      }
                    >
                      {item.label}
                    </Link>
                  )
                })}
              </nav>
              <span className="ml-auto text-xs text-muted-foreground">
                Painel de métricas — leitura apenas
              </span>
            </div>
          </header>
          <main className="mx-auto max-w-6xl space-y-4 px-4 py-6">
            <Outlet />
          </main>
          <Devtools />
        </div>
      </QueryClientProvider>
    </ThemeProvider>
  )
}

function RootErrorComponent({ error, reset }: ErrorComponentProps) {
  return (
    <div className="mx-auto max-w-2xl space-y-3 p-6">
      <ErrorState
        title="A rota quebrou"
        message={error instanceof Error ? error.message : String(error)}
        hint="O MCP pode estar fora do ar, ou a resposta não bateu com o que a viewer espera."
      />
      <button
        type="button"
        onClick={() => reset()}
        className="rounded-md border border-border px-3 py-1.5 text-sm hover:bg-accent"
      >
        Tentar de novo
      </button>
    </div>
  )
}

function NotFoundComponent() {
  return (
    <div className="mx-auto max-w-2xl p-6">
      <ErrorState
        title="Rota não encontrada"
        message="Esta URL não corresponde a nenhuma tela da viewer."
      />
    </div>
  )
}
