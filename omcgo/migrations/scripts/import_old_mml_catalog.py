#!/usr/bin/env python3
"""Import legacy OMC MML catalog (small_cell_param_group + small_cell_param)
into the new mml_param_groups / mml_commands / mml_params / mml_command_sub_fields
schema as a goose seed migration.

Output: omcgo/migrations/seed/000111_mml_old_catalog_import.sql

T-0123 follow-up. Maps the legacy catalog to the v2 schema that exists after
000095. Decisions baked in (per user approval 2026-05-16):
  - A2: legacy paths not in the standard dictionary are inserted as
        source='legacy', catalog_protected=true (preserves the standard 2001
        rows imported by seed/000096).
  - B1: mobile_support / broadband_support / platform_support land in
        mml_commands.platform_tags JSONB.
  - C1: second_confirm / confirm_en / confirm_cn land in the existing
        mml_commands.confirm_msg_i18n + require_confirm.

**Merge model (revised 2026-05-16)** — the legacy dump carries 21 product
``param_version`` slices (BSC1.0 / BTS1.0 / QC4.2 / …), each forming its own
mini-command-tree. User decision: the new catalog is ONE shared catalog (no
per-version branches in the UI); private/public custom commands keep the
existing ``mml_custom_commands`` flow. So this script collapses the 21
slices into a single ``param_version='STANDARD'`` tree:

  1. For each ``keyword`` (= logical leaf command), pick the slice whose
     ``param_version`` is highest priority in ``VERSION_PRIORITY``. Subsequent
     slices for the same keyword are dropped.
  2. Each kept leaf carries its ancestor chain (container groups with no
     keyword) — those ancestors are written with their original ``old_id``,
     so cross-version ancestors with identical names (e.g. two "Basic Info"
     parents from BSC1.0 vs QC4.2) remain distinct nodes. UI shows them as
     separate sub-trees; admin can later collapse via the catalog manager.
  3. ``mml_command_sub_fields`` for each kept keyword is the UNION of
     heuristically-matched params across all 21 slices (so DEVICE_INFO picks
     up Model Name from BSC1.0 *and* from QC4.2 if they overlap).
  4. ``mml_params`` rows for every legacy ``NAME_PATH`` are inserted under
     ``param_version='STANDARD'`` (deduped); ON CONFLICT keeps the existing
     standard dictionary rows untouched. Legacy-only paths land with
     ``source='legacy'``.
  5. ``mml_param_groups.param_version`` is always 'STANDARD' (single tree
     per the user decision).
"""

from __future__ import annotations

import json
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Iterable

REPO_ROOT = Path(__file__).resolve().parents[3]
GROUP_FILE = REPO_ROOT / "docs/files/db/mml/small_cell_param_group.sql"
PARAM_FILE = REPO_ROOT / "docs/files/db/mml/small_cell_param.sql"
OUTPUT_FILE = (
    REPO_ROOT
    / "omcgo/migrations/seed/000111_mml_old_catalog_import.sql"
)

VALID_VALUE_TYPES = {
    "string",
    "enum",
    "unsignedInt",
    "unsignedIntList",
    "stringList",
    "boolean",
    "uniqueInt",
    "int",
}

# Single param_version everything lands under after the 21-slice merge.
TARGET_PARAM_VERSION = "STANDARD"

# Priority order for selecting the canonical slice when a ``keyword`` appears
# in multiple legacy ``param_version`` slices. Earlier = higher priority.
#
# Rationale: BSC1.0 / BTS1.0 are the slices the live old OMC page renders for
# the eNB-BSC family the user inspected; we keep their layout. QC4.2 / QC3.1
# follow because they carry the richest set of generic eNB commands (256 / 237
# commands respectively). The tail handles deprecated / niche slices.
VERSION_PRIORITY: list[str] = [
    "BSC1.0",
    "BTS1.0",
    "QC4.2",
    "QC3.1",
    "MLN1.0",
    "MLQ1.0",
    "EA4.0",
    "EA4.0DUAL",
    "CR4.0",
    "BAIBLQ1.0",
    "BLX1.0",
    "BaiBNX1.0",
    "436Q1.0",
    "Nova430",
    "Nova430i",
    "QB1.0",
    "CA2.0",
    "NBIOT1.0",
    "ENB_DEFAULT_098",
    "ENB_DEFAULT_181",
    "DXDF1.0",
    "QC4.2T",
]
PRIORITY_INDEX: dict[str, int] = {v: i for i, v in enumerate(VERSION_PRIORITY)}


# ----------------------------------------------------------------------------
# MySQL INSERT parser
# ----------------------------------------------------------------------------


def parse_insert_values(line: str) -> list[str] | None:
    """Split a single-row MySQL INSERT VALUES (...) into raw field tokens.

    Handles single-quoted strings with backslash escapes; treats only
    unquoted commas as column delimiters. Returns each token verbatim
    (still with surrounding quotes if any).
    """
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
    """Convert a MySQL value token to a python string or None for NULL."""
    if token == "NULL":
        return None
    if len(token) >= 2 and token.startswith("'") and token.endswith("'"):
        # MySQL backslash escapes
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
# Value-type parsing (reused from convert_mml_to_seed.py with minor tweaks)
# ----------------------------------------------------------------------------


