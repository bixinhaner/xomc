# rpctool — TR069 RPC 任务工具

> 独立 CLI 工具，通过 TaskService 创建 TR-069 RPC 任务（双写 Redis 队列 + PostgreSQL），保证完整的任务生命周期追踪。

---

## 原理

```
┌──────────┐  CreateTask   ┌──────────────────┐   ┌─────────────────────────────┐
│ rpctool  │ ────────────→ │   TaskService    │──→│ Redis acs:taskq:{deviceSN}  │
└──────────┘               │  (双写保证)      │   └──────────────┬──────────────┘
                           │                  │                  │ Pop
                           │                  │──→┌──────────────┴──────────────┐
                           └──────────────────┘   │  PostgreSQL device_tasks    │
                                                  └─────────────────────────────┘
                                                                 │
                                    ACS 引擎从 Redis 取出任务 ──→│
                                                                 ▼
                                                           ┌────────────┐
                                                           │ CPE 基站    │
                                                           └────────────┘
```

1. rpctool 调用 `TaskService.CreateTask()` 创建任务
2. TaskService 同时写入 Redis Sorted Set（`acs:taskq:{设备SN}`）和 PostgreSQL（`device_tasks` 表）
3. 任务在 DB 中记录完整生命周期：`pending → sent → completed/failed/expired/cancelled`
4. CPE 设备下次连接 ACS 时，ACS 从 Redis 队列中 Pop 任务并发送 SOAP/XML 给 CPE
5. 命令按优先级排序（数值越小越优先），同优先级按 FIFO 顺序

---

## 构建

```bash
cd omcgo

# 方式 1: 通过 Makefile
make build-rpctool

# 方式 2: 直接编译
go build -o bin/rpctool ./scripts/rpctool/
```

---

## 配置

rpctool 加载 ACS 配置文件获取 Redis 和 PostgreSQL 连接信息，默认路径为 `cmd/acs/etc/config.dev.yaml`。

```bash
# 使用默认配置（开发环境）
rpctool upload log --sn DEVICE001 --url http://acs:7547/upload

# 指定其他环境配置
rpctool upload log --sn DEVICE001 --url http://acs:7547/upload \
  --config cmd/acs/etc/config.prod.yaml

# Dry-run 模式不需要 Redis/DB 连接
rpctool upload log --sn DEVICE001 --url http://acs:7547/upload --dry-run
```

支持通过环境变量覆盖配置（`OMCGO_` 前缀），例如：
```bash
export OMCGO_DB_DSN="postgres://user:pass@host:5432/omcgo"
export OMCGO_REDIS_ADDR="redis-prod:6379"
```

---

## 全局参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--config` | `cmd/acs/etc/config.dev.yaml` | ACS 配置文件路径 |
| `--sn` | (必填) | 设备序列号 |
| `--priority` | `10` | 任务优先级（数值越小优先级越高） |
| `--ttl` | `0` | 任务过期时间（秒），0 表示不过期 |
| `--dry-run` | `false` | 仅预览任务 JSON，不写入数据库 |
| `--source` | `api` | 任务来源标记 (api/scheduler/system) |
| `--creator` | `rpctool` | 创建者 ID |
| `--desc` | (自动生成) | 任务描述 |

---

## 子命令

### upload — 文件上传

让基站将指定类型的文件上传到给定 URL。用于采集基站日志、配置备份、性能数据等。

```bash
# 采集运行日志
rpctool upload log --sn DEVICE001 --url http://acs:7547/upload

# 采集配置文件
rpctool upload config --sn DEVICE001 --url http://acs:7547/upload

# 采集 PM 性能文件
rpctool upload pm --sn DEVICE001 --url http://acs:7547/upload

# 采集故障日志
rpctool upload fault-log --sn DEVICE001 --url http://acs:7547/upload

# 采集抓包文件
rpctool upload pcap --sn DEVICE001 --url http://acs:7547/upload

# 使用数字代码指定文件类型
rpctool upload --sn DEVICE001 -t 4 --url http://acs:7547/upload

# 延迟 60 秒后执行
rpctool upload pm --sn DEVICE001 --url http://acs:7547/upload --delay 60
```

**Upload 专属参数:**

| 参数 | 说明 |
|------|------|
| `--file-type` / `-t` | 文件类型（别名或数字代码） |
| `--url` / `-u` | 上传目标 URL（必填） |
| `--username` | HTTP 认证用户名 |
| `--password` | HTTP 认证密码 |
| `--delay` | 延迟执行秒数 |

