package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

// DocumentRef representa um documento indexado no inventário do banco.
// É a unidade de leitura usada pela reconciliação: nada aqui é mutado.
type DocumentRef struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Title string `json:"title"`
}

// PrunedDoc registra um documento removido do índice. Em dry-run, DryRun=true
// indica que o documento é apenas um candidato — nenhuma escrita foi feita.
type PrunedDoc struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	DryRun bool   `json:"dry_run"`
}

// DefaultProtectedPathPrefixes lista prefixos (slash-normalizados, minúsculos)
// que NUNCA podem ser removidos pelo pruning.
//
// `catalog-skills/` é material BRUTO (não conhecimento): ele pode estar ausente
// do índice e isso está correto. Ver ISSUE-015.
var DefaultProtectedPathPrefixes = []string{"catalog-skills/"}

// PruneOptions controla o comportamento destrutivo do pruning.
type PruneOptions struct {
	// DryRun quando true apenas devolve os candidatos, sem emitir nenhum DELETE.
	DryRun bool

	// KeepPaths são caminhos que ainda existem no disco mas estão fora do escopo
	// de indexação (exclusão de configuração). Documentos presentes neles nunca
	// são candidatos: estar ausente do índice é o estado correto.
	KeepPaths map[string]bool

	// ProtectedPrefixes sobrepõe DefaultProtectedPathPrefixes quando não-nil.
	// Uma lista vazia (não-nil) desliga a proteção — usar com cuidado.
	ProtectedPrefixes []string

	// Logger registra cada remoção com id e path. Nil = log.Default().
	Logger *log.Logger
}

func (o PruneOptions) protectedPrefixes() []string {
	if o.ProtectedPrefixes != nil {
		return normalizePrefixes(o.ProtectedPrefixes)
	}
	return normalizePrefixes(DefaultProtectedPathPrefixes)
}

func (o PruneOptions) logger() *log.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return log.Default()
}

func normalizePrefixes(prefixes []string) []string {
	out := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		p = strings.ToLower(strings.TrimSpace(filepath.ToSlash(p)))
		if p == "" {
			continue
		}
		if !strings.HasSuffix(p, "/") {
			p += "/"
		}
		out = append(out, p)
	}
	return out
}

