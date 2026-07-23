# Release 数据库 CPU 配额调整实施计划

**目标：** 将 10000 基站压测验证通过的主库/TimescaleDB/worker 10/10/3 核配置固化到 release 规划器和默认配置。

**范围：** 仅修改 release medium 档、release Compose 默认值、对应测试和资源规划说明。

## 任务

1. 在 `plan-resources-storage_test.sh` 增加 32 核 medium 档 10/10/3，以及 64 核
   large 档数据库各 21 核的输出断言。
2. 在 `storage-compose_test.sh` 增加 release Compose 10/10/3 默认值断言。
3. 先运行测试，确认现有 6/4/3 与 6/4/5 配置导致断言失败。
4. 修改 `plan-resources.sh` 和 `docker-compose.{infra,app}.yml`。
5. 更新 `RESOURCE-PLANNING.md`，记录适用机型、压测依据和 CPU 超分边界。
6. 运行两组 shell 测试、规划器 dry-run 和 Docker Compose 渲染校验。
7. 检查差异后提交、推送并创建 GitLab MR。
