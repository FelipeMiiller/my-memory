package autowire

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockSystemEnv struct {
	goos    string
	home    string
	appData string
	files   map[string]bool
	dirs    map[string]bool
}

func (m mockSystemEnv) GOOS() string    { return m.goos }
func (m mockSystemEnv) HomeDir() string { return m.home }
func (m mockSystemEnv) AppData() string { return m.appData }
func (m mockSystemEnv) Stat(path string) (os.FileInfo, error) {
	clean := filepath.Clean(path)
	if m.files[clean] {
		return mockFileInfo{name: filepath.Base(clean), isDir: false}, nil
	}
	if m.dirs[clean] {
		return mockFileInfo{name: filepath.Base(clean), isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

type mockFileInfo struct {
	name  string
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return 100 }
func (m mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

func TestDetectClients_WindowsPaths(t *testing.T) {
	env := mockSystemEnv{
		goos:    "windows",
		home:    `C:\Users\Developer`,
		appData: `C:\Users\Developer\AppData\Roaming`,
		dirs: map[string]bool{
			`C:\Users\Developer\AppData\Roaming\Claude`: true,
			`C:\Users\Developer\.cursor`:                true,
		},
		files: map[string]bool{
			`C:\Users\Developer\AppData\Roaming\Claude\claude_desktop_config.json`: true,
		},
	}

	targets, err := DetectClientsWithEnv(`C:\repo`, "all", ScopeAll, env)
	if err != nil {
		t.Fatalf("falha ao detectar clientes no Windows: %v", err)
	}

	if len(targets) != 5 {
		t.Fatalf("esperava 5 alvos no Windows, obteve %d", len(targets))
	}

	var claude, cursorWS, cursorGlobal, vscodeWS, windsurf *ClientTarget
	for i := range targets {
		tgt := &targets[i]
		switch {
		case tgt.Type == ClientClaudeDesktop:
			claude = tgt
		case tgt.Type == ClientCursor && tgt.Scope == ScopeWorkspace:
			cursorWS = tgt
		case tgt.Type == ClientCursor && tgt.Scope == ScopeGlobal:
			cursorGlobal = tgt
		case tgt.Type == ClientVSCode:
			vscodeWS = tgt
		case tgt.Type == ClientWindsurf:
			windsurf = tgt
		}
	}

	if claude == nil || !strings.Contains(claude.ConfigPath, `AppData\Roaming\Claude\claude_desktop_config.json`) {
		t.Errorf("caminho do Claude Desktop no Windows incorreto: %+v", claude)
	}
	if claude != nil && !claude.Exists {
		t.Errorf("esperava claude.Exists == true")
	}

	if cursorWS == nil || !strings.Contains(cursorWS.ConfigPath, `C:\repo\.cursor\mcp.json`) {
		t.Errorf("caminho do Cursor Workspace incorreto: %+v", cursorWS)
	}

	if cursorGlobal == nil || !strings.Contains(cursorGlobal.ConfigPath, `C:\Users\Developer\.cursor\mcp.json`) {
		t.Errorf("caminho do Cursor Global incorreto: %+v", cursorGlobal)
	}

	if vscodeWS == nil || !strings.Contains(vscodeWS.ConfigPath, `C:\repo\.vscode\mcp.json`) {
		t.Errorf("caminho do VS Code Workspace incorreto: %+v", vscodeWS)
	}

	if windsurf == nil || !strings.Contains(windsurf.ConfigPath, `C:\Users\Developer\.codeium\windsurf\mcp_config.json`) {
		t.Errorf("caminho do Windsurf incorreto: %+v", windsurf)
	}
}

func TestDetectClients_DarwinPaths(t *testing.T) {
	env := mockSystemEnv{
		goos: "darwin",
		home: "/Users/dev",
	}

	targets, err := DetectClientsWithEnv("/workspace", "claude", ScopeGlobal, env)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("esperava 1 alvo (Claude), obteve %d", len(targets))
	}

	expected := filepath.FromSlash("/Users/dev/Library/Application Support/Claude/claude_desktop_config.json")
	if targets[0].ConfigPath != expected {
		t.Errorf("esperava '%s', obteve '%s'", expected, targets[0].ConfigPath)
	}
}

func TestDetectClients_LinuxPaths(t *testing.T) {
	env := mockSystemEnv{
		goos: "linux",
		home: "/home/dev",
	}

	targets, err := DetectClientsWithEnv("/workspace", "claude", ScopeGlobal, env)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("esperava 1 alvo, obteve %d", len(targets))
	}

	expected := filepath.FromSlash("/home/dev/.config/Claude/claude_desktop_config.json")
	if targets[0].ConfigPath != expected {
		t.Errorf("esperava '%s', obteve '%s'", expected, targets[0].ConfigPath)
	}
}

func TestDetectClients_FilterScope(t *testing.T) {
	env := mockSystemEnv{
		goos: "linux",
		home: "/home/dev",
	}

	// Somente workspace
	wsTargets, _ := DetectClientsWithEnv("/workspace", "all", ScopeWorkspace, env)
	if len(wsTargets) != 2 { // Cursor e VS Code
		t.Errorf("esperava 2 alvos no workspace, obteve %d", len(wsTargets))
	}
	for _, tgt := range wsTargets {
		if tgt.Scope != ScopeWorkspace {
			t.Errorf("alvo com escopo inesperado: %+v", tgt)
		}
	}

	// Somente global
	globalTargets, _ := DetectClientsWithEnv("/workspace", "all", ScopeGlobal, env)
	if len(globalTargets) != 3 { // Claude, Cursor Global, Windsurf
		t.Errorf("esperava 3 alvos globais, obteve %d", len(globalTargets))
	}
	for _, tgt := range globalTargets {
		if tgt.Scope != ScopeGlobal {
			t.Errorf("alvo com escopo inesperado: %+v", tgt)
		}
	}
}
