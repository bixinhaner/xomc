#!/usr/bin/env python3
"""
parse_cmcc_tdlte_v23_v2.py — MML 控制台 v2.3 命令树离线解析器 (v2 schema)

方案文档：omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md
（v2 修订：§R-1 / §R-2 / §R-2.4 / §R-2.5 / §R-3.1 / §R-3.2 / §R-4.1.1 / §R-4.2.1）

职责：
    解析规范 MD → 生成 v2 schema 结构化 JSON
    产物：omcgo/datamodels/mml-catalog/cmcc-tdlte-v2.3.v2.json

约束（强调）：
    · dev-only 离线工具：开发者本地运行，产物随 PR 提交
    · 不进生产镜像 / Dockerfile / CI gate
    · 不被任何 Go 代码 import 或 shell out
    · 生产运行时由 Go catalog Loader 消费 v2.json 文件

与旧 parser 的差异（v2 vs v1）：
    · 一级 group = 18 个 SA-SR 章节（group_code = "chapter:SA"），不再是 71 个 object 组
    · command 显示名 = §R-2.4 全中文命名权威表，未命中 panic
    · 同 group_code 跨章节合并到首次出现章节（§R-3.1）
    · ADD/RMV 按 §R-3.2 非可创建对象清单过滤
    · mml_commands 不再存 path 字符串，改存 tree_node_refs（standardPath 列表，§R-2.5）
    · 每条命令解析 {i} 取值范围 → instance_range_meta（§R-4.1.1）
    · params[] 块附带每条 standardPath 的类型 / 约束 / 提示文字（§R-4.2.1）

用法：
    python3 parse_cmcc_tdlte_v23_v2.py
    python3 parse_cmcc_tdlte_v23_v2.py --dry-run
    python3 parse_cmcc_tdlte_v23_v2.py --input <spec.md> --output <out.json>
"""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional


# ─────────────────────────────────────────────────────────────────────────────
# 默认路径
# ─────────────────────────────────────────────────────────────────────────────
SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_INPUT = (
    SCRIPT_DIR
    / ".."
    / ".."
    / "规范"
    / "移动"
    / "南向数据模型"
    / "cmcc-tdlte-southbound-data-model-v2.3.md"
)
DEFAULT_OUTPUT = (
    SCRIPT_DIR / ".." / ".." / "datamodels" / "mml-catalog" / "cmcc-tdlte-v2.3.v2.json"
)


