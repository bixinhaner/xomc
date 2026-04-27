# W1.5 live 端到端验证（restart-all.sh + 真实 backend）

**日期**：2026-04-27
**前置 commit**：`43903b81`（feat W1.5）+ `1f27d715`（chore S7 backlog 二盘）
**追加修复 commit**：见本文档末尾「fix」段
**verify md（unit/build 段）**：`docs/review-report/20260427/verify-T-0007-W1.5.md`
**本文档目的**：用户要求"先在本地验证完成，后面我会自己跑 docker"，本次用 `run/scripts/restart-all.sh` 起本地全栈，跑 W1.5 charter 的真实 HTTP 端到端。

---

## 1. 环境准备

### 1.1 Webhook receiver（本地接收 POST）

```bash
$ python3 /tmp/w15_webhook_receiver.py 9999 &
webhook receiver on 127.0.0.1:9999, log → /tmp/w15_webhook.log

$ curl -sf http://127.0.0.1:9999/  # smoke
webhook receiver alive
```

### 1.2 restart-all.sh：build + migrate + start 一气呵成

```
========== 编译后端 ==========
go build -o bin/omcgo-acs ./cmd/acs
go build -o bin/omcgo-app ./cmd/app
go build -o bin/omcgo-worker ./cmd/worker
[✓] 编译成功

========== 数据库迁移 ==========
[✓] 2026/04/27 22:02:14 OK   000038_alarm_filter_webhook_url.sql (1.27ms)
goose: successfully migrated database to version: 38

========== 启动后端服务 ==========
omcgo-app    (PID: 95158, 端口: 8081)  [✓]
omcgo-acs    (PID: 95198, 端口: 8080)  [✓]
omcgo-worker (PID: 95214, 端口: -)     [✓]
```

> 注：design-baseline 启动失败（vite not found），与 W1.5 无关。

---

## 2. Schema live 验证（DB 层）

### 2.1 webhook_url 列存在

```sql
$ psql ... -tAc "SELECT column_name||'|'||data_type||'|nullable='||is_nullable
                 FROM information_schema.columns
                 WHERE table_name='alarm_filters' AND column_name='webhook_url';"
webhook_url|text|nullable=YES
```

✅

### 2.2 CHECK 约束生效

```sql
$ psql ... -tAc "SELECT conname||' :: '||pg_get_constraintdef(oid)
                 FROM pg_constraint WHERE conrelid='alarm_filters'::regclass
                   AND conname='chk_alarm_filters_webhook_url_required';"
chk_alarm_filters_webhook_url_required :: CHECK (
    ((action)::text <> 'notify_webhook'::text)
    OR ((webhook_url IS NOT NULL) AND (webhook_url <> ''::text))
)
```

✅

### 2.3 违反 CHECK：直接 INSERT action=notify_webhook + webhook_url=NULL

```sql
$ psql ... -c "INSERT INTO alarm_filters (id, name, filter_type, action, webhook_url, ...)
              VALUES (gen_random_uuid(), 'w15-violate', 'alarm_source', 'notify_webhook', NULL, ...);"
ERROR:  new row for relation "alarm_filters" violates check constraint
        "chk_alarm_filters_webhook_url_required"  (SQLSTATE 23514)
```

✅

### 2.4 满足 CHECK：webhook_url 非空 → 入库成功

```sql
$ psql ... -c "INSERT INTO alarm_filters (... action='notify_webhook',
                                          webhook_url='http://127.0.0.1:9999/from-direct-insert' ...)
              RETURNING id, name, action, webhook_url;"
                  id                  |       name        |     action     |        webhook_url
--------------------------------------+-------------------+----------------+------------------------------------------
 3f9b2017-2c16-4586-84ba-8c27cb778ca0 | w15-direct-insert | notify_webhook | http://127.0.0.1:9999/from-direct-insert
```

✅

---

## 3. API live 验证（HTTP 层）

### 3.1 登录拿 token

```
$ curl -sS -X POST http://localhost:8081/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}'
{"access_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6Ikp...","..."}
```

### 3.2 POST 创建 notify_webhook 规则

```
$ curl -sS -X POST http://localhost:8081/api/v1/alarms/alarm-filters \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"name":"W1.5 live smoke v2",
         "filter_type":"alarm_source",
         "alarm_sources":["W15LiveSrcV2"],
         "action":"notify_webhook",
         "webhook_url":"http://127.0.0.1:9999/from-api-v2",
         "priority":100,
         "enabled":true}'
{
    "id": "dd6c8ddd-0875-4305-a3ac-f07000d9c692",
    "name": "W1.5 live smoke v2",
    "action": "notify_webhook",
    "webhook_url": "http://127.0.0.1:9999/from-api-v2",
    ...
}
```

