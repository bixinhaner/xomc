# 参数同步收敛与 PM Redis 物理隔离

## 问题

- 参数同步维护任务可能向下覆盖计划任务数，使部分下发被误判为成功；维护扫描在提交行锁后才执行收敛，多副本会重复处理，且已就绪/计数漂移运行会被普通活跃运行阻塞。
- PM 小时窗口、KPI 路由缓存与 ACS 会话/任务队列共享一个 Redis，在 2 万设备关窗和 AOF 写入期间争抢 CPU、内存和磁盘 I/O。
- PM 首次发布执行无意义清理，周期重算物化大型临时表，加剧 TSDB 临时文件与 I/O 写放大。

## 修改

- 统一参数同步事件路径与维护路径的事务收敛；计划任务数只允许向上纠偏，不完整/零任务异常运行明确失败并保留计划数和实际数用于审计。
- 维护扫描在同一事务中持有 `FOR UPDATE SKIP LOCKED` 到收敛提交，优先处理 ready 和 authoritative counter drift，并在线补齐活跃运行部分索引。
- App/Worker 增加独立 PM Redis 客户端；`pmagg:*` 与 `kpi-route:*` 只走 `redis-pm`，ACS、任务、告警和普通缓存继续走 `redis-core`。
- 生产环境缺少 `pm_redis`、任一地址交叉或两个端点解析为同一 Redis 实例时拒绝启动/拒绝健康通过。
- 发布包支持旧单 Redis 安全升级：先停止写入方，记录挂载与键数，移除旧容器，验证新 core 同卷同键，再自动迁移 PM 状态；App/Worker 作为一个路由兼容单元共同切换。
- 提供 `omcctl pm-redis migrate`，使用 SCAN + pipelined PTTL/DUMP + RESTORE，支持 dry-run、冲突拒绝、显式 replace、TTL/载荷全量验证和反向回滚复制。
- PM 首次发布跳过无效旧版本清理；周期重算改为一致性快照内 keyset 分页，取消大型临时表物化，并补齐 TSDB 在线索引。
- 双 Redis 独立资源预算、AOF、数据目录、监控流水线、容量/缺失告警和运行手册。

## Fresh 验证（2026-08-02）

- `cd omcgo && go build ./... && go vet ./...`：PASS。
- `cd omcgo && go test ./... -count=1`：PASS，含 e2e/integration。
- `go test -race ./internal/paramsync ./internal/pm/stream ./cmd/omcctl ./cmd/app/provider ./cmd/worker -count=1`：PASS。
- 临时 PostgreSQL 16 应用当前基线后，partial dispatch、zero task、维护扫描事务集成测试：PASS。
- `cd omcmb && npm run typecheck`：PASS。
- 发布/部署目录全部 `*_test.sh`：PASS；`storage-compose_test.sh` 为 `PASS=278 FAIL=0`。
- Prometheus 3.5 promtool：9 个规则文件、111 条规则全部 SUCCESS。
- OTel Collector 0.103 `validate`：PASS。
- 真实 Redis 7 双实例：string/hash/set/zset/list/stream 共 6 个 `pmagg:test:*` 全部 copied=6、verified=6、failed=0、conflicts=0；TTL 差 45ms；`kpi-route:test` 未迁移。

## 上线验收

- 全新部署使用 `redis-core` / `redis-pm` 两个不同 run_id，均无 eviction、OOM、rejected connection 或 AOF 错误。
- ACS 503/会话拒绝为 0；PM 主队列、registration-wait、设备周期任务和死信回到正常水位。
- K900010006、K900010076 有数据且新鲜；小时窗口在事件水位满足后按 12 分钟关闭，当前日/周进行中结果持续更新。
- 主库、TSDB、Redis、Worker 的 CPU、内存和磁盘 I/O 无持续触顶；无新慢查询、长锁或业务告警。
