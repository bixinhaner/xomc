#!/usr/bin/env python3
"""
解析中国移动TD-LTE皮站南向数据模型Excel规范，按目录索引分组，
对每个分组内的参数path进行智能拆分，标注读写权限，生成结构化Markdown文档。
"""

import openpyxl
import re
import sys
from collections import OrderedDict, defaultdict

EXCEL_PATH = "/Users/cb/code/baicells/goomc/omcgo/规范/移动/南向数据模型/中国移动TD-LTE皮站_飞站基站设备网络管理南向接口数据配置模型规范V2.3.xlsx"
OUTPUT_PATH = "/Users/cb/code/baicells/goomc/omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md"

# ============================================================================
# 1. 解析 Excel
# ============================================================================

wb = openpyxl.load_workbook(EXCEL_PATH, data_only=True)

# --- 目录映射 ---
toc_map = OrderedDict()  # {index: {name, desc, counts}}
toc_ws = wb["目录"]
for row in toc_ws.iter_rows(min_row=2, max_row=toc_ws.max_row, values_only=True):
    idx = row[0]
    if idx is None or idx == "合计":
        continue
    idx = str(idx).strip()
    toc_map[idx] = {
        "name": str(row[1]).strip() if row[1] else "",
        "desc": str(row[2]).strip() if row[2] else "",
    }

print(f"[INFO] 目录索引: {list(toc_map.keys())}")

# --- 数字实例编号归一化 ---
# 在 TR-069 多实例对象路径中，数字编号（如 .1. .2. .3.）是 {i} 的具体实例
# 需要归一化为 {i} 以便正确分组
MULTI_INSTANCE_OBJECTS = {
    "A1MeasureCtrl", "A2MeasureCtrl", "A3MeasureCtrl",
    "A4MeasureCtrl", "A5MeasureCtrl", "B1MeasureCtrl", "B2MeasureCtrl",
    "PeriodMeasCtrl", "Carrier", "UTRANFDDFreq",
    "LTECell", "InterRATCell", "GSM", "UMTS", "NRCell",
    "SupportedAlarm", "CurrentAlarm", "ExpeditedEvent", "HistoryEvent", "QueuedEvent",
    "MmePoolConfigParam", "X2IpAddrMapInfo", "PLMNList",
    "S1U", "PdcpInitParam", "Assoc", "Config",
    "SFConfigList", "DrxInitialParam",
    "FAPService", "Interface", "WANDevice", "WANConnectionDevice", "WANIPConnection",
    "MU", "Slot", "EU", "RU", "RFChannel",
    "GERANFreqGroup",
    "Ethernet",
}

def normalize_numeric_instances(path):
    """
    Normalize numeric instance indices in TR-069 paths to {i}.
    e.g., A1MeasureCtrl.1.Enable -> A1MeasureCtrl.{i}.Enable
    Only normalizes numbers that follow known multi-instance object names.
    """
    segments = path.split(".")
    result = []
    for idx, seg in enumerate(segments):
        if idx > 0 and segments[idx - 1] in MULTI_INSTANCE_OBJECTS:
            # Check if this segment is a pure numeric instance index
            if re.match(r'^\d+$', seg):
                result.append("{i}")
            else:
                result.append(seg)
        else:
            result.append(seg)
    return ".".join(result)

# --- 多实例参数辅助表 ---
multi_instance_info = {}  # tr098_prefix -> {cn_name, rw, tr181_prefix}
mi_ws = wb["多实例参数Object个数读写属性"]
for row in mi_ws.iter_rows(min_row=2, max_row=mi_ws.max_row, values_only=True):
    tr098 = str(row[1]).strip() if row[1] else ""
    tr181 = str(row[2]).strip() if row[2] else ""
    cn_name = str(row[3]).strip() if row[3] else ""
    rw = str(row[4]).strip() if row[4] else ""
    if not tr098 or "备注" in tr098 or "FAPService.{i}；其中i=" in tr098:
        continue
    # Normalize: strip trailing dots
    key = tr098.rstrip(".")
    multi_instance_info[key] = {
        "cn_name": cn_name,
        "rw": rw,
        "tr181": tr181.rstrip("."),
    }

print(f"[INFO] 多实例参数: {len(multi_instance_info)} 条")

