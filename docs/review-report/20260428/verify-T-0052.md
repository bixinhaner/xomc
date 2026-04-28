# T-0052 验收报告 — frontend-core hooks/services 文件级对齐

- Wave / Block：W2.C.1
- 任务：让 `frontend-core/src/hooks/api/*.ts` 与 `frontend-core/src/services/api/*.ts` 文件数差距 ≤ 1
- 工作目录：worktree `agent-af49a578`，分支 `worktree-agent-af49a578`
- 日期：2026-04-28

## Baseline

```
services: 29 个 .ts（含 index.ts，实际 28 个 service）
hooks:    24 个 .ts
gap:      29 - 24 = 5
```

按文件级（章程 W2.C.1 公式：`services - hooks ≤ 1`）该差距不达标。

## 缺口分析

按文件名一对一映射后，下列 service 没有同名 hook：

| service                | 是否已被现有 hook 间接覆盖                                              | 行动              |
|------------------------|------------------------------------------------------------------------|-------------------|
| `adminApi.ts`          | `useSystem` 覆盖 user/role/group/menu/api-endpoint CRUD；其余高级运维操作未覆盖 | 新建 `useAdmin.ts` 包高级运维（changePassword、role api permission、role device groups、menu tree） |
| `apiPermissionApi.ts`  | 完全未覆盖                                                              | 新建 `useApiPermission.ts` |
| `templateApi.ts`       | `useConfig.useConfigTemplates*` 已用，但耦合 config 命名空间；为了保持文件级一一对应另建薄包装 | 新建 `useTemplate.ts` |
| `configSyncApi.ts`     | `useConfig.useUpdateConfigParam` 仅用 `pushConfig`，pull/status 未暴露   | 新建 `useConfigSync.ts` |
| `configBaselineApi.ts` | `useConfig` 全量覆盖 (Baseline / Tasks / NeighborParams)               | 不新建（已实质覆盖）|
| `authApi.ts`           | 由 `userStore` 直接消费（章程语：可豁免）                              | 不新建            |
| `index.ts`             | 不算 service                                                            | 不新建            |

## 新增产出

```
omcmb/frontend-core/src/hooks/api/useAdmin.ts        (95 行)
omcmb/frontend-core/src/hooks/api/useApiPermission.ts(45 行)
omcmb/frontend-core/src/hooks/api/useTemplate.ts     (74 行)
omcmb/frontend-core/src/hooks/api/useConfigSync.ts   (52 行)
```

### 设计要点

1. **不破坏现有 hook**。所有新 hook 使用独立 query key 命名空间（`['admin', ...]`、`['api-permissions', ...]`、`['templates', ...]`、`['config-sync', ...]`），与 `useSystem` / `useConfig` 的现有 key 不冲突，避免 invalidation cross-talk。
2. **保留 useMock 模式**。`useTemplate` 通过 `useMock ? configService.* : templateApi.*` 切换，复用现有 mock。其它三个 hook 是服务端运维端点，无 mock 等价物，直接调真实 API（与 `useLogs.useOperationLogs` 同模式）。
3. **类型安全**：禁止 `any`，从 `types/system.ts` import `ApiEndpoint` / `ApiPermission` / `MenuItem`，以及 `services/api/configSyncApi.ts` 内部对齐的 `ParameterValue` 形状。
4. **不新增依赖、不改 services/api/、不改 types/、不改 webcode/**（三 sub-agent 路径互斥）。

## 最终对齐

```
services: 29 个 .ts
hooks:    28 个 .ts
gap:      29 - 28 = 1   ✅ ≤ 1
```

## 验证

```bash
$ ls omcmb/frontend-core/src/services/api/*.ts | wc -l
29
$ ls omcmb/frontend-core/src/hooks/api/*.ts | wc -l
28
$ cd omcmb/webcode && npm run typecheck
> tsc --noEmit
# 干净退出，无错误
```

## 章程 W2.C.1 Pass 判定

| 公式 | 计算 | 结果 |
|------|------|------|
| `services - hooks ≤ 1` | `29 - 28 = 1 ≤ 1` | ✅ PASS |

## 后续建议（不在本任务范围）

- `configBaselineApi.ts` 当前由 `useConfig` 覆盖。若后续要严格保持 1:1 文件映射，可把 `useBaselineConfigs / useBaselineConfigById / useCreateBaselineConfig / useUpdateBaselineConfig / useDeleteBaselineConfigs` 抽出到独立 `useConfigBaseline.ts` —— 当前差距为 1 已达标，无紧迫性。
- `authApi.ts` 仍建议保留豁免：被 `userStore` 同步消费，引入 hook 反而增加间接层。
