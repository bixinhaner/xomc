# OMC Go 后端目录调整方案

> 基于 2026-03-11 会议讨论方向，对 `omcgo/` 工程进行分层调整评估
> 当前代码规模：**300 个 Go 文件**（含 74 个测试文件），**13 个领域模块**，**33 个数据库迁移**

---

## 一、变更总览

| # | 变更项 | 当前位置 | 目标位置 | 难度 | 影响文件数 |
|---|--------|----------|----------|------|-----------|
| C1 | 配置文件按环境隔离 | `configs/*.yaml` | `cmd/app/etc/config.{dev,test,prod}.yaml` | **S** | ~5 |
| C2 | 配置结构体独立 | `internal/config/config.go` | `cmd/app/config/` | **M** | ~14 |
| C3 | 路由注册提取 | `cmd/app/main.go` 内联 | `cmd/app/router/` | **S** | ~3 |
| C4 | Handler 层提取 | `internal/*/handler.go` (22个) | `cmd/app/handler/{module}/` | **XL** | ~120+ |
| C5 | Service 层提取 | `internal/*/service.go` (13个) | `cmd/app/service/{module}/` | **XL** | ~100+ |
| C6 | Model 合并扁平化 | 各模块 `model.go` (22个) | `internal/model/` | **XL** | ~185+ |
| C7 | Repository 合并扁平化 | 各模块 `*repository.go` (51个) | `internal/repository/` | **XL** | ~100+ |
| C8 | Middleware 提升 | `internal/common/middleware/` | `internal/middleware/` | **S** | ~3 |
| C9 | Infra 重命名 → Components | `internal/infra/` | `internal/components/` | **M** | ~15 |
| C10 | 新增 `global/` | 不存在 | `global/{vars,consts,errors}.go` | **L** | ~185+ |
| C11 | utils 目录 | 零散/不存在 | `internal/utils/` | **S** | ~2 |

---

## 二、逐项详细分析

### C1：配置文件按环境隔离 — 难度 S

**变更内容**
```
configs/app.yaml          → cmd/app/etc/config.dev.yaml
                          → cmd/app/etc/config.test.yaml
                          → cmd/app/etc/config.prod.yaml
configs/acs.yaml          → cmd/acs/etc/config.dev.yaml   (若一致)
configs/worker.yaml       → cmd/worker/etc/config.dev.yaml (若一致)
```

**需修改**
| 文件 | 修改内容 |
|------|---------|
| `cmd/app/main.go` | 默认 config 路径从 `configs/app.yaml` → `cmd/app/etc/config.dev.yaml` |
| `cmd/acs/main.go` | 同上 |
| `cmd/worker/main.go` | 同上 |
| `Makefile` | 构建/部署路径 |
| `deployments/` | Docker COPY 路径 |

**风险**：低。纯文件移动 + 路径字符串替换。
**前端影响**：无。
**建议**：✅ **可立即执行**。

---

### C2：配置结构体独立 — 难度 M

**当前问题**
`internal/config/` 是一个混合包，既包含：
- 配置加载逻辑：`config.go`（AppConfig, ACSConfig 结构体 + viper Load）
- F02 业务域：`datamodel/`、`template/`、`baseline/`、`sync_handler.go`

**变更内容**
```
internal/config/config.go   → cmd/app/config/config.go    (包名 config)
internal/config/config.go   中 Load() 函数也移过去
```
F02 业务子包 `datamodel/`, `template/`, `baseline/` 保留在 `internal/config/` 不动。

**需修改的导入** （14 个文件）
| 文件 | 当前导入 | 新导入 |
|------|---------|--------|
| `cmd/app/main.go` | `internal/config` | `cmd/app/config` |
| `cmd/acs/main.go` | `internal/config` | 需决策：共享还是各自一份？ |
| `cmd/worker/main.go` | `internal/config` | 同上 |

**⚠️ 关键决策点**：`ACSConfig` 被 `cmd/acs/` 使用，`WorkerConfig` 被 `cmd/worker/` 使用。如果放到 `cmd/app/config/`，其他二进制如何引用？

