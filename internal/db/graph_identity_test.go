package db

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestResolveCanonical prova o contrato do resolvedor canônico no SQLite
// (ADR-066 — GNI-07 a GNI-09):
//
//  1. o path, o `title:` e o `alias:` do MESMO documento devolvem o mesmo
//     `documents.id`;
//  2. uma consulta pelo título devolve o `documents.id` — nunca o id de um
//     nó de estilo título;
//  3. sem raiz do vault, um alias devolve o erro "nó '...' não encontrado no
//     grafo".
func TestResolveCanonical(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_resolve_canonical.db")
	database, err := InitDB(dbPath)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("Pulando teste: ambiente sem FTS5")
		}
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	now := time.Now().Unix()

	// Vault de teste com uma nota que declara `title:` e `alias:`.
	vault := t.TempDir()
	notaPath := filepath.Join(vault, "nota.md")
	content := "---\ntitle: Minha Nota\naliases: [Meu Alias]\n---\n\nCorpo da nota.\n"
	if err := os.WriteFile(notaPath, []byte(content), 0644); err != nil {
		t.Fatalf("escrevendo a nota: %v", err)
	}

	if err := InsertDocumentWithMeta(ctx, database, notaPath, notaPath, "Minha Nota", now, "h1", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo o documento: %v", err)
	}

	// Nó de estilo título (o outro lado da fenda): existe no grafo, mas
	// NUNCA pode sequestrar a identidade do documento.
	if _, err := database.ExecContext(ctx,
		`INSERT INTO graph_nodes (id, type, name) VALUES (?, 'note', ?)`,
		"Minha Nota", "Minha Nota"); err != nil {
		t.Fatalf("inserindo o nó de estilo título: %v", err)
	}

	// 1. Path, `title:` e `alias:` do mesmo documento → o mesmo id.
	byPath, err := ResolveNodeCanonicalIDWithRoot(ctx, database, notaPath, vault)
	if err != nil {
		t.Fatalf("resolvendo pelo path: %v", err)
	}
	byTitle, err := ResolveNodeCanonicalIDWithRoot(ctx, database, "title:Minha Nota", vault)
	if err != nil {
		t.Fatalf("resolvendo pelo title:: %v", err)
	}
	byAlias, err := ResolveNodeCanonicalIDWithRoot(ctx, database, "alias:Meu Alias", vault)
	if err != nil {
		t.Fatalf("resolvendo pelo alias:: %v", err)
	}
	if byPath != notaPath {
		t.Errorf("id pelo path = %q; want %q", byPath, notaPath)
	}
	if byTitle != byPath {
		t.Errorf("id pelo title: = %q; want %q (o mesmo nó)", byTitle, byPath)
	}
	if byAlias != byPath {
		t.Errorf("id pelo alias: = %q; want %q (o mesmo nó)", byAlias, byPath)
	}

	// 2. A consulta pelo título devolve o `documents.id`, nunca o id do nó de
	// estilo título.
	byPlainTitle, err := ResolveNodeCanonicalIDWithRoot(ctx, database, "Minha Nota", vault)
	if err != nil {
		t.Fatalf("resolvendo pelo título: %v", err)
	}
	if byPlainTitle != notaPath {
		t.Errorf("id pelo título = %q; want %q (o documents.id, não o nó de estilo título)", byPlainTitle, notaPath)
	}

	// 3. Sem raiz do vault, o alias devolve o erro de nó não encontrado.
	_, err = ResolveNodeCanonicalIDWithRoot(ctx, database, "alias:Meu Alias", "")
	if err == nil {
		t.Fatal("sem raiz do vault, o alias devolve erro; obteve nil")
	}
	if !strings.Contains(err.Error(), "não encontrado no grafo") {
		t.Errorf("erro do alias sem raiz = %q; want %q", err.Error(), "nó '...' não encontrado no grafo")
	}
}

