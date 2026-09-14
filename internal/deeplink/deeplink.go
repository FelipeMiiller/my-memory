package deeplink

import (
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DeepLinks contém os esquemas de URI para navegação em editores e sistema de arquivos
type DeepLinks struct {
	Obsidian string `json:"obsidian"`
	VSCode   string `json:"vscode"`
	File     string `json:"file"`
}

// Launcher interface para abstrair a execução de processos de abertura do sistema operacional
type Launcher interface {
	Launch(command string, args ...string) error
}

// DefaultSystemLauncher implementa Launcher utilizando os executores nativos do SO
type DefaultSystemLauncher struct{}

func (s *DefaultSystemLauncher) Launch(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	return cmd.Start()
}

// DefaultLauncher instância padrão do launcher do sistema operacional
var DefaultLauncher Launcher = &DefaultSystemLauncher{}

// NormalizePath formata o caminho com barras normais (slash) e remove ambiguidades
func NormalizePath(p string) string {
	cleaned := filepath.Clean(p)
	return filepath.ToSlash(cleaned)
}

// GenerateLinks gera as URIs correspondentes para Obsidian, VS Code e File a partir do caminho
func GenerateLinks(repoRoot, vaultName, filePath string, line int) DeepLinks {
	if filePath == "" {
		return DeepLinks{}
	}

	normRoot := NormalizePath(repoRoot)
	normPath := NormalizePath(filePath)

	var absPath string
	var relPath string

	if filepath.IsAbs(filePath) {
		absPath = normPath
		if normRoot != "" {
			if rel, err := filepath.Rel(normRoot, normPath); err == nil && !strings.HasPrefix(rel, "..") {
				relPath = NormalizePath(rel)
			} else {
				relPath = NormalizePath(filepath.Base(normPath))
			}
		} else {
			relPath = NormalizePath(filepath.Base(normPath))
		}
	} else {
		relPath = strings.TrimPrefix(normPath, "./")
		if normRoot != "" {
			absPath = NormalizePath(filepath.Join(normRoot, relPath))
		} else {
			abs, err := filepath.Abs(relPath)
			if err == nil {
				absPath = NormalizePath(abs)
			} else {
				absPath = relPath
			}
		}
	}

	// 1. Obsidian URI
	// Formato padrão: obsidian://open?vault=<vault>&file=<relPath>
	// Se vaultName for vazio, fallback para obsidian://open?path=<absPath>
	var obsidianURI string
	resolvedVault := strings.TrimSpace(vaultName)
	if resolvedVault == "" && normRoot != "" {
		resolvedVault = filepath.Base(normRoot)
	}

	if resolvedVault != "" {
		cleanRel := strings.TrimPrefix(relPath, "/")
		obsidianURI = fmt.Sprintf("obsidian://open?vault=%s&file=%s",
			url.QueryEscape(resolvedVault),
			url.QueryEscape(cleanRel))
	} else {
		obsidianURI = fmt.Sprintf("obsidian://open?path=%s", url.QueryEscape(absPath))
	}

	// 2. VS Code URI
	// Formato: vscode://file/<absPath>[:line]
	cleanAbs := absPath
	if runtime.GOOS == "windows" {
		// Garante que o drive letter não tenha barras duplicadas e formato /C:/...
		cleanAbs = strings.TrimPrefix(cleanAbs, "/")
	}
	var vscodeURI string
	if line > 0 {
		vscodeURI = fmt.Sprintf("vscode://file/%s:%d", cleanAbs, line)
	} else {
		vscodeURI = fmt.Sprintf("vscode://file/%s", cleanAbs)
	}

	// 3. File URI
	// Formato: file:///<absPath>
	cleanFileAbs := strings.TrimPrefix(absPath, "/")
	fileURI := fmt.Sprintf("file:///%s", cleanFileAbs)

	return DeepLinks{
		Obsidian: obsidianURI,
		VSCode:   vscodeURI,
		File:     fileURI,
	}
}

// Open resolve a URI adequada para o aplicativo solicitado e a dispara via Launcher
func Open(targetPathOrURI string, app string, line int, repoRoot string, vaultName string, launcher Launcher) (string, error) {
	if targetPathOrURI == "" {
		return "", fmt.Errorf("alvo para abertura não pode ser vazio")
	}

	if launcher == nil {
		launcher = DefaultLauncher
	}

	var targetURI string
	isURI := strings.Contains(targetPathOrURI, "://")

	if isURI {
		targetURI = targetPathOrURI
	} else {
		links := GenerateLinks(repoRoot, vaultName, targetPathOrURI, line)
		switch strings.ToLower(strings.TrimSpace(app)) {
		case "vscode", "code":
			targetURI = links.VSCode
		case "system", "file", "default":
			targetURI = links.File
		case "obsidian":
			targetURI = links.Obsidian
		default:
			// Padrão: Obsidian
			targetURI = links.Obsidian
		}
	}

	cmdName, cmdArgs := BuildOSCommand(targetURI)
	err := launcher.Launch(cmdName, cmdArgs...)
	if err != nil {
		return targetURI, fmt.Errorf("falha ao abrir URI '%s' com comando '%s %s': %w", targetURI, cmdName, strings.Join(cmdArgs, " "), err)
	}

	return targetURI, nil
}

// BuildOSCommand retorna o comando e argumentos nativos do SO para abrir a URI indicada
func BuildOSCommand(targetURI string) (string, []string) {
	switch runtime.GOOS {
	case "windows":
		// cmd.exe /c start "" "<targetURI>"
		return "cmd", []string{"/c", "start", "", targetURI}
	case "darwin":
		return "open", []string{targetURI}
	default:
		// linux e outros
		return "xdg-open", []string{targetURI}
	}
}