# --- 提取每个数据 Sheet 的参数 ---
DATA_SHEETS = ["SA", "SB", "SC", "SD", "SE", "SF", "SG", "SH", "SI", "SJ", "SK", "SL", "SM", "SN", "SO", "SP", "SQ", "SR"]

all_params = []  # [{sheet, tr098, tr181, name, cn, rw, type, desc, category}]

# Track seen (sheet, tr181_normalized) to deduplicate after numeric instance normalization
# Priority: keep the row whose original TR181 path already contains {i} (template definition)
# If no template row exists, keep the first occurrence
seen_tr181_per_sheet = {}  # {(sheet, tr181_norm): index_in_all_params}

for sheet_name in DATA_SHEETS:
    ws = wb[sheet_name]
    current_category = ""
    for row in ws.iter_rows(min_row=2, max_row=ws.max_row, values_only=True):
        # Columns: 0=归类, 1=TR098, 2=TR181, 3=参数名称, 4=中文名, 5=重要性, 8=读写权限, 9=类型, 10=描述
        vals = [str(v).strip() if v else "" for v in row]
        if len(vals) < 11:
            continue
        category = vals[0]
        tr098 = vals[1]
        tr181 = vals[2]
        param_name = vals[3]
        cn_name = vals[4]
        rw = vals[8]
        ptype = vals[9]
        desc = vals[10]

        # Skip empty / header / note rows
        if not tr098 and not param_name:
            if category:
                current_category = category
            continue
        if not tr098 or tr098 in ("网管数据模型(TR098)", "网管数据模型(TR181)"):
            continue
        # Skip pure note rows
        if "备注" in tr098 or "说明" in tr098 or "EU.{i}：" in tr098:
            continue

        if category:
            current_category = category

        # Clean up TR098 path - remove trailing dots
        tr098 = tr098.rstrip(".")
        tr181 = tr181.rstrip(".")

        # Skip if tr098 is empty after cleanup
        if not tr098:
            continue

        # Normalize numeric instance indices to {i}
        # e.g., A1MeasureCtrl.1.Enable -> A1MeasureCtrl.{i}.Enable
        tr098_orig = tr098
        tr181_orig = tr181
        tr098 = normalize_numeric_instances(tr098)
        tr181 = normalize_numeric_instances(tr181)

        # Deduplicate: if same normalized TR181 path already exists in this sheet
        dedup_key = (sheet_name, tr181)
        has_i_original = "{i}" in tr181_orig  # Was the original already a template?

        if dedup_key in seen_tr181_per_sheet:
            prev_idx = seen_tr181_per_sheet[dedup_key]
            prev_param = all_params[prev_idx]
            # If current row is the template ({i}) and previous was a concrete instance, replace
            if has_i_original and "{i}" not in prev_param.get("_tr181_orig", ""):
                all_params[prev_idx] = {
                    "sheet": sheet_name,
                    "tr098": tr098,
                    "tr181": tr181,
                    "name": param_name,
                    "cn": cn_name,
                    "rw": rw,
                    "type": ptype,
                    "desc": desc,
                    "category": current_category,
                    "_tr181_orig": tr181_orig,
                }
            # Otherwise skip duplicate
            continue

        param_entry = {
            "sheet": sheet_name,
            "tr098": tr098,
            "tr181": tr181,
            "name": param_name,
            "cn": cn_name,
            "rw": rw,
            "type": ptype,
            "desc": desc,
            "category": current_category,
            "_tr181_orig": tr181_orig,
        }
        seen_tr181_per_sheet[dedup_key] = len(all_params)
        all_params.append(param_entry)

# Clean up internal field
for p in all_params:
    p.pop("_tr181_orig", None)

print(f"[INFO] 总参数行: {len(all_params)}")

# Group params by sheet
params_by_sheet = defaultdict(list)
for p in all_params:
    params_by_sheet[p["sheet"]].append(p)

# ============================================================================
# 2. 智能分组算法
# ============================================================================

def get_tr181_depth_prefix(tr181, depth):
    """Extract prefix from TR181 path at given depth (number of segments)."""
    segments = tr181.split(".")
    return ".".join(segments[:depth])


def is_object_path(path):
    """Check if path ends with {i} (object node, not a leaf parameter)."""
    return path.endswith(".{i}") or path.endswith(".{i}")


def count_i_params(path):
    """Count the number of {i} in a path."""
    return path.count("{i}")


