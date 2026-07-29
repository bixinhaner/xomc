# ACS 30,000 Global Session Limit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将生产 ACS 全局会话上限持久提升到 30,000，并验证 14:00 整点负载。

**Architecture:** 保持 Redis 全局准入控制实现不变，调整生产配置并用发布契约锁定；普通升级只定向迁移历史默认值，保留运维自定义配置。部署后通过配置启动日志、Redis 槽位、应用日志和 Prometheus/数据库指标联合验收。

**Tech Stack:** Go、YAML、Bash、Docker Compose、Prometheus、PostgreSQL、Redis。

## Global Constraints

- 仅生产 Docker 配置改为 30,000。
- 不放宽 PM 背压、磁盘、I/O 和设备限流。
- 14:00–14:05 连续观察，不用单点采样代替窗口结论。

---

### Task 1: 生产配置契约

**Files:**
- Modify: `deployments/release/bundle/deploy/storage-compose_test.sh`
- Create: `deployments/release/bundle/deploy/config-upgrade-lib.sh`
- Create: `deployments/release/bundle/deploy/config-upgrade-lib_test.sh`
- Modify: `deployments/release/bundle/deploy/install.sh`
- Modify: `omcgo/cmd/acs/etc/config.prod.yaml`

**Interfaces:**
- Consumes: ACS `session.max_concurrent` YAML 配置。
- Produces: 发布镜像内持久生效的 30,000 全局准入上限。

- [x] **Step 1: 写入要求 30,000 的发布契约断言**
- [x] **Step 2: 运行契约并确认因当前值 10,000 失败**
- [x] **Step 3: 将生产配置和容量注释改为 30,000**
- [x] **Step 4: 重跑契约并确认通过**
- [x] **Step 5: 运行后端构建、测试和前端类型检查**
- [x] **Step 6: 普通升级仅迁移历史默认值并保留运维自定义值**

### Task 2: 部署与整点验证

**Files:**
- Verify: `deployments/release/bundle/deploy/healthcheck.sh`

**Interfaces:**
- Consumes: 新发布包及测试服务器现有 OMC 数据。
- Produces: 14:00–14:05 资源、队列、数据库、KPI 和 503 验证证据。

- [x] **Step 1: 构建并校验发布包**
- [x] **Step 2: 部署后确认 ACS 启动日志加载 30,000**
- [x] **Step 3: 运行健康检查**
- [x] **Step 4: 采集 14:00–14:05 指标与日志**
- [x] **Step 5: 对照通过条件给出结论**
- [x] **Step 6: 从已提交实现 SHA 重建并用普通升级路径重新部署**
