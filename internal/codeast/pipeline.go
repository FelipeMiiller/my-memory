package codeast

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

// Pipeline orquestra o code_pipeline (ADR-047 §Decision Outcome — Pipeline de indexação).
//
// Fluxo por arquivo:
//  1. Detectar linguagem (extensão + shebang).
//  2. Avaliar cache (content_hash + ast_hash) — ver CodeFileCache.
//  3. Em DecodeFirstIndex / DecodeFullReparse: parsear, extrair symbols, persistir.
//  4. Em DecodeSkipParse: nada.
//  5. Em DecodeSkipDownstream: só atualizar content_hash.
//
// O pipeline respeita cfg.Include / cfg.Exclude (ADR-016) e aceita
// langFilter (nomes canônicos) para implementar `--lang=csv`.
type Pipeline struct {
	db        *sql.DB
	cache     *CodeFileCache
	parser    Parser
	langs     []string // whitelist opcional (lower-case). nil/empty = todas top-10.
	processor *FileProcessor
}

// PipelineOptions configura a construção do Pipeline.
type PipelineOptions struct {
	DB *sql.DB
	// Parser fixa o backend de code AST. Quando nil, o pipeline usa o
	// backend padrão do build (newBackendParser): o motor Go (stdlib) para
	// arquivos Go e o MockParser (fallback determinístico do ADR-047) para as
	// demais linguagens.
	Parser    Parser         // se nil, usa o backend padrão do build.
	LangAllow []string       // whitelist (lower-case) ou nil.
	UseAst    bool           // se false, ignora ast_hash (ainda calcula content_hash).
	StatsSink *PipelineStats // opcional; quando != nil, atualiza contadores.
}

// PipelineStats mantém contadores incrementais (útil para logs e testes).
type PipelineStats struct {
	FilesScanned  int
	FilesIndexed  int
	FilesSkipped  int
	FilesErrors   int
	SymbolsTotal  int
	StartedAtUnix int64
	EndedAtUnix   int64
}

// NewPipeline cria um Pipeline a partir de opts.
func NewPipeline(opts PipelineOptions) (*Pipeline, error) {
	if opts.DB == nil {
		return nil, errors.New("NewPipeline: DB nil")
	}
	if err := db.EnsureCodeTables(context.Background(), opts.DB); err != nil {
		return nil, fmt.Errorf("NewPipeline: EnsureCodeTables falhou: %w", err)
	}

	cache := NewCodeFileCache(opts.DB)
	parserImpl := opts.Parser
	autoParser := false
	if parserImpl == nil {
		// Backend padrão do build: o motor de code AST vivo (newBackendParser).
		// O MockParser deixa de ser o caminho de produção — ele continua
		// disponível como fallback determinístico para as linguagens sem
		// backend AST (ADR-047 §Deferral).
		p, err := newBackendParser()
		if err != nil {
			return nil, fmt.Errorf("NewPipeline: backend de code AST indisponível: %w", err)
		}
		parserImpl = p
		autoParser = true
	}

	// StatsSink: se o caller passou um ponteiro externo, ele será populado
	// também (permite observar progresso fora do Pipeline).
	var stats *PipelineStats
	if opts.StatsSink != nil {
		stats = opts.StatsSink
	} else {
		stats = &PipelineStats{}
	}
	stats.StartedAtUnix = time.Now().Unix()

	processor := NewFileProcessor(opts.DB, cache, parserImpl)
	processor.stats = stats
	processor.autoParser = autoParser
	processor.batched = true

	return &Pipeline{
		db:        opts.DB,
		cache:     cache,
		parser:    parserImpl,
		langs:     normalizeLangList(opts.LangAllow),
		processor: processor,
	}, nil
}

// Stats devolve (se houver) o *PipelineStats sendo populado.
func (p *Pipeline) Stats() *PipelineStats {
	if p == nil || p.processor == nil {
		return nil
	}
	return p.processor.stats
}

