import { Markdown } from "@tanstack/markdown/react"
import { useRouter } from "@tanstack/react-router"
import type { ComponentPropsWithoutRef } from "react"
import { preprocessWikilinks } from "@/lib/wikilinks"
import type { NotesSearch } from "@/lib/note-search"

/**
 * Renderiza o Markdown de uma nota.
 *
 * Duas etapas: o pré-processador converte `[[wikilinks]]` em links Markdown, e
 * o `@tanstack/markdown` faz o render. O renderer não conhece wikilinks — sem
 * a primeira etapa, a nota apareceria com `[[X]]` literal.
 *
 * Links internos (`/notes?path=...`) são interceptados para virar navegação de
 * cliente: recarregar a página inteira exigiria a rota de fallback do servidor
 * (ADR-059 SERV-23..25), que é do lado Go.
 */
export function NoteMarkdown({ content }: { content: string }) {
  return (
    <div className="markdown-body">
      <Markdown components={{ a: MarkdownAnchor }}>
        {preprocessWikilinks(content)}
      </Markdown>
    </div>
  )
}

function MarkdownAnchor({
  href,
  children,
  ...rest
}: ComponentPropsWithoutRef<"a">) {
  const router = useRouter()

  if (typeof href !== "string" || !href.startsWith("/notes")) {
    return (
      <a href={href} {...rest}>
        {children}
      </a>
    )
  }

  const search = parseNotesSearch(href)

  return (
    <a
      href={href}
      {...rest}
      onClick={(event) => {
        event.preventDefault()
        void router.navigate({ to: "/notes", search })
      }}
    >
      {children}
    </a>
  )
}

/** Converte `/notes?path=X&anchor=Y` no objeto de search da rota. */
export function parseNotesSearch(href: string): NotesSearch {
  const query = href.includes("?") ? href.slice(href.indexOf("?") + 1) : ""
  const params = new URLSearchParams(query)
  return {
    path: params.get("path") ?? "",
    anchor: params.get("anchor") ?? "",
    block: params.get("block") ?? "",
  }
}
