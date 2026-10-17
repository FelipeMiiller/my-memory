package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// ReconcileLargeVaultThreshold define a partir de quantos documentos
// presentes a assimetria "mais não indexados que presentes" passa a exigir
// confirmação explícita. Abaixo disso o sinal é ruído de vault pequeno.
const ReconcileLargeVaultThreshold = 50

// DocumentLister expõe o inventário de documentos indexados. Uso somente-leitura.
type DocumentLister interface {
	ListDocumentPaths(ctx context.Context, repo string) ([]DocumentRef, error)
}

// StalePruner remove documentos indexados cujo arquivo sumiu do vault.
type StalePruner interface {
	PruneStaleDocuments(ctx context.Context, repo, rootDir string, live map[string]bool, opts PruneOptions) ([]PrunedDoc, error)
}

// ReconcileRequest parametriza uma reconciliação.
type ReconcileRequest struct {
	Repository string `json:"repository"`
	RootDir    string `json:"root_dir,omitempty"`
	DryRun     bool   `json:"dry_run"`
	Confirm    bool   `json:"confirm"`
}

// ReconcileReport classifica cada arquivo e documento do vault.
//
// As quatro categorias são disjuntas e cobrem o universo inteiro:
//
//	Present            — no disco (indexável) e no banco
//	MissingOnDisk      — no banco, sem arquivo no disco  → candidatos à remoção
//	NotIndexed         — no disco (indexável), sem linha no banco
//	ExcludedFromIndex  — linha no banco cujo arquivo existe mas está fora do
//	                     escopo de indexação; estar fora do índice está correto
//	                     (ex.: catalog-skills/**, material bruto)
type ReconcileReport struct {
	Repository        string        `json:"repository"`
	RootDir           string        `json:"root_dir"`
	DryRun            bool          `json:"dry_run"`
	ScannedFiles      int           `json:"scanned_files"`
	ScannedExcluded   int           `json:"scanned_excluded"`
	Present           []DocumentRef `json:"present"`
	MissingOnDisk     []DocumentRef `json:"missing_on_disk"`
	NotIndexed        []DocumentRef `json:"not_indexed"`
	ExcludedFromIndex []DocumentRef `json:"excluded_from_index"`
	Pruned            []PrunedDoc   `json:"pruned"`
	Warnings          []string      `json:"warnings"`
	Blocked           bool          `json:"blocked"`
}

// NeedsConfirmation sinaliza a assimetria perigosa: mais arquivos no disco
// fora do índice do que documentos presentes, em um vault grande demais para
// que isso seja normal. Nesse caso a remoção é suspensa sem confirmação.
func (r *ReconcileReport) NeedsConfirmation() bool {
	if r == nil {
		return false
	}
	return len(r.NotIndexed) > len(r.Present) && len(r.Present) >= ReconcileLargeVaultThreshold
}

// Reconciler confronta o inventário do banco com o estado real do disco.
type Reconciler struct {
	lister  DocumentLister
	pruner  StalePruner
	rootDir string
	cfg     *config.Config
}

// NewReconciler constrói o reconciliador. `lister` e `pruner` são satisfied por
// *PostgresStore; `cfg` é obrigatório (sem regras de escopo não há varredura).
func NewReconciler(lister DocumentLister, pruner StalePruner, rootDir string, cfg *config.Config) *Reconciler {
	return &Reconciler{lister: lister, pruner: pruner, rootDir: rootDir, cfg: cfg}
}

