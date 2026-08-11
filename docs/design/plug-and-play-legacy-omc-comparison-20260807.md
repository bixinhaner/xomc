# 即插即用：老 OMC 对比与现网补充说明

> **日期**：2026-08-07
> **状态**：已实施并完成真实浏览器验证
> **影响范围**：设备管理 / 即插即用、Provisioning Task 查询、策略产品匹配
> **参考系统**：老 OMC gNB Plug-and-play 页面
> **非本次范围**：`gNB Length` 参数注册、映射及导入问题

---

## 1. 背景

当前 OMC 已具备即插即用策略配置、设备选择和策略执行能力，但执行任务区域的信息密度和运维检索能力不足。为确认缺失项，本次实际访问老 OMC 的 gNB 即插即用页面，对其信息架构、列表字段、筛选方式和操作入口进行了对照。

本次工作的目标不是复刻老系统，而是：

1. 保留当前 OMC 的技术架构、视觉体系和任务状态机；
2. 借鉴老 OMC 中已经被运维使用验证过的信息组织方式；
3. 补齐当前 OMC 在任务可观测性和全量检索方面的缺口；
4. 不引入与现有执行模型冲突的旧式操作。

## 2. 调研范围与参考入口

### 2.1 老 OMC 访问路径

- 登录入口：`http://172.21.172.189:8081/sys/login/userLoad.htm`
- 页面路径：选择 gNB 后进入 `Advance -> Plug-and-play`
- 老页面源码：
  - `omcmb/original-omc/OMCWebServer/src/main/webapp/WEB-INF/content/cell/son/selfConfiguration/gnb_plug_and_play.jsp`
  - `omcmb/original-omc/OMCWebServer/src/main/webapp/WEB-INF/content/cell/son/selfConfiguration/gnb_add_config.jsp`

### 2.2 当前 OMC 入口

- 页面：`/device/plug-and-play`
- 前端页面：`omcmb/webcode/src/pages/device/PlugAndPlay/index.tsx`
- 任务接口：`GET /api/v1/provisioning/tasks`

### 2.3 调研方法

- 使用可见 Chrome 实际访问老 OMC；
- 切换到 gNB 即插即用页面并检查表格、筛选、页签和操作列；
- 对照老 JSP 页面确认字段语义；
- 再访问本地当前 OMC，对比实际渲染结果和网络请求；
- 完成改动后重新构建、热部署并进行浏览器回归。

## 3. 老 OMC 的整体设计

老 OMC 页面整体由两个主要区域组成。

### 3.1 策略列表

策略列表提供以下主要信息：

| 字段 | 作用 |
|---|---|
| 是否启用 | 控制策略是否参与即插即用执行 |
| 产品类型 | 标识策略适用的设备产品 |
| 策略名称 | 识别策略 |
| 执行方式 | 区分自动执行和手动执行 |
| 软件升级 | 显示是否启用以及目标版本 |
| License | 显示是否启用 License 模块 |
| 参数配置 | 显示是否启用参数自配置模块 |

策略和任务分区展示，使用户可以先确认“采用什么策略”，再查看“执行结果如何”。当前 OMC 原有的双卡片结构与这一思路一致，因此本次继续保留。

### 3.2 执行任务区

老 OMC 任务区的核心特点如下：

- 顶部显示全局成功数、失败数；
- 按模块划分任务页签：全部任务、软件升级、License、参数配置；
- 支持策略名称/设备序列号、时间范围、产品、状态组合筛选；
- 任务表中直接展示设备、产品、策略、执行方式、起止时间和进度；
- 不同模块页签可以展示模块特有信息，例如初始/目标版本或 License 文件；
- 任务操作包含重试、手动启动、删除和详情。

### 3.3 设备检测与批量选择

老 OMC 使用左右双列表组织待选设备和已选设备，并提供批量输入/导入入口。当前 OMC 的设备检测弹窗已经具备相近结构，因此这部分不作为主要缺口重复改造。

## 4. 改造前的主要差距

