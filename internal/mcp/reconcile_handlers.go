package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ReconcileRequest são os argumentos de memory_reconcile.
// DryRun é ponteiro para distinguir "ausente" (default true) de "false explícito".
type ReconcileRequest struct {
	Repository string `json:"repository,omitempty"`
	RootDir    string `json:"root_dir,omitempty"`
	DryRun     *bool  `json:"dry_run,omitempty"`
	Confirm    bool   `json:"confirm,omitempty"`
}

// DryRunEnabled devolve o valor efetivo de dry_run (default: true).
func (r ReconcileRequest) DryRunEnabled() bool {
	if r.DryRun == nil {
		return true
	}
	return *r.DryRun
}

// ReconcileDocRef é um documento ou arquivo dentro do relatório de reconciliação.
type ReconcileDocRef struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
}

// ReconcilePruned registra um documento removido (ou candidato, em dry-run).
type ReconcilePruned struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	DryRun bool   `json:"dry_run,omitempty"`
}

// ReconcileReport é o resultado da comparação entre o vault e o índice.
type ReconcileReport struct {
	Repository        string            `json:"repository"`
	RootDir           string            `json:"root_dir"`
	DryRun            bool              `json:"dry_run"`
	ScannedFiles      int               `json:"scanned_files"`
	ScannedExcluded   int               `json:"scanned_excluded"`
	Present           []ReconcileDocRef `json:"present"`
	MissingOnDisk     []ReconcileDocRef `json:"missing_on_disk"`
	NotIndexed        []ReconcileDocRef `json:"not_indexed"`
	ExcludedFromIndex []ReconcileDocRef `json:"excluded_from_index"`
	Pruned            []ReconcilePruned `json:"pruned"`
	Warnings          []string          `json:"warnings"`
	Blocked           bool              `json:"blocked"`
}

// ReconcileFunc executa a reconciliação. Recebe a requisição já validada.
type ReconcileFunc func(ctx context.Context, req ReconcileRequest) (*ReconcileReport, error)

// NewMemoryReconcileHandler cria o executor da ferramenta memory_reconcile.
//
// A ferramenta apaga dados: por isso dry_run é true por padrão e toda falha de
// precondition (vault ausente, config ausente, varredura vazia) vira erro em vez
// de um relatório com remoções.
func NewMemoryReconcileHandler(reconcileFn ReconcileFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		req := ReconcileRequest{}

		if len(args) > 0 {
			var rawMap map[string]json.RawMessage
			if err := json.Unmarshal(args, &rawMap); err != nil {
				return nil, NewError(CodeInvalidParams, fmt.Sprintf("Argumentos inválidos para memory_reconcile: %v", err), nil)
			}

			if rawRepo, ok := rawMap["repository"]; ok {
				if err := json.Unmarshal(rawRepo, &req.Repository); err != nil {
					return nil, NewError(CodeInvalidParams, fmt.Sprintf("Argumento 'repository' inválido: %v", err), nil)
				}
			}
			if rawRoot, ok := rawMap["root_dir"]; ok {
				if err := json.Unmarshal(rawRoot, &req.RootDir); err != nil {
					return nil, NewError(CodeInvalidParams, fmt.Sprintf("Argumento 'root_dir' inválido: %v", err), nil)
				}
			}
			if rawConfirm, ok := rawMap["confirm"]; ok {
				if err := json.Unmarshal(rawConfirm, &req.Confirm); err != nil {
					return nil, NewError(CodeInvalidParams, fmt.Sprintf("Argumento 'confirm' inválido: %v", err), nil)
				}
			}

			// dry_run ausente = true. Só um false explícito desliga o dry-run.
			req.DryRun = boolPtr(true)
			if rawDry, ok := rawMap["dry_run"]; ok {
				if err := json.Unmarshal(rawDry, req.DryRun); err != nil {
					return nil, NewError(CodeInvalidParams, fmt.Sprintf("Argumento 'dry_run' inválido: %v", err), nil)
				}
			}
		} else {
			req.DryRun = boolPtr(true)
		}

		if reconcileFn == nil {
			return nil, NewError(CodeInternalError, "Backend de reconciliação não configurado", nil)
		}

		report, err := reconcileFn(ctx, req)
		if err != nil {
			return nil, NewError(CodeInvalidParams, err.Error(), nil)
		}
		if report == nil {
			return nil, NewError(CodeInternalError, "Reconciliação devolveu relatório vazio", nil)
		}

		return NewTextResult(FormatReconcileReport(report)), nil
	}
}

