package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// ListVaultDocuments devolve todos os documentos do índice em ordem alfabética
// de path. É o espelho SQLite de PostgresStore.ListVaultDocuments: os dois
// engines precisam devolver a mesma projeção para que `mem files`, `mem folders`
// e `mem random` se comportem igual em qualquer vault.
//
// O SQLite não tem coluna `repository`: o vault local tem um repositório só.
func ListVaultDocuments(ctx context.Context, database *sql.DB) ([]store.VaultDocument, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT id, path, COALESCE(title, ''), COALESCE(mtime, 0), COALESCE(size, 0)
		FROM documents
		ORDER BY path ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar documentos do vault: %w", err)
	}
	defer rows.Close()

	docs := make([]store.VaultDocument, 0, 16)
	for rows.Next() {
		var doc store.VaultDocument
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
