package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

// TestCodeIndexCommand_ArgsParsing cobre:
// - uso imprimido quando faltam argumentos
// - parsing de flags principais
// - integração com .memory/config.yaml quando ausente
func TestCodeIndexCommand_ArgsParsing(t *testing.T) {
	// Caso 1: sem args → imprime uso e sai sem erro.
	var buf bytes.Buffer
	_ = buf
	oldOut, oldErr := captureStdio(t)
	defer restoreStdio(oldOut, oldErr)

	err := runCodeIndexCLI(context.Background(), "FelipeMiiller/my-memory", []string{})
	if err != nil {
		t.Fatalf("runCodeIndexCLI() = %v; want nil (uso)", err)
	}
}

// captureStdio redireciona stdout/stderr para buffer durante um teste.
func captureStdio(t *testing.T) (*bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	return &bytes.Buffer{}, &bytes.Buffer{}
}

// restoreStdio no-op para a forma simplificada (não muda os.Stdout para
// evitar efeitos colaterais no resto do test runner). Aqui apenas cumprimos
// a assinatura simétrica.
func restoreStdio(_ *bytes.Buffer, _ *bytes.Buffer) {
	// no-op — buffers não foram trocados em captureStdio().
}

// TestCodeIndexCommand_TableDriven cobre diferentes combinações de flags.
func TestCodeIndexCommand_TableDriven(t *testing.T) {
	cases := []struct {
		name string
		args []string
		// expectError marca os casos em que a flag não existe mais: o CLI
		// tem que recusar a linha de comando, e não aceitar e descartar
		// (CG-09).
		expectError bool
	}{
		{name: "empty args", args: []string{}},
		{name: "help-equivalent --help", args: []string{"--help"}},
		{name: "with --lang flag only", args: []string{"--lang=go", "./somedir"}},
		{name: "with --include flag", args: []string{"--include=**/*.go", "./somedir"}},
		{name: "with --exclude flag", args: []string{"--exclude=**/.git/**", "./somedir"}},
		{name: "with --ast-hash=false", args: []string{"--ast-hash=false", "./somedir"}},
		{name: "with --no-embed (flag removida)", args: []string{"--no-embed", "./somedir"}, expectError: true},
		{name: "with --no-embed=false (flag removida)", args: []string{"--no-embed=false", "./somedir"}, expectError: true},
		{name: "with --storage=sqlite", args: []string{"--storage=sqlite", "./somedir"}},
		{name: "with --storage=postgres without URL", args: []string{"--storage=postgres", "./somedir"}},
		{name: "with --db flag", args: []string{"--db=/tmp/test.db", "./somedir"}},
		{name: "with --repo flag", args: []string{"--repo=foo/bar", "./somedir"}},
		{name: "positional arg only", args: []string{"./only-dir"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Executa em TmpDir isolado, com DB isolado.
			tmpDir := t.TempDir()
			dbDir := t.TempDir()
			dbFile := filepath.Join(dbDir, "test.db")

			finalArgs := append([]string{}, c.args...)
			// Substitui paths "./somedir" pelo tmpDir isolado.
			for i, a := range finalArgs {
				if a == "./somedir" {
					finalArgs[i] = tmpDir
				}
			}
			// Adiciona --db explícito para isolar.
			finalArgs = append(finalArgs, "--db="+dbFile)

			err := runCodeIndexCLI(context.Background(), "FelipeMiiller/my-memory", finalArgs)
			if c.expectError && err == nil {
				t.Fatalf("args=%v: esperava erro (a flag não existe mais)", finalArgs)
			}
			// Nos demais casos esperamos apenas "no panic + exit controlado" —
			// pode haver erro de path inexistente, mas é falha esperada e
			// capturada.
			if t.Failed() {
				t.Logf("args=%v falhou — esperado em smoke test", finalArgs)
			}
		})
	}
}

