# 数据库迁移与自动初始化

## 1. 概述

OMC 项目使用 [golang-migrate](https://github.com/golang-migrate/migrate) 管理数据库 Schema。Docker Compose 部署时，app 容器会在启动应用之前自动执行所有待执行的迁移，确保数据库 Schema 与代码版本保持一致。

**核心原则**：`docker-compose up` 一键启动，数据库自动就绪，无需任何手动操作。

---

## 2. 组件说明

### 2.1 迁移文件

位置：`migrations/`

共 34 对迁移文件（`.up.sql` / `.down.sql`），按编号顺序执行：

| 编号 | 迁移名称 | 创建的表 |
|------|---------|---------|
| 000001 | create_devices | devices（按 carrier 分区）、3 个分区表 |
| 000002 | create_device_parameters | device_parameters |
| 000003 | create_data_model_definitions | data_model_definitions |
| 000004 | create_oui_registry | oui_registry |
| 000005 | create_data_model_import_log | data_model_import_log |
| 000006 | create_config_templates | config_templates |
| 000007 | create_provisioning_tasks | provisioning_tasks |
| 000008 | create_device_groups | device_groups |
| 000009 | create_device_group_members | device_group_members |
| 000010 | create_pm_counters | pm_counters（TimescaleDB hypertable） |
| 000011 | create_kpi_tables | kpi_definitions、kpi_values（hypertable） |
| 000012 | create_alarms | alarms_active、alarms_history（hypertable） |
| 000013 | create_mr_tables | mr_files、mr_parsed_data |
| 000014 | create_users_roles | users、roles、user_roles、permissions |
| 000015 | create_audit_logs | audit_logs + 种子数据（角色、权限、admin 用户） |
| 000016 | create_firmware | firmware_versions、upgrade_tasks |
| 000017 | optimize_device_sn_index | 索引优化 |
| 000018 | create_alarm_rules | alarm_rules |
| 000019 | create_kpi_thresholds | kpi_thresholds |
| 000020 | create_system_logs | system_logs、ne_message_logs |
| 000021 | optimize_indexes | 批量索引优化（BRIN、复合索引） |
| 000022 | create_backup_tables | backup_tasks、backup_schedules |
| 000023 | create_managed_files | managed_files |
| 000024 | create_mml_tables | mml_commands、mml_scripts、mml_tasks |
| 000025 | create_dashboard_widgets | dashboard_widgets |
| 000026 | create_pm_tasks | pm_tasks |
| 000027 | create_config_baselines | config_baselines、config_tasks、config_neighbors |
| 000028 | create_ftp_configs | ftp_configs |
| 000029 | create_mr_indicators | mr_indicators |
| 000030 | create_licenses | licenses |
| 000031 | create_topology_sites | topology_sites 相关表 |
| 000032 | create_reports | report_definitions、report_records |
| 000033 | create_ops_tables | ops_templates、ops_tasks、ops_command_records |
| 000034 | create_pm_files | pm_files |

### 2.2 迁移工具

位置：`cmd/migrate/main.go`

基于 `golang-migrate/v4` 封装的 CLI 工具，编译后生成 `omcgo-migrate` 二进制。

```bash
# 命令行用法
omcgo-migrate --dsn <连接串> --path <迁移目录> <命令>

# 支持的命令
up        # 执行所有待执行的迁移
down      # 回滚最近一个迁移
version   # 查看当前迁移版本
force N   # 强制设置版本号（修复 dirty 状态）
```

DSN 可通过 `--dsn` 参数或 `OMCGO_DB_DSN` 环境变量提供。

### 2.3 入口脚本

位置：`deployments/docker/entrypoint.sh`

```bash
#!/bin/sh
set -e

# 执行数据库迁移（需要 OMCGO_DB_DSN 环境变量）
if [ -n "$OMCGO_DB_DSN" ]; then
    echo "Running database migrations..."
    omcgo-migrate --dsn "$OMCGO_DB_DSN" --path /etc/omcgo/migrations up
    echo "Database migrations completed."
fi

# 启动应用，传递所有参数
exec omcgo-app "$@"
```

### 2.4 Dockerfile

位置：`deployments/docker/Dockerfile.app`

构建阶段编译两个二进制：

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-app ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-migrate ./cmd/migrate
```

运行阶段将两个二进制、配置文件、迁移文件和入口脚本复制到镜像中：

```dockerfile
COPY --from=builder /build/bin/omcgo-app /usr/local/bin/omcgo-app
COPY --from=builder /build/bin/omcgo-migrate /usr/local/bin/omcgo-migrate
COPY --from=builder /build/cmd/app/etc/config.prod.yaml /etc/omcgo/app.yaml
COPY --from=builder /build/migrations /etc/omcgo/migrations
COPY deployments/docker/entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["--config", "/etc/omcgo/app.yaml"]
```

---

## 3. 部署时执行流程

```
docker-compose up
│
├── postgres 启动
│   └── healthcheck: pg_isready -U omcgo（5s 间隔，最多 5 次重试）
│
├── app 容器启动（depends_on postgres: service_healthy）
│   └── /entrypoint.sh 执行
│       ├── omcgo-migrate --dsn $OMCGO_DB_DSN --path /etc/omcgo/migrations up
│       │   ├── 检查 schema_migrations 表（不存在则自动创建）
│       │   ├── 读取当前版本号
│       │   ├── 按顺序执行所有未执行的 .up.sql 文件
│       │   │   ├── 建表、索引、触发器、分区表
│       │   │   ├── TimescaleDB hypertable + 保留/压缩策略
│       │   │   └── 种子数据（admin 用户、系统角色、权限、OUI 注册表）
│       │   └── 更新 schema_migrations 版本号
│       │
│       └── exec omcgo-app --config /etc/omcgo/app.yaml
│           └── 应用正常启动（表已就绪）
│
├── acs / worker 启动（表已由 app 容器创建）
└── web (nginx) 启动
```

---

## 4. 关键特性

### 4.1 幂等性

`golang-migrate` 通过 `schema_migrations` 表记录当前版本号。已执行过的迁移不会重复执行。容器重启时 `migrate up` 输出 `no change` 后直接启动应用。

### 4.2 版本追溯

```bash
# 查看当前迁移版本
docker exec <app_container> omcgo-migrate \
    --dsn "$OMCGO_DB_DSN" \
    --path /etc/omcgo/migrations version
```

### 4.3 增量升级

新增迁移文件后重新构建部署，`migrate up` 只执行新增的迁移，不影响已有数据。

### 4.4 回滚

```bash
# 回滚最近一个迁移
docker exec <app_container> omcgo-migrate \
    --dsn "$OMCGO_DB_DSN" \
    --path /etc/omcgo/migrations down
```

### 4.5 种子数据

以下种子数据在迁移 `000015_create_audit_logs.up.sql` 中插入：

| 数据 | 说明 |
|------|------|
| 系统角色 | admin、operator、viewer（固定 UUID） |
| 管理员账号 | 用户名 `admin`，密码 `admin123`（bcrypt 哈希） |
| 权限矩阵 | admin: 10 资源 × 4 操作；operator: 7 资源 × 2 操作；viewer: 7 资源 × 只读 |

---

## 5. 本地开发

本地开发时无需 Docker，直接使用 Makefile：

```bash
# 启动本地 PostgreSQL 后执行迁移
make migrate-up

# 回滚
make migrate-down

# 也可以直接运行
go run ./cmd/migrate --dsn "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable" up
```

---

## 6. 新增迁移

```bash
# 创建新的迁移文件对
make migrate-create name=create_new_table

# 会生成：
# migrations/000035_create_new_table.up.sql
# migrations/000035_create_new_table.down.sql
```

编写 SQL 时遵循以下约定：

- 主键使用 `UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- 所有表���含 `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- 需要更新追踪的表包含 `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` 并创建触发器
- 使用 `IF NOT EXISTS` 确保幂等性
- down 文件使用 `DROP TABLE IF EXISTS` 确保回滚安全

---

## 7. 故障排查

### 迁移失败（dirty 状态）

如果迁移执行中途失败，`schema_migrations` 会标记为 dirty。此时需要手动修复：

```bash
# 查看当前状态
docker exec <app_container> omcgo-migrate \
    --dsn "$OMCGO_DB_DSN" \
    --path /etc/omcgo/migrations version

# 强制设置为上一个成功的版本（例如 33）
docker exec <app_container> omcgo-migrate \
    --dsn "$OMCGO_DB_DSN" \
    --path /etc/omcgo/migrations force 33

# 修复 SQL 后重新执行
docker exec <app_container> omcgo-migrate \
    --dsn "$OMCGO_DB_DSN" \
    --path /etc/omcgo/migrations up
```

### 容器启动失败

如果迁移执行失败，`entrypoint.sh` 中的 `set -e` 会导致容器退出。查看日志定位原因：

```bash
docker-compose logs app
```

---

## 8. 相关文件

| 文件 | 说明 |
|------|------|
| `migrations/*.sql` | 34 对 SQL 迁移文件 |
| `cmd/migrate/main.go` | 迁移工具源码 |
| `deployments/docker/entrypoint.sh` | 容器入口脚本 |
| `deployments/docker/Dockerfile.app` | app 镜像构建（含 migrate 二进制） |
| `deployments/docker/docker-compose.yml` | 容器编排 |
