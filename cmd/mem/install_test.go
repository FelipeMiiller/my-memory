package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/autowire"
)

// installFakeEnv é o probe de presença de instalação injetado na CLI.
//
// Sem ele, o resultado do `mem install` dependeria das ferramentas de IA
// instaladas na máquina que roda a suíte — e o teste passaria ou falharia
// conforme o desktop de quem rodou.
type installFakeEnv struct {
	goos    string
	home    string
	appData string
	dirs    map[string]bool
}

func (f *installFakeEnv) GOOS() string    { return f.goos }
func (f *installFakeEnv) HomeDir() string { return f.home }
func (f *installFakeEnv) AppData() string { return f.appData }

func (f *installFakeEnv) Stat(path string) (os.FileInfo, error) {
	if f.dirs[path] {
		return installFakeFileInfo{name: filepath.Base(path), isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

// markInstalled marca o marcador de instalação do cliente como presente.
func (f *installFakeEnv) markInstalled(t *testing.T, clientType autowire.ClientType) {
	t.Helper()

	marker := autowire.ClientInstallMarker(clientType, f)
	if marker == "" {
		t.Fatalf("marcador de instalação vazio para o cliente %q", clientType)
	}
	f.dirs[marker] = true
}

// newInstallFakeEnv monta um probe com home e AppData dentro de diretórios
// temporários: nenhum config real do usuário é lido ou escrito.
func newInstallFakeEnv(t *testing.T) *installFakeEnv {
	t.Helper()

	home := t.TempDir()
	return &installFakeEnv{
		goos:    runtime.GOOS,
		home:    home,
		appData: filepath.Join(home, "AppData", "Roaming"),
		dirs:    map[string]bool{},
	}
}

type installFakeFileInfo struct {
	name  string
	isDir bool
}

func (m installFakeFileInfo) Name() string       { return m.name }
func (m installFakeFileInfo) Size() int64        { return 0 }
func (m installFakeFileInfo) Mode() os.FileMode  { return 0755 }
func (m installFakeFileInfo) ModTime() time.Time { return time.Time{} }
func (m installFakeFileInfo) IsDir() bool        { return m.isDir }
func (m installFakeFileInfo) Sys() interface{}   { return nil }

func TestInstallCLI_DryRun(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	tempDir := t.TempDir()
	env := newInstallFakeEnv(t)
	env.markInstalled(t, autowire.ClientCursor)

	err := runInstallCommandWithEnv(ctx, "test-repo", []string{"--dry-run", "--workspace", "--dir", tempDir}, &buf, env)
	if err != nil {
		t.Fatalf("falha ao executar mem install --dry-run: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[DRY-RUN]") {
		t.Errorf("esperava indicador [DRY-RUN] na saída:\n%s", out)
	}
	if !strings.Contains(out, "Cursor IDE (Workspace)") {
		t.Errorf("esperava menção ao Cursor Workspace na saída:\n%s", out)
	}

	// Garantir que nenhum arquivo foi criado no disco
	cursorFile := filepath.Join(tempDir, ".cursor", "mcp.json")
	if _, err := os.Stat(cursorFile); !os.IsNotExist(err) {
		t.Errorf("arquivo não deveria ter sido criado durante dry-run: %s", cursorFile)
	}
}

func TestInstallCLI_WorkspaceExecution(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	tempDir := t.TempDir()
	env := newInstallFakeEnv(t)
	env.markInstalled(t, autowire.ClientCursor)

	err := runInstallCommandWithEnv(ctx, "test-repo", []string{
		"--workspace",
		"--target", "cursor",
		"--dir", tempDir,
		"--command", "C:/custom/mem.exe",
		"--db", "custom.db",
	}, &buf, env)
	if err != nil {
		t.Fatalf("falha ao executar mem install no workspace: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[INSTALADO]") {
		t.Errorf("esperava [INSTALADO] na saída:\n%s", out)
	}

	cursorFile := filepath.Join(tempDir, ".cursor", "mcp.json")
	data, err := os.ReadFile(cursorFile)
	if err != nil {
		t.Fatalf("falha ao ler arquivo gerado: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON inválido gerado: %v", err)
	}

	servers, ok := parsed["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("mcpServers ausente no JSON")
	}

	memSrv, ok := servers["my-memory"].(map[string]interface{})
	if !ok {
		t.Fatalf("my-memory ausente no mcpServers")
	}

	if memSrv["command"] != "C:/custom/mem.exe" {
		t.Errorf("command incorreto: %v", memSrv["command"])
	}
}

// TestInstallCLI_TargetRestringeAoClientePedido é a guarda que mata o mutante
// M7 ("a CLI ignora a flag --target").
//
// O lado POSITIVO do filtro — "o config do Cursor foi criado" — já era
// asserido. O que faltava era o lado NEGATIVO: com `--target cursor`, o VS
// Code não pode ser tocado nem aparecer na saída. Um mutante que detecta
// todo mundo passa no teste positivo e quebra neste.
func TestInstallCLI_TargetRestringeAoClientePedido(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	workspaceDir := t.TempDir()
	env := newInstallFakeEnv(t)
	env.markInstalled(t, autowire.ClientCursor)
	env.markInstalled(t, autowire.ClientVSCode)

	err := runInstallCommandWithEnv(ctx, "test-repo", []string{
		"--workspace",
		"--target", "cursor",
		"--dir", workspaceDir,
	}, &buf, env)
	if err != nil {
		t.Fatalf("falha ao executar mem install --target cursor --workspace: %v", err)
	}

	out := buf.String()

	// lado negativo do filtro de alvo
	vscodeFile := filepath.Join(workspaceDir, ".vscode", "mcp.json")
	if _, statErr := os.Stat(vscodeFile); !os.IsNotExist(statErr) {
		t.Errorf("--target cursor não deveria ter criado o config do VS Code: %s", vscodeFile)
	}
	if strings.Contains(out, "VS Code") {
		t.Errorf("a saída não deveria mencionar VS Code com --target cursor:\n%s", out)
	}

	// controle positivo: o cliente pedido foi configurado (o teste não é verde no vácuo)
	cursorFile := filepath.Join(workspaceDir, ".cursor", "mcp.json")
	if _, statErr := os.Stat(cursorFile); statErr != nil {
		t.Errorf("--target cursor deveria ter criado o config do Cursor: %v", statErr)
	}
}

// TestInstallCLI_GlobalNaoEscreveNoDirDoWorkspace é a guarda que mata o
// mutante M8 ("a CLI ignora --workspace/--global").
//
// Com o escopo global, o config vai para o home do usuário — nunca para dentro
// do `--dir` do repositório. Um mutante que resolve o escopo como "all"
// escreveria o `.cursor/mcp.json` e o `.vscode/mcp.json` dentro do `--dir`.
func TestInstallCLI_GlobalNaoEscreveNoDirDoWorkspace(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	workspaceDir := t.TempDir()
	env := newInstallFakeEnv(t)
	// todos os clientes instalados: só o filtro de escopo segura a escrita
	env.markInstalled(t, autowire.ClientClaudeDesktop)
	env.markInstalled(t, autowire.ClientCursor)
	env.markInstalled(t, autowire.ClientVSCode)
	env.markInstalled(t, autowire.ClientWindsurf)

	err := runInstallCommandWithEnv(ctx, "test-repo", []string{
		"--global",
		"--dir", workspaceDir,
	}, &buf, env)
	if err != nil {
		t.Fatalf("falha ao executar mem install --global: %v", err)
	}

	for _, rel := range []string{
		filepath.Join(".cursor", "mcp.json"),
		filepath.Join(".vscode", "mcp.json"),
	} {
		path := filepath.Join(workspaceDir, rel)
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Errorf("--global não deveria ter criado config de workspace dentro do --dir: %s", path)
		}
	}

	// controle positivo: o config global do Cursor foi criado no home do probe
	globalCursor := filepath.Join(env.home, ".cursor", "mcp.json")
	if _, statErr := os.Stat(globalCursor); statErr != nil {
		t.Errorf("--global deveria ter criado o config global do Cursor: %v", statErr)
	}
}

// TestInstallCLI_EscopoContraditorioFalha cobre o AC7: `--workspace` com
// `--global` juntas é erro de uso — a CLI devolve erro, e o `main` traduz em
// exit != 0. As flags são mutuamente exclusivas: a combinação não pode
// instalar em todos os escopos.
func TestInstallCLI_EscopoContraditorioFalha(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	workspaceDir := t.TempDir()
	env := newInstallFakeEnv(t)
	env.markInstalled(t, autowire.ClientCursor)

	err := runInstallCommandWithEnv(ctx, "test-repo", []string{
		"--workspace",
		"--global",
		"--dir", workspaceDir,
	}, &buf, env)
	if err == nil {
		t.Fatalf("--workspace com --global deveria falhar (exit != 0), obteve nil")
	}
	if !strings.Contains(err.Error(), "mutuamente exclusivas") {
		t.Errorf("a mensagem deveria dizer que as flags são mutuamente exclusivas: %v", err)
	}

	entries, readErr := os.ReadDir(workspaceDir)
	if readErr != nil {
		t.Fatalf("falha ao ler o --dir: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("nada deveria ter sido escrito no --dir, encontrou %d entrada(s)", len(entries))
	}
}

func TestRearrangeInstallArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "mixed flags with target and dry-run",
			input:    []string{"--dry-run", "--target", "cursor", "--workspace"},
			expected: []string{"--dry-run", "--target", "cursor", "--workspace"},
		},
		{
			name:     "flags with equals",
			input:    []string{"--target=vscode", "--dir=/tmp", "--force"},
			expected: []string{"--target=vscode", "--dir=/tmp", "--force"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rearrangeInstallArgs(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("tamanho diferente: obteve %d, esperava %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("índice %d: obteve '%s', esperava '%s'", i, got[i], tt.expected[i])
				}
			}
		})
	}
}
