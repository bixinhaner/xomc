# omcctl 使用手册

> **omcctl** — OMC 后台运维 CLI 工具
> 单一二进制 + Cobra subcommand 形态，调用 omcgo-app 的 HTTP API（少数命令直连 PostgreSQL）。
> 源码：[omcgo/cmd/omcctl/](../omcgo/cmd/omcctl/)

---

## 1. 编译与运行

```bash
cd omcgo
make build              # 或 go build -o bin/omcctl ./cmd/omcctl
./bin/omcctl --help     # 查看全局帮助
```

二进制名固定为 `omcctl`。下文示例都以 `omcctl` 直接调用为准（建议把 `omcgo/bin/` 加入 `PATH`）。

> 不想本地编译？跳到 **§7 Docker 执行方式**。

---

## 2. 全局选项

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--server` | string | `http://localhost:8080` | omcgo-app 的 HTTP base URL |
| `--api-key` | string | （读环境变量） | `X-API-Key` 头；优先级：flag > `$OMCCTL_API_KEY` |
| `--output` | string | `table` | `table` 或 `json` |
| `--help` / `-h` | bool | — | 任意子命令都有 |

**环境变量**：
- `OMCCTL_API_KEY` — 全局 API Key
- `OMCGO_DB_DSN` — `mml import-spec-md --diff-db` 默认读取
- `USER` — `device sweep-paths --operator` 默认值

**鉴权**：HTTP `X-API-Key` 头。后端中间件无 key 时返回 401（client.go:118-122）。

**输出**：
- `--output table` — 提取响应的 `items` 数组按列表渲染（对齐表格）
- `--output json` — 整段响应原样打印为 JSON

**退出码**：
- `0` — 成功
- `1` — 通用错误（参数错、HTTP ≥400、解析失败）
- `2` — RPC 全失败 / 设备不可达（仅 device sweep-paths）
- `3` — 安全门 abort（仅 device sweep-paths）
- `4` — DB 失败（仅 device sweep-paths）

---

## 3. 命令分组

### 3.1 `device` — 设备管理

| 命令 | 用途 | 后端端点 |
|------|------|---------|
| `device list` | 列设备 | `GET /api/v1/devices` |
| `device get <id>` | 设备详情 | `GET /api/v1/devices/{id}` |
| `device reboot <id>` | 远程重启 | `POST /api/v1/devices/{id}/reboot` |
| `device delete <id>` | 删除设备 | `DELETE /api/v1/devices/{id}` |
| `device stats` | 设备统计 | `GET /api/v1/devices/stats` |
| `device sweep-paths <SN>` | GPV 探测参数支持度 | `POST /api/v1/devices/{SN}/sweep-paths` |

**`device list` flags**：`--carrier {cmcc|ctcc|cucc}` `--status xxx` `--limit 20`

**`device sweep-paths`**（特别注意，有破坏性）：探测某设备对一组 TR-069 path 的支持情况，把"不支持"写回 param_mappings/discovered。

```
--apply              真正执行（缺省 dry-run）
--prefix Device.X    只测前缀子集
--batch-size 1       每次 GPV 包含 path 数（>1 出 fault 9005 难定位）
--rpc-timeout 30s    单次 RPC 超时
--rpc-rate 5.0       每秒最多发起任务数（速率限流）
--operator $USER     审计字段
--confirm-paramodel-wide  >10 设备 paramModel 必须显式确认
--force              跳过 50% 比例保护门
--json               JSON 输出
--verbose            详细日志
```

示例：
```bash
omcctl device list --carrier cmcc --limit 50
omcctl device sweep-paths ABCD1234 --verbose                  # dry-run
omcctl device sweep-paths ABCD1234 --apply --prefix Device.FAP --batch-size 1
```

---

### 3.2 `alarm` — 告警管理

| 命令 | 用途 | 后端端点 |
|------|------|---------|
| `alarm list` | 列告警 | `GET /api/v1/alarms` |
| `alarm ack <id>` | 确认 | `POST /api/v1/alarms/{id}/acknowledge` |
| `alarm clear <id>` | 清除 | `POST /api/v1/alarms/{id}/clear` |
| `alarm stats` | 告警统计 | `GET /api/v1/alarms/stats` |

**`alarm list` flags**：`--severity {critical|major|minor|warning}` `--status {active|cleared|acknowledged}` `--limit 20`

示例：
```bash
omcctl alarm list --severity critical --limit 100
omcctl alarm ack alarm-abc123
```

