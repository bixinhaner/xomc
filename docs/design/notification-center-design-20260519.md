# 消息中心（Notification Center）设计文档

> 历史设计：本文只描述站内任务消息中心（device task feedback），不描述告警邮件或 KPI 定时报表外发。当前邮件范围、权限和验收以 `docs/superpowers/reviews/2026-08-06-email-notification-scope-convergence.md` 为准。

> 版本：v0.3
> 日期：2026-05-19
> 范围：后端（internal/notification + task）+ 前端（webcode + frontend-core）
> 状态：方案已确认，等待开工通知

---

## 1. 背景

当前 OMC 网管面对用户的"操作反馈"散落多处：

- 卡片右上角 Tag（快速设置）：能持久显示，但只针对当前分组、当前设备、用户当前在视图内才能看到
- `message.success/error`（顶部居中）：4~6 秒自动消失，错过即丢
- `notification.error`（右上角弹窗，duration=0）：仅用于"入队失败 / 基站应答失败"两类，且**关闭即丢**
- 顶部铃铛图标：**纯占位，无任何功能**

用户操作（改参数、添加 / 删除对象、固件升级、备份、MML 等）会在后端产生 device_task，状态走 `pending → sent → completed | failed | expired | cancelled`。但前端只有当用户**停留在产生该任务的视图**时才会轮询状态。一旦切走，最终结果就丢失了。

**核心痛点**：用户提交一次"快速设置"后，习惯性切到其他 tab 看别的，等回来发现该有的反馈都没了，无法判断改动到底有没有生效。

## 2. 目标

提供统一的**消息中心**，让用户在系统任意位置都能看到自己最近产生的操作反馈与终态结果。

### 2.1 必须做到（MUST）

1. 顶部铃铛点击后弹出消息列表，未读条目以红色数字徽标提示
2. 快速设置（CellParameterForm / MultiInstanceTable）触发的操作 100% 进入消息中心
3. 任务状态终态（成功 / 失败 / 超时 / 取消）即使在用户切走视图后也能呈现
4. 消息条目可点击跳转回触发它的视图（如对应设备的快速设置 tab）
5. 消息持久化，单次浏览器会话（session）内不丢

### 2.2 暂不做（NOT NOW）

- 实时推送（WebSocket / SSE）——V1 前端定时 10s 轮询 `unread-count`，铃铛打开时拉列表（足够流畅）
- 告警类消息（已有独立的告警页面与顶部 AlarmBadges，本次不重复）
- 系统级公告（升级、维护通知）
- 用户偏好（开关 / 静音 / 过滤），先用默认行为

### 2.3 永不做（OUT OF SCOPE 永久）

- **邮件 / SMS / Webhook 外发**：消息中心定位为**站内查看**通道；外发由告警 / 工单等其他业务通道负责，互不重叠。后端 `notification` 模块虽然内置了 mailer，但本设计的 task 订阅器**只写 DB，不调 mailer**。

### 2.4 衡量标准

- 用户提交快速设置后立即切走，5 分钟内回来打开铃铛能看到该操作的最终状态
- 任意写操作都进消息中心，**不能漏**
- 未读徽标准确反映"用户未关注过"的条目数

---

## 3. 用户故事

| # | 角色 | 场景 | 期望 |
|---|------|------|------|
| US-01 | 网管运维 | 在快速设置改 TAC 提交后切到设备列表查另一台设备 | 切走前后右上角铃铛能显示这条任务结果 |
| US-02 | 网管运维 | 同时给多台设备下发指令 | 消息中心按时间倒序列出每台设备每次操作 |
| US-03 | 网管运维 | 点开消息中心看到一条"基站应答失败" | 点击条目直接跳回那台设备的快速设置，定位问题 |
| US-04 | 网管运维 | 已经看过的消息不再扰动 | 已读消息灰色显示，未读保持高亮（左侧蓝条 + 标题加粗）；提供"全部已读"按钮一键清徽标 |
| US-05 | 网管运维 | 历史条目堆太多 | 提供"全部已读 / 清空"操作 |
| US-06 | 网管运维 | 看消息中心想确认具体改了什么 | 主标题写出参数名 + 新旧值；多条变更在详情区逐行列出 |

---

## 4. 范围（V1）

### 4.1 来源（哪些操作进消息中心）

**后端订阅器对所有 task 通用** —— 一旦上线，任何模块产生的 `task.*` 事件都会自动写 notification，无需逐模块改造。下表列的是 **V1 前端验证范围**（保证以下场景在 Phase 1 完成时能跑通端到端），不代表后端订阅边界：

| 触发位置 | 操作 | V1 前端验证 |
|----------|------|--------------|
| CellParameterForm | 单实例分组 Save → SetParameterValues | ✅ |
| MultiInstanceTable | 行级 Save → SetParameterValues | ✅ |
| MultiInstanceTable | AddObject | ✅ |
| MultiInstanceTable | DeleteObject | ✅ |
| 参数树（ParameterTreeTab） | SetParameterValues | ⏸（后端自动覆盖，Phase 2 补 invalidate 触发点） |
| MML 控制台 | TR069 RPC | ⏸（同上） |
| 固件升级 / 备份 / 配置基线 | 各类 task | ⏸（同上） |

> V1 先打通"快速设置"全链路是因为它是用户原始诉求；后端 subscriber 通用化后，Phase 2 各模块只需补 `queryClient.invalidateQueries(['notifications'])` 触发点即可，无需改后端。

### 4.2 UI 位置与形态

