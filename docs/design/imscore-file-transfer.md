# 核心网（IMS Core）文件传输接入设计 — 参数配置（FT1~FT12）

日期：2026-08-20
状态：v1 开发中
范围：imscore_file.txt 全部序号 1~19。1~12 参数配置类；13~19 为二期补充：
日志（FT13/18/19）、License（FT15）、恢复（FT17）、下载鉴权（FT14）、备份（FT16）。

## 1. 需求与口径

| 项 | 决策 |
|---|---|
| 任务模板 | 两个：核心网文件采集（Upload RPC）/ 核心网文件下发（Download RPC），方向由 RPC 方法表达 |
| CWMP FileType（Upload/Download RPC 共用） | 统一 `Ims File`（imscore_filetask.txt） |
| 子类型标签 | 统一 `<ParameterType>`，取值 `FT_ImsCore_*`（imscore_filetask.txt 类型标识）；页面下拉叫「文件类型」 |
| ParamType 承载 | ① SOAP 报文 `<ParameterType>FT_ImsCore_Ims_User_Setting_UD</ParameterType>` 标签（CPE 识别）；② 上传/下载 URL `paramType=FT_ImsCore_Ims_User_Setting_UD` query（ACS 落盘识别） |
| 传输方向 | FT1~FT7 支持上传+下载；FT8~FT12 仅上传（方向基于核心网视角：「上传」=核心网→OMC 发文件、「下载」=OMC→核心网收文件） |
| 页面入口 | 任务管理新 tab（核心网，与配置文件备份等并行）；文件管理新 tab（参数文件库） |
| 任务创建 | 任务类型选「参数配置」（v1 = 采集 / 下发两个模板）；选参数配置后可选参数类型；下发还需选参数文件 |

### 1.1 参数类型注册表（FT_ImsCore_*）

ParamType 取值沿用 imscore_file.txt 的「File type(ID)」列，仅把 `FTx` 编号前缀改为
`FT_ImsCore`（例如 `FT3_Ims_User_Setting_UD` → `FT_ImsCore_Ims_User_Setting_UD`）。

| ParamType（= File type(ID) 改造） | 功能名称 | 方向 |
|---|---|---|
| FT_ImsCore_Policy_Setting_UD | 流量策略设置 | 上传+下载 |
| FT_ImsCore_User_Setting_UD | 开户设置 | 上传+下载 |
| FT_ImsCore_Ims_User_Setting_UD | IMS用户设置 | 上传+下载 |
| FT_ImsCore_Pbx_User_Setting_UD | PBX用户设置 | 上传+下载 |
| FT_ImsCore_User_Apn_Setting_UD | 用户APN设置 | 上传+下载 |
| FT_ImsCore_Ue_IMEI_UD | IMEI设置 | 上传+下载 |
| FT_ImsCore_Ue_Route_UD | UE 路由设置 | 上传+下载 |
| FT_ImsCore_User_Location_Info_U | 用户位置信息 | 仅上传 |
| FT_ImsCore_eNBgNB_Location_Info_U | 基站位置信息 | 仅上传 |
| FT_ImsCore_Signaling_Events_U | 信令事件 | 仅上传 |
| FT_ImsCore_Sip_Events_U | Sip事件 | 仅上传 |
| FT_ImsCore_Cdr_U | 呼叫详单 | 仅上传 |
| FT_ImsCore_Operation_Logs_U | 操作日志（13） | 仅上传 |
| FT_ImsCore_Core_Logs_U | Core日志（18） | 仅上传 |
| FT_ImsCore_Web_Logs_U | Web日志（19） | 仅上传 |
| FT_ImsCore_Download_Auth_D | 下载鉴权文件（14） | 仅下载（文件库下发） |
| FT_ImsCore_Upload_License_U | License文件（15） | 仅上传 |
| FT_ImsCore_Backups_D | 备份文件（16） | 仅下载（文件库下发） |
| FT_ImsCore_Recovery_U | 恢复数据（17） | 仅上传 |

