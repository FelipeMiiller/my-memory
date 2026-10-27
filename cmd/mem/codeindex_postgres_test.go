package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/codeast"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// capturePgAdminURL guarda a URL do Postgres antes que o TestMain de
// cmd/mem/main_test.go a neutralize.
//
// POR QUE UM init(): o init() de um pacote roda na inicialização do binário,
// ou seja, ANTES do TestMain. A captura tem que acontecer nessa ordem — depois
// que o TestMain limpa a variável, o teste de integração não tem mais como
// chegar ao servidor para criar o banco descartável.
//
// O TestMain do cmd/mem já neutraliza MY_MEMORY_PG_URL (e as demais variáveis
// de storage) e já desliga a descoberta de .env, então o resto do pacote está
// isolado. O que faltava era o acesso controlado ao Postgres para o teste de
// integração — é isto aqui.
//
// COMO RODAR:
//
//	MY_MEMORY_PG_URL=<url> MY_MEMORY_PG_TEST=1 go test -count=1 ./cmd/mem/...
func init() {
	pgtest.Capture()
}

// TestCodeIndexCLI_PostgresGravaCodeGraphNoEngineConfigurado prova o item (d)
// da ISSUE-031 §P1: a metade Postgres do critério de aceitação H4 — "code graph
// gravado no engine configurado, dialeto certo".
//
// POR QUE IMPORTA: o code graph é multi-tenant (ADR-061, CG-05/CG-06) e o
// `mem code-index` tem que gravá-lo no ENGINE CONFIGURADO — Postgres quando
// Postgres, SQLite quando SQLite. A validação anterior (H4) só tinha a metade
// SQLite; o Postgres nunca foi executado, e a hipótese "funciona igual lá"
// continuava sem prova.
//
// O QUE ESTE TESTE PROVA:
//  1. o comando escolhe o engine Postgres (`--storage=postgres`) e o relatório
//     JSON diz "engine": "postgres";
//  2. as linhas de code_files e code_symbols GRAVARAM no Postgres descartável;
//  3. os VALORES estão na coluna certa (content_hash é o sha256 do arquivo,
//     size_bytes é o tamanho, path é o caminho relativo, qualified_name é
//     pkg.<Nome>) — um binding deslocado trocaria as colunas;
//  4. a aresta `calls` foi gravada com o PONTO DE CHAMADA;
//  5. NENHUM arquivo SQLite foi criado — o code graph não foi para um SQLite
//     local paralelo.
func TestCodeIndexCLI_PostgresGravaCodeGraphNoEngineConfigurado(t *testing.T) {
	h := pgtest.New(t)

	// Vault de fixture: um pacote Go com uma chamada entre duas funções.
	vault := t.TempDir()
	pkgDir := filepath.Join(vault, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir do vault de fixture: %v", err)
	}
	// Alpha está na linha 3 e CHAMA Beta na linha 4.
	const alphaSrc = "package pkg\n\nfunc Alpha() string {\n\treturn Beta()\n}\n"
	const betaSrc = "package pkg\n\nfunc Beta() string {\n\treturn \"b\"\n}\n"
	if err := os.WriteFile(filepath.Join(pkgDir, "alpha.go"), []byte(alphaSrc), 0o644); err != nil {
		t.Fatalf("escrita de alpha.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "beta.go"), []byte(betaSrc), 0o644); err != nil {
		t.Fatalf("escrita de beta.go: %v", err)
	}

	// 1. O comando real, com o engine Postgres apontando para o banco
	//    descartável.
	var cliErr error
	out := captureStdout(t, func() {
		cliErr = runCodeIndexCLI(context.Background(), "repo-h4", []string{
			"--dir=" + vault,
			"--storage=postgres",
			"--postgres=" + h.DSN,
			"--repo=repo-h4",
			"--json",
		})
	})
	if cliErr != nil {
		t.Fatalf("runCodeIndexCLI: %v", cliErr)
	}

	var report struct {
		Engine       string `json:"engine"`
		Repository   string `json:"repository"`
		FilesScanned int    `json:"files_scanned"`
		FilesIndexed int    `json:"files_indexed"`
		SymbolsTotal int    `json:"symbols_total"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("JSON do code-index inválido: %v\nsaída bruta:\n%s", err, out)
	}
	if report.Engine != "postgres" {
		t.Errorf("engine = %q; want %q (o code graph tem que ir para o engine configurado)", report.Engine, "postgres")
	}
	if report.Repository != "repo-h4" {
		t.Errorf("repository = %q; want %q", report.Repository, "repo-h4")
	}
	if report.FilesIndexed != 2 {
		t.Errorf("files_indexed = %d; want 2 (alpha.go e beta.go)", report.FilesIndexed)
	}
	if report.SymbolsTotal < 2 {
		t.Errorf("symbols_total = %d; want >= 2 (Alpha e Beta)", report.SymbolsTotal)
	}

	// 2. As linhas foram gravadas NO POSTGRES descartável.
	pgStore, err := store.NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	defer pgStore.Close()
	database := pgStore.DB()

	// 2.1. code_files: path, language, content_hash e size_bytes CORRETOS.
	wantFiles := map[string]struct {
		language string
		hash     string
		size     int64
	}{
		"pkg/alpha.go": {language: "go", hash: codeast.HashContent([]byte(alphaSrc)), size: int64(len(alphaSrc))},
		"pkg/beta.go":  {language: "go", hash: codeast.HashContent([]byte(betaSrc)), size: int64(len(betaSrc))},
	}

	rows, err := database.Query(`
		SELECT path, language, content_hash, size_bytes, repository
		FROM code_files
		WHERE repository = $1
		ORDER BY path`, "repo-h4")
	if err != nil {
		t.Fatalf("SELECT code_files: %v", err)
	}
	gotFiles := 0
	for rows.Next() {
		var path, language, contentHash, repository string
		var sizeBytes int64
		if err := rows.Scan(&path, &language, &contentHash, &sizeBytes, &repository); err != nil {
			rows.Close()
			t.Fatalf("SCAN code_files: %v", err)
		}
		want, ok := wantFiles[path]
		if !ok {
			t.Errorf("code_files tem o path inesperado %q", path)
			continue
		}
		gotFiles++
		if language != want.language {
			t.Errorf("code_files[%s].language = %q; want %q", path, language, want.language)
		}
		if contentHash != want.hash {
			t.Errorf("code_files[%s].content_hash = %q; want %q (o valor não aterrissou na coluna certa)", path, contentHash, want.hash)
		}
		if sizeBytes != want.size {
			t.Errorf("code_files[%s].size_bytes = %d; want %d", path, sizeBytes, want.size)
		}
		if repository != "repo-h4" {
			t.Errorf("code_files[%s].repository = %q; want %q", path, repository, "repo-h4")
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("rows code_files: %v", err)
	}
	if gotFiles != len(wantFiles) {
		t.Errorf("linhas em code_files = %d; want %d (pkg/alpha.go e pkg/beta.go)", gotFiles, len(wantFiles))
	}

	// 2.2. code_symbols: qualified_name, kind e a linha da DEFINIÇÃO.
	wantSymbols := map[string]struct {
		kind      string
		startLine int
	}{
		"pkg.Alpha": {kind: "function", startLine: 3},
		"pkg.Beta":  {kind: "function", startLine: 3},
	}

	rows, err = database.Query(`
		SELECT qualified_name, kind, start_line, repository
		FROM code_symbols
		WHERE repository = $1
		ORDER BY qualified_name`, "repo-h4")
	if err != nil {
		t.Fatalf("SELECT code_symbols: %v", err)
	}
	gotSymbols := 0
	for rows.Next() {
		var qualifiedName, kind, repository string
		var startLine int
		if err := rows.Scan(&qualifiedName, &kind, &startLine, &repository); err != nil {
			rows.Close()
			t.Fatalf("SCAN code_symbols: %v", err)
		}
		want, ok := wantSymbols[qualifiedName]
		if !ok {
			t.Errorf("code_symbols tem o qualified_name inesperado %q", qualifiedName)
			continue
		}
		gotSymbols++
		if kind != want.kind {
			t.Errorf("code_symbols[%s].kind = %q; want %q", qualifiedName, kind, want.kind)
		}
		if startLine != want.startLine {
			t.Errorf("code_symbols[%s].start_line = %d; want %d", qualifiedName, startLine, want.startLine)
		}
		if repository != "repo-h4" {
			t.Errorf("code_symbols[%s].repository = %q; want %q", qualifiedName, repository, "repo-h4")
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("rows code_symbols: %v", err)
	}
	if gotSymbols != len(wantSymbols) {
		t.Errorf("linhas em code_symbols = %d; want %d", gotSymbols, len(wantSymbols))
	}

	// 2.3. A aresta `calls` foi gravada com o PONTO DE CHAMADA (linha 4), não
	//      com a linha da definição (linha 3).
	var edgeStartLine int
	var edgeKind, srcQN, dstQN string
	err = database.QueryRow(`
		SELECT e.kind, e.start_line, src.qualified_name, dst.qualified_name
		FROM code_edges e
		JOIN code_symbols src ON src.id = e.src_symbol_id
		JOIN code_symbols dst ON dst.id = e.dst_symbol_id
		WHERE e.repository = $1 AND src.qualified_name = $2 AND dst.qualified_name = $3`,
		"repo-h4", "pkg.Alpha", "pkg.Beta").Scan(&edgeKind, &edgeStartLine, &srcQN, &dstQN)
	if err != nil {
		t.Fatalf("SELECT code_edges (calls): %v", err)
	}
	if edgeKind != "calls" {
		t.Errorf("kind da aresta = %q; want %q", edgeKind, "calls")
	}
	if srcQN != "pkg.Alpha" || dstQN != "pkg.Beta" {
		t.Errorf("aresta = %s -> %s; want pkg.Alpha -> pkg.Beta", srcQN, dstQN)
	}
	if edgeStartLine != 4 {
		t.Errorf("start_line da aresta = %d; want 4 (a linha da CHAMADA, não a da definição)", edgeStartLine)
	}

	// 3. NENHUM SQLite foi criado: o code graph não foi para um SQLite local
	//    paralelo (o defeito CG-05).
	var sqliteFiles []string
	err = filepath.WalkDir(vault, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		lower := strings.ToLower(d.Name())
		if strings.HasSuffix(lower, ".db") || strings.Contains(lower, ".sqlite") {
			sqliteFiles = append(sqliteFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varredura do vault: %v", err)
	}
	if len(sqliteFiles) > 0 {
		t.Errorf("o code-index criou arquivo(s) SQLite no vault: %v (o code graph tem que ir só para o engine configurado)", sqliteFiles)
	}
}
