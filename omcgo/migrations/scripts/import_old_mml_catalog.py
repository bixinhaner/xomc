#!/usr/bin/env python3
"""Import legacy OMC MML catalog into the v2 schema (post-000113 layout).

Output: omcgo/migrations/seed/000111_mml_old_catalog_import.sql

User decision (2026-05-16):
  - One shared catalog. 21 product param_versions collapse by keyword.
  - mml_command_sub_fields.standard_path_id → standard_params (migration
    000113); no more dependency on mml_params self-dictionary.
  - Two-level group structure ONLY:
       业务大类 (top, depth=1) → keyword (leaf, depth=2)
    Commands hang off depth=2 nodes as leaves. Old multi-level OMC chain
    (BSC配置 → 基本信息 → 设备信息) collapses to (大类 → 设备信息).
  - Command display names are business-friendly:
       查询/修改/添加/删除 + {keyword 中文名}
    not the path-style "Device info(LST DEVICE_INFO)".
  - Sub-fields heuristic now targets standard_params (system-level standard
    path dictionary) and emits INSERT ... SELECT SQL so Python doesn't have
    to know which paths exist — Postgres does the matching.

Inputs:
  docs/files/db/mml/small_cell_param_group.sql  (1919 group rows)
  docs/files/db/mml/small_cell_param.sql        (7207 param rows, only used
                                                 for verb support detection)
"""

from __future__ import annotations

import hashlib
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

REPO_ROOT = Path(__file__).resolve().parents[3]
GROUP_FILE = REPO_ROOT / "docs/files/db/mml/small_cell_param_group.sql"
PARAM_FILE = REPO_ROOT / "docs/files/db/mml/small_cell_param.sql"
OUTPUT_FILE = (
    REPO_ROOT / "omcgo/migrations/seed/000111_mml_old_catalog_import.sql"
)

TARGET_PARAM_VERSION = "STANDARD"

# Highest first — same as previous revision. The first slice to declare a
# keyword wins; subsequent slices contribute nothing (the merge collapses on
# keyword identity rather than path overlap).
VERSION_PRIORITY: list[str] = [
    "BSC1.0", "BTS1.0", "QC4.2", "QC3.1", "MLN1.0", "MLQ1.0",
    "EA4.0", "EA4.0DUAL", "CR4.0", "BAIBLQ1.0", "BLX1.0", "BaiBNX1.0",
    "436Q1.0", "Nova430", "Nova430i", "QB1.0", "CA2.0", "NBIOT1.0",
    "ENB_DEFAULT_098", "ENB_DEFAULT_181", "DXDF1.0", "QC4.2T",
]
PRIORITY_INDEX = {v: i for i, v in enumerate(VERSION_PRIORITY)}

# Business category aliases — collapse synonymous parent names into a single
# bucket so the top-level UI doesn't read like a typo dump. Anything not
# listed keeps its original zh name.
CATEGORY_ALIASES: dict[str, str] = {
    "网络设置": "网络",
    "网络配置": "网络",
    "WAN": "网络",
    "WAN口检测参数": "网络",
    "Route": "网络",
    "网管设置": "网络",
    "回传链路检测": "网络",
    "接口设置": "网络",
    "高级配置": "高级",
    "高级设置": "高级",
    "上行BWP": "BWP",
    "下行BWP": "BWP",
    "同步": "同步",
    "同步设置": "同步",
    "NTP": "同步",
    "NTP设置": "同步",
    "邻频配置": "邻区",
    "邻区配置": "邻区",
    "TD-S邻频邻区": "邻区",
    "运行中邻区状态": "邻区",
    "切换算法配置": "切换",
    "SON": "切换",
    "ANR": "切换",
    "MME": "MME 与 EPC",
    "IPSec&MME Pool": "MME 与 EPC",
    "EMBEDDED_EPC设置": "MME 与 EPC",
    "下倾角配置": "天线",
    "RET添加": "天线",
    "Ald 设置": "天线",
    "天线": "天线",
    "IPSec": "安全",
    "安全设置": "安全",
    "KPI": "性能",
    "KPI配置": "性能",
    "性能配置": "性能",
    "Cs7": "信令",
    "日志设置": "运维",
    "运维": "运维",
    "Maintenance": "运维",
    "基本信息": "设备信息",
    "设备信息": "设备信息",
}
DEFAULT_CATEGORY = "其他"  # unmatched parent names fall here

