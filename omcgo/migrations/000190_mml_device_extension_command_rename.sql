-- +goose Up
-- ============================================================
-- 000189_mml_device_extension_command_rename.sql
-- 重命名 Device 扩展（chapter:SX_DEVICE_EXT）下 428 条命令的人类可读名
-- （中 + 英），让 UI 显示从"列出 Device DeviceInfo BTS "这种机械拼接的
-- 路径段，变成"查询 设备信息 · BTS"这种段间分隔 + 业务字典化的可读名。
--
-- 用户决策（2026-05-26）：方案 B —— 机械清洗 + 业务字典。
--
-- 改名涉及 mml_commands 表 4 列：
--   - command_name              (人类可读中文，UI 兜底显示)
--   - description               (= command_name，列表搜索匹配)
--   - command_name_i18n.zh/en   (UI 优先显示，含 "查询/修改" or "List/Modify" 前缀)
--   - logical_name_i18n.zh/en   (命令树标签，不含动词前缀)
--
-- 不动 command_code / logical_code / operation_type / target_paths 等业务字段
-- （它们是 mml_command_sub_fields / FAPouter / Fanouter 等下游消费方的稳定 ID）。
--
-- 规则（见 _beautify_zh / _beautify_en 函数）：
--   1. 去 path 头部 "Device " 前缀（每条都有，冗余）
--   2. 按空格切段；段间用 " · " 分隔（中文 UI 标准做法）
--   3. 中文版：每段查 30+ 条业务字典（DeviceInfo→设备信息 / FaultMgmt→
--      故障管理 / Synchronization→时钟同步 等），未命中保留英文段
--   4. {i} 占位符 → 中"(#实例)"、英"(#i)"
--   5. 加动词前缀：中 LST→"查询 " / MOD→"修改 "；英 LST→"List " / MOD→"Modify "
--
-- 重跑安全：纯 UPDATE，规则函数 IMMUTABLE，多次执行结果一致。
-- ============================================================

-- +goose StatementBegin
DO $$
DECLARE
    device_ext_group_id UUID;