- **入口**：现有顶部 Header 右侧的铃铛按钮（`omcmb/webcode/src/components/Layout/Header`）
- **未读徽标**：复用现有的 antd `<Badge count>`，红点 / 数字（≥1 显示数字，≥100 显示 `99+`）
- **弹出形态**：点击后弹出 antd `<Popover>` 面板（≈ 360px × 480px，下方对齐铃铛）
  - 头部：标题"消息中心" + 右侧"全部已读"和"清空"两个文字按钮
  - 列表：默认显示最近 20 条；超过用 antd `<List>` 内置滚动；条目按 `updatedAt` 倒序
  - 空态：插画 / 文案"暂无消息"
- **条目元素**（状态视觉与文案见 §4.4）：
  - 左侧状态图标（5 种：进行中蓝旋转 / 完成绿√ / 失败红× / 超时黄⏰ / 取消灰⊘）
  - 主标题（文案规范见 §4.5，例 "小区参数 · TAC: 3 → 4"）
  - 副标题（设备 SN + 相对时间 "3 分钟前"）
  - 状态文案（"进行中" / "已完成" / "失败: SPV reject reason..." / "超时" / "已取消"）
  - 详情区（多条变更时展开，最多 5 行；含错误信息）
  - 未读条目左侧加蓝色竖条
- **点击行为**：标记为已读 + 跳转到 `linkTo`（如 `/device/detail/{sn}?tab=quickSettings`）

> 不做独立"消息中心页面"（V1 用 Popover 已经足够；列表超过 20 条用滚动）。

### 4.3 消息生命周期

```
┌──────────┐  入队成功  ┌──────────┐  CPE 应答  ┌──────────┐
│ 用户点保存│ ────────▶ │ queued  │ ────────▶ │ completed│
└──────────┘            │ /sent   │ │         │ /failed/ │
                        └──────────┘ │         │ expired  │
                              │      │         │ /cancelled│
                              │      └──超时────┴──────────┘
                              │
                          入队失败
                              │
                              ▼
                          ┌─────────┐
                          │ failed  │（直接终态）
                          └─────────┘
```

**红线（不可逾越）**：消息中心列表里的每一条都是**从后端 GET `/notifications` 拉回**的。前端**禁止**本地造消息、禁止乐观插入、禁止用 store 缓存"待回写"的临时条目。后端是消息的唯一真相源。

**流程**：

1. 用户点保存 → 前端 POST 触发 task（如 SetParameterValues）→ 后端入队，task service publish `task.created`
2. notification subscriber 收到事件 → 渲染文案 → 写 `notifications` 表（`status='queued'`, `dedup_key=task_id`）
3. 前端拿到 POST 的 202 响应后，**主动** `queryClient.invalidateQueries(['notifications'])` 触发立即 refetch（不等 10s 轮询周期）
4. React Query 重新 GET `/notifications` → 拿到那条"进行中"消息 → 渲染
5. 后续 CPE 应答 → task 状态变更 → subscriber 收到 `task.completed`/`task.failed`/... → upsert 同一条（按 dedup_key）→ 下一轮 10s 轮询或下次铃铛打开自动拉到终态

**入队失败分支**（避免"必须做到 #2"的覆盖缺口）：

- 入队失败时 task service 还没创建 task，没有 task_id，EventBus 上不会有任何 `task.*` 事件 → subscriber 收不到
- 红线限制前端**不能本地造消息**，否则消息中心会出现"前端有后端没"的脏数据
- **解法**：device handler / config handler 等所有 `taskSvc.CreateTask` 调用点，在 err 返回路径里**同步**调一次 `notificationSvc.CreateNotification(status='failed', sender='system', title='下发失败', content=errMsg, link=...)`，dedup_key 用前端传来的 idempotency-key 或服务端 uuid
- 这保证"红线（消息只能来自后端）"与"必须 100% 进入消息中心"两条同时成立

**延迟特性**：

- 用户感知"消息出现"的延迟 ≈ 一次 HTTP RTT（~50ms），不依赖 10s 轮询周期
- 极端竞争（前端 invalidate 在 subscriber 写库之前完成）：第一次 refetch 拿不到 → 10s 后下一轮兜底 → 最坏 10s。这种情况罕见（subscriber 是 NATS 消费，毫秒级），可接受
- 终态消息出现的延迟 = task 实际完成时间 +（最多 10s 轮询周期）。可后续加 SSE 优化（§7 Phase 3）

> 容量保护：后端按 user 限制保留 N 条，超出自动 GC 最老的（Phase 3 加，V1 不限）；前端不做 LRU。

### 4.4 状态文案分层

**设计原则**：内部存储沿用后端 6 个状态原值（便于和 `useDeviceTaskStatus` 直接对接，不做翻译），UI 显示层将 6 态聚合为 5 个用户语义。

| 内部状态（后端语义，不改） | UI 显示文案 | 视觉 | 用户认知 |
|---|---|---|---|
| `queued` / `sent` | **进行中** | 蓝 + 旋转图标 | "我刚操作完，系统在处理" |
| `completed` | 已完成 | 绿 + ✓ | 成功 |
| `failed` | 失败 | 红 + × | 失败（带错误详情） |
| `expired` | 超时 | 黄 + ⏰ | 超时未应答 |
| `cancelled` | 已取消 | 灰 + ⊘ | 用户主动 / 系统取消 |

好处：
- 后端状态机演进（新增 `retrying` 等）时，UI 层只需补 case，不污染存储
- 用户视角的"还在做"被聚合为"进行中"，减少认知负担
- 卡片右上角 Tag（QuickSettings extra）也复用相同分层规则，全局文案一致

