/**
 * 标准参数树本地分页的「下一页码」决策（issue #190）。
 *
 * 前端一次性全量加载 + 本地分页。antd Table 的 pagination.onChange 会回传
 * (目标页码, 每页条数)。缺陷在于：改每页条数时若沿用回传页码，会停留在原页码
 * 导致切片不刷新。规则：每页条数变化 → 回第 1 页；否则按目标页码翻页。
 */
export function nextPageOnPaginationChange(
  targetPage: number,
  nextPageSize: number,
  currentPageSize: number,
): { page: number; pageSize: number; sizeChanged: boolean } {
  const sizeChanged = nextPageSize !== currentPageSize;
  return {
    page: sizeChanged ? 1 : targetPage,
    pageSize: nextPageSize,
    sizeChanged,
  };
}
