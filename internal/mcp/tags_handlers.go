package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// TagQueryResult é o structuredContent de memory_get_tag.
//
// A forma espelha o payload JSON de `mem tag` (tag, key, count, documents) e
// acrescenta o documento que a tag nomeia (ADR-055): a tag órfã — a que não
// nomeia nenhum documento — aparece no inventário sem resolução, e continua
// sendo uma resposta válida.
type TagQueryResult struct {
	Tag       string                `json:"tag"`
	Key       string                `json:"key"`
	Count     int                   `json:"count"`
	Documents []store.VaultDocument `json:"documents"`
	// ResolvedDocID/ResolvedTitle são o documento que a tag nomeia (ADR-055).
	ResolvedDocID string `json:"resolved_doc_id,omitempty"`
	ResolvedTitle string `json:"resolved_title,omitempty"`
}

// NewMemoryGetTagsHandler cria o executor da ferramenta MCP memory_get_tags.
//
// A tool devolve o inventário de tags do vault com a contagem de documentos que
// carregam cada uma. Com `path`, devolve só as tags daquele documento.
//
// Vault sem tags não é erro: a resposta é um inventário vazio, com `[]` no
// structuredContent e uma frase que explica o resultado no content.
func NewMemoryGetTagsHandler(listFn VaultTagListFunc, docFn VaultDocumentFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		params, err := decodeVaultParams(args)
		if err != nil {
			return nil, err
		}

		if listFn == nil {
			return nil, notConfiguredError("memory_get_tags")
		}

		filter := store.TagFilter{}
		if params.Path != "" {
			if docFn == nil {
				return nil, notConfiguredError("memory_get_tags")
			}
			doc, err := docFn(ctx, params.Repository, params.Path)
			if err != nil {
				return nil, NewError(CodeInvalidParams, fmt.Sprintf("Falha ao resolver '%s' no índice", params.Path), err.Error())
			}
			filter.DocumentID = doc.ID
		}

		tags, err := listFn(ctx, params.Repository, filter)
		if err != nil {
			return nil, NewError(CodeInternalError, "Falha ao listar as tags do vault", err.Error())
		}
		if tags == nil {
			tags = []store.VaultTag{}
		}

		// structuredContent: [{tag: string, key: string, count: number}]
		if len(tags) == 0 {
			scope := "o vault indexado"
			if params.Path != "" {
				scope = fmt.Sprintf("a nota '%s'", params.Path)
			}
			return NewStructuredResult(
				fmt.Sprintf("Nenhuma tag encontrada em %s. Um documento sem tags (ou um vault ainda sem índice de tags) é uma resposta válida, não um erro.", scope),
				tags), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("### 🏷️ Tags do vault (%d)\n\n", len(tags)))
		for _, tag := range tags {
			sb.WriteString(fmt.Sprintf("- `%s` — %d documento(s)", tag.Tag, tag.Count))
			if tag.ResolvedTitle != "" {
				sb.WriteString(fmt.Sprintf(" — nomeia `%s`", tag.ResolvedTitle))
			}
			sb.WriteString("\n")
		}

		return NewStructuredResult(strings.TrimRight(sb.String(), "\n"), tags), nil
	}
}

// NewMemoryGetTagHandler cria o executor da ferramenta MCP memory_get_tag.
//
// A tag pedida casa a tag e as descendentes (`inbox` casa `inbox` e
// `inbox/to-read`) — a mesma semântica do Obsidian e de `mem tag`.
//
// Tag que não existe não é erro: contagem 0 e lista vazia.
func NewMemoryGetTagHandler(docsFn VaultTagDocumentsFunc, listFn VaultTagListFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		params, err := decodeVaultParams(args)
		if err != nil {
			return nil, err
		}

		if params.Name == "" {
			return nil, NewError(CodeInvalidParams, "Parâmetro 'name' é obrigatório para memory_get_tag", nil)
		}

		if docsFn == nil {
			return nil, notConfiguredError("memory_get_tag")
		}

		documents, err := docsFn(ctx, params.Repository, params.Name)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Falha ao buscar documentos pela tag '%s'", params.Name), err.Error())
		}
		if documents == nil {
			documents = []store.VaultDocument{}
		}

		key := store.CanonicalTagKey(params.Name)
		result := TagQueryResult{
			Tag:       store.DisplayTag(params.Name),
			Key:       key,
			Count:     len(documents),
			Documents: documents,
		}

		// O documento resolvido vem do inventário: é a linha que nomeia um
		// documento (ADR-055). A tag órfã aparece no inventário sem resolução.
		if listFn != nil {
			inventory, err := listFn(ctx, params.Repository, store.TagFilter{})
			if err != nil {
				return nil, NewError(CodeInternalError, "Falha ao listar as tags do vault", err.Error())
			}
			for _, tag := range inventory {
				if tag.Key != key {
					continue
				}
				// O casing de exibição é o da linha criada primeiro (o do vault).
				result.Tag = tag.Tag
				result.ResolvedDocID = tag.ResolvedDocID
				result.ResolvedTitle = tag.ResolvedTitle
				break
			}
		}

		// structuredContent: {tag: string, key: string, count: number, documents: [...]}
		if len(result.Documents) == 0 {
			return NewStructuredResult(
				fmt.Sprintf("A tag `%s` (e suas descendentes) não aparece em nenhum documento do vault. Uma tag sem nota é uma resposta válida, não um erro.", result.Tag),
				result), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("### 🏷️ Tag `%s` — %d documento(s)\n\n", result.Tag, result.Count))
		for _, doc := range result.Documents {
			sb.WriteString(fmt.Sprintf("- `%s`", doc.Path))
			if doc.Title != "" {
				sb.WriteString(fmt.Sprintf(" — %s", doc.Title))
			}
			sb.WriteString("\n")
		}

		return NewStructuredResult(strings.TrimRight(sb.String(), "\n"), result), nil
	}
}
