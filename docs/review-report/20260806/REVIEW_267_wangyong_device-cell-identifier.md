# Issue 267 代码审查报告

## 审查范围

- 设备列表小区标识国际化与制式取值
- 多小区标识格式化与导出
- 设备参数同步时有效小区投影

## 审查结论

**PASS_WITH_WARNINGS**

## 检查结果

- `omcmb/webcode/src/pages/device/DeviceList/index.tsx`
  - 列表标题使用 `device.cellIdentifier` 国际化键。
  - NR/LTE/GSM 按现有后端字段展示；多值统一为英文逗号分隔。
  - 导出逻辑与表格显示口径一致。
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
  - 新增“小区标识”文案。
- `omcmb/frontend-core/src/i18n/en-US/index.ts`
  - 新增“Cell Identifier”文案。
- `omcgo/internal/device/device_info_sync.go`
  - 使用详情页小区组装逻辑投影有效小区标识。
  - 有 `NumOfCells` 时按配置数量过滤；未上报时按实际 FAPService 实例处理，兼容 BM 多小区站型。
  - GSM 使用 `InUse` 和有效小区数据判断。

## WARNING

- 已有设备的 `device_info.cell_id` 是持久化快照，需要再次执行参数同步后才能刷新历史值。
- MinIO 容器健康检查仍因镜像内缺少 `curl` 标记为 unhealthy，与本次代码无关。
- `npm run typecheck` 受现有 `SystemLicense/History.tsx` 和 `SystemLicense/index.tsx` 的 `Uint8Array`/`BlobPart` 类型错误阻断，与本次改动无关。

## 验证

- `go test ./internal/device`：通过。
- 前端生产构建：通过。
- `npm run typecheck`：未通过，失败点为上述既有 SystemLicense 类型错误。
- 本地 app/web Docker 重建：通过。
- `curl -I http://localhost:8081/`：HTTP 200。
