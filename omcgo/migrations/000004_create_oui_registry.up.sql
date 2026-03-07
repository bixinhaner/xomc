-- Phase 2: OUI Registry table
-- Maps IEEE OUI codes to manufacturers for device identification

CREATE TABLE oui_registry (
    oui              VARCHAR(6) PRIMARY KEY,
    manufacturer     VARCHAR(128) NOT NULL,
    short_name       VARCHAR(32) NOT NULL,
    country          VARCHAR(64),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oui_manufacturer ON oui_registry (short_name);

-- Seed data: major base station manufacturers
INSERT INTO oui_registry (oui, manufacturer, short_name, country) VALUES
    ('00E0FC', 'Huawei Technologies Co., Ltd.', 'Huawei', 'China'),
    ('001E7E', 'ZTE Corporation', 'ZTE', 'China'),
    ('000DB9', 'Ericsson AB', 'Ericsson', 'Sweden'),
    ('0004F2', 'Nokia Corporation', 'Nokia', 'Finland'),
    ('58FB96', 'Comba Telecom Systems', 'Comba', 'China'),
    ('D4612E', 'Datang Mobile Communications', 'Datang', 'China'),
    ('00259C', 'Cisco-Linksys LLC', 'Cisco', 'USA'),
    ('7C7A53', 'Ruijie Networks Co., Ltd.', 'Ruijie', 'China');
