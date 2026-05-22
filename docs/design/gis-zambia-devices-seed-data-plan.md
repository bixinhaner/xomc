# GIS 赞比亚设备种子数据方案

> **文档版本**: 1.0
> **创建日期**: 2026-05-22
> **状态**: 设计中
> **关联**: F06 拓扑管理、GIS 地图功能

---

## 1. 问题背景

### 1.1 问题描述

当前 GIS 地图功能的设备数据来自后端 API `/api/v1/devices/geo`。本地开发环境中，赞比亚的设备数据是前端模拟的（mockDeviceData.ts），包含：
- 3720 条原始设备数据（来自 GNB/Station CSV）
- 经纬度坐标（latitude/longitude）
- 设备状态、分组信息

部署到测试/生产环境时，如果数据库中没有对应的赞比亚设备数据，GIS 地图将只显示瓦片底图，没有设备标记点。

### 1.2 数据流向

```
前端 GISMapView.tsx
    ↓ useMapDevicesGeo(filterParams)
    ↓ topologyApi.getDevicesGeo(params)
    ↓ GET /api/v1/devices/geo
    ↓ 后端查询 devices 表 + device_group_members 表
    ↓ 返回 { items: DeviceGeo[], total: number }
    ↓ 前端显示设备标记点
```

### 1.3 无数据时的表现

| 层级 | 状态 | 说明 |
|------|------|------|
| 瓦片底图 | ✅ 正常显示 | 来自 `VITE_MAP_TILE_URL` 配置的瓦片服务 |
| 设备标记点 | ❌ 完全空白 | 数据库无设备记录 |
| 统计面板 | 显示 0 | 设备总数、在线/离线数均为 0 |

---

## 2. 数据库表结构

### 2.1 设备核心表（分区表）

```sql
-- migrations/000003_devices.sql
CREATE TABLE devices (
    id                     UUID NOT NULL DEFAULT gen_random_uuid(),
    serial_number          VARCHAR(64) NOT NULL,
    oui                    VARCHAR(6) NOT NULL,
    product_class          VARCHAR(64),
    manufacturer           VARCHAR(128),
    model_name             VARCHAR(128),
    carrier                VARCHAR(4) NOT NULL,      -- 分区键
    technology             VARCHAR(3) NOT NULL,      -- lte/nr
    data_model_id          UUID,
    status                 VARCHAR(20) NOT NULL DEFAULT 'discovered',
    firmware_version       VARCHAR(64),
    ip_address             INET,
    connection_request_url VARCHAR(256),
    site_name              VARCHAR(128),
    site_id                VARCHAR(64),
    latitude               DOUBLE PRECISION,         -- ⚠️ GIS 必需
    longitude              DOUBLE PRECISION,         -- ⚠️ GIS 必需
    extension_data         JSONB,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, carrier)
) PARTITION BY LIST (carrier);

-- 现有分区
CREATE TABLE devices_cmcc PARTITION OF devices FOR VALUES IN ('cmcc');
CREATE TABLE devices_ctcc PARTITION OF devices FOR VALUES IN ('ctcc');
CREATE TABLE devices_cucc PARTITION OF devices FOR VALUES IN ('cucc');
```

### 2.2 设备分组表

```sql
-- 设备组定义
CREATE TABLE device_groups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(128) NOT NULL,
    parent_id       UUID REFERENCES device_groups(id) ON DELETE CASCADE,
    carrier         VARCHAR(4),
    level           SMALLINT NOT NULL DEFAULT 1,  -- 1=根组, 2=子组
    -- ...
);

-- 设备分组成员关联
CREATE TABLE device_group_members (
    group_id   UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    device_id  UUID NOT NULL,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, device_id)
);
```