| 能力 | 老 OMC | 当前 OMC 改造前 | 判断 |
|---|---|---|---|
| 策略与任务双区展示 | 有 | 有 | 已具备，保留 |
| 任务模块页签 | 有 | 无 | 高价值，补充 |
| 策略名称/设备编码检索 | 有 | 仅设备编码 | 高价值，补充 |
| 时间范围筛选 | 有 | 无 | 高价值，补充 |
| 产品筛选 | 有 | 无 | 高价值，补充 |
| 状态筛选 | 有 | 有，但“执行中”口径不完整 | 调整为服务端统一口径 |
| 产品/策略/执行方式列 | 有 | 无 | 高价值，补充 |
| 成功/失败统计 | 全量统计 | 仅能基于当前页数据推导 | 修正为服务端全量统计 |
| 左右设备选择器 | 有 | 已有 | 不重复改造 |
| 任务删除 | 有 | 无 | 不照搬 |
| 待执行任务手动启动 | 有 | 当前任务模型不同 | 不照搬 |

## 5. 本次设计决策

### 5.1 借鉴的部分

本次只借鉴老 OMC 中与运维可观测性直接相关的能力：

1. **模块视图**：帮助用户快速判断问题发生在升级、License 还是参数配置阶段；
2. **组合筛选**：筛选必须作用于服务端全量数据，而不是仅过滤当前页；
3. **任务上下文**：任务行必须能回答“哪台设备、哪个产品、使用哪个策略、如何触发”；
4. **全局统计**：成功/失败数量必须与当前筛选条件一致，同时不受分页影响。

### 5.2 明确不照搬的部分

以下老 OMC 能力没有直接复制：

- **删除任务**：当前任务记录承担审计和状态追踪职责，删除会破坏可追溯性；
- **手动启动待执行任务**：当前 OMC 的手动策略通过设备选择和执行入口创建任务，不存在与老 OMC 完全一致的待执行队列；
- **旧 JSP 布局和交互组件**：继续使用当前 React、Ant Design、`DataTable` 和统一暗色/亮色主题；
- **仅前端过滤**：数据分页后再过滤会产生统计和结果不完整问题，因此新增筛选统一落到后端。

## 6. 已实施改动

### 6.1 前端页面

执行状态区域新增：

- `所有任务 / 软件升级 / License / 参数自配置` 模块切换；
- `策略名称 / 设备编码` 关键字搜索；
- 开始、结束时间范围；
- 产品名称筛选；
- 状态筛选；
- 产品名称、策略名称、执行方式三列；
- 基于接口 `status_counts` 的全量成功/失败统计。

筛选、模块切换或分页变化后，前端将条件直接传递给任务接口。页面不再依赖当前页数据执行二次任务过滤。

主要文件：

- `omcmb/webcode/src/pages/device/PlugAndPlay/index.tsx`
- `omcmb/frontend-core/src/services/api/provisionApi.ts`
- `omcmb/frontend-core/src/hooks/api/useProvisioning.ts`

### 6.2 后端接口

`GET /api/v1/provisioning/tasks` 新增以下查询参数：

| 参数 | 说明 |
|---|---|
| `search` | 模糊匹配设备序列号或策略名称 |
| `product_name` | 按产品名称精确筛选，大小写不敏感 |
| `module` | `software_upgrade`、`license` 或 `self_config` |
| `started_after` | RFC3339 格式的开始时间下限 |
| `started_before` | RFC3339 格式的开始时间上限 |
| `status=running` | 返回所有非完成、非失败的运行中状态 |

接口响应增加：

```json
{
  "items": [],
  "total": 0,
  "status_counts": {
    "completed": 0,
    "failed": 0
  }
}
```

`status_counts` 使用与列表相同的搜索、产品、模块和时间条件，但不受分页和单一状态筛选影响，用于呈现当前查询范围内的全量结果分布。

### 6.3 任务上下文查询

任务列表查询关联：

- `devices`：设备序列号、设备产品关联；
- `plug_and_play_policies`：策略名称、执行方式和旧产品字段回退；
- `products`：正式产品名称。

任务 DTO 新增：

- `product_name`
- `policy_name`
- `execute_type`
- `module`

