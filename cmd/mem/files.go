package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// vaultSlash normaliza o separador de caminho para "/".
//
// documents.path é gravado pelo walker com o separador da plataforma, e o mesmo
// vault pode ter sido indexado em sistemas diferentes. Filtro e saída precisam
// ser iguais em qualquer SO, então a comparação sempre acontece sobre a forma
// com "/".
func vaultSlash(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

// vaultExt devolve a extensão do path sem o ponto e em minúsculas.
func vaultExt(path string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
}

// matchesExt compara a extensão do documento com o filtro pedido.
//
// ext vazio é "sem filtro". "md" e ".md" são o mesmo pedido: aceitar as duas
// formas evita que o usuário descubra a diferença por tentativa e erro.
func matchesExt(path, ext string) bool {
	if ext == "" {
		return true
	}
	want := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
	return vaultExt(path) == want
}

// matchesFolder casa o prefixo vault-relative em fronteira de segmento.
//
// O path indexado pode ser absoluto (o walker grava o caminho que percorre) ou
// relativo ao vault, dependendo de como o vault foi indexado — por isso o
// prefixo é procurado em qualquer posição de segmento, e não só no começo.
// "docs" pega "docs/a.md" e "C:/vault/docs/b.md", mas não "docs-old/c.md".
func matchesFolder(path, folder string) bool {
	if folder == "" {
		return true
	}
	want := strings.Trim(vaultSlash(folder), "/")
	if want == "" || want == "." {
		return true
	}
	got := vaultSlash(path)
	return strings.HasPrefix(got, want+"/") || strings.Contains(got, "/"+want+"/")
}

// vaultFolderOf devolve a pasta que contém o documento, em "/" e sem barra
// final. Documento na raiz do vault não pertence a pasta nenhuma e devolve "".
func vaultFolderOf(path string) string {
	cleaned := strings.Trim(vaultSlash(path), "/")
	idx := strings.LastIndex(cleaned, "/")
	if idx < 0 {
		return ""
	}
	return cleaned[:idx]
}

// filterVaultDocs aplica os filtros de --ext e --folder sobre os documentos do
// índice, preservando a ordem alfabética de path vinda do engine.
func filterVaultDocs(docs []store.VaultDocument, ext, folder string) []store.VaultDocument {
	if ext == "" && folder == "" {
		return docs
	}
	filtered := make([]store.VaultDocument, 0, len(docs))
	for _, doc := range docs {
		if matchesExt(doc.Path, ext) && matchesFolder(doc.Path, folder) {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

// runFilesCLI executa o subcomando mem files direcionando para stdout
func runFilesCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runFilesCommand(ctx, defaultRepo, args, os.Stdout)
}

// runFilesCommand lista os caminhos indexados, um por linha, em ordem alfabética.
//
// Extensão e pasta desconhecidas não são erro: saem zero com lista vazia,
// como no Obsidian. "Nenhum arquivo com essa extensão" é uma resposta válida
// sobre o vault, não uma falha de comando.
func runFilesCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	filesCmd := flag.NewFlagSet("files", flag.ContinueOnError)
	filesCmd.SetOutput(out)

	ext := filesCmd.String("ext", "", "Filtra por extensão do arquivo (ex: md)")
	folder := filesCmd.String("folder", "", "Filtra por pasta relativa ao vault (ex: docs)")
	flags := registerVaultFlags(filesCmd)

	if err := filesCmd.Parse(args); err != nil {
		return err
	}

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	docs, err := reader.ListDocuments(ctx)
	if err != nil {
		return fmt.Errorf("falha ao listar documentos do vault: %w", err)
	}

	for _, doc := range filterVaultDocs(docs, *ext, *folder) {
		fmt.Fprintln(out, doc.Path)
	}
	return nil
}

// runFoldersCLI executa o subcomando mem folders direcionando para stdout
func runFoldersCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runFoldersCommand(ctx, defaultRepo, args, os.Stdout)
}

// runFoldersCommand lista as pastas que contêm ao menos um documento indexado.
//
// As pastas saem do próprio índice, nunca do disco: uma pasta apagada depois da
// indexação continua listada enquanto o documento estiver lá, e uma pasta com
// documento só no Postgres aparece mesmo sem cópia local.
func runFoldersCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	foldersCmd := flag.NewFlagSet("folders", flag.ContinueOnError)
	foldersCmd.SetOutput(out)

	flags := registerVaultFlags(foldersCmd)

	if err := foldersCmd.Parse(args); err != nil {
		return err
	}

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	docs, err := reader.ListDocuments(ctx)
	if err != nil {
		return fmt.Errorf("falha ao listar documentos do vault: %w", err)
	}

	seen := make(map[string]bool, len(docs))
	folders := make([]string, 0, len(docs))
	for _, doc := range docs {
		folder := vaultFolderOf(doc.Path)
		if folder == "" || seen[folder] {
			continue
		}
		seen[folder] = true
		folders = append(folders, folder)
	}
	sort.Strings(folders)

	for _, folder := range folders {
		fmt.Fprintln(out, folder)
	}
	return nil
}
