package db

const Schema = `
-- Documentos principais
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    title TEXT,
    updated_at INTEGER NOT NULL,
    content_hash TEXT,
    abstract TEXT,
    category TEXT DEFAULT 'resource'
);

-- Chunks textuais para busca fina
CREATE TABLE IF NOT EXISTS chunks (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL
);

-- Full-Text Search com FTS5 nativo
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
    chunk_id UNINDEXED,
    content,
    tokenize = 'porter unicode61'
);

-- Busca Vetorial via sqlite-vec padrão (dimensão 768 float32 = 3072 bytes)
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_vec USING vec0(
    chunk_id TEXT PRIMARY KEY,
    embedding float[768]
);

-- Busca Vetorial Ultracompacta via TurboQuant (4-bit = 384 bytes por chunk)
CREATE TABLE IF NOT EXISTS chunks_turboquant (
    chunk_id TEXT PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,
    scale REAL NOT NULL,
    data BLOB NOT NULL
);

-- Grafo: Nós e Arestas
CREATE TABLE IF NOT EXISTS graph_nodes (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,       -- 'note', 'tag', 'concept'
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS graph_edges (
    source_id TEXT NOT NULL REFERENCES graph_nodes(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL REFERENCES graph_nodes(id) ON DELETE CASCADE,
    relation TEXT NOT NULL,   -- 'links_to', 'implements', 'tagged_as'
    epistemic_status TEXT NOT NULL DEFAULT 'EXTRACTED', -- 'EXTRACTED', 'INFERRED'
    weight REAL NOT NULL DEFAULT 1.0,
    PRIMARY KEY (source_id, target_id, relation)
);
CREATE INDEX IF NOT EXISTS idx_edges_target ON graph_edges(target_id);

-- Event Log (ADR-043): substrato durável do event_runtime
-- sequence monotônico + event_id único garantem ordem causal e idempotência de retry.
CREATE TABLE IF NOT EXISTS event_log (
    sequence INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT UNIQUE NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1,
    event_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    revision INTEGER,
    payload BLOB NOT NULL,
    headers BLOB NOT NULL,
    created_at TEXT NOT NULL,
    acked_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_event_log_aggregate ON event_log(aggregate_id);
CREATE INDEX IF NOT EXISTS idx_event_log_unacked ON event_log(sequence) WHERE acked_at IS NULL;

-- Cursor per-subscriber: ponto de retomada crash-safe de cada projeção/dispatcher.
CREATE TABLE IF NOT EXISTS projection_cursor (
    projection_name TEXT PRIMARY KEY,
    last_sequence INTEGER NOT NULL,
    updated_at TEXT NOT NULL
);

-- Code AST (ADR-047): tabelas code_* para indexação estruturada de código-fonte
-- via tree-sitter. Persistidas no mesmo memory.db (coerente com ADR-001).
CREATE TABLE IF NOT EXISTS code_files (
    id INTEGER PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    ast_hash TEXT,
    indexed_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS code_files_language_idx ON code_files(language);
CREATE INDEX IF NOT EXISTS code_files_content_hash_idx ON code_files(content_hash);

CREATE TABLE IF NOT EXISTS code_symbols (
    id INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    qualified_name TEXT,
    signature TEXT,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    start_col INTEGER,
    end_col INTEGER,
    doc_comment TEXT,
    parent_symbol_id INTEGER REFERENCES code_symbols(id) ON DELETE SET NULL,
    UNIQUE(file_id, kind, qualified_name, start_line)
);
CREATE INDEX IF NOT EXISTS code_symbols_kind_name_idx ON code_symbols(kind, name);
CREATE INDEX IF NOT EXISTS code_symbols_qualified_name_idx ON code_symbols(qualified_name);
CREATE INDEX IF NOT EXISTS code_symbols_file_id_idx ON code_symbols(file_id);

CREATE TABLE IF NOT EXISTS code_edges (
    id INTEGER PRIMARY KEY,
    src_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    dst_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    confidence REAL NOT NULL DEFAULT 1.0,
    UNIQUE(src_symbol_id, dst_symbol_id, kind)
);
CREATE INDEX IF NOT EXISTS code_graph_kind_idx ON code_edges(kind);
CREATE INDEX IF NOT EXISTS code_graph_src_idx ON code_edges(src_symbol_id);
CREATE INDEX IF NOT EXISTS code_graph_dst_idx ON code_edges(dst_symbol_id);

-- Arestas de baixa confiança (cross-file) ficam em tabela paralela para
-- revisão manual sem poluir o grafo principal (ADR-047 §Consequences — Negative).
CREATE TABLE IF NOT EXISTS code_edges_uncertain (
    id INTEGER PRIMARY KEY,
    src_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    dst_candidate_name TEXT NOT NULL,
    kind TEXT NOT NULL,
    confidence REAL NOT NULL,
    reason TEXT,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL,
    UNIQUE(src_symbol_id, dst_candidate_name, kind)
);
CREATE INDEX IF NOT EXISTS code_edges_uncertain_src_idx ON code_edges_uncertain(src_symbol_id);
CREATE INDEX IF NOT EXISTS code_edges_uncertain_conf_idx ON code_edges_uncertain(confidence);
`