CATEGORY_DISPLAY_ORDER: list[str] = [
    "设备信息", "网络", "同步", "性能", "邻区", "切换", "BWP",
    "MME 与 EPC", "安全", "信令", "天线", "运维", "高级",
]

# Verb → 中文 / 英文 显示名（命令叶子 label 业务化）
VERB_LABEL: dict[str, tuple[str, str]] = {
    "LST": ("查询", "Query"),
    "MOD": ("修改", "Modify"),
    "ADD": ("添加", "Add"),
    "RMV": ("删除", "Delete"),
}

# ----------------------------------------------------------------------------
# MySQL INSERT parser (verbatim from previous revisions)
# ----------------------------------------------------------------------------


def parse_insert_values(line: str) -> list[str] | None:
    m = re.search(r"VALUES\s*\((.*)\)\s*;?\s*$", line, re.DOTALL)
    if not m:
        return None
    body = m.group(1)
    values: list[str] = []
    cur = ""
    in_quotes = False
    i = 0
    while i < len(body):
        ch = body[i]
        if ch == "'" and (i == 0 or body[i - 1] != "\\"):
            in_quotes = not in_quotes
            cur += ch
        elif ch == "," and not in_quotes:
            values.append(cur.strip())
            cur = ""
        else:
            cur += ch
        i += 1
    if cur.strip():
        values.append(cur.strip())
    return values


def unquote(token: str) -> str | None:
    if token == "NULL":
        return None
    if len(token) >= 2 and token.startswith("'") and token.endswith("'"):
        return (
            token[1:-1]
            .replace("\\\\", "\\")
            .replace("\\'", "'")
            .replace("\\\"", '"')
            .replace("\\n", "\n")
            .replace("\\r", "\r")
        )
    return token


# ----------------------------------------------------------------------------
# Data classes
# ----------------------------------------------------------------------------


@dataclass
class GroupRow:
    old_id: str
    param_name_zh: str | None
    keyword: str | None
    v_lst: str | None
    v_mod: str | None
    v_add: str | None
    v_rmv: str | None
    parent_id: str
    param_version: str
    add_path: str | None
    param_name_en: str | None
    mobile_support: str | None
    broadband_support: str | None
    platform_support: str | None
    cell_number: str | None
    cell_index_location: str | None
    second_confirm: str | None
    confirm_en: str | None
    confirm_cn: str | None
    dis_order: str | None


# ----------------------------------------------------------------------------
# Parsing
# ----------------------------------------------------------------------------


def parse_groups(path: Path) -> list[GroupRow]:
    rows: list[GroupRow] = []
    with path.open("r", encoding="utf-8") as fh:
        for line in fh:
            if not line.startswith("INSERT"):
                continue
            tokens = parse_insert_values(line)
            if not tokens or len(tokens) < 20:
                continue
            v = [unquote(t) for t in tokens]
            rows.append(
                GroupRow(
                    old_id=str(v[0] or "").strip("()"),
                    param_name_zh=v[1],
                    keyword=v[2],
                    v_lst=v[3],
                    v_mod=v[4],
                    v_add=v[5],
                    v_rmv=v[6],
                    parent_id=str(v[7] or "").strip("()"),
                    param_version=v[8] or "QB1.0",
                    add_path=v[9],
                    param_name_en=v[10],
                    mobile_support=v[11],
                    broadband_support=v[12],
                    platform_support=v[13],
                    cell_number=v[14],
                    cell_index_location=v[15],
                    second_confirm=v[16],
                    confirm_en=v[17],
                    confirm_cn=v[18],
                    dis_order=v[19],
                )
            )
    return rows