日志三类 FileType=`Ims Log File` + `<LogType>`；其余四类各自单一 FileType 无标签。
任务模板：核心网日志采集（LogType 三选一）/ 核心网License采集 / 核心网恢复数据采集 /
核心网鉴权文件下发 / 核心网备份文件下发（文件库选文件）。上传 URL 别名：
IMS_LOG / IMS_LICENSE / IMS_RECOVERY。MinIO 分类：ims-log / ims-license / ims-recovery
（config_backup 桶）。文件库（下发源）仅承载可下发类型：参数 FT1~7 + 鉴权 + 备份。

单一事实源：`omcgo/internal/imsparam/param_types.go`（后端校验 + `GET /imsparam/param-types` 供前端）。

## 2. 关键映射决策

两个业务方向复用现有两条成熟链路，避免新写执行器：

- **采集（设备 → OMC，Upload RPC）**：复用 `software.BatchCollect`（RUNTIME_LOG_COLLECT /
  CONFIG_BACKUP 同款），不碰 executor。
- **下发（OMC → 设备，Download RPC）**：复用 LICENSE_UPGRADE / CONFIG_RESTORE 的
  「占位任务 + 直接派发 device_tasks + CommandKey 对齐回推 TC」链路。

### 2.1 UFTE 模板（category = `ims_core`，标签「核心网」）

| TypeCode | DisplayName | RPCType | FileType（catalog） | softwareTaskType |
|---|---|---|---|---|
| `IMS_PARAM_COLLECT` | 核心网参数采集 | UPLOAD | `ImsCore Parameters File` | LogCollect（BatchCollect） |
| `IMS_PARAM_DISTRIBUTE` | 核心网参数下发 | DOWNLOAD | `IMS_PARAM_DISTRIBUTE`（反查键，见 2.2） | LogCollect（占位+派发） |

TransportPath（采集）：`/smallcell/FileUploadService?fileType=IMS_PARAM&paramType={paramType}&sn={sn}&taskId={taskId}&filename=`
TargetFileNameTemplate（采集）：`{paramTypeShort}_{yyyyMMddHHmmss}.dat`（如 Policy_Setting_UD_20260820153201.dat）
（executor 固定生成：paramType 去 FT_ImsCore_ 前缀 + 年月日时分秒，不依赖模板值。）

### 2.2 任务行的反查与 ParamType 落库

`resolveTaskType` 靠 `task.download_file_type == catalog.FileType` 精确回找模板。两个 IMS
模板若共用同一 FileType 会歧义，且 ParamType 是任务级维度。方案：

- 任务行 `download_file_type` 存 `"<反查键>:<ParamType>"`：
  - 采集：`ImsCore Parameters File:FT_ImsCore_Policy_Setting_UD`
  - 下发：`IMS_PARAM_DISTRIBUTE:FT_ImsCore_Policy_Setting_UD`
- `resolveTaskType` 增加兜底：精确匹配失败且串含 `:` 时，去掉最后一段再精确匹配一次
  （沿用 CONFIG_RESTORE 用 `<OUI>` 与 `{OUI}` 区分 catalog 字面/真实值的先例——catalog
  字面值只当反查键用，真正下发 SOAP 的 FileType 由代码写死为 `ImsCore Parameters File`）。
- `BatchCollectRequest` 增加 `StoredFileType`（落库值），与 `FileType`（CWMP 下发值）解耦。
- `PlaceholderTrackingRequest` 增加 `FileName`（下发任务存所选文件 UUID，供挂起/定时任务
  Start 时重派发）。
- `mapTask` 对 ims_core 任务把 ParamType 附加到 `typeDisplayName`（如「核心网参数采集（IMS用户设置）」）。

### 2.3 物理表（按业务拆表惯例）

