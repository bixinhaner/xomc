# DD-16: 网元直连接口（F07）

> 关联功能域：F07（网元直连接口）
> 关联 backend-design.md 章节：第二章（模块 7 — nedirect）
> 实施阶段：Phase 4（北向与规模化）
> 依赖文档：DD-05, DD-09

---

## 1. 概述

### 1.1 模块定位

网元直连接口（`internal/nedirect/`）是网元设备与网管系统之间的直连通信通道，区别于标准 TR069 南向接口。**中国移动专有功能**，其他运营商不适用。

### 1.2 核心职责

- 直连接口协议处理
- 直连模式下的设备注册与认证
- 配置管理、状态监控、故障上报
- 与 TR069 双模共存

---

## 2. 接口设计

### 2.1 NE Direct Server — `internal/nedirect/server.go`

```go
type NEDirectServer struct {
    httpServer  *http.Server
    handler     *NEDirectHandler
    deviceRepo  device.DeviceRepository
    carrierReg  *carrier.CarrierRegistry
    logger      *zap.Logger
}

func NewNEDirectServer(cfg NEDirectConfig, deps NEDirectDeps) *NEDirectServer
func (s *NEDirectServer) Start() error
func (s *NEDirectServer) Shutdown(ctx context.Context) error
```

### 2.2 Handler — `internal/nedirect/handler.go`

```go
type NEDirectHandler struct {
    deviceService *device.DeviceService
    logger        *zap.Logger
}

// HandleRegister 直连模式设备注册
func (h *NEDirectHandler) HandleRegister(w http.ResponseWriter, r *http.Request)
// HandleConfig 直连模式配置管理
func (h *NEDirectHandler) HandleConfig(w http.ResponseWriter, r *http.Request)
// HandleStatus 直连模式状态上报
func (h *NEDirectHandler) HandleStatus(w http.ResponseWriter, r *http.Request)
// HandleFault 直连模式故障上报
func (h *NEDirectHandler) HandleFault(w http.ResponseWriter, r *http.Request)
```

---

## 3. 详细设计

### 3.1 与 TR069 的关系

- 互补而非替代：同一设备可同时支持 TR069 和直连
- 直连更适合高容量设备的实时管理
- TR069 负责标准化配置，直连负责运营商特有功能

### 3.2 运营商限制

通过 Carrier 接口判断：
```go
if cmcc, ok := carrier.(interface{ SupportsDirectConnection() bool }); ok && cmcc.SupportsDirectConnection() {
    // 启用直连接口
}
```

仅 CMCC 适配器返回 `true`。

---

## 4. 实施子阶段

### 阶段 16a：接口设计 + 基础实现（Phase 4）

**交付物**：server、handler 基础框架
**验证**：直连注册请求处理

---

## 5. 文件清单

```
internal/nedirect/server.go
internal/nedirect/handler.go
```

---

## 6. 参考

- doc/features/07-ne-direct-connection.md：F07 全部子功能
- 规范 #16：网元网管直连接口功能需求 V2.1.0（移动）
