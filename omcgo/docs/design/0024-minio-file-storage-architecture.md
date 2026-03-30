# 0024 MinIO 文件存储架构 — 技术方案与目录规划

> 基于 TR069 协议文件传输规范、项目现状和运营商网管业务需求，设计统一的 MinIO 对象存储架构。

---

## 1. 背景与目标

### 1.1 当前状况

OMC 系统通过 TR069 Download/Upload RPC 与 CPE 设备交换文件。当前已实现 6 个 MinIO Bucket 的基础框架：

| Bucket | 当前用途 | 实现状态 |
|--------|---------|---------|
| `firmware` | 固件版本二进制 | ✅ 已实现（software 模块） |
| `pm-files` | PM 性能数据 XML | ✅ 已实现（pm 模块 + transfer bridge） |
| `mr-files` | MR 测量报告 XML | ✅ 已实现（mr 模块） |
| `config-backup` | 配置备份/托管文件 | ✅ 已实现（backup + filemanager 模块） |
| `logs` | 设备日志 + 数据模型 XML | ⚠️ 部分（仅数据模型上传，无日志收集） |
| `reports` | 报表文件 | ✅ 已实现（report 模块） |

**存在的问题**：

1. **Bucket 粒度不足** — `logs` bucket 混存数据模型 XML、运行日志、安全日志、异常重启日志，无法独立管理生命周期
2. **缺少专用存储** — 补丁包（PATCH）、��包文件（PCAP）、Excel 导入模板、导出文件等无明确归属
3. **路径规范不统一** — 各模块自行拼接路径，格式不一致（有的 `YYYY/MM/DD`，有的 `YYYY-MM-DD`）
4. **生命周期策略缺失** — 没有统一的文件过期清理机制
5. **容量规划缺失** — 未估算 10 万基站规模下的存储需求

### 1.2 设计目标

1. **统一目录规范** — 所有文件存储路径遵循一致的层级命名规则
2. **合理 Bucket 划分** — 按文件类型的生命周期、访问模式和保留策略拆分 Bucket
3. **TR069 FileType 完整映射** — 覆盖 TR069 规范定义的全部文件类型
4. **容量可控** — 每种文件类型有明确的保留策略和清理机制
5. **向后兼容** — 不破坏现有 6 个 Bucket 的使用，通过新增 Bucket 扩展

---

## 2. TR069 文件类型全景

### 2.1 TR069 FileType 定义（Amendment 6）

TR069 规范定义了以下标准 FileType，在 Download/Upload RPC 中使用：

| FileType 码 | 全称 | 方向 | 说明 |
|-------------|------|------|------|
| `1 Firmware Upgrade Image` | 固件主镜像 | Download (ACS→CPE) | 设备固件升级，完整镜像替换 |
| `2 Web Content` | Web 内容 | Download (ACS→CPE) | 设备 Web 管理页面更新 |
| `3 Vendor Configuration File` | 厂商配置文件 | 双向 | 下发配置 / 备份配置 |
| `4 Vendor Log File` | 厂商日志文件 | Upload (CPE→ACS) | 运行日志、安全日志、故障日志 |
| `5 Tone File` | 语音提示文件 | Download (ACS→CPE) | 不适用于基站 |
| `6 Ringer File` | 铃声文件 | Download (ACS→CPE) | 不适用于基站 |
| `X <OUI> <Vendor-specific>` | 厂商扩展 | 双向 | 厂商自定义文件类型 |

### 2.2 项目扩展 FileType（基于运营商规范）

在 TR069 标准基础上，结合运营商规范和业务需求，项目定义以下扩展文件类型：

| 项目 FileType 码 | 用途 | 方向 | 对应 TR069 |
|------------------|------|------|-----------|
| `1` | 固件主镜像 (IMG) | Download | `1 Firmware Upgrade Image` |
| `2` | 补丁包 (PATCH) | Download | `X <OUI> Patch` |
| `3` | 配置文件 (CONFIG) | 双向 | `3 Vendor Configuration File` |
| `4` | PM 性能文件 (PM) | Upload | `4 Vendor Log File` (子类型) |
| `5` | MR 测量报告 (MR) | Upload | `4 Vendor Log File` (子类型) |
| `6` | 运行日志 (RUNNING_LOG) | Upload | `4 Vendor Log File` |
| `7` | 安全日志 (SECURITY_LOG) | Upload | `4 Vendor Log File` |
| `8` | 异常重启日志 (FAULT_LOG) | Upload | `4 Vendor Log File` |
| `9` | 抓包文件 (PCAP) | Upload | `X <OUI> PacketCapture` |
| `10` | Web 内容 (WEB) | Download | `2 Web Content` |
| `11` | 数据模型 XML (DATAMODEL) | Upload | `X <OUI> ParameterModel` |

### 2.3 FileType 常量定义

当前项目缺少统一的 FileType 常量，建议新增：

```go
// pkg/tr069/filetype.go
package tr069

// FileType 定义 TR069 文件传输类型
type FileType string

const (
    FileTypeFirmware    FileType = "1"  // 固件主镜像 (IMG)
    FileTypePatch       FileType = "2"  // 补丁包 (PATCH)
    FileTypeConfig      FileType = "3"  // 配置文件
    FileTypePM          FileType = "4"  // PM 性能数据
    FileTypeMR          FileType = "5"  // MR 测量报告
    FileTypeRunningLog  FileType = "6"  // 运行日志
    FileTypeSecurityLog FileType = "7"  // 安全日志
    FileTypeFaultLog    FileType = "8"  // 异常重启/故障日志
    FileTypePCAP        FileType = "9"  // 抓包文件
    FileTypeWeb         FileType = "10" // Web 内容
    FileTypeDataModel   FileType = "11" // 数据模型 XML
)

// FileTypeLabel 返回文件类型的中文描述
func (ft FileType) Label() string {
    labels := map[FileType]string{
        FileTypeFirmware:    "固件主镜像",
        FileTypePatch:       "补丁包",
        FileTypeConfig:      "配置文件",
        FileTypePM:          "PM性能数据",
        FileTypeMR:          "MR测量报告",
        FileTypeRunningLog:  "运行日志",
        FileTypeSecurityLog: "安全日志",
        FileTypeFaultLog:    "故障日志",
        FileTypePCAP:        "抓包文件",
        FileTypeWeb:         "Web内容",
        FileTypeDataModel:   "数据模型",
    }
    return labels[ft]
}
```

---

## 3. Bucket 架构设计

### 3.1 Bucket 划分原则

| 原则 | 说明 |
|------|------|
| **生命周期分离** | 保留期不同的文件放不同 Bucket（固件永久 vs 日志 30 天） |
| **访问模式匹配** | 高频读写（PM/MR）与低频读（固件/备份）分开 |
| **安全边界** | 设备上传文件（不可信）与管理端文件（可信）分开 |
| **运维独立性** | 每个 Bucket 可独立配置配额、版本控制、复制策略 |

### 3.2 Bucket 总览（10 个 Bucket）

```
┌─────────────────────────────────────────────────────────────────┐
│                    MinIO Object Storage                         │
├─────────────────┬───────────────┬───────────────────────────────┤
│  软件包分发区     │  数据采集区    │          运维管理区             │
│  (Download)     │  (Upload)     │        (双向/管理)             │
├─────────────────┼───────────────┼───────────────────────────────┤
│ omc-firmware    │ omc-pm        │ omc-config                    │
│  ├ img/         │  └ {date}/    │  ├ backup/                    │
│  ├ patch/       │               │  ├ baseline/                  │
│  └ web/         │ omc-mr        │  └ template/                  │
│                 │  └ {date}/    │                               │
│                 │               │ omc-logs                      │
│                 │               │  ├ running/                   │
│                 │               │  ├ security/                  │
│                 │               │  ├ fault/                     │
│                 │               │  └ pcap/                      │
│                 │               │                               │
│                 │               │ omc-reports                   │
│                 │               │  └ {definition_id}/           │
│                 │               │                               │
│                 │               │ omc-exchange                  │
│                 │               │  ├ import/                    │
│                 │               │  ├ export/                    │
│                 │               │  └ datamodel/                 │
└─────────────────┴───────────────┴───────────────────────────────┘
```

