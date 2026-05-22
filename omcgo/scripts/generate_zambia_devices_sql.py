#!/usr/bin/env python3
"""
将前端 mockDeviceData.ts 转换为 SQL INSERT 语句

数据来源: omcmb/webcode/src/components/GISMap/mockDeviceData.ts
输出: omcgo/migrations/seed/000160_seed_zambia_devices.sql

使用方式:
    python3 scripts/generate_zambia_devices_sql.py > omcgo/migrations/seed/000160_seed_zambia_devices.sql
"""
import re
import json
import sys
from pathlib import Path
from datetime import datetime
from uuid import UUID, uuid4


def parse_serial_number(sn: str) -> str:
    """处理序列号，确保符合 VARCHAR(64) 约束"""
    if not sn:
        return f"ZED-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    return sn[:64]


def parse_oui(sn: str) -> str:
    """从序列号提取 OUI（前6位）"""
    if not sn or len(sn) < 6:
        return "120200"
    return sn[:6]


def parse_product_class(name: str) -> str:
    """从设备名称提取 product_class"""
    if "L700" in name:
        return "LTE700"
    elif "L800" in name:
        return "LTE800"
    elif "L1800" in name:
        return "LTE1800"
    elif "L2100" in name:
        return "LTE2100"
    elif "NR" in name:
        return "NR5G"
    return "LTE700"


def parse_manufacturer(name: str) -> str:
    """从设备名称提取制造商"""
    if "IHS" in name:
        return "IHS"
    elif "Huawei" in name:
        return "Huawei"
    elif "ZTE" in name:
        return "ZTE"
    elif "Ericsson" in name:
        return "Ericsson"
    return "ZED"


def parse_model_name(name: str) -> str:
    """从设备名称提取型号"""
    parts = name.split("_")
    if len(parts) >= 4:
        # 尝试组合型号部分
        model_parts = []
        for part in parts[2:5]:
            if part and not part.startswith("L"):
                model_parts.append(part)
        if model_parts:
            return "_".join(model_parts)[:128]
    return "Unknown"[:128]


def normalize_status(status: str) -> str:
    """规范化设备状态"""
    status_map = {
        "onlineActive": "onlineActive",
        "onlineInactive": "onlineInactive",
        "offline": "offline",
        "discovered": "discovered",
        "active": "onlineActive",
        "registered": "onlineActive",
    }
    return status_map.get(status, "discovered")


def generate_device_uuid(serial_number: str, index: int) -> str:
    """为设备生成固定的 UUID（基于序列号和索引）"""
    import hashlib
    # 使用序列号和索引生成一致的 UUID
    hash_input = f"{serial_number}-{index}".encode()
    hash_bytes = hashlib.md5(hash_input).digest()
    # 转换为 UUID 格式（使用 UUID 构造函数的 bytes 参数）
    return str(UUID(bytes=hash_bytes[:16]))


def parse_mock_ts_file(file_path: str) -> list:
    """解析 TypeScript mock 数据文件"""
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    devices = []

    # 提取 baseMockDevices 数组内容
    # 查找 const baseMockDevices: MapDevice[] = [ 和对应的结束 ];
    pattern = r'const baseMockDevices:\s*MapDevice\[\]\s*=\s*\[(.*?)\];'
    match = re.search(pattern, content, re.DOTALL)

    if not match:
        # 尝试另一种模式
        pattern = r'baseMockDevices\s*=\s*\[(.*?)\];'
        match = re.search(pattern, content, re.DOTALL)

    if not match:
        raise ValueError("未找到 baseMockDevices 数组定义")

    array_content = match.group(1)

    # 解析每个设备对象
    # 使用正则匹配每个对象块
    obj_pattern = r'\{\s*"id":\s*"(.*?)",\s*"name":\s*"(.*?)",\s*"sn":\s*"(.*?)",\s*"lng":\s*([-\d.]+),\s*"lat":\s*([-\d.]+),\s*"status":\s*"(.*?)",\s*"alarmCount":\s*\d+,\s*"groupName":\s*"(.*?)",\s*"address":\s*"(.*?)"'

    for match_obj in re.finditer(obj_pattern, array_content):
        device = {
            'id': match_obj.group(1),
            'name': match_obj.group(2).strip('"'),
            'sn': match_obj.group(3).strip('"'),
            'lng': float(match_obj.group(4)),
            'lat': float(match_obj.group(5)),
            'status': match_obj.group(6).strip('"'),
            'groupName': match_obj.group(7).strip('"'),
            'address': match_obj.group(8).strip('"')
        }
        devices.append(device)

    # 查找扩展设备数据（如果有）
    additional_pattern = r'// 扩展设备.*?const additionalDevices.*?=\s*\[(.*?)\];'
    additional_match = re.search(additional_pattern, content, re.DOTALL)

    if additional_match:
        additional_content = additional_match.group(1)
        for match_obj in re.finditer(obj_pattern, additional_content):
            device = {
                'id': match_obj.group(1),
                'name': match_obj.group(2).strip('"'),
                'sn': match_obj.group(3).strip('"'),
                'lng': float(match_obj.group(4)),
                'lat': float(match_obj.group(5)),
                'status': match_obj.group(6).strip('"'),
                'groupName': match_obj.group(7).strip('"'),
                'address': match_obj.group(8).strip('"')
            }
            devices.append(device)

    return devices


