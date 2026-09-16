package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Constantes de federação e identidade
const (
	CentralRepoID      = "repo_central"
	GlobalConfigDirEnv = "MY_MEMORY_GLOBAL_CONFIG_DIR"
)

// CentralVaultConfig define os parâmetros de vinculação ao cofre central de conhecimento
type CentralVaultConfig struct {
	Path     string   `yaml:"path,omitempty" json:"path,omitempty"`           // Caminho do cofre central (ex: "~/Google Drive/Vault" ou "${CENTRAL_VAULT}")
	ReadOnly bool     `yaml:"read_only,omitempty" json:"read_only,omitempty"` // Se true, o cofre central é tratado como somente-leitura
	Include  []string `yaml:"include,omitempty" json:"include,omitempty"`     // Padrões glob específicos a indexar do cofre central
}

// MCPConfig define preferências para o servidor Model Context Protocol (MCP)
type MCPConfig struct {
	Port int `yaml:"port,omitempty" json:"port,omitempty"` // Porta padrão de execução do servidor MCP (default: 8080)
}

// RepositoryCatalogEntry registra um repositório satélite conhecido no catálogo global
type RepositoryCatalogEntry struct {
	ID      string         `yaml:"id" json:"id"`                               // Identificador imutável repo_<12-hex-chars>
	Path    string         `yaml:"path" json:"path"`                           // Caminho absoluto para a raiz do repositório no host
	Name    string         `yaml:"name,omitempty" json:"name,omitempty"`       // Nome amigável ou slug (ex: "owner/repo")
	Storage *StorageConfig `yaml:"storage,omitempty" json:"storage,omitempty"` // Configurações de persistência específicas deste repositório
}

// GlobalConfig define o formato de configuração global do usuário (~/.memory/config.yaml)
type GlobalConfig struct {
	Version      int                      `yaml:"version" json:"version"`
	CentralVault CentralVaultConfig       `yaml:"central_vault,omitempty" json:"central_vault,omitempty"`
	Storage      StorageConfig            `yaml:"storage,omitempty" json:"storage,omitempty"`
	MCP          MCPConfig                `yaml:"mcp,omitempty" json:"mcp,omitempty"`
	Repositories []RepositoryCatalogEntry `yaml:"repositories,omitempty" json:"repositories,omitempty"` // Catálogo de repositórios registrados
}

// StorageConfig define os parâmetros da camada de persistência
type StorageConfig struct {
	Engine      string `yaml:"engine,omitempty" json:"engine,omitempty"`             // "sqlite" (default) ou "postgres"
	SQLitePath  string `yaml:"sqlite_path,omitempty" json:"sqlite_path,omitempty"`   // Caminho do banco local (default: "memory.db")
	PostgresURL string `yaml:"postgres_url,omitempty" json:"postgres_url,omitempty"` // URL de conexão PostgreSQL com pgvector
}

// EmbeddingConfig define os parâmetros do modelo de representação vetorial
type EmbeddingConfig struct {
	Provider  string `yaml:"provider,omitempty" json:"provider,omitempty"`   // "ollama" (default)
	Model     string `yaml:"model,omitempty" json:"model,omitempty"`         // "nomic-embed-text" (default)
	URL       string `yaml:"url,omitempty" json:"url,omitempty"`             // Endpoint do servidor (default: "http://localhost:11434")
	Dimension int    `yaml:"dimension,omitempty" json:"dimension,omitempty"` // Dimensão dos vetores (default: 768)
}

