# Issue #233：MME IP+PLMN 快速设置修复审查报告

- 审查日期：2026-07-31
- 审查范围：MLN/BM/BLQ 快速设置、参数模型映射、服务 PLMN 约束与下拉缓存同步
- 结论：PASS

## 变更摘要

- MLN 的 MME IP+PLMN 改为读取和写入 16 组固定索引叶，删除会误导聚合叶写入的参数映射。
- BM、BLQ 增加独立的服务 PLMN 列表配置；MLN、BM、BLQ 的 MME PLMN 改为只能从服务 PLMN 下拉列表选择。
- 服务 PLMN 保存成功后直接用已接受值或设备回读值更新参数搜索缓存，避免下拉继续显示旧候选。
- 增加 XML 加载测试和前端索引适配、PLMN 来源及缓存更新回归测试。

## 审查结果

### CRITICAL

无。

### WARNING

无。

### INFO

- 改动仅维护 V1 页面，符合当前项目单皮肤约束。
- 用户可见新增文案均通过中英文 i18n 提供。
- 未新增 SQL、数据库迁移、接口或运营商硬编码分支。
- 参数搜索接口可能短暂返回旧值；前端以任务成功值和设备回读值覆盖当前查询缓存，并保留重新查询兜底。

## 验证记录

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./internal/quicksettings/...`：通过。
- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb && npm run test --workspace webcode -- --run src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/parameterSearchRefresh.test.ts src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/mmePlmnSelection.test.ts src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/mmeIpPlmnIndexed.test.ts`：3 个测试文件、12 个测试通过。
- 本地 Docker Web/App/ACS 构建并启动成功，`http://127.0.0.1:8081/` 返回 200。
- 可见 Chrome 只读验证 BM 设备：服务 PLMN 回读值与 MME 下拉候选一致；未再次保存设备参数。

## 风险与回滚

- 风险集中在不同设备型号使用不同服务 PLMN 叶和 MLN 固定索引叶；当前通过型号白名单与单独适配函数隔离。
- 回滚可直接撤销本提交；不涉及数据库或不可逆数据变更。
