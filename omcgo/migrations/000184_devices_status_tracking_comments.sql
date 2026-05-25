-- +goose Up
-- ============================================================
-- 000184_devices_status_tracking_comments.sql
-- T-0173 Phase 1：设备在线状态异步治理 + devices 表字段 COMMENT 补全。
--
-- 背景：HeartbeatMonitor 与 OfflineDetector 双扫描器并存，HeartbeatMonitor
-- 命中路径只写 is_online=false 不写 last_offline_time，与 OfflineDetector
-- 落库口径不一致。重构合并为单一 DeviceStatusReconciler 后，需要记录"离线
-- 原因"以便审计区分（心跳超时 / 手工 / 重启等）。
--
-- 同时补齐 devices 表大量缺失的 COMMENT —— PostgreSQL \d+ 输出可见性差，
-- 现场排障 / 接手维护时只能靠源码反查字段含义，效率低。
-- ============================================================

-- ─── 1. 新增 last_offline_reason 列 ─────────────────────────────
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS last_offline_reason VARCHAR(32);

-- ─── 2. devices 表 COMMENT 补全 ─────────────────────────────────
COMMENT ON TABLE devices IS
    '设备核心表（按 carrier 分区）。承载 TR-069 设备身份、ACS 通信通道、'
    '实时在线状态、生命周期状态。device_info 表为 1:1 扩展，承载运维属性。';

COMMENT ON COLUMN devices.id IS '设备主键 UUID，由 PG gen_random_uuid() 生成。';
COMMENT ON COLUMN devices.serial_number IS '设备唯一序列号 SN，对应 TR-069 Inform DeviceID.SerialNumber。';
COMMENT ON COLUMN devices.oui IS '厂商组织唯一标识符（6 位十六进制），对应 TR-069 DeviceID.OUI。';
COMMENT ON COLUMN devices.product_class IS '产品分类字符串，对应 TR-069 DeviceID.ProductClass；配合 OUI 路由到具体产品定义。';
COMMENT ON COLUMN devices.manufacturer IS '厂商名称，对应 TR-069 DeviceID.Manufacturer。';
COMMENT ON COLUMN devices.model_name IS '设备型号名，对应 TR-181 Device.DeviceInfo.ModelName；由 ProductRegistry 按 productClass 回填。';
COMMENT ON COLUMN devices.carrier IS '运营商代码：cmcc / ctcc / cucc / other；本表的 LIST 分区键，不可 ALTER。';
COMMENT ON COLUMN devices.technology IS '制式：lte / nr / gsm；决定 KPI 计算分支与 RPC 参数路径选择。';
-- data_model_id 列在部分历史 DB 上已被 DROP（T-0098 后），用 DO 块条件性加 comment
-- 避免在不存在该列的环境上整条迁移失败。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'devices' AND column_name = 'data_model_id'
    ) THEN
        EXECUTE 'COMMENT ON COLUMN devices.data_model_id IS ''旧 datamodel 表外键（T-0098 后已废弃，保留以兼容历史数据；新代码请用 product_id 路由）。''';
    END IF;
END $$;
-- +goose StatementEnd
COMMENT ON COLUMN devices.firmware_version IS '当前运行固件版本号，对应 TR-181 Device.DeviceInfo.SoftwareVersion；变化时触发 device.firmware.changed 事件。';
COMMENT ON COLUMN devices.ip_address IS '设备公网 / 局域网 IP（INET 类型），来自 ACS HTTP RemoteAddr 或 Inform 携带的 UDPConnectionRequestAddress。';
COMMENT ON COLUMN devices.connection_request_url IS 'ACS 主动触发 Connection Request 的回调 URL，对应 TR-181 Device.ManagementServer.ConnectionRequestURL。';
COMMENT ON COLUMN devices.nat_detected IS '是否检测到设备处于 NAT 之后；为 true 时 ConnReq 走 STUN/UDP 通道而非 HTTP。';
COMMENT ON COLUMN devices.udp_connection_request_address IS 'STUN 协商出的 UDP 直达地址（host:port），NAT 穿透场景使用。';
COMMENT ON COLUMN devices.last_inform_at IS '最近一次成功接收 Inform 的时间（PG 持久化，重启不丢）；DeviceStatusReconciler 据此判定离线。';
COMMENT ON COLUMN devices.last_inform_events IS '最近一次 Inform 携带的事件码列表（JSONB，如 ["2 PERIODIC"]），用于排障审计。';
COMMENT ON COLUMN devices.inform_interval IS '心跳间隔（秒），对应 TR-181 Device.ManagementServer.PeriodicInformInterval；Reconciler 据此动态计算离线阈值 = max(2×inform_interval, 600s)。';
COMMENT ON COLUMN devices.site_name IS '所属站点名称（人工录入或运营商规划下发）。';
COMMENT ON COLUMN devices.site_id IS '所属站点 ID（运营商规划标识符）。';
COMMENT ON COLUMN devices.latitude IS '设备安装位置纬度，对应 TR-181 Device.FAP.GPS.LocationLatitude。';
COMMENT ON COLUMN devices.longitude IS '设备安装位置经度，对应 TR-181 Device.FAP.GPS.LocationLongitude。';
COMMENT ON COLUMN devices.extension_data IS '扩展字段 JSONB，用于承载未规范化的设备属性或厂商私有数据。';
COMMENT ON COLUMN devices.deleted_at IS '软删除时间戳，NULL 表示未删除；分区索引 idx_devices_deleted_at WHERE deleted_at IS NOT NULL 加速回收站查询。';
COMMENT ON COLUMN devices.deleted_by IS '执行软删除操作的用户标识（用户名或用户 ID 字符串）。';
COMMENT ON COLUMN devices.created_at IS '记录入库时间（PG 自动填充）。';
COMMENT ON COLUMN devices.updated_at IS '记录最近更新时间（PG 触发器 trigger_devices_updated_at 自动维护）。';

-- T-0162 已加的列：补充更完整说明，覆盖原迁移过于简短的注释。
COMMENT ON COLUMN devices.lifecycle_state IS
    '设备业务生命周期阶段（T-0162 解耦后单字段）。取值：discovered / registered / '
    'provisioning / commissioned / maintenance / decommissioned。与 is_online 正交：'
    'commissioned + is_online=false 表示"已入网但当前掉线"。';
COMMENT ON COLUMN devices.is_online IS
    '实时在线状态（T-0162 解耦后单字段）。true=最近一次 Inform 在心跳阈值内。'
    '写入路径：ACS Inform 接收时置 true；DeviceStatusReconciler 扫描超时设备置 false。';

-- T-0173: 新增列说明。
COMMENT ON COLUMN devices.last_offline_reason IS
    '最近一次被标记离线的原因。取值：heartbeat_timeout（Reconciler 扫描判定）/ '
    'manual（管理面手工操作）/ reboot（重启过程中短暂离线）/ NULL（从未离线或已恢复在线）。'
    '诊断字段，不参与业务决策。';

-- ─── 3. last_offline_reason 索引（可选；按需开启）────────────────
-- 当前所有过滤条件都基于 is_online，last_offline_reason 仅用于诊断展示，不加索引。

-- +goose Down
COMMENT ON COLUMN devices.last_offline_reason IS NULL;
ALTER TABLE devices DROP COLUMN IF EXISTS last_offline_reason;

-- COMMENT 不在 Down 中逐个清空 —— 表本身存在的列上的 COMMENT 不影响功能,
-- 回滚成本与收益不匹配。
