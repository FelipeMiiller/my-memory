package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/parser"
)

// NoteResult consolida metadados e estatísticas de uma nota escrita
type NoteResult struct {
	Path        string   `json:"path"`     // Caminho relativo ao vault
	AbsPath     string   `json:"abs_path"` // Caminho absoluto no disco
	SHA256      string   `json:"sha256"`   // Hash do conteúdo gravado
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Tags        []string `json:"tags"`
	Aliases     []string `json:"aliases"`
	Outgoing    []string `json:"outgoing_links"`
	EdgesCount  int      `json:"edges_count"`
	Overwritten bool     `json:"overwritten"`
}

// SafeResolvePath confina estritamente o caminho solicitado dentro dos limites do vault
func SafeResolvePath(vaultRoot, requestedPath string) (string, string, error) {
	if strings.TrimSpace(vaultRoot) == "" {
		vaultRoot = "."
	}
	absVault, err := filepath.Abs(vaultRoot)
	if err != nil {
		return "", "", fmt.Errorf("falha ao resolver caminho absoluto da raiz do vault: %w", err)
	}
	cleanVault := filepath.Clean(absVault)

	trimmedReq := strings.TrimSpace(requestedPath)
	if trimmedReq == "" {
		return "", "", fmt.Errorf("caminho da nota não pode ser vazio")
	}

	var targetAbs string
	if filepath.IsAbs(trimmedReq) {
		targetAbs = filepath.Clean(trimmedReq)
	} else {
		// Converte barras invertidas para compatibilidade cross-platform (ex: Windows separators no Linux)
		normalizedReq := filepath.FromSlash(strings.ReplaceAll(trimmedReq, "\\", "/"))
		targetAbs = filepath.Clean(filepath.Join(cleanVault, normalizedReq))
	}

	// Garante extensão .md
	if filepath.Ext(targetAbs) == "" {
		targetAbs += ".md"
	}

	// Validação de fronteira do sandbox (evita path traversal "../")
	if targetAbs != cleanVault && !strings.HasPrefix(targetAbs, cleanVault+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path traversal detectado: caminho '%s' extrapola os limites do vault '%s'", requestedPath, vaultRoot)
	}

	relPath, err := filepath.Rel(cleanVault, targetAbs)
	if err != nil {
		return "", "", fmt.Errorf("erro ao calcular caminho relativo: %w", err)
	}
	relPath = filepath.ToSlash(strings.ReplaceAll(relPath, "\\", "/"))

	return targetAbs, relPath, nil
}

// FormatFrontmatter gera o cabeçalho YAML padronizado
func FormatFrontmatter(title, noteType string, tags, aliases []string, now time.Time) string {
	var b strings.Builder
	b.WriteString("---\n")

	cleanTitle := strings.TrimSpace(title)
	cleanTitle = strings.ReplaceAll(cleanTitle, `"`, `\"`)
	b.WriteString(fmt.Sprintf("title: \"%s\"\n", cleanTitle))

	if noteType == "" {
		noteType = "concept"
	}
	b.WriteString(fmt.Sprintf("type: %s\n", strings.ToLower(noteType)))

	if len(tags) > 0 {
		b.WriteString("tags:\n")
		seenTags := make(map[string]bool)
		for _, t := range tags {
			norm := strings.TrimPrefix(strings.TrimSpace(t), "#")
			if norm != "" && !seenTags[norm] {
				seenTags[norm] = true
				b.WriteString(fmt.Sprintf("  - %s\n", norm))
			}
		}
	}

	if len(aliases) > 0 {
		b.WriteString("aliases:\n")
		seenAliases := make(map[string]bool)
		for _, a := range aliases {
			cleanAlias := strings.TrimSpace(a)
			if cleanAlias != "" && !seenAliases[cleanAlias] {
				seenAliases[cleanAlias] = true
				cleanAlias = strings.ReplaceAll(cleanAlias, `"`, `\"`)
				b.WriteString(fmt.Sprintf("  - \"%s\"\n", cleanAlias))
			}
		}
	}

	ts := now.Format(time.RFC3339)
	b.WriteString(fmt.Sprintf("created_at: %s\n", ts))
	b.WriteString(fmt.Sprintf("updated_at: %s\n", ts))
	b.WriteString("---\n")

	return b.String()
}

