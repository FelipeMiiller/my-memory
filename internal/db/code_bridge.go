package db

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
)

// Ponte entre o grafo de notas e o grafo de código (ADR-061, CG-17/CG-18).
//
// A ponte mora no GRAFO ÚNICO (`graph_nodes` + `graph_edges`): um nó
// `type='code_symbol'` (id = "code:" + repository + ":" + qualified_name) e
// uma aresta `relation='references_code'` do documento para o símbolo.
//
// A aresta carrega o status epistêmico (ADR-011):
//   - EXTRACTED — o match foi pelo qualified name LITERAL do símbolo ou pelo
//     caminho LITERAL do arquivo onde ele vive;
//   - INFERRED  — o match foi só pelo NOME do símbolo (com fronteira de
//     palavra). É um palpite, não uma extração.
//
// Prioridade de match (a primeira que casar vence):
//  1. qualified_name literal no texto do documento;
//  2. caminho do arquivo do símbolo, literal, no texto do documento;
//  3. nome do símbolo com fronteira de palavra (fraco).
//
// A função é IDEMPOTENTE: rodar duas vezes não duplica aresta (o upsert
// atualiza o status epistêmico no conflito). É chamada pelo code stage
// (depois que os símbolos foram persistidos) e pelo indexador de Markdown
// (para documentos novos).
const (
	// CodeSymbolNodeType é o `type` do graph_nodes que representa um símbolo
	// de código dentro do grafo único.
	CodeSymbolNodeType = "code_symbol"

	// CodeSymbolNodePrefix prefixa o id do nó de código:
	// "code:<repository>:<qualified_name>".
	CodeSymbolNodePrefix = "code:"

	// BridgeRelation é a relação da aresta documento → símbolo de código.
	// `graph_edges` não tem CHECK de relation — a relação nova não exige DDL.
	BridgeRelation = "references_code"

	// bridgeMaxDocBytes limita o texto carregado por documento na ponte.
	// Uma nota maior que isso é truncada: a ponte é um índice de MENÇÃO, e
	// mencionar um símbolo nas primeiras centenas de KB já é o caso comum.
	bridgeMaxDocBytes = 1 << 20
)

// CodeBridgeResult resume o que a ponte construiu num passe.
type CodeBridgeResult struct {
	// Documents é o número de documentos considerados.
	Documents int `json:"documents"`
	// Symbols é o número de símbolos de código considerados.
	Symbols int `json:"symbols"`
	// Nodes é o número de nós `code_symbol` gravados (upsert).
	Nodes int `json:"nodes"`
	// Edges é o número de arestas `references_code` gravadas (upsert).
	Edges int `json:"edges"`
	// Matched é o número de pares documento→símbolo casados.
	Matched int `json:"matched"`
}

// bridgeSymbol é a visão mínima de um code_symbol para a ponte.
type bridgeSymbol struct {
	ID            int64
	Name          string
	QualifiedName string
	FilePath      string
	// Repository é o repo_id do símbolo (code_symbols.repository): o id do
	// nó no grafo único carrega o repositório.
	Repository string
}

// NodeID devolve o id do nó no grafo único:
// "code:" + repository + ":" + qualified_name.
//
// Quando o símbolo não tem qualified name, o nome vira a identidade.
func (s bridgeSymbol) NodeID() string {
	qn := strings.TrimSpace(s.QualifiedName)
	if qn == "" {
		qn = strings.TrimSpace(s.Name)
	}
	return CodeSymbolNodeID(s.Repository, qn)
}

// BuildCodeBridge casa os documentos indexados com os símbolos de código e
// grava a ponte no grafo único.
//
// Idempotente: o upsert de `graph_edges` atualiza o status epistêmico no
// conflito, então rodar duas vezes produz exatamente o mesmo grafo.
//
// O escopo dos SÍMBOLOS é o `repository` carregado no contexto nos DOIS
// engines (code_symbols/code_files têm a coluna): um símbolo de outro
// repositório nunca é oferecido ao casamento. O escopo dos DOCUMENTOS segue a
// tabela `documents`: no SQLite (que não tem `repository`) a ponte é local ao
// vault; no PostgreSQL tudo é filtrado pelo `repository` do contexto.
func BuildCodeBridge(ctx context.Context, database *sql.DB) (*CodeBridgeResult, error) {
	return buildCodeBridge(ctx, database, "")
}

