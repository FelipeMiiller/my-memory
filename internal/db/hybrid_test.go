package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

func TestConversion_StoreToDb_UpdatedAt(t *testing.T) {
	dbRes := []SearchResult{
		{
			ChunkID:    "c1",
			DocumentID: "d1",
			Content:    "content 1",
			Distance:   0.25,
			Score:      0.88,
			Sources:    []string{"fts:1"},
			Neighbors:  []string{"d2"},
			UpdatedAt:  1700000000,
		},
	}
	storeRes := toStoreResults(dbRes)
	if len(storeRes) != 1 {
		t.Fatalf("esperava 1 item em toStoreResults, obteve %d", len(storeRes))
	}
	if storeRes[0].UpdatedAt != 1700000000 {
		t.Errorf("esperava UpdatedAt 1700000000 em store.SearchResult, obteve %d", storeRes[0].UpdatedAt)
	}

	convertedBack := fromStoreResults(storeRes)
	if len(convertedBack) != 1 {
		t.Fatalf("esperava 1 item em fromStoreResults, obteve %d", len(convertedBack))
	}
	if convertedBack[0].UpdatedAt != 1700000000 {
		t.Errorf("esperava UpdatedAt 1700000000 em db.SearchResult, obteve %d", convertedBack[0].UpdatedAt)
	}
}

