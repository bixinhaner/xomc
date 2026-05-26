#!/usr/bin/env python3
"""
TR069 CPE 模拟器 — 用于 OMC ACS 端到端测试（中国移动皮基站规范合规版）

功能：
  - 模拟真实 CPE 的完整 TR069 会话流程（Inform → RPC 处理 → Session 结束）
  - 支持所有 12 种 RPC 方法的响应（含 GetRPCMethods）
  - 支持 CPE 主动发起 TransferComplete / AutonomousTransferComplete
  - 支持中国移动专有 EventCode（101-107）
  - 支持告警模拟、PM/MR 定时上传模拟
  - 支持 Connection Request 监听
  - 内置完整参数树（LTE + 5G NR + 告警 + PM + MR + 日志 + 升级）
  - Cookie-based session tracking + HTTP Digest 认证
  - 每 CPE 报文独立存储到文件
  - 可配置 Fault 模拟

用法：
  # 单 CPE 调试
  python3 cpe_simulator.py --acs http://localhost:8080/smallcell/AcsService

  # 单 CPE + bootstrap 事件
  python3 cpe_simulator.py --acs http://localhost:8080/smallcell/AcsService --event bootstrap

  # 单 CPE + 告警模拟（每 30 秒产生告警）
  python3 cpe_simulator.py --acs http://localhost:8080/smallcell/AcsService --alarm-interval 30

  # 单 CPE + PM 文件定时上传
  python3 cpe_simulator.py --acs http://localhost:8080/smallcell/AcsService --pm-upload-interval 300

  # 多 CPE 并发
  python3 cpe_simulator.py --acs http://localhost:8080/smallcell/AcsService --count 10

  # 只发一次 Inform（不循环）
  python3 cpe_simulator.py --acs http://localhost:8080/smallcell/AcsService --once

  # 5% 概率模拟 Fault
  python3 cpe_simulator.py --acs http://localhost:8080/smallcell/AcsService --fault-rate 0.05

  # 报文日志保存到指定目录
  python3 cpe_simulator.py --acs http://localhost:8080/smallcell/AcsService --log-dir ./cpe-logs
"""

import argparse
import base64
import hashlib
import http.client
import http.server
import gzip
import io
import os
import random
import re
import signal
import sys
import threading
import time
import uuid
from datetime import datetime, timezone
from urllib.parse import urlparse
from xml.etree import ElementTree as ET

# ──────────────────────────── 颜色输出 ────────────────────────────

COLORS = {
    "reset": "\033[0m",
    "red": "\033[31m",
    "green": "\033[32m",
    "yellow": "\033[33m",
    "blue": "\033[34m",
    "magenta": "\033[35m",
    "cyan": "\033[36m",
    "gray": "\033[90m",
    "bold": "\033[1m",
}

CPE_COLORS = ["cyan", "magenta", "blue", "green", "yellow"]


def colored(text, color):
    if not sys.stdout.isatty():
        return text
    return f"{COLORS.get(color, '')}{text}{COLORS['reset']}"


# ──────────────────────────── SOAP/XML 模板 ────────────────────────────

SOAP_NS = {
    "soap": "http://schemas.xmlsoap.org/soap/envelope/",
    "soap-enc": "http://schemas.xmlsoap.org/soap/encoding/",
    "cwmp": "urn:dslforum-org:cwmp-1-0",
    "xsi": "http://www.w3.org/2001/XMLSchema-instance",
    "xsd": "http://www.w3.org/2001/XMLSchema",
}

INFORM_TEMPLATE = """<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">{cwmp_id}</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>{manufacturer}</Manufacturer>
        <OUI>{oui}</OUI>
        <ProductClass>{product_class}</ProductClass>
        <SerialNumber>{serial_number}</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[{event_count}]">
{events}
      </Event>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>{current_time}</CurrentTime>
      <RetryCount>{retry_count}</RetryCount>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[{param_count}]">
{parameters}
      </ParameterList>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>"""

EVENT_STRUCT_TEMPLATE = """        <EventStruct>
          <EventCode>{event_code}</EventCode>
          <CommandKey>{command_key}</CommandKey>
        </EventStruct>"""

PARAM_TEMPLATE = """        <ParameterValueStruct>
          <Name>{name}</Name>
          <Value xsi:type="{type}">{value}</Value>
        </ParameterValueStruct>"""


def build_rpc_response(cwmp_id, method, body_xml):
    """构建 RPC 响应的 SOAP 信封"""
    return f"""<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">{cwmp_id}</cwmp:ID>
  </soap:Header>
  <soap:Body>
    {body_xml}
  </soap:Body>
</soap:Envelope>"""


def build_rpc_request(cwmp_id, body_xml):
    """构建 CPE 主动发起的 RPC 请求 SOAP 信封"""
    return build_rpc_response(cwmp_id, "", body_xml)


# ──────────────────────────── 报文日志 ────────────────────────────


class PacketLogger:
    """将每个 CPE 的报文记录到独立文件"""

    def __init__(self, log_dir, serial_number):
        self.enabled = log_dir is not None
        if not self.enabled:
            return
        self.log_dir = log_dir
        os.makedirs(log_dir, exist_ok=True)
        safe_sn = re.sub(r'[^\w\-]', '_', serial_number)
        self.file_path = os.path.join(log_dir, f"cpe_{safe_sn}.log")
        self.lock = threading.Lock()
        # 写入文件头
        with open(self.file_path, "w", encoding="utf-8") as f:
            f.write(f"# CPE Packet Log: {serial_number}\n")
            f.write(f"# Started: {datetime.now().isoformat()}\n")
            f.write(f"{'=' * 80}\n\n")

    def log(self, direction, data, label="", headers=None):
        """记录一条报文（含 HTTP Headers），direction: 'SEND' 或 'RECV'"""
        if not self.enabled:
            return
        ts = datetime.now().strftime("%Y-%m-%d %H:%M:%S.%f")[:-3]
        if isinstance(data, bytes):
            data = data.decode("utf-8", errors="replace")
        with self.lock:
            with open(self.file_path, "a", encoding="utf-8") as f:
                f.write(f"[{ts}] [{direction}] {label}\n")
                f.write(f"{'-' * 60}\n")
                if headers:
                    for k, v in headers.items():
                        f.write(f"{k}: {v}\n")
                    f.write("\n")
                f.write(data if data else "(empty)")
                f.write(f"\n{'=' * 80}\n\n")


# ──────────────────────────── CPE 参数树 ────────────────────────────


