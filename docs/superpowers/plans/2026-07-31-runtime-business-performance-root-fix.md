# 运行时业务与性能根治实施计划

> 设计依据：
> `docs/superpowers/specs/2026-07-31-runtime-business-performance-root-fix-design.md`

## 第一轮：任务与队列根治

1. 为 UE Count Redis 到期闸门增加失败测试：
   - 冷启动设备稳定分布到 12 个槽；
   - 未到期不准入、到期只准入一个并推进下次时间；
   - 兼容旧值 `"1"`；
   - 创建失败后 5 分钟重试。
2. 最小实现 Redis Lua 到期闸门，并将默认周期改为 1 小时。
3. 为 `UECountPolicy` 增加失败测试，验证依赖失败会安排短重试，成功保持正常周期。
4. 为 GPV `nil/null` 增加失败测试，最小修改解码器。
5. 为 `TaskService.CreateTask` 增加失败测试，验证调用 context 已取消时 Redis 入队失败
   仍能执行 PG 补偿删除；实现独立有界 rollback context。
6. 为 Redis 队列观测增加失败测试：
   - key 数超过旧 10,000 上限仍采集成功；
   - `TYPE none` 删除竞态被跳过；
   - 未知类型仍失败。
7. 最小修改观测器并运行目标包回归。

## 第二轮：验证、MR 与部署

1. 运行 gofmt、目标包测试、`go build ./...`、`go test ./... -count=1`。
2. 审查 diff，提交 Conventional Commit。
3. 推送分支并创建 MR；等待流水线通过后合入。
4. 从最新 main 构建发布包，通过发布脚本全部门禁。
5. 清洁部署到 172.24.224.197。

## 第三轮：20,000 设备线上验收

1. 连续采集至少一个完整 65 分钟覆盖周期：
   - 每 5 分钟 UE Count 创建/完成/失败数；
   - PG pending/sent/failed 与 Redis taskq；
   - ACS 入队错误、GPV nil 告警；
   - NATS pending/redelivery；
   - HTTP 5xx、容器 restart/OOM。
2. 同步采集 CPU、内存、Redis/PG 连接池、磁盘吞吐/iowait/await。
3. 任一验收项失败时，保存证据并回到“失败测试 → 最小修复 → MR → 部署”循环。

## 第四轮：PM 与 KPI 完整性

1. 查询 `PMReportKeysMissingFromLibrary`、`PMKnownIndicatorsDisabled` 的实际 top keys、
   厂家、制式和文件覆盖，区分 whitelist miss、known disabled、technology mismatch。
2. 验证 `K900010006`、`K900010076` 全部依赖 Counter 的 20,000/20,000 覆盖。
3. 在第一个有效小时窗口结束后 12 分钟验证小时结果；继续验证进行中的天/周覆盖与
   最终结果语义。
4. 对确认的映射/白名单根因执行独立 TDD 修复、MR、部署与同口径复验。
