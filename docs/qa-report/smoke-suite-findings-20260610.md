# 业务冒烟套件首轮执行发现的后端/前端问题清单（2026-06-10）

> 来源：按业务域拆分的冒烟套件首次实现与执行（`omcgo/scripts/smoke/` 25 个后端脚本 + `omcmb/webcode/e2e/smoke/` 17 个前端 spec），
> 目标栈为本机容器栈（app `:18091` / acs `:7557`）。每条均含可复现端点与代码级疑因，可直接转 GitHub Issue（`bug` + `needs-triage`）。
> 冒烟脚本中已用 `known_bug` 标注的条目，修复后会自动转回硬断言。

---

## 五轮「核实→补遗漏→修复→合入」修复总结（2026-06-11 更新）

冒烟套件落地后经五轮迭代（PR #128 套件 + #133/#144/#146/#147 修复），覆盖已扩展到
**后端 26 域 1737 断言（0 FAIL / 0 KNOWN_BUG）+ 前端 111 用例全绿**。本清单的 30 个问题
及五轮中新发现的问题，修复状态如下（GitHub issue 全程 `Refs` 不关单，留待人工核销）：

| Issue | 问题 | 状态 |
|------|------|------|
| #116 | worker metrics panic 循环 + 取消任务 500 | ✅ 已修（第1轮）|
| #117 | 设备搜索命中 NULL 坐标 500 | ✅ 已修 |
| #118 | pm/tasks creator NULL 500 | ✅ 已修 |
| #119 | 告警热力图 smallint=text 500 | ✅ 已修 |
| #120 | 分组排序 CASE 类型 500 | ✅ 已修 |
| #121 | 北向死信队列 503（未装配）| ✅ 已修 |
| #122 | admin 三类系统日志无写入方 | ✅ 已修（第6轮：登录日志 + OperLogger 中间件操作日志 + CompletionRouter observer 任务日志,live 实测三类均非空）|
| #123 | 告警定义 upload-xml 静默失败 | ✅ 已修 |
| #124 | ops_audit_logs target_type 超长 | ✅ 已修（迁移 000035）|
| #125 | 错误映射批次 10 项 500→4xx | ✅ 已修（第2轮）|
| #126 | 低危/一致性 10 项 | ✅ 已修 9 项（第7项=#116，trace=wontfix）|
| #127 | 前端 /transfer/center Tab 竞态 | ✅ 已修 |
| #134 | KPI 自定义公式校验全新栈失效 | ✅ 已修 |
| #135 | admin roles search 失效 | ✅ 已修 |
| #136 | 内置角色 operator/viewer 可删 | ✅ 已修（迁移 000036 + UUID 白名单）|
| #138 | UFTE suspended 表示不一致 | ✅ 已修 |
| #139/#143 | token 强制下线秒级边界 | ✅ 已修（now+严格小于）|
| #145 | 补全新发现 5 项错误映射 + find/refreshSource 补漏 | ✅ 已修（第3/4轮）|
| #137 | 测试栈 role_menus 恢复 | ⏸️ 运维待办（恢复 SQL 留人工执行）|
| #140 | 前端 plug-and-play 纯 mock 未接线 | ⚠️ 部分（第6轮：执行任务列表接真实 provision API list/create/retry;策略管理因后端无 /policies 端点暂留前端配置态）|

**累计修复 30+ 真实后端/前端缺陷**；所有冒烟脚本的 `known_bug` 探针均已 flip 为硬断言
（后端 known_bug 计数清零）。trace `payload_size_bytes` 口径偏差经核实为有意设计（wontfix）。

---

## 一、高优先级（功能不可用 / 进程崩溃 / 静默数据丢失）

### 1. `GET /api/v1/devices/search?keyword=SM`

- **发现套件**：F06 设备管理
- **现象**：HTTP 500, msg=scan search result: can't scan into dest[5] (col: latitude): cannot scan NULL into *float64
- **疑因与修复建议**：internal/device/search.go repository 扫描行把 latitude/longitude 扫进非空 float64 目标，坐标为 NULL 的设备一旦命中关键字即整个搜索 500；应改用 *float64/pgtype 可空扫描

### 2. `PUT /api/v1/device-groups/sort`

