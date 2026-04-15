# Code Review: MML 命令参数路径与操作感知渲染

**Date**: 2026-04-15
**Reviewer**: Claude (automated)
**Scope**: mml
**Conclusion**: PASS_WITH_WARNINGS

## Summary

MML 控制台 Phase 3 增强：后端新增 param-paths 端点与操作类型字段，前端新增 ParamFormRenderer/ParamPathPanel 组件实现操作感知的参数渲染，命令树改为分页加载，种子数据扩展 operation_type/param_paths/supported_operations/help_doc/notes 五列。

## Files Changed

| File | Lines | Risk | Description |
|------|-------|------|-------------|
| `omcgo/internal/mml/handler.go` | +27 | LOW | 新增 GetCommandParamPaths 端点 |
| `omcgo/internal/mml/model.go` | +25/-11 | LOW | MMLCommand 新增 5 字段 |
| `omcgo/internal/mml/pg_repository.go` | +32/-4 | LOW | scanCommand 扩展新列 |
| `omcgo/internal/mml/service.go` | +125 | LOW | GetCommandParamPaths + normalizeCommandParamPaths |
| `omcgo/internal/mml/handler_test.go` | +44 | LOW | 新端点成功路径测试 |
| `omcgo/internal/mml/service_test.go` | +69 | LOW | 字符串/对象两种 ParamPaths 格式测试 |
| `omcgo/migrations/seed/900004_mml_enhance.sql` | +431/-217 | LOW | 幂等加列 + 25 条命令元数据增强 |
| `omcmb/webcode/src/types/mml.ts` | +26/-1 | LOW | 新增 ParamPath/MMLOperationType 类型 |
| `omcmb/webcode/src/services/api/mmlApi.ts` | +57/-3 | LOW | executeCommand 重载 + mapBackendCommand 扩展 |
| `omcmb/webcode/src/hooks/api/useMML.ts` | +10/-1 | LOW | useExecuteMMLCommand 支持 payload |
| `omcmb/webcode/src/mock/services/mmlService.ts` | +29 | LOW | mock 适配新签名 |
| `omcmb/webcode/src/pages/mml/Console/components/CommandInput.tsx` | +335 | MEDIUM | 重构为受控组件 |
| `omcmb/webcode/src/pages/mml/Console/components/CommandTree.tsx` | +150 | LOW | 接收预构建 treeData + Spin |
| `omcmb/webcode/src/pages/mml/Console/components/ParamFormRenderer.tsx` | NEW +297 | MEDIUM | 按操作类型渲染参数表单 |
| `omcmb/webcode/src/pages/mml/Console/components/ParamPathPanel.tsx` | NEW +165 | LOW | 参数路径配置面板 |
| `omcmb/webcode/src/pages/mml/Console/constants.ts` | +3 | LOW | COMMAND_PAGE_SIZE |
| `omcmb/webcode/src/pages/mml/Console/hooks/useCommandExecution.ts` | +121 | MEDIUM | buildExecutePayload 操作感知 |
| `omcmb/webcode/src/pages/mml/Console/hooks/useCommandSelection.ts` | +161 | MEDIUM | 分页加载 + 异步详情获取 |
| `omcmb/webcode/src/pages/mml/Console/index.tsx` | +205 | MEDIUM | 状态提升 + 命令行合成 |

## Findings

| # | Severity | File | Description |
|---|----------|------|-------------|
| 1 | WARNING | service.go | normalizeCommandParamPaths/isWritableOperation 缺少单元测试；空/nil 输入、无效 JSON 等边界用例未覆盖 |
| 2 | WARNING | handler_test.go | 仅测试成功路径；缺少无效 UUID、命令未找到等错误路径测试 |
| 3 | WARNING | pg_repository.go | scanCommand/scanCommandRow 约 40 行重复扫描逻辑，可提取公共辅助函数 |
| 4 | WARNING | CommandInput/ParamFormRenderer/ParamPathPanel/index.tsx | resolveOperationType 函数重复实现 4 次，应提取为共享工具函数 |
| 5 | WARNING | ParamFormRenderer.tsx | useEffect 依赖不完整：依赖 command?.id + operationType 但内部读取 value/onChange，当 value 独立变化时 formValues 将不同步 |
| 6 | WARNING | useCommandSelection.ts | selectCommand 中 getCommandById 失败时静默降级为列表级命令对象，用户无错误提示 |
| 7 | WARNING | ParamPathPanel.tsx | useEffect 依赖 onChange，若父组件未用 useCallback 稳定引用可触发无限循环 |
| 8 | INFO | handler.go | GetCommandParamPaths 使用 gin.H 响应封装，与其他 handler 直接返回实体的风格不一致 |
| 9 | INFO | mmlApi.ts | executeCommand 联合类型重载改变了方法的位置行为，建议拆分为独立方法 |
| 10 | INFO | useCommandSelection.ts | useQuery 直接使用 mmlApi 绕过 mock 开关，mock 模式下命令树无法使用 |
| 11 | INFO | index.tsx | 8+ 状态片段通过 useEffect 同步，交互微妙，建议使用 useReducer 统一管理 |

## Checklist

- [x] 类型安全：无 any 使用
- [x] SQL 安全：squirrel 参数化查询 + 种子 SQL 静态字面量
- [x] 错误处理：fmt.Errorf 正确包装
- [x] XSS：无 dangerouslySetInnerHTML
- [x] 无硬编码密钥/凭证
- [x] 测试：成功路径已覆盖，错误路径待补充
- [ ] Mock 开关：命令树 useQuery 绕过 createApiSwitch（INFO 级）
