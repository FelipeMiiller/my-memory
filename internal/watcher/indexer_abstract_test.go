package watcher

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/parser"
)

// O corpo do caso 3 é propositalmente longo: o fallback do parser tem que
// TRUNCAR em 160 caracteres, e um corpo curto não exercitaria o corte.
const (
	// absWithSummary é o `summary` do frontmatter do caso 1.
	absWithSummary = "Resumo micro gravado pelo indexador"

	// absWithAbstract é o `abstract` do frontmatter do caso 2.
	absWithAbstract = "Abstract explicito do frontmatter"

	// absFallbackBody é o corpo do caso 3 (sem summary e sem abstract).
	absFallbackBody = "Este paragrafo narrativo do corpo e propositalmente longo para forcar o corte de " +
		"cento e sessenta caracteres do fallback do parser, porque um texto curto nao provaria nada sobre " +
		"a truncagem."
)

// TestIndexSingleFile_Abstract prova a ESCOLHA do `abstract` gravado pelo
// indexador (AC5, AC6 e AC7 da spec fix-progressive-context-test-coverage).
//
// POR QUE ESTE TESTE EXISTE: o gap PCL-03 (ISSUE-044) é que o indexador não
// era asserido. O teste antigo (internal/parser/parser_test.go) verifica o
// PARSER — e o parser continua correto com o indexador quebrado, porque são
// duas unidades diferentes. Por isso o mutante M2 (o indexador deixar de usar
// o `summary` como `abstract`) sobreviveu.
//
// O QUE ESTE TESTE PROVA, ramo a ramo da cadeia de escolha do indexador:
//  1. documento com `summary` → `documents.abstract` é o summary VERBATIM;
//  2. documento com `abstract` e sem `summary` → `documents.abstract` é o
//     abstract do frontmatter;
//  3. documento sem nenhum dos dois → `documents.abstract` é o fallback do
//     parser, truncado em no máximo 160 caracteres.
func TestIndexSingleFile_Abstract(t *testing.T) {
	tmpDir := t.TempDir()

	database, err := db.InitDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "binary was compiled without cgo") {
			t.Skip("Pulando teste SQLite: ambiente sem CGO")
		}
		t.Fatalf("erro ao iniciar banco SQLite: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// 1. Documento com `summary`: o summary vira o abstract, verbatim.
	summaryDoc := "---\n" +
		"title: Nota com summary\n" +
		"category: memory\n" +
		"summary: " + absWithSummary + "\n" +
		"---\n\n" +
		"Corpo da nota que nao deve virar o abstract.\n"
	summaryPath := filepath.Join(tmpDir, "nota-com-summary.md")
	if err := os.WriteFile(summaryPath, []byte(summaryDoc), 0644); err != nil {
		t.Fatalf("erro ao escrever a nota com summary: %v", err)
	}
	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, summaryPath, false); err != nil {
		t.Fatalf("erro ao indexar a nota com summary: %v", err)
	}
	gotSummary := readDocumentAbstract(t, ctx, database, summaryPath)
	if gotSummary != absWithSummary {
		t.Errorf("documents.abstract = %q; esperava o summary verbatim %q", gotSummary, absWithSummary)
	}

	// 2. Documento com `abstract` e sem `summary`: o abstract do frontmatter
	//    vira o abstract gravado.
	abstractDoc := "---\n" +
		"title: Nota com abstract\n" +
		"category: skill\n" +
		"abstract: " + absWithAbstract + "\n" +
		"---\n\n" +
		"Corpo da nota que nao deve virar o abstract.\n"
	abstractPath := filepath.Join(tmpDir, "nota-com-abstract.md")
	if err := os.WriteFile(abstractPath, []byte(abstractDoc), 0644); err != nil {
		t.Fatalf("erro ao escrever a nota com abstract: %v", err)
	}
	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, abstractPath, false); err != nil {
		t.Fatalf("erro ao indexar a nota com abstract: %v", err)
	}
	gotAbstract := readDocumentAbstract(t, ctx, database, abstractPath)
	if gotAbstract != absWithAbstract {
		t.Errorf("documents.abstract = %q; esperava o abstract do frontmatter %q", gotAbstract, absWithAbstract)
	}

	// 3. Documento sem `summary` e sem `abstract`: o fallback do parser, com
	//    no máximo 160 caracteres.
	fallbackDoc := "---\n" +
		"title: Nota sem summary nem abstract\n" +
		"category: resource\n" +
		"---\n\n" +
		absFallbackBody + "\n"
	fallbackPath := filepath.Join(tmpDir, "nota-sem-resumo.md")
	if err := os.WriteFile(fallbackPath, []byte(fallbackDoc), 0644); err != nil {
		t.Fatalf("erro ao escrever a nota sem resumo: %v", err)
	}
	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, fallbackPath, false); err != nil {
		t.Fatalf("erro ao indexar a nota sem resumo: %v", err)
	}
	gotFallback := readDocumentAbstract(t, ctx, database, fallbackPath)

	_, body := parser.ExtractFrontmatter(fallbackDoc)
	wantFallback := parser.ExtractMicroAbstract(body, 160)
	if gotFallback != wantFallback {
		t.Errorf("documents.abstract = %q; esperava o fallback do parser %q", gotFallback, wantFallback)
	}
	if n := len([]rune(gotFallback)); n > 160 {
		t.Errorf("documents.abstract tem %d caracteres; o fallback tem no máximo 160", n)
	}
	if gotFallback == "" {
		t.Errorf("documents.abstract ficou vazio; esperava o fallback do parser")
	}
}

// readDocumentAbstract lê o valor gravado na coluna documents.abstract.
func readDocumentAbstract(t *testing.T, ctx context.Context, database *sql.DB, docID string) string {
	t.Helper()

	var abstract string
	err := database.QueryRowContext(ctx, `SELECT abstract FROM documents WHERE id = ?`, docID).Scan(&abstract)
	if err != nil {
		t.Fatalf("erro ao ler documents.abstract de %q: %v", docID, err)
	}
	return abstract
}
