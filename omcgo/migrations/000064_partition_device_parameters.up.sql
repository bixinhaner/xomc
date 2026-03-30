-- ============================================================
-- 000064: device_parameters Hash 分区 + fap_instance/param_group
-- ============================================================

-- Step 1: 重命名旧表
ALTER TABLE device_parameters RENAME TO device_parameters_old;

-- Step 2: 创建分区表
CREATE TABLE device_parameters (
    device_id        UUID         NOT NULL,
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN      DEFAULT false,
    last_updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    fap_instance     SMALLINT     NOT NULL DEFAULT 0,
    param_group      VARCHAR(32)  NOT NULL DEFAULT 'other',
    PRIMARY KEY (device_id, parameter_path)
) PARTITION BY HASH (device_id);

-- Step 3: 创建 32 个分区
DO $$
BEGIN
    FOR i IN 0..31 LOOP
        EXECUTE format(
            'CREATE TABLE device_parameters_p%s PARTITION OF device_parameters
             FOR VALUES WITH (MODULUS 32, REMAINDER %s)',
            lpad(i::text, 2, '0'), i
        );
    END LOOP;
END $$;

-- Step 4: 数据迁移 + 自动分类
INSERT INTO device_parameters (
    device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at,
    fap_instance, param_group
)
SELECT
    device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at,
    -- fap_instance: 从路径提取 FAPService.{N}
    COALESCE(
        (regexp_match(parameter_path, 'FAPService\.(\d+)\.'))[1]::SMALLINT,
        0
    ),
    -- param_group: 按关键字分类（优先级从高到低）
    CASE
        WHEN parameter_path LIKE '%MmePoolConfigParam.%'
          OR parameter_path LIKE '%X_COM_MmePool.%'          THEN 'mme_pool'
        WHEN parameter_path LIKE '%X_COM_LICENSE.%'          THEN 'license'
        WHEN parameter_path LIKE '%AntennaInfo.%'            THEN 'antenna'
        WHEN parameter_path LIKE '%FaultMgmt.CurrentAlarm.%' THEN 'alarm'
        WHEN parameter_path LIKE '%X_COM_GPS%'
          OR parameter_path LIKE '%X_COM_BDS%'
          OR parameter_path LIKE '%X_COM_1588%'
          OR parameter_path LIKE '%X_COM_GLONASS%'
          OR parameter_path LIKE 'Device.FAP.GPS.%'
          OR parameter_path LIKE '%tfcsSync%'
          OR parameter_path LIKE '%tfcsManager%'             THEN 'sync'
        WHEN parameter_path LIKE '%CellConfig%'
         AND (parameter_path LIKE '%RAN.%'
           OR parameter_path LIKE '%EPC.PLMN%')              THEN 'radio'
        WHEN parameter_path LIKE '%FAPControl.%'             THEN 'fap_control'
        WHEN parameter_path LIKE 'Device.DeviceInfo.%'       THEN 'device_info'
        WHEN parameter_path LIKE 'Device.ManagementServer.%' THEN 'management'
        WHEN parameter_path LIKE '%Ipsec%'
          OR parameter_path LIKE '%IPSEC%'                   THEN 'ipsec'
        WHEN parameter_path LIKE 'Device.IP.%'               THEN 'network'
        WHEN parameter_path LIKE 'Device.SoftwareCtrl.%'     THEN 'software'
        ELSE 'other'
    END
FROM device_parameters_old;

-- Step 5: 清理旧表
DROP TABLE device_parameters_old;

-- Step 6: 更新统计信息
ANALYZE device_parameters;