### 2.3 关键字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | UUID | ✅ | 主键，使用 `gen_random_uuid()` |
| serial_number | VARCHAR(64) | ✅ | 设备序列号，唯一约束 |
| carrier | VARCHAR(4) | ✅ | 分区键，cmcc/ctcc/cucc |
| technology | VARCHAR(3) | ✅ | lte/nr |
| status | VARCHAR(20) | ✅ | discovered/active/registered/onlineActive/onlineInactive/offline |
| **latitude** | DOUBLE PRECISION | ❌ | **GIS 必需** |
| **longitude** | DOUBLE PRECISION | ❌ | **GIS 必需** |
| site_name | VARCHAR(128) | ❌ | 站点名称 |

---

## 3. 赞比亚设备数据方案

### 3.1 载体选择

由于当前 devices 表按 `carrier` 字段分区（cmcc/ctcc/cucc），而赞比亚不属于中国三大运营商，有两种方案：

| 方案 | 优点 | 缺点 | 推荐度 |
|------|------|------|--------|
| **A. 新增 'other' 分区** | 数据隔离清晰，架构规范 | 需要 DDL 迁移，版本冲突风险 | ⭐⭐⭐⭐ |
| **B. 临时挂靠 cucc** | 无需 DDL，快速部署 | 数据语义不清晰，后续迁移成本高 | ⭐⭐ |

**推荐方案 A**：新增 `devices_other` 分区，确保数据架构的长期可维护性。

### 3.2 文件命名

```
omcgo/migrations/
├── 000159_add_devices_other_partition.sql     -- DDL: 新增分区
└── seed/
    ├── 000159_add_devices_other_partition.sql  -- DDL: 同上（种子目录也放一份）
    └── 000160_seed_zambia_devices.sql         -- DML: 赞比亚设备数据
```

**版本号说明**：
- 当前最大版本：000158
- 000159：新增分区（DDL）
- 000160：赞比亚种子数据（DML）

### 3.3 数据来源

从前端 mock 数据转换：

```typescript
// omcmb/webcode/src/components/GISMap/mockDeviceData.ts
{
  "id": "lte-2",
  "name": "ZED_LUSAKA_IHS_LSK_202A_L700_3",
  "sn": "1202000622241AD0001",
  "lng": 28.318514,
  "lat": -15.403504,
  "status": "onlineActive",
  "groupName": "LUSAKA/LTE700",
  // ...
}
```

---

## 4. SQL 文件模板

### 4.1 分区表迁移（000159）

```sql
-- +goose Up
-- ============================================================
-- 000159_add_devices_other_partition.up.sql
-- 新增 devices_other 分区（支持非中国三大运营商设备）
-- 功能域: F06 拓扑管理
-- ============================================================

-- 创建 other 运营商分区
CREATE TABLE devices_other PARTITION OF devices
FOR VALUES IN ('other');

-- 添加注释
COMMENT ON TABLE devices_other IS '设备表 - 其他运营商分区（非 cmcc/ctcc/cucc）';

-- +goose Down
-- ============================================================
-- 回滚迁移
-- ============================================================

DROP TABLE IF EXISTS devices_other;
```

### 4.2 种子数据文件（000160）

