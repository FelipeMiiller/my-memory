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

// EdgeConnection representa uma aresta dirigida tipada com semântica epistêmica
type EdgeConnection struct {
	Target          string  `json:"target"`
	Relation        string  `json:"relation"`         // "links_to", "implements", "depends_on", "supports", "refutes", "extends", "derived_from", "tagged_as"
	EpistemicStatus string  `json:"epistemic_status"` // "EXTRACTED", "INFERRED"
	Weight          float64 `json:"weight"`           // 1.0 por padrão
}

// LinkTarget representa a dissecação completa de um link estilo Obsidian
type LinkTarget struct {
	Raw       string `json:"raw"`
	Target    string `json:"target"`             // Nome da nota de destino normalizado (ex: "Arquitetura")
	Relation  string `json:"relation,omitempty"` // Tipo da relação ("implements", "depends_on", "links_to", etc)
	Anchor    string `json:"anchor,omitempty"`   // Cabeçalho / seção (ex: "Visão Geral")
	BlockID   string `json:"block_id,omitempty"` // Identificador de bloco (ex: "^c182")
	Alias     string `json:"alias,omitempty"`    // Rótulo ou texto de exibição (ex: "Visão")
	IsEmbed   bool   `json:"is_embed"`           // Verdadeiro se for ![[...]]
	IsSameDoc bool   `json:"is_same_doc"`        // Verdadeiro se for link interno [[#Secao]]
}

// ExtractedConnections consolida conexões extraídas do Markdown e frontmatter
type ExtractedConnections struct {
	OutgoingLinks []string         `json:"outgoing_links"`  // Alvos externos únicos (retrocompatível)
	Edges         []EdgeConnection `json:"edges,omitempty"` // Arestas tipadas dirigidas
	Links         []LinkTarget     `json:"links"`           // Detalhes completos de cada wikilink
	Tags          []string         `json:"tags"`            // União de tags do frontmatter e inline
	Aliases       []string         `json:"aliases"`         // Aliases declarados no frontmatter
	Frontmatter   *Frontmatter     `json:"frontmatter,omitempty"`
}

// IsKnownRelation verifica se a relação é uma relação semântica reconhecida
func IsKnownRelation(rel string) bool {
	switch strings.ToLower(rel) {
	case "links_to", "implements", "depends_on", "supports", "refutes",
		"extends", "derived_from", "fixes", "causes", "part_of", "related_to", "similar_to", "tagged_as":
		return true
	default:
		return false
	}
}

// ParseWikilink disseca uma string de wikilink individual
func ParseWikilink(raw string, isEmbed bool) LinkTarget {
	lt := LinkTarget{
		Raw:      raw,
		IsEmbed:  isEmbed,
		Relation: "links_to",
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
	var targetNote string
	if hashIdx != -1 {
		targetNote = strings.TrimSpace(destination[:hashIdx])
		subpath := strings.TrimSpace(destination[hashIdx+1:])

		if targetNote == "" {
			lt.IsSameDoc = true
		}

		if strings.HasPrefix(subpath, "^") {
			lt.BlockID = strings.TrimPrefix(subpath, "^")
		} else {
			lt.Anchor = subpath
		}
	} else {
		targetNote = destination
	}

	// 3. Extrai relação por prefixo: [[relation:Target]] ou [[rel:relation:Target]]
	colonIdx := strings.IndexByte(targetNote, ':')
	if colonIdx != -1 {
		prefix := strings.ToLower(strings.TrimSpace(targetNote[:colonIdx]))
		rest := strings.TrimSpace(targetNote[colonIdx+1:])
		if prefix == "rel" {
			c2 := strings.IndexByte(rest, ':')
			if c2 != -1 {
				lt.Relation = strings.ToLower(strings.TrimSpace(rest[:c2]))
				targetNote = strings.TrimSpace(rest[c2+1:])
			} else {
				lt.Relation = strings.ToLower(rest)
			}
		} else if IsKnownRelation(prefix) {
			lt.Relation = prefix
			targetNote = rest
		}
	}

	// 4. Se não teve relação por prefixo, verifica alias: [[Target|rel:relation]] ou [[Target|relation]]
	if lt.Relation == "links_to" && lt.Alias != "" {
		aliasLower := strings.ToLower(lt.Alias)
		if strings.HasPrefix(aliasLower, "rel:") {
			lt.Relation = strings.TrimPrefix(aliasLower, "rel:")
		} else if IsKnownRelation(aliasLower) {
			lt.Relation = aliasLower
		}
	}

	lt.Target = targetNote
	return lt
}

// ExtractConnections extrai o frontmatter YAML, [[wikilinks]] detalhados, arestas tipadas e #tags estilo Obsidian
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

	// 4. Constrói a lista deduplicada de EdgeConnection
	seenEdge := make(map[string]bool)
	addEdge := func(target, relation, epistemic string, weight float64) {
		target = strings.TrimSpace(target)
		relation = strings.TrimSpace(relation)
		if target == "" || relation == "" {
			return
		}
		key := target + "::" + relation
		if !seenEdge[key] {
			seenEdge[key] = true
			conn.Edges = append(conn.Edges, EdgeConnection{
				Target:          target,
				Relation:        relation,
				EpistemicStatus: epistemic,
				Weight:          weight,
			})
		}
	}

	// 4.1 Arestas dos wikilinks
	for _, lt := range conn.Links {
		if lt.Target != "" && !lt.IsSameDoc {
			rel := lt.Relation
			if rel == "" {
				rel = "links_to"
			}
			addEdge(lt.Target, rel, "EXTRACTED", 1.0)
		}
	}

	// 4.2 Arestas das tags
	for _, t := range conn.Tags {
		addEdge(t, "tagged_as", "EXTRACTED", 1.0)
	}

	// 4.3 Arestas do frontmatter properties/relations
	if conn.Frontmatter != nil && conn.Frontmatter.Properties != nil {
		if relationsMap, ok := conn.Frontmatter.Properties["relations"].(map[string]any); ok {
			for relName, v := range relationsMap {
				for _, t := range normalizeStringOrSlice(v) {
					addEdge(t, relName, "EXTRACTED", 1.0)
					if !seenTargets[t] {
						seenTargets[t] = true
						conn.OutgoingLinks = append(conn.OutgoingLinks, t)
					}
				}
			}
		}
		for k, v := range conn.Frontmatter.Properties {
			if IsKnownRelation(k) && k != "relations" && k != "tagged_as" {
				for _, t := range normalizeStringOrSlice(v) {
					addEdge(t, k, "EXTRACTED", 1.0)
					if !seenTargets[t] {
						seenTargets[t] = true
						conn.OutgoingLinks = append(conn.OutgoingLinks, t)
					}
				}
			}
		}
	}

	return conn
}
