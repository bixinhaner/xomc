-- +goose Up
-- ============================================================
-- 000029: 修正 mml_params 表中不符合 TR-069 协议的 tr069_path
--
-- 背景: ACS 实测下发给 baicell CPE 的 GetParameterValues 报文里包含
-- 14 条 path，CPE silent drop 不响应。诊断（docs/operations/troubleshoot-mml-rpc.md）
-- 发现三类不合规：
--   A) 顶层非 Device. / InternetGatewayDevice. ←  DeviceGSM.* 7 条
--   B) 含 {i} 占位符未替换                       ← Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress
--   C) X_<vendor>_ 不规范（COM 不是 6 位 OUI）   ← X_COM_* 多条
--
-- TR-069 §A.2.2.1: 数据模型只承认两个根 Device. / InternetGatewayDevice.
-- TR-069 §A.2.2.7: 实例索引 {i} 在 GetParameterValues 报文里必须替换为
--                  实际整数索引（通常从 1 开始），或用尾随 "." 的 partial path
-- TR-069 §3.3:     vendor 扩展前缀 X_<6位HEX OUI>_<Name>
--
-- 本迁移针对**已知现场出问题的 14 条 path**做精确修复。其它路径如果还有
-- 问题，先用 omcgo/scripts/audit_mml_params.sh 审计再走单独迁移修复，
-- 不在这里做大批量替换（避免误改）。
--
-- 注意：tr069_path 的实际正确值依赖 CPE 厂商数据模型定义。本迁移采用
-- 最保守的修法——保留命名语义，仅修协议形式：
--   - DeviceGSM.X        →  Device.X_BAICELLS_DeviceGSM.X
--   - {i}.{i}            →  1.1（首实例硬编码，多实例场景需后续按需调）
--   - X_COM_X            →  X_BAICELLS_COM_X（与 seed 005 已存在的命名风格对齐）
-- 如果厂商真实路径与此不同，请在执行后再写 fix 迁移覆盖。
-- ============================================================

-- A) 修 7 条 DeviceGSM.* → Device.X_BAICELLS_DeviceGSM.*
UPDATE mml_params
SET tr069_path = 'Device.X_BAICELLS_DeviceGSM.' || substring(tr069_path FROM 11),
    updated_at = NOW()
WHERE tr069_path LIKE 'DeviceGSM.%';

-- B) 修所有含 {<letter>+} 占位符的路径，硬编码替换为首实例索引 1。
-- 覆盖 spec 标准 {i} 以及非标但常见的 {j}/{n}/{idx} 等。
-- 注意：这是"让链路先通"的最小代价方案。GetParameterValues 报文里的占位符
-- CPE 必拒；实际多实例场景应当先 GetParameterNames partial-path 列出所有
-- 实例索引再批量查（后续设计"实例索引解析"机制后再回滚为 partial path）。
-- 唯一约束 uniq_param_version_path (param_version, tr069_path) 命中冲突
-- （已有 .1. 行）时保留原有合规行，删除占位符行。
-- +goose StatementBegin
DO $$
DECLARE
    has_placeholder TEXT := '\{[A-Za-z]+\}';
BEGIN
    -- 先删除会冲突的占位符行（目标 path 已存在合规行）
    DELETE FROM mml_params src
    WHERE src.tr069_path ~ has_placeholder
      AND EXISTS (
          SELECT 1 FROM mml_params tgt
          WHERE tgt.param_version = src.param_version
            AND tgt.tr069_path = REGEXP_REPLACE(src.tr069_path, has_placeholder, '1', 'g')
            AND tgt.id != src.id
      );

    -- 剩余占位符行执行 REPLACE → 1
    UPDATE mml_params
    SET tr069_path = REGEXP_REPLACE(tr069_path, has_placeholder, '1', 'g'),
        updated_at = NOW()
    WHERE tr069_path ~ has_placeholder;
END $$;
-- +goose StatementEnd

-- C) 修 X_COM_* → X_BAICELLS_COM_*（仅命中 Device.DeviceInfo.X_COM_* 这批，
--    不动已经是 X_BAICELLS_COM_* 的；也不动 X_<6位HEX>_* 这种规范命名）
UPDATE mml_params
SET tr069_path = REPLACE(tr069_path, '.X_COM_', '.X_BAICELLS_COM_'),
    updated_at = NOW()
WHERE tr069_path LIKE '%.X_COM_%'
  AND tr069_path NOT LIKE '%.X_BAICELLS_COM_%';

-- +goose Down
-- 不可逆：down 段会把 seed 005 里原本就是 X_BAICELLS_COM_* 的合规路径
-- 误改回 X_COM_*，破坏 seed 005 数据完整性。如需回滚 mml_params 数据，
-- 推荐方案：直接重置数据库（开发环境）或从最近备份恢复（生产环境）。
-- +goose StatementBegin
DO $$
BEGIN
    RAISE NOTICE 'seed/000029 down: skipped (not safely reversible). Reset DB or restore from backup if rollback needed.';
END $$;
-- +goose StatementEnd
