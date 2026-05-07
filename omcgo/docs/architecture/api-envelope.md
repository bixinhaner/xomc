# API 响应统一信封（v0.6 起）

> 本文档定义 OMC 后端所有 HTTP REST 接口的响应信封，以及前后端协同约定与迁移路线。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-07 | Backend/Frontend Team | 方案 D（response 包 + 渐进迁移 + 拦截器双兼容）落定；Phase 1 基础设施 + Phase 4 清理 23 处遗留 `{code,data,msg}` 完成 |

---

## 1. 决议（按方案 D + 全部推荐选项）

| # | 问题 | 决议 |
|---|------|------|
| Q1 | `ret` 数值约定 | **`1=成功 / 0=失败`**（按业务团队约定，与历史 `code=0=成功` 数值相反；本约定唯一权威）|
| Q2 | HTTP 状态码 | **保留语义**：成功 2xx / 失败 4xx/5xx；网关、监控、日志层继续依赖 |
| Q3 | 业务错误码 `biz_code` | **保留**（来自 `BusinessError.Code`），运维细粒度定位用；信封 `omitempty` |
| Q4 | `request_id` | **仅错误时包含**（`omitempty`），便于排障；成功路径不带 |
| Q5 | 已用 `{code,data,msg}` 的 23 处 | **一次性切换**到新信封（已完成，见 §6 进度）|
| Q6 | 流式响应（SSE / 文件下载）| **opt-out 不包装** —— handler 直接写 `c.Writer`，本包不介入 |
| Q7 | 实施节奏 | **分模块渐进**（Phase 2 按 admin → device → alarm → pm → ... 顺序）|
| Q8 | 兼容期前端拦截器 | **双兼容**：识别 `ret` 字段拆 data；无 ret 的裸响应原样透传（迁移完成后清理）|
| Q9 | 成功 `msg` 字段值 | **统一 `"ok"`**（`MsgOK` 常量）；保留 `OKWithMsg` 用于历史"创建成功 / 删除成功 / ..."迁移 |

---

## 2. 信封 schema

### 2.1 成功响应

```jsonc
HTTP 2xx
Content-Type: application/json; charset=utf-8

{
  "ret":  1,             // 固定 1 = 成功
  "msg":  "ok",          // 通常为 "ok"；OKWithMsg 时可自定义中文（如 "创建成功"）
  "data": <any> | null   // 业务数据节点；纯写操作（如 DELETE）使用 null
}
```

### 2.2 失败响应

```jsonc
HTTP 4xx / 5xx
Content-Type: application/json; charset=utf-8

{
  "ret":  0,                // 固定 0 = 失败
  "msg":  "device not found",  // 可读错误描述（来自 BusinessError.Message 或 err.Error()，已脱敏）
  "data": null,              // 失败时固定 null
  "biz_code":   1001,        // 可选：业务错误码（BusinessError.Code），用于程序化分支
  "request_id": "uuid-..."   // 可选：请求 ID，便于排障
}
```

### 2.3 流式响应（opt-out，不包装）

- SSE：`text/event-stream`
- 文件下载：`application/octet-stream` / 自定义 `Content-Type`
- 二进制（如导出 CSV）：直接 `c.Data(...)` 或 `c.Writer.Write(...)`

→ 这类 handler **不要**用 `response.OK`，直接走原生 Gin API。

---

## 3. 后端 helper（`internal/core/response`）

```go
package response

// 成功（HTTP 200 + msg="ok"）
func OK(c *gin.Context, data any)

// 成功 + 自定义 HTTP status（如 201 Created）
func OKWithStatus(c *gin.Context, statusCode int, data any)

// 成功 + 自定义 msg（迁移历史 "创建成功 / 查询成功 / ..." 用）
func OKWithMsg(c *gin.Context, data any, msg string)

// 失败 + HTTP status code + 简单 msg
func Fail(c *gin.Context, statusCode int, msg string)

// 失败 + 业务错误码（一般通过 errors.AbortWithError 间接调用）
func FailWithBizCode(c *gin.Context, statusCode int, msg string, bizCode int)
```

**错误路径优先用 `errors.AbortWithError`**，它会自动：
1. 从 `BusinessError` 取 `Code` → `biz_code`，`Message` → `msg`
2. 从 ctx 取 `request_id`（如果中间件已设置）
3. 通过 `redact.RedactJSON` 脱敏敏感字段
4. 输出标准失败信封