### 3.3 各 Bucket 详细说明

#### 3.3.1 `omc-firmware` — 软件包仓库

**映射关系**：合并现有 `firmware` bucket，扩展支持补丁包和 Web 内容。

| 属性 | 值 |
|------|-----|
| 来源方向 | 管理员上传 → ACS Download RPC → CPE |
| 保留策略 | **永久**（版本管理，手动清理弃用版本） |
| 预估容量 | ~50 GB（500 个固件版本 × 100 MB） |
| 版本控制 | 启用（防误删，支持回滚） |
| 访问频率 | 低（上传偶发，下载按升级任务触发） |

**目录结构**：

```
omc-firmware/
├── img/                                    # 固件主镜像 (FileType 1)
│   └── {carrier}/
│       └── {product_class}/
│           └── {version}/
│               ├── {filename}.bin          # 固件二进制
│               └── {filename}.bin.md5      # 校验文件（可选）
│
├── patch/                                  # 补丁包 (FileType 2)
│   └── {carrier}/
│       └── {product_class}/
│           └── {base_version}/             # 基线版本
│               └── {patch_version}/
│                   └── {filename}.patch
│
└── web/                                    # Web 内容 (FileType 10)
    └── {carrier}/
        └── {product_class}/
            └── {version}/
                └── {filename}.tar.gz
```

**路径示例**：

```
omc-firmware/img/cmcc/Nova436Q/V100R001C00B060/Nova436Q_V100R001C00B060.bin
omc-firmware/img/cmcc/Nova436Q/V100R001C00B060/Nova436Q_V100R001C00B060.bin.md5
omc-firmware/patch/cmcc/Nova436Q/V100R001C00B055/V100R001C00B060/hotfix_001.patch
omc-firmware/web/ctcc/Nova227/V2.0.0/web_ui_v2.tar.gz
```

**与现有代码的兼容**：

现有 `software/service.go` 使用路径 `firmware/{carrier}/{product_class}/{version}/{filename}`。迁移方案：
- 新路径：`img/{carrier}/{product_class}/{version}/{filename}`
- 兼容期：同时支持旧路径读取，新上传使用新路径
- 配置项 `buckets.firmware` 改名为 `buckets.omc_firmware`（YAML 配置）

---

#### 3.3.2 `omc-pm` — PM 性能数据

**映射关系**：替代现有 `pm-files` bucket，路径格式统一。

| 属性 | 值 |
|------|-----|
| 来源方向 | CPE Upload / AutonomousTransferComplete → MinIO |
| 保留策略 | **90 天**（与 TimescaleDB mr_records 保留策略对齐） |
| 预估容量 | ~150 GB/月（100K 设备 × 15 分钟粒度 × ~1 KB/文件） |
| 版本控制 | 禁用（一次写入不修改） |
| 访问频率 | 高写低读（写入频繁，解析后很少重读） |

**目录结构**：

```
omc-pm/
└── {carrier}/
    └── {YYYY}/{MM}/{DD}/
        └── {device_sn}/
            └── {filename}.xml
```

**路径示例**：

```
omc-pm/cmcc/2026/03/30/BCI-SN-00001/A20260330.1500+0800-1515+0800_BCI-SN-00001.xml
omc-pm/ctcc/2026/03/30/BCI-SN-00002/pm_counters_20260330150000.xml
```

**保留策略实现**：

```go
// MinIO Lifecycle Rule (通过 mc 命令或 API 配置)
// mc ilm rule add omc-pm --expiry-days 90
```

---

#### 3.3.3 `omc-mr` — MR 测量报告

**映射关系**：替代现有 `mr-files` bucket。

| 属性 | 值 |
|------|-----|
| 来源方向 | CPE Upload → MinIO |
| 保留策略 | **90 天** |
| 预估容量 | ~200 GB/月（MR 文件通常比 PM 大） |
| 版本控制 | 禁用 |
| 访问频率 | 高写低读 |

**目录结构**：

```
omc-mr/
└── {carrier}/
    └── {YYYY}/{MM}/{DD}/
        └── {device_sn}/
            └── {mr_type}/                  # mro / mrs / mre
                └── {filename}.xml
```

**路径示例**：

```
omc-mr/cmcc/2026/03/30/BCI-SN-00001/mro/A20260330.1500_MRO_BCI-SN-00001.xml
omc-mr/cmcc/2026/03/30/BCI-SN-00001/mrs/A20260330.1500_MRS_BCI-SN-00001.xml
```

---

#### 3.3.4 `omc-logs` — 设备日志

**映射关系**：替代现有 `logs` bucket，拆分为明确的子目录。

| 属性 | 值 |
|------|-----|
| 来源方向 | CPE Upload RPC → MinIO（日志收集触发） |
| 保留策略 | **30 天**（运行日志/安全日志），**180 天**（故障日志），**7 天**（抓包） |
| 预估容量 | ~80 GB/月（日志收集非持续性，按需触发） |
| 版本控制 | 禁用 |
| 访问频率 | 低写低读（按需收集和查看） |

**目录结构**：

```
omc-logs/
├── running/                                # 运行日志 (FileType 6)
│   └── {carrier}/
│       └── {YYYY}/{MM}/{DD}/
│           └── {device_sn}/
│               └── {task_id}_{timestamp}.log
│
├── security/                               # 安全日志 (FileType 7)
│   └── {carrier}/
│       └── {YYYY}/{MM}/{DD}/
│           └── {device_sn}/
│               └── {task_id}_{timestamp}.log
│
├── fault/                                  # 异常重启/故障日志 (FileType 8)
│   └── {carrier}/
│       └── {YYYY}/{MM}/{DD}/
│           └── {device_sn}/
│               └── {reboot_log_id}_{timestamp}.log
│
└── pcap/                                   # 抓包文件 (FileType 9)
    └── {carrier}/
        └── {YYYY}/{MM}/{DD}/
            └── {device_sn}/
                └── {task_id}_{duration}s.pcap
```

**路径示例**：

```
omc-logs/running/cmcc/2026/03/30/BCI-SN-00001/a1b2c3d4_20260330150000.log
omc-logs/fault/cmcc/2026/03/30/BCI-SN-00001/e5f6g7h8_20260330143022.log
omc-logs/pcap/ctcc/2026/03/30/BCI-SN-00002/i9j0k1l2_60s.pcap
```

**差异化保留策略**：

由于 MinIO Lifecycle Rule 以 prefix 为粒度，可对不同子目录设置不同过期时间：

```bash
mc ilm rule add omc-logs --prefix "running/" --expiry-days 30
mc ilm rule add omc-logs --prefix "security/" --expiry-days 30
mc ilm rule add omc-logs --prefix "fault/" --expiry-days 180
mc ilm rule add omc-logs --prefix "pcap/" --expiry-days 7
```

---

#### 3.3.5 `omc-config` — 配置管理

**映射关系**：替代现有 `config-backup` bucket，扩展支持配置基线和模板。

| 属性 | 值 |
|------|-----|
| 来源方向 | 双向（ACS→CPE 下发 / CPE→ACS 备份） |
| 保留策略 | **365 天**（备份），**永久**（基线/模板） |
| 预估容量 | ~20 GB/月（100K 设备 × 每月 1 次备份 × ~200 KB） |
| 版本控制 | 启用（配置文件需要版本追溯） |
| 访问频率 | 低 |

**目录结构**：

```
omc-config/
├── backup/                                 # 设备配置备份 (FileType 3, Upload)
│   └── {carrier}/
│       └── {device_sn}/
│           └── {YYYY}/{MM}/{DD}/
│               └── {backup_task_id}.xml
│
├── baseline/                               # 配置基线（下发模板实例）
│   └── {carrier}/
│       └── {product_class}/
│           └── {baseline_id}/
│               └── config.xml
│
└── template/                               # 配置模板
    └── {carrier}/
        └── {product_class}/
            └── {template_name}/
                └── {version}.xml
```

**路径示例**：

```
omc-config/backup/cmcc/BCI-SN-00001/2026/03/30/task_abc123.xml
omc-config/baseline/cmcc/Nova436Q/baseline_001/config.xml
omc-config/template/cmcc/Nova436Q/default_lte/v1.0.xml
```

---