# ─────────────────────────────────────────────────────────────────────────────
# §R-2.4 命令中文名权威表（71 条 standard 命令，跨章节唯一）
# ─────────────────────────────────────────────────────────────────────────────
COMMAND_ZH_NAME: dict[str, str] = {
    # SA
    "Device.DeviceInfo.*": "设备基本信息",
    "Device.DeviceInfo.SwUpgrade.*": "设备软件升级状态",
    # SB
    "Device.SoftwareCtrl.*": "软件版本控制",
    # SC
    "Device.ManagementServer.*": "基站网管连接",
    # SD
    "Device.FaultMgmt.*": "故障管理总览",
    "Device.FaultMgmt.CurrentAlarm.{i}.*": "当前告警实例",
    "Device.FaultMgmt.ExpeditedEvent.{i}.*": "实时告警实例",
    "Device.FaultMgmt.HistoryEvent.{i}.*": "历史告警实例",
    "Device.FaultMgmt.QueuedEvent.{i}.*": "队列告警实例",
    "Device.FaultMgmt.SupportedAlarm.{i}.*": "支持告警类型",
    # SE
    "Device.LogMgmt.*": "日志管理配置",
    # SF
    "Device.Services.FAPControl.LTE.*": "LTE 接入控制",
    "Device.Services.FAPControl.LTE.Gateway.*": "安全接入网关",
    "Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.*": "MME 池配置",
    "Device.Services.FAPControl.LTE.S1U.{i}.*": "S1U 用户面接口",
    "Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*": "X2 接口 IP 映射",
    "Device.Services.FAPService.{i}.*": "FAP 载波基本配置",
    "Device.Services.FAPService.{i}.Capabilities.*": "FAP 载波能力集",
    "Device.Services.FAPService.{i}.CellConfig.Capabilities.*": "小区配置能力集",
    "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.*": "EPC 核心网参数",
    "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.*": "EPC PLMN 列表",
    "Device.Services.FAPService.{i}.CellConfig.LTE.VoLTE.PdcpInitParam.{i}.*": "VoLTE PDCP 初始参数",
    # SG
    "Device.Services.FAPControl.Transport.SCTP.*": "SCTP 协议配置",
    "Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.*": "SCTP 关联状态",
    # SH
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.*": "RAN MAC 协议层",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.*": "MAC DRX 初始参数",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.*": "RAN PHY 物理层",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.*": "PHY MBSFN 子帧",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.*": "MBSFN 子帧配置列表",
    # SI
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.*": "GSM 异系统邻区",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.*": "NR 异系统邻区",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.*": "UMTS 异系统邻区",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.*": "LTE 同系统邻区",
    # SJ
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.*": "连接态 EUTRA 测量",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.*": "A1 事件测量控制",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.*": "A2 事件测量控制",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.*": "A3 事件测量控制",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.*": "A4 事件测量控制",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.*": "A5 事件测量控制",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.*": "EUTRA 周期测量控制",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.*": "连接态 IRAT 测量",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.*": "B1 事件测量控制",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.*": "B2 事件测量控制",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.*": "空闲态移动性",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.*": "空闲态 IRAT 移动性",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.*": "空闲态 GERAN 频组",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.*": "空闲态 UTRA FDD 频点",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.*": "空闲态异频载波",
    # SK
    "Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.*": "SON 自配置参数",
    "Device.Services.FAPService.{i}.FAPControl.SelfConfig.*": "自配置启动状态",
    # SL
    "Device.Ethernet.Interface.{i}.*": "以太网接口",
    "Device.Ethernet.Interface.{i}.IPv4Address.{i}.*": "接口 IPv4 地址",
    "Device.Ethernet.Interface.{i}.IPv6Address.{i}.*": "接口 IPv6 地址",
    "Device.Ethernet.Interface.{i}.VlanInterface.{i}.*": "VLAN 子接口",
    "Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.*": "VLAN 子接口 IPv4 地址",
    "Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.*": "VLAN 子接口 IPv6 地址",
    "Device.Ethernet.IpRoute.{i}.*": "静态路由表项",
    # SM
    "Device.IPsec.*": "IPsec 安全配置",
    # SN
    "Device.Time.*": "时间同步服务器",
    # SO
    "Device.FAP.GPS.*": "GPS 定位信息",
    # SP
    "Device.FAP.MRMgmt.Config.{i}.*": "MR 上报配置",
    # SQ
    "Device.FAP.PerfMgmt.Config.{i}.*": "PM 性能上报配置",
    # SR
    "Device.DeviceInfo.MU.{i}.*": "主机单元基本信息",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.*": "槽位板卡信息",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.*": "扩展单元信息",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.*": "射频远端单元信息",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.{i}.*": "射频通道信息",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.*": "射频单元软件升级",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SwUpgrade.*": "扩展单元软件升级",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.SwUpgrade.*": "板卡软件升级",
    "Device.DeviceInfo.MU.{i}.SwUpgrade.*": "主机单元软件升级",
}


# ─────────────────────────────────────────────────────────────────────────────
# §R-3.2 非可创建对象清单（17 条 path 模板，命中者不生成 ADD/RMV）
# ─────────────────────────────────────────────────────────────────────────────
NON_CREATABLE: set[str] = {
    "Device.Services.FAPService.{i}.*",
    "Device.Services.FAPService.{i}.Capabilities.*",
    "Device.Services.FAPService.{i}.CellConfig.Capabilities.*",
    "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.*",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.*",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.*",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.*",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.*",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.*",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.*",
    "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.*",
    "Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.*",
    "Device.DeviceInfo.MU.{i}.*",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.*",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.*",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.*",
    "Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.{i}.*",
}