**方案选择**：
- **A**：配置结构体放 `internal/config/`（保持共享），只移配置加载逻辑 → 改动最小
- **B**：配置结构体放 `cmd/app/config/`，acs/worker 各自定义自己的 Config → 代码重复
- **C**：配置结构体放 `pkg/config/` 作为公共包 → 最干净但 pkg 不应依赖 internal

**建议**：方案 A — 仅重命名包路径，保留在 `internal/` 下共享。

---

### C3：路由注册提取 — 难度 S

**变更内容**
从 `main.go`（502行）中提取路由注册到 `cmd/app/router/router.go`：
```go
// cmd/app/router/router.go
package router

func Setup(r *gin.Engine, deps *Dependencies) {
    v1 := r.Group("/api/v1")
    // ... 所有 RegisterRoutes 调用
}
```

**需修改**
| 文件 | 修改内容 |
|------|---------|
| `cmd/app/main.go` | 拆出路由注册部分（约 200 行） |
| `cmd/app/router/router.go` | 新建 |
| `cmd/app/router/deps.go` | 新建（依赖注入容器） |

**风险**：低。纯重构，不改变运行时行为。
**前端影响**：无（路由路径不变）。
**建议**：✅ **可立即执行**，大幅改善 main.go 可读性。

---

### C4：Handler 层提取到 cmd/app/handler/ — 难度 XL ⚠️

**当前模式（每模块自包含）**
```go
// internal/alarm/handler.go
package alarm

type Handler struct {
    engine *AlarmEngine    // ← 未导出类型，同包内私有访问
    store  AlarmStore      // ← 接口，同包定义
    logger *zap.Logger
}
```

**提取后问题**
```go
// cmd/app/handler/alarm/handler.go
package alarm

import "github.com/omcgo/omcgo/internal/alarm"

type Handler struct {
    engine *alarm.AlarmEngine  // ❌ 如果 AlarmEngine 未导出则编译失败
    store  alarm.AlarmStore
}
```

**影响范围量化**
| 指标 | 数量 |
|------|------|
| 需移动的 handler 文件 | 22 |
| 需移动的 handler 测试文件 | ~15 |
| 需导出的未导出类型 | 估计 30+ 个 struct/interface |
| 需修改导入路径的文件 | 47 个（含 RegisterRoutes） |
| 涉及重命名 `Handler` struct | 22 个同名冲突 |

**技术难点**

1. **22 个 `type Handler struct` 同名冲突**：
   - 若按 `cmd/app/handler/alarm/`、`cmd/app/handler/device/` 子包方式组织 → 需 22 个子包
   - 若放平 `cmd/app/handler/` 一个包 → 必须重命名为 `AlarmHandler`、`DeviceHandler`... 涉及所有方法签名

2. **私有类型暴露**：Handler 目前直接操作同包的 Engine/Store/Service，很多是未导出的。移出后必须全部导出，**破坏封装性**。

3. **请求/响应 DTO 归属**：每个 handler.go 内定义了请求绑定 struct（如 `alarmQuery`），这些算 handler 层还是 model 层？

**建议**：⚠️ **需深入讨论**。当前 Go 社区主流做法是按业务域分包（handler+service+repo 共存），而非按技术层分包。如果一定要拆，建议采用子包方式：
```
cmd/app/handler/
├── alarm/handler.go
├── device/handler.go
├── admin/handler.go
└── ...
```

---

### C5：Service 层提取到 cmd/app/service/ — 难度 XL ⚠️

**与 C4 相同的问题**，且更严重：

| 指标 | 数量 |
|------|------|
| service.go 文件 | 13 |
| 需同步移动的 engine/相关逻辑 | alarm/engine.go, provision/engine.go, kpi/engine.go 等 |
| 8 个 `type Service struct` 同名冲突 | 需重命名 |

