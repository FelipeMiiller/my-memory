package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/vaultfs"
)

// TestHealthScoreCategoriasDeLink prova a matemática do score diante das
// três categorias de link: só o link rompido e o não verificável penalizam,
// e o link fora do índice por decisão não penaliza.
func TestHealthScoreCategoriasDeLink(t *testing.T) {
	casos := []struct {
		nome     string
		report   DoctorReport
		esperado int
	}{
		{
			nome:     "sem anomalia de link",
			report:   DoctorReport{TotalDocuments: 10},
			esperado: 100,
		},
		{
			nome: "link fora do indice por decisao nao penaliza",
			report: DoctorReport{
				TotalDocuments: 10,
				OutOfIndexLinks: []OutOfIndexLink{
					{SourceID: "nota.md", TargetID: "review-standards", Relation: "links_to", ResolvedPath: "catalog-skills/skills/x/review-standards.md", Match: "basename"},
				},
			},
			esperado: 100,
		},
		{
			nome: "link nao verificavel penaliza como link morto",
			report: DoctorReport{
				TotalDocuments:    10,
				UnverifiableLinks: []DeadLink{{SourceID: "nota.md", TargetID: "x", Relation: "links_to"}},
			},
			esperado: 95,
		},
		{
			nome: "dead link real penaliza",
			report: DoctorReport{
				TotalDocuments: 10,
				DeadLinks:      []DeadLink{{SourceID: "nota.md", TargetID: "y", Relation: "links_to"}},
			},
			esperado: 95,
		},
		{
			nome: "a penalidade de link tem teto de 40",
			report: DoctorReport{
				TotalDocuments:    10,
				DeadLinks:         make([]DeadLink, 20),
				UnverifiableLinks: make([]DeadLink, 20),
			},
			esperado: 60,
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			score := CalculateHealthScore(c.report)
			if score != c.esperado {
				t.Fatalf("score esperado %d, obtido %d", c.esperado, score)
			}
		})
	}
}

// TestHealthScoreProvisionalQuandoHaLinkNaoVerificavel prova o sinalizador
// de score provisório: ele acende quando há link que não pôde ser verificado
// e não acende quando não há.
func TestHealthScoreProvisionalQuandoHaLinkNaoVerificavel(t *testing.T) {
	report := &DoctorReport{TotalDocuments: 10}

	report.ApplyLinkClassification(ClassifyLinks([]LinkCandidate{
		{SourceID: "nota.md", TargetID: "x", Relation: "links_to"},
	}, nil))

	if !report.HealthScoreProvisional {
		t.Fatal("o score deveria estar marcado como provisorio: ha link nao verificavel")
	}
	if len(report.UnverifiableLinks) != 1 {
		t.Fatalf("esperado 1 link nao verificavel, obtido %d", len(report.UnverifiableLinks))
	}
	if len(report.DeadLinks) != 0 {
		t.Fatalf("sem raiz legivel nenhum link pode ser declarado rompido: %+v", report.DeadLinks)
	}

	// Com todos os candidatos resolvidos em disco, o score deixa de ser provisório.
	resolver := newTestResolver(t)
	report.ApplyLinkClassification(ClassifyLinks([]LinkCandidate{
		{SourceID: "nota.md", TargetID: "material", Relation: "links_to"},
	}, resolver))

	if report.HealthScoreProvisional {
		t.Fatal("o score nao deveria estar provisorio: nao ha link nao verificavel")
	}
}

// TestClassifyLinksSeparaRompidoDeForaDoIndice prova a separação: o alvo que
// existe em disco vai para a categoria nova (com o caminho resolvido) e o
// alvo inexistente continua link rompido.
func TestClassifyLinksSeparaRompidoDeForaDoIndice(t *testing.T) {
	resolver := newTestResolver(t)

	class := ClassifyLinks([]LinkCandidate{
		{SourceID: "nota.md", TargetID: "material", Relation: "links_to"},
		{SourceID: "nota.md", TargetID: "conceito-que-nao-existe", Relation: "links_to"},
	}, resolver)

	if len(class.OutOfIndex) != 1 {
		t.Fatalf("esperado 1 link fora do indice, obtido %d: %+v", len(class.OutOfIndex), class.OutOfIndex)
	}
	if class.OutOfIndex[0].ResolvedPath != "material.md" {
		t.Fatalf("caminho resolvido esperado %q, obtido %q", "material.md", class.OutOfIndex[0].ResolvedPath)
	}
	if class.OutOfIndex[0].SourceID != "nota.md" || class.OutOfIndex[0].Relation != "links_to" {
		t.Fatalf("a categoria nova precisa trazer origem e relacao: %+v", class.OutOfIndex[0])
	}

	if len(class.Broken) != 1 {
		t.Fatalf("esperado 1 link rompido, obtido %d: %+v", len(class.Broken), class.Broken)
	}
	if class.Broken[0].TargetID != "conceito-que-nao-existe" {
		t.Fatalf("alvo rompido esperado %q, obtido %q", "conceito-que-nao-existe", class.Broken[0].TargetID)
	}

	if len(class.Unverifiable) != 0 {
		t.Fatalf("nao deveria haver link nao verificavel com raiz legivel: %+v", class.Unverifiable)
	}
}

// TestClassifyLinksSemRaizNaoDeclaraRompido prova que a classificação sem
// raiz legivel não declara link rompido: sem prova, a suspeita vira link não
// verificável, que mantém a penalidade e marca o score como provisório.
func TestClassifyLinksSemRaizNaoDeclaraRompido(t *testing.T) {
	class := ClassifyLinks([]LinkCandidate{
		{SourceID: "nota.md", TargetID: "x", Relation: "links_to"},
		{SourceID: "nota.md", TargetID: "y", Relation: "tagged_as"},
	}, nil)

	if len(class.Broken) != 0 {
		t.Fatalf("sem raiz legivel nenhum link pode ser declarado rompido: %+v", class.Broken)
	}
	if len(class.Unverifiable) != 2 {
		t.Fatalf("esperados 2 links nao verificaveis, obtido %d", len(class.Unverifiable))
	}
	if len(class.OutOfIndex) != 0 {
		t.Fatalf("nao deveria haver link fora do indice sem raiz legivel: %+v", class.OutOfIndex)
	}
}

// newTestResolver monta um vault de teste com um arquivo "material.md" e um
// resolver sobre ele.
func newTestResolver(t *testing.T) *vaultfs.Resolver {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "material.md"), []byte("material bruto\n"), 0644); err != nil {
		t.Fatalf("falha ao escrever o material de teste: %v", err)
	}
	resolver, err := vaultfs.NewResolver(root)
	if err != nil {
		t.Fatalf("falha ao criar o resolver de teste: %v", err)
	}
	return resolver
}
