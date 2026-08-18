# Code Review: PnP License 按 SN 展示

- 日期：2026-08-18
- 基线：`f6e8855`
- 作者：wangyong
- 范围：`omcmb/webcode/src/pages/device/PlugAndPlay`
- 结论：PASS

## 变更摘要

- 移除即插即用策略页 License 列表查询中的产品 ID 约束。
- 保留设备 SN 搜索，使预埋 License 在设备产品信息尚未建立时仍可展示。
- 新增查询参数单元测试，覆盖全量预埋列表与按 SN 搜索两条路径。

## 审查结果

### CRITICAL

无。

### WARNING

无。

### INFO

- 固件列表仍按产品 ID 查询，本次变更不会扩大软件升级包的可见范围。
- License 下发链路继续按设备 SN 匹配，未修改后端接口、数据库结构或文件内容。

## 验证

- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb/webcode && npm test -- --run src/pages/device/PlugAndPlay/licenseListParams.test.ts`：2 个用例通过。
- `npm run build`：通过。
- 浏览器：请求不再携带 `product_id`，预埋 License 可见。
- 本地 Docker Compose：Web/app/acs 已重建并启动，`http://localhost:8081/` 返回 200。

## 风险评估

低。策略页将展示全部预埋 License，用户仍可按 SN 搜索；实际下发仍由目标设备 SN 决定。
