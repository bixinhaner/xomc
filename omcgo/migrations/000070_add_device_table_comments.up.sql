-- ============================================================
-- 000070_add_device_table_comments.up.sql
-- 为设备相关表添加字段注释，便于数据库工具查看
-- ============================================================

-- ===== devices 表 =====
COMMENT ON TABLE devices IS '设备核心表：存储 TR069 设备身份与连接信息';

COMMENT ON COLUMN devices.id IS '设备唯一标识 (UUID)';
COMMENT ON COLUMN devices.serial_number IS '设备序列号 (SN)，全局唯一，用于设备识别';
COMMENT ON COLUMN devices.oui IS '组织唯一标识符 (OUI)，厂商标识，6位十六进制';
COMMENT ON COLUMN devices.product_class IS '产品型号，如 AirScale ASN01';
COMMENT ON COLUMN devices.manufacturer IS '厂商名称，如 Nokia、华为';
COMMENT ON COLUMN devices.model_name IS '设备型号名称';
COMMENT ON COLUMN devices.carrier IS '运营商代码：cmcc(中国移动)/ctcc(中国电信)/cucc(中国联通)';
COMMENT ON COLUMN devices.technology IS '无线制式：lte(4G)/nr(5G)';
COMMENT ON COLUMN devices.data_model_id IS '关联的数据模型定义 ID，用于 TR069 参数解析';
COMMENT ON COLUMN devices.status IS '设备生命周期状态：discovered(已发现)/registered(已注册)/provisioning(配置中)/active(活跃)/maintenance(维护中)/offline(离线)/decommissioned(已退服)';
COMMENT ON COLUMN devices.firmware_version IS '当前固件版本号';
COMMENT ON COLUMN devices.ip_address IS '设备 IP 地址 (PostgreSQL INET 类型)';
COMMENT ON COLUMN devices.connection_request_url IS 'TR069 Connection Request URL，用于 ACS 反向唤醒设备';
-- nat_detected 和 udp_connection_request_address 由迁移 000034 添加并注释，此处不重复
COMMENT ON COLUMN devices.last_inform_at IS '最后一次 TR069 Inform 时间';
COMMENT ON COLUMN devices.last_inform_events IS '最后一次 Inform 的事件码列表 (JSONB 数组)，如 ["2 PERIODIC", "6 CONNECTION REQUEST"]';
COMMENT ON COLUMN devices.inform_interval IS 'Inform 心跳间隔（秒），默认 300';
COMMENT ON COLUMN devices.site_name IS '站点名称';
COMMENT ON COLUMN devices.site_id IS '站点 ID';
COMMENT ON COLUMN devices.latitude IS '纬度（WGS84）';
COMMENT ON COLUMN devices.longitude IS '经度（WGS84）';
COMMENT ON COLUMN devices.extension_data IS '扩展数据 (JSONB)，存储运营商定制字段';
COMMENT ON COLUMN devices.created_at IS '记录创建时间';
COMMENT ON COLUMN devices.updated_at IS '记录最后更新时间（自动触发器维护）';
-- deleted_at 由迁移 000069 添加，此处不重复注释

-- ===== device_info 表 =====
COMMENT ON TABLE device_info IS '设备运维扩展信息：运维人员手动填写 + TR069 参数自动同步的数据';

COMMENT ON COLUMN device_info.device_id IS '关联 devices.id，1:1 关系，主键';
COMMENT ON COLUMN device_info.device_name IS '设备名称，运维人员自定义的易读名称';
COMMENT ON COLUMN device_info.address IS '安装地址，详细地址描述';
COMMENT ON COLUMN device_info.remark IS '备注信息';
COMMENT ON COLUMN device_info.project_status IS '项目状态：building(在建)/delivered(已交付)/operating(商用)/deactivated(退服)';
COMMENT ON COLUMN device_info.height IS '安装高度（米）';

