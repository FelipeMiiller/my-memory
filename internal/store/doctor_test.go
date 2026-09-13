package store

import (
	"testing"
)

func TestDoctorHealthScore_Calculation(t *testing.T) {
	// 1. Cenário Perfeito (sem problemas)
	perfect := DoctorReport{
		TotalDocuments: 10,
		TotalChunks:    30,
		TotalEdges:     20,
		TotalNodes:     10,
	}
	score := CalculateDoctorHealthScore(perfect)
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
	score = CalculateDoctorHealthScore(withDead)
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
	score = CalculateDoctorHealthScore(withIssues)
	expected := 100 - 5 - 5 - 8
	if score != expected {
		t.Fatalf("Esperava score %d, obteve %d", expected, score)
	}
}

func CalculateDoctorHealthScore(report DoctorReport) int {
	score := 100
	if report.TotalDocuments > 0 {
		deadPenalty := len(report.DeadLinks) * 5
		if deadPenalty > 40 {
			deadPenalty = 40
		}
		score -= deadPenalty

		orphanRatio := float64(len(report.OrphanNotes)) / float64(report.TotalDocuments)
		orphanPenalty := int(orphanRatio * 40.0)
		if orphanPenalty > 30 {
			orphanPenalty = 30
		}
		score -= orphanPenalty

		loopPenalty := len(report.SelfLoops) * 5
		if loopPenalty > 15 {
			loopPenalty = 15
		}
		score -= loopPenalty

		desyncPenalty := len(report.DesyncedChunks) * 2
		if desyncPenalty > 15 {
			desyncPenalty = 15
		}
		score -= desyncPenalty

		if score < 0 {
			score = 0
		}
	}
	return score
}
