package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// ---------------------------------------------------------------------------
// Infra de teste: um database/sql driver que registra TODOS os comandos
// emitidos. Isso permite provar, contra um repositório real em disco e o
// pipeline real do database/sql, que nenhum DELETE saiu — sem depender de um
// PostgreSQL vivo e sem mockear o código sob teste.
// ---------------------------------------------------------------------------

type stmtRecord struct {
	Query string
	Args  []string
}

type recorder struct {
	mu    sync.Mutex
	stmts []stmtRecord
	rows  []DocumentRef
}

func (r *recorder) record(query string, args []driver.NamedValue) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec := stmtRecord{Query: query}
	for _, a := range args {
		if v, ok := a.Value.(string); ok {
			rec.Args = append(rec.Args, v)
		} else {
			rec.Args = append(rec.Args, fmt.Sprintf("%v", a.Value))
		}
	}
	r.stmts = append(r.stmts, rec)
}

// deleteIDs devolve o documento de cada DELETE emitido (último argumento).
func (r *recorder) deleteIDs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var ids []string
	for _, s := range r.stmts {
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(s.Query)), "DELETE") {
			continue
		}
		if len(s.Args) >= 2 {
			ids = append(ids, s.Args[1])
		}
	}
	return ids
}

func (r *recorder) deleteCount() int {
	return len(r.deleteIDs())
}

func (r *recorder) queryCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.stmts)
}

type recConnector struct{ rec *recorder }

func (c recConnector) Connect(context.Context) (driver.Conn, error) {
	return &recConn{rec: c.rec}, nil
}

func (c recConnector) Driver() driver.Driver { return recDriver{} }

type recDriver struct{}

func (recDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use sql.OpenDB com recConnector")
}

type recConn struct{ rec *recorder }

func (c *recConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare não suportado neste driver de teste")
}

func (c *recConn) Close() error { return nil }

func (c *recConn) Begin() (driver.Tx, error) {
	c.rec.record("BEGIN", nil)
	return &recTx{rec: c.rec}, nil
}

func (c *recConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.rec.record(query, args)

	upper := strings.ToUpper(strings.TrimSpace(query))
	if !strings.HasPrefix(upper, "SELECT") {
		return nil, fmt.Errorf("query não-SELECT inesperada no teste: %s", query)
	}

	cols := []string{"id", "path"}
	if strings.Contains(upper, "TITLE") {
		cols = []string{"id", "path", "title"}
	}

	c.rec.mu.Lock()
	data := append([]DocumentRef(nil), c.rec.rows...)
	c.rec.mu.Unlock()

	return &recRows{cols: cols, data: data}, nil
}

func (c *recConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.rec.record(query, args)
	return driver.RowsAffected(0), nil
}

type recTx struct{ rec *recorder }

func (t *recTx) Commit() error {
	t.rec.record("COMMIT", nil)
	return nil
}

func (t *recTx) Rollback() error {
	t.rec.record("ROLLBACK", nil)
	return nil
}

type recRows struct {
	cols []string
	data []DocumentRef
	pos  int
}

func (r *recRows) Columns() []string { return r.cols }

func (r *recRows) Close() error { return nil }

func (r *recRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	doc := r.data[r.pos]
	r.pos++

	if len(dest) >= 3 {
		dest[0], dest[1], dest[2] = doc.ID, doc.Path, doc.Title
		return nil
	}
	dest[0], dest[1] = doc.ID, doc.Path
	return nil
}

// ---------------------------------------------------------------------------
// Helpers de vault
// ---------------------------------------------------------------------------

// newTestStore monta um PostgresStore sobre o driver registrador.
func newTestStore(t *testing.T, docs ...DocumentRef) (*PostgresStore, *recorder) {
	t.Helper()

	rec := &recorder{rows: docs}
	db := sql.OpenDB(recConnector{rec: rec})
	t.Cleanup(func() { _ = db.Close() })

	return &PostgresStore{db: db}, rec
}

// writeVaultFile grava um arquivo real no vault de teste.
func writeVaultFile(t *testing.T, root, rel string) string {
	t.Helper()

	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("não foi possível criar diretório para %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte("# "+rel+"\n\nconteúdo de teste\n"), 0o644); err != nil {
		t.Fatalf("não foi possível gravar %s: %v", rel, err)
	}
	return full
}

