-- ============================================================
-- 000081_create_sys_logs.up.sql
-- 系统日志表：登录日志、操作日志、任务日志
-- ============================================================

-- 1. 登录日志
CREATE TABLE sys_login_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    username    VARCHAR(64) NOT NULL,
    ip_address  VARCHAR(64) NOT NULL DEFAULT '',
    location    VARCHAR(128) NOT NULL DEFAULT '',     -- 登录地点（可选）
    browser     VARCHAR(128) NOT NULL DEFAULT '',
    os          VARCHAR(128) NOT NULL DEFAULT '',
    status      BOOLEAN NOT NULL DEFAULT TRUE,         -- true=成功, false=失败
    message     VARCHAR(255) NOT NULL DEFAULT '',      -- 失败原因
    login_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_logs_user_id ON sys_login_logs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_login_logs_login_at ON sys_login_logs(login_at);

COMMENT ON TABLE sys_login_logs IS '登录日志：记录用户登录/登出事件';

-- 2. 操作日志
CREATE TABLE sys_oper_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    username    VARCHAR(64) NOT NULL DEFAULT '',
    action      VARCHAR(64) NOT NULL,                  -- 操作类型：create/update/delete/export 等
    module      VARCHAR(64) NOT NULL DEFAULT '',       -- 模块：device/alarm/user/role 等
    target      VARCHAR(255) NOT NULL DEFAULT '',      -- 操作对象
    detail      TEXT NOT NULL DEFAULT '',               -- 详细描述
    ip_address  VARCHAR(64) NOT NULL DEFAULT '',
    user_agent  VARCHAR(512) NOT NULL DEFAULT '',
    status      BOOLEAN NOT NULL DEFAULT TRUE,          -- true=成功, false=失败
    error_msg   VARCHAR(512) NOT NULL DEFAULT '',
    cost_ms     INT NOT NULL DEFAULT 0,                 -- 耗时（毫秒）
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oper_logs_user_id ON sys_oper_logs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_oper_logs_action ON sys_oper_logs(action);
CREATE INDEX idx_oper_logs_created_at ON sys_oper_logs(created_at);
CREATE INDEX idx_oper_logs_module ON sys_oper_logs(module);

COMMENT ON TABLE sys_oper_logs IS '操作日志：记录用户关键操作';

-- 3. 任务日志
CREATE TABLE sys_task_logs (
    id          BIGSERIAL PRIMARY KEY,
    task_type   VARCHAR(64) NOT NULL,                  -- 任务类型：firmware/backup/pm_export 等
    task_id     VARCHAR(128) NOT NULL DEFAULT '',      -- 任务ID（关联具体任务表）
    status      VARCHAR(16) NOT NULL DEFAULT 'running', -- running/success/failed
    operator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    operator    VARCHAR(64) NOT NULL DEFAULT '',
    target      VARCHAR(255) NOT NULL DEFAULT '',      -- 操作对象
    detail      TEXT NOT NULL DEFAULT '',
    error_msg   TEXT NOT NULL DEFAULT '',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    cost_ms     INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_task_logs_task_type ON sys_task_logs(task_type);
CREATE INDEX idx_task_logs_operator_id ON sys_task_logs(operator_id) WHERE operator_id IS NOT NULL;
CREATE INDEX idx_task_logs_status ON sys_task_logs(status);
CREATE INDEX idx_task_logs_started_at ON sys_task_logs(started_at);

COMMENT ON TABLE sys_task_logs IS '任务日志：记录后台任务执行结果';
