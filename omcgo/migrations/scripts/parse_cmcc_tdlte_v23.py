#!/usr/bin/env python3
"""
parse_cmcc_tdlte_v23.py — MML 控制台 v2.3 命令树离线解析器（dev-only）

方案文档：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §5.1

职责：
    解析规范 MD（cmcc-tdlte-southbound-data-model-v2.3.md）→ 生成结构化 JSON
    （omcgo/datamodels/mml-catalog/cmcc-tdlte-v23.json）。

约束（强调）：
    · 仅离线开发工具：开发者本地运行，产物随 PR 提交
    · 不进生产镜像 / Dockerfile / CI gate
    · 不被任何 Go 代码 import 或 shell out
    · 生产运行时仅消费生成的 JSON 文件（Catalog Loader）

用法：
    python3 parse_cmcc_tdlte_v23.py \
        --input  ../../规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md \
        --output ../../datamodels/mml-catalog/cmcc-tdlte-v23.json

    # 或使用默认路径（脚本目录的相对路径解析）
    python3 parse_cmcc_tdlte_v23.py

输出 JSON schema 见方案 §5.2 / model.go 的 Catalog 结构体。
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
from dataclasses import dataclass, asdict, field
from datetime import datetime, timezone
from pathlib import Path
from typing import List, Dict, Optional, Tuple

# ─────────────────────────────────────────────────────────────────────────────
# Path defaults (relative to script location)
# ─────────────────────────────────────────────────────────────────────────────
SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_INPUT = SCRIPT_DIR / ".." / ".." / "规范" / "移动" / "南向数据模型" / "cmcc-tdlte-southbound-data-model-v2.3.md"
DEFAULT_OUTPUT = SCRIPT_DIR / ".." / ".." / "datamodels" / "mml-catalog" / "cmcc-tdlte-v23.json"


# ─────────────────────────────────────────────────────────────────────────────
# Data classes (映射方案 §5.2 schema)
# ─────────────────────────────────────────────────────────────────────────────
@dataclass
class SubField:
    mmlCode: str
    standardPath: str
    labelI18n: Dict[str, str]
    accessType: str   # RO / RW
    valueType: str
    constraint: str
    defaultSelected: bool
    sortOrder: int


@dataclass
class Command:
    operationType: str          # LST / MOD / ADD / RMV
    targetPaths: List[str]


@dataclass
class Group:
    groupCode: str              # 归一化 TR-181 路径模板
    nameI18n: Dict[str, str]
    chapterCode: str            # SA…SR
    displayOrder: int
    instanceArity: int
    instanceLevels: List[str]
    commands: List[Command]     = field(default_factory=list)
    subFields: List[SubField]   = field(default_factory=list)


@dataclass
class Catalog:
    specVersion: str
    carrier: str
    tech: str
    generatedAt: str
    sourceDocSha256: str
    groups: List[Group]         = field(default_factory=list)


# ─────────────────────────────────────────────────────────────────────────────
# Regex / 常量
# ─────────────────────────────────────────────────────────────────────────────

# ## SA - DeviceInfo
H2_CHAPTER_RE = re.compile(r"^##\s+(S[A-Z])\s+-\s+(.+?)\s*$")

# #### 命令: Device.DeviceInfo.* 📖📝
H4_COMMAND_RE = re.compile(r"^####\s+命令:\s+(\S+)(?:\s+([📖📝]+))?\s*$")

# 表格行（TR-181 是第 3 列）：
# | 1 | `IGD.x` | `Device.X.UserLabel` | UserLabel | 用户友好名 | 📝 RW | string |
TABLE_ROW_RE = re.compile(
    r"^\|\s*\d+\s*\|\s*`([^`]+)`\s*\|\s*`([^`]+)`\s*\|\s*"
    r"([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*([📖📝]\s+[RW]+)\s*\|\s*([^|]+?)\s*\|\s*$"
)

# 多层 {iα}/{iβ}/{iγ}/{iδ}/{iε} → 折叠为 {i}
GREEK_INSTANCE_RE = re.compile(r"\{i[αβγδεζηθικλμνξοπρστυφχψω]\}")
PLAIN_INSTANCE_RE = re.compile(r"\{i\}")

# 伪命令过滤（SQ 段的 `.*`）
PSEUDO_COMMAND_PATTERNS = {".*"}


# ─────────────────────────────────────────────────────────────────────────────
# 解析器
# ─────────────────────────────────────────────────────────────────────────────
class Parser:
    def __init__(self, md_path: Path):
        self.md_path = md_path
        self.groups: List[Group] = []
        self._chapter_code: Optional[str] = None
        self._chapter_name: Optional[str] = None
        self._cur_group: Optional[Group] = None
        self._display_order = 0
        self._content: str = ""
        self._sha256: str = ""

    def parse(self) -> Catalog:
        raw = self.md_path.read_bytes()
        self._sha256 = hashlib.sha256(raw).hexdigest()
        self._content = raw.decode("utf-8")

        lines = self._content.splitlines()
        i = 0
        while i < len(lines):
            line = lines[i]

            m_h2 = H2_CHAPTER_RE.match(line)
            if m_h2:
                self._chapter_code = m_h2.group(1)
                self._chapter_name = m_h2.group(2).strip()
                i += 1
                continue

            m_h4 = H4_COMMAND_RE.match(line)
            if m_h4 and self._chapter_code:
                path_template = m_h4.group(1)
                perm_marker = (m_h4.group(2) or "")  # 📖 / 📝 / 📖📝

                # 过滤伪命令
                if path_template in PSEUDO_COMMAND_PATTERNS:
                    i += 1
                    continue

                # 收集本段表格
                table_rows, i = self._consume_table(lines, i + 1)
                group = self._build_group(path_template, perm_marker, table_rows)
                if group is not None:
                    self.groups.append(group)
                continue

            i += 1

        return Catalog(
            specVersion="cmcc-tdlte-v2.3",
            carrier="cmcc",
            tech="lte",
            generatedAt=datetime.now(timezone.utc).isoformat(timespec="seconds"),
            sourceDocSha256=self._sha256,
            groups=self.groups,
        )

    # 读取一个 H4 块对应的表格行（直到下一个 H2/H4/分隔线）
    def _consume_table(self, lines: List[str], start: int) -> Tuple[List[Dict], int]:
        rows: List[Dict] = []
        i = start
        while i < len(lines):
            line = lines[i]
            if H4_COMMAND_RE.match(line) or H2_CHAPTER_RE.match(line):
                break
            m = TABLE_ROW_RE.match(line)
            if m:
                tr098, tr181, name, name_zh, perm, vtype = m.groups()
                rows.append({
                    "tr181_path": tr181.strip(),
                    "param_name": name.strip(),
                    "name_zh": name_zh.strip(),
                    "perm": perm.strip(),
                    "vtype": vtype.strip(),
                })
            i += 1
        return rows, i

    def _build_group(
        self,
        path_template: str,
        perm_marker: str,
        rows: List[Dict],
    ) -> Optional[Group]:
        if not rows:
            return None

        self._display_order += 1
        # 多层希腊字母实例 → 折叠为 {i}
        normalized = GREEK_INSTANCE_RE.sub("{i}", path_template)
        instance_arity = len(PLAIN_INSTANCE_RE.findall(normalized))
        instance_levels = self._extract_instance_levels(normalized)

        # 子字段：path → sub_field
        sub_fields: List[SubField] = []
        any_rw = False
        for idx, row in enumerate(rows):
            sf_path = GREEK_INSTANCE_RE.sub("{i}", row["tr181_path"])
            access = "RW" if "RW" in row["perm"] else "RO"
            if access == "RW":
                any_rw = True
            mml_code = self._derive_mml_code(sf_path)
            sub_fields.append(SubField(
                mmlCode=mml_code,
                standardPath=sf_path,
                labelI18n={"zh-CN": row["name_zh"], "en-US": row["param_name"]},
                accessType=access,
                valueType=self._extract_value_type(row["vtype"]),
                constraint=row["vtype"],
                defaultSelected=True,
                sortOrder=idx + 1,
            ))

        # 按权限矩阵生成 commands
        commands: List[Command] = []
        all_paths = [sf.standardPath for sf in sub_fields]
        rw_paths = [sf.standardPath for sf in sub_fields if sf.accessType == "RW"]

        # LST 恒生成
        commands.append(Command(operationType="LST", targetPaths=all_paths))
        # MOD：有 RW 时生成
        if rw_paths:
            commands.append(Command(operationType="MOD", targetPaths=rw_paths))
        # ADD/RMV：含 {i} 且有 RW 时生成（D26 启发式，后续可由 overlay 收紧）
        if instance_arity > 0 and rw_paths:
            commands.append(Command(operationType="ADD", targetPaths=[normalized]))
            commands.append(Command(operationType="RMV", targetPaths=[normalized]))

        # 中文别名：路径末段对象名 → 表格内对应中文
        chinese_alias = self._infer_chinese_alias(normalized, sub_fields)
        english_alias = self._infer_english_alias(normalized)

        return Group(
            groupCode=normalized,
            nameI18n={"zh-CN": chinese_alias, "en-US": english_alias},
            chapterCode=self._chapter_code or "",
            displayOrder=self._display_order,
            instanceArity=instance_arity,
            instanceLevels=instance_levels,
            commands=commands,
            subFields=sub_fields,
        )

    @staticmethod
    def _derive_mml_code(standard_path: str) -> str:
        # leaf 段
        parts = standard_path.rstrip(".").split(".")
        if not parts:
            return ""
        leaf = parts[-1]
        # 防止以数字开头（Python 兼容名）
        if leaf and leaf[0].isdigit():
            leaf = "X" + leaf
        return leaf

    @staticmethod
    def _extract_value_type(constraint: str) -> str:
        # "string(256)" → "string"; "unsignedInt[0:65535]" → "unsignedInt"
        base = re.split(r"[\(\[]", constraint, 1)[0]
        return base.strip()

    @staticmethod
    def _extract_instance_levels(normalized_path: str) -> List[str]:
        # Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.{i}.*
        # → ["MU", "Slot", "EU", "RU", "RFChannel"]
        levels: List[str] = []
        segs = normalized_path.split("{i}")
        for seg in segs[:-1]:
            tail = seg.rstrip(".").split(".")
            if tail:
                levels.append(tail[-1])
        return levels

    @staticmethod
    def _infer_chinese_alias(normalized_path: str, sub_fields: List[SubField]) -> str:
        # 取路径末段 object 英文名 → 在 sub_fields.param_name 找匹配 → 取中文
        # 例：Device.DeviceInfo.SwUpgrade.* → object="SwUpgrade" → 找 sub_field "SwUpgrade"
        # 若不可推断，回落英文路径末段
        cleaned = normalized_path.rstrip(".*").rstrip(".")
        if cleaned.endswith("{i}"):
            cleaned = cleaned[:-3].rstrip(".")
        object_name = cleaned.split(".")[-1] if cleaned else "Unknown"
        # 启发：使用 object_name 自身（其大多数已经有中文映射的 hard-coded 表）
        return _CHINESE_OBJECT_ALIASES.get(object_name, object_name)

    @staticmethod
    def _infer_english_alias(normalized_path: str) -> str:
        cleaned = normalized_path.rstrip(".*").rstrip(".")
        if cleaned.endswith("{i}"):
            cleaned = cleaned[:-3].rstrip(".")
        return cleaned.split(".")[-1] if cleaned else "Unknown"


# 部分核心 object → 中文别名的静态映射（其余由 _infer_chinese_alias 兜底返回英文）。
# 完整覆盖建议在生成 JSON 后人工补齐（catalog-listing.md 是参考视图）。
_CHINESE_OBJECT_ALIASES: Dict[str, str] = {
    "DeviceInfo":           "设备信息",
    "SwUpgrade":            "设备版本升级",
    "SoftwareCtrl":         "软件控制",
    "ManagementServer":     "网管参数",
    "FaultMgmt":            "故障管理",
    "CurrentAlarm":         "当前告警实例",
    "ExpeditedEvent":       "实时告警实例",
    "HistoryEvent":         "历史告警实例",
    "QueuedEvent":          "队列告警实例",
    "SupportedAlarm":       "支持告警实例",
    "LogMgmt":              "日志管理",
    "Gateway":              "安全网关",
    "MmePoolConfigParam":   "MME 池配置",
    "S1U":                  "S1U 接口",
    "X2IpAddrMapInfo":      "X2 IP 映射",
    "FAPService":           "FAPService 载波",
    "Capabilities":         "能力集",
    "EPC":                  "EPC",
    "PLMNList":             "PLMN 列表",
    "PdcpInitParam":        "PDCP 初始参数",
    "SCTP":                 "SCTP",
    "Assoc":                "SCTP Assoc",
    "MAC":                  "MAC",
    "DrxInitialParam":      "DRX 初始",
    "PHY":                  "PHY",
    "MBSFN":                "MBSFN",
    "SFConfigList":         "MBSFN 子帧配置",
    "GSM":                  "GSM 邻区",
    "NR":                   "NR 邻区",
    "UMTS":                 "UMTS 邻区",
    "LTECell":              "LTE 邻区",
    "EUTRA":                "EUTRA 测量",
    "A1MeasureCtrl":        "A1 测量控制",
    "A2MeasureCtrl":        "A2 测量控制",
    "A3MeasureCtrl":        "A3 测量控制",
    "A4MeasureCtrl":        "A4 测量控制",
    "A5MeasureCtrl":        "A5 测量控制",
    "PeriodMeasCtrl":       "周期测量控制",
    "IRAT":                 "IRAT 测量",
    "B1MeasureCtrl":        "B1 测量控制",
    "B2MeasureCtrl":        "B2 测量控制",
    "IdleMode":             "IdleMode",
    "GERAN":                "GERAN",
    "GERANFreqGroup":       "GERAN 频组",
    "UTRA":                 "UTRA",
    "UTRANFDDFreq":         "UTRA FDD 频点",
    "InterFreq":            "异频",
    "Carrier":              "异频载波",
    "SONConfigParam":       "SON 配置",
    "SelfConfig":           "自配置启动",
    "Interface":            "以太网接口",
    "IPv4Address":          "IPv4 地址",
    "IPv6Address":          "IPv6 地址",
    "VlanInterface":        "VLAN 接口",
    "IpRoute":              "IP 路由",
    "IPsec":                "IPsec",
    "Time":                 "时间服务器",
    "GPS":                  "GPS",
    "MRMgmt":               "MR 配置",
    "PerfMgmt":             "PM 配置",
    "Config":               "配置",
    "MU":                   "MU 主机单元",
    "Slot":                 "Slot 板卡",
    "EU":                   "EU 扩展单元",
    "RU":                   "RU 远端单元",
    "RFChannel":            "RFChannel 射频通道",
}


# ─────────────────────────────────────────────────────────────────────────────
# 入口
# ─────────────────────────────────────────────────────────────────────────────
def main() -> int:
    ap = argparse.ArgumentParser(description="MML v2.3 catalog MD → JSON 转换器（dev-only）")
    ap.add_argument("--input",  default=str(DEFAULT_INPUT),  help="规范 MD 文件路径")
    ap.add_argument("--output", default=str(DEFAULT_OUTPUT), help="JSON 输出路径")
    ap.add_argument("--dry-run", action="store_true",        help="仅打印汇总信息，不写文件")
    args = ap.parse_args()

    in_path = Path(args.input).resolve()
    out_path = Path(args.output).resolve()

    if not in_path.exists():
        print(f"error: input file not found: {in_path}", file=sys.stderr)
        return 2

    parser = Parser(in_path)
    catalog = parser.parse()

    # 汇总
    total_commands = sum(len(g.commands) for g in catalog.groups)
    total_paths = sum(len(sf.standardPath) > 0 for g in catalog.groups for sf in g.subFields)
    print(
        f"parsed: {len(catalog.groups)} groups, "
        f"{total_commands} per-op commands, "
        f"{total_paths} sub_fields"
    )

    if args.dry_run:
        return 0

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(
        json.dumps(asdict(catalog), ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    print(f"wrote: {out_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