// BuildCodeBridgeForDocument reconstrói a ponte de UM documento.
//
// É o caminho do indexador de Markdown (documento novo): o documento é
// casado contra os `code_symbols` já existentes. O índice de símbolos é
// cacheado por vault — durante o pipeline Markdown a tabela `code_symbols`
// não muda, então o custo é um carregamento por execução.
func BuildCodeBridgeForDocument(ctx context.Context, database *sql.DB, documentID string) (*CodeBridgeResult, error) {
	return buildCodeBridge(ctx, database, strings.TrimSpace(documentID))
}

// buildCodeBridge é o núcleo da ponte. Com documentID vazio, processa todos
// os documentos.
func buildCodeBridge(ctx context.Context, database *sql.DB, documentID string) (*CodeBridgeResult, error) {
	if database == nil {
		return nil, fmt.Errorf("BuildCodeBridge: database == nil")
	}

	dialect := DetectDialect(ctx, database)

	docs, err := loadBridgeDocuments(ctx, database, dialect)
	if err != nil {
		return nil, err
	}
	if documentID != "" {
		filtered := docs[:0]
		for _, d := range docs {
			if d.ID == documentID {
				filtered = append(filtered, d)
			}
		}
		docs = filtered
	}

	index, err := bridgeSymbolIndex(ctx, database, documentID == "")
	if err != nil {
		return nil, err
	}

	result := &CodeBridgeResult{
		Documents: len(docs),
		Symbols:   len(index.symbols),
	}
	if len(index.symbols) == 0 || len(docs) == 0 {
		return result, nil
	}

	byName := index.byName
	byPath := index.byPath
	pathsByBase := index.pathsByBase

	for _, doc := range docs {
		content := doc.Content
		if strings.TrimSpace(content) == "" {
			continue
		}
		words := bridgeWords(content)

		// status por nó do grafo: EXTRACTED sempre ganha de INFERRED.
		statuses := make(map[string]string)
		symbolOf := make(map[string]bridgeSymbol)

		mark := func(nodeID string, status string, sym bridgeSymbol) {
			if current, ok := statuses[nodeID]; ok && current == EpistemicExtracted {
				return
			}
			statuses[nodeID] = status
			symbolOf[nodeID] = sym
		}

		// 1) qualified name LITERAL.
		for _, word := range words {
			for _, s := range byName[word] {
				qn := strings.TrimSpace(s.QualifiedName)
				if qn == "" {
					continue
				}
				if strings.Contains(content, qn) {
					mark(s.NodeID(), EpistemicExtracted, s)
				}
			}
		}

		// 2) caminho do arquivo LITERAL.
		for _, word := range words {
			for _, p := range pathsByBase[word] {
				if !bridgeContainsPath(content, p) {
					continue
				}
				for _, s := range byPath[p] {
					mark(s.NodeID(), EpistemicExtracted, s)
				}
			}
		}

		// 3) nome do símbolo com fronteira de palavra (fraco).
		for _, word := range words {
			for _, s := range byName[word] {
				nodeID := s.NodeID()
				if _, ok := statuses[nodeID]; ok {
					continue
				}
				mark(nodeID, EpistemicInferred, s)
			}
		}

		if len(statuses) == 0 {
			continue
		}

		nodeIDs := make([]string, 0, len(statuses))
		for id := range statuses {
			nodeIDs = append(nodeIDs, id)
		}
		sort.Strings(nodeIDs)

		for _, nodeID := range nodeIDs {
			sym := symbolOf[nodeID]
			if err := upsertBridgeNode(ctx, database, dialect, nodeID, sym.Name); err != nil {
				return nil, err
			}
			result.Nodes++
			if err := upsertBridgeEdge(ctx, database, dialect, doc.ID, nodeID, statuses[nodeID]); err != nil {
				return nil, err
			}
			result.Edges++
			result.Matched++
		}
	}

	return result, nil
}

