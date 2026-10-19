package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// TestBuscaSQLitePopulaMTimeEmTodosOsCaminhos fecha a lacuna de cobertura do
// ADR-057 no SQLite. Só o caminho FTS tinha teste que perceberia se MTime
// parasse de ser populado: zerar `COALESCE(d.mtime, 0)` em qualquer um dos
// outros deixava a suíte inteira verde.
//
// São quatro leituras distintas da coluna, em quatro queries diferentes:
//
//	graph.go:51   SearchKNNWithOptions          (vetorial/sqlite-vec)
//	graph.go:97   SearchTurboQuantWithOptions  (vetorial/TurboQuant)
//	hybrid.go:28  SearchFTSWithOptions         (textual/FTS5)
//	hybrid.go:130 expansão de vizinhos no grafo dentro do híbrido
//
// Os documentos entram por IndexSingleFileSQLite — o mesmo caminho do CLI — e
// não por INSERT artesanal, para que o teste proves a fiação de produção do
// mtime e não só a coluna existir.
//
// A expansão no grafo é isolada de propósito: o vizinho é indexado SEM
// quantizador, então não aparece entre os candidatos do TurboQuant, e o seu
// conteúdo não contém o termo buscado, então também não aparece no FTS. O
// único caminho por onde ele chega ao resultado fundido é a query do grafo.
func TestBuscaSQLitePopulaMTimeEmTodosOsCaminhos(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.InitDB(filepath.Join(tmpDir, "mtime_paths.db"))
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "binary was compiled without cgo") {
			t.Skip("Pulando teste SQLite: ambiente sem CGO")
		}
		t.Fatalf("erro ao iniciar banco SQLite: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	quantizer := turboquant.NewQuantizer(embedder.EmbeddingDim)

	// marca única: só a semente casa com ela, então FTS e TurboQuant não
	// arrastam o vizinho para o resultado fundido.
	const marker = "zettelpathsXyz"

	seedPath := filepath.Join(tmpDir, "semente.md")
	neighborPath := filepath.Join(tmpDir, "vizinha.md")

	edited := time.Now().Add(-72 * time.Hour).Truncate(time.Second)
	if err := writeNote(seedPath, "# Semente\n\n"+marker+" conteúdo da semente.\n", edited); err != nil {
		t.Fatalf("escrever semente: %v", err)
	}
	// O vizinho tem mtime DIFERENTE do da semente: se qualquer caminho
	// devolvesse o valor do documento errado, a asserção pegaria.
	if err := writeNote(neighborPath, "# Vizinha\n\nconteúdo da vizinha, sem o marcador.\n", edited.Add(-48*time.Hour)); err != nil {
		t.Fatalf("escrever vizinha: %v", err)
	}

	// Semente com quantizador (entra nos candidatos vetoriais) e vizinha sem
	// quantizador (some do TurboQuant, sobrando só a via do grafo).
	if _, err := IndexSingleFileSQLite(ctx, database, nil, quantizer, seedPath, true); err != nil {
		t.Fatalf("indexar semente: %v", err)
	}
	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, neighborPath, true); err != nil {
		t.Fatalf("indexar vizinha: %v", err)
	}

	// A aresta é criada direto porque o mtime vem do indexador de qualquer
	// forma; o que importa aqui é que o grafo tenha a ligação seed->vizinha.
	if err := db.InsertEdgeWithProps(ctx, database, seedPath, neighborPath, "links_to", "EXTRACTED", 1.0); err != nil {
		t.Fatalf("InsertEdgeWithProps: %v", err)
	}

	wantMTime := func(path string) int64 {
		t.Helper()
		return fileMtime(t, path)
	}

	queryVec := make([]float32, embedder.EmbeddingDim)

	cases := []struct {
		name string
		// wantDoc é o documento que precisa aparecer — e traz o mtime dele.
		wantDoc string
		run     func(t *testing.T) []db.SearchResult
	}{
		{
			name:    "fts/hybrid.go:28",
			wantDoc: seedPath,
			run: func(t *testing.T) []db.SearchResult {
				res, err := db.SearchFTSWithOptions(ctx, database, marker, 5, store.SearchOptions{Level: "l1"})
				if err != nil {
					t.Fatalf("SearchFTSWithOptions: %v", err)
				}
				return res
			},
		},
		{
			name:    "turboquant/graph.go:97",
			wantDoc: seedPath,
			run: func(t *testing.T) []db.SearchResult {
				res, err := db.SearchTurboQuantWithOptions(ctx, database, quantizer, queryVec, 5, store.SearchOptions{Level: "l1"})
				if err != nil {
					t.Fatalf("SearchTurboQuantWithOptions: %v", err)
				}
				return res
			},
		},
		{
			name:    "grafo/hybrid.go:130",
			wantDoc: neighborPath,
			run: func(t *testing.T) []db.SearchResult {
				res, err := db.SearchHybridRRFWithOptions(ctx, database, quantizer, marker, queryVec, 10, 60, true,
					store.DecayOptions{Enabled: false}, store.SearchOptions{Level: "l1"})
				if err != nil {
					t.Fatalf("SearchHybridRRFWithOptions: %v", err)
				}
				return res
			},
		},
		{
			name:    "knn/graph.go:51",
			wantDoc: seedPath,
			run: func(t *testing.T) []db.SearchResult {
				if !db.HasSqliteVec {
					t.Skip("sqlite-vec indisponível nesta plataforma (o guard de HasSqliteVec impede a query); " +
						"esta query só é coberta em build cgo com a tag sqlite_fts5")
				}
				res, err := db.SearchKNNWithOptions(ctx, database, queryVec, 5, store.SearchOptions{Level: "l1"})
				if err != nil {
					t.Fatalf("SearchKNNWithOptions: %v", err)
				}
				return res
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := c.run(t)

			var hit *db.SearchResult
			for i := range res {
				if res[i].DocumentID == c.wantDoc {
					hit = &res[i]
					break
				}
			}
			if hit == nil {
				var ids []string
				for _, r := range res {
					ids = append(ids, r.DocumentID)
				}
				t.Fatalf("o caminho não devolveu o documento %q; obteve %v", c.wantDoc, ids)
			}

			want := wantMTime(c.wantDoc)
			if want <= 0 {
				t.Fatalf("mtime do arquivo é %d; o fixture não prova nada", want)
			}
			if hit.MTime != want {
				t.Fatalf("MTime = %d; esperava o mtime do arquivo %d. Este caminho não está populando o mtime, "+
					"e o --decay vai cair para updated_at", hit.MTime, want)
			}
			if hit.MTime == hit.UpdatedAt {
				t.Fatalf("MTime e UpdatedAt coincidem (%d): o mtime do disco não foi persistido, "+
					"só o instante de indexação", hit.MTime)
			}
		})
	}
}

func writeNote(path, content string, edited time.Time) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	return os.Chtimes(path, edited, edited)
}
