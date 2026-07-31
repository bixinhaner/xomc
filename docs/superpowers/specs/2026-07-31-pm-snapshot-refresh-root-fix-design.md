# PM 聚合快照空扫根治设计

## 背景与证据

生产环境 `172.24.224.197` 在版本 `100.0.0-20260731-2118` 上稳定运行，
但慢查询日志持续出现同一条查询：一次加载 31 个聚合版本的 336,186 条
`pm_aggregation_version_members`，耗时 1.15–1.31 秒。

代码取证确认有两个独立触发入口：

- worker 的 `SnapshotStore.RunRefresh` 每分钟无条件调用 `LoadMatchable`；
- app 的 `ProgressService.Query` 为每次 Dashboard 日/周进行中结果查询重新调用
  `LoadMatchable`。

两条路径都读取完整版本规则、Counter 和成员目录，即使聚合任务配置完全没有变化。
数据库没有锁等待，CPU、内存和磁盘即时指标也正常，因此根因不是资源不足或索引缺失，
而是应用层缺少变更感知缓存。

## 目标与验收标准

1. 配置未变化时，worker 分钟刷新不得再次扫描聚合版本成员。
2. 配置未变化时，Dashboard 日/周进行中查询不得再次扫描聚合版本成员。
3. 聚合任务新增、修改、删除、计划结束时间或当前版本变化后，下一次刷新必须读取新目录。
4. `pm.aggregation.task.version.changed` 事件仍强制立即刷新，不等待轮询。
5. 指纹查询失败时保留上一份可用快照并返回错误，不退化为昂贵全量查询。
6. 不支持修订指纹的测试/替代 Loader 保持原有全量刷新兼容行为。
7. 当前版本有效期、历史 45 天可匹配范围、日/周进行中结果和 12 分钟小时关窗语义不变。
8. 生产复验中同一成员全量查询在无配置变更的连续观察窗口内不再重复出现。

## 方案比较

### 方案 A：调大刷新间隔或慢查询阈值

改动最小，但只降低告警频率或隐藏日志；Dashboard 仍会按请求全量读取，配置新鲜度也被牺牲。
不采用。

### 方案 B：为成员查询增加或调整索引

当前查询已按 `task_version_id` 读取 336,186 条真实结果，主要成本是传输、扫描和构建对象，
不是找不到行。索引无法消除无意义读取。不采用。

### 方案 C：轻量修订指纹 + 进程内快照缓存

在主库通过 `pm_aggregation_tasks` 的行数、最大 `updated_at` 和全部任务行的稳定 MD5
指纹形成低成本、可比较的修订值。指纹避免“较早开始但较晚提交的事务写入较小时间戳”时
仅看最大时间会漏变更。任务保存、当前版本切换、计划结束和软删除都会更新该表；版本规则和成员
在同一事务内不可变写入，因此任务表提交点可以作为完整目录的发布水位。生产只读
`EXPLAIN ANALYZE` 显示 12 条任务行的修订查询执行约 0.6ms、26kB 排序内存。

worker 和 app 共用 `SnapshotStore` 的缓存语义：首次或修订变化时才调用 `LoadMatchable`；
未变化且没有跨越版本时间边界时直接复用已发布快照。采用此方案。

## 架构与数据流

### 修订读取契约

在现有 `MatchableLoader` 旁增加可选的 `MatchableRevisionLoader`：

```go
type MatchableRevision struct {
    TaskCount   int64
    UpdatedAt   time.Time
    Fingerprint string
}

type MatchableRevisionLoader interface {
    LoadMatchableRevision(context.Context) (MatchableRevision, error)
}
```

`PgTaskRepository` 实现该接口，使用 Squirrel + pgx 查询全部任务的 `COUNT(*)`、
`MAX(updated_at)` 和按任务 ID 排序的整行指纹。包含已删除任务，确保软删除也改变修订。

### SnapshotStore 状态机

`SnapshotStore` 增加互斥保护的源版本缓存、最近修订和初始化状态：

- `Reload`：显式强制全量加载；用于启动和版本变更事件；
- `Refresh`：用于轮询和读请求；先读取修订，未变化则不访问详情表；
- Loader 不实现修订接口时，`Refresh` 回退到 `Reload`，保持兼容；
- 全量加载前读取修订，并把该修订与加载结果绑定。若提交在加载期间发生，下一次
  `Refresh` 会看到不同修订并补一次加载，避免把旧数据错误标成最新；
- `Refresh` 和 `Reload` 共用同一互斥锁，避免并发首请求产生惊群式全量查询；
- 失败不覆盖原有原子快照。

快照发布时计算下一处时间边界：未来的 `effective_from`、`effective_to`，以及
`effective_to + 45 天` 的历史淘汰点。修订未变且当前时间未到该边界时不重建任何成员索引；
到达边界后只从缓存源版本重建一次，不访问数据库。重建使用版本结构的浅副本，避免
`BuildTaskSnapshot` 写入派生计数时修改仍被旧快照读取的源对象。这样同时保持生效切换和
45 天半开范围语义（恰到淘汰点即移除），并避免把数据库 I/O 成本转移为每分钟 CPU/GC 成本。

### worker 接入

`RunRefresh` 从每分钟调用 `Reload` 改为调用 `Refresh`。版本变化事件仍调用 `Reload`，
所以即时配置生效语义不变。

### app / Dashboard 接入

`ProgressService` 持有一个共享 `SnapshotStore`。每次 `Query` 调用 `Refresh` 后读取
`Current()`；同一 app 进程中的多个 Dashboard 请求共享缓存和并发锁。指标版本有效区间
从快照的真实任务版本生成，不再单独全量加载。app 启动时版本元数据回填使用的
`SnapshotStore` 直接注入 `ProgressService`，避免启动已全量加载后首个 Dashboard 请求
再重复一次。

## 错误处理

- 修订查询失败：包装上下文返回，保留上一份快照；Dashboard 按现有逻辑标记
  `progress_state=unavailable`，worker 记录刷新告警。
- 全量加载失败：不更新修订和快照，下一轮继续重试。
- 首次并发加载：只有一个调用执行数据库全量查询，其余等待后复用结果。
- 显式事件刷新：即使修订值碰巧相同也强制加载，避免事件语义被缓存短路。

## 测试与生产验证

单元测试先红后绿覆盖：

1. 首次 `Refresh` 全量加载一次；修订不变的后续刷新不再加载。
2. 修订变化触发且只触发一次全量加载，并发布新快照。
3. 修订查询或全量加载失败时旧快照不被覆盖。
4. 并发首次刷新只执行一次全量加载。
5. 不支持修订接口的 Loader 保持每次全量刷新。
6. `ProgressService` 使用共享快照而不是直接调用 Loader。

验证门禁：`go test ./internal/pm/stream -count=1`、`go test ./... -count=1`、
`go test -race ./internal/pm/stream -count=1`、`go vet ./...`、`git diff --check`。

部署后连续观察至少 3 个 worker 分钟刷新周期并执行多次 Dashboard 日/周查询；要求
成员全量查询最多只在部署后首次装载出现，之后无配置变化时不重复。随后复核服务健康、
ACS 503/会话拒绝、PM 两条队列、关键 KPI、14:00 关窗证据、日/周预览、数据库锁与资源指标。
