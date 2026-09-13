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

	"github.com/FelipeMiiller/my-memory/internal/compiler"
	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// runNoteCLI processa os subcomandos 'mem note create' e 'mem note append'
func runNoteCLI(ctx context.Context, emb *embedder.OllamaClient, tq *turboquant.Quantizer, defaultRepo string, args []string) error {
	if len(args) < 1 {
		fmt.Println("Uso: mem note <create|append> [opções] <caminho>")
		fmt.Println("  mem note create [--title \"T\"] [--tags \"t1,t2\"] [--type \"concept\"] [--overwrite] <caminho>")
		fmt.Println("  mem note append --heading \"## Título\" [--content \"...\"] <caminho>")
		return nil
	}

	action := args[0]
	switch action {
	case "create":
		cmd := flag.NewFlagSet("note create", flag.ExitOnError)
		title := cmd.String("title", "", "Título da nota (opcional, derivado do nome se omitido)")
		tagsStr := cmd.String("tags", "", "Lista de tags separadas por vírgula")
		aliasesStr := cmd.String("aliases", "", "Lista de aliases separados por vírgula")
		noteType := cmd.String("type", "concept", "Tipo da nota: 'concept', 'decision', 'summary', 'entity'")
		overwrite := cmd.Bool("overwrite", false, "Sobrescreve a nota se o arquivo já existir")
		content := cmd.String("content", "", "Conteúdo Markdown da nota (se omitido, lê do stdin se houver pipe)")
		dbPath := cmd.String("db", "", "Caminho do arquivo SQLite")
		pgURL := cmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
		targetRepo := cmd.String("repo", "", "Identificador/slug do repositório")
		vaultDir := cmd.String("vault", "", "Raiz do vault (padrão: descoberto via .memory/config.yaml)")
		cmd.Parse(args[1:])

		if cmd.NArg() < 1 {
			fmt.Println("Uso: mem note create [opções] <caminho>")
			return nil
		}
		targetPath := cmd.Arg(0)

		cfg := resolveConfig()
		resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)

		resolvedVault := *vaultDir
		if resolvedVault == "" {
			if cfgPath, err := config.FindConfigFile("."); err == nil {
				resolvedVault = filepath.Dir(cfgPath)
				if filepath.Base(resolvedVault) == ".memory" {
					resolvedVault = filepath.Dir(resolvedVault)
				}
			} else {
				resolvedVault = "."
			}
		}

		body := *content
		if body == "" {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				// Entrada via pipe
				bytesRead, err := io.ReadAll(os.Stdin)
				if err == nil {
					body = string(bytesRead)
				}
			}
		}
		if body == "" {
			body = fmt.Sprintf("Nota criada via `mem note create` para: %s.\n", targetPath)
		}

		var tags []string
		if *tagsStr != "" {
			for _, t := range strings.Split(*tagsStr, ",") {
				if tr := strings.TrimSpace(t); tr != "" {
					tags = append(tags, tr)
				}
			}
		}

		var aliases []string
		if *aliasesStr != "" {
			for _, a := range strings.Split(*aliasesStr, ",") {
				if ar := strings.TrimSpace(a); ar != "" {
					aliases = append(aliases, ar)
				}
			}
		}

		var database *sql.DB
		var pgStore *store.PostgresStore
		var err error

		if resolvedPG != "" {
			pgStore, err = store.NewPostgresStore(resolvedPG)
			if err != nil {
				return fmt.Errorf("erro ao conectar no PostgreSQL: %w", err)
			}
			defer pgStore.Close()
		} else {
			database, err = db.InitDB(resolvedDB)
			if err != nil {
				if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "requires cgo to work") {
					database = nil
				} else {
					return fmt.Errorf("erro ao conectar no SQLite: %w", err)
				}
			} else {
				defer database.Close()
			}
		}

		syncEngine := compiler.NewSyncEngine(database, pgStore, emb, tq, resolvedRepo)

		noteRes, idxRes, err := compiler.WriteAndSyncNote(
			ctx,
			syncEngine,
			resolvedVault,
			targetPath,
			*title,
			body,
			tags,
			aliases,
			*noteType,
			nil,
			*overwrite,
		)
		if err != nil {
			return fmt.Errorf("erro ao criar nota: %w", err)
		}

		fmt.Println("✅ Nota criada e sincronizada com sucesso!")
		fmt.Printf("   Caminho:     %s\n", noteRes.Path)
		fmt.Printf("   Título:      %s\n", noteRes.Title)
		fmt.Printf("   Tipo:        %s\n", noteRes.Type)
		fmt.Printf("   SHA-256:     %s\n", noteRes.SHA256)
		fmt.Printf("   Arestas:     %d conexões\n", noteRes.EdgesCount)
		if idxRes != nil {
			fmt.Printf("   Indexação:   %s (%v)\n", idxRes.Action, idxRes.Duration)
		}
		return nil

	case "append":
		cmd := flag.NewFlagSet("note append", flag.ExitOnError)
		heading := cmd.String("heading", "", "Título da seção sob a qual anexar (obrigatório)")
		content := cmd.String("content", "", "Conteúdo a anexar (ou passe como argumento final)")
		createIfMissing := cmd.Bool("create", true, "Cria o arquivo caso não exista")
		dbPath := cmd.String("db", "", "Caminho do arquivo SQLite")
		pgURL := cmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
		targetRepo := cmd.String("repo", "", "Identificador/slug do repositório")
		vaultDir := cmd.String("vault", "", "Raiz do vault (padrão: auto-detectado)")
		cmd.Parse(args[1:])

		if cmd.NArg() < 1 || *heading == "" {
			fmt.Println("Uso: mem note append --heading \"## Título\" [--content \"...\"] <caminho> [<conteudo>]")
			return nil
		}
		targetPath := cmd.Arg(0)

		body := *content
		if body == "" && cmd.NArg() > 1 {
			body = strings.Join(cmd.Args()[1:], " ")
		}
		if body == "" {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				bytesRead, err := io.ReadAll(os.Stdin)
				if err == nil {
					body = string(bytesRead)
				}
			}
		}
		if body == "" {
			return fmt.Errorf("conteúdo para anexar não pode ser vazio (use --content ou passe como argumento)")
		}

		cfg := resolveConfig()
		resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)

		resolvedVault := *vaultDir
		if resolvedVault == "" {
			if cfgPath, err := config.FindConfigFile("."); err == nil {
				resolvedVault = filepath.Dir(cfgPath)
				if filepath.Base(resolvedVault) == ".memory" {
					resolvedVault = filepath.Dir(resolvedVault)
				}
			} else {
				resolvedVault = "."
			}
		}

		var database *sql.DB
		var pgStore *store.PostgresStore
		var err error

		if resolvedPG != "" {
			pgStore, err = store.NewPostgresStore(resolvedPG)
			if err != nil {
				return fmt.Errorf("erro ao conectar no PostgreSQL: %w", err)
			}
			defer pgStore.Close()
		} else {
			database, err = db.InitDB(resolvedDB)
			if err != nil {
				if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "requires cgo to work") {
					database = nil
				} else {
					return fmt.Errorf("erro ao conectar no SQLite: %w", err)
				}
			} else {
				defer database.Close()
			}
		}

		syncEngine := compiler.NewSyncEngine(database, pgStore, emb, tq, resolvedRepo)

		noteRes, idxRes, err := compiler.AppendAndSyncSection(
			ctx,
			syncEngine,
			resolvedVault,
			targetPath,
			*heading,
			body,
			*createIfMissing,
		)
		if err != nil {
			return fmt.Errorf("erro ao anexar seção: %w", err)
		}

		fmt.Println("📝 Seção anexada e sincronizada com sucesso!")
		fmt.Printf("   Caminho:     %s\n", noteRes.Path)
		fmt.Printf("   Cabeçalho:   %s\n", *heading)
		fmt.Printf("   SHA-256:     %s\n", noteRes.SHA256)
		fmt.Printf("   Arestas:     %d conexões\n", noteRes.EdgesCount)
		if idxRes != nil {
			fmt.Printf("   Indexação:   %s (%v)\n", idxRes.Action, idxRes.Duration)
		}
		return nil

	default:
		fmt.Printf("Ação desconhecida: '%s'. Use 'create' ou 'append'.\n", action)
		return nil
	}
}