// Run executa o pipeline no diretório vaultRoot, respeitando cfg.Include/Exclude.
//
// walkSafety: ignora symlinks e pastas com nomes no DefaultExcludedDirs para
// não bloquear desenvolvimento. Vault cfg governs final include/exclude.
func (p *Pipeline) Run(ctx context.Context, vaultRoot string, cfg *config.Config) error {
	if p == nil || p.processor == nil {
		return errors.New("pipeline não inicializado")
	}
	if vaultRoot == "" {
		return errors.New("vaultRoot vazio")
	}
	abs, err := filepath.Abs(vaultRoot)
	if err != nil {
		return fmt.Errorf("Run: filepath.Abs falhou: %w", err)
	}

	files, err := ListCodeFiles(abs, cfg, p.langs)
	if err != nil {
		return fmt.Errorf("Run: listagem de arquivos falhou: %w", err)
	}

	for _, rel := range files {
		if err := p.processor.ProcessFile(ctx, abs, rel); err != nil {
			p.processor.stats.FilesErrors++
			fmt.Fprintf(os.Stderr, "[code-pipeline] %s: %v\n", rel, err)
		}
	}

	// As arestas acumuladas são resolvidas só agora: com todos os arquivos
	// do lote indexados, uma chamada para um símbolo de outro arquivo
	// resolve pelo qualified name (EXTRACTED) em vez de ficar sem destino.
	if err := p.processor.FlushEdges(ctx); err != nil {
		return fmt.Errorf("Run: resolução de arestas falhou: %w", err)
	}

	p.processor.stats.FilesScanned = len(files)
	p.processor.stats.EndedAtUnix = time.Now().Unix()
	return nil
}

// FileProcessor concentra a lógica por-arquivo; está exposto para
// testes via NewFileProcessor (memo interno do Pipeline).
type FileProcessor struct {
	db     *sql.DB
	cache  *CodeFileCache
	parser Parser
	stats  *PipelineStats
	// autoParser indica que o parser NÃO foi injetado pelo caller: o
	// processor escolhe o backend pela linguagem detectada (motor Go para
	// Go; MockParser como fallback nas demais).
	autoParser bool
	fallback   Parser
	// batched indica que as arestas extraídas são acumuladas e resolvidas
	// no fim do lote (FlushEdges) — assim uma chamada para um símbolo de um
	// arquivo ainda não processado resolve pelo qualified name em vez de
	// cair em "não resolvido".
	batched bool
	// pending são as arestas extraídas, aguardando resolução.
	pending []pendingEdge
}

// pendingEdge é uma aresta extraída pelo backend aguardando resolução.
type pendingEdge struct {
	kind             EdgeKind
	srcSymbolID      int64
	dstName          string
	dstQualifiedName string
	fileID           int64
	line             int
	column           int
	reason           string
}

// NewFileProcessor cria um processador isolado (útil para testes unitários).
func NewFileProcessor(database *sql.DB, cache *CodeFileCache, parser Parser) *FileProcessor {
	return &FileProcessor{db: database, cache: cache, parser: parser, stats: &PipelineStats{}}
}

// parserFor devolve o backend adequado à linguagem detectada.
//
// Quando o parser foi injetado pelo caller, ele é sempre usado (comportamento
// dos testes). No caminho de produção (autoParser), Go vai para o motor AST Go
// (stdlib) e as demais linguagens caem no MockParser — o fallback
// determinístico que a ADR-047 exige manter até o binding tree-sitter existir.
func (fp *FileProcessor) parserFor(lang string) Parser {
	if !fp.autoParser {
		return fp.parser
	}
	if lang == "go" {
		return fp.parser
	}
	if fp.fallback == nil {
		fp.fallback = NewMockParser(false)
	}
	return fp.fallback
}

