package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/parser"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// newVaultNoteFixture grava uma nota de verdade no SQLite: documento + chunks
// produzidos pelo mesmo ChunkText que o indexador usa. Reproduzir o chunker
// aqui é o que permite comparar `mem read` com o arquivo byte a byte.
func newVaultNoteFixture(t *testing.T, docPath, content string) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "vault_fixture.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	if err := db.InsertDocumentWithMeta(ctx, database, docPath, docPath, "titulo", 1700000000, "hash", "epoch", "", "resource", 1700000000, int64(len(content))); err != nil {
		t.Fatalf("InsertDocumentWithMeta falhou: %v", err)
	}
	for i, chunk := range parser.ChunkText(content, 200, 30) {
		chunkID := fmt.Sprintf("%s#%d", docPath, i)
		if err := db.InsertChunk(ctx, database, chunkID, docPath, chunk, i, nil); err != nil {
			t.Fatalf("InsertChunk falhou: %v", err)
		}
	}
	return dbPath
}

func TestRunReadCommand_ImprimeMarkdownBruto(t *testing.T) {
	content := "---\ntitle: Nota de teste\n---\n\n# Cabeçalho\n\nTexto com [[wikilink]] e acentuação.\n"
	dbPath := newVaultNoteFixture(t, "docs/nota.md", content)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "docs/nota.md"}
	if err := runReadCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runReadCommand falhou: %v", err)
	}

	if buf.String() != content {
		t.Errorf("saída difere do arquivo byte a byte:\n--- esperado ---\n%q\n--- obtido ---\n%q", content, buf.String())
	}
}

// TestRunReadCommand_BasenameUnicoResuelve cobre o caminho feliz do atalho:
// sem a pasta, `x.md` ainda acha a nota quando há um único candidato.
func TestRunReadCommand_BasenameUnicoResolve(t *testing.T) {
	dbPath := newVaultNoteFixture(t, "docs/unica.md", "# Só uma\n")

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "unica.md"}
	if err := runReadCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runReadCommand falhou: %v", err)
	}
	if !strings.Contains(buf.String(), "Só uma") {
		t.Errorf("esperava o conteúdo da nota, obteve:\n%s", buf.String())
	}
}

// TestRunReadCommand_BasenameAmbiguoReporta é a regra de paridade do Obsidian:
// quando duas pastas têm uma nota com o mesmo nome, o comando diz qual é a
// ambiguidade. Escolher uma das duas em silêncio devolveria a nota errada.
func TestRunReadCommand_BasenameAmbiguoReporta(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "vault_fixture.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}

	ctx := context.Background()
	for _, path := range []string{"a/x.md", "b/x.md"} {
		if err := db.InsertDocumentWithMeta(ctx, database, path, path, "titulo", 1700000000, "hash-"+path, "epoch", "", "resource", 1700000000, 10); err != nil {
			t.Fatalf("InsertDocumentWithMeta falhou: %v", err)
		}
		if err := db.InsertChunk(ctx, database, path+"#0", path, "# x\n", 0, nil); err != nil {
			t.Fatalf("InsertChunk falhou: %v", err)
		}
	}
	database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "x.md"}
	err = runReadCommand(ctx, "fixture-repo", args, &buf)
	if err == nil {
		t.Fatalf("basename ambíguo deveria falhar, obteve saída: %s", buf.String())
	}

	msg := err.Error()
	if !strings.Contains(msg, "ambíguo") {
		t.Errorf("o erro precisa nomear a ambiguidade, obteve: %s", msg)
	}
	for _, path := range []string{"a/x.md", "b/x.md"} {
		if !strings.Contains(msg, path) {
			t.Errorf("o erro precisa listar o candidato %s, obteve: %s", path, msg)
		}
	}
	if strings.TrimSpace(buf.String()) != "" {
		t.Errorf("nada deveria ter sido impresso numa ambiguidade, obteve:\n%s", buf.String())
	}
}

func TestRunReadCommand_PathDesconhecidoFalha(t *testing.T) {
	dbPath := newVaultNoteFixture(t, "docs/nota.md", "# Nota\n")

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "docs/inexistente.md"}
	err := runReadCommand(context.Background(), "fixture-repo", args, &buf)
	if err == nil {
		t.Fatal("caminho ausente do índice deveria falhar")
	}
	if !strings.Contains(err.Error(), "docs/inexistente.md") {
		t.Errorf("o erro precisa nomear o caminho pedido, obteve: %v", err)
	}
}

func TestRunReadCommand_SemArgumentoFalha(t *testing.T) {
	dbPath := newVaultNoteFixture(t, "docs/nota.md", "# Nota\n")

	var buf bytes.Buffer
	err := runReadCommand(context.Background(), "fixture-repo", []string{"--db", dbPath, "--storage", "sqlite"}, &buf)
	if err == nil {
		t.Fatal("mem read sem caminho deveria falhar")
	}
}

// TestJoinVaultChunks_RemoveSobreposicao fecha o buraco do chunker: sem o
// dedupe, um documento longo sairia com as 30 palavras de overlap repetidas na
// emenda — e o usuário leria o mesmo parágrafo duas vezes.
func TestJoinVaultChunks_RemoveSobreposicao(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 600; i++ {
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "palavra%03d", i)
	}
	original := b.String()

	chunks := parser.ChunkText(original, 200, 30)
	if len(chunks) < 2 {
		t.Fatalf("fixture inválido: esperava múltiplos chunks, obteve %d", len(chunks))
	}

	joined := store.JoinVaultChunks(chunks)
	if joined != original {
		t.Errorf("a remontagem não devolveu o original.\nEsperado (%d runes): %q\nObtido   (%d runes): %q",
			len([]rune(original)), original, len([]rune(joined)), joined)
	}
}

func TestJoinVaultChunks_ChunkUnicoPreservaTexto(t *testing.T) {
	if got := store.JoinVaultChunks([]string{"# Título\n\ncorpo"}); got != "# Título\n\ncorpo" {
		t.Errorf("documento de um chunk só deveria sair intacto, obteve %q", got)
	}
	if got := store.JoinVaultChunks(nil); got != "" {
		t.Errorf("sem chunks o resultado deveria ser vazio, obteve %q", got)
	}
}
