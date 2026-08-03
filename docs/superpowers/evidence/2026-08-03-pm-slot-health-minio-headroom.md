# PM 槽位健康与资源根治验收证据（2026-08-03）

## 数据清理边界

- 本轮最终清理严格限定为 OMC：仅停止 Docker Compose project `omcgo`，仅删除带 `com.docker.compose.project=omcgo` 标签的监控卷，并清空 `/home/omc-data/{postgres,timescaledb,redis,redis-pm,nats,minio}`、`/opt/omc/{data,run}`。
- 删除前逐项校验上述绝对路径，删除后逐项确认目录条目数为 0；未使用未解析变量、通配符、Docker 全局 prune 或宽目录递归删除。
- `/opt/omc/etc` 的凭据与资源配置保留；系统数据、其他 Compose project、其他容器、其他目录和其他业务数据均未触碰，也未进行数据迁移。
- 清理后安装 `100.0.0-20260803-2353`，由基线迁移重新创建 OMC 主库、时序库和种子数据。

## PM 数据链路

- 服务器：`172.24.224.197`，时区 `Asia/Shanghai`。
- 16:00 槽位文件：20,000；采集时间 15:59:59.907978–16:02:31.442315。
- 16:15 槽位文件：20,000；采集时间 16:14:59.925378–16:17:48.894036。
- 16:27 前 `pm_slot_health` 无 16:00 槽位记录，证明没有提前关闭。
- 16:00–16:15 槽位于 16:27:12.421099 评估：LTE/CMCC，expected=20,000，received=20,000，coverage=1，status=complete。
- K900010006 与 K900010076 均已有 60,000 条，最新指标时间为 16:00。

## 参数同步与数据库

- 16:26 主库快照：request succeeded=28,754、running=2,037、queued=16,729；run succeeded=28,754、executing=1,816、waiting_device=221。
- device task completed=299,054、pending=30,302、sent=1,852；outbox delivered=359,367、pending=627。
- `dead_letters=0`、`alarm_webhook_dead_letters=0`、锁等待=0。
- idle-in-transaction 观测均为 10–26ms 的事务边界瞬时状态，不存在长期 idle transaction。
- 自动准入在 2,048 runs / 55,296 tasks 上限内工作；`AUTOMATIC_BACKOFF` 是受控自动请求退避，不是 ACS 503。

## 资源快照

- 16:26 高峰时主库约 9.2 CPU、app 约 6.3 CPU，属于 20,000 设备初始参数同步的有效负载。
- 主机 iowait 1–2%，未出现磁盘饱和；`/home` 使用率 29%，可用 550GiB。
- OMC 数据占用：PostgreSQL 9.9GiB、TimescaleDB 3.7GiB、Redis core 917MiB、Redis PM 665MiB、NATS 64MiB、MinIO 944MiB。
- PostgreSQL 主库内存 5.934/7GiB，Redis core 1.78/3GiB，MinIO 776.8MiB/6GiB，均未触顶；无 OOM、无容器重启。

## 自动化验证

- `go test ./... -count=1`：完整通过（允许测试监听临时回环端口）。
- 参数同步 PostgreSQL 集成测试：验证按单 run durable progress 计算阻塞 idle age。
- Prometheus：9 个规则文件全部通过 `promtool check rules`；全部 `.test` 通过 `promtool test rules`。
- 独立代码复审发现并修复两项问题：单个异常参数同步 run/task 不再中断整批恢复，task budget 按 attempted 计数；配置升级原子替换保持原文件 mode/ownership。后续复审又发现候选优先级与游标混用可能导致跳跃或饿死，最终改为 priority 10 + forward 40 独立配额、固定 sweepEnd 上界，且仅 forward 推进游标。针对性测试、完整 Go 测试与最终复核均通过。

## 最终部署与聚合验收

