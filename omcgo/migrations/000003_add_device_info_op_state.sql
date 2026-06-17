-- +goose Up
-- 设备激活状态升级为派生列,语义改为"基站当前是否在运营"。
--
-- 新口径(取代历史 model.DeriveOpStateActivated(first_online_time)):
--   op_state = CalcCellStatus(params) == "normal" ? "1" : "0"
-- 即遍历该设备所有 FAPService.{i} 的 OpState/CellOpState trpath,任一 cell active
-- 即设备激活,否则未激活。设备级 OpState 与 cell_status 严格同源派生。
--
-- 与 first_online_time 派生口径的区别:
--   - 旧口径:激活是一次性单调持久事实(曾上线即激活,永不掉)
--   - 新口径:激活反映网元上报的实时运营态(基站全部小区下电/未启用即未激活)
-- 选择新口径是因为用户对"激活状态"的预期就是网元真实运营态,而非历史曾经露过头。
--
-- DEFAULT '0' 保证新加列对老行不报 NULL;首次 InfoSyncer 跑后立即被覆盖为真值。
-- 旧 model.DeriveOpStateActivated 函数保留(其它路径还在用),但 device_info_pg_repository
-- list 查询路径已切换到读 di.op_state。

ALTER TABLE public.device_info
    ADD COLUMN IF NOT EXISTS op_state character varying(8) DEFAULT '0';

COMMENT ON COLUMN public.device_info.op_state IS '激活状态:1=激活/0=未激活;由 InfoSyncer.CalcOpState 派生(等价 cell_status: 任一 cell active → "1")。与 first_online_time 派生口径解耦。';

-- 设备列表"激活状态"过滤是常用项,加索引(同 cell_status)。
CREATE INDEX IF NOT EXISTS idx_device_info_op_state ON public.device_info USING btree (op_state);

-- +goose Down
DROP INDEX IF EXISTS public.idx_device_info_op_state;
ALTER TABLE public.device_info DROP COLUMN IF EXISTS op_state;