// SearchConfig define as preferências padrão de busca no vault
type SearchConfig struct {
	Mode        string  `yaml:"mode,omitempty" json:"mode,omitempty"`                 // "hybrid", "vector", "fts" (default: "hybrid")
	Limit       int     `yaml:"limit,omitempty" json:"limit,omitempty"`               // Número padrão de resultados (default: 5)
	K           int     `yaml:"k,omitempty" json:"k,omitempty"`                       // Constante RRF (default: 60)
	Decay       bool    `yaml:"decay,omitempty" json:"decay,omitempty"`               // Ativa decaimento temporal (default: false)
	HalfLife    float64 `yaml:"half_life,omitempty" json:"half_life,omitempty"`       // Meia-vida em dias (default: 30.0)
	DecayWeight float64 `yaml:"decay_weight,omitempty" json:"decay_weight,omitempty"` // Peso do decaimento w (default: 0.3)
	UseTurbo    bool    `yaml:"use_turbo,omitempty" json:"use_turbo,omitempty"`       // Usar quantização 4-bit TurboQuant no SQLite
	Level       string  `yaml:"level,omitempty" json:"level,omitempty"`               // "l0", "l1", "l2" (default: "l1")
	Category    string  `yaml:"category,omitempty" json:"category,omitempty"`         // "resource", "memory", "skill" ou ""
}

// WatcherConfig define os parâmetros para monitoramento contínuo de arquivos em tempo real
type WatcherConfig struct {
	DebounceMs int `yaml:"debounce_ms,omitempty" json:"debounce_ms,omitempty"` // Janela de debounce em milissegundos (default: 500)
	IntervalMs int `yaml:"interval_ms,omitempty" json:"interval_ms,omitempty"` // Intervalo de polling em milissegundos (default: 1000)
}

// EditorConfig define as preferências de integração e navegação em editores externos
type EditorConfig struct {
	DefaultApp    string `yaml:"default_app,omitempty" json:"default_app,omitempty"`       // "obsidian", "vscode", "system" (default: "obsidian")
	ObsidianVault string `yaml:"obsidian_vault,omitempty" json:"obsidian_vault,omitempty"` // Nome customizado do vault do Obsidian (opcional)
}

// Config estrutura raiz de configuração declarativa do vault / repositório
type Config struct {
	Version      int                `yaml:"version" json:"version"`
	RepoID       string             `yaml:"repo_id,omitempty" json:"repo_id,omitempty"`             // ID criptográfico imutável do repositório (ex: "repo_a1b2c3d4e5f6")
	Repository   string             `yaml:"repository,omitempty" json:"repository,omitempty"`       // Slug do repositório (ex: "owner/repo")
	VaultName    string             `yaml:"vault_name,omitempty" json:"vault_name,omitempty"`       // Nome amigável do vault de notas
	Include      []string           `yaml:"include,omitempty" json:"include,omitempty"`             // Padrões glob de arquivos a indexar
	Exclude      []string           `yaml:"exclude,omitempty" json:"exclude,omitempty"`             // Padrões glob de pastas/arquivos a ignorar
	CentralVault CentralVaultConfig `yaml:"central_vault,omitempty" json:"central_vault,omitempty"` // Configuração de integração com o cofre central
	Storage      StorageConfig      `yaml:"storage,omitempty" json:"storage,omitempty"`
	Embedding    EmbeddingConfig    `yaml:"embedding,omitempty" json:"embedding,omitempty"`
	Search       SearchConfig       `yaml:"search,omitempty" json:"search,omitempty"`
	Watcher      WatcherConfig      `yaml:"watcher,omitempty" json:"watcher,omitempty"`
	Editor       EditorConfig       `yaml:"editor,omitempty" json:"editor,omitempty"`
	MCP          MCPConfig          `yaml:"mcp,omitempty" json:"mcp,omitempty"`
}

// DefaultConfig retorna as configurações padrão do My-Memory
func DefaultConfig() Config {
	return Config{
		Version: 1,
		Include: []string{
			"**/*.md",
		},
		Exclude: []string{
			".git/**",
			"node_modules/**",
			"vendor/**",
			".obsidian/**",
			".trash/**",
			".memory/**",
		},
		Storage: StorageConfig{
			Engine:     "sqlite",
			SQLitePath: "memory.db",
		},
		Embedding: EmbeddingConfig{
			Provider:  "ollama",
			Model:     "nomic-embed-text",
			URL:       "http://localhost:11434",
			Dimension: 768,
		},
		Search: SearchConfig{
			Mode:        "hybrid",
			Limit:       5,
			K:           60,
			Decay:       false,
			HalfLife:    30.0,
			DecayWeight: 0.3,
			UseTurbo:    false,
			Level:       "l1",
			Category:    "",
		},
		Watcher: WatcherConfig{
			DebounceMs: 500,
			IntervalMs: 1000,
		},
		Editor: EditorConfig{
			DefaultApp: "obsidian",
		},
		MCP: MCPConfig{
			Port: 8080,
		},
	}
}

