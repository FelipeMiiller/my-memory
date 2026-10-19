/**
 * Parâmetros de busca da rota `/notes`.
 *
 * Vivem num módulo próprio porque o link gerado pelo pré-processador de
 * wikilinks precisa da mesma forma: o destino é `/notes?path=<path
 * percent-encoded>`, com `anchor` e `block` opcionais.
 *
 * String vazia = ausente. É mais simples de serializar do que `undefined` e
 * evita que o roteador normalize a URL de formas diferentes.
 */
export interface NotesSearch {
  /** Path do vault da nota a abrir. */
  path: string
  /** Âncora de cabeçalho (`[[X#Secao]]`). */
  anchor: string
  /** Identificador de bloco (`[[X#^c182]]`). */
  block: string
}

/** Search inicial da rota: nada selecionado. */
export const emptyNotesSearch: NotesSearch = {
  path: "",
  anchor: "",
  block: "",
}

/** Validador de search da rota `/notes`. */
export function validateNotesSearch(
  search: Record<string, unknown>
): NotesSearch {
  return {
    path: typeof search.path === "string" ? search.path : "",
    anchor: typeof search.anchor === "string" ? search.anchor : "",
    block: typeof search.block === "string" ? search.block : "",
  }
}