// TestDoctorLegacyNodes prova o contrato de higiene do grafo para os nós
// legados de identidade (ADR-066 — GNI-13, GNI-14 e GNI-16):
//
//  1. o relatório expõe a contagem de nós que não são `documents.id` nem
//     `code_symbol`;
//  2. o `--fix` remove exatamente esses nós SEM GRAU, e nenhum outro;
//  3. um nó de código (`code_symbol`) nunca é removido;
//  4. uma aresta cujo destino é um documento indexado não é dead link.
func TestDoctorLegacyNodes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_doctor_legacy.db")
	database, err := InitDB(dbPath)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("Pulando teste: ambiente sem FTS5")
		}
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	now := time.Now().Unix()

	// Documentos indexados (o nó deles tem id = documents.id).
	if err := InsertDocumentWithMeta(ctx, database, "nota.md", "nota.md", "Minha Nota", now, "h1", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo o documento: %v", err)
	}
	if err := InsertDocumentWithMeta(ctx, database, "outra.md", "outra.md", "Outra Nota", now, "h2", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo o segundo documento: %v", err)
	}

	// Nó de estilo título SEM GRAU: o resto do fenda — o `--fix` remove.
	if _, err := database.ExecContext(ctx,
		`INSERT INTO graph_nodes (id, type, name) VALUES (?, 'note', ?)`, "Minha Nota", "Minha Nota"); err != nil {
		t.Fatalf("inserindo o nó legado sem grau: %v", err)
	}

	// Nó de estilo título COM GRAU: é a representação viva de um link que
	// existe em disco mas está fora do índice — o `--fix` preserva.
	if _, err := database.ExecContext(ctx,
		`INSERT INTO graph_nodes (id, type, name) VALUES (?, 'note', ?)`, "Alvo Vivo", "Alvo Vivo"); err != nil {
		t.Fatalf("inserindo o nó legado com grau: %v", err)
	}
	if err := InsertEdgeWithProps(ctx, database, "nota.md", "Alvo Vivo", "links_to", "EXTRACTED", 1.0); err != nil {
		t.Fatalf("inserindo a aresta viva: %v", err)
	}

	// Nó de código: nunca é removido (ADR-061).
	if _, err := database.ExecContext(ctx,
		`INSERT INTO graph_nodes (id, type, name) VALUES (?, 'code_symbol', ?)`, "code:repo:pkg.Alpha", "Alpha"); err != nil {
		t.Fatalf("inserindo o nó de código: %v", err)
	}

	// Aresta entre dois documentos indexados: as duas pontas em id de
	// documento — o doctor NÃO pode acusar isso como link rompido.
	if err := InsertEdgeWithProps(ctx, database, "nota.md", "outra.md", "links_to", "EXTRACTED", 1.0); err != nil {
		t.Fatalf("inserindo a aresta entre documentos: %v", err)
	}

	// O alvo "Alvo Vivo" existe em disco (fora do índice por decisão).
	vaultRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultRoot, "Alvo Vivo.md"), []byte("# Alvo Vivo\n"), 0644); err != nil {
		t.Fatalf("escrevendo o alvo em disco: %v", err)
	}

	// 1. O relatório expõe a contagem de nós legados.
	report, err := DiagnoseHealth(ctx, database, vaultRoot)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if report.LegacyNodeCount != 2 {
		t.Errorf("nós legados = %d; want 2 (o de estilo título sem grau e o com grau)", report.LegacyNodeCount)
	}
	if report.OrphanLegacyNodeCount != 1 {
		t.Errorf("nós legados sem grau = %d; want 1", report.OrphanLegacyNodeCount)
	}

	// 4. A aresta entre documentos indexados não é dead link.
	for _, dl := range report.DeadLinks {
		if dl.TargetID == "outra.md" {
			t.Errorf("a aresta para o documento indexado %q foi accusationada como dead link: %+v", dl.TargetID, dl)
		}
	}

	// 2 e 3. O `--fix` remove só o nó legado sem grau.
	if _, err := FixHealthIssues(ctx, database, vaultRoot); err != nil {
		t.Fatalf("FixHealthIssues falhou: %v", err)
	}

	var restantes int
	if err := database.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM graph_nodes WHERE id IN ('Minha Nota', 'Alvo Vivo', 'code:repo:pkg.Alpha', 'nota.md')`).Scan(&restantes); err != nil {
		t.Fatalf("contando os nós restantes: %v", err)
	}
	if restantes != 3 {
		t.Errorf("nós que sobraram = %d; want 3 (o nó de código, o nó com grau e o nó do documento)", restantes)
	}

	var semGrau int
	if err := database.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM graph_nodes WHERE id = ?`, "Minha Nota").Scan(&semGrau); err != nil {
		t.Fatalf("contando o nó legado sem grau: %v", err)
	}
	if semGrau != 0 {
		t.Errorf("nó legado sem grau = %d; want 0 (o --fix remove)", semGrau)
	}
}

