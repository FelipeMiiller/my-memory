package db

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveNodeCanonicalID_Contract(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_resolve_contract.db")
	database, err := InitDB(dbPath)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("Pulando teste: ambiente sem FTS5")
		}
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	now := time.Now().Unix()

	// 1. Chave exata (GPR-01, AC 1): mesma consulta devolve sempre o mesmo nó
	_ = InsertDocument(ctx, database, "doc-auth", "docs/auth.md", "Autenticação", now, "h1")
	idFirst, amb1, err1 := ResolveNodeCanonicalID(ctx, database, "doc-auth")
	if err1 != nil || idFirst != "doc-auth" || amb1 != nil {
		t.Fatalf("esperava resolver 'doc-auth' sem ambiguidade, obteve id=%s, amb=%v, err=%v", idFirst, amb1, err1)
	}
	idSecond, amb2, err2 := ResolveNodeCanonicalID(ctx, database, "doc-auth")
	if err2 != nil || idSecond != idFirst || amb2 != nil {
		t.Fatalf("reprodutibilidade falhou: primeira=%s, segunda=%s", idFirst, idSecond)
	}

	// 2. Caminho relativo resolve para o documento e não para o nome (GPR-06, AC 6)
	_ = InsertDocument(ctx, database, "doc-guide", "docs/guide.md", "User Guide", now, "h2")
	idPath, ambPath, errPath := ResolveNodeCanonicalID(ctx, database, "docs/guide.md")
	if errPath != nil || idPath != "doc-guide" || ambPath != nil {
		t.Fatalf("esperava resolver 'docs/guide.md' para 'doc-guide', obteve %s (err: %v)", idPath, errPath)
	}
	// Também com caminho sem extensão .md
	idPathNoExt, ambPathNoExt, errNoExt := ResolveNodeCanonicalID(ctx, database, "docs/guide")
	if errNoExt != nil || idPathNoExt != "doc-guide" || ambPathNoExt != nil {
		t.Fatalf("esperava resolver 'docs/guide' para 'doc-guide', obteve %s (err: %v)", idPathNoExt, errNoExt)
	}

	// 3. Desempate determinístico (GPR-03, GPR-04, AC 3, AC 4):
	// Três nós com o mesmo nome "README" inseridos em ordem inversa à esperada:
	// - nó 1: "skills/mymemory/README.md" (documento, 3 segmentos de caminho)
	// - nó 2: "README.md" (documento, 1 segmento de caminho -> deve vencer!)
	// - nó 3: "README" em graph_nodes apenas (não é documento)
	_ = InsertDocument(ctx, database, "skills/mymemory/README.md", "skills/mymemory/README.md", "README", now, "h3")
	_ = InsertDocument(ctx, database, "README.md", "README.md", "README", now, "h4")
	_, _ = database.ExecContext(ctx, "INSERT OR IGNORE INTO graph_nodes (id, type, name) VALUES (?, 'note', ?)", "README", "README")

	// Resolver "README":
	// Deve retornar "README.md" como id canônico vencedor (é documento e tem menos segmentos).
	// E deve retornar um relatório de ambiguidade com os 3 candidatos (GPR-02, AC 2).
	resolvedID, ambReport, errAmb := ResolveNodeCanonicalID(ctx, database, "README")
	if errAmb != nil {
		t.Fatalf("resolução de 'README' não deve falhar com erro, obteve: %v", errAmb)
	}
	if resolvedID != "README.md" {
		t.Errorf("desempate esperado 'README.md', obteve '%s'", resolvedID)
	}
	if ambReport == nil {
		t.Fatalf("esperava relatório de ambiguidade com múltiplos candidatos, obteve nil")
	}
	if len(ambReport.Candidates) != 3 {
		t.Fatalf("esperado 3 candidatos no relatório de ambiguidade, obteve %d", len(ambReport.Candidates))
	}

	// 4. Ordem dos candidatos no relatório de ambiguidade (GPR-07, AC 7):
	// Ordenados por caminho relativo e, no empate, por id.
	// Candidato 0: "README" (não-documento, relPath "README")
	// Candidato 1: "README.md" (documento, relPath "README.md")
	// Candidato 2: "skills/mymemory/README.md"
	if ambReport.Candidates[0].RelPath != "README" {
		t.Errorf("posição 0 esperada 'README', obteve %s", ambReport.Candidates[0].RelPath)
	}
	if ambReport.Candidates[1].RelPath != "README.md" {
		t.Errorf("posição 1 esperada 'README.md', obteve %s", ambReport.Candidates[1].RelPath)
	}
	if ambReport.Candidates[2].RelPath != "skills/mymemory/README.md" {
		t.Errorf("posição 2 esperada 'skills/mymemory/README.md', obteve %s", ambReport.Candidates[2].RelPath)
	}

	// 5. Desempate por menor ID lexicográfico quando empatam os segmentos (GPR-04, AC 4):
	// Dois documentos no mesmo diretório ("docs/a.md" e "docs/b.md") com o mesmo nome em graph_nodes
	_ = InsertDocument(ctx, database, "docs/b.md", "docs/b.md", "Same Name", now, "h5")
	_ = InsertDocument(ctx, database, "docs/a.md", "docs/a.md", "Same Name", now, "h6")

	winnerID, reportTie, errTie := ResolveNodeCanonicalID(ctx, database, "Same Name")
	if errTie != nil {
		t.Fatalf("resolução não deve falhar: %v", errTie)
	}
	// "docs/a.md" < "docs/b.md" lexicograficamente
	if winnerID != "docs/a.md" {
		t.Errorf("desempate lexicográfico esperado 'docs/a.md', obteve '%s'", winnerID)
	}
	if reportTie == nil || len(reportTie.Candidates) != 2 {
		t.Errorf("esperado 2 candidatos no desempate de mesmo nome, obteve %+v", reportTie)
	}

	// 6. Chamando novamente deve retornar o mesmo id canônico (idempotência / determinismo)
	winnerAgain, _, _ := ResolveNodeCanonicalID(ctx, database, "Same Name")
	if winnerAgain != winnerID {
		t.Errorf("resolução não foi determinística: %s != %s", winnerAgain, winnerID)
	}
}
