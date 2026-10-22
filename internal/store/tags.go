package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// VaultTag é uma tag do vault com a contagem de documentos que a carregam.
type VaultTag struct {
	// Tag é a forma de exibição: o casing com que a tag foi criada primeiro
	// (norma do Obsidian). Key é a forma canônica usada na comparação.
	Tag string `json:"tag"`
	Key string `json:"key"`
	// Count é o número de documentos distintos que carregam a tag.
	Count         int    `json:"count"`
	ResolvedDocID string `json:"resolved_doc_id,omitempty"`
	ResolvedTitle string `json:"resolved_title,omitempty"`
}

// TagBinding é o vínculo entre um documento e uma tag literal.
type TagBinding struct {
	Tag           string `json:"tag"`
	Key           string `json:"key"`
	ResolvedDocID string `json:"resolved_doc_id,omitempty"`
}

// TagFilter restringe o inventário de tags do vault.
type TagFilter struct {
	// DocumentID é o caminho de um documento; vazio = todos.
	DocumentID string
	// Folder é uma pasta relativa ao vault; vazio = todas.
	Folder string
}

// CanonicalTagKey canonicaliza uma tag literal para a chave canônica usada na
// comparação: minúsculas, sem o '#' inicial e sem espaços nas pontas.
//
// As tags são case-insensitive no Obsidian, então `Tag`, `#Tag` e `tag` são a
// mesma tag. A chave canônica é o invariante gravado em `document_tags.tag_key`
// — e a identidade da linha — e por isso é calculada em um lugar só, para que
// os dois engines gravem exatamente a mesma chave.
func CanonicalTagKey(tag string) string {
	key := strings.TrimSpace(tag)
	key = strings.TrimLeft(key, "#")
	key = strings.TrimSpace(key)
	return strings.ToLower(key)
}

// DisplayTag devolve a forma de exibição de uma tag literal: sem o '#'
// inicial, sem espaços nas pontas e preservando o casing original.
func DisplayTag(tag string) string {
	display := strings.TrimSpace(tag)
	display = strings.TrimLeft(display, "#")
	return strings.TrimSpace(display)
}

// EscapeLikePattern (internal/store/like.go) escapa os curingas do LIKE (`%`,
// `_` e `\`) para que o padrão gerado seja literal: sem isso, a tag `a_b`
// casaria `aXb` e a busca por tag devolveria documentos que não carregam a
// tag pedida.

// normalizeFolder canonicaliza o filtro de pasta: sem espaços nas pontas e sem
// barras nas pontas. Pasta vazia significa "todas".
func normalizeFolder(folder string) string {
	folder = strings.TrimSpace(folder)
	folder = strings.Trim(folder, "/")
	return folder
}

// tagInsertBatch é o conjunto de linhas de `document_tags` prontas para
// inserção, já deduplicado por `tag_key`.
type tagInsertBatch struct {
	rows []TagBinding
}

// NormalizeTagBinding canonicaliza um vínculo documento→tag.
//
// A chave canônica vem da tag literal (ou da chave, quando a literal vem
// vazia): minúsculas, sem '#' e sem espaços nas pontas. O display preserva o
// casing original. É o mesmo passo nos dois engines, para que a linha gravada
// seja a mesma em SQLite e em Postgres.
func NormalizeTagBinding(binding TagBinding) TagBinding {
	source := binding.Tag
	if strings.TrimSpace(source) == "" {
		source = binding.Key
	}

	return TagBinding{
		Tag:           DisplayTag(source),
		Key:           CanonicalTagKey(source),
		ResolvedDocID: binding.ResolvedDocID,
	}
}