### 4.5 操作文案规范

**目标**：用户在消息中心（以及保存后的 Tag）一眼看清"改了哪个参数、改成什么值"，不需要点进去翻历史。

**主标题（必带）**

| 场景 | 单条变更 | 多条变更 | 备注 |
|---|---|---|---|
| 单实例 Save（CellParameterForm） | `小区参数 · TAC: 3 → 4` | `小区参数 · 3 项变更` | 多条时副标题外加省略详情 |
| 多实例 行 Save（MultiInstanceTable） | `邻区列表 #2 · PCI: 3 → 5` | `邻区列表 #2 · 2 项变更` | `#N` 为实例号 |
| 多实例 AddObject | `邻区列表 · 新增实例 5` | — | 后端返回的新实例号 |
| 多实例 DeleteObject | `邻区列表 · 删除实例 3` | — | |

**详情区（多条变更时必带；单条可省）**

按 `参数名: 旧值 → 新值` 逐行列出，控制在 5 行内，超过的折叠 "...共 N 项"。详情区同时承载错误信息（应答失败时）：

```
TAC: 3 → 4
PCI: 3 → 5
DLEarfcn: 55340 → 55320

错误：CPE 拒绝 SPV — invalid value for PCI
```

**Tag 文案（卡片右上角）保持现状**：仍是 `进行中 · N 项 · 13:38:40` 风格（一行受限），不展开参数列表 —— 用户想看详情就点铃铛。

**实现要点**

- `handleSave` 收集 dirty 字段时，连同 schema 里的 oldValue 一并打包，传入 `addItem({ title, description })`
- 旧值取 `schemaByPath.get(path)?.currentValue`（已经在用了）
- 字段显示名优先用 `param.titleZh / titleEn`（取代 `name`），失败兜底用 path 末段
- 值如果是长字符串（>20 字符）截断 + `…`，鼠标悬停 Tooltip 显示完整值（V2）

### 4.6 任务超时规则（status='expired'）

> 当前后端**未启用** task 超时机制（详见下表"现状"）。Phase 1 必须先把超时基础设施补完，"超时"消息才可能出现。

**当前真相**

| 调用点 | `ExpiresIn` 设置 | 实际效果 |
|---|---|---|
| `device.SetParameters`（快速设置 / 参数树都走这里） | 未传 → `ExpiresAt=nil` | 永远不会 expire |
| `device.Reboot` / `config.Sync` / 多数 device 操作 | 未传 | 永远不会 expire |
| `ops.dispatchInlineRPC`（批量运维） | 硬编码 `DefaultPerCmdTimeoutSeconds = 60` | 设置了但**无 sweeper 调用 `MarkExpired`** |
| `alarm.sync_service` | 硬编码 600s | 同上 |

**Phase 1 必做**

1. **配置化默认超时**：`config.{dev,prod,test}.yaml` 新增
   ```yaml
   task:
     default_expires_in_seconds: 120  # 单 task 等待 CPE 应答的默认超时
     sweep_interval_seconds: 10       # expired 扫描周期
   ```
2. **CreateTask 默认值兜底**：若 `req.ExpiresIn == 0` 且配置中有默认值，用配置默认值；调用方可显式传 0 表示"永不超时"（仅限运维场景，必须显式声明）。各业务保留覆盖能力（如告警同步仍可传 600）
3. **worker 进程新增扫描器**：周期扫 `ExpiresAt < now() AND status IN ('pending', 'sent')` 的 task → `MarkExpired` + publish 终态事件 → 消息中心 subscriber 自动接到
4. 现有 ops / alarm 等硬编码 `ExpiresIn` 的位置**保留**（业务定制值），不强行替换为全局默认

> **C2 实施期发现**：`SubjectForStatus(expired)` 已映射到 `event.SubjectTaskFailed`（与 failed 共用 NATS 主题）。新建独立 `task.expired` 主题会牵动 CompletionRouter / event_bridge 等订阅方，零侵入价值不大。**实际做法**：sweeper 直接复用现有 `notifyCompletion` 路径走 `task.failed` 主题；消息中心 subscriber 收到事件后**按 task.Status 字段**区分 `failed` vs `expired`（C5 实施时按此对接）。订阅器逻辑简单一行 switch，比新增主题省事。

**为什么默认 120s**

- TR069 CPE 应答实际落在 5-60 秒区间（与设备弱网 / 厂商实现 / 当前会话状态有关）
- 120s = 上界 60s × 2，给慢响应留足缓冲，又不让用户傻等数分钟
- 用户感知：点保存超过 2 分钟还没结果，几乎可断定设备不通
- 必要时按 method 类型差异化（如固件升级 Download 类应该 > 5 分钟）—— Phase 2 优化

**对消息中心的影响**

- 超时 status 切换由后端 sweeper 触发，subscriber 同步 upsert notification → 前端下次轮询拉到
- 用户在消息中心看到的"超时"是真实的超时（task 状态已是 expired，且 sweeper 已写库），不是前端猜测

---

## 5. 关键设计决策

### 5.1 存储位置：后端表 + 按用户隔离

**选择：V1 直接落地后端**，复用已存在的 `internal/notification` 模块。

> 后端 70% 设施已就绪：`Notification` 模型（user_id / type=`task_complete` / title / content / link / is_read / read_at / priority）、5 个 REST API（list / unread-count / mark-read / mark-all-read / delete）、JWT 用户隔离全都在。`device_tasks` 表带 `creator_id`，task 模块已 publish `task.completed` / `task.failed` 事件。仅需新建"task → notification"桥即可。

