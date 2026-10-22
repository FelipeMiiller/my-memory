package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// tagQueryResult é o payload JSON de `mem tag <nome>`.
//
// A forma espelha `store.VaultTag` (tag, key, count, resolved_doc_id,
// resolved_title) e acrescenta a lista de documentos que carregam a tag —
// inclusive as descendentes, como no Obsidian (`tag:inbox` casa `#inbox` e
// `#inbox/to-read`).
type tagQueryResult struct {
	Tag       string                `json:"tag"`
	Key       string                `json:"key"`
	Count     int                   `json:"count"`
	Documents []store.VaultDocument `json:"documents"`
	// ResolvedDocID/ResolvedTitle são o documento que a tag nomeia (ADR-055).
	// A tag órfã — a que não nomeia nenhum documento — aparece no inventário
	// sem resolução, e continua sendo uma resposta válida.
	ResolvedDocID string `json:"resolved_doc_id,omitempty"`
	ResolvedTitle string `json:"resolved_title,omitempty"`
}

// runTagsCLI executa o subcomando mem tags direcionando para stdout
func runTagsCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runTagsCommand(ctx, defaultRepo, args, os.Stdout)
}

// runTagsCommand lista as tags do vault, uma por linha.
//
// A ordenação padrão mora no engine: `ListVaultTags` devolve as tags por
// `tag_key` crescente. A única reordenação do comando é `--sort=count`.
//
// Filtro que não casa nada não é erro: sai zero com lista vazia, como no
// Obsidian. "Nenhuma tag com esse filtro" é uma resposta válida sobre o
// vault, não uma falha de comando.
func runTagsCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	tagsCmd := flag.NewFlagSet("tags", flag.ContinueOnError)
	tagsCmd.SetOutput(out)

	counts := tagsCmd.Bool("counts", false, "Imprime 'tag<TAB>contagem' em vez de só a tag")
	sortBy := tagsCmd.String("sort", "", "Ordenação: 'name' (padrão) ou 'count'")
	filePath := tagsCmd.String("file", "", "Filtra pelas tags de um documento (caminho)")
	folder := tagsCmd.String("folder", "", "Filtra pelas tags dos documentos sob uma pasta")
	total := tagsCmd.Bool("total", false, "Imprime só a quantidade de tags distintas")
	jsonOut := tagsCmd.Bool("json", false, "Imprime o inventário em JSON")
	flags := registerVaultFlags(tagsCmd)

	if err := tagsCmd.Parse(args); err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(*sortBy)) {
	case "", "name", "count":
	default:
		return fmt.Errorf("mem tags --sort aceita 'name' ou 'count' (obtido: %q)", *sortBy)
	}

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	filter := store.TagFilter{Folder: *folder}
	if *filePath != "" {
		docID, err := resolveTagDocumentID(ctx, reader, *filePath)
		if err != nil {
			return err
		}
		if docID == "" {
			// Documento fora do índice: lista vazia, sem erro.
			return writeTagInventory(out, []store.VaultTag{}, *total, *jsonOut, *counts)
		}
		filter.DocumentID = docID
	}

	tags, err := reader.ListTags(ctx, filter)
	if err != nil {
		return fmt.Errorf("falha ao listar tags do vault: %w", err)
	}

	if strings.EqualFold(strings.TrimSpace(*sortBy), "count") {
		sortTagsByCount(tags)
	}

	return writeTagInventory(out, tags, *total, *jsonOut, *counts)
}

// runTagCLI executa o subcomando mem tag direcionando para stdout
func runTagCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runTagCommand(ctx, defaultRepo, args, os.Stdout)
}

// runTagCommand consulta uma tag do vault.
//
// A tag pedida casa a tag e as descendentes (`inbox` casa `inbox` e
// `inbox/to-read`) — a mesma semântica do Obsidian. `FindDocumentsByTag`
// canonicaliza a entrada, então `mem tag Inbox`, `mem tag #Inbox` e
// `mem tag inbox` devolvem o mesmo resultado.
//
// Tag que não existe não é erro: contagem 0 e saída zero.
func runTagCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	tagCmd := flag.NewFlagSet("tag", flag.ContinueOnError)
	tagCmd.SetOutput(out)

	verbose := tagCmd.Bool("verbose", false, "Imprime a contagem e a lista de documentos")
	total := tagCmd.Bool("total", false, "Imprime só a contagem de documentos")
	jsonOut := tagCmd.Bool("json", false, "Imprime o resultado em JSON")
	flags := registerVaultFlags(tagCmd)

	name, err := parseTagArgs(tagCmd, args)
	if err != nil {
		return err
	}
	if name == "" {
		fmt.Fprintln(out, "Uso: mem tag <nome> [--verbose] [--total] [--json] [--db <arq>] [--postgres <url>] [--repo <slug>]")
		return fmt.Errorf("mem tag exige o nome da tag (ex: mem tag inbox)")
	}

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	documents, err := reader.FindDocumentsByTag(ctx, name)
	if err != nil {
		return fmt.Errorf("falha ao buscar documentos pela tag '%s': %w", name, err)
	}
	if documents == nil {
		documents = []store.VaultDocument{}
	}

	result := tagQueryResult{
		Tag:       store.DisplayTag(name),
		Key:       store.CanonicalTagKey(name),
		Count:     len(documents),
		Documents: documents,
	}

	// O documento resolvido vem do inventário: é a linha que nomeia um
	// documento (ADR-055). A tag órfã aparece no inventário sem resolução.
	inventory, err := reader.ListTags(ctx, store.TagFilter{})
	if err != nil {
		return fmt.Errorf("falha ao listar tags do vault: %w", err)
	}
	for _, tag := range inventory {
		if tag.Key != result.Key {
			continue
		}
		// O casing de exibição é o da linha criada primeiro (o do vault).
		result.Tag = tag.Tag
		result.ResolvedDocID = tag.ResolvedDocID
		result.ResolvedTitle = tag.ResolvedTitle
		break
	}

	if *jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Fprintln(out, result.Count)
	if *verbose && !*total {
		for _, doc := range result.Documents {
			fmt.Fprintln(out, doc.Path)
		}
	}
	return nil
}

