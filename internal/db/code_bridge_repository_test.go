package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// bridgeRepoFixture cria um vault SQLite (um arquivo só) para o teste de
// escopo por repositório.
func bridgeRepoFixture(t *testing.T) *sql.DB {
	t.Helper()

	database, err := InitDB(filepath.Join(t.TempDir(), "bridge-repo.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// bridgeEdgeTargets devolve os ids dos nós de código que o documento
// referencia pela ponte `references_code`.
func bridgeEdgeTargets(t *testing.T, database *sql.DB, documentID string) []string {
	t.Helper()

	rows, err := database.Query(`
		SELECT target_id FROM graph_edges
		 WHERE source_id = ? AND relation = ?`,
		documentID, BridgeRelation)
	if err != nil {
		t.Fatalf("SELECT graph_edges(%s): %v", documentID, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var target string
		if err := rows.Scan(&target); err != nil {
			t.Fatalf("SCAN graph_edges(%s): %v", documentID, err)
		}
		out = append(out, target)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("graph_edges(%s): %v", documentID, err)
	}
	return out
}

// TestCodeBridgeRepositoryScope reproduz o defeito P5 no SQLite.
//
// Dois repositórios (`acme` e `globex`) no MESMO arquivo de banco, cada um
// com um símbolo de MESMO qualified name (`pkg.Alpha`) e uma nota que o
// cita. Antes do escopo por repositório:
//
//   - o id do nó era global ("code:pkg.Alpha"), então os dois símbolos
//     colapsavam no MESMO nó do grafo único;
//   - a nota do repo B ficava ligada ao nó do símbolo do repo A.
//
// O escopo vive no id do nó ("code:<repository>:<qualified_name>") e no
// filtro por repository da carga de símbolos — sem rebuild de tabela e sem
// mexer na PK de `graph_nodes`.
func TestCodeBridgeRepositoryScope(t *testing.T) {
	database := bridgeRepoFixture(t)

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

	// 1. Cada repo tem a sua nota e o SEU símbolo (mesmo qualified name).
	for _, r := range repos {
		ctx := WithRepository(context.Background(), r.repo)
		if err := InsertDocumentWithMeta(ctx, database, r.docID, r.docPath, r.repo,
			1700000000, "hash-"+r.repo, "", "", "resource", 0, 0); err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s): %v", r.repo, err)
		}
		if err := InsertChunk(ctx, database, "chunk-"+r.docID, r.docID,
			"A nota de "+r.repo+" descreve "+qualifiedName+".", 0, nil); err != nil {
			t.Fatalf("InsertChunk(%s): %v", r.repo, err)
		}
		bridgeSeedSymbol(t, ctx, database, r.file, "Alpha", qualifiedName)
	}

	// 2. A ponte roda para cada repositório.
	for _, r := range repos {
		ctx := WithRepository(context.Background(), r.repo)
		result, err := BuildCodeBridgeForDocument(ctx, database, r.docID)
		if err != nil {
			t.Fatalf("BuildCodeBridgeForDocument(%s): %v", r.repo, err)
		}
		// (e) o repo só enxerga os SÍMBOLOS dele: o outro não é oferecido.
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

	acmeNode := CodeSymbolNodeID("acme", qualifiedName)
	globexNode := CodeSymbolNodeID("globex", qualifiedName)
	if acmeNode == globexNode {
		t.Fatalf("os dois repositórios geraram o MESMO id de nó (%q)", acmeNode)
	}

	// (a) existem DOIS nós `code_symbol` distintos.
	var nodes int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM graph_nodes WHERE type = ?`, CodeSymbolNodeType).Scan(&nodes); err != nil {
		t.Fatalf("COUNT graph_nodes: %v", err)
	}
	if nodes != 2 {
		t.Fatalf("nós code_symbol = %d; want 2 (um por repositório)", nodes)
	}
	for _, nodeID := range []string{acmeNode, globexNode} {
		var nodeType string
		if err := database.QueryRow(
			`SELECT type FROM graph_nodes WHERE id = ?`, nodeID).Scan(&nodeType); err != nil {
			t.Fatalf("graph_nodes sem o nó %q: %v", nodeID, err)
		}
		if nodeType != CodeSymbolNodeType {
			t.Errorf("tipo do nó %q = %q; want %q", nodeID, nodeType, CodeSymbolNodeType)
		}
	}

	// (b) a nota do repo A aponta SÓ para o nó do repo A.
	targets := bridgeEdgeTargets(t, database, "doc-acme")
	if len(targets) != 1 || targets[0] != acmeNode {
		t.Errorf("arestas de doc-acme = %v; want [%s]", targets, acmeNode)
	}

	// (c) a nota do repo B aponta SÓ para o nó do repo B.
	targets = bridgeEdgeTargets(t, database, "doc-globex")
	if len(targets) != 1 || targets[0] != globexNode {
		t.Errorf("arestas de doc-globex = %v; want [%s]", targets, globexNode)
	}

	// (d) CodeBridgeDocumentsForSymbol devolve só os documentos do próprio repo.
	acmeDocs, err := CodeBridgeDocumentsForSymbol(
		WithRepository(context.Background(), "acme"), database, qualifiedName)
	if err != nil {
		t.Fatalf("CodeBridgeDocumentsForSymbol(acme): %v", err)
	}
	if len(acmeDocs) != 1 || acmeDocs[0].DocumentID != "doc-acme" {
		t.Errorf("documentos de acme = %v; want [doc-acme]", acmeDocs)
	}
	globexDocs, err := CodeBridgeDocumentsForSymbol(
		WithRepository(context.Background(), "globex"), database, qualifiedName)
	if err != nil {
		t.Fatalf("CodeBridgeDocumentsForSymbol(globex): %v", err)
	}
	if len(globexDocs) != 1 || globexDocs[0].DocumentID != "doc-globex" {
		t.Errorf("documentos de globex = %v; want [doc-globex]", globexDocs)
	}

	// (e) a resolução do nó devolve o símbolo do repositório do contexto.
	ref, err := CodeSymbolByNodeID(
		WithRepository(context.Background(), "acme"), database, acmeNode)
	if err != nil {
		t.Fatalf("CodeSymbolByNodeID(acme): %v", err)
	}
	if ref == nil || ref.File != "acme/pkg/alpha.go" {
		t.Fatalf("símbolo de acme = %+v; want o arquivo acme/pkg/alpha.go", ref)
	}
	ref, err = CodeSymbolByNodeID(
		WithRepository(context.Background(), "globex"), database, globexNode)
	if err != nil {
		t.Fatalf("CodeSymbolByNodeID(globex): %v", err)
	}
	if ref == nil || ref.File != "globex/pkg/alpha.go" {
		t.Fatalf("símbolo de globex = %+v; want o arquivo globex/pkg/alpha.go", ref)
	}
}

// TestCodeSymbolNodeIDEscopado fixa o formato do id do nó de código e o
// fallback do repositório vazio.
func TestCodeSymbolNodeIDEscopado(t *testing.T) {
	cases := []struct {
		repository string
		qualified  string
		want       string
	}{
		{"acme", "pkg.Alpha", "code:acme:pkg.Alpha"},
		{" acme ", " pkg.Alpha ", "code:acme:pkg.Alpha"},
		{"acme", "", ""},
		{"", "pkg.Alpha", "code:default:pkg.Alpha"},
	}
	for _, c := range cases {
		got := CodeSymbolNodeID(c.repository, c.qualified)
		if got != c.want {
			t.Errorf("CodeSymbolNodeID(%q, %q) = %q; want %q",
				c.repository, c.qualified, got, c.want)
		}
	}
}

// TestCodeSymbolQualifiedNameFromNodeID fixa o parsing do id do nó: o
// formato novo (com repository), o legado (sem repository) e o qualified
// name puro.
func TestCodeSymbolQualifiedNameFromNodeID(t *testing.T) {
	cases := []struct {
		nodeID string
		want   string
	}{
		{"code:acme:pkg.Alpha", "pkg.Alpha"},
		{"  code:acme:pkg.Alpha  ", "pkg.Alpha"},
		{"code:pkg.Alpha", "pkg.Alpha"},
		{"pkg.Alpha", "pkg.Alpha"},
		{"code:", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := codeSymbolQualifiedNameFromNodeID(c.nodeID)
		if got != c.want {
			t.Errorf("codeSymbolQualifiedNameFromNodeID(%q) = %q; want %q",
				c.nodeID, got, c.want)
		}
	}
}