模块根据任务当前步骤名称归类：

| 步骤名前缀 | 模块 |
|---|---|
| `software_upgrade%` | `software_upgrade` |
| `license%` | `license` |
| 其他参数配置步骤 | `self_config` |

该方案不新增数据库迁移，展示字段均从现有任务、设备、策略和产品数据组合得到。

### 6.4 产品匹配口径统一

策略配置已经采用产品名称，但 XML 参数配置执行入口仍保留了旧的 `ProductClass` 二次校验。这会导致：策略初次匹配成功，但真正执行 XML 时又因为名称和 ProductClass 不同而被拒绝。

本次将执行入口统一为产品名称校验：

1. 优先通过设备 `product_id` 解析产品名称；
2. 无产品解析能力或旧数据场景下回退到设备 ProductClass；
3. 策略产品名称比较大小写不敏感；
4. 保留旧策略字段作为兼容回退。

## 7. 浏览器验证中发现的问题

### 7.1 产品表字段名错误

第一次部署后，浏览器捕获到：

```text
GET /api/v1/provisioning/tasks -> 500
ERROR: column prod.name does not exist
```

原因是产品表实际字段名为 `product_name`，任务关联查询误写为 `prod.name`。已修正为 `prod.product_name`，并同步更新 SQL 测试。

修正后任务接口返回成功，浏览器控制台无错误响应。

### 7.2 策略执行阶段仍使用 ProductClass

代码联动检查发现 XML 执行入口仍使用旧 ProductClass 校验。该问题在策略按产品名称配置后会形成隐蔽的执行失败，已按 6.4 节统一处理并补充兼容性测试。

## 8. 验证结果

| 验证项 | 结果 |
|---|---|
| `go test ./internal/provision -count=1` | 通过 |
| 即插即用页面相关 Vitest | 17 个测试文件、67 个测试通过 |
| Provision API 测试 | 7 个测试通过 |
| 前端 ESLint（改动文件） | 通过 |
| 前端生产构建 | 通过 |
| 本地 Docker Compose 热部署 | web/app/acs/worker 已更新并运行 |
| `http://localhost:8081/` | HTTP 200 |
| 真实浏览器页面 | 新模块页签、筛选和任务列均已呈现 |
| 浏览器任务接口 | 无 4xx/5xx |
| 浏览器控制台 | 无错误 |

验证截图（本地开发验证产物）：

- 老 OMC gNB 即插即用：`/Users/wangyong/OBJECT/Codex/.codex-tools/browser-control/legacy-omc-gnb-pnp.png`
- 当前 OMC 改造结果：`/Users/wangyong/OBJECT/Codex/.codex-tools/browser-control/current-omc-pnp-reference-update.png`

## 9. 已知但与本次无关的问题

### 9.1 全仓 Go 测试

全仓测试仅有 `internal/task` 的一条测试失败，日志显示测试 Redis 临时端口连接被拒绝。Provisioning 包及本次涉及的后端测试均通过。

### 9.2 TypeScript 全量类型检查

全量 TypeScript 类型检查存在两个既有 `BlobPart` 类型错误：

- `webcode/src/pages/SystemLicense/History.tsx`
- `webcode/src/pages/SystemLicense/index.tsx`

与即插即用页面改动无关。

### 9.3 MinIO 健康检查

本地 MinIO 服务本身已启动并能对外提供端口，但镜像内缺少 `curl`，Compose 健康检查命令无法执行，因此容器显示 `unhealthy`。该环境问题曾阻断 Compose 依赖启动，本次通过确认实际依赖状态后恢复 web/app/acs/worker。

## 10. 主要代码索引

