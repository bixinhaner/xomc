-- +goose Up
-- 启用 pgcrypto 扩展。
--
-- 用途:
--   1. internal/admin/apikey 等模块自动生成 API Key 时需要 gen_random_bytes()
--      / encode() 做随机 token 生成与 hash。
--   2. diag_mml_gpn_probe.sh 诊断脚本走自动 API Key 生成路径时依赖此扩展。
--   3. 后续任何需要密码 hash / 安全随机的 SQL 路径。
--
-- 历史背景：早期部署依赖 DBA 手动 CREATE EXTENSION，新部署常踩坑（诊断脚本
-- 报 "pgcrypto 扩展未启用"）。本 migration 把启用动作纳入版本化流程，让所有
-- 新部署 / 重建数据库自动具备 pgcrypto。
--
-- 幂等：CREATE EXTENSION IF NOT EXISTS 不会报错，已启用环境无副作用。
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- +goose Down
-- 不下线 pgcrypto。理由：
--   - 多个上层功能（API Key 生成 / 密码 hash / 未来扩展）依赖它
--   - DROP EXTENSION pgcrypto 会级联删除使用它的列（如 hashed 字段），破坏性大
--   - 即使运行 down，扩展卸载后再 up 必须重新创建，没有"回滚"实际价值
-- 如确需删除，DBA 手工 DROP EXTENSION pgcrypto CASCADE。
SELECT 1;
