# MML 功能完善

1. ~~mml-requirements-design.md 文档中，完整执行流程图，调整为：mml_tasks > device_tasks~~ **已完成**
   - 新增 `internal/mml/fanout.go`：将 MML task 的 commands × devices 扇出为 device_tasks
   - 新增 `internal/mml/result_aggregator.go`：device_tasks 完成后回写 mml_tasks 统计
   - 修改 `ExecuteCommand`：immediate 类型创建后自动扇出
   - 修改 `StartTask`：paused 恢复时扇出
   - 修改 `task/service.go`：添加 `TaskCompletionCallback` 回调机制
   - 修改 `task/model.go`：CreateTaskRequest 增加 CommandKey 字段
   - 修改 `task/pg_repository.go`：Create/BatchCreate/scan 支持 3 个新列

2. ~~ACS 服务中与基站的 RPC 任务，统一从 device_tasks 获取~~ **已完成**
   - ACS Handler 已优先使用 `task.TaskService`（device_tasks）获取任务
   - 新增 `internal/task/bridge_queue.go`：BridgeQueue 适配器，将旧 cmdqueue.CommandQueue 接口桥接到 device_tasks
   - 其他模块（backup/software/provision 等）可通过 BridgeQueue 透明切换

3. 针对 device_tasks 表需要考虑分表和历史任务迁移，确保性能要求

# TR069 协议与交互流程梳理 **已完成**

输出文档: `docs/design/tr069-protocol-and-interaction-flow.md`

1. ~~CPE 通过 tr069 协议与 ACS 建立链接、发送空报文~~ — 文档第 2 节
2. ~~ACS 获取设备 RPC 任务，将任务下行给 CPE~~ — 文档第 3 节
3. ~~CPE 完成 RPC 任务，将任务结果反馈给 ACS，ACS 更新任务结果~~ — 文档第 4 节
4. ~~ACS 与 APP、WORKER 的交互，消息队列和订阅的流程~~ — 文档第 5 节
5. ~~APP 后台 MML 模块(mml 命令和 mml 脚本)，批量添加设备执行命令（任务）> 拆分到设备任务表~~ — 文档第 6 节
6. ~~ACS 从设备任务中设备 SN 的任务，下行给 CPE~~ — 文档第 3、6 节
7. ~~总结整个项目的执行流向和数据流程，怎么通过消息订阅完整整个流程，以及项目中的消息订阅流程~~ — 文档第 7 节（58 个事件主题 + 8 个 NATS JetStream 流）
8. ~~分析整个项目的代码结构，包含代码的调用流程说明，每一层的职责~~ — 文档第 8 节

# MML BUG 修复

1. ~~2 台 LST BTSSTATE 是否重复~~ **不是 Bug** — 1 command × 2 devices = 2 device_tasks 是正确行为。截图中"重复"是输入区和任务历史各显示一次
2. ~~MML 页面命令树分类筛选框取消，显示全部分类，默认折叠~~ **已修复**
   - `CommandTree.tsx`: 移除 `<Select>` 筛选框，移除 `categoryOptions`/`categoryFilter`/`onFilterChange` props
   - `useCommandSelection.ts`: 移除 `categoryFilter` 状态和自动选中首个分类的 `useEffect`，API 查询不再传 `category` 参数
   - `index.tsx`: 移除传给 `<CommandTree>` 的三个分类相关 props
   - 一级分类默认折叠（`expandedKeys` 初始为空数组）
3. ~~POST /mml/execute 500 错误 (字段数 26 vs 27)~~ **已修复**
   - 根因: migration 000020 添加 `executor` 列到 `mml_tasks` 表，但 `taskColumns` 切片未同步
   - 修复: `pg_repository.go` 的 `taskColumns` 添加 `"executor"` 列（26→27，与 `scanTask` 匹配）