---

### 3.3 `pm` — 性能指标

| 命令 | 用途 | 后端端点 |
|------|------|---------|
| `pm counters` | 原始计数器 | `GET /api/v1/pm/counters` |
| `pm kpi` | KPI 值 | `GET /api/v1/pm/kpi` |
| `pm aggregate` | 触发重算 | `POST /api/v1/pm/aggregation/recompute` |

**`pm counters` / `pm kpi` flags**：`--device-id` `--from RFC3339` `--to RFC3339` `--limit 20`，`pm kpi` 额外有 `--name`。

**`pm aggregate` flags（全必填或有默认）**：
- `--granularity {hourly|daily|weekly|monthly}` 必填
- `--dimension {device|device_group}` 默认 `device`
- `--start RFC3339` 必填
- `--end RFC3339` 必填

返回 `job_id`，**没有 list / status API**，需直接查 `async_jobs` 表（pm.go:170-172）。

示例：
```bash
omcctl pm kpi --name throughput --from 2026-05-01T00:00:00Z --to 2026-05-02T00:00:00Z
omcctl pm aggregate --granularity daily \
  --start 2026-05-01T00:00:00Z --end 2026-05-02T00:00:00Z
```

---

### 3.4 `system` — 系统

| 命令 | 用途 | 后端端点 |
|------|------|---------|
| `system info` | 系统信息（版本/构建） | `GET /api/v1/system/info` |
| `system health` | 健康检查 | `GET /healthz` |

```bash
omcctl system info --output json
omcctl system health
```

---

### 3.5 `mml` — MML catalog 管理（离线工具，最危险也最常用）

这几条**不调 HTTP**，直接读本地文件 / 直连 PG，专给 catalog 维护人员用。

#### `mml import-standard-params`

把南向规范的 `standard-model.xml` 转成 `standard_params` 表的 seed SQL。

| Flag | 默认值 | 说明 |
|------|--------|------|
| `--xml` | `omcgo/data/param-mappings/standard-model.xml` | 输入 XML |
| `--out` | `omcgo/migrations/seed/.standard_params_rows.sql` | 输出 SQL（`ON CONFLICT DO UPDATE` 幂等） |

```bash
omcctl mml import-standard-params --xml ./standard-model-v3.xml
```

#### `mml migrate-device-params`

把历史 `device_parameters.private_path` 翻译为 `standard_path`（Stage 2 一次性数据迁移）。

| Flag | 必填 / 默认 | 说明 |
|------|------------|------|
| `--dsn` | 必填 | Postgres 连接串 |
| `--dry-run` | true | 默认只输出统计 |
| `--apply` | false | 真正写库（覆盖 dry-run） |
| `--batch` | 500 | 每批次行数，避免长事务 |

翻译路径：`device.product_class` 正则匹配 → `product` → `param_mappings` 反向查 → `standard_path`。

```bash
omcctl mml migrate-device-params --dsn "postgres://omcgo:omcgo123@localhost:5432/omcgo" --apply --batch 1000
```

#### `mml import-spec-md`

解析 spec markdown，diff DB 当前 catalog，生成新增 / 更新 / 孤儿 行的 seed SQL + JSON。

| Flag | 默认值 | 说明 |
|------|--------|------|
| `--spec` | （必填） | spec md 文件路径 |
| `--diff-db` | `$OMCGO_DB_DSN` | PG DSN，做基线对比；空值时只生成不 diff |
| `--carrier` | `v2.3` | spec 版本号 |
| `--version` | `cmcc-td-lte-<carrier>` | catalog 版本号 |
| `--out` | （命名按 carrier） | 输出 seed SQL 路径 |
| `--out-json` | （命名按 carrier） | 输出 catalog JSON 路径 |
| `--dry-run` | false | 不写文件，仅 stdout 报告 |
| `--verbose` | false | 详细日志 |

```bash
omcctl mml import-spec-md --spec ./cmcc-tdlte-v2.3.md --verbose
```

---

## 4. 配置与调用模式

### 4.1 推荐的本地 alias

把鉴权和服务器固定下来，命令更短：

```bash
export OMCCTL_API_KEY=your-key-here
alias omc='omcctl --server http://172.19.1.73:8081'

omc device list --carrier cmcc
omc alarm stats --output json | jq .
```

### 4.2 脚本中调用

