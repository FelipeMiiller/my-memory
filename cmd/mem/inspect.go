package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// runInspectCLI executa o subcomando mem inspect direcionando a saída para stdout
func runInspectCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runInspectCommand(ctx, defaultRepo, args, os.Stdout)
}

// runInspectCommand executa a inspeção do nó com injeção de io.Writer para testes
func runInspectCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	inspectCmd := flag.NewFlagSet("inspect", flag.ContinueOnError)
	inspectCmd.SetOutput(out)

	jsonOutput := inspectCmd.Bool("json", false, "Exibe o tríptico em formato JSON estruturado")
	fullContent := inspectCmd.Bool("full", false, "Exibe o conteúdo integral da nota sem truncamento")
	maxLen := inspectCmd.Int("max-len", 500, "Tamanho máximo do preview de conteúdo (padrão: 500 caracteres)")
	dbPath := inspectCmd.String("db", "", "Caminho do arquivo SQLite")
	pgURL := inspectCmd.String("postgres", "", "URL de conexão PostgreSQL")
	targetRepo := inspectCmd.String("repo", "", "Slug ou identificador do repositório")

	if err := inspectCmd.Parse(rearrangeInspectArgs(args)); err != nil {
		return err
	}

	remaining := inspectCmd.Args()
	if len(remaining) == 0 {
		return fmt.Errorf("identificador do nó alvo é obrigatório (ex: mem inspect <node_id>)")
	}
	targetNode := remaining[0]

	resolvedRepo := *targetRepo
	if resolvedRepo == "" {
		resolvedRepo = defaultRepo
	}

	previewLength := *maxLen
	if *fullContent {
		previewLength = 0
	} else if previewLength <= 0 {
		previewLength = 500
	}

	var view *graph.TriptychView
	var inspectErr error

	if *pgURL != "" {
		pgStore, err := store.NewPostgresStore(*pgURL)
		if err != nil {
			return fmt.Errorf("falha ao conectar no PostgreSQL: %w", err)
		}
		defer pgStore.Close()

		view, inspectErr = pgStore.InspectNode(ctx, resolvedRepo, targetNode, previewLength)
	} else {
		resolvedDB := *dbPath
		if resolvedDB == "" {
			resolvedDB = "memory.db"
		}

		database, err := db.InitDB(resolvedDB)
		if err != nil {
			return fmt.Errorf("falha ao abrir banco SQLite '%s': %w", resolvedDB, err)
		}
		defer database.Close()

		view, inspectErr = db.InspectNode(ctx, database, targetNode, previewLength)
	}

	if inspectErr != nil {
		return fmt.Errorf("falha na inspeção do nó: %w", inspectErr)
	}

	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(view)
	}

	// Renderização Visual do Tríptico para Terminal
	fmt.Fprintf(out, "\n=== 🔬 Visualização Cirúrgica de Nó (Triptych Inspector) ===\n\n")

	// 1. Coluna Central: Metadados, Métricas e Risco do Nó
	riskEmoji := "🟢"
	switch strings.ToUpper(view.Target.RiskLevel) {
	case "CRÍTICO", "CRITICAL":
		riskEmoji = "🔴"
	case "ALTO", "HIGH":
		riskEmoji = "🟠"
	case "MODERADO", "MEDIUM":
		riskEmoji = "🟡"
	}

	fmt.Fprintf(out, "[ 🎯 NÓ CENTRAL: %s ]\n", view.Target.Title)
	fmt.Fprintf(out, "  ID Canônico:     %s\n", view.Target.ID)
	if view.Target.Path != "" {
		fmt.Fprintf(out, "  Caminho:         %s\n", view.Target.Path)
	}
	fmt.Fprintf(out, "  Tipo:            %s\n", view.Target.Type)
	fmt.Fprintf(out, "  PageRank:        %.4f\n", view.Target.PageRank)
	if view.Target.CommunityID > 0 {
		label := view.Target.CommunityLabel
		if label == "" {
			label = "Geral"
		}
		fmt.Fprintf(out, "  Comunidade:      Cluster #%d (%s)\n", view.Target.CommunityID, label)
	}
	fmt.Fprintf(out, "  Risco de Quebra: %s %.1f / 100 [%s]\n", riskEmoji, view.Target.RiskScore, view.Target.RiskLevel)

	if len(view.Target.Tags) > 0 {
		fmt.Fprintf(out, "  Tags:            %s\n", strings.Join(view.Target.Tags, ", "))
	}

	if view.Target.ContentPreview != "" {
		fmt.Fprintf(out, "\n  --- Conteúdo / Resumo (%d caracteres) ---\n", view.Target.ContentLength)
		lines := strings.Split(view.Target.ContentPreview, "\n")
		for _, l := range lines {
			fmt.Fprintf(out, "  │ %s\n", l)
		}
		fmt.Fprintf(out, "  -----------------------------------------\n")
	}

	// 2. Coluna da Esquerda: Inbound Links / Chamadores
	fmt.Fprintf(out, "\n[ ⬅️ INBOUND / CHAMADORES & DEPENDENTES (%d) ]\n", view.TotalInbound)
	if view.CriticalDependents > 0 {
		fmt.Fprintf(out, "  ⚠️ Dependentes Críticos: %d nó(s)\n", view.CriticalDependents)
	}
	if len(view.Inbound) == 0 {
		fmt.Fprintf(out, "  (nenhum chamador identificado)\n")
	} else {
		for _, in := range view.Inbound {
			badge := "🟢 [BAIXO]"
			switch in.Severity {
			case graph.SeverityCritical:
				badge = "🔴 [CRÍTICO]"
			case graph.SeverityHigh:
				badge = "🟠 [ALTO]"
			case graph.SeverityMedium:
				badge = "🟡 [MÉDIO]"
			}
			prStr := ""
			if in.PageRank > 0 {
				prStr = fmt.Sprintf(" | PR: %.4f", in.PageRank)
			}
			fmt.Fprintf(out, "  %s %s (%s)%s -> ID: %s\n", badge, in.Title, in.Relation, prStr, in.SourceID)
		}
	}

	// 3. Coluna da Direita: Outbound Links / Referências
	fmt.Fprintf(out, "\n[ ➡️ OUTBOUND / REFERÊNCIAS DE SAÍDA (%d) ]\n", view.TotalOutbound)
	if len(view.Outbound) == 0 {
		fmt.Fprintf(out, "  (nenhuma referência de saída)\n")
	} else {
		for _, outLink := range view.Outbound {
			status := "✓"
			if !outLink.Exists {
				status = "⚠️ DEAD LINK"
			}
			prStr := ""
			if outLink.PageRank > 0 {
				prStr = fmt.Sprintf(" | PR: %.4f", outLink.PageRank)
			}
			fmt.Fprintf(out, "  [%s] %s (%s) [%s]%s -> ID: %s\n",
				status, outLink.Title, outLink.Relation, outLink.Type, prStr, outLink.TargetID)
		}
	}

	fmt.Fprintln(out)
	return nil
}

func rearrangeInspectArgs(args []string) []string {
	var flags []string
	var nonFlags []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") && (arg == "--max-len" || arg == "-max-len" ||
				arg == "--db" || arg == "-db" || arg == "--postgres" || arg == "-postgres" ||
				arg == "--repo" || arg == "-repo") {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					flags = append(flags, args[i+1])
					i++
				}
			}
		} else {
			nonFlags = append(nonFlags, arg)
		}
	}
	return append(flags, nonFlags...)
}