- 新表对：`ims_param_collect_tasks` / `ims_param_collect_sub_tasks`（复制 config_backup 对的
  DDL；business_type = `ims_param_collect`），折回 `000001_init_schema.sql`。
- `device_active_tasks.business_type` CHECK 增加 `ims_param_collect`。
- `TransferRepoRouter` 增加 `ImsParamCollect` set：`ForUploadFileType` 对字面值
  `ImsCore Parameters File` 命中（route hint 由 BatchCollect 注入）；routing repos 的
  fan-out（pickByID / allTaskRepos / allSubTaskRepos / AllSets）全部纳入 → reaper / TC /
  file-landed 自动覆盖。
- 下发占位任务走 Default（upgrade_tasks 旧表），与 CONFIG_RESTORE / LICENSE_UPGRADE 一致。

### 2.4 存储布局（MinIO）

复用 `config_backup` 桶（不新增 bucket，避免配置/部署面扩大）：

- 设备采集落地：`ims-param/{yyyy/MM/dd}/{filename}`（tr069.FileType 新增 `IMS_PARAM`，
  `BucketAndCategory` → config_backup 桶 + `ims-param` 分类）。
- 参数文件库（运营者上传，供下发）：`ims-params/{FT_ImsCore_*}/{fileName}`。

### 2.5 采集完成链路

设备 PUT `?fileType=IMS_PARAM&paramType=FT_ImsCore_Ims_User_Setting_UD&sn=..&taskId=..` →

1. `normalizeFileType("IMS_PARAM")` → `tr069.FileTypeImsCoreParam`。
2. 规范命名（filename= 留空兜底）：`{paramType去前缀}_{yyyyMMddHHmmss}.dat`，与 executor 目标文件名同款格式。
3. 落 MinIO 后发 `backup.file.received`（复用）→
   - `FilePathRecorder` 写 `backup_restore_file`（UFTE 设备列表 presigned 下载反查）；
   - `HandleFileLandedForCollect` 把子任务推 Completed；后续 TC 到达走 handleTCBody 幂等。

### 2.6 下发完成链路

`CommandKey = IMS_PARAM_DISTRIBUTE_{taskId8}_{sn}`（BuildDirectDispatchCommandKey）；
device_task Download RPC：`file_type=ImsCore Parameters File` + `<ParameterType>{FT_ImsCore_*}</ParameterType>`，
`url={base}/smallcell/FileDownloadService/{bucket}/ims-params/{FT_ImsCore_*}/{fileName}?paramType={FT_ImsCore_*}`，
`target_file_name={fileName}`。TC 由 handleTCBody 按命令键回推子任务（同 LICENSE_UPGRADE）。

### 2.7 SOAP 报文 ParamType 标签

- `soap.UploadData` / `soap.DownloadData` 增 `ParamType`（json `param_type`）；
  `uploadXML` / `downloadXML` 在 `<FileType>` 后条件渲染 `<ParameterType>..</ParameterType>`
  （unqualified，vendor 扩展）。**仅 ParamType 非空时渲染**——固件/配置/日志等既有任务
  报文结构与现网完全一致，不影响任何厂商 CPE。
- 采集：`executor.ExecuteOneUpload` 把任务行落库值 `ImsCore Parameters File:FT_ImsCore_*` split 成
  CWMP FileType + ParamType，分别进 `<FileType>` 与 `<ParameterType>`（`software.splitStoredFileType`）。
- 下发：`imsparam.Service.enqueueParamDownload` 直接把 `param_type` 塞进 Download params。

## 3. 后端改动清单

