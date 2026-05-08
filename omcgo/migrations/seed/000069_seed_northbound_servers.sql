-- 北向 OSS 主备服务器配置初始化（与 migrations/000065_northbound_servers.sql 配套）
--
-- 主备各 1 行；primary 默认激活。host/port 用前端原 mock 默认值（192.168.1.100/101 :8081），
-- 实际生产部署后由管理员在 system/config 北向设置页编辑。
--
-- 幂等：ON CONFLICT (role) DO NOTHING — 已有同名 role 不覆盖，避免运维改过的配置被 seed 覆盖。

-- +goose Up
INSERT INTO northbound_servers (id, role, host, port, description, is_active) VALUES
    ('40000000-0000-0000-0000-000000000001'::uuid, 'primary', '192.168.1.100', 8081, '主用 OSS 服务器', TRUE),
    ('40000000-0000-0000-0000-000000000002'::uuid, 'standby', '192.168.1.101', 8081, '备用 OSS 服务器', FALSE)
ON CONFLICT (role) DO NOTHING;

-- +goose Down
DELETE FROM northbound_servers WHERE role IN ('primary','standby');