- **发现套件**：拓扑与设备分组(F06)
- **现象**：任何合法 payload 均 500，msg=column "sort_order" is of type integer but expression is of type text (SQLSTATE 42804)
- **疑因与修复建议**：internal/topology/pg_repository.go::BatchSort 构造 UPDATE ... SET sort_order = CASE WHEN id=$1 THEN $2 ... END 时绑定参数未显式 ::int 转型，PG 把 CASE 分支推断为 text。修复：THEN $N::int。确定性后端 bug，建议建 issue

### 3. `POST /api/v1/alarm-definitions/upload-xml`

- **发现套件**：F04 告警管理
- **现象**：上传 neType=SMKsmk178109434922378（21 字符）的合法 alarm XML → HTTP 201 且响应 {reloaded:true, orphans_deleted:0}，文件落盘成功，但 GET /alarm-definitions?ne_type=... total=0，定义 0 行入库；改用 ≤16 字符 neType 后同一流程正常入库
- **疑因与修复建议**：alarm_definitions.ne_type 为 varchar(16)，Loader 重载 UPSERT 因超长整文件事务失败，但 ReloadOne 错误在 upload handler 里只 Warn 不致命且 reloaded 仍报 true（或按文件粒度吞错），形成『导入成功但库里查不到』的静默失败。建议上传守门链增加 neType 长度校验（≤16 → 400）或把 reload 失败如实回传 data.reloaded=false

### 4. `GET /api/v1/pm/tasks`

- **发现套件**：F03 性能管理
- **现象**：HTTP 500，msg="scan pm_tasks row: can't scan into dest[9] (col: creator): cannot scan NULL into *string"
- **疑因与修复建议**：omcgo/internal/pm/pg_task_repository.go List 扫描 creator 列用 *string 接 NULL（应改 **string/pgtype.Text 或 SQL COALESCE），repository 行扫描与表数据不匹配；已用 known_bug 标注不计 FAIL

### 5. `GET /api/v1/dashboard/alarm-heatmap-by-severity`

- **发现套件**：F06 仪表盘
- **现象**：任何调用（带或不带 severity 参数）均 HTTP 500，msg=query alarm heatmap by severity: ERROR: operator does not exist: smallint = text (SQLSTATE 42883)
- **疑因与修复建议**：omcgo/internal/dashboard/heatmap.go queryHeatmapBySeverityMap 的 SQL `($2 = '' OR severity = $2)`：alarms_history.severity 列是 smallint（数值 1-4），与 text 参数直接比较；参数类型推断冲突导致语句 prepare 即失败。需把 severity 名转数值或 SQL 里 cast（如 severity::text = $2 / 用数值映射）

### 6. `POST /api/v1/ops/maintenance-windows（审计副作用 → GET /api/v1/ops/audit-logs）`

- **发现套件**：F06 运维工具
- **现象**：创建窗口 201 成功，但 ops_audit_logs 0 行；app 日志 warn logger=ops.audit：insert audit_log: ERROR: value too long for type character varying(16) (SQLSTATE 22001)
- **疑因与修复建议**：migrations/000001_init_schema.sql ops_audit_logs.target_type varchar(16) 容不下代码写入的 'maintenance_window'（18 字符，internal/ops/service_ext.go:306）；AuditLogService.Log 仅 Warn 不回传错误 → 维护窗口创建/审批审计留痕全部静默丢失，合规审计链路断

### 7. `worker 进程 internal/task/service.go:485 (TaskService.ExpireTask，另 405/448/558 同病)`

- **发现套件**：F06 报表
- **现象**：omc-worker-1 反复 panic 退出：panic: inconsistent label cardinality: expected 2 label values but got 1 in []string{"expired"}，由 ExpiredSweeper 每 10s 扫到 expired 候选任务时触发，worker Exit(2)
- **疑因与修复建议**：metrics.go:64 定义 CompletedTotal 为 2 label（source,status，completion_router.go:81 用法正确），但 service.go 4 处只传 1 个 label 值（success/failed/expired）。任何任务过期/完成走这些路径都会 panic 整个 worker，连带报表异步生成、PM/MR 处理全部停摆。修法：补 source 标签或统一改单 label。

### 8. `GET /api/v1/admin/logs/{login,operation,task}`

- **发现套件**：F06 系统配置
- **现象**：成功登录后 /admin/logs/login 仍为 0 条（total=0）；操作/任务日志同样恒空
- **疑因与修复建议**：三个日志端点是只读视图但无写入方：CreateLoginLog/CreateOperLog/CreateTaskLog 在全仓零调用（仅 internal/admin/sys_log.go 的接口+实现），auth 登录流只写 audit_logs，sys_login_logs/sys_oper_logs/sys_task_logs 三表永远为空——功能孤儿，建议建 issue 接通写入链路或下线端点

