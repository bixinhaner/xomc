-- +goose Up
-- issue #223 收尾：制式字典文案改无空格版（eNB(LTE)/gNB(NR)）。
-- 设备列表 / 回收站 / 「请选择基站制式」筛选下拉三处均读 network_type 字典 label，
-- 改这一处 label 即三处同步生效。network_type 字典 sys_dictionary_id=15，lte/nr 两项，
-- 原 seed（000001）label 为带空格的 'eNB (LTE)' / 'gNB (NR)'。
-- 带 label 旧值守卫，重复执行幂等、且不误改已改过的行。
UPDATE public.sys_dictionary_details SET label = 'eNB(LTE)', updated_at = now()
    WHERE sys_dictionary_id = 15 AND value = 'lte' AND label = 'eNB (LTE)';
UPDATE public.sys_dictionary_details SET label = 'gNB(NR)', updated_at = now()
    WHERE sys_dictionary_id = 15 AND value = 'nr' AND label = 'gNB (NR)';

-- +goose Down
-- 回滚为带空格旧 label（与 seed/000001 一致）。
UPDATE public.sys_dictionary_details SET label = 'eNB (LTE)', updated_at = now()
    WHERE sys_dictionary_id = 15 AND value = 'lte' AND label = 'eNB(LTE)';
UPDATE public.sys_dictionary_details SET label = 'gNB (NR)', updated_at = now()
    WHERE sys_dictionary_id = 15 AND value = 'nr' AND label = 'gNB(NR)';