# ----------------------------------------------------------------------------
# SQL helpers
# ----------------------------------------------------------------------------


def sql_str(value: str | None) -> str:
    if value is None:
        return "NULL"
    return "'" + value.replace("'", "''") + "'"


def sql_jsonb(obj: dict | list) -> str:
    return (
        "'"
        + json.dumps(obj, ensure_ascii=False).replace("'", "''")
        + "'::jsonb"
    )


def sql_bool(value: bool) -> str:
    return "true" if value else "false"


def make_deterministic_uuid(prefix_hex: str, key: str) -> str:
    digest = hashlib.md5(key.encode("utf-8")).hexdigest()  # noqa: S324
    body = (prefix_hex + digest)[:32]
    return f"{body[:8]}-{body[8:12]}-{body[12:16]}-{body[16:20]}-{body[20:32]}"


def chunked(items: Iterable, size: int) -> Iterable[list]:
    buf: list = []
    for x in items:
        buf.append(x)
        if len(buf) >= size:
            yield buf
            buf = []
    if buf:
        yield buf


def write_multi_row_insert(
    fh, table: str, columns: list[str], rows: list[str], conflict: str
) -> None:
    if not rows:
        return
    cols = ", ".join(columns)
    for batch in chunked(rows, 500):
        fh.write(f"INSERT INTO {table} ({cols}) VALUES\n")
        fh.write(",\n".join(batch))
        fh.write(f"\n{conflict};\n\n")


def slugify_category(name: str) -> str:
    """Produce a stable group_code label from a category zh name.

    Strategy: SHA-1 hash prefix to avoid Chinese/special chars in group_code
    (which feeds into ltree paths). Prefixed with ``cat_`` for readability.
    """
    h = hashlib.sha1(name.encode("utf-8")).hexdigest()[:12]  # noqa: S324
    return f"cat_{h}"


# ----------------------------------------------------------------------------
# Category derivation
# ----------------------------------------------------------------------------


def derive_category(leaf: GroupRow, groups_by_id: dict[str, GroupRow]) -> str:
    """Pick the category (top-level group zh name) for a canonical leaf."""
    parent = groups_by_id.get(leaf.parent_id)
    raw = ""
    if parent is not None and parent is not leaf:
        raw = (parent.param_name_zh or parent.param_name_en or "").strip()
    if not raw:
        # No parent → use the leaf's own zh name as its own category.
        raw = (leaf.param_name_zh or leaf.param_name_en or "").strip()
    return CATEGORY_ALIASES.get(raw, raw or DEFAULT_CATEGORY)


def _snake_to_pascal(snake: str) -> str:
    return "".join(part.capitalize() for part in snake.split("_") if part)


def pascal_segments_for_keyword(keyword: str) -> list[str]:
    """Path-segment-friendly forms of a keyword used for strict matching.

    Returns multiple candidates (full PascalCase + each token capitalized)
    so the runtime SQL can OR them. Empty list means "no path match
    possible for this keyword".
    """
    if not keyword:
        return []
    full = _snake_to_pascal(keyword)
    tokens = [t.capitalize() for t in keyword.split("_") if t]
    # Drop very common single-segment words ("Info" / "Config" / "Mgr") that
    # would explode matches across unrelated paths.
    NOISY = {"Info", "Config", "Mgr", "Set", "Cfg", "Setup", "Conf"}
    candidates: list[str] = []
    if full:
        candidates.append(full)
    for t in tokens:
        if t not in NOISY and t != full:
            candidates.append(t)
    # Dedup preserving order
    seen: set[str] = set()
    out: list[str] = []
    for c in candidates:
        if c not in seen:
            seen.add(c)
            out.append(c)
    return out