// SetReconcileHandler configura o executor da ferramenta memory_reconcile
func (s *Server) SetReconcileHandler(reconcileFn ReconcileFunc) {
	s.RegisterToolHandler(ToolMemoryReconcile.Name, NewMemoryReconcileHandler(reconcileFn))
}

func boolPtr(v bool) *bool {
	return &v
}

// maxListedDocs limita quantos itens são impressos por categoria: a lista
// completa fica no JSON, o texto serve para o agente decidir.
const maxListedDocs = 25

// FormatReconcileReport renderiza o relatório de reconciliação em texto legível.
func FormatReconcileReport(report *ReconcileReport) string {
	var sb strings.Builder

	mode := "⚙️  APLICAÇÃO (documentos ausentes foram removidos do índice)"
	if report.DryRun {
		mode = "🔎 DRY-RUN (nada foi escrito no banco)"
	}

	sb.WriteString("# Reconciliação Vault ↔ Índice\n\n")
	fmt.Fprintf(&sb, "- Repositório: `%s`\n", report.Repository)
	fmt.Fprintf(&sb, "- Vault (`root_dir`): `%s`\n", report.RootDir)
	fmt.Fprintf(&sb, "- Modo: **%s**\n", mode)
	fmt.Fprintf(&sb, "- Varredura do disco: %d arquivo(s) indexável(is) + %d fora do escopo de configuração\n\n",
		report.ScannedFiles, report.ScannedExcluded)

	sb.WriteString("| Categoria | Quantidade |\n|---|---:|\n")
	fmt.Fprintf(&sb, "| ✅ Presentes (disco + banco) | %d |\n", len(report.Present))
	fmt.Fprintf(&sb, "| 🗑️  Ausentes no disco (candidatos à remoção) | %d |\n", len(report.MissingOnDisk))
	fmt.Fprintf(&sb, "| 🆕 No disco, fora do banco | %d |\n", len(report.NotIndexed))
	fmt.Fprintf(&sb, "| 🚫 Fora do índice por exclusão de configuração | %d |\n\n", len(report.ExcludedFromIndex))

	for _, warning := range report.Warnings {
		fmt.Fprintf(&sb, "> [!warning]\n> %s\n\n", warning)
	}

	if report.Blocked {
		sb.WriteString("**Remoção suspensa.** Nenhum documento foi apagado.\n\n")
	}

	if len(report.MissingOnDisk) > 0 {
		sb.WriteString("## Candidatos à remoção (no índice, sem arquivo no disco)\n\n")
		for i, doc := range report.MissingOnDisk {
			if i >= maxListedDocs {
				fmt.Fprintf(&sb, "- … e mais %d documento(s).\n", len(report.MissingOnDisk)-maxListedDocs)
				break
			}
			fmt.Fprintf(&sb, "- `%s`\n", doc.Path)
		}
		sb.WriteString("\n")
	}

	if len(report.NotIndexed) > 0 {
		sb.WriteString("## No disco, fora do banco (não foram removidos)\n\n")
		for i, doc := range report.NotIndexed {
			if i >= maxListedDocs {
				fmt.Fprintf(&sb, "- … e mais %d arquivo(s).\n", len(report.NotIndexed)-maxListedDocs)
				break
			}
			fmt.Fprintf(&sb, "- `%s`\n", doc.Path)
		}
		sb.WriteString("\n")
	}

	if len(report.ExcludedFromIndex) > 0 {
		sb.WriteString("## Fora do índice por exclusão (estado correto — não tocar)\n\n")
		for i, doc := range report.ExcludedFromIndex {
			if i >= maxListedDocs {
				fmt.Fprintf(&sb, "- … e mais %d documento(s).\n", len(report.ExcludedFromIndex)-maxListedDocs)
				break
			}
			fmt.Fprintf(&sb, "- `%s`\n", doc.Path)
		}
		sb.WriteString("\n")
	}

	if len(report.Pruned) > 0 {
		label := "Documentos removidos do índice"
		if report.DryRun {
			label = "Documentos que SERIAM removidos (dry-run)"
		}
		fmt.Fprintf(&sb, "## %s (%d)\n\n", label, len(report.Pruned))
		for i, doc := range report.Pruned {
			if i >= maxListedDocs {
				fmt.Fprintf(&sb, "- … e mais %d documento(s).\n", len(report.Pruned)-maxListedDocs)
				break
			}
			fmt.Fprintf(&sb, "- `%s` (id: `%s`)\n", doc.Path, doc.ID)
		}
		sb.WriteString("\n")
	}

	if len(report.MissingOnDisk) == 0 {
		sb.WriteString("Nenhum documento obsoleto detectado: o índice está consistente com o vault.\n")
	}

	return sb.String()
}
