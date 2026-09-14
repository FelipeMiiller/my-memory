package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// CandidateEnvFileNames lista os caminhos prioritários para localização de arquivos de ambiente
var CandidateEnvFileNames = []string{
	filepath.Join(".memory", ".env"),
	".env",
}

// FindAndLoadDotEnv busca arquivos .env (priorizando .memory/.env e depois .env)
// a partir de startDir subindo até a raiz do git ou disco, carregando as variáveis encontradas.
func FindAndLoadDotEnv(startDir string) (string, error) {
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
		for _, candidate := range CandidateEnvFileNames {
			p := filepath.Join(curr, candidate)
			if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
				if err := LoadDotEnvFile(p); err != nil {
					return p, err
				}
				return p, nil
			}
		}

		// Interrompe na raiz do repositório Git para não vazar variáveis
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

// LoadDotEnvFile faz o parse de um arquivo .env simples (chave=valor) e injeta com os.Setenv
// caso a variável ainda não esteja definida no ambiente do sistema operacional.
func LoadDotEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Remove aspas simples ou duplas envolventes
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}

		if key == "" {
			continue
		}

		// Se a variável não estiver definida no processo, define-a
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}

		// Aliases universais para PostgreSQL:
		// Se POSTGRES_URL ou DATABASE_URL foi definido e MY_MEMORY_PG_URL ainda não, copia o valor
		if (key == "POSTGRES_URL" || key == "DATABASE_URL") && os.Getenv("MY_MEMORY_PG_URL") == "" {
			_ = os.Setenv("MY_MEMORY_PG_URL", val)
		}

		// Aliases universais para Embedding/IA de Indexação:
		if (key == "EMBEDDING_URL" || key == "OLLAMA_URL" || key == "OLLAMA_HOST") && os.Getenv("MY_MEMORY_EMBED_URL") == "" {
			_ = os.Setenv("MY_MEMORY_EMBED_URL", val)
		}
		if (key == "EMBEDDING_MODEL" || key == "OLLAMA_MODEL") && os.Getenv("MY_MEMORY_EMBED_MODEL") == "" {
			_ = os.Setenv("MY_MEMORY_EMBED_MODEL", val)
		}
		if key == "EMBEDDING_DIM" && os.Getenv("MY_MEMORY_EMBED_DIM") == "" {
			_ = os.Setenv("MY_MEMORY_EMBED_DIM", val)
		}
	}

	return scanner.Err()
}
