-- ============================================================
-- 000001_extensions_functions.up.sql
-- PostgreSQL 扩展和公共函数
-- ============================================================

-- TimescaleDB 扩展（可选，不存在则跳过）
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        CREATE EXTENSION IF NOT EXISTS timescaledb;
    END IF;
EXCEPTION WHEN others THEN
    RAISE NOTICE 'TimescaleDB extension not available, time-series features will be disabled';
END $$;

-- 通用 updated_at 自动更新触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
