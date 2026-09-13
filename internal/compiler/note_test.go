package compiler

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/parser"
)

func TestSafeResolvePath(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		vaultRoot   string
		path        string
		expectError bool
		expectedRel string
	}{
		{
			name:        "caminho relativo simples",
			vaultRoot:   tempDir,
			path:        "concepts/auth.md",
			expectError: false,
			expectedRel: "concepts/auth.md",
		},
		{
			name:        "caminho sem extensao adiciona .md",
			vaultRoot:   tempDir,
			path:        "decisions/001-login",
			expectError: false,
			expectedRel: "decisions/001-login.md",
		},
		{
			name:        "caminho na raiz do vault",
			vaultRoot:   tempDir,
			path:        "overview.md",
			expectError: false,
			expectedRel: "overview.md",
		},
		{
			name:        "caminho com barras invertidas normalizado",
			vaultRoot:   tempDir,
			path:        "sub\\nested\\nota.md",
			expectError: false,
			expectedRel: "sub/nested/nota.md",
		},
		{
			name:        "path traversal rejeitado",
			vaultRoot:   tempDir,
			path:        "../outside.md",
			expectError: true,
		},
		{
			name:        "path traversal profundo rejeitado",
			vaultRoot:   tempDir,
			path:        "sub/../../../../etc/passwd",
			expectError: true,
		},
		{
			name:        "caminho vazio rejeitado",
			vaultRoot:   tempDir,
			path:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abs, rel, err := SafeResolvePath(tt.vaultRoot, tt.path)
			if tt.expectError {
				if err == nil {
					t.Fatalf("esperava erro para '%s', mas obteve nil (abs=%s)", tt.path, abs)
				}
			} else {
				if err != nil {
					t.Fatalf("erro inesperado para '%s': %v", tt.path, err)
				}
				if rel != tt.expectedRel {
					t.Errorf("relPath esperado '%s', obteve '%s'", tt.expectedRel, rel)
				}
				if !strings.HasPrefix(abs, tempDir) {
					t.Errorf("caminho absoluto '%s' deve iniciar com tempDir '%s'", abs, tempDir)
				}
			}
		})
	}
}

func TestFormatFrontmatter(t *testing.T) {
	now := time.Date(2026, 9, 13, 18, 0, 0, 0, time.UTC)
	fm := FormatFrontmatter("Minha Nota", "decision", []string{"arquitetura", "#adr"}, []string{"alias1", "Nota Principal"}, now)

	if !strings.Contains(fm, "title: \"Minha Nota\"") {
		t.Errorf("frontmatter deve conter título: %s", fm)
	}
	if !strings.Contains(fm, "type: decision") {
		t.Errorf("frontmatter deve conter type: %s", fm)
	}
	if !strings.Contains(fm, "  - arquitetura") || !strings.Contains(fm, "  - adr") {
		t.Errorf("frontmatter deve conter tags normalizadas sem #: %s", fm)
	}
	if !strings.Contains(fm, "  - \"alias1\"") {
		t.Errorf("frontmatter deve conter aliases: %s", fm)
	}
	if !strings.Contains(fm, "created_at: 2026-09-13T18:00:00Z") {
		t.Errorf("frontmatter deve conter timestamp: %s", fm)
	}
}

func TestWriteAtomicNote(t *testing.T) {
	tempDir := t.TempDir()

	relPath := "decisions/001-cache.md"
	title := "Estratégia de Cache"
	content := "Implementação de cache L1 em memória e L2 em disco com [[WikilinkBasico]]."
	tags := []string{"cache", "performance"}
	aliases := []string{"CacheStrategy"}
	noteType := "decision"
	relations := []parser.EdgeConnection{
		{Target: "ADR-001", Relation: "implements"},
		{Target: "Redis", Relation: "depends_on"},
	}

	// 1. Criação com sucesso
	res, err := WriteAtomicNote(tempDir, relPath, title, content, tags, aliases, noteType, relations, false)
	if err != nil {
		t.Fatalf("erro ao gravar nota atômica: %v", err)
	}

	if res.Path != "decisions/001-cache.md" {
		t.Errorf("caminho retornado inválido: %s", res.Path)
	}
	if res.Overwritten {
		t.Errorf("não deveria marcar como sobrescrito na criação")
	}
	if res.SHA256 == "" {
		t.Errorf("hash SHA256 não deve ser vazio")
	}
	if len(res.Tags) == 0 {
		t.Errorf("deve conter tags extraídas")
	}

	// Verifica conteúdo em disco
	data, err := os.ReadFile(res.AbsPath)
	if err != nil {
		t.Fatalf("erro ao ler arquivo do disco: %v", err)
	}
	fileStr := string(data)

	if !strings.Contains(fileStr, "title: \"Estratégia de Cache\"") {
		t.Errorf("conteúdo deve conter título no frontmatter: %s", fileStr)
	}
	if !strings.Contains(fileStr, "[[rel:implements:ADR-001]]") {
		t.Errorf("conteúdo deve conter relação tipada: %s", fileStr)
	}
	if !strings.Contains(fileStr, "[[rel:depends_on:Redis]]") {
		t.Errorf("conteúdo deve conter relação tipada dependência: %s", fileStr)
	}

	// 2. Rejeição de sobrescrita sem overwrite=true
	_, err = WriteAtomicNote(tempDir, relPath, title, "Novo conteudo", nil, nil, "", nil, false)
	if err == nil {
		t.Fatalf("esperava erro ao tentar sobrescrever sem flag overwrite")
	}

	// 3. Sobrescrita autorizada com overwrite=true
	res2, err := WriteAtomicNote(tempDir, relPath, "Título Atualizado", "Conteúdo modificado com sucesso.", nil, nil, "summary", nil, true)
	if err != nil {
		t.Fatalf("erro inesperado ao sobrescrever com overwrite=true: %v", err)
	}
	if !res2.Overwritten {
		t.Errorf("deveria marcar como sobrescrito")
	}
	if res2.Title != "Título Atualizado" {
		t.Errorf("título deveria ser atualizado: %s", res2.Title)
	}
}

func TestWriteAtomicNote_TraversalBlocked(t *testing.T) {
	tempDir := t.TempDir()

	_, err := WriteAtomicNote(tempDir, "../../malicious.md", "Malicioso", "conteudo", nil, nil, "", nil, false)
	if err == nil {
		t.Fatalf("esperava rejeição por path traversal")
	}
}
