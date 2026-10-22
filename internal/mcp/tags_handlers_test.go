package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// tagAccessFixture monta o inventário de tags e os documentos de um vault
// pequeno, com uma tag que nomeia um documento (ADR-055) e uma tag órfã.
func tagAccessFixture() VaultAccess {
	docs := []store.VaultDocument{
		{ID: "doc-1", Path: "docs/a.md", Title: "A"},
		{ID: "doc-2", Path: "docs/b.md", Title: "B"},
	}

	tags := []store.VaultTag{
		{Tag: "Inbox", Key: "inbox", Count: 2, ResolvedDocID: "doc-1", ResolvedTitle: "A"},
		{Tag: "projeto/alpha", Key: "projeto/alpha", Count: 1},
	}

	return VaultAccess{
		ListTags: func(_ context.Context, _ string, filter store.TagFilter) ([]store.VaultTag, error) {
			if filter.DocumentID == "" {
				return tags, nil
			}
			filtered := []store.VaultTag{}
			for _, tag := range tags {
				if filter.DocumentID == "doc-1" && tag.Key == "inbox" {
					filtered = append(filtered, tag)
				}
			}
			return filtered, nil
		},
		FindDocument: func(_ context.Context, _, path string) (store.VaultDocument, error) {
			for _, doc := range docs {
				if doc.Path == path {
					return doc, nil
				}
			}
			return store.VaultDocument{}, errors.New("nota '" + path + "' não está no índice")
		},
		FindDocumentsByTag: func(_ context.Context, _, tag string) ([]store.VaultDocument, error) {
			if store.CanonicalTagKey(tag) == "inbox" {
				return docs, nil
			}
			return []store.VaultDocument{}, nil
		},
	}
}