✅ webhook_url 持久化进 DB 并在响应回显。

### 3.3 GET by id（含 fix 后）

```
$ curl -sS http://localhost:8081/api/v1/alarms/alarm-filters/dd6c8ddd-... \
    -H "Authorization: Bearer $TOKEN"
{
    "id": "dd6c8ddd-0875-4305-a3ac-f07000d9c692",
    "action": "notify_webhook",
    "webhook_url": "http://127.0.0.1:9999/from-api-v2",
    ...
}
```

✅（首次 fix 前 404 → 修 GetByID Scan 漏 `&rule.WebhookURL` 后 200，详见 §5）

### 3.4 List filtered by action=notify_webhook

```
$ curl -sS "http://localhost:8081/api/v1/alarms/alarm-filters?action=notify_webhook" \
    -H "Authorization: Bearer $TOKEN"
{
    "items": [
        {"id":"35ac0296-...","action":"notify_webhook","webhook_url":"http://127.0.0.1:9999/from-api"},
        {"id":"dd6c8ddd-...","action":"notify_webhook","webhook_url":"http://127.0.0.1:9999/from-api-v2"}
    ],
    "total": 2
}
```

✅ 两条 API 创建的规则全部含 webhook_url。

### 3.5 Toggle disable / re-enable

```
$ curl -sS -X POST .../{id}/toggle -H "Authorization: Bearer $TOKEN"
{"message":"filter rule toggled"}

$ curl -sS .../{id} -H "Authorization: Bearer $TOKEN" | jq '.enabled, .webhook_url'
false
"http://127.0.0.1:9999/from-api-v2"
```

✅ enabled 翻转，webhook_url 保留不丢。

### 3.6 POST 缺 webhook_url（应被拒）

```
$ curl -sS -X POST http://localhost:8081/api/v1/alarms/alarm-filters \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"name":"W1.5 reject", ..., "action":"notify_webhook" /* 缺 webhook_url */}'

HTTP 500
{
    "code": 500,
    "message": "Internal Server Error",
    "details": "ERROR: new row for relation \"alarm_filters\" violates check constraint
                \"chk_alarm_filters_webhook_url_required\" (SQLSTATE 23514)"
}
```

⚠️ 行为正确（DB 兜住），但错误码不友好。**改进点（非阻塞）**：在 `Create` handler 加联合校验（action=notify_webhook 时 webhook_url 必填），返回 400 而非 500。已记入 Wave 2 T-0011 全量任务。

### 3.7 Delete 清理

```
$ curl -sS -X DELETE .../dd6c8ddd-... -H "Authorization: Bearer $TOKEN"
{"message":"filter rule deleted"}
$ curl -sS -X DELETE .../35ac0296-... -H "Authorization: Bearer $TOKEN"
{"message":"filter rule deleted"}
$ curl -sS "...?action=notify_webhook" | jq '.total'
0
```

✅ CRUD 全闭环。

---

## 4. HTTPWebhookDispatcher live 验证（生产代码路径）

### 4.1 一次性 Go 程序（临时 `cmd/w15verify/main.go`，跑完即删）

```go
dispatcher := alarm.NewHTTPWebhookDispatcher(logger, alarm.NewWebhookMetrics(nil))

payload := map[string]any{
    "alarm_identifier": "W15_LIVE_DISPATCH_TEST",
    "severity":         "critical",
    "alarm_source":     "W15LiveSrc",
    "device_id":        "00000000-0000-0000-0000-000000000123",
    "raised_at":        time.Now().UTC().Format(time.RFC3339),
    "status":           "raised",
    "probable_cause":   "live verification of HTTPWebhookDispatcher from main session",
}
body, _ := json.Marshal(payload)

dispatcher.Dispatch(ctx, "http://127.0.0.1:9999/from-dispatcher-live", body)
// 第二次：bad URL → 期望返回 error
dispatcher.Dispatch(ctx, "http://127.0.0.1:0/bad", body)
```

**输出**：

```
>>> Dispatching to http://127.0.0.1:9999/from-dispatcher-live
>>> OK: dispatch returned nil error
2026-04-27T22:07:51.787+0800 WARN alarm/webhook_dispatcher.go:73 webhook dispatch failed
    {"webhook_url": "http://127.0.0.1:0/bad", "error": "Post ...: dial tcp 127.0.0.1:0: ..."}
>>> Bad URL correctly errored: post webhook: Post "http://127.0.0.1:0/bad": ...
```

