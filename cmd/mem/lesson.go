package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/FelipeMiiller/my-memory/internal/compiler"
	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/federation"
	"github.com/FelipeMiiller/my-memory/internal/parser"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
	"github.com/FelipeMiiller/my-memory/internal/watcher"
)

// defaultLessonFolder é a pasta, dentro do cofre central, onde as lições
// aprendidas são gravadas. O cofre central é a memória de aprendizagem
// (ADR-040/ADR-061) e as lições ficam separadas das pastas de governança
// (standards/, architecture/, ...) para continuarem navegáveis no Obsidian.
const defaultLessonFolder = "lessons"

// lessonNoteType é o `type` da nota. O cofre central usa a taxonomia
// `category` (resource|memory|skill, ADR-024) e os templates canônicos do
// bootstrap não declaram `type`; "concept" é o default de `mem note` e o
// valor coerente com uma lição (um princípio aprendido).
const lessonNoteType = "concept"

// lessonSummaryLen é o tamanho máximo do summary derivado do corpo quando o
// usuário não informa `--summary` (o mesmo teto do micro-abstract L0).
const lessonSummaryLen = 160

// repeatedTagFlag acumula valores repetidos de uma flag (--tag a --tag b).
type repeatedTagFlag []string

func (f *repeatedTagFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *repeatedTagFlag) Set(v string) error {
	for _, part := range strings.Split(v, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			*f = append(*f, trimmed)
		}
	}
	return nil
}

// lessonValueFlags e lessonBoolFlags descrevem as flags de `mem lesson`.
// O `flag` do Go interrompe a análise no primeiro argumento posicional, e o
// contrato do comando é `mem lesson <título> [flags]` — por isso o título é
// separado das flags antes do parse (splitLessonArgs).
var lessonValueFlags = map[string]bool{
	"title":    true,
	"summary":  true,
	"content":  true,
	"file":     true,
	"tags":     true,
	"tag":      true,
	"folder":   true,
	"db":       true,
	"postgres": true,
	"storage":  true,
	"repo":     true,
}

var lessonBoolFlags = map[string]bool{
	"overwrite": true,
}

// lessonOptions são as opções já normalizadas de `mem lesson`.
type lessonOptions struct {
	Title     string
	Summary   string
	Body      string
	Tags      []string
	Folder    string
	Overwrite bool

	DBPath  string
	PGURL   string
	Storage string
	Repo    string
}

// runLessonCLI processa o comando 'mem lesson': registra uma lição aprendida
// como nota no cofre central (vault central), com `category: memory`,
// `summary` (recuperação L0) e timestamp, indexando-a na hora.
//
// A lição NUNCA é gravada no vault do repositório: o destino é o cofre
// central resolvido pela configuração global (~/.memory/config.yaml,
// campo `central_vault.path`, ADR-040). Sem cofre central resolvido o
// comando falha com erro claro — gravar a lição no lugar errado é pior do
// que não gravar.
func runLessonCLI(ctx context.Context, emb embedder.Embedder, tq *turboquant.Quantizer, defaultRepo string, args []string) error {
	opts, err := parseLessonArgs(args)
	if err != nil {
		return err
	}
	if opts == nil {
		return nil
	}

	noteRes, idxRes, err := writeLesson(ctx, emb, tq, opts)
	if err != nil {
		return err
	}

	fmt.Println("✅ Lição registrada no cofre central!")
	fmt.Printf("   Caminho:     %s\n", noteRes.Path)
	fmt.Printf("   Arquivo:     %s\n", noteRes.AbsPath)
	fmt.Printf("   Título:      %s\n", noteRes.Title)
	fmt.Printf("   Categoria:   memory\n")
	fmt.Printf("   Summary:     %s\n", opts.Summary)
	fmt.Printf("   SHA-256:     %s\n", noteRes.SHA256)
	if idxRes != nil {
		fmt.Printf("   Indexação:   %s (%v)\n", idxRes.Action, idxRes.Duration)
	}
	return nil
}