def parse_v_type(raw: str | None) -> tuple[str, str]:
    """Map a legacy V_TYPE string to (value_type, value_constraint_json).

    The legacy column carries a free-form description (e.g. ``enum-{a,b}-{1,2}``
    or ``unsignedInt-[0:65535]``). We normalise it to one of the eight
    allowed values in ``mml_params.chk_value_type`` and stash the original
    in ``value_constraint`` for round-tripping.
    """
    if not raw:
        return "string", json.dumps({"type": "string"}, ensure_ascii=False)

    m = re.match(r"^enum-\{([^}]*)\}-\{([^}]*)\}$", raw)
    if m:
        labels = m.group(1).split(",")
        values = m.group(2).split(",")
        return "enum", json.dumps(
            {"type": "enum", "labels": labels, "values": values},
            ensure_ascii=False,
        )

    m = re.match(r"^enum-\{([^}]+)\}$", raw)
    if m:
        values = m.group(1).split(",")
        return "enum", json.dumps(
            {"type": "enum", "labels": values, "values": values},
            ensure_ascii=False,
        )

    m = re.match(r"^string-\[(\d*):(\d*)\]$", raw)
    if m:
        c: dict = {"type": "string"}
        if m.group(1):
            c["min_length"] = int(m.group(1))
        if m.group(2):
            c["max_length"] = int(m.group(2))
        return "string", json.dumps(c, ensure_ascii=False)

    m = re.match(r"^(unsignedInt|int|uniqueInt)-\[(-?\d*):(-?\d*)\]", raw)
    if m:
        c = {"type": m.group(1)}
        if m.group(2):
            c["min"] = int(m.group(2))
        if m.group(3):
            c["max"] = int(m.group(3))
        return m.group(1), json.dumps(c, ensure_ascii=False)

    m = re.match(r"^(unsignedIntList|stringList)-\[(\d*):(\d*)\]$", raw)
    if m:
        c = {"type": m.group(1)}
        if m.group(2):
            c["min"] = int(m.group(2))
        if m.group(3):
            c["max"] = int(m.group(3))
        return m.group(1), json.dumps(c, ensure_ascii=False)

    m = re.match(r"^([a-zA-Z]+)", raw)
    if m and m.group(1) in VALID_VALUE_TYPES:
        return m.group(1), json.dumps(
            {"type": m.group(1), "raw": raw}, ensure_ascii=False
        )

    return "string", json.dumps({"type": "string", "raw": raw}, ensure_ascii=False)


# ----------------------------------------------------------------------------
# Data model
# ----------------------------------------------------------------------------


@dataclass
class GroupRow:
    """Parsed row from small_cell_param_group."""

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
    # Derived
    children: list[str] = field(default_factory=list)
    is_root: bool = False


@dataclass
class ParamRow:
    """Parsed row from small_cell_param."""

    old_id: str
    name: str | None
    name_path: str | None
    dft_value: str | None
    v_type: str | None
    v_writable: str | None
    v_lst: str | None
    v_mod: str | None
    v_add: str | None
    v_rmv: str | None
    is_leaf: str | None
    disp_ord: str | None
    stop_sign: str | None
    memo: str | None
    v_dynamic: str | None
    param_version: str
    mib_dn: str | None
    name_en: str | None
    mobile_support: str | None
    broadband_support: str | None
    js_regex: str | None
    title_cn: str | None
    title_en: str | None
    platform_support: str | None
    cn_explanation: str | None
    en_explanation: str | None
    software_version: str | None
    second_confirm: str | None
    confirm_en: str | None
    confirm_cn: str | None


# ----------------------------------------------------------------------------
# Parsing helpers
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


def parse_params(path: Path) -> list[ParamRow]:
    rows: list[ParamRow] = []
    with path.open("r", encoding="utf-8") as fh:
        for line in fh:
            if not line.startswith("INSERT"):
                continue
            tokens = parse_insert_values(line)
            if not tokens or len(tokens) < 30:
                continue
            v = [unquote(t) for t in tokens]
            rows.append(
                ParamRow(
                    old_id=str(v[0] or "").strip("()"),
                    name=v[1],
                    name_path=v[2],
                    dft_value=v[3],
                    v_type=v[4],
                    v_writable=v[5],
                    v_lst=v[6],
                    v_mod=v[7],
                    v_add=v[8],
                    v_rmv=v[9],
                    is_leaf=v[10],
                    disp_ord=v[11],
                    stop_sign=v[12],
                    memo=v[13],
                    v_dynamic=v[14],
                    param_version=v[15] or "QB1.0",
                    mib_dn=v[16],
                    name_en=v[17],
                    mobile_support=v[18],
                    broadband_support=v[19],
                    js_regex=v[20],
                    title_cn=v[21],
                    title_en=v[22],
                    platform_support=v[23],
                    cn_explanation=v[24],
                    en_explanation=v[25],
                    software_version=v[26],
                    second_confirm=v[27] if len(v) > 27 else None,
                    confirm_en=v[28] if len(v) > 28 else None,
                    confirm_cn=v[29] if len(v) > 29 else None,
                )
            )
    return rows


# ----------------------------------------------------------------------------
# SQL emission helpers
# ----------------------------------------------------------------------------


def sql_str(value: str | None) -> str:
    """Render a value as a quoted PG literal, escaping single quotes; NULL pass-through."""
    if value is None:
        return "NULL"
    escaped = value.replace("'", "''")
    return f"'{escaped}'"


def sql_jsonb(obj: dict | list) -> str:
    """Render a python dict/list as a quoted JSONB literal."""
    return "'" + json.dumps(obj, ensure_ascii=False).replace("'", "''") + "'::jsonb"


def sql_bool(value: bool) -> str:
    return "true" if value else "false"


