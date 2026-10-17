package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// errAbort representa uma falha de precondition vinda do reconciliador.
var errAbort = errors.New("reconcile abortado: varredura do disco retornou 0 arquivos indexáveis")

func TestMCPReconcile_DryRunPadraoEtrue(t *testing.T) {
	srv := NewServer("test-server", "1.0.0", nil, nil, nil)

	var captured ReconcileRequest
	srv.SetReconcileHandler(func(_ context.Context, req ReconcileRequest) (*ReconcileReport, error) {
		captured = req
		return &ReconcileReport{
			Repository:    req.Repository,
			RootDir:       req.RootDir,
			DryRun:        req.DryRunEnabled(),
			ScannedFiles:  2,
			MissingOnDisk: []ReconcileDocRef{{ID: "vault/notas/fantasma.md", Path: "vault/notas/fantasma.md"}},
			Present:       []ReconcileDocRef{},
			NotIndexed:    []ReconcileDocRef{},
			Pruned:        []ReconcilePruned{{ID: "vault/notas/fantasma.md", Path: "vault/notas/fantasma.md", DryRun: true}},
		}, nil
	})

	res, err := srv.handleToolsCall(context.Background(), mustRawMessage(map[string]any{
		"name":      "memory_reconcile",
		"arguments": json.RawMessage(`{"repository":"central-memory","root_dir":"G:/vault"}`),
	}))
	if err != nil {
		t.Fatalf("erro ao executar memory_reconcile: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok {
		t.Fatalf("tipo de resultado inesperado: %T", res)
	}
	if callRes.IsError {
		t.Fatalf("chamada retornou erro: %s", callRes.Content[0].Text)
	}

	if !captured.DryRunEnabled() {
		t.Error("dry_run ausente deveria assumir true (nada é apagado por padrão)")
	}
	if captured.Repository != "central-memory" || captured.RootDir != "G:/vault" {
		t.Errorf("argumentos não foram repassados: %+v", captured)
	}

	text := callRes.Content[0].Text
	for _, want := range []string{"DRY-RUN", "Ausentes no disco", "vault/notas/fantasma.md", "central-memory"} {
		if !strings.Contains(text, want) {
			t.Errorf("saída deveria conter %q, obteve:\n%s", want, text)
		}
	}
}

func TestMCPReconcile_DryRunAusenteNosArgumentos(t *testing.T) {
	handler := NewMemoryReconcileHandler(func(_ context.Context, req ReconcileRequest) (*ReconcileReport, error) {
		return &ReconcileReport{DryRun: req.DryRunEnabled()}, nil
	})

	// Sem argumentos: dry_run tem que continuar sendo true.
	res, err := handler(context.Background(), nil)
	if err != nil {
		t.Fatalf("chamada sem argumentos deveria funcionar: %v", err)
	}
	if res.(CallToolResult).Content[0].Text == "" {
		t.Fatal("esperava relatório textual")
	}
}

func TestMCPReconcile_DryRunFalseExplicitoDesligaDryRun(t *testing.T) {
	var captured ReconcileRequest
	handler := NewMemoryReconcileHandler(func(_ context.Context, req ReconcileRequest) (*ReconcileReport, error) {
		captured = req
		return &ReconcileReport{DryRun: req.DryRunEnabled()}, nil
	})

	_, err := handler(context.Background(), json.RawMessage(`{"dry_run":false,"confirm":true}`))
	if err != nil {
		t.Fatalf("chamada falhou: %v", err)
	}

	if captured.DryRunEnabled() {
		t.Error("dry_run=false explícito deveria desligar o dry-run")
	}
	if !captured.Confirm {
		t.Error("confirm=true deveria ser repassado")
	}
}

func TestMCPReconcile_ErroDoBackendViraCodeInvalidParams(t *testing.T) {
	handler := NewMemoryReconcileHandler(func(_ context.Context, _ ReconcileRequest) (*ReconcileReport, error) {
		return nil, errAbort
	})

	_, err := handler(context.Background(), json.RawMessage(`{"dry_run":true}`))
	if err == nil {
		t.Fatal("erro do backend deveria propagar")
	}
	mcpErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("esperava *mcp.Error, obteve %T", err)
	}
	if mcpErr.Code != CodeInvalidParams {
		t.Errorf("guardas de precondition deveriam virar CodeInvalidParams, obteve %d", mcpErr.Code)
	}
	if !strings.Contains(mcpErr.Message, "abortado") {
		t.Errorf("mensagem deveria preservar o motivo do abort, obteve: %s", mcpErr.Message)
	}
}

