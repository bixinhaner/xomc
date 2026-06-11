-- +goose Up
-- 配置恢复主动完整性校验（issue #70，#61 finding 2 拆分）。
--
-- 背景：恢复"成功"此前只看 TransferComplete 的 FaultCode==0（CPE 仅回报"下载完成"），
-- 既不校验下发文件指纹、也不回读设备实际生效配置 —— DR 演练里设备可能"下载成功但
-- 配置没真正应用 / 应用了错误版本"而被记为成功。本迁移补三件事的存储面：
--   1. status 增加 'downloaded' 中间态：区分"下载成功"与"校验通过/完成"。
--      pending → running → downloaded → (completed | failed)。
--   2. expected_hash / hash_algo：下发 Download 时记录期望明文指纹（基线取
--      config_snapshots.md5，#61 后该列已是明文指纹语义）。
--   3. verified_hash / verification_method / verified_at：主动校验结果留痕。
--   4. downloaded_at：进入 downloaded 态的时刻（与 completed_at 区分）。
--   5. source_version：快照/备份来源的设备软件版本，供跨版本恢复对比与审计。
-- 全部新增列可空（或带 DEFAULT），不破坏存量行、不被 seed 引用，幂等可重复执行。

-- +goose StatementBegin
ALTER TABLE restore_tasks ADD COLUMN IF NOT EXISTS expected_hash       varchar(128);
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks ADD COLUMN IF NOT EXISTS hash_algo           varchar(16);
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks ADD COLUMN IF NOT EXISTS verified_hash       varchar(128);
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks ADD COLUMN IF NOT EXISTS verification_method varchar(24);
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks ADD COLUMN IF NOT EXISTS verified_at         timestamp with time zone;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks ADD COLUMN IF NOT EXISTS downloaded_at       timestamp with time zone;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks ADD COLUMN IF NOT EXISTS source_version      varchar(64);
-- +goose StatementEnd

-- config_snapshots 增加 source_version：记录快照捕获时的设备软件版本，供按快照
-- 恢复时与目标设备当前固件版本做跨版本检查。可空，由 promote / import 写入路径回填；
-- 存量行留空（未知）→ 跨版本检查对空源版本放行不告警。
-- +goose StatementBegin
ALTER TABLE config_snapshots ADD COLUMN IF NOT EXISTS source_version varchar(64);
-- +goose StatementEnd

-- status CHECK 加入 'downloaded'（status 列为 varchar(16)，'downloaded' 10 字符放得下）。
-- 先 DROP 旧约束再按新枚举重建；DROP IF EXISTS + 名称判定保证幂等。
-- +goose StatementBegin
ALTER TABLE restore_tasks DROP CONSTRAINT IF EXISTS restore_tasks_status_check;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'restore_tasks_status_check') THEN
        ALTER TABLE restore_tasks
            ADD CONSTRAINT restore_tasks_status_check
            CHECK (status::text = ANY (ARRAY[
                'pending'::text, 'running'::text, 'downloaded'::text,
                'completed'::text, 'failed'::text, 'cancelled'::text]));
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- 回退：先把任何 'downloaded' 行收敛为 'running'（否则收紧 CHECK 会失败），
-- 再恢复旧 5 值约束并删除新增列。
-- +goose StatementBegin
UPDATE restore_tasks SET status = 'running' WHERE status = 'downloaded';
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks DROP CONSTRAINT IF EXISTS restore_tasks_status_check;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'restore_tasks_status_check') THEN
        ALTER TABLE restore_tasks
            ADD CONSTRAINT restore_tasks_status_check
            CHECK (status::text = ANY (ARRAY[
                'pending'::text, 'running'::text,
                'completed'::text, 'failed'::text, 'cancelled'::text]));
    END IF;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE config_snapshots DROP COLUMN IF EXISTS source_version;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks DROP COLUMN IF EXISTS source_version;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks DROP COLUMN IF EXISTS downloaded_at;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks DROP COLUMN IF EXISTS verified_at;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks DROP COLUMN IF EXISTS verification_method;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks DROP COLUMN IF EXISTS verified_hash;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks DROP COLUMN IF EXISTS hash_algo;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE restore_tasks DROP COLUMN IF EXISTS expected_hash;
-- +goose StatementEnd