// bridgeIndex é o índice de casamento dos símbolos de código.
type bridgeIndex struct {
	symbols     []bridgeSymbol
	byName      map[string][]bridgeSymbol
	byPath      map[string][]bridgeSymbol
	pathsByBase map[string][]string
}

// bridgeIndexCache memoiza o índice de símbolos por vault.
//
// A chave de validade é o CONTAGEM de code_symbols: durante o pipeline
// Markdown a tabela não muda, então o índice é carregado uma vez por
// execução. A ponte COMPLETA (o code stage) recarrega sempre.
var bridgeIndexCache sync.Map

// bridgeSymbolIndex devolve o índice de símbolos, recarregando quando
// preciso.
func bridgeSymbolIndex(ctx context.Context, database *sql.DB, force bool) (*bridgeIndex, error) {
	key := database
	// A chave inclui o repository nos DOIS engines: o índice de símbolos é
	// filtrado por repositório, então dois repositórios no mesmo banco
	// precisam de índices separados (sem isso, o cache de um repo oferece os
	// símbolos do outro).
	repo := RepositoryFromContext(ctx)
	cacheKey := struct {
		db   *sql.DB
		repo string
	}{db: key, repo: repo}

	if !force {
		if cached, ok := bridgeIndexCache.Load(cacheKey); ok {
			if idx, ok := cached.(*bridgeIndex); ok {
				return idx, nil
			}
		}
	}

	symbols, err := loadBridgeSymbols(ctx, database)
	if err != nil {
		return nil, err
	}

	idx := &bridgeIndex{
		symbols:     symbols,
		byName:      make(map[string][]bridgeSymbol),
		byPath:      make(map[string][]bridgeSymbol),
		pathsByBase: make(map[string][]string),
	}
	for _, s := range symbols {
		if name := strings.TrimSpace(s.Name); name != "" {
			idx.byName[name] = append(idx.byName[name], s)
		}
		p := strings.TrimSpace(s.FilePath)
		if p == "" {
			continue
		}
		idx.byPath[p] = append(idx.byPath[p], s)
		base := bridgeBaseName(p)
		if base == "" {
			continue
		}
		if !containsString(idx.pathsByBase[base], p) {
			idx.pathsByBase[base] = append(idx.pathsByBase[base], p)
		}
	}
	for k := range idx.pathsByBase {
		sort.Strings(idx.pathsByBase[k])
	}

	bridgeIndexCache.Store(cacheKey, idx)
	return idx, nil
}

// upsertBridgeNode garante que o nó do símbolo existe no grafo único.
//
// O nó da origem (o documento) também é garantido: `graph_edges` tem FK
// para `graph_nodes` nos dois engines.
func upsertBridgeNode(ctx context.Context, database *sql.DB, dialect Dialect, nodeID, name string) error {
	if strings.TrimSpace(name) == "" {
		name = nodeID
	}
	if dialect.IsPostgres() {
		query := DetectDialect(ctx, database).Bind(`
			INSERT INTO graph_nodes (id, repository, type, name)
			VALUES (?, ?, '` + CodeSymbolNodeType + `', ?)
			ON CONFLICT (repository, id) DO NOTHING`)
		_, err := database.ExecContext(ctx, query, nodeID, RepositoryFromContext(ctx), name)
		return err
	}
	query := `
		INSERT OR IGNORE INTO graph_nodes (id, type, name)
		VALUES (?, '` + CodeSymbolNodeType + `', ?)`
	_, err := database.ExecContext(ctx, query, nodeID, name)
	return err
}

