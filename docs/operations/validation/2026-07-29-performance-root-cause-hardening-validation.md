# 性能根因治理部署验证报告（2026-07-29）

## 范围

- 测试服务器：`172.24.224.197`
- 最终版本：`0.1.6-20260729-1158`
- Git 提交：`27fe661cf`
- 分支：`codex/performance-root-cause-hardening`
- 页面数据：直接读取现有 eNB、gNB、GSM 小时/天/周聚合数据；仅每 5 分钟定时刷新，
  不在窗口聚焦或网络重连时即时刷新。

## 环境清理

部署前停止并删除 OMC compose 栈、10 个 OMC 命名卷及
`/opt/omc/run/logs/{app,acs,worker,nginx}` 下的运行日志。保留发布目录、实例配置和
静态模型库。清理后确认 OMC 容器、卷和运行日志文件均为 0。

该删除不可恢复。最终部署重新建立了全新的卷；发布包的幂等 seed 随后写入标准测试基线，
包括 10,000 台测试设备，因此最终环境没有历史残留，但不是空业务库。部署后外部基站压测流量
重新接入，最终时序库有 342,372 个 measurement anchor，`pm_metrics` 视图展开为
3,851,685 个指标值；连续取样 anchor 数和最大时间不再增长，展开结果没有重复 ID。

Docker 根目录为 `/home/docker-data`，PostgreSQL、TSDB、Redis、NATS、MinIO 卷均落在
`/dev/sda9`（765 GiB）上。最终使用 41 GiB、可用 724 GiB、使用率 6%，未占用 50 GiB
系统根分区承载业务卷。

## 已修复根因

1. ACS 上传背压把磁盘、I/O、队列三种信号拆为独立迟滞状态，避免一个信号恢复误释放另一个
   仍处于高水位的信号。
2. Redis 聚合窗口从每指标 5 个 hash 字段压缩为 1 个 `v1` 字段，并支持旧格式原子迁移和
   双格式读取，降低键空间、AOF 和磁盘写放大。
3. TSDB 影子维表 staging 表增加主键唯一索引和 `ANALYZE`，删除阶段改用等值主键连接，
   并记录逐表耗时，避免全表反连接放大 CPU 和磁盘读取。
4. Redis 资源规划改为 2–8 GiB 合理区间，运行策略固定为 `noeviction + AOF everysec`；
   增加聚合失败、队列采样过期、Redis 内存、磁盘、慢查询和 Dashboard 查询保护告警。
5. 线上慢查询监控发现设备组计数缓存冷启动时 27 个并发请求同时穿透。单次执行计划仅
   24–32 ms，实际并发时每条被拖至 1.0–1.48 s。现用 singleflight 合并冷缓存加载，
   32 路并发回归测试确认只执行 1 次数据库查询。
6. 干净环境初始化 10,000 台设备时，内置聚合任务成员使用单条多值 `INSERT`，成员数超过
   PostgreSQL extended protocol 的 65,535 参数上限，首次部署有 9/12 个定义失败。
   现改为同一事务内每 1,000 个成员分批写入，每批最多 6,000 个参数；失败仍整体回滚。
7. 测试服务器漏配 `OMC_PUBLIC_HOST`，Worker 曾生成 `http://:7557/...` 并跳过 SPV。
   已按服务器基站可达地址固化为 `172.24.224.197`，升级继承后容器内值和模板渲染均正常。

## 本地验证

- `go build ./...`：通过。
- `go test ./...`：完整权限下通过，包含 e2e 和 integration。
- `npm run typecheck`：通过。
- Dashboard 针对性测试：1 个文件、21 个用例全部通过，覆盖小时/天/周粒度、300,000 ms
  定时轮询、关闭窗口聚焦和重连即时刷新。
- 发布 compose 契约：94 通过、0 失败。
- 资源规划与存储路径：15 通过、0 失败；16 通过、0 失败。
- Bash 3.2 发布脚本兼容检查：全部通过。
- 最终发布包本地 SHA-256、服务器 SHA-256 和包内 944 个文件
  `checksums.sha256`：全部通过。

## 线上验证

- `healthcheck.sh`：25 通过、0 失败。
- app、ACS、worker、web 和完整监控栈均运行；四个业务容器
  `RestartCount=0`、`OOMKilled=false`。
- Prometheus 7 个规则文件全部通过 `promtool check rules`，共 73 条规则。
- Redis：`used_memory=46.82 MiB`、`maxmemory=2 GiB`、容器限额 3 GiB，
  `maxmemory-policy=noeviction`、`appendonly=yes`、`appendfsync=everysec`。
- PM 队列：pending=0、ack pending=0，队列采样正常。
- ACS：背压 active=0、拒绝总数=0、队列斜率=0。
- 聚合：`ready=1`、失败总数=0；12/12 个内置任务均有当前版本，成功写入 90,008 个
  当前版本成员。刚部署时小时/天/周表尚未到生成周期，路由和回归测试确认三种页面粒度
  直接读取已有聚合表。
- Dashboard：inflight=0，未出现超时或并发拒绝；配置校验强制 query timeout、
  statement timeout、queue timeout 和 1–64 的并发上限。
- 缓存击穿修复后新 app 进程未产生 `pgx_slow_query_total`，日志中无慢查询；
  修复前同一启动阶段已出现 27 条。
- Worker 启动日志未再出现 extended protocol 参数超限、对账超时、无效上传地址或 ERROR；
  app、ACS、worker 最近日志未发现 panic、fatal、OOM、聚合失败、背压或新慢查询。
- 验证期间外部基站流量约 498 请求/秒、852 个活跃 Nginx 连接。该负载下抽样：
  web 约 1.10 核 / 157.7 MiB，ACS 0.41 核 / 55.6 MiB，app 0.15 核 / 94.0 MiB，
  PostgreSQL 0.33 核 / 888 MiB，worker 0.01 核 / 98.8 MiB，TSDB 0.01 核 /
  1.08 GiB；宿主 CPU 仍有 83–91% 空闲。
- 主库连接 59、库大小 294 MiB；业务盘 `/dev/sda9` 使用 42 GiB/765 GiB（6%）。
- 当前 firing 告警除预期的 `DeadMansSwitch` 外，有一条
  `PMCountersDiscoveredOutsideLibrary` warning。它由持续接入的 LTE 基站流量发现指标库外
  counter 触发，数据会保留并动态登记；资源、队列和聚合失败保护告警均未触发。

## 非本次阻断项

发布包构建完成后，本机下载服务 `serve.sh restart` 因脚本第 326 行存在历史非法字符
`LOG_FILE�` 启动失败。发布包生成、校验、直传、安装和线上运行均不依赖该下载服务，
不影响本次部署结果。