# ─────────────────────────────────────────────────────────────────────────────
# Regex / 常量
# ─────────────────────────────────────────────────────────────────────────────
H2_CHAPTER_RE = re.compile(r"^##\s+(S[A-Z])\s+-\s+(.+?)\s*$")
H4_COMMAND_RE = re.compile(r"^####\s+命令:\s+(\S+)(?:\s+([📖📝]+))?\s*$")
ZHNAME_RE = re.compile(r"^\*\*(.+?)\*\*\s*$")

# 取值范围行：- **{i}** 取值范围: `<expr>` — <description>
#       或：- **{iα}** 取值范围: `<expr>` — <description>
RANGE_RE = re.compile(
    r"^\s*-\s*\*\*\{i[αβγδεζηθικλμνξοπρστυφχψω]?\}\*\*\s*取值范围[:：]\s*"
    r"`?([^`—\-]+?)`?\s*[—\-]\s*(.+?)\s*$"
)

# 表格行（TR-181 是第 3 列）：
# | 1 | `IGD.x` | `Device.X.UserLabel` | UserLabel | 用户友好名 | 📝 RW | string |
TABLE_ROW_RE = re.compile(
    r"^\|\s*\d+\s*\|\s*`([^`]+)`\s*\|\s*`([^`]+)`\s*\|\s*"
    r"([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*([📖📝]\s+[RW]+)\s*\|\s*([^|]+?)\s*\|\s*$"
)

# 多层 {iα}/{iβ}/... → 折叠为 {i}
GREEK_RE = re.compile(r"\{i[αβγδεζηθικλμνξοπρστυφχψω]\}")
PSEUDO_COMMAND = ".*"

# 类型语法解析（§R-4.2.1）
TYPE_STRING_LEN_RE = re.compile(r"^string\((\d+)\)\s*$")
TYPE_HEXBIN_LEN_RE = re.compile(r"^hexBinary\((\d+)\)\s*$")
TYPE_INT_RANGE_RE = re.compile(
    r"^(unsignedInt|int|long|unsignedLong)\[(-?\d+):(-?\d+)\]\s*$"
)
TYPE_INT_LOWER_RE = re.compile(r"^(unsignedInt|int|long|unsignedLong)\[(-?\d+):\]\s*$")
TYPE_ENUM_RE = re.compile(r"^enum\((.+?)\)\s*$")
TYPE_BARE_RE = re.compile(r"^([a-zA-Z]+)(?:\([^)]*\))?(?:\[[^\]]*\])?\s*$")


# ─────────────────────────────────────────────────────────────────────────────
# 数据类
# ─────────────────────────────────────────────────────────────────────────────
@dataclass
class InstanceRange:
    layer: int            # 1-based, 多层 {i} 各自一行
    range_expr: str       # 原始 expr，如 "0~N" / "1~3" / "1~24"
    range_min: Optional[int]
    range_max: Optional[int]
    dynamic: bool         # True if max is dynamic (N from another param)
    n_source: Optional[str]  # e.g. "MaxCurrentAlarmEntries"
    description: str      # 描述文本


@dataclass
class ParamRow:
    standard_path: str
    param_name: str
    label_zh: str
    access_type: str         # RO / RW
    value_type: str          # string / unsignedInt / boolean / dateTime / ...
    constraint: str          # 原始 "类型" 列字符串，如 "string(64)" / "unsignedInt[0:65535]"
    constraint_hint: str     # 派生的中文文字提示
    length_max: Optional[int]
    range_min: Optional[int]
    range_max: Optional[int]


@dataclass
class CommandLeaf:
    command_code: str                       # "<OP>:<groupCodeObject>"
    operation_type: str                     # LST / MOD / ADD / RMV
    rpc_method: str                         # GetParameterValues / SetParameterValues / AddObject / DeleteObject
    logical_name_i18n: dict[str, str]       # {"zh-CN": <command_zh_name>}
    group_code_object: str                  # 归一化 TR-181 path 模板
    instance_arity: int                     # {i} 占位符层数
    instance_levels: list[str]              # 每层 {i} 的对象段名
    instance_range_meta: list[dict]         # §R-4.1.1 每层 {i} 的范围 metadata
    tree_node_refs: list[str]               # §R-2.5 引用 standard_params 的 standardPath 键


