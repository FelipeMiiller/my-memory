package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CleanGitURL converte URLs remotas HTTP/HTTPS ou SSH no formato canônico "owner/repo"
func CleanGitURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	// Remove sufixo .git
	rawURL = strings.TrimSuffix(rawURL, ".git")

	// Formato SSH: git@github.com:owner/repo
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") {
		parts := strings.Split(rawURL, ":")
		if len(parts) >= 2 {
			return strings.Trim(parts[1], "/")
		}
	}

	// Formato HTTP/HTTPS: https://github.com/owner/repo
	if strings.Contains(rawURL, "://") {
		parts := strings.SplitN(rawURL, "://", 2)
		if len(parts) == 2 {
			slashIdx := strings.Index(parts[1], "/")
			if slashIdx != -1 {
				return strings.Trim(parts[1][slashIdx+1:], "/")
			}
		}
	}

	return strings.Trim(rawURL, "/")
}

// DetectRepository detecta o identificador do repositório a partir de um diretório
func DetectRepository(dir string) string {
	if envRepo := os.Getenv("MY_MEMORY_REPO"); envRepo != "" {
		return strings.TrimSpace(envRepo)
	}

	if dir == "" || dir == "." {
		cwd, err := os.Getwd()
		if err == nil {
			dir = cwd
		}
	}

	// 1. Tenta obter remote.origin.url do git
	cmd := exec.Command("git", "-C", dir, "config", "--get", "remote.origin.url")
	if out, err := cmd.Output(); err == nil {
		if slug := CleanGitURL(string(out)); slug != "" {
			return slug
		}
	}

	// 2. Tenta obter a raiz do repositório git
	cmdRoot := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	if out, err := cmdRoot.Output(); err == nil {
		topLevel := strings.TrimSpace(string(out))
		if topLevel != "" {
			return filepath.Base(filepath.Clean(topLevel))
		}
	}

	// 3. Fallback para o nome da pasta
	absDir, err := filepath.Abs(dir)
	if err == nil {
		return filepath.Base(absDir)
	}

	return filepath.Base(filepath.Clean(dir))
}
