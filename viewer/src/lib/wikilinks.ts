/**
 * Pré-processador de wikilinks.
 *
 * `@tanstack/markdown` não conhece wikilinks (a doc oficial não menciona
 * "wikilink", "Obsidian" nem `[[`), então a viewer converte `[[...]]` em link
 * Markdown **antes** de renderizar.
 *
 * A gramática é a do parser Go (`internal/parser/wikilinks.go`), não uma
 * invenção: a regex é `(!)?\[\[([^\]\r\n]+)\]\]` e a dissecação espelha
 * `ParseWikilink`. A duplicação só é segura porque a tabela de casos do teste
 * espelha a do Go — se os dois divergirem, a viewer passa a mostrar links que
 * o grafo não tem, ou seja, a mentir sobre a topologia.
 */

/** Relações semânticas reconhecidas pelo parser Go. */
const KNOWN_RELATIONS = new Set([
  "links_to",
  "implements",
  "depends_on",
  "supports",
  "refutes",
  "extends",
  "derived_from",
  "fixes",
  "causes",
  "part_of",
  "related_to",
  "similar_to",
  "tagged_as",
])

/** Regex de wikilink — idêntica à do Go (`wikilinkDetailedRegex`). */
const WIKILINK_REGEX = /(!)?\[\[([^\]\r\n]+)\]\]/g

/**
 * Contextos em que o Go **não** cria edge (`stripCodeBlocks`).
 * A ordem importa: fenced -> inline -> comentário -> tabela.
 */