def get_command_prefix(path):
    """
    Given a full TR181 parameter path, determine the command prefix.
    Rules:
    - If path has {i}, include up to and including the {i} segment's parent that groups related params
    - If path has SwUpgrade as a sub-segment, split at SwUpgrade level
    - Otherwise, group at the deepest common parent that has multiple children
    """
    segments = path.split(".")
    # Find {i} positions
    i_positions = [idx for idx, seg in enumerate(segments) if seg == "{i}"]

    # Find the "leaf grouping" level - the parent of the leaf parameter
    # But need to handle special sub-objects like SwUpgrade, Slot, EU, RU, RFChannel
    special_subobjects = {"SwUpgrade", "Slot", "EU", "RU", "RFChannel", "Capabilities",
                          "MBSFN", "PHY", "MAC", "RLC", "PDCP", "RRC", "S1",
                          "EPC", "S1U", "VoLTE", "SelfConfig", "SONConfigParam",
                          "NeighborList", "LTECell", "InterRATCell", "UMTS", "GSM",
                          "Mobility", "ConnMode", "IdleMode", "EUTRA", "IRAT",
                          "MeasureCtrl", "Carrier", "UTRA", "UTRANFDDFreq",
                          "MmePoolConfigParam", "X2IpAddrMapInfo", "PLMNList",
                          "PdcpInitParam", "Assoc", "Config", "SupportedAlarm",
                          "CurrentAlarm", "ExpeditedEvent", "HistoryEvent",
                          "QueuedEvent", "SFConfigList", "DrxInitialParam",
                          "A1MeasureCtrl", "A2MeasureCtrl", "A3MeasureCtrl",
                          "A4MeasureCtrl", "A5MeasureCtrl", "B1MeasureCtrl",
                          "B2MeasureCtrl", "PeriodMeasCtrl", "FaultMgmt",
                          "MU"}

    # Build prefix - go segment by segment
    # The command prefix should be at the level that groups multiple related params
    # but not so deep that it's just one param
    prefix_segments = []

    for idx, seg in enumerate(segments):
        prefix_segments.append(seg)
        # After {i}, continue to include the next fixed segment (the object type)
        if seg == "{i}" and idx + 1 < len(segments):
            # Check if next segment is a special sub-object
            next_seg = segments[idx + 1]
            if next_seg in special_subobjects:
                # Include the sub-object name but stop there
                # The command will be at this sub-object level
                pass  # Continue to let the loop add more segments
            # else: stop after {i} - the command is at the {i} level
            continue

        # If this is a special sub-object (not {i}), include it
        if seg in special_subobjects:
            continue

    # Actually, let's use a simpler approach: just use the TR181 path data to determine grouping
    # We'll analyze all paths in a sheet and find natural grouping boundaries
    return None  # Placeholder