#### 3.3.6 `omc-reports` — 报表文件

**映射关系**：替代现有 `reports` bucket。

| 属性 | 值 |
|------|-----|
| 来源方向 | 系统生成（PM/KPI/告警报表） |
| 保留策略 | **180 天** |
| 预估容量 | ~5 GB/月 |
| 版本控制 | 禁用 |
| 访问频率 | 低读（生成后用户下载） |

**目录结构**：

```
omc-reports/
└── {report_type}/                          # kpi / alarm / pm / device
    └── {YYYY}/{MM}/
        └── {definition_id}/
            └── {record_id}.{format}        # xlsx / pdf / csv
```

**路径示例**：

```
omc-reports/kpi/2026/03/def_001/rec_abc123.xlsx
omc-reports/alarm/2026/03/def_002/rec_def456.pdf
```

---

#### 3.3.7 `omc-exchange` — 数据交换区

**新增 Bucket**，用于系统与用户/设备之间的临时数据交换。

| 属性 | 值 |
|------|-----|
| 来源方向 | 混合（导入/导出/数据模型） |
| 保留策略 | **7 天**（导入/导出临时文件），**永久**（数据模型） |
| 预估容量 | ~5 GB |
| 版本控制 | 禁用 |
| 访问频率 | 低（按需） |

**目录结构**：

```
omc-exchange/
├── import/                                 # 批量导入文件（Excel 模板等）
│   └── {YYYY}/{MM}/{DD}/
│       └── {user_id}/
│           └── {upload_id}_{filename}
│
├── export/                                 # 导出文件（设备列表 CSV/Excel 等）
│   └── {YYYY}/{MM}/{DD}/
│       └── {user_id}/
│           └── {export_task_id}.{format}
│
├── datamodel/                              # 数据模型 XML (FileType 11)
│   └── {carrier}/
│       └── {oui}/
│           └── {product_class}/
│               └── {device_sn}_{timestamp}.xml
│
└── template/                               # 导入模板（Excel 模板下载）
    └── device_import_template.xlsx
    └── device_import_template_v2.xlsx
```

**路径示例**：

```
omc-exchange/import/2026/03/30/user_001/upload_abc_设备导入.xlsx
omc-exchange/export/2026/03/30/user_001/export_def.csv
omc-exchange/datamodel/cmcc/BAICELLS/Nova436Q/BCI-SN-00001_20260330.xml
omc-exchange/template/device_import_template.xlsx
```

---

### 3.4 Bucket 映射汇总

| Bucket 名 | 旧 Bucket | FileType | 保留策略 | 版本控制 |
|-----------|----------|----------|---------|---------|
| `omc-firmware` | `firmware` | 1, 2, 10 | 永久 | ✅ |
| `omc-pm` | `pm-files` | 4 | 90 天 | ❌ |
| `omc-mr` | `mr-files` | 5 | 90 天 | ❌ |
| `omc-logs` | `logs`（部分） | 6, 7, 8, 9 | 7-180 天 | ❌ |
| `omc-config` | `config-backup` | 3 | 365 天/永久 | ✅ |
| `omc-reports` | `reports` | — | 180 天 | ❌ |
| `omc-exchange` | 新增 | 11, 用户文件 | 7 天/永久 | ❌ |

---

## 4. 统一路径规范

### 4.1 路径格式规则

所有 MinIO 对象路径遵循以下规则：

```
{bucket}/{category}/{carrier}/{date_path}/{device_sn}/{identifier}_{timestamp}.{ext}
```

| 层级 | 格式 | 说明 |
|------|------|------|
| bucket | `omc-{domain}` | 统一前缀 `omc-` |
| category | 小写英文 | 文件子类型（img/patch/running/fault 等） |
| carrier | `cmcc`/`ctcc`/`cucc` | 运营商代码 |
| date_path | `{YYYY}/{MM}/{DD}` | 斜杠分隔的日期路径（便于按前缀列举） |
| device_sn | 原值 | 设备序列号 |
| identifier | UUID 短码或任务 ID | 防重名 |
| timestamp | `YYYYMMDDHHmmss` | 紧凑时间戳（无分隔符） |
| ext | 文件扩展名 | `.xml` `.bin` `.log` `.pcap` `.xlsx` `.csv` |

### 4.2 路径构建工具

建议新增统一的路径构建函数：

```go
// internal/core/storage/path.go
package storage

import (
    "fmt"
    "time"
)

// ObjectPath 构建 MinIO 对象路径
// category: "img", "pm", "running", "fault" 等
// carrier: "cmcc", "ctcc", "cucc"
// deviceSN: 设备序列号
// filename: 文件名
func ObjectPath(category, carrier, deviceSN, filename string) string {
    now := time.Now()
    return fmt.Sprintf("%s/%s/%s/%s/%s",
        category,
        carrier,
        now.Format("2006/01/02"),
        deviceSN,
        filename,
    )
}

// FirmwarePath 构建固件存储路径
func FirmwarePath(category, carrier, productClass, version, filename string) string {
    return fmt.Sprintf("%s/%s/%s/%s/%s",
        category, // "img" or "patch" or "web"
        carrier,
        productClass,
        version,
        filename,
    )
}

// ExchangePath 构建数据交换路径（无 device 维度）
func ExchangePath(category, userID, filename string) string {
    now := time.Now()
    return fmt.Sprintf("%s/%s/%s/%s",
        category, // "import" or "export"
        now.Format("2006/01/02"),
        userID,
        filename,
    )
}
```

---

## 5. 配置变更

### 5.1 BucketConfig 扩展

```go
// internal/core/appconfig/config.go
type BucketConfig struct {
    // 现有（保留兼容别名，过渡期后弃用）
    PMFiles      string `mapstructure:"pm_files"`
    MRFiles      string `mapstructure:"mr_files"`
    Firmware     string `mapstructure:"firmware"`
    ConfigBackup string `mapstructure:"config_backup"`
    Logs         string `mapstructure:"logs"`
    Reports      string `mapstructure:"reports"`

    // 新增
    Exchange     string `mapstructure:"exchange"`
}
```

### 5.2 YAML 配置

```yaml
minio:
  endpoint: "minio:9000"
  access_key: "${MINIO_ACCESS_KEY}"
  secret_key: "${MINIO_SECRET_KEY}"
  use_ssl: false
  buckets:
    firmware: "omc-firmware"
    pm_files: "omc-pm"
    mr_files: "omc-mr"
    logs: "omc-logs"
    config_backup: "omc-config"
    reports: "omc-reports"
    exchange: "omc-exchange"
```

### 5.3 Bucket 初始化

`EnsureBuckets()` 新增 `omc-exchange` bucket 创建。

---

## 6. FileType → Bucket 路由

### 6.1 Upload 路由（CPE→ACS）

当 ACS Upload Handler 或 TransferBridge 接收到文件时，按 FileType 路由：

```go
// internal/core/storage/router.go
package storage

// BucketAndCategory 根据 FileType 返回目标 bucket 和子目录
func BucketAndCategory(fileType string, buckets BucketConfig) (bucket, category string) {
    switch fileType {
    case "1":
        return buckets.Firmware, "img"
    case "2":
        return buckets.Firmware, "patch"
    case "3":
        return buckets.ConfigBackup, "backup"
    case "4":
        return buckets.PMFiles, ""           // PM 文件直接以 carrier/date 开头
    case "5":
        return buckets.MRFiles, ""           // MR 同理
    case "6":
        return buckets.Logs, "running"
    case "7":
        return buckets.Logs, "security"
    case "8":
        return buckets.Logs, "fault"
    case "9":
        return buckets.Logs, "pcap"
    case "10":
        return buckets.Firmware, "web"
    case "11":
        return buckets.Exchange, "datamodel"
    default:
        return buckets.Logs, "unknown"
    }
}
```

### 6.2 Download URL 构建（ACS→CPE）

Download RPC 需要为 CPE 提供可访问的文件 URL：

