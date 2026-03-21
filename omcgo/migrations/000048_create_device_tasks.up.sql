-- 设备任务表：用于管理 ACS 与 CPE 之间的 RPC 任务
-- 支持任务状态跟踪、历史记录、重试机制

CREATE TABLE device_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64) NOT NULL,
    method          VARCHAR(64) NOT NULL,
    params          JSONB,
    priority        INTEGER DEFAULT 10,
    command_key     VARCHAR(128),
    cwmp_id         VARCHAR(256),

    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    retry_count     INTEGER DEFAULT 0,
    max_retries     INTEGER DEFAULT 3,

    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMP WITH TIME ZONE,
    completed_at    TIMESTAMP WITH TIME ZONE,
    expires_at      TIMESTAMP WITH TIME ZONE,

    result          JSONB,
    error_code      INTEGER,
    error_message   TEXT,

    source          VARCHAR(32) DEFAULT 'api',
    creator_id      VARCHAR(64),
    description     TEXT,

    CONSTRAINT fk_device FOREIGN KEY (device_sn) REFERENCES devices(serial_number) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_device_tasks_device_sn ON device_tasks(device_sn);
CREATE INDEX idx_device_tasks_status ON device_tasks(status);
CREATE INDEX idx_device_tasks_created_at ON device_tasks(created_at);
CREATE INDEX idx_device_tasks_cwmp_id ON device_tasks(cwmp_id);
CREATE INDEX idx_device_tasks_pending ON device_tasks(device_sn, status, priority, created_at)
    WHERE status = 'pending';

COMMENT ON TABLE device_tasks IS '设备任务表：管理 ACS 与 CPE 之间的 RPC 任务';
COMMENT ON COLUMN device_tasks.id IS '任务 UUID';
COMMENT ON COLUMN device_tasks.device_sn IS '设备序列号';
COMMENT ON COLUMN device_tasks.method IS 'RPC 方法名 (GetParameterValues/SetParameterValues/Reboot 等)';
COMMENT ON COLUMN device_tasks.params IS '方法参数 (JSON 格式)';
COMMENT ON COLUMN device_tasks.priority IS '优先级 (0=最高, 10=默认)';
COMMENT ON COLUMN device_tasks.command_key IS 'TR069 CommandKey';
COMMENT ON COLUMN device_tasks.cwmp_id IS 'SOAP Header ID (如: ID:intrnl.unset.id.GetParameterValues1772248515705.45470049)';
COMMENT ON COLUMN device_tasks.status IS '任务状态: pending/sent/completed/failed/expired/cancelled';
COMMENT ON COLUMN device_tasks.retry_count IS '当前重试次数';
COMMENT ON COLUMN device_tasks.max_retries IS '最大重试次数';
COMMENT ON COLUMN device_tasks.created_at IS '任务创建时间';
COMMENT ON COLUMN device_tasks.sent_at IS '任务发送给 CPE 的时间';
COMMENT ON COLUMN device_tasks.completed_at IS '任务完成时间';
COMMENT ON COLUMN device_tasks.expires_at IS '任务过期时间';
COMMENT ON COLUMN device_tasks.result IS 'CPE 返回结果 (JSON 格式)';
COMMENT ON COLUMN device_tasks.error_code IS '错误码';
COMMENT ON COLUMN device_tasks.error_message IS '错误信息';
COMMENT ON COLUMN device_tasks.source IS '任务来源: api/scheduler/system';
COMMENT ON COLUMN device_tasks.creator_id IS '创建者 ID';
COMMENT ON COLUMN device_tasks.description IS '任务描述';