```go
// 推荐写法
if err := svc.DoSomething(ctx); err != nil {
    commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
    return
}
response.OK(c, result)
```

---

## 4. 前端 axios 拦截器（`http.ts`）

```ts
http.interceptors.response.use(
  (response) => {
    const body = response.data;
    if (body && typeof body === 'object' && 'ret' in body) {
      if (body.ret === 1) {
        return { ...response, data: body.data };  // 拆出 data 节点
      }
      if (body.ret === 0) {
        const err = new Error(body.msg);
        err.bizCode = body.biz_code;
        err.requestId = body.request_id;
        return Promise.reject(err);
      }
    }
    // 兼容期：未包装的裸响应原样返回（迁移完成后移除此分支）
    return response;
  },
  (error) => {
    // 失败响应：从 body.msg / body.message / body.details / body.error 提消息
    // body.biz_code → error.bizCode；body.request_id → error.requestId
    ...
  }
);
```

**业务侧无感知**：

```ts
// 后端响应：{ret:1, msg:"ok", data: {id: "...", name: "..."}}
const { data } = await http.get('/admin/users/123');
// data === {id: "...", name: "..."}  ← 已自动拆出 data 节点
```

---

## 5. 迁移规则（开发者必读）

### 5.1 必须改

| 原模式 | 新模式 |
|--------|--------|
| `c.JSON(http.StatusOK, role)` | `response.OK(c, role)` |
| `c.JSON(http.StatusOK, gin.H{"code":0, "data":x, "msg":"创建成功"})` | `response.OKWithMsg(c, x, "创建成功")` |
| `c.JSON(http.StatusCreated, user)` | `response.OKWithStatus(c, http.StatusCreated, user)` |
| `c.JSON(http.StatusNoContent, nil)` | `response.OK(c, nil)`（HTTP 改为 200 + body=`{ret:1,msg:"ok",data:null}`）|
| `c.AbortWithStatusJSON(401, gin.H{"code":401,"message":"..."})` | `commonerrors.AbortWithError(c, http.StatusUnauthorized, errors.New("..."))` |

### 5.2 不能改

| 原模式 | 不动 |
|--------|------|
| SSE 流：`c.Stream(...)` | 信封会破坏流式协议 |
| 文件下载：`c.File(...)` / `c.Data(...)` / `c.Writer.Write(...)` | 二进制不能被包装 |
| Webhook / 第三方对接：`c.JSON(200, externalSchema)` | 必须遵守第三方约定 |
| TR-069 ACS 端点 | 这是 SOAP 协议，不是 REST，不在本信封范围 |

---

## 6. 实施进度

### Phase 1：基础设施 + 错误路径 + 前端双兼容（v0.1 已完成）

- [x] 后端新建 `internal/core/response/` 包（5 个 helper）
- [x] `errors.ErrorResponse` 字段从 `{code, message, details, request_id}` 改为 `{ret, msg, data, biz_code?, request_id?}`
- [x] `errors.AbortWithError` 实现切换；保留 redact + request_id 注入逻辑
- [x] `errors_test.go` + `test/integration/api_sprint4_test.go` 断言更新
- [x] 前端 `http.ts` 拦截器双兼容：识别 `ret` 字段 → 拆 `data`；error onRejected 从 `msg` 优先提取
- [x] 前端拦截器附 `bizCode` / `requestId` 到 Error 对象，业务侧可读
- [x] `go test ./internal/core/errors/...` ✅
- [x] `go test ./test/integration/...` ✅
- [x] `npm run typecheck` ✅

### Phase 4：清理 23 处遗留 `{code, data, msg}`（v0.1 已完成）

清理涉及文件（5 个）：

| 文件 | 旧调用数 |
|------|---------|
| `internal/admin/dictionary_handler.go` | 9 |
| `internal/admin/auth_handler.go` | 3（含 1 处错误响应统一改为 AbortWithError）|
| `internal/admin/sys_config_handler.go` | 5 |
| `internal/admin/sys_log_handler.go` | 3 |
| `internal/admin/dead_letter_handler.go` | 4 |

`grep -rn "gin.H{\"code\": 0\|gin.H{\"code\":0" internal/` 现在返回 **0 个匹配**（除文档/注释中）。

### Phase 2：成功路径分模块迁移（待启动）

按模块顺序：admin → device → alarm → pm → mr → northbound → ...
每个模块单独 PR，做完一个验证 build + test 再进下一个。

#### 模块迁移清单（待办）