```sql
-- +goose Up
-- ============================================================
-- 000160_seed_zambia_devices.up.sql
-- 赞比亚设备种子数据（GIS 地图测试数据）
-- 功能域: F06 拓扑管理
-- 数据来源: GNB 20260210051225.csv + Station_20260210050140.csv
-- 设备数量: 3720 条原始数据 + 36280 条扩展数据 = 40000 条
-- ============================================================

-- 1. 创建赞比亚设备组
INSERT INTO device_groups (
    id,
    name,
    parent_id,
    carrier,
    level,
    is_default,
    status,
    created_at,
    updated_at
) VALUES
    (
        'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid,  -- 赞比亚根组 ID
        'ZAMBIA',
        NULL,
        'other',
        1,
        false,
        'active',
        NOW(),
        NOW()
    ),
    (
        'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid,  -- LUSAKA 子组 ID
        'LUSAKA',
        'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid,
        'other',
        2,
        false,
        'active',
        NOW(),
        NOW()
    ),
    (
        'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid,  -- LTE700 子组 ID
        'LTE700',
        'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid,
        'other',
        2,
        false,
        'active',
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- 2. 插入赞比亚设备数据（示例 5 条，完整 40000 条通过脚本生成）
INSERT INTO devices (
    id,
    serial_number,
    oui,
    product_class,
    manufacturer,
    model_name,
    carrier,
    technology,
    status,
    site_name,
    latitude,
    longitude,
    created_at,
    updated_at
) VALUES
    (
        gen_random_uuid(),
        '1202000622241AD0001',
        '120200',  -- OUI 从 SN 提取
        'LTE700',
        'ZED',
        'IHS_LSK_202A',
        'other',
        'lte',
        'onlineActive',
        'ZED_LUSAKA_IHS_LSK_202A_L700_3',
        -15.403504,  -- lat
        28.318514,   -- lng
        NOW(),
        NOW()
    ),
    (
        gen_random_uuid(),
        '1202000622241AD0002',
        '120200',
        'LTE700',
        'ZED',
        'IHS_LSK_102M',
        'other',
        'lte',
        'onlineActive',
        'ZED_LUSAKA_IHS_LSK_102M_L700_3',
        -15.513398,
        28.235571,
        NOW(),
        NOW()
    ),
    (
        gen_random_uuid(),
        '1202000622241AD0003',
        '120200',
        'LTE700',
        'ZED',
        'LKP0084',
        'other',
        'lte',
        'onlineActive',
        'ZED_LUSAKA_LKP0084_L700_2',
        -15.29582,
        28.421939,
        NOW(),
        NOW()
    ),
    (
        gen_random_uuid(),
        '1202000622241AD0004',
        '120200',
        'LTE700',
        'ZED',
        'IHS_LSK_203A',
        'other',
        'lte',
        'offline',
        'ZED_LUSAKA_IHS_LSK_203A_L700_3',
        -15.4500,
        28.3000,
        NOW(),
        NOW()
    ),
    (
        gen_random_uuid(),
        '1202000622241AD0005',
        '120200',
        'LTE700',
        'ZED',
        'IHS_LSK_301A',
        'other',
        'lte',
        'onlineInactive',
        'ZED_LUSAKA_IHS_LSK_301A_L700_3',
        -15.3800,
        28.2500,
        NOW(),
        NOW()
    )
ON CONFLICT (serial_number, carrier) DO NOTHING;

-- 3. 关联设备到分组（假设所有设备都关联到 LTE700 组）
-- 实际部署时需要根据设备名称/站点动态分组
INSERT INTO device_group_members (group_id, device_id, added_at)
SELECT
    'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid,  -- LTE700 组 ID
    d.id,
    NOW()
FROM devices d
WHERE d.carrier = 'other'
    AND d.serial_number LIKE '120200%'
ON CONFLICT (group_id, device_id) DO NOTHING;

-- +goose Down
-- ============================================================
-- 回滚种子数据
-- ============================================================

-- 删除设备分组关联
DELETE FROM device_group_members
WHERE group_id = 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid;

-- 删除设备（仅删除 other 分区的测试数据）
DELETE FROM devices
WHERE carrier = 'other'
    AND serial_number LIKE '120200%';

-- 删除设备组
DELETE FROM device_groups
WHERE id IN (
    'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid,  -- LTE700
    'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid,  -- LUSAKA
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid   -- ZAMBIA
);
```

---

## 5. 数据生成脚本

### 5.1 Python 转换脚本

