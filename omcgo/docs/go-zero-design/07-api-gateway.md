# 07 — API 网关与北向接口

> go-zero REST API 设计、JWT/RBAC 中间件、OSS 北向推送

---

## 1. API 网关架构

```
                          外部访问
                             │
                    ┌────────┴────────┐
                    │   Nginx/Ingress │  ← 路径分发
                    └──┬──────┬──────┬┘
                       │      │      │
              ┌────────┘      │      └────────┐
              ▼               ▼               ▼
     ┌─────────────┐ ┌──────────────┐ ┌────────────┐
     │ device-api  │ │ monitor-api  │ │ admin-api  │
     │   :8080     │ │   :8081      │ │   :8082    │
     │             │ │              │ │            │
     │ 中间件链:    │ │ 中间件链:     │ │ 中间件链:   │
     │ ┌─────────┐ │ │ ┌─────────┐  │ │ ┌─────────┐│
     │ │Recovery │ │ │ │Recovery │  │ │ │Recovery ││
     │ │Logging  │ │ │ │Logging  │  │ │ │Logging  ││
     │ │Metrics  │ │ │ │Metrics  │  │ │ │Metrics  ││
     │ │Tracing  │ │ │ │Tracing  │  │ │ │Tracing  ││
     │ │JWT Auth │ │ │ │JWT Auth │  │ │ │JWT Auth ││
     │ │RBAC     │ │ │ │RBAC     │  │ │ │RBAC     ││
     │ └─────────┘ │ │ └─────────┘  │ │ └─────────┘│
     └──────┬──────┘ └──────┬───────┘ └──────┬─────┘
            │               │                │
            ▼               ▼                ▼
        zRPC 调用       zRPC 调用        zRPC 调用
```

---

## 2. .api 文件定义

### 2.1 device-api

