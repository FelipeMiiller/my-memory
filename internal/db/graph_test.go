package db

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

func TestResolveNodeCanonicalID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_resolve.db")
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

	// 1. Inserir documentos de teste
	_ = InsertDocument(ctx, database, "adr-001", "docs/adr/001-login.md", "Autenticação e Sessões", now, "hash1")
	_ = InsertDocument(ctx, database, "auth-service", "services/auth.md", "Auth Service", now, "hash2")

	// 2. Inserir nós no grafo
	_, _ = database.ExecContext(ctx, "INSERT OR IGNORE INTO graph_nodes (id, type, name) VALUES (?, ?, ?)", "JWT", "concept", "JSON Web Token")

	// Teste: match exato por ID
	id1, err := ResolveNodeCanonicalID(ctx, database, "adr-001")
	if err != nil || id1 != "adr-001" {
		t.Errorf("esperava 'adr-001', obteve '%s' (err: %v)", id1, err)
	}

	// Teste: match por path
	id2, err := ResolveNodeCanonicalID(ctx, database, "docs/adr/001-login.md")
	if err != nil || id2 != "adr-001" {
		t.Errorf("esperava 'adr-001' via path, obteve '%s' (err: %v)", id2, err)
	}

	// Teste: match com formato wikilink [[...]]
	id3, err := ResolveNodeCanonicalID(ctx, database, "[[adr-001]]")
	if err != nil || id3 != "adr-001" {
		t.Errorf("esperava 'adr-001' via wikilink, obteve '%s' (err: %v)", id3, err)
	}

	// Teste: match por título
	id4, err := ResolveNodeCanonicalID(ctx, database, "Autenticação e Sessões")
	if err != nil || id4 != "adr-001" {
		t.Errorf("esperava 'adr-001' via título, obteve '%s' (err: %v)", id4, err)
	}

	// Teste: match em graph_nodes
	id5, err := ResolveNodeCanonicalID(ctx, database, "JWT")
	if err != nil || id5 != "JWT" {
		t.Errorf("esperava 'JWT' via graph_nodes, obteve '%s' (err: %v)", id5, err)
	}

	// Teste: inexistente retorna erro
	_, errNotFound := ResolveNodeCanonicalID(ctx, database, "nao-existe-12345")
	if errNotFound == nil {
		t.Errorf("esperava erro para nó inexistente, obteve nil")
	}
}

func TestCalculateImpactForTarget(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_impact.db")
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

	// Criação da topologia:
	// dep-2 (depends_on) -> dep-1 (implements) -> core-service
	_ = InsertDocument(ctx, database, "core-service", "services/core.md", "Core Service", now, "h1")
	_ = InsertDocument(ctx, database, "dep-1", "services/api.md", "API Gateway", now, "h2")
	_ = InsertDocument(ctx, database, "dep-2", "clients/web.md", "Web Client", now, "h3")
	_ = InsertDocument(ctx, database, "unrelated", "docs/unrelated.md", "Unrelated Doc", now, "h4")

	_ = InsertEdgeWithProps(ctx, database, "dep-1", "core-service", "implements", "EXTRACTED", 1.0)
	_ = InsertEdgeWithProps(ctx, database, "dep-2", "dep-1", "depends_on", "EXTRACTED", 1.0)
	_ = InsertEdgeWithProps(ctx, database, "unrelated", "unrelated", "links_to", "EXTRACTED", 0.5)

	res, err := CalculateImpactForTarget(ctx, database, "core-service", 2)
	if err != nil {
		t.Fatalf("CalculateImpactForTarget falhou: %v", err)
	}

	if res.TargetNode != "core-service" {
		t.Errorf("esperava target 'core-service', obteve '%s'", res.TargetNode)
	}
	if res.TotalImpacted != 2 {
		t.Fatalf("esperava 2 nós impactados, obteve %d", res.TotalImpacted)
	}
	if res.DirectDependents != 1 {
		t.Errorf("esperava 1 dependente direto (dep-1), obteve %d", res.DirectDependents)
	}
	if res.IndirectDependents != 1 {
		t.Errorf("esperava 1 dependente indireto (dep-2), obteve %d", res.IndirectDependents)
	}
	if res.Nodes[0].ID != "dep-1" || res.Nodes[0].Severity != graph.SeverityCritical {
		t.Errorf("esperava dep-1 com severidade CRITICAL, obteve %+v", res.Nodes[0])
	}
	if res.Nodes[1].ID != "dep-2" || res.Nodes[1].Severity != graph.SeverityHigh {
		t.Errorf("esperava dep-2 com severidade HIGH, obteve %+v", res.Nodes[1])
	}
	if res.RiskScore <= 0 || res.RiskScore > 100 {
		t.Errorf("RiskScore inválido: %f", res.RiskScore)
	}
}

