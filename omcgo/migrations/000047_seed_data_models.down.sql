-- ============================================================================
-- 000047: 回滚初始化数据模型定义
--    删除由 up 迁移插入的种子数据
-- ============================================================================

DELETE FROM data_model_definitions WHERE id = '30000047-0001-4000-8000-000000000001';
DELETE FROM data_model_definitions WHERE id = '30000047-0001-4000-8000-000000000002';
DELETE FROM data_model_definitions WHERE id = '30000047-0001-4000-8000-000000000003';
DELETE FROM data_model_definitions WHERE id = '30000047-0001-4000-8000-000000000004';
DELETE FROM data_model_definitions WHERE id = '30000047-0002-4000-8000-000000000001';
DELETE FROM data_model_definitions WHERE id = '30000047-0003-4000-8000-000000000001';