# ----------------------------------------------------------------------------
# Selection helpers
# ----------------------------------------------------------------------------


def select_canonical_leaves(
    leaves_by_keyword: dict[str, list[GroupRow]],
) -> list[GroupRow]:
    canonical: list[GroupRow] = []
    fallback = len(VERSION_PRIORITY)
    for keyword, slices in leaves_by_keyword.items():
        if not keyword:
            continue
        best = min(
            slices, key=lambda s: PRIORITY_INDEX.get(s.param_version, fallback)
        )
        canonical.append(best)
    return canonical


# ----------------------------------------------------------------------------
# SQL builders
# ----------------------------------------------------------------------------


def build_category_groups_sql(categories: list[str]) -> tuple[list[str], dict[str, str]]:
    """Top-level groups (depth=1, path=cat_<slug>)."""
    lines: list[str] = []
    id_map: dict[str, str] = {}
    for idx, cat in enumerate(categories):
        new_id = make_deterministic_uuid("a", f"category:{cat}")
        id_map[cat] = new_id
        slug = slugify_category(cat)
        name_i18n = {"zh-CN": cat, "en-US": cat}
        order = (
            CATEGORY_DISPLAY_ORDER.index(cat)
            if cat in CATEGORY_DISPLAY_ORDER
            else len(CATEGORY_DISPLAY_ORDER) + idx
        )
        lines.append(
            "    ("
            + ", ".join(
                [
                    f"'{new_id}'::uuid",
                    sql_str(slug),
                    sql_str(cat),
                    sql_str(cat),
                    sql_jsonb(name_i18n),
                    f"'{slug}'::ltree",
                    sql_str(TARGET_PARAM_VERSION),
                    str(order),
                    "true",
                    "'standard'",
                    "true",
                ]
            )
            + ")"
        )
    return lines, id_map


def build_leaf_groups_sql(
    canonical_leaves: list[GroupRow],
    leaf_category: dict[str, str],
    category_id: dict[str, str],
) -> tuple[list[str], dict[str, str]]:
    """Leaf groups (depth=2, one per canonical keyword)."""
    lines: list[str] = []
    leaf_id_map: dict[str, str] = {}
    used_codes: set[str] = set()
    for g in canonical_leaves:
        cat = leaf_category[g.keyword]
        parent_slug = slugify_category(cat)
        new_id = make_deterministic_uuid("a", f"leaf:{g.keyword}")
        leaf_id_map[g.keyword] = new_id
        # Build path = parent_slug.leaf_slug ; leaf_slug = keyword.lower() with
        # only [a-z0-9_]; ltree disallows dots/{}/-, so sanitize defensively.
        leaf_slug = "kw_" + re.sub(r"[^a-z0-9_]", "_", g.keyword.lower())
        if leaf_slug in used_codes:
            leaf_slug = f"{leaf_slug}_{g.old_id}"  # collision guard
        used_codes.add(leaf_slug)
        name_i18n = {
            "zh-CN": g.param_name_zh or g.keyword,
            "en-US": g.param_name_en or g.keyword,
        }
        lines.append(
            "    ("
            + ", ".join(
                [
                    f"'{new_id}'::uuid",
                    sql_str(leaf_slug),
                    sql_str(g.param_name_zh or g.keyword),
                    sql_str(g.param_name_en),
                    sql_jsonb(name_i18n),
                    f"'{parent_slug}.{leaf_slug}'::ltree",
                    sql_str(TARGET_PARAM_VERSION),
                    str(int(g.dis_order or 0)),
                    "true",
                    "'standard'",
                    "true",
                ]
            )
            + ")"
        )
    return lines, leaf_id_map


