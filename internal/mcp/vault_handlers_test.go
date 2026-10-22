package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// vaultToolsParity são as ferramentas de leitura do vault (ADR-057) e as de
// tags (paridade com `mem tags` e `mem tag`). A lista é explícita de propósito:
// uma ferramenta registrada e não listada aqui escaparia do gate de
// documentação.
var vaultToolsParity = []Tool{
	ToolMemoryListFiles,
	ToolMemoryReadNote,
	ToolMemoryNoteOutline,
	ToolMemoryFileMetadata,
	ToolMemoryGetTags,
	ToolMemoryGetTag,
}

// vaultToolRequirements descreve, por ferramenta, as propriedades que o
// inputSchema precisa expor e os parâmetros obrigatórios.
//
// As quatro primeiras são as de ADR-057: três operam sobre uma nota (`path`
// obrigatório) e uma lista o vault inteiro (`ext`/`folder` opcionais). As duas
// de tag são diferentes: memory_get_tags aceita `path` como filtro opcional e
// memory_get_tag exige `name`.
var vaultToolRequirements = map[string]struct {
	properties []string
	required   []string
}{
	ToolMemoryListFiles.Name:    {properties: []string{"ext", "folder"}},
	ToolMemoryReadNote.Name:     {properties: []string{"path"}, required: []string{"path"}},
	ToolMemoryNoteOutline.Name:  {properties: []string{"path"}, required: []string{"path"}},
	ToolMemoryFileMetadata.Name: {properties: []string{"path"}, required: []string{"path"}},
	ToolMemoryGetTags.Name:      {properties: []string{"path"}},
	ToolMemoryGetTag.Name:       {properties: []string{"name"}, required: []string{"name"}},
}

func TestVaultTools_RegistrationAndSchema(t *testing.T) {
	srv := NewServer("test", "1.0", nil, nil, nil)
	registered := make(map[string]Tool)
	for _, tool := range srv.GetTools() {
		registered[tool.Name] = tool
	}

	for _, want := range vaultToolsParity {
		got, ok := registered[want.Name]
		if !ok {
			t.Errorf("ferramenta %s não foi registrada no catálogo do servidor", want.Name)
			continue
		}

		if got.Description == "" {
			t.Errorf("%s: descrição obrigatória para o agente decidir quando chamar", want.Name)
		}
		if len(got.Description) < 40 {
			t.Errorf("%s: descrição curta demais para servir de gatilho (%d caracteres)", want.Name, len(got.Description))
		}
		if got.InputSchema == nil {
			t.Fatalf("%s: inputSchema não pode ser nil", want.Name)
		}
		if got.InputSchema["type"] != "object" {
			t.Errorf("%s: esperava inputSchema.type == 'object', obteve %v", want.Name, got.InputSchema["type"])
		}

		props, ok := got.InputSchema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s: esperava properties no inputSchema", got.Name)
		}
		if _, ok := props["repository"]; !ok {
			t.Errorf("%s: esperava propriedade 'repository' (todas as ferramentas do vault a aceitam)", got.Name)
		}

		requirements, ok := vaultToolRequirements[want.Name]
		if !ok {
			t.Errorf("%s: está na lista de paridade mas não tem requisitos de schema declarados", got.Name)
			continue
		}

		for _, property := range requirements.properties {
			if _, ok := props[property]; !ok {
				t.Errorf("%s: esperava propriedade '%s'", got.Name, property)
			}
		}

		required, hasRequired := got.InputSchema["required"].([]string)
		if len(requirements.required) == 0 {
			if hasRequired {
				t.Errorf("%s: esperava nenhum parâmetro obrigatório, obteve %v", got.Name, required)
			}
			continue
		}

		if !hasRequired || len(required) != len(requirements.required) {
			t.Errorf("%s: esperava required == %v, obteve %v", got.Name, requirements.required, got.InputSchema["required"])
			continue
		}
		for i, wantRequired := range requirements.required {
			if required[i] != wantRequired {
				t.Errorf("%s: esperava required == %v, obteve %v", got.Name, requirements.required, required)
				break
			}
		}
	}
}

// TestVaultTools_TodasRegistradasEGuardadas fecha o par registration + handler:
// uma ferramenta sem executor responderia "não encontrada" em tools/call.
func TestVaultTools_TodasRegistradasEGuardadas(t *testing.T) {
	srv := NewServer("test", "1.0", nil, nil, nil)
	srv.SetVaultAccess(VaultAccess{})

	for _, tool := range vaultToolsParity {
		if _, ok := srv.toolHandlers[tool.Name]; !ok {
			t.Errorf("ferramenta %s foi registrada sem executor", tool.Name)
		}
	}
}

// TestVaultTools_SchemaResolvivel_serializa proves que o schema sobrevive ao
// round-trip JSON que o cliente MCP faz ao ler tools/list.
func TestVaultTools_SchemaResolvivel_serializa(t *testing.T) {
	for _, tool := range vaultToolsParity {
		encoded, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("%s: inputSchema não serializa: %v", tool.Name, err)
		}

		var decoded map[string]any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("%s: inputSchema não desserializa: %v", tool.Name, err)
		}
		if decoded["type"] != "object" {
			t.Errorf("%s: tipo perdido na serialização, obteve %v", tool.Name, decoded["type"])
		}
	}
}

