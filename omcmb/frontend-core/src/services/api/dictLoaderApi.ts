import http from '../http';

/**
 * T-0183 — dictload 字典 Loader 热重载 API。
 *
 * 后端契约:`cmd/app/provider/dictload_admin.go`
 * 端点:POST /api/v1/admin/dictload/reload?name=<loader_name>
 *
 * 5 个 loader: mml-standard / param-model / indicator / alarm-definition / product
 *
 * 适用场景:
 *   - operator 手工编辑 XML 字典文件后,需要 app 重读 + 写 DB
 *   - param-model loader 完成后自动调用 InvalidateCache(清 Redis L2 + bump cache_version),
 *     无需运维额外 redis-cli DEL
 *
 * 注意:omcctl device sweep-paths --apply 写完 XML 后会调用 Applier,Applier 内部已经
 * 一步搞定 XML 写 + DB 更新 + 缓存清,不需要再触发本端点。本端点是 "operator 手编 XML"
 * 兜底流程。
 */

/** 后端响应 envelope.data 字段。 */
export interface DictLoaderReloadResult {
  loader: string;
  elapsed_ms: number;
  rows_affected: number;
  files_loaded: number;
  files_skipped: number;
  errors?: string[] | null;
  /** T-0183: param-model loader reload 后自动清的 Redis L2 键数。 */
  cache_keys_cleared?: number;
  /** T-0183: 自增后的 parammodel:cache_version 值。 */
  cache_version_after?: number;
}

/** 已知 loader 名(后端注册的 5 个),前端列表用。 */
export type DictLoaderName =
  | 'param-model'
  | 'product'
  | 'mml-standard'
  | 'indicator'
  | 'alarm-definition';

export const KNOWN_DICT_LOADERS: ReadonlyArray<{
  name: DictLoaderName;
  labelI18n: { 'zh-CN': string; 'en-US': string };
  descriptionI18n: { 'zh-CN': string; 'en-US': string };
}> = [
  {
    name: 'param-model',
    labelI18n: { 'zh-CN': '参数模型字典', 'en-US': 'Param Model Dictionary' },
    descriptionI18n: {
      'zh-CN':
        '重读 data/param-mappings/*.xml,写 param_mappings 表。完成后自动清 Redis L2 缓存。',
      'en-US':
        'Reload XML param model dictionaries into param_mappings table. Auto-invalidates Redis L2 cache.',
    },
  },
  {
    name: 'product',
    labelI18n: { 'zh-CN': '产品装配件', 'en-US': 'Product Catalog' },
    descriptionI18n: {
      'zh-CN': '重读 data/param-mappings/products.xml,写 products 表。',
      'en-US': 'Reload products.xml into products table.',
    },
  },
  {
    name: 'mml-standard',
    labelI18n: { 'zh-CN': 'MML 标准命令', 'en-US': 'MML Standard Catalog' },
    descriptionI18n: {
      'zh-CN':
        'sha256 增量重载;命令树重建后用此入口让 app 立即生效,无需重启。',
      'en-US':
        'sha256 incremental reload of MML command catalog; effective without app restart.',
    },
  },
  {
    name: 'indicator',
    labelI18n: { 'zh-CN': 'KPI 指标库', 'en-US': 'KPI Indicator Library' },
    descriptionI18n: {
      'zh-CN': '重读 KPI 指标定义。',
      'en-US': 'Reload KPI indicator definitions.',
    },
  },
  {
    name: 'alarm-definition',
    labelI18n: { 'zh-CN': '告警定义库', 'en-US': 'Alarm Definition Library' },
    descriptionI18n: {
      'zh-CN': '重读告警定义 + i18n 翻译。',
      'en-US': 'Reload alarm definitions + i18n translations.',
    },
  },
];

export const dictLoaderApi = {
  /**
   * 触发指定 loader 热重载。返回执行报告(rows / files / elapsed / errors)。
   *
   * 鉴权:需要 super_admin 角色(后端在 superAdminGroup 下挂)。
   */
  async reload(name: DictLoaderName): Promise<DictLoaderReloadResult> {
    const { data } = await http.post<DictLoaderReloadResult>(
      '/admin/dictload/reload',
      undefined,
      { params: { name } },
    );
    return data;
  },
};
