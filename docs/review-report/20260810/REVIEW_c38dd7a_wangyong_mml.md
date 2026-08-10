# Issue #274 MML 多实例修复审查报告

## 结论

PASS

未发现 CRITICAL 或 WARNING 问题。

## 审查范围

- MML 多实例路径编译、产品参数映射与 ADD/RMV/LST/MOD 执行链
- ADD `{NEW}` 实例替换、SPV 失败补偿删除及任务聚合
- RMV 实例下标兼容与重新执行安全性
- 控制台实例输入、参数校验、可选参数及 ADD 结果回显
- `000001_init_seed.sql` 的命令目录和 BM/BLN 参数模型数据

## 重点检查

- SQL：使用固定 seed 数据与关联更新，无字符串拼接 SQL 或外部输入注入点。
- 安全：未变更认证、授权、Token 或敏感数据处理。
- 兼容：兼容 `rmv_instance` 和单元素 `instance_indices`，多元素明确拒绝。
- 失败处理：ADD 后 SPV 失败会删除刚创建的实例；SPV 成功会跳过补偿命令。
- 前端：ADD 仅校验必填字段，可选空值不下发；重新执行仅使用当前历史记录快照。
- 测试：新增后端与前端回归覆盖成功、失败、缺失实例和错误命令重放场景。

## 验证结果

- `cd omcgo && go test ./internal/mml/...`：通过。
- `cd omcgo && go build ./...`：通过。
- `cd omcmb && npm run typecheck`：通过。
- MML 控制台 Vitest：4 个文件、67 个测试通过。
- 全新 Docker Compose 环境：schema、001 seed、TSDB schema 均退出码 0。
- `http://localhost:8081/`：HTTP 200。
- 空库 BM `ADD LTE_CELL`：包含 EARFCN、EnbType（默认 1）、X2Flag（默认 0）。

## 风险与建议

- 001 seed 变更范围较大，已通过清空数据卷后的从零迁移验证。
- ADD 成功结果中的参数值来自已成功提交的 SPV 快照；设备响应本身不返回写入值。