**额外复杂度**
- Service 层通常还包含 EventBus 订阅（`Subscribe(eventBus)`）、后台 goroutine（HeartbeatMonitor）
- Service 依赖 Repository 接口 → 如果 Repository 也被移到 `internal/repository/`，则形成交叉依赖

**风险**：高。Service 是业务逻辑核心，拆分不当会导致循环依赖。
**建议**：⚠️ 与 C4 一起决策。

---

### C6：Model 合并到 internal/model/ — 难度 XL ⚠️

**当前分布**
```
internal/common/model/     — 6 个文件（共享类型：Device, Pagination, Constants）
internal/alarm/model.go    — 24 个类型定义
internal/pm/               — 52 个类型定义
internal/mr/               — 34 个类型定义
internal/omcr/admin/       — 34 个类型定义
... 共计 16 个模块，439 个类型定义
```

**命名冲突统计**
| 类型名 | 出现次数 | 冲突模块 |
|--------|---------|---------|
| `Filter` 类 | 39 | alarm, device, pm, mr, ops, backup... |
| `Request` 类 | 63 | 几乎所有模块 |
| `Status` 类 | 17 | alarm, device, provision, software... |
| `Task` 类 | 7 | pm, backup, mml, ops, provision |
| `TaskStatus` | 5 | 同上 |

**影响范围**
| 指标 | 数量 |
|------|------|
| 导入 `internal/common/model` 的文件 | **185** |
| 需移动的 model 文件 | 22+ |
| 需解决的命名冲突 | 50+ 类型需加前缀 |

**解决冲突的方式**
- 所有类型加模块前缀：`AlarmFilter`、`DeviceFilter`、`PMTask`... → **大量代码改动**
- 或采用子包：`internal/model/alarm/`、`internal/model/device/` → 但这又回到了当前结构

**风险**：极高。185 个文件的导入路径变更 + 50+ 类型重命名 = 回归测试噩梦。
**建议**：⚠️ **不建议合并为单包**。如果需要统一入口，可考虑在 `internal/model/` 下保持子包结构。

---

### C7：Repository 合并到 internal/repository/ — 难度 XL ⚠️

**当前分布**
```
51 个 repository 文件分散在各模块中
每个模块 2-4 个 repository 文件（interface + pg 实现）
```

**同样面临命名冲突**
- 22 个 `NewPg*Repository` 构造函数
- Repository 接口大量同名方法（`List`, `GetByID`, `Create`, `Update`, `Delete`）

**额外问题**
- Repository 接口定义引用对应模块的 Model 类型 → 如果 Model 移到 `internal/model/`，Repository 必须也改导入
- C6 + C7 形成连锁：model 移动 → repository 移动 → service 移动 → handler 移动

**建议**：⚠️ 与 C6 一起决策。若 model 不扁平化，repository 也不应扁平化。

---

### C8：Middleware 提升 — 难度 S

**变更内容**
```
internal/common/middleware/ → internal/middleware/
```

**需修改**
| 文件 | 修改内容 |
|------|---------|
| `cmd/app/main.go` | 导入路径 |
| `cmd/acs/main.go` | 若有使用 |
| 中间件测试文件 | 包声明 |

仅 **3 个文件** 导入了 `internal/common/middleware`。

**风险**：极低。
**建议**：✅ **可立即执行**。

---

### C9：Infra 重命名为 Components — 难度 M

**变更内容**
```
internal/infra/              → internal/components/
internal/infra/db/           → internal/components/db/（或提升为 postgres/）
internal/infra/cache/        → internal/components/cache/（或提升为 redis/）
internal/infra/mq/           → internal/components/nats/
internal/infra/storage/      → internal/components/minio/
internal/infra/logger.go     → internal/components/logger/logger.go
internal/infra/metrics.go    → internal/components/monitor/metrics.go
internal/infra/health.go     → 位置待定
internal/infra/sysinfo.go    → 位置待定
internal/infra/tracer.go     → 位置待定
internal/infra/shutdown.go   → 位置待定
```

