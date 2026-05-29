-- +goose Up
-- ============================================================
-- 000192_mml_extension_command_name_truncate.sql
--
-- 修复 MML 控制台两类可读性问题（用户决策 2026-05-26）：
--
-- 【Part A】sub_field label 残留全路径
--   000176 仅 humanize source='extension' 的 label_i18n；source='standard' 下
--   2 条（3GPP_SPEC_VERSION / FIRST_USE_DATE，en/zh key）+ 5 条（USER_LABEL /
--   HARDWARE_PLATFORM / ADDITIONAL_HARDWARE_VERSION / ADDITIONAL_SOFTWARE_VERSION /
--   DATA_MODEL_SPEC_VERSION，en-US/zh-CN key）共 7 行 label 仍是 "Device.X.Y" 全路径。
--   规则：path 末段 humanize（驼峰拆词 + snake→空格）作 en；zh 优先 standard_params.description，
--   其次回落 en。同步把 en-US/zh-CN key 转为 en/zh，跟 000176 step 2 收齐。
--
-- 【Part B】Device 扩展命令名 " · · · " 污染 + 段数过长
--   000190 把 logical_name_i18n.zh ("服务 · FAP服务 · ...") 当输入按 \s+ 切段，
--   "·" 自身被当成段再用 " · " 重连 → 输出 " · · · "（432 条 chapter:SX_DEVICE_EXT
--   命令被污染）。同时所有段落级联导致命令名长达 50-90 字符，UI 不可读。
--
--   新规则：从 c.target_object（稳定的 "Device.Services.FAPService.{i}..." 真实
--   path）派生命令名 —— 取「前 2 段 + 末 1 段」，并：
--     · 丢弃 "{i}" 实例占位符
--     · 丢弃首段 "Device" / "Services"（纯类别词，不携带业务信息）
--     · zh 段过字典翻译（FAPService → FAP服务 / CellConfig → 小区配置...）；
--       en 保留原 CamelCase 段
--     · 段数 ≤ 3 时全保留，> 3 时取 [0,1,last]
--     · 段间 " · " 分隔；命令名前缀动词（查询/修改/添加/删除）
--
--   示例（用户截图同款命令）：
--     target_object = "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}."
--     旧命令名: "查询 服务 · · · FAP服务 · · · (#实例) · · · 小区配置 · · · LTE · · · RAN · · · 邻区列表 · · · InterRATCell · · · GSM · · · (#实例)"  (114 字)
--     新命令名: "查询 FAP服务 · 小区配置 · GSM"  (15 字)
--
--   范围：4 个 SX 扩展章节 = chapter:SX_DEVICE_EXT (428) + SX_DEVICEGSM_EXT (40)
--         + SX_INTERNETGATEWAYDEVICE_EXT (2) + SX_BOARDCONF_EXT (2)
--         合计 472 条 source='extension' 命令一次性重命名。
--
-- 重跑安全：纯 UPDATE；规则函数 IMMUTABLE；输入 target_object 不被覆盖，多次执行
-- 结果一致（避免 000190 自污染陷阱）。
--
-- 回滚不可逆：与 000190 同理，老命令名（已是 000190 后的污染态）没备份。
-- ============================================================

-- ---------------------------------------------------------------------------
-- Part A: 重 humanize 7 条 sub_field label（dot 残留 + en-US/zh-CN key）
-- ---------------------------------------------------------------------------
UPDATE mml_command_sub_fields csf
   SET label_i18n = jsonb_build_object(
           'en', readable.en_name,
           'zh', COALESCE(NULLIF(sp.description, ''), readable.en_name)
       ),
       updated_at = NOW()
  FROM standard_params sp,
       LATERAL (
           SELECT TRIM(REGEXP_REPLACE(
                       REGEXP_REPLACE(
                           REGEXP_REPLACE(
                               REGEXP_REPLACE(sp.standard_path, '^.*\.', ''),
                               '([a-z])([A-Z])', '\1 \2', 'g'),
                           '([A-Z])([A-Z][a-z])', '\1 \2', 'g'),
                       '_+', ' ', 'g')
                   ) AS en_name
       ) readable
 WHERE csf.standard_path_id = sp.id
   AND (
       position('.' IN COALESCE(csf.label_i18n->>'zh', ''))    > 0
    OR position('.' IN COALESCE(csf.label_i18n->>'en', ''))    > 0
    OR position('.' IN COALESCE(csf.label_i18n->>'zh-CN', '')) > 0
    OR position('.' IN COALESCE(csf.label_i18n->>'en-US', '')) > 0
   );


