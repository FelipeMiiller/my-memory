package watcher

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// TestDecayUsesFileEditTimeNotIndexTime é o teste que discrimina a correção.
//
// Dois documentos com mtimes distantes MESES um do outro são indexados na MESMA
// execução — logo, compartilham o mesmo updated_at (o instante da indexação).
// Se o decaimento lesse updated_at, os dois receberiam multiplicadores idênticos
// e a flag --decay seria um no-op silencioso. Lendo o mtime, a nota editada
// recentemente tem de ganhar do multiplicador da nota antiga.
func TestDecayUsesFileEditTimeNotIndexTime(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.InitDB(filepath.Join(tmpDir, "decay.db"))
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "binary was compiled without cgo") {
			t.Skip("Pulando teste SQLite: ambiente sem CGO")
		}
		t.Fatalf("erro ao iniciar banco SQLite: %v", err)
	}
	defer database.Close()

	oldPath := filepath.Join(tmpDir, "antiga.md")
	newPath := filepath.Join(tmpDir, "recente.md")
	files := []struct {
		path    string
		content string
		age     time.Duration
	}{
		{oldPath, "# Nota antiga\n\nzettelumantiga marker unico do documento antigo.\n", 180 * 24 * time.Hour},
		{newPath, "# Nota recente\n\nzettelumrecente marker unico do documento recente.\n", 3 * 24 * time.Hour},
	}

	now := time.Now()
	for _, f := range files {
		if err := os.WriteFile(f.path, []byte(f.content), 0644); err != nil {
			t.Fatalf("erro ao escrever %s: %v", f.path, err)
		}
		editedAt := now.Add(-f.age).Truncate(time.Second)
		if err := os.Chtimes(f.path, editedAt, editedAt); err != nil {
			t.Fatalf("erro ao ajustar mtime de %s: %v", f.path, err)
		}
	}

	// Uma única "execução de indexação": os dois arquivos entram agora.
	ctx := context.Background()
	for _, f := range files {
		if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, f.path, false); err != nil {
			t.Fatalf("erro ao indexar %s: %v", f.path, err)
		}
	}

	var oldUpdatedAt, newUpdatedAt int64
	if err := database.QueryRow(`SELECT updated_at FROM documents WHERE id = ?`, oldPath).Scan(&oldUpdatedAt); err != nil {
		t.Fatalf("documento antigo ausente do índice: %v", err)
	}
	if err := database.QueryRow(`SELECT updated_at FROM documents WHERE id = ?`, newPath).Scan(&newUpdatedAt); err != nil {
		t.Fatalf("documento recente ausente do índice: %v", err)
	}
	// Sem isto o teste não prova nada: os dois entram no mesmo run e
	// compartilham quase o mesmo updated_at, de modo que só o mtime os
	// distingue.
	//
	// updated_at NÃO precisa ser idêntico: ele é capturado por documento, no
	// momento da gravação. Num arquivo com dois documentos, indexar o segundo
	// logo após o primeiro normalmente cai no mesmo segundo, mas sob carga —
	// CI, disco lento, suite paralela — pode cruzar a fronteira. Exigir
	// igualdade exata transformava uma máquina lenta em teste instável, sem
	// que o código sob teste tivesse mudado. O invariante que importa é a
	// ORDEM DE GRANDEZA: o spread de updated_at tem de ser ordens de grandeza
	// menor que o spread de mtime, senão o decaimento poderia estar sendo
	// movido pelo instante de indexação.

	const indexSpreadTolerance = 5

	updatedSpread := oldUpdatedAt - newUpdatedAt
	if updatedSpread < 0 {
		updatedSpread = -updatedSpread
	}
	if updatedSpread > indexSpreadTolerance {
		t.Fatalf("precondição do teste quebrada: updated_at deveria ser praticamente igual (antigo=%d recente=%d, spread=%ds > tolerância de %ds). "+
			"Se este erro aparece, o teste não prova a precondição e a diferença de decaimento observada pode ser explicada pelo instante de indexação, não pelo mtime.",
			oldUpdatedAt, newUpdatedAt, updatedSpread, indexSpreadTolerance)
	}

	// As idades do fixture são 180 e 3 dias (spread de 177), mas o invariante
	// não deve depender delas: basta que o spread de mtime seja ordens de
	// grandeza maior que o de updated_at para que qualquer diferença de
	// decaimento observada seja atribuível ao mtime.
	const minMTimeSpread = 30 * 24 * 3600

	oldMtime, newMtime := fileMtime(t, oldPath), fileMtime(t, newPath)
	mtimeSpread := oldMtime - newMtime
	if mtimeSpread < 0 {
		mtimeSpread = -mtimeSpread
	}
	if mtimeSpread < minMTimeSpread {
		t.Fatalf("precondição do teste quebrada: o spread de mtime precisa ser de ao menos %ds (atual=%ds), senão os dois documentos são equivalentes para o decaimento",
			minMTimeSpread, mtimeSpread)
	}

	oldResults, err := db.SearchFTSWithOptions(ctx, database, "zettelumantiga", 5, store.SearchOptions{Level: "l1"})
	if err != nil {
		t.Fatalf("busca FTS da nota antiga falhou: %v", err)
	}
	newResults, err := db.SearchFTSWithOptions(ctx, database, "zettelumrecente", 5, store.SearchOptions{Level: "l1"})
	if err != nil {
		t.Fatalf("busca FTS da nota recente falhou: %v", err)
	}
	if len(oldResults) != 1 || len(newResults) != 1 {
		t.Fatalf("esperava 1 resultado por busca, obteve %d e %d", len(oldResults), len(newResults))
	}
	if oldResults[0].MTime <= 0 || newResults[0].MTime <= 0 {
		t.Fatalf("MTime não chegou do banco: antigo=%d recente=%d", oldResults[0].MTime, newResults[0].MTime)
	}

	// Cada documento entra como rank 1 de uma fonte diferente: a pontuação RRF
	// base é idêntica (1/(k+1)), então qualquer diferença de score final só
	// pode vir do multiplicador de decaimento.
	const k = 60
	sources := []store.RankedResultSource{
		{Name: "fts", Results: []store.SearchResult{toStoreResult(t, oldResults[0])}},
		{Name: "vector", Results: []store.SearchResult{toStoreResult(t, newResults[0])}},
	}
	opts := store.DecayOptions{Enabled: true, HalfLife: 30, Weight: 0.3, RefTime: now.Unix()}

	fused := store.FuseSearchResultsWithOptions(sources, k, 10, opts, store.SearchOptions{Level: "l1"})
	if len(fused) != 2 {
		t.Fatalf("esperava 2 resultados fundidos, obteve %d", len(fused))
	}
	base := 1.0 / float64(k+1)
	mult := map[string]float64{}
	for _, r := range fused {
		mult[r.DocumentID] = r.Score / base
	}
	multOld, multNew := mult[oldPath], mult[newPath]

	// A asserção central: multiplicadores diferentes para arquivos editados em
	// meses diferentes, apesar do updated_at idêntico.
	if math.Abs(multOld-multNew) < 1e-9 {
		t.Fatalf("multiplicadores de decaimento iguais (%.6f) — o decaimento não está lendo o mtime do arquivo", multOld)
	}
	if multNew <= multOld {
		t.Errorf("nota editada há 3 dias (mult=%.6f) deveria pontuar acima da editada há 180 dias (mult=%.6f)", multNew, multOld)
	}

	wantOld := store.CalculateTimeDecay(oldResults[0].MTime, opts.RefTime, opts.HalfLife, opts.Weight)
	wantNew := store.CalculateTimeDecay(newResults[0].MTime, opts.RefTime, opts.HalfLife, opts.Weight)
	if math.Abs(multOld-wantOld) > 1e-9 || math.Abs(multNew-wantNew) > 1e-9 {
		t.Errorf("multiplicadores não batem com CalculateTimeDecay(mtime): antiga %.6f != %.6f, recente %.6f != %.6f",
			multOld, wantOld, multNew, wantNew)
	}

	// Controle: sem decaimento os dois ficam com exatamente a mesma pontuação,
	// provando que a diferença acima vem do mtime e não do ranqueamento.
	noDecay := store.FuseSearchResultsWithOptions(sources, k, 10, store.DecayOptions{Enabled: false}, store.SearchOptions{Level: "l1"})
	if len(noDecay) != 2 {
		t.Fatalf("esperava 2 resultados sem decaimento, obteve %d", len(noDecay))
	}
	if math.Abs(noDecay[0].Score-noDecay[1].Score) > 1e-12 {
		t.Errorf("sem decaimento as pontuações deveriam ser iguais: %.12f != %.12f", noDecay[0].Score, noDecay[1].Score)
	}
}

func toStoreResult(t *testing.T, r db.SearchResult) store.SearchResult {
	t.Helper()
	return store.SearchResult{
		ChunkID:    r.ChunkID,
		DocumentID: r.DocumentID,
		Content:    r.Content,
		UpdatedAt:  r.UpdatedAt,
		MTime:      r.MTime,
		Abstract:   r.Abstract,
		Category:   r.Category,
	}
}

// fileMtime devolve o mtime do arquivo em segundos Unix, para o teste
// validar a precondição sobre o spread real entre os dois documentos.
func fileMtime(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat de %s falhou: %v", path, err)
	}
	return info.ModTime().Unix()
}