| 维度 | 前端 sessionStorage（弃） | 后端 + 按用户隔离（V1 采用） |
|------|---------------------|---------------|
| 跨浏览器 | ❌ | ✅ |
| 跨终端 / 多人协作 | ❌ | ✅ |
| 用户切换不串台 | 需手动 clear | ✅ 天然按 JWT user 隔离 |
| 历史归档 / 审计 | ❌ | ✅ |
| 工作量 | 1 个 store + 接入 | 1 个 subscriber + 前端 API hook，**比前端 store 还省**（后端 5 个 API 已实现） |

不会做的事（关闭决策）：
- 不重复造前端 store —— 直接 React Query 缓存后端数据，复用现有 `useMock` 切换
- 不做后端表 schema 变更——既有 `notifications` 表足够，只补一个 `task → notification` 触发链路

### 5.2 状态来源：后端事件桥 + 前端轮询 `unread-count`

**写入侧（后端）**

- 用户提交 task 时，App 进程在 task 表写入 `creator_id`（已存在）
- task 状态变更（pending → sent → completed / failed / expired / cancelled）由 ACS / worker 写库并 publish 到 EventBus（已存在）
- **新增** `internal/notification/task_subscriber.go`：订阅 `task.completed` / `task.failed` / 可能的 `task.expired` / `task.cancelled` 主题，把任务信息（参数变更明细 / 错误消息 / 关联 device_sn）渲染成 `Notification.Title / Content / Link`，写入 user 的消息中心
- 触发瞬间（用户点保存的 mutation 请求被 task service 入队那一刻），由 task service 自己（或同一 subscriber 监听 `task.created`）写第一条 `进行中` 状态的 notification；后续状态变更只 update 同一条（按 task_id 作为 dedup key）

**读取侧（前端）**

| 时机 | 行为 |
|---|---|
| 任意页面 | `useQuery` 每 10 秒 GET `/notifications/unread-count`，未读数变化 → 铃铛徽标更新 |
| **保存类按钮 POST 成功（写操作）** | **`queryClient.invalidateQueries(['notifications'])`** 强制立即 refetch，把后端 subscriber 刚写的"进行中"那条拉回；不依赖 10s 轮询，用户感知延迟 ~50ms |
| 用户点铃铛 | GET `/notifications?page=1&pageSize=20`，渲染列表 |
| 用户点条目 | `PUT /notifications/:id/read` + navigate(link) |
| 用户点"全部已读" | `PUT /notifications/read-all` |
| 用户点"清空" | `DELETE /notifications`（一键清空当前用户全部消息，后端 Phase 1 补） |

**红线（与 §4.3 一致）**：前端**绝不**本地造消息条目。所有显示数据 100% 来自 GET `/notifications`。前端 `invalidateQueries` 只是触发 refetch，不是写入。

> 不做 SSE/WebSocket：10s 轮询的 `unread-count` 单次响应 < 100B，对 10k 用户也只是 1k QPS，nginx + app 进程足够。实时性损失最多 10 秒，可以接受。后续真需要再补推送通道。

### 5.3 唯一标识：Notification.id（uuid）+ dedup key

后端 `Notification` 用 uuid 主键。task → notification 的关联用 dedup key：

- `Notification.Content` 中嵌入 `task_id`（结构化字段，前端按需解析）或
- 给 `notifications` 表加一个可空索引列 `dedup_key`（task uuid），订阅器在写入前 upsert：相同 `dedup_key` 已存在 → 更新；否则插入。

后者更干净，需要一条迁移（加列 + 部分唯一索引 `WHERE dedup_key IS NOT NULL`），但避免了在 `Content` 里塞协议字段。**采用后者**。

### 5.4 消息中心 vs 卡片 Tag 关系

两者**并存且互不替代**：

- 卡片 Tag：用户在视图内时的**即时反馈**，对位明确（哪个分组的哪次保存）
- 消息中心：用户**离开视图后的兜底**，跨视图汇总

写入两边的数据来源是同一个 mutation，逻辑由调用方在 `handleSave / handleAdd / handleDelete` 中显式触发，不做隐式联动。

### 5.5 阅读 / 未读规则

- 新条目默认未读
- 用户**点开 Popover** 视为查看，但不自动全部已读（保留"哪条没注意到"的能力）
- 用户**点击单条** → 该条已读
- 用户**点"全部已读"** → 全部置为已读
- 未读徽标在所有条目变已读后自动消失

> 替代方案：打开 Popover 即标记全部已读。被弃用原因：用户可能只是扫一眼徽标想知道总数，没真正读条目。

---

## 6. 数据模型（沿用后端 `Notification`，仅扩展 dedup_key）

后端 `Notification` 字段已经覆盖大部分需求，**仅需新增一列 `dedup_key`**：

| 字段 | 来源 | 用途 |
|------|------|------|
| id | 已有，uuid | 主键 |
| user_id | 已有 | 取自 `task.creator_id`，按用户隔离 |
| type | 已有，本次用 `task_complete` | 后续扩展 `alarm` `system` |
| priority | 已有 | 失败 = `high`，成功 = `normal` |
| title | 已有 | 文案规范见 §4.5（"小区参数 · TAC: 3 → 4"） |
| content | 已有 | 详情区文本：参数列表 / 错误信息（多行） |
| link | 已有 | `/device/detail/{sn}?tab=quickSettings` |
| sender | 已有 | 固定 `system` 或操作类型（"快速设置"） |
| is_read / read_at | 已有 | 已读状态 |
| created_at | 已有 | 取 `task.created_at`（用户点保存那一刻，而非 subscriber 写库时间），保证用户视角时间准确 |
| **status**（**新增**） | 五态文案分层（§4.4）的内部状态 | 前端图标 / 颜色 / 排序 / 过滤直接用 |
| **dedup_key**（**新增**） | task uuid | 状态变更时 upsert 同一条 |

