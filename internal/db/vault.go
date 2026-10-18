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

// FindVaultDocumentsByPath resolve um caminho do usuário para documentos do
// índice: path exato OU path terminado em "/<basename>", em ordem alfabética.
// Espelho SQLite de PostgresStore.FindVaultDocumentsByPath.
func FindVaultDocumentsByPath(ctx context.Context, database *sql.DB, query string) ([]store.VaultDocument, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT id, path, COALESCE(title, ''), COALESCE(mtime, 0), COALESCE(size, 0)
		FROM documents
		WHERE path = ? OR path LIKE '%/' || ?
		ORDER BY path ASC
	`, query, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar documento por caminho: %w", err)
	}
	defer rows.Close()

	docs := make([]store.VaultDocument, 0, 4)
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

// ReadVaultDocumentContent devolve o texto bruto do documento remontando os
// chunks na ordem do índice. Espelho SQLite de
// PostgresStore.ReadVaultDocumentContent.
func ReadVaultDocumentContent(ctx context.Context, database *sql.DB, docID string) (string, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT content FROM chunks
		WHERE document_id = ?
		ORDER BY chunk_index ASC
	`, docID)
	if err != nil {
		return "", fmt.Errorf("erro ao ler conteúdo do documento: %w", err)
	}
	defer rows.Close()

	var chunks []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return "", fmt.Errorf("erro ao escanear chunk do documento: %w", err)
		}
		chunks = append(chunks, content)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("erro ao iterar chunks do documento: %w", err)
	}
	if len(chunks) == 0 {
		return "", fmt.Errorf("documento %s não tem chunks indexados", docID)
	}
	return store.JoinVaultChunks(chunks), nil
}

// ListVaultDocumentsByMTime devolve os documentos mais recentemente editados,
// do mtime mais novo para o mais antigo.
//
// Documento com mtime = 0 (indexado antes de a coluna existir, ADR-057) vai
// para o FIM, não para o começo. Sem esse CASE, o zero venceria todos os
// timestamps e `mem recents` abriria listando justamente as notas cujo tempo de
// edição ele não conhece — o oposto do que o comando promete.
func ListVaultDocumentsByMTime(ctx context.Context, database *sql.DB, limit int) ([]store.VaultDocument, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT id, path, COALESCE(title, ''), COALESCE(mtime, 0), COALESCE(size, 0)
		FROM documents
		ORDER BY CASE WHEN COALESCE(mtime, 0) <= 0 THEN 1 ELSE 0 END ASC,
		         COALESCE(mtime, 0) DESC,
		         path ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar documentos recentes: %w", err)
	}
	defer rows.Close()

	docs := make([]store.VaultDocument, 0, limit)
	for rows.Next() {
		var doc store.VaultDocument
		if err := rows.Scan(&doc.ID, &doc.Path, &doc.Title, &doc.MTime, &doc.Size); err != nil {
			return nil, fmt.Errorf("erro ao escanear documento recente: %w", err)
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar documentos recentes: %w", err)
	}
	return docs, nil
}