### 9. `GET /api/v1/northbound/push/deadletter`

- **发现套件**：F08 北向接口
- **现象**：恒返回 503 {ret:0, msg:"outbox not configured"}；POST /push/deadletter/:id/replay 同样恒 503
- **疑因与修复建议**：northbound.Router 提供 SetOutboxRepo 注入点且 push.Engine 也有 SetOutboxRepo，但 cmd/app/provider/modules.go 的 initNorthboundModule 从未构造/注入 OutboxRepository，死信队列两个端点在任何环境都不可用——疑似装配遗漏（outbox 投递模式写了但没接线）

## 二、错误映射类（业务错误落 500 而非 4xx，同一修复模式）

> 共同根因：service 层未把「未找到/业务校验拒绝」包装成 sentinel/BusinessError，或 handler 未走 `HTTPStatusFromError` 映射。建议作为一个批次修复。

### 10. `POST /api/v1/auth/switch-role`

- **发现套件**：认证与会话
- **现象**：role_id 为未分配/不存在角色时返回 HTTP 500 + ret=0（biz_code 7003 "target role not assigned to user"）
- **疑因与修复建议**：业务校验拒绝应返回 4xx（403/400）而非 500；omcgo/internal/admin/auth_handler.go SwitchRole 对 service 错误一律 AbortWithError(http.StatusInternalServerError, err)，未走 HTTPStatusFromError 映射（BusinessError 包了 ErrForbidden 本可映射 403）

### 11. `POST /api/v1/device-groups/move-devices`

- **发现套件**：拓扑与设备分组(F06)
- **现象**：target_group_id 传非 UUID 字符串时返回 HTTP 500 ret=0（业务错误 ErrCodeGroupParentInvalid 未映射为 400）
- **疑因与修复建议**：service.MoveDevices 返回的 NewBusinessError(ErrCodeGroupParentInvalid) 经 HTTPStatusFromError 落到 500，参数校验类错误应为 400；轻微状态码语义问题，不影响功能

### 12. `GET /api/v1/products/match?productClass=<未命中任何 pattern 的值>`

- **发现套件**：F02 产品与参数字典
- **现象**：HTTP 500 ret=0，msg="productClass matched no pattern (orphan device)"（应返回 200 + matched=false）
- **疑因与修复建议**：internal/product/handler.go Match(L683-687) 把 registry.MatchProductClass 的所有 err 一律按 500 抛；而 registry.go:238 未命中时返回 sentinel ErrOrphan（非 nil,nil），导致 handler 里 mr==nil 的 matched=false 分支永不可达。修复应 errors.Is(err, ErrOrphan) → response.OK(matched:false)。脚本中已用 known_bug 标注（兼容修复后自动转 pass）

### 13. `POST /api/v1/backup/restore`

- **发现套件**：F06 配置备份
- **现象**：源对象不存在时返回 HTTP 500（msg=compute restore source md5 路径），而非代码注释与 handler 设计的 404
- **疑因与修复建议**：restore_service.go Create 中 s.stater 似未注入（nil 跳过 StatObject 存在性预检），缺失对象直落 computeSourceMD5 的 GetObject 错误 → 不匹配 ErrNotFound → default 500；DI 装配遗漏 SetMinIOStater 之类的 hook。断言用 check_ret_fail 仍通过（确实被拒绝），但状态码语义错误

### 14. `POST /api/v1/mml/admin/groups`

- **发现套件**：F06 MML 控制台
- **现象**：body 传不存在的 param_version 时返回 HTTP 500，msg 直接泄漏原始 SQL 错误：create group: insert group: ERROR: insert or update on table "mml_command_groups" violates foreign key constraint "mml_param_groups_param_version_fkey" (SQLSTATE 23503)
- **疑因与修复建议**：FK violation 应在 service/repo 层翻译为 400/422 业务错误（param_version 不存在），而非 500 + 裸 SQLSTATE 外泄

### 15. `POST /api/v1/mml/groups/:id/execute`

- **发现套件**：F06 MML 控制台
- **现象**：对不存在的 group UUID（全零）返回 HTTP 500，msg="group_id required"（语义错乱：参数其实传了，是分组查不到）
- **疑因与修复建议**：应返回 404 group not found；当前错误映射把 not-found 落入 500 且文案误导为缺参

