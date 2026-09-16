package drift

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GitCommit representa um commit relevante extraído do histórico
type GitCommit struct {
	Hash       string    `json:"hash"`
	ShortHash  string    `json:"short_hash"`
	Author     string    `json:"author"`
	Date       time.Time `json:"date"`
	Message    string    `json:"message"`
	FilesCount int       `json:"files_count"`
}

// FileChange representa uma alteração de arquivo individual em um diff
type FileChange struct {
	Path       string `json:"path"`
	Status     string `json:"status"` // "A" (Added), "M" (Modified), "D" (Deleted), "R" (Renamed)
	Additions  int    `json:"additions"`
	Deletions  int    `json:"deletions"`
	IsCodeFile bool   `json:"is_code_file"`
}

// GitRunner define o contrato de interação com o repositório Git
type GitRunner interface {
	Log(ctx context.Context, repoRoot, rangeStr string) ([]GitCommit, error)
	Diff(ctx context.Context, repoRoot, rangeStr string) ([]FileChange, error)
}

// OSGitRunner é a implementação padrão baseada em execução do CLI do Git
type OSGitRunner struct{}

// NewOSGitRunner cria uma instância do executor de Git do sistema operacional
func NewOSGitRunner() *OSGitRunner {
	return &OSGitRunner{}
}

// Log extrai a lista de commits na faixa de referência solicitada
func (r *OSGitRunner) Log(ctx context.Context, repoRoot, rangeStr string) ([]GitCommit, error) {
	args := []string{"log", "--pretty=format:%H|%h|%an|%cI|%s"}
	if rangeStr != "" {
		args = append(args, rangeStr)
	} else {
		args = append(args, "-n", "5")
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	if repoRoot != "" {
		cmd.Dir = repoRoot
	}

	out, err := cmd.Output()
	if err != nil {
		// Fallback gracioso se a faixa especificada não existir (ex: menos commits disponíveis)
		fallbackCmd := exec.CommandContext(ctx, "git", "log", "-n", "5", "--pretty=format:%H|%h|%an|%cI|%s")
		if repoRoot != "" {
			fallbackCmd.Dir = repoRoot
		}
		fbOut, fbErr := fallbackCmd.Output()
		if fbErr != nil {
			return nil, fmt.Errorf("falha ao executar git log no repositório %q: %w", repoRoot, err)
		}
		out = fbOut
	}

	return parseGitLogOutput(string(out))
}

// Diff extrai as alterações de arquivos com adições, deleções e status
func (r *OSGitRunner) Diff(ctx context.Context, repoRoot, rangeStr string) ([]FileChange, error) {
	// 1. Executa git diff --numstat
	numstatArgs := []string{"diff", "--numstat"}
	if rangeStr != "" {
		numstatArgs = append(numstatArgs, rangeStr)
	} else {
		numstatArgs = append(numstatArgs, "HEAD~5..HEAD")
	}

	cmdNumstat := exec.CommandContext(ctx, "git", numstatArgs...)
	if repoRoot != "" {
		cmdNumstat.Dir = repoRoot
	}

	outNumstat, err := cmdNumstat.Output()
	if err != nil {
		// Fallback para último commit
		fbCmd := exec.CommandContext(ctx, "git", "diff", "--numstat", "HEAD~1..HEAD")
		if repoRoot != "" {
			fbCmd.Dir = repoRoot
		}
		fbOut, fbErr := fbCmd.Output()
		if fbErr != nil {
			return nil, fmt.Errorf("falha ao executar git diff numstat: %w", err)
		}
		outNumstat = fbOut
	}

	// 2. Executa git diff --name-status para mapear status (A, M, D, R)
	nameStatusArgs := []string{"diff", "--name-status"}
	if rangeStr != "" {
		nameStatusArgs = append(nameStatusArgs, rangeStr)
	} else {
		nameStatusArgs = append(nameStatusArgs, "HEAD~5..HEAD")
	}

	cmdStatus := exec.CommandContext(ctx, "git", nameStatusArgs...)
	if repoRoot != "" {
		cmdStatus.Dir = repoRoot
	}
	outStatus, _ := cmdStatus.Output()

	statusMap := parseGitStatusMap(string(outStatus))
	return parseGitNumstatOutput(string(outNumstat), statusMap)
}

// parseGitLogOutput interpreta a saída formatada de git log
func parseGitLogOutput(output string) ([]GitCommit, error) {
	var commits []GitCommit
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		parsedTime, err := time.Parse(time.RFC3339, parts[3])
		if err != nil {
			parsedTime = time.Now()
		}

		commits = append(commits, GitCommit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Author:    parts[2],
			Date:      parsedTime,
			Message:   strings.Join(parts[4:], "|"),
		})
	}
	return commits, nil
}

// parseGitStatusMap mapeia cada caminho ao seu status (M, A, D, R)
func parseGitStatusMap(output string) map[string]string {
	statusMap := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			statusCode := parts[0]
			path := parts[1]
			if strings.HasPrefix(statusCode, "R") && len(parts) >= 3 {
				path = parts[2] // Destino em renomeação
			}
			normPath := filepath.ToSlash(filepath.Clean(path))
			statusMap[normPath] = string(statusCode[0])
		}
	}
	return statusMap
}

// parseGitNumstatOutput processa as linhas de adições e deleções do git diff
func parseGitNumstatOutput(output string, statusMap map[string]string) ([]FileChange, error) {
	var changes []FileChange
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		adds, _ := strconv.Atoi(fields[0])
		dels, _ := strconv.Atoi(fields[1])
		path := fields[2]

		normPath := filepath.ToSlash(filepath.Clean(path))
		status, exists := statusMap[normPath]
		if !exists {
			status = "M"
		}

		changes = append(changes, FileChange{
			Path:       normPath,
			Status:     status,
			Additions:  adds,
			Deletions:  dels,
			IsCodeFile: IsCodeFile(normPath),
		})
	}
	return changes, nil
}

// IsCodeFile verifica se o arquivo é considerado código-fonte (e não documentação/config)
func IsCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".py", ".ts", ".js", ".tsx", ".jsx", ".rs", ".java", ".c", ".cpp", ".h", ".hpp", ".cs", ".rb", ".php", ".swift", ".kt", ".sql", ".sh", ".ps1":
		return true
	default:
		return false
	}
}

// MockGitRunner permite testes determinísticos e isolados
type MockGitRunner struct {
	Commits []GitCommit
	Changes []FileChange
	ErrLog  error
	ErrDiff error
}

func (m *MockGitRunner) Log(ctx context.Context, repoRoot, rangeStr string) ([]GitCommit, error) {
	if m.ErrLog != nil {
		return nil, m.ErrLog
	}
	return m.Commits, nil
}

func (m *MockGitRunner) Diff(ctx context.Context, repoRoot, rangeStr string) ([]FileChange, error) {
	if m.ErrDiff != nil {
		return nil, m.ErrDiff
	}
	return m.Changes, nil
}
