/**
 * standard_params 无限滚动分页的纯辅助（无运行时依赖，仅类型）。
 * 抽离自 useMmlAdmin 以便单测——后者经 mmlAdminApi → http(axios) 链，测试环境难直接 import。
 */
import type { PageResponse } from '../../types/pagination';
import type { StandardParamView } from '../../types/mmlAdmin';

/** standard_params 单页拉取条数（触底分页 load-more）。后端 Limit() 上限 1000。 */
export const STANDARD_PARAMS_PAGE_SIZE = 100;

/** 把 useInfiniteQuery 的分页数据拍平为单一 items 数组，按 id 去重（保留首见顺序）。 */
export function flattenStandardParamPages(
  pages: PageResponse<StandardParamView>[] | undefined,
): StandardParamView[] {
  if (!pages) return [];
  const seen = new Set<string>();
  const out: StandardParamView[] = [];
  for (const page of pages) {
    for (const it of page.items) {
      if (seen.has(it.id)) continue;
      seen.add(it.id);
      out.push(it);
    }
  }
  return out;
}