// parseLessonArgs normaliza os argumentos de `mem lesson`.
func parseLessonArgs(args []string) (*lessonOptions, error) {
	positionalTitle, flagArgs, err := splitLessonArgs(args)
	if err != nil {
		return nil, err
	}

	cmd := flag.NewFlagSet("lesson", flag.ExitOnError)
	titleFlag := cmd.String("title", "", "Título da lição (alternativa ao argumento posicional)")
	summaryFlag := cmd.String("summary", "", "Resumo curto da lição (recuperação L0); se omitido, é derivado do corpo")
	contentFlag := cmd.String("content", "", "Conteúdo Markdown da lição (se omitido, lê de --file ou do stdin)")
	fileFlag := cmd.String("file", "", "Arquivo com o conteúdo Markdown da lição")
	tagsFlag := cmd.String("tags", "", "Lista de tags separadas por vírgula")
	var tagList repeatedTagFlag
	cmd.Var(&tagList, "tag", "Tag adicional (repetível)")
	folderFlag := cmd.String("folder", defaultLessonFolder, "Pasta dentro do cofre central onde a lição é gravada")
	overwriteFlag := cmd.Bool("overwrite", false, "Sobrescreve a nota se o arquivo já existir")
	dbFlag := cmd.String("db", "", "Caminho do arquivo SQLite")
	pgFlag := cmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
	storageFlag := cmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'")
	repoFlag := cmd.String("repo", "", "Identificador/slug do repositório (padrão: repo_central)")
	cmd.Parse(flagArgs)

	title := strings.TrimSpace(*titleFlag)
	if title == "" {
		title = strings.TrimSpace(positionalTitle)
	}
	if title == "" {
		fmt.Println("Uso: mem lesson <título> [--summary \"<resumo>\"] [--content \"<texto>\"] [--file <arquivo>] [--tags \"t1,t2\"] [--tag <tag>] [--folder <pasta>] [--storage=sqlite|postgres] [--db <arquivo>] [--postgres <url>]")
		return nil, fmt.Errorf("título da lição é obrigatório")
	}

	body, err := resolveLessonBody(*contentFlag, *fileFlag)
	if err != nil {
		return nil, err
	}

	tags := make([]string, 0, len(tagList))
	for _, part := range strings.Split(*tagsFlag, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	tags = append(tags, tagList...)

	summary := sanitizeLessonSummary(*summaryFlag)
	if summary == "" {
		summary = sanitizeLessonSummary(parser.ExtractMicroAbstract(body, lessonSummaryLen))
	}
	if summary == "" {
		summary = sanitizeLessonSummary(title)
	}

	folder := strings.TrimSpace(*folderFlag)
	if folder == "" {
		folder = defaultLessonFolder
	}

	return &lessonOptions{
		Title:     title,
		Summary:   summary,
		Body:      body,
		Tags:      tags,
		Folder:    filepath.ToSlash(folder),
		Overwrite: *overwriteFlag,
		DBPath:    strings.TrimSpace(*dbFlag),
		PGURL:     strings.TrimSpace(*pgFlag),
		Storage:   strings.TrimSpace(*storageFlag),
		Repo:      strings.TrimSpace(*repoFlag),
	}, nil
}

// splitLessonArgs separa o título (posicional) das flags, tolerando o título
// antes ou depois delas — o `flag` do Go interrompe a análise no primeiro
// argumento posicional e o contrato do comando é `mem lesson <título> [flags]`.
func splitLessonArgs(args []string) (string, []string, error) {
	var positional []string
	var rest []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" || arg == "--" {
			positional = append(positional, arg)
			continue
		}

		name := strings.TrimLeft(arg, "-")
		if idx := strings.Index(name, "="); idx >= 0 {
			rest = append(rest, arg)
			continue
		}
		if lessonBoolFlags[name] {
			rest = append(rest, arg)
			continue
		}
		if lessonValueFlags[name] {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("flag --%s exige um valor", name)
			}
			i++
			rest = append(rest, arg, args[i])
			continue
		}
		// Flag desconhecida: o flag package decide (uso/erro).
		rest = append(rest, arg)
	}

	if len(positional) > 1 {
		return "", nil, fmt.Errorf("apenas um título é aceito (recebidos %d argumentos posicionais)", len(positional))
	}

	title := ""
	if len(positional) == 1 {
		title = positional[0]
	}
	return title, rest, nil
}

// resolveLessonBody obtém o conteúdo da lição: --file, --content ou stdin.
func resolveLessonBody(content, filePath string) (string, error) {
	content = strings.TrimSpace(content)
	filePath = strings.TrimSpace(filePath)

	if filePath != "" && content != "" {
		return "", fmt.Errorf("--file e --content são mutuamente exclusivos")
	}

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("erro ao ler o arquivo da lição '%s': %w", filePath, err)
		}
		trimmed := strings.TrimSpace(string(data))
		if trimmed == "" {
			return "", fmt.Errorf("arquivo da lição '%s' está vazio", filePath)
		}
		return trimmed, nil
	}

	if content != "" {
		return content, nil
	}

	if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("erro ao ler a lição do stdin: %w", err)
		}
		if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
			return trimmed, nil
		}
	}

	return "", fmt.Errorf("conteúdo da lição é obrigatório (use --content, --file ou stdin)")
}