func TestSearchHybridRRFWithDecay_SQLite(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test_decay.db")

	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}
	defer database.Close()

	now := time.Now().Unix()
	daySeconds := int64(86400)

	oldTime := now - (90 * daySeconds) // 90 dias atrás
	newTime := now - (1 * daySeconds)  // ontem

	// Inserir dois documentos
	if err := InsertDocument(ctx, database, "doc_old", "notes/old.md", "Old Architecture", oldTime, "hash_old"); err != nil {
		t.Fatalf("falha ao inserir doc_old: %v", err)
	}
	if err := InsertDocument(ctx, database, "doc_new", "notes/new.md", "New Architecture", newTime, "hash_new"); err != nil {
		t.Fatalf("falha ao inserir doc_new: %v", err)
	}

	// Inserir chunks
	_, err = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES 
			('c_old', 'doc_old', 0, 'go concurrency patterns go concurrency patterns go concurrency patterns memory management'),
			('c_new', 'doc_new', 0, 'go concurrency patterns other text here')
	`)
	if err != nil {
		t.Fatalf("falha ao inserir chunks: %v", err)
	}

	// Inserir na tabela chunks_fts
	_, err = database.ExecContext(ctx, `
		INSERT INTO chunks_fts (chunk_id, content)
		VALUES 
			('c_old', 'go concurrency patterns go concurrency patterns go concurrency patterns memory management'),
			('c_new', 'go concurrency patterns other text here')
	`)
	if err != nil {
		t.Fatalf("falha ao indexar chunks no FTS: %v", err)
	}

	// 1. Testa SearchFTS isolado e verifica se UpdatedAt é populado corretamente
	ftsResults, err := SearchFTS(ctx, database, "go concurrency", 10)
	if err != nil {
		t.Fatalf("SearchFTS falhou: %v", err)
	}
	if len(ftsResults) != 2 {
		t.Fatalf("esperava 2 resultados FTS, obteve %d", len(ftsResults))
	}

	foundOld := false
	foundNew := false
	for _, r := range ftsResults {
		if r.ChunkID == "c_old" {
			foundOld = true
			if r.UpdatedAt != oldTime {
				t.Errorf("esperava UpdatedAt de c_old %d, obteve %d", oldTime, r.UpdatedAt)
			}
		}
		if r.ChunkID == "c_new" {
			foundNew = true
			if r.UpdatedAt != newTime {
				t.Errorf("esperava UpdatedAt de c_new %d, obteve %d", newTime, r.UpdatedAt)
			}
		}
	}
	if !foundOld || !foundNew {
		t.Errorf("esperava encontrar c_old e c_new no FTS, encontrou old=%v, new=%v", foundOld, foundNew)
	}

	// 2. Busca Híbrida sem decaimento: c_old deve vir em primeiro (mais menções de palavras-chave no FTS)
	resultsNoDecay, err := SearchHybridRRF(ctx, database, nil, "go concurrency", nil, 10, 60, false)
	if err != nil {
		t.Fatalf("SearchHybridRRF falhou: %v", err)
	}
	if len(resultsNoDecay) != 2 {
		t.Fatalf("esperava 2 resultados na busca híbrida sem decay, obteve %d", len(resultsNoDecay))
	}
	if resultsNoDecay[0].ChunkID != "c_old" {
		t.Errorf("sem decay, esperava c_old em 1º devido ao BM25 maior, obteve %s", resultsNoDecay[0].ChunkID)
	}

	// 3. Busca Híbrida com decaimento temporal acentuado (halfLife: 15 dias, peso: 0.7)
	decayOpts := store.DecayOptions{
		Enabled:  true,
		HalfLife: 15.0,
		Weight:   0.7,
		RefTime:  now,
	}
	resultsWithDecay, err := SearchHybridRRFWithDecay(ctx, database, nil, "go concurrency", nil, 10, 60, false, decayOpts)
	if err != nil {
		t.Fatalf("SearchHybridRRFWithDecay falhou: %v", err)
	}
	if len(resultsWithDecay) != 2 {
		t.Fatalf("esperava 2 resultados na busca híbrida com decay, obteve %d", len(resultsWithDecay))
	}

	// c_new deve ultrapassar c_old pela pontuação temporal
	if resultsWithDecay[0].ChunkID != "c_new" {
		t.Errorf("com decay, esperava c_new em 1º lugar por recência, obteve %s (score new=%f, old=%f)",
			resultsWithDecay[0].ChunkID, resultsWithDecay[0].Score, resultsWithDecay[1].Score)
	}

	// Verifica se UpdatedAt está preservado no SearchResult final
	if resultsWithDecay[0].UpdatedAt != newTime {
		t.Errorf("esperava UpdatedAt preservado %d, obteve %d", newTime, resultsWithDecay[0].UpdatedAt)
	}
}

func TestSearch_CategoryFilter(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test_cat.db")

	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}
	defer database.Close()

	now := time.Now().Unix()

	// Inserir documentos em categorias distintas usando InsertDocumentWithMeta
	if err := InsertDocumentWithMeta(ctx, database, "doc_mem", "notes/mem.md", "Regras do Agente", now, "h1", "Regras e comportamentos do agente", "memory"); err != nil {
		t.Fatalf("falha ao inserir doc_mem: %v", err)
	}
	if err := InsertDocumentWithMeta(ctx, database, "doc_skill", "skills/run.md", "Comandos de Deploy", now, "h2", "Instruções de deploy em produção", "skill"); err != nil {
		t.Fatalf("falha ao inserir doc_skill: %v", err)
	}
	if err := InsertDocumentWithMeta(ctx, database, "doc_res", "docs/arch.md", "Arquitetura do Sistema", now, "h3", "Especificação técnica da arquitetura", "resource"); err != nil {
		t.Fatalf("falha ao inserir doc_res: %v", err)
	}

	// Inserir chunks
	_, err = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES 
			('c_mem', 'doc_mem', 0, 'arquitetura de agentes e regras comportamentais'),
			('c_skill', 'doc_skill', 0, 'arquitetura de deploy automatizado e scripts'),
			('c_res', 'doc_res', 0, 'arquitetura de banco de dados relacional')
	`)
	if err != nil {
		t.Fatalf("falha ao inserir chunks: %v", err)
	}

	// Inserir na tabela chunks_fts
	_, err = database.ExecContext(ctx, `
		INSERT INTO chunks_fts (chunk_id, content)
		VALUES 
			('c_mem', 'arquitetura de agentes e regras comportamentais'),
			('c_skill', 'arquitetura de deploy automatizado e scripts'),
			('c_res', 'arquitetura de banco de dados relacional')
	`)
	if err != nil {
		t.Fatalf("falha ao indexar chunks no FTS: %v", err)
	}

	// 1. Busca FTS com filtro por categoria "memory"
	memRes, err := SearchFTSWithOptions(ctx, database, "arquitetura", 10, store.SearchOptions{Category: "memory"})
	if err != nil {
		t.Fatalf("SearchFTSWithOptions(memory) falhou: %v", err)
	}
	if len(memRes) != 1 {
		t.Fatalf("esperava 1 resultado para categoria 'memory', obteve %d", len(memRes))
	}
	if memRes[0].DocumentID != "doc_mem" || memRes[0].Category != "memory" {
		t.Errorf("resultado incorreto para categoria 'memory': %+v", memRes[0])
	}
	if memRes[0].Abstract != "Regras e comportamentos do agente" {
		t.Errorf("Abstract não preservado em FTS: '%s'", memRes[0].Abstract)
	}

	// 2. Busca FTS com filtro por categoria "skill"
	skillRes, err := SearchFTSWithOptions(ctx, database, "arquitetura", 10, store.SearchOptions{Category: "skill"})
	if err != nil {
		t.Fatalf("SearchFTSWithOptions(skill) falhou: %v", err)
	}
	if len(skillRes) != 1 || skillRes[0].DocumentID != "doc_skill" {
		t.Errorf("resultado incorreto para categoria 'skill': %+v", skillRes)
	}

	// 3. Busca Híbrida RRF com filtro por categoria "resource"
	resRes, err := SearchHybridRRFWithOptions(ctx, database, nil, "arquitetura", nil, 10, 60, false, store.DefaultDecayOptions(), store.SearchOptions{Category: "resource"})
	if err != nil {
		t.Fatalf("SearchHybridRRFWithOptions(resource) falhou: %v", err)
	}
	if len(resRes) != 1 || resRes[0].DocumentID != "doc_res" {
		t.Errorf("resultado incorreto na busca híbrida para categoria 'resource': %+v", resRes)
	}
	if resRes[0].Category != "resource" || resRes[0].Abstract != "Especificação técnica da arquitetura" {
		t.Errorf("Metadados de categoria/abstract divergentes: %+v", resRes[0])
	}
}

