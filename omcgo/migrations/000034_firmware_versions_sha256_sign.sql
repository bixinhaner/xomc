-- +goose Up
-- 固件完整性校验 MD5 → SHA-256 + 厂商签名验证脚手架（issue #8）。
--
-- 背景：firmware_versions 历史仅有 md5_val 做完整性校验。MD5 已被证明可碰撞，
-- 不能作为商用网管系统的固件完整性根。本迁移以"加列不动旧列"的增量方式引入
-- SHA-256，并预埋厂商签名验证所需的列：
--   · sha256_val     — 上传时计算并存储；下发前优先用它做完整性校验（存量旧行为 NULL
--                       → 校验时回退 md5_val，保证不阻断旧固件下发）。
--   · signature      — 厂商对固件的数字签名（base64）。可空；为空 = 无签名固件，
--                       签名校验跳过（向后兼容现网未签名固件，不破坏 happy-path）。
--   · signature_alg  — 签名算法标识（如 'rsa-pss-sha256' / 'ecdsa-p256-sha256'），
--                       仅在 signature 非空时有意义。
--   · public_key_id  — 验签所用厂商公钥的标识符（key id / 指纹），便于密钥轮换与吊销
--                       时定位对应公钥。仅在 signature 非空时有意义。
--
-- 所有列均可空、无默认值约束变更，存量行不需要 backfill；新上传走 SHA-256 + 可选签名。
-- 完整的厂商 PKI / 密钥轮换 / 吊销基础设施是独立子系统，本迁移只落"列 + 校验 hook"。
-- +goose StatementBegin
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS sha256_val character varying(64);
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS signature text;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS signature_alg character varying(32);
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS public_key_id character varying(128);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS public_key_id;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS signature_alg;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS signature;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS sha256_val;
-- +goose StatementEnd