def make_deterministic_uuid(prefix_hex: str, key: str) -> str:
    """Produce a syntactically valid v4-style UUID from a hex prefix + arbitrary key.

    The prefix is interpreted as raw hex (single char like ``a``, ``b``, ``c``).
    Key is hashed deterministically to fill the remaining 31 hex chars.
    """
    import hashlib

    digest = hashlib.md5(key.encode("utf-8")).hexdigest()  # noqa: S324 (non-crypto)
    # Use 31 hex chars from digest (plus 1 from prefix → 32 total)
    body = (prefix_hex + digest)[:32]
    return f"{body[:8]}-{body[8:12]}-{body[12:16]}-{body[16:20]}-{body[20:32]}"


# ----------------------------------------------------------------------------
# Group tree path computation
# ----------------------------------------------------------------------------


def build_ltree_paths(groups: list[GroupRow]) -> dict[str, str]:
    """For each group, compute its ltree path (root → leaf labels joined by '.').

    Each label is ``_g_{old_id}``. Roots are groups whose parent_id equals
    their own id, or whose parent_id is not in the id set.
    """
    by_id = {g.old_id: g for g in groups}
    paths: dict[str, str] = {}

    def label(gid: str) -> str:
        return f"_g_{gid}"

    def resolve(gid: str, seen: set[str]) -> list[str]:
        if gid in paths:
            return paths[gid].split(".")
        if gid in seen:  # cycle guard
            return [label(gid)]
        seen.add(gid)
        g = by_id[gid]
        parent = g.parent_id
        if parent == gid or parent not in by_id:
            chain = [label(gid)]
            g.is_root = True
        else:
            chain = resolve(parent, seen) + [label(gid)]
        paths[gid] = ".".join(chain)
        return chain

    for g in groups:
        resolve(g.old_id, set())

    return paths


# ----------------------------------------------------------------------------
# Sub-field heuristic matching (mirrors scripts/migrate_old_mml_data.sql)
# ----------------------------------------------------------------------------


def _snake_to_pascal(snake: str) -> str:
    """``DEVICE_INFO`` → ``DeviceInfo``; ``DNS_CONFIG`` → ``DnsConfig``."""
    return "".join(part.capitalize() for part in snake.split("_") if part)


def _heuristic_params_for_leaf(
    leaf: GroupRow, candidates: list[ParamRow]
) -> dict[str, ParamRow]:
    """Strict matching: add_path prefix OR PascalCase keyword as a complete
    TR-069 path segment.

    Earlier revisions cascaded loose strategies (substring match, MIB_DN
    stem, domain aliases) and produced 200+ sub_fields per command — too
    noisy for "LST one shot pull". We narrow to two precise rules:

      1. If the group declares ``add_path`` (e.g. ``Device.Services.FAPService.
         1.CellConfig.LTE.EPC.PLMNList.`` for PLMN), match every param whose
         tr069_path starts with that exact prefix. This is how the legacy OMC
         AddObject/DeleteObject commands flagged their member paths.
      2. Otherwise, take the PascalCase form of the keyword (``DEVICE_INFO``
         → ``DeviceInfo``) and match only paths that contain it as a
         **complete dot-separated segment** (``Device.DeviceInfo.Foo`` hits,
         ``Device.DeviceInformation.Foo`` does not). ``{i}`` instance markers
         are stripped before comparison.

    Trade-off: keywords whose pascal form does not appear verbatim in any
    path (e.g. ``BTS_INFO`` whose data lives under ``DeviceGSM.Bts.{i}.*``)
    will get zero sub_fields. Those commands still exist in mml_commands and
    can be populated via the admin UI catalog manager (T-0132). Better to
    ship clean defaults and let admin curate than auto-attach 200 noisy
    fields users have to unselect each time.
    """
    if not candidates or not leaf.keyword:
        return {}

    matched: dict[str, ParamRow] = {}

    # L1 — explicit add_path prefix (highest precision).
    if leaf.add_path:
        for p in candidates:
            if p.name_path and p.name_path.startswith(leaf.add_path):
                matched.setdefault(p.name_path, p)
        if matched:
            return matched

    # L2 — PascalCase keyword as a complete path segment.
    pascal = _snake_to_pascal(leaf.keyword)
    if not pascal:
        return {}
    for p in candidates:
        if not p.name_path:
            continue
        # Strip {i} placeholders from segments so e.g. "Bts.{i}" → "Bts".
        segments = [seg.split("{")[0] for seg in p.name_path.split(".")]
        if pascal in segments:
            matched.setdefault(p.name_path, p)

    return matched


def collect_merged_sub_fields(
    keyword: str,
    leaves_by_keyword: dict[str, list[GroupRow]],
    params_by_version: dict[str, list[ParamRow]],
) -> list[ParamRow]:
    """Pick the canonical slice for ``keyword`` and emit only its strict
    heuristic matches.

    Earlier revisions union'd across all 21 slices, which inflated sub-
    field counts (LST DEVICE_INFO → 200+ paths) because every slice's
    ``Device.DeviceInfo.*`` namespace contributed. The user explicitly
    asked for strict mode: the new behaviour matches old OMC's "one
    slice = one curated list" semantics by restricting the heuristic to
    the canonical slice's params (= highest VERSION_PRIORITY hit).

    Trade-off: keywords whose canonical slice carries few matching paths
    will yield few sub_fields. Admin UI can grow the list later; we'd
    rather ship a tight default than auto-attach noisy fields.
    """
    leaves = leaves_by_keyword.get(keyword, [])
    if not leaves:
        return []
    fallback = len(VERSION_PRIORITY)
    canonical = min(
        leaves, key=lambda l: PRIORITY_INDEX.get(l.param_version, fallback)
    )
    slice_params = params_by_version.get(canonical.param_version, [])
    return list(_heuristic_params_for_leaf(canonical, slice_params).values())