@dataclass
class CatalogGroup:
    group_code: str                         # "chapter:SA"
    chapter_code: str                       # SA-SR
    name_i18n: dict[str, str]               # {"zh-CN": "设备信息参数管理"}
    display_order: int                      # SA=1 ... SR=18
    commands: list[CommandLeaf] = field(default_factory=list)


@dataclass
class MergeLogEntry:
    group_code: str
    owner_chapter: str
    merged_chapter: str
    added_path_count: int


@dataclass
class NonCreatableHit:
    group_code_object: str
    filtered_ops: list[str]


@dataclass
class Catalog:
    spec_version: str
    schema_version: str
    carrier: str
    tech: str
    generated_at: str
    source_doc_sha256: str
    stats: dict
    groups: list[CatalogGroup]
    params: list[ParamRow]
    merge_log: list[MergeLogEntry]
    non_creatable_hits: list[NonCreatableHit]


# ─────────────────────────────────────────────────────────────────────────────
# 解析辅助
# ─────────────────────────────────────────────────────────────────────────────
def parse_range_expr(expr: str, desc: str) -> InstanceRange:
    """Parse e.g. '0~N' / '1~3' / '1~24' / '0~63' into structured InstanceRange.

    Description like '当前告警实例编号，N由MaxCurrentAlarmEntries决定' gives
    n_source. Dynamic = True if max contains 'N' (uppercase) or similar.
    """
    expr = expr.strip().strip("`")
    n_source: Optional[str] = None
    range_min: Optional[int] = None
    range_max: Optional[int] = None
    dynamic = False

    # try '<num>~<num>' or '<num>~N'
    m = re.match(r"^(-?\d+)\s*[~\-]\s*(-?\d+|N)\s*$", expr)
    if m:
        range_min = int(m.group(1))
        tail = m.group(2)
        if tail == "N":
            dynamic = True
            range_max = None
            # n_source from desc
            ms = re.search(r"N\s*由\s*([A-Za-z][A-Za-z0-9]*)\s*决定", desc)
            if ms:
                n_source = ms.group(1)
        else:
            range_max = int(tail)
            dynamic = False
    else:
        # 未知格式：原样保留
        dynamic = True

    return InstanceRange(
        layer=0,           # 上层填充
        range_expr=expr,
        range_min=range_min,
        range_max=range_max,
        dynamic=dynamic,
        n_source=n_source,
        description=desc.strip(),
    )


def parse_type_constraint(raw: str) -> tuple[str, str, Optional[int], Optional[int], Optional[int]]:
    """Parse '类型' column → (value_type, hint, length_max, range_min, range_max).

    Returns hint in Chinese per §R-4.2.1 table.
    """
    raw = raw.strip()
    if not raw:
        return "string", "字符串", None, None, None

    m = TYPE_STRING_LEN_RE.match(raw)
    if m:
        n = int(m.group(1))
        return "string", f"字符串，长度 ≤ {n}", n, None, None

    m = TYPE_HEXBIN_LEN_RE.match(raw)
    if m:
        n = int(m.group(1))
        return "hexBinary", f"十六进制字符串，长度 ≤ {n} 字节", n, None, None

    m = TYPE_INT_RANGE_RE.match(raw)
    if m:
        vt = m.group(1)
        mn = int(m.group(2))
        mx = int(m.group(3))
        signed = "无符号" if vt.startswith("unsigned") else "有符号"
        return vt, f"{signed}整数 [{mn}, {mx}]", None, mn, mx

    m = TYPE_INT_LOWER_RE.match(raw)
    if m:
        vt = m.group(1)
        mn = int(m.group(2))
        signed = "无符号" if vt.startswith("unsigned") else "有符号"
        return vt, f"{signed}整数（≥ {mn}）", None, mn, None

    m = TYPE_ENUM_RE.match(raw)
    if m:
        items = m.group(1)
        return "enum", f"枚举值之一：{items}", None, None, None

    # 裸类型（无约束）
    if raw == "string":
        return "string", "字符串", None, None, None
    if raw == "boolean":
        return "boolean", "`true` / `false`", None, None, None
    if raw == "dateTime":
        return "dateTime", "ISO 8601 时间（例 `2026-05-21T10:00:00Z`）", None, None, None
    if raw == "unsignedInt":
        return "unsignedInt", "无符号整数（≥ 0）", None, None, None
    if raw == "int":
        return "int", "有符号整数", None, None, None
    if raw == "long":
        return "long", "64 位有符号整数", None, None, None
    if raw == "unsignedLong":
        return "unsignedLong", "64 位无符号整数", None, None, None

    # fallback：未识别语法 → 原样作为提示
    m = TYPE_BARE_RE.match(raw)
    if m:
        vt = m.group(1)
        return vt, raw, None, None, None
    return "unknown", raw, None, None, None


