package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graph"
)

// identityScenario é o vault de teste da regressão ponta a ponta da
// identidade do nó (ADR-066 — GNI-10 a GNI-13).
//
// O vault tem DUAS notas ligadas por wikilink. A nota de origem também aponta
// para um alvo que NÃO existe (o link rompido que o `mem doctor` precisa
// continuar enxergando) e menciona um símbolo de código (a ponte
// `references_code`).
type identityScenario struct {
	root   string // raiz do vault no disco
	dbFile string // arquivo SQLite indexado
	docA   string // `documents.id` da nota de origem (o id de path)
	docB   string // `documents.id` da nota de destino
}

// setupIdentityScenario monta o vault de teste e o indexa.
func setupIdentityScenario(t *testing.T) identityScenario {
	t.Helper()
	ctx := context.Background()

	root := t.TempDir()
	docA := writeIdentityNote(t, root, "nota-a.md", "Nota A",
		"Aponta para [[Nota B]], para [[Alvo Inexistente]] e descreve o fluxo.Ponte.")
	docB := writeIdentityNote(t, root, "nota-b.md", "Nota B", "Nota de destino.")

	dbFile := filepath.Join(t.TempDir(), "path-identity.db")
	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("Pulando teste: ambiente sem FTS5")
		}
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()
	runIndexSQLite(ctx, database, nil, nil, &cfg, root, true, false)

	return identityScenario{root: root, dbFile: dbFile, docA: docA, docB: docB}
}

// openIdentityDB abre o banco do cenário para as asserções em SQL.
func openIdentityDB(t *testing.T, dbFile string) *sql.DB {
	t.Helper()
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	return database
}

// doctorSection devolve o trecho do relatório do doctor entre o cabeçalho
// informado e o cabeçalho da seção seguinte.
func doctorSection(t *testing.T, stdout, header, next string) string {
	t.Helper()
	start := strings.Index(stdout, header)
	if start < 0 {
		t.Fatalf("o relatório do doctor não tem a seção %q:\n%s", header, stdout)
	}
	rest := stdout[start:]
	if end := strings.Index(rest, next); end >= 0 {
		return rest[:end]
	}
	return rest
}

// deadLinkTargets devolve os ALVOS das arestas listadas como link rompido na
// seção do relatório do doctor.
//
// A linha do relatório tem o formato "  - [[origem]] -> [[alvo]] (relação: …)".
func deadLinkTargets(t *testing.T, secao string) []string {
	t.Helper()
	var targets []string
	for _, linha := range strings.Split(secao, "\n") {
		linha = strings.TrimSpace(linha)
		if !strings.HasPrefix(linha, "- [[") {
			continue
		}
		partes := strings.Split(linha, "]] -> [[")
		if len(partes) != 2 {
			continue
		}
		alvo := partes[1]
		if fim := strings.Index(alvo, "]]"); fim >= 0 {
			alvo = alvo[:fim]
		}
		targets = append(targets, strings.TrimSpace(alvo))
	}
	return targets
}