# ----------------------------------------------------------------------------
# SQL builders for each target table
# ----------------------------------------------------------------------------


def build_groups_sql(
    kept_groups: list[GroupRow], paths: dict[str, str]
) -> tuple[list[str], dict[str, str]]:
    """Return (multi-row VALUES lines, old_id → new_uuid map).

    Only emits groups in ``kept_groups`` (canonical keyword leaves + their
    ancestor chains). All rows are written under ``param_version='STANDARD'``;
    the legacy slice info goes into ``mml_commands.platform_tags.param_version``.
    """
    id_map: dict[str, str] = {}
    lines: list[str] = []
    for g in kept_groups:
        new_id = make_deterministic_uuid("a", f"group:{g.old_id}")
        id_map[g.old_id] = new_id
        group_code = f"_g_{g.old_id}"
        name_i18n = {
            "zh-CN": g.param_name_zh or "",
            "en-US": g.param_name_en or g.param_name_zh or "",
        }
        path = paths.get(g.old_id, group_code)
        display_order = int(g.dis_order or 0)
        lines.append(
            "    ("
            + ", ".join(
                [
                    f"'{new_id}'::uuid",
                    sql_str(group_code),
                    sql_str(g.param_name_zh or g.param_name_en or group_code),
                    sql_str(g.param_name_en),
                    sql_jsonb(name_i18n),
                    f"'{path}'::ltree",
                    sql_str(TARGET_PARAM_VERSION),
                    str(display_order),
                    "true",  # is_active
                    "'standard'",  # source
                    "true",  # catalog_protected
                ]
            )
            + ")"
        )
    return lines, id_map


def build_params_sql(params: list[ParamRow]) -> list[str]:
    """Build INSERT rows for mml_params under a single STANDARD version.

    Legacy slices share the same TR-069 paths across versions; we collapse
    them to one row per ``tr069_path`` (first slice wins for metadata). The
    on-disk standard dictionary (seed/000096) is preserved via
    ``ON CONFLICT (param_version, tr069_path) DO NOTHING``.
    """
    lines: list[str] = []
    seen: set[str] = set()  # tr069_path (under STANDARD only)
    for p in params:
        if not p.name_path:
            continue
        if p.name_path in seen:
            continue
        seen.add(p.name_path)
        value_type, value_constraint = parse_v_type(p.v_type)
        access_type = "READ_WRITE" if p.v_writable == "W" else "READ_ONLY"
        is_object = p.is_leaf != "Y"
        name_i18n = {
            "zh-CN": p.name or "",
            "en-US": p.name_en or p.name or "",
        }
        explanation_i18n: dict[str, str] = {}
        if p.cn_explanation:
            explanation_i18n["zh-CN"] = p.cn_explanation
        if p.en_explanation:
            explanation_i18n["en-US"] = p.en_explanation
        constraint_text_i18n: dict[str, str] = {}
        if p.title_cn:
            constraint_text_i18n["zh-CN"] = p.title_cn
        if p.title_en:
            constraint_text_i18n["en-US"] = p.title_en
        param_code = (p.mib_dn or p.name or "").strip()
        if not param_code:
            param_code = f"LEGACY_{p.old_id}"
        display_order = int(p.disp_ord or 0)
        lines.append(
            "    ("
            + ", ".join(
                [
                    "gen_random_uuid()",
                    sql_str(param_code),
                    sql_str(p.name or param_code),
                    sql_str(p.name_en),
                    sql_str(p.name_path),
                    sql_str(value_type),
                    "'" + value_constraint.replace("'", "''") + "'::jsonb",
                    sql_str(p.dft_value),
                    sql_str(p.js_regex),
                    sql_str(p.cn_explanation),
                    sql_str(p.en_explanation),
                    sql_str(p.title_cn),
                    sql_str(p.title_en),
                    "true",  # is_leaf (default)
                    str(display_order),
                    sql_str(TARGET_PARAM_VERSION),
                    "true",  # is_active
                    sql_str(access_type),
                    sql_bool(is_object),
                    "false",  # supports_add (legacy doesn't track)
                    "false",  # supports_delete
                    sql_str("Immediate"),
                    sql_jsonb(constraint_text_i18n),
                    "true",  # catalog_protected
                    "'legacy'",  # source (A2 decision)
                    sql_jsonb(name_i18n),
                    sql_jsonb(explanation_i18n),
                ]
            )
            + ")"
        )
    return lines


