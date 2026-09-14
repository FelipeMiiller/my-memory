package parser

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter guarda metadados estruturados extraídos do topo do documento Markdown
type Frontmatter struct {
	Title      string         `json:"title,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	Aliases    []string       `json:"aliases,omitempty"`
	Category   string         `json:"category,omitempty"`
	Summary    string         `json:"summary,omitempty"`
	Abstract   string         `json:"abstract,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

// ExtractFrontmatter extrai o bloco YAML delimitado por --- no início do arquivo
// Retorna a estrutura Frontmatter e o restante do corpo do Markdown
func ExtractFrontmatter(content string) (*Frontmatter, string) {
	trimmed := strings.TrimLeft(content, " \t\r\n")
	if !strings.HasPrefix(trimmed, "---") {
		return nil, content
	}

	// Localiza o fechamento do frontmatter
	rest := trimmed[3:]
	// O primeiro \n após ---
	firstNewline := strings.IndexByte(rest, '\n')
	if firstNewline == -1 {
		return nil, content
	}

	rest = rest[firstNewline+1:]
	endIndex := strings.Index(rest, "\n---")
	if endIndex == -1 {
		return nil, content
	}

	yamlBlock := rest[:endIndex]
	body := rest[endIndex+4:]
	// Pula quebra de linha após o fechamento do --- se houver
	if strings.HasPrefix(body, "\r\n") {
		body = body[2:]
	} else if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}

	var rawMap map[string]any
	if err := yaml.Unmarshal([]byte(yamlBlock), &rawMap); err != nil {
		return nil, content
	}

	fm := &Frontmatter{
		Properties: make(map[string]any),
	}

	for k, v := range rawMap {
		switch strings.ToLower(k) {
		case "title":
			if s, ok := v.(string); ok {
				fm.Title = strings.TrimSpace(s)
			}
		case "tags", "tag":
			fm.Tags = append(fm.Tags, normalizeStringOrSlice(v)...)
		case "aliases", "alias":
			fm.Aliases = append(fm.Aliases, normalizeStringOrSlice(v)...)
		case "category":
			if s, ok := v.(string); ok {
				fm.Category = strings.ToLower(strings.TrimSpace(s))
			}
		case "summary":
			if s, ok := v.(string); ok {
				fm.Summary = strings.TrimSpace(s)
			}
		case "abstract":
			if s, ok := v.(string); ok {
				fm.Abstract = strings.TrimSpace(s)
			}
		default:
			fm.Properties[k] = v
		}
	}

	// Normaliza tags do frontmatter removendo '#' inicial caso exista
	for i, t := range fm.Tags {
		fm.Tags[i] = strings.TrimPrefix(t, "#")
	}

	return fm, body
}

// normalizeStringOrSlice converte string ou slice de any/string em slice de strings limpas
func normalizeStringOrSlice(val any) []string {
	var result []string
	switch v := val.(type) {
	case string:
		// Pode ser string única ou separada por vírgula
		parts := strings.Split(v, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				trimmed := strings.TrimSpace(s)
				if trimmed != "" {
					result = append(result, trimmed)
				}
			}
		}
	case []string:
		for _, s := range v {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return result
}
