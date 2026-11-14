package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// seedDoctorVault monta um vault SQLite com uma nota e um link para um
// material que EXISTE no disco do vault mas não está indexado — o caso real do
// cofre central (material bruto fora do escopo de indexação).
//
// Devolve o caminho do banco e a raiz do vault em disco.
func seedDoctorVault(t *testing.T) (dbPath string, vaultRoot string) {
	t.Helper()

	rootDir := t.TempDir()
	dbPath = filepath.Join(rootDir, "vault.db")
	vaultRoot = filepath.Join(rootDir, "vault")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()
	ctx := context.Background()

	if err := db.InsertDocumentWithMeta(ctx, database, "nota-a", "notas/a.md", "nota-a", 1_700_000_000, "hash-a", "", "Resumo", "resource", 1_700_000_000, 100); err != nil {
		t.Fatalf("insert do documento: %v", err)
	}
	// O link para o material bruto: o alvo existe no disco, mas não no índice.
	if err := db.InsertEdge(ctx, database, "nota-a", "material", "links_to"); err != nil {
		t.Fatalf("insert da aresta: %v", err)
	}

	materialDir := filepath.Join(vaultRoot, "catalog-skills", "skills", "x")
	if err := os.MkdirAll(materialDir, 0755); err != nil {
		t.Fatalf("mkdir do vault: %v", err)
	}
	if err := os.WriteFile(filepath.Join(materialDir, "material.md"), []byte("material bruto\n"), 0644); err != nil {
		t.Fatalf("escrita do material: %v", err)
	}

	return dbPath, vaultRoot
}

// countFromReport le a contagem de uma seção do relatório da CLI.
func countFromReport(t *testing.T, stdout, pattern string) int {
	t.Helper()

	m := regexp.MustCompile(pattern).FindStringSubmatch(stdout)
	if m == nil {
		t.Fatalf("a seção %q não apareceu no relatório:\n%s", pattern, stdout)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("contagem inválida %q: %v", m[1], err)
	}
	return n
}

// TestDoctorVaultFlagMudaARaiz prova a AC9: a flag --vault aponta a
// checagem em disco para outra raiz, e o link que existe lá deixa de ser
// contado como rompido.
func TestDoctorVaultFlagMudaARaiz(t *testing.T) {
	dbPath, vaultRoot := seedDoctorVault(t)
	runDir := t.TempDir()

	stdout, stderr, exit := runMemDoctor(t, runDir, "doctor", "--storage=sqlite", "--db", dbPath, "--repo", "acme", "--vault", vaultRoot)
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exit, stderr, stdout)
	}

	if got := countFromReport(t, stdout, `Links Fora do Índice por Decisão] \((\d+)\)`); got != 1 {
		t.Errorf("links fora do índice = %d; want 1 (o alvo existe no disco apontado pela flag):\n%s", got, stdout)
	}
	if got := countFromReport(t, stdout, `Links Quebrados / Dead Links] \((\d+)\)`); got != 0 {
		t.Errorf("links quebrados = %d; want 0 (o alvo existe no disco):\n%s", got, stdout)
	}
	if !strings.Contains(stdout, "Health Score: 100/100") {
		t.Errorf("a categoria nova não pode penalizar o Health Score:\n%s", stdout)
	}
	if !strings.Contains(stdout, "catalog-skills/skills/x/material.md") {
		t.Errorf("o relatório precisa nomear o caminho em disco onde o alvo foi encontrado:\n%s", stdout)
	}
}

// TestDoctorVaultSemFlagMantemLinkRompido prova o outro lado da flag: sem
// ela, a raiz é a que o config global resolver e o material não está lá — o
// link continua rompido.
func TestDoctorVaultSemFlagMantemLinkRompido(t *testing.T) {
	dbPath, _ := seedDoctorVault(t)
	runDir := t.TempDir()

	stdout, stderr, exit := runMemDoctor(t, runDir, "doctor", "--storage=sqlite", "--db", dbPath, "--repo", "acme")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exit, stderr, stdout)
	}

	if got := countFromReport(t, stdout, `Links Quebrados / Dead Links] \((\d+)\)`); got != 1 {
		t.Errorf("links quebrados = %d; want 1 (sem a flag, o material não está na raiz checada):\n%s", got, stdout)
	}
	if got := countFromReport(t, stdout, `Links Fora do Índice por Decisão] \((\d+)\)`); got != 0 {
		t.Errorf("links fora do índice = %d; want 0:\n%s", got, stdout)
	}
}

// TestDoctorVaultRaizInacessivelFicaProvisorio prova a AC5 na superfície: com
// a raiz apontada para um caminho que não existe, o link fica não
// verificável e o score é declarado provisório.
func TestDoctorVaultRaizInacessivelFicaProvisorio(t *testing.T) {
	dbPath, _ := seedDoctorVault(t)
	runDir := t.TempDir()
	raizInexistente := filepath.Join(runDir, "vault-que-nao-existe")

	stdout, stderr, exit := runMemDoctor(t, runDir, "doctor", "--storage=sqlite", "--db", dbPath, "--repo", "acme", "--vault", raizInexistente)
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exit, stderr, stdout)
	}

	if got := countFromReport(t, stdout, `Links Não Verificáveis] \((\d+)\)`); got != 1 {
		t.Errorf("links não verificáveis = %d; want 1:\n%s", got, stdout)
	}
	if got := countFromReport(t, stdout, `Links Quebrados / Dead Links] \((\d+)\)`); got != 0 {
		t.Errorf("links quebrados = %d; want 0 (sem prova, nenhum link é declarado rompido):\n%s", got, stdout)
	}
	if !strings.Contains(stdout, "Score provisório") {
		t.Errorf("o relatório precisa avisar que o score é provisório:\n%s", stdout)
	}
}