**需修改**
| 指标 | 数量 |
|------|------|
| 导入 `internal/infra` 的文件 | 3（main.go × 3） |
| 导入 `internal/infra/db` 的文件 | ~5 |
| 导入 `internal/infra/cache` 的文件 | ~3 |
| 导入 `internal/infra/mq` 的文件 | ~3 |
| 导入 `internal/infra/storage` 的文件 | ~2 |
| **合计** | **~15** |

**风险**：低。纯 rename + import 路径替换。
**注意**：`infra` 下的 `health.go`, `sysinfo.go`, `tracer.go`, `shutdown.go` 不是"组件"，需要决定去向。

**建议**：✅ 可执行，但需先明确非组件文件的归属。

---

### C10：新增 global/ — 难度 L ⚠️

**提案内容**
```go
// global/vars.go
var (
    DB    *pgxpool.Pool
    Redis *redis.Client
    NATS  *nats.Conn
    Config *config.AppConfig
)

// global/consts.go — 从 internal/common/model/constants.go 迁移
// global/errors.go — 从 internal/common/errors/codes.go 迁移
```

**⚠️ 重大架构决策**

当前代码采用**依赖注入模式**（通过构造函数传递依赖）：
```go
// 当前模式（好的实践）
func NewPgDeviceRepository(pool *pgxpool.Pool) *PgDeviceRepository {
    return &PgDeviceRepository{pool: pool}
}
```

改为全局变量后：
```go
// global 模式（有争议）
func NewPgDeviceRepository() *PgDeviceRepository {
    return &PgDeviceRepository{pool: global.DB}
}
```

**对比分析**

| 维度 | 当前 DI 模式 | global 模式 |
|------|------------|------------|
| 可测试性 | ✅ 注入 mock pool | ❌ 需要设置全局变量 |
| 并发安全 | ✅ 无共享可变状态 | ⚠️ 需确保初始化顺序 |
| 依赖清晰度 | ✅ 构造函数签名即文档 | ❌ 隐式依赖 |
| 多实例支持 | ✅ 可创建多个 pool | ❌ 全局唯一 |
| 代码简洁度 | ⚠️ 构造函数参数多 | ✅ 不用传递 |

**影响范围**
| 指标 | 数量 |
|------|------|
| 需改为引用 global 的 repository 构造函数 | 51 |
| 需改为引用 global 的测试文件 | 74 |
| 导入 `internal/common/model` 待迁移的文件 | 185 |
| 导入 `internal/common/errors` 待迁移的文件 | 62 |

**`errors.go` 迁移的额外障碍**：
`internal/common/errors/errors.go` 使用了 `gin.Context`（`AbortWithError` 方法），不能简单移到 `global/`（global 不应依赖 HTTP 框架）。需要拆分为：
- `global/errors.go` — 纯错误码定义
- `internal/common/errors/gin_errors.go` — gin 相关的错误响应

**建议**：
- `global/consts.go` ✅ 可以做（从 `common/model/constants.go` 迁移纯常量）
- `global/errors.go` ⚠️ 需拆分，有一定工作量
- `global/vars.go` ❌ **强烈不建议** — 放弃 DI 模式是架构倒退

---

### C11：utils 目录 — 难度 S

**变更内容**
```
新建 internal/utils/
迁移各处零散的 helper 函数（如有）
```

**风险**：极低。
**建议**：✅ 需要时自然创建即可。

---

## 三、工作量综合评估

### 按难度分级

| 难度 | 变更项 | 预估改动文件 | 可独立执行 |
|------|--------|-------------|-----------|
| **S（半天内）** | C1 配置文件, C3 路由提取, C8 中间件, C11 utils | ~13 | ✅ |
| **M（1-2天）** | C2 配置结构体, C9 infra→components | ~29 | ✅ |
| **L（3-5天）** | C10 global/ | ~185+ | 部分可 |
| **XL（5-10天/每项）** | C4 handler, C5 service, C6 model, C7 repo | ~全量 | ❌ 强耦合 |

