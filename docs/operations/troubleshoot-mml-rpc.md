# MML RPC 任务排查手册

> 适用：从 OMC 管理后台 → MML 控制台或 ScriptTask 下发的命令未拿到 RPC 结果。
> 目标：定位 MML → device_tasks → ACS → CPE → 回流 mml_tasks 这条链路上**任何一处停滞**的位置。
> 配套自动化脚本：`omcgo/scripts/diag_mml_task.sh`（一键扫所有 checkpoint）。
> 设计依据：[docs/design/to-do-list.md](../design/to-do-list.md)

---

## 0. 环境准备

### 0.1 部署形态与访问方式

OMC 全部依赖（PostgreSQL / Redis / NATS / MinIO / ACS / APP / Worker / Web）由 [deployments/docker/docker-compose.yml](../../deployments/docker/docker-compose.yml) 编排。所有诊断命令都基于 docker compose 执行，**宿主无需装 psql / redis-cli**，只要有 `docker` + `jq` 即可。

```bash
# 仓库根（任何子目录都能拿到）
REPO_ROOT=$(git rev-parse --show-toplevel)
COMPOSE="docker compose -f $REPO_ROOT/deployments/docker/docker-compose.yml"

# PostgreSQL — 通过 postgres 容器
psql_run() { $COMPOSE exec -T -e PGPASSWORD=omcgo123 postgres psql -U omcgo -d omcgo "$@"; }

# Redis — 通过 redis 容器
redis_run() { $COMPOSE exec -T redis redis-cli "$@"; }

# 日志拉取（docker-compose 也把容器 /run/logs 挂到宿主 $REPO_ROOT/run/logs，
# 如果文件存在可直接 grep；否则用 docker compose logs 抓取）
ACS_LOG=$REPO_ROOT/run/logs/acs/acs.log     # 文件挂载时直接用
APP_LOG=$REPO_ROOT/run/logs/app/app.log
# 也可实时抓取最近窗口：
# $COMPOSE logs --no-color --since 30m acs > /tmp/acs.log
# $COMPOSE logs --no-color --since 30m app > /tmp/app.log
```

> 后续章节示例若用 `psql -c "..."`，请理解为 `psql_run -c "..."`；
> `$REDIS XXX` 同理换成 `redis_run XXX`。
> 一键脚本 `omcgo/scripts/diag_mml_task.sh` 默认就是 docker 模式，直接调用即可。

### 0.2 输入参数

通常已知的两个值：

| 变量 | 含义 | 来源 |
|------|------|------|
| `SN` | 设备序列号 | MML 控制台执行时选的设备 |
| `MML` | mml_tasks.id（UUID） | 执行 API 返回 / mml_tasks 表查 |

```bash
SN=1202000240194DP0026
MML=00000000-0000-0000-0000-000000000000   # 改成实际的 UUID
```

### 0.3 工具

```bash
# psql / redis-cli 必备
# jq 用来过滤 zap json 日志
# xmllint 用来美化 SOAP body
brew install jq libxml2  # macOS
```

---

## 1. 总览：8 个 Checkpoint

每条 RPC 任务从产生到回流要走 8 步。任意一步卡住，下游都不会推进。

```
[1] mml_tasks 写入        ← MML execute API 入口
[2] device_tasks 派生     ← Fanouter（每设备 × 每命令一行）
[3] device_tasks 入 Redis ← TaskService.BatchCreateTasks
[4] CPE Inform            ← 设备主动连
[5] ACS 下发 SOAP         ← ACS pop task + MarkSent
[6] CPE 响应              ← CPE 回 POST 给 ACS
[7] ACS 标记 completed    ← MarkTaskCompleted/Failed
[8] MML 聚合器收尾        ← NATS → CompletionRouter → ResultAggregator
```

每一步对应一个 SQL/Redis/log 查询命令。下面按顺序给。

---

## 2. Checkpoint 1：`mml_tasks` 是否写入？

```sql
SELECT id, task_name, status, execute_type,
       array_length(device_sns, 1) AS device_count,
       jsonb_array_length(commands) AS cmd_count,
       created_at, started_at, finished_at,
       success_count, failed_count
FROM mml_tasks
WHERE id = '<MML>';
```

**判定**：
- 没行 → 执行 API 没落库（前端 / handler 异常）。看 APP 日志：`grep "mml task created" $APP_LOG`
- 有行，`status='pending'`，`started_at IS NULL` → 留意 [3]：可能 Fanouter 没触发
- 有行，`status='running'`，`success_count + failed_count < device_count * cmd_count` → 进入 [2]
- 有行，`status='completed'` 或 `'failed'` → 任务已收尾，问题在前端展示或 SSE

