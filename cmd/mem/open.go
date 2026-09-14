package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/deeplink"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// OpenOutput representa o resultado estruturado em JSON do comando mem open
type OpenOutput struct {
	Node      string             `json:"node"`
	Path      string             `json:"path"`
	App       string             `json:"app"`
	Line      int                `json:"line,omitempty"`
	Links     deeplink.DeepLinks `json:"links"`
	TargetURI string             `json:"target_uri"`
	Opened    bool               `json:"opened"`
}

// runOpenCLI executa o subcomando mem open direcionando a saída padrão para o terminal
func runOpenCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runOpenCommand(ctx, defaultRepo, args, os.Stdout, deeplink.DefaultLauncher)
}

// runOpenCommand executa a resolução e abertura de nós com injeção de dependências para testes
func runOpenCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer, launcher deeplink.Launcher) error {
	openCmd := flag.NewFlagSet("open", flag.ContinueOnError)
	openCmd.SetOutput(out)

	appFlag := openCmd.String("app", "", "Aplicativo alvo: 'obsidian', 'vscode'/'code', 'system'/'file' (padrão: definido em config.yaml ou 'obsidian')")
	lineFlag := openCmd.Int("line", 0, "Número da linha no arquivo para focar o cursor (opcional)")
	jsonOutput := openCmd.Bool("json", false, "Exibe os links e metadados de resolução em formato JSON sem abrir o editor")
	dryRun := openCmd.Bool("dry-run", false, "Simula a abertura, imprimindo a URI e comando sem iniciar o processo")
	dbPath := openCmd.String("db", "", "Caminho do arquivo SQLite")
	pgURL := openCmd.String("postgres", "", "URL de conexão PostgreSQL")
	targetRepo := openCmd.String("repo", "", "Slug ou identificador do repositório")

	if err := openCmd.Parse(rearrangeOpenArgs(args)); err != nil {
		return err
	}

	remaining := openCmd.Args()
	if len(remaining) == 0 {
		return fmt.Errorf("identificador do nó ou caminho do arquivo é obrigatório (ex: mem open <nota_ou_caminho>)")
	}
	targetNode := strings.TrimSpace(remaining[0])

	// 1. Resolução de configuração e repositório
	cfg := resolveConfig()
	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)

	cwd, _ := os.Getwd()
	repoRoot := cwd
	if cfgPath, err := filepath.Abs(resolvedDB); err == nil {
		dir := filepath.Dir(cfgPath)
		if filepath.Base(dir) == ".memory" {
			repoRoot = filepath.Dir(dir)
		}
	}

	// 2. Resolução do caminho do arquivo no disco ou no grafo
	resolvedPath := ""

	// 2.1. Verifica se targetNode já é um arquivo existente no disco
	if stat, err := os.Stat(targetNode); err == nil && !stat.IsDir() {
		resolvedPath = targetNode
	} else if stat, err := os.Stat(filepath.Join(repoRoot, targetNode)); err == nil && !stat.IsDir() {
		resolvedPath = filepath.Join(repoRoot, targetNode)
	}

	// 2.2. Se não encontrado diretamente no disco, resolve via banco de dados
	if resolvedPath == "" {
		if resolvedPG != "" && (*pgURL != "" || (cfg != nil && cfg.Storage.Engine == "postgres")) {
			if pgStore, err := store.NewPostgresStore(resolvedPG); err == nil {
				defer pgStore.Close()
				if canonical, err := pgStore.ResolveNodeCanonicalID(ctx, resolvedRepo, targetNode); err == nil && canonical != "" {
					resolvedPath = canonical
				}
			}
		} else {
			if database, err := db.InitDB(resolvedDB); err == nil {
				defer database.Close()
				if canonical, err := db.ResolveNodeCanonicalID(ctx, database, targetNode); err == nil && canonical != "" {
					resolvedPath = canonical
				}
			}
		}
	}

	// 2.3. Fallback: se parece com caminho .md ou wikilink mas não achou no índice
	if resolvedPath == "" {
		clean := strings.TrimPrefix(targetNode, "[[")
		clean = strings.TrimSuffix(clean, "]]")
		if strings.HasSuffix(clean, ".md") || strings.Contains(clean, "/") || strings.Contains(clean, "\\") {
			resolvedPath = clean
		} else {
			return fmt.Errorf("nó ou arquivo '%s' não encontrado no grafo de memória nem no sistema de arquivos", targetNode)
		}
	}

	// 3. Resolução de preferências do editor e vault
	vaultName := cfg.ResolveObsidianVault(repoRoot)
	selectedApp := strings.TrimSpace(*appFlag)
	if selectedApp == "" {
		selectedApp = cfg.ResolveDefaultApp()
	}

	// 4. Geração de Links Canônicos
	links := deeplink.GenerateLinks(repoRoot, vaultName, resolvedPath, *lineFlag)

	var targetURI string
	switch strings.ToLower(selectedApp) {
	case "vscode", "code":
		targetURI = links.VSCode
	case "system", "file", "default":
		targetURI = links.File
	case "obsidian":
		targetURI = links.Obsidian
	default:
		targetURI = links.Obsidian
	}

	// 5. Execução real da abertura se não for dry-run
	opened := false
	if !*dryRun {
		if launcher == nil {
			launcher = deeplink.DefaultLauncher
		}
		_, err := deeplink.Open(resolvedPath, selectedApp, *lineFlag, repoRoot, vaultName, launcher)
		if err != nil {
			return fmt.Errorf("erro ao abrir nota no editor: %w", err)
		}
		opened = true
	}

	// 6. Tratamento de saída JSON
	if *jsonOutput {
		outObj := OpenOutput{
			Node:      targetNode,
			Path:      resolvedPath,
			App:       selectedApp,
			Line:      *lineFlag,
			Links:     links,
			TargetURI: targetURI,
			Opened:    opened,
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(outObj)
	}

	// 7. Tratamento de modo Dry-Run
	if *dryRun {
		cmdName, cmdArgs := deeplink.BuildOSCommand(targetURI)
		fmt.Fprintf(out, "🔎 [dry-run] Simulação de abertura:\n")
		fmt.Fprintf(out, "   Nó/Arquivo: %s\n", resolvedPath)
		fmt.Fprintf(out, "   App:        %s\n", selectedApp)
		if *lineFlag > 0 {
			fmt.Fprintf(out, "   Linha:      %d\n", *lineFlag)
		}
		fmt.Fprintf(out, "   URI:        %s\n", targetURI)
		fmt.Fprintf(out, "   Comando:    %s %s\n", cmdName, strings.Join(cmdArgs, " "))
		return nil
	}

	fmt.Fprintf(out, "🚀 Abrindo no %s...\n", selectedApp)
	fmt.Fprintf(out, "   Arquivo: %s\n", resolvedPath)
	fmt.Fprintf(out, "   URI:     %s\n", targetURI)
	return nil
}

// rearrangeOpenArgs reorganiza os argumentos permitindo flags posicionadas antes ou após o nó alvo
func rearrangeOpenArgs(args []string) []string {
	var flags []string
	var nonFlags []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") && (arg == "--app" || arg == "-app" ||
				arg == "--line" || arg == "-line" ||
				arg == "--db" || arg == "-db" ||
				arg == "--postgres" || arg == "-postgres" ||
				arg == "--repo" || arg == "-repo") {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					flags = append(flags, args[i+1])
					i++
				}
			}
		} else {
			nonFlags = append(nonFlags, arg)
		}
	}
	return append(flags, nonFlags...)
}
