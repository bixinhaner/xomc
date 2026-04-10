-- ============================================================
-- 000079_create_dictionary_tables.up.sql
-- 字典管理表：统一枚举值管理
-- ============================================================

-- 1. 字典主表
CREATE TABLE sys_dictionaries (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,                      -- 字典名称（中文）
    type        VARCHAR(255) NOT NULL,                      -- 字典类型（英文标识，全局唯一）
    status      BOOLEAN NOT NULL DEFAULT TRUE,              -- 启用状态
    description VARCHAR(255) NOT NULL DEFAULT '',           -- 描述
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ                                 -- 软删除时间
);

-- 部分唯一索引：仅对未软删除的记录强制 type 唯一
CREATE UNIQUE INDEX uniq_dict_type_active ON sys_dictionaries(type) WHERE deleted_at IS NULL;

CREATE INDEX idx_sys_dict_deleted_at ON sys_dictionaries(deleted_at) WHERE deleted_at IS NOT NULL;

COMMENT ON TABLE sys_dictionaries IS '字典主表：统一管理枚举值分类';
COMMENT ON COLUMN sys_dictionaries.type IS '字典类型（英文标识），全局唯一，如 gender、status';
COMMENT ON COLUMN sys_dictionaries.status IS '启用状态：true=启用，false=禁用';

-- 2. 字典详情表
CREATE TABLE sys_dictionary_details (
    id                 BIGSERIAL PRIMARY KEY,
    label              VARCHAR(255) NOT NULL,               -- 展示值
    value              VARCHAR(255) NOT NULL,               -- 字典值
    extend             VARCHAR(255) NOT NULL DEFAULT '',    -- 扩展值
    status             BOOLEAN NOT NULL DEFAULT TRUE,       -- 启用状态
    sort               INT NOT NULL DEFAULT 0,              -- 排序（越小越靠前）
    sys_dictionary_id  BIGINT NOT NULL REFERENCES sys_dictionaries(id) ON DELETE CASCADE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ
);

CREATE INDEX idx_sys_dict_detail_dict_id ON sys_dictionary_details(sys_dictionary_id);
CREATE INDEX idx_sys_dict_detail_deleted_at ON sys_dictionary_details(deleted_at) WHERE deleted_at IS NOT NULL;

COMMENT ON TABLE sys_dictionary_details IS '字典详情表：存储字典的具体选项值';
COMMENT ON COLUMN sys_dictionary_details.sort IS '排序标记，数值越小越靠前';

-- 3. 初始化数据：6 个基础字典
INSERT INTO sys_dictionaries (name, type, status, description) VALUES
('性别', 'gender', TRUE, '用户性别'),
('数据库int类型', 'int', TRUE, '整型映射'),
('时间日期类型', 'time.Time', TRUE, '时间类型映射'),
('浮点型', 'float64', TRUE, '浮点类型映射'),
('字符串', 'string', TRUE, '字符串类型映射'),
('布尔类型', 'bool', TRUE, '布尔类型映射');

-- 性别字典项
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('男', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'gender')),
('女', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'gender'));

-- int 类型字典项
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('int', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int8', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int16', '3', 3, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int32', '4', 4, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int64', '5', 5, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('rune', '6', 6, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint', '7', 7, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint8', '8', 8, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint16', '9', 9, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint32', '10', 10, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint64', '11', 11, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uintptr', '12', 12, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('byte', '13', 13, (SELECT id FROM sys_dictionaries WHERE type = 'int'));

-- time.Time 类型字典项
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('time.Time', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'time.Time'));

-- float64 类型字典项
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('float32', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'float64')),
('float64', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'float64'));

-- string 类型字典项
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('string', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'string'));

-- bool 类型字典项
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('bool', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'bool'));