def build_commands_sql(
    canonical_leaves: list[GroupRow],
    leaves_by_keyword: dict[str, list[GroupRow]],
    leaf_id_map: dict[str, str],
) -> tuple[list[str], dict[tuple[str, str], str]]:
    """One command per (keyword, verb). Verb support unioned across slices.

    command_name_i18n carries the business-friendly label
    ("查询 设备信息" / "Query Device Info") so the front-end CommandTree
    leaf renders it directly without further verb-keyword splicing.
    """
    lines: list[str] = []
    cmd_id: dict[tuple[str, str], str] = {}
    rpc = {
        "LST": "GetParameterValues",
        "MOD": "SetParameterValues",
        "ADD": "AddObject",
        "RMV": "DeleteObject",
    }
    for g in canonical_leaves:
        if not g.keyword:
            continue
        slices = leaves_by_keyword.get(g.keyword, [g])
        verb_ok = {"LST": False, "MOD": False, "ADD": False, "RMV": False}
        for s in slices:
            for verb, attr in (
                ("LST", "v_lst"),
                ("MOD", "v_mod"),
                ("ADD", "v_add"),
                ("RMV", "v_rmv"),
            ):
                if (getattr(s, attr) or "").startswith("Y"):
                    verb_ok[verb] = True

        param_versions = sorted({s.param_version for s in slices})
        mobile_any = any((s.mobile_support or "Y") == "Y" for s in slices)
        broadband_any = any((s.broadband_support or "Y") == "Y" for s in slices)

        # logical_name_i18n 只放业务名（"设备信息"），后端 buildDisplayName
        # 会拼 verbLabel(op, lang) + " " + logicalName → "查询 设备信息"
        # command_name_i18n 同义保留，便于跨场景使用。
        logical_zh = g.param_name_zh or g.keyword
        logical_en = g.param_name_en or g.keyword
        for verb in ("LST", "MOD", "ADD", "RMV"):
            if not verb_ok[verb]:
                continue
            uid = make_deterministic_uuid("b", f"cmd:{g.keyword}:{verb}")
            cmd_id[(g.keyword, verb)] = uid
            zh_verb, en_verb = VERB_LABEL[verb]
            zh_full = f"{zh_verb} {logical_zh}"
            en_full = f"{en_verb} {logical_en}"
            confirm_i18n: dict[str, str] = {}
            if g.confirm_cn:
                confirm_i18n["zh-CN"] = g.confirm_cn
            if g.confirm_en:
                confirm_i18n["en-US"] = g.confirm_en
            require_confirm = bool(g.second_confirm and g.second_confirm.strip())
            platform_tags = {
                "mobile": mobile_any,
                "broadband": broadband_any,
                "param_versions": param_versions,
            }
            lines.append(
                "    ("
                + ", ".join(
                    [
                        f"'{uid}'::uuid",
                        sql_str(zh_full),  # command_name (含 verb)
                        sql_str(f"{verb}_{g.keyword}"),
                        sql_str("CONFIG"),
                        sql_str(logical_zh),  # description = business name
                        sql_str(rpc[verb]),
                        sql_str(verb),
                        "'[]'::jsonb",
                        sql_str(g.keyword),
                        f"'{leaf_id_map[g.keyword]}'::uuid",
                        sql_jsonb({"zh-CN": zh_full, "en-US": en_full}),  # command_name_i18n
                        sql_bool(require_confirm),
                        sql_jsonb(confirm_i18n),
                        sql_str(g.keyword),
                        # logical_name_i18n：仅业务名，无 verb，避免 buildDisplayName
                        # 与前端二次拼接造成 "查询 查询 设备信息" 的双重前缀。
                        sql_jsonb({"zh-CN": logical_zh, "en-US": logical_en}),
                        "'standard'",
                        "true",
                        sql_jsonb(platform_tags),
                    ]
                )
                + ")"
            )
    return lines, cmd_id


