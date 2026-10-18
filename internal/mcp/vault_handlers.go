package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// Superfície de leitura do vault (ADR-057) que as quatro ferramentas de paridade
// consomem: memory_list_files, memory_read_note, memory_note_outline e
// memory_file_metadata.
//
// As funções são injetadas pelo servidor MCP, que resolve o engine (ADR-040) e
// aponta para db.* ou PostgresStore.*. Nenhuma delas escreve: todas leem o
// índice, então funcionam num vault que só existe no Postgres.
type (
	// VaultListFunc lista os documentos do vault aplicando os filtros de
	// extensão e pasta. Extensão ou pasta desconhecida é lista vazia, não erro.
	VaultListFunc func(ctx context.Context, repo, ext, folder string) ([]store.VaultDocument, error)
	// VaultDocumentFunc resolve o caminho pedido no documento único que ele
	// identifica. Caminho ausente do índice, ou basename ambíguo, é erro — nunca
	// uma escolha silenciosa entre candidatos.
	VaultDocumentFunc func(ctx context.Context, repo, path string) (store.VaultDocument, error)
	// VaultContentFunc devolve o Markdown bruto do documento, remontando os
	// chunks indexados.
	VaultContentFunc func(ctx context.Context, repo, docID string) (string, error)
	// VaultHeadingsFunc extrai os títulos ATX de um Markdown na ordem do
	// documento.
	VaultHeadingsFunc func(content string) []OutlineHeading
)

// OutlineHeading é um título de uma nota com a profundidade do nível ATX
// (1 para `#`, 2 para `##`, e assim por diante).
type OutlineHeading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// VaultAccess é o conjunto completo de acessos de leitura que as ferramentas de
// paridade precisam. Uma struct só, em vez de um setter por ferramenta, porque
// as quatro leem o mesmo índice e uma configuração parcial não tem sentido.
type VaultAccess struct {
	ListDocuments VaultListFunc
	FindDocument  VaultDocumentFunc
	ReadContent   VaultContentFunc
	ParseHeadings VaultHeadingsFunc
}

// vaultToolParams são os argumentos que as quatro ferramentas compartilham.
// `repository` é opcional nas quatro: vazio significa o repositório padrão ou
// global, o mesmo idioma do resto do catálogo MCP.
type vaultToolParams struct {
	Path       string `json:"path"`
	Ext        string `json:"ext"`
	Folder     string `json:"folder"`
	Repository string `json:"repository"`
}

// decodeVaultParams faz o unmarshal dos argumentos comuns. JSON malformado é
// erro de parâmetro, não resultado vazio.
func decodeVaultParams(args json.RawMessage) (vaultToolParams, error) {
	var params vaultToolParams
	if len(args) > 0 && string(args) != "{}" {
		if err := json.Unmarshal(args, &params); err != nil {
			return params, NewError(CodeInvalidParams, "Formato JSON de argumentos inválido", err.Error())
		}
	}

	params.Path = strings.TrimSpace(params.Path)
	params.Ext = strings.TrimSpace(params.Ext)
	params.Folder = strings.TrimSpace(params.Folder)
	params.Repository = strings.TrimSpace(params.Repository)

	return params, nil
}

// notConfiguredError é o que uma ferramenta responde quando o servidor ainda não
// recebeu o acesso ao vault. É melhor um erro explícito do que uma lista vazia
// que o agente leria como "o vault não tem nada".
func notConfiguredError(toolName string) error {
	return NewError(CodeInternalError,
		fmt.Sprintf("Backend de leitura do vault não configurado para %s", toolName), nil)
}

// requirePath devolve o caminho pedido ou o erro de parâmetro obrigatório.
// Usada pelas três ferramentas que operam sobre uma nota.
func (p vaultToolParams) requirePath(toolName string) (string, error) {
	if p.Path == "" {
		return "", NewError(CodeInvalidParams, fmt.Sprintf("Parâmetro 'path' é obrigatório para %s", toolName), nil)
	}
	return p.Path, nil
}

// NewMemoryListFilesHandler cria o executor da ferramenta MCP memory_list_files
func NewMemoryListFilesHandler(listFn VaultListFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		params, err := decodeVaultParams(args)
		if err != nil {
			return nil, err
		}

		if listFn == nil {
			return nil, notConfiguredError("memory_list_files")
		}

		docs, err := listFn(ctx, params.Repository, params.Ext, params.Folder)
		if err != nil {
			return nil, NewError(CodeInternalError, "Falha ao listar documentos do vault", err.Error())
		}

		if len(docs) == 0 {
			return NewTextResult(fmt.Sprintf(
				"Nenhum documento indexado corresponde a esses filtros (ext='%s', folder='%s').",
				params.Ext, params.Folder)), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("### 📁 Documentos indexados (%d)\n\n", len(docs)))
		for _, doc := range docs {
			sb.WriteString(fmt.Sprintf("- `%s`", doc.Path))
			if doc.Title != "" {
				sb.WriteString(fmt.Sprintf(" — %s", doc.Title))
			}
			sb.WriteString("\n")
		}

		return NewTextResult(strings.TrimRight(sb.String(), "\n")), nil
	}
}

