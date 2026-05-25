-- +goose Up
-- ============================================================
-- 900002_seed_devices.sql
-- 设备管理种子数据：30,000 台设备、15 个设备分组、10 条设备规则、30 台回收站设备
-- 幂等：快速检查设备数量，已存在则跳过
-- ============================================================

-- 命名空间 UUID 用于 uuid_generate_v5 确定性 ID 生成
-- Namespace: 6ba7b812-9dad-11d1-80b4-00c04fd430c8 (DNS)
-- 分组/规则 ID = uuid_generate_v5(ns, 'seed-group-1') 等

-- 快速检查 + 全部逻辑封装在一个事务中
-- +goose StatementBegin
DO $$
DECLARE
    v_count INT;
BEGIN
    -- 幂等检查：已有 >= 1000 台设备则跳过
    SELECT count(*) INTO v_count FROM devices WHERE deleted_at IS NULL;
    IF v_count >= 1000 THEN
        RAISE NOTICE 'Device seed data already exists (% devices), skipping 900002', v_count;
        RETURN;
    END IF;

    -- ============================================================
    -- 1. 设备分组 (15 条: 5 个一级 + 10 个二级)
    -- ============================================================
    -- 一级分组
    INSERT INTO device_groups (id, name, parent_id, carrier, description, sort_order, level, status, created_by)
    VALUES
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cmcc'), '移动设备域', NULL, 'cmcc', '中国移动设备管理域', 1, 1, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-ctcc'), '电信设备域', NULL, 'ctcc', '中国电信设备管理域', 2, 1, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cucc'), '联通设备域', NULL, 'cucc', '中国联通设备管理域', 3, 1, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-test'), '测试设备域', NULL, 'cmcc', '测试与验收设备', 4, 1, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-ops'),   '运维设备域', NULL, 'ctcc', '运维专用设备', 5, 1, 'active', 'system')
    ON CONFLICT (id) DO NOTHING;

    -- 二级分组
    INSERT INTO device_groups (id, name, parent_id, carrier, description, sort_order, level, status, created_by)
    VALUES
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cmcc-bj'), '北京移动', uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cmcc'), 'cmcc', '北京地区移动设备', 1, 2, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cmcc-sh'), '上海移动', uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cmcc'), 'cmcc', '上海地区移动设备', 2, 2, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cmcc-gd'), '广东移动', uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cmcc'), 'cmcc', '广东地区移动设备', 3, 2, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-ctcc-js'), '江苏电信', uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-ctcc'), 'ctcc', '江苏地区电信设备', 1, 2, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-ctcc-zj'), '浙江电信', uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-ctcc'), 'ctcc', '浙江地区电信设备', 2, 2, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cucc-sd'), '山东联通', uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cucc'), 'cucc', '山东地区联通设备', 1, 2, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cucc-hn'), '河南联通', uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-cucc'), 'cucc', '河南地区联通设备', 2, 2, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-test-compat'), '联调测试组', uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-test'), 'cmcc', '联调测试专用设备', 1, 2, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-test-conform'), '一致性测试组', uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-test'), 'cmcc', '一致性验证设备', 2, 2, 'active', 'system'),
        (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-ops-patrol'), '巡检设备组', uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid, 'seed-group-ops'), 'ctcc', '日常巡检设备', 1, 2, 'active', 'system')
    ON CONFLICT (id) DO NOTHING;

    -- ============================================================
    -- 2. 设备规则 — migration 000162_drop_device_rules 已删表，本段下线
    --   原 10 条 device_rules INSERT 删除（device_rules 表不再存在）
    -- ============================================================

    RAISE NOTICE 'Seed groups created (device_rules removed in 000162)';
END;
$$;
-- +goose StatementEnd

-- ============================================================
-- 3. 批量生成 30,000 台设备
--    分布: cmcc 12,000 / ctcc 10,000 / cucc 8,000
--    生命周期分布（migration 000137 后 status 拆为 lifecycle_state + is_online）：
--      70% commissioned + online=true        （原 'active'）
--      15% commissioned + online=false       （原 'offline'，曾上线过的离线）
--      10% registered   + online=false       （原 'registered'，从未上线）
--       5% maintenance  + online=true        （原 'maintenance'）
-- ============================================================

-- CMCC 设备 (12,000 台)
-- +goose StatementBegin
DO $$
DECLARE
    i INT; dev_id UUID; dev_sn VARCHAR;
    lifecycle_v VARCHAR; online_v BOOLEAN;
    oui_v VARCHAR; pc_v VARCHAR; mfr_v VARCHAR; model_v VARCHAR; tech_v VARCHAR; fw_v VARCHAR;
    site_v VARCHAR; lat_v FLOAT; lon_v FLOAT;
    ns UUID := '6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid;
BEGIN
    FOR i IN 1..12000 LOOP
        IF    i <= 8400  THEN lifecycle_v := 'commissioned'; online_v := TRUE;
        ELSIF i <= 10200 THEN lifecycle_v := 'commissioned'; online_v := FALSE;
        ELSIF i <= 11400 THEN lifecycle_v := 'registered';   online_v := FALSE;
        ELSE                  lifecycle_v := 'maintenance';  online_v := TRUE;
        END IF;
        IF i % 3 = 0 THEN tech_v := 'nr'; oui_v := '00A0C6'; pc_v := 'gNB-100'; mfr_v := 'Huawei'; model_v := 'AAU5613'; fw_v := 'V100R019C10';
        ELSIF i % 3 = 1 THEN tech_v := 'lte'; oui_v := '001A2B'; pc_v := 'SmallCell-LTE'; mfr_v := 'BaiCells'; model_v := 'BC-ENB-100'; fw_v := 'V3.2.1';
        ELSE tech_v := 'lte'; oui_v := '00E0FC'; pc_v := 'FAP-LTE-200'; mfr_v := 'ZTE'; model_v := 'ZXSDR-B8200'; fw_v := 'V4.16.30P4'; END IF;
        CASE i % 6
            WHEN 0 THEN site_v := 'Beijing-CMCC-Site'; lat_v := 39.90 + random()*0.1-0.05; lon_v := 116.40 + random()*0.1-0.05;
            WHEN 1 THEN site_v := 'Shanghai-CMCC-Site'; lat_v := 31.23 + random()*0.1-0.05; lon_v := 121.47 + random()*0.1-0.05;
            WHEN 2 THEN site_v := 'Guangzhou-CMCC-Site'; lat_v := 23.13 + random()*0.1-0.05; lon_v := 113.26 + random()*0.1-0.05;
            WHEN 3 THEN site_v := 'Shenzhen-CMCC-Site'; lat_v := 22.54 + random()*0.1-0.05; lon_v := 114.06 + random()*0.1-0.05;
            WHEN 4 THEN site_v := 'Chengdu-CMCC-Site'; lat_v := 30.57 + random()*0.1-0.05; lon_v := 104.07 + random()*0.1-0.05;
            WHEN 5 THEN site_v := 'Wuhan-CMCC-Site'; lat_v := 30.59 + random()*0.1-0.05; lon_v := 114.31 + random()*0.1-0.05;
        END CASE;
        dev_sn := 'CMCC-' || lpad(i::text, 6, '0');
        dev_id := uuid_generate_v5(ns, 'cmcc-dev-' || i::text);
        INSERT INTO devices (id, serial_number, oui, product_class, manufacturer, model_name, carrier, technology, lifecycle_state, is_online, firmware_version, ip_address, inform_interval, site_name, latitude, longitude, last_inform_at, created_at, updated_at)
        VALUES (dev_id, dev_sn, oui_v, pc_v, mfr_v, model_v, 'cmcc', tech_v, lifecycle_v, online_v, fw_v, ('10.1.' || ((i/256)::int % 256) || '.' || (i%256))::inet, CASE WHEN online_v THEN 300 ELSE 600 END, site_v, lat_v, lon_v,
            CASE
                WHEN online_v THEN NOW()-(random()*INTERVAL '30 minutes')
                WHEN lifecycle_v = 'commissioned' THEN NOW()-(random()*INTERVAL '7 days')
                ELSE NULL
            END,
            NOW()-(random()*INTERVAL '90 days'), NOW()-(random()*INTERVAL '7 days'))
        ON CONFLICT DO NOTHING;
    END LOOP;
    RAISE NOTICE 'CMCC devices: 12000 inserted';
END;
$$;
-- +goose StatementEnd

-- CTCC 设备 (10,000 台)
-- +goose StatementBegin
DO $$
DECLARE
    i INT; dev_id UUID; dev_sn VARCHAR;
    lifecycle_v VARCHAR; online_v BOOLEAN;
    oui_v VARCHAR; pc_v VARCHAR; mfr_v VARCHAR; model_v VARCHAR; tech_v VARCHAR; fw_v VARCHAR;
    site_v VARCHAR; lat_v FLOAT; lon_v FLOAT;
    ns UUID := '6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid;
BEGIN
    FOR i IN 1..10000 LOOP
        IF    i <= 7000 THEN lifecycle_v := 'commissioned'; online_v := TRUE;
        ELSIF i <= 8500 THEN lifecycle_v := 'commissioned'; online_v := FALSE;
        ELSIF i <= 9500 THEN lifecycle_v := 'registered';   online_v := FALSE;
        ELSE                 lifecycle_v := 'maintenance';  online_v := TRUE;
        END IF;
        IF i % 2 = 0 THEN tech_v := 'lte'; oui_v := '001E4F'; pc_v := 'FAP-LTE-200'; mfr_v := 'ZTE'; model_v := 'ZXSDR-B8200'; fw_v := 'V4.16.30P4';
        ELSE tech_v := 'lte'; oui_v := '000DB9'; pc_v := 'FAP-LTE-300'; mfr_v := 'Ericsson'; model_v := 'AIR6488'; fw_v := 'CXP9024418_R57A'; END IF;
        CASE i % 5
            WHEN 0 THEN site_v := 'Nanjing-CTCC-Site'; lat_v := 32.06 + random()*0.1-0.05; lon_v := 118.80 + random()*0.1-0.05;
            WHEN 1 THEN site_v := 'Hangzhou-CTCC-Site'; lat_v := 30.27 + random()*0.1-0.05; lon_v := 120.15 + random()*0.1-0.05;
            WHEN 2 THEN site_v := 'Suzhou-CTCC-Site'; lat_v := 31.30 + random()*0.1-0.05; lon_v := 120.62 + random()*0.1-0.05;
            WHEN 3 THEN site_v := 'Fuzhou-CTCC-Site'; lat_v := 26.07 + random()*0.1-0.05; lon_v := 119.30 + random()*0.1-0.05;
            WHEN 4 THEN site_v := 'Hefei-CTCC-Site'; lat_v := 31.82 + random()*0.1-0.05; lon_v := 117.23 + random()*0.1-0.05;
        END CASE;
        dev_sn := 'CTCC-' || lpad(i::text, 6, '0');
        dev_id := uuid_generate_v5(ns, 'ctcc-dev-' || i::text);
        INSERT INTO devices (id, serial_number, oui, product_class, manufacturer, model_name, carrier, technology, lifecycle_state, is_online, firmware_version, ip_address, inform_interval, site_name, latitude, longitude, last_inform_at, created_at, updated_at)
        VALUES (dev_id, dev_sn, oui_v, pc_v, mfr_v, model_v, 'ctcc', tech_v, lifecycle_v, online_v, fw_v, ('10.2.' || ((i/256)::int % 256) || '.' || (i%256))::inet, CASE WHEN online_v THEN 300 ELSE 600 END, site_v, lat_v, lon_v,
            CASE
                WHEN online_v THEN NOW()-(random()*INTERVAL '30 minutes')
                WHEN lifecycle_v = 'commissioned' THEN NOW()-(random()*INTERVAL '7 days')
                ELSE NULL
            END,
            NOW()-(random()*INTERVAL '90 days'), NOW()-(random()*INTERVAL '7 days'))
        ON CONFLICT DO NOTHING;
    END LOOP;
    RAISE NOTICE 'CTCC devices: 10000 inserted';
END;
$$;
-- +goose StatementEnd

-- CUCC 设备 (8,000 台)
-- +goose StatementBegin
DO $$
DECLARE
    i INT; dev_id UUID; dev_sn VARCHAR;
    lifecycle_v VARCHAR; online_v BOOLEAN;
    oui_v VARCHAR; pc_v VARCHAR; mfr_v VARCHAR; model_v VARCHAR; tech_v VARCHAR; fw_v VARCHAR;
    site_v VARCHAR; lat_v FLOAT; lon_v FLOAT;
    ns UUID := '6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid;
BEGIN
    FOR i IN 1..8000 LOOP
        IF    i <= 5600 THEN lifecycle_v := 'commissioned'; online_v := TRUE;
        ELSIF i <= 6800 THEN lifecycle_v := 'commissioned'; online_v := FALSE;
        ELSIF i <= 7600 THEN lifecycle_v := 'registered';   online_v := FALSE;
        ELSE                 lifecycle_v := 'maintenance';  online_v := TRUE;
        END IF;
        IF i % 2 = 0 THEN tech_v := 'nr'; oui_v := '64700E'; pc_v := 'gNB-200'; mfr_v := 'Ericsson'; model_v := 'AIR6488'; fw_v := 'CXP9024418_R58A';
        ELSE tech_v := 'lte'; oui_v := '58FB96'; pc_v := 'FAP-LTE-100'; mfr_v := 'Comba'; model_v := 'CB-ENB-200'; fw_v := 'V2.5.0'; END IF;
        CASE i % 4
            WHEN 0 THEN site_v := 'Jinan-CUCC-Site'; lat_v := 36.65 + random()*0.1-0.05; lon_v := 117.00 + random()*0.1-0.05;
            WHEN 1 THEN site_v := 'Zhengzhou-CUCC-Site'; lat_v := 34.75 + random()*0.1-0.05; lon_v := 113.65 + random()*0.1-0.05;
            WHEN 2 THEN site_v := 'Qingdao-CUCC-Site'; lat_v := 36.07 + random()*0.1-0.05; lon_v := 120.38 + random()*0.1-0.05;
            WHEN 3 THEN site_v := 'Taiyuan-CUCC-Site'; lat_v := 37.87 + random()*0.1-0.05; lon_v := 112.55 + random()*0.1-0.05;
        END CASE;
        dev_sn := 'CUCC-' || lpad(i::text, 6, '0');
        dev_id := uuid_generate_v5(ns, 'cucc-dev-' || i::text);
        INSERT INTO devices (id, serial_number, oui, product_class, manufacturer, model_name, carrier, technology, lifecycle_state, is_online, firmware_version, ip_address, inform_interval, site_name, latitude, longitude, last_inform_at, created_at, updated_at)
        VALUES (dev_id, dev_sn, oui_v, pc_v, mfr_v, model_v, 'cucc', tech_v, lifecycle_v, online_v, fw_v, ('10.3.' || ((i/256)::int % 256) || '.' || (i%256))::inet, CASE WHEN online_v THEN 300 ELSE 600 END, site_v, lat_v, lon_v,
            CASE
                WHEN online_v THEN NOW()-(random()*INTERVAL '30 minutes')
                WHEN lifecycle_v = 'commissioned' THEN NOW()-(random()*INTERVAL '7 days')
                ELSE NULL
            END,
            NOW()-(random()*INTERVAL '90 days'), NOW()-(random()*INTERVAL '7 days'))
        ON CONFLICT DO NOTHING;
    END LOOP;
    RAISE NOTICE 'CUCC devices: 8000 inserted';
END;
$$;
-- +goose StatementEnd

-- ============================================================
-- 4. 设备分组绑定
-- ============================================================
-- +goose StatementBegin
DO $$
DECLARE
    grp_id UUID; dev_id UUID; batch_cursor REFCURSOR;
    ns UUID := '6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid;
BEGIN
    -- 北京移动: CMCC 000001~002000
    grp_id := uuid_generate_v5(ns, 'seed-group-cmcc-bj');
    FOR dev_id IN SELECT id FROM devices WHERE carrier='cmcc' AND serial_number >= 'CMCC-000001' AND serial_number < 'CMCC-002001' AND deleted_at IS NULL LIMIT 2000 LOOP
        INSERT INTO device_group_members (group_id, device_id, added_at) VALUES (grp_id, dev_id, NOW()-(random()*INTERVAL '30 days')) ON CONFLICT (device_id) DO NOTHING;
    END LOOP;
    -- 上海移动: CMCC 002001~003500
    grp_id := uuid_generate_v5(ns, 'seed-group-cmcc-sh');
    FOR dev_id IN SELECT id FROM devices WHERE carrier='cmcc' AND serial_number >= 'CMCC-002001' AND serial_number < 'CMCC-003501' AND deleted_at IS NULL LIMIT 1500 LOOP
        INSERT INTO device_group_members (group_id, device_id, added_at) VALUES (grp_id, dev_id, NOW()-(random()*INTERVAL '30 days')) ON CONFLICT (device_id) DO NOTHING;
    END LOOP;
    -- 江苏电信: CTCC 000001~001500
    grp_id := uuid_generate_v5(ns, 'seed-group-ctcc-js');
    FOR dev_id IN SELECT id FROM devices WHERE carrier='ctcc' AND serial_number >= 'CTCC-000001' AND serial_number < 'CTCC-001501' AND deleted_at IS NULL LIMIT 1500 LOOP
        INSERT INTO device_group_members (group_id, device_id, added_at) VALUES (grp_id, dev_id, NOW()-(random()*INTERVAL '30 days')) ON CONFLICT (device_id) DO NOTHING;
    END LOOP;
    -- 山东联通: CUCC 000001~001200
    grp_id := uuid_generate_v5(ns, 'seed-group-cucc-sd');
    FOR dev_id IN SELECT id FROM devices WHERE carrier='cucc' AND serial_number >= 'CUCC-000001' AND serial_number < 'CUCC-001201' AND deleted_at IS NULL LIMIT 1200 LOOP
        INSERT INTO device_group_members (group_id, device_id, added_at) VALUES (grp_id, dev_id, NOW()-(random()*INTERVAL '30 days')) ON CONFLICT (device_id) DO NOTHING;
    END LOOP;
    -- 联调测试组: 50 台 CMCC
    grp_id := uuid_generate_v5(ns, 'seed-group-test-compat');
    FOR dev_id IN SELECT id FROM devices WHERE carrier='cmcc' AND serial_number >= 'CMCC-011900' AND deleted_at IS NULL LIMIT 50 LOOP
        INSERT INTO device_group_members (group_id, device_id, added_at) VALUES (grp_id, dev_id, NOW()-(random()*INTERVAL '7 days')) ON CONFLICT (device_id) DO NOTHING;
    END LOOP;
    RAISE NOTICE 'Device group bindings created';
END;
$$;
-- +goose StatementEnd

-- ============================================================
-- 5. 回收站设备 (30 台: 每运营商 10 台)
-- ============================================================
-- +goose StatementBegin
DO $$
DECLARE
    i INT; dev_id UUID; dev_sn VARCHAR;
    oui_v VARCHAR; pc_v VARCHAR; mfr_v VARCHAR; model_v VARCHAR; tech_v VARCHAR;
    ns UUID := '6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid;
BEGIN
    FOR i IN 1..10 LOOP
        dev_sn := 'CMCC-DEL-' || lpad(i::text, 4, '0');
        dev_id := uuid_generate_v5(ns, 'cmcc-del-' || i::text);
        IF i%2=0 THEN tech_v := 'nr'; oui_v := '00A0C6'; pc_v := 'gNB-100'; mfr_v := 'Huawei'; model_v := 'AAU5613';
        ELSE tech_v := 'lte'; oui_v := '001A2B'; pc_v := 'SmallCell-LTE'; mfr_v := 'BaiCells'; model_v := 'BC-ENB-100'; END IF;
        INSERT INTO devices (id, serial_number, oui, product_class, manufacturer, model_name, carrier, technology, lifecycle_state, is_online, firmware_version, ip_address, inform_interval, site_name, latitude, longitude, last_inform_at, deleted_at, deleted_by, created_at, updated_at)
        VALUES (dev_id, dev_sn, oui_v, pc_v, mfr_v, model_v, 'cmcc', tech_v, 'decommissioned', FALSE, 'V1.0.0', ('192.168.1.'||(i+100))::inet, 600, 'Retired-Site', 39.0+i*0.01, 116.0+i*0.01, NOW()-INTERVAL '30 days', NOW()-(random()*INTERVAL '30 days'), 'admin', NOW()-INTERVAL '180 days', NOW()-INTERVAL '30 days')
        ON CONFLICT DO NOTHING;
    END LOOP;
    FOR i IN 1..10 LOOP
        dev_sn := 'CTCC-DEL-' || lpad(i::text, 4, '0');
        dev_id := uuid_generate_v5(ns, 'ctcc-del-' || i::text);
        INSERT INTO devices (id, serial_number, oui, product_class, manufacturer, model_name, carrier, technology, lifecycle_state, is_online, firmware_version, ip_address, inform_interval, site_name, latitude, longitude, last_inform_at, deleted_at, deleted_by, created_at, updated_at)
        VALUES (dev_id, dev_sn, '001E4F', 'FAP-LTE-200', 'ZTE', 'ZXSDR-B8200', 'ctcc', 'lte', 'decommissioned', FALSE, 'V2.0.0', ('192.168.2.'||(i+100))::inet, 600, 'Retired-Site', 31.0+i*0.01, 121.0+i*0.01, NOW()-INTERVAL '30 days', NOW()-(random()*INTERVAL '30 days'), 'admin', NOW()-INTERVAL '180 days', NOW()-INTERVAL '30 days')
        ON CONFLICT DO NOTHING;
    END LOOP;
    FOR i IN 1..10 LOOP
        dev_sn := 'CUCC-DEL-' || lpad(i::text, 4, '0');
        dev_id := uuid_generate_v5(ns, 'cucc-del-' || i::text);
        INSERT INTO devices (id, serial_number, oui, product_class, manufacturer, model_name, carrier, technology, lifecycle_state, is_online, firmware_version, ip_address, inform_interval, site_name, latitude, longitude, last_inform_at, deleted_at, deleted_by, created_at, updated_at)
        VALUES (dev_id, dev_sn, '58FB96', 'FAP-LTE-100', 'Comba', 'CB-ENB-200', 'cucc', 'lte', 'decommissioned', FALSE, 'V1.5.0', ('192.168.3.'||(i+100))::inet, 600, 'Retired-Site', 36.0+i*0.01, 117.0+i*0.01, NOW()-INTERVAL '30 days', NOW()-(random()*INTERVAL '30 days'), 'admin', NOW()-INTERVAL '180 days', NOW()-INTERVAL '30 days')
        ON CONFLICT DO NOTHING;
    END LOOP;
    RAISE NOTICE 'Recycled devices: 30 inserted';
END;
$$;
-- +goose StatementEnd

-- 验证数据量
SELECT 'devices_total' AS entity, COUNT(*) AS count FROM devices
UNION ALL SELECT 'devices_online',  COUNT(*) FROM devices WHERE is_online = TRUE  AND deleted_at IS NULL
UNION ALL SELECT 'devices_offline', COUNT(*) FROM devices WHERE is_online = FALSE AND deleted_at IS NULL
UNION ALL SELECT 'devices_recycled', COUNT(*) FROM devices WHERE deleted_at IS NOT NULL
UNION ALL SELECT 'device_groups_seed', COUNT(*) FROM device_groups WHERE name IN ('移动设备域','电信设备域','联通设备域','测试设备域','运维设备域')
UNION ALL SELECT 'group_members', COUNT(*) FROM device_group_members;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    DELETE FROM device_group_members WHERE group_id IN (
        SELECT id FROM device_groups WHERE name IN ('移动设备域','电信设备域','联通设备域','测试设备域','运维设备域')
    );
    DELETE FROM device_group_members WHERE group_id IN (
        SELECT id FROM device_groups WHERE name IN ('北京移动','上海移动','广东移动','江苏电信','浙江电信','山东联通','河南联通','联调测试组','一致性测试组','巡检设备组')
    );
    DELETE FROM device_groups WHERE name IN ('北京移动','上海移动','广东移动','江苏电信','浙江电信','山东联通','河南联通','联调测试组','一致性测试组','巡检设备组');
    DELETE FROM device_groups WHERE name IN ('移动设备域','电信设备域','联通设备域','测试设备域','运维设备域');
    DELETE FROM devices WHERE serial_number LIKE 'CMCC-%' OR serial_number LIKE 'CTCC-%' OR serial_number LIKE 'CUCC-%';
END;
$$;
-- +goose StatementEnd