func TestMCPReconcile_SemBackendConfigurado(t *testing.T) {
	srv := NewServer("test-server", "1.0.0", nil, nil, nil)

	_, err := srv.handleToolsCall(context.Background(), mustRawMessage(map[string]any{
		"name":      "memory_reconcile",
		"arguments": json.RawMessage(`{"dry_run":true}`),
	}))
	if err == nil {
		t.Fatal("sem backend a ferramenta deve recusar em vez de fingir sucesso")
	}
}

func TestMCPReconcile_ArgumentosInvalidos(t *testing.T) {
	handler := NewMemoryReconcileHandler(func(_ context.Context, _ ReconcileRequest) (*ReconcileReport, error) {
		t.Fatal("backend não deveria ser chamado com argumentos inválidos")
		return nil, nil
	})

	if _, err := handler(context.Background(), json.RawMessage(`{"dry_run":"talvez"}`)); err == nil {
		t.Fatal("dry_run com tipo errado deveria ser rejeitado")
	}
}

func TestMCPReconcile_OperacaoBloqueadaApareceNoRelatorio(t *testing.T) {
	srv := NewServer("test-server", "1.0.0", nil, nil, nil)
	srv.SetReconcileHandler(func(_ context.Context, _ ReconcileRequest) (*ReconcileReport, error) {
		return &ReconcileReport{
			RootDir:    "G:/vault",
			DryRun:     false,
			Blocked:    true,
			Present:    []ReconcileDocRef{{ID: "a.md", Path: "a.md"}},
			Warnings:   []string{"⚠️ ASSIMETRIA ALTA: 99 arquivo(s) no disco fora do índice contra 10 documento(s) presentes."},
			Pruned:     []ReconcilePruned{},
			NotIndexed: []ReconcileDocRef{},
		}, nil
	})

	res, err := srv.handleToolsCall(context.Background(), mustRawMessage(map[string]any{
		"name":      "memory_reconcile",
		"arguments": json.RawMessage(`{"dry_run":false}`),
	}))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	text := res.(CallToolResult).Content[0].Text
	if !strings.Contains(text, "Remoção suspensa") {
		t.Errorf("relatório deveria avisar que a remoção foi suspensa:\n%s", text)
	}
	if !strings.Contains(text, "ASSIMETRIA ALTA") {
		t.Errorf("relatório deveria conter o aviso alto:\n%s", text)
	}
}

func TestMCPReconcile_CatalogoApareceComoExclusao(t *testing.T) {
	text := FormatReconcileReport(&ReconcileReport{
		Repository:        "central-memory",
		RootDir:           "G:/central-memory",
		DryRun:            true,
		ScannedFiles:      120,
		ScannedExcluded:   700,
		Present:           []ReconcileDocRef{},
		MissingOnDisk:     []ReconcileDocRef{},
		NotIndexed:        []ReconcileDocRef{},
		ExcludedFromIndex: []ReconcileDocRef{{ID: "x", Path: "catalog-skills/skills/bruto.md"}},
		Pruned:            []ReconcilePruned{},
	})

	if !strings.Contains(text, "catalog-skills/skills/bruto.md") {
		t.Errorf("material bruto deveria aparecer na categoria de exclusão:\n%s", text)
	}
	if !strings.Contains(text, "Fora do índice por exclusão") {
		t.Errorf("categoria de exclusão deveria estar rotulada:\n%s", text)
	}
	if !strings.Contains(text, "não tocar") {
		t.Errorf("material bruto deveria estar marcado como estado correto:\n%s", text)
	}
}

func TestMCPReconcile_ListaLongaEDecimada(t *testing.T) {
	var many []ReconcileDocRef
	for i := 0; i < 40; i++ {
		many = append(many, ReconcileDocRef{ID: "x", Path: "notas/doc.md"})
	}

	text := FormatReconcileReport(&ReconcileReport{
		RootDir:       "G:/vault",
		DryRun:        true,
		MissingOnDisk: many,
	})

	if !strings.Contains(text, "e mais 15 documento(s)") {
		t.Errorf("lista longa deveria ser truncada com contagem do restante:\n%s", text)
	}
}

func TestMCPReconcile_FerramentaRegistrada(t *testing.T) {
	srv := NewServer("test-server", "1.0.0", nil, nil, nil)

	found := false
	for _, tool := range srv.GetTools() {
		if tool.Name == "memory_reconcile" {
			found = true
			if tool.InputSchema["type"] != "object" {
				t.Error("inputSchema deveria ser um objeto")
			}
			props, ok := tool.InputSchema["properties"].(map[string]any)
			if !ok {
				t.Fatal("propriedades ausentes no inputSchema")
			}
			for _, want := range []string{"dry_run", "repository", "root_dir", "confirm"} {
				if _, ok := props[want]; !ok {
					t.Errorf("propriedade %q ausente no inputSchema", want)
				}
			}
		}
	}
	if !found {
		t.Fatal("memory_reconcile não foi registrada na listagem de tools")
	}
}