```bash
# 看 service 是否记下了创建日志
grep -E "mml task (created|fan-out completed)" $APP_LOG | jq 'select(.mml_task_id=="<MML>")'
```

---

## 3. Checkpoint 2：`device_tasks` 是否派生？

```sql
SELECT id, device_sn, source, source_id, method, status, sent_at,
       retry_count, cwmp_id,
       result IS NOT NULL AS has_result,
       params
FROM device_tasks
WHERE source = 'mml' AND source_id = '<MML>'
ORDER BY device_sn, command_index, device_index;
```

**判定**：
- 0 行 → Fanouter 没派生（**Q4 根因常见情形**）。看 APP 日志：
  ```bash
  grep -E "build tr069 params failed|skip command without rpc_method" $APP_LOG \
    | jq 'select(.mml_task_id=="<MML>")'
  ```
  常见原因：mml_command 没绑定 param_refs / rpc_method 解析失败 / formValues 全空。
- N 行 但 `params` 形态可疑（不含 `names` / `values` 等关键字段）→ 翻译层 bug：
  ```bash
  psql -t -A -c "SELECT params FROM device_tasks WHERE source_id='<MML>' LIMIT 1;" | jq
  ```
- N 行 形态正确 → 进入 [3]

**关键日志**（fanout.go 翻译完成后输出）：
```bash
grep "device_task params built" $APP_LOG | jq 'select(.mml_task_id=="<MML>")'
# 期望字段：has_names=true (LST/DSP) 或 has_values=true (MOD)
```

---

## 4. Checkpoint 3：device_tasks 是否进入 Redis 队列？

```bash
# 设备的 ACS 任务队列（Sorted Set，member=task_id，score=优先级）
$REDIS ZRANGE acs:taskq:$SN 0 -1 WITHSCORES

# 任务详情 Hash（应包含 method/params/status 等字段）
TASK_ID=$(psql -t -A -c "SELECT id FROM device_tasks WHERE source_id='<MML>' AND device_sn='$SN' LIMIT 1;")
$REDIS HGETALL acs:task:$TASK_ID
```

**判定**：
- ZRange 为空但 device_tasks 有 `status='pending'` → Redis 与 PG 不一致：
  ```bash
  # APP 启动会 RestorePendingQueues：
  grep "task recovered\|RestorePendingQueues" $APP_LOG
  ```
- ZRange 有 task_id，task Hash 有 data → 等 Inform 即可（[4]）
- ZRange 没 task_id，但 task Hash 有 data 且 status=sent → 已经派发完，进 [5]

---

## 5. Checkpoint 4：CPE 是否上来 Inform？

```bash
# 看心跳时间戳（每次 Inform 都会刷新）
$REDIS GET acs:heartbeat:$SN
$REDIS TTL acs:heartbeat:$SN  # 应小于 2× inform_interval

# 看 ACS 日志
grep "ACS Inform processing" $ACS_LOG | jq 'select(.device_sn=="'$SN'")' | tail -5
```

**判定**：
- `heartbeat` 不存在或 TTL 已过 → CPE 离线或网络隔离
  - 用 `tcpdump -i any -A 'tcp port 7547 and host <CPE_IP>'` 抓包确认
  - 看 ACS 是否收到任何 POST：`grep "ACS detected RPC method" $ACS_LOG`
- `heartbeat` 在更新但 ACS log 没下发 → 进入 [5] 看 PopTask 失败原因

---

## 6. Checkpoint 5：ACS 是否下发 SOAP？

**这是排查 Q4 根因最关键的一步**。新增的日志字段直接给出 SOAP 报文。

```bash
# 5.1 拿到 cwmp_id（ACS 把 task 派发后会写到 PG 与 Redis 映射）
psql -t -A -c "SELECT cwmp_id FROM device_tasks WHERE id='$TASK_ID';"
# 或：
$REDIS GET acs:cwmp2task:<反向查找时用>

# 5.2 找到 ACS 下发日志
grep "ACS sending RPC request from task" $ACS_LOG | jq 'select(.task_id=="'$TASK_ID'")'

# 5.3 提取并美化 SOAP body
grep "ACS sending RPC request from task" $ACS_LOG \
  | jq -r 'select(.task_id=="'$TASK_ID'") | .soap_body' \
  | xmllint --format -
```

