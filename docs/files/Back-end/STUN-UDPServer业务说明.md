# STUN UDP Server 业务规格说明书

> **文档版本**: v3.1
> **创建时间**: 2026年3月19日
> **文档用途**: 系统业务逻辑规格说明，用于AI代码生成
> **作者**: 尚颖彬

---

## 📋 文档说明

### 文档目标
本文档描述 stunUDPServer 系统的完整业务逻辑，不包含具体代码实现，可用于：
- AI 代码生成输入
- 系统架构设计参考
- 业务逻辑理解文档
- 新开发者入门指南

### 核心功能概述
stunUDPServer 是一个基于 Netty 的 UDP 服务器，用于：
1. **STUN 协议处理** - 支持标准和非标准 STUN 协议
2. **设备地址发现** - 获取设备在 NAT 后的公网 IP 和端口
3. **设备主动通知** - 通过 UDP Connection Request 触发设备立即发起 TR069 Inform 会话
4. **设备远程管理** - 支持设备重启、日志监控等管理功能

---

## 🏗️ 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────┐
│              Spring Boot 应用容器                        │
│  - 服务发现（Eureka）                                    │
│  - 定时任务调度                                          │
└──────────────────────────┬──────────────────────────────┘
                           │ SmartLifecycle
           ┌───────────────┴───────────────┐
           │                               │
    ┌──────▼─────────┐              ┌────────▼────────┐
    │  UDP Server    │              │  AppLifecycle   │
    │  (Netty NIO)    │              │  生命周期管理   │
    └──────┬─────────┘              └────────┬────────┘
           │                              │
    ┌──────┴──────────┐        ┌───────────┴──────────┐
    │ UDP 消息接收处理│        │ 初始化/销毁组件:      │
    └──────┬──────────┘        │ - UDP Server         │
           │                   │ - 消费线程           │
    ┌──────┴──────┐            │ - 批量保存线程       │
    │标准STUN处理 │            │ - 监控线程           │
    │非标准STUN  │            │ - 缓存管理           │
    └──────┬──────┘            └──────────────────────┘
           │
    ┌──────┴──────────────────┐
    │                         │
┌───▼────┐  ┌──────────┐  ┌───▼────┐  ┌─────────┐
│CPE缓存 │  │ ENB缓存  │  │ Redis  │  │RabbitMQ │
│(内存)  │  │ (内存)   │  │ 消息队列│  │ 消息队列│
└────────┘  └──────────┘  └────────┘  └─────────┘
```

### 分层架构

| 层级 | 职责 | 组件 |
|-----|------|------|
| **接入层** | UDP 协议接入、STUN 消息解析 | STUNServer、STUNMsgInboundHandler |
| **业务层** | 设备类型判断、消息处理、通知管理 | PacketProcessorStandard/NonStandard、ConnRequestMgr |
| **数据层** | 设备地址缓存 | CpeStunInfoStorage、EnbStunInfoStorage |
| **集成层** | 消息队列 | Redis 消费者、RabbitMQ 监听器 |
| **监控层** | 缓存监控、JVM 监控 | DeleteTimeoutStunInfoExecutor、JvmInfoPrintThread |

---

## 🔧 核心模块业务逻辑

### 1. UDP 服务器模块

#### 业务职责
- 监听指定 UDP 端口（默认 3478）
- 接收设备上报的 STUN 消息
- 将消息分发到对应的处理器

#### 消息分发逻辑
```
接收 UDP 数据包
  ↓
尝试按标准 STUN 协议解析
  ├─ 解析成功 → PacketProcessorStandard 处理（CPE 设备）
  └─ 解析失败 → PacketProcessorNonStandard 处理（基站设备）