def derive_instance_levels(norm_path: str) -> list[str]:
    """Extract per-layer instance level names from normalized path.

    e.g. 'Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.*' →
         ['MU', 'Slot', 'EU']
    """
    levels = []
    segs = norm_path.rstrip(".*").rstrip(".").split(".")
    for i, seg in enumerate(segs):
        if seg == "{i}" and i > 0:
            levels.append(segs[i - 1])
    return levels


def parent_object_path(norm_path: str) -> str:
    cleaned = norm_path
    if cleaned.endswith(".*"):
        cleaned = cleaned[:-1]  # keep trailing dot
    return cleaned


# ─────────────────────────────────────────────────────────────────────────────
# 主解析器
# ─────────────────────────────────────────────────────────────────────────────
class Parser:
    def __init__(self, md_path: Path):
        self.md_path = md_path
        self._chapters: list[dict] = []
        self._sha256 = ""

    def parse(self) -> Catalog:
        raw_bytes = self.md_path.read_bytes()
        self._sha256 = hashlib.sha256(raw_bytes).hexdigest()
        lines = raw_bytes.decode("utf-8").splitlines()

        self._collect_chapters(lines)
        merge_log = self._merge_same_group_codes()
        groups, all_params, non_creatable_hits = self._build_groups_and_params()

        total_op_leaves = sum(len(g.commands) for g in groups)
        stats = {
            "chapters": len(groups),
            "objectsTotal": sum(
                len({c.group_code_object for c in g.commands}) for g in groups
            ),
            "commandLeavesTotal": total_op_leaves,
            "uniqueParams": len(all_params),
            "mergeEvents": len(merge_log),
            "nonCreatableFiltered": len(non_creatable_hits),
        }

        return Catalog(
            spec_version="cmcc-tdlte-v2.3",
            schema_version="v2",
            carrier="cmcc",
            tech="lte",
            generated_at=datetime.now(timezone.utc).isoformat(timespec="seconds"),
            source_doc_sha256=self._sha256,
            stats=stats,
            groups=groups,
            params=all_params,
            merge_log=merge_log,
            non_creatable_hits=non_creatable_hits,
        )

    # ----- 步骤 1：扫描 spec MD 收集 chapters / commands / rows -----
    def _collect_chapters(self, lines: list[str]) -> None:
        cur: Optional[dict] = None
        skip_block = False
        i = 0
        while i < len(lines):
            line = lines[i].rstrip()
            # 跳过派生 / 已迁移区块，避免重复抓
            if line.startswith("## 分组+命令的结构图") or line.startswith(
                "## 参数管理类别分组清单"
            ):
                skip_block = True
                i += 1
                continue
            if skip_block:
                if H2_CHAPTER_RE.match(line):
                    skip_block = False
                else:
                    i += 1
                    continue

            m_h2 = H2_CHAPTER_RE.match(line)
            if m_h2:
                if cur:
                    self._chapters.append(cur)
                cur = {
                    "code": m_h2.group(1),
                    "obj_root": m_h2.group(2).strip(),
                    "zh": "",
                    "commands": [],
                }
                i += 1
                continue

            if cur and not cur["zh"]:
                m_zh = ZHNAME_RE.match(line)
                if m_zh:
                    cur["zh"] = m_zh.group(1)
                    i += 1
                    continue

            if cur and (m_h4 := H4_COMMAND_RE.match(line)):
                path = m_h4.group(1)
                perm = m_h4.group(2) or ""
                if path == PSEUDO_COMMAND:
                    i += 1
                    continue
                norm = GREEK_RE.sub("{i}", path)
                cmd = {
                    "norm": norm,
                    "perm": perm,
                    "rows": [],
                    "ranges": [],   # parsed instance ranges
                }
                i = self._consume_command_body(lines, i + 1, cmd)
                cur["commands"].append(cmd)
                continue

            i += 1

        if cur:
            self._chapters.append(cur)

    def _consume_command_body(self, lines: list[str], start: int, cmd: dict) -> int:
        """Consume table rows + range annotations until next H4/H2."""
        i = start
        layer_counter = 0
        while i < len(lines):
            line = lines[i].rstrip()
            if H4_COMMAND_RE.match(line) or H2_CHAPTER_RE.match(line):
                break
            # range annotation
            m_range = RANGE_RE.match(line)
            if m_range:
                layer_counter += 1
                expr = m_range.group(1)
                desc = m_range.group(2)
                ir = parse_range_expr(expr, desc)
                ir.layer = layer_counter
                cmd["ranges"].append(ir)
                i += 1
                continue
            # table row
            m_row = TABLE_ROW_RE.match(line)
            if m_row:
                tr181 = m_row.group(2).strip()
                cmd["rows"].append(
                    {
                        "path": GREEK_RE.sub("{i}", tr181),
                        "param_name": m_row.group(3).strip(),
                        "name_zh": m_row.group(4).strip(),
                        "perm": m_row.group(5).strip(),
                        "vtype": m_row.group(6).strip(),
                    }
                )
            i += 1
        return i

    # ----- 步骤 2：§R-3.1 同 group_code 跨章节合并 -----
    def _merge_same_group_codes(self) -> list[MergeLogEntry]:
        seen: dict[str, tuple[int, int]] = {}  # norm -> (chapter_idx, cmd_idx_in_chapter)
        merge_log: list[MergeLogEntry] = []

        for ch_idx, ch in enumerate(self._chapters):
            new_cmds = []
            for cmd in ch["commands"]:
                if cmd["norm"] in seen:
                    owner_ci, owner_cmi = seen[cmd["norm"]]
                    owner_cmd = self._chapters[owner_ci]["commands"][owner_cmi]
                    existing_paths = {r["path"] for r in owner_cmd["rows"]}
                    added = 0
                    for row in cmd["rows"]:
                        if row["path"] not in existing_paths:
                            owner_cmd["rows"].append(row)
                            existing_paths.add(row["path"])
                            added += 1
                    for emoji in ("📖", "📝"):
                        if (
                            emoji in cmd["perm"]
                            and emoji not in owner_cmd["perm"]
                        ):
                            owner_cmd["perm"] += emoji
                    # 合并 ranges：以 owner 为准（同 group_code 通常 ranges 一致）
                    if not owner_cmd["ranges"] and cmd["ranges"]:
                        owner_cmd["ranges"] = cmd["ranges"]
                    merge_log.append(
                        MergeLogEntry(
                            group_code=cmd["norm"],
                            owner_chapter=self._chapters[owner_ci]["code"],
                            merged_chapter=ch["code"],
                            added_path_count=added,
                        )
                    )
                    continue
                seen[cmd["norm"]] = (ch_idx, len(new_cmds))
                new_cmds.append(cmd)
            ch["commands"] = new_cmds

        return merge_log

    # ----- 步骤 3：派生 op leaves + params + non-creatable hits -----
    def _build_groups_and_params(
        self,
    ) -> tuple[list[CatalogGroup], list[ParamRow], list[NonCreatableHit]]:
        groups: list[CatalogGroup] = []
        param_by_path: dict[str, ParamRow] = {}
        nc_hits: list[NonCreatableHit] = []
        missing_names: list[str] = []

        for idx, ch in enumerate(self._chapters):
            display_order = ord(ch["code"][1]) - ord("A") + 1
            grp = CatalogGroup(
                group_code=f"chapter:{ch['code']}",
                chapter_code=ch["code"],
                name_i18n={"zh-CN": ch["zh"]},
                display_order=display_order,
            )
            for cmd in ch["commands"]:
                # 收集 params 到全局表（去重）
                for row in cmd["rows"]:
                    if row["path"] in param_by_path:
                        continue
                    vt, hint, lmax, rmin, rmax = parse_type_constraint(row["vtype"])
                    access = "RW" if "RW" in row["perm"] else "RO"
                    param_by_path[row["path"]] = ParamRow(
                        standard_path=row["path"],
                        param_name=row["param_name"],
                        label_zh=row["name_zh"],
                        access_type=access,
                        value_type=vt,
                        constraint=row["vtype"],
                        constraint_hint=hint,
                        length_max=lmax,
                        range_min=rmin,
                        range_max=rmax,
                    )

                # 派生 op leaves
                norm = cmd["norm"]
                if norm not in COMMAND_ZH_NAME:
                    missing_names.append(norm)
                    continue
                zh_name = COMMAND_ZH_NAME[norm]
                all_paths = [r["path"] for r in cmd["rows"]]
                rw_paths = [r["path"] for r in cmd["rows"] if "RW" in r["perm"]]
                has_i = "{i}" in norm
                has_rw = len(rw_paths) > 0
                is_non_creatable = norm in NON_CREATABLE

                ranges_dict = [
                    {
                        "layer": r.layer,
                        "rangeExpr": r.range_expr,
                        "rangeMin": r.range_min,
                        "rangeMax": r.range_max,
                        "dynamic": r.dynamic,
                        "nSource": r.n_source,
                        "description": r.description,
                    }
                    for r in cmd["ranges"]
                ]
                instance_levels = derive_instance_levels(norm)
                instance_arity = norm.count("{i}")

                def make_leaf(op: str, paths: list[str]) -> CommandLeaf:
                    rpc = {
                        "LST": "GetParameterValues",
                        "MOD": "SetParameterValues",
                        "ADD": "AddObject",
                        "RMV": "DeleteObject",
                    }[op]
                    refs: list[str]
                    if op in ("ADD", "RMV"):
                        refs = [parent_object_path(norm)]
                    else:
                        refs = paths
                    return CommandLeaf(
                        command_code=f"{op}:{norm}",
                        operation_type=op,
                        rpc_method=rpc,
                        logical_name_i18n={"zh-CN": zh_name},
                        group_code_object=norm,
                        instance_arity=instance_arity,
                        instance_levels=instance_levels,
                        instance_range_meta=ranges_dict,
                        tree_node_refs=refs,
                    )

                if all_paths:
                    grp.commands.append(make_leaf("LST", all_paths))
                if has_rw:
                    grp.commands.append(make_leaf("MOD", rw_paths))
                if has_i and has_rw:
                    if is_non_creatable:
                        nc_hits.append(
                            NonCreatableHit(
                                group_code_object=norm, filtered_ops=["ADD", "RMV"]
                            )
                        )
                    else:
                        grp.commands.append(make_leaf("ADD", []))
                        grp.commands.append(make_leaf("RMV", []))

            groups.append(grp)

        if missing_names:
            print(
                "ERROR: §R-2.4 命令中文名权威表缺少以下 group_code 的中文名（请补 COMMAND_ZH_NAME 字典）：",
                file=sys.stderr,
            )
            for n in missing_names:
                print(f"  - {n}", file=sys.stderr)
            raise SystemExit(1)

        # 命中表唯一性（§R-2.4 跨章节同名禁令）
        seen_names: dict[str, str] = {}
        for g in groups:
            for c in g.commands:
                key = c.logical_name_i18n["zh-CN"]
                # 同 group_code 内 LST/MOD/ADD/RMV 共享同一中文名是允许的
                # 这里只检查"不同 group_code 不能用同一中文名"
                if key in seen_names and seen_names[key] != c.group_code_object:
                    raise SystemExit(
                        f"ERROR: §R-2.4 命令中文名 '{key}' 跨 group_code 撞名："
                        f"{seen_names[key]} vs {c.group_code_object}"
                    )
                seen_names[key] = c.group_code_object

        # params 列表按 standardPath 字典序输出（稳定）
        params_sorted = [param_by_path[p] for p in sorted(param_by_path.keys())]
        return groups, params_sorted, nc_hits