def build_sub_fields_sql_stmts(
    canonical_leaves: list[GroupRow],
    cmd_id: dict[tuple[str, str], str],
) -> list[str]:
    """Per-command INSERT ... SELECT from standard_params.

    Match rule:
      - PascalCase keyword OR its non-noisy tokens must appear as exact
        path segments of standard_params.standard_path (after stripping
        '{i}' instance markers).
      - Path depth ≤ 4 (=path has ≤ 3 dots between segments excluding
        trailing dot) to avoid pulling in deeply nested AntennaInfo /
        BWP sub-trees that are rarely meaningful as LST one-shot pulls.
      - add_path mode (Strategy 1) is encoded as a separate WHERE branch.

    mml_code is the last non-{i} path segment (e.g.
    "Device.DeviceInfo.SoftwareVersion" → "SoftwareVersion").
    """
    stmts: list[str] = []
    for g in canonical_leaves:
        if not g.keyword:
            continue
        # Build the WHERE filter for this keyword.
        clauses: list[str] = []
        if g.add_path:
            clauses.append(
                f"sp.standard_path LIKE {sql_str(g.add_path + '%')}"
            )
        pascal_candidates = pascal_segments_for_keyword(g.keyword)
        if pascal_candidates:
            arr_literal = "ARRAY[" + ", ".join(
                sql_str(c) for c in pascal_candidates
            ) + "]::text[]"
            # Strip '{i}' markers from each segment, then test ANY overlap.
            clauses.append(
                f"(ARRAY(SELECT regexp_replace(s, '\\{{i\\}}', '', 'g') "
                f"FROM unnest(string_to_array(sp.standard_path, '.')) AS s) "
                f"&& {arr_literal})"
            )
        if not clauses:
            continue
        where = "(" + " OR ".join(clauses) + ")"
        # Depth cap: number of dots in standard_path ≤ 5 (allows up to 6
        # segments — covers Device.X.Y.{i}.Foo.Bar). Trailing-dot object
        # entries get filtered to entry_type='parameter'.
        depth_cap = "LENGTH(sp.standard_path) - LENGTH(REPLACE(sp.standard_path, '.', '')) <= 5"
        leaf_only = "sp.entry_type = 'parameter'"
        for verb in ("LST", "MOD", "ADD", "RMV"):
            uid = cmd_id.get((g.keyword, verb))
            if not uid:
                continue
            # Verb capability:
            #   MOD requires writable (access IN READ_WRITE / WRITE_ONLY)
            #   ADD / RMV typically need access on objects, but standard_params
            #     entry_type='object' rows describe instance containers.
            access_filter = ""
            if verb == "MOD":
                access_filter = (
                    " AND COALESCE(sp.access, '') IN ('READ_WRITE', 'WRITE_ONLY')"
                )
            stmts.append(
                "INSERT INTO mml_command_sub_fields "
                "(id, command_id, standard_path_id, mml_code, label_i18n, "
                "default_selected, is_required, sort_order)\n"
                "SELECT\n"
                "    gen_random_uuid(),\n"
                f"    '{uid}'::uuid,\n"
                "    sp.id,\n"
                "    -- mml_code = path末段(去{i}) + '_' + sp.id 前 8 hex\n"
                "    --   保证 (command_id, mml_code) 唯一（path 末段可能重复，如多个 Enable）\n"
                "    LEFT(\n"
                "        COALESCE(\n"
                "            (SELECT s FROM unnest(\n"
                "                string_to_array(\n"
                "                    regexp_replace(sp.standard_path, '\\.$', ''),\n"
                "                    '.'\n"
                "                )\n"
                "             ) WITH ORDINALITY AS x(s, ord)\n"
                "             WHERE s !~ '\\{i\\}' AND s <> ''\n"
                "             ORDER BY ord DESC LIMIT 1),\n"
                "            'path'\n"
                "        ) || '_' || SUBSTRING(REPLACE(sp.id::text, '-', '') FROM 1 FOR 8),\n"
                "        100\n"
                "    ),\n"
                "    jsonb_build_object(\n"
                "        'zh-CN', sp.standard_path,\n"
                "        'en-US', sp.standard_path\n"
                "    ),\n"
                "    true,\n"
                "    false,\n"
                "    ROW_NUMBER() OVER (ORDER BY sp.standard_path) - 1\n"
                "FROM standard_params sp\n"
                f"WHERE {where} AND {depth_cap} AND {leaf_only}{access_filter}\n"
                "ON CONFLICT (command_id, standard_path_id) DO NOTHING;"
            )
    return stmts