// upsertBridgeEdge grava (ou atualiza) a aresta documento → símbolo.
func upsertBridgeEdge(ctx context.Context, database *sql.DB, dialect Dialect, documentID, nodeID, status string) error {
	if strings.TrimSpace(status) == "" {
		status = EpistemicInferred
	}
	if dialect.IsPostgres() {
		query := DetectDialect(ctx, database).Bind(`
			INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status, weight)
			VALUES (?, ?, ?, '` + BridgeRelation + `', ?, 1.0)
			ON CONFLICT (repository, source_id, target_id, relation)
			DO UPDATE SET epistemic_status = excluded.epistemic_status`)
		_, err := database.ExecContext(ctx, query, documentID, nodeID, RepositoryFromContext(ctx), status)
		return err
	}
	query := `
		INSERT INTO graph_edges (source_id, target_id, relation, epistemic_status, weight)
		VALUES (?, ?, '` + BridgeRelation + `', ?, 1.0)
		ON CONFLICT(source_id, target_id, relation)
		DO UPDATE SET epistemic_status = excluded.epistemic_status`
	_, err := database.ExecContext(ctx, query, documentID, nodeID, status)
	return err
}

// loadBridgeSymbols carrega os símbolos de código com o path do arquivo.
//
// O filtro por repository vale para os DOIS engines: é ele que impede a nota
// de um repositório de casar com o símbolo de outro. No SQLite as tabelas
// code_files/code_symbols têm a coluna `repository` (ao contrário de
// graph_nodes), então o mesmo filtro funciona sem mudar o schema.
func loadBridgeSymbols(ctx context.Context, database *sql.DB) ([]bridgeSymbol, error) {
	query := `
		SELECT s.id, s.name, s.qualified_name, f.path, s.repository
		  FROM code_symbols s
		  JOIN code_files f ON f.id = s.file_id
		 WHERE f.repository = ? AND s.repository = ?`
	args := []any{RepositoryFromContext(ctx), RepositoryFromContext(ctx)}
	query += ` ORDER BY s.id ASC`

	rows, err := database.QueryContext(ctx, DetectDialect(ctx, database).Bind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("BuildCodeBridge: carga de code_symbols: %w", err)
	}
	defer rows.Close()

	var out []bridgeSymbol
	for rows.Next() {
		var s bridgeSymbol
		var qn sql.NullString
		var p sql.NullString
		if err := rows.Scan(&s.ID, &s.Name, &qn, &p, &s.Repository); err != nil {
			return nil, err
		}
		if qn.Valid {
			s.QualifiedName = qn.String
		}
		if p.Valid {
			s.FilePath = p.String
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// bridgeDocument é a visão mínima de um documento para a ponte.
type bridgeDocument struct {
	ID      string
	Path    string
	Content string
}

// loadBridgeDocuments carrega os documentos indexados e o seu texto (chunks).
func loadBridgeDocuments(ctx context.Context, database *sql.DB, dialect Dialect) ([]bridgeDocument, error) {
	docQuery := `SELECT id, path FROM documents`
	var docArgs []any
	if dialect.IsPostgres() {
		docQuery += ` WHERE repository = ?`
		docArgs = append(docArgs, RepositoryFromContext(ctx))
	}
	docQuery += ` ORDER BY id ASC`

	docRows, err := database.QueryContext(ctx, DetectDialect(ctx, database).Bind(docQuery), docArgs...)
	if err != nil {
		return nil, fmt.Errorf("BuildCodeBridge: carga de documents: %w", err)
	}
	defer docRows.Close()

	index := make(map[string]int)
	var docs []bridgeDocument
	for docRows.Next() {
		var d bridgeDocument
		if err := docRows.Scan(&d.ID, &d.Path); err != nil {
			return nil, err
		}
		index[d.ID] = len(docs)
		docs = append(docs, d)
	}
	if err := docRows.Err(); err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}

	chunkQuery := `SELECT document_id, content FROM chunks`
	var chunkArgs []any
	if dialect.IsPostgres() {
		chunkQuery += ` WHERE repository = ?`
		chunkArgs = append(chunkArgs, RepositoryFromContext(ctx))
	}
	chunkQuery += ` ORDER BY document_id, chunk_index`

	chunkRows, err := database.QueryContext(ctx, DetectDialect(ctx, database).Bind(chunkQuery), chunkArgs...)
	if err != nil {
		return nil, fmt.Errorf("BuildCodeBridge: carga de chunks: %w", err)
	}
	defer chunkRows.Close()

	for chunkRows.Next() {
		var docID, content string
		if err := chunkRows.Scan(&docID, &content); err != nil {
			return nil, err
		}
		pos, ok := index[docID]
		if !ok {
			continue
		}
		if len(docs[pos].Content) >= bridgeMaxDocBytes {
			continue
		}
		if docs[pos].Content != "" {
			docs[pos].Content += "\n"
		}
		docs[pos].Content += content
	}
	return docs, chunkRows.Err()
}

// bridgeWords extrai os tokens identificadores do texto (sem repetição).
//
// O token é a fronteira de palavra do match fraco (prioridade 3): "Search"
// casa com "Search" mas não com "SearchCode" nem com "research".
func bridgeWords(content string) []string {
	seen := make(map[string]bool)
	var out []string
	for i := 0; i < len(content); {
		r := content[i]
		if !isBridgeWordByte(r) {
			i++
			continue
		}
		j := i
		for j < len(content) && isBridgeWordByte(content[j]) {
			j++
		}
		word := content[i:j]
		if !seen[word] {
			seen[word] = true
			out = append(out, word)
		}
		i = j
	}
	return out
}

// isBridgeWordByte devolve true para os bytes que compõem um identificador.
func isBridgeWordByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '_' || b == '$':
		return true
	default:
		return false
	}
}

