package db

const Schema = `
-- Documentos principais
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    title TEXT,
    updated_at INTEGER NOT NULL
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
    relation TEXT NOT NULL,   -- 'links_to', 'tagged_as'
    PRIMARY KEY (source_id, target_id, relation)
);
CREATE INDEX IF NOT EXISTS idx_edges_target ON graph_edges(target_id);
`
