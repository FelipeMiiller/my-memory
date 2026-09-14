package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	Version    int             `yaml:"version" json:"version"`
	Repository string          `yaml:"repository,omitempty" json:"repository,omitempty"` // Slug do repositório (ex: "owner/repo")
	VaultName  string          `yaml:"vault_name,omitempty" json:"vault_name,omitempty"` // Nome amigável do vault de notas
	Include    []string        `yaml:"include,omitempty" json:"include,omitempty"`       // Padrões glob de arquivos a indexar
	Exclude    []string        `yaml:"exclude,omitempty" json:"exclude,omitempty"`       // Padrões glob de pastas/arquivos a ignorar
	Storage    StorageConfig   `yaml:"storage,omitempty" json:"storage,omitempty"`
	Embedding  EmbeddingConfig `yaml:"embedding,omitempty" json:"embedding,omitempty"`
	Search     SearchConfig    `yaml:"search,omitempty" json:"search,omitempty"`
	Watcher    WatcherConfig   `yaml:"watcher,omitempty" json:"watcher,omitempty"`
	Editor     EditorConfig    `yaml:"editor,omitempty" json:"editor,omitempty"`
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

	curr := absDir
	for {
		for _, candidate := range CandidateConfigFileNames {
			p := filepath.Join(curr, candidate)
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

	// Expande variáveis de ambiente (${VAR}) em caminhos e URLs
	cfg.Storage.SQLitePath = os.ExpandEnv(cfg.Storage.SQLitePath)
	cfg.Storage.PostgresURL = os.ExpandEnv(cfg.Storage.PostgresURL)
	cfg.Embedding.URL = os.ExpandEnv(cfg.Embedding.URL)
	cfg.Repository = os.ExpandEnv(cfg.Repository)

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