// ProcessFile processa um único arquivo (relPath relativo à raiz).
func (fp *FileProcessor) ProcessFile(ctx context.Context, rootDir, relPath string) error {
	if fp == nil || fp.parser == nil {
		return errors.New("FileProcessor inválido")
	}
	full := filepath.Join(rootDir, relPath)

	// Detecta linguagem
	content, err := ReadContent(full)
	if err != nil {
		fp.stats.FilesErrors++
		return fmt.Errorf("ler %s: %w", relPath, err)
	}
	lang, ok := DetectLanguage(full, content)
	if !ok {
		// Não suportado → silencioso (não é erro).
		fp.stats.FilesSkipped++
		return nil
	}

	// Avalia cache
	res, err := fp.cache.Evaluate(ctx, relPath, content, nil)
	if err != nil {
		fp.stats.FilesErrors++
		return fmt.Errorf("avaliar cache %s: %w", relPath, err)
	}
	switch res.Status {
	case DecodeSkipParse:
		fp.stats.FilesSkipped++
		return nil
	case DecodeSkipDownstream:
		// Atualiza content_hash mas mantém ast_hash antigo.
		_, err := fp.cache.UpdateAfterIndex(ctx, relPath, lang, content, nil, time.Now().Unix())
		if err != nil {
			fp.stats.FilesErrors++
			return err
		}
		fp.stats.FilesSkipped++
		return nil
	}

	// DecodeFirstIndex ou DecodeFullReparse → parsear
	parsed, err := fp.parserFor(lang).ParseFile(relPath, content)
	if err != nil {
		// Erros do backend tree-sitter (ex: ErrTreesitterDisabled): ainda
		// tenta persistir como "first index" vazio para evitar reprocessar.
		if errors.Is(err, ErrTreesitterDisabled) {
			_, uerr := fp.cache.UpdateAfterIndex(ctx, relPath, lang, content, nil, time.Now().Unix())
			if uerr != nil {
				fp.stats.FilesErrors++
				return uerr
			}
			fp.stats.FilesIndexed++
			return nil
		}
		fp.stats.FilesErrors++
		return fmt.Errorf("ParseFile %s: %w", relPath, err)
	}
	if parsed == nil {
		fp.stats.FilesSkipped++
		return nil
	}

	symbols := parsed.Symbols
	fileID, err := fp.cache.UpdateAfterIndex(ctx, relPath, parsed.Language, content, symbols, time.Now().Unix())
	if err != nil {
		fp.stats.FilesErrors++
		return err
	}

	ids, err := fp.cache.PersistSymbols(ctx, fileID, symbols)
	if err != nil {
		fp.stats.FilesErrors++
		return err
	}

	// Arestas estruturais: o backend Go entrega `calls` com o PONTO DE
	// CHAMADA. A varredida por nome (legado) NÃO roda nesse caso — ela
	// duplicaria as arestas do parser e não teria call-site.
	//
	// Quando o parser não devolve arestas (linguagens sem suporte
	// estrutural), o comportamento legado continua idêntico: as arestas
	// `references` por nome seguem valendo.
	if len(parsed.Edges) > 0 {
		fp.collectEdges(ctx, parsed.Edges, symbols, ids, fileID)
	} else if e := resolveAndInsertEdges(ctx, fp.db, symbols, ids, relPath, fileID); e != nil {
		// Não falhamos o pipeline inteiro por causa das edges — logamos.
		fmt.Fprintf(os.Stderr, "[code-pipeline] %s: edges: %v\n", relPath, e)
	}

	fp.stats.FilesIndexed++
	fp.stats.SymbolsTotal += len(symbols)
	return nil
}

// collectEdges transforma as arestas do backend em arestas pendentes,
// ligando cada uma ao id do símbolo ORIGEM (que é sempre um símbolo do
// arquivo recém-indexado).
func (fp *FileProcessor) collectEdges(ctx context.Context, edges []Edge, symbols []Symbol, ids []int64, fileID int64) {
	byQualified := make(map[string]int64, len(symbols))
	for i, s := range symbols {
		if i >= len(ids) {
			break
		}
		key := s.QualifiedName
		if key == "" {
			key = s.Name
		}
		if _, exists := byQualified[key]; !exists {
			byQualified[key] = ids[i]
		}
	}

	for _, e := range edges {
		srcID, ok := byQualified[e.SrcQualifiedName]
		if !ok {
			// A origem não é um símbolo deste arquivo: sem src_symbol_id a
			// aresta não pode ser gravada.
			continue
		}
		pe := pendingEdge{
			kind:             e.Kind,
			srcSymbolID:      srcID,
			dstName:          e.DstName,
			dstQualifiedName: e.DstQualifiedName,
			fileID:           fileID,
			line:             e.Line,
			column:           e.Column,
			reason:           e.Reason,
		}
		if fp.batched {
			fp.pending = append(fp.pending, pe)
			continue
		}
		if err := fp.resolveEdge(ctx, pe, newEdgeResolver(ctx, fp.db)); err != nil {
			fmt.Fprintf(os.Stderr, "[code-pipeline] aresta %s: %v\n", e.Kind, err)
		}
	}
}

// FlushEdges resolve as arestas acumuladas no lote.
//
// A resolução acontece DEPOIS que todos os arquivos foram indexados: assim
// uma chamada para um símbolo definido em um arquivo ainda não processado
// resolve pelo qualified name (EXTRACTED) em vez de cair em "não resolvido".
func (fp *FileProcessor) FlushEdges(ctx context.Context) error {
	if len(fp.pending) == 0 {
		return nil
	}
	pending := fp.pending
	fp.pending = nil
	resolver := newEdgeResolver(ctx, fp.db)
	for _, pe := range pending {
		if err := fp.resolveEdge(ctx, pe, resolver); err != nil {
			return err
		}
	}
	return nil
}

// edgeResolver resolve destinos de aresta com cache por destino: o mesmo
// destino é procurado uma única vez por lote.
type edgeResolver struct {
	ctx   context.Context
	db    *sql.DB
	cache map[string][]int64
}