const FallbackSchema = `
-- Documentos principais
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    title TEXT,
    updated_at INTEGER NOT NULL,
    content_hash TEXT,
    abstract TEXT,
    category TEXT DEFAULT 'resource'
);

-- Chunks textuais para busca fina
CREATE TABLE IF NOT EXISTS chunks (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL
);

-- Full-Text Search com FTS5 nativo
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
    chunk_id UNINDEXED,
    content,
    tokenize = 'porter unicode61'
);

-- Tabela padrão de vetores para plataformas sem extensão C vec0
CREATE TABLE IF NOT EXISTS chunks_vec (
    chunk_id TEXT PRIMARY KEY,
    embedding BLOB
);

-- Busca Vetorial Ultracompacta via TurboQuant (4-bit = 384 bytes por chunk)
CREATE TABLE IF NOT EXISTS chunks_turboquant (
    chunk_id TEXT PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,
    scale REAL NOT NULL,
    data BLOB NOT NULL
);

-- Grafo: Nós e Arestas
CREATE TABLE IF NOT EXISTS graph_nodes (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,       -- 'note', 'tag', 'concept'
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS graph_edges (
    source_id TEXT NOT NULL REFERENCES graph_nodes(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL REFERENCES graph_nodes(id) ON DELETE CASCADE,
    relation TEXT NOT NULL,   -- 'links_to', 'implements', 'tagged_as'
    epistemic_status TEXT NOT NULL DEFAULT 'EXTRACTED', -- 'EXTRACTED', 'INFERRED'
    weight REAL NOT NULL DEFAULT 1.0,
    PRIMARY KEY (source_id, target_id, relation)
);
CREATE INDEX IF NOT EXISTS idx_edges_target ON graph_edges(target_id);

-- Event Log (ADR-043): substrato durável do event_runtime
-- sequence monotônico + event_id único garantem ordem causal e idempotência de retry.
CREATE TABLE IF NOT EXISTS event_log (
    sequence INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT UNIQUE NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1,
    event_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    revision INTEGER,
    payload BLOB NOT NULL,
    headers BLOB NOT NULL,
    created_at TEXT NOT NULL,
    acked_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_event_log_aggregate ON event_log(aggregate_id);
CREATE INDEX IF NOT EXISTS idx_event_log_unacked ON event_log(sequence) WHERE acked_at IS NULL;

-- Cursor per-subscriber: ponto de retomada crash-safe de cada projeção/dispatcher.
CREATE TABLE IF NOT EXISTS projection_cursor (
    projection_name TEXT PRIMARY KEY,
    last_sequence INTEGER NOT NULL,
    updated_at TEXT NOT NULL
);

-- Code AST (ADR-047, FallbackSchema): mesmas tabelas code_* — DDL portável.
CREATE TABLE IF NOT EXISTS code_files (
    id INTEGER PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    ast_hash TEXT,
    indexed_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS code_files_language_idx ON code_files(language);
CREATE INDEX IF NOT EXISTS code_files_content_hash_idx ON code_files(content_hash);

CREATE TABLE IF NOT EXISTS code_symbols (
    id INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    qualified_name TEXT,
    signature TEXT,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    start_col INTEGER,
    end_col INTEGER,
    doc_comment TEXT,
    parent_symbol_id INTEGER REFERENCES code_symbols(id) ON DELETE SET NULL,
    UNIQUE(file_id, kind, qualified_name, start_line)
);
CREATE INDEX IF NOT EXISTS code_symbols_kind_name_idx ON code_symbols(kind, name);
CREATE INDEX IF NOT EXISTS code_symbols_qualified_name_idx ON code_symbols(qualified_name);
CREATE INDEX IF NOT EXISTS code_symbols_file_id_idx ON code_symbols(file_id);

CREATE TABLE IF NOT EXISTS code_edges (
    id INTEGER PRIMARY KEY,
    src_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    dst_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    confidence REAL NOT NULL DEFAULT 1.0,
    UNIQUE(src_symbol_id, dst_symbol_id, kind)
);
CREATE INDEX IF NOT EXISTS code_graph_kind_idx ON code_edges(kind);
CREATE INDEX IF NOT EXISTS code_graph_src_idx ON code_edges(src_symbol_id);
CREATE INDEX IF NOT EXISTS code_graph_dst_idx ON code_edges(dst_symbol_id);

CREATE TABLE IF NOT EXISTS code_edges_uncertain (
    id INTEGER PRIMARY KEY,
    src_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    dst_candidate_name TEXT NOT NULL,
    kind TEXT NOT NULL,
    confidence REAL NOT NULL,
    reason TEXT,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL,
    UNIQUE(src_symbol_id, dst_candidate_name, kind)
);
CREATE INDEX IF NOT EXISTS code_edges_uncertain_src_idx ON code_edges_uncertain(src_symbol_id);
CREATE INDEX IF NOT EXISTS code_edges_uncertain_conf_idx ON code_edges_uncertain(confidence);
`