// WriteAtomicNote grava uma nova nota atômica no vault garantindo UTF-8 sem BOM e frontmatter
func WriteAtomicNote(vaultRoot, requestedPath, title, content string, tags, aliases []string, noteType string, relations []parser.EdgeConnection, overwrite bool) (*NoteResult, error) {
	absPath, relPath, err := SafeResolvePath(vaultRoot, requestedPath)
	if err != nil {
		return nil, err
	}

	// Verifica se o arquivo já existe
	existed := false
	if _, statErr := os.Stat(absPath); statErr == nil {
		if !overwrite {
			return nil, fmt.Errorf("arquivo já existe: %s (utilize overwrite: true para substituir)", relPath)
		}
		existed = true
	}

	now := time.Now()
	cleanContent := strings.TrimSpace(content)

	var fullBody strings.Builder

	// Se o conteúdo já começar com frontmatter YAML, preservamos ou mesclamos
	if strings.HasPrefix(cleanContent, "---\n") || strings.HasPrefix(cleanContent, "---\r\n") {
		fullBody.WriteString(cleanContent)
	} else {
		// Monta frontmatter padronizado
		noteTitle := title
		if noteTitle == "" {
			baseName := filepath.Base(relPath)
			noteTitle = strings.TrimSuffix(baseName, filepath.Ext(baseName))
		}
		fullBody.WriteString(FormatFrontmatter(noteTitle, noteType, tags, aliases, now))
		fullBody.WriteString("\n")
		fullBody.WriteString(cleanContent)
	}

	// Anexa conexões tipadas se fornecidas e não existentes no corpo
	if len(relations) > 0 {
		existingBody := fullBody.String()
		var relsToAdd []string
		for _, rel := range relations {
			target := strings.TrimSpace(rel.Target)
			if target == "" {
				continue
			}
			relationType := strings.TrimSpace(rel.Relation)
			if relationType == "" {
				relationType = "links_to"
			}
			linkStr := fmt.Sprintf("[[rel:%s:%s]]", relationType, target)
			if !strings.Contains(existingBody, linkStr) && !strings.Contains(existingBody, fmt.Sprintf("[[%s]]", target)) {
				relsToAdd = append(relsToAdd, fmt.Sprintf("- %s", linkStr))
			}
		}

		if len(relsToAdd) > 0 {
			fullBody.WriteString("\n\n## Conexões e Relações\n")
			for _, r := range relsToAdd {
				fullBody.WriteString(r)
				fullBody.WriteString("\n")
			}
		}
	}

	finalBytes := []byte(fullBody.String())

	// Garante diretório pai
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return nil, fmt.Errorf("falha ao criar diretório pai '%s': %w", filepath.Dir(absPath), err)
	}

	// Grava no disco
	if err := os.WriteFile(absPath, finalBytes, 0644); err != nil {
		return nil, fmt.Errorf("falha ao gravar arquivo '%s': %w", absPath, err)
	}

	// Extrai conexões e metadados para feedback imediato
	conns := parser.ExtractConnections(string(finalBytes))
	hasher := sha256.New()
	hasher.Write(finalBytes)
	shaHex := hex.EncodeToString(hasher.Sum(nil))

	noteTitle := title
	if noteTitle == "" && conns.Frontmatter != nil && conns.Frontmatter.Title != "" {
		noteTitle = conns.Frontmatter.Title
	}
	if noteTitle == "" {
		baseName := filepath.Base(relPath)
		noteTitle = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}

	resType := noteType
	if resType == "" && conns.Frontmatter != nil && conns.Frontmatter.Properties != nil {
		if tVal, ok := conns.Frontmatter.Properties["type"].(string); ok && tVal != "" {
			resType = tVal
		}
	}
	if resType == "" {
		resType = "concept"
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
		Overwritten: existed,
	}, nil
}