**关于"状态"**（O-04 已定：加显式 `status` 列）：
- 用户提交 → 写一条 `status='queued', title="..."`
- task 进入 `sent` → upsert，`status='sent'`
- task 终态 → upsert 同一条，`status ∈ {completed, failed, expired, cancelled}`，title / content 同步更新
- priority 与 status 解耦：`failed / expired` 自动升 `high`，其他保持 `normal`

容量上限：后端可按 user 限制保留 N 条，超出自动删除最老的（V2 加，V1 不限）。

---

## 7. 阶段计划

### Phase 1 — 后端缝合 + 快速设置接入（本次）

**A. 后端 — 超时基础设施（见 §4.6）**
1. 配置：`config.{dev,prod,test}.yaml` 新增 `task.default_expires_in_seconds=120` + `task.sweep_interval_seconds=10`
2. `task.CreateTask` 默认值兜底：`ExpiresIn==0 && 配置有默认值` → 用默认
3. worker 新增 `task_sweeper.go`：周期扫 `ExpiresAt < now() AND status IN ('pending','sent')` → `MarkExpired` + publish `task.expired`

**B. 后端 — 消息中心数据层**
4. 迁移：给 `notifications` 表加 `status VARCHAR(20) NOT NULL DEFAULT 'queued'` + `dedup_key VARCHAR(64) NULL`，并建部分唯一索引 `(user_id, dedup_key) WHERE dedup_key IS NOT NULL`
5. model / repository 加 `status` 字段，service 加 `UpsertByDedup(notif)` 方法
6. handler 新增 `DELETE /notifications` 一键清空当前用户全部消息（一行 SQL）

**C. 后端 — task → notification 订阅链路**
7. 查 task service 当前 publish 的事件清单（O-05）：
   - 若已 publish `task.created` → 订阅；写 `status='queued'` 消息
   - 若未 publish → 补 publish，或退化为仅消费终态事件、跳过"进行中"过渡
8. 新建 `internal/notification/task_subscriber.go`：
   - 订阅 `task.created` / `task.sent`（若有）/ `task.completed` / `task.failed` / `task.expired` / `task.cancelled`
   - 按 task 参数变更明细（来自 task.Payload）渲染 §4.5 的标题 / 详情
   - 按用户 locale 写入文案（O-03）
   - upsert 时按 dedup_key=task_id 保证幂等
   - `notification.created_at` 取 `task.created_at`（保证用户视角"触发那一刻"准确，而非 subscriber 写库时间）
9. 注册到 worker 进程（或 app 进程，看 EventBus 单/多实例策略）

**D. 后端 — 入队失败兜底（§4.3 入队失败分支）**
10. 所有调用 `taskSvc.CreateTask` 的 handler / service（device / config / ops / alarm 等）在 err 返回路径里同步调 `notificationSvc.CreateNotification(status='failed', sender='system', title='下发失败', content=errMsg, link=...)`，dedup_key 用 client-uuid 或服务端 uuid 兜底
11. AddObject / DeleteObject 等当前不返 taskId 的接口（O-02）：统一改造为也产出 task_id（与 SetParameterValues 同体系）

**E. 前端 — hook 与组件**
12. 新建 `useNotifications` / `useUnreadCount` / `useMarkRead` / `useMarkAllRead` / `useDeleteNotification` / `useClearAllNotifications` hooks（frontend-core）
13. 头部铃铛接 `<Popover>` + Badge 显示 unreadCount，未读 / 已读视觉对齐 §4.2（蓝条+加粗 / 灰色）
14. 消息列表 UI（按 created_at 倒序、点击跳转 + markRead、全部已读 / 清空按钮）
15. 默认每 10 秒轮询 unread-count；铃铛打开时拉列表

**F. 前端 — 快速设置接入**
16. CellParameterForm / MultiInstanceTable 的 mutation 调用方式不变（task 自动被订阅器消费）
17. mutation 成功 / 失败回调里调 `queryClient.invalidateQueries(['notifications'])` + `['notifications', 'unread-count']`，让消息中心 ~50ms 内出现"进行中"或"入队失败"那条；前端不本地造消息
18. 卡片右上角 Tag 改为 hook 当前 task 状态（沿用 `useDeviceTaskStatus`），不读 quickSettingsFeedbackStore；store 文件可以保留作过渡

### Phase 2 — 扩展接入（后续 sprint）

只要其他业务也走 task / 产生 task_id，**它们自动出现在消息中心，无需改一行前端 / 订阅器代码**。需要单独接入的：
- 不走 task 的操作（直接 PUT / POST 类）—— 手动调 `notificationSvc.CreateNotification`
- 参数树（ParameterTreeTab）—— 它走的也是 task，自动覆盖
- MML 控制台 / 固件升级 / 备份 / 配置基线—— 同上
- 系统级提示（升级、维护通知）

### Phase 3 — 实时与归档（远期）
- SSE / WebSocket 推送（替代 10s 轮询）
- 每用户保留上限自动 GC
- 用户偏好（静音 / 类型过滤）

> 邮件 / SMS / Webhook 外发明确在 §2.3 永不做之列。

---

## 8. 验收

