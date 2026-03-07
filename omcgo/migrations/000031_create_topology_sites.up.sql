-- Sites table for GIS / topology site management
CREATE TABLE sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    domain_id UUID REFERENCES device_groups(id) ON DELETE SET NULL,
    address TEXT,
    longitude DOUBLE PRECISION,
    latitude DOUBLE PRECISION,
    device_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sites_domain ON sites(domain_id);
CREATE INDEX idx_sites_status ON sites(status);
CREATE INDEX idx_sites_geo ON sites(longitude, latitude);
CREATE TRIGGER trigger_sites_updated_at
    BEFORE UPDATE ON sites FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Topology nodes
CREATE TABLE topo_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label VARCHAR(200) NOT NULL,
    node_type VARCHAR(20) NOT NULL,
    x DOUBLE PRECISION NOT NULL DEFAULT 0,
    y DOUBLE PRECISION NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'online',
    device_sn VARCHAR(64),
    site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    domain_id UUID REFERENCES device_groups(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_topo_nodes_type ON topo_nodes(node_type);
CREATE INDEX idx_topo_nodes_domain ON topo_nodes(domain_id);
CREATE INDEX idx_topo_nodes_device ON topo_nodes(device_sn);
CREATE TRIGGER trigger_topo_nodes_updated_at
    BEFORE UPDATE ON topo_nodes FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Topology edges
CREATE TABLE topo_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES topo_nodes(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES topo_nodes(id) ON DELETE CASCADE,
    label VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_topo_edges_source ON topo_edges(source_id);
CREATE INDEX idx_topo_edges_target ON topo_edges(target_id);
