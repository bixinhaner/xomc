export interface PageRequest {
  page: number;
  pageSize: number;
  sortField?: string;
  sortOrder?: 'ascend' | 'descend';
}

export interface PageResponse<T, S = unknown> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  /** 后端 ListResponse.stats —— 附加统计/元数据，由各 API 定义具体形状。 */
  stats?: S;
}
