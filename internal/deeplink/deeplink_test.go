package deeplink

import (
	"runtime"
	"strings"
	"testing"
)

type mockLauncher struct {
	lastCmd     string
	lastArgs    []string
	callCount   int
	errToReturn error
}

func (m *mockLauncher) Launch(command string, args ...string) error {
	m.callCount++
	m.lastCmd = command
	m.lastArgs = args
	return m.errToReturn
}

func TestGenerateLinks_Basic(t *testing.T) {
	repoRoot := "/workspace/my-vault"
	if runtime.GOOS == "windows" {
		repoRoot = "C:/workspace/my-vault"
	}
	vaultName := "My Vault"
	relPath := "docs/adr/001-architecture.md"

	links := GenerateLinks(repoRoot, vaultName, relPath, 0)

	// Obsidian
	if !strings.HasPrefix(links.Obsidian, "obsidian://open?") {
		t.Errorf("esperava prefixo obsidian://open?, obteve: %s", links.Obsidian)
	}
	if !strings.Contains(links.Obsidian, "vault=My+Vault") && !strings.Contains(links.Obsidian, "vault=My%20Vault") {
		t.Errorf("esperava vault escapado na URI, obteve: %s", links.Obsidian)
	}
	if !strings.Contains(links.Obsidian, "file=docs%2Fadr%2F001-architecture.md") && !strings.Contains(links.Obsidian, "file=docs/adr/001-architecture.md") {
		t.Errorf("esperava file escapado na URI, obteve: %s", links.Obsidian)
	}

	// VSCode
	if !strings.HasPrefix(links.VSCode, "vscode://file/") {
		t.Errorf("esperava prefixo vscode://file/, obteve: %s", links.VSCode)
	}
	if !strings.HasSuffix(links.VSCode, "docs/adr/001-architecture.md") {
		t.Errorf("esperava sufixo docs/adr/001-architecture.md, obteve: %s", links.VSCode)
	}

	// File
	if !strings.HasPrefix(links.File, "file:///") {
		t.Errorf("esperava prefixo file:///, obteve: %s", links.File)
	}
}

func TestGenerateLinks_WithLineNumber(t *testing.T) {
	repoRoot := "/workspace/my-vault"
	if runtime.GOOS == "windows" {
		repoRoot = "C:/workspace/my-vault"
	}
	relPath := "README.md"
	line := 42

	links := GenerateLinks(repoRoot, "TestVault", relPath, line)

	if !strings.HasSuffix(links.VSCode, ":42") {
		t.Errorf("esperava número de linha :42 no VS Code, obteve: %s", links.VSCode)
	}
}

func TestGenerateLinks_FallbackVault(t *testing.T) {
	repoRoot := "/workspace/cool-notes"
	if runtime.GOOS == "windows" {
		repoRoot = "C:/workspace/cool-notes"
	}
	// vaultName vazio -> deve usar cool-notes
	links := GenerateLinks(repoRoot, "", "notes.md", 0)

	if !strings.Contains(links.Obsidian, "vault=cool-notes") {
		t.Errorf("esperava fallback para nome do repositório 'cool-notes', obteve: %s", links.Obsidian)
	}
}

func TestGenerateLinks_EmptyPath(t *testing.T) {
	links := GenerateLinks("/repo", "vault", "", 0)
	if links.Obsidian != "" || links.VSCode != "" || links.File != "" {
		t.Errorf("esperava links vazios para caminho vazio, obteve: %+v", links)
	}
}

func TestOpen_WithMockLauncher(t *testing.T) {
	repoRoot := "/workspace/vault"
	if runtime.GOOS == "windows" {
		repoRoot = "C:/workspace/vault"
	}
	mock := &mockLauncher{}

	// 1. Abrir no Obsidian
	uri, err := Open("docs/spec.md", "obsidian", 0, repoRoot, "vault", mock)
	if err != nil {
		t.Fatalf("erro inesperado ao abrir no obsidian: %v", err)
	}
	if !strings.HasPrefix(uri, "obsidian://") {
		t.Errorf("esperava URI do Obsidian, obteve: %s", uri)
	}
	if mock.callCount != 1 {
		t.Errorf("esperava 1 chamada no launcher, obteve %d", mock.callCount)
	}

	// 2. Abrir no VS Code
	uri, err = Open("docs/spec.md", "vscode", 15, repoRoot, "vault", mock)
	if err != nil {
		t.Fatalf("erro inesperado ao abrir no vscode: %v", err)
	}
	if !strings.HasPrefix(uri, "vscode://") || !strings.HasSuffix(uri, ":15") {
		t.Errorf("esperava URI do VS Code com linha :15, obteve: %s", uri)
	}
	if mock.callCount != 2 {
		t.Errorf("esperava 2 chamadas no launcher, obteve %d", mock.callCount)
	}

	// 3. Abrir no Sistema (File)
	uri, err = Open("docs/spec.md", "system", 0, repoRoot, "vault", mock)
	if err != nil {
		t.Fatalf("erro inesperado ao abrir no sistema: %v", err)
	}
	if !strings.HasPrefix(uri, "file:///") {
		t.Errorf("esperava URI file:///, obteve: %s", uri)
	}

	// 4. Abrir URI direta
	rawURI := "obsidian://open?vault=Custom&file=Intro"
	uri, err = Open(rawURI, "", 0, repoRoot, "", mock)
	if err != nil {
		t.Fatalf("erro ao abrir URI direta: %v", err)
	}
	if uri != rawURI {
		t.Errorf("esperava URI inalterada %s, obteve %s", rawURI, uri)
	}

	// 5. Target vazio
	_, err = Open("", "obsidian", 0, repoRoot, "", mock)
	if err == nil {
		t.Errorf("esperava erro ao passar alvo vazio")
	}
}

func TestBuildOSCommand(t *testing.T) {
	cmd, args := BuildOSCommand("obsidian://test")
	if cmd == "" || len(args) == 0 {
		t.Errorf("esperava comando e argumentos válidos, obteve cmd='%s' args=%v", cmd, args)
	}
	if runtime.GOOS == "windows" {
		if cmd != "cmd" {
			t.Errorf("no windows esperava cmd, obteve %s", cmd)
		}
	}
}

func TestFormatFederatedURI(t *testing.T) {
	tests := []struct {
		repo     string
		docPath  string
		anchor   string
		expected string
	}{
		{
			repo:     "central",
			docPath:  "standards/oauth2",
			anchor:   "",
			expected: "memory://central/standards/oauth2",
		},
		{
			repo:     "central",
			docPath:  "/architecture/pgvector.md",
			anchor:   "Configuração",
			expected: "memory://central/architecture/pgvector.md#Configuração",
		},
		{
			repo:     "repo_1234567890ab",
			docPath:  "docs/api.md",
			anchor:   "#Rotas",
			expected: "memory://repo_1234567890ab/docs/api.md#Rotas",
		},
		{
			repo:     "org/project",
			docPath:  "",
			anchor:   "",
			expected: "memory://org/project",
		},
	}

	for _, tc := range tests {
		got := FormatFederatedURI(tc.repo, tc.docPath, tc.anchor)
		if got != tc.expected {
			t.Errorf("FormatFederatedURI(%q, %q, %q) = %q, esperava %q", tc.repo, tc.docPath, tc.anchor, got, tc.expected)
		}
	}
}

