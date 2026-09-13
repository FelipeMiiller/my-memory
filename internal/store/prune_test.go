package store

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorReport_Serialization(t *testing.T) {
	report := DoctorReport{
		TotalDocuments: 10,
		TotalChunks:    45,
		TotalEdges:     18,
		TotalNodes:     15,
		HealthScore:    85,
		DeadLinks: []DeadLink{
			{SourceID: "notes/a.md", TargetID: "NotaInexistente", Relation: "links_to"},
		},
		OrphanNotes: []OrphanNote{
			{ID: "notes/orfa.md", Title: "orfa"},
		},
		SelfLoops: []SelfLoop{
			{NodeID: "notes/loop.md", Relation: "links_to"},
		},
		DesyncedChunks: []DesyncedChunk{},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Erro ao serializar DoctorReport: %v", err)
	}

	var decoded DoctorReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Erro ao deserializar DoctorReport: %v", err)
	}

	if decoded.TotalDocuments != 10 || decoded.HealthScore != 85 {
		t.Errorf("Dados de DoctorReport incorretos: %+v", decoded)
	}
	if len(decoded.DeadLinks) != 1 || decoded.DeadLinks[0].TargetID != "NotaInexistente" {
		t.Errorf("DeadLinks incorretos: %+v", decoded.DeadLinks)
	}
	if len(decoded.OrphanNotes) != 1 || decoded.OrphanNotes[0].Title != "orfa" {
		t.Errorf("OrphanNotes incorretos: %+v", decoded.OrphanNotes)
	}
	if len(decoded.SelfLoops) != 1 || decoded.SelfLoops[0].NodeID != "notes/loop.md" {
		t.Errorf("SelfLoops incorretos: %+v", decoded.SelfLoops)
	}
}

func TestIdentifyPrunedDocuments(t *testing.T) {
	storedPaths := []string{
		"C:/repo/notes/a.md",
		"C:/repo/notes/b.md",
		"C:/repo/notes/c.md",
		"C:/repo/docs/manual.md",
	}

	activeFiles := []string{
		"C:/repo/notes/a.md",
		"C:/repo/notes/c.md",
	}

	activeMap := make(map[string]bool)
	for _, f := range activeFiles {
		activeMap[filepath.Clean(f)] = true
	}

	cleanRoot := strings.ToLower(filepath.Clean("C:/repo/notes"))

	var toPrune []string
	for _, p := range storedPaths {
		cleanP := strings.ToLower(filepath.Clean(p))
		if strings.HasPrefix(cleanP, cleanRoot) {
			if !activeMap[filepath.Clean(p)] {
				toPrune = append(toPrune, p)
			}
		}
	}

	if len(toPrune) != 1 || toPrune[0] != "C:/repo/notes/b.md" {
		t.Fatalf("Esperava podar apenas 'notes/b.md', obteve: %v", toPrune)
	}
}
