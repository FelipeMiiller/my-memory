import type { ReactNode } from "react"
import type { McpToolResult } from "@/lib/mcp-client"
import { RawText } from "./ui-kit"

/**
 * Renderiza o resultado de uma tool MCP.
 *
 * Duas fontes, uma tela:
 *   - `structuredContent` validado → o painel desenha os dados;
 *   - ausente ou inválido → o texto cru que o MCP devolveu, com um aviso
 *     discreto. Nunca uma tela branca (AC-34 e AC-36 da spec).
 */
export function ToolPayload<T>({
  result,
  children,
}: {
  result: McpToolResult<T>
  children: (data: T) => ReactNode
}) {
  if (result.hasStructured && result.structured !== null) {
    return <>{children(result.structured)}</>
  }

  return (
    <div className="space-y-2">
      {result.validationError ? (
        <p className="rounded-lg border border-amber-500/40 bg-amber-500/5 px-3 py-2 text-xs text-amber-700 dark:text-amber-400">
          O payload estruturado desta tool não bate com o schema esperado (
          {result.validationError}). Exibindo a resposta crua do MCP.
        </p>
      ) : null}
      <RawText text={result.text} />
    </div>
  )
}
