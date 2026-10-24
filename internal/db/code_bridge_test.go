package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// bridgeTestFixture cria um vault SQLite com um documento (com chunk) e um
// símbolo de código, e devolve o handle do banco.
func bridgeTestFixture(t *testing.T) *sql.DB {
	t.Helper()

	database, err := InitDB(filepath.Join(t.TempDir(), "bridge.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	return database
}

// bridgeSeedSymbol grava um arquivo + símbolo de código no vault.
func bridgeSeedSymbol(t *testing.T, ctx context.Context, database *sql.DB, path, name, qualifiedName string) int64 {
	t.Helper()

	fileID, err := UpsertCodeFile(ctx, database, path, "go", "content-hash-"+path, "ast-hash-"+path, 128, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	symbolID, err := UpsertCodeSymbol(ctx, database, fileID, "function", name, qualifiedName,
		"func "+name+"()", "", 1, 3, 0, 0)
	if err != nil {
		t.Fatalf("UpsertCodeSymbol: %v", err)
	}
	return symbolID
}

// TestDocumentCodeSymbolEdge cobre a ponte notas↔código (ADR-061, CG-17):
//
//   - o documento que referencia um símbolo existente cria a aresta TIPADA
//     `references_code` no grafo ÚNICO (graph_nodes + graph_edges);
//   - o nó do símbolo é `type='code_symbol'` com id `code:<qualified_name>`;
//   - o status epistêmico é EXTRACTED quando o match foi pelo qualified name
//     literal (ADR-011);
//   - a operação é IDEMPOTENTE: rodar duas vezes não duplica a aresta.
func TestDocumentCodeSymbolEdge(t *testing.T) {
	ctx := context.Background()
	database := bridgeTestFixture(t)
	defer database.Close()

	// 1. Documento que menciona o qualified name do símbolo.
	const docID = "doc-bridge-1"
	if err := InsertDocumentWithMeta(ctx, database, docID, "docs/bridge.md", "Bridge",
		1700000000, "doc-hash", "", "", "resource", 0, 0); err != nil {
		t.Fatalf("InsertDocumentWithMeta: %v", err)
	}
	if err := InsertChunk(ctx, database, "chunk-bridge-1", docID,
		"Esta nota descreve o fluxo foo.Hello e o arquivo pkg/foo.go.", 0, nil); err != nil {
		t.Fatalf("InsertChunk: %v", err)
	}

	// 2. Símbolo de código indexado.
	bridgeSeedSymbol(t, ctx, database, "pkg/foo.go", "Hello", "foo.Hello")

	// 3. A ponte roda.
	result, err := BuildCodeBridge(ctx, database)
	if err != nil {
		t.Fatalf("BuildCodeBridge: %v", err)
	}
	if result.Matched == 0 {
		t.Fatalf("a ponte não casou documento→símbolo: %+v", result)
	}
	if result.Edges == 0 {
		t.Fatalf("a ponte não gravou aresta: %+v", result)
	}

	// 4. O nó do símbolo existe no grafo único, com o tipo certo.
	var nodeType, nodeName string
	err = database.QueryRow(`SELECT type, name FROM graph_nodes WHERE id = ?`, CodeSymbolNodeID("foo.Hello")).
		Scan(&nodeType, &nodeName)
	if err != nil {
		t.Fatalf("graph_nodes sem o nó do símbolo: %v", err)
	}
	if nodeType != CodeSymbolNodeType {
		t.Errorf("tipo do nó = %q; want %q", nodeType, CodeSymbolNodeType)
	}
	if nodeName != "Hello" {
		t.Errorf("nome do nó = %q; want %q", nodeName, "Hello")
	}

	// 5. A aresta é tipada e carrega o status epistêmico.
	var status, relation string
	err = database.QueryRow(`
		SELECT relation, epistemic_status
		  FROM graph_edges
		 WHERE source_id = ? AND target_id = ? AND relation = ?`,
		docID, CodeSymbolNodeID("foo.Hello"), BridgeRelation).Scan(&relation, &status)
	if err != nil {
		t.Fatalf("graph_edges sem a aresta references_code: %v", err)
	}
	if relation != BridgeRelation {
		t.Errorf("relation = %q; want %q", relation, BridgeRelation)
	}
	if status != EpistemicExtracted {
		t.Errorf("epistemic_status = %q; want %q (match literal pelo qualified name)",
			status, EpistemicExtracted)
	}

	// 6. IDEMPOTÊNCIA: rodar a ponte de novo não duplica a aresta.
	second, err := BuildCodeBridge(ctx, database)
	if err != nil {
		t.Fatalf("BuildCodeBridge (2a vez): %v", err)
	}
	var edges int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM graph_edges
		 WHERE source_id = ? AND target_id = ? AND relation = ?`,
		docID, CodeSymbolNodeID("foo.Hello"), BridgeRelation).Scan(&edges); err != nil {
		t.Fatalf("COUNT graph_edges: %v", err)
	}
	if edges != 1 {
		t.Errorf("arestas = %d depois de 2 passes; want 1 (a ponte é idempotente)", edges)
	}
	var nodes int
	if err := database.QueryRow(`SELECT COUNT(*) FROM graph_nodes WHERE type = ?`, CodeSymbolNodeType).
		Scan(&nodes); err != nil {
		t.Fatalf("COUNT graph_nodes: %v", err)
	}
	if nodes != 1 {
		t.Errorf("nós code_symbol = %d depois de 2 passes; want 1", nodes)
	}
	_ = second
}

// TestDocumentCodeSymbolEdgeInferido cobre o match FRACO: quando o documento
// só menciona o NOME do símbolo (sem o qualified name), a aresta é criada
// mesmo assim — mas com status epistêmico INFERRED (ADR-011).
func TestDocumentCodeSymbolEdgeInferido(t *testing.T) {
	ctx := context.Background()
	database := bridgeTestFixture(t)
	defer database.Close()

	const docID = "doc-bridge-2"
	if err := InsertDocumentWithMeta(ctx, database, docID, "docs/inferido.md", "Inferido",
		1700000000, "doc-hash-2", "", "", "resource", 0, 0); err != nil {
		t.Fatalf("InsertDocumentWithMeta: %v", err)
	}
	if err := InsertChunk(ctx, database, "chunk-bridge-2", docID,
		"A nota menciona a função Hello em geral, sem o qualified name.", 0, nil); err != nil {
		t.Fatalf("InsertChunk: %v", err)
	}

	bridgeSeedSymbol(t, ctx, database, "pkg/foo.go", "Hello", "foo.Hello")

	if _, err := BuildCodeBridge(ctx, database); err != nil {
		t.Fatalf("BuildCodeBridge: %v", err)
	}

	var status string
	err := database.QueryRow(`
		SELECT epistemic_status FROM graph_edges
		 WHERE source_id = ? AND target_id = ? AND relation = ?`,
		docID, CodeSymbolNodeID("foo.Hello"), BridgeRelation).Scan(&status)
	if err != nil {
		t.Fatalf("aresta references_code ausente: %v", err)
	}
	if status != EpistemicInferred {
		t.Errorf("epistemic_status = %q; want %q (match só pelo nome)", status, EpistemicInferred)
	}
}

// TestBuildCodeBridgeForDocument cobre o caminho do indexador de Markdown:
// só o documento informado é reconstruído.
func TestBuildCodeBridgeForDocument(t *testing.T) {
	ctx := context.Background()
	database := bridgeTestFixture(t)
	defer database.Close()

	for _, doc := range []struct{ id, body string }{
		{"doc-a", "Fala sobre foo.Hello."},
		{"doc-b", "Fala sobre bar.Bar."},
	} {
		if err := InsertDocumentWithMeta(ctx, database, doc.id, "docs/"+doc.id+".md", doc.id,
			1700000000, "hash-"+doc.id, "", "", "resource", 0, 0); err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s): %v", doc.id, err)
		}
		if err := InsertChunk(ctx, database, "chunk-"+doc.id, doc.id, doc.body, 0, nil); err != nil {
			t.Fatalf("InsertChunk(%s): %v", doc.id, err)
		}
	}

	bridgeSeedSymbol(t, ctx, database, "pkg/foo.go", "Hello", "foo.Hello")
	bridgeSeedSymbol(t, ctx, database, "pkg/bar.go", "Bar", "bar.Bar")

	result, err := BuildCodeBridgeForDocument(ctx, database, "doc-a")
	if err != nil {
		t.Fatalf("BuildCodeBridgeForDocument: %v", err)
	}
	if result.Documents != 1 {
		t.Errorf("documents = %d; want 1 (só o documento pedido)", result.Documents)
	}

	var total int
	if err := database.QueryRow(`SELECT COUNT(*) FROM graph_edges WHERE relation = ?`, BridgeRelation).
		Scan(&total); err != nil {
		t.Fatalf("COUNT graph_edges: %v", err)
	}
	if total != 1 {
		t.Errorf("arestas references_code = %d; want 1 (só doc-a casa)", total)
	}
}

// TestCodeBridgeQueries cobre os helpers de consulta da ponte: documentos que
// referenciam um símbolo e símbolos que um documento referencia.
func TestCodeBridgeQueries(t *testing.T) {
	ctx := context.Background()
	database := bridgeTestFixture(t)
	defer database.Close()

	const docID = "doc-bridge-3"
	if err := InsertDocumentWithMeta(ctx, database, docID, "docs/consulta.md", "Consulta",
		1700000000, "doc-hash-3", "", "", "resource", 0, 0); err != nil {
		t.Fatalf("InsertDocumentWithMeta: %v", err)
	}
	if err := InsertChunk(ctx, database, "chunk-bridge-3", docID,
		"Fala sobre foo.Hello.", 0, nil); err != nil {
		t.Fatalf("InsertChunk: %v", err)
	}
	bridgeSeedSymbol(t, ctx, database, "pkg/foo.go", "Hello", "foo.Hello")

	if _, err := BuildCodeBridge(ctx, database); err != nil {
		t.Fatalf("BuildCodeBridge: %v", err)
	}

	// Documentos que referenciam o símbolo.
	docs, err := CodeBridgeDocumentsForSymbol(ctx, database, "foo.Hello")
	if err != nil {
		t.Fatalf("CodeBridgeDocumentsForSymbol: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("documentos = %d; want 1", len(docs))
	}
	if docs[0].DocumentID != docID {
		t.Errorf("document_id = %q; want %q", docs[0].DocumentID, docID)
	}
	if docs[0].Status != EpistemicExtracted {
		t.Errorf("status = %q; want %q", docs[0].Status, EpistemicExtracted)
	}

	// Símbolos que o documento referencia.
	symbols, err := CodeBridgeSymbolsForDocument(ctx, database, docID)
	if err != nil {
		t.Fatalf("CodeBridgeSymbolsForDocument: %v", err)
	}
	if len(symbols) != 1 || symbols[0] != CodeSymbolNodeID("foo.Hello") {
		t.Errorf("símbolos = %v; want [%s]", symbols, CodeSymbolNodeID("foo.Hello"))
	}

	// Resolução do nó do grafo para o símbolo de código.
	ref, err := CodeSymbolByNodeID(ctx, database, CodeSymbolNodeID("foo.Hello"))
	if err != nil {
		t.Fatalf("CodeSymbolByNodeID: %v", err)
	}
	if ref == nil {
		t.Fatal("CodeSymbolByNodeID devolveu nil")
	}
	if ref.QualifiedName != "foo.Hello" {
		t.Errorf("qualified_name = %q; want %q", ref.QualifiedName, "foo.Hello")
	}
	if ref.File != "pkg/foo.go" {
		t.Errorf("file = %q; want %q", ref.File, "pkg/foo.go")
	}
}
