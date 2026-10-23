package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// InsertDocumentTags substitui as tags de um documento pelas recebidas.
//
// Substitui, não acumula: as tags anteriores do documento são apagadas antes
// das novas, dentro da mesma transação, para que reindexar o mesmo arquivo
// nunca duplique tag.
func InsertDocumentTags(ctx context.Context, database *sql.DB, documentID string, tags []store.TagBinding) error {
	if documentID == "" {
		return fmt.Errorf("documento sem id para tags")
	}

	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("erro ao iniciar transacao de tags no sqlite: %w", err)
	}
	defer tx.Rollback()

	// Preserva o created_at das tags que já existiam: o "casing de exibição" é
	// o da linha criada primeiro, e reindexar o documento não pode inverter
	// essa ordem.
	existing := make(map[string]int64, len(tags))
	rows, err := tx.QueryContext(ctx, `
		SELECT tag_key, created_at FROM document_tags WHERE document_id = ?
	`, documentID)
	if err != nil {
		return fmt.Errorf("erro ao ler tags existentes no sqlite: %w", err)
	}
	for rows.Next() {
		var key string
		var createdAt int64
		if err := rows.Scan(&key, &createdAt); err != nil {
			rows.Close()
			return fmt.Errorf("erro ao escanear tag existente no sqlite: %w", err)
		}
		existing[key] = createdAt
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("erro ao iterar tags existentes no sqlite: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM document_tags WHERE document_id = ?
	`, documentID); err != nil {
		return fmt.Errorf("erro ao apagar tags anteriores no sqlite: %w", err)
	}

	now := time.Now().UnixNano()
	seen := make(map[string]bool, len(tags))
	for _, binding := range tags {
		normalized := store.NormalizeTagBinding(binding)
		key := normalized.Key
		if key == "" {
			continue
		}
		// A identidade da linha é (document_id, tag_key): `#Tag` e `#TAG` são a
		// mesma linha e, sem esta guarda, a segunda inserção violaria a
		// primary key.
		if seen[key] {
			continue
		}
		seen[key] = true

		createdAt := now
		if previous, ok := existing[key]; ok {
			createdAt = previous
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO document_tags (document_id, tag, tag_key, resolved_doc_id, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, documentID, normalized.Tag, key, nullIfEmpty(normalized.ResolvedDocID), createdAt); err != nil {
			return fmt.Errorf("erro ao inserir tag no sqlite: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao commitar tags no sqlite: %w", err)
	}
	return nil
}

// DeleteDocumentTags remove todas as tags de um documento.
func DeleteDocumentTags(ctx context.Context, database *sql.DB, documentID string) error {
	_, err := database.ExecContext(ctx, `
		DELETE FROM document_tags WHERE document_id = ?
	`, documentID)
	if err != nil {
		return fmt.Errorf("erro ao apagar tags no sqlite: %w", err)
	}
	return nil
}

// ListVaultTags devolve o inventário de tags do vault com a contagem de
// documentos distintos que carregam cada tag.
//
// A ordenação mora aqui, no engine: as tags saem em ordem de `tag_key`
// crescente. O casing de exibição é o da linha representativa — a de menor
// `created_at`, desempate por `document_id` e depois pela própria `tag` —,
// regra determinística e idêntica à do engine Postgres.
func ListVaultTags(ctx context.Context, database *sql.DB, filter store.TagFilter) ([]store.VaultTag, error) {
	folder := normalizeTagFolder(filter.Folder)
	folderPrefix := ""
	folderInner := ""
	if folder != "" {
		// O path indexado usa o separador da plataforma e pode ser absoluto
		// ou relativo ao vault. A comparação acontece sobre a forma com "/",
		// e a pasta precisa ser um segmento inteiro — "docs" não pode casar
		// "docs-old". É a mesma regra de `matchesFolder`, em
		// `cmd/mem/files.go`.
		escaped := store.EscapeLikePattern(folder)
		folderPrefix = escaped + "/%"
		folderInner = "%/" + escaped + "/%"
	}

	rows, err := database.QueryContext(ctx, `
		SELECT dt.tag_key, dt.tag, dt.document_id, dt.resolved_doc_id, dt.created_at,
		       COALESCE(r.title, r.id, '')
		FROM document_tags dt
		JOIN documents d ON d.id = dt.document_id
		LEFT JOIN documents r ON r.id = dt.resolved_doc_id
		WHERE (? = '' OR dt.document_id = ?)
		  AND (? = '' OR REPLACE(d.path, '\', '/') LIKE ? ESCAPE '\'
		            OR REPLACE(d.path, '\', '/') LIKE ? ESCAPE '\')
		ORDER BY dt.tag_key ASC, dt.created_at ASC, dt.document_id ASC, dt.tag ASC
	`, filter.DocumentID, filter.DocumentID, folder, folderPrefix, folderInner)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar tags do vault no sqlite: %w", err)
	}
	defer rows.Close()

	tags := make([]store.VaultTag, 0, 16)
	index := make(map[string]int)
	for rows.Next() {
		var key, tag, documentID string
		var resolvedDocID sql.NullString
		var createdAt int64
		var resolvedTitle string

		if err := rows.Scan(&key, &tag, &documentID, &resolvedDocID, &createdAt, &resolvedTitle); err != nil {
			return nil, fmt.Errorf("erro ao escanear tag do vault no sqlite: %w", err)
		}

		if position, ok := index[key]; ok {
			// A primary key (document_id, tag_key) garante um único registro
			// por documento e tag: cada linha nova é um documento distinto.
			tags[position].Count++
			continue
		}

		index[key] = len(tags)
		// A linha representativa — a de menor created_at, desempate por
		// document_id e depois pela própria tag — define o casing de exibição
		// e a resolução. O título vem do documento resolvido
		// (COALESCE(title, id)) e fica vazio quando a tag não resolve para
		// nenhum documento.
		//
		// O documento resolvido pode ter sido podado depois: a linha da tag
		// sobrevive ao documento, porque a chave não é uma foreign key. O id
		// pendurado não pode vazar para a saída, e `resolvedTitle` só é vazio
		// quando o documento não existe mais.
		resolvedID := resolvedDocID.String
		if resolvedTitle == "" {
			resolvedID = ""
		}
		tags = append(tags, store.VaultTag{
			Tag:           tag,
			Key:           key,
			Count:         1,
			ResolvedDocID: resolvedID,
			ResolvedTitle: resolvedTitle,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar tags do vault no sqlite: %w", err)
	}

	return tags, nil
}

// FindVaultDocumentsByTag devolve os documentos que carregam a tag pedida ou
// qualquer uma das suas descendentes (`a` casa `a/b`).
//
// O padrão do LIKE é escapado porque `_` e `%` são curingas: sem o escape, a
// tag `a_b` casaria `aXb`.
func FindVaultDocumentsByTag(ctx context.Context, database *sql.DB, tag string) ([]store.VaultDocument, error) {
	key := store.CanonicalTagKey(tag)
	if key == "" {
		return []store.VaultDocument{}, nil
	}

	rows, err := database.QueryContext(ctx, `
		SELECT DISTINCT d.id, d.path, COALESCE(d.title, d.id), COALESCE(d.mtime, 0), COALESCE(d.size, 0)
		FROM document_tags dt
		JOIN documents d ON d.id = dt.document_id
		WHERE dt.tag_key = ? OR dt.tag_key LIKE ? ESCAPE '\'
		ORDER BY d.path ASC
	`, key, store.EscapeLikePattern(key)+"/%")
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar documentos por tag no sqlite: %w", err)
	}
	defer rows.Close()

	documents := make([]store.VaultDocument, 0, 16)
	for rows.Next() {
		var doc store.VaultDocument
		if err := rows.Scan(&doc.ID, &doc.Path, &doc.Title, &doc.MTime, &doc.Size); err != nil {
			return nil, fmt.Errorf("erro ao escanear documento por tag no sqlite: %w", err)
		}
		documents = append(documents, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar documentos por tag no sqlite: %w", err)
	}

	return documents, nil
}

// ListDocumentIDTitles devolve o mapa título → id dos documentos indexados,
// com o título no sentido de `COALESCE(title, id)`.
func ListDocumentIDTitles(ctx context.Context, database *sql.DB) (map[string]string, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT COALESCE(title, id), id FROM documents ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar titulos de documentos no sqlite: %w", err)
	}
	defer rows.Close()

	titles := make(map[string]string)
	for rows.Next() {
		var title, id string
		if err := rows.Scan(&title, &id); err != nil {
			return nil, fmt.Errorf("erro ao escanear titulo de documento no sqlite: %w", err)
		}
		if _, exists := titles[title]; !exists {
			titles[title] = id
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar titulos de documentos no sqlite: %w", err)
	}

	return titles, nil
}

// nullIfEmpty converte string vazia em NULL, para que `resolved_doc_id`
// guarde ausência de resolução em vez de string vazia.
func nullIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

// normalizeTagFolder canonicaliza o filtro de pasta: sem espaços nas pontas,
// com o separador "/" e sem barras nas pontas. Pasta vazia significa "todas".
func normalizeTagFolder(folder string) string {
	folder = strings.TrimSpace(folder)
	folder = strings.ReplaceAll(folder, `\`, "/")
	return strings.Trim(folder, "/")
}
