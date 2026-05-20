#!/usr/bin/env python3
"""
T-0162: 批量补全 paramModel XML 的取值范围（min/max/enum）。

数据源：/Users/shangyingbin/Documents/notes/参数设置页/all_param_type_range.json
目标 XML：omcgo/data/param-mappings/{BLQ,MLQ,MLN,BaiBNQ,BSC,BTS,ENB_DEFAULT_098,ENB_DEFAULT_181}.xml

策略：
- 不动 BM.xml（JSON 无对应 product_model）
- 不动未映射的 4197 条 product_model（QAFA/QAFB/CR-B4860/RTD/RTS/QB/Nova430i/Nova430/RTD-CA/NBIOT/DXDF/QATA）
- 多 product 系列合并到同一 paramModel 时（BAIBLQ/BLX/QRTB → BLQ），同 path 不同 range 跳过 + 日志
- 已有 min/max/enumValues 的 param 不动
- type 冲突（JSON unsignedInt vs XML STRING 等）以 JSON 为准覆盖 XML 原 type + 日志（用户决定）
- enum 不设上限（照实补，前端 AntD Select 支持 showSearch）
- 文本级修改，保留 XML 原格式（属性顺序/缩进/换行）
"""

import json
import os
import re
import sys
from collections import defaultdict, OrderedDict

JSON_PATH = '/Users/shangyingbin/Documents/notes/参数设置页/all_param_type_range.json'
XML_DIR = '/Users/shangyingbin/project/goomc/omcgo/data/param-mappings'

# product_model → paramModel XML（小写键，便于匹配）
PM_MAP = {
    'BAIBLQ': 'BLQ',
    'BLX': 'BLQ',
    'QRTB': 'BLQ',
    'MLQ': 'MLQ',
    'MLN': 'MLN',
    'BaiBNX': 'BaiBNQ',
    'BSC': 'BSC',
    'BTS': 'BTS',
    'ENB_DEFAULT_098': 'ENB_DEFAULT_098',
    'ENB_DEFAULT_181': 'ENB_DEFAULT_181',
}

# JSON 中 type 字面 → XML type 枚举
TYPE_NORM = {
    'string': 'STRING',
    'String': 'STRING',
    'int': 'INT',
    'Int': 'INT',
    'strint': 'INT',
    'unsignedInt': 'U_INT',
    'eunsignedInt': 'U_INT',
    'uniqueInt': 'U_INT',
    'boolean': 'BOOLEAN',
    'Boolean': 'BOOLEAN',
    'Ipv4Addr': 'STRING',
    'Ipv6Addr': 'STRING',
    'Ipv4AddrArr': 'STRING',
    'LocalUplinkIpAddr': 'STRING',
    'RemoteUplinkIpAddr': 'STRING',
    # list 类暂保 STRING（前端解析）
    'stringList': 'STRING',
    'unsignedIntList': 'STRING',
}


def parse_data_type_range(raw):
    """
    解析 data_type_range 字符串 → {type, min, max, enumValues, enumLabels} (None=无)。
    返回 None 表示无法解析或无有效约束。

    支持形态：
      type-[min:max]            e.g. unsignedInt-[0:65535]
      enum-{v1,v2}-{l1,l2}      e.g. enum-{0,1}-{Disable,Enable}
      enum-{v1,v2}              （无 label）
      type                      （无约束，返回 None）
      unsignedInt[0:256]        （少连字符，容错）
    """
    if not raw:
        return None
    s = raw.strip().lstrip(' ')  # 去掉前缀空格（' enum-...'）

    # 形态 1: enum-{label_group}-{value_group} 或 enum-{value_group}
    # 真实语义（基于 195 条 enum-{Enable,Disable}-{1,0} / {Up,NoLink,...} 等样本验证）：
    #   第一组 = UI label（用户看到的文案）
    #   第二组 = 下发到基站的实际值
    # 单组形态（无第二组）时，第一组即下发值兼显示（无独立 label）
    m = re.match(r'^enum-\{(.*?)\}(?:-\{(.*?)\})?$', s)
    if m:
        first = m.group(1)
        second = m.group(2)
        if first == '' or first is None:
            return None
        if second is not None and second != '':
            # 两组：第二组=value(下发)，第一组=label(显示)
            return {'enumValues': second, 'enumLabels': first}
        # 单组：作 value，无 label
        return {'enumValues': first}

    # 形态 2: type-[min:max] 或 type[min:max] (容错)
    m = re.match(r'^([A-Za-z0-9_]+)-?\[([^:]+):([^\]]+)\]$', s)
    if m:
        raw_type = m.group(1)
        lo = m.group(2).strip()
        hi = m.group(3).strip()
        xml_type = TYPE_NORM.get(raw_type)
        if xml_type is None:
            return None
        # 校验 min/max 是合法数字（带负号）
        if not re.match(r'^-?\d+(\.\d+)?$', lo) or not re.match(r'^-?\d+(\.\d+)?$', hi):
            return None
        return {'type': xml_type, 'min': lo, 'max': hi}

    # 形态 3: 纯 type，无约束
    if re.match(r'^[A-Za-z0-9_]+$', s):
        return None

    # 形态 4: [1.5:25] 缺 type 前缀 —— 跳过
    return None


