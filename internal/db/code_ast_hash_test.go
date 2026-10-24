package db

import (
	"context"
	"path/filepath"
	"testing"
)

// TestAstHashEnabledDefault cobre o default da política de ast_hash: sem
// nada no contexto, o ast_hash é armazenado (comportamento histórico).
func TestAstHashEnabledDefault(t *testing.T) {
	if !AstHashEnabled(context.Background()) {
		t.Error("AstHashEnabled(ctx vazio) = false; want true (o default armazena o ast_hash)")
	}
	if !AstHashEnabled(nil) {
		t.Error("AstHashEnabled(nil) = false; want true")
	}
}

// TestAstHashEnabledDesligado cobre --ast-hash=false: a política desligada
// viaja no contexto até o helper de escrita.
func TestAstHashEnabledDesligado(t *testing.T) {
	ctx := WithAstHash(context.Background(), false)
	if AstHashEnabled(ctx) {
		t.Error("AstHashEnabled(WithAstHash(false)) = true; want false")
	}

	on := WithAstHash(context.Background(), true)
	if !AstHashEnabled(on) {
		t.Error("AstHashEnabled(WithAstHash(true)) = false; want true")
	}
}

// TestUpsertCodeFileRespeitaAstHashDesligado prova o efeito da política:
// com ela desligada, o ast_hash não é persistido.
func TestUpsertCodeFileRespeitaAstHashDesligado(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ast_hash.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	if err := EnsureCodeTables(context.Background(), d); err != nil {
		t.Fatalf("EnsureCodeTables falhou: %v", err)
	}

	ctx := WithAstHash(context.Background(), false)
	if _, err := UpsertCodeFile(ctx, d, "a/b.go", "go", "h", "ast-1", 10, 1); err != nil {
		t.Fatalf("UpsertCodeFile falhou: %v", err)
	}

	var stored string
	if err := d.QueryRow(`SELECT COALESCE(ast_hash, '') FROM code_files LIMIT 1`).Scan(&stored); err != nil {
		t.Fatalf("SELECT ast_hash falhou: %v", err)
	}
	if stored != "" {
		t.Errorf("ast_hash = %q; want vazio/NULL com a política desligada", stored)
	}

	// Com o default, o ast_hash é guardado.
	if _, err := UpsertCodeFile(context.Background(), d, "a/c.go", "go", "h2", "ast-2", 10, 1); err != nil {
		t.Fatalf("UpsertCodeFile falhou: %v", err)
	}
	var storedOn string
	if err := d.QueryRow(`SELECT COALESCE(ast_hash, '') FROM code_files WHERE path = 'a/c.go'`).Scan(&storedOn); err != nil {
		t.Fatalf("SELECT ast_hash falhou: %v", err)
	}
	if storedOn != "ast-2" {
		t.Errorf("ast_hash = %q; want ast-2 com a política padrão", storedOn)
	}
}
