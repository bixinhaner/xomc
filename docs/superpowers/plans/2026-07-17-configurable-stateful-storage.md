# Configurable Stateful Storage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 PostgreSQL、TimescaleDB、Redis、NATS、MinIO 的宿主机数据路径可独立配置，并让 release 资源规划器为新部署选择可用空间最大的持久文件系统。

**Architecture:** 新建一个无 Docker 依赖的 shell 函数库，集中完成挂载点选择、`.env` 非覆盖写入和绝对路径校验；release compose 改用五个 bind mount，安装与服务控制入口统一调用校验。开发 compose 保留命名卷默认行为，仅在显式配置时切换路径。

**Tech Stack:** Bash 4+、Docker Compose、GNU `df`/`awk`、现有 release shell 测试风格。

## Global Constraints

- 五个组件必须使用独立环境变量：`POSTGRES_DATA_PATH`、`TSDB_DATA_PATH`、`REDIS_DATA_PATH`、`NATS_DATA_PATH`、`MINIO_DATA_PATH`。
- 资源规划只填充缺失或空值，不覆盖人工配置，不自动迁移已有数据。
- release 路径必须为绝对路径，升级必须保留人工配置。
- 最大盘按“可用字节数”选择，排除伪文件系统；无额外 SSD/NVMe 时不宣称硬件收益。
- 重启容器化服务只走 release `svc.sh`/`install.sh` 或项目 compose，不使用裸进程脚本。

---

### Task 1: 存储规划函数库

**Files:**
- Create: `deployments/release/bundle/deploy/storage-paths-lib.sh`
- Create: `deployments/release/bundle/deploy/storage-paths-lib_test.sh`

**Interfaces:**
- Produces: `storage_select_largest_mount` 从 `可用字节|挂载点` 输入输出最佳挂载点。
- Produces: `storage_apply_recommended_paths <env-file> <mount>` 只补空值。
- Produces: `storage_validate_env_paths <env-file>` 校验五个绝对路径。
- Produces: `storage_prepare_env_paths <env-file>` 创建并检查五个目录。

- [ ] **Step 1: 写选择、非覆盖和路径校验失败测试**

测试注入 `/ 100GiB`、`/data 500GiB`、`/nvme 400GiB`，断言选择 `/data`；预置
`POSTGRES_DATA_PATH=/custom/pg` 后应用推荐，断言该值不变而其余四项为
`/data/omc-data/<component>`；相对路径必须返回非零。

- [ ] **Step 2: 运行测试确认失败**

Run: `bash deployments/release/bundle/deploy/storage-paths-lib_test.sh`
Expected: FAIL，提示 `storage-paths-lib.sh` 或目标函数不存在。

- [ ] **Step 3: 实现最小函数库**

使用固定键到子目录映射；用 `awk -F'|'` 比较首列整数；`.env` 更新通过临时文件和
`awk` 完成，已有非空值保留；校验拒绝空值和非 `/` 开头值。

- [ ] **Step 4: 运行测试确认通过**

Run: `bash deployments/release/bundle/deploy/storage-paths-lib_test.sh`
Expected: 全部 PASS。

- [ ] **Step 5: 提交**

```bash
git add deployments/release/bundle/deploy/storage-paths-lib.sh deployments/release/bundle/deploy/storage-paths-lib_test.sh
git commit -m "feat: 添加有状态存储路径规划库"
```

### Task 2: release 规划器自动选择最大存储

**Files:**
- Modify: `deployments/release/bundle/deploy/plan-resources.sh`
- Create: `deployments/release/bundle/deploy/plan-resources-storage_test.sh`

**Interfaces:**
- Consumes: Task 1 的 `storage_select_largest_mount` 和 `storage_apply_recommended_paths`。
- Produces: `OMC_PROBE_STORAGE_MOUNTS` 测试覆盖接口，格式为每行 `可用字节|挂载点`。

- [ ] **Step 1: 写失败测试**

