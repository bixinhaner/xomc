# 系统配置文件上传契约回归修复设计

## 背景与归因

2026-07-21 的系统配置 MR 引入了两项仍在当前主线中的回归：

1. `46d7afd8` 将设备上传 URL 的查询参数解析为 `url.Values` 后再次编码。Go 会按参数名排序，破坏 OMC 与现网设备约定的 `filename=`/`fileName=` 尾部占位协议。
2. `d0602700` 新增 `GET /api/v1/admin/sysConfig/apply-batches/:id`，但没有补齐 !228 恢复的端点级 RBAC 授权，非 builtIn 用户可能无法轮询配置应用状态。

RFC1918 私网地址误拦截已由后续提交修复，本次不重复修改。运营商识别、MinIO 下载覆盖、旧库基线升级和 PM 保留值关系不纳入本次修复。

## 业务契约

- 运行日志上传模板以 `filename=` 结尾，设备在等号后自行追加文件名。
- 故障日志上传模板以 `fileName=` 结尾，设备在等号后自行追加文件名。
- XML/NV 配置备份保持 `fileType -> sn -> taskId -> filename` 的既有顺序，且 `filename` 位于最后。
- URL 基础地址仍执行现有 scheme、host、路径越界、凭据、fragment 等校验；生产环境继续允许设备网络可达的 RFC1918 地址。
- `GET /api/v1/admin/sysConfig/apply-batches/:id` 遵循现有内置角色兼容规则：admin/operator 获得全部端点，viewer 获得 GET 端点。

## 设计

### 设备上传 URL

保留通用 `BuildURL` 供下载对象路径使用。新增面向设备上传模板的构造入口：它验证基础 URL 和相对服务路径，但将已经由业务模板提供的原始查询串作为不透明协议片段保留，不做 `url.Values.Encode()` 重排。

软件任务直接传入 UFTE 模板渲染后的原始查询串。配置备份按固定业务顺序显式编码每个值，并保证 `filename` 最后。测试必须断言完整 URL 字符串，而不是仅用 `url.Parse().Query()` 比较参数集合。

### RBAC 数据迁移

新增 seed 增量迁移，在应用启动前幂等注册应用状态端点，并按现有内置角色规则写入 `role_api_permissions`。迁移可在已运行过 seed baseline 的数据库上前向执行，也可在全新数据库的 baseline 之后执行。

## 验证

- URL 单元测试覆盖运行日志、故障日志、XML 配置备份、NV 配置备份、反向代理前缀、特殊字符编码和非法相对路径。
- seed 静态测试确认迁移注册精确路径和 GET 方法，并向 admin/operator/viewer 三类内置角色授权。
- 运行相关 Go 包测试、后端全量构建与全量测试。

