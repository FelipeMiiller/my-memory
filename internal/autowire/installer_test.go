package autowire

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// TestInstaller_BackupEhPublicadoPorRename prova o AC25: o .bak é escrito num
// temporário no MESMO diretório do destino e publicado com um rename.
//
// A publicação é injetada com falha: é a falha que o .bak anterior precisa
// sobreviver, e é o que distingue "publicado por rename" de "escrito direto
// no lugar". Um mutante que troca o rename por `os.WriteFile` direto nem
// chega a publicar — e este teste morre.
func TestInstaller_BackupEhPublicadoPorRename(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "mcp.json")
	backupFile := configFile + ".bak"

	original := []byte("{\n    \"zetaGlobal\": true,\n    \"mcpServers\": {\n        \"other-server\": {}\n    }\n}\n")
	if err := os.WriteFile(configFile, original, 0644); err != nil {
		t.Fatalf("falha ao preparar o config inicial: %v", err)
	}
	// um .bak anterior existe: a falha da publicação tem que preservá-lo
	previousBackup := []byte("backup anterior")
	if err := os.WriteFile(backupFile, previousBackup, 0644); err != nil {
		t.Fatalf("falha ao preparar o .bak anterior: %v", err)
	}

	var tempPath string
	var tempData []byte
	renameFile = func(oldpath, newpath string) error {
		tempPath = oldpath
		tempData, _ = os.ReadFile(oldpath)
		return fmt.Errorf("falha simulada na publicação do backup")
	}
	t.Cleanup(func() { renameFile = os.Rename })

	target := ClientTarget{
		Type:       ClientCursor,
		Name:       "Cursor IDE (Workspace)",
		Scope:      ScopeWorkspace,
		ConfigPath: configFile,
	}

	if _, err := InjectMCPServer(target, "my-memory", ServerConfig{
		Command: "mem",
		Args:    []string{"mcp"},
	}, false, false); err != nil {
		t.Fatalf("falha ao injetar o servidor: %v", err)
	}

	// o backup foi preparado num temporário e só depois publicado
	if tempPath == "" {
		t.Fatalf("o .bak deveria ser publicado por rename (temporário + os.Rename)")
	}
	if filepath.Dir(tempPath) != filepath.Dir(backupFile) {
		t.Errorf("o temporário do backup deveria estar no mesmo diretório do destino: %q", tempPath)
	}
	if !bytes.Equal(tempData, original) {
		t.Errorf("o temporário do backup deveria segurar o backup completo antes da publicação:\nobtido:  %q\nesperado: %q", tempData, original)
	}

	// a publicação falhou: o .bak anterior permanece intacto
	currentBackup, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("falha ao ler o .bak: %v", err)
	}
	if !bytes.Equal(currentBackup, previousBackup) {
		t.Errorf("o .bak anterior deveria permanecer intacto quando a publicação falha:\nobtido:  %q\nesperado: %q", currentBackup, previousBackup)
	}
}

// TestInstaller_BackupEhByteIdenticoAoConfigSubstituido prova o AC26: o .bak
// final é byte-idêntico ao arquivo de configuração que ele substitui, e não
// sobra temporário no diretório.
func TestInstaller_BackupEhByteIdenticoAoConfigSubstituido(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "mcp.json")
	backupFile := configFile + ".bak"

	original := []byte("{\n    \"zetaGlobal\": true,\n    \"mcpServers\": {\n        \"other-server\": {}\n    }\n}\n")
	if err := os.WriteFile(configFile, original, 0644); err != nil {
		t.Fatalf("falha ao preparar o config inicial: %v", err)
	}

	target := ClientTarget{
		Type:       ClientCursor,
		Name:       "Cursor IDE (Workspace)",
		Scope:      ScopeWorkspace,
		ConfigPath: configFile,
	}

	report, err := InjectMCPServer(target, "my-memory", ServerConfig{
		Command: "mem",
		Args:    []string{"mcp", "--db", "custom.db"},
	}, false, false)
	if err != nil {
		t.Fatalf("falha ao injetar o servidor: %v", err)
	}

	if report.BackupPath != backupFile {
		t.Errorf("o report.BackupPath deveria apontar para o .bak final: '%s'", report.BackupPath)
	}

	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("falha ao ler o .bak: %v", err)
	}
	if !bytes.Equal(backupData, original) {
		t.Errorf("o .bak deveria ser byte-idêntico ao config substituído:\nobtido:  %q\nesperado: %q", backupData, original)
	}

	// o temporário não sobra no diretório
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("falha ao listar o diretório: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Errorf("o temporário do backup não deveria sobrar no diretório: %s", entry.Name())
		}
	}
}

