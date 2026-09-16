package federation

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SearchFunc define a assinatura de uma função de busca compatível com MCP
type SearchFunc func(ctx context.Context, params SearchParams) ([]SearchResult, error)

// FederatedSearcher coordena a busca híbrida unificada entre o repositório local e o cofre central
type FederatedSearcher struct {
	localFn          SearchFunc
	centralFn        SearchFunc
	centralVaultPath string
	centralReadOnly  bool
	logger           func(format string, args ...any)
}

// NewFederatedSearcher instancia um novo motor de busca federada
func NewFederatedSearcher(localFn, centralFn SearchFunc, centralVaultPath string, centralReadOnly bool) *FederatedSearcher {
	return &FederatedSearcher{
		localFn:          localFn,
		centralFn:        centralFn,
		centralVaultPath: centralVaultPath,
		centralReadOnly:  centralReadOnly,
	}
}

// SetLogger define a função de logging para avisos não-bloqueantes da federação
func (fs *FederatedSearcher) SetLogger(l func(format string, args ...any)) {
	fs.logger = l
}

// Search executa a busca federada unificada com Reciprocal Rank Fusion (RRF)
func (fs *FederatedSearcher) Search(ctx context.Context, params SearchParams) ([]SearchResult, error) {
	// Se não houver função local nem central
	if fs.localFn == nil && fs.centralFn == nil {
		return nil, fmt.Errorf("nenhum backend de busca (local ou central) configurado")
	}

	// Se não houver central configurado, consulta diretamente o local
	if fs.centralFn == nil || strings.TrimSpace(fs.centralVaultPath) == "" {
		if fs.localFn != nil {
			return fs.localFn(ctx, params)
		}
		return nil, fmt.Errorf("repositório local não configurado")
	}

	// Se houver caminho central mas ele não existir fisicamente no disco (ex: Google Drive desconectado)
	if _, err := os.Stat(fs.centralVaultPath); err != nil {
		if fs.logger != nil {
			fs.logger("[federation] aviso: cofre central '%s' inacessível: %v; operando apenas em modo local\n", fs.centralVaultPath, err)
		}
		if fs.localFn != nil {
			return fs.localFn(ctx, params)
		}
		return nil, fmt.Errorf("cofre central indisponível e repositório local não configurado: %w", err)
	}

	type searchOutcome struct {
		results []SearchResult
		err     error
		origin  string
	}

	localCh := make(chan searchOutcome, 1)
	centralCh := make(chan searchOutcome, 1)

	// Busca concorrente no repositório local
	go func() {
		if fs.localFn == nil {
			localCh <- searchOutcome{results: nil, err: fmt.Errorf("sem local"), origin: "local"}
			return
		}
		res, err := fs.localFn(ctx, params)
		localCh <- searchOutcome{results: res, err: err, origin: "local"}
	}()

	// Busca concorrente no cofre central
	go func() {
		res, err := fs.centralFn(ctx, params)
		centralCh <- searchOutcome{results: res, err: err, origin: "central"}
	}()

	localOut := <-localCh
	centralOut := <-centralCh

	// Caso ambas falhem
	if localOut.err != nil && centralOut.err != nil {
		return nil, fmt.Errorf("busca federada falhou em ambas as fontes: local (%v), central (%v)", localOut.err, centralOut.err)
	}

	// Se o cofre central falhou, faz fallback suave retornando os resultados locais
	if centralOut.err != nil {
		if fs.logger != nil {
			fs.logger("[federation] aviso: falha ao consultar cofre central: %v; retornando apenas contexto local\n", centralOut.err)
		}
		return localOut.results, nil
	}

	// Se o local falhou (ex: executado fora de um repo git), retorna os resultados do central
	if localOut.err != nil {
		return AnnotateProvenance(centralOut.results, "central"), nil
	}

	// Ambas as buscas tiveram sucesso: mescla via RRF
	k := params.K
	if k <= 0 {
		k = 60
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 5
	}

	return MergeFederatedResults(localOut.results, centralOut.results, k, limit), nil
}