// ResolveObsidianVault determina o nome do vault do Obsidian a ser usado em deep links
func (c *Config) ResolveObsidianVault(repoRoot string) string {
	if c != nil {
		if strings.TrimSpace(c.Editor.ObsidianVault) != "" {
			return strings.TrimSpace(c.Editor.ObsidianVault)
		}
		if strings.TrimSpace(c.VaultName) != "" {
			return strings.TrimSpace(c.VaultName)
		}
	}
	if repoRoot != "" {
		return filepath.Base(repoRoot)
	}
	return ""
}

// ResolveDefaultApp determina o aplicativo padrão configurado ("obsidian", "vscode", "system")
func (c *Config) ResolveDefaultApp() string {
	if c != nil && strings.TrimSpace(c.Editor.DefaultApp) != "" {
		return strings.ToLower(strings.TrimSpace(c.Editor.DefaultApp))
	}
	return "obsidian"
}

// CandidateConfigFileNames lista em ordem de prioridade os arquivos de configuração procurados
var CandidateConfigFileNames = []string{
	filepath.Join(".memory", "config.yaml"),
	filepath.Join(".memory", "config.yml"),
	filepath.Join(".memory", "config.json"),
	".mem.yaml",
	".mem.yml",
	".mem.json",
}

// FindConfigFile procura recursivamente em startDir e em seus diretórios pais por um arquivo de configuração.
// Interrompe a busca se atingir a raiz do repositório Git (.git) ou a raiz do sistema operacional.
func FindConfigFile(startDir string) (string, error) {
	if startDir == "" || startDir == "." {
		cwd, err := os.Getwd()
		if err == nil {
			startDir = cwd
		}
	}

	absDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	realHome, _ := os.UserHomeDir()
	realHomeGlobalCfg := ""
	if realHome != "" {
		realHomeGlobalCfg = filepath.Join(realHome, ".memory", "config.yaml")
	}
	globalCfgPath, _ := GlobalConfigPath()

	curr := absDir
	for {
		for _, candidate := range CandidateConfigFileNames {
			p := filepath.Join(curr, candidate)
			// Nunca confunde o arquivo de configuração global do usuário (~/.memory/config.yaml) com cofre local de projeto
			if globalCfgPath != "" && filepath.Clean(p) == filepath.Clean(globalCfgPath) {
				continue
			}
			if realHomeGlobalCfg != "" && filepath.Clean(p) == filepath.Clean(realHomeGlobalCfg) {
				continue
			}
			if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
				return p, nil
			}
		}

		// Se encontramos a raiz do Git neste nível e nenhum config foi achado, interrompe para não vazar do repo
		gitDir := filepath.Join(curr, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			break
		}

		parent := filepath.Dir(curr)
		if parent == curr || parent == "" {
			break
		}
		curr = parent
	}

	return "", os.ErrNotExist
}