```
// service/device/api/device.api
syntax = "v1"

info(
    title: "OMC Device API"
    desc: "设备管理、配置、拓扑、开站 REST API"
    version: "1.0"
)

// ========== 类型定义 ==========

type (
    // 通用分页
    PageReq {
        Page     int    `form:"page,default=1"`
        PageSize int    `form:"page_size,default=20"`
    }

    PageResp {
        Total int64 `json:"total"`
    }

    // 设备
    DeviceListReq {
        PageReq
        Carrier    string `form:"carrier,optional"`
        Technology string `form:"technology,optional"`
        Status     string `form:"status,optional"`
        OUI        string `form:"oui,optional"`
        SiteName   string `form:"site_name,optional"`
        Search     string `form:"search,optional"`
        GroupID    int64  `form:"group_id,optional"`
    }

    DeviceItem {
        ID                   int64  `json:"id"`
        SerialNumber         string `json:"serial_number"`
        OUI                  string `json:"oui"`
        ProductClass         string `json:"product_class"`
        Manufacturer         string `json:"manufacturer"`
        Carrier              string `json:"carrier"`
        Technology           string `json:"technology"`
        Status               string `json:"status"`
        FirmwareVersion      string `json:"firmware_version"`
        IPAddress            string `json:"ip_address"`
        SiteName             string `json:"site_name"`
        LastInformAt         int64  `json:"last_inform_at"`
        CreatedAt            int64  `json:"created_at"`
    }

    DeviceListResp {
        PageResp
        List []DeviceItem `json:"list"`
    }

    DeviceDetailResp {
        DeviceItem
        HardwareVersion      string `json:"hardware_version"`
        ConnectionRequestURL string `json:"connection_request_url"`
        DataModelID          int64  `json:"data_model_id"`
        UpdatedAt            int64  `json:"updated_at"`
    }

    UpdateDeviceReq {
        SiteName   string `json:"site_name,optional"`
        Carrier    string `json:"carrier,optional"`
        Technology string `json:"technology,optional"`
    }

    // 设备命令
    CommandResp {
        CommandID string `json:"command_id"`
        Queued    bool   `json:"queued"`
        Message   string `json:"message"`
    }

    SetParametersReq {
        Parameters []ParameterItem `json:"parameters"`
    }

    ParameterItem {
        Name  string `json:"name"`
        Value string `json:"value"`
        Type  string `json:"type"`
    }

    ParametersResp {
        Parameters []ParameterItem `json:"parameters"`
    }

    // 数据模型
    DataModelListReq {
        PageReq
        Carrier    string `form:"carrier,optional"`
        Technology string `form:"technology,optional"`
        Scope      string `form:"scope,optional"`
        Status     string `form:"status,optional"`
    }

    DataModelItem {
        ID           int64  `json:"id"`
        Carrier      string `json:"carrier"`
        Technology   string `json:"technology"`
        Scope        string `json:"scope"`
        OUI          string `json:"oui"`
        ProductClass string `json:"product_class"`
        Version      string `json:"version"`
        Status       string `json:"status"`
        CreatedAt    int64  `json:"created_at"`
    }

    DataModelListResp {
        PageResp
        List []DataModelItem `json:"list"`
    }

    // 拓扑
    TopologyTreeResp {
        Nodes []TreeNode `json:"nodes"`
    }

    TreeNode {
        GroupID     int64      `json:"group_id"`
        Name        string     `json:"name"`
        Type        string     `json:"type"`
        ParentID    int64      `json:"parent_id"`
        DeviceCount int        `json:"device_count"`
        Children    []TreeNode `json:"children"`
    }

    CreateGroupReq {
        Name        string `json:"name"`
        Type        string `json:"type"`
        ParentID    int64  `json:"parent_id,optional"`
        Description string `json:"description,optional"`
    }

    GroupResp {
        ID          int64  `json:"id"`
        Name        string `json:"name"`
        Type        string `json:"type"`
        ParentID    int64  `json:"parent_id"`
        Description string `json:"description"`
        DeviceCount int    `json:"device_count"`
    }

    // 开站任务
    ProvisioningTaskItem {
        TaskID       int64  `json:"task_id"`
        DeviceID     int64  `json:"device_id"`
        DeviceSerial string `json:"device_serial"`
        State        string `json:"state"`
        TemplateID   int64  `json:"template_id"`
        ErrorMessage string `json:"error_message"`
        RetryCount   int    `json:"retry_count"`
        StartedAt    int64  `json:"started_at"`
        CompletedAt  int64  `json:"completed_at"`
    }

    ProvisioningTaskListReq {
        PageReq
        State   string `form:"state,optional"`
        Carrier string `form:"carrier,optional"`
    }

    ProvisioningTaskListResp {
        PageResp
        List []ProvisioningTaskItem `json:"list"`
    }
)

// ========== 路由定义 ==========

@server(
    prefix: /api/v1
    group: device
    middleware: JwtAuthMiddleware,RbacMiddleware
)
service device-api {
    @doc "设备列表"
    @handler ListDevices
    get /devices (DeviceListReq) returns (DeviceListResp)

    @doc "设备详情"
    @handler GetDevice
    get /devices/:id returns (DeviceDetailResp)

    @doc "更新设备"
    @handler UpdateDevice
    put /devices/:id (UpdateDeviceReq) returns (DeviceDetailResp)

    @doc "删除设备"
    @handler DeleteDevice
    delete /devices/:id

    @doc "重启设备"
    @handler RebootDevice
    post /devices/:id/reboot returns (CommandResp)

    @doc "恢复出厂"
    @handler FactoryResetDevice
    post /devices/:id/factory-reset returns (CommandResp)

    @doc "Connection Request"
    @handler ConnectionRequest
    post /devices/:id/connection-request returns (CommandResp)

    @doc "读取参数"
    @handler GetParameters
    get /devices/:id/parameters returns (ParametersResp)

    @doc "写入参数"
    @handler SetParameters
    post /devices/:id/parameters (SetParametersReq) returns (CommandResp)
}

@server(
    prefix: /api/v1
    group: topology
    middleware: JwtAuthMiddleware,RbacMiddleware
)
service device-api {
    @doc "拓扑树"
    @handler GetTopologyTree
    get /topology/tree returns (TopologyTreeResp)

    @doc "创建设备组"
    @handler CreateGroup
    post /topology/groups (CreateGroupReq) returns (GroupResp)

    @doc "删除设备组"
    @handler DeleteGroup
    delete /topology/groups/:id

    @doc "分配设备到组"
    @handler AssignDevice
    post /topology/groups/:id/devices
}

@server(
    prefix: /api/v1
    group: datamodel
    middleware: JwtAuthMiddleware,RbacMiddleware
)
service device-api {
    @doc "数据模型列表"
    @handler ListDataModels
    get /datamodels (DataModelListReq) returns (DataModelListResp)

    @doc "激活数据模型"
    @handler ActivateDataModel
    post /datamodels/:id/activate

    @doc "导入数据模型"
    @handler ImportDataModel
    post /datamodels/import
}

@server(
    prefix: /api/v1
    group: provisioning
    middleware: JwtAuthMiddleware,RbacMiddleware
)
service device-api {
    @doc "开站任务列表"
    @handler ListProvisioningTasks
    get /provisioning/tasks (ProvisioningTaskListReq) returns (ProvisioningTaskListResp)

    @doc "开站任务详情"
    @handler GetProvisioningTask
    get /provisioning/tasks/:id returns (ProvisioningTaskItem)

    @doc "重试开站"
    @handler RetryProvisioning
    post /provisioning/tasks/:id/retry returns (ProvisioningTaskItem)
}
```