def build_commands_for_sheet(sheet_name, params):
    """
    Build command groups for a sheet's parameters.
    Uses TR181 paths for consistent grouping.
    Implements the "depth-first claim" algorithm from learned skills.
    """
    if not params:
        return []

    # Step 1: Determine the command prefix for each parameter
    # Group by the "command base path" - the deepest common ancestor
    # that groups multiple related params, respecting special sub-objects

    # First, for each param, compute all possible prefix depths
    path_groups = defaultdict(list)  # prefix -> [param_indices]

    for idx, p in enumerate(params):
        tr181 = p["tr181"]
        segments = tr181.split(".")
        # Try different prefix depths
        for depth in range(2, len(segments) + 1):
            prefix = ".".join(segments[:depth])
            path_groups[prefix].append(idx)

    # Step 2: Determine optimal grouping
    # For each path, find the most specific (deepest) prefix that still groups
    # at least 2 params, OR use the leaf parent if it's a standalone param

    # Better approach: analyze path hierarchy and find natural break points
    # Build a tree of all paths

    # Group params by their "command base" using these rules:
    # 1. {i} segments are grouping boundaries - include them in the command prefix
    # 2. Special sub-objects (SwUpgrade, Slot, EU, RU, etc.) are grouping boundaries
    # 3. At leaf level, group params under their immediate parent

    commands = OrderedDict()  # command_key -> {prefix, params: []}

    for idx, p in enumerate(params):
        tr181 = p["tr181"]
        segments = tr181.split(".")

        # Determine the command prefix for this path
        # Walk the path and find the natural grouping point
        cmd_segments = []
        i_count = 0
        found_grouping_point = False

        for seg_idx, seg in enumerate(segments):
            if seg == "{i}":
                i_count += 1
                cmd_segments.append(seg)
                # After {i}, we need to include the next segment(s) that define the object type
                # Continue to next segment(s)
                continue

            cmd_segments.append(seg)

            # Check if this is a special sub-object that should be a separate command
            # Rule: SwUpgrade, Slot, EU, RU, RFChannel, etc. create sub-commands
            special_objects = {
                "SwUpgrade", "Slot", "EU", "RU", "RFChannel",
                "MBSFN", "PHY", "MAC", "RLC", "PDCP", "RRC",
                "S1", "EPC", "S1U", "VoLTE",
                "SelfConfig", "SONConfigParam",
                "NeighborList", "LTECell", "InterRATCell", "UMTS", "GSM", "NRCell",
                "Mobility", "ConnMode", "IdleMode", "EUTRA", "IRAT",
                "A1MeasureCtrl", "A2MeasureCtrl", "A3MeasureCtrl",
                "A4MeasureCtrl", "A5MeasureCtrl", "B1MeasureCtrl", "B2MeasureCtrl",
                "PeriodMeasCtrl", "Carrier", "UTRANFDDFreq",
                "MmePoolConfigParam", "X2IpAddrMapInfo", "PLMNList",
                "PdcpInitParam", "Assoc",
                "SupportedAlarm", "CurrentAlarm", "ExpeditedEvent", "HistoryEvent", "QueuedEvent",
                "SFConfigList", "DrxInitialParam",
                "Config",
                "Capabilities",
                "FaultMgmt",
                "MU",
            }

            if seg in special_objects and seg_idx > 0:
                # This segment defines a command boundary
                # The command prefix includes this segment
                found_grouping_point = True
                # Don't break - we might have deeper nesting
                # But for the command key, use up to this point
                # Actually, we need to check if there are deeper special objects
                # We'll handle this by processing deepest first

        # The command key is the full prefix up to the deepest special object or {i}
        # But we need the MINIMAL grouping - not too deep, not too shallow

        # Let's use a different strategy: compute the command key as the path
        # up to (and including) the LAST special sub-object or {i} that appears
        # BEFORE the leaf parameter name

        cmd_key = _compute_command_key(tr181, segments)
        if cmd_key not in commands:
            commands[cmd_key] = {"prefix": cmd_key, "params": [], "i_count": count_i_params(cmd_key)}
        commands[cmd_key]["params"].append(idx)

    # Step 3: Depth-first claim algorithm for parent-child dedup
    # Sort command keys by depth (deepest first)
    sorted_keys = sorted(commands.keys(), key=lambda k: (k.count("."), k), reverse=True)

    claimed = set()  # Set of param indices claimed by deeper commands
    result_commands = []

    for cmd_key in sorted_keys:
        cmd = commands[cmd_key]
        unclaimed_params = [i for i in cmd["params"] if i not in claimed]
        if unclaimed_params:
            result_commands.append({
                "prefix": cmd_key,
                "params": unclaimed_params,
                "i_count": count_i_params(cmd_key),
            })
            claimed.update(unclaimed_params)

    # Sort result by prefix for readability
    result_commands.sort(key=lambda c: c["prefix"])

    # Verify: every param is claimed exactly once
    all_claimed = set()
    for cmd in result_commands:
        for i in cmd["params"]:
            assert i not in all_claimed, f"Param {i} claimed multiple times!"
            all_claimed.add(i)
    assert all_claimed == set(range(len(params))), "Some params not claimed!"

    return result_commands


