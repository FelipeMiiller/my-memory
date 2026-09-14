package autowire

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// SystemEnv abstrai acessos ao ambiente do sistema operacional para permitir testes determinísticos
type SystemEnv interface {
	GOOS() string
	HomeDir() string
	AppData() string
	Stat(path string) (os.FileInfo, error)
}

type realSystemEnv struct{}

func (realSystemEnv) GOOS() string {
	return runtime.GOOS
}

func (realSystemEnv) HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func (realSystemEnv) AppData() string {
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return appData
		}
	}
	return ""
}

func (realSystemEnv) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// DetectClients detecta clientes MCP candidatos usando o ambiente real do sistema
func DetectClients(rootDir string, targetFilter string, scopeFilter ConfigScope) ([]ClientTarget, error) {
	return DetectClientsWithEnv(rootDir, targetFilter, scopeFilter, realSystemEnv{})
}

// DetectClientsWithEnv detecta clientes com injeção de ambiente para testes unitários
func DetectClientsWithEnv(rootDir string, targetFilter string, scopeFilter ConfigScope, env SystemEnv) ([]ClientTarget, error) {
	if rootDir == "" || rootDir == "." {
		cwd, err := os.Getwd()
		if err == nil {
			rootDir = cwd
		}
	}

	normTarget := strings.ToLower(strings.TrimSpace(targetFilter))
	if normTarget == "" {
		normTarget = "all"
	}

	normScope := ConfigScope(strings.ToLower(strings.TrimSpace(string(scopeFilter))))
	if normScope == "" {
		normScope = ScopeAll
	}

	var candidates []ClientTarget

	// 1. Claude Desktop (Global)
	if matchTarget(normTarget, "claude") && matchScope(normScope, ScopeGlobal) {
		var claudePath string
		goos := env.GOOS()
		switch goos {
		case "windows":
			appData := env.AppData()
			if appData != "" {
				claudePath = filepath.Join(appData, "Claude", "claude_desktop_config.json")
			}
		case "darwin":
			home := env.HomeDir()
			if home != "" {
				claudePath = filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
			}
		default: // linux / bsd
			home := env.HomeDir()
			if home != "" {
				claudePath = filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
			}
		}

		if claudePath != "" {
			candidates = append(candidates, makeClientTarget(
				ClientClaudeDesktop,
				"Claude Desktop (Global)",
				ScopeGlobal,
				claudePath,
				env,
			))
		}
	}

	// 2. Cursor IDE (Workspace)
	if matchTarget(normTarget, "cursor") && matchScope(normScope, ScopeWorkspace) {
		cursorWS := filepath.Join(rootDir, ".cursor", "mcp.json")
		candidates = append(candidates, makeClientTarget(
			ClientCursor,
			"Cursor IDE (Workspace)",
			ScopeWorkspace,
			cursorWS,
			env,
		))
	}

	// 3. Cursor IDE (Global)
	if matchTarget(normTarget, "cursor") && matchScope(normScope, ScopeGlobal) {
		home := env.HomeDir()
		if home != "" {
			cursorGlobal := filepath.Join(home, ".cursor", "mcp.json")
			candidates = append(candidates, makeClientTarget(
				ClientCursor,
				"Cursor IDE (Global)",
				ScopeGlobal,
				cursorGlobal,
				env,
			))
		}
	}

	// 4. VS Code / Copilot (Workspace)
	if matchTarget(normTarget, "vscode") && matchScope(normScope, ScopeWorkspace) {
		vscodeWS := filepath.Join(rootDir, ".vscode", "mcp.json")
		candidates = append(candidates, makeClientTarget(
			ClientVSCode,
			"VS Code / GitHub Copilot (Workspace)",
			ScopeWorkspace,
			vscodeWS,
			env,
		))
	}

	// 5. Windsurf / Codeium (Global)
	if matchTarget(normTarget, "windsurf") && matchScope(normScope, ScopeGlobal) {
		home := env.HomeDir()
		if home != "" {
			windsurfGlobal := filepath.Join(home, ".codeium", "windsurf", "mcp_config.json")
			candidates = append(candidates, makeClientTarget(
				ClientWindsurf,
				"Windsurf / Codeium (Global)",
				ScopeGlobal,
				windsurfGlobal,
				env,
			))
		}
	}

	return candidates, nil
}

func makeClientTarget(cType ClientType, name string, scope ConfigScope, configPath string, env SystemEnv) ClientTarget {
	target := ClientTarget{
		Type:       cType,
		Name:       name,
		Scope:      scope,
		ConfigPath: filepath.Clean(configPath),
	}

	if stat, err := env.Stat(target.ConfigPath); err == nil && !stat.IsDir() {
		target.Exists = true
	}

	parentDir := filepath.Dir(target.ConfigPath)
	if pStat, err := env.Stat(parentDir); err == nil && pStat.IsDir() {
		target.ParentDirOK = true
	}

	return target
}

func matchTarget(filter string, targetName string) bool {
	return filter == "all" || filter == targetName
}

func matchScope(filter ConfigScope, targetScope ConfigScope) bool {
	return filter == ScopeAll || filter == targetScope
}
