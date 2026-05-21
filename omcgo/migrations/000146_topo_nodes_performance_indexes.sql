-- +goose Up
-- Performance indexes for topology graph queries.
-- These indexes optimize the most common query patterns for the topology graph API.

-- Composite index for domain/node_type/status filtering (most common filter combination)
CREATE INDEX IF NOT EXISTS idx_topo_nodes_domain_type_status
    ON topo_nodes(domain_id, node_type, status);

-- Index for label sorting (used when alphabetical ordering is needed)
CREATE INDEX IF NOT EXISTS idx_topo_nodes_label_active
    ON topo_nodes(label ASC);

-- Index for source_id and target_id in edges (used by ListByNodeIDs)
CREATE INDEX IF NOT EXISTS idx_topo_edges_source_id
    ON topo_edges(source_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_topo_edges_target_id
    ON topo_edges(target_id)
    WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS idx_topo_nodes_domain_type_status;
DROP INDEX IF EXISTS idx_topo_nodes_label_active;
DROP INDEX IF EXISTS idx_topo_edges_source_id;
DROP INDEX IF EXISTS idx_topo_edges_target_id;