def build_commands_sql(
    canonical_leaves: list[GroupRow],
    group_id_map: dict[str, str],
    leaves_by_keyword: dict[str, list[GroupRow]],
) -> tuple[list[str], dict[tuple[str, str], str]]:
    """Build mml_commands inserts for the merged (per-keyword) catalog.

    One row per (keyword, verb). ``canonical_leaves`` already deduped by
    keyword; we still union the four verb flags across all slices that
    carry that keyword, so if BSC1.0 says ``v_lst=Y, v_mod=N`` but BTS1.0
    says ``v_lst=Y, v_mod=Y``, the merged command supports both LST and MOD.

    Returns (lines, (keyword, verb) → command_uuid map). The map keys
    intentionally use ``keyword`` instead of ``old_id`` so callers can look
    up by the canonical command identifier regardless of which slice it
    came from.
    """
    lines: list[str] = []
    cmd_id_map: dict[tuple[str, str], str] = {}
    verb_flags = (
        ("LST", "v_lst"),
        ("MOD", "v_mod"),
        ("ADD", "v_add"),
        ("RMV", "v_rmv"),
    )
    seen_codes: set[str] = set()  # command_code is UNIQUE — guard against dups
    rpc_methods = {
        "LST": "GetParameterValues",
        "MOD": "SetParameterValues",
        "ADD": "AddObject",
        "RMV": "DeleteObject",
    }
    for g in canonical_leaves:
        if not g.keyword:
            continue
        slices = leaves_by_keyword.get(g.keyword, [g])
        # Union verb support across all slices that carry the keyword.
        verb_support = {verb: False for verb, _ in verb_flags}
        for s in slices:
            for verb, attr in verb_flags:
                flag = getattr(s, attr) or ""
                if flag.startswith("Y"):
                    verb_support[verb] = True
        # Union platform_tags across slices.
        param_versions_seen = sorted({s.param_version for s in slices})
        mobile_any = any((s.mobile_support or "Y") == "Y" for s in slices)
        broadband_any = any((s.broadband_support or "Y") == "Y" for s in slices)
        platform_codes = sorted({
            s.platform_support for s in slices if s.platform_support
        })

        for verb, _ in verb_flags:
            if not verb_support[verb]:
                continue
            command_code = f"{verb}_{g.keyword}"
            if command_code in seen_codes:
                continue
            seen_codes.add(command_code)
            cmd_uuid = make_deterministic_uuid("b", f"cmd:{g.keyword}:{verb}")
            cmd_id_map[(g.keyword, verb)] = cmd_uuid
            logical_name_i18n = {
                "zh-CN": g.param_name_zh or g.keyword,
                "en-US": g.param_name_en or g.keyword,
            }
            command_name_i18n = {
                "zh-CN": f"{verb} {g.keyword}",
                "en-US": f"{verb} {g.keyword}",
            }
            confirm_msg_i18n: dict[str, str] = {}
            if g.confirm_cn:
                confirm_msg_i18n["zh-CN"] = g.confirm_cn
            if g.confirm_en:
                confirm_msg_i18n["en-US"] = g.confirm_en
            require_confirm = bool(g.second_confirm and g.second_confirm.strip())
            platform_tags = {
                "mobile": mobile_any,
                "broadband": broadband_any,
                "platform_codes": platform_codes,
                "param_versions": param_versions_seen,
            }
            lines.append(
                "    ("
                + ", ".join(
                    [
                        f"'{cmd_uuid}'::uuid",
                        sql_str(f"{verb} {g.keyword}"),
                        sql_str(command_code),
                        sql_str("CONFIG"),  # category fallback
                        sql_str(g.param_name_zh),  # description
                        sql_str(rpc_methods[verb]),
                        sql_str(verb),
                        "'[]'::jsonb",  # target_paths (refreshed by trigger after sub_fields)
                        sql_str(g.keyword),  # target_object
                        f"'{group_id_map[g.old_id]}'::uuid",
                        sql_jsonb(command_name_i18n),
                        sql_bool(require_confirm),
                        sql_jsonb(confirm_msg_i18n),
                        sql_str(g.keyword),  # logical_code
                        sql_jsonb(logical_name_i18n),
                        "'standard'",  # source
                        "true",  # catalog_protected
                        sql_jsonb(platform_tags),
                    ]
                )
                + ")"
            )
    return lines, cmd_id_map


def build_sub_fields_sql(
    canonical_leaves: list[GroupRow],
    leaves_by_keyword: dict[str, list[GroupRow]],
    params_by_version: dict[str, list[ParamRow]],
    cmd_id_map: dict[tuple[str, str], str],
) -> list[str]:
    """Build mml_command_sub_fields INSERTs for the merged catalog.

    For each canonical leaf, union the heuristically-matched params from
    every slice that carries the same ``keyword``. Sub-fields are deduped
    by ``mml_code`` (= legacy MIB_DN) inside one command — that's also the
    constraint key in ``mml_command_sub_fields``. param_id is resolved
    at insert time by sub-query against ``mml_params`` (STANDARD slice).
    """
    lines: list[str] = []
    for g in canonical_leaves:
        if not g.keyword:
            continue
        merged_params = collect_merged_sub_fields(
            g.keyword, leaves_by_keyword, params_by_version
        )
        if not merged_params:
            continue

        # Dedupe per command by mml_code (UNIQUE constraint); keep first
        # occurrence for stable ordering. tr069_path also dedupes incidentally.
        seen_codes: set[str] = set()
        seen_paths: set[str] = set()
        ordered: list[ParamRow] = []
        for p in merged_params:
            mib = (p.mib_dn or "").strip()
            if not mib or mib in seen_codes:
                continue
            if not p.name_path or p.name_path in seen_paths:
                continue
            seen_codes.add(mib)
            seen_paths.add(p.name_path)
            ordered.append(p)

        for verb in ("LST", "MOD", "ADD", "RMV"):
            cmd_uuid = cmd_id_map.get((g.keyword, verb))
            if not cmd_uuid:
                continue
            sort_order = 0
            for p in ordered:
                # Verb capability filter: LST shows all matches; MOD / ADD /
                # RMV require the underlying param to declare matching support.
                if verb == "MOD" and (p.v_mod or "").upper() != "Y":
                    continue
                if verb == "ADD" and not (p.v_add or "").startswith("Y"):
                    continue
                if verb == "RMV" and not (p.v_rmv or "").startswith("Y"):
                    continue
                mml_code = (p.mib_dn or "")[:100]
                if not mml_code:
                    continue
                sf_uuid = make_deterministic_uuid(
                    "c", f"sf:{g.keyword}:{verb}:{p.name_path}"
                )
                label_i18n = {
                    "zh-CN": p.name or "",
                    "en-US": p.name_en or p.name or "",
                }
                # default_selected=true mirrors legacy OMC LST behaviour
                # (all sub-fields ticked by default; user un-ticks unwanted).
                default_selected = "true"
                is_required = "true" if (
                    verb in ("MOD", "ADD") and (p.v_mod or "").upper() == "Y"
                ) else "false"
                lines.append(
                    "INSERT INTO mml_command_sub_fields "
                    "(id, command_id, param_id, mml_code, label_i18n, "
                    "default_selected, is_required, sort_order) "
                    "SELECT "
                    f"'{sf_uuid}'::uuid, '{cmd_uuid}'::uuid, mp.id, "
                    f"{sql_str(mml_code)}, {sql_jsonb(label_i18n)}, "
                    f"{default_selected}, {is_required}, {sort_order} "
                    "FROM mml_params mp "
                    f"WHERE mp.tr069_path = {sql_str(p.name_path)} "
                    f"AND mp.param_version = {sql_str(TARGET_PARAM_VERSION)} "
                    "LIMIT 1 "
                    "ON CONFLICT (command_id, mml_code) DO NOTHING;"
                )
                sort_order += 1
    return lines


