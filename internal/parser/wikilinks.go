package parser

import (
	"regexp"
	"strings"
)

var (
	// wikilinkDetailedRegex captura ![[...]] ou [[...]]
	// Grupo 1: opcional '!' (indica embed/transclusão)
	// Grupo 2: conteúdo interno entre [[ e ]]
	wikilinkDetailedRegex = regexp.MustCompile(`(!)?\[\[([^\]\r\n]+)\]\]`)

	// tagRegex captura #tag e #hierarquia/subtag (iniciando com letra)
	tagRegex = regexp.MustCompile(`(?:^|[\s\(\[\{,;:])#([a-zA-Z][a-zA-Z0-9_\-\/]*)`)
)

// LinkTarget representa a dissecação completa de um link estilo Obsidian
type LinkTarget struct {
	Raw       string `json:"raw"`
	Target    string `json:"target"`              // Nome da nota de destino normalizado (ex: "Arquitetura")
	Anchor    string `json:"anchor,omitempty"`    // Cabeçalho / seção (ex: "Visão Geral")
	BlockID   string `json:"block_id,omitempty"`  // Identificador de bloco (ex: "^c182")
	Alias     string `json:"alias,omitempty"`     // Rótulo ou texto de exibição (ex: "Visão")
	IsEmbed   bool   `json:"is_embed"`            // Verdadeiro se for ![[...]]
	IsSameDoc bool   `json:"is_same_doc"`         // Verdadeiro se for link interno [[#Secao]]
}

// ExtractedConnections consolida conexões extraídas do Markdown e frontmatter
type ExtractedConnections struct {
	OutgoingLinks []string     `json:"outgoing_links"` // Alvos externos únicos (retrocompatível)
	Links         []LinkTarget `json:"links"`          // Detalhes completos de cada wikilink
	Tags          []string     `json:"tags"`           // União de tags do frontmatter e inline
	Aliases       []string     `json:"aliases"`        // Aliases declarados no frontmatter
	Frontmatter   *Frontmatter `json:"frontmatter,omitempty"`
}

// ParseWikilink disseca uma string de wikilink individual
func ParseWikilink(raw string, isEmbed bool) LinkTarget {
	lt := LinkTarget{
		Raw:     raw,
		IsEmbed: isEmbed,
	}

	trimmed := strings.TrimSpace(raw)

	// 1. Separa o alias se houver (|)
	pipeIdx := strings.IndexByte(trimmed, '|')
	var destination string
	if pipeIdx != -1 {
		destination = strings.TrimSpace(trimmed[:pipeIdx])
		lt.Alias = strings.TrimSpace(trimmed[pipeIdx+1:])
	} else {
		destination = trimmed
	}

	// 2. Separa âncora de cabeçalho ou bloco (#)
	hashIdx := strings.IndexByte(destination, '#')
	if hashIdx != -1 {
		targetNote := strings.TrimSpace(destination[:hashIdx])
		subpath := strings.TrimSpace(destination[hashIdx+1:])

		lt.Target = targetNote
		if targetNote == "" {
			lt.IsSameDoc = true
		}

		if strings.HasPrefix(subpath, "^") {
			lt.BlockID = strings.TrimPrefix(subpath, "^")
		} else {
			lt.Anchor = subpath
		}
	} else {
		lt.Target = destination
	}

	return lt
}

// ExtractConnections extrai o frontmatter YAML, [[wikilinks]] detalhados e #tags estilo Obsidian
func ExtractConnections(content string) ExtractedConnections {
	var conn ExtractedConnections

	// 1. Extrai frontmatter YAML se presente
	fm, body := ExtractFrontmatter(content)
	if fm != nil {
		conn.Frontmatter = fm
		conn.Aliases = fm.Aliases
		conn.Tags = append(conn.Tags, fm.Tags...)
	}

	// 2. Extrai wikilinks do corpo
	matches := wikilinkDetailedRegex.FindAllStringSubmatch(body, -1)
	seenTargets := make(map[string]bool)

	for _, m := range matches {
		if len(m) > 2 {
			isEmbed := m[1] == "!"
			rawLink := m[2]

			lt := ParseWikilink(rawLink, isEmbed)
			conn.Links = append(conn.Links, lt)

			// Só adiciona a OutgoingLinks se tiver destino externo
			if lt.Target != "" && !lt.IsSameDoc {
				if !seenTargets[lt.Target] {
					seenTargets[lt.Target] = true
					conn.OutgoingLinks = append(conn.OutgoingLinks, lt.Target)
				}
			}
		}
	}

	// 3. Extrai tags inline do corpo (removendo wikilinks para não capturar âncoras [[#Secao]])
	bodyWithoutWikilinks := wikilinkDetailedRegex.ReplaceAllString(body, " ")

	seenTags := make(map[string]bool)
	for _, t := range conn.Tags {
		seenTags[t] = true
	}

	tagMatches := tagRegex.FindAllStringSubmatch(bodyWithoutWikilinks, -1)
	for _, m := range tagMatches {
		if len(m) > 1 {
			tag := strings.TrimSpace(m[1])
			if tag != "" && !seenTags[tag] {
				seenTags[tag] = true
				conn.Tags = append(conn.Tags, tag)
			}
		}
	}

	return conn
}
