/**
 * T-0123-P2-a Console React Query hooks — frontend-core
 *
 * 5 hooks 与后端 5 endpoint 一一对应：
 *   useGroupTree           — GET /mml/group-tree
 *   useCommandSubFields    — GET /mml/commands/:id/sub-fields
 *   useRenderMML           — POST /mml/render (mutation)
 *   useParseMML            — POST /mml/parse  (mutation)
 *   useExecuteStatements   — POST /mml/execute-statements (mutation)
 *
 * 设计要点：
 *   - useGroupTree / useCommandSubFields 走 useQuery，staleTime 较长（数据低频变更）
 *   - render / parse / execute 走 useMutation（写操作 + 一次性结果）
 *   - mock fallback：P2-a 暂未实施 mock console service；hooks 直接走 mmlApi。
 *     P2-b 实施时若需 storybook 场景再补 mock。
 */

import { useMutation, useQuery } from '@tanstack/react-query';

import { mmlApi } from '../../services/api/mmlApi';
import type {
  GroupTreeNode,
  SubFieldDef,
  RenderRequest,
  ParseRequest,
  ParseResponse,
  ExecuteStatementsRequest,
  StructuredExecuteRequest,
  CommandCompatibility,
  FlatGroupTreeResponse,
} from '../../types/mmlConsole';
import type { MMLTask } from '../../types/mml';

/**
 * 命令分组树。
 *
 * @param root 根 group_code（空 = 全树）
 * @param lang 'zh-CN' | 'en-US'
 *
 * staleTime 30min — 命令树是字典型数据，admin 改后通过 cache_version 失效。
 * P2-a 阶段简单 staleTime；P3 admin UI 加 invalidate hook 后再细化。
 */
export function useGroupTree(
  root?: string,
  lang: string = 'zh-CN'
): ReturnType<typeof useQuery<GroupTreeNode[]>> {
  return useQuery({
    queryKey: ['mml', 'console', 'group-tree', root ?? '', lang],
    queryFn: () => mmlApi.buildGroupTree(root, lang),
    staleTime: 30 * 60 * 1000,
  });
}

/**
 * Task #9: format=flat 命令树。2 层结构（分组 → 命令叶子），
 * 命令叶子直接携带 object_path。
 *
 * 与 useGroupTree 并存：新控制台页面走 flat，旧 admin/Catalog 页面仍走 hierarchical。
 * staleTime 同为 30min。
 */
export function useGroupTreeFlat(
  lang: string = 'zh-CN'
): ReturnType<typeof useQuery<FlatGroupTreeResponse>> {
  return useQuery({
    queryKey: ['mml', 'console', 'group-tree', 'flat', lang],
    queryFn: () => mmlApi.buildGroupTreeFlat(lang),
    staleTime: 30 * 60 * 1000,
  });
}

/**
 * 命令的 sub-fields（join standard_params 元数据；mml_params 表已下线）。
 *
 * enabled = Boolean(commandId) 防止首次渲染没选命令时空查。
 * staleTime 与 GroupTree 一致。
 */
export function useCommandSubFields(
  commandId: string | undefined,
  lang: string = 'zh-CN'
): ReturnType<typeof useQuery<SubFieldDef[]>> {
  return useQuery({
    queryKey: ['mml', 'console', 'sub-fields', commandId ?? '', lang],
    queryFn: () => mmlApi.getCommandSubFields(commandId!, lang),
    staleTime: 30 * 60 * 1000,
    enabled: Boolean(commandId),
  });
}

/**
 * R-8.5 命令兼容性警告：对当前选中的 product_class 计算"不兼容命令 ID 集合"。
 *
 * select 把 unsupportedCommandIds 数组转 Set，调用方按 commandID O(1) 查询。
 *
 * enabled = !!productClass —— 首次渲染或字典空时 productClass 为空，不发起查询。
 * staleTime 5min —— product / ParamModel 字典低频变更（admin 改后才触发）。
 */
export function useCommandCompatibility(
  productClass: string | undefined
): ReturnType<typeof useQuery<CommandCompatibility, Error, Set<string>>> {
  return useQuery({
    queryKey: ['mml', 'console', 'command-compatibility', productClass ?? ''],
    queryFn: () => mmlApi.getCommandCompatibility(productClass!),
    staleTime: 5 * 60 * 1000,
    enabled: Boolean(productClass),
    select: (data) => new Set(data.unsupportedCommandIds ?? []),
  });
}

/**
 * Statement → mml 字符串片段（UI 操作 → textbox 同步方向）。
 *
 * P2-a 阶段优先走本地简化渲染器（store/mmlConsoleStore.ts 内 `renderStatementLocal`）；
 * 本 mutation 仅在以下场景调用：
 *   - admin 自定义命令复杂模板，本地 renderer 兜不住 → 调服务端
 *   - 客户端 unit test / debug 需要权威渲染
 *
 * 一般业务无需调用此 hook（store 内部走本地渲染省网络）。
 */
export function useRenderMML(): ReturnType<typeof useMutation<string, Error, RenderRequest>> {
  return useMutation({
    mutationFn: (req: RenderRequest) => mmlApi.renderMML(req),
  });
}

/**
 * mml 字符串 → Statement[]（textbox 改 → debounce → 同步方向）。
 *
 * mutation 而非 query 因为：
 *   1. 副作用：解析依赖服务端命令字典查找，命中后填 commandId / subFields
 *   2. 调用时机：用户输入 300ms 防抖后触发，非声明式订阅
 *   3. 错误处理：parse_errors 在 body 内（始终 200），调用方按需处理
 *
 * store 内部用 setTimeout 调度此 mutation。
 */
export function useParseMML(): ReturnType<typeof useMutation<ParseResponse, Error, ParseRequest>> {
  return useMutation({
    mutationFn: (req: ParseRequest) => mmlApi.parseMML(req),
  });
}

/**
 * N 设备 × M statements 扇出执行。
 *
 * 后端 201 Created；返回 MMLTask（含 task_id / status / commands）。
 * 调用方通常导航到任务详情页 `/mml/tasks/${task.id}` 看推进状态。
 *
 * 失败语义：
 *   - 400 — 编译失败（sub_field 缺失 / target_object 缺失 / RMV 缺 index）
 *   - 404 — command_id 未命中
 *   - 500 — DB / fanout 内部错
 *   响应 envelope 已被 http 拦截器剥壳，error.userMessage 含中文提示。
 */
export function useExecuteStatements(): ReturnType<
  typeof useMutation<MMLTask, Error, ExecuteStatementsRequest>
> {
  return useMutation({
    mutationFn: (req: ExecuteStatementsRequest) => mmlApi.executeStatements(req),
  });
}

/**
 * R-9.2 结构化通道 mutation：POST /mml/console/execute-statements-structured。
 *
 * 与 useExecuteStatements 的差异：
 *   - 入参带 paths/values(key=standardPath)/instanceIndices，无需 MML 文本 round-trip
 *   - 422 错误体附带 unknown_paths 元数据（http 拦截器透传到 error.response.data）
 *
 * 调用方：RightPanel.ConsoleActionBar.onExecute（命令树选中命令后走此通道）。
 * MmlEditor（多语句脚本 / 裸 MML）继续走 useExecuteStatements（旧通道），因为
 * parser-emitted statement 可能缺 subFields 元数据无法做 structured 转换。
 */
export function useExecuteStatementsStructured(): ReturnType<
  typeof useMutation<MMLTask, Error, StructuredExecuteRequest>
> {
  return useMutation({
    mutationFn: (req: StructuredExecuteRequest) => mmlApi.executeStatementsStructured(req),
  });
}