```bash
#!/usr/bin/env bash
set -euo pipefail

# 失败立刻退出；--output json + jq 拿字段
device_count=$(omcctl device list --output json | jq '.items | length')
[[ "$device_count" -lt 1 ]] && { echo "no devices"; exit 1; }

# 批量 ack 所有 minor 告警
omcctl alarm list --severity minor --output json --limit 1000 \
  | jq -r '.items[].id' \
  | while read id; do omcctl alarm ack "$id"; done
```

---

## 5. 运维提示（最常踩坑）

| 场景 | 提示 |
|------|------|
| **`device sweep-paths` 反复 ABORT (exit 3)** | >10 设备触发的 paramModel 级影响需加 `--confirm-paramodel-wide`；不支持率 >50% 再加 `--force` |
| **`alarm ack` 看不到反馈** | 默认 `table` 输出可能压缩了字段；加 `--output json` 看完整响应；先确认 ID 真存在 |
| **`mml migrate-device-params` 全部跳过** | 9 成是 `device.product_class` 字符串与 `products.product_class_pattern` 正则不匹配；`--dry-run` 看统计数确认 |
| **`pm aggregate` 提交后无下文** | 接口返回 job_id 后异步执行，没有 list API；用 `psql` 查 `SELECT * FROM async_jobs WHERE id = '<job_id>'` |
| **`mml import-spec-md` 撞数据** | source='admin' 的行不被覆盖；要换 spec 内容必须先手工删 admin 行再跑 |
| **看不到 `--output json` 的某字段** | 后端响应外层是 `{data: {...}}`；CLI 已剥外壳，但某些列表用 `items` 包内层数据 |
| **被 401/403 卡住** | 检查 `OMCCTL_API_KEY` 是否设置、API Key 是否过期；后端 super_admin 走 builtIn 旁路 |

---

## 6. 命令快速索引

```
device  list | get | reboot | delete | stats | sweep-paths
alarm   list | ack | clear  | stats
pm      counters | kpi | aggregate
system  info | health
mml     import-standard-params | migrate-device-params | import-spec-md
```

总共 **5 组 / 17 条** 命令。源码入口：[omcgo/cmd/omcctl/main.go](../omcgo/cmd/omcctl/main.go)。

---

## 7. Docker 执行方式

> ✅ **omcctl 与 worker 同镜像发布**（[`deployments/docker/Dockerfile.worker`](../deployments/docker/Dockerfile.worker)，commit `c0b2edaf`）。
> compose up 之后 worker 容器长跑，operator 直接 `docker exec` 进去执行即可，**不需要独立镜像 / 独立 compose 服务**。
> 镜像内 `ENV OMCCTL_SERVER=http://app:8081` 已预设 server 地址，命令行不必再传 `--server`（走 TLS / 外部 host 时仍可用 `--server` 覆盖）。

### 7.0 一次性配置 API key（首次使用必读）

omcctl 调 app HTTP API 必须带 API key（管理面要鉴权）。配置方式：

**方案 A：宿主 `.env` 文件（推荐，所有 exec 自动带）**

在 `deployments/docker/.env`（gitignored）写一行：
```
OMCCTL_API_KEY=你的key
```

或在 shell 里 `export OMCCTL_API_KEY=...`（一次性）。`docker compose up -d worker` 重启后 worker 容器的 `OMCCTL_API_KEY` 环境变量从这里读，omcctl 内部默认值 `os.Getenv("OMCCTL_API_KEY")` 自动拿到。**之后 `docker exec docker-worker-1 omcctl ...` 不用再传 `--api-key`**。

**方案 B：每次 `docker exec` 时显式 `-e` 传**
```bash
docker exec -e OMCCTL_API_KEY=$YOUR_KEY docker-worker-1 \
    omcctl device list
```

**方案 C：CLI flag 显式传**
```bash
docker exec docker-worker-1 omcctl device list --api-key $YOUR_KEY
```

> Key 怎么生成 / 从哪儿拿？走管理面登录后端点（参 app 鉴权模块）；也可由超级管理员在 admin UI / SQL 直接发一份给 ops。


### 7.1 基本调用（推荐日常）

```bash
# OMCCTL_SERVER 已在 ENV,只需 --api-key
docker exec docker-worker-1 omcctl device list \
    --api-key $OMCCTL_API_KEY

docker exec docker-worker-1 omcctl alarm list \
    --api-key $OMCCTL_API_KEY --severity critical

docker exec docker-worker-1 omcctl system health \
    --api-key $OMCCTL_API_KEY
```