```
┌──────────┐    Download RPC (URL)      ┌─────────┐
│   ACS    │ ──────────────────────────► │   CPE   │
│ (Server) │    FileType + URL           │ (设备)  │
└────┬─────┘                             └────┬────┘
     │                                        │
     │  1. 查询 firmware_versions 表           │  3. HTTP GET URL
     │  2. 生成 presigned URL 或代理 URL       │  4. 下载文件
     ▼                                        ▼
┌──────────┐                             ┌──────────┐
│ MinIO    │ ◄───────────────────────────│ 文件下载  │
│ (存储)   │         GET Object          │          │
└──────────┘                             └──────────┘
```

**URL 策略**：

| 方案 | 说明 | 适用场景 |
|------|------|---------|
| **代理模式**（当前） | ACS 从 MinIO 读取后转发给 CPE | CPE 无法直连 MinIO |
| **Presigned URL** | 生成带签名的 MinIO 临时直连 URL | CPE 可直连 MinIO，减轻 ACS 负载 |
| **CDN 模式** | 固件存 CDN，URL 指向 CDN 域名 | 大规模升级场景（万台并发） |

当前使用代理模式（`minio://{bucket}/{path}` 内部格式）。建议后续大规模升级时引入 Presigned URL。

---

## 7. 容量规划

### 7.1 10 万基站规模月存储估算

| 文件类型 | 单文件大小 | 频率 | 月增量 | 保留期 | 峰值存储 |
|---------|----------|------|--------|--------|---------|
| PM XML | ~1-5 KB | 15 分钟/台 | ~150 GB | 90 天 | ~450 GB |
| MR XML | ~5-20 KB | 15 分钟/台 | ~200 GB | 90 天 | ~600 GB |
| 固件 IMG | ~50-200 MB | 新版本发布 | ~2 GB | 永久 | ~50 GB |
| 补丁 PATCH | ~5-50 MB | 偶发 | ~500 MB | 永久 | ~10 GB |
| 配置备份 | ~100-500 KB | 月度/按需 | ~20 GB | 365 天 | ~240 GB |
| 运行日志 | ~1-10 MB | 按需收集 | ~10 GB | 30 天 | ~10 GB |
| 故障日志 | ~500 KB-5 MB | 异常重启 | ~5 GB | 180 天 | ~30 GB |
| 抓包 PCAP | ~10-100 MB | 按需 | ~2 GB | 7 天 | ~2 GB |
| 报表 | ~100 KB-5 MB | 日/周/月 | ~5 GB | 180 天 | ~30 GB |
| 数据模型 | ~50-200 KB | 设备首次上线 | ~2 GB | 永久 | ~20 GB |
| 导入/导出 | ~1-10 MB | 按需 | ~1 GB | 7 天 | ~1 GB |
| **合计** | | | **~397 GB/月** | | **~1.4 TB** |

### 7.2 存储配置建议

```yaml
# 生产环境 MinIO 配置建议
# 10 万基站规模
minio:
  # 最小 4 节点，每节点 4 块盘（纠删码 EC:4）
  # 总裸容量：16 × 1 TB = 16 TB
  # 可用容量（EC:4 冗余）：~10 TB
  # 留有 7× 余量应对突发和增长
  erasure_code: "EC:4"
  total_nodes: 4
  drives_per_node: 4
  drive_size: "1TB"
```

### 7.3 清理策略

| 策略 | 实现方式 | 触发条件 |
|------|---------|---------|
| **TTL 自动过期** | MinIO ILM Rule (expiry-days) | 文件年龄超过保留期 |
| **磁盘水位清理** | Worker 定时检查 | 磁盘使用率 > 80% |
| **单设备上限** | 业务层删除 | 单设备日志超 100 条时删最早记录 |
| **全局总量清理** | Worker 定时检查 | 某类文件总量超阈值 |

**ILM 初始化脚本**：

```bash
#!/bin/bash
# scripts/minio_lifecycle.sh — MinIO 生命周期策略配置

MC="mc"
ALIAS="omc"

# PM: 90 天
$MC ilm rule add ${ALIAS}/omc-pm --expiry-days 90

# MR: 90 天
$MC ilm rule add ${ALIAS}/omc-mr --expiry-days 90

# 日志分级
$MC ilm rule add ${ALIAS}/omc-logs --prefix "running/" --expiry-days 30
$MC ilm rule add ${ALIAS}/omc-logs --prefix "security/" --expiry-days 30
$MC ilm rule add ${ALIAS}/omc-logs --prefix "fault/" --expiry-days 180
$MC ilm rule add ${ALIAS}/omc-logs --prefix "pcap/" --expiry-days 7

# 报表: 180 天
$MC ilm rule add ${ALIAS}/omc-reports --expiry-days 180

# 交换区: 7 天（排除模板和数据模型）
$MC ilm rule add ${ALIAS}/omc-exchange --prefix "import/" --expiry-days 7
$MC ilm rule add ${ALIAS}/omc-exchange --prefix "export/" --expiry-days 7

# 配置备份: 365 天
$MC ilm rule add ${ALIAS}/omc-config --prefix "backup/" --expiry-days 365

echo "MinIO lifecycle rules configured."
```

---

## 8. 文件元数据管理

### 8.1 元数据存储原则

MinIO 是纯对象存储，所有业务查询通过 PostgreSQL 元数据表驱动：

```
┌─────────────┐   minio_path   ┌──────────┐
│ PostgreSQL  │ ──────────────► │  MinIO   │
│ (元数据)    │   按需读取文件   │  (文件)  │
│             │ ◄────────────── │          │
└─────────────┘                └──────────┘
```

**规则**：
- ��种文件类型在 PostgreSQL 中有对应的元数据表（已有: `firmware_versions`、`pm_files`、`mr_files`、`managed_files`、`backup_tasks`）
- 列表查询、过滤、统计全部走 PostgreSQL
- MinIO 只在下载文件内容时通过 `minio_path` 读取
- 禁止通过 `mc ls` 或 `ListObjects` 做业务查询

### 8.2 需新建的元数据表

| 表名 | 对应 Bucket/路径 | 参考 Gap 编号 |
|------|-----------------|-------------|
| `log_collect_tasks` | `omc-logs/running/`、`omc-logs/security/` | G15, G16 |
| `device_reboot_logs` | `omc-logs/fault/` | G24 |
| `packet_capture_tasks` | `omc-logs/pcap/` | G11 |
| `device_export_tasks` | `omc-exchange/export/` | G04 |
| `device_import_records` | `omc-exchange/import/` | G21 |

这些表已在 0023 号设计文档中规划，此处仅列出与存储的映射关系。

---

## 9. 安全设计

### 9.1 访问控制

| 维度 | 策略 |
|------|------|
| **Bucket Policy** | 全部 Private，禁止匿名访问 |
| **Service Account** | 每个部署单元独立 AK/SK（app / acs / worker） |
| **权限最小化** | ACS 只写不删，App 读写删，Worker 读删 |
| **网络隔离** | MinIO 仅对内部网络开放，不暴露公网 |

### 9.2 各部署单元的 Bucket 权限

| 部署单元 | omc-firmware | omc-pm | omc-mr | omc-logs | omc-config | omc-reports | omc-exchange |
|---------|-------------|--------|--------|----------|-----------|------------|-------------|
| **app** | R/W/D | R | R | R | R/W/D | R/W/D | R/W/D |
| **acs** | R | W | W | W | W | — | W |
| **worker** | — | R/D | R/D | R/D | R | R/W | R/D |

> R=读, W=写, D=删除

### 9.3 CPE 上传安全

当前实现（参考 0017 号设计文档）：
- CPE 上传通过 `/smallcell/FileUploadService` 代理端点
- HTTP Basic Auth 认证（全局凭据）
- 服务端转存 MinIO，CPE 不直接访问 MinIO
- 文件大小限制（`upload.max_file_size`）
- 路径遍历防护（filename 净化）

---

## 10. 迁移方案

### 10.1 Bucket 重命名迁移

从旧 Bucket 名迁移到新 `omc-` 前缀命名：

| 阶段 | 操作 | 风险 |
|------|------|------|
| **1. 创建新 Bucket** | `EnsureBuckets` 创建 `omc-*` 系列 | 无 |
| **2. 双写** | 新文件同时写入新旧 Bucket | 存储翻倍（临时） |
| **3. 数据同步** | `mc mirror` 从旧 Bucket 复制到新 Bucket | 需要额外带宽 |
| **4. 切读** | 所有读操作指向新 Bucket | 需要更新所有 `minio_path` 或做路径映射 |
| **5. 删旧** | 确认无残留读取后删除旧 Bucket | 不可逆 |