### Phase 1 完成定义

| # | 验收项 | 方式 |
|---|--------|------|
| 1 | 铃铛点击弹出 Popover，徽标显示未读数 | 浏览器手测 |
| 2 | 快速设置保存后立即出现在消息中心，状态正确演进 | Playwright 自动化 |
| 3 | 用户提交后立刻切到设备列表，再打开铃铛能看到最终状态 | Playwright 自动化 |
| 4 | **用户 A 与用户 B 登录同一浏览器（先后切换），各自只能看到自己触发的消息** | 浏览器手测 |
| 5 | **跨浏览器一致：用户 A 在 Chrome 触发任务，在 Firefox 登录同账号能看到** | 浏览器手测 |
| 6 | 点击消息条目跳转到对应设备快速设置 | 浏览器手测 |
| 7 | "全部已读"、"清空"按钮工作 | 浏览器手测 |
| 8 | 刷新 / 关闭重开浏览器消息不丢（后端持久化） | 浏览器手测 |
| 9 | **超时机制**：CPE 模拟器对一次 SPV 不应答，等过 `default_expires_in_seconds`，消息条目从"进行中"切到"超时"，状态由后端 sweeper 推进 | Playwright + CPE simulator 自动化 |
| 10 | **入队失败兜底**：模拟 task service 入队 5xx，消息中心立即出现"入队失败"条目（来自 §4.3 入队失败分支） | mock 后端注入错误 + Playwright |
| 11 | 后端 typecheck + 单元测试覆盖订阅器 mapper + 入队失败分支 | `go test ./internal/notification/... ./internal/task/...` |
| 12 | 前端 typecheck 全绿 | `npm run typecheck` |

---

## 9. 风险与开放问题

| # | 项 | 影响 | 缓解 |
|---|------|------|------|
| R-01 | 订阅器消费失败 / 进程崩溃，task 状态变更没有写入 notification | 消息丢失 | NATS JetStream 至少一次投递 + 订阅器幂等（dedup_key）；故障重启后未确认消息会重投 |
| R-02 | 用户在终态前断网 / 关闭浏览器，"进行中"条目永久卡住 | 列表噪声 | 订阅器接收 `task.expired` 也会 upsert，最终所有任务都会到终态 |
| R-03 | 10s 轮询 unread-count 对后端的额外压力 | 单次响应 < 100B，10k 用户 × 0.1 QPS = 1k QPS | nginx 可缓存 + app 端有 Redis 计数器即可；不行再上 SSE |
| R-04 | task.created 事件 task 模块可能未 publish | "进行中"条目缺失，但终态条目仍能到 | 检查 task service：若无则订阅器仅消费终态事件，title 直接写"已完成"，跳过"进行中"过渡 |
| ~~O-01~~ | "基站应答失败"时是否**同时**保留右上角 `notification.error` 弹窗？ | — | **决定：保留**。弹窗承担"立即注意"，消息中心承担"事后回看"，职责互补 |
| ~~O-02~~ | dedup_key 用 `task_id` 是否够？ | — | **决定**：以 `task_id` 为主键。若 AddObject 等当前不返 taskId，统一改造为也走 task 体系（产出 task_id），保证所有写操作都能进消息中心；若改造成本高再退到 `creator_id + opType + nano-ts` 复合 key |
| ~~O-03~~ | 跨语言（i18n）支持？ | — | **决定：写入时按用户 locale 渲染**（user 表已有偏好），存到 `title / content` 后切换语言不重写历史 |
| ~~O-04~~ | 是否给 notifications 表加显式 `status` 列 | — | **决定：加**。`status VARCHAR(20)` 取值与 §4.4 五态对齐（`queued / sent / completed / failed / expired / cancelled`），前端图标 / 颜色靠它，未来扩展无歧义 |
| ~~O-05~~ | task.created 是否需要 task 模块补 publish？ | — | **决定**：实施前先查后端 task service 当前事件清单。如缺则补 `task.created` 让"进行中"条目能立即出现；若架构上不便，订阅器降级为只消费终态事件、跳过"进行中"过渡（用户体验稍降） |

---

## 10. 实施计划（Phase 1 可执行版）

> 把 §7 的 18 个工作项重新按"**可独立验证 / 可独立 commit**"颗粒度归并为 10 个 commit（C1-C10），每个有出口门、依赖、估时、激活专家。开工时按 dev-pipeline S2→S3→S4→S5→S6 一笔笔走。
>
> 注：实施开始前需在 `docs/project/backlog.md` 登记 umbrella task（建议 `T-NNNN 消息中心 V1`），并把 C1–C10 作为 sub-task 写到 `docs/project/backlog/subtasks/T-NNNN-notification-center.md`。

### 10.1 任务分解（按 commit 颗粒）