```

#### 关键配置
- **监听端口**: 可配置（通常为 3478）
- **工作线程数**: 可配置（建议为 CPU 核心数的 1-2 倍）
- **缓冲区分配器**: 使用池化分配器提升性能

---

### 2. 标准 STUN 消息处理（CPE 设备）

#### 业务职责
接收 CPE 设备发送的标准 STUN Binding Request，**记录设备的公网地址**到本地缓存，用于后续主动通知

#### 核心业务目标
**主要目的**: 服务器端获取并记录设备的公网 IP:Port
- 设备通常位于 NAT/防火墙后，其公网地址对设备本身可能是透明的
- 服务器需要这个地址才能通过 Connection Request 主动联系设备
- 返回 Binding Response 是 STUN 协议要求，但不是主要业务目的

#### 消息处理流程

**输入**:
- 标准 STUN Binding Request 消息（基于 TR069 规范）
- UDP 包源地址（设备的公网 IP:Port）

**处理步骤**:
1. 解析 STUN 消息头和属性
2. 验证消息类型是否为 Binding Request
3. **从 UDP 包源地址提取设备的公网 IP:Port**（核心步骤）
4. 提取可选属性 ResponseAddress
5. **记录设备公网地址到本地缓存** `CpeStunInfoStorage`（核心步骤）
6. 构造 Binding Response 消息（协议要求）
7. 添加关键属性：
   - **MappedAddress**: 设备的公网 IP:Port（回送给设备，协议要求）
   - **ChangedAddress**: 服务器主 IP:Port（无实际意义，协议填充）
   - **SourceAddress**: 服务器主 IP:Port（无实际意义，协议填充）
8. 确定响应地址（优先使用 ResponseAddress，否则使用源地址）
9. 通过 UDP 发送响应给设备（协议要求）

**输出**:
- **核心输出**: 设备公网地址保存到 `CpeStunInfoStorage` 本地缓存
- **协议输出**: UDP Binding Response 消息（满足 STUN 协议要求）

#### 数据流向
```
CPE 设备（NAT 后）
  ↓ 发送 STUN Binding Request
  ↓ UDP 包源地址 = 设备公网 IP:Port
stunUDPServer
  ↓ 提取源地址
  ↓ 记录到 CpeStunInfoStorage
  ↓ 构造 Binding Response
  ↓ 发送响应
CPE 设备收到响应
```

#### 异常处理
- **解析失败**: 返回 Binding Error Response
- **未知属性**: 返回错误响应及未知属性列表

---

### 3. 非标准 STUN 消息处理（基站设备）

#### 业务职责
处理基站设备的非标准 STUN 消息，获取并保存设备公网地址

#### 消息处理流程

**输入**:
- 非 STUN 协议的 UDP 包
- 包内容为基站 SN 号（UTF-8 编码的纯文本）
- UDP 包源地址（基站的公网 IP:Port）

**处理步骤**:
1. 解析 SN 号（UTF-8 解码）
2. 提取源 IP 和端口
3. 通过 SN 号查询设备编码（多层缓存策略）
4. 保存设备地址信息到本地缓存
5. 构造确认消息 "echoreply"
6. 通过 UDP 发送确认给基站

**输出**:
- 基站地址信息保存到缓存
- UDP 确认消息 "echoreply"

#### 设备编码查询策略（多层缓存）
```
第一层: 本地缓存（EnbLocalCache）
  ↓ 未命中
第二层: Redis 缓存
  ↓ 未命中
第三层: 数据库查询
  ↓
更新所有缓存层
```

---

### 4. 连接请求管理模块

#### 业务职责
通过 UDP Connection Request 主动通知设备，触发设备立即发起 TR069 Inform 会话

#### TR069 会话模式背景

**TR069 协议限制**:
- 设备必须主动向 OMC 发起会话
- OMC 无法直接建立连接到设备
- 设备定期心跳 Inform（通常 60s-300s 间隔）

**Connection Request 解决方案**:
- OMC 通过 UDP 主动通知设备
- 设备收到通知后立即发起新的 Inform 会话
- 响应时间从分钟级降低到秒级

#### 连接请求发送流程

**输入**:
- 设备编码（deviceCode）

**处理步骤**:
1. 判断设备类型（基站或 CPE）
2. 从本地缓存查询设备 STUN 信息
   - 基站: EnbStunInfoStorage
   - CPE: CpeStunInfoStorage
3. 获取设备公网 IP:Port
4. 根据设备类型生成对应格式的 Connection Request 消息
5. 通过 UDP 发送给设备
6. 记录日志

**输出**:
- UDP Connection Request 消息发送到设备公网地址

#### 基站 Connection Request（非标准格式）

**消息格式**:
- 内容: 纯文本字符串 `"infromrequest"`
- 编码: UTF-8
- 目标: 设备公网 IP:Port

#### CPE Connection Request（类 HTTP 格式的 UDP 消息）

**💡 通俗理解：这就是一个"伪装成 HTTP 的纸条"**

想象一下你要给朋友发一个通知：
- 你把通知内容写在一张纸条上
- 纸条上写着类似 `GET http://...` 这样的格式（像网页浏览器的请求）
- 但你不是用电子邮件（HTTP）发送
- 而是叫快递员（UDP）把这张纸条直接送到朋友家