func newEdgeResolver(ctx context.Context, database *sql.DB) *edgeResolver {
	return &edgeResolver{ctx: ctx, db: database, cache: make(map[string][]int64)}
}

// lookup devolve os ids dos símbolos que casam com a coluna (qualified_name
// ou name) no repositório do contexto. Devolve no máximo 3 ids (o bastante
// para distinguir 0, 1 e "mais de um").
func (r *edgeResolver) lookup(column, value string) ([]int64, error) {
	key := column + "=" + value
	if ids, ok := r.cache[key]; ok {
		return ids, nil
	}
	query := `SELECT id FROM code_symbols WHERE repository = ? AND ` + column + ` = ? ORDER BY id LIMIT 3`
	rows, err := r.db.QueryContext(r.ctx, db.DetectDialect(r.ctx, r.db).Bind(query),
		db.RepositoryFromContext(r.ctx), value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.cache[key] = ids
	return ids, nil
}

// resolveEdge resolve UMA aresta nos três níveis de resolução e grava o
// resultado:
//
//  1. escopo de pacote (qualified name bate e é único) → EXTRACTED
//  2. só por nome (único no repositório)              → INFERRED
//  3. mais de um candidato ou nada resolvido         → code_edges_uncertain
//
// O nível 3 NUNCA promove o candidato a aresta resolvida.
func (fp *FileProcessor) resolveEdge(ctx context.Context, pe pendingEdge, resolver *edgeResolver) error {
	// Nível 1: escopo de pacote.
	if pe.dstQualifiedName != "" {
		ids, err := resolver.lookup("qualified_name", pe.dstQualifiedName)
		if err != nil {
			return err
		}
		if len(ids) == 1 {
			return db.UpsertCodeEdgeWithStatus(ctx, fp.db, pe.srcSymbolID, ids[0], pe.fileID,
				string(pe.kind), pe.line, pe.line, 1.0, db.EpistemicExtracted)
		}
		if len(ids) > 1 {
			return fp.insertUncertain(ctx, pe, fmt.Sprintf("ambíguo: %d símbolos com o qualified name %q",
				len(ids), pe.dstQualifiedName), 0.5)
		}
	}

	// Nível 2: só por nome (único no repositório).
	if pe.dstName != "" {
		ids, err := resolver.lookup("name", pe.dstName)
		if err != nil {
			return err
		}
		if len(ids) == 1 {
			return db.UpsertCodeEdgeWithStatus(ctx, fp.db, pe.srcSymbolID, ids[0], pe.fileID,
				string(pe.kind), pe.line, pe.line, 0.8, db.EpistemicInferred)
		}
		if len(ids) > 1 {
			return fp.insertUncertain(ctx, pe, fmt.Sprintf("ambíguo: %d símbolos com o nome %q",
				len(ids), pe.dstName), 0.5)
		}
	}

	// Nível 3: não resolvido — vai para o bucket de incertas (nunca vira
	// aresta resolvida).
	reason := pe.reason
	if reason == "" {
		reason = fmt.Sprintf("destino não encontrado no repositório: %q", pe.dstName)
	}
	return fp.insertUncertain(ctx, pe, reason, 0.3)
}

// insertUncertain grava a aresta no bucket de incertas.
func (fp *FileProcessor) insertUncertain(ctx context.Context, pe pendingEdge, reason string, confidence float64) error {
	candidate := pe.dstQualifiedName
	if candidate == "" {
		candidate = pe.dstName
	}
	return db.InsertCodeEdgeUncertain(ctx, fp.db, pe.srcSymbolID, pe.fileID, candidate,
		string(pe.kind), reason, confidence, pe.line)
}

// resolveAndInsertEdges faz a varredura simples para v1 (ADR-047 §CA-01 —
// "resolução literal de referências").
//
// É o CAMINHO LEGADO: só roda quando o parser NÃO devolveu arestas
// estruturais (linguagens sem backend AST). Quando o parser devolve arestas
// (backend Go), a varredida por nome NÃO roda — ela duplicaria as arestas
// extraídas e não carregaria o call-site.
//
// Política por symbol name:
//   - 0 matches → sem edge.
//   - 1 match entre todos os code_symbols → edge com confidence = 0.8.
//   - ≥2 matches → recorda em code_edges_uncertain com confidence = 0.5.
//
// A provenance das arestas `references` é INFERRED: elas foram resolvidas por
// NOME, não pela estrutura do código.
func resolveAndInsertEdges(ctx context.Context, database *sql.DB, symbols []Symbol, ids []int64, srcPath string, fileID int64) error {
	if len(symbols) == 0 || len(symbols) != len(ids) || database == nil {
		return nil
	}

	// Carrega todos os symbols agrupados por name (uma query só).
	rows, err := database.QueryContext(ctx,
		`SELECT id, qualified_name, name, file_id FROM code_symbols`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	byName := make(map[string][]int64)
	byNameFileID := make(map[int64]string)
	for rows.Next() {
		var id int64
		var qn sql.NullString
		var name string
		var fid int64
		if err := rows.Scan(&id, &qn, &name, &fid); err != nil {
			continue
		}
		byNameFileID[fid] = ""
		key := name
		if qn.Valid && qn.String != "" {
			key = qn.String
		}
		byName[key] = append(byName[key], id)
	}

	// Para cada symbol do source, procura match.
	for i, s := range symbols {
		if s.Kind == KindImport {
			continue // imports tratados separadamente em T+ (parser mais rico)
		}
		key := s.Name
		if s.QualifiedName != "" {
			key = s.QualifiedName
		}
		candidates := byName[key]
		// Remove self-match (mesmo id/mesma definição).
		filtered := candidates[:0]
		for _, c := range candidates {
			if c != ids[i] {
				filtered = append(filtered, c)
			}
		}
		candidates = filtered
		if len(candidates) == 0 {
			continue
		}
		if len(candidates) == 1 {
			if err := db.UpsertCodeEdgeWithStatus(ctx, database, ids[i], candidates[0], fileID, "references",
				s.StartLine, s.EndLine, 0.8, db.EpistemicInferred); err != nil {
				return err
			}
			continue
		}
		// Múltiplos matches: vira uncertain (confidence < 0.5)
		for _, c := range candidates {
			_ = c
		}
		if err := db.InsertCodeEdgeUncertain(ctx, database, ids[i], fileID, key, "references",
			fmt.Sprintf("ambíguo: %d matches", len(candidates)),
			0.5, s.StartLine); err != nil {
			return err
		}
	}
	return nil
}

// --- escopo / lista de arquivos ------------------------------------------------

// ListCodeFiles retorna a lista de paths relativos (normalizados com forward slash)
// que devem ser parseados pelo pipeline.
//
// Regras:
//   - Ignora DefaultExcludedDirs (config.go) sempre.
//   - Filtra pelo **escopo de código** (`code.include`/`code.exclude`) e NUNCA
//     por `cfg.ShouldIndex` — aquele é o filtro do pipeline Markdown (ADR-061
//     §4, CG-01). Um `include` global como `**/*.go` não pode escolher o que o
//     code pipeline parseia.
//   - Quando langs != nil, restringe a arquivos cujas extensões batem com AllowList.
//   - Filtra symlinks.
func ListCodeFiles(rootDir string, cfg *config.Config, langs []string) ([]string, error) {
	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}
	// Escopo de código efetivo (independente do escopo de notas).
	scope := cfg.CodeScope()
	allowedExt := map[string]bool{}
	if len(langs) > 0 {
		for _, lang := range langs {
			for _, ext := range LanguageSupportedExtensions(lang) {
				allowedExt[strings.ToLower(strings.TrimPrefix(ext, "."))] = true
			}
		}
	} else {
		for _, ext := range SupportedExtensions() {
			allowedExt[strings.ToLower(strings.TrimPrefix(ext, "."))] = true
		}
	}

	var out []string
	walkErr := filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Skip dirs padrão
			name := d.Name()
			if name == "." || name == ".." {
				return nil
			}
			for _, ex := range config.DefaultExcludedDirs {
				if name == ex {
					return filepath.SkipDir
				}
			}
			// Se for symlink, skip
			if d.Type()&fs.ModeSymlink != 0 {
				return filepath.SkipDir
			}
			return nil
		}
		// É arquivo — normaliza caminho relativo (com forward slashes para
		// consistência com config.Include/Exclude que são Unix-style).
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)

		// Escopo de código (ADR-061 §4): o filtro é o code scope, nunca o
		// ShouldIndex — que é o filtro do pipeline Markdown.
		if !scope.Matches(relSlash) {
			return nil
		}

		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
		if allowedExt != nil && len(allowedExt) > 0 && !allowedExt[ext] {
			return nil
		}
		out = append(out, relSlash)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Strings(out)
	return out, nil
}

func normalizeLangList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	out := make([]string, 0, len(in))
	for _, n := range in {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" || seen[n] {
			continue
		}
		if IsSupportedLanguage(n) {
			seen[n] = true
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