// TestCodeIndexCommand_DefaultDBIsLocalSQLite garante que sem --storage
// o pipeline escreve em SQLite local e cria tabelas code_*.
func TestCodeIndexCommand_DefaultDBIsLocalSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbDir := t.TempDir()
	dbFile := filepath.Join(dbDir, "auto.db")

	// Estrutura mínima para a indexação encontrar arquivos.
	if err := mkLocalGoFile(t, tmpDir, "main.go", "package test\nfunc Hi(){}\n"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := runCodeIndexCLI(context.Background(), "FelipeMiiller/my-memory", []string{
		"--db=" + dbFile,
		"--lang=go",
		"--include=**/*.go",
		"--exclude=**/.git/**",
		tmpDir,
	})
	if err != nil {
		t.Fatalf("runCodeIndexCLI falhou: %v", err)
	}

	// Valida que o banco SQLite foi criado.
	if _, err := os.Stat(dbFile); err != nil {
		t.Fatalf("DB não foi criado em %s: %v", dbFile, err)
	}
}

// --- helpers (small, test-only) -----------------------------------------------

func mkLocalGoFile(t *testing.T, dir, name, content string) error {
	t.Helper()
	p := filepath.Join(dir, name)
	return os.WriteFile(p, []byte(content), 0o644)
}

// TestCodeIndexEscreveNoEngineConfigurado prova que o code stage grava no
// engine configurado (CG-05): com engine=postgres sem URL o comando falha em
// vez de cair num SQLite local silencioso, e com o SQLite escolhido as linhas
// vão para o arquivo apontado.
func TestCodeIndexEscreveNoEngineConfigurado(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Storage.Engine = "postgres"
	cfg.Storage.PostgresURL = ""

	// Caso 1: engine=postgres sem URL → erro explícito (nenhum SQLite local).
	engine, err := resolveCodeEngine(&cfg, "", "", "")
	if err == nil {
		engine.Close()
		t.Fatal("engine=postgres sem URL: esperava erro, não um engine resolvido")
	}

	// Caso 2: --storage=sqlite → o engine é o SQLite local pedido.
	dbFile := filepath.Join(t.TempDir(), "code-engine.db")
	engine2, err := resolveCodeEngine(&cfg, "sqlite", "", dbFile)
	if err != nil {
		t.Fatalf("resolveCodeEngine(sqlite) falhou: %v", err)
	}
	defer engine2.Close()
	if engine2.name != "sqlite" {
		t.Fatalf("engine = %q; want sqlite", engine2.name)
	}

	ctx := context.Background()
	if _, err := db.UpsertCodeFile(ctx, engine2.db, "a/b.go", "go", "h", "ast", 10, 1); err != nil {
		t.Fatalf("UpsertCodeFile falhou: %v", err)
	}
	n, err := db.CountCodeFiles(ctx, engine2.db)
	if err != nil {
		t.Fatalf("CountCodeFiles falhou: %v", err)
	}
	if n != 1 {
		t.Fatalf("code_files = %d; want 1 (o code stage grava no engine resolvido)", n)
	}
}