**这就是"类HTTP格式的UDP消息"的本质！**

**实际发送的文本示例**：
```
GET http://203.0.113.100:3478/?ts=1647852345123&id=12345&un=dps&cn=67890&sig=ABC123DEF456 HTTP/1.1

```

**技术细节**：
- **传输方式**: UDP（像发快递一样直接送）
- **消息内容**: HTTP GET 格式的文本字符串（纸条内容）
- **编码方式**: UTF-8

**URL 参数详解**：

| 参数 | 中文意思 | 举例 | 作用 |
|------|---------|------|------|
| **ts** | 时间戳 | `1647852345123` | 防止坏人重复使用旧消息 |
| **id** | 随机编号 | `12345` | 唯一标识这次通知 |
| **un** | 用户名 | `dps` | 验证身份（默认值） |
| **cn** | 随机数 | `67890` | 增加随机性 |
| **sig** | 签名 | `ABC123DEF456` | 防止消息被篡改（最重要） |

**🔐 签名验证（通俗解释）**：

**签名的目的**：确保这个通知真的是OMC服务器发的，不是坏人伪造的

**签名计算过程**：
```
1. 拼接字符串：
   "1647852345123" + "12345" + "dps" + "67890"
   = "164785234512312345dps67890"

2. 用密钥 "dps" 对这个字符串进行加密（HMAC-SHA1算法）
   得到：ABC123DEF456...

3. 把签名放到消息里：
   &sig=ABC123DEF456...
```

**设备端验证**：
```
1. 收到UDP包，解析出文本内容
2. 提取 ts, id, un, cn, sig
3. 用同样的密钥 "dps" 重新计算签名
4. 比对计算结果和收到的 sig 是否一致
5. 一致就执行，不一致就忽略（可能是伪造的）
```

**⚠️ 重要澄清**：
- ❌ 这不是真正的HTTP连接（没有TCP握手）
- ✅ 只是消息内容长得像HTTP请求
- ✅ 底层用UDP协议发送（直接扔数据包）
- ✅ 设备收到后解析文本内容，验证签名

**🔄 重试机制**：
- 默认发送 3 次（可配置）
- 原因：UDP不可靠，可能丢包
- 无需等待设备回复（发完就行）

**🤔 为什么CPE要用这么复杂的格式？**

| 方面 | 基站（简单） | CPE（复杂） |
|------|------------|-----------|
| **消息内容** | 纯文本 "infromrequest" | 完整的HTTP格式+签名 |
| **验证机制** | 无 | HMAC-SHA1签名 |
| **安全性** | 低 | 高（防伪造） |
| **原因** | 厂商自定义格式 | **这是 TR069 协议的标准要求** |

#### 消息队列使用说明

**stunUDPServer 模块使用了两种消息队列系统，各有不同的用途**：

| 队列类型 | 用途 | 队列名称 | 生产者 | 消费者 |
|---------|------|---------|--------|--------|
| **Redis 队列** | 设备主动通知（Connection Request） | `udpConnRequestQueue` | MethodExecuteHandler 模块 | `NoticeENodeBInformThread` |
| **RabbitMQ 队列** | 基站远程重启（Reboot） | `QUEUE_NAME_ENB_STUN_REBOOT` | OMC 管理界面或其他模块 | `RebootMsgListener` |

**⚠️ 重要说明**：
- **RabbitMQ 仅用于基站重启功能**
- **Connection Request 使用 Redis 队列**，不使用 RabbitMQ
- 两种队列各自独立，互不干扰

---

#### 完整业务流程（Connection Request）

```
┌──────────────┐
│   OMC 系统   │
│ 需要通知设备 │
└──────┬───────┘
       │ 1. 推送设备编码到 Redis
       ▼
┌─────────────────────────────┐
│ Redis 队列:                 │
│ udpConnRequestQueue         │
└──────┬──────────────────────┘
       │ 2. 消费队列（多线程）
       ▼
┌─────────────────────────────┐
│ NoticeENodeBInformThread     │
│ 调用 ConnRequestMgr         │
└──────┬──────────────────────┘
       │ 3. 查询缓存、生成消息
       ▼
┌─────────────────────────────┐
│ UDP Server                  │
│ 发送 Connection Request     │
└──────┬──────────────────────┘
       │ 4. UDP 通知
       ▼
┌─────────────────────────────┐
│ 设备（CPE/基站）            │
│ 收到通知                    │
└──────┬──────────────────────┘
       │ 5. 立即发起 Inform 会话
       ▼
┌─────────────────────────────┐
│ OMC TR069 服务器            │
│ 接收 Inform，建立会话       │
└──────┬──────────────────────┘
       │ 6. 下发指令
       ▼
┌─────────────────────────────┐
│ 设备执行指令                │
│ 上报结果                    │
└─────────────────────────────┘
```