### 依赖关系图

```
C1（配置文件）──独立
C2（配置结构体）──独立
C3（路由提取）──独立
C8（中间件）──独立
C9（infra改名）──独立
C11（utils）──独立

C6（model扁平化）──→ C7（repo扁平化）──→ C5（service提取）──→ C4（handler提取）
        ↑
C10（global/errors,consts 部分）
```

**C4-C7 形成依赖链条**：必须按 C6→C7→C5→C4 顺序执行，或作为一个整体一次性完成。

### 总工作量估算

| 方案 | 范围 | 预估工时 | 回归测试工时 |
|------|------|---------|-------------|
| **最小方案**（C1+C3+C8+C11） | 仅安全变更 | 1 人日 | 0.5 人日 |
| **推荐方案**（+C2+C9+C10 部分） | 中等调整 | 3-5 人日 | 1-2 人日 |
| **完整方案**（含 C4-C7） | 全量重构 | 15-25 人日 | 5-8 人日 |

---

## 四、前端影响分析

### API 路由层面 — **不受影响** ✅

| 维度 | 说明 |
|------|------|
| REST 路由路径 | `/api/v1/devices`, `/api/v1/alarms` 等由 `RegisterRoutes()` 定义，无论 handler 文件在哪个目录，路由路径不变 |
| 请求/响应格式 | JSON 结构由 model struct 的 `json` tag 决定，只要 struct 定义不变，前端无感 |
| HTTP 状态码 | 由 handler 逻辑决定，与目录无关 |
| 错误码 | 数值定义在 `codes.go`，只要数值不变即可 |

### 唯一风险场景
- 若 C6（model 扁平化）过程中**误改 JSON tag** 或**遗漏字段**，会导致前端解析失败
- 缓解措施：执行 331 条 E2E 测试（`scripts/e2e_verify.sh`）可完整覆盖

**结论：目录调整不影响前端，但 model 合并需格外谨慎。**

---

## 五、后续影响

### 5.1 对开发流程的影响

| 影响 | 说明 |
|------|------|
| **Git 历史断裂** | 文件移动后 `git log --follow` 才能追溯历史，review 不便 |
| **分支冲突** | 重构期间任何并行开发分支都会产生大量 merge conflict |
| **IDE 重构** | GoLand/VSCode 的 package rename 可自动更新导入路径，但需逐一验证 |
| **CI/CD** | Makefile、Dockerfile 中硬编码的路径需同步更新 |
| **文档** | CLAUDE.md（441行）、doc/ 目录（20+文件）中的路径引用全部过期 |

### 5.2 对多二进制的影响

`cmd/acs/` 和 `cmd/worker/` 也从 `internal/` 导入：

| 二进制 | 当前导入 | 受影响的变更 |
|--------|---------|------------|
| `cmd/acs/main.go` | `internal/config`, `internal/infra/*`, `internal/acs/*` | C2, C9 |
| `cmd/worker/main.go` | `internal/config`, `internal/infra/*`, `internal/pm/*`, `internal/mr/*`, `internal/alarm/*` | C2, C6, C7, C9 |

### 5.3 对测试的影响

| 测试类型 | 影响 |
|----------|------|
| 单元测试（74 文件） | package 移动后 `package xxx_test` 声明需全部更新 |
| E2E 测试（331 条） | 走 HTTP → 不受影响（路由不变） |
| 集成测试 | 导入路径需更新 |

---

## 六、推荐方案：分阶段实施

### 第一阶段：安全重构（1-2 天）✅

**立即可做，零风险**

```
C1 — 配置文件环境隔离
     configs/*.yaml → cmd/app/etc/config.{dev,test,prod}.yaml

C3 — 路由注册提取
     main.go 中 ~200 行路由代码 → cmd/app/router/router.go

C8 — Middleware 提升
     internal/common/middleware/ → internal/middleware/

C11 — 创建 utils 骨架
     internal/utils/（按需）
```