// bridgeBaseName devolve o nome do arquivo sem diretório nem extensão.
func bridgeBaseName(p string) string {
	base := path.Base(strings.TrimSpace(p))
	base = strings.TrimSuffix(base, path.Ext(base))
	return base
}

// bridgeContainsPath devolve true quando o texto contém o caminho do
// arquivo — tanto na forma Unix (forward slash) quanto na forma Windows.
func bridgeContainsPath(content, p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	if strings.Contains(content, p) {
		return true
	}
	return strings.Contains(content, strings.ReplaceAll(p, "/", "\\"))
}

// containsString devolve true quando s está na lista.
func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// CodeSymbolNodeID devolve o id do nó do grafo único para um símbolo de
// código: "code:<repository>:<qualified_name>".
//
// O id carrega o repository porque a identidade do nó no SQLite é SÓ o id
// (a PK de `graph_nodes` é `id TEXT PRIMARY KEY`): sem o escopo no id, dois
// repositórios com o mesmo qualified name colapsam no mesmo nó e a nota de
// um repositório passa a apontar para o símbolo do outro. No PostgreSQL a
// identidade é (repository, id), então o formato é o mesmo nos dois engines.
func CodeSymbolNodeID(repository, qualifiedName string) string {
	qn := strings.TrimSpace(qualifiedName)
	if qn == "" {
		return ""
	}
	repo := strings.TrimSpace(repository)
	if repo == "" {
		repo = DefaultRepository
	}
	return CodeSymbolNodePrefix + repo + ":" + qn
}

// CodeBridgeDocument é um documento que referencia um símbolo de código.
type CodeBridgeDocument struct {
	DocumentID string `json:"document_id"`
	Path       string `json:"path"`
	Title      string `json:"title"`
	// Status é o status epistêmico da aresta `references_code` (ADR-011).
	Status string `json:"epistemic_status"`
}