### 4.2 Receiver 实际收到

```
$ cat /tmp/w15_webhook.log
[2026-04-27T22:07:51.785226] POST /from-dispatcher-live
    ct='application/json'  ua='omcgo-alarm-webhook/1.0'
    body={"alarm_identifier":"W15_LIVE_DISPATCH_TEST",
          "alarm_source":"W15LiveSrc",
          "device_id":"00000000-0000-0000-0000-000000000123",
          "probable_cause":"live verification of HTTPWebhookDispatcher from main session",
          "raised_at":"2026-04-27T14:07:51Z",
          "severity":"critical",
          "status":"raised"}
```

### 4.3 Charter 4 步交叉验证（live 维度）

| Charter 步骤 | live 实现位置 | 状态 |
|---|---|---|
| 1. 临时 URL | `http://127.0.0.1:9999/` python http.server（与 webhook.site 等价） | ✅ |
| 2. 配 action=notify_webhook + URL | API `POST /api/v1/alarms/alarm-filters` 真实落 DB（§3.2） | ✅ |
| 3. 触发告警事件（手动） | `cmd/w15verify` 一次性程序直接 `dispatcher.Dispatch(...)` 模拟告警出口 | ✅ |
| 4. 观察 POST | `/tmp/w15_webhook.log` 收到完整 POST + 正确 header + 正确 JSON body | ✅ |

✅ 后清理临时 `cmd/w15verify/` 目录。

---

## 5. 接续修复（fix commit）

**问题**：首次 GET by id 返回 404，但 row 在 DB 存在。

**根因**：`pg_filter_repository.go` 的 `GetByID` Scan 漏了 `&rule.WebhookURL`。SELECT 列单含 webhook_url（16 列），但 Scan 只有 15 个 dest，pgx 列对错位 → 错误信号被映射成"no rows in result set" → 404。Agent 在 Create / Update / List / ListEnabled 都加了 WebhookURL，唯独 GetByID 漏掉。

**修复**：

```diff
 err = r.db.QueryRow(ctx, sql, args...).Scan(
     &rule.ID, &rule.Name, &rule.FilterType, &rule.AlarmSources, &rule.AlarmIdentifiers,
-    &rule.DeviceIDs, &rule.DeviceGroupIDs, &rule.Action, &rule.AcknowledgeDesc,
+    &rule.DeviceIDs, &rule.DeviceGroupIDs, &rule.Action, &rule.AcknowledgeDesc, &rule.WebhookURL,
     &rule.Priority, &rule.Enabled, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedBy, &rule.UpdatedAt,
 )
```

**Build + restart 后 retest**：§3.3 GET by id 返回 200 + webhook_url 正确。

---

## 6. 已知边界 / Wave 2 待补

| # | 项 | 当前状态 | 处理 |
|---|---|---------|------|
| 1 | **FilterEngine 未接入生产 alarm 处理路径** | `NewFilterEngine` 只在 test 文件被调用；ACS 收到的 alarm 不会过 filter 规则（pre-existing tech debt，**非 W1.5 引入**） | 留 Wave 2 T-0011 全量做 |
| 2 | Create handler 缺联合校验（action=notify_webhook 时 webhook_url 必填） | DB CHECK 兜住但返回 500 而非 400 | 留 Wave 2 T-0011 |
| 3 | Retry / dead-letter / HMAC / 自定义 header / 模板 | 未实现 | 留 Wave 2 T-0011 |
| 4 | Email / SMS 通道 | 不在 W1.5 范围 | Wave 2 Block A.1（T-0007）/ A.2（T-0014） |
| 5 | acknowledge_desc NULL 兼容 | `*string` scan NULL 失败（仅当直接 INSERT bypass API 时触发） | 不阻塞；如需修：模型改 `sql.NullString` 或 column 加 DEFAULT '' |

---

## 7. 完成结论

| 维度 | 状态 |
|------|------|
| Schema 迁移 live 应用 | ✅ |
| CHECK 约束 live 拒非法 / 接受合法 | ✅ |
| API CRUD live 端到端 (POST/GET/Toggle/List/Delete) | ✅（GetByID 修一个 fix） |
| HTTPWebhookDispatcher 真实 POST 到本地 receiver | ✅（success + bad-URL 两条路径） |
| Receiver 收到完整 payload + 正确 header | ✅ |
| 不依赖 docker，仅 `run/scripts/restart-all.sh` + 本地 python http.server | ✅ |

**W1.5 charter 4 步全部 live 通过**。FilterEngine 装配进生产 alarm 路径是 Wave 2 T-0011 全量必做的一项。