### 5. 消息队列消费模块

#### 业务职责
从 Redis 队列消费设备编码，触发连接请求发送

#### 消费逻辑

**队列信息**:
- Redis Key: `udpConnRequestQueue`
- 数据结构: List
- 操作方式: FIFO（先进先出）

**消费流程**:
1. 从队列左侧弹出一条数据（lpop）
2. 如果队列为空，休眠 1 秒后继续
3. 如果取到数据，调用 ConnRequestMgr 发送连接请求
4. 循环执行直到应用停止

**并发控制**:
- 支持多线程并发消费（可配置线程数）
- 每个线程独立消费，提高吞吐量
- 典型配置: 2-4 个消费线程

---

### 6. CPE STUN 信息完整流转流程

#### 6.1 业务背景

**问题**: CPE 设备如何将自己的公网地址告知 OMC 系统？

**解决方案**: 采用两阶段上报机制
1. **第一阶段**: CPE 通过 STUN 协议从 stunUDPServer 获取自己的公网地址
2. **第二阶段**: CPE 通过 TR-069 Inform 消息将公网地址上报给 OMC，最终存储到 stunUDPServer 内存

---

#### 6.2 完整数据流转图

```
┌─────────────┐                    ┌─────────────────┐
│    CPE      │                    │ stunUDPServer   │
│  (NAT后)     │                    │   :3478         │
└──────┬──────┘                    └────────┬────────┘
       │                                   │
       │ ① STUN Binding Request           │
       │──────────────────────────────────>│
       │    (UDP包源地址=公网IP:Port)       │
       │                                   │
       │ ② STUN Binding Response           │
       │<──────────────────────────────────│
       │  MappedAddress: 公网IP:Port       │
       │                                   │
       │ ③ CPE 得知自己的公网地址           │
       │                                   │
       │ ④ TR-069 Inform                   │  ┌──────────────┐
       │  UDPConnectionRequestAddress      │  │ TR069Server  │
       │  = "公网IP:Port"                  │  │              │
       │────────────────────────────────────────────────────>│
       │                                         │          │
       │                                         │ ⑤-⑦     │
       │                                         │ 格式化/队列│
       │                                         │ /Redis   │
       │                                         │          │
       │  ⑧ BatchSaveStandardStunAddrInfoThread │          │
       │                                         │          │
       │  ⑨ 从Redis LPOP                        │          │
       │  ┌─────────────────────────────────────┘          │
       │  │                                                  │
       │  ▼                                                  ▼
       │  ┌──────────────────────────────────────────────────────┐
       │  │            CpeStunInfoStorage                        │
       │  │            (stunUDPServer 本地内存)                   │
       │  └──────────────────────────────────────────────────────┘
       │
       ▼
  ⑩ 用于 Connection Request 主动通知
```

---

#### 6.3 详细流程说明

##### 步骤①-③: CPE 通过 STUN 获取公网地址

1. CPE 向 stunUDPServer 发送 STUN Binding Request
2. stunUDPServer 从 UDP 包的**源地址**提取 CPE 的公网 IP 和端口
3. stunUDPServer 构造 Binding Response，在 **MappedAddress** 属性中填入公网地址
4. stunUDPServer 发送响应给 CPE
5. CPE 解析响应，从 MappedAddress 属性中得知自己的公网地址

##### 步骤④-⑦: CPE 通过 Inform 上报公网地址

1. CPE 发送 TR-069 Inform 消息到 TR069Server
2. Inform 消息中携带参数 `UDPConnectionRequestAddress`，值为 `"公网IP:端口"`
3. TR069Server 解析该参数，提取 IP 和端口
4. TR069Server 格式化为 `"设备编码_公网IP_端口"`（如 `CPE001_1.2.3.4_5000`）
5. 加入 DataBufferHandler 的内存队列 `cpeStunAddrQueue`
6. CpeStunAddrThread 后台线程循环刷新，批量将数据 RPUSH 到 Redis 的 `cpeStunAddr` 队列

