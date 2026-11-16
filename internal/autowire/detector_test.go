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
	norm := path
	if m.goos == "windows" {
		norm = strings.ReplaceAll(path, "/", `\`)
	} else {
		norm = filepath.Clean(path)
	}
	if m.files[norm] || m.files[path] {
		return mockFileInfo{name: filepath.Base(norm), isDir: false}, nil
	}
	if m.dirs[norm] || m.dirs[path] {
		return mockFileInfo{name: filepath.Base(norm), isDir: true}, nil
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

func TestAcceptedTargets_ConjuntoExatoDocumentado(t *testing.T) {
	want := []string{"all", "claude", "claude-desktop", "cursor", "vscode", "windsurf"}

	got := AcceptedTargets()
	if len(got) != len(want) {
		t.Fatalf("conjunto aceito com tamanho %d, esperava %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("índice %d: obteve %q, esperava %q", i, got[i], want[i])
		}
	}

	// todo valor do conjunto é aceito
	for _, value := range want {
		if !IsAcceptedTarget(value) {
			t.Errorf("valor do conjunto aceito foi rejeitado: %q", value)
		}
	}

	// valor fora do conjunto é rejeitado
	for _, value := range []string{"xxxx", "claude code", "vs-code", "cursor2", ""} {
		if IsAcceptedTarget(value) {
			t.Errorf("valor fora do conjunto foi aceito: %q", value)
		}
	}
}

func TestDetectClients_ClaudeDesktopSinonimoResolveSoClaude(t *testing.T) {
	env := mockSystemEnv{
		goos:    "windows",
		home:    `C:\Users\Developer`,
		appData: `C:\Users\Developer\AppData\Roaming`,
	}

	byAlias, err := DetectClientsWithEnv(`C:\repo`, "claude-desktop", ScopeAll, env)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}

	// conjunto exato: só o Claude Desktop, e nada mais
	if len(byAlias) != 1 {
		t.Fatalf("esperava 1 alvo com 'claude-desktop', obteve %d: %+v", len(byAlias), byAlias)
	}
	if byAlias[0].Type != ClientClaudeDesktop {
		t.Errorf("tipo de alvo inesperado: %+v", byAlias[0])
	}
	for _, tgt := range byAlias {
		switch tgt.Type {
		case ClientCursor, ClientVSCode, ClientWindsurf:
			t.Errorf("'claude-desktop' resolveu um cliente que nao deve: %+v", tgt)
		}
	}

	// o sinônimo resolve o mesmo alvo do valor canônico
	byCanonical, err := DetectClientsWithEnv(`C:\repo`, "claude", ScopeAll, env)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if len(byCanonical) != 1 {
		t.Fatalf("esperava 1 alvo com 'claude', obteve %d: %+v", len(byCanonical), byCanonical)
	}
	if byAlias[0].ConfigPath != byCanonical[0].ConfigPath {
		t.Errorf("'claude-desktop' resolveu '%s' e 'claude' resolveu '%s'", byAlias[0].ConfigPath, byCanonical[0].ConfigPath)
	}
}

func TestDetectClients_PresencaSegueMarcadorDeInstalacao(t *testing.T) {
	env := mockSystemEnv{
		goos:    "windows",
		home:    `C:\Users\Developer`,
		appData: `C:\Users\Developer\AppData\Roaming`,
		dirs: map[string]bool{
			`C:\Users\Developer\.cursor`:                true,
			`C:\Users\Developer\AppData\Roaming\Claude`: true,
		},
	}

	targets, err := DetectClientsWithEnv(`C:\repo`, "all", ScopeAll, env)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}

	presente := map[ClientType]bool{}
	for _, tgt := range targets {
		if tgt.Installed {
			presente[tgt.Type] = true
		}
	}

	// controle positivo: o marcador existe, o cliente sai marcado como instalado
	if !presente[ClientCursor] {
		t.Errorf("Cursor deveria estar marcado como instalado (marcador existe): %+v", targets)
	}
	if !presente[ClientClaudeDesktop] {
		t.Errorf("Claude Desktop deveria estar marcado como instalado (marcador existe): %+v", targets)
	}

	// controle negativo: o marcador nao existe, o cliente sai marcado como ausente
	if presente[ClientVSCode] {
		t.Errorf("VS Code nao deveria estar marcado como instalado (marcador ausente): %+v", targets)
	}
	if presente[ClientWindsurf] {
		t.Errorf("Windsurf nao deveria estar marcado como instalado (marcador ausente): %+v", targets)
	}
}

func TestClientInstallMarker_PorSistemaOperacional(t *testing.T) {
	win := mockSystemEnv{goos: "windows", home: `C:\Users\Developer`, appData: `C:\Users\Developer\AppData\Roaming`}
	darwin := mockSystemEnv{goos: "darwin", home: "/Users/dev"}
	linux := mockSystemEnv{goos: "linux", home: "/home/dev"}

	cases := []struct {
		name    string
		env     mockSystemEnv
		client  ClientType
		expected string
	}{
		{"windows claude", win, ClientClaudeDesktop, `C:\Users\Developer\AppData\Roaming\Claude`},
		{"windows cursor", win, ClientCursor, `C:\Users\Developer\.cursor`},
		{"windows vscode", win, ClientVSCode, `C:\Users\Developer\AppData\Roaming\Code`},
		{"windows windsurf", win, ClientWindsurf, `C:\Users\Developer\.codeium\windsurf`},
		{"darwin claude", darwin, ClientClaudeDesktop, "/Users/dev/Library/Application Support/Claude"},
		{"darwin cursor", darwin, ClientCursor, "/Users/dev/.cursor"},
		{"darwin vscode", darwin, ClientVSCode, "/Users/dev/Library/Application Support/Code"},
		{"darwin windsurf", darwin, ClientWindsurf, "/Users/dev/.codeium/windsurf"},
		{"linux claude", linux, ClientClaudeDesktop, "/home/dev/.config/Claude"},
		{"linux cursor", linux, ClientCursor, "/home/dev/.cursor"},
		{"linux vscode", linux, ClientVSCode, "/home/dev/.config/Code"},
		{"linux windsurf", linux, ClientWindsurf, "/home/dev/.codeium/windsurf"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := ClientInstallMarker(tt.client, tt.env)
			if got != tt.expected {
				t.Errorf("marcador obtido '%s', esperava '%s'", got, tt.expected)
			}
		})
	}
}
