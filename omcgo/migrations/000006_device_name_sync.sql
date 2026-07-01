-- +goose Up
-- +goose StatementBegin

-- Issue #758: 设备名称同步功能
--
-- 背景：基站在 LMT 上设置的名字与网管存的设备名称可能不一致。
-- 用户打开 nameSettingEnable 开关后，系统应自动检测并按配置方向同步。
--
-- 新增字段：
--   - name_sync_pending: 布尔，标记是否需要人工处理（前端小红点）
--   - lmt_device_name: 缓存从 LMT 读到的设备名称，供前端对比展示

ALTER TABLE public.device_info
    ADD COLUMN IF NOT EXISTS name_sync_pending boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS lmt_device_name varchar(255);

COMMENT ON COLUMN public.device_info.name_sync_pending IS
    '设备名称同步待处理标记（true=需人工确认，前端显示小红点）。Path B 同步检测到 LMT 名称与网管不一致且 prompt=true 时置 true；用户确认或下次同步名称一致时自动清 false。';

COMMENT ON COLUMN public.device_info.lmt_device_name IS
    '从 LMT 读取的设备名称（HNBName / gNBName）缓存。供前端在名称冲突时对比展示"LMT 名称"与"网管名称"。';

-- 查询待处理设备的索引（设备列表页筛选"有名称冲突"）
CREATE INDEX IF NOT EXISTS idx_device_info_name_sync_pending
    ON public.device_info (device_id)
    WHERE name_sync_pending = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_device_info_name_sync_pending;

ALTER TABLE public.device_info
    DROP COLUMN IF EXISTS lmt_device_name,
    DROP COLUMN IF EXISTS name_sync_pending;

-- +goose StatementEnd
