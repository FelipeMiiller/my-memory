package autowire

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectMCPServer_NewFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "subdir", "config.json")

	target := ClientTarget{
		Type:       ClientCursor,
		Name:       "Cursor Workspace",
		Scope:      ScopeWorkspace,
		ConfigPath: configFile,
	}

	cfg := ServerConfig{
		Command: "C:/bin/mem.exe",
		Args:    []string{"mcp", "--db", "custom.db"},
	}

	report, err := InjectMCPServer(target, "my-memory", cfg, false, false)
	if err != nil {
		t.Fatalf("falha ao injetar servidor em novo arquivo: %v", err)
	}

	if report.Status != StatusInstalled {
		t.Errorf("esperava StatusInstalled, obteve %s", report.Status)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("arquivo não foi criado: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON gerado inválido: %v", err)
	}

	servers, ok := parsed["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("mcpServers ausente no JSON gerado")
	}

	memSrv, ok := servers["my-memory"].(map[string]interface{})
	if !ok {
		t.Fatalf("my-memory ausente em mcpServers")
	}

	if memSrv["command"] != "C:/bin/mem.exe" {
		t.Errorf("command incorreto: %v", memSrv["command"])
	}
}

func TestInjectMCPServer_PreserveExistingServers(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "claude.json")

	initialJSON := `{
  "globalSetting": true,
  "mcpServers": {
    "other-server": {
      "command": "node",
      "args": ["server.js"]
    }
  }
}`
	if err := os.WriteFile(configFile, []byte(initialJSON), 0644); err != nil {
		t.Fatalf("falha ao preparar arquivo inicial: %v", err)
	}

	target := ClientTarget{
		Type:       ClientClaudeDesktop,
		Name:       "Claude Desktop",
		Scope:      ScopeGlobal,
		ConfigPath: configFile,
		Exists:     true,
	}

	cfg := ServerConfig{
		Command: "mem",
		Args:    []string{"mcp"},
	}

	report, err := InjectMCPServer(target, "my-memory", cfg, false, false)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}

	if report.Status != StatusInstalled && report.Status != StatusUpdated {
		t.Errorf("status inesperado: %s", report.Status)
	}

	// Verificar se o backup foi criado
	if report.BackupPath == "" {
		t.Errorf("esperava criação de backup .bak")
	}
	if _, err := os.Stat(configFile + ".bak"); err != nil {
		t.Errorf("arquivo de backup não encontrado no disco: %v", err)
	}

	// Verificar se o outro servidor foi mantido
	data, _ := os.ReadFile(configFile)
	var parsed map[string]interface{}
	_ = json.Unmarshal(data, &parsed)

	if parsed["globalSetting"] != true {
		t.Errorf("globalSetting foi perdido")
	}

	servers := parsed["mcpServers"].(map[string]interface{})
	if _, ok := servers["other-server"]; !ok {
		t.Errorf("other-server foi apagado acidentalmente!")
	}
	if _, ok := servers["my-memory"]; !ok {
		t.Errorf("my-memory não foi adicionado!")
	}
}

func TestInjectMCPServer_Idempotent(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")

	target := ClientTarget{
		Type:       ClientVSCode,
		Name:       "VSCode",
		Scope:      ScopeWorkspace,
		ConfigPath: configFile,
	}

	cfg := ServerConfig{
		Command: "mem",
		Args:    []string{"mcp"},
	}

	// 1. Primeira injeção
	_, err := InjectMCPServer(target, "my-memory", cfg, false, false)
	if err != nil {
		t.Fatalf("primeira injeção falhou: %v", err)
	}

	// 2. Segunda injeção idêntica
	report, err := InjectMCPServer(target, "my-memory", cfg, false, false)
	if err != nil {
		t.Fatalf("segunda injeção falhou: %v", err)
	}

	if report.Status != StatusAlreadyUpToDate {
		t.Errorf("esperava StatusAlreadyUpToDate na repetição, obteve %s", report.Status)
	}

	// 3. Terceira injeção com force = true
	forceReport, err := InjectMCPServer(target, "my-memory", cfg, false, true)
	if err != nil {
		t.Fatalf("injeção forçada falhou: %v", err)
	}
	if forceReport.Status != StatusUpdated {
		t.Errorf("esperava StatusUpdated com force=true, obteve %s", forceReport.Status)
	}
}

func TestInjectMCPServer_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")

	target := ClientTarget{
		Type:       ClientWindsurf,
		Name:       "Windsurf",
		Scope:      ScopeGlobal,
		ConfigPath: configFile,
	}

	cfg := ServerConfig{
		Command: "mem",
		Args:    []string{"mcp"},
	}

	report, err := InjectMCPServer(target, "my-memory", cfg, true, false)
	if err != nil {
		t.Fatalf("dry-run falhou: %v", err)
	}

	if report.Status != StatusDryRun {
		t.Errorf("esperava StatusDryRun, obteve %s", report.Status)
	}

	if report.Diff == "" {
		t.Errorf("esperava Diff preenchido em dry-run")
	}

	// Verificar que nada foi gravado no disco
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		t.Errorf("arquivo não deveria existir em dry-run")
	}
}

func TestInjectMCPServer_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "corrupt.json")

	_ = os.WriteFile(configFile, []byte(`{ invalid json ...`), 0644)

	target := ClientTarget{
		Type:       ClientCursor,
		Name:       "Cursor",
		ConfigPath: configFile,
	}

	_, err := InjectMCPServer(target, "my-memory", ServerConfig{}, false, false)
	if err == nil {
		t.Fatalf("esperava erro para JSON corrompido, obteve nil")
	}
	if !strings.Contains(err.Error(), "JSON inválido") {
		t.Errorf("mensagem de erro inesperada: %v", err)
	}
}