**建议**：如果当前生产数据量不大（< 100 GB），可直接切换配置名（仅改 YAML），让旧数据自然过期。新 Bucket 从空开始。

### 10.2 代码改动清单

| 文件 | 改动 | 优先级 |
|------|------|--------|
| `pkg/tr069/filetype.go` | 新增 FileType 常量定义 | P0 |
| `internal/core/storage/path.go` | 新增统一路径构建函数 | P0 |
| `internal/core/storage/router.go` | 新增 FileType→Bucket 路由 | P0 |
| `internal/core/appconfig/config.go` | BucketConfig 新增 Exchange | P0 |
| `internal/core/components/minio/minio.go` | EnsureBuckets 新增 Exchange | P0 |
| `cmd/app/etc/config.*.yaml` | 更新 Bucket 名称 | P0 |
| `internal/acs/upload/handler.go` | 使用 storage.BucketAndCategory() | P1 |
| `internal/transfer/bridge.go` | 使用 storage.ObjectPath() | P1 |
| `internal/software/service.go` | 使用 storage.FirmwarePath() | P1 |
| `internal/pm/collector/collector.go` | 使用统一路径 | P2 |
| `internal/mr/pg_store.go` | 使用统一路径 | P2 |

---

## 11. 完整文件类型 × 存储位置矩阵

```
                          omc-       omc-   omc-   omc-    omc-     omc-      omc-
文件类型                  firmware    pm     mr     logs    config   reports   exchange
─────────────────────────────────────────────────────────────────────────────────────
固件主镜像 (IMG)          img/
补丁包 (PATCH)            patch/
Web 内容                  web/
PM 性能 XML                         ✓
MR 测量报告                                 ✓
运行日志                                           running/
安全日志                                           security/
故障日志                                           fault/
抓包 PCAP                                          pcap/
配置备份                                                   backup/
配置基线                                                   baseline/
配置模板                                                   template/
KPI/告警报表                                                          ✓
数据模型 XML                                                                   datamodel/
导入文件                                                                       import/
导出文件                                                                       export/
导入模板                                                                       template/
```

---

## 12. 关键设计决策记录

| 决策 | 选择 | 备选方案 | 理由 |
|------|------|---------|------|
| Bucket 数量 | 7 个 | 按 FileType 每种一个（11+个） | 7 个足够区分生命周期和访问模式，过多增加运维负担 |
| 路径日期格式 | `YYYY/MM/DD` | `YYYY-MM-DD` 或 `YYYYMMDD` | 斜杠分隔便于 `ListObjects` 按日期前缀过滤 |
| 新旧 Bucket 迁移 | YAML 改名 + 旧数据自然过期 | `mc mirror` 全量复制 | 当前数据量小，不值得全量迁移 |
| 补丁包存储 | 与固件同 Bucket 不同 category | 单独 Bucket | 补丁与固件生命周期相同（永久），访问模式一致 |
| 日志分子目录 | 同 Bucket 不同 prefix | 多个 Bucket（omc-running-logs 等） | MinIO ILM 支持 prefix 级别规则，无需拆 Bucket |
| 数据交换区 | 独立 Bucket | 混入 logs 或 config | 短 TTL（7天）、混合用途，与其他 Bucket 生命周期差异大 |

---

## 附录 A: 现有代码路径引用

| 模块 | 当前路径格式 | 目标路径格式 |
|------|-------------|-------------|
| software/service.go | `firmware/{carrier}/{product_class}/{version}/{filename}` | `img/{carrier}/{product_class}/{version}/{filename}` |
| transfer/bridge.go | `{YYYY/MM/DD}/{device_sn}/{filename}` | 无变化（已符合规范） |
| acs/upload/uploader.go | `{YYYY/MM/DD}/{device_sn}/{filename}` | 无变化 |
| filemanager/service.go | `managed-files/{file_type}/{YYYY-MM-DD}/{filename}` | 日期改为 `YYYY/MM/DD` |
| report/generator.go | `reports/{definition_id}/{period}/{record_id}.json` | `{report_type}/{YYYY}/{MM}/{definition_id}/{record_id}.{format}` |
| provision/model_upload.go | `datamodel_{device_sn}_{uuid}.xml`（扁平） | `datamodel/{carrier}/{oui}/{product_class}/{device_sn}_{timestamp}.xml` |

---

## 13. 部署架构

### 13.1 部署模式概述

MinIO 支持三种部署模式，按基站规模选用：

| 模式 | 节点数 | 适用规模 | 数据保护 | 运维复杂度 |
|------|--------|---------|---------|-----------|
| **Single-Node Single-Drive (SNSD)** | 1 节点 1 盘 | 开发/测试 | 无冗余 | 极低 |
| **Single-Node Multi-Drive (SNMD)** | 1 节点 N 盘 | ≤ 10,000 基站 | 纠删码（本机冗余） | 低 |
| **Multi-Node Multi-Drive (MNMD)** | 4+ 节点 | ≥ 30,000 基站 | 纠删码（跨节点冗余） | 中 |

```
开发/测试          小规模生产              中/大规模生产
(当前 docker-compose)

┌──────┐         ┌──────────┐         ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
│MinIO │         │  MinIO   │         │Node-1│ │Node-2│ │Node-3│ │Node-4│
│ SNSD │         │  SNMD    │         │ 4盘  │ │ 4盘  │ │ 4盘  │ │ 4盘  │
│/data │         │/data{1-4}│         └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘
└──────┘         └──────────┘            └────────┴────────┴────────┘
                                              纠删码 EC:4
1 台设备            1 台服务器                     4 台服务器
无冗余              允许 1 盘故障                   允许 1 节点故障
```

### 13.2 当前开发环境（docker-compose SNSD）

现有 `deployments/docker/docker-compose.yml` 中的 MinIO 配置：

```yaml
minio:
  image: minio/minio:latest
  environment:
    MINIO_ROOT_USER: minioadmin
    MINIO_ROOT_PASSWORD: minioadmin
  ports:
    - "9000:9000"   # S3 API
    - "9001:9001"   # Web Console
  volumes:
    - miniodata:/data
  command: server /data --console-address ":9001"
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
    interval: 5s
    timeout: 5s
    retries: 5
```

**问题**：单盘单节点，无纠删码，无资源限制，仅适合开发调试。

---

### 13.3 分级部署方案

#### Tier 1: 10,000 基站 — 单节点多盘 (SNMD)

**存储需求估算**：

| 指标 | 值 | 计算依据 |
|------|-----|---------|
| PM 月增量 | ~15 GB | 10K × 96 次/天 × 30 天 × 1.5 KB |
| MR 月增量 | ~20 GB | 10K × 96 次/天 × 30 天 × 7 KB |
| 配置备份月增量 | ~2 GB | 10K × 300 KB/月 |
| 固件仓库 | ~10 GB | 累积（版本不删） |
| 日志/报表/交换 | ~5 GB/月 | 按需触发，量小 |
| **月总增量** | **~42 GB** | |
| **峰值存储**（含保留期） | **~150 GB** | PM/MR 90 天 + 配置 365 天 + 固件永久 |

**硬件资源**：

| 资源 | 要求 | 说明 |
|------|------|------|
| **CPU** | 2 核 | MinIO 轻量，主要消耗在纠删码计算 |
| **内存** | 4 GB | 官方最低推荐；缓存热文件元数据 |
| **磁盘** | 4 × 250 GB SSD | SNMD 纠删码需至少 4 盘；可用容量 ~500 GB（EC:4 约 50% 冗余） |
| **网络** | 1 Gbps | 满足 PM/MR 并发上传 |
| **IOPS** | ≥ 1,000 | SSD 即可满足 |

**Docker Compose 配置**：