// buildTagInsertBatch canonicaliza as tags recebidas e remove as duplicatas
// por chave, preservando a primeira ocorrência (é ela que define o casing de
// exibição do documento).
//
// A identidade da linha é `(document_id, tag_key)`, então `#Tag` e `#TAG` no
// mesmo documento são a mesma linha: sem esta deduplicação, a inserção
// violaria a primary key.
func buildTagInsertBatch(tags []TagBinding) tagInsertBatch {
	batch := tagInsertBatch{rows: make([]TagBinding, 0, len(tags))}
	seen := make(map[string]bool, len(tags))

	for _, binding := range tags {
		normalized := NormalizeTagBinding(binding)
		if normalized.Key == "" {
			continue
		}
		if seen[normalized.Key] {
			continue
		}
		seen[normalized.Key] = true

		batch.rows = append(batch.rows, normalized)
	}

	return batch
}

// InsertDocumentTags substitui as tags de um documento pelas recebidas.
//
// Substitui, não acumula: as tags anteriores do documento são apagadas antes
// das novas, dentro da mesma transação, para que reindexar o mesmo arquivo
// nunca duplique tag.
func (s *PostgresStore) InsertDocumentTags(ctx context.Context, repository, documentID string, tags []TagBinding) error {
	if repository == "" {
		repository = "default"
	}
	if documentID == "" {
		return fmt.Errorf("documento sem id para tags")
	}

	batch := buildTagInsertBatch(tags)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("erro ao iniciar transacao de tags no postgres: %w", err)
	}
	defer tx.Rollback()

	// Preserva o created_at das tags que já existiam: o "casing de exibição" é
	// o da linha criada primeiro, e reindexar o documento não pode inverter
	// essa ordem.
	existing := make(map[string]int64, len(batch.rows))
	rows, err := tx.QueryContext(ctx, `
		SELECT tag_key, created_at FROM document_tags
		WHERE repository = $1 AND document_id = $2
	`, repository, documentID)
	if err != nil {
		return fmt.Errorf("erro ao ler tags existentes no postgres: %w", err)
	}
	for rows.Next() {
		var key string
		var createdAt int64
		if err := rows.Scan(&key, &createdAt); err != nil {
			rows.Close()
			return fmt.Errorf("erro ao escanear tag existente no postgres: %w", err)
		}
		existing[key] = createdAt
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("erro ao iterar tags existentes no postgres: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM document_tags WHERE repository = $1 AND document_id = $2
	`, repository, documentID); err != nil {
		return fmt.Errorf("erro ao apagar tags anteriores no postgres: %w", err)
	}

	now := time.Now().UnixNano()
	for _, binding := range batch.rows {
		createdAt := now
		if previous, ok := existing[binding.Key]; ok {
			createdAt = previous
		}

		var resolved interface{}
		if binding.ResolvedDocID != "" {
			resolved = binding.ResolvedDocID
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO document_tags (repository, document_id, tag, tag_key, resolved_doc_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, repository, documentID, binding.Tag, binding.Key, resolved, createdAt); err != nil {
			return fmt.Errorf("erro ao inserir tag no postgres: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao commitar tags no postgres: %w", err)
	}
	return nil
}

// DeleteDocumentTags remove todas as tags de um documento.
func (s *PostgresStore) DeleteDocumentTags(ctx context.Context, repository, documentID string) error {
	if repository == "" {
		repository = "default"
	}

	_, err := s.db.ExecContext(ctx, `
		DELETE FROM document_tags WHERE repository = $1 AND document_id = $2
	`, repository, documentID)
	if err != nil {
		return fmt.Errorf("erro ao apagar tags no postgres: %w", err)
	}
	return nil
}

// ListVaultTags devolve o inventário de tags do vault com a contagem de
// documentos distintos que carregam cada tag.
//
// A ordenação mora aqui, no engine: as tags saem em ordem de `tag_key`
// crescente. O casing de exibição é o da linha representativa — a de menor
// `created_at`, desempate por `document_id` e depois pela própria `tag` —,
// regra determinística e idêntica à do engine SQLite.
func (s *PostgresStore) ListVaultTags(ctx context.Context, repository string, filter TagFilter) ([]VaultTag, error) {
	folder := normalizeFolder(filter.Folder)
	folderPattern := ""
	if folder != "" {
		folderPattern = EscapeLikePattern(folder) + "/%"
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT dt.tag_key, dt.tag, dt.document_id, dt.resolved_doc_id, dt.created_at,
		       COALESCE(r.title, r.id, '')
		FROM document_tags dt
		JOIN documents d ON d.id = dt.document_id AND d.repository = dt.repository
		LEFT JOIN documents r ON r.id = dt.resolved_doc_id AND r.repository = dt.repository
		WHERE ($1 = '' OR dt.repository = $1)
		  AND ($2 = '' OR dt.document_id = $2)
		  AND ($3 = '' OR d.path LIKE $4 ESCAPE '\')
		ORDER BY dt.tag_key ASC, dt.created_at ASC, dt.document_id ASC, dt.tag ASC
	`, repository, filter.DocumentID, folder, folderPattern)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar tags do vault no postgres: %w", err)
	}
	defer rows.Close()

	tags := make([]VaultTag, 0, 16)
	index := make(map[string]int)
	for rows.Next() {
		var key, tag, documentID string
		var resolvedDocID sql.NullString
		var createdAt int64
		var resolvedTitle string

		if err := rows.Scan(&key, &tag, &documentID, &resolvedDocID, &createdAt, &resolvedTitle); err != nil {
			return nil, fmt.Errorf("erro ao escanear tag do vault no postgres: %w", err)
		}

		if position, ok := index[key]; ok {
			// A primary key (repository, document_id, tag_key) garante um único
			// registro por documento e tag: cada linha nova é um documento
			// distinto.
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
		tags = append(tags, VaultTag{
			Tag:           tag,
			Key:           key,
			Count:         1,
			ResolvedDocID: resolvedID,
			ResolvedTitle: resolvedTitle,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar tags do vault no postgres: %w", err)
	}

	return tags, nil
}

// FindVaultDocumentsByTag devolve os documentos que carregam a tag pedida ou
// qualquer uma das suas descendentes (`a` casa `a/b`).
//
// A tag recebida é canonicalizada antes da comparação: `mem tag Tag`,
// `mem tag #Tag` e `mem tag tag` devolvem o mesmo resultado.
func (s *PostgresStore) FindVaultDocumentsByTag(ctx context.Context, repository string, tag string) ([]VaultDocument, error) {
	key := CanonicalTagKey(tag)
	if key == "" {
		return []VaultDocument{}, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT d.id, d.path, COALESCE(d.title, d.id), COALESCE(d.mtime, 0), COALESCE(d.size, 0)
		FROM document_tags dt
		JOIN documents d ON d.id = dt.document_id AND d.repository = dt.repository
		WHERE ($1 = '' OR dt.repository = $1)
		  AND (dt.tag_key = $2 OR dt.tag_key LIKE $3 ESCAPE '\')
		ORDER BY d.path ASC
	`, repository, key, EscapeLikePattern(key)+"/%")
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar documentos por tag no postgres: %w", err)
	}
	defer rows.Close()

	documents := make([]VaultDocument, 0, 16)
	for rows.Next() {
		var doc VaultDocument
		if err := rows.Scan(&doc.ID, &doc.Path, &doc.Title, &doc.MTime, &doc.Size); err != nil {
			return nil, fmt.Errorf("erro ao escanear documento por tag no postgres: %w", err)
		}
		documents = append(documents, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar documentos por tag no postgres: %w", err)
	}

	return documents, nil
}

// ListDocumentIDTitles devolve o mapa título → id dos documentos indexados,
// com o título no sentido de `COALESCE(title, id)`.
func (s *PostgresStore) ListDocumentIDTitles(ctx context.Context, repository string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(title, id), id FROM documents
		WHERE ($1 = '' OR repository = $1)
		ORDER BY id ASC
	`, repository)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar titulos de documentos no postgres: %w", err)
	}
	defer rows.Close()

	titles := make(map[string]string)
	for rows.Next() {
		var title, id string
		if err := rows.Scan(&title, &id); err != nil {
			return nil, fmt.Errorf("erro ao escanear titulo de documento no postgres: %w", err)
		}
		if _, exists := titles[title]; !exists {
			titles[title] = id
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar titulos de documentos no postgres: %w", err)
	}

	return titles, nil
}
