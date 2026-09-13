package compiler

import (
	"os"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/parser"
)

func TestAppendSection(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Arquivo não existente com createIfMissing = false deve falhar
	_, err := AppendSection(tempDir, "nao-existe.md", "## Nova Secao", "conteudo", false)
	if err == nil {
		t.Fatalf("esperava erro ao tentar anexar em arquivo inexistente com createIfMissing=false")
	}

	// 2. Arquivo não existente com createIfMissing = true deve criar
	res1, err := AppendSection(tempDir, "novo.md", "## Histórico", "Primeira entrada registrada.", true)
	if err != nil {
		t.Fatalf("erro inesperado ao criar nota via AppendSection: %v", err)
	}
	if res1.Path != "novo.md" {
		t.Errorf("caminho esperado 'novo.md', obteve '%s'", res1.Path)
	}

	data1, _ := os.ReadFile(res1.AbsPath)
	if !strings.Contains(string(data1), "## Histórico") || !strings.Contains(string(data1), "Primeira entrada registrada.") {
		t.Errorf("conteúdo gravado incorreto: %s", string(data1))
	}

	// 3. Anexar sob cabeçalho que NÃO existe (anexa ao final)
	res2, err := AppendSection(tempDir, "novo.md", "## Decisões", "Decisão de usar PostgreSQL opcional.", true)
	if err != nil {
		t.Fatalf("erro ao anexar nova seção: %v", err)
	}

	data2, _ := os.ReadFile(res2.AbsPath)
	s2 := string(data2)
	if !strings.Contains(s2, "## Decisões") || !strings.Contains(s2, "Decisão de usar PostgreSQL opcional.") {
		t.Errorf("conteúdo deve conter nova seção no final: %s", s2)
	}

	// 4. Anexar sob cabeçalho que JÁ EXISTE (insere dentro da seção existente)
	res3, err := AppendSection(tempDir, "novo.md", "## Histórico", "Segunda entrada de log adicionada.", true)
	if err != nil {
		t.Fatalf("erro ao anexar em seção existente: %v", err)
	}

	data3, _ := os.ReadFile(res3.AbsPath)
	s3 := string(data3)

	histIdx := strings.Index(s3, "## Histórico")
	decIdx := strings.Index(s3, "## Decisões")
	segIdx := strings.Index(s3, "Segunda entrada de log adicionada.")

	if histIdx == -1 || decIdx == -1 || segIdx == -1 {
		t.Fatalf("seções esperadas não encontradas no documento: %s", s3)
	}

	if segIdx < histIdx || segIdx > decIdx {
		t.Errorf("o novo conteúdo deveria ter sido inserido entre '## Histórico' e '## Decisões': %s", s3)
	}
}

func TestCompileTopicNote(t *testing.T) {
	tempDir := t.TempDir()

	sources := []CompiledSource{
		{
			DocPath: "docs/adr/001-sqlite.md",
			Title:   "ADR-001 SQLite",
			Content: "SQLite foi adotado como motor relacional e vetorial unificado para zero-config local.",
			Score:   0.88,
		},
		{
			DocPath: "docs/adr/003-turboquant.md",
			Title:   "ADR-003 TurboQuant",
			Content: "Quantização vetorial de 4-bit com rotações ortogonais de Householder.",
			Score:   0.82,
		},
	}

	relations := []parser.EdgeConnection{
		{Target: "ADR-009", Relation: "implements"},
	}

	res, err := CompileTopicNote(tempDir, "syntheses/armazenamento.md", "Arquitetura de Armazenamento", "banco de dados e vetores", sources, []string{"storage"}, relations, false)
	if err != nil {
		t.Fatalf("erro ao compilar nota de tópico: %v", err)
	}

	if res.Path != "syntheses/armazenamento.md" {
		t.Errorf("caminho incorreto: %s", res.Path)
	}

	data, err := os.ReadFile(res.AbsPath)
	if err != nil {
		t.Fatalf("falha ao ler arquivo compilado: %v", err)
	}
	s := string(data)

	if !strings.Contains(s, "title: \"Arquitetura de Armazenamento\"") {
		t.Errorf("frontmatter deve conter título: %s", s)
	}
	if !strings.Contains(s, "type: summary") {
		t.Errorf("frontmatter deve conter type summary: %s", s)
	}
	if !strings.Contains(s, "[[ADR-001 SQLite]]") || !strings.Contains(s, "[[ADR-003 TurboQuant]]") {
		t.Errorf("documento compilado deve conter referências aos títulos das fontes: %s", s)
	}
	if !strings.Contains(s, "[[rel:derived_from:ADR-001 SQLite]]") {
		t.Errorf("documento compilado deve conter backlinks tipados derived_from: %s", s)
	}
	if !strings.Contains(s, "[[rel:implements:ADR-009]]") {
		t.Errorf("documento compilado deve conter relação implements: %s", s)
	}

	// Tópico vazio deve ser rejeitado
	_, err = CompileTopicNote(tempDir, "syntheses/invalida.md", "", "", nil, nil, nil, false)
	if err == nil {
		t.Fatalf("esperava erro para tópico vazio")
	}
}