// NewMemoryReadNoteHandler cria o executor da ferramenta MCP memory_read_note
func NewMemoryReadNoteHandler(docFn VaultDocumentFunc, readFn VaultContentFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		params, err := decodeVaultParams(args)
		if err != nil {
			return nil, err
		}
		path, err := params.requirePath("memory_read_note")
		if err != nil {
			return nil, err
		}

		if docFn == nil || readFn == nil {
			return nil, notConfiguredError("memory_read_note")
		}

		doc, err := docFn(ctx, params.Repository, path)
		if err != nil {
			return nil, NewError(CodeInvalidParams, fmt.Sprintf("Falha ao resolver '%s' no índice", path), err.Error())
		}

		content, err := readFn(ctx, params.Repository, doc.ID)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Falha ao ler o conteúdo de '%s'", doc.Path), err.Error())
		}

		return NewTextResult(content), nil
	}
}

// NewMemoryNoteOutlineHandler cria o executor da ferramenta MCP memory_note_outline
func NewMemoryNoteOutlineHandler(docFn VaultDocumentFunc, readFn VaultContentFunc, headingsFn VaultHeadingsFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		params, err := decodeVaultParams(args)
		if err != nil {
			return nil, err
		}
		path, err := params.requirePath("memory_note_outline")
		if err != nil {
			return nil, err
		}

		if docFn == nil || readFn == nil || headingsFn == nil {
			return nil, notConfiguredError("memory_note_outline")
		}

		doc, err := docFn(ctx, params.Repository, path)
		if err != nil {
			return nil, NewError(CodeInvalidParams, fmt.Sprintf("Falha ao resolver '%s' no índice", path), err.Error())
		}

		content, err := readFn(ctx, params.Repository, doc.ID)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Falha ao ler o conteúdo de '%s'", doc.Path), err.Error())
		}

		headings := headingsFn(content)
		if len(headings) == 0 {
			return NewTextResult(fmt.Sprintf("A nota '%s' não tem nenhum título ATX.", doc.Path)), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("### 🧾 Outline de `%s` (%d títulos)\n\n", doc.Path, len(headings)))
		for _, heading := range headings {
			indent := strings.Repeat("  ", maxInt(heading.Level-1, 0))
			sb.WriteString(fmt.Sprintf("%s- (h%d) %s\n", indent, heading.Level, heading.Text))
		}

		return NewTextResult(strings.TrimRight(sb.String(), "\n")), nil
	}
}

// NewMemoryFileMetadataHandler cria o executor da ferramenta MCP memory_file_metadata
func NewMemoryFileMetadataHandler(docFn VaultDocumentFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		params, err := decodeVaultParams(args)
		if err != nil {
			return nil, err
		}
		path, err := params.requirePath("memory_file_metadata")
		if err != nil {
			return nil, err
		}

		if docFn == nil {
			return nil, notConfiguredError("memory_file_metadata")
		}

		doc, err := docFn(ctx, params.Repository, path)
		if err != nil {
			return nil, NewError(CodeInvalidParams, fmt.Sprintf("Falha ao resolver '%s' no índice", path), err.Error())
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("### 📄 Metadados de `%s`\n\n", doc.Path))
		sb.WriteString(fmt.Sprintf("- **Path**: `%s`\n", doc.Path))
		if doc.Title != "" {
			sb.WriteString(fmt.Sprintf("- **Title**: %s\n", doc.Title))
		}
		sb.WriteString(fmt.Sprintf("- **Size**: %d bytes\n", doc.Size))
		// mtime = 0 é documento indexado antes de a coluna existir (ADR-057).
		// Dizer "1970" seria mentira; o texto abaixo diz a verdade.
		if doc.MTime > 0 {
			sb.WriteString(fmt.Sprintf("- **Mtime**: %s (unix %d)\n",
				time.Unix(doc.MTime, 0).UTC().Format(time.RFC3339), doc.MTime))
		} else {
			sb.WriteString("- **Mtime**: desconhecido — documento indexado antes de o metadado existir (ADR-057)\n")
		}

		return NewTextResult(strings.TrimRight(sb.String(), "\n")), nil
	}
}

// SetVaultAccess registra as quatro ferramentas de leitura do vault no servidor.
// NewServer já publica os schemas; aqui entram só os executores, com o acesso
// ao índice resolvido para o engine configurado.
func (s *Server) SetVaultAccess(access VaultAccess) {
	s.RegisterToolHandler(ToolMemoryListFiles.Name, NewMemoryListFilesHandler(access.ListDocuments))
	s.RegisterToolHandler(ToolMemoryReadNote.Name, NewMemoryReadNoteHandler(access.FindDocument, access.ReadContent))
	s.RegisterToolHandler(ToolMemoryNoteOutline.Name, NewMemoryNoteOutlineHandler(access.FindDocument, access.ReadContent, access.ParseHeadings))
	s.RegisterToolHandler(ToolMemoryFileMetadata.Name, NewMemoryFileMetadataHandler(access.FindDocument))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
