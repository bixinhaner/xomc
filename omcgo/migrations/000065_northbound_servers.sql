-- 北向 OSS 主备服务器配置（system/config 北向设置 切换功能）
--
-- 业务背景：ACS 向上游 OSS 推送数据（PM/告警/配置）的目标服务器需要主备双活配置，
-- 由系统管理员在 UI 上手动切换激活组（primary <-> standby）。当前 wave 仅做配置
-- 存储 + 切换 API + UI 接入，实际推送 engine 读 active server 由后续 wave 接入。
--
-- 关联模块：F08 北向/OSS 接口；前端 system/config NorthboundSettings 主备切换 UI。
-- partial unique index 保证全表只能有 1 行 is_active=true，防裸 SQL 误写。

-- +goose Up
CREATE TABLE IF NOT EXISTS northbound_servers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role        VARCHAR(16)  NOT NULL UNIQUE
                CHECK (role IN ('primary','standby')),
    host        VARCHAR(255) NOT NULL,
    port        INTEGER      NOT NULL CHECK (port > 0 AND port <= 65535),
    description VARCHAR(255) NOT NULL DEFAULT '',
    is_active   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 全表只能有一行 is_active=true（partial unique index）
CREATE UNIQUE INDEX IF NOT EXISTS uniq_northbound_servers_active
    ON northbound_servers (is_active)
    WHERE is_active = TRUE;

-- updated_at 触发器（与项目其它表对齐，复用 000001 的 update_updated_at_column）
DROP TRIGGER IF EXISTS trg_northbound_servers_updated_at ON northbound_servers;
CREATE TRIGGER trg_northbound_servers_updated_at
    BEFORE UPDATE ON northbound_servers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS trg_northbound_servers_updated_at ON northbound_servers;
DROP INDEX IF EXISTS uniq_northbound_servers_active;
DROP TABLE IF EXISTS northbound_servers;