def load_json_constraints():
    """
    读 JSON，按 paramModel 分组，返回:
      { paramModel: { private_path: {'constraint': dict, 'sources': [pm1,pm2]} } }

    多 product 合并到同 paramModel 时（BAIBLQ/BLX/QRTB → BLQ），同 path 不同 range 跳过并记录冲突。
    """
    with open(JSON_PATH, encoding='utf-8-sig') as f:
        data = json.load(f)

    # paramModel → path → list of (pm, constraint)
    bucket = defaultdict(lambda: defaultdict(list))
    skip_no_constraint = 0
    skip_unmapped_pm = 0
    parse_fail = 0
    unmapped_pms = defaultdict(int)

    for d in data:
        pm = d.get('product_model', '')
        path = d.get('private_path', '')
        raw = d.get('data_type_range', '')
        if not path:
            continue
        target = PM_MAP.get(pm)
        if target is None:
            skip_unmapped_pm += 1
            unmapped_pms[pm] += 1
            continue
        c = parse_data_type_range(raw)
        if c is None:
            if not raw or raw.strip() in ('', ):
                skip_no_constraint += 1
            elif re.match(r'^[A-Za-z0-9_]+$', raw.strip().lstrip(' ')):
                skip_no_constraint += 1  # 纯 type 无约束
            else:
                parse_fail += 1
            continue
        bucket[target][path].append((pm, c))

    # 合并：同 path 多 pm 来源时，约束必须严格相同；否则跳过
    merged = defaultdict(dict)
    conflicts = []
    for target, paths in bucket.items():
        for path, entries in paths.items():
            if len(entries) == 1:
                merged[target][path] = {'constraint': entries[0][1], 'sources': [entries[0][0]]}
                continue
            # 多源：比较 constraint
            first = entries[0][1]
            sources = [entries[0][0]]
            ok = True
            for pm, c in entries[1:]:
                if c != first:
                    ok = False
                    conflicts.append((target, path, [(p, cc) for p, cc in entries]))
                    break
                sources.append(pm)
            if ok:
                merged[target][path] = {'constraint': first, 'sources': sources}

    return merged, {
        'skip_unmapped_pm': skip_unmapped_pm,
        'unmapped_pms': dict(unmapped_pms),
        'skip_no_constraint': skip_no_constraint,
        'parse_fail': parse_fail,
        'multi_source_conflicts': conflicts,
    }


PARAM_LINE_RE = re.compile(r'^(\s*)<param\s+(.*?)\s*/>\s*$')
ATTR_RE = re.compile(r'(\w+)="([^"]*)"')


def parse_attrs(s):
    """保序解析属性，返回 OrderedDict。"""
    od = OrderedDict()
    for k, v in ATTR_RE.findall(s):
        od[k] = v
    return od


def render_attrs(od):
    return ' '.join(f'{k}="{v}"' for k, v in od.items())


def has_any_constraint(attrs):
    """已有 min / max / enumValues 任一即视为有约束（不再覆盖）。"""
    return 'min' in attrs or 'max' in attrs or 'enumValues' in attrs