| Commit | 标题 | 改动范围 | 激活专家（§16） | 估时 |
|---|---|---|---|---|
| **C1** | task 超时配置化 + CreateTask 默认值兜底 | `cmd/app/etc/config.*.yaml`、`internal/appconfig/`、`internal/task/{model,service}.go` + `_test.go` | Go 工程 + 架构 + TR-069 | 0.5d |
| **C2** | worker 新增 task_sweeper 周期扫描器 | `cmd/worker/main.go`、`internal/task/sweeper.go` + `_test.go` | Go 工程 + 运维 | 0.5d |
| **C3** | notifications 表迁移 + model/repo 扩 `status` & `dedup_key` | `migrations/000NNN_*.sql` (up/down)、`internal/notification/{model,pg_repository,service}.go` + `_test.go` | 数据 + 安全 | 0.5d |
| **C4** | `DELETE /notifications` clear-all API | `internal/notification/{handler,service,pg_repository}.go` + handler_test | Go 工程 + 安全 | 0.2d |
| **C5** | task → notification 订阅器（含 O-05 检查 / 补 publish） | `internal/task/service.go`（可能补 publish）、`internal/notification/task_subscriber.go` + `_test.go`、`cmd/worker/main.go`（注册） | 架构 + Go 工程 + 电信业务 | 1.5d |
| **C6** | 入队失败兜底（§4.3 入队失败分支） | `internal/{device,config,ops,alarm}/*.go` 所有 `taskSvc.CreateTask` 调用点 + `_test.go` | Go 工程 + 安全 | 1.0d |
| **C7** | AddObject / DeleteObject 改造返 task_id（O-02 决议） | `internal/device/device_param_handler.go`、`internal/device/device_service.go`、`frontend-core/hooks/api/useDeviceParameters.ts`、`frontend-core/types/deviceParameter.ts` | 前端 + Go 工程 + TR-069 | 1.0d |
| **C8** | 前端 notification API + 6 个 hook | `frontend-core/services/api/notificationApi.ts`、`frontend-core/hooks/api/useNotifications.ts`、`frontend-core/types/notification.ts`、`frontend-core/mock/services/notificationService.ts` + vitest | 前端 | 0.5d |
| **C9** | Header 铃铛 Popover + 消息列表组件 | `webcode/components/Layout/Header/index.tsx`、`webcode/components/NotificationCenter/{index,Item,EmptyState}.tsx`、i18n 语料 | 前端 | 1.5d |
| **C10** | 快速设置 mutation 触发 invalidate + Tag 改读 task 状态 | `webcode/pages/device/DeviceDetail/QuickSettingsTab/{CellParameterForm,MultiInstanceTable}.tsx` + Playwright E2E | 前端 + 测试 | 0.5d |

**总估时**：7.7 人天（不含联调 + 验收）

### 10.2 拓扑依赖与并行机会

```
C1 ──→ C2                                            （后端运维，关键路径 1）
                                                     
C3 ──┬──→ C5 ──┐                                     （后端消息中心，关键路径 2）
     │         │
     └──→ C6 ──┤                                     
                │                                    
C4 ──────────────────────────┐                       
                              │                       
                              └──→ C8 ──→ C9 ──→ C10  （前端，关键路径 3）
                                                     
C7（独立改造，无依赖，可早开）─────┘                  
```

**关键路径**：C3 → C5 → C8 → C9 → C10 ≈ 4.5d
**可并行**：C1+C2 与 C3+C4+C6 可同一周并行（不同模块文件不冲突），C7 全程任意时段开始

### 10.3 每 commit 出口门（S3 终态硬命令）

| Commit | 出口门 |
|---|---|
| C1 | `go build ./... && go test ./internal/task/... && go test ./internal/appconfig/...` 全绿；新增 ExpiresIn 默认值的 table-driven 用例覆盖 3 种分支（未传 / 显式 0 / 显式值） |
| C2 | `go test ./internal/task/...` 全绿；造一条 ExpiresAt 过期的 task，跑一轮 sweeper 后断言状态变 expired + publish 一次事件 |
| C3 | `goose up && goose down && goose up` 三遍均成功；新字段 read/write 单元测试；UpsertByDedup 命中已存在条目时不重复插入 |
| C4 | `go test ./internal/notification/...` 全绿；接口契约：未登录返 401 / 跨用户尝试删除返 403（虽然是清自己的，验证 user_id 隔离） |
| C5 | 6 个状态事件各跑一次集成测试，断言写库；同一 task_id 的多次状态变更 upsert 同一条；locale='zh-CN'/'en-US' 文案两路用例 |
| C6 | 所有 CreateTask 调用点 mock 返 err，断言对应 handler 调一次 CreateNotification(status='failed')；dedup_key 来源验证 |
| C7 | 前端 `npm run typecheck` + 后端 `go test`；deviceParameterApi.addObject / deleteObject 返回类型新增 taskId |
| C8 | `npm run typecheck && npm run test`；mock 与 real 两路 hook 行为一致；轮询 10s 间隔可调（test override） |
| C9 | `npm run typecheck`；Playwright：登录 → 点铃铛 → 看到空态；mock 一条消息 → 列表显示 + 点击 markRead + 跳转 |
| C10 | Playwright E2E：登录 → 快速设置改 TAC 保存 → 50ms 内消息中心出现"进行中" → 5 秒后切到终态（mock CPE 应答） |

### 10.4 风险检查点（每 commit 后做的）

- **C1 后**：所有现有 task 调用点回归（reboot / config sync / SPV / alarm sync），确认默认值兜底不破坏现有行为
- **C2 后**：观察 task_sweeper 日志一天，确认没有把活跃 task 误标 expired（边界：1 秒内到期）
- **C3 后**：跑一次 `dev-pipeline audit`，确认迁移文件版本号未与并行 PR 冲突
- **C5 后**：用 ACS simulator 跑一次完整 SPV → 应答路径，肉眼看 notifications 表是否多了一行 status=completed
- **C6 后**：临时把 task service 改成必失败，验证消息中心是否出现"入队失败"
- **C9 后**：在 prod 风格的 nginx 后跑 10 个并发用户 5 分钟，看 unread-count 轮询是否引起明显 QPS 抖动
- **C10 后**：与 §8 验收 #1-#10 逐项过

### 10.5 回滚策略

