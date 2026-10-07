package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/FelipeMiiller/my-memory/internal/parser"
)

// stripBOM remove o Byte Order Mark (U+FEFF) do início de uma string.
//
// O PowerShell injeta BOM em pipes e em Set-Content no Windows, e
// strings.TrimSpace NÃO remove U+FEFF porque unicode.IsSpace(U+FEFF) é false
// (BOM é categoria Cf, não White_Space). Isso quebrava a detecção de
// frontmatter em WriteAtomicNote e produzia frontmatter duplicado no arquivo.
//
// Ver ISSUE-013.
func stripBOM(s string) string {
	return strings.TrimPrefix(s, "\ufeff")
}

// splitFrontmatter separa um bloco YAML de abertura no início do conteúdo.
// Retorna (blocoYaml, resto, true) quando existe frontmatter válido e
// fechado por um segundo "---".
func splitFrontmatter(content string) (string, string, bool) {
	if !strings.HasPrefix(content, "---") {
		return "", content, false
	}
	rest := content[3:]
	rest = strings.TrimPrefix(rest, "\r")
	rest = strings.TrimPrefix(rest, "\n")
	if rest == "" {
		return "", content, false
	}
	// Procura a linha de fechamento "---" (ignorando prefixo de '-').
	for i, line := range strings.Split(rest, "\n") {
		trimmed := strings.TrimRight(strings.TrimLeft(line, " \t"), "\r")
		if strings.HasPrefix(trimmed, "---") {
			before := rest
			if i > 0 {
				before = strings.Join(strings.Split(rest, "\n")[:i], "\n")
			}
			afterLines := strings.Split(rest, "\n")
			after := ""
			if i+1 < len(afterLines) {
				after = strings.Join(afterLines[i+1:], "\n")
			}
			return before, strings.TrimLeft(after, "\r\n"), true
		}
	}
	return "", content, false
}

// mergeStringLists uniona duas listas de strings normalizando o prefixo '#'
// e removendo duplicatas, preservando a ordem (existentes primeiro).
func mergeStringLists(existing any, incoming []string) []string {
	out := []string{}
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimPrefix(strings.TrimSpace(s), "#")
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	switch v := existing.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				add(s)
			}
		}
	case []string:
		for _, s := range v {
			add(s)
		}
	case string:
		add(v)
	}
	for _, s := range incoming {
		add(s)
	}
	return out
}

// mergeFrontmatter aplica os argumentos explícitos da CLI sobre um frontmatter
// já presente no conteúdo.
//
// Antes desta função, quando o conteúdo trazia frontmatter próprio os flags
// --title/--tags/--type/--aliases eram silenciosamente descartados: o comando
// reportava sucesso com os valores passados, mas gravava o frontmatter original
// intacto. Isso quebra o fluxo "IA gera o markdown completo -> mem note create
// --title X --tags Y".
//
// Chaves desconhecidas (category, summary, etc.) são preservadas.
// Ver ISSUE-013.
func mergeFrontmatter(fmText string, title, noteType string, tags, aliases []string, now time.Time) string {
	fields := map[string]any{}
	if err := yaml.Unmarshal([]byte(fmText), &fields); err != nil || len(fields) == 0 {
		// Frontmatter não parseável: preserva o original em vez de destruir
		// metadados que não entendemos.
		return fmText
	}

	if s := strings.TrimSpace(title); s != "" {
		fields["title"] = s
	}
	if s := strings.TrimSpace(noteType); s != "" && s != "concept" {
		fields["type"] = s
	}
	if len(tags) > 0 {
		fields["tags"] = mergeStringLists(fields["tags"], tags)
	}
	if len(aliases) > 0 {
		fields["aliases"] = mergeStringLists(fields["aliases"], aliases)
	}
	if _, ok := fields["created_at"]; !ok {
		fields["created_at"] = now.Format(time.RFC3339)
	}
	fields["updated_at"] = now.Format(time.RFC3339)

	out, err := yaml.Marshal(fields)
	if err != nil {
		return fmText
	}
	return "---\n" + string(out) + "---\n"
}

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
	cleanContent := strings.TrimSpace(stripBOM(content))

	var fullBody strings.Builder

	// Se o conteúdo já tiver frontmatter, MESCLAMOS os argumentos explícitos da
	// CLI sobre ele (preservando chaves desconhecidas) em vez de apenas
	// preservá-lo e descartar --title/--tags/--type.
	if fmText, body, ok := splitFrontmatter(cleanContent); ok {
		fullBody.WriteString(mergeFrontmatter(fmText, title, noteType, tags, aliases, now))
		fullBody.WriteString(body)
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
