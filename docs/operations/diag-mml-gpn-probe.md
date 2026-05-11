# diag_mml_gpn_probe.sh — MML 路径诊断工具

> 解决「LST DEVICE_INFO 等 MML 命令被 CPE 静默丢弃」类问题的诊断脚本。
> 工作机制：向一台在线 CPE 发 `GetParameterNames("Device.", false)`，
> 拿回真实数据模型，与 `mml_params` 表对比，输出三类差异。

---

## 1. 何时用

| 症状 | 用这个工具 |
|---|---|
| MML 控制台执行 LST/DSP 后 task 状态 `pending` 一直不回 | ✅ |
| task 标记 `failed`，error_message 是 SOAP Fault 9005 | ✅ |
| 同一命令对 A 设备能跑、对 B 设备不行（疑似数据模型差异） | ✅ |
| 想批量补齐某产品的 mml_params（先 dump CPE 数据模型再写 SQL） | ✅ |
| CPE 完全离线（last_inform_at 远超 inform 间隔） | ❌ 工具会 timeout，先排查 ACS 连通性 |
| 想看 ACS 自身 RPC 状态机 / inform 行为 | ❌ 用 `diag_mml_task.sh` 或直接看 ACS 日志 |

> 配套：[troubleshoot-mml-rpc.md](./troubleshoot-mml-rpc.md) 是更上游的整链路排查手册；本工具补充了它「确认 CPE 真实数据模型」这一步的自动化。

---

## 2. 工作原理（一图看懂）

```
   omcctl (你)                              omcgo-app                       omcgo-acs                    CPE
       │                                       │                                │                          │
       │ 1. SELECT devices WHERE status='active' (psql)                          │                          │
       │ <─────────  device_sn ──────────────────                               │                          │
       │                                       │                                │                          │
       │ 2. POST /devices/tasks                │                                │                          │
       │   method=GetParameterNames            │                                │                          │
       │   params={path:"Device.", next_level:false}                            │                          │
       │ ─────────────────────────────────────>│                                │                          │
       │                                       │ INSERT device_tasks            │                          │
       │                                       │ ZADD acs:taskq:{sn}            │                          │
       │                                       │ Connection Request ───────────>│ ── HTTP wake ───────────>│
       │ <───── { task_id } ─────────────────  │                                │                          │
       │                                       │                                │<── Inform ────────────── │
       │                                       │                                │── GetParameterNames ───>│
       │                                       │                                │<── GPN Response ─────── │
       │                                       │                                │  (写入 device_tasks.    │
       │                                       │                                │   result.raw_response)  │
       │                                       │                                │                          │
       │ 3. 轮询 SELECT status FROM device_tasks WHERE id=...                    │                          │
       │ 4. SELECT result->>'raw_response' FROM device_tasks                     │                          │
       │ 5. xmllint 解析 ParameterInfoStruct/Name → cpe_paths.txt               │                          │
       │ 6. SELECT DISTINCT tr069_path FROM mml_params → db_paths.txt           │                          │
       │ 7. comm -23/-13/-12 计算差异 → 生成 report.md / fixup.sql              │                          │
```

工具不修改任何 DB 数据；`fixup.sql` 全部为注释，人工 review 后才执行。

---

## 3. 部署 & 依赖

### 部署位置

工具是单文件 bash 脚本，**不需要 build**。可放在任何能访问：

- OMC 的 PostgreSQL（pg_hba 放行）
- omcgo-app 的 HTTP API（默认 `:8081`）

的位置执行。常见场景：

| 场景 | 部署位置 |
|---|---|
| 本机调试 | 仓库内 `omcgo/scripts/diag_mml_gpn_probe.sh` |
| 现网 omcgo-app 容器旁 | scp 到 app pod 同节点；用容器外 psql 直连 PG |
| 远程运维跳板机 | scp 到跳板机；用 PG/HTTP 内网地址 |

### 系统依赖

```bash
# RHEL / CentOS / Rocky
yum install -y postgresql jq libxml2  # psql / jq / xmllint
# Debian / Ubuntu
apt-get install -y postgresql-client jq libxml2-utils
# macOS
brew install libpq jq libxml2  # 然后把 libpq 的 bin 加 PATH
```

`curl` 通常已自带。`bash` 需 4.0+。

### 鉴权前置 — 获取 OMCCTL_API_KEY

工具走 `X-API-Key` 鉴权（与 `omcctl` 一致）。OMC 登录强制 RSA-OAEP 加密密码
（[admin/model.go LoginRequest](../../omcgo/internal/admin/model.go)），命令行没法
直接 curl 登录拿 JWT。两条获取路径：

#### 路径 A（推荐，运维场景）：用 helper 脚本直接生成