// parseTagArgs parseia as flags do `mem tag` e devolve o nome da tag.
//
// O pacote `flag` do Go para de interpretar flags no primeiro argumento que
// não seja flag: `mem tag inbox --verbose` deixaria "--verbose" como
// argumento posicional e o comando perderia a flag. O segundo parse pega as
// flags que vieram depois do nome da tag, para que a ordem não importe —
// `mem tag --verbose inbox` e `mem tag inbox --verbose` são o mesmo comando.
//
// Nome vazio significa "o usuário não passou a tag".
func parseTagArgs(tagCmd *flag.FlagSet, args []string) (string, error) {
	if err := tagCmd.Parse(args); err != nil {
		return "", err
	}
	if tagCmd.NArg() == 0 {
		return "", nil
	}

	name := tagCmd.Arg(0)
	rest := tagCmd.Args()[1:]
	if len(rest) > 0 {
		if err := tagCmd.Parse(rest); err != nil {
			return "", err
		}
	}
	return name, nil
}

// writeTagInventory renderiza o inventário de tags no formato pedido.
//
// O inventário vazio é `[]` em JSON, nunca `null` — a lista vazia é uma
// resposta válida sobre o vault.
func writeTagInventory(out io.Writer, tags []store.VaultTag, total, jsonOut, counts bool) error {
	if total {
		fmt.Fprintln(out, len(tags))
		return nil
	}

	if jsonOut {
		if tags == nil {
			tags = []store.VaultTag{}
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(tags)
	}

	for _, tag := range tags {
		if counts {
			fmt.Fprintf(out, "%s\t%d\n", tag.Tag, tag.Count)
			continue
		}
		fmt.Fprintln(out, tag.Tag)
	}
	return nil
}

// sortTagsByCount ordena o inventário por contagem decrescente.
//
// O desempate é a tag ascendente: sem ele, duas tags com a mesma contagem
// trocariam de lugar entre execuções e a saída do comando não seria
// reproduzível.
func sortTagsByCount(tags []store.VaultTag) {
	sort.SliceStable(tags, func(i, j int) bool {
		if tags[i].Count != tags[j].Count {
			return tags[i].Count > tags[j].Count
		}
		if tags[i].Tag != tags[j].Tag {
			return tags[i].Tag < tags[j].Tag
		}
		return tags[i].Key < tags[j].Key
	})
}

// matchesVaultPath compara o path de um documento com o caminho pedido.
//
// A comparação é feita sobre a forma normalizada em "/" porque o path
// gravado no índice usa o separador da plataforma: sem isso, um path do
// Windows colado num vault Linux (ou o inverso) nunca casaria.
func matchesVaultPath(docPath, query string) bool {
	got := vaultSlash(docPath)
	want := vaultSlash(query)
	if want == "" {
		return false
	}
	return got == want || strings.HasSuffix(got, "/"+want)
}

// resolveTagDocumentID resolve o caminho pedido para o id do documento.
//
// O filtro do engine compara `document_tags.document_id` com `documents.id`.
// O id é o path do arquivo, mas o path gravado usa o separador da plataforma
// e o usuário pode digitar o caminho relativo ou só o basename — por isso o
// caminho é resolvido pelo mesmo caminho de `mem file`/`mem read`, com a
// comparação final na forma normalizada.
//
// Documento fora do índice devolve string vazia (e, portanto, lista vazia);
// caminho ambíguo é erro, como nos outros comandos de leitura do vault.
func resolveTagDocumentID(ctx context.Context, reader vaultReader, query string) (string, error) {
	candidates, err := reader.FindByPath(ctx, query)
	if err != nil {
		return "", fmt.Errorf("falha ao buscar '%s' no índice: %w", query, err)
	}

	matches := make([]store.VaultDocument, 0, len(candidates))
	for _, doc := range candidates {
		if matchesVaultPath(doc.Path, query) {
			matches = append(matches, doc)
		}
	}

	if len(matches) == 0 {
		// A busca do engine casa path exato ou basename com "/" — um path
		// gravado com "\" não casa. A varredura da lista de documentos
		// resolve pela forma normalizada.
		docs, err := reader.ListDocuments(ctx)
		if err != nil {
			return "", fmt.Errorf("falha ao listar documentos do vault: %w", err)
		}
		for _, doc := range docs {
			if matchesVaultPath(doc.Path, query) {
				matches = append(matches, doc)
			}
		}
	}

	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0].ID, nil
	default:
		return "", ambiguousVaultPathError(query, matches)
	}
}
