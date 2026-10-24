package drift

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/codeast"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

// structuralFixture prepara um repo temporário com DOIS arquivos Go:
//
//	pkg/alpha.go — vai receber uma mudança de ASSINATURA;
//	pkg/beta.go  — vai receber só uma mudança de COMENTÁRIO.
//
// Os dois são indexados pelo code stage (ast_hash gravado) e referenciados
// por uma nota (a ponte `references_code`).
func structuralFixture(t *testing.T) (*MockGitRunner, string, string, *sql.DB) {
	t.Helper()

	ctx := context.Background()
	repoRoot := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "structural.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	originals := map[string]string{
		"pkg/alpha.go": "package pkg\n\n// Alpha faz o trabalho.\nfunc Alpha(a int) error {\n\treturn nil\n}\n",
		"pkg/beta.go":  "package pkg\n\n// Beta faz outro trabalho.\nfunc Beta() error {\n\treturn nil\n}\n",
	}

	symbolsByFile := map[string][]codeast.Symbol{}
	for rel, content := range originals {
		full := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		parsed, err := codeast.NewGoParser().ParseFile(rel, []byte(content))
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", rel, err)
		}
		symbolsByFile[rel] = parsed.Symbols

		astHash := codeast.HashSymbols(parsed.Symbols)
		fileID, err := db.UpsertCodeFile(ctx, database, rel, "go", "hash-"+rel, astHash,
			int64(len(content)), 1700000000)
		if err != nil {
			t.Fatalf("UpsertCodeFile(%s): %v", rel, err)
		}
		for _, s := range parsed.Symbols {
			if _, err := db.UpsertCodeSymbol(ctx, database, fileID, string(s.Kind), s.Name,
				s.QualifiedName, s.Signature, s.DocComment, s.StartLine, s.EndLine,
				s.StartCol, s.EndCol); err != nil {
				t.Fatalf("UpsertCodeSymbol(%s): %v", s.QualifiedName, err)
			}
		}
	}

	// A nota que referencia os símbolos (a ponte `references_code`).
	const docID = "docs/nota-alpha.md"
	if err := db.InsertDocumentWithMeta(ctx, database, docID, docID, "Nota Alpha",
		1700000000, "doc-hash", "", "", "resource", 0, 0); err != nil {
		t.Fatalf("InsertDocumentWithMeta: %v", err)
	}
	if err := db.InsertChunk(ctx, database, "chunk-alpha", docID,
		"A nota descreve pkg.Alpha e pkg.Beta.", 0, nil); err != nil {
		t.Fatalf("InsertChunk: %v", err)
	}
	for _, qn := range []string{"pkg.Alpha", "pkg.Beta"} {
		nodeID := db.CodeSymbolNodeID(qn)
		if _, err := database.ExecContext(ctx,
			`INSERT OR IGNORE INTO graph_nodes (id, type, name) VALUES (?, ?, ?)`,
			nodeID, db.CodeSymbolNodeType, qn); err != nil {
			t.Fatalf("graph_nodes(%s): %v", nodeID, err)
		}
		if _, err := database.ExecContext(ctx, `
			INSERT OR REPLACE INTO graph_edges (source_id, target_id, relation, epistemic_status, weight)
			VALUES (?, ?, ?, ?, 1.0)`,
			docID, nodeID, db.BridgeRelation, db.EpistemicExtracted); err != nil {
			t.Fatalf("graph_edges(%s): %v", nodeID, err)
		}
	}

	mock := &MockGitRunner{
		Commits: []GitCommit{
			{
				Hash:      "abc1234",
				ShortHash: "abc1234",
				Author:    "Dev",
				Date:      time.Now().Add(-2 * time.Hour),
				Message:   "chore: adjust the code",
			},
		},
		Changes: []FileChange{
			{Path: "pkg/alpha.go", Status: "M", Additions: 2, Deletions: 1, IsCodeFile: true},
			{Path: "pkg/beta.go", Status: "M", Additions: 1, Deletions: 0, IsCodeFile: true},
		},
	}

	return mock, repoRoot, docID, database
}