// sanitizeLessonSummary normaliza o summary para uma única linha.
func sanitizeLessonSummary(summary string) string {
	cleaned := strings.TrimSpace(summary)
	if cleaned == "" {
		return ""
	}
	cleaned = strings.ReplaceAll(cleaned, "\r\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	return strings.TrimSpace(cleaned)
}

// writeLesson grava a lição no cofre central e a indexa na hora.
func writeLesson(ctx context.Context, emb embedder.Embedder, tq *turboquant.Quantizer, opts *lessonOptions) (*compiler.NoteResult, *watcher.IndexResult, error) {
	centralPath, err := resolveCentralVaultPath()
	if err != nil {
		return nil, nil, err
	}

	repo, dbPath, pgURL, err := resolveLessonStorage(opts, centralPath)
	if err != nil {
		return nil, nil, err
	}

	content, err := buildLessonContent(opts.Title, opts.Summary, opts.Body)
	if err != nil {
		return nil, nil, err
	}

	noteRelPath := filepath.ToSlash(filepath.Join(opts.Folder, slugifyLessonTitle(opts.Title)+".md"))

	var database *sql.DB
	var pgStore *store.PostgresStore

	if pgURL != "" {
		pgStore, err = store.NewPostgresStore(pgURL)
		if err != nil {
			return nil, nil, fmt.Errorf("erro ao conectar no PostgreSQL: %w", err)
		}
		defer pgStore.Close()
	} else {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, nil, fmt.Errorf("erro ao criar o diretório do banco '%s': %w", filepath.Dir(dbPath), err)
		}
		database, err = db.InitDB(dbPath)
		if err != nil {
			if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "requires cgo to work") {
				database = nil
			} else {
				return nil, nil, fmt.Errorf("erro ao conectar no SQLite: %w", err)
			}
		} else {
			defer database.Close()
		}
	}

	syncEngine := compiler.NewSyncEngine(database, pgStore, emb, tq, repo)

	noteRes, idxRes, err := compiler.WriteAndSyncNote(
		ctx,
		syncEngine,
		centralPath,
		noteRelPath,
		opts.Title,
		content,
		opts.Tags,
		nil,
		lessonNoteType,
		nil,
		opts.Overwrite,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao registrar a lição: %w", err)
	}

	return noteRes, idxRes, nil
}

// resolveCentralVaultPath resolve a raiz do cofre central pela configuração
// global (~/.memory/config.yaml, campo `central_vault.path` — ADR-040).
//
// Não há fallback para o vault do repositório: sem cofre central resolvido o
// comando falha, porque gravar a lição no vault errado é pior do que não
// gravar.
func resolveCentralVaultPath() (string, error) {
	gcfg, err := config.LoadGlobalConfig()
	if err != nil {
		return "", fmt.Errorf("erro ao carregar a configuração global do my-memory: %w", err)
	}

	if gcfg == nil || strings.TrimSpace(gcfg.CentralVault.Path) == "" {
		globalPath, _ := config.GlobalConfigPath()
		return "", fmt.Errorf("cofre central não configurado: defina 'central_vault.path' em %s (use 'mem setup' para configurar)", globalPath)
	}

	centralPath := config.ExpandPath(gcfg.CentralVault.Path)
	if stat, statErr := os.Stat(centralPath); statErr != nil || !stat.IsDir() {
		return "", fmt.Errorf("cofre central não encontrado em '%s' (verifique 'central_vault.path' na configuração global)", centralPath)
	}

	return centralPath, nil
}

// resolveLessonStorage resolve o storage da lição: sempre o do cofre central
// (repo_central), nunca o do repositório.
func resolveLessonStorage(opts *lessonOptions, centralPath string) (string, string, string, error) {
	gcfg, err := config.LoadGlobalConfig()
	if err != nil {
		return "", "", "", fmt.Errorf("erro ao carregar a configuração global do my-memory: %w", err)
	}

	cfg := &config.Config{}
	if gcfg != nil {
		cfg.Storage = gcfg.Storage
	}

	repo := opts.Repo
	if repo == "" {
		repo = config.CentralRepoID
	}

	centralDB := federation.ResolveCentralSQLitePath(centralPath)
	if opts.DBPath != "" {
		centralDB = opts.DBPath
	}

	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, repo, centralDB, opts.PGURL, config.CentralRepoID)

	resolvedDB, resolvedPG, err = applyStorageOverride(opts.Storage, resolvedDB, resolvedPG)
	if err != nil {
		return "", "", "", err
	}

	return resolvedRepo, resolvedDB, resolvedPG, nil
}

// buildLessonContent monta o Markdown da lição.
//
// O frontmatter é montado com `category: memory` e `summary`; o restante
// (title, tags, aliases, created_at, updated_at) é preenchido pelo mesmo
// construtor de frontmatter que o `mem note` usa — o `mergeFrontmatter` do
// `compiler`, que preserva chaves desconhecidas como `category` e `summary`.
func buildLessonContent(title, summary, body string) (string, error) {
	fmBytes, err := yaml.Marshal(map[string]any{
		"category": "memory",
		"summary":  summary,
	})
	if err != nil {
		return "", fmt.Errorf("erro ao montar o frontmatter da lição: %w", err)
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(fmBytes)
	b.WriteString("---\n")
	b.WriteString("# " + title + "\n\n")
	b.WriteString(body)
	b.WriteString("\n")

	return b.String(), nil
}

// slugifyLessonTitle converte o título da lição em um nome de arquivo seguro.
func slugifyLessonTitle(title string) string {
	var b strings.Builder
	lastDash := false

	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "licao"
	}
	return slug
}
