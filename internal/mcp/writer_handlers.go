package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/compiler"
	"github.com/FelipeMiiller/my-memory/internal/parser"
)

// RawRelation auxilia o unmarshaling de relações em tools/call
type RawRelation struct {
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

// NewMemoryWriteNoteHandler cria o handler executor para memory_write_note
func NewMemoryWriteNoteHandler(engine *compiler.SyncEngine, defaultVaultRoot string) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var raw struct {
			Path       string        `json:"path"`
			Title      string        `json:"title"`
			Content    string        `json:"content"`
			Tags       []string      `json:"tags"`
			Aliases    []string      `json:"aliases"`
			NoteType   string        `json:"note_type"`
			Relations  []RawRelation `json:"relations"`
			Overwrite  bool          `json:"overwrite"`
			Repository string        `json:"repository"`
			VaultRoot  string        `json:"vault_root"`
		}

		if len(args) == 0 {
			return nil, NewError(CodeInvalidParams, "Parâmetros obrigatórios ausentes ('path' e 'content')", nil)
		}

		if err := json.Unmarshal(args, &raw); err != nil {
			return nil, NewError(CodeInvalidParams, fmt.Sprintf("Erro ao decodificar argumentos de memory_write_note: %v", err), nil)
		}

		if strings.TrimSpace(raw.Path) == "" {
			return nil, NewError(CodeInvalidParams, "O parâmetro 'path' é obrigatório", nil)
		}
		if strings.TrimSpace(raw.Content) == "" {
			return nil, NewError(CodeInvalidParams, "O parâmetro 'content' não pode ser vazio", nil)
		}

		vRoot := defaultVaultRoot
		if strings.TrimSpace(raw.VaultRoot) != "" {
			vRoot = raw.VaultRoot
		}
		if vRoot == "" {
			vRoot = "."
		}

		var parserEdges []parser.EdgeConnection
		for _, r := range raw.Relations {
			target := strings.TrimSpace(r.Target)
			if target != "" {
				rel := strings.TrimSpace(r.Relation)
				if rel == "" {
					rel = "links_to"
				}
				parserEdges = append(parserEdges, parser.EdgeConnection{
					Target:          target,
					Relation:        rel,
					EpistemicStatus: "EXTRACTED",
					Weight:          1.0,
				})
			}
		}

		noteRes, idxRes, err := compiler.WriteAndSyncNote(
			ctx,
			engine,
			vRoot,
			raw.Path,
			raw.Title,
			raw.Content,
			raw.Tags,
			raw.Aliases,
			raw.NoteType,
			parserEdges,
			raw.Overwrite,
		)
		if err != nil {
			return CallToolResult{
				IsError: true,
				Content: []ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("❌ Erro ao gravar nota: %v", err),
				}},
			}, nil
		}

		var sb strings.Builder
		sb.WriteString("## ✅ Nota Gravada e Sincronizada com Sucesso\n\n")
		sb.WriteString(fmt.Sprintf("- **Caminho:** `%s`\n", noteRes.Path))
		sb.WriteString(fmt.Sprintf("- **Título:** %s\n", noteRes.Title))
		sb.WriteString(fmt.Sprintf("- **Tipo:** `%s`\n", noteRes.Type))
		sb.WriteString(fmt.Sprintf("- **SHA-256:** `%s`\n", noteRes.SHA256))
		if len(noteRes.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("- **Tags:** %s\n", strings.Join(noteRes.Tags, ", ")))
		}
		if len(noteRes.Aliases) > 0 {
			sb.WriteString(fmt.Sprintf("- **Aliases:** %s\n", strings.Join(noteRes.Aliases, ", ")))
		}
		sb.WriteString(fmt.Sprintf("- **Arestas de Grafo:** %d conexões extraídas\n", noteRes.EdgesCount))
		if idxRes != nil {
			sb.WriteString(fmt.Sprintf("- **Status da Indexação:** `%s` (duração: %v)\n", idxRes.Action, idxRes.Duration))
		}
		if noteRes.Overwritten {
			sb.WriteString("- **Sobrescrita:** Sim (arquivo pré-existente substituído)\n")
		}

		return NewTextResult(strings.TrimSpace(sb.String())), nil
	}
}

