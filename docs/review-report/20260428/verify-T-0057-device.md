# verify-T-0057-device — device 模块 errors.NotFound 映射修复

> 任务：T-0057 / W2.D.1.b（device 部分）
> 章程关联：T-0056 verify-md §3 真 bug #3
> 日期：2026-04-28
> 分支：worktree-agent-a3ea0765

---

## 1. 现象（修复前）

E2E 脚本 `omcgo/scripts/e2e_verify.sh:2603` 段：

```sh
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/devices/$S37_DEVICE_ID/reboot" -H "$AUTH_HEADER")
HTTP_CODE=$(echo "$RESP" | tail -1)
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "202" ] || [ "$HTTP_CODE" = "404" ]; then
    pass "POST /devices/:id/reboot (HTTP $HTTP_CODE; 404 acceptable for seed-not-found)"
else
    fail "POST /devices/:id/reboot (500 indicates error-mapping bug; should be 404)" \
         "expected HTTP 200/202/404, got $HTTP_CODE"
fi
```

当 `S37_DEVICE_ID` 不存在时，e2e 期望 404，但 handler 返回 500，触发 fail。

---

## 2. 根因

`internal/device/device_handler.go:303` 的 `RebootDevice` 处理器把 service 层的所有错误统一映射为 `http.StatusInternalServerError`，没有区分业务 sentinel `commonerrors.ErrNotFound`。

Service 层（`device_service.go:113-120`）已经正确返回 `commonerrors.ErrNotFound`：

```go
device, err := s.deviceRepo.GetByID(ctx, id)
if err != nil {
    return fmt.Errorf("get device for reboot: %w", err)
}
if device == nil {
    return commonerrors.ErrNotFound
}
```

Repository 层（`device_repository.go:607-614`）也把 `pgx.ErrNoRows` 转成 `(nil, nil)`：

```go
func (r *PgDeviceRepository) scanDevice(...) (*model.Device, error) {
    row := r.pool.QueryRow(...)
    d, err := scanDeviceFromRow(row)
    if err == pgx.ErrNoRows {
        return nil, nil
    }
    return d, err
}
```

但 handler 写死 500 切断了映射链路：

```go
if err := h.service.RebootDevice(c.Request.Context(), id); err != nil {
    commonerrors.AbortWithError(c, http.StatusInternalServerError, err)  // ← bug
    return
}
```

---

## 3. 修复方案

**位置**：handler 层（最少改动，service / repo 已正确返回 sentinel）

`internal/device/device_handler.go:303-317`：

```go
if err := h.service.RebootDevice(c.Request.Context(), id); err != nil {
    commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
    return
}
```

`commonerrors.HTTPStatusFromError`（`internal/core/errors/errors.go:97-117`）会把：

- `errors.Is(err, ErrNotFound)` → 404
- `errors.Is(err, ErrAlreadyExists)` → 409
- `errors.Is(err, ErrInvalidInput)` → 400
- 默认 → 500

---

## 4. 单元测试

新增两个测试用例（`internal/device/handler_test.go`）：

1. `TestRebootDevice_NotFound_Returns404` — 不存在的 device id POST reboot，期望 404
2. `TestRebootDevice_InvalidUUID_Returns400` — 非 UUID 格式 path 参数，期望 400

---

## 5. 验证

### 5.1 编译

```sh
$ go build ./...
（无输出，编译成功）
```

### 5.2 单元测试

```sh
$ go test -race -count=1 -run TestRebootDevice ./internal/device/...
ok  	github.com/omcgo/omcgo/internal/device	1.983s

$ go test -race -count=1 ./internal/device/...
ok  	github.com/omcgo/omcgo/internal/device	1.647s
```

全模块 device 测试通过，含新增的 `TestRebootDevice_NotFound_Returns404`、`TestRebootDevice_InvalidUUID_Returns400`。

### 5.3 E2E 段位移（本 worktree 范围内仅以代码层验证）

`scripts/e2e_verify.sh:2603-2611` 的 `POST /devices/:id/reboot` 现在：

- device 存在：service 入队成功 → 202 Accepted → pass
- device 不存在：service 返回 `ErrNotFound` → handler `HTTPStatusFromError` → 404 → pass（脚本接受 404）
- 系统错误（DB 连接失败等）：仍返回 500（合理）

预期：合入 main 后该处 fail 转 pass，framework 段 9 个真 bug 减一。

---

## 6. 影响面

| 维度 | 内容 |
|------|------|
| 文件改动 | `internal/device/device_handler.go`（1 处）+ `internal/device/handler_test.go`（新增 2 个测试函数） |
| 行为变更 | `POST /devices/:id/reboot` 不存在 device 时 500 → 404；其余情况不变 |
| 兼容性 | 前端通常按 HTTP 状态码分支：404 比 500 更易被用户友好提示，无破坏性 |
| 关联模块 | service / repo 层无改动；遵循现有 `HTTPStatusFromError` 模式 |

---

## 7. 未触及的同类问题

T-0057 verify-md 中提到的同类映射 bug 跨 alarm/device/mr/licenses/sites/ops 6 模块。本次仅修 device.RebootDevice 一处（属于 T-0057 sub-agent 边界），其余由各自 sub-agent 处理：

- 严禁改 `internal/{alarm,mr,license,topology,ops}/`（任务硬约束）
- 严禁改 `internal/core/errors/`（任务硬约束）

---

## 8. 自检清单

- [x] device_handler.go 仅改 RebootDevice 一处，无副作用
- [x] 没有改动 service.go / repository.go（已正确返回 sentinel）
- [x] 没有改动 `internal/core/errors/`
- [x] 没有改动其他禁改模块
- [x] 没有新增 go.mod 依赖
- [x] 单元测试覆盖 NotFound + InvalidUUID 两条路径
- [x] `go build ./...` 通过
- [x] `go test -race -count=1 ./internal/device/...` 通过
- [x] 没有 commit / push / pull