// TestNodeIdentity prova o contrato do nó canônico no SQLite (ADR-066 —
// GNI-05 e GNI-06):
//
//  1. o nó de um documento indexado tem `id` = `documents.id` e `name` =
//     `documents.title` — o id é a chave, o título é o rótulo;
//  2. um documento NÃO ocupa dois nós (não existe nó com o id = título);
//  3. o nó de destino não resolvido (o texto do wikilink) continua sendo
//     criado, com o texto como nome — a FK fica satisfeita e o `mem doctor`
//     continua enxergando o link;
//  4. o nó de código (ADR-061) é preservado.
func TestNodeIdentity(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_node_identity.db")
	database, err := InitDB(dbPath)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("Pulando teste: ambiente sem FTS5")
		}
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	now := time.Now().Unix()

	// Documento indexado.
	if err := InsertDocumentWithMeta(ctx, database, "nota.md", "nota.md", "Minha Nota", now, "h1", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo o documento: %v", err)
	}

	// Nó de código (ADR-061): tem id próprio e não é documento.
	if _, err := database.ExecContext(ctx,
		`INSERT INTO graph_nodes (id, type, name) VALUES (?, 'code_symbol', ?)`,
		"code:repo:pkg.Alpha", "Alpha"); err != nil {
		t.Fatalf("inserindo o nó de código: %v", err)
	}

	// A aresta do documento para um alvo que NÃO resolve.
	if err := InsertEdgeWithProps(ctx, database, "nota.md", "Alvo Inexistente", "links_to", "EXTRACTED", 1.0); err != nil {
		t.Fatalf("inserindo a aresta: %v", err)
	}

	// 1. O nó do documento tem id = documents.id e name = documents.title.
	var nodeType, name string
	if err := database.QueryRowContext(ctx,
		`SELECT type, name FROM graph_nodes WHERE id = ?`, "nota.md").Scan(&nodeType, &name); err != nil {
		t.Fatalf("nó do documento não encontrado: %v", err)
	}
	if name != "Minha Nota" {
		t.Errorf("name do nó = %q; want %q (o título é o rótulo do nó)", name, "Minha Nota")
	}
	if nodeType != "note" {
		t.Errorf("type do nó = %q; want %q", nodeType, "note")
	}

	// 2. O documento não ocupa dois nós.
	var porTitulo int
	if err := database.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM graph_nodes WHERE id = ?`, "Minha Nota").Scan(&porTitulo); err != nil {
		t.Fatalf("contando os nós por título: %v", err)
	}
	if porTitulo != 0 {
		t.Errorf("nós com id = título do documento = %d; want 0 (uma identidade por documento)", porTitulo)
	}

	// 3. O nó do alvo não resolvido continua existindo, com o texto como nome.
	var danglingName string
	if err := database.QueryRowContext(ctx,
		`SELECT name FROM graph_nodes WHERE id = ?`, "Alvo Inexistente").Scan(&danglingName); err != nil {
		t.Fatalf("nó do alvo não resolvido não encontrado: %v", err)
	}
	if danglingName != "Alvo Inexistente" {
		t.Errorf("name do nó não resolvido = %q; want %q", danglingName, "Alvo Inexistente")
	}

	// 4. O nó de código é preservado.
	var codeName string
	if err := database.QueryRowContext(ctx,
		`SELECT name FROM graph_nodes WHERE id = ?`, "code:repo:pkg.Alpha").Scan(&codeName); err != nil {
		t.Fatalf("nó de código não encontrado: %v", err)
	}
	if codeName != "Alpha" {
		t.Errorf("name do nó de código = %q; want %q (o nó de código é preservado)", codeName, "Alpha")
	}

	// 5. O rótulo acompanha o título: reindexar com outro título renomeia o nó.
	if err := InsertDocumentWithMeta(ctx, database, "nota.md", "nota.md", "Titulo Novo", now, "h2", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("reindexando o documento: %v", err)
	}
	if err := database.QueryRowContext(ctx,
		`SELECT name FROM graph_nodes WHERE id = ?`, "nota.md").Scan(&name); err != nil {
		t.Fatalf("nó do documento não encontrado: %v", err)
	}
	if name != "Titulo Novo" {
		t.Errorf("name do nó após a reindexação = %q; want %q", name, "Titulo Novo")
	}
}