-- 无线参数（从 TR069 参数同步）
COMMENT ON COLUMN device_info.eci IS 'E-UTRAN 小区标识符 (28位)：PLMN(6位) + CellID(22位)';
COMMENT ON COLUMN device_info.pci IS '物理小区标识 (Physical Cell ID)：LTE 范围 0-503，NR 范围 0-1007';
COMMENT ON COLUMN device_info.cell_id IS '逻辑小区 ID';
COMMENT ON COLUMN device_info.freq_point IS '频点号：LTE 为 EARFCN，NR 为 NRARFCN';
COMMENT ON COLUMN device_info.bandwidth IS '载波带宽（MHz）';
COMMENT ON COLUMN device_info.transmit_power IS '发射功率（dBm）';
COMMENT ON COLUMN device_info.plmn IS '公众陆地移动网标识：MCC(3位) + MNC(2-3位)，如 46000';

-- 状态字段（从 TR069 参数计算）
COMMENT ON COLUMN device_info.rf_status IS '射频状态：on(开启发射)/off(关闭)/error(故障)';
COMMENT ON COLUMN device_info.cell_status IS '小区状态：normal(正常服务)/inactive(未激活)/fault(故障)/decommissioned(已退服)';
COMMENT ON COLUMN device_info.mme_status IS 'MME/AMF 连接状态：connected(2+连接)/partial(仅1连接)/disconnected(断开)';
COMMENT ON COLUMN device_info.sync_status IS '时钟同步源状态：gps(GPS同步)/beidou(北斗同步)/ntp(NTP/1588同步)/error(失步)';
COMMENT ON COLUMN device_info.kpi_status IS 'KPI 状态（保留字段）';
COMMENT ON COLUMN device_info.gps_status IS 'GPS 定位状态：normal(正常)/abnormal(异常)/no_signal(无信号)';
COMMENT ON COLUMN device_info.alarm_severity IS '当前最高告警级别：critical(严重)/major(主要)/minor(次要)/warning(警告)';
COMMENT ON COLUMN device_info.license_status IS '许可证状态：active(有效)/expiring(即将过期≤30天)/expired(已过期)';

-- 硬件信息
COMMENT ON COLUMN device_info.mac IS 'MAC 地址';
COMMENT ON COLUMN device_info.hardware_version IS '硬件版本号';

-- 时间信息
COMMENT ON COLUMN device_info.first_online_time IS '首次上线时间';
COMMENT ON COLUMN device_info.last_offline_time IS '最后离线时间';
COMMENT ON COLUMN device_info.run_time IS '累计运行时长（秒）';

-- 审计字段
COMMENT ON COLUMN device_info.creator IS '创建人用户名';
COMMENT ON COLUMN device_info.updater IS '最后更新人用户名';
COMMENT ON COLUMN device_info.created_at IS '记录创建时间';
COMMENT ON COLUMN device_info.updated_at IS '记录最后更新时间';

-- ===== device_parameters 表 =====
COMMENT ON TABLE device_parameters IS 'TR069 参数值存储：每设备每参数一条记录，用于参数树展示和配置同步';

COMMENT ON COLUMN device_parameters.device_id IS '关联 devices.id';
COMMENT ON COLUMN device_parameters.parameter_path IS 'TR069 参数全路径，如 Device.FAPControl.RFTxStatus';
COMMENT ON COLUMN device_parameters.parameter_value IS '参数值（字符串形式）';
COMMENT ON COLUMN device_parameters.parameter_type IS '参数类型：string/int/unsignedInt/boolean/dateTime/base64';
COMMENT ON COLUMN device_parameters.writable IS '参数是否可写（来自 GetParameterNames 响应）';
COMMENT ON COLUMN device_parameters.last_updated_at IS '参数最后更新时间';

-- ===== device_registrations 表 =====
COMMENT ON TABLE device_registrations IS '设备预注册表：批量导入待上线设备的 SN 及规划信息，实现零接触部署';

