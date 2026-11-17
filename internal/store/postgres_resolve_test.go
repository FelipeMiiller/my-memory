package store

import (
	"context"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

func TestPostgresResolveNodeCanonicalID(t *testing.T) {
	h := pgtest.New(t)
	ctx := context.Background()

	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	defer s.Close()

	const repoA = "repo-a"
	const repoB = "repo-b"
	now := time.Now().Unix()

	// 1. Chave exata (GPR-01, AC 1): mesma consulta devolve o mesmo nó
	if err := s.InsertDocumentWithMeta(ctx, repoA, "doc-auth", "docs/auth.md", "Autenticação", now, "h1", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo doc-auth: %v", err)
	}
	idFirst, amb1, err1 := s.ResolveNodeCanonicalID(ctx, repoA, "doc-auth")
	if err1 != nil || idFirst != "doc-auth" || amb1 != nil {
		t.Fatalf("esperava resolver 'doc-auth' sem ambiguidade, obteve id=%s, amb=%v, err=%v", idFirst, amb1, err1)
	}
	idSecond, amb2, err2 := s.ResolveNodeCanonicalID(ctx, repoA, "doc-auth")
	if err2 != nil || idSecond != idFirst || amb2 != nil {
		t.Fatalf("reprodutibilidade falhou: primeira=%s, segunda=%s", idFirst, idSecond)
	}

	// 2. Caminho relativo resolve para o documento (GPR-06, AC 6)
	if err := s.InsertDocumentWithMeta(ctx, repoA, "doc-guide", "docs/guide.md", "User Guide", now, "h2", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo doc-guide: %v", err)
	}
	idPath, ambPath, errPath := s.ResolveNodeCanonicalID(ctx, repoA, "docs/guide.md")
	if errPath != nil || idPath != "doc-guide" || ambPath != nil {
		t.Fatalf("esperava resolver 'docs/guide.md' para 'doc-guide', obteve %s (err: %v)", idPath, errPath)
	}
	idPathNoExt, ambPathNoExt, errNoExt := s.ResolveNodeCanonicalID(ctx, repoA, "docs/guide")
	if errNoExt != nil || idPathNoExt != "doc-guide" || ambPathNoExt != nil {
		t.Fatalf("esperava resolver 'docs/guide' para 'doc-guide', obteve %s (err: %v)", idPathNoExt, errNoExt)
	}

	// 3. Desempate determinístico e ambiguidade no Postgres (GPR-03, GPR-04, GPR-08, AC 3, AC 4, AC 8)
	// Três nós com nome "README" no repoA:
	// - "skills/mymemory/README.md" (documento, 3 segmentos)
	// - "README.md" (documento, 1 segmento -> deve vencer!)
	// - "README" (apenas em graph_nodes, não-documento)
	if err := s.InsertDocumentWithMeta(ctx, repoA, "skills/mymemory/README.md", "skills/mymemory/README.md", "README", now, "h3", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo skills readme: %v", err)
	}
	if err := s.InsertDocumentWithMeta(ctx, repoA, "README.md", "README.md", "README", now, "h4", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo root readme: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, 'note', $3) ON CONFLICT DO NOTHING", "README", repoA, "README"); err != nil {
		t.Fatalf("inserindo graph_nodes README: %v", err)
	}

	// Inserir nó homônimo em repoB (isolamento de repositório)
	if err := s.InsertDocumentWithMeta(ctx, repoB, "other/README.md", "other/README.md", "README", now, "hb1", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo repoB readme: %v", err)
	}

	// Resolver "README" dentro de repoA:
	resolvedID, ambReport, errAmb := s.ResolveNodeCanonicalID(ctx, repoA, "README")
	if errAmb != nil {
		t.Fatalf("resolução de 'README' no repoA falhou: %v", errAmb)
	}
	// GPR-08: o id canônico devolvido deve ser EXATAMENTE "README.md", o mesmo do SQLite!
	if resolvedID != "README.md" {
		t.Errorf("desempate esperado 'README.md' (mesmo do SQLite), obteve '%s'", resolvedID)
	}
	if ambReport == nil {
		t.Fatalf("esperava relatório de ambiguidade com candidatos, obteve nil")
	}
	// No repoA, devem ser exatamente 3 candidatos (o de repoB não pode entrar)
	if len(ambReport.Candidates) != 3 {
		t.Fatalf("esperado exatamente 3 candidatos do repoA, obteve %d", len(ambReport.Candidates))
	}

	// Verificar isolamento de repositório (candidato do repoB não pode constar)
	for _, c := range ambReport.Candidates {
		if c.ID == "other/README.md" || c.RelPath == "other/README.md" {
			t.Errorf("violação de isolamento: candidato do repoB vazou para repoA: %+v", c)
		}
	}

	// Verificar ordem dos candidatos no relatório (GPR-07, AC 7)
	if ambReport.Candidates[0].RelPath != "README" {
		t.Errorf("posição 0 esperada 'README', obteve %s", ambReport.Candidates[0].RelPath)
	}
	if ambReport.Candidates[1].RelPath != "README.md" {
		t.Errorf("posição 1 esperada 'README.md', obteve %s", ambReport.Candidates[1].RelPath)
	}
	if ambReport.Candidates[2].RelPath != "skills/mymemory/README.md" {
		t.Errorf("posição 2 esperada 'skills/mymemory/README.md', obteve %s", ambReport.Candidates[2].RelPath)
	}
}