const FENCED_CODE_REGEX = /```[\s\S]*?```/g
const INLINE_CODE_REGEX = /`[^`\r\n]+`/g
const HTML_COMMENT_REGEX = /<!--[\s\S]*?-->/g
const TABLE_ROW_REGEX = /^\s*\|.*$/gm

/** Destruição de um wikilink, espelhando `parser.LinkTarget`. */
export interface WikilinkTarget {
  /** Conteúdo bruto entre `[[` e `]]`. */
  raw: string
  /** Nota de destino normalizada. Vazio em link interno (`[[#Secao]]`). */
  target: string
  /** Relação semântica: `links_to` por padrão. */
  relation: string
  /** Âncora de cabeçalho (`[[X#Secao]]`). */
  anchor: string
  /** Identificador de bloco (`[[X#^c182]]`). */
  blockId: string
  /** Rótulo (`[[X|Label]]`). */
  alias: string
  /** `true` quando o link é um embed (`![[X]]`). */
  isEmbed: boolean
  /** `true` quando o link aponta para o próprio documento (`[[#Secao]]`). */
  isSameDoc: boolean
  /** `true` quando o alvo é um vault federado (`memory://`). */
  isFederated: boolean
  /** Vault federado de destino. */
  federatedRepo: string
  /** Caminho dentro do vault federado. */
  federatedPath: string
}

/** Verifica se a relação é reconhecida pelo parser Go. */
export function isKnownRelation(relation: string): boolean {
  return KNOWN_RELATIONS.has(relation.toLowerCase())
}

/**
 * Disseca um wikilink individual. Espelha `parser.ParseWikilink` passo a passo.
 */
export function parseWikilink(raw: string, isEmbed: boolean): WikilinkTarget {
  const link: WikilinkTarget = {
    raw,
    target: "",
    relation: "links_to",
    anchor: "",
    blockId: "",
    alias: "",
    isEmbed,
    isSameDoc: false,
    isFederated: false,
    federatedRepo: "",
    federatedPath: "",
  }

  const trimmed = raw.trim()

  // 1. Separa o alias se houver (|)
  const pipeIdx = trimmed.indexOf("|")
  let destination: string
  if (pipeIdx !== -1) {
    destination = trimmed.slice(0, pipeIdx).trim()
    link.alias = trimmed.slice(pipeIdx + 1).trim()
  } else {
    destination = trimmed
  }

  // 2. Separa âncora de cabeçalho ou bloco (#)
  const hashIdx = destination.indexOf("#")
  let targetNote: string
  if (hashIdx !== -1) {
    targetNote = destination.slice(0, hashIdx).trim()
    const subpath = destination.slice(hashIdx + 1).trim()
    if (targetNote === "") link.isSameDoc = true
    if (subpath.startsWith("^")) link.blockId = subpath.slice(1)
    else link.anchor = subpath
  } else {
    targetNote = destination
  }

  // 3. Relação por prefixo: [[relation:Target]] ou [[rel:relation:Target]].
  //    `memory://` não pode ser confundido com uma relação.
  const colonIdx = targetNote.indexOf(":")
  if (colonIdx !== -1) {
    const prefix = targetNote.slice(0, colonIdx).trim().toLowerCase()
    const rest = targetNote.slice(colonIdx + 1).trim()
    if (prefix === "rel") {
      const second = rest.indexOf(":")
      if (second !== -1) {
        const candidate = rest.slice(0, second).trim().toLowerCase()
        if (candidate !== "memory") {
          link.relation = candidate
          targetNote = rest.slice(second + 1).trim()
        }
      } else {
        link.relation = rest.toLowerCase()
      }
    } else if (prefix !== "memory" && isKnownRelation(prefix)) {
      link.relation = prefix
      targetNote = rest
    }
  }

  // 4. Sem relação por prefixo: [[Target|rel:relation]] ou [[Target|relation]]
  if (link.relation === "links_to" && link.alias !== "") {
    const aliasLower = link.alias.toLowerCase()
    if (aliasLower.startsWith("rel:")) {
      link.relation = aliasLower.slice("rel:".length)
    } else if (isKnownRelation(aliasLower)) {
      link.relation = aliasLower
    }
  }

  // 5. URI federada memory://<repo>/<path>
  if (targetNote.startsWith("memory://")) {
    link.isFederated = true
    const uriRest = targetNote.slice("memory://".length)
    const slashIdx = uriRest.indexOf("/")
    if (slashIdx !== -1) {
      link.federatedRepo = uriRest.slice(0, slashIdx).trim()
      link.federatedPath = trimSlashes(uriRest.slice(slashIdx + 1))
    } else {
      link.federatedRepo = uriRest.trim()
      link.federatedPath = ""
    }
    link.target =
      link.federatedPath !== ""
        ? `memory://${link.federatedRepo}/${link.federatedPath}`
        : `memory://${link.federatedRepo}`
  } else {
    link.target = trimSlashes(targetNote)
    if (link.target === "..." || link.target === ".") link.target = ""
  }

  return link
}

function trimSlashes(value: string): string {
  return value.replace(/^[ /\r\n\t]+/, "").replace(/[ /\r\n\t]+$/, "")
}

/**
 * Rótulo visível do link.
 *
 * A relação **não** é exibida: ela é metadado de grafo. `[[X|rel:implements]]`
 * mostra "X" — o alias que é marcador de relação não vira rótulo.
 */
export function resolveWikilinkLabel(link: WikilinkTarget): string {
  const alias = link.alias
  if (alias !== "") {
    const lower = alias.toLowerCase()
    const isRelationMarker =
      lower.startsWith("rel:") || KNOWN_RELATIONS.has(lower)
    if (!isRelationMarker) return alias
  }
  if (link.target !== "") return link.target
  return link.anchor || link.blockId
}

/** Destino do link: rota interna da SPA, com o path percent-encoded. */
export function buildWikilinkHref(link: WikilinkTarget): string {
  let href = `/notes?path=${encodeURIComponent(link.target)}`
  if (link.anchor !== "") href += `&anchor=${encodeURIComponent(link.anchor)}`
  if (link.blockId !== "") href += `&block=${encodeURIComponent(link.blockId)}`
  return href
}

function escapeLabel(label: string): string {
  return label.replace(/([\\[\]])/g, "\\$1")
}

function escapeTitle(title: string): string {
  return title.replace(/\\/g, "\\\\").replace(/"/g, '\\"')
}

/**
 * Converte um wikilink dissecado no link Markdown correspondente.
 *
 * - federado (`memory://`) → texto literal: a viewer não resolve vault externo
 * - link interno (`[[#Secao]]`) → âncora local, sem navegação
 * - sem destino (`[[...]]`) → texto literal
 * - demais → link para a rota `/notes`, com a relação no `title`
 */
export function renderWikilink(link: WikilinkTarget, original: string): string {
  if (link.isFederated) return original

  if (link.isSameDoc) {
    const fragment = link.blockId !== "" ? link.blockId : link.anchor
    if (fragment === "") return original
    const label = escapeLabel(resolveWikilinkLabel(link))
    return `[${label}](#${encodeURIComponent(fragment)})`
  }

  if (link.target === "") return original

  const label = escapeLabel(resolveWikilinkLabel(link))
  const href = buildWikilinkHref(link)
  const title =
    link.relation !== "" && link.relation !== "links_to"
      ? ` "${escapeTitle(link.relation)}"`
      : ""
  return `[${label}](${href}${title})`
}

/**
 * Localiza o fim do frontmatter YAML, no mesmo formato do Go
 * (`parser.ExtractFrontmatter`). Devolve -1 quando não há frontmatter.
 */
export function frontmatterEnd(content: string): number {
  let i = 0
  if (content.charCodeAt(0) === 0xfeff) i = 1
  while (i < content.length && " \t\r\n".includes(content[i] as string)) i++
  if (!content.startsWith("---", i)) return -1
  const firstNewline = content.indexOf("\n", i + 3)
  if (firstNewline === -1) return -1
  const closeIdx = content.indexOf("\n---", firstNewline + 1)
  if (closeIdx === -1) return -1
  return closeIdx + 4
}

/**
 * Máscara dos trechos em que o Go não cria edge.
 *
 * O Go substitui esses trechos por um espaço e perde os offsets; aqui eles são
 * apenas marcados, o que produz o **mesmo** conjunto de wikilinks com link:
 * o conteúdo mascarado nunca produz link, e nada fora dele é afetado.
 */
function buildStripMask(text: string): Uint8Array {
  const mask = new Uint8Array(text.length)
  const patterns = [
    FENCED_CODE_REGEX,
    INLINE_CODE_REGEX,
    HTML_COMMENT_REGEX,
    TABLE_ROW_REGEX,
  ]
  for (const pattern of patterns) {
    for (const match of text.matchAll(pattern)) {
      const start = match.index
      const end = start + match[0].length
      for (let i = start; i < end; i++) mask[i] = 1
    }
  }
  return mask
}

function isMasked(mask: Uint8Array, start: number, end: number): boolean {
  for (let i = start; i < end; i++) {
    if (mask[i] === 1) return true
  }
  return false
}

function preprocessBody(body: string): string {
  const mask = buildStripMask(body)
  let out = ""
  let last = 0

  for (const match of body.matchAll(WIKILINK_REGEX)) {
    const full = match[0]
    const start = match.index
    const end = start + full.length
    if (isMasked(mask, start, end)) continue

    const link = parseWikilink(match[2] ?? "", match[1] === "!")
    out += body.slice(last, start)
    out += renderWikilink(link, full)
    last = end
  }

  out += body.slice(last)
  return out
}

/**
 * Converte os wikilinks de um Markdown em links Markdown.
 *
 * O frontmatter é preservado intacto: o Go também só varre o corpo.
 */
export function preprocessWikilinks(markdown: string): string {
  const end = frontmatterEnd(markdown)
  if (end === -1) return preprocessBody(markdown)
  return markdown.slice(0, end) + preprocessBody(markdown.slice(end))
}

/**
 * Lista os wikilinks de um documento, com a mesma regra de stripping do Go.
 * Útil para diagnóstico e para a tabela de casos do teste.
 */
export function extractWikilinks(markdown: string): WikilinkTarget[] {
  const end = frontmatterEnd(markdown)
  const body = end === -1 ? markdown : markdown.slice(end)
  const mask = buildStripMask(body)
  const links: WikilinkTarget[] = []

  for (const match of body.matchAll(WIKILINK_REGEX)) {
    const full = match[0]
    const start = match.index
    if (isMasked(mask, start, start + full.length)) continue
    links.push(parseWikilink(match[2] ?? "", match[1] === "!"))
  }

  return links
}