##### 步骤⑧-⑨: stunUDPServer 从 Redis 消费并存储

1. BatchSaveStandardStunAddrInfoThread 后台线程从 Redis LPOP 数据
2. 解析字符串 `"设备编码_公网IP_端口"`
3. 调用 CpeStunInfoStorage.addCpeStunInfo() 存储到本地内存
4. 最终数据结构: `{"设备编码": {"ip": "公网IP", "port": 端口, "timeSeconds": "时间戳"}}`

##### 步骤⑩: 使用 STUN 信息主动通知 CPE

1. OMC 需要主动通知 CPE 时，调用 ConnRequestMgr
2. 先从 EnbStunInfoStorage 查询（未命中）
3. 再从 CpeStunInfoStorage 查询 CPE 的公网地址
4. 获取到公网 IP 和端口后，发送 UDP Connection Request

---

#### 6.4 数据格式变化

| 阶段 | 位置 | 数据格式示例 |
|------|------|------------|
| STUN 响应 | MappedAddress 属性 | `1.2.3.4:5000` |
| Inform 参数 | UDPConnectionRequestAddress | `1.2.3.4:5000` |
| CPEService | 格式化字符串 | `CPE001_1.2.3.4_5000` |
| Redis 队列 | cpeStunAddr | `["CPE001_1.2.3.4_5000", ...]` |
| 内存缓存 | CpeStunInfoStorage | `{"ip":"1.2.3.4", "port":5000, ...}` |

---

#### 6.5 关键组件职责

| 组件 | 所属模块 | 职责 |
|------|---------|------|
| PacketProcessorStandard | stunUDPServer | 处理 STUN 请求，提取公网地址并返回给 CPE |
| Inform | TR069Server | 解析 Inform 消息，提取 UDPConnectionRequestAddress 参数 |
| CPEService | TR069Server | 格式化数据，加入内存队列 |
| DataBufferHandler | TR069Server | 内存队列管理，刷新到 Redis |
| CpeStunAddrThread | TR069Server | 后台线程，循环刷新到 Redis |
| BatchSaveStandardStunAddrInfoThread | stunUDPServer | 后台线程，从 Redis 消费并存储到内存 |
| CpeStunInfoStorage | stunUDPServer | 本地内存缓存，存储 CPE STUN 信息 |
| ConnRequestMgr | stunUDPServer | 主动通知时查询 STUN 信息 |

---

#### 6.6 设计特点

**为什么采用两阶段上报？**
- STUN 服务器只能被动响应，无法主动存储设备地址
- TR-069 协议要求 CPE 必须主动上报所有参数信息
- 职责分离：stunUDPServer 专注 STUN 协议，TR069Server 专注设备管理

**为什么使用 Redis 作为中间缓冲？**
- 解耦模块：TR069Server 和 stunUDPServer 独立部署
- 异步处理：避免阻塞 Inform 消息处理
- 数据持久化：Redis 提供临时持久化，防止数据丢失

**为什么最终存储在本地内存？**
- 高性能查询：O(1) 时间复杂度
- 数据时效性：NAT 映射随时过期，存储旧地址无意义
- 简化设计：避免数据库和网络操作

---

### 7. 数据存储模块

#### 7.1 CPE 设备地址存储

**存储位置**: CpeStunInfoStorage（本地内存缓存）

**数据结构**:
```
Key: 设备编码（deviceCode）
Value: {
  ip: 设备公网 IP,
  port: 设备公网端口,
  timeSeconds: 时间戳（秒）
}
```

**特点**:
- 线程安全（ConcurrentHashMap）
- O(1) 查询速度
- 应用重启后数据丢失

**数据来源**:
- PacketProcessorStandard 处理 STUN 消息后直接保存

#### 7.2 基站设备地址存储

**存储位置**: EnbStunInfoStorage（本地内存缓存）

**数据结构**: 同 CPE

**数据来源**:
- PacketProcessorNonStandard 处理非 STUN 消息后直接保存

#### 7.3 基站编码映射缓存

**存储位置**: EnbLocalCache（本地内存缓存）

**数据结构**:
```
Key: 基站 SN 号
Value: 基站设备编码（ENB Code）
```

**用途**:
- 通过 SN 号快速查询设备编码
- 多层缓存策略（本地 → Redis → 数据库）