# ─────────────────────────────────────────────────────────────────────────────
# JSON 序列化（dataclass → dict，保留语义字段名）
# ─────────────────────────────────────────────────────────────────────────────
def catalog_to_dict(catalog: Catalog) -> dict:
    def group_to_dict(g: CatalogGroup) -> dict:
        return {
            "groupCode": g.group_code,
            "chapterCode": g.chapter_code,
            "nameI18n": g.name_i18n,
            "displayOrder": g.display_order,
            "commands": [
                {
                    "commandCode": c.command_code,
                    "operationType": c.operation_type,
                    "rpcMethod": c.rpc_method,
                    "logicalNameI18n": c.logical_name_i18n,
                    "groupCodeObject": c.group_code_object,
                    "instanceArity": c.instance_arity,
                    "instanceLevels": c.instance_levels,
                    "instanceRangeMeta": c.instance_range_meta,
                    "treeNodeRefs": c.tree_node_refs,
                }
                for c in g.commands
            ],
        }

    def param_to_dict(p: ParamRow) -> dict:
        return {
            "standardPath": p.standard_path,
            "paramName": p.param_name,
            "labelI18n": {"zh-CN": p.label_zh},
            "accessType": p.access_type,
            "valueType": p.value_type,
            "constraint": p.constraint,
            "constraintHint": p.constraint_hint,
            "lengthMax": p.length_max,
            "rangeMin": p.range_min,
            "rangeMax": p.range_max,
        }

    return {
        "specVersion": catalog.spec_version,
        "schemaVersion": catalog.schema_version,
        "carrier": catalog.carrier,
        "tech": catalog.tech,
        "generatedAt": catalog.generated_at,
        "sourceDocSha256": catalog.source_doc_sha256,
        "stats": catalog.stats,
        "groups": [group_to_dict(g) for g in catalog.groups],
        "params": [param_to_dict(p) for p in catalog.params],
        "mergeLog": [
            {
                "groupCode": m.group_code,
                "ownerChapter": m.owner_chapter,
                "mergedChapter": m.merged_chapter,
                "addedPathCount": m.added_path_count,
            }
            for m in catalog.merge_log
        ],
        "nonCreatableHits": [
            {
                "groupCodeObject": n.group_code_object,
                "filteredOps": n.filtered_ops,
            }
            for n in catalog.non_creatable_hits
        ],
    }