// NormalizePathKey reduz um caminho a uma chave canônica de comparação:
// absoluta quando possível, limpa (sem ".."/"."), com separadores "/" e em
// minúsculas. Separador e caixa não devem gerar falso "arquivo ausente".
//
// A conversão de "\" para "/" acontece ANTES do filepath.Clean, e é
// incondicional. `filepath` é dependente de SO: em Unix a "\" é um caractere
// comum, não um separador, então um caminho gravado pelo indexador do Windows
// passaria inteiro pelo Clean sem ser normalizado, e `filepath.ToSlash` é
// no-op fora do Windows. O efeito era o reconciliador e o pruner não casarem
// NENHUM caminho de um índice criado no Windows — que é exatamente o cenário
// que estes testes existem para cobrir.
func NormalizePathKey(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, `\`, "/")
	p = filepath.Clean(p)
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return strings.ToLower(filepath.ToSlash(p))
}

// AddPathKey registra no conjunto a chave canônica do caminho informado.
func AddPathKey(set map[string]bool, p string) {
	if key := NormalizePathKey(p); key != "" {
		set[key] = true
	}
}

// normalizeRoot devolve a chave canônica da raiz do vault ("" quando vazio).
func normalizeRoot(rootDir string) string {
	return NormalizePathKey(rootDir)
}

// pathKeyVariants devolve as chaves candidatas de um documento persistido.
//
// O banco pode guardar o mesmo arquivo como id relativo ("notes/a.md") ou como
// path absoluto, em qualquer caixa e com "\" ou "/". Todas as formas são
// geradas para que a comparação nunca conclua "ausente" só por diferença de escrita.
func pathKeyVariants(rootDir, p string) []string {
	if strings.TrimSpace(p) == "" {
		return nil
	}

	seen := make(map[string]bool, 3)
	keys := make([]string, 0, 3)
	add := func(candidate string) {
		key := NormalizePathKey(candidate)
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		keys = append(keys, key)
	}

	add(p)
	if normalizeRoot(rootDir) != "" && !filepath.IsAbs(p) {
		// Documento persistido de forma relativa: a raiz do vault é a âncora.
		add(filepath.Join(rootDir, p))
	}

	return keys
}

// pathWithinRoot verifica se o caminho está dentro da raiz do vault,
// respeitando a fronteira de segmento ("/repo/notes" não casa "/repo/notes2").
func pathWithinRoot(rootKey, p string) bool {
	if rootKey == "" {
		return true
	}
	key := NormalizePathKey(p)
	return key == rootKey || strings.HasPrefix(key, rootKey+"/")
}

// buildActiveSet monta o conjunto de comparação a partir de uma lista de caminhos.
func buildActiveSet(paths []string) map[string]bool {
	active := make(map[string]bool, len(paths)*2)
	for _, p := range paths {
		AddPathKey(active, p)
	}
	return active
}

// documentIsActive informa se o documento corresponde a alguma entrada viva.
func documentIsActive(active map[string]bool, rootDir string, ref DocumentRef) bool {
	if len(active) == 0 {
		return false
	}
	for _, base := range []string{ref.ID, ref.Path} {
		for _, key := range pathKeyVariants(rootDir, base) {
			if active[key] {
				return true
			}
		}
	}
	return false
}

// isProtectedPath informa se o documento cai em algum prefixo protegido,
// seja pelo caminho absoluto quanto pelo caminho relativo à raiz do vault.
func isProtectedPath(rootDir string, ref DocumentRef, prefixes []string) bool {
	if len(prefixes) == 0 {
		return false
	}

	rootKey := strings.ToLower(filepath.ToSlash(normalizeRoot(rootDir)))
	candidates := make([]string, 0, 4)
	seen := make(map[string]bool, 4)
	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			return
		}
		seen[candidate] = true
		candidates = append(candidates, candidate)
	}

	for _, base := range []string{ref.ID, ref.Path} {
		add(NormalizePathKey(base))
		// Relativo à raiz do vault: é assim que o prefixo é escrito no
		// .memory/config.yaml ("catalog-skills/**").
		if rootKey != "" {
			raw := strings.ToLower(filepath.ToSlash(filepath.Clean(base)))
			if strings.HasPrefix(raw, rootKey+"/") {
				add(strings.TrimPrefix(raw, rootKey+"/"))
			}
		}
	}

	for _, prefix := range prefixes {
		for _, candidate := range candidates {
			if strings.HasPrefix(candidate, prefix) {
				return true
			}
		}
	}
	return false
}

// selectPrunableDocuments lista os documentos do repositório que sobrevivem ao
// filtro de raiz e que NÃO correspondem a nenhum caminho vivo. Não muta nada.
func (s *PostgresStore) selectPrunableDocuments(ctx context.Context, repo, rootDir string, active map[string]bool) ([]DocumentRef, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path, COALESCE(title, '') FROM documents WHERE repository = $1 ORDER BY id
	`, repo)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar documentos para pruning: %w", err)
	}
	defer rows.Close()

	rootKey := normalizeRoot(rootDir)
	var candidates []DocumentRef
	for rows.Next() {
		var ref DocumentRef
		if err := rows.Scan(&ref.ID, &ref.Path, &ref.Title); err != nil {
			continue
		}

		if rootKey != "" && !pathWithinRoot(rootKey, ref.Path) && !pathWithinRoot(rootKey, ref.ID) {
			continue
		}

		if documentIsActive(active, rootDir, ref) {
			continue
		}

		candidates = append(candidates, ref)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao varrer documentos para pruning: %w", err)
	}
	return candidates, nil
}