---

### 8. RabbitMQ 基站重启功能

#### 业务职责
通过 RabbitMQ 消息队列接收基站重启指令，构造并发送加密的 UDP 重启命令

#### 业务场景
**注意**: 只有基站异常时需要用这种方式

- 设备运行异常需要重启恢复

#### 重启流程

**消息接收**:
- 监听 RabbitMQ 队列: `QUEUE_NAME_ENB_STUN_REBOOT`
- 消息格式: 逗号分隔的基站编码（如 `"ENB_001,ENB_002"`）
- 并发消费: 1-5 个消费者（可配置）

**重启命令构造**:
1. 从本地缓存获取基站 STUN 信息
2. 提取基站 SN（从设备编码中提取）
3. 构造加密消息: `/restart_{md5_hash}`
4. MD5 加密源文: 基站 SN + 配置的密钥后缀
5. 通过 UDP 发送到设备公网地址

**设备端验证**（推测）:
1. 接收 UDP 消息
2. 验证消息格式（`/restart_` 开头）
3. 使用本地 SN + 相同后缀计算 MD5
4. 比对 MD5 是否匹配
5. 匹配则执行重启，否则忽略

**安全机制**:
- MD5 签名防止未授权重启
- 密钥后缀可配置
- 设备端验证签名

---

### 9. 缓存大小监控模块

#### 业务职责
定期打印本地缓存大小统计，用于监控内存使用和容量规划

#### 监控指标
- 基站 STUN 信息缓存大小（EnbStunInfoStorage）
- CPE STUN 信息缓存大小（CpeStunInfoStorage）
- 基站 SN→编码映射缓存大小（EnbLocalCache）

#### 工作机制

**监控流程**:
1. 定时触发（每 30 秒一次）
2. 读取各缓存的 size
3. 打印日志
4. 循环执行

**日志示例**:
```
Enb stun info size: 1523
Cpe stun info size: 8456
Enb snEnbCode cache size: 1523
```

---

## 🔄 完整业务场景

### 场景 1: CPE 设备地址发现

```
1. CPE 设备发送 STUN Binding Request
   - 目标: stunUDPServer:3478
   - 设备通常位于 NAT/防火墙后

2. UDP Server 接收并分发
   - 识别为标准 STUN 消息
   - 分发到 PacketProcessorStandard

3. 标准消息处理
   - 解析 STUN 消息头和属性
   - 【核心】从 UDP 包源地址提取设备公网 IP:Port
   - 【核心】记录到 CpeStunInfoStorage（本地缓存）
   - 构造 Binding Response（协议要求）
   - 添加 MappedAddress（回送设备公网地址）

4. 发送响应
   - UDP 发送 Binding Response 给设备（协议要求）

5. 服务器端获得设备地址
   - CpeStunInfoStorage 保存设备公网 IP:Port
   - 用于后续 Connection Request 主动通知
```

**业务价值**:
- **主要目的**: 服务器端记录设备公网地址
- **关键用途**: 使得服务器可以主动联系设备（Connection Request）
- **协议要求**: 返回 Binding Response 满足 STUN 协议标准

---

### 场景 2: 基站设备地址发现

```
1. 基站发送非 STUN 消息
   - 内容: 基站 SN 号（UTF-8 编码）
   - 目标: stunUDPServer:3478

2. UDP Server 接收并分发
   - 尝试标准 STUN 解析失败
   - 分发到 PacketProcessorNonStandard

3. 非标准消息处理
   - 解析 SN 号
   - 提取设备公网 IP:Port
   - 通过 SN 查询设备编码（多层缓存）
   - 构造确认消息 "echoreply"

4. 发送确认
   - UDP 发送 "echoreply" 给基站

5. 保存地址信息
   - 保存到 EnbStunInfoStorage（本地缓存）

6. 基站收到确认
   - 确认服务器已收到
```

### 场景 3: OMC 主动通知设备（Connection Request）

