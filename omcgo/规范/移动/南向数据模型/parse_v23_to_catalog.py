#!/usr/bin/env python3
"""
parse_v23_to_catalog.py — MML 控制台 v2.4 P4 数据 ingest 脚本

将 spec MD (cmcc-tdlte-southbound-data-model-v2.3.md) 解析为
catalogloader 可消费的 catalog JSON，写到
omcgo/datamodels/mml-catalog/cmcc-tdlte-v2.3.json。

启动期 catalogloader.LoadOnce 会扫这个目录把所有 *.json 一次性入库
(mml_param_groups / mml_commands / mml_command_sub_fields)。

JSON schema 参见 omcgo/internal/mml/catalogloader/model.go。
"""

from __future__ import annotations

import datetime as _dt
import hashlib
import json
import logging
import re
import sys
from dataclasses import dataclass, field, asdict
from pathlib import Path
from typing import Iterable

# ──────────────────────────────────────────────────────────────────────
# 路径与常量
# ──────────────────────────────────────────────────────────────────────

SCRIPT_DIR = Path(__file__).resolve().parent
SPEC_MD = SCRIPT_DIR / "cmcc-tdlte-southbound-data-model-v2.3.md"
PROJECT_ROOT = SCRIPT_DIR.parents[2]  # .../omcgo
OUTPUT_PATH = PROJECT_ROOT / "datamodels" / "mml-catalog" / "cmcc-tdlte-v2.3.json"

SPEC_VERSION = "cmcc-tdlte-v2.3"
CARRIER = "cmcc"
TECH = "lte"

# 章节标题正则：  "## SA - DeviceInfo" / "## SF - Services.FAPService"
CHAPTER_RE = re.compile(r"^##\s+(S[A-Z])\s+-\s+(.+?)\s*$")

# 伪命令过滤：spec 1442 行只有 ".*" 的占位
PSEUDO_CMD_RE = re.compile(r"^####\s+命令:\s*\.\*\s*$")

# 真命令标题:   "#### 命令: Device.DeviceInfo.* 📖📝"
CMD_RE = re.compile(r"^####\s+命令:\s+(\S+)(?:\s+(.*))?$")

# 表格分隔行（| --- | --- | ...）
TABLE_SEP_RE = re.compile(r"^\|\s*-+\s*(\|\s*-+\s*)+\|\s*$")

# 表头行（认参数表的标志：包含 "TR-181 路径"）
TABLE_HEADER_RE = re.compile(r"^\|\s*#\s*\|.*TR-181.*\|\s*$")

# 表数据行：第一列是序号
TABLE_DATA_RE = re.compile(r"^\|\s*\d+\s*\|")

# 权限 emoji → access type
PERM_RO_RE = re.compile(r"📖")
PERM_RW_RE = re.compile(r"📝")

# {i} / {iα} / {iβ} / {iγ} / {iδ} / {iε} → arity 占位符
INSTANCE_PLACEHOLDER_RE = re.compile(r"\{i[αβγδε]?\}")

# 章节自然顺序映射
CHAPTER_ORDER = {
    f"S{c}": i + 1
    for i, c in enumerate("ABCDEFGHIJKLMNOPQR")  # SA..SR
}

# ──────────────────────────────────────────────────────────────────────
# Group 中文名映射  (chapter, path_template) → zh-CN name
# 来自任务规格 catalog-listing G-01..G-72
# ──────────────────────────────────────────────────────────────────────

