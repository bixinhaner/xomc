-- +goose Up
-- ISSUE-478：清理「设备被移动/添加到『未分组设备』内置节点(...0002)」产生的脏数据。
--
-- 根因（见 T1）：写入侧曾把设备显式写进内置虚拟节点 DefaultLevel2GroupID
-- (00000000-0000-0000-0000-000000000002)，与读取侧「未分组设备 = NOT EXISTS
-- device_group_members」口径矛盾，导致中招设备在分组页两头落空、彻底消失。
--
-- T1 已把写入语义改为「移出分组（删全部归属记录）」，杜绝再产生此类记录；本迁移
-- 一次性清掉现存的存量脏记录，使已中招设备恢复正常显示（回到「未分组设备」节点）。
--
-- 与 seed/000001_init_seed.sql 配套：000001 只在全新建库生效，修不到已部署库（含
-- dev 库），故本迁移对现存数据做 DELETE。
--
-- 幂等：WHERE group_id = ...0002，仅删指向内置节点的归属记录；重复执行第二次起
-- WHERE 不命中、零行删除。只清 ...0002 这一类，不动任何指向真实分组的归属记录。
-- +goose StatementBegin
DELETE FROM public.device_group_members
 WHERE group_id = '00000000-0000-0000-0000-000000000002';
-- +goose StatementEnd

-- +goose Down
-- 不可逆：删除的是「设备指向未分组内置节点」的脏归属记录，原本就是 bug 产物，
-- 无从（也不应）恢复——恢复即重新引入与读取侧口径矛盾的脏数据。Down 段为 no-op。
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
