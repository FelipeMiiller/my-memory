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
	return runInstallCommandWithEnv(ctx, defaultRepo, args, out, nil)
}

// runInstallCommandWithEnv permite injetar o probe de presença de instalação
// dos clientes (nil = ambiente real da máquina).
//
// É a costura que deixa a suíte hermética: o resultado do comando não depende
// das ferramentas de IA instaladas na máquina que roda o teste.
func runInstallCommandWithEnv(ctx context.Context, defaultRepo string, args []string, out io.Writer, env autowire.SystemEnv) error {
	installCmd := flag.NewFlagSet("install", flag.ContinueOnError)
	installCmd.SetOutput(out)

	target := installCmd.String("target", "all", "Cliente MCP alvo: 'all', 'claude', 'claude-desktop', 'cursor', 'vscode' ou 'windsurf' (padrão: all)")
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

	// 1. Validar as flags: erro de uso sai com exit != 0, nunca em silêncio
	if err := validateInstallFlags(*target, *workspaceOnly, *globalOnly); err != nil {
		return err
	}

	// 2. Determinar escopo
	scope := autowire.ScopeAll
	if *workspaceOnly {
		scope = autowire.ScopeWorkspace
	} else if *globalOnly {
		scope = autowire.ScopeGlobal
	}

	// 3. Resolver executável do my-memory
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

	// 4. Resolver banco de dados e repositório com auto-scoping de vault
	cfg := resolveConfig()
	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, "", defaultRepo)

	// ADR-040: a flag --db explícita vence o Postgres resolvido por env/config.
	// O override é o mesmo do resto da CLI (applyStorageOverride), para que a
	// precedência e o log de decisão sejam os mesmos em todo comando.
	if *dbPath != "" && resolvedPG != "" {
		var overrideErr error
		resolvedDB, resolvedPG, overrideErr = applyStorageOverride("sqlite", resolvedDB, resolvedPG)
		if overrideErr != nil {
			return overrideErr
		}
	}

	if resolvedDB != "" {
		absDB, absErr := filepath.Abs(resolvedDB)
		if absErr == nil {
			resolvedDB = absDB
		}
	}

	// 5. Preparar ServerConfig — o payload declara o ENGINE VENCEDOR (ADR-040)
	engine := "sqlite"
	serverArgs := []string{"mcp"}
	if resolvedPG != "" {
		engine = "postgres"
		// A URL do Postgres NÃO vai para o payload: o config do editor vive
		// dentro do repositório versionado e a URL pode carregar senha. O
		// processo filho resolve a URL pela mesma precedência (flag > env >
		// config) e falha alto se não resolver.
		serverArgs = append(serverArgs, "--storage", "postgres")
	} else if resolvedDB != "" {
		serverArgs = append(serverArgs, "--db", resolvedDB)
	}
	if resolvedRepo != "" {
		serverArgs = append(serverArgs, "--repo", resolvedRepo)
	}

	serverCfg := autowire.ServerConfig{
		Command: commandPath,
		Args:    serverArgs,
	}

	// 6. Detectar ferramentas alvo
	var targets []autowire.ClientTarget
	var err error
	if env != nil {
		targets, err = autowire.DetectClientsWithEnv(*rootDir, *target, scope, env)
	} else {
		targets, err = autowire.DetectClients(*rootDir, *target, scope)
	}
	if err != nil {
		return fmt.Errorf("falha ao detectar clientes MCP: %w", err)
	}

	if len(targets) == 0 {
		fmt.Fprintf(out, "Nenhum cliente MCP compatível encontrado para o filtro '--target %s' e escopo '%s'.\n", *target, scope)
		fmt.Fprintf(out, "ℹ  Nenhuma configuração foi aplicada.\n")
		return nil
	}

	// Só o cliente INSTALADO na máquina é configurado: o auto-wiring escreve a
	// config do editor, e config de ferramenta que o usuário não tem é lixo
	// (no workspace, ainda suja o repositório versionado).
	configurable := 0
	for _, tgt := range targets {
		if tgt.Installed {
			configurable++
		}
	}

	// 7. Executar injeção
	modeStr := "Executando instalação"
	if *dryRun {
		modeStr = "🔍 [DRY-RUN] Simulando instalação (nenhum arquivo será alterado)"
	}
	fmt.Fprintf(out, "=== %s do My-Memory em Clientes MCP ===\n", modeStr)
	fmt.Fprintf(out, "Binário:     %s\n", commandPath)
	fmt.Fprintf(out, "Engine:      %s\n", engine)
	if engine == "sqlite" && resolvedDB != "" {
		fmt.Fprintf(out, "Database:    %s\n", resolvedDB)
	}
	if resolvedRepo != "" {
		fmt.Fprintf(out, "Repository:  %s\n", resolvedRepo)
	}
	fmt.Fprintf(out, "Alvos:       %d ferramenta(s) a configurar\n\n", configurable)

	hasErrors := false
	configured := 0
	for _, tgt := range targets {
		if !tgt.Installed {
			fmt.Fprintf(out, "⏭ [PULADO] %s\n", tgt.Name)
			fmt.Fprintf(out, "   Motivo:    cliente não instalado nesta máquina\n\n")
			continue
		}

		configured++
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

	switch {
	case configured == 0:
		fmt.Fprintf(out, "ℹ  Nenhuma configuração foi aplicada: nenhum cliente detectado está instalado nesta máquina.\n")
	case *dryRun:
		fmt.Fprintf(out, "💡 Para aplicar as alterações de fato, execute novamente sem a flag '--dry-run'.\n")
	case !hasErrors:
		fmt.Fprintf(out, "🎉 Sucesso! Reinicie os clientes de IA para carregar o servidor MCP 'my-memory'.\n")
	}

	return nil
}

// validateInstallFlags rejeita as flags inválidas do `mem install`.
//
// Um `--target` fora do conjunto aceito é **erro de uso**: o comando devolve
// erro (a CLI traduz em exit != 0) e a mensagem lista os valores aceitos.
// Antes ele caía no "Nenhum cliente MCP compatível encontrado" e saía com 0 —
// o usuário lia um erro de digitação como "a máquina está vazia".
//
// `--workspace` com `--global` juntas também é erro: as flags são mutuamente
// exclusivas e a combinação não pode instalar em todos os escopos.
func validateInstallFlags(target string, workspaceOnly, globalOnly bool) error {
	if !autowire.IsAcceptedTarget(target) {
		return fmt.Errorf("--target inválido: %q (valores aceitos: %s)", target, strings.Join(autowire.AcceptedTargets(), ", "))
	}
	if workspaceOnly && globalOnly {
		return fmt.Errorf("--workspace e --global são mutuamente exclusivas: escolha um único escopo")
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
