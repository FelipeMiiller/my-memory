package graph

import (
	"testing"
)

func TestTruncateContent(t *testing.T) {
	text := "Esta é uma nota de teste com conteúdo relativamente longo para validar o truncamento cirúrgico."
	truncated := TruncateContent(text, 20)
	expectedPrefix := "Esta é uma nota de t"
	if len([]rune(truncated)) < 20 {
		t.Fatalf("esperado truncamento com tamanho adequado, obtido: %s", truncated)
	}
	if !containsSubstring(truncated, expectedPrefix) {
		t.Fatalf("esperado conter prefixo '%s', obtido: %s", expectedPrefix, truncated)
	}

	// Não trunca se maxLen >= len
	notTruncated := TruncateContent(text, 500)
	if notTruncated != text {
		t.Fatalf("esperado texto integral, obtido: %s", notTruncated)
	}

	// Sem limite
	noLimit := TruncateContent(text, 0)
	if noLimit != text {
		t.Fatalf("esperado texto integral para maxLen 0, obtido: %s", noLimit)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && s[:len(sub)] == sub
}

func TestBuildTriptychView(t *testing.T) {
	target := NodeSummary{
		ID:             "note-core",
		Title:          "Core Architecture Note",
		Path:           "docs/core.md",
		Type:           "decision",
		Tags:           []string{"#arch", "#core"},
		ContentPreview: "Esta nota descreve o núcleo do sistema e todas as suas dependências fundamentais.",
	}

	edges := []WeightedEdge{
		{Source: "caller-impl", Target: "note-core", Type: "implements", Weight: 1.0},
		{Source: "caller-dep", Target: "note-core", Type: "depends_on", Weight: 1.0},
		{Source: "caller-ref", Target: "note-core", Type: "links_to", Weight: 0.5},
		{Source: "note-core", Target: "target-valid", Type: "links_to", Weight: 1.0},
		{Source: "note-core", Target: "target-dead", Type: "depends_on", Weight: 1.0},
		{Source: "other-node", Target: "someone-else", Type: "links_to", Weight: 1.0}, // não afeta o alvo
	}

	opts := InspectorOptions{
		MaxContentLength: 30,
		PageRanks: map[string]float64{
			"caller-impl":  0.25,
			"caller-dep":   0.45,
			"caller-ref":   0.10,
			"target-valid": 0.30,
			"target-dead":  0.05,
		},
		Titles: map[string]string{
			"caller-impl":  "Implementation Service",
			"caller-dep":   "Dependency Client",
			"target-valid": "Valid Target Doc",
		},
		NodeTypes: map[string]string{
			"caller-impl": "concept",
			"caller-dep":  "decision",
		},
		ExistingNodes: map[string]bool{
			"target-valid": true,
			"target-dead":  false, // dead link!
		},
		Communities: map[string]int{
			"note-core": 1,
		},
		CommunityLabels: map[int]string{
			1: "Kernel",
		},
		RiskResult: &ImpactResult{
			RiskScore: 78.5,
			RiskLevel: "ALTO",
		},
	}

	view, err := BuildTriptychView(target, edges, opts)
	if err != nil {
		t.Fatalf("erro inesperado ao montar tríptico: %v", err)
	}

	if view.Target.ID != "note-core" {
		t.Fatalf("esperado ID 'note-core', obtido '%s'", view.Target.ID)
	}
	if view.Target.RiskScore != 78.5 || view.Target.RiskLevel != "ALTO" {
		t.Fatalf("esperado risk score 78.5 [ALTO], obtido %.1f [%s]", view.Target.RiskScore, view.Target.RiskLevel)
	}
	if view.Target.CommunityID != 1 || view.Target.CommunityLabel != "Kernel" {
		t.Fatalf("esperado community 1 (Kernel), obtido %d (%s)", view.Target.CommunityID, view.Target.CommunityLabel)
	}

	// Inbound checks
	if view.TotalInbound != 3 {
		t.Fatalf("esperado 3 inbound links, obtido %d", view.TotalInbound)
	}
	if view.CriticalDependents != 2 {
		t.Fatalf("esperado 2 dependentes críticos (implements e depends_on), obtido %d", view.CriticalDependents)
	}

	// Ordenação Inbound: caller-dep e caller-impl são CRITICAL. caller-dep tem maior PageRank (0.45 vs 0.25)
	if view.Inbound[0].SourceID != "caller-dep" {
		t.Fatalf("esperado primeiro inbound como 'caller-dep' (CRITICAL com PR maior), obtido '%s'", view.Inbound[0].SourceID)
	}
	if view.Inbound[1].SourceID != "caller-impl" {
		t.Fatalf("esperado segundo inbound como 'caller-impl', obtido '%s'", view.Inbound[1].SourceID)
	}
	if view.Inbound[2].SourceID != "caller-ref" {
		t.Fatalf("esperado terceiro inbound como 'caller-ref' (MEDIUM), obtido '%s'", view.Inbound[2].SourceID)
	}

	// Outbound checks
	if view.TotalOutbound != 2 {
		t.Fatalf("esperado 2 outbound links, obtido %d", view.TotalOutbound)
	}
	// target-valid exists=true vem antes de target-dead exists=false
	if !view.Outbound[0].Exists || view.Outbound[0].TargetID != "target-valid" {
		t.Fatalf("esperado target-valid como primeiro outbound (válido), obtido '%s' (exists=%v)", view.Outbound[0].TargetID, view.Outbound[0].Exists)
	}
	if view.Outbound[1].Exists || view.Outbound[1].TargetID != "target-dead" {
		t.Fatalf("esperado target-dead como segundo outbound (dead link), obtido '%s' (exists=%v)", view.Outbound[1].TargetID, view.Outbound[1].Exists)
	}

	// Truncamento de preview verificado
	if view.Target.ContentLength <= 0 {
		t.Fatalf("esperado ContentLength > 0, obtido %d", view.Target.ContentLength)
	}
	if len([]rune(view.Target.ContentPreview)) > 50 {
		t.Fatalf("esperado preview truncado em ~30 chars mais sufixo, obtido tamanho %d", len([]rune(view.Target.ContentPreview)))
	}
}

func TestBuildTriptychViewValidation(t *testing.T) {
	_, err := BuildTriptychView(NodeSummary{ID: ""}, nil, InspectorOptions{})
	if err == nil {
		t.Fatal("esperado erro para target vazio")
	}
}