def insert_attrs_after(od, after_key, new_kvs):
    """在 after_key 之后插入 new_kvs（保持原属性顺序）。after_key 找不到时追加末尾。"""
    if after_key not in od:
        for k, v in new_kvs:
            od[k] = v
        return od
    new_od = OrderedDict()
    inserted = False
    for k, v in od.items():
        new_od[k] = v
        if k == after_key and not inserted:
            for nk, nv in new_kvs:
                if nk not in new_od and nk not in od:
                    new_od[nk] = nv
            inserted = True
    # 如果某个 new_kv 的 key 已存在于原 od，跳过（不覆盖）
    return new_od


def process_xml(xml_path, path_to_entry):
    """
    对单个 XML 应用约束。返回 (stats dict, new_text)。

    stats:
      applied: 成功补充数
      skipped_already: 已有约束
      skipped_type_conflict: type 冲突（XML 已有不同 type）
      not_found_in_xml: JSON 有路径但 XML 无对应 param
      type_conflicts: [(path, json_type, xml_type)]
    """
    with open(xml_path, encoding='utf-8') as f:
        text = f.read()
    lines = text.split('\n')

    stats = {
        'applied': 0,
        'skipped_already': 0,
        'type_overridden': 0,
        'type_override_log': [],
        'applied_samples': [],
    }
    matched_paths = set()

    new_lines = []
    for line in lines:
        m = PARAM_LINE_RE.match(line)
        if not m:
            new_lines.append(line)
            continue
        indent, attrs_str = m.group(1), m.group(2)
        attrs = parse_attrs(attrs_str)
        name = attrs.get('name', '')

        # 找到 JSON 约束（精确路径 + {i} 模板都查）
        entry = path_to_entry.get(name)
        if entry is None:
            new_lines.append(line)
            continue

        matched_paths.add(name)

        if has_any_constraint(attrs):
            stats['skipped_already'] += 1
            new_lines.append(line)
            continue

        c = entry['constraint']
        xml_type = attrs.get('type', '')

        # type 冲突：JSON 为准，原地覆盖 XML 原 type（保持属性顺序）
        if 'type' in c:
            wanted = c['type']
            if xml_type and xml_type != wanted:
                attrs['type'] = wanted
                stats['type_overridden'] += 1
                stats['type_override_log'].append((name, xml_type, wanted))

        # 构造要插入的 kv（保持 type → min → max → enumValues → enumLabels 顺序）
        kvs = []
        if 'type' in c and 'type' not in attrs:
            kvs.append(('type', c['type']))
        if 'min' in c:
            kvs.append(('min', c['min']))
        if 'max' in c:
            kvs.append(('max', c['max']))
        if 'enumValues' in c:
            kvs.append(('enumValues', c['enumValues']))
        if 'enumLabels' in c:
            kvs.append(('enumLabels', c['enumLabels']))

        if not kvs:
            new_lines.append(line)
            continue

        # 插入位置：type 之后；若无 type 则 changeApplies 之后；再不行追加末尾
        anchor = 'changeApplies' if 'changeApplies' in attrs else ('type' if 'type' in attrs else None)
        if anchor:
            new_attrs = insert_attrs_after(attrs, anchor, kvs)
        else:
            new_attrs = OrderedDict(attrs)
            for k, v in kvs:
                new_attrs[k] = v

        new_line = f'{indent}<param {render_attrs(new_attrs)}/>'
        new_lines.append(new_line)
        stats['applied'] += 1
        if len(stats['applied_samples']) < 5:
            stats['applied_samples'].append((name, dict(kvs)))

    stats['not_found_in_xml'] = len(path_to_entry) - len(matched_paths)
    return stats, '\n'.join(new_lines)


