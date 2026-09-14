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