| 模块 | `c.JSON(http.StatusOK, x)` 调用数（约）| 状态 |
|------|---------------------------------------|------|
| admin（`*_handler.go` 各文件）| 多 | 🟡 部分迁移（dictionary/auth/sysConfig/sysLog/deadLetter 已完成）|
| device | 多 | ⏳ 未开始 |
| alarm | 多 | ⏳ |
| pm | 多 | ⏳ |
| mr | 中 | ⏳ |
| northbound | 多 | ⏳ |
| 其它（topology / software / backup / mml / ops / ... ）| 中 | ⏳ |

**关键提示**：兼容期内未迁移的 handler 仍返回裸业务体，前端拦截器直接透传 → 不影响业务。

### Phase 3：前端去兼容（待 Phase 2 全部完成后）

- 移除 `http.ts` 拦截器中「未包装裸响应原样返回」分支
- 业务接口若仍返回裸响应将报错（强制完成迁移）

---

## 7. 验证清单（DoD）

- [x] `internal/core/response/` 包提供 5 个 helper，函数签名稳定
- [x] `internal/core/errors/errors.go` 输出新信封；`AbortWithError` 内部生成 `{ret:0, msg, data:null, biz_code?, request_id?}`
- [x] 前端 `http.ts` 拦截器对成功 + 失败两路径都识别新信封
- [x] 旧 `gin.H{"code":0, ...}` 在 `internal/` 下零残留
- [x] `go build ./...` 全过
- [x] `go test ./...` 业务模块全过（`acs/rpc TestDownloadHandler` 与 `task` 是 pre-existing 失败，与本改造无关）
- [x] `npm run typecheck` 全过
- [ ] Phase 2 各模块迁移完毕（持续推进）
- [ ] Phase 3 前端拦截器去掉兼容分支
- [ ] 文档：开发者新建 handler 默认引用本文档第 5 节迁移规则

---

## 8. 兼容期 FAQ

**Q1**：未迁移的 handler 仍返回 `c.JSON(200, role)`，前端会怎样？

A：前端拦截器看到 body 没有 `ret` 字段，走兼容分支原样透传。`const { data } = await http.get(...)` 拿到的就是 role 对象，与未改造前一致。前端业务代码无需修改。

**Q2**：错误响应在迁移期会有「已经新信封 vs 仍是老信封」混合吗？

A：**不会**。`errors.AbortWithError` 是错误路径**唯一入口**（统一在 v0.1 切换），所有用 `AbortWithError` 的 219 处错误响应一次到位变成 `{ret:0, msg, data:null, biz_code?, request_id?}`。极少数手写 `c.AbortWithStatusJSON(...)` 的（如本次清理的 auth_handler.go:195 处 401）已统一改为 `AbortWithError`。

**Q3**：HTTP 状态码 vs `ret` 是否冗余？

A：**保留双重表达**。HTTP status 给网关/监控/日志层用；`ret` 给业务前端用。两者**总是同步**（成功 2xx + ret=1；失败 4xx/5xx + ret=0），不允许 HTTP 200 + ret=0 的不一致组合。

**Q4**：`biz_code` 与 HTTP status 又冗余？

A：HTTP status 是 HTTP 级语义（5 大类，~50 种）；`biz_code` 是业务级语义（已分配的几百个业务错误码，如 7012 = 账户锁定，1001 = 设备未找到）。粒度不同，前端可基于 `biz_code` 做精细化分支，HTTP status 只用于通用兜底。

**Q5**：前端如何区分 `ret=0` 业务错误 vs HTTP 4xx 网络/协议错误？

A：拦截器把两者都转成 `Promise.reject(err)`，业务侧统一 try/catch；如需细分：

```ts
try { ... } catch (e) {
  if (e.bizCode) {
    // 业务级错误（biz_code 7012 = 账户锁定）
  } else if (e.response?.status === 401) {
    // HTTP 鉴权错误
  }
}
```

---

## 9. 决策依据 / 历史

完整方案对比（A/B/C/D 四选一）见 git history 上的 PRD 讨论。最终选 **D**（response 包 + 渐进迁移 + 双兼容）的核心理由：

1. **最小风险**：错误路径一处改，成功路径分批改，前端拦截器双兼容，全程无中断
2. **HTTP 语义保留**：网关、监控、日志层无感
3. **可恢复**：每个模块独立 PR，回归点定位清晰
4. **业务码保留**：`biz_code` 让运维定位「7012 账户锁定」级精确，HTTP status 兜底
