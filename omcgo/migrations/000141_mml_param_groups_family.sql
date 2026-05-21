-- +goose Up
-- ============================================================
-- 000141 — mml_param_groups 加 family_code / family_name_zh 持久化列
--
-- 背景（设计文档 §15.6 + §15.8 P3）：
--   v2.4 D32+D38 path-prefix-family 推断当前由 internal/mml/family.go InferFamily
--   在 BuildTree 运行时后处理执行（attachFamily 递归遍历）。
--
--   P3 把 family 持久化到 DB：
--     1) catalogloader upsertGroup 在 INSERT / UPDATE 时同步写入 family
--     2) BuildTree SELECT 直接读 family_code / family_name_zh
--     3) attachFamily 退化为防御性 fallback —— 仅当 DB 返回空字符串时调用 InferFamily
--
-- 影响：
--   - 列加 DEFAULT '' NOT NULL（空字符串与 Go 端 "self-family" 语义一致）
--   - 部分索引 WHERE family_code <> '' AND deleted_at IS NULL，避免索引大量空值
--   - 历史回填用 CASE WHEN 镜像 InferFamily 9 条规则（顺序敏感，与 family.go 完全一致）
-- ============================================================

ALTER TABLE mml_param_groups
    ADD COLUMN IF NOT EXISTS family_code VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS family_name_zh VARCHAR(128) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_mml_param_groups_family_code
    ON mml_param_groups(family_code)
    WHERE family_code <> '' AND deleted_at IS NULL;

-- 历史数据回填：用 CASE WHEN 镜像 InferFamily 的 9 条规则（顺序敏感）。
-- 规则顺序对应 internal/mml/family.go familyRules 切片：
--   1. hardware_upgrade  — MU.*SwUpgrade*（必须先于 hardware_units / device_info）
--   2. hardware_units    — Device.DeviceInfo.MU.*
--   3. device_info       — Device.DeviceInfo.*
--   4. alarm_instances   — Device.FaultMgmt.*
--   5. capabilities      — *.Capabilities.* (作为路径分量)
--   6. sctp              — Device.Services.FAPControl.Transport.SCTP*
--   7. measure_ctrl      — *MeasureCtrl*
--   8. ethernet_ip       — Device.Ethernet.*
--   9. idle_mode         — *IdleMode*
UPDATE mml_param_groups SET
    family_code = CASE
        WHEN object_path_template LIKE 'Device.DeviceInfo.MU.%SwUpgrade%' THEN 'hardware_upgrade'
        WHEN object_path_template LIKE 'Device.DeviceInfo.MU.%'           THEN 'hardware_units'
        WHEN object_path_template LIKE 'Device.DeviceInfo.%'              THEN 'device_info'
        WHEN object_path_template LIKE 'Device.FaultMgmt.%'               THEN 'alarm_instances'
        WHEN object_path_template LIKE '%.Capabilities.%'                 THEN 'capabilities'
        WHEN object_path_template LIKE 'Device.Services.FAPControl.Transport.SCTP%' THEN 'sctp'
        WHEN object_path_template LIKE '%MeasureCtrl%'                    THEN 'measure_ctrl'
        WHEN object_path_template LIKE 'Device.Ethernet.%'                THEN 'ethernet_ip'
        WHEN object_path_template LIKE '%IdleMode%'                       THEN 'idle_mode'
        ELSE ''
    END,
    family_name_zh = CASE
        WHEN object_path_template LIKE 'Device.DeviceInfo.MU.%SwUpgrade%' THEN '硬件升级'
        WHEN object_path_template LIKE 'Device.DeviceInfo.MU.%'           THEN '硬件单元'
        WHEN object_path_template LIKE 'Device.DeviceInfo.%'              THEN '设备信息'
        WHEN object_path_template LIKE 'Device.FaultMgmt.%'               THEN '告警实例'
        WHEN object_path_template LIKE '%.Capabilities.%'                 THEN '能力集'
        WHEN object_path_template LIKE 'Device.Services.FAPControl.Transport.SCTP%' THEN 'SCTP'
        WHEN object_path_template LIKE '%MeasureCtrl%'                    THEN '测量控制'
        WHEN object_path_template LIKE 'Device.Ethernet.%'                THEN '以太网/IP'
        WHEN object_path_template LIKE '%IdleMode%'                       THEN '空闲态移动性'
        ELSE ''
    END
WHERE (family_code IS NULL OR family_code = '')
  AND object_path_template IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_mml_param_groups_family_code;
ALTER TABLE mml_param_groups
    DROP COLUMN IF EXISTS family_name_zh,
    DROP COLUMN IF EXISTS family_code;
