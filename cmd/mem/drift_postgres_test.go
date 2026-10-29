package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/drift"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// fakeGitRunner devolve uma lista fixa de commits e de mudanças — o teste do
// drift não precisa de um repositório git de verdade.
type fakeGitRunner struct {
	changes []drift.FileChange
}

func (f fakeGitRunner) Log(ctx context.Context, repoRoot, rangeStr string) ([]drift.GitCommit, error) {
	return []drift.GitCommit{{Hash: "abc1234", ShortHash: "abc1234"}}, nil
}

func (f fakeGitRunner) Diff(ctx context.Context, repoRoot, rangeStr string) ([]drift.FileChange, error) {
	return f.changes, nil
}

// TestDriftCLI_PostgresNaoAcusaMudancaEstruturalEmArquivoInalterado prova o
// ISSUE-035: no PostgreSQL, o drift é escopado por REPOSITÓRIO.
//
// Sem o repositório no contexto, a consulta dos símbolos gravados volta vazia
// (no Postgres ela filtra por `repository`), o hash estrutural gravado fica ""
// e TODO arquivo indexado vira "mudança estrutural" — inclusive arquivos que
// NÃO mudaram desde o último índice.
//
// O defeito é invisível no SQLite (lá não há filtro de repositório), então o
// teste roda contra Postgres real.
func TestDriftCLI_PostgresNaoAcusaMudancaEstruturalEmArquivoInalterado(t *testing.T) {
	// Banco DESCARTÁVEL próprio: o Postgres do dono nunca é alvo.
	h := pgtest.New(t)

	// Vault de fixture: um arquivo Go com uma função.
	vault := t.TempDir()
	pkgDir := filepath.Join(vault, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir do vault de fixture: %v", err)
	}
	const alphaSrc = "package pkg\n\nfunc Alpha() string {\n\treturn \"a\"\n}\n"
	if err := os.WriteFile(filepath.Join(pkgDir, "alpha.go"), []byte(alphaSrc), 0o644); err != nil {
		t.Fatalf("escrita de alpha.go: %v", err)
	}

	// 1. O code graph é indexado NO Postgres descartável, no repo "repo-drift".
	if err := runCodeIndexCLI(context.Background(), "repo-drift", []string{
		"--dir=" + vault,
		"--storage=postgres",
		"--postgres=" + h.DSN,
		"--repo=repo-drift",
	}); err != nil {
		t.Fatalf("runCodeIndexCLI: %v", err)
	}

	// 2. O drift roda com um runner falso: o git diz que pkg/alpha.go mudou —
	//    mas o arquivo é o MESMO que foi indexado (nada foi editado), ou seja,
	//    mudança COSMÉTICA, que o drift estrutural tem que SUPRIMIR.
	t.Chdir(vault)
	runner := fakeGitRunner{changes: []drift.FileChange{{
		Path:       "pkg/alpha.go",
		Status:     "M",
		Additions:  1,
		Deletions:  1,
		IsCodeFile: true,
	}}}

	var driftErr error
	out := captureStdout(t, func() {
		driftErr = runDriftCommand(context.Background(), "repo-drift", []string{
			"--storage=postgres",
			"--postgres=" + h.DSN,
			"--repo=repo-drift",
			"--json",
		}, os.Stdout, runner)
	})
	if driftErr != nil {
		t.Fatalf("runDriftCommand: %v", driftErr)
	}

	var report struct {
		StructuralChanges []struct {
			FilePath string `json:"file_path"`
		} `json:"structural_changes"`
		HighCount int `json:"high_count"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("JSON do drift inválido: %v\nsaída bruta:\n%s", err, out)
	}
	if len(report.StructuralChanges) != 0 {
		t.Errorf("o drift acusou %d mudança(s) ESTRUTURAL(is) em arquivo INALTERADO: %+v (o hash estrutural gravado tem que ser comparado com o atual)", len(report.StructuralChanges), report.StructuralChanges)
	}
}
