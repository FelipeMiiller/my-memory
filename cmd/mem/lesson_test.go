package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/parser"
)

// listMarkdown devolve todos os arquivos .md sob root.
func listMarkdown(t *testing.T, root string) []string {
	t.Helper()

	var found []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".md") {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("erro ao listar .md em '%s': %v", root, err)
	}
	return found
}

// TestMemLessonGravaNotaNoVaultCentral cobre a AC 28/29 da US-11: a lição é
// gravada no cofre central (não no vault do repositório) como nota com
// `category: memory` e `summary`, com timestamp, e é indexada na hora — sem
// code graph e sem Postgres.
func TestMemLessonGravaNotaNoVaultCentral(t *testing.T) {
	ctx := context.Background()

	// 1. Isola a configuração global (o cofre central vem dela — ADR-040).
	globalDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, globalDir)
	t.Setenv("MY_MEMORY_PG_URL", "")
	t.Setenv("POSTGRES_URL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("MY_MEMORY_REPO", "")

	// 2. Cofre central temporário + vault de repositório temporário.
	centralDir := filepath.Join(t.TempDir(), "cofre-central")
	if err := os.MkdirAll(centralDir, 0755); err != nil {
		t.Fatalf("erro ao criar o cofre central: %v", err)
	}

	repoDir := t.TempDir()
	t.Chdir(repoDir)

	if err := config.SaveGlobalConfig(&config.GlobalConfig{
		Version:      1,
		CentralVault: config.CentralVaultConfig{Path: centralDir},
	}); err != nil {
		t.Fatalf("erro ao salvar a configuração global: %v", err)
	}

	// 3. Banco temporário: sem Postgres, sem code graph.
	dbPath := filepath.Join(t.TempDir(), "central.db")

	// 4. Executa o comando (título antes das flags — o contrato do comando).
	var runErr error
	out := captureStdout(t, func() {
		runErr = runLessonCLI(ctx, nil, nil, "repo-de-teste", []string{
			"Lição: sempre rodar gofmt antes do commit",
			"--summary=Gofmt em Go é o gate de formatação do repositório.",
			"--content=Aprendi que `gofmt -w .` mascara defeito de line ending no checkout local.",
			"--tags=go,lições",
			"--storage=sqlite",
			"--db", dbPath,
		})
	})
	if runErr != nil {
		t.Fatalf("runLessonCLI retornou erro: %v", runErr)
	}

	// 5. A nota foi gravada NO COFRE CENTRAL.
	notes := listMarkdown(t, centralDir)
	if len(notes) != 1 {
		t.Fatalf("esperava 1 nota no cofre central, obteve %d: %v", len(notes), notes)
	}
	notePath := notes[0]
	if !strings.HasPrefix(notePath, centralDir) {
		t.Fatalf("a nota foi gravada fora do cofre central: %s", notePath)
	}

	// 6. O vault do repositório NÃO foi tocado.
	if touched := listMarkdown(t, repoDir); len(touched) != 0 {
		t.Errorf("o vault do repositório foi tocado: %v", touched)
	}
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		t.Fatalf("erro ao ler o vault do repositório: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("o vault do repositório ganhou %d entrada(s): %v", len(entries), entries)
	}

	// 7. Frontmatter: category: memory, summary e timestamp.
	data, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("erro ao ler a nota: %v", err)
	}
	fm, body := parser.ExtractFrontmatter(string(data))
	if fm == nil {
		t.Fatalf("a nota não tem frontmatter:\n%s", string(data))
	}
	if fm.Category != "memory" {
		t.Errorf("esperava category 'memory', obteve '%s'", fm.Category)
	}
	if fm.Summary != "Gofmt em Go é o gate de formatação do repositório." {
		t.Errorf("esperava o summary informado, obteve '%s'", fm.Summary)
	}
	if fm.Title != "Lição: sempre rodar gofmt antes do commit" {
		t.Errorf("esperava o título informado, obteve '%s'", fm.Title)
	}
	if !strings.Contains(body, "gofmt -w .") {
		t.Errorf("o corpo da lição não foi preservado:\n%s", body)
	}

	createdRaw, ok := fm.Properties["created_at"]
	if !ok {
		t.Errorf("a nota não tem timestamp (created_at):\n%s", string(data))
	} else if ts, isTime := createdRaw.(time.Time); isTime {
		if ts.IsZero() {
			t.Errorf("created_at inválido: %v", ts)
		}
	} else if s, isStr := createdRaw.(string); !isStr || strings.TrimSpace(s) == "" {
		t.Errorf("created_at inesperado: %#v", createdRaw)
	}
	if _, ok := fm.Properties["updated_at"]; !ok {
		t.Errorf("a nota não tem updated_at:\n%s", string(data))
	}

	// 8. A nota foi INDEXADA (category e summary persistidos no banco).
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("erro ao abrir o banco temporário: %v", err)
	}
	defer database.Close()

	var indexedPath, indexedTitle, indexedCategory, indexedAbstract string
	err = database.QueryRowContext(ctx,
		`SELECT path, title, category, abstract FROM documents WHERE id = ?`, notePath).
		Scan(&indexedPath, &indexedTitle, &indexedCategory, &indexedAbstract)
	if err != nil {
		t.Fatalf("a nota não foi indexada em 'documents': %v", err)
	}
	if indexedCategory != "memory" {
		t.Errorf("esperava category indexada 'memory', obteve '%s'", indexedCategory)
	}
	if indexedAbstract != fm.Summary {
		t.Errorf("esperava o summary indexado '%s', obteve '%s'", fm.Summary, indexedAbstract)
	}
	if indexedTitle != fm.Title {
		t.Errorf("esperava o título indexado '%s', obteve '%s'", fm.Title, indexedTitle)
	}

	// 9. A busca encontra a lição (a nota está no índice, não só no disco).
	results, err := db.SearchFTS(ctx, database, "gofmt", 5)
	if err != nil {
		t.Fatalf("erro na busca FTS: %v", err)
	}
	if len(results) == 0 {
		t.Errorf("a busca FTS não encontrou a lição indexada")
	}

	// 10. A saída traz o path da nota gravada.
	if !strings.Contains(out, notePath) {
		t.Errorf("a saída não traz o path da nota:\n%s", out)
	}
}