| 位置 | 作用 |
|---|---|
| `omcmb/webcode/src/pages/device/PlugAndPlay/index.tsx` | 页面模块视图、筛选、统计和任务列 |
| `omcmb/frontend-core/src/services/api/provisionApi.ts` | 任务请求参数、响应 DTO 和视图映射 |
| `omcmb/frontend-core/src/hooks/api/useProvisioning.ts` | React Query 参数透传 |
| `omcgo/internal/provision/handler.go` | 任务筛选参数解析、统计响应、产品匹配执行入口 |
| `omcgo/internal/provision/model.go` | 任务上下文字段和筛选模型 |
| `omcgo/internal/provision/repository.go` | Repository 统计接口 |
| `omcgo/internal/provision/pg_repository.go` | 任务关联查询、筛选和状态统计 |
| `omcgo/internal/provision/handler_test.go` | HTTP 参数和统计行为测试 |
| `omcgo/internal/provision/pg_repository_test.go` | SQL 关联和筛选测试 |
| `omcgo/internal/provision/policy_test.go` | 产品名称及旧字段兼容测试 |

## 11. 后续可选增强

以下能力可根据真实运维需求继续演进，但不应仅因为老 OMC 存在就直接实现：

1. 模块专属字段：软件升级展示初始/目标版本，License 展示文件名称；
2. 任务详情增加各模块耗时和事件时间线；
3. 成功/失败统计扩展为执行中、等待中、跳过等完整分布；
4. 将任务导出与当前组合筛选条件保持一致；
5. 修复本地 MinIO 健康检查命令，避免热部署被错误健康状态阻断。

是否实施上述增强，应以当前任务数据模型和运维场景为依据，不以复刻老 OMC 为目标。

## 12. 参数配置页面补充对比

### 12.1 调研范围和证据口径

本节继续对比策略编辑页中的 gNB 参数配置能力。这里保留调研时的对比结论；后续落地结果见第 13 节。

老 OMC 结论来自已登录页面的真实浏览器操作：进入 `Advance -> Plug-and-play`，查看策略 `BNQtest`，切换到 `Parameters Config`，分别检查 `Common Parameters` 和 `Batch Import`。当前 OMC 结论来自 `/device/plug-and-play` 的新增策略页和现有 gNB 策略详情页。

浏览器检查期间，老 OMC 参数页相关请求返回 HTTP 200，控制台和页面均未捕获错误。老策略的 `Advance` 当前没有展开出已选配置项，因此 AMF、WAN/LAN、IPSec、静态路由、HaloB、管理服务和 NTP 等可选组仅以 JSP 源码作为能力补充，不视为该策略已经配置的浏览器事实。

### 12.2 老 OMC 的参数配置模型

老 OMC 在参数配置模块内进一步拆成两条独立路径。

#### 公共参数（Common Parameters）

公共参数作为适用于同产品设备的通用规则，页面实际展示以下结构：

- `GNB`：GNB Name、GNB ID Length、GNB ID；
- `CELL`：PCI，以及可展开的频段、上下行 NRARFCN、上下行带宽、SSB 频点、双工模式、上下行子载波间隔、收发天线数、两组 TDD Pattern 和 PRACH 参数；
- `PLMN`：NCI/TAC/RANAC、PLMN ID/主 PLMN、切片 SD/SNSSAI；
- `Advance`：按需增加高级配置组；
- `Customized Parameters`：名称、值和设备参数路径。

其中 GNB ID 和 PCI 不仅接受单值，还支持 `1..100;200..1000` 一类范围表达式，由 OMC 在设备之间自动分配。这是老页面与当前逐基站配置模型最实质的差异。

#### 指定设备计划（Batch Import）

该页签面向逐设备参数计划，提供独立启用开关、按序列号检索和参数列表。浏览器中确认的列表字段为：

| 字段 | 说明 |
|---|---|
| Serial Number | 目标设备序列号 |
| GNB Name | 基站名称 |
| GNB ID Length | gNB 标识长度 |
| GNB ID | gNB 标识 |
| PCI | 小区 PCI |

当前被查看的老策略没有指定设备数据，因此页面显示 `No Data`。编辑态源码还提供导入以及单行编辑、删除、查看操作，但本轮没有执行任何写入或删除动作。

### 12.3 当前 OMC 的参数配置模型

当前 OMC 没有与老页面同级的“公共参数”页签，而是以“每个基站一条结构化配置”为主：

