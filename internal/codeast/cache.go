package codeast

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// CodeFileCache implementa o cache incremental do code_pipeline (CA-04 / ADR-047).
//
// Política (ADR-047 §Decision Outcome — item 3 do pipeline):
//  1. Se `content_hash` do arquivo em disco == content_hash armazenado → skip parse.
//  2. Se `content_hash` mudou MAS `ast_hash` (dos símbolos extraídos) é idêntico
//     ao armazenado → atualiza apenas content_hash, skipa embed/downstream.
//     (Cobre mudanças cosméticas: whitespace, comentários, refactor de formatação.)
//  3. Caso contrário, parsed + everything from scratch.
type CodeFileCache struct {
	db *sql.DB
}

// NewCodeFileCache cria um cache conectado ao memory.db (sqlite) descrito em db.go.
func NewCodeFileCache(database *sql.DB) *CodeFileCache {
	return &CodeFileCache{db: database}
}

// DecodeStatus descreve o que o pipeline deve fazer após comparar hashes.
type DecodeStatus int

const (
	// DecodeSkipParse: content_hash idêntico ao armazenado. Skip total.
	DecodeSkipParse DecodeStatus = iota
	// DecodeSkipDownstream: content_hash mudou mas ast_hash idêntico.
	// Atualizar content_hash; pular embed/downstream.
	DecodeSkipDownstream
	// DecodeFullReparse: hashes diferentes. Re-parse + re-index.
	DecodeFullReparse
	// DecodeFirstIndex: arquivo nunca visto antes (path não está no DB).
	DecodeFirstIndex
)

// String devolve a forma legível do status.
func (s DecodeStatus) String() string {
	switch s {
	case DecodeSkipParse:
		return "skip_parse"
	case DecodeSkipDownstream:
		return "skip_downstream"
	case DecodeFullReparse:
		return "full_reparse"
	case DecodeFirstIndex:
		return "first_index"
	default:
		return fmt.Sprintf("DecodeStatus(%d)", int(s))
	}
}

// ComputeContentHash devolve o SHA-256 hex do conteúdo bruto (reuso ADR-010
// via internal/store.CalculateContentHash).
func ComputeContentHash(content []byte) string {
	return HashContent(content)
}

// ComputeAstHash devolve o SHA-256 hex de um slice de Symbols já extraídos.
// Ordem determinística (HashSymbols internal).
func ComputeAstHash(symbols []Symbol) string {
	return HashSymbols(symbols)
}

// DecodeResult é o output completo de Evaluate: status + hashes computados/recuperados.
type DecodeResult struct {
	Status             DecodeStatus
	StoredContentHash  string
	StoredAstHash      string
	CurrentContentHash string
	CurrentAstHash     string
}

// Evaluate aplica a política de cache. Recebe os bytes brutos do arquivo,
// os símbolos já extraídos (pode ser vazio quando o caller ainda não fez parse),
// e o path normalizado (deve ser idêntico ao caminho usado em code_files.path).
//
// Se symbols for nil/empty e o status for DecodeSkipParse, significa que a fase
// de parse pode ser pulada com segurança — não há necessidade de chamar o parser.
func (c *CodeFileCache) Evaluate(ctx context.Context, path string, content []byte, symbols []Symbol) (*DecodeResult, error) {
	if c == nil || c.db == nil {
		return nil, errors.New("CodeFileCache: db == nil")
	}
	if path == "" {
		return nil, errors.New("CodeFileCache: path vazio")
	}

	storedContent, storedAst, ok, err := db.GetCodeFileHash(ctx, c.db, path)
	if err != nil {
		return nil, fmt.Errorf("Evaluate: lookup code_files falhou: %w", err)
	}

	currentContent := ComputeContentHash(content)
	res := &DecodeResult{
		StoredContentHash:  storedContent,
		StoredAstHash:      storedAst,
		CurrentContentHash: currentContent,
	}

	if !ok {
		res.Status = DecodeFirstIndex
		return res, nil
	}

	if storedContent == currentContent {
		res.Status = DecodeSkipParse
		return res, nil
	}

	// content_hash mudou → comparar ast_hash.
	// Se symbols for vazio (caller quer apenas avaliar antes de parsear),
	// devolvemos DecodeFullReparse para forçar o caller a parsear e re-chamar.
	if len(symbols) == 0 {
		res.Status = DecodeFullReparse
		return res, nil
	}

	currentAst := ComputeAstHash(symbols)
	res.CurrentAstHash = currentAst
	if storedAst != "" && storedAst == currentAst {
		res.Status = DecodeSkipDownstream
		return res, nil
	}

	res.Status = DecodeFullReparse
	return res, nil
}

// UpdateAfterIndex persiste (ou atualiza) code_files.path com os novos hashes
// calculados durante uma DecodeFullReparse ou DecodeFirstIndex.
//
// Reutiliza db.UpsertCodeFile (T2) — content_hash e ast_hash ambos versionados
// no mesmo ON CONFLICT(path) DO UPDATE.
func (c *CodeFileCache) UpdateAfterIndex(ctx context.Context, path, language string, content []byte, symbols []Symbol, indexedAtUnix int64) (int64, error) {
	if c == nil || c.db == nil {
		return 0, errors.New("CodeFileCache: db == nil")
	}
	contentHash := ComputeContentHash(content)
	astHash := ComputeAstHash(symbols)
	if astHash == "" {
		// Sem símbolos: mantém ast_hash NULL para sinalizar "nenhum símbolo conhecido".
		astHash = ""
	}
	return db.UpsertCodeFile(ctx, c.db, path, language, contentHash, astHash, int64(len(content)), indexedAtUnix)
}

// PersistSymbols grava os símbolos extraídos em code_symbols e devolve um
// slice de IDs na mesma ordem de symbols.
//
// Caller deve fazer o commit/rollback do tx; esta função usa o db diretamente
// para simplificar (CASO T3 só precisa validar a forma; T4 introduzirá tx).
func (c *CodeFileCache) PersistSymbols(ctx context.Context, fileID int64, symbols []Symbol) ([]int64, error) {
	if c == nil || c.db == nil {
		return nil, errors.New("CodeFileCache: db == nil")
	}
	if len(symbols) == 0 {
		return nil, nil
	}
	ids := make([]int64, 0, len(symbols))
	for _, s := range symbols {
		id, err := db.UpsertCodeSymbol(ctx, c.db, fileID, string(s.Kind), s.Name, s.QualifiedName, s.Signature, s.DocComment, s.StartLine, s.EndLine, s.StartCol, s.EndCol)
		if err != nil {
			return nil, fmt.Errorf("PersistSymbols falhou em %q: %w", s.HashKey(), err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ReadContent é helper para isolar leitura de arquivo (testável).
// Lê todo o conteúdo em memória; arquivos > 50 MB devem ser tratados fora do cache.
func ReadContent(path string) ([]byte, error) {
	return os.ReadFile(path)
}