func vaultAccessFixture() VaultAccess {
	docs := []store.VaultDocument{
		{ID: "doc-1", Path: "docs/adr/057-mtime.md", Title: "ADR-057", MTime: 1750000000, Size: 2048},
		{ID: "doc-2", Path: "docs/notes/outro.md", Title: "Outro", MTime: 0, Size: 512},
	}

	return VaultAccess{
		ListDocuments: func(_ context.Context, _, _, _ string) ([]store.VaultDocument, error) {
			return docs, nil
		},
		FindDocument: func(_ context.Context, _, path string) (store.VaultDocument, error) {
			for _, doc := range docs {
				if doc.Path == path {
					return doc, nil
				}
			}
			return store.VaultDocument{}, errors.New("nota '" + path + "' não está no índice")
		},
		ReadContent: func(_ context.Context, _ string, docID string) (string, error) {
			if docID != "doc-1" {
				return "", errors.New("documento sem chunks")
			}
			return "# ADR-057\n\ncorpo\n", nil
		},
		ParseHeadings: func(content string) []OutlineHeading {
			if !strings.HasPrefix(content, "# ") {
				return nil
			}
			return []OutlineHeading{{Level: 1, Text: "ADR-057"}}
		},
	}
}

// TestVaultTools_CaminhoAusenteRetornaErro é o critério PARITY-33: path que não
// existe no índice precisa virar erro de tool, não string vazia — string vazia
// seria lida pelo agente como "nota sem conteúdo".
func TestVaultTools_CaminhoAusenteRetornaErro(t *testing.T) {
	ctx := context.Background()
	access := vaultAccessFixture()
	args := json.RawMessage(`{"path": "docs/nao-existe.md"}`)

	cases := []struct {
		name    string
		handler ToolHandlerFunc
	}{
		{"memory_read_note", NewMemoryReadNoteHandler(access.FindDocument, access.ReadContent)},
		{"memory_note_outline", NewMemoryNoteOutlineHandler(access.FindDocument, access.ReadContent, access.ParseHeadings)},
		{"memory_file_metadata", NewMemoryFileMetadataHandler(access.FindDocument)},
	}

	for _, tc := range cases {
		res, err := tc.handler(ctx, args)
		if err == nil {
			t.Errorf("%s: esperava erro para caminho ausente, obteve resultado %+v", tc.name, res)
			continue
		}
		mcpErr, ok := err.(*Error)
		if !ok {
			t.Errorf("%s: esperava *mcp.Error, obteve %T", tc.name, err)
			continue
		}
		if mcpErr.Code != CodeInvalidParams {
			t.Errorf("%s: esperava CodeInvalidParams (%d), obteve %d", tc.name, CodeInvalidParams, mcpErr.Code)
		}
		if !strings.Contains(mcpErr.Message, "docs/nao-existe.md") {
			t.Errorf("%s: a mensagem do erro precisa nomear o caminho pedido, obteve %q", tc.name, mcpErr.Message)
		}
	}
}

func TestVaultTools_PathObrigatorio(t *testing.T) {
	ctx := context.Background()
	access := vaultAccessFixture()

	cases := []struct {
		name    string
		handler ToolHandlerFunc
	}{
		{"memory_read_note", NewMemoryReadNoteHandler(access.FindDocument, access.ReadContent)},
		{"memory_note_outline", NewMemoryNoteOutlineHandler(access.FindDocument, access.ReadContent, access.ParseHeadings)},
		{"memory_file_metadata", NewMemoryFileMetadataHandler(access.FindDocument)},
	}

	for _, tc := range cases {
		_, err := tc.handler(ctx, json.RawMessage(`{}`))
		mcpErr, ok := err.(*Error)
		if !ok || mcpErr.Code != CodeInvalidParams {
			t.Errorf("%s: esperava CodeInvalidParams sem 'path', obteve %v", tc.name, err)
			continue
		}
		if !strings.Contains(mcpErr.Message, "path") {
			t.Errorf("%s: a mensagem precisa dizer qual parâmetro falta, obteve %q", tc.name, mcpErr.Message)
		}
	}
}

func TestVaultTools_ListFiles(t *testing.T) {
	ctx := context.Background()
	access := vaultAccessFixture()

	res, err := NewMemoryListFilesHandler(access.ListDocuments)(ctx, json.RawMessage(`{"ext": "md"}`))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	text := toolResultText(t, res)
	for _, want := range []string{"docs/adr/057-mtime.md", "docs/notes/outro.md", "ADR-057"} {
		if !strings.Contains(text, want) {
			t.Errorf("esperava '%s' na listagem, obteve:\n%s", want, text)
		}
	}
}