# ----------------------------------------------------------------------------
# Output assembly
# ----------------------------------------------------------------------------


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


def emit_seed(
    output: Path,
    groups: list[GroupRow],
    params: list[ParamRow],
    paths: dict[str, str],
    group_id_map: dict[str, str],
    group_rows_sql: list[str],
    param_rows_sql: list[str],
    cmd_rows_sql: list[str],
    sub_field_stmts: list[str],
    cmd_id_map: dict[tuple[str, str], str],
) -> None:
    header = (
        "-- +goose Up\n"
        "-- ============================================================\n"
        "-- 000111_mml_old_catalog_import.sql\n"
        "-- 老 OMC MML catalog 合并导入（T-0123 后续）\n"
        "--\n"
        "-- 输入：docs/files/db/mml/small_cell_param_group.sql\n"
        "--       docs/files/db/mml/small_cell_param.sql\n"
        "--\n"
        "-- 合并策略（用户决策 2026-05-16）：\n"
        "--   老数据 21 个 param_version 切片合并为一份 catalog（param_version='STANDARD'）。\n"
        "--   同一个 keyword（如 DEVICE_INFO）出现在多个版本时，按 VERSION_PRIORITY\n"
        "--   选优先级最高的切片作为 canonical（决定父链 / 名称 / 二次确认文案），\n"
        "--   sub-fields 取所有切片启发式匹配结果的并集。私有/公有命令仍走\n"
        "--   既有 mml_custom_commands 表，不在本 seed 范围内。\n"
        "--\n"
        "-- 决策：A2 缺失 path 自动补建 source='legacy';\n"
        "--       B1 platform_tags 在 mml_commands（合并后含 param_versions 列表）;\n"
        "--       C1 复用 confirm_msg_i18n + require_confirm.\n"
        "--\n"
        "-- 此文件由 omcgo/migrations/scripts/import_old_mml_catalog.py 生成。\n"
        "-- ============================================================\n\n"
    )

    with output.open("w", encoding="utf-8") as fh:
        fh.write(header)

        fh.write(
            "-- ============================================================\n"
            "-- 0. mml_param_versions 兜底（仅 STANDARD，幂等）。\n"
            "--    老的 21 个版本号合并到 STANDARD；如未来 admin 需细分版本，\n"
            "--    再通过 admin UI 单独建。\n"
            "-- ============================================================\n"
            "INSERT INTO mml_param_versions (version_code, version_name, "
            "description, is_active, source) VALUES\n"
            f"    ('{TARGET_PARAM_VERSION}', 'TR-069 Standard Model', "
            "'Merged legacy OMC catalog (21 product slices collapsed by keyword)', "
            "true, 'standard')\n"
            "ON CONFLICT (version_code) DO NOTHING;\n\n"
        )

        fh.write(
            "-- ============================================================\n"
            "-- 1. mml_params 补建（已存在 standard 行不动；缺失行 legacy 新建）\n"
            "-- ============================================================\n"
        )
        write_multi_row_insert(
            fh,
            "mml_params",
            [
                "id",
                "param_code",
                "param_name_zh",
                "param_name_en",
                "tr069_path",
                "value_type",
                "value_constraint",
                "default_value",
                "js_regex",
                "explanation_zh",
                "explanation_en",
                "title_zh",
                "title_en",
                "is_leaf",
                "display_order",
                "param_version",
                "is_active",
                "access_type",
                "is_object",
                "supports_add",
                "supports_delete",
                "change_applies",
                "constraint_text_i18n",
                "catalog_protected",
                "source",
                "name_i18n",
                "explanation_i18n",
            ],
            param_rows_sql,
            "ON CONFLICT (param_version, tr069_path) DO NOTHING",
        )

        fh.write(
            "-- ============================================================\n"
            "-- 2. mml_param_groups（业务分类树，1919 行）\n"
            "-- ============================================================\n"
        )
        write_multi_row_insert(
            fh,
            "mml_param_groups",
            [
                "id",
                "group_code",
                "group_name_zh",
                "group_name_en",
                "name_i18n",
                "path",
                "param_version",
                "display_order",
                "is_active",
                "source",
                "catalog_protected",
            ],
            group_rows_sql,
            "ON CONFLICT (param_version, group_code) DO NOTHING",
        )

        fh.write(
            "-- ============================================================\n"
            "-- 3. mml_commands（聚合命令，每个叶子按 v_lst/v_mod/v_add/v_rmv 拆 1-4 条）\n"
            "-- ============================================================\n"
        )
        write_multi_row_insert(
            fh,
            "mml_commands",
            [
                "id",
                "command_name",
                "command_code",
                "category",
                "description",
                "rpc_method",
                "operation_type",
                "target_paths",
                "target_object",
                "group_id",
                "command_name_i18n",
                "require_confirm",
                "confirm_msg_i18n",
                "logical_code",
                "logical_name_i18n",
                "source",
                "catalog_protected",
                "platform_tags",
            ],
            cmd_rows_sql,
            "ON CONFLICT (command_code) DO NOTHING",
        )

        fh.write(
            "-- ============================================================\n"
            "-- 4. mml_command_sub_fields（启发式 add_path 前缀 + keyword 模糊匹配）\n"
            "--    param_id 通过子查询从 mml_params (tr069_path, param_version) 解析。\n"
            "-- ============================================================\n"
        )
        for stmt in sub_field_stmts:
            fh.write(stmt + "\n")

        fh.write(
            "\n\n-- +goose Down\n"
            "-- ============================================================\n"
            "-- 反向：删本次导入的 source='standard' commands/groups + source='legacy' params。\n"
            "-- 标识：mml_param_groups.group_code 以 '_g_' 开头（导入工具约定）。\n"
            "-- ============================================================\n"
            "DELETE FROM mml_command_sub_fields WHERE command_id IN ("
            "SELECT id FROM mml_commands WHERE source = 'standard' "
            "AND group_id IN (SELECT id FROM mml_param_groups "
            "WHERE source = 'standard' "
            "AND group_code LIKE '\\_g\\_%' ESCAPE '\\'));\n"
            "DELETE FROM mml_commands WHERE source = 'standard' "
            "AND group_id IN (SELECT id FROM mml_param_groups "
            "WHERE source = 'standard' "
            "AND group_code LIKE '\\_g\\_%' ESCAPE '\\');\n"
            "DELETE FROM mml_param_groups WHERE source = 'standard' "
            "AND group_code LIKE '\\_g\\_%' ESCAPE '\\';\n"
            "DELETE FROM mml_params WHERE source = 'legacy';\n"
        )


