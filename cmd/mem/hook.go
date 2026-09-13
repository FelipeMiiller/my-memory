package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const HookSignature = "# My-Memory Pre-Commit Hook"

const PreCommitHookTemplate = `#!/usr/bin/env bash
# ==============================================================================
# My-Memory Pre-Commit Hook
# Garante que todas as notas Markdown alteradas estejam indexadas antes do commit
# ==============================================================================

# Se mem estiver no PATH, usa 'mem', caso contrário tenta './bin/mem' ou 'go run ./cmd/mem'
if command -v mem >/dev/null 2>&1; then
    MEM_CMD="mem"
elif [ -f "./bin/mem" ]; then
    MEM_CMD="./bin/mem"
elif [ -f "./bin/mem.exe" ]; then
    MEM_CMD="./bin/mem.exe"
else
    MEM_CMD="go run ./cmd/mem"
fi

echo "🧠 [my-memory] Sincronizando índice e validando notas antes do commit..."
$MEM_CMD index
STATUS=$?

if [ $STATUS -ne 0 ]; then
    echo "❌ [my-memory] Falha na indexação de notas. Commit abortado."
    exit $STATUS
fi

echo "✅ [my-memory] Notas verificadas e sincronizadas com sucesso."
exit 0
`

// findGitDir procura o diretório .git subindo a partir de startDir
func findGitDir(startDir string) (string, error) {
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
		gitPath := filepath.Join(curr, ".git")
		if stat, err := os.Stat(gitPath); err == nil && stat.IsDir() {
			return gitPath, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr || parent == "" {
			break
		}
		curr = parent
	}

	return "", errors.New("repositório Git (.git) não foi encontrado no caminho atual ou diretórios superiores")
}

// InstallGitHook instala o script pre-commit no repositório Git
func InstallGitHook(targetDir string, force bool) (string, error) {
	gitDir, err := findGitDir(targetDir)
	if err != nil {
		return "", err
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return "", fmt.Errorf("erro ao criar pasta de hooks: %w", err)
	}

	hookPath := filepath.Join(hooksDir, "pre-commit")
	if data, err := os.ReadFile(hookPath); err == nil && !force {
		if !strings.Contains(string(data), HookSignature) {
			return "", fmt.Errorf("já existe um pre-commit hook de outro software em %s. Use --force para sobrescrever", hookPath)
		}
	}

	if err := os.WriteFile(hookPath, []byte(PreCommitHookTemplate), 0755); err != nil {
		return "", fmt.Errorf("erro ao gravar pre-commit hook: %w", err)
	}

	return hookPath, nil
}

// UninstallGitHook remove com segurança o pre-commit hook instalado pelo My-Memory
func UninstallGitHook(targetDir string) error {
	gitDir, err := findGitDir(targetDir)
	if err != nil {
		return err
	}

	hookPath := filepath.Join(gitDir, "hooks", "pre-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Já não existe, idempotente
		}
		return err
	}

	if !strings.Contains(string(data), HookSignature) {
		return errors.New("o hook existente não foi gerado pelo My-Memory; remoção abortada para evitar perda de dados")
	}

	return os.Remove(hookPath)
}