GROUP_NAME_MAP: dict[tuple[str, str], str] = {
    ("SA", "Device.DeviceInfo.*"): "设备信息",
    ("SA", "Device.DeviceInfo.SwUpgrade.*"): "设备版本升级",
    ("SB", "Device.SoftwareCtrl.*"): "软件控制",
    ("SC", "Device.ManagementServer.*"): "网管参数",
    ("SD", "Device.FaultMgmt.*"): "故障管理",
    ("SD", "Device.FaultMgmt.CurrentAlarm.{i}.*"): "当前告警实例",
    ("SD", "Device.FaultMgmt.ExpeditedEvent.{i}.*"): "实时告警实例",
    ("SD", "Device.FaultMgmt.HistoryEvent.{i}.*"): "历史告警实例",
    ("SD", "Device.FaultMgmt.QueuedEvent.{i}.*"): "队列告警实例",
    ("SD", "Device.FaultMgmt.SupportedAlarm.{i}.*"): "支持告警实例",
    ("SE", "Device.LogMgmt.*"): "日志管理",
    ("SF", "Device.Services.FAPControl.LTE.*"): "FAPControl LTE",
    ("SF", "Device.Services.FAPControl.LTE.Gateway.*"): "安全/接入网关",
    ("SF", "Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.*"): "MME 池配置",
    ("SF", "Device.Services.FAPControl.LTE.S1U.{i}.*"): "S1U",
    ("SF", "Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*"): "X2 IP 映射",
    ("SF", "Device.Services.FAPService.{i}.*"): "FAPService 载波",
    ("SF", "Device.Services.FAPService.{i}.Capabilities.*"): "FAPService Capabilities",
    ("SF", "Device.Services.FAPService.{i}.CellConfig.Capabilities.*"): "CellConfig Capabilities",
    ("SF", "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.*"): "EPC",
    ("SF", "Device.Services.FAPService.{iα}.CellConfig.LTE.EPC.PLMNList.{iβ}.*"): "PLMN 列表",
    ("SF", "Device.Services.FAPService.{iα}.CellConfig.LTE.VoLTE.PdcpInitParam.{iβ}.*"): "VoLTE PDCP 初始",
    ("SG", "Device.Services.FAPControl.Transport.SCTP.*"): "SCTP",
    ("SG", "Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.*"): "SCTP Assoc",
    # SH 与 SF 同 path 冲突 → 复合 key 区分（zh-CN 为 "RRC Timers"），
    # groupCode 在导出时加 @SH 后缀避免 catalog dedup
    ("SH", "Device.Services.FAPService.{i}.*"): "RRC Timers",
    ("SH", "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.*"): "MAC",
    ("SH", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{iβ}.*"): "DRX 初始",
    ("SH", "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.*"): "PHY",
    ("SH", "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.*"): "PHY MBSFN",
    ("SH", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{iβ}.*"): "MBSFN SFConfigList",
    ("SI", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{iβ}.*"): "GSM 邻区",
    ("SI", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{iβ}.*"): "NR 邻区",
    ("SI", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{iβ}.*"): "UMTS 邻区",
    ("SI", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.LTECell.{iβ}.*"): "LTE 邻区",
    ("SJ", "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.*"): "ConnMode EUTRA",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{iβ}.*"): "A1 测量控制",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{iβ}.*"): "A2 测量控制",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{iβ}.*"): "A3 测量控制",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{iβ}.*"): "A4 测量控制",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{iβ}.*"): "A5 测量控制",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{iβ}.*"): "周期测量控制",
    ("SJ", "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.*"): "ConnMode IRAT",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{iβ}.*"): "B1 测量控制",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{iβ}.*"): "B2 测量控制",
    ("SJ", "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.*"): "IdleMode",
    ("SJ", "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.*"): "IdleMode IRAT",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{iβ}.*"): "GERAN 频组",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{iβ}.*"): "UTRA FDD 频点",
    ("SJ", "Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{iβ}.*"): "异频载波",
    ("SK", "Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.*"): "SON 配置",
    ("SK", "Device.Services.FAPService.{i}.FAPControl.SelfConfig.*"): "自配置启动",
    ("SL", "Device.Ethernet.Interface.{i}.*"): "以太网接口",
    ("SL", "Device.Ethernet.Interface.{iα}.IPv4Address.{iβ}.*"): "IPv4 地址",
    ("SL", "Device.Ethernet.Interface.{iα}.IPv6Address.{iβ}.*"): "IPv6 地址",
    ("SL", "Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.*"): "VLAN 接口",
    ("SL", "Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.IPv4Address.{iγ}.*"): "VLAN IPv4",
    ("SL", "Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.IPv6Address.{iγ}.*"): "VLAN IPv6",
    ("SL", "Device.Ethernet.IpRoute.{i}.*"): "IP 路由",
    ("SM", "Device.IPsec.*"): "IPsec",
    ("SN", "Device.Time.*"): "时间服务器",
    ("SO", "Device.FAP.GPS.*"): "GPS",
    ("SP", "Device.FAP.MRMgmt.Config.{i}.*"): "MR 配置",
    ("SQ", "Device.FAP.PerfMgmt.Config.{i}.*"): "PM 配置",
    ("SR", "Device.DeviceInfo.MU.{i}.*"): "MU 主机单元",
    ("SR", "Device.DeviceInfo.MU.{iα}.Slot.{iβ}.*"): "Slot 板卡",
    ("SR", "Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.*"): "EU 扩展单元",
    ("SR", "Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.*"): "RU 远端单元",
    ("SR", "Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.RFChannel.{iε}.*"): "RFChannel 射频通道",
    ("SR", "Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.SwUpgrade.*"): "RU 升级",
    ("SR", "Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.SwUpgrade.*"): "EU 升级",
    ("SR", "Device.DeviceInfo.MU.{iα}.Slot.{iβ}.SwUpgrade.*"): "Slot 升级",
    ("SR", "Device.DeviceInfo.MU.{i}.SwUpgrade.*"): "MU 升级",
}


# ──────────────────────────────────────────────────────────────────────
# 数据类
# ──────────────────────────────────────────────────────────────────────

@dataclass
class ParsedSubField:
    mml_code: str
    standard_path: str
    label_zh: str
    access_type: str       # RO | RW
    value_type: str
    constraint: str
    sort_order: int

    def to_catalog(self) -> dict:
        return {
            "mmlCode": self.mml_code,
            "standardPath": self.standard_path,
            "labelI18n": {"zh-CN": self.label_zh},
            "accessType": self.access_type,
            "valueType": self.value_type,
            "constraint": self.constraint,
            "defaultSelected": True,
            "sortOrder": self.sort_order,
        }


@dataclass
class ParsedCommand:
    chapter_code: str
    path_template: str         # 原始 path（含 {iα} 等）
    line_no: int               # MD 行号（debug 用）
    sub_fields: list[ParsedSubField] = field(default_factory=list)

    @property
    def instance_arity(self) -> int:
        return len(INSTANCE_PLACEHOLDER_RE.findall(self.path_template))

    @property
    def instance_levels(self) -> list[str]:
        return [f"i{n + 1}" for n in range(self.instance_arity)]


# ──────────────────────────────────────────────────────────────────────
# 工具函数
# ──────────────────────────────────────────────────────────────────────

logger = logging.getLogger(__name__)


def strip_backticks(s: str) -> str:
    s = s.strip()
    if s.startswith("`") and s.endswith("`"):
        return s[1:-1].strip()
    return s


def parse_perm(perm_text: str) -> str:
    """Spec 表格 '权限' 列 → RO / RW。"""
    if PERM_RW_RE.search(perm_text):
        return "RW"
    if PERM_RO_RE.search(perm_text):
        return "RO"
    raise ValueError(f"unrecognized perm cell: {perm_text!r}")


# valueType 的合法 base set（spec 内常见），用于在 split 时识别 base
_VALUE_TYPE_BASE = {
    "string": "string",
    "boolean": "boolean",
    "datetime": "dateTime",
    "unsignedint": "unsignedInt",
    "unsignedlong": "unsignedLong",
    "int": "int",
    "long": "long",
    "hexbinary": "hexBinary",
    "base64": "base64",
    "float": "float",
    "double": "double",
}


def split_type_cell(type_text: str) -> tuple[str, str]:
    """
    把 spec 表第 7 列 "类型" 拆成 (valueType, constraint)。

    例:
      "string"                       → ("string", "")
      "string(64)"                   → ("string", "(64)")
      "unsignedInt[0:65535]"         → ("unsignedInt", "[0:65535]")
      "int[-1:65535]"                → ("int", "[-1:65535]")
      "unsignedInt[0,50,100]"        → ("unsignedInt", "[0,50,100]")
      "unsignedInt[1:4, 6, 8, 10]"   → ("unsignedInt", "[1:4, 6, 8, 10]")
      "Boolean"                      → ("boolean", "")
      ""                             → ("string", "")  (fallback)
    """
    raw = (type_text or "").strip()
    if not raw:
        return "string", ""

    # 找第一个 [ 或 ( 分隔点
    sep_idx = -1
    for idx, ch in enumerate(raw):
        if ch in "[(":
            sep_idx = idx
            break

    if sep_idx == -1:
        base_token = raw
        constraint = ""
    else:
        base_token = raw[:sep_idx].strip()
        constraint = raw[sep_idx:].strip()

    base_lower = base_token.lower()
    value_type = _VALUE_TYPE_BASE.get(base_lower, base_token)  # fallback 保留原 token

    return value_type, constraint


def normalize_placeholder_for_match(path: str) -> str:
    """把 {iα}/{iβ}/... 归一化为 {i}（仅匹配用，不修改 stored path）"""
    return INSTANCE_PLACEHOLDER_RE.sub("{i}", path)


def derive_addrmv_parent_path(path_template: str) -> str | None:
    """
    按 R-3 派生 ADD/RMV 的父集合路径：用 {i} 占位最深一层之外。

    例 'Device.FaultMgmt.SupportedAlarm.{i}.*'
       → 'Device.FaultMgmt.SupportedAlarm.'

      'Device.Services.FAPService.{iα}.CellConfig.LTE.EPC.PLMNList.{iβ}.*'
       → 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.'

      'Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.RFChannel.{iε}.*'
       → 'Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.'
    """
    # 找最后一个占位符的位置
    matches = list(INSTANCE_PLACEHOLDER_RE.finditer(path_template))
    if not matches:
        return None
    last = matches[-1]
    # 截断到最后一个占位符之前（保留前面的 path 段并把前面所有占位符归一化为 {i}）
    head = path_template[: last.start()]
    head = INSTANCE_PLACEHOLDER_RE.sub("{i}", head)
    # 末尾保证以 "." 结尾（spec 模板形如 "X.{i}.*"，截到 last 之前会落到 "X." 上）
    if not head.endswith("."):
        # 极少数 path_template 可能不规则，加保护性补点
        head = head + "."
    return head


# ──────────────────────────────────────────────────────────────────────
# MD 解析
# ──────────────────────────────────────────────────────────────────────

def parse_spec_md(path: Path) -> list[ParsedCommand]:
    """
    扫 spec MD 把每个 #### 命令: <path> 块解析成 ParsedCommand。
    伪命令 ".*" 被过滤掉。
    """
    text = path.read_text(encoding="utf-8")
    lines = text.splitlines()

    commands: list[ParsedCommand] = []
    current_chapter: str | None = None
    current_cmd: ParsedCommand | None = None
    in_table_header = False
    sort_order = 0

    for i, line in enumerate(lines, 1):
        # 章节切换
        m = CHAPTER_RE.match(line)
        if m:
            current_chapter = m.group(1)
            # 章节切换 = 上一命令结束
            current_cmd = None
            in_table_header = False
            continue

        # 伪命令过滤（spec 1442 行）
        if PSEUDO_CMD_RE.match(line):
            current_cmd = None
            in_table_header = False
            logger.info("跳过伪命令 line %d: %r", i, line)
            continue

        m = CMD_RE.match(line)
        if m:
            if current_chapter is None:
                logger.warning("命令出现在章节之前 line %d: %r", i, line)
                continue
            path_tpl = m.group(1).strip()
            current_cmd = ParsedCommand(
                chapter_code=current_chapter,
                path_template=path_tpl,
                line_no=i,
            )
            commands.append(current_cmd)
            in_table_header = False
            sort_order = 0
            continue

        if current_cmd is None:
            continue

        # 在 cmd 块内寻找参数表
        if TABLE_HEADER_RE.match(line):
            in_table_header = True
            continue

        if in_table_header and TABLE_SEP_RE.match(line):
            # 表头分隔行，确认进入数据区
            continue

        if in_table_header and TABLE_DATA_RE.match(line):
            sub_field = parse_table_row(line, line_no=i)
            if sub_field is None:
                continue
            sort_order += 1
            sub_field.sort_order = sort_order
            current_cmd.sub_fields.append(sub_field)
            continue

        # 表外的内容（空行 / "- **{i}** ..." 说明等）—— 退出表状态
        if in_table_header and line.strip() == "":
            # 表后空行结束当前表
            in_table_header = False
            continue

        # 非表行也可能在表之间（如下个命令的说明 "- **{i}**"），不影响
        # 但是若 line 又匹配 cmd 标头会在上一分支被捕获

    return commands


def parse_table_row(line: str, *, line_no: int) -> ParsedSubField | None:
    """
    解析 spec 参数表数据行：
    | 1 | `InternetGatewayDevice.X.Y` | `Device.X.Y` | ParamName | 中文名 | 📝 RW | string(64) |

    返回 ParsedSubField（sort_order 由调用方填）；
    若行格式不符合预期 → log 警告并返回 None（不中断）。
    """
    # 末尾 "|" 后可能有空字符，分割后 strip 即可
    parts = [p.strip() for p in line.split("|")]
    # parts[0]==""  parts[-1]==""  中间是各列
    cols = parts[1:-1] if parts and parts[0] == "" and parts[-1] == "" else parts
    # 期望 7 列: # / TR-098 / TR-181 / 参数名 / 中文名 / 权限 / 类型
    if len(cols) < 7:
        logger.warning("表行列数不足 7 (line %d): %d cols, raw=%r", line_no, len(cols), line)
        return None
    # 部分行（如伪命令下面的混乱行）解析后 mml_code/path 可能为空，跳过
    try:
        standard_path = strip_backticks(cols[2])
        mml_code = cols[3].strip()
        label_zh = cols[4].strip()
        perm = parse_perm(cols[5])
        value_type, constraint = split_type_cell(cols[6])
    except Exception as exc:
        logger.warning("表行解析失败 (line %d): %s, raw=%r", line_no, exc, line)
        return None

    if not standard_path or not mml_code:
        logger.warning("表行 standardPath/mmlCode 为空 (line %d): raw=%r", line_no, line)
        return None

    return ParsedSubField(
        mml_code=mml_code,
        standard_path=standard_path,
        label_zh=label_zh,
        access_type=perm,
        value_type=value_type,
        constraint=constraint,
        sort_order=0,
    )


# ──────────────────────────────────────────────────────────────────────
# Group 构建
# ──────────────────────────────────────────────────────────────────────

def fallback_zh_name(path_template: str) -> str:
    """从 path 末段推断 zh-CN name（fallback）"""
    # 去尾 "*"，取最后一段非 {i*} 的字面名
    parts = [p for p in path_template.split(".") if p and not p.startswith("{") and p != "*"]
    return parts[-1] if parts else path_template


def build_groups(commands: list[ParsedCommand]) -> list[dict]:
    """把 ParsedCommand 列表转成 Catalog.Groups（按 spec 顺序、应用 R-3 派生 op）"""
    groups: list[dict] = []
    chapter_running_order: dict[str, int] = {}

    # 跟踪同章节内的 path 重复（如 SD 多个 FaultMgmt.* 子对象），用于命名提示
    # 跟踪全局 groupCode 唯一（注意 SF/SH 共享 path 的情况）
    seen_codes: dict[str, tuple[str, int]] = {}

    for cmd in commands:
        chapter = cmd.chapter_code
        chapter_running_order[chapter] = chapter_running_order.get(chapter, 0) + 1
        display_order = chapter_running_order[chapter]

        path_tpl = cmd.path_template
        # name 映射
        key = (chapter, path_tpl)
        zh_name = GROUP_NAME_MAP.get(key)
        if zh_name is None:
            zh_name = fallback_zh_name(path_tpl)
            logger.warning(
                "GROUP_NAME_MAP miss: chapter=%s path=%s line=%d → fallback zh-CN=%r",
                chapter, path_tpl, cmd.line_no, zh_name,
            )

        # group_code：默认就是 path_template；
        # 但若同 path 在多 chapter 出现，后出现者必须加 @<chapter> 后缀避免 catalog dedup
        group_code = path_tpl
        prior = seen_codes.get(group_code)
        if prior is not None:
            prior_chapter, prior_line = prior
            new_code = f"{path_tpl}@{chapter}"
            logger.info(
                "groupCode 冲突 path=%s (prior chapter=%s line=%d, this chapter=%s line=%d) → 改用 %s",
                path_tpl, prior_chapter, prior_line, chapter, cmd.line_no, new_code,
            )
            group_code = new_code
        seen_codes[group_code] = (chapter, cmd.line_no)

        # sub_fields → catalog dict
        sub_fields_json = [sf.to_catalog() for sf in cmd.sub_fields]

        # R-3 派生 Commands
        cmd_objs = build_commands(cmd, group_code)

        group_json = {
            "groupCode": group_code,
            "nameI18n": {"zh-CN": zh_name},
            "chapterCode": chapter,
            "displayOrder": display_order,
            "instanceArity": cmd.instance_arity,
            "instanceLevels": cmd.instance_levels,
            "commands": cmd_objs,
            "subFields": sub_fields_json,
        }
        groups.append(group_json)

    return groups


def build_commands(cmd: ParsedCommand, group_code: str) -> list[dict]:
    """
    按 R-3 派生 LST / MOD / ADD / RMV：

    - LST 永远生成，TargetPaths = [groupCode]
    - MOD 仅当 ≥1 RW subField，TargetPaths = RW subField.standardPath list
    - ADD/RMV 仅当 arity≥1 且 RW>0，TargetPaths = 父集合路径（最后一个 {i} 之前截断）
    """
    rw_paths = [sf.standard_path for sf in cmd.sub_fields if sf.access_type == "RW"]
    has_rw = len(rw_paths) > 0
    arity = cmd.instance_arity

    commands: list[dict] = []

    # LST — 恒生成
    commands.append({
        "operationType": "LST",
        "targetPaths": [group_code],
    })

    # MOD — 至少 1 个 RW
    if has_rw:
        commands.append({
            "operationType": "MOD",
            "targetPaths": rw_paths,
        })

    # ADD / RMV — 多实例 + 至少 1 个 RW
    if arity >= 1 and has_rw:
        parent_path = derive_addrmv_parent_path(cmd.path_template)
        if parent_path:
            commands.append({
                "operationType": "ADD",
                "targetPaths": [parent_path],
            })
            commands.append({
                "operationType": "RMV",
                "targetPaths": [parent_path],
            })
        else:
            logger.warning(
                "ADD/RMV 父路径推导失败 chapter=%s path=%s line=%d",
                cmd.chapter_code, cmd.path_template, cmd.line_no,
            )

    return commands


# ──────────────────────────────────────────────────────────────────────
# 主流程
# ──────────────────────────────────────────────────────────────────────

def compute_sha256(path: Path) -> str:
    h = hashlib.sha256()
    h.update(path.read_bytes())
    return h.hexdigest()


def main() -> int:
    logging.basicConfig(
        level=logging.INFO,
        format="%(levelname)s %(message)s",
    )

    if not SPEC_MD.exists():
        logger.error("spec MD 不存在: %s", SPEC_MD)
        return 2

    commands = parse_spec_md(SPEC_MD)
    if not commands:
        logger.error("spec MD 解析得到 0 个命令")
        return 3

    groups = build_groups(commands)

    catalog = {
        "specVersion": SPEC_VERSION,
        "carrier": CARRIER,
        "tech": TECH,
        "generatedAt": _dt.datetime.now(_dt.timezone.utc).isoformat(timespec="seconds"),
        "sourceDocSha256": compute_sha256(SPEC_MD),
        "groups": groups,
    }

    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT_PATH.write_text(
        json.dumps(catalog, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )

    # 统计输出
    chapters: set[str] = set()
    cmd_count = 0
    sub_count = 0
    for g in groups:
        chapters.add(g["chapterCode"])
        cmd_count += len(g["commands"])
        sub_count += len(g["subFields"])

    logger.info("=" * 60)
    logger.info("catalog 生成完毕: %s", OUTPUT_PATH)
    logger.info("  groups=%d chapters=%d commands=%d sub_fields=%d",
                len(groups), len(chapters), cmd_count, sub_count)
    logger.info("=" * 60)

    return 0


if __name__ == "__main__":
    sys.exit(main())
