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

// FindVaultDocumentsByPath resolve um caminho do usuário para documentos do
// índice. A query é o path exato OU qualquer path terminado em "/<basename>",
// para que `mem read x.md` funcione sem o usuário escrever a pasta inteira.
//
// Devolve todos os candidatos: escolher entre eles é decisão do comando, que
// precisa reportar ambiguidade em vez de chutar um. repo vazio = todos os
// repositórios.
func (s *PostgresStore) FindVaultDocumentsByPath(ctx context.Context, repo, query string) ([]VaultDocument, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path, COALESCE(title, ''), COALESCE(mtime, 0), COALESCE(size, 0)
		FROM documents
		WHERE ($1 = '' OR repository = $1)
		  AND (path = $2 OR path LIKE '%/' || $2)
		ORDER BY path ASC
	`, repo, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar documento por caminho no postgres: %w", err)
	}
	defer rows.Close()

	docs := make([]VaultDocument, 0, 4)
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

// ReadVaultDocumentContent devolve o texto bruto do documento remontando os
// chunks na ordem do índice.
func (s *PostgresStore) ReadVaultDocumentContent(ctx context.Context, repo, docID string) (string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT content FROM chunks
		WHERE ($1 = '' OR repository = $1) AND document_id = $2
		ORDER BY chunk_index ASC
	`, repo, docID)
	if err != nil {
		return "", fmt.Errorf("erro ao ler conteúdo do documento no postgres: %w", err)
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
	return JoinVaultChunks(chunks), nil
}

// ListVaultDeadends devolve as notas que têm ao menos uma aresta de saída e
// nenhuma de entrada. Espelho PostgreSQL de db.ListVaultDeadends.
func (s *PostgresStore) ListVaultDeadends(ctx context.Context, repo string) ([]VaultDocument, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.path, COALESCE(d.title, ''), COALESCE(d.mtime, 0), COALESCE(d.size, 0)
		FROM documents d
		WHERE EXISTS (SELECT 1 FROM graph_edges e WHERE e.repository = d.repository AND e.source_id = d.id)
		  AND NOT EXISTS (SELECT 1 FROM graph_edges e WHERE e.repository = d.repository AND e.target_id = d.id)
		ORDER BY d.path ASC
	`, repo)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar deadends no postgres: %w", err)
	}
	defer rows.Close()

	docs := make([]VaultDocument, 0, 8)
	for rows.Next() {
		var doc VaultDocument
		if err := rows.Scan(&doc.ID, &doc.Path, &doc.Title, &doc.MTime, &doc.Size); err != nil {
			return nil, fmt.Errorf("erro ao escanear deadend: %w", err)
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar deadends: %w", err)
	}
	return docs, nil
}