// TestMemLessonSemCofreCentralFalha prova que, sem cofre central resolvido, o
// comando falha com erro claro — em vez de gravar a lição no vault do
// repositório.
func TestMemLessonSemCofreCentralFalha(t *testing.T) {
	ctx := context.Background()

	globalDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, globalDir)
	t.Setenv("MY_MEMORY_PG_URL", "")
	t.Setenv("POSTGRES_URL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("MY_MEMORY_REPO", "")

	repoDir := t.TempDir()
	t.Chdir(repoDir)

	// Config global sem 'central_vault.path'.
	if err := config.SaveGlobalConfig(&config.GlobalConfig{Version: 1}); err != nil {
		t.Fatalf("erro ao salvar a configuração global: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "central.db")

	var runErr error
	captureStdout(t, func() {
		runErr = runLessonCLI(ctx, nil, nil, "repo-de-teste", []string{
			"Lição sem destino",
			"--content=corpo",
			"--storage=sqlite",
			"--db", dbPath,
		})
	})

	if runErr == nil {
		t.Fatalf("esperava erro sem cofre central resolvido, obteve sucesso")
	}
	if !strings.Contains(runErr.Error(), "cofre central") {
		t.Errorf("esperava erro mencionando o cofre central, obteve: %v", runErr)
	}

	// Nada foi gravado no vault do repositório.
	if touched := listMarkdown(t, repoDir); len(touched) != 0 {
		t.Errorf("o vault do repositório foi tocado: %v", touched)
	}
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		t.Fatalf("erro ao ler o vault do repositório: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("o vault do repositório ganhou %d entrada(s): %v", len(entries), entries)
	}
}

// TestMemLessonStorageFlagVenceEnvVar é a regressão do incidente do D8.
//
// Com MY_MEMORY_PG_URL no ambiente, o `mem lesson` ignorava o config de
// teste e conectava no Postgres real — em silêncio, sem aviso. A flag
// --storage=sqlite tem que vencer a variável de ambiente: a lição vai para
// o SQLite indicado e o Postgres não é tocado.
//
// A env var aponta para uma porta fechada (127.0.0.1:1): se a flag NÃO
// vencesse, o comando falharia ao conectar — e o teste falharia junto.
func TestMemLessonStorageFlagVenceEnvVar(t *testing.T) {
	ctx := context.Background()

	globalDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, globalDir)
	t.Setenv("MY_MEMORY_REPO", "")
	t.Setenv("MY_MEMORY_PG_URL", "postgres://user:pass@127.0.0.1:1/db-inexistente")
	t.Setenv("POSTGRES_URL", "")
	t.Setenv("DATABASE_URL", "")

	centralDir := filepath.Join(t.TempDir(), "cofre-central")
	if err := os.MkdirAll(centralDir, 0755); err != nil {
		t.Fatalf("erro ao criar o cofre central: %v", err)
	}

	repoDir := t.TempDir()
	t.Chdir(repoDir)

	if err := config.SaveGlobalConfig(&config.GlobalConfig{
		Version:      1,
		CentralVault: config.CentralVaultConfig{Path: centralDir},
	}); err != nil {
		t.Fatalf("erro ao salvar a configuração global: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "central.db")

	var runErr error
	captureStdout(t, func() {
		runErr = runLessonCLI(ctx, nil, nil, "repo-de-teste", []string{
			"Lição isolada do Postgres",
			"--content=corpo",
			"--storage=sqlite",
			"--db", dbPath,
		})
	})
	if runErr != nil {
		t.Fatalf("runLessonCLI retornou erro (a flag NÃO venceu a env var): %v", runErr)
	}

	// A lição foi indexada no SQLite indicado.
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("erro ao abrir o banco temporário: %v", err)
	}
	defer database.Close()

	var total int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents`).Scan(&total); err != nil {
		t.Fatalf("erro ao contar os documentos: %v", err)
	}
	if total != 1 {
		t.Errorf("esperava 1 documento no SQLite indicado, obteve %d", total)
	}

	// E a nota foi gravada no cofre central (o vault do repositório, não).
	notes := listMarkdown(t, centralDir)
	if len(notes) != 1 {
		t.Fatalf("esperava 1 nota no cofre central, obteve %d: %v", len(notes), notes)
	}
	if touched := listMarkdown(t, repoDir); len(touched) != 0 {
		t.Errorf("o vault do repositório foi tocado: %v", touched)
	}
}