// TestInstaller_FormatacaoCanonicaPreservaConteudo prova o AC27/AC28: o
// config escrito usa 2 espaços com as chaves de topo em ordem alfabética, e
// o conteúdo pré-existente (chaves de topo e servidores) é preservado.
func TestInstaller_FormatacaoCanonicaPreservaConteudo(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "mcp.json")

	// fixture proposital: 4 espaços e "zetaGlobal" antes de "alphaSetting"
	initial := `{
    "zetaGlobal": true,
    "alphaSetting": "valor",
    "mcpServers": {
        "other-server": {
            "command": "node",
            "args": ["server.js"]
        }
    }
}`
	if err := os.WriteFile(configFile, []byte(initial), 0644); err != nil {
		t.Fatalf("falha ao preparar o config inicial: %v", err)
	}

	target := ClientTarget{
		Type:       ClientCursor,
		Name:       "Cursor IDE (Workspace)",
		Scope:      ScopeWorkspace,
		ConfigPath: configFile,
	}

	report, err := InjectMCPServer(target, "my-memory", ServerConfig{
		Command: "mem",
		Args:    []string{"mcp"},
	}, false, false)
	if err != nil {
		t.Fatalf("falha ao injetar o servidor: %v", err)
	}
	if report.Status != StatusInstalled {
		t.Errorf("status inesperado: %s", report.Status)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("falha ao ler o config escrito: %v", err)
	}
	content := string(data)

	// AC27: 2 espaços de indentação nas chaves de topo
	for _, key := range []string{"alphaSetting", "mcpServers", "zetaGlobal"} {
		if !strings.Contains(content, "\n  \""+key+"\"") {
			t.Errorf("a chave de topo %q deveria sair com 2 espaços de indentação:\n%s", key, content)
		}
		if strings.Contains(content, "\n    \""+key+"\"") {
			t.Errorf("a chave de topo %q não deveria sair com 4 espaços de indentação:\n%s", key, content)
		}
	}
	if strings.Contains(content, "\t") {
		t.Errorf("o config escrito não deveria usar tabulação:\n%s", content)
	}

	// AC27: chaves de topo em ordem alfabética
	alphaIdx := strings.Index(content, "\"alphaSetting\"")
	mcpIdx := strings.Index(content, "\"mcpServers\"")
	zetaIdx := strings.Index(content, "\"zetaGlobal\"")
	if alphaIdx < 0 || mcpIdx < 0 || zetaIdx < 0 {
		t.Fatalf("o config escrito perdeu uma chave de topo:\n%s", content)
	}
	if !(alphaIdx < mcpIdx && mcpIdx < zetaIdx) {
		t.Errorf("as chaves de topo deveriam sair em ordem alfabética "+
			"(alphaSetting, mcpServers, zetaGlobal):\n%s", content)
	}

	// AC28: o conteúdo pré-existente é preservado
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON inválido no config escrito: %v", err)
	}
	if parsed["zetaGlobal"] != true {
		t.Errorf("zetaGlobal foi perdido")
	}
	if parsed["alphaSetting"] != "valor" {
		t.Errorf("alphaSetting foi perdido")
	}

	servers, ok := parsed["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("mcpServers ausente no config escrito")
	}
	if _, ok := servers["other-server"]; !ok {
		t.Errorf("o servidor pré-existente 'other-server' foi apagado")
	}
	if _, ok := servers["my-memory"]; !ok {
		t.Errorf("o servidor 'my-memory' não foi adicionado")
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