# ----------------------------------------------------------------------------
# Emission
# ----------------------------------------------------------------------------


def emit_seed(
    output: Path,
    canonical_leaves: list[GroupRow],
    leaf_category: dict[str, str],
    cat_rows: list[str],
    leaf_rows: list[str],
    cmd_rows: list[str],
    sub_field_stmts: list[str],
) -> None:
    header = (
        "-- +goose Up\n"
        "-- ============================================================\n"
        "-- 000111_mml_old_catalog_import.sql\n"
        "-- 老 OMC MML catalog 合并导入（v3 — standard_params + 二层结构）\n"
        "--\n"
        "-- 用户决策（2026-05-16）：\n"
        "--   - 21 个 param_version 切片合并为一份 catalog（param_version='STANDARD'）\n"
        "--   - 命令分组 2 层：业务大类（一级） → keyword（二级），命令挂二级下\n"
        "--   - 命令命名业务化：\"查询/修改/添加/删除 + 中文名\"（替代 path-style）\n"
        "--   - mml_command_sub_fields.standard_path_id → standard_params（系统级标准 path 字典）\n"
        "--   - 不再写 mml_params（sub_fields 不依赖 mml 模块自有字典）\n"
        "--   - 当前不考虑私有 path；运行时 ACS 通过 ParamModel Translator 翻译（暂未启用）\n"
        "--\n"
        "-- 输入：docs/files/db/mml/small_cell_param_group.sql\n"
        "-- 此文件由 omcgo/migrations/scripts/import_old_mml_catalog.py 生成。\n"
        "-- ============================================================\n\n"
    )

    with output.open("w", encoding="utf-8") as fh:
        fh.write(header)

        fh.write(
            "-- ============================================================\n"
            "-- 0. mml_param_versions：仅 STANDARD（幂等）\n"
            "-- ============================================================\n"
            "INSERT INTO mml_param_versions (version_code, version_name, "
            "description, is_active, source) VALUES\n"
            f"    ('{TARGET_PARAM_VERSION}', 'TR-069 Standard Model', "
            "'Merged legacy OMC catalog — 2-level group structure', "
            "true, 'standard')\n"
            "ON CONFLICT (version_code) DO NOTHING;\n\n"
        )

        fh.write(
            "-- ============================================================\n"
            f"-- 1. mml_param_groups 业务大类（一级，{len(cat_rows)} 行）\n"
            "-- ============================================================\n"
        )
        write_multi_row_insert(
            fh,
            "mml_param_groups",
            [
                "id", "group_code", "group_name_zh", "group_name_en",
                "name_i18n", "path", "param_version",
                "display_order", "is_active", "source", "catalog_protected",
            ],
            cat_rows,
            "ON CONFLICT (param_version, group_code) DO NOTHING",
        )

        fh.write(
            "-- ============================================================\n"
            f"-- 2. mml_param_groups 命令分组（二级，{len(leaf_rows)} 行）\n"
            "-- ============================================================\n"
        )
        write_multi_row_insert(
            fh,
            "mml_param_groups",
            [
                "id", "group_code", "group_name_zh", "group_name_en",
                "name_i18n", "path", "param_version",
                "display_order", "is_active", "source", "catalog_protected",
            ],
            leaf_rows,
            "ON CONFLICT (param_version, group_code) DO NOTHING",
        )

        fh.write(
            "-- ============================================================\n"
            f"-- 3. mml_commands（业务化命名，{len(cmd_rows)} 行）\n"
            "-- ============================================================\n"
        )
        write_multi_row_insert(
            fh,
            "mml_commands",
            [
                "id", "command_name", "command_code", "category", "description",
                "rpc_method", "operation_type", "target_paths", "target_object",
                "group_id", "command_name_i18n", "require_confirm",
                "confirm_msg_i18n", "logical_code", "logical_name_i18n",
                "source", "catalog_protected", "platform_tags",
            ],
            cmd_rows,
            "ON CONFLICT (command_code) DO NOTHING",
        )

        fh.write(
            "-- ============================================================\n"
            f"-- 4. mml_command_sub_fields ← standard_params (启发式 SQL，{len(sub_field_stmts)} 条)\n"
            "--    每条 INSERT ... SELECT 用 standard_path 段匹配 PascalCase keyword,\n"
            "--    或 add_path 严格前缀；depth ≤ 6 段防爆。\n"
            "-- ============================================================\n"
        )
        for stmt in sub_field_stmts:
            fh.write(stmt + "\n\n")

        fh.write(
            "\n-- +goose Down\n"
            "-- ============================================================\n"
            "DELETE FROM mml_command_sub_fields\n"
            "    WHERE command_id IN (\n"
            "        SELECT id FROM mml_commands WHERE source = 'standard'\n"
            "    );\n"
            "DELETE FROM mml_commands WHERE source = 'standard';\n"
            "DELETE FROM mml_param_groups WHERE source = 'standard';\n"
        )