**判定**（极重要）：

| 现象 | 结论 |
|------|------|
| 无任何下发日志 | ACS 进程没跑到 PopTask；看 [3] 队列是否真有 task；看 ACS 是否启动正常 |
| 下发日志中 `soap_body` 含 `<ParameterNames soap-enc:arrayType="xsd:string[0]"/>` | **Q4 根因**（params 没翻译成 names）。重新部署本次修复版本 |
| `soap_body` 含若干 `<string>Device.X.Y</string>` | 报文合法，进 [6] 看 CPE 响应 |
| 出现 `build RPC request from task` Error | 翻译层下游：`omcgo/internal/acs/rpc/dispatcher.go` BuildRequest 拒绝。看 error 字段 |

抓包对照（最权威）：
```bash
sudo tcpdump -i any -A -s 0 'tcp port 7547 and host <CPE_IP>' -w /tmp/cpe.pcap
# 然后 wireshark / tshark 看请求 body 是否与日志的 soap_body 一致
```

---

## 7. Checkpoint 6：CPE 是否回响应？

```bash
# CWMPID 来自 [5]
CWMP_ID=$(psql -t -A -c "SELECT cwmp_id FROM device_tasks WHERE id='$TASK_ID';")

# ACS 收到响应日志
grep -E "RPC response received|ACS received SOAP Fault" $ACS_LOG \
  | jq 'select(.cwmp_id=="'$CWMP_ID'")'

# Debug 级看完整 XML（生产可能没开 Debug）
grep "ACS received RPC response" $ACS_LOG | jq 'select(.cwmp_id=="'$CWMP_ID'") | .xml'
```

**判定**：
- 无任何匹配 → CPE 没回，或 Cookie 失效：
  ```bash
  grep "no valid session cookie for RPC response" $ACS_LOG
  # 抓包看 CPE 是否真的回了 POST
  ```
- 命中 `RPC response received` → 进 [7]
- 命中 `ACS received SOAP Fault` → CPE 拒绝了报文（**强信号是 Q4 根因**）：
  ```bash
  grep "task failed with SOAP fault" $ACS_LOG \
    | jq 'select(.task_id=="'$TASK_ID'")'
  # 字段：fault_code（9003/9005 等）、fault_msg
  ```

---

## 8. Checkpoint 7：ACS 是否标记任务完成？

```sql
SELECT id, status, sent_at, completed_at, error_code, error_message,
       result IS NOT NULL AS has_result,
       result
FROM device_tasks
WHERE id = '<TASK_ID>';
```

**判定**：
- `status='completed'` + `result IS NOT NULL` → 进 [8]
- `status='failed'` + `error_code/error_message` → CPE 报错，看 error_message
- `status='sent'` 持续不动 → CPE 没回，或 cwmp_id 没匹配上：
  ```bash
  grep -E "get task by cwmp_id|no task found for cwmp_id" $ACS_LOG \
    | jq 'select(.cwmp_id=="'$CWMP_ID'")'
  ```

ACS 日志：
```bash
grep -E "task completed|task failed with SOAP fault" $ACS_LOG \
  | jq 'select(.task_id=="'$TASK_ID'")'
```

---

## 9. Checkpoint 8：MML 聚合器是否收尾？

```bash
# 启动期订阅日志（一次性）
grep "task completion event bridge subscribed" $APP_LOG | jq

# 失败兜底
grep "no completion handler registered for task source" $APP_LOG \
  | jq 'select(.source=="mml")'

# 聚合器最终化
grep "mml task finalized" $APP_LOG | jq 'select(.mml_task_id=="<MML>")'
```

**判定**：
- 无 `task completion event bridge subscribed` → APP 启动时 NATS 没就绪，桥接没装：
  ```bash
  grep "subscribe task completion bridge" $APP_LOG  # 看 error
  ```
- 命中 `no completion handler registered` 且 `source=mml` → `cmd/app/provider/modules.go:272` 的 `Register` 没调
- 看 NATS 直接订阅（实时观察）：
  ```bash
  nats sub 'task.>'
  # 重新执行命令，看是否有 task.completed / task.failed
  ```
- 命中 `mml task finalized` 但前端没刷新 → SSE 通道问题：
  ```bash
  grep "mml_task_completed" $APP_LOG | jq 'select(.mml_task_id=="<MML>")'
  # 浏览器开 Network 看 /api/v1/sse/messages 是否在线
  ```

---

## 10. 决策树（按现象快速跳转）

