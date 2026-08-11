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

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { mmlApi, type UnsupportedPathInfo } from '../../services/api/mmlApi';
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
  SearchCommand,
} from '../../types/mmlConsole';
import type { MMLCustomCommandPathDef, MMLTask } from '../../types/mml';
import {
  MML_CONSOLE_SUB_FIELDS_QUERY_KEY,
  MML_CUSTOM_COMMAND_PATHS_QUERY_KEY,
} from './mmlQueryKeys';

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
  lang: string = 'zh-CN',
  productClass?: string,
  deviceKey?: string,
): ReturnType<typeof useQuery<GroupTreeNode[]>> {
  // T-0172/T-0170: 产品/设备上下文加入 queryKey，切换设备时重取同源过滤树。
  return useQuery({
    queryKey: ['mml', 'console', 'group-tree', root ?? '', lang, productClass ?? '', deviceKey ?? ''],
    queryFn: () => mmlApi.buildGroupTree(root, lang, productClass, deviceKey),
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
  lang: string = 'zh-CN',
  productClass?: string,
  deviceKey?: string,
): ReturnType<typeof useQuery<FlatGroupTreeResponse>> {
  return useQuery({
    queryKey: ['mml', 'console', 'group-tree', 'flat', lang, productClass ?? '', deviceKey ?? ''],
    queryFn: () => mmlApi.buildGroupTreeFlat(lang, productClass, deviceKey),
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
  lang: string = 'zh-CN',
  deviceKey?: string,
  productClass?: string,
): ReturnType<typeof useQuery<SubFieldDef[]>> {
  return useQuery({
    // T-0170: queryKey 含设备/产品上下文，让切换目标后重新拉对应支持集合的 sub_field。
    queryKey: [
      ...MML_CONSOLE_SUB_FIELDS_QUERY_KEY,
      commandId ?? '',
      lang,
      deviceKey ?? '',
      productClass ?? '',
    ],
    queryFn: () => mmlApi.getCommandSubFields(commandId!, lang, deviceKey, productClass),
    staleTime: 30 * 60 * 1000,
    enabled: Boolean(commandId),
  });
}

export function useCustomCommandPaths(
  commandId?: string,
): ReturnType<typeof useQuery<MMLCustomCommandPathDef[]>> {
  return useQuery({
    queryKey: [...MML_CUSTOM_COMMAND_PATHS_QUERY_KEY, commandId ?? ''],
    queryFn: () => mmlApi.getTemplatePaths(commandId!),
    staleTime: 30 * 60 * 1000,
    enabled: Boolean(commandId),
  });
}

/**
 * 该产品已记录的「不支持参数 PATH」集合（含读/写标记，执行 path 不支持类故障自学习表）。
 * 「选择命令 / 配置参数」据此按命令读/写类型过滤。productId 为空时不请求、返回空。
 */
export function useUnsupportedPaths(
  productId?: string,
): ReturnType<typeof useQuery<UnsupportedPathInfo[]>> {
  return useQuery({
    queryKey: ['mml', 'console', 'unsupported-paths', productId ?? ''],
    queryFn: () => mmlApi.getUnsupportedPaths(productId),
    staleTime: 5 * 60 * 1000,
    enabled: Boolean(productId),
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
 * Bundle C — 命令搜索（按 command_code / logical_name / path / description 联合 ILIKE）。
 *
 * 调用方应**先做 debounce 300ms** 再把 q 传进来（在 CommandTree 用 useDebounce
 * 或类似 hook），避免每键击触发 RTT。
 *
 * enabled = q.trim() !== ''：空查询不触发请求；React Query 会保留上次结果，
 * CommandTree 据此切换"搜索结果列表"和"完整树视图"。
 * staleTime 30s — 命令字典低频变更，但用户连续搜索时应允许短时间复用结果。
 */
export function useSearchCommands(
  q: string,
  lang: string = 'zh-CN',
  limit: number = 50,
): ReturnType<typeof useQuery<SearchCommand[]>> {
  const trimmed = q.trim();
  return useQuery({
    queryKey: ['mml', 'console', 'search-commands', trimmed, lang, limit],
    queryFn: () => mmlApi.searchCommands(trimmed, lang, limit),
    staleTime: 30 * 1000,
    enabled: trimmed !== '',
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
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ExecuteStatementsRequest) => mmlApi.executeStatements(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] }),
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
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: StructuredExecuteRequest) => mmlApi.executeStatementsStructured(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] }),
  });
}

/**
 * 全设备汇总结果 CSV 导出（落 MinIO，地址记入 mml_tasks）。
 * 返回 { object, downloadUrl }；调用方拿 downloadUrl 触发浏览器下载。
 */
export function useExportTaskCSV(): ReturnType<
  typeof useMutation<Blob, Error, string>
> {
  return useMutation({
    // 同源流式下载（GET responseType=blob），调用方拿 Blob 触发浏览器保存。
    mutationFn: (taskId: string) => mmlApi.downloadTaskCsv(taskId),
  });
}

/** 单设备结果 CSV 同源流式下载（GET，返回 Blob 由调用方保存）。 */
export function useExportTaskDeviceCSV(): ReturnType<
  typeof useMutation<Blob, Error, { taskId: string; deviceSn: string }>
> {
  return useMutation({
    mutationFn: ({ taskId, deviceSn }: { taskId: string; deviceSn: string }) =>
      mmlApi.downloadTaskDeviceCsv(taskId, deviceSn),
  });
}
