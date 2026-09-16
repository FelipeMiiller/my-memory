package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/federation"
)

// runSetupCLI executa o assistente de configuração mem setup conectando ao stdin/stdout
func runSetupCLI(args []string) error {
	return runSetupCommand(args, os.Stdin, os.Stdout)
}

// runSetupCommand permite injeção de streams para testes automatizados
func runSetupCommand(args []string, in io.Reader, out io.Writer) error {
	setupCmd := flag.NewFlagSet("setup", flag.ContinueOnError)
	setupCmd.SetOutput(out)

	centralPath := setupCmd.String("central", "", "Caminho da pasta do Cofre Central (Google Drive/OneDrive)")
	engine := setupCmd.String("engine", "", "Motor de banco de dados: 'postgres' ou 'sqlite'")
	pgURL := setupCmd.String("postgres-url", "", "URL de conexão PostgreSQL (com pgvector)")
	mcpPort := setupCmd.Int("mcp-port", 0, "Porta padrão para o servidor MCP HTTP (ex: 8080)")
	nonInteractive := setupCmd.Bool("yes", false, "Executa sem prompts interativos, usando flags e defaults")

	if err := setupCmd.Parse(args); err != nil {
		return err
	}

	globalPath, err := config.GlobalConfigPath()
	if err != nil {
		return fmt.Errorf("falha ao determinar caminho da configuração global: %w", err)
	}

	gcfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("falha ao ler configuração global existente: %w", err)
	}
	if gcfg == nil {
		gcfg = &config.GlobalConfig{
			Version: 1,
			Storage: config.StorageConfig{
				Engine:     "sqlite",
				SQLitePath: "memory.db",
			},
			MCP: config.MCPConfig{
				Port: 8080,
			},
		}
	}

	reader := bufio.NewReader(in)

	fmt.Fprintf(out, "\n🏛️ === Assistente de Configuração Global do My-Memory ===\n")
	fmt.Fprintf(out, "Configuração persistida em: %s\n\n", globalPath)

	// 1. Central Vault
	chosenCentral := *centralPath
	if chosenCentral == "" && !*nonInteractive {
		currentVal := gcfg.CentralVault.Path
		if currentVal == "" {
			currentVal = "~/Google Drive/Meu Drive/KnowledgeVault"
		}
		fmt.Fprintf(out, "1. Caminho do Cofre Central (Google Drive / OneDrive):\n   [Padrão: %s]: ", currentVal)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			chosenCentral = line
		} else {
			chosenCentral = currentVal
		}
	} else if chosenCentral == "" {
		chosenCentral = gcfg.CentralVault.Path
	}
	if chosenCentral != "" {
		gcfg.CentralVault.Path = chosenCentral
	}

	// 2. Storage Engine ("ou tudo PostgreSQL, ou tudo SQLite")
	chosenEngine := strings.ToLower(strings.TrimSpace(*engine))
	if chosenEngine == "" && !*nonInteractive {
		currentEngine := gcfg.Storage.Engine
		if currentEngine == "" {
			currentEngine = "sqlite"
		}
		fmt.Fprintf(out, "\n2. Motor de Banco de Dados ('postgres' ou 'sqlite'):\n   [Padrão: %s]: ", currentEngine)
		line, _ := reader.ReadString('\n')
		line = strings.ToLower(strings.TrimSpace(line))
		if line != "" {
			chosenEngine = line
		} else {
			chosenEngine = currentEngine
		}
	} else if chosenEngine == "" {
		chosenEngine = gcfg.Storage.Engine
		if chosenEngine == "" {
			chosenEngine = "sqlite"
		}
	}
	gcfg.Storage.Engine = chosenEngine

	// 3. PostgreSQL URL
	if chosenEngine == "postgres" {
		chosenPG := *pgURL
		if chosenPG == "" && !*nonInteractive {
			currentPG := gcfg.Storage.PostgresURL
			if currentPG == "" {
				currentPG = "postgres://postgres:postgres@localhost:5432/my_memory?sslmode=disable"
			}
			fmt.Fprintf(out, "\n3. URL de Conexão PostgreSQL:\n   [Padrão: %s]: ", currentPG)
			line, _ := reader.ReadString('\n')
			line = strings.TrimSpace(line)
			if line != "" {
				chosenPG = line
			} else {
				chosenPG = currentPG
			}
		} else if chosenPG == "" {
			chosenPG = gcfg.Storage.PostgresURL
			if chosenPG == "" {
				chosenPG = "postgres://postgres:postgres@localhost:5432/my_memory?sslmode=disable"
			}
		}
		gcfg.Storage.PostgresURL = chosenPG
	}

	// 4. MCP Port
	chosenPort := *mcpPort
	if chosenPort == 0 && !*nonInteractive {
		currentPort := gcfg.MCP.Port
		if currentPort == 0 {
			currentPort = 8080
		}
		fmt.Fprintf(out, "\n4. Porta do Servidor MCP HTTP:\n   [Padrão: %d]: ", currentPort)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			var p int
			if _, err := fmt.Sscanf(line, "%d", &p); err == nil && p > 0 {
				chosenPort = p
			}
		}
		if chosenPort == 0 {
			chosenPort = currentPort
		}
	} else if chosenPort == 0 {
		chosenPort = gcfg.MCP.Port
		if chosenPort == 0 {
			chosenPort = 8080
		}
	}
	gcfg.MCP.Port = chosenPort

	// Salva a configuração global
	if err := config.SaveGlobalConfig(gcfg); err != nil {
		return fmt.Errorf("erro ao salvar configuração global: %w", err)
	}

	fmt.Fprintf(out, "\n✅ Configuração global salva com sucesso em: %s\n", globalPath)

	// Se o cofre central estiver definido, verifica se necessita de auto-bootstrap
	if gcfg.CentralVault.Path != "" {
		expandedCentral := config.ExpandPath(gcfg.CentralVault.Path)
		if !federation.IsCentralVaultInitialized(expandedCentral) {
			fmt.Fprintf(out, "🚀 Inicializando estrutura canônica no Cofre Central (%s)...\n", expandedCentral)
			if err := federation.BootstrapCentralVault(expandedCentral); err != nil {
				fmt.Fprintf(out, "⚠️ Aviso no bootstrap do cofre central: %v\n", err)
			} else {
				fmt.Fprintf(out, "✨ Cofre Central inicializado com sucesso com 11 pastas canônicas e templates!\n")
			}
		} else {
			fmt.Fprintf(out, "ℹ️ Cofre Central em '%s' já inicializado e ativo.\n", expandedCentral)
		}
	}

	fmt.Fprintln(out)
	return nil
}