```yaml
# deployments/docker/docker-compose.10k.yml
services:
  minio:
    image: minio/minio:RELEASE.2025-03-12T18-04-18Z
    environment:
      TZ: "Asia/Shanghai"
      MINIO_ROOT_USER: "${MINIO_ROOT_USER}"
      MINIO_ROOT_PASSWORD: "${MINIO_ROOT_PASSWORD}"
      # 监控
      MINIO_PROMETHEUS_AUTH_TYPE: "public"
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - /data/minio/disk1:/data1
      - /data/minio/disk2:/data2
      - /data/minio/disk3:/data3
      - /data/minio/disk4:/data4
      - /etc/localtime:/etc/localtime:ro
    command: server /data{1...4} --console-address ":9001"
    deploy:
      resources:
        limits:
          cpus: "2"
          memory: 4G
        reservations:
          cpus: "1"
          memory: 2G
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    restart: unless-stopped
```

**磁盘挂载要求**：

```bash
# 4 块独立物理 SSD 挂载到不同路径（不要用同一块盘分区！）
# XFS 文件系统性能最��
mkfs.xfs /dev/sdb && mount /dev/sdb /data/minio/disk1
mkfs.xfs /dev/sdc && mount /dev/sdc /data/minio/disk2
mkfs.xfs /dev/sdd && mount /dev/sdd /data/minio/disk3
mkfs.xfs /dev/sde && mount /dev/sde /data/minio/disk4
```

> **注意**：SNMD 可容忍最多 N/2 - 1 盘故障（4 盘 = 1 盘故障），但服务器本身是单点。适用于允许短暂停机的场景。

---

#### Tier 2: 30,000 基站 — 4 节点分布式 (MNMD)

**存储需求估算**：

| 指标 | 值 |
|------|-----|
| PM 月增量 | ~45 GB |
| MR 月增量 | ~60 GB |
| 配置备份月增量 | ~6 GB |
| 固件仓库 | ~20 GB（累积） |
| 日志/报表/交换 | ~15 GB/月 |
| **月总增量** | **~126 GB** |
| **峰值存储** | **~500 GB** |

**硬件资源（每节点）**：

| 资源 | 要求 | 说明 |
|------|------|------|
| **CPU** | 4 核 | 分担纠删码计算 + 并发 I/O |
| **内存** | 8 GB | 官方推荐：每 TB 存储 1 GB 内存 |
| **磁盘** | 4 × 500 GB SSD | 每节点 4 盘；集群总裸容量 32 TB |
| **网络** | 10 Gbps | 节点间数据同步需要高带宽 |
| **IOPS** | ≥ 3,000/盘 | NVMe 优先 |

**集群总容量**：

```
裸容量：4 节点 × 4 盘 × 500 GB = 8 TB
可用容量（EC:4）：~4 TB（50% 纠删码开销）
安全水位（80%）：~3.2 TB
预留增长：峰值 500 GB → 6.4 倍余量，充裕
```

**Docker Compose 配置（4 节点 docker swarm 或独立主机）**：

```yaml
# deployments/docker/docker-compose.30k.yml
#
# 方案 A: 4 台独立主机各运行一个 MinIO 实例
# 方案 B: Docker Swarm / Compose 模拟 4 节点（下例为单机模拟，仅用于测试）
#
# 生产环境建议方案 A 或 K8s Operator

x-minio-common: &minio-common
  image: minio/minio:RELEASE.2025-03-12T18-04-18Z
  environment:
    TZ: "Asia/Shanghai"
    MINIO_ROOT_USER: "${MINIO_ROOT_USER}"
    MINIO_ROOT_PASSWORD: "${MINIO_ROOT_PASSWORD}"
    MINIO_PROMETHEUS_AUTH_TYPE: "public"
  command: >
    server
    http://minio-{1...4}/data{1...4}
    --console-address ":9001"
  deploy:
    resources:
      limits:
        cpus: "4"
        memory: 8G
      reservations:
        cpus: "2"
        memory: 4G
  healthcheck:
    test: ["CMD", "mc", "ready", "local"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 60s
  restart: unless-stopped

services:
  minio-1:
    <<: *minio-common
    hostname: minio-1
    volumes:
      - minio1-data1:/data1
      - minio1-data2:/data2
      - minio1-data3:/data3
      - minio1-data4:/data4

  minio-2:
    <<: *minio-common
    hostname: minio-2
    volumes:
      - minio2-data1:/data1
      - minio2-data2:/data2
      - minio2-data3:/data3
      - minio2-data4:/data4

  minio-3:
    <<: *minio-common
    hostname: minio-3
    volumes:
      - minio3-data1:/data1
      - minio3-data2:/data2
      - minio3-data3:/data3
      - minio3-data4:/data4

  minio-4:
    <<: *minio-common
    hostname: minio-4
    volumes:
      - minio4-data1:/data1
      - minio4-data2:/data2
      - minio4-data3:/data3
      - minio4-data4:/data4

  # Nginx 负载均衡器（4 节点前置）
  minio-lb:
    image: nginx:alpine
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - ./minio-nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - minio-1
      - minio-2
      - minio-3
      - minio-4

volumes:
  minio1-data1:
  minio1-data2:
  minio1-data3:
  minio1-data4:
  minio2-data1:
  minio2-data2:
  minio2-data3:
  minio2-data4:
  minio3-data1:
  minio3-data2:
  minio3-data3:
  minio3-data4:
  minio4-data1:
  minio4-data2:
  minio4-data3:
  minio4-data4:
```

**Nginx 负载均衡配置**：

```nginx
# deployments/docker/minio-nginx.conf
events {
    worker_connections 4096;
}

http {
    upstream minio_s3 {
        least_conn;
        server minio-1:9000;
        server minio-2:9000;
        server minio-3:9000;
        server minio-4:9000;
    }

    upstream minio_console {
        least_conn;
        server minio-1:9001;
        server minio-2:9001;
        server minio-3:9001;
        server minio-4:9001;
    }

    server {
        listen 9000;
        server_name _;

        # 允许大文件上传（固件 200MB、PM/MR 批量）
        client_max_body_size 500M;

        # 禁用缓冲，直接转发（流式上传）
        proxy_buffering off;
        proxy_request_buffering off;

        location / {
            proxy_pass http://minio_s3;
            proxy_set_header Host $http_host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            proxy_connect_timeout 300;
            proxy_http_version 1.1;
            proxy_set_header Connection "";
            chunked_transfer_encoding off;
        }
    }

    server {
        listen 9001;
        server_name _;

        location / {
            proxy_pass http://minio_console;
            proxy_set_header Host $http_host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}
```

**关键特性**：
- 允许 1 节点完全宕机，服务不中断
- 数据自动跨节点分布（纠删码 EC:4）
- Nginx 负载均衡器分发 S3 请求

---

#### Tier 3: 50,000 基站 — 4 节点分布式（扩容磁盘）

**存储需求估算**：

| 指标 | 值 |
|------|-----|
| PM 月增量 | ~75 GB |
| MR 月增量 | ~100 GB |
| 配置备份月增量 | ~10 GB |
| 固件仓库 | ~30 GB（累积） |
| 日志/报表/交换 | ~25 GB/月 |
| **月总增量** | **~210 GB** |
| **峰值存储** | **~800 GB** |

**硬件资源（每节点）**：

| 资源 | 要求 | 说明 |
|------|------|------|
| **CPU** | 4 核 | 与 30K 相同（MinIO CPU 瓶颈出现在 100K+ 并发） |
| **内存** | 16 GB | 更多缓存提升读性能 |
| **磁盘** | 4 × 1 TB NVMe SSD | ��磁盘容量而非节点数 |
| **网络** | 10 Gbps | 与 30K 相同 |
| **IOPS** | ≥ 5,000/盘 | NVMe 推荐 |

**集群总容量**：

```
裸容量：4 节点 × 4 盘 × 1 TB = 16 TB
可用容量（EC:4）：~8 TB
安全水位（80%）：~6.4 TB
峰值 800 GB → 8 倍余量
```

**与 Tier 2 的区别**：
- 架构不变（仍为 4 节点 MNMD）
- 每节点磁盘从 500 GB 升级到 1 TB
- 内存从 8 GB 升级到 16 GB
- Docker Compose 配置复用 Tier 2 模板，仅调整 resource limits

---

#### Tier 4: 100,000 基站 — 8 节点分布式 + 专用网络

**存储需求估算**：