// TestDoctorVaultMCPECLIPontamOMesmoNumero prova a AC8: para o mesmo vault, o
// payload do MCP e o relatório da CLI contam a mesma história.
func TestDoctorVaultMCPECLIPontamOMesmoNumero(t *testing.T) {
	dbPath, vaultRoot := seedDoctorVault(t)
	runDir := t.TempDir()

	stdout, stderr, exit := runMemDoctor(t, runDir, "doctor", "--storage=sqlite", "--db", dbPath, "--repo", "acme", "--vault", vaultRoot)
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exit, stderr, stdout)
	}

	deadNaCLI := countFromReport(t, stdout, `Links Quebrados / Dead Links] \((\d+)\)`)
	foraNaCLI := countFromReport(t, stdout, `Links Fora do Índice por Decisão] \((\d+)\)`)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	report, err := db.DiagnoseHealth(context.Background(), database, vaultRoot)
	if err != nil {
		t.Fatalf("DiagnoseHealth: %v", err)
	}
	mcpReport := convertStoreDoctorReportToMCP(report)

	if len(mcpReport.DeadLinks) != deadNaCLI {
		t.Errorf("dead links no MCP = %d; want %d (o mesmo número da CLI)", len(mcpReport.DeadLinks), deadNaCLI)
	}
	if len(mcpReport.OutOfIndexLinks) != foraNaCLI {
		t.Errorf("links fora do índice no MCP = %d; want %d (o mesmo número da CLI)", len(mcpReport.OutOfIndexLinks), foraNaCLI)
	}
	if len(mcpReport.OutOfIndexLinks) == 1 && mcpReport.OutOfIndexLinks[0].ResolvedPath != "catalog-skills/skills/x/material.md" {
		t.Errorf("caminho resolvido no MCP = %q; want %q", mcpReport.OutOfIndexLinks[0].ResolvedPath, "catalog-skills/skills/x/material.md")
	}
}

// TestDoctorVaultPayloadMCPCarregaAsTresCategorias prova que o payload do MCP
// leva as três categorias — sem elas, o agente veria o vault como se todo
// link vivo fosse link rompido.
func TestDoctorVaultPayloadMCPCarregaAsTresCategorias(t *testing.T) {
	rep := &store.DoctorReport{
		TotalDocuments: 3,
		HealthScore:    95,
		DeadLinks: []store.DeadLink{
			{SourceID: "a.md", TargetID: "x", Relation: "links_to"},
		},
		OutOfIndexLinks: []store.OutOfIndexLink{
			{SourceID: "a.md", TargetID: "material", Relation: "links_to", ResolvedPath: "catalog-skills/skills/x/material.md", Match: "basename"},
		},
		UnverifiableLinks: []store.DeadLink{
			{SourceID: "a.md", TargetID: "y", Relation: "links_to"},
		},
		HealthScoreProvisional: true,
	}

	mcpReport := convertStoreDoctorReportToMCP(rep)

	if len(mcpReport.DeadLinks) != 1 {
		t.Errorf("dead links = %d; want 1", len(mcpReport.DeadLinks))
	}
	if len(mcpReport.OutOfIndexLinks) != 1 {
		t.Errorf("links fora do índice = %d; want 1", len(mcpReport.OutOfIndexLinks))
	}
	if mcpReport.OutOfIndexLinks[0].ResolvedPath != "catalog-skills/skills/x/material.md" {
		t.Errorf("caminho resolvido = %q; want %q", mcpReport.OutOfIndexLinks[0].ResolvedPath, "catalog-skills/skills/x/material.md")
	}
	if mcpReport.OutOfIndexLinks[0].Match != "basename" {
		t.Errorf("modo de casamento = %q; want %q", mcpReport.OutOfIndexLinks[0].Match, "basename")
	}
	if len(mcpReport.UnverifiableLinks) != 1 {
		t.Errorf("links não verificáveis = %d; want 1", len(mcpReport.UnverifiableLinks))
	}
	if !mcpReport.HealthScoreProvisional {
		t.Error("o payload deveria marcar o score como provisório")
	}
}

// TestDoctorVaultPayloadMCPSemNull prova o contrato do wire: as listas
// viajam sempre como array, nunca como null (o StructuredContent proíbe null).
func TestDoctorVaultPayloadMCPSemNull(t *testing.T) {
	cheio := convertStoreDoctorReportToMCP(&store.DoctorReport{
		TotalDocuments:    1,
		DeadLinks:         []store.DeadLink{{SourceID: "a", TargetID: "b", Relation: "links_to"}},
		OutOfIndexLinks:   []store.OutOfIndexLink{{SourceID: "a", TargetID: "c", Relation: "links_to"}},
		UnverifiableLinks: []store.DeadLink{{SourceID: "a", TargetID: "d", Relation: "links_to"}},
	})
	vazio := convertStoreDoctorReportToMCP(&store.DoctorReport{})

	payloadCheio, err := json.Marshal(cheio)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	payloadVazio, err := json.Marshal(vazio)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	for _, chave := range []string{"dead_links", "out_of_index_links", "unverifiable_links"} {
		if !strings.Contains(string(payloadCheio), `"`+chave+`":[`) {
			t.Errorf("a chave %q deveria viajar como array no payload:\n%s", chave, payloadCheio)
		}
		if strings.Contains(string(payloadVazio), `"`+chave+`":null`) {
			t.Errorf("a chave %q não pode viajar como null:\n%s", chave, payloadVazio)
		}
	}
}