// testVaultConfig monta a config do vault de teste.
func testVaultConfig(include, exclude []string) *config.Config {
	cfg := config.DefaultConfig()
	if include != nil {
		cfg.Include = include
	}
	if exclude != nil {
		cfg.Exclude = exclude
	}
	return &cfg
}

func mdOnlyConfig() *config.Config {
	return testVaultConfig([]string{"**/*.md"}, nil)
}

func refFor(root, rel, title string) DocumentRef {
	return DocumentRef{
		ID:    filepath.Join(root, filepath.FromSlash(rel)),
		Path:  filepath.Join(root, filepath.FromSlash(rel)),
		Title: title,
	}
}

func pathsOf(refs []DocumentRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.Path)
	}
	return out
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Guardas de segurança
// ---------------------------------------------------------------------------

func TestReconcile_VarreduraVaziaAborta(t *testing.T) {
	vault := t.TempDir() // diretório existe, mas não contém nenhum arquivo indexável

	// O banco "lembra" de 316 documentos: sem o guard, tudo seria removido.
	st, rec := newTestStore(t,
		refFor(vault, "ghost-a.md", "A"),
		refFor(vault, "ghost-b.md", "B"),
	)

	rep, err := NewReconciler(st, st, vault, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: false})
	if err == nil {
		t.Fatal("varredura vazia deveria abortar com erro, mas retornou relatório")
	}
	if !strings.Contains(err.Error(), "0 arquivos indexáveis") {
		t.Errorf("mensagem de erro deveria explicar a varredura vazia, obteve: %v", err)
	}
	if rep != nil {
		t.Errorf("nenhum relatório deveria ser produzido no abort, obteve: %+v", rep)
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("guard de 0 arquivos falhou: %d DELETE(s) emitido(s) contra um vault vazio", got)
	}
}

func TestReconcile_RootDirInexistenteAborta(t *testing.T) {
	vault := filepath.Join(t.TempDir(), "vault-nao-existe")

	st, rec := newTestStore(t, refFor(vault, "ghost.md", "Ghost"))

	_, err := NewReconciler(st, st, vault, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: false})
	if err == nil {
		t.Fatal("root_dir inexistente deveria abortar com erro")
	}
	if !strings.Contains(err.Error(), "reconcile abortado") {
		t.Errorf("erro deveria sinalizar abort, obteve: %v", err)
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("root_dir inexistente não pode apagar nada, mas houve %d DELETE(s)", got)
	}
}

func TestReconcile_RootDirNaoDiretorioAborta(t *testing.T) {
	vault := t.TempDir()
	notADir := writeVaultFile(t, vault, "notas.md")

	st, rec := newTestStore(t)

	_, err := NewReconciler(st, st, notADir, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: false})
	if err == nil {
		t.Fatal("root_dir apontando para arquivo deveria abortar com erro")
	}
	if !strings.Contains(err.Error(), "não é um diretório") {
		t.Errorf("erro deveria dizer que não é diretório, obteve: %v", err)
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("root_dir inválido não pode apagar nada, mas houve %d DELETE(s)", got)
	}
}

func TestReconcile_SemRootDirAborta(t *testing.T) {
	st, rec := newTestStore(t)

	_, err := NewReconciler(st, st, "", mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: false})
	if err == nil {
		t.Fatal("ausência de root_dir deveria abortar com erro")
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("sem root_dir não pode apagar nada, mas houve %d DELETE(s)", got)
	}
}

func TestReconcile_ConfigAusenteAborta(t *testing.T) {
	vault := t.TempDir()
	writeVaultFile(t, vault, "notas/a.md")

	st, rec := newTestStore(t, refFor(vault, "notas/a.md", "A"))

	_, err := NewReconciler(st, st, vault, nil).Reconcile(context.Background(), ReconcileRequest{DryRun: false})
	if err == nil {
		t.Fatal("config ausente deveria abortar com erro")
	}
	if !strings.Contains(err.Error(), "config do vault não carregada") {
		t.Errorf("erro deveria citar a config ausente, obteve: %v", err)
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("config ausente não pode apagar nada, mas houve %d DELETE(s)", got)
	}
}

// ---------------------------------------------------------------------------
// Classificação
// ---------------------------------------------------------------------------

