import { useEffect } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import type { ErrorComponentProps } from "@tanstack/react-router"
import { NoteMarkdown } from "@/components/note-markdown"
import { ToolPayload } from "@/components/tool-payload"
import {
  EmptyState,
  ErrorState,
  KeyValue,
  Loading,
  Panel,
} from "@/components/ui-kit"
import { formatBytes, formatUnixDate } from "@/lib/format"
import { describeUnknownError } from "@/lib/errors"
import { validateNotesSearch } from "@/lib/note-search"
import {
  useFileMetadata,
  useNoteContent,
  useNoteOutline,
  useVaultFiles,
} from "@/lib/queries"
import { useViewerStore } from "@/store/viewer-store"

/**
 * Notas: a listagem do vault e a leitura de uma nota.
 *
 * Consome `memory_list_files` (com filtro de extensão e de pasta),
 * `memory_read_note`, `memory_note_outline` e `memory_file_metadata`.
 *
 * A seleção e os filtros vivem no Zustand (estado de cliente); o conteúdo
 * vem do Query (estado de servidor). A URL carrega o path da nota, para que
 * a tela atual seja um deep link.
 */
export const Route = createFileRoute("/notes")({
  component: NotesRoute,
  validateSearch: (search: Record<string, unknown>) =>
    validateNotesSearch(search),
  errorComponent: NotesError,
})

function NotesRoute() {
  const navigate = useNavigate()
  const search = Route.useSearch()

  const extFilter = useViewerStore((state) => state.extFilter)
  const folderFilter = useViewerStore((state) => state.folderFilter)
  const selectedPath = useViewerStore((state) => state.selectedPath)
  const setExtFilter = useViewerStore((state) => state.setExtFilter)
  const setFolderFilter = useViewerStore((state) => state.setFolderFilter)
  const selectPath = useViewerStore((state) => state.selectPath)

  const files = useVaultFiles(extFilter, folderFilter)
  const note = useNoteContent(selectedPath)
  const outline = useNoteOutline(selectedPath)
  const metadata = useFileMetadata(selectedPath)

  // A URL é a fonte da verdade quando ela muda por fora (wikilink, deep link).
  useEffect(() => {
    if (search.path !== "" && search.path !== selectedPath) {
      selectPath(search.path)
    }
  }, [search.path, selectedPath, selectPath])

  function handleSelect(path: string) {
    selectPath(path)
    void navigate({ to: "/notes", search: { path, anchor: "", block: "" } })
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[20rem_1fr]">
      <div className="space-y-4">
        <Panel
          title="Filtros"
          description="memory_list_files — extensão e pasta do vault."
        >
          <div className="space-y-2">
            <label className="block text-xs font-medium text-muted-foreground">
              Extensão
              <input
                value={extFilter}
                onChange={(event) => setExtFilter(event.target.value.trim())}
                placeholder="md"
                className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
              />
            </label>
            <label className="block text-xs font-medium text-muted-foreground">
              Pasta
              <input
                value={folderFilter}
                onChange={(event) => setFolderFilter(event.target.value.trim())}
                placeholder="docs/adr"
                className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm outline-none focus-visible:border-ring"
              />
            </label>
          </div>
        </Panel>

        <Panel title="Documentos indexados">
          {files.isPending ? (
            <Loading />
          ) : files.isError ? (
            <ErrorState message={files.error.message} />
          ) : (
            <ToolPayload result={files.data}>
              {(documents) =>
                documents.length === 0 ? (
                  <EmptyState message="Nenhum documento indexado corresponde a esses filtros." />
                ) : (
                  <ul className="max-h-[28rem] space-y-0.5 overflow-auto">
                    {documents.map((document) => (
                      <li key={document.id}>
                        <button
                          type="button"
                          onClick={() => handleSelect(document.path)}
                          className={
                            document.path === selectedPath
                              ? "w-full rounded-md bg-secondary px-2 py-1.5 text-left text-sm font-medium text-secondary-foreground"
                              : "w-full rounded-md px-2 py-1.5 text-left text-sm hover:bg-accent"
                          }
                        >
                          <span className="block truncate">
                            {document.title || document.path}
                          </span>
                          <span className="block truncate text-[11px] text-muted-foreground">
                            {document.path}
                          </span>
                        </button>
                      </li>
                    ))}
                  </ul>
                )
              }
            </ToolPayload>
          )}
        </Panel>
      </div>

      <div className="space-y-4">
        {selectedPath === "" ? (
          <Panel title="Nenhuma nota selecionada">
            <EmptyState message="Escolha um documento na lista para ler o conteúdo." />
          </Panel>
        ) : (
          <>
            <Panel
              title={selectedPath}
              description="memory_read_note — conteúdo renderizado com TanStack Markdown."
            >
              {note.isPending ? (
                <Loading />
              ) : note.isError ? (
                <ErrorState
                  title="Não foi possível ler a nota"
                  message={note.error.message}
                  hint="Um basename ambíguo é erro do índice, não falta da nota."
                />
              ) : (
                <ToolPayload result={note.data}>
                  {(result) => <NoteMarkdown content={result.content} />}
                </ToolPayload>
              )}
            </Panel>

            <div className="grid gap-4 md:grid-cols-2">
              <Panel title="Outline" description="memory_note_outline">
                {outline.isPending ? (
                  <Loading />
                ) : outline.isError ? (
                  <ErrorState message={outline.error.message} />
                ) : (
                  <ToolPayload result={outline.data}>
                    {(headings) =>
                      headings.length === 0 ? (
                        <EmptyState message="A nota não tem títulos ATX." />
                      ) : (
                        <ul className="space-y-1">
                          {headings.map((heading, index) => (
                            <li
                              key={`${heading.level}-${index}`}
                              style={{
                                paddingLeft: `${(heading.level - 1) * 12}px`,
                              }}
                              className="text-sm"
                            >
                              <span className="text-[11px] text-muted-foreground">
                                h{heading.level}
                              </span>{" "}
                              {heading.text}
                            </li>
                          ))}
                        </ul>
                      )
                    }
                  </ToolPayload>
                )}
              </Panel>

              <Panel title="Metadados" description="memory_file_metadata">
                {metadata.isPending ? (
                  <Loading />
                ) : metadata.isError ? (
                  <ErrorState message={metadata.error.message} />
                ) : (
                  <ToolPayload result={metadata.data}>
                    {(result) => (
                      <div className="divide-y divide-border">
                        <KeyValue label="Path" value={result.path} />
                        <KeyValue label="Título" value={result.title || "—"} />
                        <KeyValue
                          label="Tamanho"
                          value={formatBytes(result.size)}
                        />
                        <KeyValue
                          label="Modificado em"
                          value={formatUnixDate(result.mtime)}
                        />
                      </div>
                    )}
                  </ToolPayload>
                )}
              </Panel>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

function NotesError({ error }: ErrorComponentProps) {
  return (
    <ErrorState
      title="Erro na rota de Notas"
      message={describeUnknownError(error)}
    />
  )
}
