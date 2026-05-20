#!/usr/bin/env python3
"""
对比两份南向数据模型Markdown文档的差异。
1. omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md (新，数字实例归一化+去重)
2. docs/design/cmcc-tdlte-southbound-data-model-v2.3.md (旧，保留原始数字实例)
"""

import re
from collections import defaultdict

DOC1 = "/Users/cb/code/baicells/goomc/omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md"
DOC2 = "/Users/cb/code/baicells/goomc/docs/design/cmcc-tdlte-southbound-data-model-v2.3.md"

def parse_doc(filepath):
    """
    解析文档，提取所有参数信息。
    返回: {
        group: str,
        commands: {cmd_name: [params]},
        all_params: {tr181_path: {group, cmd, name, cn, rw, type}}
    }
    """
    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()
    
    lines = content.splitlines()
    
    current_group = ""
    current_cmd = ""
    params = {}  # tr181_path -> {group, cmd, name, cn, rw, type}
    commands = defaultdict(list)  # group -> [(cmd_name, [tr181_paths])]
    group_cmds = {}  # group -> {cmd_name: [tr181_paths]}
    
    for i, line in enumerate(lines):
        # Detect group header
        m = re.match(r'^## ([A-Z]{2}) - ', line)
        if m:
            current_group = m.group(1)
            group_cmds[current_group] = {}
            continue
        
        # Detect command header
        m = re.match(r'^#### 命令: [`]?(.+?)[`* ]*$', line)
        if not m:
            m = re.match(r'^#### 命令: (.+?)[ 📖📝]*$', line)
        if m:
            cmd_name = m.group(1).strip().strip('`').rstrip('.*').strip()
            current_cmd = cmd_name
            if current_group and current_cmd:
                group_cmds.setdefault(current_group, {})[current_cmd] = []
            continue
        
        # Detect parameter table rows
        if line.startswith("|") and current_cmd and current_group:
            parts = [p.strip() for p in line.split("|")]
            # Filter out header/separator lines
            if len(parts) < 5:
                continue
            if any(h in line for h in ["TR-098", "TR-181", "参数路径", "序号", "---"]):
                continue
            
            # Try to extract TR-181 path
            tr181 = ""
            name = ""
            cn = ""
            rw = ""
            ptype = ""
            
            # Determine which column layout
            # Doc1: | # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
            # Doc2: | 序号 | 参数路径（TR-181） | 参数名 | 中文名 | 重要性 | 读写 | 类型 | 描述 |
            
            # Find the column with Device. or InternetGatewayDevice.
            for part in parts:
                p = part.strip().strip('`')
                if p.startswith("Device.") or p.startswith("InternetGatewayDevice."):
                    if not tr181 or p.startswith("Device."):
                        tr181 = p
            
            if not tr181:
                continue
            
            # Normalize: convert to Device. format if it's TR-098
            # We want TR-181 paths for comparison
            if tr181.startswith("InternetGatewayDevice."):
                # Skip - we want TR-181, not TR-098
                # Check if there's a separate TR-181 column
                # In Doc1, column 3 is TR-181
                if len(parts) >= 4:
                    for part in parts[3:4]:  # Check column 3
                        p = part.strip().strip('`')
                        if p.startswith("Device."):
                            tr181 = p
                            break
            
            # Extract name, rw, type from remaining columns
            clean_parts = [p.strip().strip('`') for p in parts if p.strip()]
            
            for part in clean_parts:
                # Skip the path itself
                if part.startswith("Device.") or part.startswith("InternetGatewayDevice."):
                    continue
                # Check for RW/R pattern
                if "RW" in part or part.strip() in ("R", "RW"):
                    rw = part.replace("📖", "").replace("📝", "").strip()
                # Parameter name is usually short alphanumeric
                if not name and re.match(r'^[A-Za-z]\w*$', part) and part not in ("R", "RW", "A", "B", "CA", "CB"):
                    name = part
            
            if tr181 and current_group and current_cmd:
                params[tr181] = {
                    "group": current_group,
                    "cmd": current_cmd,
                    "name": name,
                    "rw": rw,
                }
                if current_group in group_cmds and current_cmd in group_cmds[current_group]:
                    group_cmds[current_group][current_cmd].append(tr181)
    
    return params, group_cmds