复制临时 `.env`，以 32 核、64GiB 探测覆盖和三条虚拟挂载记录运行规划器，断言
`.env` 五项写到最大挂载点；再预置 MinIO 自定义路径运行，断言不覆盖；`--dry-run`
断言文件不变且输出人工修改提示。

- [ ] **Step 2: 运行测试确认失败**

Run: `bash deployments/release/bundle/deploy/plan-resources-storage_test.sh`
Expected: FAIL，`.env` 尚未产生五个路径键。

- [ ] **Step 3: 接入存储探测和提示**

Linux 默认使用 `df -B1 --output=avail,target` 并排除
`tmpfs/devtmpfs/overlay/squashfs/proc/sysfs/cgroup/cgroup2`；测试时使用
`OMC_PROBE_STORAGE_MOUNTS`。非 dry-run 调用函数库补 `.env`，输出五个最终路径、
“安装前人工检查 `.env`”和“不会迁移已有数据”。

- [ ] **Step 4: 运行测试确认通过**

Run: `bash deployments/release/bundle/deploy/plan-resources-storage_test.sh`
Expected: 全部 PASS。

- [ ] **Step 5: 提交**

```bash
git add deployments/release/bundle/deploy/plan-resources.sh deployments/release/bundle/deploy/plan-resources-storage_test.sh
git commit -m "feat: 规划最大可用数据存储路径"
```

### Task 3: compose 与生命周期入口

**Files:**
- Modify: `deployments/release/bundle/deploy/docker-compose.infra.yml`
- Modify: `deployments/release/bundle/deploy/install.sh`
- Modify: `deployments/release/bundle/deploy/svc.sh`
- Modify: `deployments/release/build-release.sh`
- Modify: `deployments/docker/docker-compose.yml`
- Create: `deployments/release/bundle/deploy/storage-compose_test.sh`

**Interfaces:**
- Consumes: Task 1 的 `storage_validate_env_paths`、`storage_prepare_env_paths`。
- Produces: release `.env` 五个默认键和五个 bind mount。

- [ ] **Step 1: 写 compose/升级继承失败测试**

断言 release compose 五个服务均读取对应变量；构建模板包含五键；安装白名单包含五键；
以临时目录运行 compose config，断言宿主路径渲染正确；开发 compose 未传变量时仍为命名卷。

- [ ] **Step 2: 运行测试确认失败**

Run: `bash deployments/release/bundle/deploy/storage-compose_test.sh`
Expected: FAIL，release compose 仍引用命名卷。

- [ ] **Step 3: 实现 bind mount、默认键和生命周期校验**

release compose 分别使用 `${KEY:-/opt/omc/storage/<component>}:/container/path`；
`build-release.sh` 生成五个空键并附人工修改说明；`install.sh` 白名单加入五键，在 compose
启动前准备目录；`svc.sh` 的 start/up/restart 前同样准备目录。开发 compose 使用
`${KEY:-volume-name}` 保持向后兼容。

- [ ] **Step 4: 运行测试确认通过**

Run: `bash deployments/release/bundle/deploy/storage-compose_test.sh`
Expected: 全部 PASS。

- [ ] **Step 5: 提交**

```bash
git add deployments/release/bundle/deploy/docker-compose.infra.yml deployments/release/bundle/deploy/install.sh deployments/release/bundle/deploy/svc.sh deployments/release/build-release.sh deployments/docker/docker-compose.yml deployments/release/bundle/deploy/storage-compose_test.sh
git commit -m "feat: 支持有状态服务独立数据路径"
```

### Task 4: 运维迁移方案和收益边界

**Files:**
- Modify: `deployments/release/bundle/deploy/RESOURCE-PLANNING.md`
- Modify: `deployments/docker/RESOURCE-PLANNING.md`
- Modify: `deployments/docker/README.md`
- Modify: `docs/operations/OMC内网离线部署手册（运维侧）.md`