| 范围 | 回滚 |
|---|---|
| C1-C2（task 超时） | 配置文件还原；回滚迁移（C2 无 schema 变更） |
| C3（notifications 迁移） | `goose down -1`，drop status + dedup_key 列；前端 hook 暂停发请求即可（消息中心隐藏铃铛或显示空态） |
| C4 | 删 handler 路由 + service 方法 |
| C5（订阅器） | worker 进程关掉订阅器；不影响 task 主链路 |
| C6（入队失败兜底） | handler 还原（无 schema 变更） |
| C7（AddObject 返 task_id） | API 兼容：响应里 task_id 字段可选；旧前端忽略；新前端容错处理 |
| C8-C10（前端） | Header 还原铃铛为占位、移除 invalidate 调用即可 |

### 10.6 联调与验收节奏

- **C1+C2 完成**：单点演练超时机制（脱离消息中心也能验）
- **C3+C4+C5 完成**：用 curl 直接验后端 5 个 API（list / unread / mark-read / mark-all-read / delete + clear-all），断言 user 隔离
- **C6 完成**：复盘所有调用点的入队失败链路
- **C8+C9 完成**：前端单独跑 mock 数据，铃铛 + Popover 视觉确认
- **C10 完成**：端到端 Playwright（含跨 tab 切走切回、跨浏览器、用户切换）→ 触发 §8 验收

### 10.7 backlog 登记建议

开工前在 `docs/project/backlog.md` 登记：

```
| T-NNNN | feat(notification): 消息中心 V1 — task 事件桥 + 快速设置接入 | P1 | planned | docs/design/notification-center-design-20260519.md | 7.7d | 跨域(后端+前端) |
```

并在 `docs/project/backlog/subtasks/T-NNNN-notification-center.md` 落地 C1-C10 sub-task 表（沿用上 §10.1 的列）。

### 10.8 提交 footer 三元组（dev-pipeline S6 要求）

所有 C1-C10 commit footer 必含：

```
PRD: docs/design/notification-center-design-20260519.md
Sprint: sprint-NN  # 开工时填
Risk: R-01..R-04 / N/A  # 该 commit 涉及的 R 编号
```

---

## 11. 实施期遗留问题登记（运行索引）

> 实施 C1-C10 过程中，凡决定"**暂不解决**"的问题统一登记本节，确保从设计文档一眼看到本功能的所有遗留点。每条同步登记 `docs/project/backlog.md` Proposed 区或 §3 Active 表，但**最终真相源以本节为准**。
>
> 字段说明：
> - **ID**：`L-NN`（Legacy 缩写），文档内编号
> - **关联**：发现于哪个 commit / 涉及哪个文件
> - **Backlog Task**：若已立项，填 T-NNNN；未立项填 `—`
> - **延迟原因**：为什么这次不做（范围 / 优先级 / 依赖未到 / 复杂度）
> - **触发再评估**：什么时候应该回头看（具体里程碑 / 条件）

| ID | 项 | 关联 | Backlog Task | 延迟原因 | 触发再评估 |
|---|---|---|---|---|---|
| L-01 | `device.Reboot` / `config.Sync` / `provision` / `upgrade` / `backup` 等模块的 CreateTask 调用点**未填 CreatorID**，订阅器跳过这些 task，消息中心**收不到**这些操作的通知 | C5 / device_service / config / provision 等模块 | — | Phase 1 范围聚焦快速设置（C5 已补 `device.SetParameters` + C7 补 `AddObject` / `DeleteObject`）；其他模块改造跨多领域代码量大 | T-0157 Phase 2 扩展接入（§7）；或前端用户报告"X 操作没消息"时单独补 |
| L-02 | `notification.created_at` 取 DB `NOW()` 而非 `task.created_at`，跨进程订阅链路延迟（通常 <100ms）会让用户视角的"触发那一刻"略偏后 | C5 / pg_repository Create/UpsertByDedup 签名 | — | 改 Create/UpsertByDedup 签名接受 `createdAt` 参数 + 30+ 调用方传值，工作量与 100ms 偏移收益不匹配 | 用户反馈消息时间戳与"我点保存的时刻"明显错位时 |
| L-03 | V1 文案**只显示新值无旧值**（如 `TAC = 4` 而非 `TAC: 3 → 4`），与 §4.5 文案规范的"旧值 → 新值"目标偏差 | C5 / task_subscriber.go renderNotifContent | — | 后端订阅器拿不到旧值（schema 在前端 Form 内存）；要么前端 mutation 多带 oldValue（侵入 API），要么后端读 device_parameters 表（多一次 DB 查询） | C10 实施 Tag 改造时一并评估前端发 oldValue 方案 |
| L-04 | 快速设置参数取值范围校验**仅支持 minValue/maxValue**（按类型解释：string=长度 / 数字=值），**不支持 enum + length pattern**：后端 `ParamMapping` struct 只有 MinValue/MaxValue 两列复用，未透传 `enum_values / max_length / min_length / pattern`；XML 里有 `<enumeration>` 等但 dictloader 没解析；枚举显示值 ≠ 下发值场景需要 paramModel XML 补充 `label / value` 配对字段 | parammodel/pg_handler_repo.go ParamMapping struct + dictloader 解析逻辑 + device_param_handler constraintsFromMapping + 部分 XML 模板补 enum 元素 | — | 工作量约 0.5d 后端 +XML 补充；当前业务场景以数字范围为主，枚举可暂用 string + pattern 兜底 | 用户报告"想填枚举/字符串长度/正则但前端不校验"或具体 XML 需求时 |