def generate_sql(devices: list, batch_size: int = 100) -> str:
    """生成 SQL INSERT 语句"""

    # 统计信息
    status_count = {}
    for dev in devices:
        status = normalize_status(dev['status'])
        status_count[status] = status_count.get(status, 0) + 1

    lines = [
        "-- +goose Up",
        "-- ============================================================",
        "-- 000160_seed_zambia_devices.up.sql",
        "-- 赞比亚设备种子数据（GIS 地图测试数据）",
        "-- 功能域: F06 拓扑管理",
        "-- 数据来源: GNB 20260210051225.csv + Station_20260210050140.csv",
        f"-- 设备数量: {len(devices)} 条",
        "--",
        "-- 说明：",
        "--   - 使用 carrier='other' 隔离数据，不影响现有 cmcc/ctcc/cucc 设备",
        "--   - 所有 UUID 使用固定值，便于回滚",
        "--   - 使用 ON CONFLICT DO NOTHING 保证幂等性",
        "--   - 设备分布在赞比亚卢萨卡地区",
        f"--   - 生成时间: {datetime.now().isoformat()}",
        "-- ============================================================",
        "",
        "-- ============================================================",
        "-- 1. 创建赞比亚设备组",
        "-- ============================================================",
        "",
        "-- 根组: ZAMBIA (id: zambia-root)",
        "INSERT INTO device_groups (",
        "    id,",
        "    name,",
        "    parent_id,",
        "    carrier,",
        "    level,",
        "    is_default,",
        "    status,",
        "    sort_order,",
        "    created_at,",
        "    updated_at",
        ") VALUES",
        "    (",
        "        'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid,",
        "        'ZAMBIA',",
        "        NULL,",
        "        'other',",
        "        1,",
        "        false,",
        "        'active',",
        "        100,",
        "        NOW(),",
        "        NOW()",
        "    )",
        "ON CONFLICT (id) DO NOTHING;",
        "",
        "-- 子组: LUSAKA (id: zambia-lusaka)",
        "INSERT INTO device_groups (",
        "    id,",
        "    name,",
        "    parent_id,",
        "    carrier,",
        "    level,",
        "    is_default,",
        "    status,",
        "    sort_order,",
        "    created_at,",
        "    updated_at",
        ") VALUES",
        "    (",
        "        'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid,",
        "        'LUSAKA',",
        "        'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid,",
        "        'other',",
        "        2,",
        "        false,",
        "        'active',",
        "        1,",
        "        NOW(),",
        "        NOW()",
        "    )",
        "ON CONFLICT (id) DO NOTHING;",
        "",
        "-- 子组: LTE700 (id: zambia-lte700)",
        "INSERT INTO device_groups (",
        "    id,",
        "    name,",
        "    parent_id,",
        "    carrier,",
        "    level,",
        "    is_default,",
        "    status,",
        "    sort_order,",
        "    created_at,",
        "    updated_at",
        ") VALUES",
        "    (",
        "        'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid,",
        "        'LTE700',",
        "        'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid,",
        "        'other',",
        "        2,",
        "        false,",
        "        'active',",
        "        1,",
        "        NOW(),",
        "        NOW()",
        "    )",
        "ON CONFLICT (id) DO NOTHING;",
        "",
        "-- ============================================================",
        f"-- 2. 插入赞比亚设备数据（{len(devices)}条）",
        "-- ============================================================",
        "",
        "INSERT INTO devices (",
        "    id,",
        "    serial_number,",
        "    oui,",
        "    product_class,",
        "    manufacturer,",
        "    model_name,",
        "    carrier,",
        "    technology,",
        "    status,",
        "    site_name,",
        "    latitude,",
        "    longitude,",
        "    created_at,",
        "    updated_at",
        ") VALUES"
    ]

    # 生成设备数据 INSERT 语句
    for i, dev in enumerate(devices):
        sn = parse_serial_number(dev.get('sn', ''))
        oui = parse_oui(sn)
        product_class = parse_product_class(dev.get('name', ''))
        manufacturer = parse_manufacturer(dev.get('name', ''))
        model_name = parse_model_name(dev.get('name', ''))
        lat = dev.get('lat', 0)
        lng = dev.get('lng', 0)
        status = normalize_status(dev.get('status', 'discovered'))
        site_name = dev.get('name', '')[:128]

        device_uuid = generate_device_uuid(sn, i)

        lines.append(f"    ('{device_uuid}',")
        lines.append(f"     '{sn}',")
        lines.append(f"     '{oui}',")
        lines.append(f"     '{product_class}',")
        lines.append(f"     '{manufacturer}',")
        lines.append(f"     '{model_name}',")
        lines.append(f"     'other',")
        lines.append(f"     'lte',")
        lines.append(f"     '{status}',")
        lines.append(f"     '{site_name}',")
        lines.append(f"     {lat},")
        lines.append(f"     {lng},")
        lines.append(f"     NOW(),")
        lines.append(f"     NOW())")

        if i < len(devices) - 1:
            lines[-1] += ","
        else:
            lines[-1] += " "

    lines.append("ON CONFLICT (serial_number, carrier) DO NOTHING;")
    lines.append("")
    lines.append("-- ============================================================")
    lines.append("-- 3. 关联设备到分组（所有设备关联到 LTE700 组）")
    lines.append("-- ============================================================")
    lines.append("")
    lines.append("INSERT INTO device_group_members (group_id, device_id, added_at)")
    lines.append("SELECT")
    lines.append("    'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid,  -- LTE700 组 ID")
    lines.append("    d.id,")
    lines.append("    NOW()")
    lines.append("FROM devices d")
    lines.append("WHERE d.carrier = 'other'")
    lines.append("    AND d.serial_number LIKE '120200%'")
    lines.append("ON CONFLICT (group_id, device_id) DO NOTHING;")
    lines.append("")
    lines.append("-- ============================================================")
    lines.append("-- 4. 数据统计说明")
    lines.append("-- ============================================================")
    lines.append(f"-- 设备总数: {len(devices)}")
    lines.append("-- 状态分布:")

    for status, count in sorted(status_count.items()):
        lines.append(f"--   - {status}: {count}")

    # 计算经纬度范围
    lats = [dev.get('lat', 0) for dev in devices]
    lngs = [dev.get('lng', 0) for dev in devices]
    lines.append("-- 地理范围: 赞比亚卢萨卡及周边")
    lines.append(f"-- 纬度范围: {min(lats):.6f} ~ {max(lats):.6f}")
    lines.append(f"-- 经度范围: {min(lngs):.6f} ~ {max(lngs):.6f}")
    lines.append("")
    lines.append("-- +goose Down")
    lines.append("-- ============================================================")
    lines.append("-- 回滚种子数据")
    lines.append("-- ============================================================")
    lines.append("")
    lines.append("-- 删除设备分组关联")
    lines.append("DELETE FROM device_group_members")
    lines.append("WHERE group_id = 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid")
    lines.append("    AND device_id IN (")
    lines.append("        SELECT id FROM devices")
    lines.append("        WHERE carrier = 'other'")
    lines.append("            AND serial_number LIKE '120200%'")
    lines.append("    );")
    lines.append("")
    lines.append("-- 删除设备（仅删除 other 分区的测试数据）")
    lines.append("DELETE FROM devices")
    lines.append("WHERE carrier = 'other'")
    lines.append("    AND serial_number LIKE '120200%'")
    lines.append(";")
    lines.append("")
    lines.append("-- 删除设备组（按依赖顺序，从子到父）")
    lines.append("DELETE FROM device_group_members")
    lines.append("WHERE group_id IN (")
    lines.append("    'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid,")
    lines.append("    'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid,")
    lines.append("    'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid")
    lines.append(");")
    lines.append("")
    lines.append("DELETE FROM device_groups")
    lines.append("WHERE id IN (")
    lines.append("    'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid,  -- LTE700")
    lines.append("    'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid,  -- LUSAKA")
    lines.append("    'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid   -- ZAMBIA")
    lines.append(");")

    return "\n".join(lines)


def main():
    # 确定路径
    script_dir = Path(__file__).parent
    project_root = script_dir.parent
    mock_file = project_root / "omcmb" / "webcode" / "src" / "components" / "GISMap" / "mockDeviceData.ts"

    if not mock_file.exists():
        # 尝试其他可能的路径
        mock_file = Path("/Users/hezg/Documents/work/omc-project-code/goomc/omcmb/webcode/src/components/GISMap/mockDeviceData.ts")

    if not mock_file.exists():
        print(f"错误: 找不到 mockDeviceData.ts 文件", file=sys.stderr)
        print(f"尝试的路径: {mock_file}", file=sys.stderr)
        sys.exit(1)

    print(f"正在解析: {mock_file}", file=sys.stderr)

    try:
        devices = parse_mock_ts_file(str(mock_file))
        print(f"找到 {len(devices)} 条设备数据", file=sys.stderr)

        if len(devices) == 0:
            print("警告: 未找到任何设备数据", file=sys.stderr)
            sys.exit(1)

        sql = generate_sql(devices)
        print(sql)

    except Exception as e:
        print(f"错误: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