def _compute_command_key(tr181, segments):
    """
    Compute the command key (grouping prefix) for a TR181 path.
    Rules:
    - Include {i} segments in the prefix
    - Split at special sub-object boundaries (SwUpgrade, Slot, EU, RU, etc.)
    - Device.DeviceInfo.* does NOT include Device.DeviceInfo.SwUpgrade.*
    - Device.DeviceInfo.MU.{i}.* does NOT include Device.DeviceInfo.MU.{i}.Slot.{i}.*
    """
    # Define special sub-objects that create command boundaries
    special_objects = {
        "SwUpgrade", "Slot", "EU", "RU", "RFChannel",
        "MBSFN", "PHY", "MAC", "RLC", "PDCP", "RRC",
        "S1", "EPC", "S1U", "VoLTE",
        "SelfConfig", "SONConfigParam",
        "NeighborList", "LTECell", "InterRATCell", "UMTS", "GSM", "NRCell",
        "Mobility", "ConnMode", "IdleMode", "EUTRA", "IRAT",
        "A1MeasureCtrl", "A2MeasureCtrl", "A3MeasureCtrl",
        "A4MeasureCtrl", "A5MeasureCtrl", "B1MeasureCtrl", "B2MeasureCtrl",
        "PeriodMeasCtrl", "Carrier", "UTRANFDDFreq",
        "MmePoolConfigParam", "X2IpAddrMapInfo", "PLMNList",
        "PdcpInitParam", "Assoc",
        "SupportedAlarm", "CurrentAlarm", "ExpeditedEvent", "HistoryEvent", "QueuedEvent",
        "SFConfigList", "DrxInitialParam",
        "Config",
        "Capabilities",
        "FaultMgmt",
        "MU",
    }

    # Walk segments and find the deepest special sub-object or {i}
    # that defines the command boundary
    # The command key = prefix up to and including the deepest boundary segment

    last_boundary_idx = -1

    for idx, seg in enumerate(segments):
        if seg == "{i}":
            last_boundary_idx = idx
        elif seg in special_objects and idx < len(segments) - 1:
            # Only consider it a boundary if it's not the leaf
            last_boundary_idx = idx

    if last_boundary_idx >= 0:
        # Command key includes up to and including the last boundary segment
        cmd_key = ".".join(segments[:last_boundary_idx + 1])
    else:
        # No special boundaries - group at the parent of the leaf
        # e.g., Device.DeviceInfo.ModelName -> Device.DeviceInfo
        cmd_key = ".".join(segments[:-1])

    return cmd_key


# ============================================================================
# 3. {i} 取值范围推断
# ============================================================================

def infer_i_ranges(command_prefix, params, all_sheet_params):
    """
    Infer the range for each {i} in the command prefix based on
    TR-069 protocol knowledge and the multi-instance info sheet.
    """
    i_count = command_prefix.count("{i}")
    if i_count == 0:
        return []

    ranges = []
    # Find the {i} positions in the prefix
    segments = command_prefix.split(".")
    i_positions = [idx for idx, seg in enumerate(segments) if seg == "{i}"]

    # Greek letter labels for multi-{i} commands
    greek_labels = ["α", "β", "γ", "δ", "ε", "ζ", "η", "θ"]

    for i_idx, pos in enumerate(i_positions):
        # Get the context around this {i}
        # The segment before {i} indicates the object type
        parent_seg = segments[pos - 1] if pos > 0 else ""

        label = f"i{greek_labels[i_idx]}" if i_count > 1 else "i"
        range_info = _get_range_for_object(parent_seg, pos, segments, i_idx, i_count, label)
        ranges.append(range_info)

    return ranges