BEGIN
    SELECT id INTO device_ext_group_id
    FROM mml_command_groups
    WHERE group_code = 'chapter:SX_DEVICE_EXT';

    IF device_ext_group_id IS NULL THEN
        RAISE NOTICE 'group chapter:SX_DEVICE_EXT not found — skip rename (新部署 / 数据未灌）';
        RETURN;
    END IF;

    -- 临时函数：本 migration 跑完即 DROP，不污染全局命名空间。

    -- 业务字典：path 段 → 中文。用 jsonb 当 map，O(1) 查询。
    -- 不在字典里的段保留英文原样，配合段间 " · " 分隔已经比拼接路径可读得多。
    CREATE TEMP TABLE IF NOT EXISTS _path_seg_dict (m jsonb);
    INSERT INTO _path_seg_dict VALUES ('{
        "DeviceInfo": "设备信息",
        "AntennaInfo": "天线信息",
        "BTS": "BTS",
        "GPS": "GPS",
        "GSM": "GSM",
        "SAS": "SAS",
        "CPI": "CPI",
        "FAP": "FAP",
        "FAPService": "FAP服务",
        "EU": "EU",
        "RU": "RU",
        "NR": "NR",
        "Nr": "NR",
        "LTE": "LTE",
        "LTECell": "LTE小区",
        "EUTRA": "EUTRA",
        "IRAT": "异系统",
        "RAN": "RAN",
        "CN": "CN",
        "TA": "TA",
        "PLMNList": "PLMN列表",
        "SliceList": "切片列表",
        "Mobility": "移动性",
        "IdleMode": "空闲态",
        "ConnMode": "连接态",
        "Carrier": "载波",
        "NeighborList": "邻区列表",
        "CellConfig": "小区配置",
        "Capabilities": "能力",
        "AccessMgmt": "接入管理",
        "EnergySaveParam": "节能参数",
        "NrSibParams": "NR SIB参数",
        "MacSchedulerInfo": "MAC调度信息",
        "Duf1uLog": "DU F1U日志",
        "B1MeasureCtrl": "B1测量控制",
        "B2MeasureCtrl": "B2测量控制",
        "A1MeasureCtrl": "A1测量控制",
        "A2MeasureCtrl": "A2测量控制",
        "FaultMgmt": "故障管理",
        "CurrentAlarm": "当前告警",
        "ExpeditedEvent": "紧急事件",
        "HistoryEvent": "历史事件",
        "QueuedEvent": "排队事件",
        "Synchronization": "时钟同步",
        "PTP1588": "PTP1588",
        "TFCS": "TFCS",
        "TfcsParams": "TFCS参数",
        "HANRU": "HANRU",
        "NL": "NL",
        "LanConfig": "LAN配置",
        "Ipsec": "IPsec",
        "MRMgmt": "MR管理",
        "Config": "配置",
        "config": "配置",
        "gNB": "gNB",
        "Services": "服务",
        "Embedded": "嵌入式",
        "EMBEDDED_EPCBearerLBOQos": "EPC承载LBO QoS",
        "EMBEDDED_EPCBearerLBOTft": "EPC承载LBO TFT",
        "ManagementServer": "管理服务器",
        "sslStatus": "SSL状态",
        "Https": "HTTPS",
        "HaltReason": "停机原因",
        "RemoteDeviceList": "远端设备列表",
        "SignallingTrace": "信令跟踪",
        "UeInfoUpload": "UE信息上传",
        "UeAccess": "UE接入",
        "Ethernet": "以太网",
        "Interface": "接口",
        "IPv4Address": "IPv4地址",
        "IPv6Address": "IPv6地址",
        "PppoeAddress": "PPPoE地址",
        "VlanInterface": "VLAN接口",
        "VlanPppoeAddress": "VLAN PPPoE地址",
        "DefaultIpRoute": "默认IP路由",
        "IpRoute": "IP路由",
        "IP": "IP",
        "KeepalivedMgmt": "Keepalived管理",
        "VrrpMgmt": "VRRP管理",
        "VirtualIpList": "虚拟IP列表",
        "LAN_HostConfigManagement": "LAN主机配置管理",
        "IPInterface": "IP接口",
        "NgapMgmt": "NGAP管理",
        "NRCU": "NR-CU",
        "NRDU": "NR-DU"
    }'::jsonb);

    -- 工具函数：把"Device A B C {i}"形态格式化为中文/英文段连接。
    CREATE OR REPLACE FUNCTION pg_temp._beautify(input text, lang text) RETURNS text AS $beautify$
    DECLARE
        dict jsonb;
        cleaned text;
        segs text[];
        out_parts text[] := '{}';
        s text;
        zh text;
    BEGIN
        IF input IS NULL THEN
            RETURN NULL;
        END IF;
        SELECT m INTO dict FROM _path_seg_dict LIMIT 1;
        -- 1) 去前后空白 + 去头部 "Device "
        cleaned := btrim(input);
        cleaned := regexp_replace(cleaned, '^Device\s+', '');
        IF cleaned = '' THEN
            cleaned := 'Device';
        END IF;
        -- 2) 切段（多个空白视为一个分隔符）
        segs := regexp_split_to_array(cleaned, '\s+');
        -- 3) 段级转换
        FOREACH s IN ARRAY segs LOOP
            IF s = '' THEN CONTINUE; END IF;
            IF s = '{i}' THEN
                out_parts := array_append(out_parts, CASE WHEN lang='zh' THEN '(#实例)' ELSE '(#i)' END);
            ELSIF lang = 'zh' AND dict ? s THEN
                zh := dict ->> s;
                out_parts := array_append(out_parts, zh);
            ELSE
                out_parts := array_append(out_parts, s);
            END IF;
        END LOOP;
        -- 4) 段间 " · " 分隔
        RETURN array_to_string(out_parts, ' · ');
    END;
    $beautify$ LANGUAGE plpgsql IMMUTABLE;

    -- ────────────────────────────────────────────────────────────
    -- 主 UPDATE：428 条命令一次性改名
    -- ────────────────────────────────────────────────────────────
    UPDATE mml_commands c
    SET
        command_name = CASE c.operation_type
            WHEN 'LST' THEN '查询 '
            WHEN 'MOD' THEN '修改 '
            ELSE c.operation_type || ' '
        END || pg_temp._beautify(
            -- 复用 logical_name_i18n.zh 作输入（值形如 "Device DeviceInfo BTS "）
            COALESCE(c.logical_name_i18n->>'zh', c.command_name),
            'zh'
        ),
        description = CASE c.operation_type
            WHEN 'LST' THEN '查询 '
            WHEN 'MOD' THEN '修改 '
            ELSE c.operation_type || ' '
        END || pg_temp._beautify(
            COALESCE(c.logical_name_i18n->>'zh', c.command_name),
            'zh'
        ),
        command_name_i18n = jsonb_build_object(
            'zh', (CASE c.operation_type
                       WHEN 'LST' THEN '查询 '
                       WHEN 'MOD' THEN '修改 '
                       ELSE c.operation_type || ' '
                   END) || pg_temp._beautify(COALESCE(c.logical_name_i18n->>'zh', c.command_name), 'zh'),
            'en', (CASE c.operation_type
                       WHEN 'LST' THEN 'List '
                       WHEN 'MOD' THEN 'Modify '
                       ELSE c.operation_type || ' '
                   END) || pg_temp._beautify(COALESCE(c.logical_name_i18n->>'en', c.command_name), 'en')
        ),
        logical_name_i18n = jsonb_build_object(
            'zh', pg_temp._beautify(COALESCE(c.logical_name_i18n->>'zh', c.command_name), 'zh'),
            'en', pg_temp._beautify(COALESCE(c.logical_name_i18n->>'en', c.command_name), 'en')
        ),
        updated_at = NOW()
    WHERE c.group_id = device_ext_group_id;

    RAISE NOTICE 'mml_device_extension_command_rename: updated rows for group %', device_ext_group_id;
END $$;
-- +goose StatementEnd

-- +goose Down
-- 回滚不可逆 —— 老名字（"列出 Device A B C "形态）已被覆盖，原值没备份。
-- 若必须回滚，重新跑 000174 的 INSERT...SELECT 逻辑可重建（但会丢 sub_field 关联）。
SELECT 1;  -- no-op down
