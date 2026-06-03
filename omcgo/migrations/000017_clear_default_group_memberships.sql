-- +goose Up
-- 2026-06-03 用户决策「未分组设备 = 未绑定任何分组的设备」。
-- 历史上:设备注册无预登记时被自动归入默认 L2 组(DefaultLevel2GroupID),删组时成员也回退该组。
-- 改造后:"未分组"由 NOT EXISTS device_group_members 判定,默认 L2 组不再持有成员。
-- 故一次性清除所有指向默认 L2 组的成员关系,让这些设备回到真正"未分组"状态,
-- 与新的查询/注册/删组语义对齐(否则这些行既不在 NOT EXISTS 集合、默认组又变虚拟,两边都不显示)。
DELETE FROM device_group_members
 WHERE group_id = '00000000-0000-0000-0000-000000000002';

-- +goose Down
-- 不可逆:被删除的成员关系是历史自动归组派生数据,无法还原。no-op 占位。
SELECT 1;