| 指标 | 值 |
|------|-----|
| PM 月增量 | ~150 GB |
| MR 月增量 | ~200 GB |
| 配置备份月增量 | ~20 GB |
| 固件仓库 | ~50 GB（累积） |
| 日志/报表/交换 | ~50 GB/月 |
| **月总增量** | **~470 GB** |
| **峰值存储** | **~1.8 TB** |

**硬件资源（每节点）**：

| 资源 | 要求 | 说明 |
|------|------|------|
| **CPU** | 8 核 | 高并发纠删码计算 + 大批量 PM/MR 写入 |
| **内存** | 32 GB | 每 TB 存储 1 GB（保守）+ 系统开销 |
| **磁盘** | 4 × 2 TB NVMe SSD | 集群总裸容量 64 TB |
| **网络** | 25 Gbps | 节点间数据同步 + 大批量升级下发 |
| **IOPS** | ≥ 10,000/盘 | 高端 NVMe |

**集群总容量**：

```
裸容量：8 节点 × 4 盘 × 2 TB = 64 TB
可用容量（EC:4）：~32 TB
安全水位（80%）：~25.6 TB
峰值 1.8 TB → 14 倍余量（充分考虑未来增长到 50 万基站）
```

**关键差异（相比 Tier 2/3）**：
- 节点数翻倍（4→8），吞吐线性扩展
- 允许 2 节点同时故障（8 节点 EC:4 纠删码）
- 建议独立存储网络（VLAN 隔离），避免 PM/MR 批量上传抢占业务带宽
- 建议 Kubernetes Operator 部署（MinIO Operator），自动扩缩容

**Kubernetes 部署建议**：

```yaml
# deployments/k8s/infra/minio-tenant.yaml (MinIO Operator CRD)
apiVersion: minio.min.io/v2
kind: Tenant
metadata:
  name: omc-minio
  namespace: omc
spec:
  image: minio/minio:RELEASE.2025-03-12T18-04-18Z
  pools:
    - name: pool-0
      servers: 8                    # 8 节点
      volumesPerServer: 4           # 每节点 4 盘
      volumeClaimTemplate:
        spec:
          storageClassName: local-nvme   # 本地 NVMe StorageClass
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 2Ti          # 每盘 2 TB
      resources:
        requests:
          cpu: "4"
          memory: 16Gi
        limits:
          cpu: "8"
          memory: 32Gi
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: omc-minio
              topologyKey: "kubernetes.io/hostname"
  # S3 端点
  requestAutoCert: false
  exposeServices:
    minio: true
    console: true
  # Prometheus 指标
  prometheusOperator: true
  env:
    - name: MINIO_PROMETHEUS_AUTH_TYPE
      value: "public"
```

---

### 13.4 资源汇总对照表

| 维度 | Tier 1 (10K) | Tier 2 (30K) | Tier 3 (50K) | Tier 4 (100K) |
|------|-------------|-------------|-------------|--------------|
| **部署模式** | SNMD | MNMD | MNMD | MNMD |
| **节点数** | 1 | 4 | 4 | 8 |
| **每节点 CPU** | 2 核 | 4 核 | 4 核 | 8 核 |
| **每节点内存** | 4 GB | 8 GB | 16 GB | 32 GB |
| **每节点磁盘** | 4 × 250 GB SSD | 4 × 500 GB SSD | 4 × 1 TB NVMe | 4 × 2 TB NVMe |
| **总 CPU** | 2 核 | 16 核 | 16 核 | 64 核 |
| **总内存** | 4 GB | 32 GB | 64 GB | 256 GB |
| **总裸容量** | 1 TB | 8 TB | 16 TB | 64 TB |
| **可用容量 (EC:4)** | ~500 GB | ~4 TB | ~8 TB | ~32 TB |
| **峰值存储** | ~150 GB | ~500 GB | ~800 GB | ~1.8 TB |
| **余量倍数** | 3.3× | 8× | 10× | 17.8× |
| **网络** | 1 Gbps | 10 Gbps | 10 Gbps | 25 Gbps |
| **容错** | 1 盘 | 1 节点 | 1 节点 | 2 节点 |
| **月存储增长** | ~42 GB | ~126 GB | ~210 GB | ~470 GB |
| **PM/MR 写入峰值** | ~110 obj/s | ~330 obj/s | ~550 obj/s | ~1,100 obj/s |
| **推荐部署方式** | Docker Compose | Docker Compose + LB | K8s 或 Docker + LB | K8s Operator |

### 13.5 I/O 吞吐量估算

PM/MR 文件上传是 MinIO 的主要写入压力源。按 15 分钟采集粒度计算峰值（所有设备在 1 分钟内集中上传）：

| 规模 | 设备数 | 每周期文件数 (PM+MR) | 峰值窗口 | 峰值 TPS | 峰值带宽 |
|------|--------|---------------------|---------|---------|---------|
| 10K | 10,000 | 20,000 | 60s | ~333 obj/s | ~3 MB/s |
| 30K | 30,000 | 60,000 | 60s | ~1,000 obj/s | ~10 MB/s |
| 50K | 50,000 | 100,000 | 60s | ~1,667 obj/s | ~17 MB/s |
| 100K | 100,000 | 200,000 | 60s | ~3,333 obj/s | ~33 MB/s |

> MinIO 单节点 4 盘 NVMe 可达 ~5,000+ PUT/s，4 节点集群可达 ~20,000 PUT/s，100K 规模绰绰有余。

**大批量固件升级场景**（万台同时下载）：

| 规模 | 并发下载数 | 固件大小 | 所需带宽 | MinIO 节点数建议 |
|------|----------|---------|---------|----------------|
| 1,000 台 | ~100 并发 | 100 MB | ~1 GB/s 峰值 | 1 节点足够 |
| 5,000 台 | ~500 并发 | 100 MB | ~5 GB/s 峰值 | 4 节点 + 10G 网络 |
| 10,000 台 | ~1,000 并发 | 100 MB | ~10 GB/s 峰值 | 8 节点 + 25G 网络 |

> 大规模升级建议使用 Presigned URL 让 CPE 直连 MinIO（或 CDN），避免 ACS 成为瓶颈。

---

### 13.6 与 OMC 其他组件的协同部署

完整 OMC 平台的全组件资源规划（仅列 MinIO 相关组件的协同关系）：

```
┌──────────────────────────────────────────────────────────────┐
│                    OMC 完整部署拓扑                            │
│                                                              │
│   ┌─────────┐     ┌─────────┐     ┌───────────┐             │
│   │ Nginx   │────►│ omcgo-  │────►│ PostgreSQL│             │
│   │ :8080   │     │ app     │     │ :5432     │             │
│   │ :8081   │     │ :8082   │     └───────────┘             │
│   └────┬────┘     └────┬────┘            ▲                  │
│        │               │                 │                  │
│   ┌────▼────┐     ┌────▼────┐     ┌──────┴──────┐          │
│   │ omcgo-  │     │  Redis  │     │ TimescaleDB │          │
│   │ acs     │     │  :6379  │     │  (PM/KPI)   │          │
│   │ :7547   │     └─────────┘     └─────────────┘          │
│   └────┬────┘                                               │
│        │          ┌─────────┐     ┌─────────────┐          │
│        │          │  NATS   │     │ omcgo-      │          │
│        └─────────►│  :4222  │────►│ worker      │          │
│                   └─────────┘     └──────┬──────┘          │
│                                          │                  │
│                                   ┌──────▼──────┐          │
│                                   │   MinIO     │          │
│                                   │   :9000     │          │
│                                   │  (集群)     │          │
│                                   └─────────────┘          │
└──────────────────────────────────────────────────────────────┘
```

**各规模完整平台资源**（含 MinIO + 其他组件）：

