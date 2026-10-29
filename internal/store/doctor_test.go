package store

import (
	"testing"
)

// TestDoctorHealthScore_Calculation fixa as penalidades da ADR-013 sobre a
// função de produção: 5 pontos por dead link (máx. 40), razão de órfãs × 40
// (máx. 30), 5 pontos por self-loop (máx. 15) e 2 pontos por chunk
// dessincronizado (máx. 15).
func TestDoctorHealthScore_Calculation(t *testing.T) {
	// 1. Cenário Perfeito (sem problemas)
	perfect := DoctorReport{
		TotalDocuments: 10,
		TotalChunks:    30,
		TotalEdges:     20,
		TotalNodes:     10,
	}
	score := CalculateHealthScore(perfect)
	if score != 100 {
		t.Fatalf("Esperava score 100 para relatório perfeito, obteve %d", score)
	}

	// 2. Cenário com Dead Links (2 dead links = -10 pts)
	withDead := DoctorReport{
		TotalDocuments: 10,
		DeadLinks: []DeadLink{
			{SourceID: "a", TargetID: "ghost1", Relation: "links_to"},
			{SourceID: "b", TargetID: "ghost2", Relation: "links_to"},
		},
	}
	score = CalculateHealthScore(withDead)
	if score != 90 {
		t.Fatalf("Esperava score 90 para 2 dead links, obteve %d", score)
	}

	// 3. Cenário com Self Loops e Órfãos
	withIssues := DoctorReport{
		TotalDocuments: 5,
		DeadLinks: []DeadLink{
			{SourceID: "a", TargetID: "ghost", Relation: "links_to"}, // -5
		},
		SelfLoops: []SelfLoop{
			{NodeID: "a", Relation: "links_to"}, // -5
		},
		OrphanNotes: []OrphanNote{
			{ID: "b", Title: "b"}, // 1/5 = 20% -> 0.20 * 40 = -8
		},
	}
	score = CalculateHealthScore(withIssues)
	expected := 100 - 5 - 5 - 8
	if score != expected {
		t.Fatalf("Esperava score %d, obteve %d", expected, score)
	}
}

// TestDoctorHealthScore_ChunksDessincronizados fixa a quarta penalidade da
// ADR-013: 2 pontos por chunk dessincronizado, no máximo 15.
func TestDoctorHealthScore_ChunksDessincronizados(t *testing.T) {
	report := DoctorReport{
		TotalDocuments: 10,
		DesyncedChunks: []DesyncedChunk{
			{ChunkID: "c1", DocumentID: "d1", Issue: "sem vetor"},
			{ChunkID: "c2", DocumentID: "d2", Issue: "sem vetor"},
		},
	}
	if score := CalculateHealthScore(report); score != 96 {
		t.Fatalf("Esperava score 96 para 2 chunks dessincronizados, obteve %d", score)
	}
}

// TestDoctorHealthScore_VazioNaoSeAplica fecha a ISSUE-019.
//
// Um vault sem documentos indexados não tem denominador para a métrica: o
// Health Score não se aplica. Antes da correção, a guarda que pulava o cálculo
// inteiro deixava o score no valor inicial (100) — um veredito de saúde
// perfeita para um vault que não existe.
func TestDoctorHealthScore_VazioNaoSeAplica(t *testing.T) {
	vazio := DoctorReport{}

	if vazio.HealthScoreApplicable() {
		t.Fatal("um relatório sem documentos indexados não pode ter score aplicável")
	}

	score := CalculateHealthScore(vazio)
	if score == 100 {
		t.Fatalf("vault vazio não pode receber a nota máxima, obteve %d", score)
	}
	if score != 0 {
		t.Fatalf("score inerte de vault vazio deveria ser 0, obteve %d", score)
	}
}

// TestDoctorHealthScore_ComDocumentoTemDenominador fecha o outro lado da
// ISSUE-019: a correção do caso "vault vazio" não pode desligar a métrica
// para os vaults que têm conteúdo.
func TestDoctorHealthScore_ComDocumentoTemDenominador(t *testing.T) {
	report := DoctorReport{TotalDocuments: 1}

	if !report.HealthScoreApplicable() {
		t.Fatal("um relatório com documento indexado tem score aplicável")
	}
	if score := CalculateHealthScore(report); score != 100 {
		t.Fatalf("vault com 1 documento e nenhuma anomalia deveria pontuar 100, obteve %d", score)
	}
}

// TestDoctorHealthScore_PenalidadesNaoSaoPuladosPorGuarda prova que a
// penalidade continua sendo aplicada quando há documento: a guarda que
// causava o defeito pulava o cálculo inteiro, e o sintoma era um score que
// não reagia às anomalias.
func TestDoctorHealthScore_PenalidadesNaoSaoPuladosPorGuarda(t *testing.T) {
	comAnomalia := DoctorReport{
		TotalDocuments: 10,
		DeadLinks: []DeadLink{
			{SourceID: "a", TargetID: "ghost", Relation: "links_to"},
		},
	}
	if score := CalculateHealthScore(comAnomalia); score != 95 {
		t.Fatalf("1 dead link deveria custar 5 pontos (95), obteve %d", score)
	}
}