### 16. `DELETE /api/v1/station-logs/:id 与 DELETE /api/v1/device-abnormal-reboots/:id`

- **发现套件**：F06 日志类
- **现象**：对不存在但格式合法的 UUID 返回 HTTP 500（service.Delete 把 "log file not found" 包成普通 error，handler 统一按 InternalServerError 抛出）
- **疑因与修复建议**：应返回 404：internal/stationlog/service.go Delete/DownloadURL 在 f==nil 时返回 fmt.Errorf 而非 sentinel NotFound，handler 无法区分未找到与内部错误（对照同模块 GetAbnormalReboot 已正确返回 404）

### 17. `DELETE /api/v1/devices/tasks/:task_id（不存在 ID）`

- **发现套件**：统一任务队列
- **现象**：返回 HTTP 500 internal server error 而非 404
- **疑因与修复建议**：handler.go CancelTask 用 err.Error()=="task not found" 精确字符串比对，但 service.CancelTask 返回 fmt.Errorf("task not found: %s", taskID) 带 id 后缀，永远匹配不上，落入 500 分支；应改用 sentinel error + errors.Is

### 18. `POST /api/v1/interop/run（及 /run/:category、/run/report 同路径）`

- **发现套件**：F10 互操作测试
- **现象**：device_sn 不存在时返回 HTTP 500 ret=0，而非 404
- **疑因与修复建议**：omcgo/internal/interop/runner.go RunAll/RunByCategory 中 device not found 用裸 fmt.Errorf("device not found: %s")，未 wrap commonerrors.ErrNotFound，HTTPStatusFromError 落 default 500

### 19. `POST /api/v1/interop/validate/:deviceId`

- **发现套件**：F10 互操作测试
- **现象**：deviceId 为合法 UUID 但设备不存在时返回 HTTP 500 ret=0（错误信息为 'device productClass missing'，语义误导），而非 404
- **疑因与修复建议**：omcgo/internal/interop/validator.go ValidateDevice：deviceRepo.GetByID 未命中返回 (nil,nil) 后未判 nil 直接进 resolveExpectedParams，报裸错误 → 默认 500；应先判 dev==nil 返回 ErrNotFound

## 三、低危语义/一致性问题

### 20. `GET /api/v1/trace/tasks/:id/messages/:msgId/payload`

- **发现套件**：F01 南向 TR069
- **现象**：返回体 918 字节，而该报文 payload_size_bytes 记录为 721 字节
- **疑因与修复建议**：payload_size_bytes 统计口径与实际返回体不一致（疑似按截断/压缩前后不同口径计数），功能正常，仅口径偏差，低优先级

### 21. `POST /api/v1/device-registrations (不带 group_id)`

- **发现套件**：F06 设备管理
- **现象**：HTTP 500, msg=create registration: insert registration: ERROR: null value in column "group_id" of relation "device_registrations" violates not-null constraint (SQLSTATE 23502)
- **疑因与修复建议**：CreateRegistrationRequest.GroupID 无 binding:required，但 migrations/000001 中 device_registrations.group_id 为 NOT NULL——handler 契约与 DB 约束不一致，缺参应 400 参数校验而非落库炸 500

### 22. `POST /api/v1/config/sync/push/:deviceId（pull 同）`

- **发现套件**：F02 配置管理
- **现象**：代码审阅（omcgo/internal/config/sync_handler.go）：handler 不校验设备存在性，直接 CreateTask(DeviceSN=路径参数)；合法 body + 任意不存在 SN 推测会 200 入队产生孤儿任务（未实测以免污染任务队列）
- **疑因与修复建议**：缺设备存在性预检，建议入队前查 device 表返回 404

### 23. `GET /api/v1/pm/counters/aggregated`

- **发现套件**：F03 性能管理
- **现象**：不带 page/page_size 时 400（Field validation for 'Page' failed on the 'min' tag）
- **疑因与修复建议**：轻微不一致：ListAggregatedCounters 绑定 counterQuery 前未像同文件 ListCounters 那样预填 DefaultListRequest，且该查询根本不用分页字段；建议补 q.ListRequest = model.DefaultListRequest()。冒烟已显式带参规避

### 24. `GET /api/v1/mr/tasks/:id/progress`