// Reconcile varre o disco, consulta o inventário do banco e classifica a
// divergência. Só executa remoções quando dry_run=false E as guardas passam.
//
// Guardas (todas abortam antes de qualquer escrita):
//   - root_dir inexistente, ilegível ou que não é diretório
//   - config do vault ausente
//   - varredura do disco com 0 arquivos indexáveis (vault não montado,
//     Google Drive desconectado) — o teste mais importante desta ferramenta
func (r *Reconciler) Reconcile(ctx context.Context, req ReconcileRequest) (*ReconcileReport, error) {
	rootDir := strings.TrimSpace(req.RootDir)
	if rootDir == "" {
		rootDir = strings.TrimSpace(r.rootDir)
	}
	if rootDir == "" {
		return nil, errors.New("reconcile abortado: root_dir é obrigatório (sem âncora de vault não é possível distinguir ausente de fora de escopo)")
	}

	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("reconcile abortado: root_dir inexistente ou inacessível (%s): %w", rootDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("reconcile abortado: root_dir não é um diretório (%s)", rootDir)
	}

	if r.cfg == nil {
		return nil, errors.New("reconcile abortado: config do vault não carregada (sem regras de include/exclude a varredura seria cega)")
	}

	if r.lister == nil {
		return nil, errors.New("reconcile abortado: inventário de documentos não configurado")
	}

	scan, err := scanVault(rootDir, r.cfg)
	if err != nil {
		return nil, fmt.Errorf("reconcile abortado: falha ao varrer o disco (%s): %w", rootDir, err)
	}
	if len(scan.live) == 0 {
		return nil, fmt.Errorf("reconcile abortado: varredura do disco retornou 0 arquivos indexáveis em %s — vault provavelmente não montado; nada foi removido", rootDir)
	}

	docs, err := r.lister.ListDocumentPaths(ctx, req.Repository)
	if err != nil {
		return nil, err
	}

	report := &ReconcileReport{
		Repository:        req.Repository,
		RootDir:           rootDir,
		DryRun:            req.DryRun,
		ScannedFiles:      len(scan.live),
		ScannedExcluded:   len(scan.excluded),
		Present:           []DocumentRef{},
		MissingOnDisk:     []DocumentRef{},
		NotIndexed:        []DocumentRef{},
		ExcludedFromIndex: []DocumentRef{},
		Pruned:            []PrunedDoc{},
		Warnings:          []string{},
	}

	rootKey := normalizeRoot(rootDir)
	prefixes := normalizePrefixes(DefaultProtectedPathPrefixes)
	indexedKeys := make(map[string]bool, len(docs)*2)

	for _, doc := range docs {
		for _, base := range []string{doc.ID, doc.Path} {
			for _, key := range pathKeyVariants(rootDir, base) {
				indexedKeys[key] = true
			}
		}

		switch {
		case isProtectedPath(rootDir, doc, prefixes):
			// Material bruto: estar ausente do índice é o estado correto.
			report.ExcludedFromIndex = append(report.ExcludedFromIndex, doc)
		case matchesAny(scan.live, rootDir, doc.ID, doc.Path):
			report.Present = append(report.Present, doc)
		case matchesAny(scan.excluded, rootDir, doc.ID, doc.Path):
			// O arquivo existe, mas a configuração o exclui do escopo de indexação.
			report.ExcludedFromIndex = append(report.ExcludedFromIndex, doc)
		case rootKey != "" && !pathWithinRoot(rootKey, doc.Path) && !pathWithinRoot(rootKey, doc.ID):
			// Fora do recorte do vault: não é responsabilidade desta reconciliação.
		default:
			report.MissingOnDisk = append(report.MissingOnDisk, doc)
		}
	}

	for key, rel := range scan.liveRel {
		if indexedKeys[key] {
			continue
		}
		report.NotIndexed = append(report.NotIndexed, DocumentRef{ID: rel, Path: rel})
	}

	sortDocs(report.Present)
	sortDocs(report.MissingOnDisk)
	sortDocs(report.NotIndexed)
	sortDocs(report.ExcludedFromIndex)

	if len(report.ExcludedFromIndex) > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf(
			"%d documento(s) permanecem no índice apesar da exclusão de configuração — classificados como 'fora do índice por exclusão', nunca como órfãos para apagar.",
			len(report.ExcludedFromIndex)))
	}

	if report.NeedsConfirmation() {
		acao := "A remoção foi SUSPENSA; reexecute com confirm=true para forçar."
		if req.Confirm {
			acao = "confirm=true foi informado: a aplicação prossegue."
		}
		report.Warnings = append(report.Warnings, fmt.Sprintf(
			"⚠️ ASSIMETRIA ALTA: %d arquivo(s) no disco fora do índice contra %d documento(s) presentes em vault com >= %d entradas. %s",
			len(report.NotIndexed), len(report.Present), ReconcileLargeVaultThreshold, acao))
		if !req.Confirm {
			report.Blocked = true
		}
	}

	if report.Blocked {
		return report, nil
	}

	if r.pruner == nil {
		return nil, errors.New("reconcile abortado: removedor de documentos obsoletos não configurado")
	}

	// KeepPaths garante que a lista de removíveis do store seja idêntica às
	// categorias do relatório: o que existe no disco mas está fora do escopo de
	// configuração nunca é removível.
	pruned, err := r.pruner.PruneStaleDocuments(ctx, req.Repository, rootDir, scan.live, PruneOptions{
		DryRun:            req.DryRun,
		KeepPaths:         scan.excluded,
		ProtectedPrefixes: DefaultProtectedPathPrefixes,
	})
	if err != nil {
		return nil, err
	}
	report.Pruned = pruned

	return report, nil
}

// matchesAny informa se alguma variante do documento está no conjunto informado.
func matchesAny(set map[string]bool, rootDir string, candidates ...string) bool {
	if len(set) == 0 {
		return false
	}
	for _, candidate := range candidates {
		for _, key := range pathKeyVariants(rootDir, candidate) {
			if set[key] {
				return true
			}
		}
	}
	return false
}

func sortDocs(docs []DocumentRef) {
	sort.Slice(docs, func(i, j int) bool {
		if docs[i].Path != docs[j].Path {
			return docs[i].Path < docs[j].Path
		}
		return docs[i].ID < docs[j].ID
	})
}

// vaultScan é o resultado bruto da varredura de disco.
type vaultScan struct {
	live     map[string]bool // chave canônica → presente e indexável
	liveRel  map[string]string
	excluded map[string]bool // chave canônica → existe mas fora do escopo
}

// scanVault percorre o vault uma única vez separando o que a configuração
// considera indexável do que ela exclui.
func scanVault(rootDir string, cfg *config.Config) (vaultScan, error) {
	scan := vaultScan{
		live:     make(map[string]bool),
		liveRel:  make(map[string]string),
		excluded: make(map[string]bool),
	}

	// Diretórios de sistema: pulados por desempenho, nunca são indexáveis
	// (ShouldIndex rejeita qualquer segmento com estes nomes).
	skipDirs := make(map[string]bool, len(config.DefaultExcludedDirs))
	for _, dir := range config.DefaultExcludedDirs {
		skipDirs[dir] = true
	}

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path != rootDir && skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		key := NormalizePathKey(path)
		if key == "" {
			return nil
		}

		if cfg.ShouldIndex(rel) {
			scan.live[key] = true
			scan.liveRel[key] = rel
		} else {
			scan.excluded[key] = true
		}
		return nil
	})

	return scan, err
}