| 文件 | 改动 |
|---|---|
| `pkg/tr069/filetype.go` | 新增 `FileTypeImsCoreParam "IMS_PARAM"`（label/IsUpload/IsDownload） |
| `pkg/soap/templates.go` | UploadData/DownloadData 增 `ParamType`；uploadXML/downloadXML 条件渲染 `<ParameterType>` |
| `internal/core/storage/router.go` | IMS_PARAM → config_backup 桶 + `ims-param` 分类 |
| `internal/imsparam/`（新模块） | param_types.go（FT_ImsCore_* 注册表）、model.go、repository.go（squirrel）、service.go（导入/列表/删除/下载 + DispatchByFileID + Preview）、handler.go（`/imsparam/param-files`、`/imsparam/param-types`） |
| `internal/software/service.go` | BatchCollectRequest.StoredFileType；PlaceholderTrackingRequest.FileName；splitStoredFileType |
| `internal/software/executor.go` | ExecuteOneUpload split 落库值 + params 增 param_type |
| `internal/software/transfer_router.go` / `routing_repos.go` | ImsParamCollect set + fan-out |
| `internal/transfer/repo/repository.go` | Business/Table 常量 |
| `internal/ufte/model.go` | 2 个内置模板；resolveTaskType 尾段剥离重试；CreateTaskRequest 增 `paramType` / `fileId` |
| `internal/ufte/service.go` | CreateTask 两条分支；StartTask / startDirectDispatchTask / ResumeLogCollectSubTask 支持 IMS；ImsParamDispatcher 注入；mapTask 显示名附加 ParamType |
| `internal/acs/upload/handler.go` | normalizeFileType 别名；IMS 规范命名；发 backup.file.received |
| `migrations/000001_init_schema.sql` | ims_param_collect 任务表对 + ims_param_files + CHECK + 索引 |
| `migrations/seed/000001_init_seed.sql` | ufte_task_types 两行内置模板 |
| `cmd/app/provider/modules.go` / `router.go` | repo/路由/派发器 wiring |

## 4. 前端改动清单（仅 V1 webcode）

| 文件 | 改动 |
|---|---|
| `frontend-core/src/types/unifiedFileTransfer.ts` | CreateInput 增 `paramType` / `fileId` |
| `frontend-core/src/services/api/imsParamApi.ts`（新） | 列表/类型/导入/下载/删除/批删 |
| `frontend-core/src/hooks/api/useImsParam.ts`（新） | react-query hooks |
| `webcode/src/pages/transfer/ImsParamLibrary/index.tsx`（新） | 参数文件库（筛选/上传 Modal/下载/删除） |
| `webcode/src/pages/transfer/FileManagement/index.tsx` | 新 tab `imsParam`（与配置文件并行） |
| `webcode/src/pages/transfer/shared.ts` | BUILTIN_TYPE/CATEGORY 集合、DEFAULT_CATEGORY_ORDER、任务名前缀 |
| `webcode/src/pages/transfer/FileTransferCenter/index.tsx` | 创建抽屉：参数类型 Select（下发限 FT1~FT7）、下发文件 Select + 「打开参数文件管理」 |
| i18n zh-CN / en-US | 新增 `ufte.builtin.category.ims_core`、`ufte.builtin.type/desc.IMS_PARAM_*`、`transfer.imsParamLib.*` 等 |

## 5. 验证计划

- 后端：`go build ./...` + 相关包 `go test ./internal/imsparam/... ./internal/ufte/... ./internal/software/... ./internal/acs/upload/...`。
- 迁移：`bash scripts/check-migrations.sh`（无 000002+ 新文件，基线折回合规）。
- 前端：`npm run build`（webcode）+ lint；交互按 AGENTS.md 需真实浏览器验证（待环境）。
- 端到端：cpe 模拟器 Upload（fileType=IMS_PARAM）→ 任务推进 + 文件可下载；下发任务 TC 回推。

## 6. 后续迭代（不在本期）

- 日志类（FT13/18/19）、License（FT15）、备份/恢复（FT16/17）、下载鉴权文件（FT14）模板。
- 采集文件 promote 进参数文件库（当前采集文件仅任务设备列表可下载）。
- `backup_restore_file` 增加 param_type 列 / 文件管理展示采集文件。
