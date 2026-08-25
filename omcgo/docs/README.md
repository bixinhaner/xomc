# OMC 基站管理系统 - 文档目录

本目录是 OMC 项目的统一文档中心，包含所有技术设计、架构文档、功能规范和开发报告。

功能分析基于 `规范/` 目录下三大运营商（中国移动、中国电信、中国联通）的 55 份技术规范文档（移动 19 份、电信 21 份、联通 15 份），涵盖 LTE 和 5G NR 两种制式。

## 目录结构

```
omcgo/doc/
├── README.md                    # 本文件 - 文档索引
├── 功能索引.md                   # 功能分类总览、运营商交叉引用矩阵
├── development-plan.md          # 开发计划
├── phase4-analysis-report.md    # Phase 4 分析报告
│
├── architecture/                # 系统架构文档
│   ├── system-overview.md       # OMC 系统定位、设备类型、运营商差异
│   ├── backend-design.md        # Go 后端架构设计
│   ├── interface-topology.md    # 南向/北向/直连接口拓扑
│   └── framework-comparison.md  # 技术框架选型分析
│
├── design/                      # 技术设计文档
│   ├── acs-service-flow.md      # ACS 服务流程设计
│   ├── database-migration.md    # 数据库迁移方案
│   ├── pm-kpi-flow.md           # PM/KPI 数据流设计
│   ├── session-design.md        # 会话设计
│   └── ...                      # 其他设计文档
│
├── detailed-design/             # 详细设计文档 (F01-F21)
│   ├── 01-project-scaffolding.md
│   ├── 07-acs-engine.md
│   └── ...
│
├── features/                    # 功能域文档 (F01-F10)
│   ├── 01-southbound-interface.md
│   ├── 06-omc-core-functions.md
│   └── ...
│
├── go-zero-design/              # Go-Zero 架构设计
│   ├── 01-architecture-overview.md
│   ├── 03-acs-engine.md
│   └── ...
│
├── operations/                  # 运维文档
│   └── deployment-guide.md      # 部署指南
│
├── reports/                     # 分析报告
│   ├── phase1-completion-report.md
│   ├── phase2-completion-report.md
│   ├── phase3-completion-report.md
│   ├── phase4-completion-report.md
│   ├── review-report/           # 代码审查报告
│   │   └── YYYYMMDD/
│   └── ...
│
└── specs-inventory/             # 规范清单
    ├── document-catalog.md      # 55 份规范文档完整目录
    └── carrier-comparison.md    # 三大运营商规范覆盖对比
```

## 文档导航

### 核心索引

| 文件 | 说明 |
|------|------|
| [功能索引.md](功能索引.md) | 功能分类总览、详细功能列表、运营商交叉引用矩阵、文档映射索引 |

### 系统架构

| 文件 | 说明 |
|------|------|
| [system-overview.md](architecture/system-overview.md) | OMC 系统定位、设备类型、制式支持、运营商差异概述 |
| [interface-topology.md](architecture/interface-topology.md) | 南向/北向/直连接口拓扑关系、协议栈、数据流向 |
| [backend-design.md](architecture/backend-design.md) | **Go 后端架构设计**：技术栈选型、模块分解、ACS 引擎、数据存储、扩展性设计（10万→100万） |

### 技术设计

| 文件 | 说明 |
|------|------|
| [0026-compose-watchdog-design.md](design/0026-compose-watchdog-design.md) | **Docker Compose Watchdog 开发设计**：探针能力边界、故障门限、可热加载配置、Docker API、自愈策略、Prometheus 分工与 Watchdog 自保护 |

### 规范清单

| 文件 | 说明 |
|------|------|
| [document-catalog.md](specs-inventory/document-catalog.md) | 55 份规范文档完整目录，含文件路径，按运营商/类型/制式三维视图 |
| [carrier-comparison.md](specs-inventory/carrier-comparison.md) | 三大运营商规范覆盖对比、功能差异分析、版本演进 |

### 功能域详细文档

| 文件 | 功能域 | 子功能数 |
|------|--------|---------|
| [01-southbound-interface.md](features/01-southbound-interface.md) | F01 南向接口管理（TR069） | 6 |
| [02-data-model.md](features/02-data-model.md) | F02 数据模型与配置管理 | 7 |
| [03-performance-management.md](features/03-performance-management.md) | F03 性能管理（PM/KPI） | 5 |
| [04-alarm-management.md](features/04-alarm-management.md) | F04 告警管理 | 3 |
| [05-measurement-reports.md](features/05-measurement-reports.md) | F05 测量报告（MR） | 4 |
| [06-omc-core-functions.md](features/06-omc-core-functions.md) | F06 OMC-R 核心功能 | 5 |
| [07-ne-direct-connection.md](features/07-ne-direct-connection.md) | F07 网元直连接口 | 3 |
| [08-northbound-oss.md](features/08-northbound-oss.md) | F08 北向/OSS 接口 | 4 |
| [09-auto-provisioning.md](features/09-auto-provisioning.md) | F09 自动开站/自动开通 | 3 |
| [10-interop-testing.md](features/10-interop-testing.md) | F10 互操作测试 | 3 |

## 功能域速查

- **F01** 南向接口管理（TR069）
- **F02** 数据模型与配置管理
- **F03** 性能管理（PM/KPI）
- **F04** 告警管理
- **F05** 测量报告（MR）
- **F06** OMC-R 核心功能
- **F07** 网元直连接口
- **F08** 北向/OSS 接口
- **F09** 自动开站/自动开通
- **F10** 互操作测试