// TestDriftDetectaMudancaEstrutural cobre o AC 26/27:
//
//   - o arquivo com mudança de ASSINATURA é drift estrutural;
//   - o arquivo com mudança só de COMENTÁRIO NÃO é drift (o ast_hash não
//     muda);
//   - um arquivo SEM ast_hash gravado não quebra o drift.
func TestDriftDetectaMudancaEstrutural(t *testing.T) {
	ctx := context.Background()
	mock, repoRoot, docID, database := structuralFixture(t)

	// 1. alpha.go: mudança de ASSINATURA (o mesmo número de linhas).
	alphaChanged := "package pkg\n\n// Alpha faz o trabalho.\nfunc Alpha(a int, b string) error {\n\treturn nil\n}\n"
	if err := os.WriteFile(filepath.Join(repoRoot, "pkg", "alpha.go"), []byte(alphaChanged), 0o644); err != nil {
		t.Fatalf("write alpha: %v", err)
	}

	// 2. beta.go: só o TEXTO do comentário muda (o mesmo número de linhas):
	// o ast_hash do arquivo não muda — é a mudança COSMÉTICA.
	betaChanged := "package pkg\n\n// Beta faz um trabalho diferente.\nfunc Beta() error {\n\treturn nil\n}\n"
	if err := os.WriteFile(filepath.Join(repoRoot, "pkg", "beta.go"), []byte(betaChanged), 0o644); err != nil {
		t.Fatalf("write beta: %v", err)
	}

	report, err := AnalyzeDrift(ctx, mock, database, repoRoot, "Vault", "HEAD~1..HEAD", 0.1, true)
	if err != nil {
		t.Fatalf("AnalyzeDrift: %v", err)
	}

	// 3. Só o alpha.go é mudança estrutural.
	if len(report.StructuralChanges) != 1 {
		t.Fatalf("structural_changes = %d; want 1: %+v", len(report.StructuralChanges), report.StructuralChanges)
	}
	change := report.StructuralChanges[0]
	if change.FilePath != "pkg/alpha.go" {
		t.Errorf("file_path = %q; want %q", change.FilePath, "pkg/alpha.go")
	}
	if change.Attribution != "bridge" {
		t.Errorf("attribution = %q; want %q (a ponte notas↔código)", change.Attribution, "bridge")
	}
	if len(change.NoteIDs) != 1 || change.NoteIDs[0] != docID {
		t.Errorf("note_ids = %v; want [%s]", change.NoteIDs, docID)
	}
	if len(change.SymbolsChanged) == 0 {
		t.Errorf("symbols_changed vazio: %+v", change)
	}

	// 4. A nota que referencia o símbolo aparece com a mudança estrutural.
	if len(report.DriftedNotes) == 0 {
		t.Fatal("esperava ao menos uma nota com desvio")
	}
	found := false
	for _, nd := range report.DriftedNotes {
		if nd.NoteID != docID {
			continue
		}
		found = true
		if len(nd.StructuralChanges) != 1 || nd.StructuralChanges[0] != "pkg/alpha.go" {
			t.Errorf("structural_changes da nota = %v; want [pkg/alpha.go]", nd.StructuralChanges)
		}
	}
	if !found {
		t.Errorf("a nota %q não apareceu no drift: %+v", docID, report.DriftedNotes)
	}

	// 5. O arquivo só de comentário NÃO entrou no drift.
	for _, uc := range report.UncoveredCode {
		if uc.FilePath == "pkg/beta.go" {
			t.Errorf("pkg/beta.go (mudança cosmética) não devia ser drift: %+v", uc)
		}
	}
	for _, nd := range report.DriftedNotes {
		for _, affected := range nd.AffectedFiles {
			if affected == "pkg/beta.go" {
				t.Errorf("pkg/beta.go (mudança cosmética) não devia estar em affected_files: %v", nd.AffectedFiles)
			}
		}
	}
}

// TestDriftSemAstHashNaoQuebra cobre o AC 27: um arquivo SEM ast_hash
// gravado não é erro — o drift segue funcionando (mecanismo textual).
func TestDriftSemAstHashNaoQuebra(t *testing.T) {
	ctx := context.Background()
	mock, repoRoot, docID, database := structuralFixture(t)

	// Apaga o ast_hash gravado: simula um arquivo que o code stage nunca
	// indexou.
	if _, err := database.ExecContext(ctx,
		`UPDATE code_files SET ast_hash = '' WHERE path = 'pkg/alpha.go'`); err != nil {
		t.Fatalf("UPDATE code_files: %v", err)
	}

	alphaChanged := "package pkg\n\n// Alpha faz o trabalho.\nfunc Alpha(a int, b string) error {\n\treturn nil\n}\n"
	if err := os.WriteFile(filepath.Join(repoRoot, "pkg", "alpha.go"), []byte(alphaChanged), 0o644); err != nil {
		t.Fatalf("write alpha: %v", err)
	}

	report, err := AnalyzeDrift(ctx, mock, database, repoRoot, "Vault", "HEAD~1..HEAD", 0.1, true)
	if err != nil {
		t.Fatalf("AnalyzeDrift: %v", err)
	}
	if len(report.StructuralChanges) != 0 {
		t.Errorf("sem ast_hash não há mudança estrutural: %+v", report.StructuralChanges)
	}

	// O drift continua funcionando: a nota é escolhida pelo mecanismo
	// textual (menção do path) — aqui o path é mencionado uma vez só, então
	// o arquivo fica como código órfão, e o drift NÃO quebra.
	if len(report.DriftedNotes) != 0 {
		t.Errorf("drifted_notes = %d; want 0 (menção do path abaixo de MinMatchOccurrences)",
			len(report.DriftedNotes))
	}
	if len(report.UncoveredCode) == 0 {
		t.Errorf("sem ast_hash o arquivo deveria aparecer como código órfão: %+v", report.UncoveredCode)
	}
	_ = docID
}