```bash
# 在能访问 PG 的位置（容器内 / 跳板机）跑
export OMCCTL_API_KEY="$(./omcgo/scripts/gen_api_key.sh \
    --dsn "postgres://omc:omc@localhost:5432/omcgo" \
    --user admin \
    --name mml-diag-probe \
    --expires 7)"

echo "$OMCCTL_API_KEY"   # 应该看到 omk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

`gen_api_key.sh` 做的事：
- 读 `users` 表找指定用户 → 拿 user_id
- 生成 36 字符 key (`omk_` + 32 hex，与 [apikey_service.go:40-46](../../omcgo/internal/admin/apikey_service.go#L40-L46) 严格一致)
- 用 pgcrypto `crypt(key, gen_salt('bf', 10))` 算 bcrypt 哈希（与 Go bcrypt 100% 兼容）
- `INSERT api_keys`，stdout 输出明文 key（仅这一次显示）

> 要求 PG 启用 pgcrypto 扩展（omcgo 标准部署已启用，自检：
> `psql "$DSN" -c "SELECT 1 FROM pg_extension WHERE extname='pgcrypto'"`）

#### 路径 B（有前端访问权限）：浏览器 + curl

1. 浏览器登录前端
2. devtools → Application → Storage → `omc-app-store` 或 Cookies 里找 JWT（具体存放位置看前端实现，搜关键字 `accessToken` / `Bearer`）
3. 用 JWT 调创建接口：

```bash
TOKEN='<paste-from-devtools>'
RESP=$(curl -fsSL -X POST http://localhost:8081/api/v1/api-keys \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"name":"mml-diag","scopes":[]}')
export OMCCTL_API_KEY=$(echo "$RESP" | jq -r '.data.key // .key')
```

#### 验证

无论哪条路径，验证一下：

```bash
curl -fsSL -H "X-API-Key: $OMCCTL_API_KEY" \
    http://localhost:8081/api/v1/devices?page=1\&page_size=1
```

返回 JSON 即可。返回 401 表示 key 无效或用户没权限。

---

## 4. 使用

### 4.1 最简调用（自动选设备）

```bash
./omcgo/scripts/diag_mml_gpn_probe.sh \
    --dsn "postgres://omc:omc@localhost:5432/omcgo" \
    --api-key "$OMCCTL_API_KEY"
```

工具自动：

1. 从 `devices` 表挑一台 `status='active'` 且 `last_inform_at < 10 分钟` 的设备
2. POST `/api/v1/devices/tasks` 创建 GPN task（默认 `path="Device."`, `next_level=false`）
3. 轮询 task 状态（每 2s 一次，默认超时 120s）
4. 解析响应，对比 mml_params，输出到 `/tmp/mml-diag-<timestamp>/`

### 4.2 指定设备 + 自定义输出目录

```bash
./omcgo/scripts/diag_mml_gpn_probe.sh \
    --dsn "postgres://omc:omc@10.0.0.5:5432/omcgo" \
    --api "http://10.0.0.5:8081" \
    --api-key "$OMCCTL_API_KEY" \
    --device-sn "BAI-A2B3C4-001" \
    --output ./diag-2026-05-11
```

### 4.3 只取根目录下的直接子节点（轻量探针）

```bash
./omcgo/scripts/diag_mml_gpn_probe.sh \
    --dsn ... --api-key ... \
    --next-level
```

`--next-level` 设置 NextLevel=true，CPE 只返回 `Device.` 的直接子节点（约 10-20 条）。
**适用于探活；不适合做全量数据模型对比**（默认 `false` 才是全量）。

### 4.4 探测特定子树（如只看 GSM 模组）

```bash
./omcgo/scripts/diag_mml_gpn_probe.sh \
    --dsn ... --api-key ... \
    --root-path "Device.X_BAICELLS_DeviceGSM."
```

---

## 5. 输出说明

```
/tmp/mml-diag-20260511-203045/
├── report.md            # 人看：摘要 + 头部样本
├── raw_response.xml     # CPE 原始 SOAP body（调试用）
├── cpe_paths.txt        # CPE 真实 path（每行一条，sort -u）
├── db_paths.txt         # DB mml_params 全部不重复 tr069_path
├── missing_in_cpe.txt   # ❌ DB 有但 CPE 没有 — MML 命令被丢的根因
├── extra_in_cpe.txt     # ℹ️ CPE 有但 DB 未收录 — 候选扩充
├── matched.txt          # ✅ 双方都有
├── fixup.sql            # 修复草稿（全注释，需手动启用）
└── meta.json            # 元数据（设备/task/统计数，机器消费）
```

### `report.md` 摘要示例

```
| 维度 | 数量 |
|---|---|
| CPE 真实 path | 1247 |
| DB mml_params 不重复 path | 156 |
| ✅ 双方都有 (matched) | 89  |
| ❌ DB 配了但 CPE 不存在 | 67  |  ← 这就是 MML 报文被丢的根因
| ℹ️ CPE 有但 DB 未收录 | 1158 |
```

### `missing_in_cpe.txt` 是行动入口

每一行就是一条「DB 配了但 CPE 没有」的路径。修法：

| 情况 | 修法 |
|---|---|
| path 拼写错误（如 `X_COM_` 应为 `X_BAICELLS_COM_`） | `UPDATE mml_params SET tr069_path = '<正确>' WHERE tr069_path = '<错>';` |
| path 不属于此 product 型号 | 把该 param 从 `mml_command_params_rel` 解绑；或按 product 拆 param 库 |
| CPE 数据模型真没有 | DB 暂保留，通知设备侧补 |

`fixup.sql` 已经把每条 missing 路径写成注释模板：

```sql
-- ❌ MISSING IN CPE: Device.X_BAICELLS_DeviceGSM.Mcc
-- UPDATE mml_params SET tr069_path = '<CORRECT_PATH>' WHERE tr069_path = 'Device.X_BAICELLS_DeviceGSM.Mcc';
```

review 后取消注释、填正确路径，再执行。

### `extra_in_cpe.txt` 是扩充入口

CPE 数据模型里有但 DB 没收录的 path，可以选感兴趣的补到 mml_params，让 MML 控制台能用。

---

## 6. 故障排查

| 现象 | 可能原因 | 排查命令 |
|---|---|---|
| `没找到在线设备` | 全部设备 last_inform_at > 10 分钟 | `psql "$DSN" -c "SELECT serial_number,status,last_inform_at FROM devices ORDER BY last_inform_at DESC NULLS LAST LIMIT 10"` |
| `任务创建 HTTP 调用失败` | API 不通 / API key 失效 / 无权限 | `curl -v -H "X-API-Key: $OMCCTL_API_KEY" $API_URL/api/v1/devices` |
| `等待超时 120s` | CPE Connection Request 不响应；或 inform 间隔很长 | 看 ACS 日志：`grep $TASK_ID acs.log`；扩 `--timeout` |
| `task 失败 — code=9005` | CPE 收到了但 path 它自己也不认 — `Device.` 必须合法 | 该 CPE 厂商可能用 `InternetGatewayDevice.`：`--root-path "InternetGatewayDevice."` 再跑 |
| `xmllint 解析后 0 条 path` | CPE 返回的不是 GPN Response 而是 Fault | 看 `raw_response.xml`，里面应该有 `<cwmp:Fault>` |
| `device_tasks.result.raw_response 为空` | ACS 异常路径 mark completed 但未写 raw_response | 看 ACS 日志 `grep "task completed"`；同时 `psql -c "SELECT result FROM device_tasks WHERE id='$TASK_ID'"` |
| 报告所有 path 都 missing | DB 路径全错 + CPE 数据模型完全不同根 | 这就是设计缺陷确凿证据，按 §5 修法表治理 |

### 看 ACS 日志关联 task

```bash
# 在 omcgo-acs 容器内
grep "$TASK_ID" /var/log/omcgo-acs.log
grep "GPN response parsed" /var/log/omcgo-acs.log
```

关键字段：`task_id`、`device_sn`、`cwmp_id`、`parameter_count`。

---

## 7. 与其他工具的关系

| 工具 | 用途 | 适用阶段 |
|---|---|---|
| `diag_mml_task.sh` | 整链路 8-checkpoint 巡检（数据库/Redis/ACS/CPE） | MML 任务执行异常的**起步排查** |
| `audit_mml_params.sh` | 离线审计 mml_params 表里的路径是否符合 TR-069 协议格式 | 看哪些 path 含 `{i}`、错前缀等**协议形式问题** |
| **`diag_mml_gpn_probe.sh`** | 与 CPE 实时握手，对比真实数据模型 | 解决「协议形式 OK 但 CPE 仍然丢」的**根因诊断** |

三件套覆盖了从静态 SQL 审计到在线握手验证的完整链路。

---

## 8. 常见 FAQ

**Q：会不会对线上 CPE 造成压力？**
A：单次 `GetParameterNames("Device.", false)` 是 CPE 标准 RPC，响应大小 50-200KB（取决于 path 数量），耗时秒级，不构成压力。

**Q：能不能并发对多台设备跑？**
A：现版不支持。如需批量：`for sn in <list>; do ./diag_mml_gpn_probe.sh --device-sn $sn ...; done`，或者后续按需扩展 `--batch <file>` 参数。

**Q：JSON 输出能不能让 Prometheus 抓？**
A：`meta.json` 已经是结构化数据。如果要定期跑，套个 cron + `jq` 转 Prometheus textfile collector 即可。

**Q：CPE 返回的 path 超过 1 万条会怎样？**
A：测试过 5000+ 条没问题。瓶颈是 xmllint 的内存（大概 50MB）和 device_tasks.result JSONB 大小（默认 PG 限制 1GB）。
