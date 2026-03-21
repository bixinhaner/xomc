# test_acs_inform.sh - ACS Inform 测试脚本

> TR069/CWMP ACS Inform 消息测试工具，用于模拟 CPE 设备向 ACS 发送 Inform 请求。

## 用途

本脚本用于：
- ACS 引擎功能验证
- TR069 会话建立测试
- 设备注册流程调试
- 不同事件类型（Bootstrap/Periodic/Value Change）模拟

## 前置条件

- ACS 服务已启动并可访问
- 系统已安装 `curl` 命令
- 系统已安装 `uuidgen` 命令（可选，用于生成唯一 CWMP ID）

## 使用方法

```bash
./test_acs_inform.sh [ACS_URL] [EVENT_TYPE]
```

### 参数说明

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `ACS_URL` | 否 | `http://localhost:8080/smallcell/AcsService` | ACS 服务地址 |
| `EVENT_TYPE` | 否 | `periodic` | 事件类型，见下表 |

### 事件类型

| 参数值 | EventCode | 说明 |
|--------|-----------|------|
| `bootstrap` / `boot` | `0 BOOTSTRAP` | 设备首次注册/出厂重置后 |
| `periodic` | `2 PERIODIC` | 周期性心跳（默认） |
| `value` | `4 VALUE CHANGE` | 参数值变更通知 |

## 示例

### 基本用法（使用默认值）

```bash
# 发送 Periodic Inform 到本地 ACS
./test_acs_inform.sh
```

### 指定 ACS 地址

```bash
# 发送到远程 ACS 服务
./test_acs_inform.sh http://192.168.1.100:7547
```

### 发送 Bootstrap 事件

```bash
# 模拟设备首次注册
./test_acs_inform.sh http://localhost:8080/smallcell/AcsService bootstrap
```

### 发送 Value Change 事件

```bash
# 模拟参数变更通知
./test_acs_inform.sh http://localhost:8080/smallcell/AcsService value
```

## 输出示例

```
=== ACS Inform Test ===
URL: http://localhost:8080/smallcell/AcsService
Event: 2 PERIODIC
CWMP ID: cwmp-550e8400-e29b-41d4-a716-446655440000
========================

*   Trying 127.0.0.1:8080...
* Connected to localhost (127.0.0.1) port 8080
> POST /smallcell/AcsService HTTP/1.1
> Host: localhost:8080
> Content-Type: text/xml; charset=utf-8
> SOAPAction:
> Content-Length: 1856
>
* Request completely sent off
< HTTP/1.1 200 OK
< Content-Type: text/xml; charset=utf-8
< Set-Cookie: SESSION=abc123; Path=/; HttpOnly
<
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">cwmp-550e8400-e29b-41d4-a716-446655440000</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:InformResponse>
      <MaxEnvelopes>1</MaxEnvelopes>
    </cwmp:InformResponse>
  </soap:Body>
</soap:Envelope>
```

## SOAP 消息结构

### DeviceId

| 字段 | 值 | 说明 |
|------|-----|------|
| Manufacturer | Baicells | 设备厂商 |
| OUI | 001A2B | IEEE OUI（组织唯一标识符） |
| ProductClass | SmallCell-LTE | 产品类型 |
| SerialNumber | BCTest00123456 | 设备序列号 |

### ParameterList

脚本携带以下参数：

| 参数路径 | 类型 | 值 |
|----------|------|-----|
| Device.DeviceInfo.Manufacturer | string | Baicells |
| Device.DeviceInfo.ProductClass | string | SmallCell-LTE |
| Device.DeviceInfo.SerialNumber | string | BCTest00123456 |
| Device.DeviceInfo.HardwareVersion | string | v2.0 |
| Device.DeviceInfo.SoftwareVersion | string | 1.5.3.2 |
| Device.ManagementServer.ConnectionRequestURL | string | http://192.168.1.100:7547 |
| Device.ManagementServer.PeriodicInformInterval | unsignedInt | 300 |
| Device.Time.CurrentLocalTime | dateTime | 当前 UTC 时间 |

## 预期响应

### 成功响应 (HTTP 200)

- 返回 `InformResponse` SOAP 消息
- 包含 `MaxEnvelopes` 元素（通常为 1）
- 设置 `SESSION` Cookie（用于后续会话关联）

### 失败响应

| HTTP 状态码 | 可能原因 |
|-------------|----------|
| 400 | XML 格式错误 / SOAP 解析失败 |
| 401 | 认证失败（需 Digest/Basic 认证） |
| 429 | 请求被限流 |
| 500 | ACS 内部错误 |
| 503 | 服务不可用 / 并发会话数超限 |

## 调试技巧

### 查看完整请求/响应

```bash
# curl -v 参数已包含，可看到完整的 HTTP 头
./test_acs_inform.sh 2>&1 | less
```

### 检查 ACS 日志

```bash
# 查看 ACS 服务日志（根据实际部署位置调整）
tail -f /var/log/omcgo/acs.log
```

### 测试会话保持

```bash
# 发送 Inform 后，使用返回的 SESSION cookie 发送空 POST
# 这模拟了完整的 TR069 会话流程
SESSION_COOKIE=$(./test_acs_inform.sh 2>&1 | grep -oP 'SESSION=\K[^;]+')
curl -X POST http://localhost:8080/smallcell/AcsService \
  -H "Content-Type: text/xml" \
  -b "SESSION=$SESSION_COOKIE" \
  -d ""
```

## 相关文档

- [TR069/CWMP 协议规范](../doc/architecture/interface-topology.md)
- [ACS 会话状态机](../internal/acs/session.go)
- [SOAP 模板定义](../pkg/soap/templates.go)

## 扩展开发

如需自定义测试场景，可修改脚本中的以下部分：

1. **修改设备信息**：编辑 `DeviceId` 节点
2. **添加参数**：在 `ParameterList` 中添加 `ParameterValueStruct`
3. **新增事件类型**：扩展 `case` 语句块

```bash
# 示例：添加自定义事件类型
case "$EVENT_TYPE" in
  # ... 现有类型 ...
  alarm)
    EVENT_CODE="6 KICKED"
    COMMAND_KEY="alarm-key-001"
    ;;
esac
```