func TestReconcile_ClassificaPresenteAusenteEForaDoBanco(t *testing.T) {
	vault := t.TempDir()
	writeVaultFile(t, vault, "notas/presente.md")
	writeVaultFile(t, vault, "notas/fora-do-banco.md")

	st, rec := newTestStore(t,
		refFor(vault, "notas/presente.md", "Presente"),
		refFor(vault, "notas/fantasma.md", "Fantasma"),
	)

	rep, err := NewReconciler(st, st, vault, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: true})
	if err != nil {
		t.Fatalf("reconcile inesperado erro: %v", err)
	}

	if len(rep.Present) != 1 || !containsPath(pathsOf(rep.Present), filepath.Join(vault, "notas", "presente.md")) {
		t.Errorf("esperava 1 documento presente, obteve: %v", pathsOf(rep.Present))
	}
	if len(rep.MissingOnDisk) != 1 || !containsPath(pathsOf(rep.MissingOnDisk), filepath.Join(vault, "notas", "fantasma.md")) {
		t.Errorf("esperava 1 documento ausente no disco, obteve: %v", pathsOf(rep.MissingOnDisk))
	}
	if len(rep.NotIndexed) != 1 || !containsPath(pathsOf(rep.NotIndexed), "notas/fora-do-banco.md") {
		t.Errorf("esperava 1 arquivo no disco fora do banco, obteve: %v", pathsOf(rep.NotIndexed))
	}
	if len(rep.ExcludedFromIndex) != 0 {
		t.Errorf("nada deveria estar fora do índice por exclusão, obteve: %v", pathsOf(rep.ExcludedFromIndex))
	}
	if rep.ScannedFiles != 2 {
		t.Errorf("esperava 2 arquivos varridos, obteve: %d", rep.ScannedFiles)
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("dry-run não pode emitir DELETE, mas houve %d", got)
	}
}

func TestReconcile_DryRunNaoExecutaNenhumDelete(t *testing.T) {
	vault := t.TempDir()
	writeVaultFile(t, vault, "notas/presente.md")
	writeVaultFile(t, vault, "notas/b.md")

	st, rec := newTestStore(t,
		refFor(vault, "notas/presente.md", "Presente"),
		refFor(vault, "notas/obsoleto.md", "Obsoleto"),
	)

	rep, err := NewReconciler(st, st, vault, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: true})
	if err != nil {
		t.Fatalf("reconcile inesperado erro: %v", err)
	}

	if len(rep.Pruned) != 1 {
		t.Fatalf("esperava 1 candidato, obteve: %+v", rep.Pruned)
	}
	if !rep.Pruned[0].DryRun {
		t.Error("candidato em dry-run deveria estar marcado como DryRun")
	}
	if rep.Pruned[0].ID != filepath.Join(vault, "notas", "obsoleto.md") {
		t.Errorf("candidato inesperado: %s", rep.Pruned[0].ID)
	}

	// A prova central: o pipeline real do database/sql recebeu apenas SELECT.
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("dry-run emitiu %d DELETE(s) — nenhum delete pode acontecer em dry-run", got)
	}
	if rec.queryCount() == 0 {
		t.Error("esperava ao menos uma consulta de leitura ao inventário")
	}
}

func TestReconcile_AplicacaoRemoveSomenteOsAusentes(t *testing.T) {
	vault := t.TempDir()
	writeVaultFile(t, vault, "notas/presente.md")
	writeVaultFile(t, vault, "notas/b.md")

	st, rec := newTestStore(t,
		refFor(vault, "notas/presente.md", "Presente"),
		refFor(vault, "notas/obsoleto.md", "Obsoleto"),
	)

	rep, err := NewReconciler(st, st, vault, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: false})
	if err != nil {
		t.Fatalf("reconcile inesperado erro: %v", err)
	}

	if len(rep.Pruned) != 1 || rep.Pruned[0].ID != filepath.Join(vault, "notas", "obsoleto.md") {
		t.Fatalf("esperava remover apenas o obsoleto, obteve: %+v", rep.Pruned)
	}

	deleted := rec.deleteIDs()
	if len(deleted) == 0 {
		t.Fatal("esperava DELETEs para o documento obsoleto")
	}
	for _, id := range deleted {
		if id != filepath.Join(vault, "notas", "obsoleto.md") {
			t.Errorf("DELETE em documento que não deveria ser removido: %s", id)
		}
	}
}

// ---------------------------------------------------------------------------
// catalog-skills: material bruto, fora do índice por exclusão
// ---------------------------------------------------------------------------

