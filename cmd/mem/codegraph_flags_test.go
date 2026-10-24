package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// indexVaultComRepo indexa um vault Go no banco indicado, no repositório
// pedido.
func indexVaultComRepo(t *testing.T, dbFile, repo, vault, file, pkg, symbol string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(vault, "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "package " + pkg + "\n\nfunc " + symbol + "() {}\n"
	if err := os.WriteFile(filepath.Join(vault, "pkg", file), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := runCodeIndexCLI(context.Background(), repo, []string{
		"--db=" + dbFile,
		"--storage=sqlite",
		"--lang=go",
		"--include=**/*.go",
		"--repo=" + repo,
		vault,
	}); err != nil {
		t.Fatalf("runCodeIndexCLI(%s): %v", repo, err)
	}
}

// TestCodeGraphCLI_HonoraRepo prova o defeito D5: o `--repo` do
// `mem code-graph` era aceito e DESCARTADO (`_ = repoSlug`) — a consulta
// lia o code graph inteiro, sem filtro de repositório.
//
// Depois da correção, `--repo=<slug>` filtra: o símbolo de outro repositório
// não é encontrado.
func TestCodeGraphCLI_HonoraRepo(t *testing.T) {
	dbFile := filepath.Join(t.TempDir(), "repos.db")
	vaultA := t.TempDir()
	vaultB := t.TempDir()

	indexVaultComRepo(t, dbFile, "repoA", vaultA, "a.go", "apkg", "Alpha")
	indexVaultComRepo(t, dbFile, "repoB", vaultB, "b.go", "bpkg", "Beta")

	// 1. O símbolo do repoB NÃO pode ser visto pelo repoA.
	err := runCodeGraphCLI(context.Background(), "repoA", []string{
		"--db=" + dbFile,
		"--storage=sqlite",
		"--repo=repoA",
		"bpkg.Beta",
	})
	if err == nil {
		t.Fatal("o --repo não filtrou: bpkg.Beta (repoB) foi encontrado pelo repoA")
	}
	if !strings.Contains(err.Error(), "não encontrado") {
		t.Errorf("erro inesperado: %v", err)
	}

	// 2. O símbolo do repoB é visto pelo repoB.
	if err := runCodeGraphCLI(context.Background(), "repoB", []string{
		"--db=" + dbFile,
		"--storage=sqlite",
		"--repo=repoB",
		"bpkg.Beta",
	}); err != nil {
		t.Fatalf("o símbolo do próprio repo não foi encontrado: %v", err)
	}

	// 3. E o do repoA pelo repoA.
	if err := runCodeGraphCLI(context.Background(), "repoA", []string{
		"--db=" + dbFile,
		"--storage=sqlite",
		"--repo=repoA",
		"apkg.Alpha",
	}); err != nil {
		t.Fatalf("o símbolo do próprio repo não foi encontrado: %v", err)
	}
}

// TestCodeGraphCLI_UsaBancoIndicado prova a outra metade do defeito D5:
// `--storage=sqlite --db=<arq>` tem que escolher o banco indicado (antes o
// comando usava sempre db.InitDB("memory.db") local e ignorava as flags).
func TestCodeGraphCLI_UsaBancoIndicado(t *testing.T) {
	dbFile := filepath.Join(t.TempDir(), "graph.db")
	otherFile := filepath.Join(t.TempDir(), "other.db")

	indexVaultComRepo(t, dbFile, "repoA", t.TempDir(), "a.go", "apkg", "Alpha")

	// 1. Com o banco indicado, o símbolo é encontrado.
	if err := runCodeGraphCLI(context.Background(), "repoA", []string{
		"--storage=sqlite",
		"--db=" + dbFile,
		"--repo=repoA",
		"apkg.Alpha",
	}); err != nil {
		t.Fatalf("o símbolo não foi encontrado no banco indicado: %v", err)
	}

	// 2. Com OUTRO banco (vazio), o símbolo NÃO é encontrado: a flag --db
	//    não está sendo ignorada.
	if err := runCodeGraphCLI(context.Background(), "repoA", []string{
		"--storage=sqlite",
		"--db=" + otherFile,
		"--repo=repoA",
		"apkg.Alpha",
	}); err == nil {
		t.Fatal("o --db foi ignorado: o símbolo foi encontrado em outro banco")
	}
}

// TestCodeGraphCLI_StoragePostgresSemURLFalha prova que a flag --storage
// não é decorativa: com Postgres pedido e URL inválida, o comando falha com
// erro explícito (e não cai silenciosamente no SQLite local).
//
// A URL aponta para uma porta local fechada: o teste não toca em nenhum
// Postgres real.
func TestCodeGraphCLI_StoragePostgresSemURLFalha(t *testing.T) {
	err := runCodeGraphCLI(context.Background(), "repoA", []string{
		"--storage=postgres",
		"--postgres=postgres://user:pass@127.0.0.1:1/nao_existe?sslmode=disable",
		"--repo=repoA",
		"apkg.Alpha",
	})
	if err == nil {
		t.Fatal("esperava erro: --storage=postgres com URL inalcançável")
	}
}
