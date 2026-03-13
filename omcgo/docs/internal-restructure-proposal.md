# 方案：internal/ 目录重构 — 分离核心公共代码与功能模块

## 1. 现状分析

### 1.1 规模

| 指标 | 数值 |
|------|------|
| 顶层包数量 | 32 |
| Go 文件数 | 214 |
| 代码行数 | ~41,500 |

### 1.2 当前结构（全部扁平）

```
internal/
├── acs/            # F01 ACS 引擎
├── admin/          # F06 用户权限
├── alarm/          # F04 告警管理
├── appconfig/      # 配置结构体
├── backup/         # F06 备份恢复
├── bootstrap/      # 启动框架
├── carrier/        # 运营商适配器
├── components/     # 基础设施连接器
├── config/         # F02 数据模型与配置
├── dashboard/      # F06 仪表盘
├── device/         # F06 设备管理
├── errors/         # 错误码
├── event/          # 事件总线
├── filemanager/    # F06 文件管理
├── interop/        # F10 互操作测试
├── license/        # F06 许可管理
├── middleware/     # HTTP 中间件
├── mml/            # F06 MML 控制台
├── model/          # 领域模型
├── mr/             # F05 测量报告
├── nedirect/       # F07 网元直连
├── northbound/     # F08 北向接口
├── ops/            # F06 运维工具
├── pm/             # F03 性能管理
├── provision/      # F09 自动开站
├── report/         # F06 报表
├── software/       # F06 固件管理
├── syslog/         # F06 系统日志
├── topology/       # F06 拓扑管理
├── transfer/       # 数据传输
└── utils/          # 工具函数
```

### 1.3 问题

1. **核心与业务混杂** — `model/`、`errors/`、`event/` 等被 30+ ���包依赖的基础包和 `alarm/`、`ops/` 等业务模块并列，无法一眼区分层级
2. **新人认知负担** — 32 个扁平包，没有结构暗示哪些是基础设施、哪些是可选业务模块
3. **依赖方向不清** — 从目录结构无法看出 `model` 是底层包还是和 `mml` 平级
4. **未来扩展风险** — 随着新功能域添加，包数量会继续增长，扁平结构越来越难管理

---

## 2. 依赖分析

### 2.1 依赖层级（自底向上）

```
Level 0 — 无依赖（纯定义）
  ├── model        (被 33 个包导入)
  ├── errors       (被 21 个包导入)
  ├── event        (被 13 个包导入)
  ├── appconfig    (被 10 个包导入)
  ├── middleware    (被 1 个包导入)
  └── utils        (被 0 个包导入)

Level 1 — 仅依赖 Level 0
  ├── carrier      (被 10 个包导入)  → model
  ├── components   (被 1 个包导入)   → appconfig
  ├── admin        → errors, model
  ├── license      → errors, model
  ├── mml          → errors, model
  ├── ops          → errors, model
  ├── syslog       → errors, model
  └── topology     → errors, model

Level 2 — 依赖 Level 0-1
  ├── acs          → appconfig, event, acs/*
  ├── alarm        → carrier, errors, event, model
  ├── device       → carrier, errors, event, model
  ├── pm           → carrier, errors, event, model, pm/*
  ├── mr           → errors, event, model, mr/*
  └── ...

Level 3+ — 跨模块依赖
  ├── dashboard    → admin, alarm, device, pm/kpi, topology
  ├── provision    → acs/cmdqueue, carrier, config/*, device
  ├── northbound   → alarm, device, pm/counter, pm/kpi
  └── ...

Level TOP — 启动框架
  └── bootstrap    → appconfig, carrier, components, event, acs/cmdqueue
```

### 2.2 核心公共包（被 10+ 个包依赖）

| 包 | 被导入次数 | 角色 |
|----|-----------|------|
| model | 33 | 领域模型、常量、类型定义 |
| errors | 21 | 63 个错误码 |
| event | 13 | 事件总线抽象（NATS/Channel） |
| appconfig | 10 | 三种配置结构体（App/ACS/Worker） |
| carrier | 10 | 运营商适配器接口 + 三运营商实现 |

### 2.3 基础设施包（被 main/bootstrap 使用）