def normalize_path_for_comparison(path):
    """Normalize path for comparison: replace numeric indices with {i}"""
    segments = path.split(".")
    # Known multi-instance objects
    multi_obj = {
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
        "GERANFreqGroup", "Ethernet",
    }
    result = []
    for idx, seg in enumerate(segments):
        if idx > 0 and segments[idx - 1] in multi_obj:
            if re.match(r'^\d+$', seg):
                result.append("{i}")
            else:
                result.append(seg)
        else:
            result.append(seg)
    return ".".join(result)


# Parse both documents
print("解析文档1 (omcgo/规范/移动/南向数据模型/) ...")
params1, cmds1 = parse_doc(DOC1)
print(f"  参数数: {len(params1)}")

print("解析文档2 (docs/design/) ...")
params2, cmds2 = parse_doc(DOC2)
print(f"  参数数: {len(params2)}")

# Normalize both for comparison
norm1 = {}  # normalized_tr181 -> original_tr181
norm2 = {}
for p, info in params1.items():
    np = normalize_path_for_comparison(p)
    norm1[np] = p
for p, info in params2.items():
    np = normalize_path_for_comparison(p)
    norm2[np] = p

set1 = set(norm1.keys())
set2 = set(norm2.keys())

print("\n" + "="*80)
print("对比结果")
print("="*80)

# 1. Parameters only in doc1
only_in_1 = set1 - set2
print(f"\n### 仅在文档1中存在的参数: {len(only_in_1)}")
for p in sorted(only_in_1)[:20]:
    info = params1[norm1[p]]
    print(f"  [{info['group']}] {p}")

if len(only_in_1) > 20:
    print(f"  ... (还有 {len(only_in_1) - 20} 个)")

# 2. Parameters only in doc2
only_in_2 = set2 - set1
print(f"\n### 仅在文档2中存在的参数: {len(only_in_2)}")
for p in sorted(only_in_2)[:20]:
    info = params2[norm2[p]]
    print(f"  [{info['group']}] {p}")

if len(only_in_2) > 20:
    print(f"  ... (还有 {len(only_in_2) - 20} 个)")

# 3. Common parameters with differences
common = set1 & set2
print(f"\n### 共有参数: {len(common)}")

rw_diff = []
for p in sorted(common):
    info1 = params1[norm1[p]]
    info2 = params2[norm2[p]]
    # Compare RW
    rw1 = info1.get("rw", "").replace("📖", "").replace("📝", "").strip()
    rw2 = info2.get("rw", "").replace("📖", "").replace("📝", "").strip()
    if rw1 != rw2:
        rw_diff.append((p, rw1, rw2, info1["group"], info2["group"]))

print(f"\n### 读写权限差异: {len(rw_diff)}")
for p, rw1, rw2, g1, g2 in rw_diff[:30]:
    print(f"  [{g1}/{g2}] {p}: 文档1={rw1} vs 文档2={rw2}")

# 4. Group-level comparison
print(f"\n### 分组级别对比")
all_groups = set(list(cmds1.keys()) + list(cmds2.keys()))
for g in sorted(all_groups):
    c1 = cmds1.get(g, {})
    c2 = cmds2.get(g, {})
    p1_count = sum(len(v) for v in c1.values())
    p2_count = sum(len(v) for v in c2.values())
    c1_names = set(c1.keys())
    c2_names = set(c2.keys())
    
    diff_marker = ""
    if p1_count != p2_count:
        diff_marker = f" ⚠️ 参数数不同"
    if c1_names != c2_names:
        diff_marker += f" ⚠️ 命令不同"
    
    if diff_marker:
        print(f"  {g}: 文档1={p1_count}参数/{len(c1_names)}命令 vs 文档2={p2_count}参数/{len(c2_names)}命令{diff_marker}")
        
        # Show command differences
        only_c1 = c1_names - c2_names
        only_c2 = c2_names - c1_names
        if only_c1:
            print(f"    仅文档1命令: {sorted(only_c1)}")
        if only_c2:
            print(f"    仅文档2命令: {sorted(only_c2)}")
    else:
        print(f"  {g}: 一致 ({p1_count}参数/{len(c1_names)}命令)")

# 5. Summary of structural differences
print(f"\n{'='*80}")
print("### 结构差异汇总")
print(f"  文档1参数总数: {len(params1)} (归一化去重后)")
print(f"  文档2参数总数: {len(params2)} (含数字实例)")
print(f"  归一化后唯一参数 - 文档1: {len(set1)}, 文档2: {len(set2)}")
print(f"  仅文档1: {len(only_in_1)}, 仅文档2: {len(only_in_2)}, 共有: {len(common)}")