### 第二阶段：基础设施调整（2-3 天）✅

```
C9 — Infra → Components 重命名
     internal/infra/ → internal/components/
     sub-packages: db/, cache/ → postgres/, redis/; mq/ → nats/; storage/ → minio/
     新增: logger/, monitor/

C2 — 配置结构体分离
     从 internal/config/ 中拆出纯配置结构体
     保持 F02 业务域（datamodel/template/baseline）不动

C10 — global/ 部分
     global/consts.go ← internal/common/model/constants.go 中的纯常量
     global/errors.go ← internal/common/errors/codes.go 中的纯错误码
     ❌ 不引入 global/vars.go（保持 DI）
```

### 第三阶段：领域模块重组（需团队讨论后决策）⚠️

**两个方向**：

#### 方向 A — 按技术层扁平化（提案原方案）
```
cmd/app/handler/alarm/      ← internal/alarm/handler.go
cmd/app/handler/device/     ← internal/omcr/device/handler.go
cmd/app/service/alarm/      ← internal/alarm/engine.go
internal/model/alarm/       ← internal/alarm/model.go
internal/repository/alarm/  ← internal/alarm/pg_store.go
```
- 工作量：**15-25 人日**
- 优点：层次分明，职责清晰
- 缺点：打散业务内聚，Go 社区非主流，循环依赖风险大

#### 方向 B — 保持领域自包含，仅消除 omcr 大包（推荐 ✅）
```
internal/
├── alarm/            ← 保持不变
├── device/           ← 从 internal/omcr/device/ 提升
├── admin/            ← 从 internal/omcr/admin/ 提升
├── topology/         ← 从 internal/omcr/topology/ 提升
├── software/         ← 从 internal/omcr/software/ 提升
├── backup/           ← 从 internal/omcr/backup/ 提升
├── dashboard/        ← 从 internal/omcr/dashboard/ 提升
├── ops/              ← 从 internal/omcr/ops/ 提升
├── report/           ← 从 internal/omcr/report/ 提升
├── mml/              ← 从 internal/omcr/mml/ 提升
├── filemanager/      ← 从 internal/omcr/filemanager/ 提升
├── syslog/           ← 从 internal/omcr/syslog/ 提升
├── license/          ← 从 internal/omcr/license/ 提升
├── pm/               ← 保持不变
├── mr/               ← 保持不变
├── config/           ← 保持不变（F02 业务域）
├── provision/        ← 保持不变
├── acs/              ← 保持不变
├── carrier/          ← 保持不变
├── northbound/       ← 保持不变
├── interop/          ← 保持不变
├── nedirect/         ← 保持不变
├── components/       ← 从 infra/ 改名
├── middleware/       ← 从 common/middleware/ 提升
├── model/            ← 从 common/model/ 改名（仅共享类型）
├── errors/           ← 从 common/errors/ 提升
├── event/            ← 从 common/event/ 提升
├── response/         ← 从 common/response/ 提升
└── utils/            ← 新建
```
- 工作量：**5-8 人日**
- 优点：消除 `omcr` 和 `common` 两个大杂烩包，模块扁平化但保持自包含
- 缺点：仍然不是技术层分层

---

## 七、方向 B 详细目标结构（推荐）