# ----------------------------------------------------------------------------
# Entry point
# ----------------------------------------------------------------------------


def merge_same_name_containers(
    kept_groups: list[GroupRow],
    canonical_leaves: list[GroupRow],
    paths: dict[str, str],
    groups_by_id: dict[str, GroupRow],
) -> dict[str, str]:
    """Collapse containers (keyword=None) that share the same Chinese name and
    canonical parent, returning ``old_id → canonical_old_id`` mapping.

    Without this step the merged catalog still shows N copies of "基站配置",
    "无线配置", etc. — one per legacy ``param_version`` slice. Containers
    are merged only when their (canonical_parent, zh-CN name) tuple matches;
    leaves are not touched (they were already keyword-deduped upstream).

    Processing order is BFS by path depth so that a level's parent has
    already been resolved before the child is considered.
    """
    leaf_ids = {g.old_id for g in canonical_leaves}
    fallback_priority = len(VERSION_PRIORITY)
    mapping: dict[str, str] = {gid: gid for gid in leaf_ids}

    # Depth-stratified container list (leaves excluded).
    depth_of = {g.old_id: paths[g.old_id].count(".") for g in kept_groups}
    sorted_depths = sorted({depth_of[g.old_id] for g in kept_groups})

    for depth in sorted_depths:
        # Bucket containers at this depth by (canonical_parent_id, zh-CN name).
        buckets: dict[tuple[str | None, str], list[GroupRow]] = {}
        for g in kept_groups:
            if g.old_id in leaf_ids:
                continue
            if depth_of[g.old_id] != depth:
                continue
            path = paths[g.old_id]
            if "." in path:
                parent_label = path.split(".")[-2]
                parent_old_id = parent_label[len("_g_"):]
                canonical_parent = mapping.get(parent_old_id, parent_old_id)
            else:
                canonical_parent = None
            name_key = (g.param_name_zh or g.param_name_en or "").strip().lower()
            buckets.setdefault((canonical_parent, name_key), []).append(g)

        # Inside each bucket pick the canonical by VERSION_PRIORITY.
        for bucket_groups in buckets.values():
            canonical = min(
                bucket_groups,
                key=lambda g: PRIORITY_INDEX.get(g.param_version, fallback_priority),
            )
            for g in bucket_groups:
                mapping[g.old_id] = canonical.old_id

    return mapping


