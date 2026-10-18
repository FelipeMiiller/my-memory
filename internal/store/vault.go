package store

import (
	"context"
	"fmt"
)

// VaultDocument é a projeção mínima de `documents` que os comandos de leitura
// do vault consomem (files, folders, read, outline, wordcount, random,
// deadends, file, recents). Existe para que a semântica de cada campo fique
// escrita uma vez e os dois engines devolvam exatamente a mesma coisa.
type VaultDocument struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Title string `json:"title"`
	// MTime é o instante da última edição do arquivo, persistido na indexação
	// (ADR-057). 0 = documento indexado antes de a coluna existir.
	MTime int64 `json:"mtime"`
	// Size são os bytes do arquivo no momento da indexação. 0 = desconhecido.
	Size int64 `json:"size"`
}

// ListVaultDocuments devolve todos os documentos do repositório em ordem
// alfabética de path. repo vazio significa "todos os repositórios", o mesmo
// idioma de ListDocumentTitles e GetDocumentsMetadata.
//
// A ordenação mora aqui, no engine, e não no comando: é o que torna a saída
// de `mem files` estável entre execuções e passível de diff.
func (s *PostgresStore) ListVaultDocuments(ctx context.Context, repo string) ([]VaultDocument, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path, COALESCE(title, ''), COALESCE(mtime, 0), COALESCE(size, 0)
		FROM documents
		WHERE ($1 = '' OR repository = $1)
		ORDER BY path ASC
	`, repo)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar documentos do vault no postgres: %w", err)
	}
	defer rows.Close()

	docs := make([]VaultDocument, 0, 16)
	for rows.Next() {
		var doc VaultDocument
		if err := rows.Scan(&doc.ID, &doc.Path, &doc.Title, &doc.MTime, &doc.Size); err != nil {
			return nil, fmt.Errorf("erro ao escanear documento do vault: %w", err)
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar documentos do vault: %w", err)
	}
	return docs, nil
}
