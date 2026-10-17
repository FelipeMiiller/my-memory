package store

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorReport_Serialization(t *testing.T) {
	report := DoctorReport{
		TotalDocuments: 10,
		TotalChunks:    45,
		TotalEdges:     18,
		TotalNodes:     15,
		HealthScore:    85,
		DeadLinks: []DeadLink{
			{SourceID: "notes/a.md", TargetID: "NotaInexistente", Relation: "links_to"},
		},
		OrphanNotes: []OrphanNote{
			{ID: "notes/orfa.md", Title: "orfa"},
		},
		SelfLoops: []SelfLoop{
			{NodeID: "notes/loop.md", Relation: "links_to"},
		},
		DesyncedChunks: []DesyncedChunk{},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Erro ao serializar DoctorReport: %v", err)
	}

	var decoded DoctorReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Erro ao deserializar DoctorReport: %v", err)
	}

	if decoded.TotalDocuments != 10 || decoded.HealthScore != 85 {
		t.Errorf("Dados de DoctorReport incorretos: %+v", decoded)
	}
	if len(decoded.DeadLinks) != 1 || decoded.DeadLinks[0].TargetID != "NotaInexistente" {
		t.Errorf("DeadLinks incorretos: %+v", decoded.DeadLinks)
	}
	if len(decoded.OrphanNotes) != 1 || decoded.OrphanNotes[0].Title != "orfa" {
		t.Errorf("OrphanNotes incorretos: %+v", decoded.OrphanNotes)
	}
	if len(decoded.SelfLoops) != 1 || decoded.SelfLoops[0].NodeID != "notes/loop.md" {
		t.Errorf("SelfLoops incorretos: %+v", decoded.SelfLoops)
	}
}

func TestIdentifyPrunedDocuments(t *testing.T) {
	storedPaths := []string{
		"C:/repo/notes/a.md",
		"C:/repo/notes/b.md",
		"C:/repo/notes/c.md",
		"C:/repo/docs/manual.md",
	}

	activeFiles := []string{
		"C:/repo/notes/a.md",
		"C:/repo/notes/c.md",
	}

	activeMap := make(map[string]bool)
	for _, f := range activeFiles {
		activeMap[filepath.Clean(f)] = true
	}

	cleanRoot := strings.ToLower(filepath.Clean("C:/repo/notes"))

	var toPrune []string
	for _, p := range storedPaths {
		cleanP := strings.ToLower(filepath.Clean(p))
		if strings.HasPrefix(cleanP, cleanRoot) {
			if !activeMap[filepath.Clean(p)] {
				toPrune = append(toPrune, p)
			}
		}
	}

	if len(toPrune) != 1 || toPrune[0] != "C:/repo/notes/b.md" {
		t.Fatalf("Esperava podar apenas 'notes/b.md', obteve: %v", toPrune)
	}
}

// ---------------------------------------------------------------------------
// Normalização de caminho compartilhada entre o pruning e a reconciliação
// ---------------------------------------------------------------------------

func TestNormalizePathKey(t *testing.T) {
	root := NormalizePathKey(t.TempDir())
	if root == "" {
		t.Fatal("a raiz do temp dir deveria produzir uma chave não vazia")
	}

	if got := NormalizePathKey(root + "\\Notas\\Alpha.MD"); got != root+"/notas/alpha.md" {
		t.Errorf("separador e caixa deveriam ser normalizados, obteve: %q", got)
	}
	if got := NormalizePathKey(root + "/notas/./alpha.md"); got != root+"/notas/alpha.md" {
		t.Errorf("caminho não limpo deveria ser normalizado, obteve: %q", got)
	}
	if got := NormalizePathKey(root + "/notas/sub/../alpha.md"); got != root+"/notas/alpha.md" {
		t.Errorf("'..' deveria ser resolvido, obteve: %q", got)
	}
	if got := NormalizePathKey("   "); got != "" {
		t.Errorf("caminho em branco deveria virar chave vazia, obteve: %q", got)
	}
	if got := NormalizePathKey(root + "/notas/alpha.md"); got != NormalizePathKey(root+`\Notas\Alpha.md`) {
		t.Errorf("formas equivalentes deveriam gerar a mesma chave: %q vs %q", got, NormalizePathKey(root+`\Notas\Alpha.md`))
	}
}

func TestPathWithinRoot_RespeitaFronteiraDeSegmento(t *testing.T) {
	root := NormalizePathKey(t.TempDir())

	if !pathWithinRoot(root, root+"/notas/a.md") {
		t.Error("arquivo dentro do vault deveria casar")
	}
	if !pathWithinRoot(root, root) {
		t.Error("a própria raiz deveria casar")
	}
	if pathWithinRoot(root, root+"-outro/notas/a.md") {
		t.Error("prefixo de string não pode vazar para diretório irmão")
	}
	if !pathWithinRoot("", "qualquer/coisa.md") {
		t.Error("raiz vazia significa 'sem recorte'")
	}
}

func TestIsProtectedPath(t *testing.T) {
	vault := t.TempDir()
	prefixes := normalizePrefixes(DefaultProtectedPathPrefixes)

	bruto := DocumentRef{ID: vault + "/catalog-skills/skills/x.md", Path: vault + "/catalog-skills/skills/x.md"}
	if !isProtectedPath(vault, bruto, prefixes) {
		t.Error("catalog-skills/** precisa estar protegido")
	}

	conhecimento := DocumentRef{ID: vault + "/notas/x.md", Path: vault + "/notas/x.md"}
	if isProtectedPath(vault, conhecimento, prefixes) {
		t.Error("notas/** não deveria estar protegido")
	}
}