```
omcgo/
├── cmd/
│   ├── app/
│   │   ├── main.go            # 精简后 ~150 行（调用 router.Setup）
│   │   ├── etc/               # 配置文件 (dev/test/prod)
│   │   ├── config/            # AppConfig 结构体 + Load()
│   │   └── router/            # 路由注册 + 依赖注入容器
│   ├── acs/
│   ├── worker/
│   ├── migrate/
│   └── omcctl/
│
├── internal/
│   ├── device/                # ← omcr/device（handler+service+repo+model 自包含）
│   ├── admin/                 # ← omcr/admin
│   ├── topology/              # ← omcr/topology
│   ├── software/              # ← omcr/software
│   ├── backup/                # ← omcr/backup
│   ├── dashboard/             # ← omcr/dashboard
│   ├── ops/                   # ← omcr/ops
│   ├── report/                # ← omcr/report
│   ├── mml/                   # ← omcr/mml
│   ├── filemanager/           # ← omcr/filemanager
│   ├── syslog/                # ← omcr/syslog
│   ├── license/               # ← omcr/license
│   ├── alarm/                 # 保持
│   ├── pm/                    # 保持
│   ├── mr/                    # 保持
│   ├── config/                # 保持 (datamodel/template/baseline)
│   ├── provision/             # 保持
│   ├── acs/                   # 保持
│   ├── carrier/               # 保持
│   ├── northbound/            # 保持
│   ├── interop/               # 保持
│   ├── nedirect/              # 保持
│   ├── model/                 # ← common/model（仅共享类型）
│   ├── errors/                # ← common/errors
│   ├── event/                 # ← common/event
│   ├── response/              # ← common/response
│   ├── middleware/             # ← common/middleware
│   ├── components/            # ← infra/
│   │   ├── postgres/          # ← infra/db
│   │   ├── redis/             # ← infra/cache
│   │   ├── nats/              # ← infra/mq
│   │   ├── minio/             # ← infra/storage
│   │   ├── logger/            # ← infra/logger.go
│   │   └── monitor/           # ← infra/metrics.go
│   └── utils/
│
├── global/
│   ├── consts.go              # 全局常量（CarrierCode 等）
│   └── errors.go              # 错误码定义（纯值，不含 gin 依赖）
│
├── pkg/                       # 保持不变
│   ├── soap/
│   ├── tr069/
│   └── xmlutil/（或 algorithms/）
│
├── migrations/                # 保持不变
├── configs/                   # 废弃（移到 cmd/app/etc/）
├── deployments/               # 保持不变
├── doc/                       # 路径引用更新
├── scripts/                   # 保持不变
├── test/                      # 保持不变
└── Makefile                   # 路径更新
```

---

## 八、文档调整清单

| 文档 | 需更新内容 |
|------|-----------|
| `CLAUDE.md`（441行） | 模块路径、目录结构描述、启动命令 |
| `doc/` 目录（20+文件） | 涉及模块路径的引用 |
| `Makefile` | build/run/test 命令中的路径 |
| `deployments/Dockerfile` | COPY 配置文件路径 |
| `deployments/k8s/` | ConfigMap 路径 |
| `.golangci.yml` | 如有路径排除规则 |
| `scripts/e2e_verify.sh` | 如有路径引用 |
| 根仓 `前后端整合方案.md` | 后端目录结构描述 |
| auto-memory `MEMORY.md` | Key File Paths 部分 |
| `README.md`（如有） | 快速上手路径 |

---

## 九、风险矩阵

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| model 合并导致命名冲突编译失败 | 高 | 高 | 不采用单包合并（方向 B） |
| 并行开发分支冲突 | 高 | 中 | 冻结主线 2-3 天，集中做重构 |
| global 变量导致测试隔离破坏 | 中 | 高 | 不引入 global/vars.go |
| 遗漏导入路径更新导致编译失败 | 中 | 低 | `go build ./...` 编译验证 |
| E2E 测试覆盖遗漏 | 低 | 高 | 重构后跑 331 条 E2E 全量 |
| 运维脚本/CI 路径失效 | 中 | 中 | grep 全局搜索旧路径 |

---

## 十、执行检查清单

每个阶段完成后执行：

```bash
# 1. 编译检查
go build ./...

# 2. 单元测试
go test ./... -count=1

# 3. lint 检查
golangci-lint run

# 4. E2E 测试（需启动服务）
bash ./scripts/e2e_verify.sh http://localhost:8080

# 5. 搜索残留旧路径
grep -rn 'internal/omcr/' --include="*.go" .
grep -rn 'internal/common/' --include="*.go" .
grep -rn 'internal/infra/' --include="*.go" .
grep -rn 'configs/' --include="*.go" .
```
