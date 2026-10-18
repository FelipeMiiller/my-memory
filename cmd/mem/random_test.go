package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunRandomCommand_ImprimeCaminhoETitulo(t *testing.T) {
	dbPath := newVaultFixture(t, "docs/alpha.md", "docs/beta.md", "notes/zeta.md")

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite"}
	if err := runRandomCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runRandomCommand falhou: %v", err)
	}

	lines := strings.Fields(strings.TrimSpace(buf.String()))
	if len(lines) == 0 {
		t.Fatal("mem random não imprimiu nada")
	}
	if !strings.HasSuffix(lines[0], ".md") {
		t.Errorf("esperava um caminho de documento na primeira linha, obteve:\n%s", buf.String())
	}
}

func TestRunRandomCommand_JSON(t *testing.T) {
	dbPath := newVaultFixture(t, "docs/alpha.md", "docs/beta.md")

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "--json"}
	if err := runRandomCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runRandomCommand --json falhou: %v", err)
	}

	var resp RandomResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("saída não é um JSON válido: %v, saída: %s", err, buf.String())
	}
	if resp.Path == "" || resp.Title == "" {
		t.Errorf("o JSON precisa trazer path e title preenchidos: %+v", resp)
	}
	if !strings.HasSuffix(resp.Path, ".md") {
		t.Errorf("o path sorteado não é um documento do fixture: %+v", resp)
	}
}

// TestRunRandomCommand_IndiceVazioFalha é o critério "falhar alto": um índice
// vazio não pode devolver uma nota inventada nem sair zero fingindo sucesso,
// porque quem chama isso em script precisa saber que o vault não foi indexado.
func TestRunRandomCommand_IndiceVazioFalha(t *testing.T) {
	dbPath := newVaultFixture(t)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite"}
	err := runRandomCommand(context.Background(), "fixture-repo", args, &buf)
	if err == nil {
		t.Fatal("índice vazio deveria sair com erro")
	}
	if !strings.Contains(err.Error(), "vazio") {
		t.Errorf("a mensagem precisa dizer que o índice está vazio, obteve: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "" {
		t.Errorf("nada deveria ter sido impresso, obteve:\n%s", buf.String())
	}
}

func TestRunRandomCommand_IndiceVazioFalhaNoJSON(t *testing.T) {
	dbPath := newVaultFixture(t)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "--json"}
	err := runRandomCommand(context.Background(), "fixture-repo", args, &buf)
	if err == nil {
		t.Fatal("índice vazio em --json também deveria sair com erro")
	}
	if json.Valid(buf.Bytes()) {
		t.Errorf("erro não pode virar JSON válido na saída: %s", buf.String())
	}
}