func TestInspectNode(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_inspect.db")
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

	// Inserir nós e documentos
	_ = InsertDocument(ctx, database, "target-doc", "docs/target.md", "Target Architectural Spec", now, "hash-t")
	_ = InsertChunk(ctx, database, "chunk-1", "target-doc", "Este é o conteúdo principal da especificação de arquitetura.", 0, nil)

	_ = InsertDocument(ctx, database, "caller-svc", "services/caller.md", "Caller Service", now, "hash-c")
	_ = InsertDocument(ctx, database, "callee-dep", "deps/callee.md", "Callee Dependency", now, "hash-d")

	// Arestas: caller-svc -> target-doc -> callee-dep
	_ = InsertEdgeWithProps(ctx, database, "caller-svc", "target-doc", "implements", "EXTRACTED", 1.0)
	_ = InsertEdgeWithProps(ctx, database, "target-doc", "callee-dep", "depends_on", "EXTRACTED", 1.0)
	// Tag
	_ = InsertEdgeWithProps(ctx, database, "target-doc", "#architecture", "tagged_as", "EXTRACTED", 0.5)

	view, err := InspectNode(ctx, database, "Target Architectural Spec", 100)
	if err != nil {
		t.Fatalf("InspectNode falhou: %v", err)
	}

	if view.Target.ID != "target-doc" {
		t.Errorf("esperava Target.ID = 'target-doc', obteve '%s'", view.Target.ID)
	}
	if view.Target.Title != "Target Architectural Spec" {
		t.Errorf("esperava Target.Title = 'Target Architectural Spec', obteve '%s'", view.Target.Title)
	}
	if !strings.Contains(view.Target.ContentPreview, "conteúdo principal") {
		t.Errorf("esperava preview com conteúdo, obteve: %s", view.Target.ContentPreview)
	}
	if view.TotalInbound != 1 {
		t.Errorf("esperava 1 inbound link, obteve %d", view.TotalInbound)
	}
	if view.Inbound[0].SourceID != "caller-svc" || view.Inbound[0].Severity != graph.SeverityCritical {
		t.Errorf("esperava inbound caller-svc com severidade CRITICAL, obteve %+v", view.Inbound[0])
	}
	// TotalOutbound inclui callee-dep e a tag #architecture
	if view.TotalOutbound < 1 {
		t.Errorf("esperava pelo menos 1 outbound link, obteve %d", view.TotalOutbound)
	}
	var foundCallee bool
	for _, out := range view.Outbound {
		if out.TargetID == "callee-dep" {
			foundCallee = true
			if !out.Exists {
				t.Errorf("esperava callee-dep como existente")
			}
		}
	}
	if !foundCallee {
		t.Errorf("esperava encontrar callee-dep nos outbounds")
	}
}

func TestResolveNodeCanonicalID_SlashNormalization(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_slash.db")
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

	// Inserir com barra invertida (padrão Windows)
	_ = InsertDocument(ctx, database, "doc-windows", "docs\\adr\\001-login.md", "Login Spec", now, "h1")

	// 1. Busca usando barra normal (Unix)
	id1, err := ResolveNodeCanonicalID(ctx, database, "docs/adr/001-login.md")
	if err != nil || id1 != "doc-windows" {
		t.Errorf("esperava resolver 'docs/adr/001-login.md' para 'doc-windows', obteve %s (err: %v)", id1, err)
	}

	// 2. Busca usando sem extensão .md
	id2, err := ResolveNodeCanonicalID(ctx, database, "docs/adr/001-login")
	if err != nil || id2 != "doc-windows" {
		t.Errorf("esperava resolver sem .md, obteve %s (err: %v)", id2, err)
	}
}

func TestInspectNode_DeadLinks(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_dead_links.db")
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

	_ = InsertDocument(ctx, database, "source-doc", "docs/source.md", "Source Doc", now, "hs")
	// Aresta para nó inexistente
	_ = InsertEdgeWithProps(ctx, database, "source-doc", "ghost-target", "links_to", "EXTRACTED", 1.0)

	view, err := InspectNode(ctx, database, "source-doc", 200)
	if err != nil {
		t.Fatalf("InspectNode falhou: %v", err)
	}

	if view.TotalOutbound != 1 {
		t.Fatalf("esperado 1 outbound link, obteve %d", view.TotalOutbound)
	}
	if view.Outbound[0].Exists {
		t.Errorf("esperado Outbound[0].Exists = false para ghost-target")
	}
}

func TestFindPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_find_path.db")
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

	// Topologia:
	// A -> B (implements, EXTRACTED, 1.0)
	// B -> C (depends_on, EXTRACTED, 1.0)
	// A -> C (tag, TAG, 0.3)
	_ = InsertDocument(ctx, database, "node-a", "docs/a.md", "Node A", now, "h1")
	_ = InsertDocument(ctx, database, "node-b", "docs/b.md", "Node B", now, "h2")
	_ = InsertDocument(ctx, database, "node-c", "docs/c.md", "Node C", now, "h3")

	_ = InsertEdgeWithProps(ctx, database, "node-a", "node-b", "implements", "EXTRACTED", 1.0)
	_ = InsertEdgeWithProps(ctx, database, "node-b", "node-c", "depends_on", "EXTRACTED", 1.0)
	_ = InsertEdgeWithProps(ctx, database, "node-a", "node-c", "tag", "TAG", 0.3)

	// 1. Busca no modo epistêmico (deve preferir A -> B -> C por ser EXTRACTED)
	res, err := FindPath(ctx, database, "a.md", "c.md", graph.DefaultPathOptions())
	if err != nil {
		t.Fatalf("FindPath falhou: %v", err)
	}
	if !res.Found {
		t.Fatalf("caminho deveria ter sido encontrado")
	}
	if res.Hops != 2 {
		t.Errorf("esperado 2 saltos no modo epistêmico, obteve %d", res.Hops)
	}
	if len(res.Nodes) != 3 || res.Nodes[1] != "node-b" {
		t.Errorf("caminho esperado [node-a, node-b, node-c], obteve %v", res.Nodes)
	}

	// 2. Busca no modo hops (deve preferir salto direto A -> C)
	optsHops := graph.PathOptions{MaxDepth: 6, Directed: true, CostMode: graph.CostModeHops}
	resHops, err := FindPath(ctx, database, "node-a", "node-c", optsHops)
	if err != nil {
		t.Fatalf("FindPath hops falhou: %v", err)
	}
	if !resHops.Found {
		t.Fatalf("caminho hops deveria ter sido encontrado")
	}
	if resHops.Hops != 1 {
		t.Errorf("esperado 1 salto no modo hops, obteve %d", resHops.Hops)
	}
}

func TestPackContext(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_pack_context.db")
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

	_ = InsertDocumentWithMeta(ctx, database, "doc-auth", "docs/auth.md", "Autenticação", now, "h1", "Resumo do auth", "resource")
	_ = InsertDocumentWithMeta(ctx, database, "doc-jwt", "docs/jwt.md", "Tokens JWT", now, "h2", "Resumo do JWT", "resource")
	_ = InsertDocumentWithMeta(ctx, database, "doc-db", "docs/db.md", "Banco de Dados", now, "h3", "Resumo do DB", "resource")

	// Chunks para auth e jwt
	_ = InsertChunk(ctx, database, "c1", "doc-auth", "Documentação do serviço de autenticação com detalhes.", 0, nil)
	_ = InsertChunk(ctx, database, "c2", "doc-jwt", "Especificação de segurança e geração de tokens JWT.", 0, nil)

	_ = InsertEdgeWithProps(ctx, database, "doc-auth", "doc-jwt", "implements", "EXTRACTED", 1.0)
	_ = InsertEdgeWithProps(ctx, database, "doc-jwt", "doc-db", "depends_on", "EXTRACTED", 0.9)

	opts := graph.PackOptions{
		MaxDepth:               2,
		MaxTokens:              1500,
		Direction:              "both",
		IncludeFringeAbstracts: true,
	}

	res, err := PackContext(ctx, database, "docs/auth.md", opts)
	if err != nil {
		t.Fatalf("PackContext falhou: %v", err)
	}

	if res.RootID != "doc-auth" {
		t.Errorf("RootID esperado 'doc-auth', obteve '%s'", res.RootID)
	}
	if res.CoreCount < 2 {
		t.Errorf("CoreCount esperado >= 2, obteve %d", res.CoreCount)
	}
	if !strings.Contains(res.Markdown, "Pacote de Contexto: Autenticação") {
		t.Errorf("Markdown não contém título da nota raiz")
	}
}