# ----------------------------------------------------------------------------
# Entry
# ----------------------------------------------------------------------------


def main() -> int:
    if not GROUP_FILE.exists():
        print(f"missing group file: {GROUP_FILE}", file=sys.stderr)
        return 1

    print(f"reading groups: {GROUP_FILE}")
    groups = parse_groups(GROUP_FILE)
    print(f"  parsed groups: {len(groups)}")

    groups_by_id = {g.old_id: g for g in groups}

    leaves_by_keyword: dict[str, list[GroupRow]] = {}
    for g in groups:
        if g.keyword:
            leaves_by_keyword.setdefault(g.keyword, []).append(g)
    print(f"  unique keywords: {len(leaves_by_keyword)}")

    canonical_leaves = select_canonical_leaves(leaves_by_keyword)
    print(f"  canonical leaves: {len(canonical_leaves)}")

    leaf_category: dict[str, str] = {}
    for leaf in canonical_leaves:
        leaf_category[leaf.keyword] = derive_category(leaf, groups_by_id)

    categories = sorted(
        set(leaf_category.values()),
        key=lambda c: (
            CATEGORY_DISPLAY_ORDER.index(c)
            if c in CATEGORY_DISPLAY_ORDER
            else len(CATEGORY_DISPLAY_ORDER),
            c,
        ),
    )
    print(f"  business categories: {len(categories)}")
    print(f"    " + ", ".join(categories[:15]) + (" ..." if len(categories) > 15 else ""))

    cat_rows, cat_id = build_category_groups_sql(categories)
    leaf_rows, leaf_id = build_leaf_groups_sql(canonical_leaves, leaf_category, cat_id)
    cmd_rows, cmd_id = build_commands_sql(canonical_leaves, leaves_by_keyword, leaf_id)
    sub_field_stmts = build_sub_fields_sql_stmts(canonical_leaves, cmd_id)

    print(f"  mml_param_groups (lvl1 + lvl2): {len(cat_rows)} + {len(leaf_rows)}")
    print(f"  mml_commands: {len(cmd_rows)}")
    print(f"  mml_command_sub_fields stmts: {len(sub_field_stmts)}")

    OUTPUT_FILE.parent.mkdir(parents=True, exist_ok=True)
    emit_seed(
        OUTPUT_FILE,
        canonical_leaves,
        leaf_category,
        cat_rows,
        leaf_rows,
        cmd_rows,
        sub_field_stmts,
    )
    print(f"wrote: {OUTPUT_FILE}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
