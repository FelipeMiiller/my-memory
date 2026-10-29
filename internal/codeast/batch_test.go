package codeast

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// loteSymbols monta N símbolos determinísticos (qualified name preenchido,
// como o parser Go faz).
func loteSymbols(n int) []Symbol {
	syms := make([]Symbol, 0, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("Sym%d", i)
		syms = append(syms, Symbol{
			Kind:          KindFunction,
			Name:          name,
			QualifiedName: fmt.Sprintf("pkg.%s", name),
			Signature:     fmt.Sprintf("func %s()", name),
			StartLine:     i*10 + 1,
			EndLine:       i*10 + 3,
			StartCol:      1,
			EndCol:        2,
		})
	}
	return syms
}

// loteDump devolve as linhas da tabela como strings (para comparar bancos).
func loteDump(t *testing.T, database *sql.DB, query string) []string {
	t.Helper()

	rows, err := database.Query(query)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatalf("columns: %v", err)
	}
	var out []string
	for rows.Next() {
		values := make([]any, len(cols))
		holders := make([]any, len(cols))
		for i := range values {
			holders[i] = &values[i]
		}
		if err := rows.Scan(holders...); err != nil {
			t.Fatalf("scan: %v", err)
		}
		parts := make([]string, len(cols))
		for i, v := range values {
			parts[i] = fmt.Sprintf("%v", v)
		}
		out = append(out, strings.Join(parts, "|"))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

const loteSymbolsQuery = `SELECT id, file_id, kind, name, qualified_name, signature, doc_comment,
	start_line, end_line, start_col, end_col FROM code_symbols ORDER BY id ASC`

// TestLoteSimbolosEquivaleAoCaminhoPorSimbolo prova (SYM-03) que o caminho em
// LOTE grava as MESMAS linhas e resolve os MESMOS ids que o caminho por
// símbolo — o estado final dos dois bancos é idêntico.
func TestLoteSimbolosEquivaleAoCaminhoPorSimbolo(t *testing.T) {
	ctx := context.Background()
	syms := loteSymbols(6)

	// Banco A: caminho POR SÍMBOLO (o de antes).
	hPor := newCodeastTestDB(t)
	defer hPor.close()
	por := toSQLDB(hPor)
	fileA, err := db.UpsertCodeFile(ctx, por, "pkg/a.go", "go", "hash-a", "ast-a", 100, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	idsPor := make([]int64, 0, len(syms))
	for _, s := range syms {
		id, err := db.UpsertCodeSymbol(ctx, por, fileA, string(s.Kind), s.Name, s.QualifiedName,
			s.Signature, s.DocComment, s.StartLine, s.EndLine, s.StartCol, s.EndCol)
		if err != nil {
			t.Fatalf("UpsertCodeSymbol: %v", err)
		}
		idsPor = append(idsPor, id)
	}

	// Banco B: caminho EM LOTE (o novo).
	hLote := newCodeastTestDB(t)
	defer hLote.close()
	lote := toSQLDB(hLote)
	fileB, err := db.UpsertCodeFile(ctx, lote, "pkg/a.go", "go", "hash-a", "ast-a", 100, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	cache := NewCodeFileCache(lote)
	idsLote, err := cache.PersistSymbols(ctx, fileB, syms)
	if err != nil {
		t.Fatalf("PersistSymbols: %v", err)
	}

	// Os ids são os mesmos, na mesma ordem.
	if len(idsPor) != len(idsLote) {
		t.Fatalf("ids: %d vs %d", len(idsPor), len(idsLote))
	}
	for i := range idsPor {
		if idsPor[i] != idsLote[i] {
			t.Errorf("id[%d]: por símbolo = %d, lote = %d", i, idsPor[i], idsLote[i])
		}
	}

	// As linhas são as mesmas.
	rowsPor := loteDump(t, por, loteSymbolsQuery)
	rowsLote := loteDump(t, lote, loteSymbolsQuery)
	if len(rowsPor) != len(rowsLote) {
		t.Fatalf("linhas: %d vs %d\n%v\n%v", len(rowsPor), len(rowsLote), rowsPor, rowsLote)
	}
	for i := range rowsPor {
		if rowsPor[i] != rowsLote[i] {
			t.Errorf("linha %d:\n por símbolo: %s\n lote:       %s", i, rowsPor[i], rowsLote[i])
		}
	}
}

// TestLoteSimbolosMapeiaIds prova (SYM-02) que o id devolvido na posição `i`
// é o id do símbolo `i` — é desse mapeamento que as arestas dependem.
func TestLoteSimbolosMapeiaIds(t *testing.T) {
	ctx := context.Background()
	syms := loteSymbols(6)

	h := newCodeastTestDB(t)
	defer h.close()
	database := toSQLDB(h)
	fileID, err := db.UpsertCodeFile(ctx, database, "pkg/a.go", "go", "hash-a", "ast-a", 100, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}

	ids, err := NewCodeFileCache(database).PersistSymbols(ctx, fileID, syms)
	if err != nil {
		t.Fatalf("PersistSymbols: %v", err)
	}
	if len(ids) != len(syms) {
		t.Fatalf("ids = %d; want %d", len(ids), len(syms))
	}

	for i, s := range syms {
		var name, qualifiedName string
		var startLine int
		err := database.QueryRow(`
			SELECT name, COALESCE(qualified_name, ''), start_line
			  FROM code_symbols WHERE id = ?`, ids[i]).Scan(&name, &qualifiedName, &startLine)
		if err != nil {
			t.Fatalf("símbolo %d (id %d) não está no banco: %v", i, ids[i], err)
		}
		if name != s.Name || qualifiedName != s.QualifiedName || startLine != s.StartLine {
			t.Errorf("id[%d] = %d aponta para %q/%q@%d; want %q/%q@%d",
				i, ids[i], name, qualifiedName, startLine, s.Name, s.QualifiedName, s.StartLine)
		}
	}
}

// TestLoteSimbolosIdempotente prova (SYM-04) que rodar 2× não duplica linha
// nem muda o id devolvido.
func TestLoteSimbolosIdempotente(t *testing.T) {
	ctx := context.Background()
	syms := loteSymbols(4)

	h := newCodeastTestDB(t)
	defer h.close()
	database := toSQLDB(h)
	fileID, err := db.UpsertCodeFile(ctx, database, "pkg/a.go", "go", "hash-a", "ast-a", 100, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}

	cache := NewCodeFileCache(database)
	first, err := cache.PersistSymbols(ctx, fileID, syms)
	if err != nil {
		t.Fatalf("PersistSymbols (1a vez): %v", err)
	}
	second, err := cache.PersistSymbols(ctx, fileID, syms)
	if err != nil {
		t.Fatalf("PersistSymbols (2a vez): %v", err)
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("id[%d] mudou entre as rodadas: %d → %d", i, first[i], second[i])
		}
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM code_symbols`).Scan(&count); err != nil {
		t.Fatalf("COUNT code_symbols: %v", err)
	}
	if count != len(syms) {
		t.Errorf("code_symbols = %d depois de 2 rodadas; want %d (sem duplicata)", count, len(syms))
	}
}

// TestLoteSimbolosChaveDuplicada prova (SYM-05) que dois símbolos com a MESMA
// chave de conflito viram UMA linha (o payload do último), e que os dois
// ids devolvidos são o mesmo — como no caminho por símbolo.
func TestLoteSimbolosChaveDuplicada(t *testing.T) {
	ctx := context.Background()

	h := newCodeastTestDB(t)
	defer h.close()
	database := toSQLDB(h)
	fileID, err := db.UpsertCodeFile(ctx, database, "pkg/a.go", "go", "hash-a", "ast-a", 100, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}

	syms := []Symbol{
		{Kind: KindFunction, Name: "Alpha", QualifiedName: "pkg.Alpha", StartLine: 1, EndLine: 2},
		{Kind: KindFunction, Name: "Other", QualifiedName: "pkg.Other", StartLine: 5, EndLine: 6},
		// Mesma chave do primeiro: o payload do ÚLTIMO vence.
		{Kind: KindFunction, Name: "AlphaV2", QualifiedName: "pkg.Alpha", StartLine: 1, EndLine: 9},
	}

	ids, err := NewCodeFileCache(database).PersistSymbols(ctx, fileID, syms)
	if err != nil {
		t.Fatalf("PersistSymbols: %v", err)
	}
	if len(ids) != len(syms) {
		t.Fatalf("ids = %d; want %d", len(ids), len(syms))
	}
	if ids[0] != ids[2] {
		t.Errorf("símbolos com a mesma chave receberam ids diferentes: %d e %d", ids[0], ids[2])
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM code_symbols`).Scan(&count); err != nil {
		t.Fatalf("COUNT code_symbols: %v", err)
	}
	if count != 2 {
		t.Errorf("code_symbols = %d; want 2 (a chave duplicada vira uma linha)", count)
	}

	var name string
	if err := database.QueryRow(`SELECT name FROM code_symbols WHERE id = ?`, ids[0]).Scan(&name); err != nil {
		t.Fatalf("SELECT code_symbols: %v", err)
	}
	if name != "AlphaV2" {
		t.Errorf("name = %q; want %q (o payload do último vence)", name, "AlphaV2")
	}
}

// TestLoteArestasEstadoIgualAoCaminhoPorAresta prova (EDG-02) que o LOTE de
// arestas deixa `code_edges` no mesmo estado do caminho por aresta — inclusive
// quando a mesma aresta aparece duas vezes no lote (fica a menor posição e o
// payload do último).
func TestLoteArestasEstadoIgualAoCaminhoPorAresta(t *testing.T) {
	ctx := context.Background()

	edges := []db.CodeEdgeRow{
		{SrcSymbolID: 1, DstSymbolID: 2, FileID: 1, Kind: "calls", StartLine: 30, EndLine: 30, Confidence: 1.0, EpistemicStatus: db.EpistemicExtracted},
		{SrcSymbolID: 2, DstSymbolID: 3, FileID: 1, Kind: "calls", StartLine: 40, EndLine: 40, Confidence: 1.0, EpistemicStatus: db.EpistemicExtracted},
		// Duplicata da primeira, com posição MENOR: a menor posição vence.
		{SrcSymbolID: 1, DstSymbolID: 2, FileID: 1, Kind: "calls", StartLine: 10, EndLine: 12, Confidence: 0.9, EpistemicStatus: db.EpistemicInferred},
	}

	// Banco A: caminho POR ARESTA (o de antes).
	hPor := newCodeastTestDB(t)
	defer hPor.close()
	por := toSQLDB(hPor)
	fileA, err := db.UpsertCodeFile(ctx, por, "pkg/a.go", "go", "hash-a", "ast-a", 100, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	for _, s := range loteSymbols(3) {
		if _, err := db.UpsertCodeSymbol(ctx, por, fileA, string(s.Kind), s.Name, s.QualifiedName,
			s.Signature, s.DocComment, s.StartLine, s.EndLine, s.StartCol, s.EndCol); err != nil {
			t.Fatalf("UpsertCodeSymbol: %v", err)
		}
	}
	for _, e := range edges {
		if err := db.UpsertCodeEdgeWithStatus(ctx, por, e.SrcSymbolID, e.DstSymbolID, fileA, e.Kind,
			e.StartLine, e.EndLine, e.Confidence, e.EpistemicStatus); err != nil {
			t.Fatalf("UpsertCodeEdgeWithStatus: %v", err)
		}
	}

	// Banco B: caminho EM LOTE.
	hLote := newCodeastTestDB(t)
	defer hLote.close()
	lote := toSQLDB(hLote)
	fileB, err := db.UpsertCodeFile(ctx, lote, "pkg/a.go", "go", "hash-a", "ast-a", 100, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	for _, s := range loteSymbols(3) {
		if _, err := db.UpsertCodeSymbol(ctx, lote, fileB, string(s.Kind), s.Name, s.QualifiedName,
			s.Signature, s.DocComment, s.StartLine, s.EndLine, s.StartCol, s.EndCol); err != nil {
			t.Fatalf("UpsertCodeSymbol: %v", err)
		}
	}
	if err := db.UpsertCodeEdges(ctx, lote, edges); err != nil {
		t.Fatalf("UpsertCodeEdges: %v", err)
	}

	const edgeQuery = `SELECT src_symbol_id, dst_symbol_id, kind, file_id, start_line, end_line, confidence, epistemic_status
		FROM code_edges ORDER BY src_symbol_id, dst_symbol_id, kind`

	rowsPor := loteDump(t, por, edgeQuery)
	rowsLote := loteDump(t, lote, edgeQuery)
	if len(rowsPor) != len(rowsLote) {
		t.Fatalf("arestas: %d vs %d\n%v\n%v", len(rowsPor), len(rowsLote), rowsPor, rowsLote)
	}
	for i := range rowsPor {
		if rowsPor[i] != rowsLote[i] {
			t.Errorf("aresta %d:\n por aresta: %s\n lote:       %s", i, rowsPor[i], rowsLote[i])
		}
	}
}

// TestLoteIncertasEstadoIgualAoCaminhoPorAresta prova (UNC-03) que o LOTE do
// bucket `code_edges_uncertain` deixa a tabela no mesmo estado do caminho por
// aresta — inclusive quando a mesma aresta aparece duas vezes no lote (vira
// uma linha, com o payload do ÚLTIMO, que é o que o ON CONFLICT grava).
func TestLoteIncertasEstadoIgualAoCaminhoPorAresta(t *testing.T) {
	ctx := context.Background()

	rows := []db.CodeEdgeUncertainRow{
		{SrcSymbolID: 1, FileID: 1, DstCandidateName: "pkg.Foo", Kind: "calls",
			Confidence: 0.3, Reason: "destino não encontrado", StartLine: 10},
		{SrcSymbolID: 1, FileID: 1, DstCandidateName: "pkg.Bar", Kind: "calls",
			Confidence: 0.5, Reason: "ambíguo: 2 símbolos", StartLine: 20},
		// Duplicata da primeira: o payload do ÚLTIMO vence.
		{SrcSymbolID: 1, FileID: 1, DstCandidateName: "pkg.Foo", Kind: "calls",
			Confidence: 0.8, Reason: "atualizado", StartLine: 99},
	}

	seed := func(t *testing.T, database *sql.DB) {
		t.Helper()
		fileID, err := db.UpsertCodeFile(ctx, database, "pkg/a.go", "go", "hash-a", "ast-a", 100, 1700000000)
		if err != nil {
			t.Fatalf("UpsertCodeFile: %v", err)
		}
		symbolID, err := db.UpsertCodeSymbol(ctx, database, fileID, "function", "Caller", "pkg.Caller",
			"func Caller()", "", 1, 3, 0, 0)
		if err != nil {
			t.Fatalf("UpsertCodeSymbol: %v", err)
		}
		for i := range rows {
			rows[i].SrcSymbolID = symbolID
			rows[i].FileID = fileID
		}
	}

	// Banco A: caminho POR ARESTA (o de antes).
	hPor := newCodeastTestDB(t)
	defer hPor.close()
	por := toSQLDB(hPor)
	seed(t, por)
	for _, r := range rows {
		if err := db.InsertCodeEdgeUncertain(ctx, por, r.SrcSymbolID, r.FileID, r.DstCandidateName,
			r.Kind, r.Reason, r.Confidence, r.StartLine); err != nil {
			t.Fatalf("InsertCodeEdgeUncertain: %v", err)
		}
	}

	// Banco B: caminho EM LOTE.
	hLote := newCodeastTestDB(t)
	defer hLote.close()
	lote := toSQLDB(hLote)
	seed(t, lote)
	if err := db.InsertCodeEdgeUncertains(ctx, lote, rows); err != nil {
		t.Fatalf("InsertCodeEdgeUncertains: %v", err)
	}

	const uncertainQuery = `SELECT src_symbol_id, dst_candidate_name, kind, file_id, confidence, reason, start_line
		FROM code_edges_uncertain ORDER BY src_symbol_id, dst_candidate_name, kind`

	rowsPor := loteDump(t, por, uncertainQuery)
	rowsLote := loteDump(t, lote, uncertainQuery)
	if len(rowsPor) != len(rowsLote) {
		t.Fatalf("arestas incertas: %d vs %d\n%v\n%v", len(rowsPor), len(rowsLote), rowsPor, rowsLote)
	}
	for i := range rowsPor {
		if rowsPor[i] != rowsLote[i] {
			t.Errorf("aresta incerta %d:\n por aresta: %s\n lote:       %s", i, rowsPor[i], rowsLote[i])
		}
	}
	if len(rowsLote) != 2 {
		t.Errorf("code_edges_uncertain = %d linhas; want 2 (a duplicata do lote vira uma)", len(rowsLote))
	}
}

// TestLoteArestasRecusaBaixaConfianca prova (EDG-03) que a recusa de
// confidence < 0.5 continua igual à do caminho por aresta.
func TestLoteArestasRecusaBaixaConfianca(t *testing.T) {
	ctx := context.Background()

	h := newCodeastTestDB(t)
	defer h.close()
	database := toSQLDB(h)

	err := db.UpsertCodeEdges(ctx, database, []db.CodeEdgeRow{{
		SrcSymbolID: 1, DstSymbolID: 2, FileID: 1, Kind: "calls",
		StartLine: 1, EndLine: 1, Confidence: 0.3, EpistemicStatus: db.EpistemicExtracted,
	}})
	if err == nil {
		t.Fatal("esperava erro ao gravar aresta de baixa confiança em code_edges")
	}
	if !strings.Contains(err.Error(), "UseInsertCodeEdgeUncertain") {
		t.Errorf("erro = %q; want o mesmo texto do caminho por aresta", err.Error())
	}
}
