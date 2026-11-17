package graph

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestAmbiguousNode_ZeroCandidates(t *testing.T) {
	rep := NewAmbiguityReport("README", []Candidate{})
	if rep.Query != "README" {
		t.Fatalf("esperado query 'README', obteve '%s'", rep.Query)
	}
	if len(rep.Candidates) != 0 {
		t.Fatalf("esperado 0 candidatos, obteve %d", len(rep.Candidates))
	}
	errMsg := rep.Error()
	if !strings.Contains(errMsg, "identificador 'README' é ambíguo (0 candidatos encontrados)") {
		t.Errorf("mensagem de erro inesperada: %s", errMsg)
	}
}

func TestAmbiguousNode_SingleCandidate(t *testing.T) {
	cand := Candidate{
		ID:      "doc-1",
		RelPath: "docs/readme.md",
		Title:   "Main Readme",
	}
	rep := NewAmbiguityReport("readme", []Candidate{cand})
	if len(rep.Candidates) != 1 {
		t.Fatalf("esperado 1 candidato, obteve %d", len(rep.Candidates))
	}
	if rep.Candidates[0].ID != "doc-1" {
		t.Errorf("esperado ID 'doc-1', obteve '%s'", rep.Candidates[0].ID)
	}
	msg := rep.Error()
	if !strings.Contains(msg, "docs/readme.md") {
		t.Errorf("esperado conter caminho relativo docs/readme.md, obteve: %s", msg)
	}
	if !strings.Contains(msg, "Main Readme") {
		t.Errorf("esperado conter título Main Readme, obteve: %s", msg)
	}
	if !strings.Contains(msg, "title:Main Readme") {
		t.Errorf("esperado conter chave de desambiguação title:Main Readme, obteve: %s", msg)
	}
	if !strings.Contains(msg, ".md") {
		t.Errorf("esperado mencionar sufixo .md, obteve: %s", msg)
	}
}

func TestAmbiguousNode_MultipleCandidatesAndSorting(t *testing.T) {
	// 3 candidatos inseridos fora de ordem
	cands := []Candidate{
		{ID: "z-id", RelPath: "skills/mymemory/README.md", Title: "Skill Readme"},
		{ID: "b-id", RelPath: "README.md", Title: "Root Readme"},
		{ID: "a-id", RelPath: "README.md", Title: "Another Root Readme"},
	}

	rep := NewAmbiguityReport("README", cands)
	if len(rep.Candidates) != 3 {
		t.Fatalf("esperado 3 candidatos, obteve %d", len(rep.Candidates))
	}

	// Ordem esperada:
	// 1. RelPath: "README.md", ID: "a-id"
	// 2. RelPath: "README.md", ID: "b-id"
	// 3. RelPath: "skills/mymemory/README.md", ID: "z-id"
	if rep.Candidates[0].ID != "a-id" || rep.Candidates[0].RelPath != "README.md" {
		t.Errorf("posição 0 esperada 'a-id' em 'README.md', obteve %+v", rep.Candidates[0])
	}
	if rep.Candidates[1].ID != "b-id" || rep.Candidates[1].RelPath != "README.md" {
		t.Errorf("posição 1 esperada 'b-id' em 'README.md', obteve %+v", rep.Candidates[1])
	}
	if rep.Candidates[2].ID != "z-id" || rep.Candidates[2].RelPath != "skills/mymemory/README.md" {
		t.Errorf("posição 2 esperada 'z-id' em 'skills/mymemory/README.md', obteve %+v", rep.Candidates[2])
	}

	// Validar que cada candidato está listado em uma linha na mensagem
	msg := rep.Error()
	lines := strings.Split(msg, "\n")
	candLines := 0
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "-") {
			candLines++
		}
	}
	if candLines != 3 {
		t.Errorf("esperado 3 linhas de candidatos, obteve %d em:\n%s", candLines, msg)
	}

	// Validar menção às chaves de desambiguação
	if !strings.Contains(msg, "title:") {
		t.Errorf("mensagem deve nomear a chave 'title:'")
	}
	if !strings.Contains(msg, ".md") {
		t.Errorf("mensagem deve mencionar opcionalidade de '.md'")
	}
}

func TestAmbiguousNode_ErrorPredicateAndWrapping(t *testing.T) {
	rep := NewAmbiguityReport("adr", []Candidate{
		{ID: "adr-1", RelPath: "docs/adr/001.md", Title: "ADR 1"},
		{ID: "adr-2", RelPath: "docs/adr/002.md", Title: "ADR 2"},
	})

	wrappedErr := fmt.Errorf("falha ao resolver nó de origem: %w", rep)

	if !IsAmbiguousNode(wrappedErr) {
		t.Errorf("IsAmbiguousNode deveria retornar true para erro encapsulado")
	}

	extracted, ok := AsAmbiguityReport(wrappedErr)
	if !ok || extracted == nil {
		t.Fatalf("AsAmbiguityReport deveria extrair o relatório de ambiguidade")
	}
	if extracted.Query != "adr" || len(extracted.Candidates) != 2 {
		t.Errorf("relatório extraído inválido: %+v", extracted)
	}

	// Erro genérico não é erro de ambiguidade
	genericErr := errors.New("erro qualquer de banco")
	if IsAmbiguousNode(genericErr) {
		t.Errorf("IsAmbiguousNode deveria retornar false para erro genérico")
	}
	if _, ok := AsAmbiguityReport(genericErr); ok {
		t.Errorf("AsAmbiguityReport deveria retornar false para erro genérico")
	}
}

func TestAmbiguousNode_PathResultIntegration(t *testing.T) {
	rep := NewAmbiguityReport("README", []Candidate{
		{ID: "1", RelPath: "README.md", Title: "R1"},
		{ID: "2", RelPath: "pkg/README.md", Title: "R2"},
	})

	res := &PathResult{
		Found:     true,
		Source:    "1",
		Target:    "target-node",
		Ambiguity: rep,
	}

	if res.Ambiguity == nil || len(res.Ambiguity.Candidates) != 2 {
		t.Fatalf("PathResult deve carregar o AmbiguityReport intacto")
	}
}