func TestVaultTools_ListFilesVazioNaoErra(t *testing.T) {
	ctx := context.Background()
	handler := NewMemoryListFilesHandler(func(_ context.Context, _, _, _ string) ([]store.VaultDocument, error) {
		return nil, nil
	})

	res, err := handler(ctx, json.RawMessage(`{"ext": "pdf"}`))
	if err != nil {
		t.Fatalf("extensão desconhecida é lista vazia, não erro: %v", err)
	}
	if !strings.Contains(toolResultText(t, res), "Nenhum documento") {
		t.Errorf("saída inesperada: %s", toolResultText(t, res))
	}
}

func TestVaultTools_ReadNoteDevolveMarkdownBruto(t *testing.T) {
	ctx := context.Background()
	access := vaultAccessFixture()

	res, err := NewMemoryReadNoteHandler(access.FindDocument, access.ReadContent)(ctx, json.RawMessage(`{"path": "docs/adr/057-mtime.md"}`))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if got := toolResultText(t, res); got != "# ADR-057\n\ncorpo\n" {
		t.Errorf("esperava o Markdown bruto da nota, obteve %q", got)
	}
}

func TestVaultTools_NoteOutline(t *testing.T) {
	ctx := context.Background()
	access := vaultAccessFixture()

	res, err := NewMemoryNoteOutlineHandler(access.FindDocument, access.ReadContent, access.ParseHeadings)(ctx, json.RawMessage(`{"path": "docs/adr/057-mtime.md"}`))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	text := toolResultText(t, res)
	if !strings.Contains(text, "(h1) ADR-057") {
		t.Errorf("esperava o heading com nível, obteve:\n%s", text)
	}
}

func TestVaultTools_NoteOutlineSemTitulos(t *testing.T) {
	ctx := context.Background()
	access := vaultAccessFixture()

	res, err := NewMemoryNoteOutlineHandler(access.FindDocument, access.ReadContent, func(string) []OutlineHeading {
		return nil
	})(ctx, json.RawMessage(`{"path": "docs/adr/057-mtime.md"}`))
	if err != nil {
		t.Fatalf("nota sem título é resposta válida, não erro: %v", err)
	}
	if !strings.Contains(toolResultText(t, res), "não tem nenhum título") {
		t.Errorf("saída inesperada: %s", toolResultText(t, res))
	}
}

func TestVaultTools_FileMetadata(t *testing.T) {
	ctx := context.Background()
	access := vaultAccessFixture()

	res, err := NewMemoryFileMetadataHandler(access.FindDocument)(ctx, json.RawMessage(`{"path": "docs/adr/057-mtime.md"}`))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	text := toolResultText(t, res)
	for _, want := range []string{"docs/adr/057-mtime.md", "ADR-057", "2048 bytes", "1750000000"} {
		if !strings.Contains(text, want) {
			t.Errorf("esperava '%s' nos metadados, obteve:\n%s", want, text)
		}
	}
}

// TestVaultTools_FileMetadata_MTimeZeroNaoFingeData cobre o documento indexado
// antes da coluna existir (ADR-057): mtime = 0 não pode virar uma data de 1970.
func TestVaultTools_FileMetadata_MTimeZeroNaoFingeData(t *testing.T) {
	ctx := context.Background()
	access := vaultAccessFixture()

	res, err := NewMemoryFileMetadataHandler(access.FindDocument)(ctx, json.RawMessage(`{"path": "docs/notes/outro.md"}`))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	text := toolResultText(t, res)
	if strings.Contains(text, "1970") {
		t.Errorf("mtime = 0 não deve ser apresentado como data de 1970:\n%s", text)
	}
	if !strings.Contains(text, "desconhecido") {
		t.Errorf("esperava aviso de mtime desconhecido, obteve:\n%s", text)
	}
}

// TestVaultTools_BackendNaoConfigurado: sem acesso ao índice, a ferramenta tem
// que dizer que o servidor está incompleto em vez de devolver vault vazio.
func TestVaultTools_BackendNaoConfigurado(t *testing.T) {
	ctx := context.Background()

	res, err := NewMemoryListFilesHandler(nil)(ctx, json.RawMessage(`{}`))
	if err == nil {
		t.Fatalf("esperava erro de backend não configurado, obteve %+v", res)
	}
	if !strings.Contains(err.Error(), "não configurado") {
		t.Errorf("mensagem inesperada: %v", err)
	}
}

func TestVaultTools_ArgumentosInvalidos(t *testing.T) {
	ctx := context.Background()
	access := vaultAccessFixture()
	srv := NewServer("test", "1.0", nil, nil, nil)
	srv.SetVaultAccess(access)

	res, err := srv.handleToolsCall(ctx, json.RawMessage(`{"name": "memory_read_note", "arguments": {"path": 123}}`))
	if err == nil {
		t.Fatalf("path não-string deveria falhar o unmarshal, obteve %+v", res)
	}
}

func toolResultText(t *testing.T, res any) string {
	t.Helper()

	callRes, ok := res.(CallToolResult)
	if !ok || len(callRes.Content) == 0 {
		t.Fatalf("resultado inválido: %+v", res)
	}
	return callRes.Content[0].Text
}