- 最终版本：`100.0.0-20260803-2045`；离线包 SHA-256：`92aca09bfb232e0fc194fa2d6b171aeb54db6172ce1916dcdbf2c88a762d06fb`。
- 全新安装未迁移任何 OMC 业务数据；设备重新接入后为 20,000/20,000 在线。
- 参数同步缺失结果恢复配置已生效：`recovery_run_limit=200`、单 run task limit=200、单轮 task budget=200；旧默认值 20 的升级迁移、自定义值保留和幂等均有回归测试。
- 32 核压测资源方案已重新生效：app 10C/1.5GiB、ACS 双实例各 5C/4GiB、worker 8C/2GiB、PostgreSQL 12C/7GiB、TSDB 16C/7GiB、Redis Core 2C/3GiB、Redis PM 2C/8GiB、NATS 1C/1GiB、MinIO 4C/6GiB。
- 独立部署健康检查：104 项通过、0 项失败；最终资源重建后 app/ACS/worker/两库均 restart=0、OOM=false。
- 最终包以普通升级方式部署，保留清理后重新产生的 OMC 数据；安装器内置健康检查在启动高峰中单轮超时，随后独立完整健康检查 104/104 通过。
- 清库后的启动洪峰中，主库和 app 持续处理 20,000 设备重注册及首次参数同步；主机 iowait 0–1%、无数据库阻塞锁、ACS 当前实例 503/准入拒绝=0。
- 19:29–19:39：device task pending 46,046→28,269，outbox pending 19,984→1,165；缺失结果由维护器持续收敛，`param_sync_run_counter_drift=0`、活跃 run 的 `terminal_task_missing_result=0`。
- 线上进一步定位并根治主库 CPU 放大器：参数同步收敛候选查询原先为挑选 50 个 run 先横向统计全部活跃 run 的任务/结果，单次 4.8–6.7 秒；改为只扫描活跃 run 元数据、权威计数仍限于选中 50 个 run。相同现网数据 `EXPLAIN ANALYZE` 为 20.8ms；最终 2045 部署后原慢查询 hash `4387aa1211c4` 未再出现，候选扫描同时具备固定高水位和可证明的有限收尾。
- 清理后聚合任务版本从 20:00 生效，因此 20:00 前槽位按版本有效区间正确排除；20:00–20:15 首个槽位于 20:27:56 评估，LTE/CMCC expected=20,000、received=20,000、coverage=1、status=complete，没有提前关闭。
- 2045 部署后 20:30 槽位于 20:57:55 评估，LTE/CMCC expected=20,000、received=20,000、coverage=1、status=complete；K900010006、K900010076 各 124,404 行，最新指标时间均为 20:30。
- 真实浏览器验证首页：版本 `100.0.0-20260803-2011` 时已正确显示最新 PM 槽位 20,000/20,000、100%；页面只请求 `dashboard/kpi-time-series` 的 hourly 聚合接口；12:05:46 与 12:10:46 UTC 恰好间隔 5 分钟，两次均 200，耗时 22ms/7ms，中间没有 KPI 事件即时刷新。最终 2045 页面复核同样显示 20,000/20,000、100%，小时图有聚合数据；天显示进行中 1/24（4.2%），周显示进行中 1/168（0.6%）。
- 最终完整 `go test ./... -count=1`（含 e2e/integration）通过；发布门禁 281/281、16/16 等全部通过；独立复审确认两项意见均完整修复且无新增阻断项。
- 2045 独立健康检查 104/104，通过后主机可用内存约 19GiB、iowait 1%；初始参数同步队列继续下降（device task pending 9,881→5,656，executing run 889→547），ACS 两实例准入拒绝/限流拒绝/PM 背压拒绝均为 0。
- 最终参数同步 run 为 succeeded=40,022、failed=49，executing/waiting 均为 0；device task pending=0、sent=155（正常在途周期任务），主库 active>5s=0、lock waiter=0，业务死信与告警 webhook 死信均为 0。所有核心容器 restart=0、OOM=false。
- 20:00 小时窗口先于 21:05 完成不可见的 revision 预计算，但 `pm_aggregation_publications` 保持 revision=0/preparing；21:12:29 才原子切换 revision=1/published，严格没有在 12 分钟保护期前对 Dashboard 可见。
- 发布窗口 received_slots=4、expected_slots=4、version_slice_complete=true、period_complete=true。20,000 条 hourly rollup 事件随后由 16 路消费者全部消费，pending=0、ack_pending=0、redelivered=0；当前日/周进行中结果分别为 1/24 和 1/168。

## Redis 终态压缩与本轮干净部署复验

