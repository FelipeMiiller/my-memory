package drift

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/deeplink"
)

// DriftSeverity representa a intensidade do desvio detectado
type DriftSeverity string

const (
	SeverityLow      DriftSeverity = "LOW"
	SeverityMedium   DriftSeverity = "MEDIUM"
	SeverityHigh     DriftSeverity = "HIGH"
	SeverityCritical DriftSeverity = "CRITICAL"
)

// NoteDrift representa uma nota com desvio detectado em relação ao código-fonte
type NoteDrift struct {
	NoteID        string              `json:"note_id"`
	Title         string              `json:"title"`
	Path          string              `json:"path"`
	DriftScore    float64             `json:"drift_score"` // 0.0 a 100.0
	Severity      DriftSeverity       `json:"severity"`
	CommitsBehind int                 `json:"commits_behind"`
	LinesChanged  int                 `json:"lines_changed"`
	AffectedFiles []string            `json:"affected_files"`
	Reason        string              `json:"reason"`
	PageRank      float64             `json:"pagerank"`
	Links         *deeplink.DeepLinks `json:"links,omitempty"`
}

// UncoveredCode representa código-fonte modificado sem notas associadas
type UncoveredCode struct {
	FilePath        string `json:"file_path"`
	Package         string `json:"package"`
	Status          string `json:"status"`
	Additions       int    `json:"additions"`
	Deletions       int    `json:"deletions"`
	SuggestedAction string `json:"suggested_action"`
}

// DriftReport consolida a análise completa de desvio de conhecimento
type DriftReport struct {
	RangeStr      string          `json:"range"`
	TotalCommits  int             `json:"total_commits"`
	TotalChanged  int             `json:"total_changed_files"`
	DriftedNotes  []NoteDrift     `json:"drifted_notes"`
	UncoveredCode []UncoveredCode `json:"uncovered_code"`
	OverallScore  float64         `json:"overall_score"` // 0 a 100
	CriticalCount int             `json:"critical_count"`
	HighCount     int             `json:"high_count"`
	MediumCount   int             `json:"medium_count"`
	LowCount      int             `json:"low_count"`
	AnalyzedAt    time.Time       `json:"analyzed_at"`
}

// DocumentMeta armazena metadados do banco para cruzamento com o Git
type DocumentMeta struct {
	ID        string
	Path      string
	Title     string
	UpdatedAt time.Time
	Content   string
	PageRank  float64
}