// AnnotateProvenance anota a proveniência de cada resultado ([local] ou [central])
func AnnotateProvenance(results []SearchResult, origin string) []SearchResult {
	prefix := "[" + origin + "] "
	annotated := make([]SearchResult, len(results))
	for i, r := range results {
		res := r
		if !strings.HasPrefix(res.DocumentID, prefix) {
			res.DocumentID = prefix + res.DocumentID
		}
		if res.Repository == "" {
			res.Repository = "[" + origin + "]"
		} else if !strings.HasPrefix(res.Repository, prefix) {
			res.Repository = prefix + res.Repository
		}
		res.Sources = append([]string(nil), res.Sources...)
		res.Sources = append(res.Sources, origin)
		annotated[i] = res
	}
	return annotated
}

// MergeFederatedResults combina os resultados locais e centrais utilizando Reciprocal Rank Fusion (RRF)
func MergeFederatedResults(localResults, centralResults []SearchResult, k int, limit int) []SearchResult {
	if k <= 0 {
		k = 60
	}

	type fusedItem struct {
		result SearchResult
		score  float64
	}

	fusedMap := make(map[string]*fusedItem)

	// Processa os resultados locais (RRF Score: 1 / (k + rank))
	for rank, r := range localResults {
		score := 1.0 / float64(k+rank+1)
		key := r.ChunkID
		if key == "" {
			key = r.DocumentID
		}

		annotated := r
		if !strings.HasPrefix(annotated.DocumentID, "[local] ") {
			annotated.DocumentID = "[local] " + annotated.DocumentID
		}
		if annotated.Repository == "" {
			annotated.Repository = "[local]"
		} else if !strings.HasPrefix(annotated.Repository, "[local] ") {
			annotated.Repository = "[local] " + annotated.Repository
		}
		annotated.Sources = append([]string(nil), annotated.Sources...)
		annotated.Sources = append(annotated.Sources, "local")

		if existing, ok := fusedMap[key]; ok {
			existing.score += score
			existing.result.Sources = append(existing.result.Sources, "local")
		} else {
			fusedMap[key] = &fusedItem{
				result: annotated,
				score:  score,
			}
		}
	}

	// Processa os resultados centrais (RRF Score: 1 / (k + rank))
	for rank, r := range centralResults {
		score := 1.0 / float64(k+rank+1)
		key := r.ChunkID
		if key == "" {
			key = r.DocumentID
		}

		annotated := r
		if !strings.HasPrefix(annotated.DocumentID, "[central] ") {
			annotated.DocumentID = "[central] " + annotated.DocumentID
		}
		if annotated.Repository == "" {
			annotated.Repository = "[central]"
		} else if !strings.HasPrefix(annotated.Repository, "[central] ") {
			annotated.Repository = "[central] " + annotated.Repository
		}
		annotated.Sources = append([]string(nil), annotated.Sources...)
		annotated.Sources = append(annotated.Sources, "central")

		if existing, ok := fusedMap[key]; ok {
			existing.score += score
			existing.result.Sources = append(existing.result.Sources, "central")
		} else {
			fusedMap[key] = &fusedItem{
				result: annotated,
				score:  score,
			}
		}
	}

	// Converte para slice
	items := make([]*fusedItem, 0, len(fusedMap))
	for _, item := range fusedMap {
		item.result.Score = item.score
		items = append(items, item)
	}

	// Ordena decrescente pelo score RRF acumulado
	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	// Aplica o limite
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	merged := make([]SearchResult, len(items))
	for i, item := range items {
		merged[i] = item.result
	}

	return merged
}

// ResolveCentralSQLitePath retorna o caminho canônico do banco SQLite no cofre central
func ResolveCentralSQLitePath(centralVaultPath string) string {
	if centralVaultPath == "" {
		return ""
	}
	preferred := filepath.Join(centralVaultPath, ".memory", "storage", "memory.db")
	if _, err := os.Stat(preferred); err == nil {
		return preferred
	}
	// Fallback para caso legado
	fallback := filepath.Join(centralVaultPath, ".memory", "memory.db")
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	return preferred
}

// ReplaceDatabaseInPostgresURL substitui o nome do banco de dados em uma URL PostgreSQL
func ReplaceDatabaseInPostgresURL(urlStr, newDBName string) (string, error) {
	if strings.TrimSpace(urlStr) == "" {
		return "", fmt.Errorf("URL postgres não pode ser vazia")
	}
	if strings.TrimSpace(newDBName) == "" {
		return "", fmt.Errorf("nome do banco de dados não pode ser vazio")
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("URL postgres inválida: %w", err)
	}

	u.Path = "/" + strings.TrimPrefix(newDBName, "/")
	return u.String(), nil
}