```
1. OMC 管理系统发起通知
   - 原因: 配置变更、软件升级、故障处理等
   - 操作: 将设备编码推送到 Redis 队列

2. Redis 队列缓冲
   - 队列: udpConnRequestQueue
   - 数据: 设备编码列表

3. 消费线程处理
   - NoticeENodeBInformThread 从队列消费
   - 多线程并发提高吞吐量

4. 连接请求管理
   - ConnRequestMgr 处理
   - 判断设备类型（基站/CPE）
   - 查询设备 STUN 信息（本地缓存）
   - 获取设备公网 IP:Port
   - 生成对应格式的 Connection Request

5. UDP 发送通知
   - 基站: 发送 "infromrequest"
   - CPE: 发送类 HTTP 格式的文本消息（带签名）
   - 支持 3 次重试

6. 设备接收通知
   - 验证消息合法性
   - 立即发起 TR069 Inform 会话

7. OMC 接收 Inform
   - 建立 TR069 会话
   - 下发各种指令
   - 设备执行并上报结果
```

### 场景 4: 基站远程重启

```
1. 运维人员发起重启
   - OMC 管理界面操作
   - 选择需要重启的基站

2. 推送重启指令
   - 将基站编码推送到 RabbitMQ 队列
   - 队列: QUEUE_NAME_ENB_STUN_REBOOT
   - 消息: "ENB_001,ENB_002,ENB_003"

3. RabbitMQ 消费
   - RebootMsgListener 监听队列
   - 并发消费（1-5 个线程）
   - 解析逗号分隔的设备编码

4. 逐个处理重启
   - 调用 RebootService.reboot(deviceCode)
   - 从本地缓存获取基站 STUN 信息
   - 提取基站 SN
   - 构造加密消息: /restart_{md5}

5. UDP 发送重启命令
   - 发送到基站公网 IP:Port
   - 携带 MD5 签名验证

6. 基站接收并验证
   - 验证消息格式
   - 计算本地 MD5
   - 比对签名
   - 匹配则执行重启
```

---

## 🧵 多线程模型

### 线程类型与职责

| 线程类型 | 数量 | 生命周期 | 核心职责 |
|---------|------|---------|---------|
| **Netty EventLoop** | 可配置 | 应用全程 | UDP 报文接收/发送 |
| **NoticeENodeBInformThread** | 可配置（2-4） | 应用全程 | 消费 Redis 队列，发送连接请求 |
| **BatchSaveStandardStunAddrInfoThread** | 1 | 应用全程 | ⚠️ 尝试消费 Redis 队列（队列为空，未实际工作） |
| **DeleteTimeoutStunInfoExecutor** | 1 | 应用全程 | 定时打印缓存大小（30 秒） |
| **RabbitMQ Reboot Consumers** | 可配置（1-5） | 应用全程 | 消费基站重启消息 |

### 线程间协作

```
Netty EventLoops (NIO)
  ↓ 接收 UDP 包
STUNMsgInboundHandler
  ↓ 消息分发
PacketProcessorStandard/NonStandard
  ↓ 保存到本地缓存
CpeStunInfoStorage / EnbStunInfoStorage

Redis Queue: udpConnRequestQueue
  ↓ 消费
NoticeENodeBInformThread × N
  ↓ 调用
ConnRequestMgr
  ↓ 查询缓存
  ↓ 通过 Netty 发送 UDP 消息

RabbitMQ Queue: QUEUE_NAME_ENB_STUN_REBOOT
  ↓ 消费
RebootMsgListener × N
  ↓ 调用
RebootService
  ↓ 通过 Netty 发送 UDP 消息
```

---

## ⚙️ 关键配置参数

### UDP 服务器配置
```properties
# UDP 监听端口
listenPort=3478

# Netty 工作线程数
nettyWorkThreadCount=4
```

## 🎯 核心业务价值

### 1. TR069 协议限制的突破

**问题**: TR069 协议要求设备主动发起会话，OMC 无法主动联系设备

**解决**:
- 使用 UDP Connection Request 主动通知
- 设备收到后立即发起 Inform 会话
- 响应时间从分钟级（60s-300s）降低到秒级

### 2. NAT 穿透和地址发现

**业务价值**:
- **服务器端获取**: 记录 NAT 后设备的公网 IP:Port
- **主动联系能力**: 使得服务器可以通过 Connection Request 主动联系设备
- **协议支持**: 支持标准（CPE）和非标准（基站）两种协议
- **快速查询**: 本地缓存实现 O(1) 查询速度

### 3. 高性能异步处理

**优势**:
- Redis 队列缓冲，解耦组件
- 多线程并发消费
- 本地缓存 O(1) 查询
- Netty NIO 高性能网络通信

### 4. 完整的设备管理能力

**功能**:
- 设备远程重启（RabbitMQ + UDP）
- 缓存大小监控
- JVM 状态监控