def _get_range_for_object(parent_seg, pos, segments, i_idx, total_i_count, label):
    """Get the range info for a specific {i} based on its parent object type.
    label is pre-computed by the caller using Greek letter labels for multi-{i} commands.
    """

    # Based on TR-069 standard and the multi-instance sheet
    range_map = {
        "FAPService": {
            "range": "1~3",
            "desc": "载波实例（i=1:载波1, i=2:载波2, i=3:载波3）",
        },
        "LTECell": {
            "range": "0~N",
            "desc": "LTE邻区实例编号，N由LTECellNumberOfEntries决定",
        },
        "UMTS": {
            "range": "0~N",
            "desc": "UMTS邻区实例编号，N由对应NumberOfEntries决定",
        },
        "GSM": {
            "range": "0~N",
            "desc": "GSM邻区实例编号，N由对应NumberOfEntries决定",
        },
        "NRCell": {
            "range": "0~N",
            "desc": "NR邻区实例编号，N由对应NumberOfEntries决定",
        },
        "SupportedAlarm": {
            "range": "0~N",
            "desc": "告警实例编号，N由SupportedAlarmNumberOfEntries决定",
        },
        "CurrentAlarm": {
            "range": "0~N",
            "desc": "当前告警实例编号，N由MaxCurrentAlarmEntries决定",
        },
        "ExpeditedEvent": {
            "range": "0~N",
            "desc": "实时告警实例编号",
        },
        "HistoryEvent": {
            "range": "0~N",
            "desc": "历史告警实例编号，N由HistoryEventNumberOfEntries决定",
        },
        "QueuedEvent": {
            "range": "0~N",
            "desc": "队列告警实例编号",
        },
        "Config": {
            "range": "0~N",
            "desc": "配置实例编号，N由对应NumberOfEntries决定",
        },
        "Assoc": {
            "range": "0~N",
            "desc": "SCTP关联实例编号，N由AssocNumberOfEntries决定",
        },
        "MmePoolConfigParam": {
            "range": "0~N",
            "desc": "MME池配置实例编号，N由对应NumberOfEntries决定",
        },
        "X2IpAddrMapInfo": {
            "range": "0~N",
            "desc": "X2接口IP映射实例编号",
        },
        "PLMNList": {
            "range": "0~N",
            "desc": "PLMN列表实例编号，N由PLMNListNumberOfEntries决定",
        },
        "S1U": {
            "range": "0~N",
            "desc": "S1U实例编号",
        },
        "PdcpInitParam": {
            "range": "0~N",
            "desc": "PDCP初始参数实例编号",
        },
        "SFConfigList": {
            "range": "0~N",
            "desc": "子帧配置实例编号",
        },
        "DrxInitialParam": {
            "range": "0~N",
            "desc": "DRX初始参数实例编号",
        },
        "A1MeasureCtrl": {"range": "0~N", "desc": "A1测量控制实例编号"},
        "A2MeasureCtrl": {"range": "0~N", "desc": "A2测量控制实例编号"},
        "A3MeasureCtrl": {"range": "0~N", "desc": "A3测量控制实例编号"},
        "A4MeasureCtrl": {"range": "0~N", "desc": "A4测量控制实例编号"},
        "A5MeasureCtrl": {"range": "0~N", "desc": "A5测量控制实例编号"},
        "B1MeasureCtrl": {"range": "0~N", "desc": "B1测量控制实例编号"},
        "B2MeasureCtrl": {"range": "0~N", "desc": "B2测量控制实例编号"},
        "PeriodMeasCtrl": {"range": "0~N", "desc": "周期性测量控制实例编号"},
        "Carrier": {"range": "0~N", "desc": "载波实例编号"},
        "UTRANFDDFreq": {"range": "0~N", "desc": "UTRA频点实例编号"},
        "Interface": {"range": "0~N", "desc": "以太网接口实例编号，N由InterfaceNumberOfEntries决定"},
        "WANDevice": {"range": "0~N", "desc": "WAN设备实例编号"},
        "WANConnectionDevice": {"range": "0~N", "desc": "WAN连接设备实例编号"},
        "WANIPConnection": {"range": "0~N", "desc": "WAN IP连接实例编号"},
        "MU": {"range": "1~N", "desc": "主机单元(MU)实例编号"},
        "Slot": {"range": "0~N", "desc": "板卡槽位实例编号"},
        "EU": {
            "range": "0~65535",
            "desc": "扩展单元(EU)实例编号（i=0仅用于远端单元直连主机单元）",
        },
        "RU": {
            "range": "0~65535",
            "desc": "远端单元(RU)实例编号",
        },
        "RFChannel": {
            "range": "0~N",
            "desc": "射频通道实例编号",
        },
        "InterRATCell": {"range": "0~N", "desc": "异系统邻区实例编号"},
    }

    if parent_seg in range_map:
        info = range_map[parent_seg]
        return {
            "label": label,
            "range": info["range"],
            "desc": info["desc"],
        }

    # Default
    return {
        "label": label,
        "range": "0~N",
        "desc": f"{parent_seg}实例编号，N由对应NumberOfEntries参数决定",
    }


# ============================================================================
# 4. 生成 Markdown 文档
# ============================================================================