- 根因是 `acs:task:*` 完成记录长期按完整参数和结果保存在 Redis Core；2 万设备启动洪峰下单条平均约 9.8KiB，最终把 `noeviction` 实例推到上限并造成 ACS `OOM command not allowed`。现在仅在 PostgreSQL 已同步且补偿标记全部清零后，原子压缩为只含 `status` 的终态墓碑；未完成或待补偿任务仍保留完整内容，任务详情对终态记录回源 PostgreSQL。
- 单元、竞态、真实 Redis 容量测试均通过；32KiB 参数与结果的完成任务压缩后 `HLEN=1`、`MEMORY USAGE=232` 字节。完整 `GOCACHE=/tmp/omc-go-cache go test ./... -count=1` 通过，发布门禁 300/300 通过。
- 最终包 `100.0.0-20260803-2353`，内嵌提交 `9090b9de17a00e8c1ca1ba4ceb3d2d1caf3f23db`，SHA-256 `dff742167c9433b7ca8f20f56265d9fa38d3003e13969b03ca659ea4b8f70b1d`；服务器校验与包内校验全部通过，独立健康检查 104/104。
- 清理后 20,000/20,000 设备在线，ACS 两实例 503、会话拒绝和 Redis OOM 均为 0；真实负载随机抽样的完成任务均为一字段、232 字节，`task:transition:pending=0`，Redis Core 在约 40 万任务键时仅使用 176MiB/3GiB。
- 00:15 槽位 20,000 个 PM 文件全部解析；00:00 槽位开始早于设备在 00:09–00:10 建立的新基线，按启动段隔离规则不生成健康记录。聚合任务版本统一从 01:00 生效，避免把新旧版本切在自然小时中间；这正是“不迁移旧数据、启动段不完整 PM 可丢弃”的预期行为。
- 真实浏览器分别请求 hourly、daily、weekly 聚合接口，均返回 200，耗时约 1ms（缓存）、132ms、127ms；weekly 在 16:16:15 与 16:21:16 UTC 各请求一次，中间无事件即时刷新，证明仅按 5 分钟定时刷新且未扫描原始 PM 表。
- 启动负载中 `parameter_sync_outbox` 从 17,007 降到数百，设备任务持续以约 6,000 条/分钟完成；死信和两库锁等待均为 0。主机 iowait 稳定约 1%，无 swap in/out，主库与应用的 CPU/累计写入是 2 万设备重注册和参数同步的有效启动负载，不是磁盘饱和。
- 合入最新 `origin/main` 后重新完成全量验证与打包：最终版本 `100.0.0-20260804-0055`，内嵌提交 `3e8592f0cbdab380653d246ac627123e9f227a58`，SHA-256 `3887b6c489eaa4b723a75e21a9130bf7c92f5e32fed276cac70a92b3e3f620d2`。服务器外部校验、包内校验和独立健康检查均通过，健康检查为 104/104；所有 OMC 容器均无重启、无 OOM。
- 最终普通升级保留了清理后新产生的 OMC 数据，没有再次清理，更没有触碰非 OMC 数据。升级后 20,000/20,000 设备在线，ACS/Web 近 15 分钟 503 为 0，PM 主队列、`pm-registration-wait` 和 15 分钟聚合队列 pending/ack_pending/redelivered 均为 0；死信、两库锁等待和超过 5 秒的活动 SQL 均为 0。
- 01:00–01:15 是聚合任务版本生效后的首个合法槽位，01:15 后文件按设备持续到达并全部进入聚合流；版本生效前的启动段按既定规则隔离。因此部署验收时尚未到该小时窗口的 02:12 发布时点，数据库无 hourly/daily/weekly 结果是时间水位未到，而不是聚合丢失或 Dashboard 全表扫描。
- 最终资源快照中主机 CPU idle 78–83%、iowait 0%、无 swap in/out；主库约 3.2 CPU、5.36/7GiB，TSDB 2.06/7GiB，Redis Core 193MiB/3GiB、Redis PM 1.34MiB/6GiB。设备任务 pending 18,102→16,437，参数同步 outbox 降至 1，终态计数漂移 1,045→971，表明清库后的 20,000 设备初始同步仍在受控收敛而非卡死。
- 01:00–01:15 首个合法槽位在 01:26:59 的观察轮次尚未达到 12 分钟水位，因此没有提前关闭；下一轮于 01:28:59 正确生成 `complete` 健康记录，expected=20,000、received=20,000、coverage=1。K900010006 与 K900010076 各 105,829 行，最新时间均为 01:15；20,000 个 hourly 窗口保持 open，等待自然小时结束后的 02:12 发布水位。
- 槽位关闭和下一批 PM 到达时出现短时计算峰值，随后主机 CPU idle 回升到 25–33%，iowait 维持 1–2%；可用内存约 19GiB，Redis Core 约 224MiB，完成任务现场抽样仍为 `HLEN=1`、232 字节。主库/时序库均无锁等待和超过 5 秒的活动 SQL，参数同步终态计数漂移继续从 971 降到 376，属于清库后初始同步的受控收敛。