// TestPathIdentity fecha o ciclo da identidade do nó (ADR-066 — GNI-10 a
// GNI-13): um vault com notas ligadas por wikilink é indexado e a SUPERFÍCIE
// CLI fala com UMA identidade por documento.
//
//  1. o `mem path` entre as duas notas acha a rota de 1 salto;
//  2. cada documento ocupa UMA identidade de nó e a aresta gravada tem as duas
//     pontas no `documents.id`;
//  3. o `mem impact`, o `mem inspect` e o `mem path` devolvem o MESMO id
//     canônico para o mesmo documento;
//  4. a ponte `references_code` mantém `source_id` = `documents.id`;
//  5. o `mem doctor` não acusa a aresta de documento como link rompido — e
//     continua apontando o link que está mesmo rompido.
func TestPathIdentity(t *testing.T) {
	scenario := setupIdentityScenario(t)

	t.Run("o mem path acha a rota de 1 salto entre as notas ligadas por wikilink", func(t *testing.T) {
		ctx := context.Background()

		var textBuf bytes.Buffer
		err := runPathCommand(ctx, "test-repo",
			[]string{"--storage=sqlite", "--db", scenario.dbFile, "Nota A", "Nota B"}, &textBuf)
		if err != nil {
			t.Fatalf("runPathCommand falhou: %v", err)
		}
		out := textBuf.String()
		if !strings.Contains(out, "✅ Encontrado") {
			t.Errorf("o mem path não reportou a rota como encontrada:\n%s", out)
		}
		if !strings.Contains(out, "Saltos:          1") {
			t.Errorf("o mem path não reportou 1 salto:\n%s", out)
		}

		var jsonBuf bytes.Buffer
		err = runPathCommand(ctx, "test-repo",
			[]string{"--storage=sqlite", "--db", scenario.dbFile, "--json", "Nota A", "Nota B"}, &jsonBuf)
		if err != nil {
			t.Fatalf("runPathCommand (json) falhou: %v", err)
		}
		var res graph.PathResult
		if err := json.Unmarshal(jsonBuf.Bytes(), &res); err != nil {
			t.Fatalf("falha ao decodificar o JSON do mem path: %v. Raw:\n%s", err, jsonBuf.String())
		}
		if !res.Found {
			t.Errorf("mem path: found = false; want true (a rota de 1 salto existe)")
		}
		if res.Hops != 1 {
			t.Errorf("mem path: hops = %d; want 1", res.Hops)
		}
		if len(res.Nodes) != 2 || res.Nodes[0] != scenario.docA || res.Nodes[1] != scenario.docB {
			t.Errorf("mem path: nodes = %v; want [%q %q] (as duas pontas em id de documento)",
				res.Nodes, scenario.docA, scenario.docB)
		}
	})

	t.Run("cada documento ocupa uma identidade de no e a aresta liga as duas pontas em id de documento", func(t *testing.T) {
		ctx := context.Background()
		database := openIdentityDB(t, scenario.dbFile)
		defer database.Close()

		// Nenhum nó de estilo título: o documento NÃO ocupa duas identidades.
		var porTitulo int
		if err := database.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM graph_nodes WHERE id IN (?, ?)`, "Nota A", "Nota B").Scan(&porTitulo); err != nil {
			t.Fatalf("contando os nós de estilo título: %v", err)
		}
		if porTitulo != 0 {
			t.Errorf("nós com id = título do documento = %d; want 0 (uma identidade por documento)", porTitulo)
		}

		// Um nó por documento, com o id de path como chave.
		var nos int
		if err := database.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM graph_nodes WHERE id IN (?, ?)`, scenario.docA, scenario.docB).Scan(&nos); err != nil {
			t.Fatalf("contando os nós por documento: %v", err)
		}
		if nos != 2 {
			t.Errorf("nós dos dois documentos = %d; want 2 (um para cada documento)", nos)
		}

		// O título é o RÓTULO do nó, não a sua chave.
		var rotulo string
		if err := database.QueryRowContext(ctx,
			`SELECT name FROM graph_nodes WHERE id = ?`, scenario.docA).Scan(&rotulo); err != nil {
			t.Fatalf("nó da nota de origem não encontrado: %v", err)
		}
		if rotulo != "Nota A" {
			t.Errorf("rótulo do nó = %q; want %q (o título é o rótulo)", rotulo, "Nota A")
		}

		// A aresta gravada tem as DUAS pontas no `documents.id`.
		rows, err := database.QueryContext(ctx,
			`SELECT source_id, target_id FROM graph_edges WHERE relation = 'links_to' AND source_id = ?`,
			scenario.docA)
		if err != nil {
			t.Fatalf("listando as arestas da nota de origem: %v", err)
		}
		defer rows.Close()

		destinos := map[string]string{}
		for rows.Next() {
			var source, target string
			if err := rows.Scan(&source, &target); err != nil {
				t.Fatalf("lendo a aresta: %v", err)
			}
			if source != scenario.docA {
				t.Errorf("origem da aresta = %q; want %q (o documents.id de origem)", source, scenario.docA)
			}
			destinos[target] = target
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("iterando as arestas: %v", err)
		}

		// O wikilink que resolve vira o `documents.id` do destino.
		if _, ok := destinos[scenario.docB]; !ok {
			t.Errorf("nenhuma aresta aponta para %q (o documents.id do destino); destinos = %v",
				scenario.docB, destinos)
		}
		// A identidade antiga (o id de título) não sobrevive.
		if _, ok := destinos["Nota B"]; ok {
			t.Errorf("a aresta aponta para o título %q; want o documents.id do destino", "Nota B")
		}
		// O wikilink que NÃO resolve mantém o texto — é o link rompido.
		if _, ok := destinos["Alvo Inexistente"]; !ok {
			t.Errorf("o link rompido sumiu do grafo; destinos = %v", destinos)
		}
	})

	t.Run("o mem impact, o mem inspect e o mem path devolvem o mesmo id canonico", func(t *testing.T) {
		ctx := context.Background()

		var pathBuf bytes.Buffer
		err := runPathCommand(ctx, "test-repo",
			[]string{"--storage=sqlite", "--db", scenario.dbFile, "--json", "Nota A", "Nota B"}, &pathBuf)
		if err != nil {
			t.Fatalf("runPathCommand (json) falhou: %v", err)
		}
		var pathRes graph.PathResult
		if err := json.Unmarshal(pathBuf.Bytes(), &pathRes); err != nil {
			t.Fatalf("falha ao decodificar o JSON do mem path: %v. Raw:\n%s", err, pathBuf.String())
		}

		var impactBuf bytes.Buffer
		err = runImpactCommand(ctx, "test-repo",
			[]string{"--storage=sqlite", "--db", scenario.dbFile, "--json", "Nota A"}, &impactBuf)
		if err != nil {
			t.Fatalf("runImpactCommand falhou: %v", err)
		}
		var impactRes graph.ImpactResult
		if err := json.Unmarshal(impactBuf.Bytes(), &impactRes); err != nil {
			t.Fatalf("falha ao decodificar o JSON do mem impact: %v. Raw:\n%s", err, impactBuf.String())
		}

		var inspectBuf bytes.Buffer
		err = runInspectCommand(ctx, "test-repo",
			[]string{"--storage=sqlite", "--db", scenario.dbFile, "--json", "Nota A"}, &inspectBuf)
		if err != nil {
			t.Fatalf("runInspectCommand falhou: %v", err)
		}
		var inspectRes graph.TriptychView
		if err := json.Unmarshal(inspectBuf.Bytes(), &inspectRes); err != nil {
			t.Fatalf("falha ao decodificar o JSON do mem inspect: %v. Raw:\n%s", err, inspectBuf.String())
		}

		// Os três consomidores falam com o MESMO id canônico do documento.
		if impactRes.TargetNode != scenario.docA {
			t.Errorf("mem impact: target_node = %q; want %q (o documents.id)", impactRes.TargetNode, scenario.docA)
		}
		if inspectRes.Target.ID != scenario.docA {
			t.Errorf("mem inspect: target.id = %q; want %q (o documents.id)", inspectRes.Target.ID, scenario.docA)
		}
		if pathRes.Source != scenario.docA {
			t.Errorf("mem path: source = %q; want %q (o documents.id)", pathRes.Source, scenario.docA)
		}
		if pathRes.Target != scenario.docB {
			t.Errorf("mem path: target = %q; want %q (o documents.id)", pathRes.Target, scenario.docB)
		}
		if impactRes.TargetNode != pathRes.Source || inspectRes.Target.ID != pathRes.Source {
			t.Errorf("os consumidores divergiram: path=%q impact=%q inspect=%q; want o mesmo id",
				pathRes.Source, impactRes.TargetNode, inspectRes.Target.ID)
		}
	})

	t.Run("a ponte references_code mantem o source_id no documents.id", func(t *testing.T) {
		ctx := context.Background()
		database := openIdentityDB(t, scenario.dbFile)
		defer database.Close()

		// Símbolo de código indexado, citado pelo qualified name na nota A.
		fileID, err := db.UpsertCodeFile(ctx, database, "pkg/ponte.go", "go",
			"content-hash-ponte", "ast-hash-ponte", 128, 1700000000)
		if err != nil {
			t.Fatalf("UpsertCodeFile: %v", err)
		}
		if _, err := db.UpsertCodeSymbol(ctx, database, fileID, "function", "Ponte", "fluxo.Ponte",
			"func Ponte()", "", 1, 3, 0, 0); err != nil {
			t.Fatalf("UpsertCodeSymbol: %v", err)
		}

		// A ponte roda (o mesmo caminho do watcher).
		result, err := db.BuildCodeBridgeForDocument(ctx, database, scenario.docA)
		if err != nil {
			t.Fatalf("BuildCodeBridgeForDocument: %v", err)
		}
		if result.Edges == 0 {
			t.Fatalf("a ponte não gravou a aresta references_code: %+v", result)
		}

		// A origem da ponte é o `documents.id` da nota que referencia o código.
		var source, target, relation string
		err = database.QueryRowContext(ctx, `
			SELECT source_id, target_id, relation
			  FROM graph_edges
			 WHERE relation = 'references_code'
		`).Scan(&source, &target, &relation)
		if err != nil {
			t.Fatalf("aresta references_code não encontrada: %v", err)
		}
		if source != scenario.docA {
			t.Errorf("source_id da ponte = %q; want %q (o documents.id da nota)", source, scenario.docA)
		}
		if relation != "references_code" {
			t.Errorf("relation da ponte = %q; want %q", relation, "references_code")
		}
		wantTarget := db.CodeSymbolNodeID(db.RepositoryFromContext(ctx), "fluxo.Ponte")
		if target != wantTarget {
			t.Errorf("target_id da ponte = %q; want %q (o nó do símbolo de código)", target, wantTarget)
		}
	})

	t.Run("o mem doctor nao acusa a aresta de documento como link rompido", func(t *testing.T) {
		stdout, stderr, exit := runMemDoctor(t, scenario.root,
			"doctor", "--storage=sqlite", "--db", scenario.dbFile, "--repo", "test-repo", "--vault", scenario.root)
		if exit != 0 {
			t.Fatalf("exit=%d stderr=%s stdout=%s", exit, stderr, stdout)
		}

		secao := doctorSection(t, stdout, "Links Quebrados / Dead Links]", "Links Fora do Índice por Decisão]")

		// A aresta entre dois documentos indexados NÃO é link rompido.
		for _, target := range deadLinkTargets(t, secao) {
			if target == scenario.docB {
				t.Errorf("o doctor acusou a aresta de documento como link rompido:\n%s", secao)
			}
		}

		// E o link que ESTÁ rompido continua apontado — o doctor não ficou
		// cego para a detecção de link morto.
		if !slices.Contains(deadLinkTargets(t, secao), "Alvo Inexistente") {
			t.Errorf("o doctor deixou de apontar o link rompido:\n%s", secao)
		}
	})
}