| 包 | 角色 |
|----|------|
| components/* | PostgreSQL/Redis/NATS/MinIO/Logger 连接器 |
| bootstrap | App 容器 + 初始化编排 |
| middleware | CORS/RequestLogger/Prometheus 中间件 |

---

## 3. 方案设计

### 3.1 核心原则

- **依赖只能向下** — 业务模块可以导入 `core/`，`core/` 绝不导入业务模块
- **最小变更量** — 只移动必要的包，不为了美观做不必要的调整
- **一次到位** — 避免分多次小步移动导致中间状态混乱

### 3.2 重构后的目录结构

```
internal/
├── core/                        # ← 核心���共代码（被所有模块依赖）
│   ├── model/                   #   领域模型、常量、类型定义
│   ├── errors/                  #   错误码定义
│   ├── event/                   #   事件总线抽象
│   ├── appconfig/               #   配置结构体
│   ├── carrier/                 #   运营商适配器接口
│   │   ├── cmcc/                #     中国移动
│   │   ├── ctcc/                #     中国电信
│   │   └── cucc/                #     中国联通
│   ├── components/              #   基础设施连接器
│   │   ├── logger/              #     Zap 日志
│   │   ├── postgres/            #     pgx 连接池
│   │   ├── redis/               #     go-redis 客户端
│   │   ├── nats/                #     NATS JetStream
│   │   ├── minio/               #     MinIO/S3
│   │   └── monitor/             #     Prometheus
│   ├── middleware/              #   HTTP 中间件
│   ├── bootstrap/               #   启动框架
│   └── utils/                   #   工具函数
│
│   # ── 以下为功能模块，按功能域组织，保持原位 ──
│
├── acs/                         # F01 南向接口（TR069 ACS 引擎）
│   ├── auth/
│   ├── cmdqueue/                #   命令队列（多模块共用）
│   ├── connreq/                 #   Connection Request
│   └── rpc/
├── config/                      # F02 数据模型与配置
│   ├── datamodel/               #   TR069 参数树
│   ├── template/                #   配置模板
│   └── baseline/                #   配置基线
├── pm/                          # F03 性能管理
│   ├── collector/               #   PM 文件采集
│   ├── counter/                 #   计数器存储
│   ├── kpi/                     #   KPI 计算
│   └── aggregation/             #   时间聚合
├── alarm/                       # F04 告警管理
├── mr/                          # F05 测量报告
│   ├── collector/
│   └── parser/
├── device/                      # F06 设备管理
├── admin/                       # F06 用户与权限���RBAC）
├── topology/                    # F06 拓扑管理
├── software/                    # F06 固件管理
├── backup/                      # F06 备份恢复
├── dashboard/                   # F06 仪表盘
├── ops/                         # F06 运维工具
├── report/                      # F06 报表
├── mml/                         # F06 MML 控制台
├── filemanager/                 # F06 文件管理
├── syslog/                      # F06 系统日志
├── license/                     # F06 许可管理
├── nedirect/                    # F07 网元直连
├── northbound/                  # F08 北向接口
│   ├── push/
│   └── sync/
├── provision/                   # F09 自动开站
├── interop/                     # F10 互操作测试
│   └── cases/
└── transfer/                    # 数据传输桥接
```

### 3.3 移入 core/ 的包（共 9 个顶层包）

| 原路径 | 新路径 | 理由 |
|--------|--------|------|
| `internal/model` | `internal/core/model` | 被 33 个包导入，纯类型定义 |
| `internal/errors` | `internal/core/errors` | 被 21 个包导入，纯错误码定义 |
| `internal/event` | `internal/core/event` | 被 13 个包导入，事件总线抽象 |
| `internal/appconfig` | `internal/core/appconfig` | 被 10 个包导入，配置结构体 |
| `internal/carrier` | `internal/core/carrier` | 被 10 个包导入，运营商适配接口 |
| `internal/components` | `internal/core/components` | 基础设施连接器，仅 bootstrap 使用 |
| `internal/middleware` | `internal/core/middleware` | HTTP 中间件，仅 router 使用 |
| `internal/bootstrap` | `internal/core/bootstrap` | 启动框架，仅 main.go 使用 |
| `internal/utils` | `internal/core/utils` | 工具函数 |

### 3.4 保持原位的包（共 22 个功能模块）

所有业务功能模块保持在 `internal/` 下的原位置不变：
- acs, config, pm, alarm, mr, device, admin, topology, software, backup, dashboard, ops, report, mml, filemanager, syslog, license, nedirect, northbound, provision, interop, transfer

### 3.5 关于 acs/cmdqueue

`acs/cmdqueue` 被 9 个非 ACS 模块导入，可能有人会建议将它提取到 `core/`。但分析后建议**保持原位**：

- **语义明确** — 命令队列是 TR069 ACS 协议的一部分（SPV/GPV/Download/Reboot 等 RPC 命令排队）
- **导入路径自解释** — `import "internal/acs/cmdqueue"` 明确表达"我要向 ACS 引擎排队命令"
- **不是通用组件** — 它不是通用的消息队列，而是专门为 TR069 RPC 设计的

---

## 4. 导入路径变更

### 4.1 变更清单

```
github.com/omcgo/omcgo/internal/model      → github.com/omcgo/omcgo/internal/core/model
github.com/omcgo/omcgo/internal/errors      → github.com/omcgo/omcgo/internal/core/errors
github.com/omcgo/omcgo/internal/event       → github.com/omcgo/omcgo/internal/core/event
github.com/omcgo/omcgo/internal/appconfig   → github.com/omcgo/omcgo/internal/core/appconfig
github.com/omcgo/omcgo/internal/carrier     → github.com/omcgo/omcgo/internal/core/carrier
github.com/omcgo/omcgo/internal/components  → github.com/omcgo/omcgo/internal/core/components
github.com/omcgo/omcgo/internal/middleware  → github.com/omcgo/omcgo/internal/core/middleware
github.com/omcgo/omcgo/internal/bootstrap   → github.com/omcgo/omcgo/internal/core/bootstrap
github.com/omcgo/omcgo/internal/utils       → github.com/omcgo/omcgo/internal/core/utils
```

### 4.2 影响范围

| 旧导入路径 | 受影响文件数 | 说明 |
|-----------|-------------|------|
| `internal/model` | ~150 | 几乎每个业务文件 |
| `internal/errors` | ~80 | 所有 handler + service |
| `internal/event` | ~40 | 所有需要事件的模块 |
| `internal/appconfig` | ~15 | main.go + 部分服务 |
| `internal/carrier` | ~30 | 设备/PM/告警相关 |
| `internal/components` | ~5 | bootstrap + router |
| `internal/middleware` | ~3 | router.go |
| `internal/bootstrap` | ~3 | 三个 main.go |
| `internal/utils` | ~1 | 极少使用 |

**总计**: 约 200 个文件需要更新导入路径。

### 4.3 自动化替换

所有替换可通过一条 sed 命令完成（纯字符串替换，无歧义）：

```bash
# 一次性替换所有导入路径
find . -name '*.go' -exec sed -i \
  -e 's|"github.com/omcgo/omcgo/internal/model"|"github.com/omcgo/omcgo/internal/core/model"|g' \
  -e 's|"github.com/omcgo/omcgo/internal/errors"|"github.com/omcgo/omcgo/internal/core/errors"|g' \
  -e 's|"github.com/omcgo/omcgo/internal/event"|"github.com/omcgo/omcgo/internal/core/event"|g' \
  -e 's|"github.com/omcgo/omcgo/internal/appconfig"|"github.com/omcgo/omcgo/internal/core/appconfig"|g' \
  -e 's|"github.com/omcgo/omcgo/internal/carrier"|"github.com/omcgo/omcgo/internal/core/carrier"|g' \
  -e 's|"github.com/omcgo/omcgo/internal/components"|"github.com/omcgo/omcgo/internal/core/components"|g' \
  -e 's|"github.com/omcgo/omcgo/internal/middleware"|"github.com/omcgo/omcgo/internal/core/middleware"|g' \
  -e 's|"github.com/omcgo/omcgo/internal/bootstrap"|"github.com/omcgo/omcgo/internal/core/bootstrap"|g' \
  -e 's|"github.com/omcgo/omcgo/internal/utils"|"github.com/omcgo/omcgo/internal/core/utils"|g' \
  {} +
```

替换后运行 `go build ./...` 验证编译通过即可。

---

## 5. 实施步骤

### Step 1: 创建 core/ 目录并移动包

```bash
mkdir -p internal/core

# 移动 9 个包
git mv internal/model internal/core/model
git mv internal/errors internal/core/errors
git mv internal/event internal/core/event
git mv internal/appconfig internal/core/appconfig
git mv internal/carrier internal/core/carrier
git mv internal/components internal/core/components
git mv internal/middleware internal/core/middleware
git mv internal/bootstrap internal/core/bootstrap
git mv internal/utils internal/core/utils
```

### Step 2: 批量替换导入路径

```bash
find . -name '*.go' -exec sed -i \
  -e 's|omcgo/internal/model|omcgo/internal/core/model|g' \
  -e 's|omcgo/internal/errors|omcgo/internal/core/errors|g' \
  -e 's|omcgo/internal/event|omcgo/internal/core/event|g' \
  -e 's|omcgo/internal/appconfig|omcgo/internal/core/appconfig|g' \
  -e 's|omcgo/internal/carrier|omcgo/internal/core/carrier|g' \
  -e 's|omcgo/internal/components|omcgo/internal/core/components|g' \
  -e 's|omcgo/internal/middleware|omcgo/internal/core/middleware|g' \
  -e 's|omcgo/internal/bootstrap|omcgo/internal/core/bootstrap|g' \
  -e 's|omcgo/internal/utils|omcgo/internal/core/utils|g' \
  {} +
```

> **注意**: `internal/carrier` 会匹配 `internal/carrier/cmcc` 等子路径，sed 替换 `internal/carrier` → `internal/core/carrier` 会自动正确处理 `internal/carrier/cmcc` → `internal/core/carrier/cmcc`。

### Step 3: 编译验证

```bash
go build ./...
go test ./...
```

### Step 4: 提交

一次性提交，commit message：`refactor: 分离核心公共代码到 internal/core/`

---

## 6. 重构前后对比

### 目录对比

| 变更前 | 变更后 |
|--------|--------|
| 32 个包全部扁平 | core/ 下 9 个包 + 22 个业务模块 |
| 无法区分基础设施和业务代码 | 一眼可见 core/ = 公共基础，其余 = 业务模块 |
| 新人需要读代码才能理解层级 | 目录结构即文档 |

### 导入对比

```go
// 变更前 — 无法从导入看出层级
import (
    "github.com/omcgo/omcgo/internal/model"      // 基础类型
    "github.com/omcgo/omcgo/internal/errors"      // 基础错误
    "github.com/omcgo/omcgo/internal/alarm"       // 业务模块
    "github.com/omcgo/omcgo/internal/device"      // 业务模块
)

// 变更后 — 层级一目了然
import (
    "github.com/omcgo/omcgo/internal/core/model"   // ← 核心层
    "github.com/omcgo/omcgo/internal/core/errors"   // ← 核心层
    "github.com/omcgo/omcgo/internal/alarm"          // 业务层
    "github.com/omcgo/omcgo/internal/device"         // 业务层
)
```

### 依赖规则

```
┌──────────────────────────────────────────┐
│           cmd/ (main.go 入口)              │
│     只导入 core/bootstrap + 业务模块        │
└──────────┬──────────┬────────────────────┘
           │          │
           ▼          ▼
┌──────────────┐ ┌──────────────────────────┐
│  core/       │ │  业务模块                  │
│  model       │ │  device, alarm, pm, ...   │
│  errors      │ │                           │
│  event       │◄├──── 业务模块可导入 core/    │
│  carrier     │ │     core/ 绝不导入业务模块  │
│  appconfig   │ │                           │
│  components  │ │  业务模块间可按需相互导入   │
│  middleware   │ │  （需保持无循环）           │
│  bootstrap   │ │                           │
└──────────────┘ └──────────────────────────┘
```

---

## 7. 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 大量文件导入路径变更 | ~200 个文件需修改 | sed 自动替换，`go build` 一次验证 |
| git 历史断裂 | `git log --follow` 可追踪 `git mv` | 使用 `git mv` 而非删除+新建 |
| 外部引用（如果有） | internal/ 不可外部引用 | 无风险，internal/ 是 Go 内部包 |
| 合并冲突 | 其他分支的 import 路径不同 | 一次性实施，及时合并，冲突用 sed 解决 |
| ���漏替换 | 编译失败 | `go build ./...` 会立即发现 |

**风险等级: 低** — 这是一个纯机械的目录移动 + 字符串替换操作，`go build` 可 100% 验证正确性。

---

## 8. 不做的事情

以下调整**不在本次范围内**，避免过度工程化：

1. **不对业务模块再分组** — 不创建 `internal/omcr/` 或 `internal/f06/`，22 个业务模块扁平排列已经足够清晰
2. **不移动 acs/cmdqueue** — 虽然被多模块依赖，但语义上属于 ACS 南向接口
3. **不重命名包** — 不改 `model` 为 `domain`，不改 `errors` 为 `apperrors`
4. **不拆分大包** — 不将 `admin`(3473行) 或 `device`(3024行) 拆为更小的子包
5. **不引入接口层** — 不创建 `internal/core/repository/` 等抽象层

这些可以作为后续优化，但当前的核心/业务分离是最高优先级、最低风险的改进。