- **发现套件**：F05 测量报告
- **现象**：对不存在的 task_id（00000000-0000-0000-0000-00000000dead）返回 200 ret=1 空列表，而详情 GET /mr/tasks/:id 同 ID 返回 404
- **疑因与修复建议**：低危/语义不一致：ListProgress 未先校验任务存在（omcgo/internal/mr/task/handler.go ListProgress 直接按 TaskID 过滤查询），与 Get/Stop/Delete 的 404 行为不一致，建议补任务存在性校验或确认为有意设计

### 25. `GET/POST /api/v1/ops/{diagnostics,downloads,audit-logs,maintenance-windows,playbooks,break-glass,commands}`

- **发现套件**：F06 运维工具
- **现象**：成功响应为裸 JSON（如 {items,total,page,...}、{active:false}、201 直出窗口对象），无统一 {ret,msg,data} 信封；同域 templates/tasks/command-records 走信封
- **疑因与修复建议**：internal/ops/handler_ext.go 成功路径用 c.JSON 直出而非 response.OK，与项目统一信封约定（internal/core/response）漂移，前端/脚本需对 ops 扩展端点特判

### 26. `DELETE /api/v1/devices/tasks/:task_id`

- **发现套件**：统一任务队列
- **现象**：取消成功（Redis 删除+PG 状态置 cancelled）但响应 HTTP 500：panic recovered 'inconsistent label cardinality: expected 2 label values but got 1'，栈指向 internal/task/service.go:558 CancelTask 调 metrics.CompletedTotal.WithLabelValues("expired")，而 metrics.go 定义该 CounterVec 为 {source,status} 两个 label
- **疑因与修复建议**：CancelTask 的 metrics 调用少传 source label 触发 prometheus panic；且语义上取消计入 'expired' label 也不对，应为 ('api 等 source','cancelled')。前端会把成功取消当失败展示

### 27. `POST /api/v1/devices/tasks/:task_id/retry`

- **发现套件**：统一任务队列
- **现象**：对已取消（cancelled）任务 retry 返回 200 并重新入队回 pending；且创建任务传 max_retries=0 被静默改为默认 3
- **疑因与修复建议**：model.go CanRetry() 只判断 retry_count<max_retries 不校验任务状态，cancelled/completed 任务可被 retry 复活；NewTask 仅在 req.MaxRetries>0 时采纳调用方值，0（禁止重试）无法表达。建议 CanRetry 限定 failed/expired 状态

### 28. `GET /api/v1/events/stream`

- **发现套件**：通知中心
- **现象**：建连后服务端不 flush 响应头：curl 4s 内 0 字节（http_code=000、rc=28），首字节最早为 30s keepalive 注释；另观测到 ~18s hub 侧关连接返回 200 空体
- **疑因与修复建议**：internal/events/handler.go Stream 在 Subscribe 成功后应立即 WriteHeader(200)+Flush（或先发一条 :connected 注释）；否则浏览器 EventSource onopen 延迟最长 30s，且中间代理/LB 可能按首字节超时切断 SSE 长连接

### 29. `POST /api/v1/provisioning/tasks`

- **发现套件**：F09 自动开站
- **现象**：body 传格式合法但不存在的 device_id（aaaaaaaa-bbbb-cccc-dddd-eeeeffff0000）返回 201 ret=1，成功建出 status=discovered 的孤儿任务
- **疑因与修复建议**：handler.Create 只做 UUID 格式解析（internal/provision/handler.go:89），未校验设备存在性即落库（provisioning_tasks 表对 device_id 无外键），应返回 404/400；孤儿任务最终被 15 分钟超时 reaper 置 failed

### 30. 前端 `/transfer/center` 默认 Tab 竞态

- **发现套件**：前端文件传输域冒烟（e2e/smoke/file.spec.ts）
- **现象**：首屏 `useUnifiedFileTransferTaskTypes` 未返回时 categories 只含虚拟项 [mr_measurement, kpi_export]，useEffect 把 selectedCategory 置为 'mr_measurement'；task-types 数据到达后不回退，页面默认落在「MR 测量」Tab，执行视图卡片被隐藏，而非预期的首个真实分类。
- **疑因与修复建议**：`omcmb/webcode/src/pages/transfer/FileTransferCenter/index.tsx` L707-715 useEffect 与 L176-193 categories useMemo（虚拟分类在竞态时成为 categories[0]）；冒烟用例已通过点击真实 Tab 绕过并在注释标注。