func TestSearch_LevelL0Projection(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test_l0.db")

	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}
	defer database.Close()

	now := time.Now().Unix()

	// Inserir documento com múltiplos chunks
	if err := InsertDocumentWithMeta(ctx, database, "doc_multi", "notes/multi.md", "Multi Chunks Note", now, "h_multi", "Micro-abstract cirúrgico L0 para economia de tokens", "resource"); err != nil {
		t.Fatalf("falha ao inserir doc_multi: %v", err)
	}
	if err := InsertDocumentWithMeta(ctx, database, "doc_single", "notes/single.md", "Single Note", now, "h_single", "Outro micro-abstract L0", "resource"); err != nil {
		t.Fatalf("falha ao inserir doc_single: %v", err)
	}

	// Inserir 3 chunks para doc_multi e 1 para doc_single
	_, err = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES 
			('c_m1', 'doc_multi', 0, 'conteudo extenso do chunk 1 sobre protocolos de comunicacao e sockets'),
			('c_m2', 'doc_multi', 1, 'conteudo extenso do chunk 2 sobre protocolos de rede e roteamento'),
			('c_m3', 'doc_multi', 2, 'conteudo extenso do chunk 3 sobre protocolos criptograficos e tls'),
			('c_s1', 'doc_single', 0, 'conteudo do chunk unico sobre protocolos de comunicacao')
	`)
	if err != nil {
		t.Fatalf("falha ao inserir chunks: %v", err)
	}

	// Indexar no FTS
	_, err = database.ExecContext(ctx, `
		INSERT INTO chunks_fts (chunk_id, content)
		VALUES 
			('c_m1', 'conteudo extenso do chunk 1 sobre protocolos de comunicacao e sockets'),
			('c_m2', 'conteudo extenso do chunk 2 sobre protocolos de rede e roteamento'),
			('c_m3', 'conteudo extenso do chunk 3 sobre protocolos criptograficos e tls'),
			('c_s1', 'conteudo do chunk unico sobre protocolos de comunicacao')
	`)
	if err != nil {
		t.Fatalf("falha ao indexar chunks no FTS: %v", err)
	}

	// Busca no nível padrão L1: deve retornar múltiplos chunks de doc_multi
	l1Res, err := SearchHybridRRFWithOptions(ctx, database, nil, "protocolos", nil, 10, 60, false, store.DefaultDecayOptions(), store.SearchOptions{Level: "l1"})
	if err != nil {
		t.Fatalf("SearchHybridRRFWithOptions(l1) falhou: %v", err)
	}
	if len(l1Res) <= 2 {
		t.Errorf("no nível L1 esperava múltiplos chunks, obteve %d", len(l1Res))
	}
	for _, r := range l1Res {
		if r.Content == "" {
			t.Errorf("no nível L1, Content não deve ser vazio")
		}
	}

	// Busca no nível L0: deve DEDUPLICAR por DocumentID e limpar Content (Zero File Reads)
	l0Res, err := SearchHybridRRFWithOptions(ctx, database, nil, "protocolos", nil, 10, 60, false, store.DefaultDecayOptions(), store.SearchOptions{Level: "l0"})
	if err != nil {
		t.Fatalf("SearchHybridRRFWithOptions(l0) falhou: %v", err)
	}

	if len(l0Res) != 2 {
		t.Fatalf("no nível L0 esperava exatamente 2 documentos deduplicados (doc_multi e doc_single), obteve %d", len(l0Res))
	}

	for _, r := range l0Res {
		if r.Content != "" {
			t.Errorf("no nível L0, Content DEVE ser vazio para economizar janela de contexto, obteve: '%s'", r.Content)
		}
		if r.Abstract == "" {
			t.Errorf("no nível L0, Abstract deve estar preenchido para orientar o agente")
		}
	}
}