// NewMemoryAppendSectionHandler cria o handler executor para memory_append_section
func NewMemoryAppendSectionHandler(engine *compiler.SyncEngine, defaultVaultRoot string) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var raw struct {
			Path            string `json:"path"`
			Heading         string `json:"heading"`
			Content         string `json:"content"`
			CreateIfMissing *bool  `json:"create_if_missing"`
			Repository      string `json:"repository"`
			VaultRoot       string `json:"vault_root"`
		}

		if len(args) == 0 {
			return nil, NewError(CodeInvalidParams, "Parâmetros obrigatórios ausentes ('path', 'heading', 'content')", nil)
		}

		if err := json.Unmarshal(args, &raw); err != nil {
			return nil, NewError(CodeInvalidParams, fmt.Sprintf("Erro ao decodificar argumentos de memory_append_section: %v", err), nil)
		}

		if strings.TrimSpace(raw.Path) == "" {
			return nil, NewError(CodeInvalidParams, "O parâmetro 'path' é obrigatório", nil)
		}
		if strings.TrimSpace(raw.Heading) == "" {
			return nil, NewError(CodeInvalidParams, "O parâmetro 'heading' é obrigatório", nil)
		}
		if strings.TrimSpace(raw.Content) == "" {
			return nil, NewError(CodeInvalidParams, "O parâmetro 'content' não pode ser vazio", nil)
		}

		createIfMissing := true
		if raw.CreateIfMissing != nil {
			createIfMissing = *raw.CreateIfMissing
		}

		vRoot := defaultVaultRoot
		if strings.TrimSpace(raw.VaultRoot) != "" {
			vRoot = raw.VaultRoot
		}
		if vRoot == "" {
			vRoot = "."
		}

		noteRes, idxRes, err := compiler.AppendAndSyncSection(
			ctx,
			engine,
			vRoot,
			raw.Path,
			raw.Heading,
			raw.Content,
			createIfMissing,
		)
		if err != nil {
			return CallToolResult{
				IsError: true,
				Content: []ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("❌ Erro ao anexar seção: %v", err),
				}},
			}, nil
		}

		var sb strings.Builder
		sb.WriteString("## 📝 Seção Anexada e Sincronizada com Sucesso\n\n")
		sb.WriteString(fmt.Sprintf("- **Caminho:** `%s`\n", noteRes.Path))
		sb.WriteString(fmt.Sprintf("- **Cabeçalho:** `%s`\n", raw.Heading))
		sb.WriteString(fmt.Sprintf("- **Título da Nota:** %s\n", noteRes.Title))
		sb.WriteString(fmt.Sprintf("- **SHA-256 Atualizado:** `%s`\n", noteRes.SHA256))
		sb.WriteString(fmt.Sprintf("- **Total de Arestas:** %d conexões\n", noteRes.EdgesCount))
		if idxRes != nil {
			sb.WriteString(fmt.Sprintf("- **Status da Indexação:** `%s` (duração: %v)\n", idxRes.Action, idxRes.Duration))
		}

		return NewTextResult(strings.TrimSpace(sb.String())), nil
	}
}

