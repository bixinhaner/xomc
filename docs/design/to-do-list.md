# MML BUG 修复 和功能完善

1. ~~1 1 台·LST ALMAF·LST 数字显示重复~~ **已修复**
   - `CommandInput.tsx:292`: 移除重复的 `{selectedDevices.length}` 输出，仅保留 `t('mml.console.deviceUnit', {count})` 渲染 "{count} 台"

2. ~~POST /mml/execute 500 错误 (result NULL scan)~~ **已修复**
   - 根因: `MMLTask.Result` 类型为 `TaskResult` (string)，新创建任务 result=NULL 导致 pgx 扫描失败
   - 修复: `model.go` 中 `Result TaskResult` → `Result *TaskResult` (指针类型)
   - 同步修改 `result_aggregator.go`: 赋值改为 `&finalResult`，SSE payload 解引用

3. ~~GET /mml/tasks 500 错误 (同 #2)~~ **已修复** — 同一个根因，Result 指针类型修复覆盖所有 scan 路径

4. ~~device_tasks 按 device_sn 分表~~ **已实施**
   - 新增 `migrations/000024_device_tasks_hash_partition.sql`
   - HASH 分区 on device_sn, MODULUS 16, 16 个分区 (p00-p15)
   - PK 改为 `(id, device_sn)` 复合主键
   - 保留所有现有索引 (status, created_at, cwmp_id, pending, parent, GIN)
   - 无 FK 约束阻塞 (已确认无外键引用)
   - 主路径 (按 device_sn 查询) 自动获得分区裁剪，PG fallback 路径仍可正常工作
