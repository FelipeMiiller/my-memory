package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallAndUninstallGitHook(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create fake .git dir: %v", err)
	}

	// 1. Instalar hook pela primeira vez
	hookPath, err := InstallGitHook(tempDir, false)
	if err != nil {
		t.Fatalf("expected successful install, got: %v", err)
	}

	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read created hook: %v", err)
	}
	if !strings.Contains(string(content), HookSignature) {
		t.Fatalf("hook does not contain signature: %s", string(content))
	}

	// 2. Re-instalação idempotente (já tem assinatura)
	_, err = InstallGitHook(tempDir, false)
	if err != nil {
		t.Fatalf("expected idempotent re-install to succeed, got: %v", err)
	}

	// 3. Desinstalar hook existente
	if err := UninstallGitHook(tempDir); err != nil {
		t.Fatalf("expected successful uninstall, got: %v", err)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatalf("expected hook file to be removed, but it still exists")
	}

	// 4. Desinstalar quando já não existe deve ser no-op sem erro
	if err := UninstallGitHook(tempDir); err != nil {
		t.Fatalf("expected uninstall to be idempotent when file does not exist, got: %v", err)
	}
}

func TestForeignHookProtectionAndForce(t *testing.T) {
	tempDir := t.TempDir()
	gitHooksDir := filepath.Join(tempDir, ".git", "hooks")
	if err := os.MkdirAll(gitHooksDir, 0755); err != nil {
		t.Fatalf("failed to create fake .git/hooks: %v", err)
	}

	foreignPath := filepath.Join(gitHooksDir, "pre-commit")
	foreignContent := "#!/bin/sh\necho 'foreign linter running'\nexit 0\n"
	if err := os.WriteFile(foreignPath, []byte(foreignContent), 0755); err != nil {
		t.Fatalf("failed to write foreign hook: %v", err)
	}

	// 1. Tentar instalar sem --force deve falhar para proteger hook alheio
	_, err := InstallGitHook(tempDir, false)
	if err == nil {
		t.Fatalf("expected error installing over foreign hook without force, got nil")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected error message to mention --force, got: %v", err)
	}

	// 2. Tentar desinstalar hook alheio deve falhar
	err = UninstallGitHook(tempDir)
	if err == nil {
		t.Fatalf("expected error uninstalling foreign hook, got nil")
	}

	// 3. Instalar com --force deve sobrescrever
	hookPath, err := InstallGitHook(tempDir, true)
	if err != nil {
		t.Fatalf("expected force install to succeed, got: %v", err)
	}

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read hook: %v", err)
	}
	if !strings.Contains(string(data), HookSignature) {
		t.Fatalf("expected overwritten hook to have signature")
	}

	// 4. Agora que tem a assinatura, uninstall deve suceder
	if err := UninstallGitHook(tempDir); err != nil {
		t.Fatalf("expected uninstall to succeed after overwrite, got: %v", err)
	}
}

func TestInstallGitHookNoGitDir(t *testing.T) {
	tempDir := t.TempDir() // sem .git
	_, err := InstallGitHook(tempDir, false)
	if err == nil {
		t.Fatalf("expected error when .git not found, got nil")
	}
}