def fix_enum_order(xml_path, path_to_entry):
    """
    --fix-enum-order 模式：扫描 XML 中已有 enumValues+enumLabels 的 param，
    对照 JSON 真实语义（第二组=value，第一组=label）：
      - JSON 给出的 (correct_values, correct_labels) 与 XML 当前 (xml_values, xml_labels)
      - 若 (xml_values, xml_labels) == (correct_labels, correct_values) → 反了，对调
      - 若 (xml_values, xml_labels) == (correct_values, correct_labels) → 已正确，不动
      - 否则 → JSON 无此 path 或值不匹配（如 T-0158 手动补的内容不同），不动 + 日志

    返回 (stats, new_text)。
    """
    with open(xml_path, encoding='utf-8') as f:
        text = f.read()
    lines = text.split('\n')

    stats = {
        'scanned_enum_pairs': 0,
        'fixed': 0,
        'already_correct': 0,
        'no_json_ref': 0,
        'mismatch_no_action': 0,
        'fixed_samples': [],
        'mismatch_samples': [],
    }

    new_lines = []
    for line in lines:
        m = PARAM_LINE_RE.match(line)
        if not m:
            new_lines.append(line)
            continue
        indent, attrs_str = m.group(1), m.group(2)
        attrs = parse_attrs(attrs_str)
        name = attrs.get('name', '')

        xml_values = attrs.get('enumValues')
        xml_labels = attrs.get('enumLabels')
        if not xml_values or not xml_labels:
            new_lines.append(line)
            continue

        stats['scanned_enum_pairs'] += 1
        entry = path_to_entry.get(name)
        if entry is None:
            stats['no_json_ref'] += 1
            new_lines.append(line)
            continue

        c = entry['constraint']
        correct_values = c.get('enumValues')
        correct_labels = c.get('enumLabels')
        if correct_values is None or correct_labels is None:
            stats['no_json_ref'] += 1
            new_lines.append(line)
            continue

        if xml_values == correct_values and xml_labels == correct_labels:
            stats['already_correct'] += 1
            new_lines.append(line)
            continue

        if xml_values == correct_labels and xml_labels == correct_values:
            # 反了 → 对调
            attrs['enumValues'] = correct_values
            attrs['enumLabels'] = correct_labels
            new_line = f'{indent}<param {render_attrs(attrs)}/>'
            new_lines.append(new_line)
            stats['fixed'] += 1
            if len(stats['fixed_samples']) < 5:
                stats['fixed_samples'].append((name, xml_values, xml_labels, correct_values, correct_labels))
            continue

        # 内容既不正、也不正好反 → 不动（可能 T-0158 手动补的语义不同）
        stats['mismatch_no_action'] += 1
        if len(stats['mismatch_samples']) < 5:
            stats['mismatch_samples'].append((name, xml_values, xml_labels, correct_values, correct_labels))
        new_lines.append(line)

    return stats, '\n'.join(new_lines)


