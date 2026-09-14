package compiler

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/parser"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

func TestSyncEngine_SQLite(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	dbPath := filepath.Join(tempDir, "memory.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "requires cgo to work") || strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("Pulando teste SQLite: ambiente sem CGO ou FTS5")
		}
		t.Fatalf("erro ao inicializar SQLite: %v", err)
	}
	defer database.Close()

	tq := turboquant.NewQuantizer(768)
	syncEngine := NewSyncEngine(database, nil, nil, tq, "local/vault")

	// 1. Gravar e Sincronizar Nota
	noteRes, indexRes, err := WriteAndSyncNote(
		ctx,
		syncEngine,
		tempDir,
		"concepts/auth-architecture.md",
		"Autenticação e Sessões",
		"Conteúdo da nota descrevendo fluxo de JWT e OAuth2 com [[ADR-001]].",
		[]string{"auth", "security"},
		[]string{"AuthArch"},
		"concept",
		[]parser.EdgeConnection{{Target: "ADR-001", Relation: "implements"}},
		false,
	)
	if err != nil {
		t.Fatalf("falha em WriteAndSyncNote: %v", err)
	}

	if noteRes.Path != "concepts/auth-architecture.md" {
		t.Errorf("caminho da nota incorreto: %s", noteRes.Path)
	}
	if indexRes.Action != "indexed" {
		t.Errorf("ação esperada 'indexed', obteve '%s'", indexRes.Action)
	}

	// Verifica se foi indexado no banco SQLite
	hash, err := db.GetDocumentHash(ctx, database, noteRes.AbsPath)
	if err != nil {
		t.Fatalf("falha ao consultar hash do documento no SQLite: %v", err)
	}
	if hash == "" {
		t.Fatalf("documento não foi persistido no banco SQLite")
	}

	// 2. Apensar Seção e Sincronizar
	appNote, appIdx, err := AppendAndSyncSection(
		ctx,
		syncEngine,
		tempDir,
		"concepts/auth-architecture.md",
		"## Atualizações Recentes",
		"Adicionado suporte a Refresh Tokens.",
		true,
	)
	if err != nil {
		t.Fatalf("falha em AppendAndSyncSection: %v", err)
	}
	if appIdx.Action != "indexed" {
		t.Errorf("ação esperada 'indexed' após append, obteve '%s'", appIdx.Action)
	}
	if appNote.Path != "concepts/auth-architecture.md" {
		t.Errorf("caminho da nota inconsistente: %s", appNote.Path)
	}

	// 3. Compilar Tópico e Sincronizar
	sources := []CompiledSource{
		{
			DocPath: noteRes.AbsPath,
			Title:   "Autenticação e Sessões",
			Content: "Fluxo de JWT e OAuth2.",
			Score:   0.95,
		},
	}
	compNote, compIdx, err := CompileAndSyncTopicNote(
		ctx,
		syncEngine,
		tempDir,
		"syntheses/auth-synthesis.md",
		"Síntese de Autenticação",
		"mecanismos de login",
		sources,
		[]string{"auth"},
		nil,
		false,
	)
	if err != nil {
		t.Fatalf("falha em CompileAndSyncTopicNote: %v", err)
	}
	if compIdx.Action != "indexed" {
		t.Errorf("ação esperada 'indexed' em compilação, obteve '%s'", compIdx.Action)
	}

	// Confirma presença da nota compilada no banco
	compHash, err := db.GetDocumentHash(ctx, database, compNote.AbsPath)
	if err != nil || compHash == "" {
		t.Fatalf("nota compilada não encontrada no SQLite: %v", err)
	}

}

func TestSyncEngine_Nil(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// 1. SyncFile com engine nil
	skipRes, err := (*SyncEngine)(nil).SyncFile(ctx, "qualquer-caminho.md")
	if err != nil {
		t.Fatalf("não deve retornar erro para SyncEngine nil: %v", err)
	}
	if skipRes.Action != "skipped" {
		t.Errorf("ação esperada 'skipped', obteve '%s'", skipRes.Action)
	}

	// 2. WriteAndSyncNote com engine nil
	noteRes, idxRes, err := WriteAndSyncNote(ctx, nil, tempDir, "standalone.md", "Nota Standalone", "Conteudo", nil, nil, "concept", nil, false)
	if err != nil {
		t.Fatalf("falha ao gravar nota com engine nil: %v", err)
	}
	if noteRes.Path != "standalone.md" {
		t.Errorf("caminho incorreto: %s", noteRes.Path)
	}
	if idxRes.Action != "skipped" {
		t.Errorf("esperava action 'skipped', obteve '%s'", idxRes.Action)
	}
}