```python
#!/usr/bin/env python3
"""
将前端 mockDeviceData.ts 转换为 SQL INSERT 语句
"""
import json
import re
from pathlib import Path

def parse_mock_ts(file_path: str) -> list:
    """解析 TypeScript mock 数据文件"""
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 提取 baseMockDevices 数组
    match = re.search(r'const baseMockDevices: MapDevice\[\] = \[(.*?)\];', content, re.DOTALL)
    if not match:
        raise ValueError("未找到 baseMockDevices 数组")
    
    # 简单解析（实际可用正则或 ast 解析）
    devices = []
    for line in match.group(1).split('\n'):
        if '{' in line:
            devices.append({})
    
    return devices

def generate_sql_insert(devices: list) -> str:
    """生成 SQL INSERT 语句"""
    lines = []
    for dev in devices:
        sn = dev.get('sn', '')
        oui = sn[:6] if len(sn) >= 6 else '000000'
        lat = dev.get('lat', 0)
        lng = dev.get('lng', 0)
        
        lines.append(f"""
    (
        gen_random_uuid(),
        '{sn}',
        '{oui}',
        'LTE700',
        'ZED',
        '{dev.get('name', '')[:50]}',
        'other',
        'lte',
        '{dev.get('status', 'discovered')}',
        '{dev.get('name', '')[:128]}',
        {lat},
        {lng},
        NOW(),
        NOW()
    ),""")
    
    return '\n'.join(lines)

if __name__ == '__main__':
    mock_file = Path(__file__).parent.parent / 'omcmb/webcode/src/components/GISMap/mockDeviceData.ts'
    devices = parse_mock_ts(mock_file)
    sql = generate_sql_insert(devices)
    print(sql)
```

### 5.2 使用方式

```bash
# 1. 导出完整 SQL
cd /path/to/goomc
python3 scripts/generate_zambia_devices_sql.py > omcgo/migrations/seed/000160_seed_zambia_devices.sql

# 2. 手动编辑补充设备组定义

# 3. 执行迁移
cd omcgo
make migrate-up
goose -dir migrations/seed up
```

---

## 6. 执行验证

### 6.1 数据验证 SQL

```sql
-- 验证设备组
SELECT id, name, parent_id, carrier, level
FROM device_groups
WHERE carrier = 'other'
ORDER BY level, name;

-- 验证设备数量
SELECT COUNT(*) as device_count
FROM devices
WHERE carrier = 'other';

-- 验证设备分组关联
SELECT dg.name, COUNT(dgm.device_id) as member_count
FROM device_groups dg
LEFT JOIN device_group_members dgm ON dg.id = dgm.group_id
WHERE dg.carrier = 'other'
GROUP BY dg.id, dg.name;

-- 验证经纬度数据
SELECT serial_number, site_name, latitude, longitude, status
FROM devices
WHERE carrier = 'other'
ORDER BY site_name
LIMIT 10;

-- 验证地图统计接口
SELECT COUNT(*) as total,
    SUM(CASE WHEN status = 'onlineActive' THEN 1 ELSE 0 END) as online_active,
    SUM(CASE WHEN status = 'onlineInactive' THEN 1 ELSE 0 END) as online_inactive,
    SUM(CASE WHEN status = 'offline' THEN 1 ELSE 0 END) as offline
FROM devices
WHERE carrier = 'other';
```

### 6.2 前端验证

1. 启动前端：`cd omcmb/webcode && npm run dev`
2. 访问 GIS 地图页面
3. 验证：
   - [ ] 地图显示赞比亚区域（卢萨卡）
   - [ ] 设备标记点正确显示
   - [ ] 统计面板显示正确数量
   - [ ] 设备组筛选正常工作
   - [ ] 搜索功能正常

---

## 7. 后续优化

### 7.1 短期（当前 Sprint）

- [ ] 创建 000159/000160 迁移文件
- [ ] 执行迁移并验证
- [ ] 更新前端 mock 数据开关逻辑

### 7.2 中期（下个 Sprint）

- [ ] 考虑增加 `carrier` 枚举校验
- [ ] 新增 `devices_other` 的索引优化
- [ ] GIS 地图支持按 carrier 筛选

### 7.3 长期

- [ ] 支持多国家/多运营商设备
- [ ] 设备组支持按地理位置自动分组
- [ ] GIS 地图支持设备热力图

---

## 8. 参考资料

- [数据库迁移规范](../../omcgo/CLAUDE.md#55-数据库迁移规范)
- [种子数据示例](../../omcgo/migrations/seed/000005_seed_mml_param_library.sql)
- [前端 GIS 组件](../../omcmb/webcode/src/components/GISMap/)
- [设备表 DDL](../../omcgo/migrations/000003_devices.sql)

---

**文档结束**