func TestReconcile_CatalogSkillsNuncaEhCandidato(t *testing.T) {
	vault := t.TempDir()
	writeVaultFile(t, vault, "notas/conhecimento.md")
	writeVaultFile(t, vault, "catalog-skills/skills/bruto.md")

	cfg := testVaultConfig([]string{"**/*.md"}, []string{".git/**", "catalog-skills/skills/**"})

	st, rec := newTestStore(t,
		refFor(vault, "notas/conhecimento.md", "Conhecimento"),
		refFor(vault, "catalog-skills/skills/bruto.md", "Material Bruto"),
	)

	rep, err := NewReconciler(st, st, vault, cfg).Reconcile(context.Background(), ReconcileRequest{DryRun: false})
	if err != nil {
		t.Fatalf("reconcile inesperado erro: %v", err)
	}

	if len(rep.MissingOnDisk) != 0 {
		t.Errorf("catalog-skills não pode aparecer como órfão para apagar: %v", pathsOf(rep.MissingOnDisk))
	}
	if len(rep.ExcludedFromIndex) != 1 || !containsPath(pathsOf(rep.ExcludedFromIndex), filepath.Join(vault, "catalog-skills", "skills", "bruto.md")) {
		t.Fatalf("esperava catalog-skills como 'fora do índice por exclusão', obteve: %v", pathsOf(rep.ExcludedFromIndex))
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("material bruto não pode ser removido, mas houve %d DELETE(s)", got)
	}
	if len(rep.Pruned) != 0 {
		t.Errorf("nada deveria ter sido removido, obteve: %+v", rep.Pruned)
	}
}

func TestReconcile_ArquivoExcluidoPorConfigNaoEhCandidato(t *testing.T) {
	vault := t.TempDir()
	writeVaultFile(t, vault, "notas/mantida.md")
	writeVaultFile(t, vault, "lixo/apagavel.md")

	cfg := testVaultConfig([]string{"**/*.md"}, []string{".git/**", "lixo/**"})

	st, rec := newTestStore(t,
		refFor(vault, "notas/mantida.md", "Mantida"),
		refFor(vault, "lixo/apagavel.md", "Fora de escopo"),
	)

	rep, err := NewReconciler(st, st, vault, cfg).Reconcile(context.Background(), ReconcileRequest{DryRun: false})
	if err != nil {
		t.Fatalf("reconcile inesperado erro: %v", err)
	}

	if len(rep.MissingOnDisk) != 0 {
		t.Errorf("arquivo excluído por configuração não é órfão: %v", pathsOf(rep.MissingOnDisk))
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("esperava zero DELETE, houve %d", got)
	}
	if len(rep.ExcludedFromIndex) != 1 {
		t.Errorf("esperava 1 documento fora do índice por exclusão, obteve: %v", pathsOf(rep.ExcludedFromIndex))
	}
	if len(rep.Warnings) == 0 {
		t.Error("esperava um aviso explicando a exclusão por configuração")
	}
}

// ---------------------------------------------------------------------------
// Normalização de caminho
// ---------------------------------------------------------------------------

func TestReconcile_IdRelativoCaixaSeparadorNaoGeraFalsoAusente(t *testing.T) {
	vault := t.TempDir()
	writeVaultFile(t, vault, "Notas/Alpha.md")

	st, rec := newTestStore(t,
		// Banco guardou o mesmo arquivo de três formas diferentes.
		DocumentRef{ID: "notas/alpha.md", Path: `Notas\Alpha.md`, Title: "Alpha"},
	)

	rep, err := NewReconciler(st, st, vault, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: true})
	if err != nil {
		t.Fatalf("reconcile inesperado erro: %v", err)
	}

	if len(rep.MissingOnDisk) != 0 {
		t.Fatalf("diferença de escrita gerou falso positivo de ausente: %v", pathsOf(rep.MissingOnDisk))
	}
	if len(rep.Present) != 1 {
		t.Fatalf("esperava o documento como presente, obteve: %v", pathsOf(rep.Present))
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("nenhum DELETE esperado, houve %d", got)
	}
}

// ---------------------------------------------------------------------------
// Assimetria perigosa: muito disco não indexado
// ---------------------------------------------------------------------------

