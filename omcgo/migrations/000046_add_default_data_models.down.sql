-- ============================================================================
-- 000046: 移除默认数据模型定义 (回滚)
-- ============================================================================

DELETE FROM data_model_definitions WHERE id = '30000000-0001-4000-8000-000000000001';
DELETE FROM data_model_definitions WHERE id = '30000000-0001-4000-8000-000000000002';
DELETE FROM data_model_definitions WHERE id = '30000000-0001-4000-8000-000000000003';
DELETE FROM data_model_definitions WHERE id = '30000000-0001-4000-8000-000000000004';
DELETE FROM data_model_definitions WHERE id = '30000000-0001-4000-8000-000000000010';
DELETE FROM data_model_definitions WHERE id = '30000000-0001-4000-8000-000000000020';
