-- +goose Up
-- issue #203：设备离线判定阈值接 sys_configs 实时配置。
-- 复用通用键值表 sys_configs (category='device') 存两类阈值，键名与前端
-- 「系统配置 → 设备设置」面板表单字段名一致（DeviceSettings.tsx）：
--   - enbTimeout：基站类（覆盖 2G/4G/5G）无心跳判离线阈值（秒），默认 100
--   - cpeTimeout：CPE 类无心跳判离线阈值（秒），默认 600
-- 对账器（DeviceStatusReconciler）每轮扫描读这两个键的最新值，按设备类应用
-- 各自阈值；改配置后下一轮扫描即生效，无需重启。
--
-- 幂等：ON CONFLICT (category, key) DO NOTHING —— 已有值（如用户在 UI 改过）
-- 不被种子覆盖；仅在缺失时补默认值。
INSERT INTO sys_configs (category, key, value, value_type, description, is_public)
VALUES
    ('device', 'enbTimeout', '100', 'int', '基站类无心跳判离线阈值（秒）', false),
    ('device', 'cpeTimeout', '600', 'int', 'CPE 类无心跳判离线阈值（秒）', false)
ON CONFLICT (category, key) DO NOTHING;

-- +goose Down
DELETE FROM sys_configs
WHERE category = 'device' AND key IN ('enbTimeout', 'cpeTimeout');