// AnalyzeDrift executa a análise de correlação entre alterações do Git e a base de notas
func AnalyzeDrift(
	ctx context.Context,
	runner GitRunner,
	db *sql.DB,
	repoRoot string,
	vaultName string,
	rangeStr string,
	threshold float64,
	includeUncovered bool,
) (*DriftReport, error) {
	if runner == nil {
		runner = NewOSGitRunner()
	}

	// 1. Coleta commits e alterações de arquivos do Git
	commits, err := runner.Log(ctx, repoRoot, rangeStr)
	if err != nil {
		return nil, fmt.Errorf("erro ao extrair commits do git: %w", err)
	}

	changes, err := runner.Diff(ctx, repoRoot, rangeStr)
	if err != nil {
		return nil, fmt.Errorf("erro ao extrair diff do git: %w", err)
	}

	report := &DriftReport{
		RangeStr:     rangeStr,
		TotalCommits: len(commits),
		TotalChanged: len(changes),
		AnalyzedAt:   time.Now(),
	}

	if len(changes) == 0 {
		return report, nil
	}

	// 2. Carrega documentos e chunks do banco de dados
	docMap, err := loadDocumentsMetadata(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar documentos do banco: %w", err)
	}

	// Mapeia arquivos de código modificados que foram associados a alguma nota
	coveredFiles := make(map[string]bool)
	noteMatches := make(map[string]*NoteDrift)

	// 3. Cruzamento de cada alteração de arquivo com notas indexadas
	for _, ch := range changes {
		if !ch.IsCodeFile {
			continue
		}

		dirName := filepath.ToSlash(filepath.Dir(ch.Path))
		pkgName := filepath.Base(dirName)

		fileWasCovered := false

		for _, doc := range docMap {
			contentLower := strings.ToLower(doc.Content)
			pathLower := strings.ToLower(ch.Path)

			// ADR-039: match estrito - só path completo (sem basename) evita
			// falso positivo com palavras comuns (ex: 'main.go' em qualquer doc
			// que fale de Go). MinMatchOccurrences (2): nota só entra se mencionar
			// o path ≥ 2 vezes, eliminando listas genéricas (validation.md,
			// tasks.md) que listam todos os arquivos do projeto sem correlação
			// semântica real. Só conta directMatches para evitar double-counting
			// entre path completo e substrings de package.
			const MinMatchOccurrences = 2
			directMatches := strings.Count(contentLower, pathLower)
			hasMatch := directMatches >= MinMatchOccurrences

			if hasMatch {
				fileWasCovered = true
				coveredFiles[ch.Path] = true

				nd, exists := noteMatches[doc.ID]
				if !exists {
					links := deeplink.GenerateLinks(repoRoot, vaultName, doc.Path, 0)
					nd = &NoteDrift{
						NoteID:        doc.ID,
						Title:         doc.Title,
						Path:          doc.Path,
						PageRank:      doc.PageRank,
						Links:         &links,
						AffectedFiles: []string{},
					}
					noteMatches[doc.ID] = nd
				}

				if !containsString(nd.AffectedFiles, ch.Path) {
					nd.AffectedFiles = append(nd.AffectedFiles, ch.Path)
				}
				nd.LinesChanged += (ch.Additions + ch.Deletions)
			}
		}

		// 4. Detecção de código órfão
		if !fileWasCovered && includeUncovered && ch.IsCodeFile {
			suggested := "Documentar mudanças arquiteturais ou criar nota de componente"
			if ch.Status == "A" {
				suggested = "Criar nova especificação ou ADR para novo módulo de código"
			} else if ch.Status == "D" {
				suggested = "Verificar remoção de dependências e documentar depreciação"
			}

			report.UncoveredCode = append(report.UncoveredCode, UncoveredCode{
				FilePath:        ch.Path,
				Package:         pkgName,
				Status:          ch.Status,
				Additions:       ch.Additions,
				Deletions:       ch.Deletions,
				SuggestedAction: suggested,
			})
		}
	}

	// 5. Cálculo de Drift Score e Severidade para cada nota
	for _, nd := range noteMatches {
		doc := docMap[nd.NoteID]

		// Conta commits posteriores ao updated_at da nota que tocaram seus arquivos
		commitsBehind := 0
		for _, commit := range commits {
			if doc.UpdatedAt.IsZero() || commit.Date.After(doc.UpdatedAt) {
				commitsBehind++
			}
		}
		if commitsBehind == 0 && len(commits) > 0 {
			commitsBehind = 1 // Pelo menos 1 commit recente tocou o código afetado
		}
		nd.CommitsBehind = commitsBehind

		// Fórmula de Drift Score (ADR-039 - calibração)
		// Score = min(100, min(40, commits × 12) + min(15, PR × 30) + min(25, ln(1+lines) × 5))
		// Antes: PR × 250 saturava em 100 para qualquer nota com PR > 0.12.
		// Agora: peso do PageRank reduzido em 88%, com cap apropriado.
		// Ground truth (sessão 2026-09-17): commits=5, PR=0.5, lines=183 → score 80 (CRITICAL).
		prFactor := nd.PageRank * 30.0
		if prFactor > 15.0 {
			prFactor = 15.0
		}
		linesFactor := math.Log1p(float64(nd.LinesChanged)) * 5.0
		if linesFactor > 25.0 {
			linesFactor = 25.0
		}
		commitFactor := float64(nd.CommitsBehind) * 12.0
		if commitFactor > 40.0 {
			commitFactor = 40.0
		}

		score := commitFactor + prFactor + linesFactor
		if score > 100.0 {
			score = 100.0
		}
		nd.DriftScore = math.Round(score*10) / 10

		// Severidade (ADR-039 - limiares ajustados)
		// CRITICAL ≥ 75 (era 65), HIGH ≥ 55 (era 45), MEDIUM ≥ 30 (era 25).
		// Threshold mais alto evita que o gate --strict bloqueie PRs por falso positivo.
		switch {
		case nd.DriftScore >= 75.0:
			nd.Severity = SeverityCritical
			report.CriticalCount++
		case nd.DriftScore >= 55.0:
			nd.Severity = SeverityHigh
			report.HighCount++
		case nd.DriftScore >= 30.0:
			nd.Severity = SeverityMedium
			report.MediumCount++
		default:
			nd.Severity = SeverityLow
			report.LowCount++
		}

		nd.Reason = fmt.Sprintf("%d commit(s) alteraram %d arquivo(s) associado(s) (%d linhas modificadas)",
			nd.CommitsBehind, len(nd.AffectedFiles), nd.LinesChanged)

		if threshold <= 0 || (nd.DriftScore/100.0) >= threshold {
			report.DriftedNotes = append(report.DriftedNotes, *nd)
		}
	}

	// Ordena notas por maior Drift Score decrescente
	sort.Slice(report.DriftedNotes, func(i, j int) bool {
		return report.DriftedNotes[i].DriftScore > report.DriftedNotes[j].DriftScore
	})

	// Ordena código órfão por volume de adições
	sort.Slice(report.UncoveredCode, func(i, j int) bool {
		return (report.UncoveredCode[i].Additions + report.UncoveredCode[i].Deletions) >
			(report.UncoveredCode[j].Additions + report.UncoveredCode[j].Deletions)
	})

	// Score global do repositório
	if len(report.DriftedNotes) > 0 {
		sum := 0.0
		for _, n := range report.DriftedNotes {
			sum += n.DriftScore
		}
		report.OverallScore = math.Round((sum/float64(len(report.DriftedNotes)))*10) / 10
	}

	return report, nil
}

