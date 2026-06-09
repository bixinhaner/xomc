-- +goose Up
-- 删除 12 条「参数 PATH 为空」的 LST/MOD 命令 + 删后无任何命令的分组。
--
-- 背景：这 12 条 LST/MOD 命令声明的 standardPath 均未进 standard_params 字典
--       (NR 异系统邻区 + 扩展型一体化皮基站硬件盘点/软件升级)，致 mml_command_sub_fields
--       无法建立外键 → console-v2 选中后参数 PATH 恒为空、三个执行按钮置灰不可用。
--       规范 PATH 对照留档：docs/design/mml-empty-path-commands-spec-mapping-20260609.md。
-- sub_fields 经 FK ON DELETE CASCADE 连带（这 12 条本就 0 个 sub_field）。
-- 幂等：删除后重跑命中 0 行，安全；fresh 库(seed/000152 含原始数据)与存量库都会收敛。
DELETE FROM mml_commands
WHERE command_code IN (
    'LST RU_SW_UPGRADE',
    'LST RU',
    'LST RF_CHANNEL',
    'LST EU',
    'LST EU_SW_UPGRADE',
    'LST SLOT',
    'MOD RU',
    'MOD RF_CHANNEL',
    'MOD EU',
    'MOD SLOT',
    'LST INTER_RAT_CELL_NR',
    'MOD INTER_RAT_CELL_NR'
);

-- 删除 4 个 source='extension'、从未关联任何命令的空扩展桶（前端树里显示但点开无命令）。
-- 按 group_code 显式删除 → 所有环境确定性一致；并加 NOT EXISTS 守护：仅当该分组确无任何命令
-- 时才删（防止未来某环境给它挂了命令时被误删）。这 4 个在 init_seed 即建、各环境均为空。
-- 说明：本次只删这 12 条命令，chapter:SR/SI 两分组删后仍各剩 6/14 条命令、非空、不在此列。
DELETE FROM mml_command_groups g
WHERE g.group_code IN (
    'chapter:SX_BOARDCONF_EXT',
    'chapter:SX_DEVICE_EXT',
    'chapter:SX_DEVICEGSM_EXT',
    'chapter:SX_INTERNETGATEWAYDEVICE_EXT'
)
  AND NOT EXISTS (SELECT 1 FROM mml_commands c WHERE c.group_id = g.id);

-- +goose Down
-- +goose StatementBegin
-- 删除不可逆(命令/分组已移除)；down 无操作。如需恢复请重跑 seed/000152 等源种子或从 catalog 重新导入。
SELECT 1;
-- +goose StatementEnd