// TestMemoryGetTags_InventarioComContagens prova que a tool devolve o
// inventário com a contagem de documentos, no texto legível e no payload.
func TestMemoryGetTags_InventarioComContagens(t *testing.T) {
	ctx := context.Background()
	access := tagAccessFixture()

	res, err := NewMemoryGetTagsHandler(access.ListTags, access.FindDocument)(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	text := toolResultText(t, res)
	for _, want := range []string{"Inbox", "projeto/alpha", "2 documento(s)"} {
		if !strings.Contains(text, want) {
			t.Errorf("esperava '%s' no inventário, obteve:\n%s", want, text)
		}
	}

	payload, err := json.Marshal(res.(CallToolResult).StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}
	expected := `[{"tag":"Inbox","key":"inbox","count":2,"resolved_doc_id":"doc-1","resolved_title":"A"},` +
		`{"tag":"projeto/alpha","key":"projeto/alpha","count":1}]`
	if string(payload) != expected {
		t.Errorf("payload inesperado:\n obtive: %s\n esperava: %s", payload, expected)
	}
}

// TestMemoryGetTags_PathFiltraAsTagsDoDocumento prova que o `path` restringe o
// inventário às tags daquele documento.
func TestMemoryGetTags_PathFiltraAsTagsDoDocumento(t *testing.T) {
	ctx := context.Background()
	access := tagAccessFixture()

	res, err := NewMemoryGetTagsHandler(access.ListTags, access.FindDocument)(ctx, json.RawMessage(`{"path": "docs/a.md"}`))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	text := toolResultText(t, res)
	if !strings.Contains(text, "Inbox") {
		t.Errorf("esperava a tag do documento, obteve:\n%s", text)
	}
	if strings.Contains(text, "projeto/alpha") {
		t.Errorf("o filtro por path trouxe tag de outro documento:\n%s", text)
	}

	payload, err := json.Marshal(res.(CallToolResult).StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}
	if strings.Contains(string(payload), "projeto/alpha") {
		t.Errorf("o payload trouxe tag de outro documento: %s", payload)
	}
}

// TestMemoryGetTags_VaultSemTagsNaoErra: um vault sem tags é uma resposta
// válida — e o payload viaja como `[]`, nunca como `null` (TAG-32).
func TestMemoryGetTags_VaultSemTagsNaoErra(t *testing.T) {
	ctx := context.Background()

	handler := NewMemoryGetTagsHandler(func(_ context.Context, _ string, _ store.TagFilter) ([]store.VaultTag, error) {
		return nil, nil
	}, nil)

	res, err := handler(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("vault sem tags não é erro: %v", err)
	}

	text := toolResultText(t, res)
	if !strings.Contains(text, "Nenhuma tag") {
		t.Errorf("o texto precisa explicar o resultado, obteve:\n%s", text)
	}

	payload, err := json.Marshal(res.(CallToolResult).StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}
	if string(payload) != "[]" {
		t.Errorf("esperava '[]' no payload, obteve: %s", payload)
	}
	if strings.Contains(string(payload), "null") {
		t.Errorf("o payload serializou null: %s", payload)
	}
}

// TestMemoryGetTag_ContagemEDocumentos prova que a tool devolve a contagem e a
// lista de documentos da tag.
func TestMemoryGetTag_ContagemEDocumentos(t *testing.T) {
	ctx := context.Background()
	access := tagAccessFixture()

	res, err := NewMemoryGetTagHandler(access.FindDocumentsByTag, access.ListTags)(ctx, json.RawMessage(`{"name": "#Inbox"}`))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	text := toolResultText(t, res)
	for _, want := range []string{"Inbox", "2 documento(s)", "docs/a.md", "docs/b.md"} {
		if !strings.Contains(text, want) {
			t.Errorf("esperava '%s' na resposta, obteve:\n%s", want, text)
		}
	}

	payload, err := json.Marshal(res.(CallToolResult).StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}
	for _, want := range []string{`"key":"inbox"`, `"count":2`, `"path":"docs/a.md"`, `"path":"docs/b.md"`} {
		if !strings.Contains(string(payload), want) {
			t.Errorf("esperava %s no payload, obteve: %s", want, payload)
		}
	}
}

// TestMemoryGetTag_TagInexistenteNaoErra: tag sem nota é resposta válida.
func TestMemoryGetTag_TagInexistenteNaoErra(t *testing.T) {
	ctx := context.Background()
	access := tagAccessFixture()

	res, err := NewMemoryGetTagHandler(access.FindDocumentsByTag, access.ListTags)(ctx, json.RawMessage(`{"name": "inexistente"}`))
	if err != nil {
		t.Fatalf("tag sem nota não é erro: %v", err)
	}

	text := toolResultText(t, res)
	if !strings.Contains(text, "não aparece em nenhum documento") {
		t.Errorf("o texto precisa explicar o resultado, obteve:\n%s", text)
	}

	payload, err := json.Marshal(res.(CallToolResult).StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}
	if !strings.Contains(string(payload), `"count":0`) {
		t.Errorf("esperava contagem 0, obteve: %s", payload)
	}
	if !strings.Contains(string(payload), `"documents":[]`) {
		t.Errorf("a lista de documentos deveria ser [], obteve: %s", payload)
	}
	if strings.Contains(string(payload), "null") {
		t.Errorf("o payload serializou null: %s", payload)
	}
}

// TestMemoryGetTag_NameObrigatorio: sem 'name' a tool erra com parâmetro
// inválido, em vez de devolver um inventário inteiro sem querer.
func TestMemoryGetTag_NameObrigatorio(t *testing.T) {
	ctx := context.Background()
	access := tagAccessFixture()

	_, err := NewMemoryGetTagHandler(access.FindDocumentsByTag, access.ListTags)(ctx, json.RawMessage(`{}`))
	mcpErr, ok := err.(*Error)
	if !ok || mcpErr.Code != CodeInvalidParams {
		t.Fatalf("esperava CodeInvalidParams sem 'name', obteve %v", err)
	}
	if !strings.Contains(mcpErr.Message, "name") {
		t.Errorf("a mensagem precisa dizer qual parâmetro falta, obteve %q", mcpErr.Message)
	}
}

// TestToolsDeTag_EstaoNaListaDeParidade: as duas tools de tag são parte da
// superfície de leitura do vault e não podem escapar do gate de documentação.
func TestToolsDeTag_EstaoNaListaDeParidade(t *testing.T) {
	listed := make(map[string]bool, len(vaultToolsParity))
	for _, tool := range vaultToolsParity {
		listed[tool.Name] = true
	}

	for _, want := range []string{ToolMemoryGetTags.Name, ToolMemoryGetTag.Name} {
		if !listed[want] {
			t.Errorf("a ferramenta %s não está na lista vaultToolsParity", want)
		}
	}
}

// TestToolsDeTag_ExecutoresRegistrados: com o acesso ao vault configurado, as
// duas tools têm executor — uma tool sem executor responde "não encontrada".
func TestToolsDeTag_ExecutoresRegistrados(t *testing.T) {
	srv := NewServer("test", "1.0", nil, nil, nil)
	srv.SetVaultAccess(tagAccessFixture())

	for _, name := range []string{ToolMemoryGetTags.Name, ToolMemoryGetTag.Name} {
		if _, ok := srv.toolHandlers[name]; !ok {
			t.Errorf("a ferramenta %s foi registrada sem executor", name)
		}
	}
}
