package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// vaultFixtureDoc é um documento do fixture de mtime, com metadados explícitos
// para que o teste controle a ordem esperada.
type vaultFixtureDoc struct {
	path  string
	title string
	mtime int64
	size  int64
}

// newMTTimeFixture grava documentos com mtime controlado.
func newMTTimeFixture(t *testing.T, docs ...vaultFixtureDoc) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "vault_fixture.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	for _, doc := range docs {
		err := db.InsertDocumentWithMeta(ctx, database, doc.path, doc.path, doc.title, 1700000000, "hash-"+doc.path, "epoch", "", "resource", doc.mtime, doc.size)
		if err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s) falhou: %v", doc.path, err)
		}
	}
	return dbPath
}

func TestRunFileCommand_RelataMetadados(t *testing.T) {
	edited := time.Date(2026, 3, 14, 15, 9, 26, 0, time.UTC).Unix()
	dbPath := newMTTimeFixture(t, vaultFixtureDoc{
		path:  "docs/adr/001.md",
		title: "ADR 001",
		mtime: edited,
		size:  2048,
	})

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "docs/adr/001.md"}
	if err := runFileCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runFileCommand falhou: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"Path:    docs/adr/001.md",
		"Title:   ADR 001",
		"Size:    2048 bytes",
		"MTime:   2026-03-14T15:09:26Z",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("esperava %q na saída, obteve:\n%s", want, out)
		}
	}
}

// TestRunFileCommand_MTimeZeroEDesconhecido: um vault indexado antes da
// ADR-057 tem mtime = 0 em toda linha. Reportar isso como desconhecido é o que
// diferencia "não editado" de "não sabemos quando foi editado".
func TestRunFileCommand_MTimeZeroEDesconhecido(t *testing.T) {
	dbPath := newMTTimeFixture(t, vaultFixtureDoc{path: "legado.md", title: "Legado", mtime: 0, size: 10})

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "legado.md"}
	if err := runFileCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runFileCommand falhou: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "MTime:   desconhecido") {
		t.Errorf("mtime = 0 deveria sair como desconhecido, obteve:\n%s", out)
	}
	if strings.Contains(out, "1970") {
		t.Errorf("mtime = 0 não pode ser apresentado como 1970, obteve:\n%s", out)
	}
}

func TestRunFileCommand_SemArgumentoFalha(t *testing.T) {
	dbPath := newMTTimeFixture(t, vaultFixtureDoc{path: "docs/a.md", title: "A", mtime: 1, size: 1})

	var buf bytes.Buffer
	err := runFileCommand(context.Background(), "fixture-repo", []string{"--db", dbPath, "--storage", "sqlite"}, &buf)
	if err == nil {
		t.Fatal("mem file sem caminho deveria falhar")
	}
}

func TestRunRecentsCommand_OrdenaPorMTimeDecrescente(t *testing.T) {
	dbPath := newMTTimeFixture(t,
		vaultFixtureDoc{path: "antiga.md", title: "Antiga", mtime: 1000, size: 1},
		vaultFixtureDoc{path: "recente.md", title: "Recente", mtime: 9000, size: 1},
		vaultFixtureDoc{path: "media.md", title: "Media", mtime: 5000, size: 1},
	)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite"}
	if err := runRecentsCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runRecentsCommand falhou: %v", err)
	}

	got := recentsOrder(buf.String())
	want := []string{"recente.md", "media.md", "antiga.md"}
	if len(got) != len(want) {
		t.Fatalf("esperava %v, obteve %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q", i, want[i], got[i])
		}
	}
}

// TestRunRecentsCommand_MTimeZeroVaiParaOFim é o critério que separa "mais
// recente" de "zero побеждает": sem o CASE na ordenação, um documento com
// mtime = 0 abriria a lista dos recentes, que é o oposto do que o comando
// promete sobre a ordem temporal.
func TestRunRecentsCommand_MTimeZeroVaiParaOFim(t *testing.T) {
	dbPath := newMTTimeFixture(t,
		vaultFixtureDoc{path: "legado.md", title: "Legado", mtime: 0, size: 1},
		vaultFixtureDoc{path: "antiga.md", title: "Antiga", mtime: 1000, size: 1},
	)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite"}
	if err := runRecentsCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runRecentsCommand falhou: %v", err)
	}

	got := recentsOrder(buf.String())
	want := []string{"antiga.md", "legado.md"}
	if len(got) != len(want) {
		t.Fatalf("esperava %v, obteve %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q", i, want[i], got[i])
		}
	}
}

func TestRunRecentsCommand_RespeitaLimit(t *testing.T) {
	dbPath := newMTTimeFixture(t,
		vaultFixtureDoc{path: "a.md", title: "A", mtime: 3000, size: 1},
		vaultFixtureDoc{path: "b.md", title: "B", mtime: 2000, size: 1},
		vaultFixtureDoc{path: "c.md", title: "C", mtime: 1000, size: 1},
	)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "--limit", "2"}
	if err := runRecentsCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runRecentsCommand --limit falhou: %v", err)
	}

	got := recentsOrder(buf.String())
	want := []string{"a.md", "b.md"}
	if len(got) != len(want) {
		t.Fatalf("esperava %v, obteve %v", want, got)
	}
}

func TestRunRecentsCommand_LimitInvalidoFalha(t *testing.T) {
	dbPath := newMTTimeFixture(t, vaultFixtureDoc{path: "a.md", title: "A", mtime: 1, size: 1})

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "--limit", "0"}
	if err := runRecentsCommand(context.Background(), "fixture-repo", args, &buf); err == nil {
		t.Fatal("--limit 0 deveria falhar")
	}
}

// recentsOrder extrai os caminhos da saída de `mem recents`, que traz o mtime
// seguido do caminho na mesma linha.
func recentsOrder(out string) []string {
	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		paths = append(paths, fields[len(fields)-1])
	}
	return paths
}