# ─────────────────────────────────────────────────────────────────────────────
# 入口
# ─────────────────────────────────────────────────────────────────────────────
def main() -> int:
    ap = argparse.ArgumentParser(
        description="MML v2.3 catalog MD → JSON 转换器 v2 schema（dev-only）"
    )
    ap.add_argument("--input", default=str(DEFAULT_INPUT), help="规范 MD 文件路径")
    ap.add_argument("--output", default=str(DEFAULT_OUTPUT), help="JSON 输出路径")
    ap.add_argument(
        "--dry-run", action="store_true", help="仅打印汇总信息，不写文件"
    )
    args = ap.parse_args()

    in_path = Path(args.input).resolve()
    out_path = Path(args.output).resolve()

    if not in_path.exists():
        print(f"error: input file not found: {in_path}", file=sys.stderr)
        return 2

    parser = Parser(in_path)
    catalog = parser.parse()

    print(
        f"parsed: {catalog.stats['chapters']} chapters, "
        f"{catalog.stats['commandLeavesTotal']} op leaves, "
        f"{catalog.stats['uniqueParams']} unique params, "
        f"{catalog.stats['mergeEvents']} merge events, "
        f"{catalog.stats['nonCreatableFiltered']} non-creatable filtered"
    )

    if args.dry_run:
        return 0

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(
        json.dumps(catalog_to_dict(catalog), ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    size_kb = out_path.stat().st_size / 1024
    print(f"wrote: {out_path} ({size_kb:.1f} KB)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