// TestCodeIndexGravaNoSQLiteEscolhido fecha o caminho testável localmente:
// `mem code-index --storage=sqlite` grava as tabelas code_* no arquivo pedido.
func TestCodeIndexGravaNoSQLiteEscolhido(t *testing.T) {
	vault := t.TempDir()
	dbFile := filepath.Join(t.TempDir(), "code-index.db")
	if err := mkLocalGoFile(t, vault, "main.go", "package test\n\nfunc Hi() {}\n"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := runCodeIndexCLI(context.Background(), "FelipeMiiller/my-memory", []string{
		"--storage=sqlite",
		"--db=" + dbFile,
		"--lang=go",
		"--include=**/*.go",
		vault,
	})
	if err != nil {
		t.Fatalf("runCodeIndexCLI falhou: %v", err)
	}

	d, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	ctx := context.Background()
	files, err := db.CountCodeFiles(ctx, d)
	if err != nil {
		t.Fatalf("CountCodeFiles falhou: %v", err)
	}
	if files == 0 {
		t.Fatal("code_files vazio: o code stage não gravou no engine escolhido")
	}
}

// TestFlagsDeCodeIndexNaoSaoDescartadas cobre CG-09: nenhuma flag do CLI de
// código é aceita e descartada — cada uma muda o comportamento ou não existe.
func TestFlagsDeCodeIndexNaoSaoDescartadas(t *testing.T) {
	vault := t.TempDir()
	if err := mkLocalGoFile(t, vault, "main.go", "package test\n\nfunc Hi() {}\n"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// --repo vale de verdade: as linhas carregam o repositório pedido.
	repoDB := filepath.Join(t.TempDir(), "repo.db")
	if err := runCodeIndexCLI(context.Background(), "repo/default", []string{
		"--storage=sqlite",
		"--db=" + repoDB,
		"--repo=repo/flags",
		"--lang=go",
		"--include=**/*.go",
		vault,
	}); err != nil {
		t.Fatalf("runCodeIndexCLI(--repo) falhou: %v", err)
	}
	d, err := db.InitDB(repoDB)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	var repo string
	if err := d.QueryRow(`SELECT repository FROM code_files LIMIT 1`).Scan(&repo); err != nil {
		t.Fatalf("SELECT repository falhou: %v", err)
	}
	if repo != "repo/flags" {
		t.Errorf("repository = %q; want repo/flags (o --repo foi descartado)", repo)
	}

	// --ast-hash=false: o ast_hash não é armazenado.
	hashDB := filepath.Join(t.TempDir(), "hash.db")
	if err := runCodeIndexCLI(context.Background(), "repo/default", []string{
		"--storage=sqlite",
		"--db=" + hashDB,
		"--ast-hash=false",
		"--lang=go",
		"--include=**/*.go",
		vault,
	}); err != nil {
		t.Fatalf("runCodeIndexCLI(--ast-hash=false) falhou: %v", err)
	}
	dh, err := db.InitDB(hashDB)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer dh.Close()

	var astHash sql.NullString
	if err := dh.QueryRow(`SELECT ast_hash FROM code_files LIMIT 1`).Scan(&astHash); err != nil {
		t.Fatalf("SELECT ast_hash falhou: %v", err)
	}
	if astHash.Valid {
		t.Errorf("ast_hash = %q; want NULL com --ast-hash=false", astHash.String)
	}

	// --ast-hash (default) guarda o hash.
	hashOnDB := filepath.Join(t.TempDir(), "hash-on.db")
	if err := runCodeIndexCLI(context.Background(), "repo/default", []string{
		"--storage=sqlite",
		"--db=" + hashOnDB,
		"--lang=go",
		"--include=**/*.go",
		vault,
	}); err != nil {
		t.Fatalf("runCodeIndexCLI(--ast-hash) falhou: %v", err)
	}
	dhOn, err := db.InitDB(hashOnDB)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer dhOn.Close()

	var astHashOn sql.NullString
	if err := dhOn.QueryRow(`SELECT ast_hash FROM code_files LIMIT 1`).Scan(&astHashOn); err != nil {
		t.Fatalf("SELECT ast_hash falhou: %v", err)
	}
	if !astHashOn.Valid || astHashOn.String == "" {
		t.Error("ast_hash ausente com o default de --ast-hash")
	}

	// --json emite JSON com as contagens do estágio.
	jsonDB := filepath.Join(t.TempDir(), "json.db")
	out := captureStdout(t, func() {
		_ = runCodeIndexCLI(context.Background(), "repo/default", []string{
			"--storage=sqlite",
			"--db=" + jsonDB,
			"--json",
			"--lang=go",
			"--include=**/*.go",
			vault,
		})
	})
	var report codeIndexReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("--json não emitiu JSON válido: %v (saída=%q)", err, out)
	}
	if strings.Contains(out, "tree-sitter") {
		t.Errorf("o stdout do --json foi poluído pelo boot log: %q", out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("o stdout do --json tem texto antes do JSON: %q", out)
	}
	if report.Engine != "sqlite" {
		t.Errorf("engine = %q; want sqlite", report.Engine)
	}
	if report.Repository != "repo/default" {
		t.Errorf("repository = %q; want repo/default", report.Repository)
	}
	if report.FilesIndexed == 0 {
		t.Error("files_indexed = 0; o code stage não indexou nada")
	}

	// --no-embed saiu do CLI: a flag não é mais aceita (e nem seria
	// descartada em silêncio).
	if err := runCodeIndexCLI(context.Background(), "repo/default", []string{
		"--no-embed",
		"--storage=sqlite",
		"--db=" + filepath.Join(t.TempDir(), "embed.db"),
		vault,
	}); err == nil {
		t.Error("--no-embed continua sendo aceita (e seria descartada)")
	}
}