// loadDocumentsMetadata carrega documentos, chunks agregados e PageRank
func loadDocumentsMetadata(ctx context.Context, database *sql.DB) (map[string]DocumentMeta, error) {
	docMap := make(map[string]DocumentMeta)
	if database == nil {
		return docMap, nil
	}

	// 1. Busca documentos básicos
	rows, err := database.QueryContext(ctx, `
		SELECT id, path, title, updated_at
		FROM documents
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, path, title string
		var rawUpdated int64
		if err := rows.Scan(&id, &path, &title, &rawUpdated); err != nil {
			return nil, err
		}

		var updateTime time.Time
		if rawUpdated > 0 {
			if rawUpdated > 1000000000000 { // milissegundos
				updateTime = time.UnixMilli(rawUpdated)
			} else {
				updateTime = time.Unix(rawUpdated, 0)
			}
		}

		docMap[id] = DocumentMeta{
			ID:        id,
			Path:      path,
			Title:     title,
			UpdatedAt: updateTime,
			PageRank:  0.015,
		}
	}

	// 2. Agrega conteúdo dos chunks da tabela correta 'chunks'
	chunkRows, err := database.QueryContext(ctx, `
		SELECT document_id, content
		FROM chunks
		ORDER BY document_id, chunk_index
	`)
	if err == nil {
		defer chunkRows.Close()
		for chunkRows.Next() {
			var docID, content string
			if err := chunkRows.Scan(&docID, &content); err == nil {
				if dm, exists := docMap[docID]; exists {
					if dm.Content == "" {
						dm.Content = content
					} else {
						dm.Content += " " + content
					}
					docMap[docID] = dm
				}
			}
		}
	}

	return docMap, nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
