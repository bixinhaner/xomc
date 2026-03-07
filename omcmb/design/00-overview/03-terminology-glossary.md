# 术语表 (Terminology Glossary)

中英双语术语对照表，按功能分类组织。

## 网络元素 (Network Elements)

| 中文 | English | 缩写 | 说明 |
|------|---------|------|------|
| 网元 | Network Element | NE | 网络中可管理的最小单元 |
| 基带单元 | Baseband Unit | BU/BBU | 处理基带信号的设备 |
| 扩展单元 | Extension Unit | EU | 扩展基带处理能力 |
| 射频单元 | Radio Unit | RU/RRU | 射频信号处理单元 |
| 小区 | Cell | — | 基站覆盖的逻辑区域 |
| 板卡 | Slot/Board | — | 设备内部功能插件 |
| 近端单元 | Near-End Unit | — | 靠近核心网侧的处理单元 |
| 远端单元 | Far-End Unit | — | 靠近天线侧的处理单元 |

## 设备类型 (Device Types)

| 中文 | English | 缩写 | 说明 |
|------|---------|------|------|
| 4G 基站 | 4G Base Station | eNB | LTE 演进型基站 |
| 5G 基站 | 5G Base Station | gNB | NR 下一代基站 |
| 用户终端设备 | Customer Premises Equipment | CPE | 用户侧接入设备 |
| 网关 | Gateway | eGW | 网络边界网关设备 |
| 微站 | Picostation | Pico | 小型基站（家庭/企业/扩展/大功率） |
| 无线接入点 | Wi-Fi Access Point | Wi-Fi AP | 无线局域网接入 |

## 核心网 (Core Network)

| 中文 | English | 缩写 | 说明 |
|------|---------|------|------|
| 接入和移动管理功能 | Access and Mobility Management Function | AMF | 5G 核心网元 |
| 移动性管理实体 | Mobility Management Entity | MME | 4G 核心网元 |
| 服务网关 | Serving Gateway | S-GW | 4G 数据面网关 |
| PDN 网关 | Packet Data Network Gateway | P-GW | 4G 外部网络网关 |

## 标识符 (Identifiers)

| 中文 | English | 缩写 | 说明 |
|------|---------|------|------|
| 序列号 | Serial Number | SN | 设备唯一物理标识 |
| 物理小区标识 | Physical Cell Identifier | PCI | 小区物理层标识 |
| 5G 小区标识 | NR Cell Identifier | NCI | 5G 小区全局标识 |
| 4G 小区标识 | E-UTRAN Cell Identifier | ECI | 4G 小区全局标识 |
| 公共陆地移动网络 | Public Land Mobile Network | PLMN | 运营商网络标识 |
| 跟踪区域代码 | Tracking Area Code | TAC | LTE 跟踪区域标识 |
| 位置区域代码 | Location Area Code | LAC | GSM 位置区标识 |
| 绝对射频信道号 | E-UTRA Absolute Radio Frequency Channel Number | EARFCN | LTE 频点 |
| NR 绝对射频信道号 | NR Absolute Radio Frequency Channel Number | NRARFCN | 5G 频点 |
| 基站标识 | Base Station ID | eNB ID / gNB ID | 基站在网络中的标识 |

## 告警 (Alarm)

| 中文 | English | 等级 | 颜色 |
|------|---------|------|------|
| 紧急 | Critical | Level 1 (最高) | 红色 `#F5222D` |
| 主要 | Major | Level 2 | 橙色 `#FA8C16` |
| 次要 | Minor | Level 3 | 黄色 `#FAAD14` |
| 警告 | Warning | Level 4 (最低) | 蓝色 `#1890FF` |
| 告警码 | Alarm Code | — | 告警唯一分类编码 |
| 告警确认 | Alarm Acknowledge | — | 运维人员确认知晓 |
| 告警清除 | Alarm Clear | — | 告警条件消除 |
| 告警屏蔽 | Alarm Mute/Suppress | — | 临时屏蔽特定告警 |

## 性能 (Performance)

| 中文 | English | 缩写 | 说明 |
|------|---------|------|------|
| 关键性能指标 | Key Performance Indicator | KPI | 网络性能度量指标 |
| 参考信号接收功率 | Reference Signal Received Power | RSRP | 信号强度指标 |
| 参考信号接收质量 | Reference Signal Received Quality | RSRQ | 信号质量指标 |
| 信号与干扰加噪声比 | Signal to Interference plus Noise Ratio | SINR | 信噪比指标 |
| 吞吐量 | Throughput | — | 数据传输速率 |
| 测量报告 | Measurement Report | MR | 终端测量上报数据 |
| 性能门限 | Performance Threshold | — | KPI 告警触发阈值 |

## 运维操作 (Operations)

| 中文 | English | 缩写 | 说明 |
|------|---------|------|------|
| 人机语言 | Man-Machine Language | MML | 设备命令行接口 |
| 开站 | Commissioning | — | 新基站入网开通 |
| 交维 | Handover to Maintenance | — | 从工程移交运维 |
| 基线 | Baseline | — | 参数配置标准模板 |
| 固件 | Firmware | — | 设备底层软件 |
| 参数同步 | Parameter Synchronization | — | 从设备同步参数到网管 |
| 软件激活 | Software Activation | — | 激活新版本软件 |
| 版本升级 | Version Upgrade | — | 设备软件版本更新 |
| 配置下发 | Configuration Delivery | — | 从网管下发参数到设备 |
| 邻区 | Neighbor Cell | — | 相邻小区关系 |