### 2.2 monitor-api（简要）

```
// service/monitor/api/monitor.api
@server(prefix: /api/v1, group: pm, middleware: JwtAuthMiddleware)
service monitor-api {
    @handler QueryCounters
    get /pm/counters returns (CounterListResp)

    @handler QueryKPI
    get /pm/kpi returns (KPIListResp)

    @handler GetKPIDefinitions
    get /pm/kpi/definitions returns (KPIDefListResp)

    @handler TriggerExport
    post /pm/export returns (ExportResp)
}

@server(prefix: /api/v1, group: alarm, middleware: JwtAuthMiddleware)
service monitor-api {
    @handler ListActiveAlarms
    get /alarms/active returns (AlarmListResp)

    @handler ListHistoryAlarms
    get /alarms/history returns (AlarmListResp)

    @handler AcknowledgeAlarm
    post /alarms/:id/acknowledge

    @handler ClearAlarm
    post /alarms/:id/clear

    @handler GetAlarmStatistics
    get /alarms/statistics returns (AlarmStatsResp)
}

@server(prefix: /api/v1, group: mr, middleware: JwtAuthMiddleware)
service monitor-api {
    @handler QueryMR
    get /mr/reports returns (MRListResp)
}

@server(prefix: /api/v1, group: northbound, middleware: ApiKeyMiddleware)
service monitor-api {
    @handler NorthboundPM
    get /northbound/pm returns (CounterListResp)

    @handler NorthboundAlarms
    get /northbound/alarms returns (AlarmListResp)

    @handler NorthboundSync
    post /northbound/sync returns (SyncResp)
}
```

### 2.3 admin-api（简要）

```
// service/admin/api/admin.api
@server(prefix: /api/v1, group: auth)
service admin-api {
    @handler Login
    post /auth/login returns (LoginResp)

    @handler RefreshToken
    post /auth/refresh returns (TokenResp)
}

@server(prefix: /api/v1, group: user, middleware: JwtAuthMiddleware,RbacMiddleware)
service admin-api {
    @handler ListUsers
    get /users returns (UserListResp)

    @handler CreateUser
    post /users returns (UserResp)

    @handler UpdateUser
    put /users/:id returns (UserResp)

    @handler DeleteUser
    delete /users/:id
}

@server(prefix: /api/v1, group: role, middleware: JwtAuthMiddleware,RbacMiddleware)
service admin-api {
    @handler ListRoles
    get /roles returns (RoleListResp)

    @handler CreateRole
    post /roles returns (RoleResp)

    @handler UpdatePermissions
    put /roles/:id/permissions
}

@server(prefix: /api/v1, group: audit, middleware: JwtAuthMiddleware,RbacMiddleware)
service admin-api {
    @handler ListAuditLogs
    get /audit/logs returns (AuditLogListResp)
}

@server(prefix: /api/v1, group: firmware, middleware: JwtAuthMiddleware,RbacMiddleware)
service admin-api {
    @handler ListFirmware
    get /firmware returns (FirmwareListResp)

    @handler UploadFirmware
    post /firmware returns (FirmwareResp)

    @handler TriggerUpgrade
    post /firmware/:id/upgrade returns (UpgradeTaskResp)
}
```

---

## 3. JWT 认证中间件

### 3.1 go-zero 内置 JWT

```go
// service/device/api/internal/middleware/jwtauthmiddleware.go

type JwtAuthMiddleware struct {
    secret string
}

func NewJwtAuthMiddleware(secret string) *JwtAuthMiddleware {
    return &JwtAuthMiddleware{secret: secret}
}

func (m *JwtAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // go-zero 内置 JWT 解析
        token := r.Header.Get("Authorization")
        if token == "" {
            httpx.ErrorCtx(r.Context(), w, errorx.NewCodeError(401, "未授权"))
            return
        }

        // 去掉 Bearer 前缀
        token = strings.TrimPrefix(token, "Bearer ")

        claims, err := jwt.Parse(token, m.secret)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, errorx.NewCodeError(401, "Token 无效"))
            return
        }

        // 将用户信息注入 context
        ctx := context.WithValue(r.Context(), "userID", claims["user_id"])
        ctx = context.WithValue(ctx, "username", claims["username"])
        ctx = context.WithValue(ctx, "roles", claims["roles"])

        next(w, r.WithContext(ctx))
    }
}
```

### 3.2 RBAC 中间件