> 容器名 `docker-worker-1` 是 compose 默认命名（项目目录名为 `docker` → 服务 `worker` → 实例 `1`）；多实例 / swarm 部署请先 `docker compose ps worker` 查实际名。

### 7.2 交互式（看 --help / 多步操作）

```bash
docker exec -it docker-worker-1 omcctl device sweep-paths --help
docker exec -it docker-worker-1 omcctl mml --help
```

### 7.3 覆盖 server / TLS / 注入环境变量

```bash
# 走 TLS 端点
docker exec docker-worker-1 omcctl device list \
    --server https://app:8444 --api-key $OMCCTL_API_KEY

# 一次性注入临时环境变量(如更高 verbosity)
docker exec -e OMCCTL_DEBUG=1 docker-worker-1 \
    omcctl device sweep-paths 1202000240194DP0026 \
    --api-key $OMCCTL_API_KEY --json
```

### 7.4 alias 速记

```bash
# 加到 ~/.bashrc / ~/.zshrc
alias omcctl-d='docker exec docker-worker-1 omcctl'
alias omcctl-di='docker exec -it docker-worker-1 omcctl'

omcctl-d device list --api-key $OMCCTL_API_KEY
omcctl-di device sweep-paths --help
omcctl-d device sweep-paths SN --api-key $KEY --apply --json
```

### 7.5 涉及本地文件的命令（mml import-*）

`omcctl mml import-standard-params` / `import-spec-md` 等读**本地文件**的命令，需要先把文件复制进 worker 容器再调：

```bash
# 1. 把文件 cp 进容器
docker cp ./spec.xml docker-worker-1:/tmp/spec.xml

# 2. 容器内引用 /tmp/spec.xml 路径调命令
docker exec docker-worker-1 omcctl mml import-standard-params \
    --file /tmp/spec.xml --api-key $OMCCTL_API_KEY

# 3. 用完清理
docker exec docker-worker-1 rm /tmp/spec.xml
```

### 7.6 `mml migrate-device-params` 的特殊情况

这条命令**直连 PostgreSQL**（不走 HTTP），worker 容器在 compose 网络内，postgres 服务名直达：

```bash
docker exec docker-worker-1 omcctl mml migrate-device-params \
    --dsn "postgres://omcgo:omcgo123@postgres:5432/omcgo?sslmode=disable"
```

> 如果在宿主机直接编译的 omcctl 跑，要把 host 改成 `localhost`（走 compose 暴露的 5432 端口）。

### 7.7 离线 / 临时探查（不依赖 worker 容器）

worker 容器**重启窗口**或本地刚 clone 还没 docker up 时，临时跑 omcctl 的回退方式：

```bash
# A. 直接源码编译跑(项目根目录)
cd omcgo && go build -o bin/omcctl ./cmd/omcctl
./bin/omcctl device list --server http://localhost:8081 \
    --api-key $OMCCTL_API_KEY

# B. 一次性 go run(更慢,每次重编)
docker run --rm -it -v "$(pwd)/omcgo:/src" -w /src \
    --network host golang:1.25-alpine \
    go run ./cmd/omcctl device list \
        --server http://localhost:8081 \
        --api-key $OMCCTL_API_KEY
```

### 7.8 踩坑提醒

- **worker 没 jq / bash**：基于 alpine + 仅 omcctl 二进制；管道处理 / 复杂 shell 在容器外做。`omcctl ... --json` 后管道接 `jq` 时 `jq` 写在外面。
- **worker 重启时 docker exec 失败**：发布 / `docker compose up -d worker` 窗口期间 exec 会拒；运维操作避开 deploy 时刻。
- **多实例 / swarm**：容器名后缀 `-1` `-2` 不稳定，先 `docker compose ps worker` 或 `docker ps --filter 'name=worker'` 确认实际名。
- **import-* 文件路径**：见 §7.5，必须先 `docker cp` 进容器；mount volume 也行但单次操作 cp 更直接。
- **OMCCTL_API_KEY 注入**：每次 `docker exec` 默认带宿主当前 env，但 sudo / 不同 shell session 可能丢；显式 `-e OMCCTL_API_KEY=$KEY` 兜底。
- **TLS 自签证书**：worker 镜像有 `ca-certificates`，但项目自签 cert 不在系统 CA bundle 里；走 `--server https://...` 前先确认证书已挂载（或在 `--insecure` 范围内验证）。