- 模块工具栏提供下载模板、导入和导出；
- 支持按基站编码搜索；
- 列表展示基站编码、基站名称、支持频段、带宽、eNB 频率、子帧配比、更新人和更新时间；
- 每行可打开结构化详情或编辑器；
- 详情按 NTP、同步源、IPSec、小区参数、核心网参数、TDD、IP、PLMN、切片、其他模板参数和自定义参数分组；
- PLMN、切片和自定义参数支持多项结构，而不是只保留一个扁平参数表。

现有 gNB 策略 `gnb` 中有一条导入记录，基站编码为 `1202000534228JB0007`。真实详情页能够呈现具体参数值和校验范围，说明导入后的数据可回到结构化表单查看，而不是不可解释的文件黑盒。

### 12.4 逐项差异

| 能力 | 老 OMC | 当前 OMC | 判断 |
|---|---|---|---|
| 公共规则与逐设备计划分层 | 两个独立页签 | 已提供“公共参数配置”和“指定设备覆盖”两个页签 | 已补齐公共规则层，覆盖数据不再是使用公共配置的前置条件 |
| GNB ID/PCI 范围自动分配 | 支持范围和多段范围 | 支持起止值、步长和多个保留区间 | 当前使用结构化规则，避免老页面范围字符串难校验的问题 |
| 参数录入 | 页面手工配置 + 指定设备导入 | 公共参数页面手工配置；指定设备支持结构化编辑、导入和导出 | 手工公共配置为主入口，文件导入作为单站覆盖辅助入口 |
| 参数分组 | GNB、CELL、PLMN、Advance、自定义 | NTP、同步源、IPSec、小区、核心网、TDD、IP、PLMN/切片、其他、自定义 | 当前分组更贴近最终配置对象，但需建立字段映射矩阵确认完整性 |
| 列表预览 | 指定设备表聚焦 GNB ID、PCI | 包含审计字段，但仍有 `eNB 频率`、`子帧配比` 等 LTE 倾向字段 | 当前 gNB 列表列需要按制式调整 |
| 审计信息 | 浏览器可见表中无更新人/更新时间 | 有更新人和更新时间 | 当前更利于追踪导入来源和变更时间 |
| 自定义参数 | 名称、值、路径 | 支持自定义参数组 | 两者方向一致 |
| 查看体验 | 公共规则直接展开；指定设备表查看 | 列表摘要 + 结构化详情抽屉 | 当前更适合参数量较大的单站配置 |

### 12.5 设计判断

已采用“继承老 OMC 的业务能力，但不复制范围字符串交互”的方案。参数自配置默认打开公共参数页面，按 NTP、同步源、IPSec、小区、核心网、TDD、IP、PLMN/切片、其他和自定义参数分组录入；gNB ID 与 PCI 使用结构化起止值、步长和保留区间配置。

公共参数按策略的产品名称匹配设备，执行时不要求预先维护设备 SN。后端按设备创建时间和设备编码稳定排序，计算每台设备的 gNB ID/PCI，并跳过保留区间和指定设备覆盖中已经占用的值。指定设备列表和文件导入继续保留，但其定位是覆盖公共规则，而不是公共参数生效的前置条件。批量执行仍复用检测页的多设备选择，一次最多选择 50 台基站。

另外，当前 gNB 列表应优先把 `eNB 频率`、`子帧配比` 调整为按产品制式动态展示的摘要字段。对 5G 产品，更有价值的预览字段是 gNB ID、PCI、NRARFCN/SSB 频点、带宽和 TAC；当前示例记录在现有列中大部分为空，降低了列表检索价值。

### 12.6 补充验证截图

- 老 OMC 公共参数：`/Users/wangyong/OBJECT/Codex/.codex-tools/browser-control/legacy-omc-parameter-config-common.png`
- 老 OMC 指定设备计划：`/Users/wangyong/OBJECT/Codex/.codex-tools/browser-control/legacy-omc-parameter-config-batch.png`
- 当前 OMC 新增策略参数区：`/Users/wangyong/OBJECT/Codex/.codex-tools/browser-control/current-omc-add-policy-parameter-config.png`
- 当前 OMC gNB 参数详情：`/Users/wangyong/OBJECT/Codex/.codex-tools/browser-control/current-omc-policy-detail-parameter-config.png`

