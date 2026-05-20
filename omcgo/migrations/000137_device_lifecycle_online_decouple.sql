-- +goose Up
-- ============================================================
-- 000137_device_lifecycle_online_decouple.sql
-- T-0162: 设备列表前后端对齐总改 — Schema 解耦
--
-- 把 devices.status 单字段承载的 7 状态（discovered/registered/provisioning/
-- active/maintenance/offline/decommissioned）拆解为两个正交字段：
--
--   1) lifecycle_state — 业务流程进度
--      取值：discovered / registered / provisioning / commissioned /
--           maintenance / decommissioned   （**不含 offline**，offline 移到 is_online）
--      写入：Admin API / Provisioning Engine / ACS Bootstrap / 运维操作
--
--   2) is_online — 实时心跳活跃度
--      取值：true / false
--      写入：HeartbeatMonitor / ACS Inform 接收
--
-- 设计文档：docs/design/device-lifecycle-online-status-decouple-20260520.md
--
-- D1（用户决策）：硬切，同迁移内 DROP 老 status 列，**不留过渡期**。
-- ============================================================

-- ─── 1. 新增 lifecycle_state ────────────────────────────────────────
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS lifecycle_state VARCHAR(20) NOT NULL DEFAULT 'registered';

-- 从老 status 回填（带 device_info JOIN 拿 first_online_time 判断"是否曾激活"）
-- 规则：
--   active                                → commissioned          （在 § ACS Inform 路径恢复后由 is_online 表达"在线"）
--   offline + first_online_time NOT NULL  → commissioned          （曾上线过，本期掉线 → 视为已入网但 is_online=false）
--   offline + first_online_time NULL      → registered            （从未上线过，视为未完成入网）
--   其他（discovered/registered/provisioning/maintenance/decommissioned）→ 原样保留
UPDATE devices d
SET lifecycle_state = CASE
    WHEN d.status = 'active'  THEN 'commissioned'
    WHEN d.status = 'offline' THEN
        CASE WHEN di.first_online_time IS NOT NULL
             THEN 'commissioned'
             ELSE 'registered'
        END
    ELSE d.status
END
FROM device_info di
WHERE di.device_id = d.id;

-- 没有 device_info 记录的设备兜底（罕见但要处理）
UPDATE devices SET lifecycle_state = CASE
    WHEN status = 'active'  THEN 'commissioned'
    WHEN status = 'offline' THEN 'registered'
    ELSE status
END WHERE NOT EXISTS (SELECT 1 FROM device_info di WHERE di.device_id = devices.id);

ALTER TABLE devices
    ADD CONSTRAINT chk_devices_lifecycle_state
    CHECK (lifecycle_state IN (
        'discovered', 'registered', 'provisioning',
        'commissioned', 'maintenance', 'decommissioned'
    ));

CREATE INDEX IF NOT EXISTS idx_devices_lifecycle_state
    ON devices (lifecycle_state);

-- ─── 2. 新增 is_online ──────────────────────────────────────────────
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS is_online BOOLEAN NOT NULL DEFAULT FALSE;

-- 回填：
--   status='active'   → is_online=true
--   status='offline'  → is_online=false
--   过渡态根据 last_inform_at 距今时长判定（90s 心跳窗口与 HeartbeatMonitor 默认值对齐）
UPDATE devices SET is_online = CASE
    WHEN status = 'active'  THEN TRUE
    WHEN status = 'offline' THEN FALSE
    WHEN last_inform_at IS NOT NULL
         AND last_inform_at > NOW() - INTERVAL '90 seconds' THEN TRUE
    ELSE FALSE
END;

-- 部分索引：dashboard / alert 大多查"在线"，部分索引比全表索引省空间
CREATE INDEX IF NOT EXISTS idx_devices_is_online
    ON devices (is_online) WHERE is_online = TRUE;

-- ─── 3. D1 硬切：DROP 老 status 列 + 相关索引 ──────────────────────
DROP INDEX IF EXISTS idx_devices_carrier_status;
DROP INDEX IF EXISTS idx_devices_status;
ALTER TABLE devices DROP COLUMN status;

-- 用新维度重建复合索引（替代被删的 idx_devices_carrier_status）
CREATE INDEX IF NOT EXISTS idx_devices_carrier_lifecycle
    ON devices (carrier, lifecycle_state);

-- ─── 4. 字段注释 ────────────────────────────────────────────────────
COMMENT ON COLUMN devices.lifecycle_state IS
    'T-0162: 设备业务生命周期，与在线状态解耦。取值见 chk_devices_lifecycle_state。';
COMMENT ON COLUMN devices.is_online IS
    'T-0162: 实时在线状态，由 HeartbeatMonitor / ACS Inform 维护。'
    'true=最近一次 inform 在心跳窗口（默认 90s）内。';


-- +goose Down
-- ============================================================
-- 反向：重建 status 列 + 删两个新列
-- ============================================================

ALTER TABLE devices
    ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active';

-- 反向回填：从 lifecycle_state + is_online 还原老 status 语义
--   commissioned + is_online=true    → active
--   commissioned + is_online=false   → offline
--   maintenance                      → maintenance
--   decommissioned                   → decommissioned
--   discovered/registered/provisioning → 同名
UPDATE devices SET status = CASE
    WHEN lifecycle_state = 'commissioned' AND is_online = TRUE  THEN 'active'
    WHEN lifecycle_state = 'commissioned' AND is_online = FALSE THEN 'offline'
    WHEN lifecycle_state = 'maintenance'                        THEN 'maintenance'
    WHEN lifecycle_state = 'decommissioned'                     THEN 'decommissioned'
    ELSE lifecycle_state
END;

CREATE INDEX IF NOT EXISTS idx_devices_carrier_status ON devices (carrier, status);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices (status);

DROP INDEX IF EXISTS idx_devices_carrier_lifecycle;
DROP INDEX IF EXISTS idx_devices_is_online;
DROP INDEX IF EXISTS idx_devices_lifecycle_state;
ALTER TABLE devices DROP CONSTRAINT IF EXISTS chk_devices_lifecycle_state;
ALTER TABLE devices
    DROP COLUMN IF EXISTS lifecycle_state,
    DROP COLUMN IF EXISTS is_online;