// runCompileCLI processa o comando 'mem compile'
func runCompileCLI(ctx context.Context, emb *embedder.OllamaClient, tq *turboquant.Quantizer, defaultRepo string, args []string) error {
	cmd := flag.NewFlagSet("compile", flag.ExitOnError)
	topic := cmd.String("topic", "", "Tópico ou termo de busca para compilar (obrigatório)")
	outPath := cmd.String("out", "", "Caminho relativo de destino da nota gerada (obrigatório)")
	title := cmd.String("title", "", "Título da nota sintetizada (opcional)")
	mode := cmd.String("mode", "hybrid", "Modo de busca: 'hybrid', 'vector' ou 'fts'")
	limit := cmd.Int("limit", 5, "Número máximo de documentos fonte a recuperar")
	tagsStr := cmd.String("tags", "", "Tags adicionais separadas por vírgula")
	overwrite := cmd.Bool("overwrite", false, "Sobrescreve se o arquivo de destino já existir")
	dbPath := cmd.String("db", "", "Caminho do arquivo SQLite")
	pgURL := cmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
	targetRepo := cmd.String("repo", "", "Identificador/slug do repositório")
	vaultDir := cmd.String("vault", "", "Raiz do vault (padrão: auto-detectado)")
	cmd.Parse(args)

	cleanTopic := *topic
	if cleanTopic == "" && cmd.NArg() > 0 {
		cleanTopic = cmd.Arg(0)
	}

	if cleanTopic == "" || *outPath == "" {
		fmt.Println("Uso: mem compile --topic \"<termo>\" --out \"<caminho.md>\" [--title \"...\"] [--limit 5] [--mode hybrid|vector|fts] [--overwrite]")
		return nil
	}

	cfg := resolveConfig()
	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)

	resolvedVault := *vaultDir
	if resolvedVault == "" {
		if cfgPath, err := config.FindConfigFile("."); err == nil {
			resolvedVault = filepath.Dir(cfgPath)
			if filepath.Base(resolvedVault) == ".memory" {
				resolvedVault = filepath.Dir(resolvedVault)
			}
		} else {
			resolvedVault = "."
		}
	}

	var tags []string
	if *tagsStr != "" {
		for _, t := range strings.Split(*tagsStr, ",") {
			if tr := strings.TrimSpace(t); tr != "" {
				tags = append(tags, tr)
			}
		}
	}

	var sources []compiler.CompiledSource
	var database *sql.DB
	var pgStore *store.PostgresStore
	var err error

	fmt.Printf("🔍 Recuperando fontes sobre '%s' (modo: %s, limite: %d)...\n", cleanTopic, *mode, *limit)

	if resolvedPG != "" {
		pgStore, err = store.NewPostgresStore(resolvedPG)
		if err != nil {
			return fmt.Errorf("erro ao conectar no PostgreSQL: %w", err)
		}
		defer pgStore.Close()

		var queryVec []float32
		if *mode != "fts" && emb != nil {
			queryVec, _ = emb.GenerateEmbedding(cleanTopic)
		}

		pgResults, errSearch := pgStore.SearchHybridRRFWithDecay(ctx, resolvedRepo, cleanTopic, queryVec, *limit, 60, store.DefaultDecayOptions())
		if errSearch == nil {
			for _, r := range pgResults {
				sources = append(sources, compiler.CompiledSource{
					DocPath: r.DocumentID,
					Title:   r.DocumentID,
					Content: r.Content,
					Score:   r.Score,
				})
			}
		}
	} else {
		database, err = db.InitDB(resolvedDB)
		if err != nil {
			if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "requires cgo to work") {
				database = nil
			} else {
				return fmt.Errorf("erro ao conectar no SQLite: %w", err)
			}
		} else {
			defer database.Close()
		}

		if database != nil {
			var dbResults []db.SearchResult
			switch *mode {
			case "fts":
				dbResults, _ = db.SearchFTS(ctx, database, cleanTopic, *limit)
			case "vector":
				if emb != nil {
					qVec, embErr := emb.GenerateEmbedding(cleanTopic)
					if embErr == nil {
						dbResults, _ = db.SearchKNN(ctx, database, qVec, *limit)
					}
				}
			default: // hybrid
				var qVec []float32
				if emb != nil {
					qVec, _ = emb.GenerateEmbedding(cleanTopic)
				}
				dbResults, _ = db.SearchHybridRRFWithDecay(ctx, database, tq, cleanTopic, qVec, *limit, 60, false, store.DefaultDecayOptions())
			}

			for _, r := range dbResults {
				sources = append(sources, compiler.CompiledSource{
					DocPath: r.DocumentID,
					Title:   r.DocumentID,
					Content: r.Content,
					Score:   r.Score,
				})
			}
		}
	}

	fmt.Printf("📑 %d fonte(s) recuperada(s). Gerando nota compilada...\n", len(sources))

	syncEngine := compiler.NewSyncEngine(database, pgStore, emb, tq, resolvedRepo)

	noteRes, idxRes, err := compiler.CompileAndSyncTopicNote(
		ctx,
		syncEngine,
		resolvedVault,
		*outPath,
		*title,
		cleanTopic,
		sources,
		tags,
		nil,
		*overwrite,
	)
	if err != nil {
		return fmt.Errorf("erro ao compilar nota: %w", err)
	}

	fmt.Println("🧠 Conhecimento compilado com sucesso!")
	fmt.Printf("   Caminho:     %s\n", noteRes.Path)
	fmt.Printf("   Título:      %s\n", noteRes.Title)
	fmt.Printf("   Fontes:      %d documentos vinculados\n", len(sources))
	fmt.Printf("   Arestas:     %d conexões\n", noteRes.EdgesCount)
	fmt.Printf("   SHA-256:     %s\n", noteRes.SHA256)
	if idxRes != nil {
		fmt.Printf("   Indexação:   %s (%v)\n", idxRes.Action, idxRes.Duration)
	}
	return nil
}