// LoadConfig carrega a configuração a partir de um arquivo YAML ou JSON mesclando com os defaults
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de configuração '%s': %w", path, err)
	}

	cfg := DefaultConfig()

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("erro ao decodificar JSON de '%s': %w", path, err)
		}
	} else {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("erro ao decodificar YAML de '%s': %w", path, err)
		}
	}

	// Garante que campos essenciais nunca fiquem vazios
	if cfg.Storage.Engine == "" {
		cfg.Storage.Engine = "sqlite"
	}
	if cfg.Storage.SQLitePath == "" {
		cfg.Storage.SQLitePath = "memory.db"
	}
	if cfg.Embedding.Model == "" {
		cfg.Embedding.Model = "nomic-embed-text"
	}
	if cfg.Embedding.URL == "" {
		cfg.Embedding.URL = "http://localhost:11434"
	}
	if cfg.Embedding.Dimension <= 0 {
		cfg.Embedding.Dimension = 768
	}
	if cfg.Search.Mode == "" {
		cfg.Search.Mode = "hybrid"
	}
	if cfg.Search.Limit <= 0 {
		cfg.Search.Limit = 5
	}
	if cfg.Search.K <= 0 {
		cfg.Search.K = 60
	}
	if cfg.Search.HalfLife <= 0 {
		cfg.Search.HalfLife = 30.0
	}
	if cfg.Search.DecayWeight < 0.0 || cfg.Search.DecayWeight > 1.0 {
		cfg.Search.DecayWeight = 0.3
	}

	// Expande variáveis de ambiente (${VAR}) e caminhos com til (~)
	cfg.Storage.SQLitePath = ExpandPath(cfg.Storage.SQLitePath)
	cfg.Storage.PostgresURL = os.ExpandEnv(cfg.Storage.PostgresURL)
	cfg.Embedding.URL = os.ExpandEnv(cfg.Embedding.URL)
	cfg.Repository = os.ExpandEnv(cfg.Repository)
	cfg.CentralVault.Path = ExpandPath(cfg.CentralVault.Path)
	if cfg.MCP.Port <= 0 {
		cfg.MCP.Port = 8080
	}

	return &cfg, nil
}

// SaveConfig grava a configuração no caminho especificado em formato YAML
func SaveConfig(path string, cfg *Config) error {
	if cfg == nil {
		return errors.New("configuração não pode ser nula")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório para config: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("erro ao serializar config para YAML: %w", err)
	}

	header := []byte("# Configuração Declarativa do My-Memory\n# Mais detalhes em docs/REPOSITORY_BRAIN.md\n\n")
	content := append(header, data...)

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("erro ao salvar arquivo de config: %w", err)
	}

	return nil
}

// ExpandPath expande variáveis de ambiente (${VAR} ou $VAR) e til (~) para o diretório home do usuário
func ExpandPath(path string) string {
	if path == "" {
		return ""
	}
	expanded := os.ExpandEnv(path)
	if strings.HasPrefix(expanded, "~/") || strings.HasPrefix(expanded, "~\\") {
		if home, err := os.UserHomeDir(); err == nil {
			expanded = filepath.Join(home, expanded[2:])
		}
	} else if expanded == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			expanded = home
		}
	}
	return expanded
}

// GenerateRepoID gera um identificador único criptográfico no formato "repo_<12-hex-chars>"
func GenerateRepoID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("repo_%012x", time.Now().UnixNano()&0xFFFFFFFFFFFF)
	}
	return fmt.Sprintf("repo_%s", hex.EncodeToString(b))
}

// EnsureRepoID garante que a configuração possui um repo_id válido.
// Se estiver vazio, gera um novo repo_id criptográfico e retorna true.
func EnsureRepoID(cfg *Config) bool {
	if cfg == nil {
		return false
	}
	if strings.TrimSpace(cfg.RepoID) == "" {
		cfg.RepoID = GenerateRepoID()
		return true
	}
	return false
}

// ResolveRepoDatabaseName determina o nome do banco de dados no PostgreSQL para o repositório ou central
func ResolveRepoDatabaseName(repoID string) string {
	if strings.TrimSpace(repoID) == "" || repoID == CentralRepoID {
		return "my_memory_central"
	}
	cleanID := strings.TrimPrefix(repoID, "repo_")
	return fmt.Sprintf("my_memory_repo_%s", cleanID)
}

// UserHomeConfigDir retorna o caminho padrão para o diretório de configuração global (~/.memory)
func UserHomeConfigDir() (string, error) {
	if envDir := os.Getenv(GlobalConfigDirEnv); envDir != "" {
		return envDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("erro ao obter diretório home do usuário: %w", err)
	}
	return filepath.Join(home, ".memory"), nil
}

