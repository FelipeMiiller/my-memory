package store

import (
	"path/filepath"
	"sort"
	"strings"
)

// DocumentIdentity identifica um documento indexado para a resolução do alvo
// de uma aresta: o `id` (que no indexador É o path do documento) e o `title`
// (o rótulo humano do documento).
//
// A identidade canônica do nó no grafo é o `documents.id` (ADR-066): o título
// é rótulo, e o path é o mesmo id — as duas coisas são aliases do mesmo nó.
type DocumentIdentity struct {
	// ID é o `documents.id` — a identidade canônica do nó.
	ID string
	// Title é o `documents.title` — o rótulo do nó.
	Title string
}

// DocumentIdentitiesFromTitleMap converte o mapa título → id (o que o
// indexador já carrega) na lista de identidades que alimenta o índice de
// aliases.
//
// A lista vem **ordenada por id**: a ordem decide o desempate quando dois
// documentos compartilham um título, e o desempate entre homônimos em si é da
// spec `fix-graph-path-discovery-gaps` (ISSUE-046).
func DocumentIdentitiesFromTitleMap(titleToID map[string]string) []DocumentIdentity {
	identities := make([]DocumentIdentity, 0, len(titleToID))
	for title, id := range titleToID {
		identities = append(identities, DocumentIdentity{ID: id, Title: title})
	}
	sort.Slice(identities, func(i, j int) bool {
		return identities[i].ID < identities[j].ID
	})
	return identities
}

// BuildDocumentAliasIndex monta o índice de aliases do documento:
// alias (título, path, basename, caminho relativo) → `documents.id`.
//
// A chave é minúscula porque o match de wikilink no Obsidian é
// case-insensitive. O primeiro documento que registra um alias vence, e a
// ordem de entrada é a do slice — o chamador ordena por id, então o
// desempate entre homônimos é determinístico.
func BuildDocumentAliasIndex(identities []DocumentIdentity, rootDir string) map[string]string {
	index := make(map[string]string, len(identities)*4)
	for _, identity := range identities {
		if identity.ID == "" {
			continue
		}
		for _, alias := range documentAliases(identity, rootDir) {
			key := strings.ToLower(strings.TrimSpace(alias))
			if key == "" {
				continue
			}
			if _, exists := index[key]; !exists {
				index[key] = identity.ID
			}
		}
	}
	return index
}

// documentAliases devolve os aliases de um documento: o título, o id (o path
// do documento), o basename e o caminho relativo dentro do vault — cada um
// com e sem a extensão `.md` e nas duas variantes de barra.
func documentAliases(identity DocumentIdentity, rootDir string) []string {
	aliases := []string{identity.Title}

	paths := []string{identity.ID}
	if base := filepath.Base(identity.ID); base != "" {
		paths = append(paths, base)
	}
	if rootDir != "" {
		if rel, err := filepath.Rel(rootDir, identity.ID); err == nil && !strings.HasPrefix(rel, "..") {
			paths = append(paths, rel)
		}
	}

	for _, path := range paths {
		aliases = append(aliases, path)
		aliases = append(aliases, stripMarkdownExtension(path))
		// A variante com barra invertida é a mesma identidade no Windows.
		back := strings.ReplaceAll(path, "/", "\\")
		aliases = append(aliases, back)
		aliases = append(aliases, stripMarkdownExtension(back))
	}
	return aliases
}

// stripMarkdownExtension remove a extensão `.md` (qualquer caixa) do caminho.
func stripMarkdownExtension(path string) string {
	if len(path) > 3 && strings.EqualFold(path[len(path)-3:], ".md") {
		return path[:len(path)-3]
	}
	return path
}

// ResolveEdgeTarget resolve o alvo de uma aresta para o `documents.id` do
// documento que ele nomeia (ADR-066).
//
// Quando o alvo não resolve para um documento indexado, o **texto do wikilink
// é preservado**: é assim que o `mem doctor` enxerga o link rompido
// (ADR-063) e que um link para material fora do índice por decisão continua
// vivo no Obsidian.
func ResolveEdgeTarget(target string, aliasIndex map[string]string) string {
	key := strings.ToLower(strings.TrimSpace(target))
	if id, ok := aliasIndex[key]; ok && id != "" {
		return id
	}
	return target
}

// NodeResolutionOptions carrega o contexto de disco que a resolução de nó
// precisa para resolver um `alias:` de frontmatter.
//
// O alias não é persistido no índice (a ADR-056 não foi implementada), então
// ele só resolve contra o disco — e só quando a raiz do vault é conhecida.
type NodeResolutionOptions struct {
	// VaultRoot é a raiz do vault no disco. Vazia significa "sem raiz
	// conhecida": o alias devolve o erro de nó não encontrado.
	VaultRoot string
}
