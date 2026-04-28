# T-0057 verify — alarm 模块 errors.NotFound 映射 fix

> Wave 2 / W2.D.1.b — sub-agent agent-ab7e7719
> 章程关联：T-0056 verify-md §3 真 bug #1 + #2
> 范围：`internal/alarm/{handler.go, pg_store.go, engine_test.go, handler_test.go}`
> Worktree: `.claude/worktrees/agent-ab7e7719`，分支 `worktree-agent-ab7e7719`

---

## §1 baseline (2 fail)

线上 docker app（旧二进制）+ 真实 admin token 复现：

```
POST /api/v1/alarms/<random-uuid>/acknowledge   →  HTTP 500
{"code":500,"message":"Internal Server Error",
 "details":"get alarm: scan alarm: no rows in result set"}

POST /api/v1/alarms/<random-uuid>/clear         →  HTTP 500
{"code":500,"message":"Internal Server Error",
 "details":"get alarm: scan alarm: no rows in result set"}
```

根因：`PgAlarmStore.scanAlarmRow` 把 `pgx.ErrNoRows` 直接 wrap 成
`fmt.Errorf("scan alarm: %w", err)`，handler 用 `http.StatusInternalServerError`
固定 500，不识别 ErrNotFound。

---

## §2 fix 位置 + 改前/改后 diff 关键行

### 2.1 repository 层（最深层、最少散点）

`internal/alarm/pg_store.go` — 在 `scanAlarmRow` 入口处把 `pgx.ErrNoRows`
转成 `commonerrors.ErrNotFound`，参考 `internal/provision/pg_repository.go:310`
已有模式。

```diff
 import (
     "context"
     "encoding/json"
+    gerr "errors"
     "fmt"
     "time"

     "github.com/Masterminds/squirrel"
     "github.com/google/uuid"
+    "github.com/jackc/pgx/v5"
     "github.com/jackc/pgx/v5/pgxpool"
+    commonerrors "github.com/omcgo/omcgo/internal/core/errors"
     "github.com/omcgo/omcgo/internal/core/model"
     "github.com/omcgo/omcgo/internal/core/storage"
 )

 func scanAlarmRow(row scannable) (*model.Alarm, error) {
     var a model.Alarm
     var additionalJSON []byte
     if err := row.Scan(...); err != nil {
+        if gerr.Is(err, pgx.ErrNoRows) {
+            return nil, fmt.Errorf("alarm not found: %w", commonerrors.ErrNotFound)
+        }
         return nil, fmt.Errorf("scan alarm: %w", err)
     }
     ...
 }
```

效益：`GetActiveByID` / `GetActiveByDeviceAndIdentifier` /
`scanActiveAlarm` 等所有走 `scanAlarmRow` 的入口都自动获得正确语义，无需在 engine
层逐一判断。Engine 已经做了 `fmt.Errorf("get alarm: %w", err)`，`%w` 链式包裹保
留 sentinel。

### 2.2 handler 层

`internal/alarm/handler.go` — 把固定 500 改成 `HTTPStatusFromError(err)`，由
sentinel 决定 4xx / 5xx。

```diff
 // Acknowledge
-if err := h.engine.Acknowledge(...); err != nil {
-    commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
-    return
-}
+if err := h.engine.Acknowledge(...); err != nil {
+    commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
+    return
+}

 // ClearAlarm — 同上
```

`HTTPStatusFromError` 已在 `internal/core/errors/errors.go:97-117` 实现，
`ErrNotFound → 404`，其它路径默认 500，与原行为兼容。

### 2.3 mock store（测试基础设施）

`internal/alarm/engine_test.go` — `mockAlarmStore.GetActiveByID` 之前返回
`fmt.Errorf("not found")`（plain error，不带 sentinel），单测用 mock 验证 handler
404 路径时无法触发。改为返回包裹 `ErrNotFound` 的错误，与生产实现保持一致。

```diff
 func (m *mockAlarmStore) GetActiveByID(_ context.Context, id uuid.UUID) (*model.Alarm, error) {
     a, ok := m.active[id]
     if !ok {
-        return nil, fmt.Errorf("not found")
+        return nil, fmt.Errorf("alarm not found: %w", commonerrors.ErrNotFound)
     }
     return a, nil
 }
```

---

## §3 单测 PASS 证据

### 3.1 新增 2 个测试

`internal/alarm/handler_test.go`:
- `TestAcknowledge_NotFound_Returns404`
- `TestClear_NotFound_Returns404`

两测均用未 seed 的随机 UUID 调对应 endpoint，断言 `w.Code == 404`。

### 3.2 修复存量测试

原 `TestHandler_Acknowledge_NotFoundAlarm` / `TestHandler_ClearAlarm_NotFound`
之前断言 `500` —— 实际上是验证 bug 状态。本次同步改为断言 `404`。

### 3.3 跑测证据

