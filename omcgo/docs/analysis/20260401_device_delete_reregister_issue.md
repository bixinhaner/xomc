# 设备删除后不重新注册问题分析

**日期**：2026-04-01  
**模块**：`internal/device`  
**严重程度**：中（数据一致性 Bug，手动运维操作后设备无法恢复入库）

---

## 一、问题描述

手动从数据库 `devices` 表删除某设备的记录后，CPE 重新发送 Inform 报文，
系统**不会重新向 `devices` 表写入该设备**，设备在系统侧永久"消失"。

---

## 二、根因分析

### 2.1 正常注册路径

```
CPE 发送 Inform (Bootstrap/Periodic/ValueChange)
  ↓ acs/handler.go 发布事件
  ↓ InformHandler.handleBootstrap / handlePeriodic
  ↓ RegisterFromInform / GetBySerialNumber
  ↓ getDeviceBySerialNumber
  ↓ Redis Cache Miss → deviceRepo.GetBySerialNumber (PostgreSQL)
  ↓ existing == nil → deviceRepo.Create → 写入 devices 表 ✓
```

### 2.2 问题路径（手动删 DB 后）

```
CPE 发送 Inform（Bootstrap / Periodic）
  ↓ InformHandler.handleBootstrap / handlePeriodic
  ↓ getDeviceBySerialNumber
  ↓ Redis Cache HIT（TTL = 10 分钟，缓存仍持有旧 device 对象）
  ↓ existing != nil → UpdateFromInform
  ↓ UPDATE devices SET ... WHERE id = <已删除的 UUID>
  ↓ 影响 0 行，但函数返回 nil error（静默成功）
  ↓ devices 表无任何写入 ✗
```

### 2.3 关键代码位置

| 文件 | 行号 | 说明 |
|------|------|------|
| `internal/device/device_cache.go` | L14-L16 | `deviceCacheTTL = 10 * time.Minute`，Redis TTL |
| `internal/device/device_cache.go` | L74-L97 | `GetOrLoad`：缓存命中直接返回，不回源 DB |
| `internal/device/service.go` | L349-L362 | `RegisterFromInform`：`existing != nil` → 走 Update 分支 |
| `internal/device/service.go` | L777-L785 | `getDeviceBySerialNumber`：Redis cache-aside 入口 |
| `internal/device/inform_handler.go` | L149-L174 | `handlePeriodic`：设备不存在时走 auto-register，但受缓存影响同样查不到 |

### 2.4 受影响的两条路径

| 事件类型 | 入口函数 | 是否受缓存影响 |
|---------|---------|--------------|
| Bootstrap / Boot | `handleBootstrap → RegisterFromInform` | **是**，`getDeviceBySerialNumber` 走 Redis |
| Periodic / ValueChange | `handlePeriodic → GetBySerialNumber → RegisterFromInform` | **是**，同一缓存路径 |

两条路径均受影响，行为完全一致。

---

## 三、修复方案

### 方案 A — Update 结果检查，0 行时降级 Create（推荐）

在 `deviceRepo.Update` 返回后检查影响行数，若为 0 说明记录已被删除，
自动降级为 `deviceRepo.Create` + `cache.Set`。

**优点**：无需修改删除逻辑，自愈能力强，即使 Delete 时忘记清缓存也能恢复。  
**缺点**：需要 Repository 接口返回 `rowsAffected`，需小范围改动接口。

### 方案 B — Delete 时同步清除 Redis 缓存

在 `deviceRepo.Delete` 之后调用 `cache.Delete(sn)`，确保下次 Inform
时缓存未命中，走 DB 查询 → `existing == nil` → `Create`。

**优点**：改动最小，逻辑清晰。  
**缺点**：依赖调用方记得清缓存，属于"约定"而非"防御"；若通过 SQL 工具
直接删除（绕过应用层），缓存依然残留，无法自愈。

### 推荐选择

**两个方案都实施**：
- 方案 B 作为主防线（正常删除流程保持一致性）
- 方案 A 作为兜底防御（应对直接操作 DB 的场景）

---

## 四、待修复文件清单

- `internal/device/service.go` — `UpdateFromInform` 检查 rowsAffected，0 行时 Create
- `internal/device/repo.go`（或 `batch_processor.go`）— `Update` 接口返回 `(int64, error)`
- `internal/device/service.go` — `DeleteDevice` 方法调用 `cache.Delete`

---

## 五、验证步骤

1. 运行服务，让某 CPE 完成正常注册
2. 直接执行 `DELETE FROM devices WHERE serial_number = 'xxx'`
3. 等待 CPE 下次 Periodic Inform（或手动触发 Bootstrap）
4. 检查 `devices` 表是否重新出现该 SN 的记录
5. 检查 `device_info` 表是否同步创建