**Upload 文件类型对照表 (TR-069 Upload: CPE → ACS):**

| 别名 | 代码 | TR-069 FileType | 说明 |
|------|------|----------------|------|
| `config` | 1 | 1 Vendor Configuration File | 配置文件 |
| `log` / `running-log` | 2 | 2 Vendor Log File | 运行日志 |
| `security-log` | — | 2 Vendor Security Log | 安全日志（扩展） |
| `fault-log` | — | 2 Vendor Fault Log | 故障日志（扩展） |
| `pm` | 4 | 4 Vendor PM File | 性能管理文件 |
| `mr` | 5 | 5 Vendor MR File | 测量报告文件 |
| `pcap` | 9 | 9 Vendor PCAP | 抓包文件 |
| `datamodel` | 11 | 11 OUI Parameter Model | 数据模型文件 |

> **TR-069 标准说明**: Upload 操作中，配置文件的 FileType 为 `"1 Vendor Configuration File"`（不是 "11 Configuration File"），日志文件为 `"2 Vendor Log File"`。编号 4 及以上为运营商私有扩展。

---

### download — 文件下载

让基站从指定 URL 下载文件（固件升级、配置下发等）。

```bash
# 固件升级
rpctool download firmware --sn DEVICE001 \
  --url http://minio:9000/firmware/v2.0.bin --file-size 10485760

# 下载配置文件
rpctool download config --sn DEVICE001 \
  --url http://minio:9000/configs/device.xml

# 下载 Web 内容
rpctool download web --sn DEVICE001 \
  --url http://minio:9000/web/portal.zip
```

**Download 专属参数:**

| 参数 | 说明 |
|------|------|
| `--file-type` / `-t` | 文件类型 |
| `--url` / `-u` | 下载地址（必填） |
| `--username` | HTTP 认证用户名 |
| `--password` | HTTP 认证密码 |
| `--file-size` | 文件大小（字节） |
| `--target` | 目标文件名 |
| `--delay` | 延迟执行秒数 |

**Download 文件类型对照表 (TR-069 Download: ACS → CPE):**

| 别名 | 代码 | TR-069 FileType | 说明 |
|------|------|----------------|------|
| `firmware` | 1 | 1 Firmware Upgrade Image | 固件升级 |
| `web` | 2 | 2 Web Content | Web 内容 |
| `config` | 3 | 3 Vendor Configuration File | 配置文件 |

> **注意**: Download 中的配置文件 FileType 为 `"3 Vendor Configuration File"`（代码 3），与 Upload 中的 `"1 Vendor Configuration File"`（代码 1）不同。

---

### getpv — 读取参数值

```bash
# 读取设备信息子树
rpctool getpv --sn DEVICE001 Device.DeviceInfo.

# 读取特定参数
rpctool getpv --sn DEVICE001 \
  Device.DeviceInfo.Manufacturer \
  Device.DeviceInfo.SoftwareVersion

# 读取管理服务器配置
rpctool getpv --sn DEVICE001 Device.ManagementServer.
```

---

### setpv — 设置参数值

```bash
# 修改周期上报间隔为 600 秒
rpctool setpv --sn DEVICE001 \
  -p "Device.ManagementServer.PeriodicInformInterval=600:xsd:unsignedInt"

# 设置多个参数
rpctool setpv --sn DEVICE001 \
  -p "Device.ManagementServer.PeriodicInformInterval=600:xsd:unsignedInt" \
  -p "Device.ManagementServer.PeriodicInformEnable=true:xsd:boolean"

# 设置字符串参数（默认类型 xsd:string）
rpctool setpv --sn DEVICE001 -p "Device.X_VENDOR.CustomField=hello"
```

**参数格式:** `name=value` 或 `name=value:type`

常用类型: `xsd:string`, `xsd:unsignedInt`, `xsd:boolean`, `xsd:int`, `xsd:dateTime`

---

### getpn — 获取参数树结构

```bash
# 获取 Device. 下一级节点
rpctool getpn --sn DEVICE001 --path Device.

# 获取完整参数树（递归）
rpctool getpn --sn DEVICE001 --path Device. --next-level=false

# 获取特定子树
rpctool getpn --sn DEVICE001 --path Device.ManagementServer.
```

---

### reboot — 重启设备