```
device_tasks.status = ?
├─ pending           → [3] Redis 队列 → [4] CPE Inform
├─ sent / sent_at有  → [5] ACS 下发日志 + soap_body
│                       ├─ soap_body 含空 ParameterNames → Q4 根因
│                       └─ soap_body 合法 → [6] CPE 响应抓包
├─ failed / fault    → [7] error_message + [6] SOAP Fault
└─ completed         → [8] mml task finalized + SSE
```

---

## 11. 一键自动化

所有命令都包在 [omcgo/scripts/diag_mml_task.sh](../../omcgo/scripts/diag_mml_task.sh) 里。**默认 docker 模式**，宿主只需 `docker` + `jq`（用全角宽度对齐表格时另需 `python3`，多数 Mac/Linux 自带）。

### 最简调用

```bash
# 自动取最新一条 mml_task + 它的首个 device_sn
bash omcgo/scripts/diag_mml_task.sh --auto

# 显式指定
bash omcgo/scripts/diag_mml_task.sh \
  --sn 1202000240194DP0026 \
  --mml-task 6e58a13a-4b6d-4c5b-9e9d-83e7a4b1c2f0
```

### 输出说明

```
  Step | Status | Checkpoint             | Detail
  -----+--------+------------------------+---------------------------------
  [1]  |   ✓    | mml_tasks 写入         | status=running ...
  [2]  |   ✓    | device_tasks 派生      | total=1 ... shape=names(7)
  [3]  |   ✓    | Redis 队列             | 队列已派发；status=sent
  [4]  |   ✓    | CPE Inform             | heartbeat=... ttl=240s
  [5]  |   !    | ACS 下发 SOAP          | ParameterNames 为空（Q4 根因！）
  [6]  |   ─    | CPE 响应               | 上游未通过
  [7]  |   ✗    | ACS 标记完成           | device_task 持续 status=sent
  [8]  |   ─    | MML 聚合器收尾         | 上游未到收尾阶段

下一步建议：部署 BuildTR069Params 修复版；或检查 mml_command 的 param_refs 是否绑定
```

| 符号 | 含义 |
|------|------|
| ✓ 绿 | 通过 |
| ! 黄 | 通过但有可疑（如 SOAP 形态不对、CPE 返回 Fault） |
| ✗ 红 | 卡住，看 detail + 下一步建议 |
| ─ 灰 | 上游未通过，跳过 |

### 常用选项

```bash
# 看上一条任务（最新的下一条）
bash omcgo/scripts/diag_mml_task.sh --auto --offset 1

# 给定 SN 找含它的最新 mml_task
bash omcgo/scripts/diag_mml_task.sh --auto --sn TEST-SN-001

# 拉更长的日志窗口（默认 30m，发现 ACS 没下发记录时常需要）
bash omcgo/scripts/diag_mml_task.sh --auto --logs-since 6h

# 打印实际下发给 CPE 的 SOAP 报文（自动识别 xmllint 美化）
bash omcgo/scripts/diag_mml_task.sh --auto --show-soap

# RPC 报文装配链深度分析（CPE 不响应、报文形态可疑等场景必开）
# 输出 D1-D5：用户意图层 / 协议装配层 / 路径合规性 / SOAP 结构 / CPE 拒绝原因清单
bash omcgo/scripts/diag_mml_task.sh --auto --diagnose-rpc

# 三连套（强烈建议遇到 status=sent 长期不动时直接用）
bash omcgo/scripts/diag_mml_task.sh --auto --logs-since 2h --show-soap --diagnose-rpc

# JSON 输出供程序消费（CI / 监控告警）
bash omcgo/scripts/diag_mml_task.sh --auto --json | jq

# 服务名定制（项目名不是 docker 时容器名前缀不同；或多套环境）
bash omcgo/scripts/diag_mml_task.sh --auto \
  --postgres-svc postgres-prod --redis-svc redis-prod \
  --acs-svc acs-blue --app-svc app-blue

# 切到 host 模式（宿主直连 5432/6379 端口，需要本地装 psql/redis-cli）
bash omcgo/scripts/diag_mml_task.sh --auto --mode host
```

### 退出码

| Code | 含义 |
|------|------|
| 0 | 链路完整推进到 [8]（任务已收尾） |
| 1 | 中途卡住（最后一个 ✓ 之后停滞） |
| 2 | 参数错 / 工具不可用 / 自动检测未找到 mml_task |

完整选项见 `bash omcgo/scripts/diag_mml_task.sh --help`。