COMMENT ON COLUMN device_registrations.id IS '预注册记录 ID (UUID)';
COMMENT ON COLUMN device_registrations.serial_number IS '设备序列号 (SN)，待上线设备的唯一标识';
COMMENT ON COLUMN device_registrations.group_id IS '预分配的设备组 ID，设备上线后自动加入';
COMMENT ON COLUMN device_registrations.device_id IS '设备实际上线后关联的 devices.id';
COMMENT ON COLUMN device_registrations.status IS '状态：pending(待上线)/online(已上线)/expired(已过期)';

-- 规划信息（导入时填写，上线后同步到 device_info）
COMMENT ON COLUMN device_registrations.site_name IS '规划站点名称';
COMMENT ON COLUMN device_registrations.device_name IS '规划设备名称';
COMMENT ON COLUMN device_registrations.longitude IS '规划经度';
COMMENT ON COLUMN device_registrations.latitude IS '规划纬度';
COMMENT ON COLUMN device_registrations.height IS '规划安装高度（米）';
COMMENT ON COLUMN device_registrations.azimuth IS '水平方位角（度），范围 0-359';
COMMENT ON COLUMN device_registrations.tilt_angle IS '机械下倾角（度），范围 0-9';
COMMENT ON COLUMN device_registrations.beam_width IS '垂直3dB波束宽度，范围 1-9';
COMMENT ON COLUMN device_registrations.remark IS '备注信息';

-- 审计
COMMENT ON COLUMN device_registrations.created_by IS '创建人用户名';
COMMENT ON COLUMN device_registrations.import_batch_id IS '批量导入批次 ID，用于追溯导入来源';
COMMENT ON COLUMN device_registrations.created_at IS '记录创建时间';
COMMENT ON COLUMN device_registrations.updated_at IS '记录最后更新时间';

-- ===== device_groups 表 =====
COMMENT ON TABLE device_groups IS '设备分组表：支持树形层级拓扑，用于组织架构和权限控制';

COMMENT ON COLUMN device_groups.id IS '分组 ID (UUID)';
COMMENT ON COLUMN device_groups.name IS '分组名称';
COMMENT ON COLUMN device_groups.parent_id IS '父分组 ID，NULL 表示根分组';
COMMENT ON COLUMN device_groups.carrier IS '所属运营商代码，NULL 表示跨运营商通用分组';
COMMENT ON COLUMN device_groups.description IS '分组描述';
COMMENT ON COLUMN device_groups.sort_order IS '同级排序序号，数字越小越靠前';
COMMENT ON COLUMN device_groups.created_at IS '创建时间';
COMMENT ON COLUMN device_groups.updated_at IS '最后更新时间';

-- ===== device_group_members 表 =====
COMMENT ON TABLE device_group_members IS '设备分组成员关联表：设备与分组的多对多关系';

COMMENT ON COLUMN device_group_members.group_id IS '分组 ID';
COMMENT ON COLUMN device_group_members.device_id IS '设备 ID';
COMMENT ON COLUMN device_group_members.added_at IS '加入分组时间';

-- ===== user_column_configs 表 =====
COMMENT ON TABLE user_column_configs IS '用户自定义列配置：存储列表页面的列显示/隐藏/宽度等个性化设置';

COMMENT ON COLUMN user_column_configs.user_id IS '用户 ID，关联 users.id';
COMMENT ON COLUMN user_column_configs.page_key IS '页面标识：device_list/alarm_list/pm_list/kpi_list 等';
COMMENT ON COLUMN user_column_configs.columns IS '列配置 JSON 数组，格式：[{"key":"serial_number","visible":true,"width":120,"fixed":"left"}]';
COMMENT ON COLUMN user_column_configs.created_at IS '创建时间';
COMMENT ON COLUMN user_column_configs.updated_at IS '最后更新时间';