// runCentralCLI executa subcomandos relacionados ao cofre central (mem central status | bootstrap)
func runCentralCLI(args []string) error {
	return runCentralCommand(args, os.Stdout)
}

// runCentralCommand gerencia operações no cofre central
func runCentralCommand(args []string, out io.Writer) error {
	if len(args) < 1 {
		fmt.Fprintln(out, "Uso: mem central <status|bootstrap> [opções]")
		return nil
	}

	action := strings.ToLower(args[0])
	subArgs := args[1:]

	gcfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração global: %w", err)
	}

	var centralPath string
	if gcfg != nil && gcfg.CentralVault.Path != "" {
		centralPath = config.ExpandPath(gcfg.CentralVault.Path)
	}

	switch action {
	case "status":
		fmt.Fprintf(out, "\n=== Status do Cofre Central de Conhecimento (Global Brain) ===\n")
		if centralPath == "" {
			fmt.Fprintln(out, "Status: ⚠️ Não configurado. Execute 'mem setup' para vincular uma pasta no Google Drive/OneDrive.")
			return nil
		}

		fmt.Fprintf(out, "Caminho: %s\n", centralPath)
		if stat, err := os.Stat(centralPath); err != nil || !stat.IsDir() {
			fmt.Fprintf(out, "Status: ⚠️ Inacessível ou pasta não encontrada no host (%v)\n", err)
			return nil
		}

		initialized := federation.IsCentralVaultInitialized(centralPath)
		if !initialized {
			fmt.Fprintln(out, "Status: ⚠️ Pasta presente, mas cofre central virgem (não inicializado).")
			fmt.Fprintln(out, "Execute 'mem central bootstrap' para criar a estrutura canônica.")
			return nil
		}

		fmt.Fprintln(out, "Status: ✅ Inicializado e Conectado")
		sqlitePath := federation.ResolveCentralSQLitePath(centralPath)
		if stat, err := os.Stat(sqlitePath); err == nil {
			fmt.Fprintf(out, "Banco SQLite Central: %s (tamanho: %d bytes)\n", sqlitePath, stat.Size())
		}
		fmt.Fprintln(out)

	case "bootstrap":
		bootCmd := flag.NewFlagSet("central-bootstrap", flag.ContinueOnError)
		bootCmd.SetOutput(out)
		targetPath := bootCmd.String("path", centralPath, "Caminho do cofre a ser inicializado")
		if err := bootCmd.Parse(subArgs); err != nil {
			return err
		}

		chosenPath := *targetPath
		if chosenPath == "" && bootCmd.NArg() > 0 {
			chosenPath = bootCmd.Arg(0)
		}
		if chosenPath == "" {
			return fmt.Errorf("nenhum caminho informado para bootstrap do cofre central. Use: mem central bootstrap <caminho>")
		}

		expanded := config.ExpandPath(chosenPath)
		fmt.Fprintf(out, "Iniciando Auto-Bootstrap do Cofre Central em: %s...\n", expanded)
		if err := federation.BootstrapCentralVault(expanded); err != nil {
			return fmt.Errorf("erro no bootstrap do cofre central: %w", err)
		}
		fmt.Fprintln(out, "✅ Cofre Central inicializado com sucesso com 11 pastas canônicas, templates e isolamento.")

	default:
		return fmt.Errorf("ação desconhecida: '%s'. Use 'status' ou 'bootstrap'", action)
	}

	return nil
}

