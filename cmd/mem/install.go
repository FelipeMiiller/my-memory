package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/autowire"
)

// runInstallCLI executa o comando mem install / mem setup direcionando a saída para stdout
func runInstallCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runInstallCommand(ctx, defaultRepo, args, os.Stdout)
}

// runInstallCommand executa o auto-wiring de clientes MCP permitindo injeção de io.Writer para testes
func runInstallCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	installCmd := flag.NewFlagSet("install", flag.ContinueOnError)
	installCmd.SetOutput(out)

	target := installCmd.String("target", "all", "Cliente MCP alvo: 'claude', 'cursor', 'vscode', 'windsurf' ou 'all' (padrão: all)")
	dryRun := installCmd.Bool("dry-run", false, "Simula a instalação exibindo arquivos e payloads sem modificar o disco")
	workspaceOnly := installCmd.Bool("workspace", false, "Instala apenas configurações locais do repositório (.cursor/, .vscode/)")
	globalOnly := installCmd.Bool("global", false, "Instala apenas configurações globais da máquina do usuário")
	force := installCmd.Bool("force", false, "Força atualização mesmo se o servidor já estiver configurado de forma idêntica")
	dbPath := installCmd.String("db", "", "Caminho explícito para o banco de dados SQLite")
	targetRepo := installCmd.String("repo", "", "Slug ou identificador do repositório")
	execOverride := installCmd.String("command", "", "Caminho explícito para o binário executável mem (opcional)")
	rootDir := installCmd.String("dir", ".", "Diretório raiz do repositório para configurações de workspace")

	if err := installCmd.Parse(rearrangeInstallArgs(args)); err != nil {
		return err
	}

	// 1. Determinar escopo
	scope := autowire.ScopeAll
	if *workspaceOnly && !*globalOnly {
		scope = autowire.ScopeWorkspace
	} else if *globalOnly && !*workspaceOnly {
		scope = autowire.ScopeGlobal
	}

	// 2. Resolver executável do my-memory
	commandPath := *execOverride
	if commandPath == "" {
		if exe, err := os.Executable(); err == nil {
			if absExe, err := filepath.Abs(exe); err == nil {
				commandPath = absExe
			} else {
				commandPath = exe
			}
		} else {
			commandPath = "mem"
		}
	}

	// 3. Resolver banco de dados e repositório com auto-scoping de vault
	cfg := resolveConfig()
	resolvedRepo, resolvedDB, _ := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, "", defaultRepo)

	absDB, err := filepath.Abs(resolvedDB)
	if err == nil {
		resolvedDB = absDB
	}

	// 4. Preparar ServerConfig
	serverArgs := []string{"mcp"}
	if resolvedDB != "" {
		serverArgs = append(serverArgs, "--db", resolvedDB)
	}
	if resolvedRepo != "" {
		serverArgs = append(serverArgs, "--repo", resolvedRepo)
	}

	serverCfg := autowire.ServerConfig{
		Command: commandPath,
		Args:    serverArgs,
	}

	// 5. Detectar ferramentas alvo
	targets, err := autowire.DetectClients(*rootDir, *target, scope)
	if err != nil {
		return fmt.Errorf("falha ao detectar clientes MCP: %w", err)
	}

	if len(targets) == 0 {
		fmt.Fprintf(out, "Nenhum cliente MCP compatível encontrado para o filtro '--target %s' e escopo '%s'.\n", *target, scope)
		return nil
	}

	// 6. Executar injeção
	modeStr := "Executando instalação"
	if *dryRun {
		modeStr = "🔍 [DRY-RUN] Simulando instalação (nenhum arquivo será alterado)"
	}
	fmt.Fprintf(out, "=== %s do My-Memory em Clientes MCP ===\n", modeStr)
	fmt.Fprintf(out, "Binário:     %s\n", commandPath)
	fmt.Fprintf(out, "Database:    %s\n", resolvedDB)
	if resolvedRepo != "" {
		fmt.Fprintf(out, "Repository:  %s\n", resolvedRepo)
	}
	fmt.Fprintf(out, "Alvos:       %d ferramenta(s) identificada(s)\n\n", len(targets))

	hasErrors := false
	for _, tgt := range targets {
		report, injectErr := autowire.InjectMCPServer(tgt, "my-memory", serverCfg, *dryRun, *force)
		if injectErr != nil {
			hasErrors = true
			fmt.Fprintf(out, "❌ [ERRO] %s\n   Arquivo: %s\n   Falha:   %v\n\n", tgt.Name, tgt.ConfigPath, injectErr)
			continue
		}

		statusBadge := "✔ [INSTALADO]"
		switch report.Status {
		case autowire.StatusUpdated:
			statusBadge = "🔄 [ATUALIZADO]"
		case autowire.StatusAlreadyUpToDate:
			statusBadge = "⏩ [UP-TO-DATE]"
		case autowire.StatusDryRun:
			statusBadge = "🔎 [SIMULAÇÃO]"
		case autowire.StatusError:
			statusBadge = "❌ [ERRO]"
			hasErrors = true
		}

		fmt.Fprintf(out, "%s %s\n", statusBadge, tgt.Name)
		fmt.Fprintf(out, "   Arquivo: %s\n", tgt.ConfigPath)
		if report.BackupPath != "" {
			fmt.Fprintf(out, "   Backup:  %s\n", report.BackupPath)
		}
		if report.Message != "" {
			fmt.Fprintf(out, "   Status:  %s\n", report.Message)
		}
		if *dryRun && report.Diff != "" {
			fmt.Fprintf(out, "   Conteúdo JSON:\n%s\n", indentText(report.Diff, "     "))
		}
		fmt.Fprintln(out)
	}

	if *dryRun {
		fmt.Fprintf(out, "💡 Para aplicar as alterações de fato, execute novamente sem a flag '--dry-run'.\n")
	} else if !hasErrors {
		fmt.Fprintf(out, "🎉 Sucesso! Reinicie os clientes de IA para carregar o servidor MCP 'my-memory'.\n")
	}

	return nil
}

func rearrangeInstallArgs(args []string) []string {
	var flags []string
	var nonFlags []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") && (arg == "--target" || arg == "-target" ||
				arg == "--db" || arg == "-db" || arg == "--repo" || arg == "-repo" ||
				arg == "--command" || arg == "-command" || arg == "--dir" || arg == "-dir") {
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

func indentText(text, prefix string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
