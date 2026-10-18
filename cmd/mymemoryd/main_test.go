package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/supervisor"
)

// Os testes abaixo cobrem apenas os caminhos que RETORNAM antes de
// mgr.Run(ctx), que bloqueia até o contexto ser cancelado por SIGINT/SIGTERM
// (internal/supervisor.Manager.Run). Disparar sinal dentro do processo de
// teste é impraticável no Windows, então o caminho feliz fica para o smoke
// test de binário; o que importa aqui é o contrato fail-closed do ADR-050
// (LLM10): entrada inválida sai com código 2, nunca com fallback silencioso.

// writeProfile cria <memoryDir>/profiles/<name>.yaml com conteúdo mínimo.
func writeProfile(t *testing.T, memoryDir, name string) {
	t.Helper()

	dir := filepath.Join(memoryDir, "profiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("erro ao criar %s: %v", dir, err)
	}
	path := filepath.Join(dir, name+".yaml")
	if err := os.WriteFile(path, []byte("name: "+name+"\n"), 0o644); err != nil {
		t.Fatalf("erro ao escrever %s: %v", path, err)
	}
}

func TestRun_FlagDesconhecida(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"--nao-existe"}, &stdout, &stderr)

	if code != exitGenericError {
		t.Errorf("esperava exit %d para flag desconhecida, obteve %d", exitGenericError, code)
	}
	if stdout.Len() != 0 {
		t.Errorf("esperava stdout vazio, obteve %q", stdout.String())
	}
}

// Perfil ausente tem de sair com 2 (não 1): é entrada inválida para o
// supervisor, não falha genérica. Um vault sem profiles/ não pode virar um
// supervisor rodando com defaults silenciosos.
func TestRun_PerfilAusente(t *testing.T) {
	memoryDir := t.TempDir()

	var stdout, stderr bytes.Buffer
	code := run([]string{"--memory-dir", memoryDir}, &stdout, &stderr)

	if code != exitInvalidProfile {
		t.Errorf("esperava exit %d para perfil ausente, obteve %d", exitInvalidProfile, code)
	}
	if !strings.Contains(stderr.String(), "profile") {
		t.Errorf("esperava diagnóstico de perfil no stderr, obteve %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("esperava stdout vazio, obteve %q", stdout.String())
	}
}

// Lock fresco com PID vivo é outra instância: recusa com 1 em vez de
// sobrescrever o lock alheio. Usa o próprio PID do processo de teste, que
// está vivo por definição.
func TestRun_LockOcupadoPorProcessoVivo(t *testing.T) {
	memoryDir := t.TempDir()
	writeProfile(t, memoryDir, "default")

	lock, err := supervisor.AcquireLock(memoryDir, "default")
	if err != nil {
		t.Fatalf("erro ao adquirir lock de teste: %v", err)
	}
	if lock.PID != os.Getpid() {
		t.Fatalf("esperava lock com PID %d, obteve %d", os.Getpid(), lock.PID)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--memory-dir", memoryDir}, &stdout, &stderr)

	if code != exitGenericError {
		t.Errorf("esperava exit %d com lock ocupado, obteve %d", exitGenericError, code)
	}
	if !strings.Contains(stderr.String(), "already running") {
		t.Errorf("esperava diagnóstico de instância concorrente no stderr, obteve %q", stderr.String())
	}
}