// runReposCLI lista todos os repositórios registrados no catálogo global (~/.memory/config.yaml)
func runReposCLI(args []string) error {
	return runReposCommand(args, os.Stdout)
}

// runReposCommand executa a listagem com injeção de saída
func runReposCommand(args []string, out io.Writer) error {
	gcfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração global: %w", err)
	}

	globalPath, _ := config.GlobalConfigPath()
	fmt.Fprintf(out, "\n=== Catálogo de Repositórios Registrados no My-Memory ===\n")
	fmt.Fprintf(out, "Arquivo Global: %s\n\n", globalPath)

	if gcfg == nil || len(gcfg.Repositories) == 0 {
		fmt.Fprintln(out, "Nenhum repositório registrado no catálogo ainda.")
		fmt.Fprintln(out, "Execute 'mem init' dentro de qualquer projeto para registrá-lo automaticamente.")
		return nil
	}

	fmt.Fprintf(out, "%-20s | %-25s | %s\n", "REPO ID", "NOME", "CAMINHO NO DISCO")
	fmt.Fprintf(out, "%s\n", strings.Repeat("-", 75))
	for _, repo := range gcfg.Repositories {
		status := "✅"
		memDir := filepath.Join(repo.Path, ".memory")
		if _, err := os.Stat(memDir); err != nil {
			status = "⚠️"
		}
		name := repo.Name
		if name == "" {
			name = "-"
		}
		fmt.Fprintf(out, "%-20s | %-25s | %s %s\n", repo.ID, name, status, repo.Path)
	}
	fmt.Fprintln(out)
	return nil
}