// GlobalConfigPath retorna o caminho absoluto para o arquivo de configuração global (~/.memory/config.yaml)
func GlobalConfigPath() (string, error) {
	dir, err := UserHomeConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// LoadGlobalConfig carrega a configuração global de ~/.memory/config.yaml se existir.
// Retorna nil, nil caso o arquivo não exista.
func LoadGlobalConfig() (*GlobalConfig, error) {
	path, err := GlobalConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao ler config global em '%s': %w", path, err)
	}

	var gcfg GlobalConfig
	if err := yaml.Unmarshal(data, &gcfg); err != nil {
		return nil, fmt.Errorf("erro ao decodificar YAML de config global '%s': %w", path, err)
	}

	gcfg.CentralVault.Path = ExpandPath(gcfg.CentralVault.Path)
	gcfg.Storage.SQLitePath = ExpandPath(gcfg.Storage.SQLitePath)
	gcfg.Storage.PostgresURL = os.ExpandEnv(gcfg.Storage.PostgresURL)

	return &gcfg, nil
}

// SaveGlobalConfig salva a configuração global em ~/.memory/config.yaml
func SaveGlobalConfig(cfg *GlobalConfig) error {
	if cfg == nil {
		return errors.New("configuração global não pode ser nula")
	}
	path, err := GlobalConfigPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório para config global: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("erro ao serializar config global: %w", err)
	}

	header := []byte("# Configuração Global do My-Memory (~/.memory/config.yaml)\n# Compartilhada entre todos os repositórios locais e central vault\n\n")
	content := append(header, data...)

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("erro ao salvar arquivo de config global: %w", err)
	}
	return nil
}

// RegisterRepositoryInGlobalConfig cadastra ou atualiza um repositório no catálogo de ~/.memory/config.yaml de forma idempotente
func RegisterRepositoryInGlobalConfig(entry RepositoryCatalogEntry) error {
	if strings.TrimSpace(entry.ID) == "" {
		return errors.New("identificador do repositório (id) não pode ser vazio")
	}
	if strings.TrimSpace(entry.Path) == "" {
		return errors.New("caminho do repositório (path) não pode ser vazio")
	}

	gcfg, err := LoadGlobalConfig()
	if err != nil {
		return err
	}
	if gcfg == nil {
		gcfg = &GlobalConfig{
			Version: 1,
			Storage: StorageConfig{
				Engine:     "sqlite",
				SQLitePath: "memory.db",
			},
			MCP: MCPConfig{
				Port: 8080,
			},
		}
	}

	cleanPath, err := filepath.Abs(ExpandPath(entry.Path))
	if err != nil {
		cleanPath = filepath.Clean(ExpandPath(entry.Path))
	}
	entry.Path = cleanPath

	foundIndex := -1
	for i, r := range gcfg.Repositories {
		if r.ID == entry.ID || filepath.Clean(r.Path) == cleanPath {
			foundIndex = i
			break
		}
	}

	if foundIndex >= 0 {
		if entry.Storage == nil && gcfg.Repositories[foundIndex].Storage != nil {
			entry.Storage = gcfg.Repositories[foundIndex].Storage
		}
		gcfg.Repositories[foundIndex] = entry
	} else {
		gcfg.Repositories = append(gcfg.Repositories, entry)
	}

	return SaveGlobalConfig(gcfg)
}

