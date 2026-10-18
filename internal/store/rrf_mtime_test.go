package store

import (
	"math"
	"testing"
)

// TestDecayFallsBackToUpdatedAtWhenMTimeIsZero fixa o contrato do fallback:
// uma linha legada (mtime = 0) precisa receber exatamente o multiplicador que
// updated_at produziria, e não 1.0 (que a trataria como documento eterno).
func TestDecayFallsBackToUpdatedAtWhenMTimeIsZero(t *testing.T) {
	const k = 60
	updatedAt := int64(1_600_000_000)
	opts := DecayOptions{Enabled: true, HalfLife: 30, Weight: 0.3, RefTime: 1_700_000_000}

	withMTimeZero := SearchResult{
		ChunkID:    "chunk-1",
		DocumentID: "doc-1",
		UpdatedAt:  updatedAt,
		MTime:      0,
	}
	sameResultWithoutMTime := SearchResult{
		ChunkID:    "chunk-2",
		DocumentID: "doc-2",
		UpdatedAt:  updatedAt,
	}

	fused := FuseSearchResultsWithOptions(
		[]RankedResultSource{
			{Name: "fts", Results: []SearchResult{withMTimeZero}},
			{Name: "vector", Results: []SearchResult{sameResultWithoutMTime}},
		},
		k, 10, opts, SearchOptions{Level: "l1"},
	)
	if len(fused) != 2 {
		t.Fatalf("esperava 2 resultados fundidos, obteve %d", len(fused))
	}

	base := 1.0 / float64(k+1)
	want := CalculateTimeDecay(updatedAt, opts.RefTime, opts.HalfLife, opts.Weight)
	if want >= 1.0 {
		t.Fatalf("precondição inválida: updated_at deveria produzir decaimento, obteve %.6f", want)
	}

	byID := map[string]float64{}
	for _, r := range fused {
		byID[r.DocumentID] = r.Score / base
	}
	gotFallback := byID["doc-1"]
	if math.Abs(gotFallback-want) > 1e-9 {
		t.Errorf("multiplicador com mtime = 0 = %.9f, esperado o de updated_at = %.9f", gotFallback, want)
	}
	if math.Abs(gotFallback-byID["doc-2"]) > 1e-12 {
		t.Errorf("mtime = 0 deveria se comportar como updated_at: %.9f != %.9f", gotFallback, byID["doc-2"])
	}

	// E o oposto: com mtime positivo, updated_at é ignorado. Dois documentos com
	// o mesmo updated_at mas mtimes distintos não podem receber o mesmo score.
	oldMTime := SearchResult{ChunkID: "c3", DocumentID: "doc-3", UpdatedAt: updatedAt, MTime: 1_600_000_000}
	newMTime := SearchResult{ChunkID: "c4", DocumentID: "doc-4", UpdatedAt: updatedAt, MTime: 1_699_000_000}
	fusedMTime := FuseSearchResultsWithOptions(
		[]RankedResultSource{
			{Name: "fts", Results: []SearchResult{oldMTime}},
			{Name: "vector", Results: []SearchResult{newMTime}},
		},
		k, 10, opts, SearchOptions{Level: "l1"},
	)
	if len(fusedMTime) != 2 {
		t.Fatalf("esperava 2 resultados fundidos, obteve %d", len(fusedMTime))
	}
	if math.Abs(fusedMTime[0].Score-fusedMTime[1].Score) < 1e-12 {
		t.Errorf("documentos com mtimes distintos receberam scores iguais (%.12f)", fusedMTime[0].Score)
	}
}