## 13. 参数配置能力落地结果

### 13.1 已落地范围

本次将参数自配置调整为“公共页面手工配置为主、指定设备覆盖为辅”，落地能力如下：

1. **按制式展示参数摘要**：gNB 列表展示 gNB ID、PCI、支持频段、带宽、下行 NRARFCN、SSB 频率号和 TAC；eNB 继续使用 LTE 摘要列。
2. **统一字段映射矩阵**：列表摘要、详情表单和 Excel 工作簿读取共用 eNB/gNB 字段映射定义，避免同一字段在不同入口使用不同表头。
3. **公共参数手工配置页**：参数自配置默认进入公共参数页，复用快速设置的卡片分组，提供 NTP、同步源、IPSec、小区、核心网、TDD、IP、PLMN/切片、其他及自定义参数，不依赖模板文件才能配置。
4. **运行时批量编号分配**：gNB ID/PCI 支持起始值、结束值、步长和多个保留区间；执行时按同产品设备稳定排序、跳过保留值和指定设备已占用值，重复显式编号或可用编号耗尽时阻止执行。
5. **产品级触发**：公共参数适用于符合策略产品名称的设备，不要求设备 SN 预先存在于配置表；新加入的同产品设备也可使用同一规则。
6. **指定设备覆盖**：原有结构化编辑、模板导入、导出和审计信息保留在第二页签，仅覆盖个别设备差异；覆盖内容与公共配置深度合并。
7. **多基站批量下发**：检测对话框增加一键全选，复用现有批量执行接口，一次最多选择 50 台设备，并继续执行产品名称一致性校验。
8. **导入预检与状态审计**：文件选中后显示新增、覆盖和冲突明细；列表保留来源、校验状态、最近执行以及按当前策略查询任务的能力。

导入配置继续标记为 `import`，既有或手工编辑的设备覆盖配置归为 `manual`。公共参数不预生成逐设备记录，而是在下发时物化为单台设备配置，因此不会因为策略创建时尚未录入 SN 而遗漏设备。

### 13.2 验证结果

| 验证项 | 结果 |
|---|---|
| gNB/eNB 参数规划与冲突单元测试 | 通过 |
| 参数摘要、来源、校验与导入预检单元测试 | 通过 |
| 即插即用相关 Vitest | 18 个测试文件、71 个测试通过 |
| 公共参数分配后端测试 | 稳定分配、覆盖合并、重复 PCI、范围耗尽、gNB ID 位长边界均通过 |
| `go test ./internal/provision -count=1` | 通过 |
| 前端生产构建 | 通过 |
| 本地 Docker Compose 热部署 | web/app/acs 已重新构建并运行 |
| `http://127.0.0.1:8081/` | HTTP 200 |
| 浏览器 gNB 动态列 | 全部呈现 |
| 浏览器公共参数页面 | NTP、同步源、IPSec、小区、核心网、TDD 及 gNB ID/PCI 分配卡片均可见 |
| 浏览器指定设备覆盖 | 导入入口仅位于覆盖页签，公共配置不依赖导入 |
| 浏览器批量选择 | 一键全选可用，已选设备数量正确更新 |
| 当前策略任务请求 | 携带 `policy_id`，HTTP 200 |
| 浏览器请求与页面运行 | 无 4xx/5xx，无页面异常 |

全量 TypeScript 检查仍只有第 9.2 节记录的两个既有 `BlobPart` 错误，与本次参数配置改动无关。

### 13.3 落地截图

- 公共参数配置与批量编号规则：`/Users/wangyong/OBJECT/Codex/.codex-tools/browser-control/current-omc-common-parameter-config.png`
- 老 OMC 风格的手工参数卡片：`/Users/wangyong/OBJECT/Codex/.codex-tools/browser-control/current-omc-manual-parameter-cards.png`
- 多基站批量选择：`/Users/wangyong/OBJECT/Codex/.codex-tools/browser-control/current-omc-batch-dispatch-selection.png`