```go
// service/device/api/internal/middleware/rbacmiddleware.go

type RbacMiddleware struct {
    adminRpc adminclient.AdminService
}

func (m *RbacMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := r.Context().Value("userID").(int64)

        // 从路由提取资源和操作
        resource := extractResource(r.URL.Path)  // e.g., "devices"
        action := mapMethodToAction(r.Method)     // GET→read, POST→create, PUT→update, DELETE→delete

        // 调用 admin-rpc 检查权限
        resp, err := m.adminRpc.CheckPermission(r.Context(), &admin.CheckPermissionReq{
            UserId:   userID,
            Resource: resource,
            Action:   action,
        })
        if err != nil || !resp.Allowed {
            httpx.ErrorCtx(r.Context(), w, errorx.NewCodeError(403, "权限不足"))
            return
        }

        next(w, r)
    }
}
```

---

## 4. 统一响应格式

```go
// common/result/response.go

type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

func Success(w http.ResponseWriter, data interface{}) {
    httpx.OkJsonCtx(context.Background(), w, &Response{
        Code:    0,
        Message: "success",
        Data:    data,
    })
}

func Error(w http.ResponseWriter, err error) {
    code := 500
    msg := "内部错误"

    if codeErr, ok := err.(*errorx.CodeError); ok {
        code = codeErr.Code
        msg = codeErr.Msg
    }

    httpx.WriteJsonCtx(context.Background(), w, http.StatusOK, &Response{
        Code:    code,
        Message: msg,
    })
}
```

---

## 5. 北向/OSS 接口（F08）

### 5.1 接口类型

| 类型 | 说明 | 认证方式 |
|------|------|---------|
| 拉取接口 | OSS 主动查询 PM/告警/配置 | API Key |
| 推送接口 | OMC 主动推送告警/PM 到 OSS | 目标地址 + 认证配置 |
| 全量同步 | 按需导出所有数据 | API Key |

### 5.2 推送引擎

```go
// service/monitor/api/internal/logic/northbound/pusher.go

type NorthboundPusher struct {
    targets []PushTarget
    nats    *nats.Conn
}

type PushTarget struct {
    Name     string
    URL      string
    AuthType string // api_key / basic / mtls
    AuthData map[string]string
    Events   []string // 订阅的事件类型
}

func (p *NorthboundPusher) Start() {
    // 订阅告警转发事件
    p.nats.QueueSubscribe("oss.alarm.forward", "nb-pushers",
        func(msg *nats.Msg) {
            for _, target := range p.targets {
                if contains(target.Events, "alarm") {
                    go p.push(target, msg.Data)
                }
            }
        })

    // 订阅 PM 导出事件
    p.nats.QueueSubscribe("oss.pm.export", "nb-pushers",
        func(msg *nats.Msg) {
            for _, target := range p.targets {
                if contains(target.Events, "pm") {
                    go p.push(target, msg.Data)
                }
            }
        })
}

func (p *NorthboundPusher) push(target PushTarget, data []byte) error {
    req, _ := http.NewRequest("POST", target.URL, bytes.NewReader(data))
    req.Header.Set("Content-Type", "application/json")

    switch target.AuthType {
    case "api_key":
        req.Header.Set("X-API-Key", target.AuthData["key"])
    case "basic":
        req.SetBasicAuth(target.AuthData["username"], target.AuthData["password"])
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("push to %s: %w", target.Name, err)
    }
    defer resp.Body.Close()
    return nil
}
```

### 5.3 北向认证（API Key）

```go
// common/middleware/apikey.go

type ApiKeyMiddleware struct {
    validKeys map[string]string // key → description
}

func (m *ApiKeyMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("X-API-Key")
        if apiKey == "" {
            apiKey = r.URL.Query().Get("api_key")
        }

        if _, ok := m.validKeys[apiKey]; !ok {
            httpx.ErrorCtx(r.Context(), w, errorx.NewCodeError(401, "无效的 API Key"))
            return
        }

        next(w, r)
    }
}
```

---

## 6. Swagger 文档

### 6.1 goctl 生成方式

```bash
# goctl 从 .api 文件直接生成 Swagger JSON
goctl api plugin -plugin goctl-swagger="swagger -filename device-api.json" \
    -api service/device/api/device.api \
    -dir service/device/api/doc

# 或使用 swaggo 注解方式
# 在 handler 函数上添加 @Summary, @Description, @Param, @Success 注释
```

### 6.2 Swagger UI 端点

每个 API 网关暴露 `/swagger/` 端点：

```go
// service/device/api/device.go (main)
func main() {
    // ...
    server := rest.MustNewServer(c.RestConf)

    // 挂载 Swagger UI
    server.AddRoute(rest.Route{
        Method:  http.MethodGet,
        Path:    "/swagger/:any",
        Handler: httpSwagger.WrapHandler,
    })
}
```