def generate_markdown():
    lines = []

    # Header
    lines.append("# 中国移动TD-LTE皮站/飞站基站设备南向接口数据配置模型规范 V2.3")
    lines.append("")
    lines.append("> 基于规范文件自动解析生成，按目录索引分组，命令智能拆分")
    lines.append(">")
    lines.append("> - 命令拆分规则：按路径公共前缀最小化划分，{i}代表数组实例")
    lines.append("> - 父级命令不含子级路径（如 DeviceInfo.* 不含 SwUpgrade.* 子路径）")
    lines.append("> - 每个路径唯一归属一个最精确命令，无重复映射")
    lines.append(f"> - 📖=只读(R)  📝=可读可写(RW)")
    lines.append("")

    # Statistics
    total_params = len(all_params)
    total_groups = len(toc_map)
    lines.append(f"**参数总数**: {total_params}  ")
    lines.append(f"**分组数**: {total_groups}")
    lines.append("")

    # TOC
    lines.append("---")
    lines.append("")
    lines.append("## 目录索引")
    lines.append("")
    lines.append("| 索引 | 参数管理类别 | 说明 |")
    lines.append("|------|-------------|------|")
    for idx, info in toc_map.items():
        lines.append(f"| {idx} | {info['name']} | {info['desc']} |")
    lines.append("")

    # Each group
    for sheet_name in DATA_SHEETS:
        if sheet_name not in toc_map:
            continue
        group_info = toc_map[sheet_name]
        params = params_by_sheet.get(sheet_name, [])
        if not params:
            continue

        lines.append("---")
        lines.append("")
        lines.append(f"## {sheet_name} - {group_info['name']}")
        lines.append("")
        lines.append(f"**{group_info['desc']}**  ")
        lines.append(f"参数数量: {len(params)}")
        lines.append("")

        # Build commands
        commands = build_commands_for_sheet(sheet_name, params)

        for cmd in commands:
            prefix = cmd["prefix"]
            cmd_params = [params[i] for i in cmd["params"]]
            i_count = cmd["i_count"]

            # Build command title with {i} labels
            cmd_title = prefix
            if i_count > 0:
                # Replace {i} with labeled versions
                i_ranges = infer_i_ranges(prefix, cmd_params, params)
                for ri in i_ranges:
                    cmd_title = cmd_title.replace("{i}", f"{{{ri['label']}}}", 1)

            # Count R and RW params
            r_count = sum(1 for p in cmd_params if p["rw"] == "R")
            rw_count = sum(1 for p in cmd_params if p["rw"] == "RW")

            perm_mark = ""
            if r_count > 0 and rw_count > 0:
                perm_mark = " 📖📝"
            elif r_count > 0:
                perm_mark = " 📖"
            elif rw_count > 0:
                perm_mark = " 📝"

            lines.append(f"#### 命令: {cmd_title}.*{perm_mark}")
            lines.append("")

            # {i} range info
            if i_count > 0:
                i_ranges = infer_i_ranges(prefix, cmd_params, params)
                for ri in i_ranges:
                    lines.append(f"- **{{{ri['label']}}}** 取值范围: `{ri['range']}` — {ri['desc']}")
                lines.append("")

            # Permission summary
            if r_count > 0 and rw_count > 0:
                lines.append(f"只读(📖): {r_count} | 可写(📝): {rw_count}")
                lines.append("")

            # Parameter table
            lines.append("| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |")
            lines.append("|---|------------|------------|--------|--------|------|------|")

            for pidx, p in enumerate(cmd_params, 1):
                perm_icon = "📖" if p["rw"] == "R" else "📝" if p["rw"] == "RW" else p["rw"]
                lines.append(
                    f"| {pidx} | `{p['tr098']}` | `{p['tr181']}` | {p['name']} | {p['cn']} | {perm_icon} {p['rw']} | {p['type']} |"
                )

            lines.append("")

    # Appendix: multi-instance reference
    lines.append("---")
    lines.append("")
    lines.append("## 附录A: 多实例参数参考表")
    lines.append("")
    lines.append("| TR-098 对象路径 | TR-181 对象路径 | 中文名称 | 读写属性 |")
    lines.append("|----------------|----------------|---------|---------|")
    for key, info in multi_instance_info.items():
        lines.append(f"| `{key}.{{i}}` | `{info['tr181']}.{{i}}` | {info['cn_name']} | {info['rw']} |")
    lines.append("")

    return "\n".join(lines)


# ============================================================================
# Main
# ============================================================================

if __name__ == "__main__":
    md_content = generate_markdown()

    with open(OUTPUT_PATH, "w", encoding="utf-8") as f:
        f.write(md_content)

    print(f"\n[OK] 文档已生成: {OUTPUT_PATH}")
    print(f"[OK] 文档行数: {len(md_content.splitlines())}")

    # Print summary per sheet
    print("\n=== 各分组统计 ===")
    for sheet_name in DATA_SHEETS:
        params = params_by_sheet.get(sheet_name, [])
        if not params:
            continue
        commands = build_commands_for_sheet(sheet_name, params)
        print(f"  {sheet_name}: {len(params)} 参数, {len(commands)} 命令")