| 组件 | 10K 基站 | 30K 基站 | 50K 基站 | 100K 基站 |
|------|---------|---------|---------|----------|
| **omcgo-app** | 1×(2C/4G) | 2×(4C/8G) | 2×(4C/8G) | 3×(8C/16G) |
| **omcgo-acs** | 1×(2C/2G) | 2×(2C/4G) | 3×(4C/8G) | 4×(4C/8G) |
| **omcgo-worker** | 1×(2C/4G) | 2×(2C/4G) | 2×(4C/8G) | 3×(4C/8G) |
| **PostgreSQL** | 1×(4C/16G) | 1×(8C/32G) | 2×(8C/32G) HA | 3×(16C/64G) HA |
| **TimescaleDB** | 共用 PG | 独立 1×(4C/16G) | 独立 1×(8C/32G) | 独立 2×(8C/32G) HA |
| **Redis** | 1×(1C/2G) | 3×(2C/4G) Cluster | 6×(2C/4G) Cluster | 6×(4C/8G) Cluster |
| **NATS** | 1×(1C/1G) | 3×(1C/2G) | 3×(2C/4G) | 3×(2C/4G) |
| **MinIO** | 1×(2C/4G) SNMD | 4×(4C/8G) MNMD | 4×(4C/16G) MNMD | 8×(8C/32G) MNMD |
| **Nginx** | 1×(1C/1G) | 2×(2C/2G) | 2×(2C/2G) | 2×(4C/4G) |
| **总 CPU** | ~15 核 | ~60 核 | ~100 核 | ~250 核 |
| **总内存** | ~34 GB | ~130 GB | ~240 GB | ~600 GB |
| **服务器数** | 1-2 台 | 6-8 台 | 10-12 台 | 20-25 台 |

---

### 13.7 Docker Compose 部署指南

#### 开发环境（现有，无改动）

```bash
cd omcgo/deployments/docker
docker compose up -d                    # 启动全部服务（SNSD MinIO）
docker compose logs -f minio            # 查看 MinIO 日志
```

#### 小规模生产 (10K SNMD)

**前置条件**：1 台服务器，4 块独立 SSD。

```bash
# 1. 格式化并挂载 4 块 SSD
for i in b c d e; do
    mkfs.xfs /dev/sd${i}
    mkdir -p /data/minio/disk${i}
    mount /dev/sd${i} /data/minio/disk${i}
done

# 2. 写入 /etc/fstab 持久化挂载
# /dev/sdb  /data/minio/diskb  xfs  defaults,noatime  0  2
# ... (每块盘一行)

# 3. 设置环境变量
export MINIO_ROOT_USER=<strong_username>
export MINIO_ROOT_PASSWORD=<strong_password_32chars>

# 4. 启动
docker compose -f docker-compose.10k.yml up -d

# 5. 初始化 Bucket + 生命周期策略
mc alias set omc http://localhost:9000 $MINIO_ROOT_USER $MINIO_ROOT_PASSWORD
bash ../../scripts/minio_lifecycle.sh
```

#### 中规模生产 (30K/50K MNMD)

**前置条件**：4 台服务器，每台 4 块 SSD，10G 互联。

```bash
# 每台服务器上：
# 1. 格式化挂载磁盘（同上）
# 2. 确保 4 台主机名可解析（DNS 或 /etc/hosts）
#    minio-1 → 10.0.1.11
#    minio-2 → 10.0.1.12
#    minio-3 → 10.0.1.13
#    minio-4 → 10.0.1.14

# 方案 A: 每台服务器独立运行 MinIO（推荐生产）
# 在每台服务器上执行:
docker run -d --name minio \
  --net=host \
  -e MINIO_ROOT_USER=$MINIO_ROOT_USER \
  -e MINIO_ROOT_PASSWORD=$MINIO_ROOT_PASSWORD \
  -v /data/minio/disk1:/data1 \
  -v /data/minio/disk2:/data2 \
  -v /data/minio/disk3:/data3 \
  -v /data/minio/disk4:/data4 \
  minio/minio:RELEASE.2025-03-12T18-04-18Z \
  server http://minio-{1...4}/data{1...4} --console-address ":9001"

# 方案 B: Docker Compose 单机模拟（仅测试用）
docker compose -f docker-compose.30k.yml up -d
```

---

### 13.8 运维操作

#### 监控指标

MinIO 暴露 Prometheus 指标（`/minio/v2/metrics/cluster`），关键 Dashboard 指标：

| 指标 | 含义 | 告警阈值 |
|------|------|---------|
| `minio_cluster_capacity_usable_free_bytes` | 可用空间 | < 20% 总容量 |
| `minio_cluster_disk_offline_total` | 离线磁盘数 | > 0 |
| `minio_node_drive_free_bytes` | 单盘剩余 | < 10% |
| `minio_s3_requests_total` | S3 请求总数 | 监控趋势 |
| `minio_s3_requests_errors_total` | S3 错误数 | 错误率 > 1% |
| `minio_s3_traffic_sent_bytes` | 出流量 | 监控峰值 |
| `minio_s3_traffic_received_bytes` | 入流量 | 监控峰值 |
| `minio_heal_objects_total` | 修复对象数 | 监控修复进度 |

#### 扩容流程

MinIO 支持通过添加 **Server Pool** 在线扩容（不中断服务）：

```bash
# 增加第二个 pool（4 个新节点）
# 修改启动命令，追加�� pool 的 URL
server http://minio-{1...4}/data{1...4} http://minio-{5...8}/data{1...4}
```

> **注意**：MinIO 不支持向现有 pool 添加节点，只能添加�� pool。新 pool 的节点数和盘数可以不同。

#### 备份与灾备

| 策略 | 工具 | 频率 | 说明 |
|------|------|------|------|
| **Bucket 镜像** | `mc mirror` | 实时 | 镜像到异地 MinIO/S3 |
| **定期快照** | `mc cp --recursive` | 每日 | 冷备到廉价存储 |
| **版本恢复** | MinIO Versioning | 按需 | firmware bucket 启用版本控制 |
| **纠删码自愈** | MinIO 内置 Healing | 自动 | 磁盘恢复后自动重建丢失分片 |

```bash
# 设置异地镜像（灾备）
mc mirror --watch omc/omc-firmware remote/omc-firmware
mc mirror --watch omc/omc-config remote/omc-config
# PM/MR 不镜像（已解析入 TimescaleDB，文件为中间产物）
```

---

### 13.9 操作系统调优

生产环境 Linux 内核参数调优（所有 MinIO 节点）：

```bash
# /etc/sysctl.d/99-minio.conf

# 文件描述符（MinIO 大量并发连接 + 文件句柄）
fs.file-max = 1048576
fs.nr_open = 1048576

# 网络连接优化
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_tw_reuse = 1

# 网络缓冲区（10G/25G 网络需要更大缓冲）
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216

# 虚拟内存
vm.swappiness = 1                  # 尽量不用 swap
vm.dirty_ratio = 40                # 脏页比例（SSD 可适当放大）
vm.dirty_background_ratio = 10
```

```bash
# /etc/security/limits.d/99-minio.conf
minio    soft    nofile    1048576
minio    hard    nofile    1048576
minio    soft    nproc     65535
minio    hard    nproc     65535
```

XFS 挂载选项（MinIO 官方推荐）：

```bash
# /etc/fstab 示例
/dev/nvme0n1  /data/minio/disk1  xfs  defaults,noatime,nodiratime,logbufs=8,logbsize=256k  0  2
```

---

### 13.10 关键部署决策记录

| 决策 | 选择 | 备选方案 | 理由 |
|------|------|---------|------|
| 10K 部署模式 | SNMD (1 节点 4 盘) | 4 节点 MNMD | 10K 规模数据量小（月 42 GB），单节点足够；节省 3 台服务器成本 |
| 30K 起步��� 4 节点 | 4 节点 MNMD | 3 节点 | MinIO 纠删码最少需要 4 盘，4 节点 × 4 盘 = 16 盘是最小分布式配置 |
| 100K 用 8 节点 | 8 节点 MNMD | 4 节点扩盘 | 吞吐线性扩展（PM/MR 3,333 obj/s 峰值），2 节点容错 |
| 负载均衡 | Nginx L7 | HAProxy L4 | Nginx 已在 OMC 架构中使用，复用运维经验；支持 WebSocket（Console） |
| 文件系统 | XFS | ext4 | MinIO 官方推荐 XFS；大文件顺序写优于 ext4 |
| 磁盘类型 | SSD (10K/30K), NVMe (50K/100K) | HDD | PM/MR 小文件高并发写入，HDD 随机 I/O 性能不足 |
| K8s Operator | 100K 使用 | 全量 K8s | 10K-50K 规模 Docker Compose 足够，K8s 引入不必要的运维复杂度 |
