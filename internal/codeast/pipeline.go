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
	DB        *sql.DB
	Parser    Parser         // se nil, usa NewMockParser (testes e no-op).
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
	if parserImpl == nil {
		parserImpl = NewMockParser(false)
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
}

// NewFileProcessor cria um processador isolado (útil para testes unitários).
func NewFileProcessor(database *sql.DB, cache *CodeFileCache, parser Parser) *FileProcessor {
	return &FileProcessor{db: database, cache: cache, parser: parser, stats: &PipelineStats{}}
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
	parsed, err := fp.parser.ParseFile(relPath, content)
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

	// Cross-file reference resolution (passo leve): varrer todos os símbolos
	// (incluindo os do arquivo atual) por nome e criar edges quando o destino
	// bate por nome único. Heurística simples — v2 pode promover.
	if e := resolveAndInsertEdges(ctx, fp.db, symbols, ids, relPath, fileID); e != nil {
		// Não falhamos o pipeline inteiro por causa das edges — logamos.
		fmt.Fprintf(os.Stderr, "[code-pipeline] %s: edges: %v\n", relPath, e)
	}

	fp.stats.FilesIndexed++
	fp.stats.SymbolsTotal += len(symbols)
	return nil
}

// resolveAndInsertEdges faz a varredura simples para v1 (ADR-047 §CA-01 —
// "resolução literal de referências").
//
// Política por symbol name:
//   - 0 matches → sem edge.
//   - 1 match entre todos os code_symbols → edge com confidence = 1.0.
//   - ≥2 matches → recorda em code_edges_uncertain com confidence = 0.5.
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
			if err := db.UpsertCodeEdge(ctx, database, ids[i], candidates[0], fileID, "references", s.StartLine, s.EndLine, 1.0); err != nil {
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
//   - Aplica cfg.Include/Exclude quando cfg != nil.
//   - Quando langs != nil, restringe a arquivos cujas extensões batem com AllowList.
//   - Filtra symlinks.
func ListCodeFiles(rootDir string, cfg *config.Config, langs []string) ([]string, error) {
	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}
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

		// Config scope (ADR-016): se cfg disse para pular, skip
		if cfg != nil && !cfg.ShouldIndex(relSlash) {
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