// ---------------------------------------------------------------------------
// PruneDeletedDocuments (regressão: o helper extraído não pode quebrar)
// ---------------------------------------------------------------------------

func TestPruneDeletedDocuments_RemoveSomenteInativos(t *testing.T) {
	vault := t.TempDir()

	st, rec := newTestStore(t,
		refFor(vault, "notas/ativa.md", "Ativa"),
		refFor(vault, "notas/sumiu.md", "Sumiu"),
	)

	pruned, err := st.PruneDeletedDocuments(context.Background(), "repo", vault,
		[]string{filepath.Join(vault, "notas", "ativa.md")})
	if err != nil {
		t.Fatalf("PruneDeletedDocuments inesperado erro: %v", err)
	}

	if len(pruned) != 1 || pruned[0] != filepath.Join(vault, "notas", "sumiu.md") {
		t.Fatalf("esperava remover apenas 'sumiu.md', obteve: %v", pruned)
	}
	if rec.deleteCount() == 0 {
		t.Error("esperava DELETEs emitido para o documento removido")
	}
}

func TestPruneDeletedDocuments_RespeitaRecorteDeRoot(t *testing.T) {
	vault := t.TempDir()
	outroVault := filepath.Join(t.TempDir(), "outro")

	st, rec := newTestStore(t,
		refFor(vault, "notas/a.md", "A"),
		refFor(outroVault, "notas/b.md", "B"),
	)

	pruned, err := st.PruneDeletedDocuments(context.Background(), "repo", vault, nil)
	if err != nil {
		t.Fatalf("PruneDeletedDocuments inesperado erro: %v", err)
	}

	if len(pruned) != 1 || pruned[0] != filepath.Join(vault, "notas", "a.md") {
		t.Fatalf("documento de outro vault não pode ser tocado: %v", pruned)
	}
	for _, id := range rec.deleteIDs() {
		if strings.HasPrefix(id, outroVault) {
			t.Fatalf("DELETE fora do recorte: %s", id)
		}
	}
}

// ---------------------------------------------------------------------------
// PruneStaleDocuments
// ---------------------------------------------------------------------------

func TestPruneStaleDocuments_DryRunNaoDeleta(t *testing.T) {
	vault := t.TempDir()
	ativa := filepath.Join(vault, "notas", "ativa.md")
	obsoleta := filepath.Join(vault, "notas", "obsoleta.md")

	st, rec := newTestStore(t,
		DocumentRef{ID: ativa, Path: ativa, Title: "Ativa"},
		DocumentRef{ID: obsoleta, Path: obsoleta, Title: "Obsoleta"},
	)

	live := map[string]bool{NormalizePathKey(ativa): true}

	pruned, err := st.PruneStaleDocuments(context.Background(), "repo", vault, live, PruneOptions{DryRun: true, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("PruneStaleDocuments inesperado erro: %v", err)
	}

	if len(pruned) != 1 || pruned[0].ID != obsoleta || !pruned[0].DryRun {
		t.Fatalf("esperava 1 candidato marcado como dry-run, obteve: %+v", pruned)
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("dry-run emitiu %d DELETE(s)", got)
	}
}

func TestPruneStaleDocuments_PreservaPrefixoProtegido(t *testing.T) {
	vault := t.TempDir()
	conhecimento := filepath.Join(vault, "notas", "conhecimento.md")
	bruto := filepath.Join(vault, "catalog-skills", "skills", "bruto.md")
	obsoleta := filepath.Join(vault, "notas", "obsoleta.md")

	st, rec := newTestStore(t,
		DocumentRef{ID: conhecimento, Path: conhecimento},
		DocumentRef{ID: bruto, Path: bruto},
		DocumentRef{ID: obsoleta, Path: obsoleta},
	)

	live := map[string]bool{NormalizePathKey(conhecimento): true}

	pruned, err := st.PruneStaleDocuments(context.Background(), "repo", vault, live, PruneOptions{Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("PruneStaleDocuments inesperado erro: %v", err)
	}

	if len(pruned) != 1 || pruned[0].ID != obsoleta {
		t.Fatalf("material bruto protegido não pode entrar na lista: %+v", pruned)
	}
	for _, id := range rec.deleteIDs() {
		if id == bruto {
			t.Fatal("catalog-skills/** não pode ser removido do índice")
		}
	}
}

func TestPruneStaleDocuments_ExigeAncorasDeSeguranca(t *testing.T) {
	vault := t.TempDir()
	st, _ := newTestStore(t)

	if _, err := st.PruneStaleDocuments(context.Background(), "repo", "", map[string]bool{"x": true}, PruneOptions{}); err == nil {
		t.Error("rootDir vazio deveria ser erro")
	}
	if _, err := st.PruneStaleDocuments(context.Background(), "repo", vault, nil, PruneOptions{}); err == nil {
		t.Error("árvore viva vazia deveria ser erro (varredura de disco vazia aborta)")
	}
}

func TestListDocumentPaths(t *testing.T) {
	vault := t.TempDir()

	st, rec := newTestStore(t,
		refFor(vault, "notas/b.md", "B"),
		refFor(vault, "notas/a.md", "A"),
	)

	docs, err := st.ListDocumentPaths(context.Background(), "repo")
	if err != nil {
		t.Fatalf("ListDocumentPaths inesperado erro: %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("esperava 2 documentos, obteve: %d", len(docs))
	}
	if docs[0].ID == "" || docs[1].ID == "" {
		t.Errorf("inventário deveria devolver id de cada documento: %+v", docs)
	}
	if docs[0].Title == "" {
		t.Error("título deveria ser devolvido")
	}
	if rec.deleteCount() != 0 {
		t.Error("ListDocumentPaths é somente-leitura")
	}
}