// CodeBridgeDocumentsForSymbol devolve os documentos que referenciam o
// símbolo (pelo qualified name) através da ponte `references_code`.
//
// É o que permite ao memory_code_neighbors incluir os DOCUMENTOS que citam o
// símbolo (AC 19) e ao drift atribuir a mudança estrutural às notas que
// dependem do símbolo (AC 26).
func CodeBridgeDocumentsForSymbol(ctx context.Context, database *sql.DB, qualifiedName string) ([]CodeBridgeDocument, error) {
	nodeID := CodeSymbolNodeID(RepositoryFromContext(ctx), qualifiedName)
	if nodeID == "" {
		return nil, nil
	}
	return CodeBridgeDocumentsForNode(ctx, database, nodeID)
}

// CodeBridgeDocumentsForNode devolve os documentos que referenciam o nó de
// código informado.
func CodeBridgeDocumentsForNode(ctx context.Context, database *sql.DB, nodeID string) ([]CodeBridgeDocument, error) {
	if database == nil {
		return nil, fmt.Errorf("CodeBridgeDocumentsForNode: database == nil")
	}
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return nil, nil
	}

	dialect := DetectDialect(ctx, database)
	query := `
		SELECT e.source_id, COALESCE(d.path, ''), COALESCE(d.title, ''), e.epistemic_status
		  FROM graph_edges e
		  LEFT JOIN documents d ON d.id = e.source_id`
	var args []any
	if dialect.IsPostgres() {
		query += ` AND d.repository = e.repository`
	}
	query += `
		 WHERE e.target_id = ? AND e.relation = '` + BridgeRelation + `'`
	args = append(args, nodeID)
	if dialect.IsPostgres() {
		query += ` AND e.repository = ?`
		args = append(args, RepositoryFromContext(ctx))
	}
	query += ` ORDER BY e.source_id ASC`

	rows, err := database.QueryContext(ctx, DetectDialect(ctx, database).Bind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("CodeBridgeDocumentsForNode: %w", err)
	}
	defer rows.Close()

	var out []CodeBridgeDocument
	for rows.Next() {
		var d CodeBridgeDocument
		var status sql.NullString
		if err := rows.Scan(&d.DocumentID, &d.Path, &d.Title, &status); err != nil {
			return nil, err
		}
		if status.Valid {
			d.Status = status.String
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CodeBridgeSymbolsForDocument devolve os ids dos nós de código que o
// documento REFERENCIA (a ponte no sentido documento → símbolo).
//
// É o caminho reverso de CodeBridgeDocumentsForNode: o impacto que começa
// num documento alcança os símbolos de código que ele referencia (AC 19).
func CodeBridgeSymbolsForDocument(ctx context.Context, database *sql.DB, documentID string) ([]string, error) {
	if database == nil {
		return nil, fmt.Errorf("CodeBridgeSymbolsForDocument: database == nil")
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return nil, nil
	}

	dialect := DetectDialect(ctx, database)
	query := `
		SELECT target_id
		  FROM graph_edges
		 WHERE source_id = ? AND relation = '` + BridgeRelation + `'`
	args := []any{documentID}
	if dialect.IsPostgres() {
		query += ` AND repository = ?`
		args = append(args, RepositoryFromContext(ctx))
	}
	query += ` ORDER BY target_id ASC`

	rows, err := database.QueryContext(ctx, DetectDialect(ctx, database).Bind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("CodeBridgeSymbolsForDocument: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// CodeFileHash devolve o `ast_hash` gravado de um arquivo de código.
//
// A busca é TOLERANTE: primeiro tenta o repository do contexto; se não
// achar, usa o arquivo de qualquer repository (o vault local tem um só) —
// assim o drift funciona mesmo quando o comando não carrega o repository no
// contexto.
//
// Devolve ("", false, nil) quando não há hash gravado: isso NÃO é erro
// (AC 27), é a ausência de referência.
func CodeFileHash(ctx context.Context, database *sql.DB, path string) (string, bool, error) {
	if database == nil {
		return "", false, fmt.Errorf("CodeFileHash: database == nil")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false, nil
	}

	// 1) o repository do contexto.
	_, astHash, ok, err := GetCodeFileHash(ctx, database, path)
	if err != nil {
		return "", false, err
	}
	if ok && strings.TrimSpace(astHash) != "" {
		return astHash, true, nil
	}

	// 2) fallback: qualquer repository.
	query := `
		SELECT ast_hash
		  FROM code_files
		 WHERE path = ? AND ast_hash IS NOT NULL AND ast_hash <> ''
		 ORDER BY indexed_at DESC
		 LIMIT 1`
	rows, err := database.QueryContext(ctx, DetectDialect(ctx, database).Bind(query), path)
	if err != nil {
		return "", false, err
	}
	defer rows.Close()
	if rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return "", false, err
		}
		if strings.TrimSpace(hash) != "" {
			return hash, true, nil
		}
	}
	return "", false, rows.Err()
}

// CodeSymbolSignature é um símbolo gravado em code_symbols com a sua
// assinatura — a referência que o drift compara com o arquivo atual para
// detectar mudança ESTRUTURAL (AC 26).
type CodeSymbolSignature struct {
	QualifiedName string `json:"qualified_name"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Signature     string `json:"signature"`
	StartLine     int    `json:"start_line"`
}

// CodeSymbolSignatures devolve os símbolos gravados do arquivo de código
// (code_files.path).
func CodeSymbolSignatures(ctx context.Context, database *sql.DB, path string) ([]CodeSymbolSignature, error) {
	if database == nil {
		return nil, fmt.Errorf("CodeSymbolSignatures: database == nil")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}

	dialect := DetectDialect(ctx, database)
	query := `
		SELECT COALESCE(s.qualified_name, ''), s.name, s.kind, COALESCE(s.signature, ''), s.start_line
		  FROM code_symbols s
		  JOIN code_files f ON f.id = s.file_id
		 WHERE f.path = ?`
	args := []any{path}
	if dialect.IsPostgres() {
		query += ` AND f.repository = ? AND s.repository = ?`
		args = append(args, RepositoryFromContext(ctx), RepositoryFromContext(ctx))
	}
	query += ` ORDER BY s.id ASC`

	rows, err := database.QueryContext(ctx, DetectDialect(ctx, database).Bind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("CodeSymbolSignatures: %w", err)
	}
	defer rows.Close()

	var out []CodeSymbolSignature
	for rows.Next() {
		var s CodeSymbolSignature
		if err := rows.Scan(&s.QualifiedName, &s.Name, &s.Kind, &s.Signature, &s.StartLine); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CodeBridgeDocumentsForSymbols devolve os documentos que referenciam
// QUALQUER um dos símbolos informados.
//
// É o caminho de atribuição do drift estrutural: os documentos que citam os
// símbolos do arquivo que mudou (AC 26).
func CodeBridgeDocumentsForSymbols(ctx context.Context, database *sql.DB, qualifiedNames []string) ([]CodeBridgeDocument, error) {
	if database == nil {
		return nil, fmt.Errorf("CodeBridgeDocumentsForSymbols: database == nil")
	}
	ids := make([]string, 0, len(qualifiedNames))
	repository := RepositoryFromContext(ctx)
	for _, qn := range qualifiedNames {
		if nodeID := CodeSymbolNodeID(repository, qn); nodeID != "" {
			ids = append(ids, nodeID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	seen := make(map[string]bool)
	var out []CodeBridgeDocument
	// Lotes de 200: o SQLite tem limite de variáveis por statement.
	const batch = 200
	for start := 0; start < len(ids); start += batch {
		end := start + batch
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]

		dialect := DetectDialect(ctx, database)
		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)+1)
		for _, id := range chunk {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		query := `
			SELECT e.source_id, COALESCE(d.path, ''), COALESCE(d.title, ''), e.epistemic_status
			  FROM graph_edges e
			  LEFT JOIN documents d ON d.id = e.source_id`
		if dialect.IsPostgres() {
			query += ` AND d.repository = e.repository`
		}
		query += `
			 WHERE e.target_id IN (` + strings.Join(placeholders, ", ") + `)
			   AND e.relation = '` + BridgeRelation + `'`
		if dialect.IsPostgres() {
			query += ` AND e.repository = ?`
			args = append(args, RepositoryFromContext(ctx))
		}
		query += ` ORDER BY e.source_id ASC`

		rows, err := database.QueryContext(ctx, DetectDialect(ctx, database).Bind(query), args...)
		if err != nil {
			return nil, fmt.Errorf("CodeBridgeDocumentsForSymbols: %w", err)
		}
		for rows.Next() {
			var d CodeBridgeDocument
			var status sql.NullString
			if err := rows.Scan(&d.DocumentID, &d.Path, &d.Title, &status); err != nil {
				rows.Close()
				return nil, err
			}
			if seen[d.DocumentID] {
				continue
			}
			seen[d.DocumentID] = true
			if status.Valid {
				d.Status = status.String
			}
			out = append(out, d)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// CodeSymbolRef é a referência a um símbolo de código resolvida a partir do
// nó do grafo (`code:<repository>:<qualified_name>`).
type CodeSymbolRef struct {
	SymbolID      int64  `json:"symbol_id"`
	QualifiedName string `json:"qualified_name"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Language      string `json:"language"`
	File          string `json:"file"`
	StartLine     int    `json:"start_line"`
	EndLine       int    `json:"end_line"`
}

// CodeSymbolByNodeID devolve o símbolo de código do nó
// `code:<repository>:<qualified_name>` do grafo.
//
// Aceita o id completo do nó, o formato legado (`code:<qualified_name>`) ou o
// qualified name puro.
func CodeSymbolByNodeID(ctx context.Context, database *sql.DB, nodeID string) (*CodeSymbolRef, error) {
	if database == nil {
		return nil, fmt.Errorf("CodeSymbolByNodeID: database == nil")
	}
	qn := codeSymbolQualifiedNameFromNodeID(nodeID)
	if qn == "" {
		return nil, nil
	}

	query := `
		SELECT s.id, COALESCE(s.qualified_name, ''), s.name, s.kind,
		       f.language, f.path, s.start_line, s.end_line
		  FROM code_symbols s
		  JOIN code_files f ON f.id = s.file_id
		 WHERE s.qualified_name = ?
		   AND s.repository = ?
		   AND f.repository = ?`
	args := []any{qn, RepositoryFromContext(ctx), RepositoryFromContext(ctx)}
	query += ` ORDER BY s.id ASC LIMIT 1`

	rows, err := database.QueryContext(ctx, DetectDialect(ctx, database).Bind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("CodeSymbolByNodeID: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	var ref CodeSymbolRef
	if err := rows.Scan(&ref.SymbolID, &ref.QualifiedName, &ref.Name, &ref.Kind,
		&ref.Language, &ref.File, &ref.StartLine, &ref.EndLine); err != nil {
		return nil, err
	}
	return &ref, nil
}

// codeSymbolQualifiedNameFromNodeID extrai o qualified name do id do nó.
//
// O id atual é "code:<repository>:<qualified_name>": o primeiro ":" DEPOIS do
// prefixo separa o repository do qualified name. O id legado
// ("code:<qualified_name>") e o qualified name puro continuam válidos.
func codeSymbolQualifiedNameFromNodeID(nodeID string) string {
	id := strings.TrimSpace(nodeID)
	if id == "" {
		return ""
	}
	if !strings.HasPrefix(id, CodeSymbolNodePrefix) {
		// Qualified name puro (o caminho já suportado).
		return id
	}
	rest := strings.TrimPrefix(id, CodeSymbolNodePrefix)
	if rest == "" {
		return ""
	}
	if _, qn, ok := strings.Cut(rest, ":"); ok {
		return strings.TrimSpace(qn)
	}
	// Formato legado: "code:<qualified_name>".
	return strings.TrimSpace(rest)
}
