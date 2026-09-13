package store

import (
	"math"
	"testing"
	"time"
)

func TestDefaultDecayOptions(t *testing.T) {
	opts := DefaultDecayOptions()
	if opts.Enabled {
		t.Errorf("esperava Enabled false por padrão, obteve %v", opts.Enabled)
	}
	if opts.HalfLife != 30.0 {
		t.Errorf("esperava HalfLife 30.0, obteve %v", opts.HalfLife)
	}
	if opts.Weight != 0.3 {
		t.Errorf("esperava Weight 0.3, obteve %v", opts.Weight)
	}
	if opts.RefTime != 0 {
		t.Errorf("esperava RefTime 0, obteve %v", opts.RefTime)
	}
}

func TestTimeDecay_MathematicalCorrectness(t *testing.T) {
	now := int64(1700000000)
	halfLifeDays := 30.0
	weight := 0.3
	oneDay := int64(86400)

	// Caso 1: updatedAt <= 0 (sem timestamp registrado)
	if m := CalculateTimeDecay(0, now, halfLifeDays, weight); m != 1.0 {
		t.Errorf("esperava 1.0 para updatedAt=0, obteve %f", m)
	}
	if m := CalculateTimeDecay(-10, now, halfLifeDays, weight); m != 1.0 {
		t.Errorf("esperava 1.0 para updatedAt=-10, obteve %f", m)
	}

	// Caso 2: DeltaT == 0 (documento criado exatamente no tempo de referência)
	if m := CalculateTimeDecay(now, now, halfLifeDays, weight); m != 1.0 {
		t.Errorf("esperava 1.0 para deltaT=0, obteve %f", m)
	}

	// Caso 3: DeltaT no futuro (updatedAt > refTime) deve ser tratado como deltaT = 0 -> 1.0
	if m := CalculateTimeDecay(now+1000, now, halfLifeDays, weight); m != 1.0 {
		t.Errorf("esperava 1.0 para timestamp futuro, obteve %f", m)
	}

	// Caso 4: Exatamente 1 meia-vida (30 dias)
	// Multiplier = (1 - 0.3) + 0.3 * (0.5) = 0.7 + 0.15 = 0.85
	halfLifeDocTime := now - (30 * oneDay)
	mHalf := CalculateTimeDecay(halfLifeDocTime, now, halfLifeDays, weight)
	expectedHalf := 0.85
	if math.Abs(mHalf-expectedHalf) > 1e-6 {
		t.Errorf("esperava %f para 1 meia-vida, obteve %f", expectedHalf, mHalf)
	}

	// Caso 5: Exatamente 2 meias-vidas (60 dias)
	// Multiplier = 0.7 + 0.3 * 0.25 = 0.7 + 0.075 = 0.775
	twoHalfLifeDocTime := now - (60 * oneDay)
	mTwoHalf := CalculateTimeDecay(twoHalfLifeDocTime, now, halfLifeDays, weight)
	expectedTwoHalf := 0.775
	if math.Abs(mTwoHalf-expectedTwoHalf) > 1e-6 {
		t.Errorf("esperava %f para 2 meias-vidas, obteve %f", expectedTwoHalf, mTwoHalf)
	}

	// Caso 6: Documento antiquíssimo (100 meias-vidas) deve tender assintoticamente a (1 - w) = 0.70
	ancientDocTime := now - (3000 * oneDay)
	mAncient := CalculateTimeDecay(ancientDocTime, now, halfLifeDays, weight)
	expectedFloor := 1.0 - weight // 0.70
	if math.Abs(mAncient-expectedFloor) > 1e-4 {
		t.Errorf("esperava piso assintótico %f para documento antigo, obteve %f", expectedFloor, mAncient)
	}

	// Caso 7: Clamp de pesos extremos
	if m := CalculateTimeDecay(halfLifeDocTime, now, halfLifeDays, -0.5); m != 1.0 {
		t.Errorf("peso negativo deveria ser clampado para 0 -> multiplier 1.0, obteve %f", m)
	}
	mMaxWeight := CalculateTimeDecay(halfLifeDocTime, now, halfLifeDays, 2.0)
	if math.Abs(mMaxWeight-0.5) > 1e-6 {
		t.Errorf("peso > 1.0 deveria ser clampado para 1.0 -> multiplier 0.5 na meia-vida, obteve %f", mMaxWeight)
	}
}

func TestTimeDecay_RankingInversion(t *testing.T) {
	now := time.Now().Unix()
	oneDay := int64(86400)

	// docOld: 120 dias atrás
	// docNew: 1 dia atrás
	docOld := SearchResult{
		ChunkID:    "chunk_old",
		DocumentID: "doc_old",
		Content:    "arquitetura legada de microserviços",
		UpdatedAt:  now - (120 * oneDay),
	}
	docNew := SearchResult{
		ChunkID:    "chunk_new",
		DocumentID: "doc_new",
		Content:    "arquitetura moderna com event driven",
		UpdatedAt:  now - (1 * oneDay),
	}

	// Na busca textual, docOld é rank 1 e docNew é rank 2
	// Na busca vetorial, docOld é rank 1 e docNew é rank 2
	sources := []RankedResultSource{
		{
			Name:    "fts",
			Results: []SearchResult{docOld, docNew},
		},
		{
			Name:    "vector",
			Results: []SearchResult{docOld, docNew},
		},
	}

	// 1. Sem decaimento temporal: docOld deve ser 1º lugar
	withoutDecay := FuseSearchResults(sources, 60, 10)
	if len(withoutDecay) != 2 {
		t.Fatalf("esperava 2 resultados sem decay, obteve %d", len(withoutDecay))
	}
	if withoutDecay[0].ChunkID != "chunk_old" {
		t.Errorf("sem decay esperava chunk_old em 1º, obteve %s", withoutDecay[0].ChunkID)
	}

	// 2. Com decaimento temporal acentuado (halfLife: 15 dias, peso: 0.6)
	decayOpts := DecayOptions{
		Enabled:  true,
		HalfLife: 15.0,
		Weight:   0.6,
		RefTime:  now,
	}
	withDecay := FuseSearchResultsWithDecay(sources, 60, 10, decayOpts)
	if len(withDecay) != 2 {
		t.Fatalf("esperava 2 resultados com decay, obteve %d", len(withDecay))
	}

	// chunk_new deve ter ultrapassado chunk_old por ser muito mais recente
	if withDecay[0].ChunkID != "chunk_new" {
		t.Errorf("com decay esperava chunk_new em 1º lugar por recência, obteve %s (score new: %f, score old: %f)",
			withDecay[0].ChunkID, withDecay[0].Score, withDecay[1].Score)
	}

	if withDecay[0].UpdatedAt != docNew.UpdatedAt {
		t.Errorf("esperava preservação de UpdatedAt em chunk_new, obteve %d", withDecay[0].UpdatedAt)
	}
}