def rewrite_paths(
    kept_groups: list[GroupRow],
    mapping: dict[str, str],
    original_paths: dict[str, str],
) -> dict[str, str]:
    """Recompute each kept group's ltree path so non-canonical containers
    are replaced by their canonical id in the chain.

    Canonical groups (``mapping[gid] == gid``) get a path whose every
    segment is itself canonical — duplicates collapse naturally. Non-
    canonical groups share the path of their canonical (they're going to
    be dropped by ``filter_canonical_groups`` anyway, but we still write
    the path so debugging traces line up).
    """
    new_paths: dict[str, str] = {}

    def resolve(gid: str, seen: set[str]) -> str:
        if gid in new_paths:
            return new_paths[gid]
        if gid in seen:  # cycle guard (defensive)
            return f"_g_{gid}"
        seen.add(gid)
        canonical = mapping.get(gid, gid)
        if canonical != gid:
            # Non-canonical → shares the canonical's path.
            new_paths[gid] = resolve(canonical, seen)
            return new_paths[gid]
        # gid is canonical itself — rebuild from its (canonical) parent.
        orig = original_paths.get(gid, f"_g_{gid}")
        if "." not in orig:
            new_paths[gid] = f"_g_{gid}"
            return new_paths[gid]
        parent_label = orig.split(".")[-2]
        parent_old_id = parent_label[len("_g_"):]
        canonical_parent = mapping.get(parent_old_id, parent_old_id)
        parent_path = resolve(canonical_parent, seen)
        new_paths[gid] = f"{parent_path}._g_{gid}"
        return new_paths[gid]

    for g in kept_groups:
        resolve(g.old_id, set())
    return new_paths


def select_canonical_leaves(
    leaves_by_keyword: dict[str, list[GroupRow]],
) -> list[GroupRow]:
    """Pick one canonical ``GroupRow`` per ``keyword`` using VERSION_PRIORITY."""
    canonical: list[GroupRow] = []
    fallback = len(VERSION_PRIORITY)  # any unknown version sorts last
    for keyword, slices in leaves_by_keyword.items():
        if not keyword:
            continue
        best = min(
            slices, key=lambda s: PRIORITY_INDEX.get(s.param_version, fallback)
        )
        canonical.append(best)
    return canonical


def collect_ancestor_ids(
    leaves: list[GroupRow], groups_by_id: dict[str, GroupRow]
) -> set[str]:
    """For each leaf, walk parent_id chain and collect every group id seen.

    Stops at self-referential roots (``parent == self``) or missing parents.
    """
    keep: set[str] = set()
    for leaf in leaves:
        cur: GroupRow | None = leaf
        while cur is not None and cur.old_id not in keep:
            keep.add(cur.old_id)
            parent = groups_by_id.get(cur.parent_id)
            if parent is None or parent is cur:
                break
            cur = parent
    return keep


def main() -> int:
    if not GROUP_FILE.exists():
        print(f"missing group file: {GROUP_FILE}", file=sys.stderr)
        return 1
    if not PARAM_FILE.exists():
        print(f"missing param file: {PARAM_FILE}", file=sys.stderr)
        return 1

    print(f"reading groups: {GROUP_FILE}")
    groups = parse_groups(GROUP_FILE)
    print(f"  parsed groups: {len(groups)}")

    print(f"reading params: {PARAM_FILE}")
    params = parse_params(PARAM_FILE)
    print(f"  parsed params: {len(params)}")

    paths = build_ltree_paths(groups)
    print(f"  ltree paths built: {len(paths)}")

    # ----- merge step --------------------------------------------------
    leaves_by_keyword: dict[str, list[GroupRow]] = {}
    for g in groups:
        if g.keyword:
            leaves_by_keyword.setdefault(g.keyword, []).append(g)
    print(f"  unique keywords across slices: {len(leaves_by_keyword)}")

    canonical_leaves = select_canonical_leaves(leaves_by_keyword)
    print(f"  canonical leaves (one per keyword): {len(canonical_leaves)}")

    groups_by_id = {g.old_id: g for g in groups}
    keep_ids = collect_ancestor_ids(canonical_leaves, groups_by_id)
    pre_merge_kept = [g for g in groups if g.old_id in keep_ids]
    print(f"  pre-merge kept groups: {len(pre_merge_kept)}")

    container_mapping = merge_same_name_containers(
        pre_merge_kept, canonical_leaves, paths, groups_by_id
    )
    merged_count = sum(1 for gid, c in container_mapping.items() if gid != c)
    print(f"  containers merged into canonical: {merged_count}")

    new_paths = rewrite_paths(pre_merge_kept, container_mapping, paths)

    # Drop non-canonical containers (their children now point to canonical).
    kept_groups = [
        g for g in pre_merge_kept if container_mapping.get(g.old_id, g.old_id) == g.old_id
    ]
    print(f"  final kept groups (canonical only): {len(kept_groups)}")

    # Commands attach to canonical container of their canonical leaf's parent
    # — but canonical_leaves themselves are leaves (own keyword), so their
    # group_id_map lookup still works since leaves are self-canonical.

    params_by_version: dict[str, list[ParamRow]] = {}
    for p in params:
        params_by_version.setdefault(p.param_version, []).append(p)

    group_rows_sql, group_id_map = build_groups_sql(kept_groups, new_paths)
    param_rows_sql = build_params_sql(params)
    cmd_rows_sql, cmd_id_map = build_commands_sql(
        canonical_leaves, group_id_map, leaves_by_keyword
    )
    sub_field_stmts = build_sub_fields_sql(
        canonical_leaves, leaves_by_keyword, params_by_version, cmd_id_map
    )

    print(f"  mml_param_groups rows: {len(group_rows_sql)}")
    print(f"  mml_params rows: {len(param_rows_sql)}")
    print(f"  mml_commands rows: {len(cmd_rows_sql)}")
    print(f"  mml_command_sub_fields stmts: {len(sub_field_stmts)}")

    OUTPUT_FILE.parent.mkdir(parents=True, exist_ok=True)
    emit_seed(
        OUTPUT_FILE,
        kept_groups,
        params,
        paths,
        group_id_map,
        group_rows_sql,
        param_rows_sql,
        cmd_rows_sql,
        sub_field_stmts,
        cmd_id_map,
    )
    print(f"wrote: {OUTPUT_FILE}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
