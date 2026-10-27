package db_test

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// O teste de integração PostgreSQL da ponte notas↔código (ADR-061).
//
// POR QUE ELE EXISTE: o defeito P5 (o id do nó de código colapsando entre
// repositórios) era visível no SQLite. O PostgreSQL já tinha a identidade
// (repository, id) em `graph_nodes`, então este teste é a TRAVA: garante que
// o id escopado por repositório ("code:<repository>:<qualified_name>") e o
// filtro por repository da carga de símbolos continuam corretos no engine
// que o dono usa.
//
// SEGURANÇA: o teste NÃO escreve no banco configurado em MY_MEMORY_PG_URL.
// O harness pgtest cria um BANCO TEMPORÁRIO próprio (`mymemory_pgtest_*`),
// roda tudo dentro dele e o derruba no fim. O `default` role do dono nunca é
// usado para tocar o banco de produção.
//
// COMO RODAR:
//
//	MY_MEMORY_PG_URL=<url> MY_MEMORY_PG_TEST=1 go test -count=1 ./internal/db/...
//
// Sem MY_MEMORY_PG_TEST=1 o teste é pulado (a suíte padrão é hermética).
func TestCodeBridgeRepositoryScopePostgres(t *testing.T) {
	// Banco DESCARTÁVEL próprio: o Postgres do dono nunca é alvo.
	h := pgtest.New(t)

	ctx := context.Background()

	// Aplica o schema do engine no banco temporário.
	pgStore, err := store.NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	database := pgStore.DB()
	defer pgStore.Close()

	var version string
	if err := database.QueryRow("SELECT version()").Scan(&version); err != nil {
		t.Fatalf("SELECT version(): %v", err)
	}
	t.Logf("PostgreSQL: %s", strings.SplitN(version, " on ", 2)[0])

	if db.DetectDialect(ctx, database) != db.DialectPostgres {
		t.Fatalf("dialeto = %q; want %q", db.DetectDialect(ctx, database), db.DialectPostgres)
	}

	// 1. Dois repositórios, MESMO qualified name, cada um com a sua nota.
	const qualifiedName = "pkg.Alpha"
	repos := []struct {
		repo    string
		docID   string
		file    string
		docPath string
	}{
		{repo: "acme", docID: "doc-acme", file: "acme/pkg/alpha.go", docPath: "docs/acme.md"},
		{repo: "globex", docID: "doc-globex", file: "globex/pkg/alpha.go", docPath: "docs/globex.md"},
	}

	for _, r := range repos {
		repoCtx := db.WithRepository(ctx, r.repo)
		// InsertDocumentWithMeta (o caminho real do indexador) cria também o
		// nó `note` do documento em graph_nodes — de onde sai a FK da aresta.
		if err := pgStore.InsertDocumentWithMeta(repoCtx, r.repo, r.docID, r.docPath, r.repo,
			1700000000, "hash-"+r.repo, "", "", "resource", 0, 0); err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s): %v", r.repo, err)
		}
		if _, err := database.ExecContext(repoCtx, `
			INSERT INTO chunks (id, repository, document_id, chunk_index, content)
			VALUES ($1, $2, $3, 0, $4)`,
			"chunk-"+r.docID, r.repo, r.docID,
			"A nota de "+r.repo+" descreve "+qualifiedName+"."); err != nil {
			t.Fatalf("INSERT chunks(%s): %v", r.repo, err)
		}

		fileID, err := db.UpsertCodeFile(repoCtx, database, r.file, "go",
			"content-hash-"+r.file, "ast-hash-"+r.file, 128, 1700000000)
		if err != nil {
			t.Fatalf("UpsertCodeFile(%s): %v", r.repo, err)
		}
		if _, err := db.UpsertCodeSymbol(repoCtx, database, fileID, "function", "Alpha", qualifiedName,
			"func Alpha()", "", 1, 3, 0, 0); err != nil {
			t.Fatalf("UpsertCodeSymbol(%s): %v", r.repo, err)
		}
	}

	// 2. A ponte roda para cada repositório (no PostgreSQL o documento
	// também é escopado por repository).
	for _, r := range repos {
		repoCtx := db.WithRepository(ctx, r.repo)
		result, err := db.BuildCodeBridge(repoCtx, database)
		if err != nil {
			t.Fatalf("BuildCodeBridge(%s): %v", r.repo, err)
		}
		if result.Symbols != 1 {
			t.Errorf("[%s] símbolos considerados = %d; want 1 (só os do próprio repo)",
				r.repo, result.Symbols)
		}
		if result.Matched != 1 {
			t.Errorf("[%s] casamentos = %d; want 1", r.repo, result.Matched)
		}
		if result.Edges != 1 {
			t.Errorf("[%s] arestas = %d; want 1", r.repo, result.Edges)
		}
	}

	acmeNode := db.CodeSymbolNodeID("acme", qualifiedName)
	globexNode := db.CodeSymbolNodeID("globex", qualifiedName)
	if acmeNode == globexNode {
		t.Fatalf("os dois repositórios geraram o MESMO id de nó (%q)", acmeNode)
	}

	// (a) DOIS nós `code_symbol` distintos.
	var nodes int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM graph_nodes WHERE type = $1`, db.CodeSymbolNodeType).Scan(&nodes); err != nil {
		t.Fatalf("COUNT graph_nodes: %v", err)
	}
	if nodes != 2 {
		t.Fatalf("nós code_symbol = %d; want 2 (um por repositório)", nodes)
	}
	for _, nodeID := range []string{acmeNode, globexNode} {
		var nodeType string
		if err := database.QueryRow(
			`SELECT type FROM graph_nodes WHERE id = $1`, nodeID).Scan(&nodeType); err != nil {
			t.Fatalf("graph_nodes sem o nó %q: %v", nodeID, err)
		}
		if nodeType != db.CodeSymbolNodeType {
			t.Errorf("tipo do nó %q = %q; want %q", nodeID, nodeType, db.CodeSymbolNodeType)
		}
	}

	// (b) e (c) cada nota aponta SÓ para o nó do próprio repositório.
	for _, r := range repos {
		want := db.CodeSymbolNodeID(r.repo, qualifiedName)
		rows, err := database.Query(`
			SELECT target_id FROM graph_edges
			 WHERE source_id = $1 AND relation = $2`, r.docID, db.BridgeRelation)
		if err != nil {
			t.Fatalf("SELECT graph_edges(%s): %v", r.docID, err)
		}
		var targets []string
		for rows.Next() {
			var target string
			if err := rows.Scan(&target); err != nil {
				rows.Close()
				t.Fatalf("SCAN graph_edges(%s): %v", r.docID, err)
			}
			targets = append(targets, target)
		}
		rows.Close()
		if len(targets) != 1 || targets[0] != want {
			t.Errorf("arestas de %s = %v; want [%s]", r.docID, targets, want)
		}
	}

	// (d) CodeBridgeDocumentsForSymbol devolve só os documentos do próprio repo.
	acmeDocs, err := db.CodeBridgeDocumentsForSymbol(
		db.WithRepository(ctx, "acme"), database, qualifiedName)
	if err != nil {
		t.Fatalf("CodeBridgeDocumentsForSymbol(acme): %v", err)
	}
	if len(acmeDocs) != 1 || acmeDocs[0].DocumentID != "doc-acme" {
		t.Errorf("documentos de acme = %v; want [doc-acme]", acmeDocs)
	}
	globexDocs, err := db.CodeBridgeDocumentsForSymbol(
		db.WithRepository(ctx, "globex"), database, qualifiedName)
	if err != nil {
		t.Fatalf("CodeBridgeDocumentsForSymbol(globex): %v", err)
	}
	if len(globexDocs) != 1 || globexDocs[0].DocumentID != "doc-globex" {
		t.Errorf("documentos de globex = %v; want [doc-globex]", globexDocs)
	}

	// (e) a resolução do nó devolve o símbolo do repositório do contexto.
	ref, err := db.CodeSymbolByNodeID(db.WithRepository(ctx, "acme"), database, acmeNode)
	if err != nil {
		t.Fatalf("CodeSymbolByNodeID(acme): %v", err)
	}
	if ref == nil || ref.File != "acme/pkg/alpha.go" {
		t.Fatalf("símbolo de acme = %+v; want o arquivo acme/pkg/alpha.go", ref)
	}
	ref, err = db.CodeSymbolByNodeID(db.WithRepository(ctx, "globex"), database, globexNode)
	if err != nil {
		t.Fatalf("CodeSymbolByNodeID(globex): %v", err)
	}
	if ref == nil || ref.File != "globex/pkg/alpha.go" {
		t.Fatalf("símbolo de globex = %+v; want o arquivo globex/pkg/alpha.go", ref)
	}
}
