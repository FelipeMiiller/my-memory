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
`
