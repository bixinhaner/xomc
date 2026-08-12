# 即插即用策略提交保护审查报告

## 审查范围

- `webcode/src/pages/device/PlugAndPlay/AddPolicyPage.tsx`
- `webcode/src/pages/device/PlugAndPlay/policySubmitAvailability.ts`
- `webcode/src/pages/device/PlugAndPlay/policySubmitAvailability.test.ts`

## 结论

PASS

未发现 CRITICAL 或 WARNING 级问题。

## 审查摘要

- 编辑模式在持久化策略与产品目录准备完成前阻止提交，避免使用未完成加载的数据覆盖策略。
- 同步提交锁与 mutation 状态共同防止重复提交，锁在 `finally` 中可靠释放。
- 表单校验失败后滚动到首个错误字段，且保留原有 409 冲突提示。
- 提交可用性逻辑抽取为纯函数，并覆盖加载、就绪和提交中状态。
- 未引入 API、数据模型、权限或持久化结构变更。

## 验证

- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb/webcode && npm test -- --run src/pages/device/PlugAndPlay/policySubmitAvailability.test.ts`：2 个测试通过。
- `npm run build`：通过，并已部署到本地 Compose 环境。
- `curl -I http://localhost:8081/`：返回 `HTTP/1.1 200 OK`。

## 风险与建议

- 本次变更仅覆盖主皮肤 `webcode` 中现有页面；其他皮肤未发现对应的策略编辑页面改动点。
- 建议 MR 评审时重点复核策略详情请求失败时按钮保持禁用的产品预期。