// deleteDocumentRows remove documento, chunks, tags, arestas e nós numa única transação.
//
// As tags são removidas explicitamente: no Postgres `document_tags` não tem
// foreign key para `documents` (a identidade do documento lá é
// `(repository, id)`), então o prune apagaria o documento e deixaria as linhas
// de tag órfãs.
func (s *PostgresStore) deleteDocumentRows(ctx context.Context, repo, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	statements := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM chunks WHERE repository = $1 AND document_id = $2`, []any{repo, id}},
		{`DELETE FROM document_tags WHERE repository = $1 AND document_id = $2`, []any{repo, id}},
		{`DELETE FROM graph_edges WHERE repository = $1 AND (source_id = $2 OR target_id = $2)`, []any{repo, id}},
		{`DELETE FROM documents WHERE repository = $1 AND id = $2`, []any{repo, id}},
		{`DELETE FROM graph_nodes WHERE repository = $1 AND id = $2`, []any{repo, id}},
	}

	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt.query, stmt.args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListDocumentPaths devolve o inventário de documentos indexados do repositório.
// É estritamente de leitura: existe para a reconciliação comparar o banco com
// o disco sem precisar reindexar nada.
func (s *PostgresStore) ListDocumentPaths(ctx context.Context, repo string) ([]DocumentRef, error) {
	if repo == "" {
		repo = "default"
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path, COALESCE(title, '') FROM documents WHERE repository = $1 ORDER BY id
	`, repo)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar documentos do índice: %w", err)
	}
	defer rows.Close()

	docs := make([]DocumentRef, 0, 64)
	for rows.Next() {
		var ref DocumentRef
		if err := rows.Scan(&ref.ID, &ref.Path, &ref.Title); err != nil {
			continue
		}
		docs = append(docs, ref)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao varrer documentos do índice: %w", err)
	}
	return docs, nil
}

// PruneStaleDocuments remove documentos indexados cujo arquivo não existe mais
// no vault, sem reindexar o repositório inteiro.
//
// `live` é o conjunto de caminhos presentes no disco (já normalizados ou
// normalizáveis). `rootDir` é obrigatório: sem âncora de vault, qualquer
// documento do repositório seria considerado removível.
//
// É a contraparte de `PruneDeletedDocuments`, que só roda como efeito colateral
// de `mem index --prune` e depende de o conjunto ativo vir da indexação.
func (s *PostgresStore) PruneStaleDocuments(ctx context.Context, repo, rootDir string, live map[string]bool, opts PruneOptions) ([]PrunedDoc, error) {
	if repo == "" {
		repo = "default"
	}
	if strings.TrimSpace(rootDir) == "" {
		return nil, errors.New("prune stale: rootDir é obrigatório (sem âncora de vault o recorte seria inseguro)")
	}
	if len(live) == 0 {
		return nil, errors.New("prune stale: nenhum arquivo vivo informado (varredura de disco vazia — abortando para não apagar o índice inteiro)")
	}

	active := make(map[string]bool, len(live)*2)
	for key := range live {
		active[NormalizePathKey(key)] = true
	}
	// Arquivos que ainda existem no disco mas estão fora do escopo de
	// indexação também são "vivos": removê-los seria apagar conhecimento
	// correto. É o que garante que o relatório e a remoção concordem.
	for key := range opts.KeepPaths {
		if normalized := NormalizePathKey(key); normalized != "" {
			active[normalized] = true
		}
	}

	candidates, err := s.selectPrunableDocuments(ctx, repo, rootDir, active)
	if err != nil {
		return nil, err
	}

	prefixes := opts.protectedPrefixes()
	logger := opts.logger()

	pruned := make([]PrunedDoc, 0, len(candidates))
	for _, ref := range candidates {
		if isProtectedPath(rootDir, ref, prefixes) {
			logger.Printf("[reconcile] preservado (material bruto, fora do índice por exclusão): id=%s path=%s", ref.ID, ref.Path)
			continue
		}

		if opts.DryRun {
			pruned = append(pruned, PrunedDoc{ID: ref.ID, Path: ref.Path, DryRun: true})
			continue
		}

		if err := s.deleteDocumentRows(ctx, repo, ref.ID); err != nil {
			logger.Printf("[reconcile] falha ao remover id=%s path=%s: %v", ref.ID, ref.Path, err)
			continue
		}

		logger.Printf("[reconcile] removido do índice: id=%s path=%s", ref.ID, ref.Path)
		pruned = append(pruned, PrunedDoc{ID: ref.ID, Path: ref.Path})
	}

	return pruned, nil
}