// LoadCascadingConfig carrega a configuração aplicando a hierarquia em cascata:
// 1. Defaults do sistema (DefaultConfig)
// 2. Sobrescrito pela configuração global (~/.memory/config.yaml), se existir
// 3. Sobrescrito pela configuração local (.memory/config.yaml), se existir
// Retorna a configuração resultante, o caminho do arquivo local (se encontrado) e erro.
func LoadCascadingConfig(repoDir string) (*Config, string, error) {
	cfg := DefaultConfig()

	// Camada 2: Configuração Global (~/.memory/config.yaml)
	globalCfg, globalErr := LoadGlobalConfig()
	if globalErr == nil && globalCfg != nil {
		if strings.TrimSpace(globalCfg.CentralVault.Path) != "" {
			cfg.CentralVault = globalCfg.CentralVault
		}
		if globalCfg.Storage.Engine != "" {
			cfg.Storage.Engine = globalCfg.Storage.Engine
		}
		if globalCfg.Storage.SQLitePath != "" {
			cfg.Storage.SQLitePath = globalCfg.Storage.SQLitePath
		}
		if globalCfg.Storage.PostgresURL != "" {
			cfg.Storage.PostgresURL = globalCfg.Storage.PostgresURL
		}
		if globalCfg.MCP.Port > 0 {
			cfg.MCP.Port = globalCfg.MCP.Port
		}
	}

	// Camada 3: Configuração Local (.memory/config.yaml)
	cfgPath, err := FindConfigFile(repoDir)
	var localRepoID string
	if err == nil {
		data, readErr := os.ReadFile(cfgPath)
		if readErr != nil {
			return nil, cfgPath, fmt.Errorf("erro ao ler arquivo local '%s': %w", cfgPath, readErr)
		}

		type rawStorageCheck struct {
			RepoID  string `yaml:"repo_id" json:"repo_id"`
			Storage *struct {
				Engine string `yaml:"engine" json:"engine"`
			} `yaml:"storage" json:"storage"`
		}
		var check rawStorageCheck
		ext := strings.ToLower(filepath.Ext(cfgPath))
		if ext == ".json" {
			_ = json.Unmarshal(data, &check)
			if unmarshalErr := json.Unmarshal(data, &cfg); unmarshalErr != nil {
				return nil, cfgPath, fmt.Errorf("erro ao decodificar JSON de '%s': %w", cfgPath, unmarshalErr)
			}
		} else {
			_ = yaml.Unmarshal(data, &check)
			if unmarshalErr := yaml.Unmarshal(data, &cfg); unmarshalErr != nil {
				return nil, cfgPath, fmt.Errorf("erro ao decodificar YAML de '%s': %w", cfgPath, unmarshalErr)
			}
		}
		localRepoID = check.RepoID

		// Se o repositório local declarar explicitamente storage: sqlite, limpa postgres_url herdado
		if check.Storage != nil && check.Storage.Engine == "sqlite" {
			cfg.Storage.Engine = "sqlite"
			cfg.Storage.PostgresURL = ""
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, "", err
	}

	// Camada 2.5: Verificação de overrides específicos no catálogo global de repositórios
	if globalCfg != nil && len(globalCfg.Repositories) > 0 {
		absDir, _ := filepath.Abs(repoDir)
		for _, repoEntry := range globalCfg.Repositories {
			entryAbs, _ := filepath.Abs(ExpandPath(repoEntry.Path))
			matchesID := localRepoID != "" && repoEntry.ID == localRepoID
			matchesPath := absDir != "" && entryAbs == absDir
			if matchesID || matchesPath {
				if repoEntry.Storage != nil {
					if repoEntry.Storage.Engine != "" {
						cfg.Storage.Engine = repoEntry.Storage.Engine
					}
					if repoEntry.Storage.PostgresURL != "" {
						cfg.Storage.PostgresURL = repoEntry.Storage.PostgresURL
					}
					if repoEntry.Storage.SQLitePath != "" {
						cfg.Storage.SQLitePath = repoEntry.Storage.SQLitePath
					}
				}
				break
			}
		}
	}

	// Expansão final de caminhos e envs
	cfg.Storage.SQLitePath = ExpandPath(cfg.Storage.SQLitePath)
	cfg.Storage.PostgresURL = os.ExpandEnv(cfg.Storage.PostgresURL)
	cfg.Embedding.URL = os.ExpandEnv(cfg.Embedding.URL)
	cfg.Repository = os.ExpandEnv(cfg.Repository)
	cfg.CentralVault.Path = ExpandPath(cfg.CentralVault.Path)
	if cfg.MCP.Port <= 0 {
		cfg.MCP.Port = 8080
	}

	return &cfg, cfgPath, nil
}