// NewMemoryCompileNoteHandler cria o handler executor para memory_compile_note (Compile-not-Retrieve)
func NewMemoryCompileNoteHandler(engine *compiler.SyncEngine, searchFn AdvancedSearchFunc, defaultVaultRoot string) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var raw struct {
			Topic      string        `json:"topic"`
			TargetPath string        `json:"target_path"`
			Title      string        `json:"title"`
			SearchMode string        `json:"search_mode"`
			Limit      int           `json:"limit"`
			Tags       []string      `json:"tags"`
			Relations  []RawRelation `json:"relations"`
			Overwrite  bool          `json:"overwrite"`
			Repository string        `json:"repository"`
			VaultRoot  string        `json:"vault_root"`
		}

		if len(args) == 0 {
			return nil, NewError(CodeInvalidParams, "Parâmetros obrigatórios ausentes ('topic' e 'target_path')", nil)
		}

		if err := json.Unmarshal(args, &raw); err != nil {
			return nil, NewError(CodeInvalidParams, fmt.Sprintf("Erro ao decodificar argumentos de memory_compile_note: %v", err), nil)
		}

		if strings.TrimSpace(raw.Topic) == "" {
			return nil, NewError(CodeInvalidParams, "O parâmetro 'topic' é obrigatório", nil)
		}
		if strings.TrimSpace(raw.TargetPath) == "" {
			return nil, NewError(CodeInvalidParams, "O parâmetro 'target_path' é obrigatório", nil)
		}

		searchMode := raw.SearchMode
		if searchMode == "" {
			searchMode = "hybrid"
		}
		limit := raw.Limit
		if limit <= 0 {
			limit = 5
		}

		vRoot := defaultVaultRoot
		if strings.TrimSpace(raw.VaultRoot) != "" {
			vRoot = raw.VaultRoot
		}
		if vRoot == "" {
			vRoot = "."
		}

		var sources []compiler.CompiledSource
		if searchFn != nil {
			searchResults, err := searchFn(ctx, SearchParams{
				Repo:  raw.Repository,
				Query: raw.Topic,
				Mode:  searchMode,
				Limit: limit,
				K:     60,
			})
			if err == nil {
				for _, sr := range searchResults {
					sources = append(sources, compiler.CompiledSource{
						DocPath: sr.DocumentID,
						Title:   sr.DocumentID,
						Content: sr.Content,
						Score:   sr.Score,
					})
				}
			}
		}

		var parserEdges []parser.EdgeConnection
		for _, r := range raw.Relations {
			target := strings.TrimSpace(r.Target)
			if target != "" {
				rel := strings.TrimSpace(r.Relation)
				if rel == "" {
					rel = "links_to"
				}
				parserEdges = append(parserEdges, parser.EdgeConnection{
					Target:          target,
					Relation:        rel,
					EpistemicStatus: "EXTRACTED",
					Weight:          1.0,
				})
			}
		}

		noteRes, idxRes, err := compiler.CompileAndSyncTopicNote(
			ctx,
			engine,
			vRoot,
			raw.TargetPath,
			raw.Title,
			raw.Topic,
			sources,
			raw.Tags,
			parserEdges,
			raw.Overwrite,
		)
		if err != nil {
			return CallToolResult{
				IsError: true,
				Content: []ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("❌ Erro ao compilar nota: %v", err),
				}},
			}, nil
		}

		var sb strings.Builder
		sb.WriteString("## 🧠 Conhecimento Compilado com Sucesso (Compile-not-Retrieve)\n\n")
		sb.WriteString(fmt.Sprintf("- **Tópico:** `%s`\n", raw.Topic))
		sb.WriteString(fmt.Sprintf("- **Caminho de Destino:** `%s`\n", noteRes.Path))
		sb.WriteString(fmt.Sprintf("- **Título:** %s\n", noteRes.Title))
		sb.WriteString(fmt.Sprintf("- **Fontes Agregadas:** %d fragmento(s)\n", len(sources)))
		sb.WriteString(fmt.Sprintf("- **SHA-256:** `%s`\n", noteRes.SHA256))
		sb.WriteString(fmt.Sprintf("- **Conexões e Backlinks:** %d arestas geradas\n", noteRes.EdgesCount))
		if idxRes != nil {
			sb.WriteString(fmt.Sprintf("- **Status da Indexação:** `%s` (duração: %v)\n", idxRes.Action, idxRes.Duration))
		}

		return NewTextResult(strings.TrimSpace(sb.String())), nil
	}
}

// SetCompilerEngine registra ou atualiza os executores de escrita e compilação no servidor MCP
func (s *Server) SetCompilerEngine(engine *compiler.SyncEngine, searchFn AdvancedSearchFunc, defaultVaultRoot string) {
	s.RegisterToolHandler("memory_write_note", NewMemoryWriteNoteHandler(engine, defaultVaultRoot))
	s.RegisterToolHandler("memory_append_section", NewMemoryAppendSectionHandler(engine, defaultVaultRoot))
	s.RegisterToolHandler("memory_compile_note", NewMemoryCompileNoteHandler(engine, searchFn, defaultVaultRoot))
}
