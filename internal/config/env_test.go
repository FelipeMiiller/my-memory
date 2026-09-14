package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	content := `# Comentário
MY_TEST_VAR="hello_world"
MY_TEST_PG_URL='postgres://user:pass@localhost:5432/testdb'
DATABASE_URL="postgres://fallback:pass@localhost:5432/fallbackdb"
export EXPORTED_VAR=unquoted_val
OLLAMA_HOST="http://custom-ollama:11434"
EMBEDDING_MODEL="bge-m3"
EMBEDDING_DIM="1024"
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("erro ao criar arquivo .env: %v", err)
	}

	// Garante limpeza de envs antes do teste
	os.Unsetenv("MY_TEST_VAR")
	os.Unsetenv("MY_TEST_PG_URL")
	os.Unsetenv("MY_MEMORY_PG_URL")
	os.Unsetenv("EXPORTED_VAR")
	os.Unsetenv("MY_MEMORY_EMBED_URL")
	os.Unsetenv("MY_MEMORY_EMBED_MODEL")
	os.Unsetenv("MY_MEMORY_EMBED_DIM")

	if err := LoadDotEnvFile(envPath); err != nil {
		t.Fatalf("LoadDotEnvFile retornou erro: %v", err)
	}

	if val := os.Getenv("MY_TEST_VAR"); val != "hello_world" {
		t.Errorf("esperava MY_TEST_VAR='hello_world', obteve '%s'", val)
	}
	if val := os.Getenv("MY_TEST_PG_URL"); val != "postgres://user:pass@localhost:5432/testdb" {
		t.Errorf("esperava MY_TEST_PG_URL='postgres://user:pass@localhost:5432/testdb', obteve '%s'", val)
	}
	if val := os.Getenv("MY_MEMORY_PG_URL"); val != "postgres://fallback:pass@localhost:5432/fallbackdb" {
		t.Errorf("esperava alias MY_MEMORY_PG_URL populado a partir de DATABASE_URL, obteve '%s'", val)
	}
	if val := os.Getenv("MY_MEMORY_EMBED_URL"); val != "http://custom-ollama:11434" {
		t.Errorf("esperava alias MY_MEMORY_EMBED_URL populado a partir de OLLAMA_HOST, obteve '%s'", val)
	}
	if val := os.Getenv("MY_MEMORY_EMBED_MODEL"); val != "bge-m3" {
		t.Errorf("esperava alias MY_MEMORY_EMBED_MODEL populado a partir de EMBEDDING_MODEL, obteve '%s'", val)
	}
	if val := os.Getenv("MY_MEMORY_EMBED_DIM"); val != "1024" {
		t.Errorf("esperava alias MY_MEMORY_EMBED_DIM populado a partir de EMBEDDING_DIM, obteve '%s'", val)
	}
	if val := os.Getenv("EXPORTED_VAR"); val != "unquoted_val" {
		t.Errorf("esperava EXPORTED_VAR='unquoted_val', obteve '%s'", val)
	}
}

func TestFindAndLoadDotEnv(t *testing.T) {
	tmpDir := t.TempDir()
	memDir := filepath.Join(tmpDir, ".memory")
	if err := os.MkdirAll(memDir, 0755); err != nil {
		t.Fatalf("erro ao criar pasta .memory: %v", err)
	}

	envFile := filepath.Join(memDir, ".env")
	if err := os.WriteFile(envFile, []byte("MY_MEMORY_PG_URL=postgres://test:test@localhost:5432/db"), 0644); err != nil {
		t.Fatalf("erro ao escrever .memory/.env: %v", err)
	}

	os.Unsetenv("MY_MEMORY_PG_URL")

	foundPath, err := FindAndLoadDotEnv(tmpDir)
	if err != nil {
		t.Fatalf("FindAndLoadDotEnv falhou: %v", err)
	}

	if foundPath != envFile {
		t.Errorf("esperava foundPath %s, obteve %s", envFile, foundPath)
	}

	if val := os.Getenv("MY_MEMORY_PG_URL"); val != "postgres://test:test@localhost:5432/db" {
		t.Errorf("esperava MY_MEMORY_PG_URL setado no env, obteve '%s'", val)
	}
}
