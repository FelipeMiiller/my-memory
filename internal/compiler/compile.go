package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/parser"
)

// CompiledSource representa um fragmento ou documento fonte utilizado na compilação
type CompiledSource struct {
	DocPath string  `json:"doc_path"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Score   float64 `json:"score,omitempty"`
}

// normalizeHeading assegura que o título de seção tenha prefixo markdown '#'
func normalizeHeading(heading string) (string, int) {
	clean := strings.TrimSpace(heading)
	if clean == "" {
		clean = "## Notas Adicionais"
	}
	if !strings.HasPrefix(clean, "#") {
		clean = "## " + clean
	}

	// Calcula o nível do cabeçalho
	level := 0
	for level < len(clean) && clean[level] == '#' {
		level++
	}
	return clean, level
}

// AppendSection anexa cirurgicamente um bloco de conteúdo sob uma seção específica ou ao final da nota
func AppendSection(vaultRoot, requestedPath, heading, content string, createIfMissing bool) (*NoteResult, error) {
	absPath, relPath, err := SafeResolvePath(vaultRoot, requestedPath)
	if err != nil {
		return nil, err
	}

	cleanContent := strings.TrimSpace(content)
	if cleanContent == "" {
		return nil, fmt.Errorf("conteúdo para apensamento não pode ser vazio")
	}

	cleanHeading, headingLevel := normalizeHeading(heading)

	// Se o arquivo não existir
	if _, statErr := os.Stat(absPath); os.IsNotExist(statErr) {
		if !createIfMissing {
			return nil, fmt.Errorf("arquivo não encontrado: %s", relPath)
		}
		initialBody := fmt.Sprintf("%s\n\n%s\n", cleanHeading, cleanContent)
		return WriteAtomicNote(vaultRoot, requestedPath, "", initialBody, nil, nil, "concept", nil, false)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo existente '%s': %w", absPath, err)
	}

	rawText := string(data)
	lines := strings.Split(rawText, "\n")

	headingLineIdx := -1
	for i, l := range lines {
		trimmedLine := strings.TrimSpace(l)
		if strings.EqualFold(trimmedLine, cleanHeading) {
			headingLineIdx = i
			break
		}
	}

	var updatedText string
	if headingLineIdx != -1 {
		// A seção já existe: localiza onde termina essa seção (próximo cabeçalho de mesmo nível ou superior)
		insertIdx := len(lines)
		for i := headingLineIdx + 1; i < len(lines); i++ {
			trimmedLine := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmedLine, "#") {
				currLevel := 0
				for currLevel < len(trimmedLine) && trimmedLine[currLevel] == '#' {
					currLevel++
				}
				if currLevel > 0 && currLevel <= headingLevel {
					insertIdx = i
					break
				}
			}
		}

		var b strings.Builder
		for i := 0; i < insertIdx; i++ {
			b.WriteString(lines[i])
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(cleanContent)
		b.WriteString("\n\n")
		for i := insertIdx; i < len(lines); i++ {
			b.WriteString(lines[i])
			if i < len(lines)-1 {
				b.WriteString("\n")
			}
		}
		updatedText = b.String()
	} else {
		// A seção ainda não existe: anexa ao final do documento
		var b strings.Builder
		b.WriteString(strings.TrimRight(rawText, "\r\n"))
		b.WriteString("\n\n")
		b.WriteString(cleanHeading)
		b.WriteString("\n\n")
		b.WriteString(cleanContent)
		b.WriteString("\n")
		updatedText = b.String()
	}

	finalBytes := []byte(updatedText)
	if err := os.WriteFile(absPath, finalBytes, 0644); err != nil {
		return nil, fmt.Errorf("falha ao regravar arquivo '%s': %w", absPath, err)
	}

	conns := parser.ExtractConnections(updatedText)
	hasher := sha256.New()
	hasher.Write(finalBytes)
	shaHex := hex.EncodeToString(hasher.Sum(nil))

	noteTitle := ""
	if conns.Frontmatter != nil && conns.Frontmatter.Title != "" {
		noteTitle = conns.Frontmatter.Title
	}
	if noteTitle == "" {
		baseName := filepath.Base(relPath)
		noteTitle = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}

	resType := "concept"
	if conns.Frontmatter != nil && conns.Frontmatter.Properties != nil {
		if tVal, ok := conns.Frontmatter.Properties["type"].(string); ok && tVal != "" {
			resType = tVal
		}
	}

	return &NoteResult{
		Path:        relPath,
		AbsPath:     absPath,
		SHA256:      shaHex,
		Title:       noteTitle,
		Type:        resType,
		Tags:        conns.Tags,
		Aliases:     conns.Aliases,
		Outgoing:    conns.OutgoingLinks,
		EdgesCount:  len(conns.Edges),
		Overwritten: true,
	}, nil
}

// CompileTopicNote sintetiza resultados de busca em uma nota Markdown atômica consolidada com backlinks
func CompileTopicNote(vaultRoot, requestedPath, title, topic string, sources []CompiledSource, tags []string, relations []parser.EdgeConnection, overwrite bool) (*NoteResult, error) {
	cleanTopic := strings.TrimSpace(topic)
	if cleanTopic == "" {
		return nil, fmt.Errorf("tópico para compilação não pode ser vazio")
	}

	noteTitle := strings.TrimSpace(title)
	if noteTitle == "" {
		noteTitle = fmt.Sprintf("Síntese: %s", cleanTopic)
	}

	allTags := append([]string{"compiled", "wiki"}, tags...)

	var body strings.Builder
	body.WriteString(fmt.Sprintf("# %s\n\n", noteTitle))
	body.WriteString(fmt.Sprintf("> 🧠 **Compilação de Conhecimento (Compile-not-Retrieve)**\n> Tópico investigado: `%s`\n> Fontes agregadas: %d documento(s).\n\n", cleanTopic, len(sources)))

	body.WriteString("## Visão Geral e Síntese\n\n")
	if len(sources) == 0 {
		body.WriteString("Nenhum documento prévio foi encontrado para este tópico. Esta nota serve como ponto de partida (*stub*).\n\n")
	} else {
		body.WriteString("Abaixo estão os pontos de destaque consolidados a partir da recuperação de conhecimento no vault:\n\n")
		for i, src := range sources {
			docName := src.Title
			if docName == "" {
				docName = filepath.Base(src.DocPath)
				docName = strings.TrimSuffix(docName, filepath.Ext(docName))
			}
			snippet := strings.TrimSpace(src.Content)
			if len(snippet) > 280 {
				snippet = snippet[:280] + "..."
			}
			body.WriteString(fmt.Sprintf("### %d. [[%s]]\n", i+1, docName))
			if src.Score > 0 {
				body.WriteString(fmt.Sprintf("*Relevância RRF / Score: %.4f*\n\n", src.Score))
			}
			body.WriteString(fmt.Sprintf("> %s\n\n", strings.ReplaceAll(snippet, "\n", "\n> ")))
		}
	}

	// Seção de Backlinks Explícitos
	body.WriteString("## Fontes e Backlinks\n\n")
	seenDocs := make(map[string]bool)
	hasLinks := false
	for _, src := range sources {
		docName := src.Title
		if docName == "" {
			docName = filepath.Base(src.DocPath)
			docName = strings.TrimSuffix(docName, filepath.Ext(docName))
		}
		if docName != "" && !seenDocs[docName] {
			seenDocs[docName] = true
			hasLinks = true
			body.WriteString(fmt.Sprintf("- [[rel:derived_from:%s]]\n", docName))
		}
	}
	if !hasLinks {
		body.WriteString("- Nenhuma fonte externa vinculada.\n")
	}

	// Adiciona relações fornecidas
	var finalRelations []parser.EdgeConnection
	finalRelations = append(finalRelations, relations...)

	return WriteAtomicNote(vaultRoot, requestedPath, noteTitle, body.String(), allTags, nil, "summary", finalRelations, overwrite)
}