def build_default_param_tree(serial_number, oui, product_class):
    """构建模拟设备的完整参数树（含告警/PM/MR/日志/5G NR）"""
    conn_req_ip = f"10.10.3.{random.randint(2, 254)}"
    ip_addr = f"10.0.{random.randint(1, 254)}.{random.randint(2, 254)}"
    pci = str(random.randint(0, 503))
    cell_id = str(random.randint(1, 268435455))
    nr_cell_id = str(random.randint(1, 68719476735))

    return {
        # ── DeviceInfo ──
        "Device.DeviceInfo.Manufacturer": ("xsd:string", "Baicells"),
        "Device.DeviceInfo.ManufacturerOUI": ("xsd:string", oui),
        "Device.DeviceInfo.ProductClass": ("xsd:string", product_class),
        "Device.DeviceInfo.SerialNumber": ("xsd:string", serial_number),
        "Device.DeviceInfo.HardwareVersion": ("xsd:string", "HW-3.1.0"),
        "Device.DeviceInfo.SoftwareVersion": ("xsd:string", "FW-2.5.1-build.2026"),
        "Device.DeviceInfo.ProvisioningCode": ("xsd:string", ""),
        "Device.DeviceInfo.UpTime": ("xsd:unsignedInt", "86400"),
        "Device.DeviceInfo.ModelName": ("xsd:string", "pCRB2000"),
        "Device.DeviceInfo.Description": ("xsd:string", "Baicells Small Cell CPE Simulator"),
        "Device.DeviceInfo.X_COM_STATION_RUN_Time": ("xsd:string", ""),
        "Device.DeviceInfo.X_COM_CloudKey": ("xsd:string", ""),
        "Device.RootDataModelVersion": ("xsd:string", "2.8"),
        # ── ManagementServer ──
        "Device.ManagementServer.URL": ("xsd:string", "http://localhost:8080/smallcell/AcsService"),
        "Device.ManagementServer.Username": ("xsd:string", "cpe"),
        "Device.ManagementServer.Password": ("xsd:string", ""),
        "Device.ManagementServer.PeriodicInformEnable": ("xsd:boolean", "true"),
        "Device.ManagementServer.PeriodicInformInterval": ("xsd:unsignedInt", "60"),
        "Device.ManagementServer.PeriodicInformTime": ("xsd:dateTime", "1970-01-01T00:00:00Z"),
        "Device.ManagementServer.ConnectionRequestURL": ("xsd:string", f"http://{conn_req_ip}:7547"),
        "Device.ManagementServer.ConnectionRequestUsername": ("xsd:string", "acs"),
        "Device.ManagementServer.ConnectionRequestPassword": ("xsd:string", ""),
        "Device.ManagementServer.AliasBasedAddressing": ("xsd:string", "0"),
        "Device.ManagementServer.ParameterKey": ("xsd:string", ""),
        # ── Network ──
        "Device.IP.Interface.1.IPv4Address.1.IPAddress": ("xsd:string", ip_addr),
        "Device.IP.Interface.1.IPv4Address.1.SubnetMask": ("xsd:string", "255.255.255.0"),
        # ── FAPService.1 (LTE) ──
        "Device.Services.FAPService.1.FAPControl.LTE.AdminState": ("xsd:boolean", "true"),
        "Device.Services.FAPService.1.FAPControl.LTE.OpState": ("xsd:boolean", "true"),
        "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": ("xsd:boolean", "true"),
        "Device.Services.FAPService.1.FAPControl.LTE.PhyCellID": ("xsd:string", pci),
        "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity": ("xsd:unsignedInt", cell_id),
        "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL": ("xsd:unsignedInt", "38400"),
        "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.FreqBandIndicator": ("xsd:unsignedInt", "40"),
        "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth": ("xsd:string", "50"),
        "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ULBandwidth": ("xsd:string", "50"),
        "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ReferenceSignalPower": ("xsd:int", "-10"),
        "Device.Services.FAPService.1.FAPControl.LTE.Gateway.S1SigLinkPort": ("xsd:unsignedInt", "36412"),
        "Device.Services.FAPService.1.FAPControl.LTE.Gateway.S1SigLinkServerList": ("xsd:string", "10.0.0.1"),
        "Device.Services.FAPService.1.FAPControl.LTE.Slot": ("xsd:string", "1"),
        # ── FAPService.2 (5G NR) ──
        "Device.Services.FAPService.2.FAPControl.NR.AdminState": ("xsd:boolean", "true"),
        "Device.Services.FAPService.2.FAPControl.NR.OpState": ("xsd:boolean", "true"),
        "Device.Services.FAPService.2.FAPControl.NR.RFTxStatus": ("xsd:boolean", "true"),
        "Device.Services.FAPService.2.FAPControl.NR.PhyCellID": ("xsd:string", str(random.randint(0, 1007))),
        "Device.Services.FAPService.2.CellConfig.NR.RAN.Common.CellIdentity": ("xsd:unsignedInt", nr_cell_id),
        "Device.Services.FAPService.2.CellConfig.NR.RAN.RF.NRARFCN": ("xsd:unsignedInt", "633984"),
        "Device.Services.FAPService.2.CellConfig.NR.RAN.RF.NRBand": ("xsd:unsignedInt", "78"),
        "Device.Services.FAPService.2.CellConfig.NR.RAN.RF.DLBandwidth": ("xsd:string", "100"),
        "Device.Services.FAPService.2.CellConfig.NR.RAN.RF.ULBandwidth": ("xsd:string", "100"),
        "Device.Services.FAPService.2.CellConfig.NR.RAN.RF.ReferenceSignalPower": ("xsd:int", "-10"),
        "Device.Services.FAPService.2.FAPControl.NR.Gateway.NGSigLinkPort": ("xsd:unsignedInt", "38412"),
        "Device.Services.FAPService.2.FAPControl.NR.Gateway.NGSigLinkServerList": ("xsd:string", "10.0.0.1"),
        # ── 告警参数 (规范表35) ──
        "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AlarmIdentifier": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.PerceivedSeverity": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.FaultLocation": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.EventType": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.NotificationType": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.ProbableCause": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.EventTime": ("xsd:dateTime", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.SpecificProblem": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AdditionalInformation": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AdditionalText": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus": ("xsd:string", "0"),
        # ── PM 任务参数 (规范表28) ──
        "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Enable": ("xsd:boolean", "false"),
        "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Alias": ("xsd:string", "PM-Task-1"),
        "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.URL": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Username": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.Password": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.PeriodicUploadInterval": ("xsd:unsignedInt", "900"),
        "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.PeriodicUploadTime": ("xsd:dateTime", "1970-01-01T00:00:00Z"),
        # PM 补采参数 (规范表32)
        "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.ReplenishEnable": ("xsd:boolean", "false"),
        "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.ReplenishStartTime": ("xsd:dateTime", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.ReplenishEndTime": ("xsd:dateTime", ""),
        # ── MR 参数 (规范表63) ──
        "Device.Services.FAPService.1.FAPControl.LTE.MR.MrEnable": ("xsd:boolean", "false"),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.MrUrl": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.MrUsername": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.MrPassword": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.MeasureType": ("xsd:string", "MRO,MRE,MRS"),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.OmcName": ("xsd:string", "OMC-R"),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.SamplePeriod": ("xsd:unsignedInt", "10240"),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.UploadPeriod": ("xsd:unsignedInt", "15"),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.SampleBeginTime": ("xsd:dateTime", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.SampleEndTime": ("xsd:dateTime", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.PrbNum": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.SubFrameNum": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.MR.MRNCGIList": ("xsd:string", "all"),
        # ── 日志参数 (规范表51) ──
        "Device.Services.FAPService.1.FAPControl.LTE.LogUpload.PeriodicUploadEnable": ("xsd:boolean", "false"),
        "Device.Services.FAPService.1.FAPControl.LTE.LogUpload.URL": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.LogUpload.Username": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.LogUpload.Password": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.LogUpload.PeriodicUploadInterval": ("xsd:unsignedInt", "3600"),
        # ── 升级参数 (规范表46/48) ──
        "Device.Services.FAPService.1.FAPControl.LTE.UpgradeInfo.Status": ("xsd:unsignedInt", "0"),
        "Device.Services.FAPService.1.FAPControl.LTE.UpgradeInfo.FailureCause": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.UpgradeInfo.Stage": ("xsd:unsignedInt", "0"),
        # ── 开站参数 (规范 EventCode 105/106) ──
        "Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.Stage": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.Result": ("xsd:string", ""),
        "Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.FailureCause": ("xsd:string", ""),
        # ── CPRI ──
        "Device.DeviceInfo.CpriPort.1.Status": ("xsd:string", "1"),
        "Device.DeviceInfo.CpriPort.2.Status": ("xsd:string", "0"),
        # ── Vendor Extensions ──
        "Device.X_0000B9_Config.AntennaConfig": ("xsd:string", "2T2R"),
        "Device.X_0000B9_Config.TestParam": ("xsd:string", "default_value"),
        "Device.X_Test.Param": ("xsd:string", "test_attr_value"),
        # ── Time ──
        "Device.Time.CurrentLocalTime": ("xsd:dateTime", ""),
    }


# ──────────────────────────── 告警模板 ────────────────────────────

ALARM_TEMPLATES = [
    {
        "EventType": "equipmentAlarm",
        "ProbableCause": "powerSupplyFailure",
        "SpecificProblem": "Power supply unit voltage out of range",
        "PerceivedSeverity": "Major",
    },
    {
        "EventType": "communicationsAlarm",
        "ProbableCause": "lossOfSignal",
        "SpecificProblem": "S1 link connection lost",
        "PerceivedSeverity": "Critical",
    },
    {
        "EventType": "qualityOfServiceAlarm",
        "ProbableCause": "thresholdCrossed",
        "SpecificProblem": "RRC connection success rate below threshold",
        "PerceivedSeverity": "Minor",
    },
    {
        "EventType": "environmentalAlarm",
        "ProbableCause": "highTemperature",
        "SpecificProblem": "Device temperature exceeds safe limit",
        "PerceivedSeverity": "Warning",
    },
    {
        "EventType": "processingErrorAlarm",
        "ProbableCause": "softwareError",
        "SpecificProblem": "Baseband processing unit exception",
        "PerceivedSeverity": "Major",
    },
]


# ──────────────────────────── Digest 认证 ────────────────────────────


def parse_digest_challenge(header):
    """解析 WWW-Authenticate: Digest ... 响应头"""
    parts = {}
    for m in re.finditer(r'(\w+)=(?:"([^"]+)"|(\S+))', header):
        key = m.group(1)
        value = m.group(2) if m.group(2) is not None else m.group(3)
        parts[key] = value
    return parts


def compute_digest_response(username, password, realm, nonce, method, uri, qop=None, nc=None, cnonce=None):
    """计算 HTTP Digest 认证的 response 值"""
    ha1 = hashlib.md5(f"{username}:{realm}:{password}".encode()).hexdigest()
    ha2 = hashlib.md5(f"{method}:{uri}".encode()).hexdigest()
    if qop:
        response = hashlib.md5(f"{ha1}:{nonce}:{nc}:{cnonce}:{qop}:{ha2}".encode()).hexdigest()
    else:
        response = hashlib.md5(f"{ha1}:{nonce}:{ha2}".encode()).hexdigest()
    return response


# ──────────────────────────── RPC 响应处理 ────────────────────────────


class RPCHandler:
    """处理 ACS 下发的各类 RPC 请求，生成对应的 CPE 响应"""

    def __init__(self, param_tree, logger, fault_rate=0.0):
        self.param_tree = param_tree
        self.log = logger
        self.fault_rate = fault_rate
        self.next_instance_number = 100
        # 记录需要异步处理的操作
        self.pending_transfer = None  # (command_key, file_type, is_download, url)

    def handle(self, cwmp_id, method, body_element):
        """根据 RPC 方法分派处理"""
        handler_map = {
            "GetRPCMethods": self._handle_get_rpc_methods,
            "GetParameterValues": self._handle_get_values,
            "SetParameterValues": self._handle_set_values,
            "GetParameterNames": self._handle_get_names,
            "GetParameterAttributes": self._handle_get_attrs,
            "SetParameterAttributes": self._handle_set_attrs,
            "AddObject": self._handle_add_object,
            "DeleteObject": self._handle_delete_object,
            "Download": self._handle_download,
            "Upload": self._handle_upload,
            "Reboot": self._handle_reboot,
            "FactoryReset": self._handle_factory_reset,
        }

        handler = handler_map.get(method)
        if handler:
            # 随机 Fault 模拟
            if self.fault_rate > 0 and random.random() < self.fault_rate:
                fault_code = random.choice([9001, 9002, 9003])
                fault_msgs = {9001: "Request denied", 9002: "Internal error", 9003: "Invalid arguments"}
                self.log(f"  → [FAULT 模拟] {method} → {fault_code}", "red")
                return self._build_fault(cwmp_id, fault_code, fault_msgs[fault_code])
            return handler(cwmp_id, body_element)

        self.log(f"未知 RPC 方法: {method}，返回 Fault 9000", "yellow")
        return self._build_fault(cwmp_id, 9000, "Method not supported")

    def _build_fault(self, cwmp_id, code, message):
        """构建 CWMP Fault 响应"""
        fault_xml = f"""<soap:Fault>
      <faultcode>Client</faultcode>
      <faultstring>CWMP fault</faultstring>
      <detail>
        <cwmp:Fault>
          <FaultCode>{code}</FaultCode>
          <FaultString>{message}</FaultString>
        </cwmp:Fault>
      </detail>
    </soap:Fault>"""
        return build_rpc_response(cwmp_id, "Fault", fault_xml)

    def _handle_get_rpc_methods(self, cwmp_id, body_el):
        """GetRPCMethods: 返回 CPE 支持的全部 RPC 方法列表"""
        methods = [
            "Inform", "TransferComplete", "AutonomousTransferComplete",
            "GetRPCMethods", "GetParameterValues", "SetParameterValues",
            "GetParameterNames", "GetParameterAttributes", "SetParameterAttributes",
            "AddObject", "DeleteObject", "Download", "Upload", "Reboot", "FactoryReset",
        ]
        methods_xml = "\n".join(f'        <string>{m}</string>' for m in methods)
        self.log(f"  → GetRPCMethodsResponse: {len(methods)} 个方法", "green")
        body = f"""<cwmp:GetRPCMethodsResponse>
      <MethodList soap:arrayType="xsd:string[{len(methods)}]">
{methods_xml}
      </MethodList>
    </cwmp:GetRPCMethodsResponse>"""
        return build_rpc_response(cwmp_id, "GetRPCMethodsResponse", body)

    def _handle_get_values(self, cwmp_id, body_el):
        """GetParameterValues: 返回请求的参数值"""
        names = []
        for string_el in body_el.iter():
            if string_el.tag == "string" or (string_el.tag and string_el.tag.endswith("}string")):
                if string_el.text:
                    names.append(string_el.text.strip())
        if not names:
            for el in body_el.iter():
                if el.tag == "Name" or (el.tag and el.tag.endswith("}Name")):
                    if el.text:
                        names.append(el.text.strip())

        params_xml = []
        for name in names:
            if name.endswith("."):
                matches = {k: v for k, v in self.param_tree.items() if k.startswith(name)}
            else:
                matches = {}
                if name in self.param_tree:
                    matches[name] = self.param_tree[name]
            for pname, (ptype, pval) in matches.items():
                params_xml.append(
                    f'        <ParameterValueStruct>\n'
                    f'          <Name>{pname}</Name>\n'
                    f'          <Value xsi:type="{ptype}">{pval}</Value>\n'
                    f'        </ParameterValueStruct>'
                )

        count = len(params_xml)
        if count == 0 and names:
            # 无效参数名 → Fault 9005
            self.log(f"  → GetParameterValues: 参数未找到 {names}, Fault 9005", "yellow")
            return self._build_fault(cwmp_id, 9005, "Invalid parameter name")

        self.log(f"  → GetParameterValuesResponse: {count} 个参数", "green")
        body = f"""<cwmp:GetParameterValuesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[{count}]">
{chr(10).join(params_xml)}
      </ParameterList>
    </cwmp:GetParameterValuesResponse>"""
        return build_rpc_response(cwmp_id, "GetParameterValuesResponse", body)

    def _handle_set_values(self, cwmp_id, body_el):
        """SetParameterValues: 写入参数并返回 Status=0，支持 ParameterKey"""
        count = 0
        parameter_key = ""
        for child in body_el.iter():
            tag = child.tag.split("}")[-1] if "}" in child.tag else child.tag
            if tag == "ParameterKey" and child.text:
                parameter_key = child.text.strip()

        for pvs in body_el.iter():
            tag = pvs.tag.split("}")[-1] if "}" in pvs.tag else pvs.tag
            if tag == "ParameterValueStruct":
                name_el = None
                value_el = None
                for child in pvs:
                    ctag = child.tag.split("}")[-1] if "}" in child.tag else child.tag
                    if ctag == "Name":
                        name_el = child
                    elif ctag == "Value":
                        value_el = child
                if name_el is not None and name_el.text and value_el is not None:
                    pname = name_el.text.strip()
                    pval = value_el.text.strip() if value_el.text else ""
                    ptype = value_el.attrib.get(
                        "{http://www.w3.org/2001/XMLSchema-instance}type", "xsd:string"
                    )
                    # 检查只读参数
                    if self._is_read_only(pname):
                        self.log(f"  → Set 失败: {pname} 只读, Fault 9008", "yellow")
                        return self._build_fault(cwmp_id, 9008, f"Attempt to set a non-writable parameter: {pname}")
                    old_val = self.param_tree.get(pname, (ptype, "N/A"))
                    self.param_tree[pname] = (ptype, pval)
                    self.log(f"  → Set: {pname} = '{pval}' (was '{old_val[1]}')", "green")
                    count += 1

        # 更新 ParameterKey
        if parameter_key:
            self.param_tree["Device.ManagementServer.ParameterKey"] = ("xsd:string", parameter_key)
            self.log(f"  → ParameterKey = '{parameter_key}'", "green")

        body = """<cwmp:SetParameterValuesResponse>
      <Status>0</Status>
    </cwmp:SetParameterValuesResponse>"""
        self.log(f"  → SetParameterValuesResponse: Status=0, {count} 个参数已更新", "green")
        return build_rpc_response(cwmp_id, "SetParameterValuesResponse", body)

    def _is_read_only(self, pname):
        """判断参数是否为只读（简单规则：非 Config/X_ 路径且非 ManagementServer）"""
        writable_patterns = ["Config", "X_", "ManagementServer", "PeriodicStatistics", "MR.", "LogUpload", "UpgradeInfo", "StartupInfo"]
        return not any(p in pname for p in writable_patterns)

    def _handle_get_names(self, cwmp_id, body_el):
        """GetParameterNames: 返回参数路径列表"""
        path = ""
        next_level = False
        for el in body_el.iter():
            tag = el.tag.split("}")[-1] if "}" in el.tag else el.tag
            if tag == "ParameterPath" and el.text:
                path = el.text.strip()
            elif tag == "NextLevel" and el.text:
                next_level = el.text.strip().lower() in ("true", "1")

        params_xml = []
        seen_dirs = set()
        for pname in sorted(self.param_tree.keys()):
            if not pname.startswith(path):
                continue
            remainder = pname[len(path):]
            if next_level:
                parts = remainder.split(".")
                if len(parts) == 1:
                    writable = "true" if not self._is_read_only(pname) else "false"
                    params_xml.append(
                        f"        <ParameterInfoStruct>\n"
                        f"          <Name>{pname}</Name>\n"
                        f"          <Writable>{writable}</Writable>\n"
                        f"        </ParameterInfoStruct>"
                    )
                elif parts[0] and f"{path}{parts[0]}." not in seen_dirs:
                    dir_path = f"{path}{parts[0]}."
                    seen_dirs.add(dir_path)
                    params_xml.append(
                        f"        <ParameterInfoStruct>\n"
                        f"          <Name>{dir_path}</Name>\n"
                        f"          <Writable>false</Writable>\n"
                        f"        </ParameterInfoStruct>"
                    )
            else:
                # TR-069 A.3.2.3: NextLevel=false requires returning ALL parameters
                # AND intermediate object paths (e.g., "Device.", "Device.DeviceInfo.")
                # First, collect intermediate object paths from this parameter
                parts = pname.split(".")
                for i in range(1, len(parts)):
                    obj_path = ".".join(parts[:i]) + "."
                    if obj_path.startswith(path) and obj_path not in seen_dirs:
                        seen_dirs.add(obj_path)
                        params_xml.append(
                            f"        <ParameterInfoStruct>\n"
                            f"          <Name>{obj_path}</Name>\n"
                            f"          <Writable>false</Writable>\n"
                            f"        </ParameterInfoStruct>"
                        )
                # Then add the leaf parameter itself
                writable = "true" if not self._is_read_only(pname) else "false"
                params_xml.append(
                    f"        <ParameterInfoStruct>\n"
                    f"          <Name>{pname}</Name>\n"
                    f"          <Writable>{writable}</Writable>\n"
                    f"        </ParameterInfoStruct>"
                )

        count = len(params_xml)
        self.log(f"  → GetParameterNamesResponse: {count} 个路径 (path={path}, next_level={next_level})", "green")
        body = f"""<cwmp:GetParameterNamesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterInfoStruct[{count}]">
{chr(10).join(params_xml)}
      </ParameterList>
    </cwmp:GetParameterNamesResponse>"""
        return build_rpc_response(cwmp_id, "GetParameterNamesResponse", body)

    def _handle_get_attrs(self, cwmp_id, body_el):
        """GetParameterAttributes: 返回参数属性"""
        names = []
        for el in body_el.iter():
            tag = el.tag.split("}")[-1] if "}" in el.tag else el.tag
            if tag == "string" and el.text:
                names.append(el.text.strip())

        params_xml = []
        for name in names:
            notification = "2" if "Alarm" in name else "1" if "Status" in name else "0"
            params_xml.append(
                f"        <ParameterAttributeStruct>\n"
                f"          <Name>{name}</Name>\n"
                f"          <Notification>{notification}</Notification>\n"
                f"          <AccessList>\n"
                f"            <string>Subscriber</string>\n"
                f"          </AccessList>\n"
                f"        </ParameterAttributeStruct>"
            )

        count = len(params_xml)
        self.log(f"  → GetParameterAttributesResponse: {count} 个属性", "green")
        body = f"""<cwmp:GetParameterAttributesResponse>
      <ParameterList soap:arrayType="cwmp:ParameterAttributeStruct[{count}]">
{chr(10).join(params_xml)}
      </ParameterList>
    </cwmp:GetParameterAttributesResponse>"""
        return build_rpc_response(cwmp_id, "GetParameterAttributesResponse", body)

    def _handle_set_attrs(self, cwmp_id, body_el):
        """SetParameterAttributes: 设置参数属性，返回空响应"""
        self.log("  → SetParameterAttributesResponse (ok)", "green")
        body = "<cwmp:SetParameterAttributesResponse/>"
        return build_rpc_response(cwmp_id, "SetParameterAttributesResponse", body)

    def _handle_add_object(self, cwmp_id, body_el):
        """AddObject: 创建对象实例，支持 ParameterKey"""
        object_name = ""
        parameter_key = ""
        for el in body_el.iter():
            tag = el.tag.split("}")[-1] if "}" in el.tag else el.tag
            if tag == "ObjectName" and el.text:
                object_name = el.text.strip()
            elif tag == "ParameterKey" and el.text:
                parameter_key = el.text.strip()

        instance = self.next_instance_number
        self.next_instance_number += 1

        if object_name:
            base = f"{object_name}{instance}."
            self.param_tree[f"{base}Enable"] = ("xsd:boolean", "false")
            self.param_tree[f"{base}Status"] = ("xsd:string", "Disabled")

        if parameter_key:
            self.param_tree["Device.ManagementServer.ParameterKey"] = ("xsd:string", parameter_key)

        self.log(f"  → AddObjectResponse: {object_name}{instance} (InstanceNumber={instance})", "green")
        body = f"""<cwmp:AddObjectResponse>
      <InstanceNumber>{instance}</InstanceNumber>
      <Status>0</Status>
    </cwmp:AddObjectResponse>"""
        return build_rpc_response(cwmp_id, "AddObjectResponse", body)

    def _handle_delete_object(self, cwmp_id, body_el):
        """DeleteObject: 删除对象实例，支持 ParameterKey"""
        object_name = ""
        parameter_key = ""
        for el in body_el.iter():
            tag = el.tag.split("}")[-1] if "}" in el.tag else el.tag
            if tag == "ObjectName" and el.text:
                object_name = el.text.strip()
            elif tag == "ParameterKey" and el.text:
                parameter_key = el.text.strip()

        to_remove = [k for k in self.param_tree if k.startswith(object_name)]
        for k in to_remove:
            del self.param_tree[k]

        if parameter_key:
            self.param_tree["Device.ManagementServer.ParameterKey"] = ("xsd:string", parameter_key)

        self.log(f"  → DeleteObjectResponse: {object_name} (removed {len(to_remove)} params)", "green")
        body = """<cwmp:DeleteObjectResponse>
      <Status>0</Status>
    </cwmp:DeleteObjectResponse>"""
        return build_rpc_response(cwmp_id, "DeleteObjectResponse", body)

    def _handle_download(self, cwmp_id, body_el):
        """Download: 返回 Status=1（异步），触发后续 TransferComplete 会话"""
        url = ""
        file_type = ""
        command_key = ""
        for el in body_el.iter():
            tag = el.tag.split("}")[-1] if "}" in el.tag else el.tag
            if tag == "URL" and el.text:
                url = el.text.strip()
            elif tag == "FileType" and el.text:
                file_type = el.text.strip()
            elif tag == "CommandKey" and el.text:
                command_key = el.text.strip()

        # 记录待处理的异步传输
        self.pending_transfer = (command_key, file_type, True, url)

        now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        self.log(f"  → DownloadResponse: type='{file_type}' url='{url}' (Status=1, 异步)", "green")

        body = f"""<cwmp:DownloadResponse>
      <Status>1</Status>
      <StartTime>0001-01-01T00:00:00Z</StartTime>
      <CompleteTime>0001-01-01T00:00:00Z</CompleteTime>
    </cwmp:DownloadResponse>"""
        return build_rpc_response(cwmp_id, "DownloadResponse", body)

    def _handle_upload(self, cwmp_id, body_el):
        """Upload: 返回 Status=1（异步），触发后续 TransferComplete 会话"""
        url = ""
        file_type = ""
        command_key = ""
        for el in body_el.iter():
            tag = el.tag.split("}")[-1] if "}" in el.tag else el.tag
            if tag == "URL" and el.text:
                url = el.text.strip()
            elif tag == "FileType" and el.text:
                file_type = el.text.strip()
            elif tag == "CommandKey" and el.text:
                command_key = el.text.strip()

        # 记录待处理的异步传输
        self.pending_transfer = (command_key, file_type, False, url)

        self.log(f"  → UploadResponse: type='{file_type}' (Status=1, 异步)", "green")

        body = f"""<cwmp:UploadResponse>
      <Status>1</Status>
      <StartTime>0001-01-01T00:00:00Z</StartTime>
      <CompleteTime>0001-01-01T00:00:00Z</CompleteTime>
    </cwmp:UploadResponse>"""
        return build_rpc_response(cwmp_id, "UploadResponse", body)

    def _handle_reboot(self, cwmp_id, body_el):
        """Reboot: 返回成功（模拟，不真正重启）"""
        self.log("  → RebootResponse (模拟重启，2秒后发送 1 BOOT Inform)", "yellow")
        body = "<cwmp:RebootResponse/>"
        return build_rpc_response(cwmp_id, "RebootResponse", body)

    def _handle_factory_reset(self, cwmp_id, body_el):
        """FactoryReset: 返回成功"""
        self.log("  → FactoryResetResponse (模拟恢复出厂)", "yellow")
        body = "<cwmp:FactoryResetResponse/>"
        return build_rpc_response(cwmp_id, "FactoryResetResponse", body)


# ──────────────────────────── XML 解析工具 ────────────────────────────


def extract_cwmp_id(xml_bytes):
    """从 SOAP 响应中提取 cwmp:ID"""
    text = xml_bytes.decode("utf-8", errors="replace")
    m = re.search(r"<cwmp:ID[^>]*>(.*?)</cwmp:ID>", text)
    if m:
        return m.group(1).strip()
    try:
        root = ET.fromstring(xml_bytes)
        for el in root.iter():
            if el.tag and "ID" in el.tag and el.text:
                return el.text.strip()
    except ET.ParseError:
        pass
    return ""


def detect_rpc_method(xml_bytes):
    """检测 ACS 下发的 RPC 方法名"""
    text = xml_bytes.decode("utf-8", errors="replace")
    rpc_methods = [
        "GetRPCMethods",
        "GetParameterValues",
        "SetParameterValues",
        "GetParameterNames",
        "GetParameterAttributes",
        "SetParameterAttributes",
        "AddObject",
        "DeleteObject",
        "Download",
        "Upload",
        "Reboot",
        "FactoryReset",
    ]
    for method in rpc_methods:
        if f"<cwmp:{method}" in text or f":{method}" in text:
            return method
    return None


def parse_soap_body(xml_bytes):
    """解析 SOAP Body 元素"""
    try:
        for prefix, uri in SOAP_NS.items():
            ET.register_namespace(prefix, uri)
        root = ET.fromstring(xml_bytes)
        for el in root.iter():
            tag = el.tag.split("}")[-1] if "}" in el.tag else el.tag
            if tag == "Body":
                return el
    except ET.ParseError:
        pass
    return None


# ──────────────────────────── EventCode 映射 ────────────────────────────

EVENT_MAP = {
    "bootstrap": [("0 BOOTSTRAP", "")],
    "boot": [("1 BOOT", "")],
    "periodic": [("2 PERIODIC", "")],
    "value_change": [("2 PERIODIC", ""), ("4 VALUE CHANGE", "")],
    "transfer_complete": [("7 TRANSFER COMPLETE", "")],
    "connection_request": [("6 CONNECTION REQUEST", "")],
    "alarm": [("101 ALARM", "")],
    "upgrade_finish": [("102 UPGRADE FINISH", "")],
    "add_object": [("103 ADD OBJECT", "")],
    "delete_object": [("104 DELETE OBJECT", "")],
    "startup_stage": [("105 STARTUP STAGE REPORT", "")],
    "startup_result": [("106 STARTUP RESULT REPORT", "")],
    "unit_upgrade": [("107 UNIT UPGRADE RESULT", "")],
}


# ──────────────────────────── Connection Request Server ────────────────────────────


class ConnectionRequestHandler(http.server.BaseHTTPRequestHandler):
    """处理 ACS 发起的 Connection Request"""

    def do_GET(self):
        # 基本认证检查
        auth = self.headers.get("Authorization", "")
        if self.server.cpe_ref and auth:
            # 简化处理：接受任何认证
            pass
        self.send_response(200)
        self.send_header("Content-Length", "0")
        self.end_headers()
        # 通知 CPE 发起 Connection Request 会话
        if self.server.cpe_ref:
            self.server.cpe_ref.trigger_connection_request()

    def log_message(self, format, *args):
        # 静默
        pass


class ConnectionRequestServer:
    """在 ConnectionRequestURL 端口监听 ACS 的 Connection Request"""

    def __init__(self, port, cpe_ref):
        self.port = port
        self.cpe_ref = cpe_ref
        self.server = None
        self.thread = None

    def start(self):
        try:
            self.server = http.server.HTTPServer(("0.0.0.0", self.port), ConnectionRequestHandler)
            self.server.cpe_ref = self.cpe_ref
            self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
            self.thread.start()
            return True
        except OSError:
            return False

    def stop(self):
        if self.server:
            self.server.shutdown()


# ──────────────────────────── CPE 模拟器核心 ────────────────────────────


class CPESimulator:
    """单个 CPE 设备模拟器"""

    def __init__(
        self,
        acs_url,
        serial_number,
        oui="48BF74",
        product_class="FAP/pCRB2000/SC",
        manufacturer="Baicells",
        interval=60,
        event_type="periodic",
        color="cyan",
        cpe_index=0,
        fault_rate=0.0,
        log_dir=None,
        alarm_interval=0,
        pm_upload_interval=0,
        mr_upload_interval=0,
        enable_connreq=False,
    ):
        self.acs_url = acs_url
        self.parsed_url = urlparse(acs_url)
        self.serial_number = serial_number
        self.oui = oui
        self.product_class = product_class
        self.manufacturer = manufacturer
        self.interval = interval
        self.initial_event = event_type
        self.color = color
        self.cpe_index = cpe_index
        self.fault_rate = fault_rate
        self.alarm_interval = alarm_interval
        self.pm_upload_interval = pm_upload_interval
        self.mr_upload_interval = mr_upload_interval
        self.enable_connreq = enable_connreq

        self.session_cookie = None
        self.cwmp_id_counter = random.randint(10000, 99999)
        self.inform_count = 0
        self.rpc_count = 0
        self.error_count = 0
        self.running = True
        self.first_inform = True
        self.pending_boot = False
        self.pending_connection_request = False
        self.alarm_counter = 0
        self.active_alarms = []  # 当前活跃告警 ID 列表

        # 会话级连接
        self._session_conn = None

        # 参数树
        self.param_tree = build_default_param_tree(serial_number, oui, product_class)
        self.rpc_handler = RPCHandler(self.param_tree, self._log, fault_rate=fault_rate)

        # Digest auth state
        self.digest_nonce = None
        self.digest_realm = None
        self.digest_nc = 0

        # 报文日志
        self.pkt_logger = PacketLogger(log_dir, serial_number)

        # Connection Request 服务器
        self.connreq_server = None

        # 会话锁（防止并发会话冲突）
        self._session_lock = threading.Lock()

    def _log(self, msg, color=None):
        ts = datetime.now().strftime("%H:%M:%S.%f")[:-3]
        prefix = colored(f"[{ts}]", "gray")
        sn_short = self.serial_number[-8:]
        tag = colored(f"[CPE-{sn_short}]", self.color)
        if color:
            msg = colored(msg, color)
        print(f"{prefix} {tag} {msg}")

    def _next_cwmp_id(self):
        self.cwmp_id_counter += 1
        return str(self.cwmp_id_counter)

    def trigger_connection_request(self):
        """被 ConnectionRequestServer 调用，触发 6 CONNECTION REQUEST 会话"""
        self.pending_connection_request = True
        self._log("收到 Connection Request，将发起新会话", "yellow")

    def _get_inform_params(self, event_codes=None):
        """获取 Inform 中携带的关键参数，根据事件类型动态调整"""
        base_params = [
            "Device.DeviceInfo.HardwareVersion",
            "Device.DeviceInfo.SoftwareVersion",
            "Device.DeviceInfo.X_COM_STATION_RUN_Time",
            "Device.DeviceInfo.X_COM_CloudKey",
            "Device.ManagementServer.ConnectionRequestURL",
            "Device.ManagementServer.PeriodicInformInterval",
            "Device.ManagementServer.AliasBasedAddressing",
            "Device.ManagementServer.ParameterKey",
            "Device.RootDataModelVersion",
            "Device.DeviceInfo.ProvisioningCode",
            "Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus",
        ]

        # 根据事件类型添加额外参数
        if event_codes:
            event_strs = [ec for ec, _ in event_codes]
            if "101 ALARM" in event_strs:
                # 告警事件携带告警参数
                base_params.extend([
                    "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AlarmIdentifier",
                    "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.PerceivedSeverity",
                    "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.FaultLocation",
                    "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.EventType",
                    "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.NotificationType",
                    "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.ProbableCause",
                    "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.EventTime",
                    "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.SpecificProblem",
                    "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AdditionalInformation",
                    "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AdditionalText",
                ])
            if any("107 UNIT UPGRADE RESULT" in e for e in event_strs):
                base_params.extend([
                    "Device.Services.FAPService.1.FAPControl.LTE.UpgradeInfo.Status",
                    "Device.Services.FAPService.1.FAPControl.LTE.UpgradeInfo.FailureCause",
                ])
            if any("102 UPGRADE FINISH" in e for e in event_strs):
                base_params.extend([
                    "Device.Services.FAPService.1.FAPControl.LTE.UpgradeInfo.Status",
                    "Device.Services.FAPService.1.FAPControl.LTE.UpgradeInfo.FailureCause",
                ])
            if any("105 STARTUP" in e for e in event_strs):
                base_params.extend([
                    "Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.Stage",
                    "Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.Result",
                    "Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.FailureCause",
                ])
            if any("106 STARTUP" in e for e in event_strs):
                base_params.extend([
                    "Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.Stage",
                    "Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.Result",
                    "Device.Services.FAPService.1.FAPControl.LTE.StartupInfo.FailureCause",
                ])

        result = []
        seen = set()
        for name in base_params:
            if name in seen:
                continue
            seen.add(name)
            if name in self.param_tree:
                ptype, pval = self.param_tree[name]
                result.append((name, ptype, pval))
        return result

    def _build_inform(self, event_codes):
        """构建 Inform SOAP 消息"""
        cwmp_id = self._next_cwmp_id()
        now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

        self.param_tree["Device.Time.CurrentLocalTime"] = ("xsd:dateTime", now)
        self.param_tree["Device.DeviceInfo.UpTime"] = (
            "xsd:unsignedInt",
            str(self.inform_count * self.interval),
        )

        events_xml = "\n".join(
            EVENT_STRUCT_TEMPLATE.format(event_code=ec, command_key=ck) for ec, ck in event_codes
        )

        params = self._get_inform_params(event_codes)
        params_xml = "\n".join(
            PARAM_TEMPLATE.format(name=name, type=ptype, value=pval) for name, ptype, pval in params
        )

        xml = INFORM_TEMPLATE.format(
            cwmp_id=cwmp_id,
            manufacturer=self.manufacturer,
            oui=self.oui,
            product_class=self.product_class,
            serial_number=self.serial_number,
            event_count=len(event_codes),
            events=events_xml,
            current_time=now,
            retry_count=0,
            param_count=len(params),
            parameters=params_xml,
        )
        return cwmp_id, xml

    def _create_session_conn(self):
        """创建会话级 HTTP 连接"""
        conn_cls = (
            http.client.HTTPSConnection
            if self.parsed_url.scheme == "https"
            else http.client.HTTPConnection
        )
        port = self.parsed_url.port or (443 if self.parsed_url.scheme == "https" else 80)
        return conn_cls(self.parsed_url.hostname, port, timeout=30)

    def _http_request(self, body=None, retry_auth=True):
        """发送 HTTP 请求到 ACS，支持 Digest 认证、Cookie、会话级连接复用"""
        path = self.parsed_url.path or "/"

        headers = {
            "User-Agent": f"TR069-CPE-Sim/{self.serial_number}",
            "Connection": "keep-alive",
            "SOAPAction": '""',
        }

        if body:
            body_bytes = body.encode("utf-8")
            headers["Content-Type"] = "text/xml; charset=utf-8"
            headers["Content-Length"] = str(len(body_bytes))
        else:
            # 空 POST：不设 Content-Type（无内容）
            body_bytes = b""
            headers["Content-Length"] = "0"

        if self.session_cookie:
            headers["Cookie"] = f"SESSION={self.session_cookie}"

        if self.digest_nonce and self.digest_realm:
            self.digest_nc += 1
            nc = f"{self.digest_nc:08x}"
            cnonce = uuid.uuid4().hex[:16]
            response = compute_digest_response(
                "cpe", "cpe_pass", self.digest_realm, self.digest_nonce, "POST", path, "auth", nc, cnonce
            )
            headers["Authorization"] = (
                f'Digest username="cpe", realm="{self.digest_realm}", '
                f'nonce="{self.digest_nonce}", uri="{path}", '
                f'cnonce="{cnonce}", nc={nc}, qop=auth, '
                f'response="{response}", algorithm=MD5'
            )

        # 报文日志 - 发送（含 HTTP Headers）
        label = f"HTTP POST {path} (body={len(body_bytes)} bytes)"
        self.pkt_logger.log("SEND", body if body else "(empty POST)", label, headers=headers)

        # 使用会话级连接
        try:
            if self._session_conn is None:
                self._session_conn = self._create_session_conn()
            self._session_conn.request("POST", path, body=body_bytes, headers=headers)
            resp = self._session_conn.getresponse()
        except (http.client.RemoteDisconnected, ConnectionError, OSError):
            # 连接断开，重建
            self._session_conn = self._create_session_conn()
            self._session_conn.request("POST", path, body=body_bytes, headers=headers)
            resp = self._session_conn.getresponse()

        resp_body = resp.read()

        # 报文日志 - 接收（含 HTTP 响应 Headers）
        resp_headers = dict(resp.getheaders())
        resp_label = f"HTTP {resp.status} {resp.reason} (body={len(resp_body)} bytes)"
        self.pkt_logger.log(
            "RECV",
            resp_body.decode("utf-8", errors="replace") if resp_body else "(empty)",
            resp_label,
            headers=resp_headers,
        )

        # 处理 401 Digest 认证挑战
        if resp.status == 401 and retry_auth:
            auth_header = resp.getheader("WWW-Authenticate", "")
            if "Digest" in auth_header:
                challenge = parse_digest_challenge(auth_header)
                self.digest_nonce = challenge.get("nonce", "")
                self.digest_realm = challenge.get("realm", "ACS")
                self.digest_nc = 0
                return self._http_request(body=body, retry_auth=False)

        # 提取 Set-Cookie
        for header_name, header_value in resp.getheaders():
            if header_name.lower() == "set-cookie":
                m = re.search(r"SESSION=([^;]+)", header_value)
                if m:
                    self.session_cookie = m.group(1)

        return resp.status, resp_body

    def _close_session_conn(self):
        """关闭会话级连接"""
        if self._session_conn:
            try:
                self._session_conn.close()
            except Exception:
                pass
            self._session_conn = None

    def _do_file_upload(self, url, file_type):
        """实际向 ACS FileUploadService 上传一个模拟文件，返回文件大小"""
        try:
            parsed = urlparse(url)
            # 生成模拟 PM XML 数据
            pm_data = f"""<?xml version="1.0"?>
<measCollec beginTime="{datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')}">
  <measData>
    <managedElement dn="SubNetwork=OMC,MeContext={self.serial_number}"/>
    <measInfo measInfoId="PM-Counter-Group-1">
      <measType p="1">RRC.ConnEstabAtt</measType>
      <measType p="2">RRC.ConnEstabSucc</measType>
      <measType p="3">RRC.ConnMean</measType>
      <measValue measObjLdn="FAPService.1">
        <r p="1">{random.randint(100, 500)}</r>
        <r p="2">{random.randint(90, 500)}</r>
        <r p="3">{random.randint(1, 50)}</r>
      </measValue>
    </measInfo>
  </measData>
</measCollec>"""
            buf = io.BytesIO()
            with gzip.GzipFile(fileobj=buf, mode="wb") as gz:
                gz.write(pm_data.encode("utf-8"))
            compressed = buf.getvalue()

            conn_cls = http.client.HTTPSConnection if parsed.scheme == "https" else http.client.HTTPConnection
            port = parsed.port or (443 if parsed.scheme == "https" else 80)
            conn = conn_cls(parsed.hostname, port)
            path_with_query = parsed.path + ("?" + parsed.query if parsed.query else "")

            upload_headers = {
                "Content-Type": "application/octet-stream",
                "Content-Length": str(len(compressed)),
            }
            cred = base64.b64encode(b"upload_user:secure_upload_password_123").decode()
            upload_headers["Authorization"] = f"Basic {cred}"

            conn.request("POST", path_with_query, body=compressed, headers=upload_headers)
            resp = conn.getresponse()
            self._log(f"  → 文件上传 {file_type}: HTTP {resp.status} ({len(compressed)} bytes)", "green")
            resp.read()
            conn.close()
            return len(compressed)
        except Exception as e:
            self._log(f"  → 文件上传失败: {e}", "red")
            return 0

    def _run_transfer_complete_session(self, command_key, file_type, is_download, url):
        """运行 TransferComplete 通知会话（Download/Upload 后异步调用）"""
        with self._session_lock:
            self._log(f"发起 TransferComplete 会话 (type={file_type}, download={is_download})", "yellow")

            # 如果是 Upload，先执行文件上传
            file_size = 0
            if not is_download and "FileUploadService" in url:
                file_size = self._do_file_upload(url, file_type)

            m_event = "M Download" if is_download else "M Upload"
            events = [("7 TRANSFER COMPLETE", command_key), (m_event, command_key)]

            # 1. Inform
            self._session_conn = None
            cwmp_id, inform_xml = self._build_inform(events)
            self._log(f"→ Inform (TransferComplete session, cwmp:ID={cwmp_id})")

            try:
                status, resp_body = self._http_request(body=inform_xml)
            except Exception as e:
                self._log(f"✗ TransferComplete Inform 失败: {e}", "red")
                self._close_session_conn()
                return

            if status != 200:
                self._log(f"✗ TransferComplete Inform 失败: HTTP {status}", "red")
                self._close_session_conn()
                return

            self._log(f"← InformResponse OK", "green")

            # 2. 直接发送 TransferComplete（TR-069: CPE 主动 RPC 紧跟 InformResponse）
            now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
            tc_body = f"""<cwmp:TransferComplete>
      <CommandKey>{command_key}</CommandKey>
      <FaultStruct>
        <FaultCode>0</FaultCode>
        <FaultString></FaultString>
      </FaultStruct>
      <StartTime>{now}</StartTime>
      <CompleteTime>{now}</CompleteTime>
    </cwmp:TransferComplete>"""
            tc_xml = build_rpc_request(self._next_cwmp_id(), tc_body)
            self._log(f"→ TransferComplete (CommandKey={command_key})")

            try:
                status, resp_body = self._http_request(body=tc_xml)
            except Exception as e:
                self._log(f"✗ TransferComplete 发送失败: {e}", "red")
                self._close_session_conn()
                return

            self._log(f"← TransferCompleteResponse: HTTP {status}", "green")

            # 3. 处理可能的后续 RPC（ACS 可能在响应中管道化 RPC）
            self._process_rpc_until_done(status, resp_body)

            # 4. 空 POST 结束会话
            try:
                self._http_request(body=None)
            except Exception:
                pass

            self._close_session_conn()
            self.session_cookie = None

    def _run_autonomous_transfer_complete(self, file_type, transfer_url, file_size):
        """PM/MR/日志文件上传后发送 AutonomousTransferComplete"""
        with self._session_lock:
            self._log(f"发起 AutonomousTransferComplete 会话 (type={file_type})", "yellow")

            events = [("10 AUTONOMOUS TRANSFER COMPLETE", "")]

            # 1. Inform
            self._session_conn = None
            cwmp_id, inform_xml = self._build_inform(events)
            self._log(f"→ Inform (AutonomousTransferComplete, cwmp:ID={cwmp_id})")

            try:
                status, resp_body = self._http_request(body=inform_xml)
            except Exception as e:
                self._log(f"✗ ATC Inform 失败: {e}", "red")
                self._close_session_conn()
                return

            if status != 200:
                self._log(f"✗ ATC Inform 失败: HTTP {status}", "red")
                self._close_session_conn()
                return

            self._log(f"← InformResponse OK", "green")

            # 2. 直接发送 AutonomousTransferComplete（TR-069: CPE 主动 RPC 紧跟 InformResponse）
            now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
            atc_body = f"""<cwmp:AutonomousTransferComplete>
      <AnnounceURL></AnnounceURL>
      <TransferURL>{transfer_url}</TransferURL>
      <IsDownload>false</IsDownload>
      <FileType>{file_type}</FileType>
      <FileSize>{file_size}</FileSize>
      <TargetFileName></TargetFileName>
      <FaultStruct>
        <FaultCode>0</FaultCode>
        <FaultString></FaultString>
      </FaultStruct>
      <StartTime>{now}</StartTime>
      <CompleteTime>{now}</CompleteTime>
    </cwmp:AutonomousTransferComplete>"""
            atc_xml = build_rpc_request(self._next_cwmp_id(), atc_body)
            self._log(f"→ AutonomousTransferComplete (FileType={file_type})")

            try:
                status, resp_body = self._http_request(body=atc_xml)
            except Exception as e:
                self._log(f"✗ ATC 发送失败: {e}", "red")
                self._close_session_conn()
                return

            self._log(f"← AutonomousTransferCompleteResponse: HTTP {status}", "green")

            # 3. 处理可能的后续 RPC + 空 POST 结束会话
            self._process_rpc_until_done(status, resp_body)
            try:
                self._http_request(body=None)
            except Exception:
                pass

            self._close_session_conn()
            self.session_cookie = None

    def _simulate_alarm(self):
        """模拟产生/清除告警"""
        now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        prefix = "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1."

        # 50% 概率清除已有告警，50% 概率产生新告警
        if self.active_alarms and random.random() < 0.5:
            alarm_id = self.active_alarms.pop(0)
            self.param_tree[f"{prefix}AlarmIdentifier"] = ("xsd:string", alarm_id)
            self.param_tree[f"{prefix}NotificationType"] = ("xsd:string", "ClearedAlarm")
            self.param_tree[f"{prefix}EventTime"] = ("xsd:dateTime", now)
            self._log(f"模拟告警清除: {alarm_id}", "green")
        else:
            self.alarm_counter += 1
            alarm_id = f"ALM-{self.serial_number[-6:]}-{self.alarm_counter:04d}"
            tmpl = random.choice(ALARM_TEMPLATES)
            self.param_tree[f"{prefix}AlarmIdentifier"] = ("xsd:string", alarm_id)
            self.param_tree[f"{prefix}PerceivedSeverity"] = ("xsd:string", tmpl["PerceivedSeverity"])
            self.param_tree[f"{prefix}FaultLocation"] = ("xsd:string", f"SubNetwork=OMC,MeContext={self.serial_number},ManagedElement=1,FAPService=1")
            self.param_tree[f"{prefix}EventType"] = ("xsd:string", tmpl["EventType"])
            self.param_tree[f"{prefix}NotificationType"] = ("xsd:string", "NewAlarm")
            self.param_tree[f"{prefix}ProbableCause"] = ("xsd:string", tmpl["ProbableCause"])
            self.param_tree[f"{prefix}EventTime"] = ("xsd:dateTime", now)
            self.param_tree[f"{prefix}SpecificProblem"] = ("xsd:string", tmpl["SpecificProblem"])
            self.param_tree[f"{prefix}AdditionalInformation"] = ("xsd:string", f"Simulated alarm #{self.alarm_counter}")
            self.param_tree[f"{prefix}AdditionalText"] = ("xsd:string", f"VENDOR-{random.randint(1000, 9999)}")
            self.active_alarms.append(alarm_id)
            self._log(f"模拟告警产生: {alarm_id} ({tmpl['PerceivedSeverity']})", "yellow")

        # 发起告警 Inform 会话
        self._run_session([("101 ALARM", "")])

    def _simulate_pm_upload(self):
        """模拟 PM 文件定时上传"""
        pm_url = self.param_tree.get(
            "Device.Services.FAPService.1.FAPControl.LTE.PeriodicStatistics.1.URL",
            ("xsd:string", "")
        )[1]
        if not pm_url:
            # ACS 真实 endpoint 是 /smallcell/FileUploadService（见 internal/acs/server.go:105）。
            # 模拟器历史拼成 /FileUploadService 命中 404；T-0164-P1 G1 E2E 验证时发现并修复。
            pm_url = f"{self.parsed_url.scheme}://{self.parsed_url.hostname}:{self.parsed_url.port or 80}/smallcell/FileUploadService?fileType=PM&fileName=pm-{self.serial_number}.xml.gz&sn={self.serial_number}"
        self._log("模拟 PM 文件上传", "blue")
        file_size = self._do_file_upload(pm_url, "104 PM File")
        if file_size > 0:
            # 发送 AutonomousTransferComplete
            threading.Thread(
                target=self._run_autonomous_transfer_complete,
                args=("104 PM File", pm_url, file_size),
                daemon=True,
            ).start()

    def _simulate_mr_upload(self):
        """模拟 MR 文件定时上传"""
        mr_url = self.param_tree.get(
            "Device.Services.FAPService.1.FAPControl.LTE.MR.MrUrl",
            ("xsd:string", "")
        )[1]
        if not mr_url:
            mr_url = f"{self.parsed_url.scheme}://{self.parsed_url.hostname}:{self.parsed_url.port or 80}/smallcell/FileUploadService?fileType=MR&fileName=mr-{self.serial_number}.xml.gz&sn={self.serial_number}"
        self._log("模拟 MR 文件上传", "blue")
        file_size = self._do_file_upload(mr_url, "105 MR File")
        if file_size > 0:
            threading.Thread(
                target=self._run_autonomous_transfer_complete,
                args=("105 MR File", mr_url, file_size),
                daemon=True,
            ).start()

    def _run_session(self, event_codes):
        """运行一次完整的 TR069 会话（带会话级连接复用）"""
        with self._session_lock:
            self._run_session_inner(event_codes)

    def _run_session_inner(self, event_codes):
        """实际会话逻辑"""
        self.inform_count += 1
        events_str = ", ".join(ec for ec, _ in event_codes)
        self._log(
            f"━━━ Session #{self.inform_count} ━━━ Events: [{events_str}]",
            "bold",
        )

        # 创建会话级连接
        self._session_conn = None

        # Step 1: 发送 Inform
        cwmp_id, inform_xml = self._build_inform(event_codes)
        self._log(f"→ Inform (cwmp:ID={cwmp_id}, {len(inform_xml)} bytes)")

        try:
            status, resp_body = self._http_request(body=inform_xml)
        except Exception as e:
            self._log(f"✗ 连接失败: {e}", "red")
            self.error_count += 1
            self._close_session_conn()
            return

        if status != 200:
            self._log(f"✗ Inform 失败: HTTP {status}", "red")
            if resp_body:
                self._log(f"  Body: {resp_body.decode('utf-8', errors='replace')[:200]}", "red")
            self.error_count += 1
            self._close_session_conn()
            return

        resp_cwmp_id = extract_cwmp_id(resp_body)
        self._log(f"← InformResponse OK (cwmp:ID={resp_cwmp_id})", "green")

        # Step 2: 进入 RPC 处理循环
        rpc_round = 0
        while self.running:
            rpc_round += 1
            self._log(f"→ Empty POST (polling RPC #{rpc_round})")

            try:
                status, resp_body = self._http_request(body=None)
            except Exception as e:
                self._log(f"✗ RPC 轮询失败: {e}", "red")
                self.error_count += 1
                break

            # 处理 ACS 响应（可能包含管道化 RPC，循环处理直到 204/empty）
            session_done = self._process_rpc_response_loop(status, resp_body, rpc_round)
            if session_done:
                break

        # 关闭会话级连接
        self._close_session_conn()
        self.session_cookie = None

        # 处理异步 TransferComplete（Download/Upload 后）
        pending = self.rpc_handler.pending_transfer
        if pending:
            self.rpc_handler.pending_transfer = None
            command_key, file_type, is_download, url = pending
            self._log(f"调度异步 TransferComplete (delay=2s)", "yellow")
            def _delayed_tc():
                time.sleep(2)
                self._run_transfer_complete_session(command_key, file_type, is_download, url)
            threading.Thread(target=_delayed_tc, daemon=True).start()

    def _process_rpc_response_loop(self, status, resp_body, rpc_round):
        """处理 ACS 响应中的 RPC，支持无限层管道化。返回 True 表示会话结束。"""
        while self.running:
            # 检查会话结束
            if status == 204 or (status == 200 and len(resp_body) == 0):
                self._log(f"← 204 No Content — Session 结束 (执行了 {rpc_round} 个 RPC)", "green")
                return True

            if status != 200:
                self._log(f"✗ 意外响应: HTTP {status}", "red")
                self.error_count += 1
                return True

            # 检测 RPC 方法
            method = detect_rpc_method(resp_body)
            if not method:
                if b"</soap:" in resp_body and len(resp_body) < 500:
                    self._log("← 空 SOAP 响应 — Session 结束", "green")
                    return True
                self._log(f"✗ 无法识别 RPC 方法", "yellow")
                self._log(f"  Body: {resp_body.decode('utf-8', errors='replace')[:300]}", "gray")
                return True

            resp_cwmp_id = extract_cwmp_id(resp_body)
            self._log(f"← RPC: {colored(method, 'bold')} (cwmp:ID={resp_cwmp_id})")
            self.rpc_count += 1

            body_el = parse_soap_body(resp_body)
            if body_el is None:
                self._log("✗ 无法解析 SOAP Body", "red")
                self.error_count += 1
                return True

            rpc_response = self.rpc_handler.handle(resp_cwmp_id, method, body_el)

            # 发送 RPC 响应，ACS 可能在回复中管道化下一个 RPC
            try:
                status, resp_body = self._http_request(body=rpc_response)
            except Exception as e:
                self._log(f"✗ 发送 RPC 响应失败: {e}", "red")
                self.error_count += 1
                return True

            # 循环继续：检查这个响应是否又包含 RPC（管道化）

        return True  # self.running == False

    def _process_rpc_until_done(self, status, resp_body):
        """处理可能的管道化 RPC 响应，直到会话结束。用于 TransferComplete 等会话。"""
        while self.running:
            if status == 204 or (status == 200 and len(resp_body) == 0):
                break
            if status != 200:
                break
            method = detect_rpc_method(resp_body)
            if not method:
                break
            resp_cwmp_id = extract_cwmp_id(resp_body)
            self._log(f"← RPC: {colored(method, 'bold')} (cwmp:ID={resp_cwmp_id})")
            self.rpc_count += 1
            body_el = parse_soap_body(resp_body)
            if body_el is None:
                break
            rpc_response = self.rpc_handler.handle(resp_cwmp_id, method, body_el)
            try:
                status, resp_body = self._http_request(body=rpc_response)
            except Exception:
                break

    def run(self, once=False):
        """主运行循环"""
        self._log(f"启动 CPE 模拟器", "bold")
        self._log(f"  ACS URL:    {self.acs_url}")
        self._log(f"  序列号:     {self.serial_number}")
        self._log(f"  OUI:        {self.oui}")
        self._log(f"  ProductClass: {self.product_class}")
        self._log(f"  Inform间隔: {self.interval}s")
        self._log(f"  初始事件:   {self.initial_event}")
        if self.fault_rate > 0:
            self._log(f"  Fault模拟:  {self.fault_rate * 100:.0f}%")
        if self.alarm_interval > 0:
            self._log(f"  告警模拟:   每{self.alarm_interval}s")
        if self.pm_upload_interval > 0:
            self._log(f"  PM上传:     每{self.pm_upload_interval}s")
        if self.mr_upload_interval > 0:
            self._log(f"  MR上传:     每{self.mr_upload_interval}s")
        self._log("")

        # 启动 Connection Request 服务器
        if self.enable_connreq:
            cr_url = self.param_tree.get("Device.ManagementServer.ConnectionRequestURL", ("", ""))[1]
            try:
                cr_port = int(urlparse(cr_url).port or 7547)
            except (ValueError, TypeError):
                cr_port = 7547 + self.cpe_index
            # 为多 CPE 模式避免端口冲突
            cr_port = cr_port + self.cpe_index
            self.param_tree["Device.ManagementServer.ConnectionRequestURL"] = (
                "xsd:string", f"http://127.0.0.1:{cr_port}"
            )
            self.connreq_server = ConnectionRequestServer(cr_port, self)
            if self.connreq_server.start():
                self._log(f"  ConnReq 服务: 监听端口 {cr_port}", "green")
            else:
                self._log(f"  ConnReq 服务: 端口 {cr_port} 不可用", "yellow")
                self.connreq_server = None

        # 启动告警模拟定时器
        if self.alarm_interval > 0:
            def _alarm_loop():
                while self.running:
                    for _ in range(self.alarm_interval * 10):
                        if not self.running:
                            return
                        time.sleep(0.1)
                    if self.running:
                        self._simulate_alarm()
            threading.Thread(target=_alarm_loop, daemon=True).start()

        # 启动 PM 上传模拟定时器
        if self.pm_upload_interval > 0:
            def _pm_loop():
                while self.running:
                    for _ in range(self.pm_upload_interval * 10):
                        if not self.running:
                            return
                        time.sleep(0.1)
                    if self.running:
                        self._simulate_pm_upload()
            threading.Thread(target=_pm_loop, daemon=True).start()

        # 启动 MR 上传模拟定时器
        if self.mr_upload_interval > 0:
            def _mr_loop():
                while self.running:
                    for _ in range(self.mr_upload_interval * 10):
                        if not self.running:
                            return
                        time.sleep(0.1)
                    if self.running:
                        self._simulate_mr_upload()
            threading.Thread(target=_mr_loop, daemon=True).start()

        # 第一次 Inform
        events = EVENT_MAP.get(self.initial_event, [("2 PERIODIC", "")])
        self._run_session(events)

        if once:
            self._print_stats()
            return

        # 后续 Periodic Inform 循环
        while self.running:
            for _ in range(self.interval * 10):
                if not self.running:
                    break
                # 检查 Connection Request
                if self.pending_connection_request:
                    self.pending_connection_request = False
                    self._run_session([("6 CONNECTION REQUEST", "")])
                time.sleep(0.1)

            if not self.running:
                break

            if self.pending_boot:
                events = [("1 BOOT", ""), ("M Reboot", "")]
                self.pending_boot = False
            else:
                events = [("2 PERIODIC", "")]

            self._run_session(events)

        self._print_stats()

    def _print_stats(self):
        self._log("")
        self._log(f"══ 统计 ══", "bold")
        self._log(f"  Inform 会话: {self.inform_count}")
        self._log(f"  RPC 执行:    {self.rpc_count}")
        self._log(f"  错误:        {self.error_count}")

    def stop(self):
        self.running = False
        if self.connreq_server:
            self.connreq_server.stop()


# ──────────────────────────── 多 CPE 管理器 ────────────────────────────


class MultiCPEManager:
    """管理多个并发 CPE 模拟器"""

    def __init__(self, acs_url, count, base_sn="SIM-CPE-", oui="48BF74", product_class="FAP/pCRB2000/SC",
                 interval=60, event_type="bootstrap", stagger=True, fault_rate=0.0, log_dir=None,
                 alarm_interval=0, pm_upload_interval=0, mr_upload_interval=0, enable_connreq=False):
        self.acs_url = acs_url
        self.count = count
        self.base_sn = base_sn
        self.oui = oui
        self.product_class = product_class
        self.interval = interval
        self.event_type = event_type
        self.stagger = stagger
        self.fault_rate = fault_rate
        self.log_dir = log_dir
        self.alarm_interval = alarm_interval
        self.pm_upload_interval = pm_upload_interval
        self.mr_upload_interval = mr_upload_interval
        self.enable_connreq = enable_connreq
        self.cpes = []
        self.threads = []

    def start(self, once=False):
        print(colored(f"\n{'═' * 60}", "bold"))
        print(colored(f"  OMC TR069 CPE 模拟器 — 多设备模式", "bold"))
        print(colored(f"  设备数量: {self.count}", "bold"))
        print(colored(f"  ACS URL:  {self.acs_url}", "bold"))
        print(colored(f"  Inform 间隔: {self.interval}s", "bold"))
        print(colored(f"  初始事件: {self.event_type}", "bold"))
        if self.fault_rate > 0:
            print(colored(f"  Fault 模拟: {self.fault_rate * 100:.0f}%", "bold"))
        if self.log_dir:
            print(colored(f"  报文日志: {self.log_dir}", "bold"))
        print(colored(f"{'═' * 60}\n", "bold"))

        for i in range(self.count):
            sn = f"{self.base_sn}{i + 1:04d}"
            color = CPE_COLORS[i % len(CPE_COLORS)]
            cpe = CPESimulator(
                acs_url=self.acs_url,
                serial_number=sn,
                oui=self.oui,
                product_class=self.product_class,
                interval=self.interval,
                event_type=self.event_type,
                color=color,
                cpe_index=i,
                fault_rate=self.fault_rate,
                log_dir=self.log_dir,
                alarm_interval=self.alarm_interval,
                pm_upload_interval=self.pm_upload_interval,
                mr_upload_interval=self.mr_upload_interval,
                enable_connreq=self.enable_connreq,
            )
            self.cpes.append(cpe)

            def run_cpe(c=cpe, o=once, delay=i):
                if self.stagger and delay > 0:
                    stagger_delay = delay * min(2.0, self.interval / self.count)
                    time.sleep(stagger_delay)
                c.run(once=o)

            t = threading.Thread(target=run_cpe, daemon=True, name=f"cpe-{sn}")
            self.threads.append(t)
            t.start()

        try:
            for t in self.threads:
                t.join()
        except KeyboardInterrupt:
            self.stop()

    def stop(self):
        print(colored("\n\n停止所有 CPE...", "yellow"))
        for cpe in self.cpes:
            cpe.stop()
        for t in self.threads:
            t.join(timeout=5)
        self._print_summary()

    def _print_summary(self):
        print(colored(f"\n{'═' * 60}", "bold"))
        print(colored(f"  汇总统计", "bold"))
        print(colored(f"{'═' * 60}", "bold"))
        total_inform = sum(c.inform_count for c in self.cpes)
        total_rpc = sum(c.rpc_count for c in self.cpes)
        total_err = sum(c.error_count for c in self.cpes)
        print(f"  设备数:     {self.count}")
        print(f"  总 Inform:  {total_inform}")
        print(f"  总 RPC:     {total_rpc}")
        print(f"  总错误:     {colored(str(total_err), 'red' if total_err else 'green')}")
        print(colored(f"{'═' * 60}\n", "bold"))


# ──────────────────────────── 入口 ────────────────────────────


def main():
    parser = argparse.ArgumentParser(
        description="OMC TR069 CPE 模拟器（中国移动皮基站规范合规版）",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  # 单 CPE 调试（默认 periodic 事件）
  python3 %(prog)s --acs http://localhost:8080/smallcell/AcsService

  # 单 CPE + bootstrap 事件（注册新设备）
  python3 %(prog)s --acs http://localhost:8080/smallcell/AcsService --event bootstrap

  # 单 CPE + 告警模拟（每 30 秒产生/清除告警）
  python3 %(prog)s --acs http://localhost:8080/smallcell/AcsService --alarm-interval 30

  # 单 CPE + PM 文件定时上传 + MR 文件定时上传
  python3 %(prog)s --acs http://localhost:8080/smallcell/AcsService --pm-upload-interval 300 --mr-upload-interval 900

  # 5%% 概率模拟 Fault 响应
  python3 %(prog)s --acs http://localhost:8080/smallcell/AcsService --fault-rate 0.05

  # 只运行一次 Inform（不循环）
  python3 %(prog)s --acs http://localhost:8080/smallcell/AcsService --once

  # 10 个 CPE 并发测试 + 报文日志
  python3 %(prog)s --acs http://localhost:8080/smallcell/AcsService --count 10 --log-dir ./cpe-logs

  # 启用 Connection Request 监听
  python3 %(prog)s --acs http://localhost:8080/smallcell/AcsService --enable-connreq
""",
    )

    parser.add_argument(
        "--acs",
        default="http://localhost:8080/smallcell/AcsService",
        help="ACS URL (default: http://localhost:8080/smallcell/AcsService)",
    )
    parser.add_argument("--sn", default=None, help="CPE 序列号 (单 CPE 模式，默认自动生成)")
    parser.add_argument("--oui", default="48BF74", help="OUI (default: 48BF74)")
    parser.add_argument("--product-class", default="FAP/pCRB2000/SC", help="Product Class")
    parser.add_argument("--manufacturer", default="Baicells", help="Manufacturer")
    parser.add_argument(
        "--interval", type=int, default=60, help="Periodic Inform 间隔秒数 (default: 60)"
    )
    parser.add_argument(
        "--event",
        choices=list(EVENT_MAP.keys()),
        default="periodic",
        help="初始 Inform 事件类型 (default: periodic)",
    )
    parser.add_argument("--count", type=int, default=1, help="CPE 数量 (>1 启用多 CPE 模式)")
    parser.add_argument("--base-sn", default="SIM-CPE-", help="多 CPE 模式下的序列号前缀")
    parser.add_argument("--once", action="store_true", help="只运行一次 Inform 后退出")
    parser.add_argument("--no-stagger", action="store_true", help="多 CPE 同时启动（不错开）")
    parser.add_argument(
        "--fault-rate", type=float, default=0.0,
        help="RPC Fault 模拟概率 (0.0-1.0, default: 0.0)"
    )
    parser.add_argument(
        "--log-dir", default=None,
        help="报文日志保存目录（每 CPE 一个文件，默认不保存）"
    )
    parser.add_argument(
        "--alarm-interval", type=int, default=0,
        help="告警模拟间隔秒数 (0=关闭, default: 0)"
    )
    parser.add_argument(
        "--pm-upload-interval", type=int, default=0,
        help="PM 文件上传模拟间隔秒数 (0=关闭, default: 0)"
    )
    parser.add_argument(
        "--mr-upload-interval", type=int, default=0,
        help="MR 文件上传模拟间隔秒数 (0=关闭, default: 0)"
    )
    parser.add_argument(
        "--enable-connreq", action="store_true",
        help="启用 Connection Request 监听服务"
    )

    args = parser.parse_args()

    # 信号处理
    if args.count > 1:
        mgr = MultiCPEManager(
            acs_url=args.acs,
            count=args.count,
            base_sn=args.base_sn,
            oui=args.oui,
            product_class=args.product_class,
            interval=args.interval,
            event_type=args.event,
            stagger=not args.no_stagger,
            fault_rate=args.fault_rate,
            log_dir=args.log_dir,
            alarm_interval=args.alarm_interval,
            pm_upload_interval=args.pm_upload_interval,
            mr_upload_interval=args.mr_upload_interval,
            enable_connreq=args.enable_connreq,
        )

        def signal_handler(sig, frame):
            mgr.stop()
            sys.exit(0)

        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)

        mgr.start(once=args.once)
    else:
        sn = args.sn or f"SIM-CPE-{uuid.uuid4().hex[:8].upper()}"
        cpe = CPESimulator(
            acs_url=args.acs,
            serial_number=sn,
            oui=args.oui,
            product_class=args.product_class,
            manufacturer=args.manufacturer,
            interval=args.interval,
            event_type=args.event,
            color="cyan",
            fault_rate=args.fault_rate,
            log_dir=args.log_dir,
            alarm_interval=args.alarm_interval,
            pm_upload_interval=args.pm_upload_interval,
            mr_upload_interval=args.mr_upload_interval,
            enable_connreq=args.enable_connreq,
        )

        def signal_handler(sig, frame):
            cpe.stop()

        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)

        print(colored(f"\n{'═' * 60}", "bold"))
        print(colored(f"  OMC TR069 CPE 模拟器 — 单设备模式（规范合规版）", "bold"))
        print(colored(f"{'═' * 60}\n", "bold"))

        cpe.run(once=args.once)


# ============================================================================
# F05 MR 直传模式（与完整 TR-069 会话解耦）
# ----------------------------------------------------------------------------
# 用途：E2E 验证 F05 MR 测量任务功能。OMC 开启 MR 任务后会下发 SPV，
# 设备应按 UploadPeriod 周期 HTTP POST MR XML 到
# `http://{OMC_HOST}/smallcell/FileUploadService?fileType=MR&cellCode=...&filename=...`
# 这个函数模拟那个上传行为，但不跑完整 TR-069 会话，让 E2E 测试更快、更可控。
#
# 用法（独立 entrypoint）：
#   python3 scripts/cpe_simulator.py --mr-direct-upload \
#       --omc-base-url http://localhost:8088 \
#       --cell-code CELL001 \
#       --sn SN001 \
#       --upload-period 15 \   # 间隔分钟数，1 表示每分钟一次
#       --count 3              # 总共上传几次（0=无限循环到 Ctrl-C）
# ============================================================================

def _build_sample_mr_xml(cell_code, mr_type="MRS"):
    """生成一份最小可解析的 MR XML（结构对齐 MR_Feature_Analysis.md §9）。"""
    import datetime as _dt
    now = _dt.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S+00:00")
    return f'''<?xml version="1.0" encoding="UTF-8"?>
<mfm>
  <fileHeader startTime="{now}" endTime="{now}" period="900"/>
  <eNB>
    <measurement>
      <smr>MR.LteScRSRP MR.DegreesLatitude MR.DegreesLongitude MR.LteScPCI</smr>
      <object MmeUeS1apId="12345" TimeStamp="{now}">
        <v>60 NIL NIL 101</v>
      </object>
    </measurement>
  </eNB>
</mfm>'''


def _build_sample_mr_filename(cell_code, sn, mr_type="MRS"):
    """按文档 §8 格式生成文件名：FDD-baicells-MRS-{IP}-{CellSN}-{SN}-{datetime}.xml"""
    import datetime as _dt
    ts = _dt.datetime.utcnow().strftime("%Y%m%d%H%M%S")
    return f"FDD-baicells-{mr_type}-127.0.0.1-{cell_code}-{sn}-{ts}.xml"


def run_mr_direct_upload(omc_base_url, cell_code, sn, upload_period_minutes, count):
    """循环上传 MR XML，用于 F05 E2E 验证。"""
    import time as _time
    import urllib.request as _ur
    import urllib.parse as _up

    interval_sec = max(upload_period_minutes * 60, 5)  # 测试时允许 5s 起步
    base = omc_base_url.rstrip("/")
    sent = 0
    print(f"[mr-direct-upload] target={base}/smallcell/FileUploadService cell={cell_code} sn={sn} interval={interval_sec}s count={count or '∞'}")

    try:
        while count == 0 or sent < count:
            filename = _build_sample_mr_filename(cell_code, sn)
            body = _build_sample_mr_xml(cell_code).encode("utf-8")
            url = (
                f"{base}/smallcell/FileUploadService"
                f"?fileType=MR&cellCode={_up.quote(cell_code)}"
                f"&sn={_up.quote(sn)}&filename={_up.quote(filename)}"
            )
            req = _ur.Request(url, data=body, method="POST",
                              headers={"Content-Type": "application/xml"})
            try:
                resp = _ur.urlopen(req, timeout=10)
                status = resp.getcode()
                print(f"[mr-direct-upload] #{sent + 1} POST {status} {filename}")
            except Exception as e:
                print(f"[mr-direct-upload] #{sent + 1} FAILED: {e}")
            sent += 1
            if count != 0 and sent >= count:
                break
            _time.sleep(interval_sec)
    except KeyboardInterrupt:
        print(f"[mr-direct-upload] interrupted after {sent} uploads")


def _mr_direct_upload_entrypoint(argv=None):
    p = argparse.ArgumentParser(description="F05 MR 直传模式（E2E 验证用）")
    p.add_argument("--mr-direct-upload", action="store_true", required=True,
                   help="启用 MR 直传模式")
    p.add_argument("--omc-base-url", default="http://localhost:8088",
                   help="OMC 主地址（不含 /smallcell/FileUploadService）")
    p.add_argument("--cell-code", required=True, help="目标 cell code")
    p.add_argument("--sn", required=True, help="设备 SN")
    p.add_argument("--upload-period", type=int, default=15,
                   help="上传间隔（分钟）；测试时可填 1 加快验证")
    p.add_argument("--count", type=int, default=3,
                   help="上传次数；0=无限循环到 Ctrl-C")
    args = p.parse_args(argv)
    run_mr_direct_upload(args.omc_base_url, args.cell_code, args.sn,
                         args.upload_period, args.count)


if __name__ == "__main__":
    import sys
    # 子命令优先：sys.argv 含 --mr-direct-upload → 走独立 entrypoint
    if "--mr-direct-upload" in sys.argv:
        _mr_direct_upload_entrypoint()
    else:
        main()
