-- +goose Up
-- qa-614 c6 / #365 #373：补 2G(GSM) 升级内置任务类型。
-- 前提：2G/GSM 属于本产品 V0.5.0 支持范围（测试用真实 BSC/GSM 站，products.xml
-- 有 tech="2G" 的 BSC(^FAP/PGSM$)/BTS(^FAP/BTS$) 产品）。
-- 镜像下载链路本身与制式无关，复用与 4G/5G 同一条 Download 链路。
-- 与后端 ufte.builtInTaskTypes() 的 GSM_IMG_UPGRADE 定义一致；运行期
-- EnsureBuiltInTaskTypes 也会 upsert 同条目，本 seed 仅保证全新部署 baseline 有此行。
-- ON CONFLICT DO NOTHING 保证幂等（type_code 主键已存在则跳过）。
ALTER TABLE public.ufte_task_types DISABLE TRIGGER ALL;

INSERT INTO public.ufte_task_types VALUES
	('GSM_IMG_UPGRADE', 'gsm_upgrade', '2G升级', '2G 基站软件升级', '复用现网软件升级链路，统一承载 2G(GSM) 基站镜像升级任务。', 'DOWNLOAD', true, true, '["CHECK_PERMISSION", "CHECK_ONLINE", "CHECK_CONFLICT", "SEND_RPC", "WAIT_RPC_RESPONSE", "WAIT_FILE_TRANSFER", "WAIT_TRANSFER_COMPLETE"]', '', 'CODE_GSM_UPGRADE_IMAGE', '["2G BSC", "2G BTS", "BSC", "BTS", "PGSM"]', '1 Firmware Upgrade Image', '1 Firmware Upgrade Image', true, 'firmware/{minio_path}', '{firmware_name}', '{firmware_name}', 'firmware.fileSize', 'firmware.md5', 'false', 0, '/smallcell/FileDownloadService/firmware/img/{path}', 'system', now(), now(), NULL, 19)
	ON CONFLICT DO NOTHING;

ALTER TABLE public.ufte_task_types ENABLE TRIGGER ALL;

-- +goose Down
DELETE FROM public.ufte_task_types WHERE type_code = 'GSM_IMG_UPGRADE';