def main():
    print('=== T-0162 批量补全 paramModel XML 取值范围 ===\n')
    constraints, diag = load_json_constraints()

    if '--fix-enum-order' in sys.argv:
        print('[模式] --fix-enum-order：仅对调 enumValues/enumLabels 反掉的字段\n')
        dry_run = '--dry-run' in sys.argv
        grand = {'scanned': 0, 'fixed': 0, 'correct': 0, 'no_ref': 0, 'mismatch': 0}
        all_mismatch = []
        for target in sorted(set(PM_MAP.values())):
            xml_path = os.path.join(XML_DIR, f'{target}.xml')
            if not os.path.exists(xml_path):
                continue
            entries = constraints.get(target, {})
            stats, new_text = fix_enum_order(xml_path, entries)
            print(f'[{target}.xml] 已有 enum 对: {stats["scanned_enum_pairs"]}')
            print(f'  对调修正：{stats["fixed"]}')
            print(f'  已正确：{stats["already_correct"]}')
            print(f'  JSON 无引用：{stats["no_json_ref"]}')
            print(f'  内容不匹配（不动）：{stats["mismatch_no_action"]}')
            for n, xv, xl, cv, cl in stats['fixed_samples']:
                print(f'    fix: {n}  ({xv!r}→{cv!r} / {xl!r}→{cl!r})')
            for n, xv, xl, cv, cl in stats['mismatch_samples']:
                print(f'    mismatch: {n}  XML(v={xv!r},l={xl!r}) vs JSON(v={cv!r},l={cl!r})')
                all_mismatch.append((target, n, xv, xl, cv, cl))
            grand['scanned'] += stats['scanned_enum_pairs']
            grand['fixed'] += stats['fixed']
            grand['correct'] += stats['already_correct']
            grand['no_ref'] += stats['no_json_ref']
            grand['mismatch'] += stats['mismatch_no_action']
            if not dry_run and stats['fixed'] > 0:
                with open(xml_path, 'w', encoding='utf-8') as f:
                    f.write(new_text)
                print(f'  ✓ 写回')
            print()
        print('=== 总计 ===')
        for k, v in grand.items():
            print(f'  {k}: {v}')
        if all_mismatch:
            report_path = '/tmp/T-0162-enum-mismatch.txt'
            with open(report_path, 'w', encoding='utf-8') as f:
                f.write('paramModel\tparam_name\txml_values\txml_labels\tjson_values\tjson_labels\n')
                for tgt, n, xv, xl, cv, cl in all_mismatch:
                    f.write(f'{tgt}\t{n}\t{xv}\t{xl}\t{cv}\t{cl}\n')
            print(f'  不匹配清单：{report_path}')
        return

    print(f'[JSON 诊断]')
    print(f'  未映射 product_model 跳过：{diag["skip_unmapped_pm"]} 条')
    for pm, n in sorted(diag['unmapped_pms'].items(), key=lambda x: -x[1]):
        print(f'    {pm}: {n}')
    print(f'  无约束（纯 type / 空）跳过：{diag["skip_no_constraint"]} 条')
    print(f'  解析失败跳过：{diag["parse_fail"]} 条')
    print(f'  多源 path 约束冲突跳过：{len(diag["multi_source_conflicts"])} 条')
    if diag['multi_source_conflicts']:
        print('  （前 5 条冲突示例）')
        for tgt, path, entries in diag['multi_source_conflicts'][:5]:
            print(f'    [{tgt}] {path}')
            for pm, c in entries:
                print(f'        ← {pm}: {c}')
    print()

    grand_total = {'applied': 0, 'skipped_already': 0, 'type_overridden': 0, 'not_found_in_xml': 0}
    all_type_overrides = []

    dry_run = '--dry-run' in sys.argv

    for target in sorted(set(PM_MAP.values())):
        xml_path = os.path.join(XML_DIR, f'{target}.xml')
        if not os.path.exists(xml_path):
            print(f'!! {target}.xml 不存在，跳过')
            continue
        entries = constraints.get(target, {})
        print(f'[{target}.xml] JSON 候选 {len(entries)} 条')
        stats, new_text = process_xml(xml_path, entries)
        print(f'  补充：{stats["applied"]}')
        print(f'  已有约束跳过：{stats["skipped_already"]}')
        print(f'  type 覆盖（JSON 为准）：{stats["type_overridden"]}')
        print(f'  JSON 有路径但 XML 无对应：{stats["not_found_in_xml"]}')
        if stats['applied_samples']:
            print('  样例：')
            for name, kvs in stats['applied_samples']:
                print(f'    {name} → {kvs}')
        if stats['type_override_log']:
            print(f'  type 覆盖明细（前 3）：')
            for name, old, new in stats['type_override_log'][:3]:
                print(f'    {name}  {old} → {new}')
            all_type_overrides.extend([(target, n, o, w) for n, o, w in stats['type_override_log']])

        for k in grand_total:
            grand_total[k] += stats[k]

        if not dry_run and stats['applied'] > 0:
            with open(xml_path, 'w', encoding='utf-8') as f:
                f.write(new_text)
            print(f'  ✓ 写回 {xml_path}')
        elif dry_run:
            print(f'  (dry-run，未写回)')
        print()

    print('=== 总计 ===')
    for k, v in grand_total.items():
        print(f'  {k}: {v}')
    print(f'  type 覆盖总数: {len(all_type_overrides)}')

    if all_type_overrides:
        report_path = '/tmp/T-0162-type-overrides.txt'
        with open(report_path, 'w', encoding='utf-8') as f:
            f.write('paramModel\tparam_name\told_type\tnew_type\n')
            for tgt, n, old, new in all_type_overrides:
                f.write(f'{tgt}\t{n}\t{old}\t{new}\n')
        print(f'  type 覆盖清单：{report_path}')


if __name__ == '__main__':
    main()
