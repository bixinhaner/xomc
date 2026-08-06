# Review: Admin State 与多实例下发操作禁用

## 范围

本次提交仅包含前端 TypeScript/TSX 文件，不提交任何 YAML 文件。

## 变更摘要

- 修正小区 `CellEnable.AdminState` 显示口径：`1` 为未锁定，`0` 为锁定。
- 多实例批量提交下发期间，禁用每行的编辑和删除操作。
- 保留原有 IPsec 专用状态限制和已有的新增按钮禁用逻辑。
- `batchAwaitingDevice` 持续为真期间保持禁用，设备任务完成并回读后恢复。

## 文件审查

- `omcmb/webcode/src/pages/device/DeviceDetail/index.tsx`
  - 修正详情页小区 Admin State 数值到标签和颜色的映射。
- `omcmb/webcode/src/pages/device/DeviceList/index.tsx`
  - 修正设备列表 Admin State 数值到标签和颜色的映射。
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/MultiInstanceTable.tsx`
  - 对普通多实例列表的编辑、删除按钮统一应用提交中和等待设备回读状态。

## YAML 文件排除

以下本地改动未纳入本次提交：

- `deployments/docker/docker-compose.yml`
- `omcgo/cmd/worker/etc/config.dev.yaml`
- `omcgo/cmd/worker/etc/config.local.yaml`

## 验证

- `npm run typecheck`：未通过，存在两个与本次改动无关的既有 `BlobPart` 类型错误：
  - `src/pages/SystemLicense/History.tsx:37`
  - `src/pages/SystemLicense/index.tsx:184`
- 本次修改文件未发现新增类型错误。
- 本地 Docker Web 页面此前已验证返回 `HTTP/1.1 200 OK`。

## 风险与影响

- Admin State 的 `1/0` 映射仅针对当前前端字段展示口径；原始后端值不变。
- 未修改接口、数据库和 YAML 配置。
- 提交下发期间禁止继续修改或删除，避免覆盖设备在途配置。