```bash
rpctool reboot --sn DEVICE001

# 高优先级重启
rpctool reboot --sn DEVICE001 --priority 1
```

---

### reset — 恢复出厂设置

```bash
rpctool reset --sn DEVICE001
```

> **警告:** 此操作会清除设备所有配置！

---

### addobject / deleteobject — 对象实例管理

```bash
# 创建对象实例
rpctool addobject --sn DEVICE001 "Device.Services.FAPService."

# 删除对象实例
rpctool deleteobject --sn DEVICE001 "Device.Services.FAPService.1."
```

---

### queue — 队列与历史管理

```bash
# 查看 Redis 队列中所有待执行任务
rpctool queue list --sn DEVICE001

# 查看队首任务
rpctool queue peek --sn DEVICE001

# 查看队列长度
rpctool queue len --sn DEVICE001

# 清空队列（需 --force 确认）
rpctool queue clear --sn DEVICE001 --force

# 查询数据库中的历史任务
rpctool queue history --sn DEVICE001

# 按状态过滤历史任务
rpctool queue history --sn DEVICE001 --status completed

# 查看最近 7 天失败的任务
rpctool queue history --sn DEVICE001 --status failed --days 7
```

**history 子命令参数:**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--status` | (全部) | 按状态过滤 (pending/sent/completed/failed/expired/cancelled) |
| `--page` | `1` | 页码 |
| `--page-size` | `20` | 每页条数 |
| `--days` | `0` | 最近 N 天 (0=不限) |

---

## 典型使用场景

### 场景 1: 调试日志采集

```bash
# 1. 创建日志上传任务
rpctool upload log --sn BCL00A1B2C3 \
  --url http://192.168.1.100:7547/upload

# 2. 确认任务已入队
rpctool queue peek --sn BCL00A1B2C3

# 3. 等待设备 Inform 后自动执行

# 4. 查看任务执行结果
rpctool queue history --sn BCL00A1B2C3 --status completed
```

### 场景 2: 批量参数配置

```bash
for sn in DEVICE001 DEVICE002 DEVICE003; do
  rpctool setpv --sn $sn \
    -p "Device.ManagementServer.PeriodicInformInterval=300:xsd:unsignedInt"
  echo "已创建任务: $sn"
done
```

### 场景 3: 固件升级测试

```bash
# 创建固件下载任务（高优先级）
rpctool download firmware --sn DEVICE001 \
  --url http://minio:9000/firmware/SmallCell-LTE-v3.0.bin \
  --file-size 52428800 \
  --priority 1

# 查看任务状态
rpctool queue history --sn DEVICE001
```

### 场景 4: Dry-Run 预览

不确定命令格式时，使用 `--dry-run` 预览（不需要 Redis/DB 连接）：

```bash
rpctool upload log --sn DEVICE001 --url http://acs:7547/upload --dry-run
```

输出示例：
```
=== Dry Run (未写入数据库) ===
设备 SN:  DEVICE001
方法:     Upload
优先级:   10
来源:     api
创建者:   rpctool
任务 JSON:
{
  "id": "a1b2c3d4-...",
  "device_sn": "DEVICE001",
  "method": "Upload",
  "params": {"file_type": "2 Vendor Log File", "url": "http://acs:7547/upload", ...},
  "status": "pending",
  ...
}
```

---

## ACS 端处理流程

```
CPE                         ACS                      Redis + PostgreSQL
 │── Inform ──────────────→│                          │
 │←── InformResponse ──────│                          │
 │── Empty POST ──────────→│── Pop ─────────────────→ │ (Redis)
 │                          │←── Task ──────────────── │
 │                          │── MarkSent ───────────→ │ (Redis + PG)
 │←── Upload SOAP ─────────│                          │
 │── UploadResponse ──────→│── MarkCompleted ──────→ │ (Redis + PG)
 │←── 204 No Content ──────│                          │
```

---

## 注意事项

- rpctool 需要连接 Redis 和 PostgreSQL（通过 ACS 配置文件），`--dry-run` 模式除外
- 任务创建后需要等待设备下次连接 ACS 才会执行
- 如果设备长时间不连接，可以通过 ACS 的 Connection Request 功能主动唤醒
- `--priority 1` 为最高优先级，适用于紧急操作（如重启）
- 使用 `--ttl` 设置过期时间，避免过期任务在设备重连后被执行
- 任务完整生命周期记录在 `device_tasks` 表中，可通过 `queue history` 查询