func TestReconcile_VaultPequenoTolerado(t *testing.T) {
	vault := t.TempDir()
	writeVaultFile(t, vault, "a.md")
	writeVaultFile(t, vault, "b.md")
	writeVaultFile(t, vault, "c.md")

	st, rec := newTestStore(t, refFor(vault, "a.md", "A"))

	rep, err := NewReconciler(st, st, vault, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: false})
	if err != nil {
		t.Fatalf("reconcile inesperado erro: %v", err)
	}
	if rep.Blocked {
		t.Error("vault pequeno não deve bloquear (tolerar ruído)")
	}
	if got := rec.deleteCount(); got != 0 {
		t.Errorf("não havia o que remover, houve %d DELETE(s)", got)
	}
}

func TestReconcile_AssimetriaAltaBloqueiaSemConfirmacao(t *testing.T) {
	vault := t.TempDir()

	var docs []DocumentRef
	// 50 documentos presentes + 51 arquivos no disco fora do índice.
	for i := 0; i < 50; i++ {
		rel := fmt.Sprintf("notas/doc-%02d.md", i)
		writeVaultFile(t, vault, rel)
		docs = append(docs, refFor(vault, rel, fmt.Sprintf("Doc %02d", i)))
	}
	for i := 0; i < 51; i++ {
		writeVaultFile(t, vault, fmt.Sprintf("soltos/solto-%02d.md", i))
	}
	// Um documento realmente obsoleto, para provar que o prune Faria algo.
	docs = append(docs, refFor(vault, "notas/obsoleto.md", "Obsoleto"))

	st, rec := newTestStore(t, docs...)

	rep, err := NewReconciler(st, st, vault, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: false})
	if err != nil {
		t.Fatalf("reconcile inesperado erro: %v", err)
	}

	if !rep.Blocked {
		t.Fatal("assimetria em vault grande deveria bloquear a remoção")
	}
	if len(rep.Warnings) == 0 || !strings.Contains(rep.Warnings[len(rep.Warnings)-1], "ASSIMETRIA ALTA") {
		t.Errorf("esperava aviso alto de assimetria, obteve: %v", rep.Warnings)
	}
	if got := rec.deleteCount(); got != 0 {
		t.Fatalf("operação bloqueada não pode apagar nada, houve %d DELETE(s)", got)
	}
}

func TestReconcile_ConfirmacaoExplicitaDesbloqueia(t *testing.T) {
	vault := t.TempDir()

	var docs []DocumentRef
	for i := 0; i < 50; i++ {
		rel := fmt.Sprintf("notas/doc-%02d.md", i)
		writeVaultFile(t, vault, rel)
		docs = append(docs, refFor(vault, rel, fmt.Sprintf("Doc %02d", i)))
	}
	for i := 0; i < 51; i++ {
		writeVaultFile(t, vault, fmt.Sprintf("soltos/solto-%02d.md", i))
	}
	obsoleto := refFor(vault, "notas/obsoleto.md", "Obsoleto")
	docs = append(docs, obsoleto)

	st, rec := newTestStore(t, docs...)

	rep, err := NewReconciler(st, st, vault, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: false, Confirm: true})
	if err != nil {
		t.Fatalf("reconcile inesperado erro: %v", err)
	}

	if rep.Blocked {
		t.Fatal("com confirm=true a operação não deveria ficar bloqueada")
	}
	if len(rep.Pruned) != 1 || rep.Pruned[0].ID != obsoleto.ID {
		t.Fatalf("esperava remover apenas o obsoleto, obteve: %+v", rep.Pruned)
	}
	for _, id := range rec.deleteIDs() {
		if id != obsoleto.ID {
			t.Errorf("DELETE em documento indevido: %s", id)
		}
	}
}

func TestReconcile_IgnoraDiretoriosDeSistema(t *testing.T) {
	vault := t.TempDir()
	writeVaultFile(t, vault, "notas/real.md")
	writeVaultFile(t, vault, ".memory/interno.md")
	writeVaultFile(t, vault, ".obsidian/cache.md")

	st, _ := newTestStore(t, refFor(vault, "notas/real.md", "Real"))

	rep, err := NewReconciler(st, st, vault, mdOnlyConfig()).Reconcile(context.Background(), ReconcileRequest{DryRun: true})
	if err != nil {
		t.Fatalf("reconcile inesperado erro: %v", err)
	}

	if rep.ScannedExcluded != 0 {
		t.Errorf("diretórios de sistema não devem ser contados como 'fora do escopo': %d", rep.ScannedExcluded)
	}
	if rep.ScannedFiles != 1 {
		t.Errorf("esperava exatamente 1 arquivo indexável, obteve: %d", rep.ScannedFiles)
	}
}
