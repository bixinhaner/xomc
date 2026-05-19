# Code Review — T-0157 C4 DELETE /notifications 一键清空 API

- **日期**：2026-05-19
- **范围**：`internal/notification/{repository,pg_repository,service,handler,mock_repository_test,handler_test}.go`
- **作者**：Claude
- **Reviewer**：Claude（self-review）
- **关联**：T-0157 Phase 1 sub-task **C4**

---

## 变更概要

1. `Repository.DeleteAllByUser(ctx, userID) (int64, error)` 接口 + pg 实现（一行 SQL `DELETE FROM notifications WHERE user_id = ?`）
2. `Service.DeleteAllByUser` 包装
3. `Handler.DeleteAll` + 路由 `DELETE /notifications`；返回 `{"deleted": N}`
4. `mockRepository.DeleteAllByUser` + 4 个 handler_test（OK / 空 / 401 / 500）

---

## 出口门验证

| 门 | 结果 |
|---|---|
| `go build ./...` | ✅ |
| `go test ./internal/notification/...` | ✅ ok 0.76s |
| 新增 4 个 DeleteAll 测试 | ✅ 全 PASS |
| user 隔离断言 | ✅ bob 的消息未被 alice 的 DELETE /notifications 清掉 |

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| 用户隔离 | WHERE user_id = ?，与 5 个既有 API 一致 | ✅ |
| 路由冲突 | gin 自动区分 `DELETE /notifications` vs `DELETE /notifications/:id` | ✅ |
| 鉴权 | 401 检查 username | ✅ |
| 错误传播 | repo 错误 → HTTPStatusFromError → 500 | ✅ |
| 返回契约 | `{"deleted": int64}` 给前端"清空"按钮展示 toast | ✅ |
| 空消息处理 | 0 deleted 不算错误，返 200 + {"deleted": 0} | ✅ |

---

## 发现

无 CRITICAL / WARNING。

INFO：
- 一键清空对用户来说不可逆，本期不加二次确认（属前端 UI 关注点，C9 实施时 Popover 加 antd Popconfirm）
- 没加 metrics（清空动作低频，无观测价值）

---

## 结论

**PASS**

可合入，可继续 C5（task → notification 订阅器）。
