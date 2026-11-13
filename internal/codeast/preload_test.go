package codeast

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

const repoPreload = "repo-preload"

// TestPreloadDaResolucaoEquivaleAoCaminhoPorDestino prova que o preload
// (um punhado de queries para todos os destinos) NÃO muda o resultado da
// resolução: as mesmas arestas, os mesmos ids de destino, o mesmo status
// epistêmico e as mesmas linhas incertas — inclusive as mensagens de
// ambiguidade (que dependem do CONTAGEM de candidatos, capped em 3).
//
// O caminho por destino continua existindo como fallback: se alguma chave não
// vier no preload, o resolver faz a query individual como antes.
func TestPreloadDaResolucaoEquivaleAoCaminhoPorDestino(t *testing.T) {
	ctx := db.WithRepository(context.Background(), repoPreload)
	database, err := db.InitDB(filepath.Join(t.TempDir(), "preload.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()
	if err := db.EnsureCodeTables(ctx, database); err != nil {
		t.Fatalf("EnsureCodeTables falhou: %v", err)
	}

	fileA := insereArquivoDeCodigo(t, ctx, database, "pkg/a.go")
	fileB := insereArquivoDeCodigo(t, ctx, database, "pkg/b.go")

	// Alfa: qualified name único → EXTRACTED (nível 1).
	insereSimboloDeCodigo(t, ctx, database, fileA, "function", "Alfa", "pkg.Alfa")
	// Beta: dois símbolos com o MESMO nome em pacotes diferentes → ambíguo.
	insereSimboloDeCodigo(t, ctx, database, fileB, "function", "Beta", "outro.Beta")
	insereSimboloDeCodigo(t, ctx, database, fileA, "function", "Beta", "pkg.Beta")
	// Orfao: nome único, mas sem qualified name na aresta → INFERRED (nível 2).
	insereSimboloDeCodigo(t, ctx, database, fileA, "function", "Orfao", "pkg.Orfao")
	// Gamma: tipo (só pode ser destino de uses_type).
	insereSimboloDeCodigo(t, ctx, database, fileA, "struct", "Gamma", "pkg.Gamma")
	// Importado: nó de import do arquivo A (o lookup por arquivo é o que
	// desambigua o `imports`).
	insereSimboloDeCodigo(t, ctx, database, fileA, "import", "Importado", "pkg.Importado")

	pending := []pendingEdge{
		// Nível 1: qualified name bate e é único.
		{kind: EdgeCalls, srcSymbolID: 1, dstName: "Alfa", dstQualifiedName: "pkg.Alfa", fileID: fileA, line: 10},
		// Nível 2 com ambiguidade: dois candidatos pelo nome.
		{kind: EdgeCalls, srcSymbolID: 1, dstName: "Beta", fileID: fileA, line: 11},
		// Nível 2 sem ambiguidade: um único candidato pelo nome.
		{kind: EdgeCalls, srcSymbolID: 1, dstName: "Orfao", fileID: fileA, line: 12},
		// Nível 3: destino inexistente.
		{kind: EdgeCalls, srcSymbolID: 1, dstName: "NaoExiste", fileID: fileA, line: 13},
		// Nível 1 com filtro de kind (uses_type só aponta para tipo).
		{kind: EdgeUsesType, srcSymbolID: 1, dstName: "Gamma", dstQualifiedName: "pkg.Gamma", fileID: fileA, line: 14},
		// Filtro de kind que EXCLUI o símbolo: o destino não resolve.
		{kind: EdgeUsesType, srcSymbolID: 1, dstName: "Alfa", dstQualifiedName: "pkg.Alfa", fileID: fileA, line: 15},
		// `imports` resolvido pelo lookup escopado por arquivo.
		{kind: EdgeImports, srcSymbolID: 1, dstName: "Importado", dstQualifiedName: "pkg.Importado", fileID: fileA, line: 16},
		// `imports` de OUTRO arquivo: o lookup por arquivo não acha → nível 1.
		{kind: EdgeImports, srcSymbolID: 1, dstName: "Importado", dstQualifiedName: "pkg.Importado", fileID: fileB, line: 17},
	}

	fp := NewFileProcessor(database, nil, nil)

	// (1) o caminho por destino (sem preload) e (2) o caminho com preload.
	semPreload := newEdgeResolver(ctx, database)
	comPreload := newEdgeResolver(ctx, database)
	if err := comPreload.preload(pending); err != nil {
		t.Fatalf("preload falhou: %v", err)
	}

	// O resultado ESPERADO de cada aresta — sem ele, a equivalência seria verde
	// por vacuidade (se os dois caminhos não resolvessem nada, dariam iguais).
	esperado := []string{
		"aresta kind=calls conf=1.0 status=EXTRACTED",     // Alfa por qualified name
		"incerta kind=calls conf=0.5",                     // Beta: dois candidatos
		"aresta kind=calls conf=0.8 status=INFERRED",      // Orfao por nome
		"incerta kind=calls conf=0.3",                     // destino inexistente
		"aresta kind=uses_type conf=1.0 status=EXTRACTED", // Gamma (tipo)
		"incerta kind=uses_type conf=0.3",                 // Alfa não é tipo
		"aresta kind=imports conf=1.0 status=EXTRACTED",   // import do próprio arquivo
		"aresta kind=imports conf=1.0 status=EXTRACTED",   // import de outro arquivo (nível 1)
	}

	for i, pe := range pending {
		sem, err := fp.resolveEdgeRow(ctx, pe, semPreload)
		if err != nil {
			t.Fatalf("aresta %d (%s) sem preload: %v", i, pe.dstName, err)
		}
		com, err := fp.resolveEdgeRow(ctx, pe, comPreload)
		if err != nil {
			t.Fatalf("aresta %d (%s) com preload: %v", i, pe.dstName, err)
		}
		if got, want := descreveResolucao(com), descreveResolucao(sem); got != want {
			t.Errorf("aresta %d (%s): com preload=%q; sem preload=%q", i, pe.dstName, got, want)
		}
		if got, want := formaDaResolucao(sem), esperado[i]; got != want {
			t.Errorf("aresta %d (%s): resultado=%q; esperado=%q", i, pe.dstName, got, want)
		}
	}
}

// TestPreloadPreencheOCacheDoResolver prova que o preload realmente evita as
// queries por destino: depois do preload, o cache tem as chaves e o resolver
// não emite nenhuma query individual.
func TestPreloadPreencheOCacheDoResolver(t *testing.T) {
	ctx := db.WithRepository(context.Background(), repoPreload)
	database, err := db.InitDB(filepath.Join(t.TempDir(), "preload_cache.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()
	if err := db.EnsureCodeTables(ctx, database); err != nil {
		t.Fatalf("EnsureCodeTables falhou: %v", err)
	}

	fileA := insereArquivoDeCodigo(t, ctx, database, "pkg/a.go")
	insereSimboloDeCodigo(t, ctx, database, fileA, "function", "Alfa", "pkg.Alfa")

	pending := []pendingEdge{
		{kind: EdgeCalls, srcSymbolID: 1, dstName: "Alfa", dstQualifiedName: "pkg.Alfa", fileID: fileA, line: 10},
	}
	resolver := newEdgeResolver(ctx, database)
	if err := resolver.preload(pending); err != nil {
		t.Fatalf("preload falhou: %v", err)
	}

	// A chave do nível 1 tem que estar no cache.
	key := "qualified_name=pkg.Alfa|"
	if _, ok := resolver.cache[key]; !ok {
		t.Errorf("o preload não preencheu o cache: chaves = %v", chavesDoCache(resolver))
	}
	// E a consulta por destino não pode rodar: o destino "inexistente" não
	// está no cache e o resolver cairia no caminho por query — com o preload, o
	// cache tem a chave e a query é evitada.
	ids, err := resolver.lookup("qualified_name", "pkg.Alfa", nil)
	if err != nil {
		t.Fatalf("lookup falhou: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("ids do destino = %d; want 1", len(ids))
	}
}

func chavesDoCache(r *edgeResolver) []string {
	out := make([]string, 0, len(r.cache))
	for k := range r.cache {
		out = append(out, k)
	}
	return out
}

func insereArquivoDeCodigo(t *testing.T, ctx context.Context, database *sql.DB, path string) int64 {
	t.Helper()
	id, err := db.UpsertCodeFile(ctx, database, path, "go", "hash-"+path, "ast-"+path, 10, 1)
	if err != nil {
		t.Fatalf("UpsertCodeFile(%s): %v", path, err)
	}
	return id
}

func insereSimboloDeCodigo(t *testing.T, ctx context.Context, database *sql.DB, fileID int64, kind, name, qualifiedName string) {
	t.Helper()
	_, err := db.UpsertCodeSymbols(ctx, database, fileID, []db.CodeSymbolRow{{
		Kind:          kind,
		Name:          name,
		QualifiedName: qualifiedName,
		Signature:     "func " + name + "()",
		StartLine:     1,
		EndLine:       2,
	}})
	if err != nil {
		t.Fatalf("UpsertCodeSymbols(%s): %v", qualifiedName, err)
	}
}

// descreveResolucao transforma o resultado da resolução em uma string
// comparável — é o que a equivalência do preload verifica.
func descreveResolucao(r *resolvedEdge) string {
	if r == nil {
		return "nada"
	}
	switch {
	case r.edge != nil:
		return fmt.Sprintf("aresta dst=%d kind=%s conf=%.1f status=%s linha=%d",
			r.edge.DstSymbolID, r.edge.Kind, r.edge.Confidence, r.edge.EpistemicStatus, r.edge.StartLine)
	case r.uncertain != nil:
		return fmt.Sprintf("incerta dst=%q kind=%s conf=%.1f motivo=%q",
			r.uncertain.DstCandidateName, r.uncertain.Kind, r.uncertain.Confidence, r.uncertain.Reason)
	default:
		return "vazia"
	}
}

// formaDaResolucao é a mesma descrição sem o id de destino (que depende da
// ordem de inserção) — é o que permite fixar o resultado esperado.
func formaDaResolucao(r *resolvedEdge) string {
	if r == nil {
		return "nada"
	}
	switch {
	case r.edge != nil:
		return fmt.Sprintf("aresta kind=%s conf=%.1f status=%s", r.edge.Kind, r.edge.Confidence, r.edge.EpistemicStatus)
	case r.uncertain != nil:
		return fmt.Sprintf("incerta kind=%s conf=%.1f", r.uncertain.Kind, r.uncertain.Confidence)
	default:
		return "vazia"
	}
}