```
$ cd omcgo && go test -count=1 ./internal/alarm/...
ok      github.com/omcgo/omcgo/internal/alarm   1.803s

$ go test -race -count=1 -run \
  'TestAcknowledge_NotFound_Returns404|TestClear_NotFound_Returns404|TestHandler_Acknowledge_NotFoundAlarm|TestHandler_ClearAlarm_NotFound|TestHandler_Acknowledge_OK|TestHandler_ClearAlarm_OK' \
  ./internal/alarm/...
ok      github.com/omcgo/omcgo/internal/alarm   1.644s
```

> 注：`go test -race -count=1 ./internal/alarm/...` 全量跑会触发预存在的
> `TestIntegration_FullPipeline_RaceCondition_NewAlarmNotYetProcessed` race
> （位置：`expedited_receiver.go` 与 `core/event/channel_bus.go` goroutine 竞争）。
> 在 `git stash` 后回到干净 main 同样 FAIL —— 与本次 T-0057 无关，建议另立任务
> 处理（不在本 sub-agent 工作范围内，未触及相关文件）。

`go build ./...` 全量编译 PASS，无新增警告。

---

## §4 e2e 实跑证据（acknowledge / clear 转 PASS）

为避免动 docker container 中的旧二进制，本 sub-agent 在 host 编译新二进制，
临时绑端口 `:18081` 与 docker 共用 PG/Redis/NATS（DSN 改 localhost），用真实
admin token 验证 fix 行为。

### 4.1 baseline（docker app, OLD code, :8081）

```
$ TOKEN=$(curl -s -X POST localhost:8081/api/v1/auth/login -d '{...admin/admin123...}' | jq -r .access_token)
$ curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
    -d '{"acknowledged_by":"admin"}' \
    -w "\n  HTTP %{http_code}\n" \
    http://localhost:8081/api/v1/alarms/$(uuidgen)/acknowledge

{"code":500,"message":"Internal Server Error",
 "details":"get alarm: scan alarm: no rows in result set",
 "request_id":"app-20260428150259-b82dc074"}
  HTTP 500

$ curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -w "\n  HTTP %{http_code}\n" \
    http://localhost:8081/api/v1/alarms/$(uuidgen)/clear

{"code":500,"message":"Internal Server Error",
 "details":"get alarm: scan alarm: no rows in result set",
 "request_id":"app-20260428150259-d7ba951d"}
  HTTP 500
```

### 4.2 post-fix（local rebuilt app, NEW code, :18081）

```
$ go build -o /tmp/omcgo-app-fixed ./cmd/app/
$ /tmp/omcgo-app-fixed --config /tmp/config.fix.yaml &     # 18081 + localhost DSNs

$ TOKEN=$(curl -s -X POST localhost:18081/api/v1/auth/login -d '{...admin/admin123...}' | jq -r .access_token)
$ curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
    -d '{"acknowledged_by":"admin"}' \
    -w "\n  HTTP %{http_code}\n" \
    http://localhost:18081/api/v1/alarms/$(uuidgen)/acknowledge

{"code":404,"message":"Not Found",
 "details":"get alarm: alarm not found: resource not found",
 "request_id":"app-20260428150520-6a615a64"}
  HTTP 404

$ curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -w "\n  HTTP %{http_code}\n" \
    http://localhost:18081/api/v1/alarms/$(uuidgen)/clear

{"code":404,"message":"Not Found",
 "details":"get alarm: alarm not found: resource not found",
 "request_id":"app-20260428150520-b69da8fd"}
  HTTP 404
```

**两条 endpoint 全部从 500 → 404，错误体保留 root cause 链 `get alarm:
alarm not found: resource not found`，前端能从 sentinel 文本识别"资源不存
在"。**

待 main 合并、docker app 重新构建后，`scripts/e2e_verify.sh` 中相关
`check_status` 自动从 fail 转 pass，无需脚本本身改动。

---

## 产出清单

| 文件 | 变更 |
|------|------|
| `omcgo/internal/alarm/pg_store.go` | import + scanAlarmRow 增加 ErrNoRows → ErrNotFound 映射 |
| `omcgo/internal/alarm/handler.go` | Acknowledge / ClearAlarm 改用 HTTPStatusFromError |
| `omcgo/internal/alarm/engine_test.go` | mockAlarmStore.GetActiveByID 同步用 ErrNotFound |
| `omcgo/internal/alarm/handler_test.go` | 新增 TestAcknowledge_NotFound_Returns404 / TestClear_NotFound_Returns404；修复存量 NotFound 断言 500 → 404 |
| `docs/review-report/20260428/verify-T-0057-alarm.md` | 本文 |

DoD 自查：

- [x] go build ./... PASS
- [x] go test -count=1 ./internal/alarm/... PASS
- [x] 新增功能含成功+失败两条路径单测
- [x] 错误处理走 sentinel + HTTPStatusFromError，与项目模式一致
- [x] 无新增依赖、未改 backlog/charter/migrations/cross-module 文件
- [x] 实跑 baseline + post-fix 两端对照，500 → 404 显式可见

---

End of verify-T-0057-alarm.md