**Interfaces:**
- Consumes: 五个路径变量和 planner 行为。
- Produces: 停服复制、校验、切换、回滚以及多盘优先级的可执行说明。

- [ ] **Step 1: 写文档校验命令并确认当前失败**

Run: `rg -n "POSTGRES_DATA_PATH|storage-paths|rsync.*numeric-ids" deployments/release/bundle/deploy/RESOURCE-PLANNING.md docs/operations`
Expected: 未覆盖五变量和迁移步骤。

- [ ] **Step 2: 补齐运维文档**

写明先 `svc.sh stop`，再用 `rsync -aHAX --numeric-ids` 分组件复制，核对 `du`/文件数，
修改 `.env`，执行 planner/compose config 检查后 `svc.sh up`，检查五个健康状态并保留
旧目录。记录 NVMe A=TimescaleDB、NVMe B=主库、SSD/NVMe C=MinIO 的建议以及单 SSD
的退化方案。

- [ ] **Step 3: 验证文档覆盖**

Run: `rg -n "POSTGRES_DATA_PATH|TSDB_DATA_PATH|REDIS_DATA_PATH|NATS_DATA_PATH|MINIO_DATA_PATH|rsync -aHAX --numeric-ids" deployments/release/bundle/deploy/RESOURCE-PLANNING.md docs/operations/OMC内网离线部署手册（运维侧）.md`
Expected: 所有变量和迁移命令均命中。

- [ ] **Step 4: 提交**

```bash
git add deployments/release/bundle/deploy/RESOURCE-PLANNING.md deployments/docker/RESOURCE-PLANNING.md deployments/docker/README.md docs/operations
git commit -m "docs: 补充有状态数据拆盘迁移方案"
```

### Task 5: 全量验证、目标机部署和 KPI 复测

**Files:**
- Modify only if verification finds a defect.

**Interfaces:**
- Consumes: Tasks 1-4 完整交付。
- Produces: 可复现的本地验证结果、目标机挂载/健康状态和硬件未迁移性能基线。

- [ ] **Step 1: 本地静态和脚本验证**

Run:

```bash
bash deployments/release/bundle/deploy/storage-paths-lib_test.sh
bash deployments/release/bundle/deploy/plan-resources-storage_test.sh
bash deployments/release/bundle/deploy/storage-compose_test.sh
bash -n deployments/release/bundle/deploy/{storage-paths-lib.sh,plan-resources.sh,install.sh,svc.sh}
docker compose -f deployments/docker/docker-compose.yml config
```

Expected: 全部退出 0。

- [ ] **Step 2: 项目回归**

Run: `cd omcgo && go build ./... && go test ./...`
Expected: build/test 全部通过。

- [ ] **Step 3: 构建 release 并复制到目标机**

沿用现有 release 构建命令生成新版本，通过 `scp` 复制到 `172.24.224.78`；在服务器运行
planner，人工配置值保持当前数据所在路径，先用 compose config 核对五个 source，再运行
`install.sh --yes` 更新服务。

- [ ] **Step 4: 健康和路径验收**

检查 compose `ps`、app/ACS/worker 健康接口、PostgreSQL/TimescaleDB `pg_isready`、
Redis `PING`、NATS/MinIO 健康；用 `docker inspect` 断言五个容器 mount source 与 `.env`
一致。

- [ ] **Step 5: 重复 KPI 压测**

用与前次相同的压测口径采集 FileUpload、ACS、worker、NATS pending、`iostat -x` 和
容器 CPU/内存。明确当前无额外 SSD/NVMe，结果是兼容性基线；给出与前次旋转盘指标的
差异，不推断物理迁移收益。

- [ ] **Step 6: 提交修正并创建 MR**

确认工作树只含本任务改动后推送功能分支，用 `glab mr create --target-branch main`
创建 MR；描述包含实现、迁移操作、验证结果、目标机复测以及“未实际更换硬件”的限制。