#### 实际数据存储架构

```
CPE STUN 信息 → CpeStunInfoStorage（ConcurrentHashMap）
  - 本地内存缓存
  - O(1) 查询速度
  - 应用重启后丢失

基站 STUN 信息 → EnbStunInfoStorage（ConcurrentHashMap）
  - 本地内存缓存
  - O(1) 查询速度
  - 应用重启后丢失

```

#### 为什么只使用本地缓存？

**设计原因**:
1. **性能优先**: 本地缓存（ConcurrentHashMap）查询速度极快，O(1) 时间复杂度
2. **数据实效性强**: STUN 地址信息时效性很短，设备几分钟重新上线后地址可能变化
   - NAT 映射可能随时过期
   - 设备重启后端口可能改变
   - 存储旧地址没有实际意义
3. **简化设计**: 避免数据库和网络操作的复杂性

**潜在影响**:
- 应用重启后所有设备地址信息丢失（设备重新上报后会快速恢复）
- 多实例部署时无法共享缓存（每个实例独立维护）

---

## 🔍 常见业务场景

### 场景 1: 配置参数变更

**业务需求**: OMC 需要立即修改设备配置参数

**操作流程**:
1. OMC 推送设备编码到 Redis 队列
2. stunUDPServer 消费并发送 Connection Request
3. 设备收到后立即发起 Inform
4. OMC 在 Inform 会话中下发 SetParameterValues
5. 设备执行配置变更并上报结果

**响应时间**: < 20 秒

### 场景 2: 软件升级通知

**业务需求**: 基站固件版本升级

**操作流程**:
1. OMC 推送设备编码到 Redis 队列
2. 设备发起 Inform 会话
3. OMC 下发 Download 指令（固件下载 URL）
4. 设备下载并安装固件
5. OMC 发送重启命令（RabbitMQ）
6. 设备执行重启
7. 设备上线后上报新版本

### 场景 3: 设备异常告警处理

**业务需求**: 设备运行异常，需要远程重启

**操作流程**:
1. 监控系统检测到设备异常
2. 运维人员确认需要重启
3. OMC 推送重启指令到 RabbitMQ
4. RebootMsgListener 消费消息
5. RebootService 构造加密重启命令
6. UDP 发送到设备
7. 设备验证签名并执行重启

### 场景 4: 批量设备维护

**业务需求**: 对数百台设备进行批量维护

**操作流程**:
1. OMC 批量推送设备编码到 Redis 队列
2. 多个消费线程并发处理
3. 逐个发送 Connection Request
4. 设备逐个发起 Inform
5. OMC 在会话中下发维护指令
6. 监控执行进度和结果

**性能**: 秒级处理数千设备

---

## 📝 总结

### 系统特点

1. **基于 Netty 的高性能 UDP 服务器**
   - NIO 非阻塞模型
   - 池化缓冲区分配器
   - 多线程并发处理

2. **支持标准和非标准两种 STUN 协议**
   - 标准 STUN（TR069 CPE 设备）
   - 非标准 STUN（基站设备）

3. **完整的设备地址发现机制**
   - NAT 穿透
   - 服务器端获取设备公网地址
   - 本地缓存维护

4. **Connection Request 主动通知**
   - 触发设备立即发起 Inform
   - 突破 TR069 协议限制
   - 秒级响应时间

5. **丰富的设备管理功能**
   - 远程重启（RabbitMQ + UDP）
   - 缓存监控
   - 健康检查

### 核心价值

stunUDPServer 是 OMC 系统中的关键组件，实现了：
- ✅ 设备实时通知（从分钟级 60s-300s 降低到秒级）
- ✅ NAT 穿透和地址发现
- ✅ 高性能 UDP 通信
- ✅ 本地缓存快速查询
- ✅ 完整的设备管理能力
- ✅ 异步解耦的高性能设计

### 设计特点

该模块设计清晰、扩展性强，是微服务架构中异步、解耦通信的良好示范。通过 Redis 队列、RabbitMQ、本地缓存等技术组合，实现了高性能、高可用的设备管理平台。

---

**文档版本**: v3.1
**最后更新**: 2026年3月20日
**作者**: 尚颖彬
**文档类型**: 业务规格说明书（用于AI代码生成）

## 📝 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| v3.1 | 2026-03-20 | 新增第6章：CPE STUN 信息完整流转流程 |
| v3.0 | 2026-03-19 | 完整业务规格说明书 |