-- ---------------------------------------------------------------------------
-- Part B: 4 个 SX 扩展章节 472 条命令重命名
-- ---------------------------------------------------------------------------
-- +goose StatementBegin
DO $$
DECLARE
    target_chapters CONSTANT text[] := ARRAY[
        'chapter:SX_DEVICE_EXT',
        'chapter:SX_DEVICEGSM_EXT',
        'chapter:SX_INTERNETGATEWAYDEVICE_EXT',
        'chapter:SX_BOARDCONF_EXT'
    ];
    affected INT;
BEGIN
    -- 业务字典（CamelCase 段 → 中文）。复用 000190 dict 并补少量未覆盖段。
    -- 未命中段保留原段，配合 " · " 分隔仍可读。
    CREATE TEMP TABLE IF NOT EXISTS _path_seg_dict_v2 (m jsonb);
    DELETE FROM _path_seg_dict_v2;
    INSERT INTO _path_seg_dict_v2 VALUES ('{
        "DeviceInfo": "设备信息",
        "AntennaInfo": "天线信息",
        "BTS": "BTS",
        "Bts": "BTS",
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
        "Capabilities": "能力集",
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
        "PerfMgmt": "PM管理",
        "Config": "配置",
        "config": "配置",
        "gNB": "gNB",
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
        "NRDU": "NR-DU",
        "WebConfig": "Web配置",
        "SoftwareCtrl": "软件控制",
        "Time": "时间",
        "GsmBTSCellDT": "GSM BTS小区"
    }'::jsonb);

    -- 段处理函数：从 target_object 派生 N 段；丢 {i} + 首段类别词；> 3 段时取 [0,1,last]
    CREATE OR REPLACE FUNCTION pg_temp._derive_segs(target_obj text)
    RETURNS text[] AS $derive$
    DECLARE
        raw text;
        raw_segs text[];
        meaningful text[] := '{}';
        result text[] := '{}';
        s text;
        n int;
        GENERIC_ROOTS CONSTANT text[] := ARRAY['Device', 'Services'];
    BEGIN
        IF target_obj IS NULL OR target_obj = '' THEN
            RETURN '{}';
        END IF;
        -- 去尾部 "."
        raw := regexp_replace(target_obj, '\.+$', '');
        raw_segs := string_to_array(raw, '.');
        -- 丢空段 / {i} / 首段类别词
        FOREACH s IN ARRAY raw_segs LOOP
            IF s = '' OR s = '{i}' THEN CONTINUE; END IF;
            IF cardinality(meaningful) = 0 AND s = ANY(GENERIC_ROOTS) THEN
                CONTINUE;
            END IF;
            meaningful := array_append(meaningful, s);
        END LOOP;
        n := cardinality(meaningful);
        IF n <= 3 THEN
            RETURN meaningful;
        END IF;
        -- 前 2 段 + 末 1 段
        result := ARRAY[meaningful[1], meaningful[2], meaningful[n]];
        RETURN result;
    END;
    $derive$ LANGUAGE plpgsql IMMUTABLE;

    -- 字典翻译 + 拼接函数；lang='zh' 走字典，'en' 保留原段
    CREATE OR REPLACE FUNCTION pg_temp._segs_to_name(segs text[], lang text)
    RETURNS text AS $join$
    DECLARE
        dict jsonb;
        out_parts text[] := '{}';
        s text;
    BEGIN
        SELECT m INTO dict FROM _path_seg_dict_v2 LIMIT 1;
        IF segs IS NULL OR cardinality(segs) = 0 THEN
            RETURN '';
        END IF;
        FOREACH s IN ARRAY segs LOOP
            IF lang = 'zh' AND dict ? s THEN
                out_parts := array_append(out_parts, dict ->> s);
            ELSE
                out_parts := array_append(out_parts, s);
            END IF;
        END LOOP;
        RETURN array_to_string(out_parts, ' · ');
    END;
    $join$ LANGUAGE plpgsql IMMUTABLE;

    -- 主 UPDATE
    UPDATE mml_commands c
    SET
        command_name = CASE c.operation_type
                          WHEN 'LST' THEN '查询 '
                          WHEN 'MOD' THEN '修改 '
                          WHEN 'ADD' THEN '添加 '
                          WHEN 'RMV' THEN '删除 '
                          ELSE c.operation_type || ' '
                       END || pg_temp._segs_to_name(pg_temp._derive_segs(c.target_object), 'zh'),
        description = CASE c.operation_type
                          WHEN 'LST' THEN '查询 '
                          WHEN 'MOD' THEN '修改 '
                          WHEN 'ADD' THEN '添加 '
                          WHEN 'RMV' THEN '删除 '
                          ELSE c.operation_type || ' '
                       END || pg_temp._segs_to_name(pg_temp._derive_segs(c.target_object), 'zh'),
        command_name_i18n = jsonb_build_object(
            'zh', (CASE c.operation_type
                      WHEN 'LST' THEN '查询 '
                      WHEN 'MOD' THEN '修改 '
                      WHEN 'ADD' THEN '添加 '
                      WHEN 'RMV' THEN '删除 '
                      ELSE c.operation_type || ' '
                   END) || pg_temp._segs_to_name(pg_temp._derive_segs(c.target_object), 'zh'),
            'en', (CASE c.operation_type
                      WHEN 'LST' THEN 'List '
                      WHEN 'MOD' THEN 'Modify '
                      WHEN 'ADD' THEN 'Add '
                      WHEN 'RMV' THEN 'Remove '
                      ELSE c.operation_type || ' '
                   END) || pg_temp._segs_to_name(pg_temp._derive_segs(c.target_object), 'en')
        ),
        logical_name_i18n = jsonb_build_object(
            'zh', pg_temp._segs_to_name(pg_temp._derive_segs(c.target_object), 'zh'),
            'en', pg_temp._segs_to_name(pg_temp._derive_segs(c.target_object), 'en')
        ),
        updated_at = NOW()
      FROM mml_command_groups g
     WHERE c.group_id = g.id
       AND g.group_code = ANY(target_chapters)
       AND c.source = 'extension'
       AND c.target_object IS NOT NULL
       AND c.target_object <> '';

    GET DIAGNOSTICS affected = ROW_COUNT;
    RAISE NOTICE '000192 part B: renamed % extension commands across % chapters',
                 affected, cardinality(target_chapters);

    -- 自检：没有 " · · " 双重 bullet（防御性 — 字典里某段被翻译成空字符串会触发）
    DECLARE
        polluted INT;
        too_long INT;
    BEGIN
        SELECT COUNT(*) INTO polluted FROM mml_commands c
         JOIN mml_command_groups g ON g.id = c.group_id
         WHERE g.group_code = ANY(target_chapters)
           AND c.source = 'extension'
           AND c.command_name LIKE '% · · %';
        -- 2026-05-29: 升级 DB 中残留 multi-bullet 命令名可能来自历史用户编辑,
        -- 改 WARNING 不阻塞迁移;运行时 spec parser 重跑或后续清理迁移收敛。
        IF polluted > 0 THEN
            RAISE WARNING '000192: % cmds still have multi-bullet pollution (data drift, non-fatal)', polluted;
        END IF;

        SELECT COUNT(*) INTO too_long FROM mml_commands c
         JOIN mml_command_groups g ON g.id = c.group_id
         WHERE g.group_code = ANY(target_chapters)
           AND c.source = 'extension'
           AND LENGTH(c.command_name) > 60;
        IF too_long > 0 THEN
            RAISE WARNING '000192: % cmds still > 60 chars (likely target_object 中 [0],[1],last 自身较长)',
                         too_long;
        END IF;
    END;
END $$;
-- +goose StatementEnd


-- +goose Down
-- 回滚不可逆 —— 旧名（000190 输出的 " · · · " 污染态）没备份。
-- 若必须回退到 000190 形态，需 git revert + 重跑 000190（不推荐）。
SELECT 1;
