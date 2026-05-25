-- +goose Up
-- ============================================================
-- 000185_device_info_status_tracking_comments.sql
-- T-0173 Phase 1：device_info 累计在线时长字段 + 字段 COMMENT 全量补全。
--
-- 背景：
--   - device_info 表自 000003 引入以来,仅在后续迁移逐次补的少量列有 COMMENT,
--     基础 25+ 字段全部缺 COMMENT；
--   - 当前"在线时长"前端是 SQL 派生（NOW - last_online_time）,把离线段也算
--     进去导致前端展示长期偏大。新增 cumulative_online_duration 字段,由
--     DeviceStatusReconciler 在 online→offline 边沿事务性累加,
--     "总在线时长"前端读 cumulative + (is_online ? NOW - last_online_time : 0)
--     即可得到精确值。
-- ============================================================

-- ─── 1. 新增 cumulative_online_duration 列 ──────────────────────
ALTER TABLE device_info
    ADD COLUMN IF NOT EXISTS cumulative_online_duration BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN device_info.cumulative_online_duration IS
    '累计在线总时长（秒）。online→offline 转换时由 DeviceStatusReconciler 事务性累加 '
    '(NOW - last_online_time)。新字段从启用日开始累计,存量设备初值为 0;查询"总在线时长" '
    '应用 = cumulative_online_duration + (is_online ? NOW - last_online_time : 0)。';

-- ─── 2. device_info 表 COMMENT 补全 ─────────────────────────────
COMMENT ON TABLE device_info IS
    '设备运维扩展信息表（与 devices 1:1）。承载 device_name / 位置 / 小区配置 / 运维状态聚合 / '
    '生命周期时间戳 / 累计时长等"快速查询"列;扩展字段由 InfoSyncer 从 device_parameters 投影。';

COMMENT ON COLUMN device_info.device_id IS '外键 → devices.id（1:1 关系,与 devices 主键同分布）。';
COMMENT ON COLUMN device_info.device_name IS '设备显示名（站点名 / 别名）;运营商规划或人工设置,区别于 serial_number。';
COMMENT ON COLUMN device_info.address IS '设备安装地址（文本）,人工录入,与 latitude / longitude 互补。';
COMMENT ON COLUMN device_info.remark IS '运维备注文本,人工录入。';
COMMENT ON COLUMN device_info.project_status IS '工程阶段状态:planned / installed / commissioned / decommissioned,与 devices.lifecycle_state 业务进度互补。';
COMMENT ON COLUMN device_info.height IS '天线安装挂高（米）。';

-- 小区配置类
COMMENT ON COLUMN device_info.eci IS 'E-UTRAN Cell Identifier (28 bit),LTE 小区全局标识 = eNodeB ID (20 bit) << 8 | Cell ID (8 bit)。';
COMMENT ON COLUMN device_info.pci IS '物理小区标识 (Physical Cell ID,0-503),对应 TR-181 Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PhyCellID。';
COMMENT ON COLUMN device_info.cell_id IS '小区 ID (8 bit),ECI 的低 8 位。';
COMMENT ON COLUMN device_info.freq_point IS '频点号（LTE EARFCN / NR NRARFCN）。';
COMMENT ON COLUMN device_info.bandwidth IS '工作带宽（MHz,小数支持 1.4 / 3 / 5 / 10 / 15 / 20 等）。';
COMMENT ON COLUMN device_info.transmit_power IS '发射功率（dBm）,对应 TR-181 FAPService.{i}.Capabilities.MaxTxPower。';
COMMENT ON COLUMN device_info.plmn IS '公共陆地移动网络号（MCC+MNC,如 46000 表示 CMCC LTE）。';

-- 运维状态聚合（由 InfoSyncer 计算）
COMMENT ON COLUMN device_info.rf_status IS 'RF 状态聚合:on/off,多小区时可逗号分隔;由 InfoSyncer.CalcRFStatus 从参数派生。';
COMMENT ON COLUMN device_info.cell_status IS '小区状态聚合:active / inactive / locked;由 InfoSyncer.CalcCellStatus 派生。';
COMMENT ON COLUMN device_info.mme_status IS 'MME (4G) / AMF (5G) 连接状态:connected / disconnected;由 InfoSyncer.CalcMMEStatus 派生。';
COMMENT ON COLUMN device_info.sync_status IS '时钟同步状态:synchronized / gps / 1588 / rem / not_synchronized;由 InfoSyncer.CalcSyncStatus 派生。';
COMMENT ON COLUMN device_info.kpi_status IS 'KPI 采集状态:reporting / silent,反映 PM 文件上报是否正常。';
COMMENT ON COLUMN device_info.num_of_cells IS '小区数（多小区设备）,由 InfoSyncer.CalcNumOfCells 计数派生。';
COMMENT ON COLUMN device_info.gps_status IS 'GPS 锁定状态:locked / searching / disabled;由 InfoSyncer.CalcGPSStatus 派生。';
COMMENT ON COLUMN device_info.alarm_severity IS '当前最严重告警级别冗余字段:critical / major / minor / warning / none;由告警模块维护。';
COMMENT ON COLUMN device_info.license_status IS '设备 License 状态:valid / expired / missing;由 InfoSyncer.CalcLicenseStatus 派生。';

-- 标识 / 版本
COMMENT ON COLUMN device_info.mac IS '设备 MAC 地址（HEX,无分隔符）,对应 TR-181 Device.Ethernet.Interface.{i}.MACAddress。';
COMMENT ON COLUMN device_info.hardware_version IS '硬件版本号,对应 TR-181 Device.DeviceInfo.HardwareVersion。';

-- 生命周期时间戳（T-0173 重新确立语义）
COMMENT ON COLUMN device_info.first_online_time IS
    '设备首次成功 Inform 注册的时刻;一旦写入永不变更;由 InfoSyncer.RecordOnline 在 first_online_time IS NULL 时填充。';
COMMENT ON COLUMN device_info.last_online_time IS
    '本次上线时刻（offline→online 边沿）。由 DeviceService.UpdateFromInform 在检测到状态翻转时,'
    '或 InfoSyncer.RecordOnline 调用时写入。前端"当前在线时长" = is_online ? NOW - last_online_time : 0。';
COMMENT ON COLUMN device_info.last_offline_time IS
    '本次离线时刻（online→offline 边沿）。由 DeviceStatusReconciler.markOffline 事务性写入,'
    '同时累加 cumulative_online_duration += NOW - last_online_time。';
COMMENT ON COLUMN device_info.run_time IS
    '设备自报运行时长（秒）,对应 TR-181 Device.DeviceInfo.UpTime;由 InfoSyncer.SyncFromParameters 在每次参数同步时刷新。'
    '注意:这是"设备从上次本地重启算起的时长",与 OMC 视角的累计在线时长（cumulative_online_duration）含义不同。';

-- 元数据
COMMENT ON COLUMN device_info.creator IS '记录创建者用户标识。';
COMMENT ON COLUMN device_info.updater IS '记录最近更新者用户标识。';
COMMENT ON COLUMN device_info.created_at IS '记录入库时间（PG 自动填充）。';
COMMENT ON COLUMN device_info.updated_at IS '记录最近更新时间（PG 触发器 trigger_device_info_updated_at 自动维护）。';

-- +goose Down
COMMENT ON COLUMN device_info.cumulative_online_duration IS NULL;
ALTER TABLE device_info DROP COLUMN IF EXISTS cumulative_online_duration;
