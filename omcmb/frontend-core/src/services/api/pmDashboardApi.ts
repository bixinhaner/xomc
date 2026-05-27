/**
 * T-0164-P6 / G6 PM 性能查看仪表盘 REST API 客户端。
 *
 * 后端路由（cmd/app/provider/router.go 注册到 permGroup("pm")）：
 *   GET    /api/v1/pm/dashboards                列表（owner + shared）
 *   POST   /api/v1/pm/dashboards                创建
 *   GET    /api/v1/pm/dashboards/:id            详情（含 panels）
 *   PUT    /api/v1/pm/dashboards/:id            更新（仅 owner）
 *   DELETE /api/v1/pm/dashboards/:id            删除（仅 owner）
 *   POST   /api/v1/pm/dashboards/:id/fork       派生
 *   POST   /api/v1/pm/dashboards/:id/share      分享
 *   DELETE /api/v1/pm/dashboards/:id/share/:u   取消分享
 *   POST   /api/v1/pm/dashboards/:id/panels     创建 panel
 *   PUT    /api/v1/pm/dashboards/:id/panels/:p  更新 panel
 *   DELETE /api/v1/pm/dashboards/:id/panels/:p  删除 panel
 *   GET    /api/v1/pm/user-preferences/dashboard
 *   PUT    /api/v1/pm/user-preferences/dashboard
 */

import http from '../http';
import type {
  Dashboard,
  Panel,
  UserDashboardPreferences,
  UpsertUserPreferencesInput,
  CreateDashboardInput,
  UpdateDashboardInput,
  CreatePanelInput,
  ForkInput,
  ShareInput,
  BackendDashboard,
  BackendPanel,
  BackendUserPreferences,
  PanelType,
  Granularity,
  Dimension,
  CompareMode,
  Technology,
  AggregatedQueryParams,
  AggregatedRow,
  BackendAggregatedRow,
} from '../../types/pmDashboard';
import {
  mapBackendDashboard,
  mapBackendPanel,
  mapBackendPreferences,
  mapBackendAggregatedRow,
} from '../../types/pmDashboard';

interface ListResponse {
  items: BackendDashboard[];
  total: number;
}

interface DetailResponse {
  dashboard: BackendDashboard;
  panels: BackendPanel[];
}

interface DashboardDetail {
  dashboard: Dashboard;
  panels: Panel[];
}

// http 拦截器把 camelCase 自动转 snake_case；这里手工 body 构造 snake_case 形式更易调试 + 对齐后端 DTO。
function dashboardCreateBody(input: CreateDashboardInput) {
  return {
    name: input.name,
    description: input.description ?? '',
    technology: input.technology,
    layout: input.layout ?? { panels: [] },
  };
}

function dashboardUpdateBody(input: UpdateDashboardInput) {
  const body: Record<string, unknown> = {};
  if (input.name !== undefined) body.name = input.name;
  if (input.description !== undefined) body.description = input.description;
  if (input.technology !== undefined) body.technology = input.technology;
  if (input.layout !== undefined) body.layout = input.layout;
  return body;
}

function panelBody(input: CreatePanelInput) {
  return {
    panel_type: input.panelType,
    title: input.title,
    metric_paths: input.metricPaths,
    granularities: input.granularities,
    dimension: input.dimension,
    device_sns: input.deviceSns,
    device_group_ids: input.deviceGroupIds,
    time_range: input.timeRange,
    compare_mode: input.compareMode,
    adhoc_task_id: input.adhocTaskId,
    config: input.config ?? {},
  };
}

// ── 真实 API 服务对象 ────────────────────────────────────────────────

export const pmDashboardApi = {
  // Dashboard CRUD
  async list(): Promise<Dashboard[]> {
    const { data } = await http.get<ListResponse>('/pm/dashboards');
    return (data.items ?? []).map(mapBackendDashboard);
  },

  async get(id: string): Promise<DashboardDetail> {
    const { data } = await http.get<DetailResponse>(`/pm/dashboards/${id}`);
    return {
      dashboard: mapBackendDashboard(data.dashboard),
      panels: (data.panels ?? []).map(mapBackendPanel),
    };
  },

  async create(input: CreateDashboardInput): Promise<Dashboard> {
    const { data } = await http.post<BackendDashboard>('/pm/dashboards', dashboardCreateBody(input));
    return mapBackendDashboard(data);
  },

  async update(id: string, input: UpdateDashboardInput): Promise<void> {
    await http.put(`/pm/dashboards/${id}`, dashboardUpdateBody(input));
  },

  async remove(id: string): Promise<void> {
    await http.delete(`/pm/dashboards/${id}`);
  },

  async fork(input: ForkInput): Promise<Dashboard> {
    const { data } = await http.post<BackendDashboard>(`/pm/dashboards/${input.sourceId}/fork`, {
      new_name: input.newName,
    });
    return mapBackendDashboard(data);
  },

  async share(input: ShareInput): Promise<void> {
    await http.post(`/pm/dashboards/${input.dashboardId}/share`, { user_ids: input.userIds });
  },

  async unshare(dashboardId: string, userId: string): Promise<void> {
    await http.delete(`/pm/dashboards/${dashboardId}/share/${userId}`);
  },

  // Panel CRUD
  async createPanel(input: CreatePanelInput): Promise<Panel> {
    const { data } = await http.post<BackendPanel>(
      `/pm/dashboards/${input.dashboardId}/panels`,
      panelBody(input),
    );
    return mapBackendPanel(data);
  },

  async updatePanel(input: CreatePanelInput & { panelId: string }): Promise<void> {
    await http.put(
      `/pm/dashboards/${input.dashboardId}/panels/${input.panelId}`,
      panelBody(input),
    );
  },

  async deletePanel(dashboardId: string, panelId: string): Promise<void> {
    await http.delete(`/pm/dashboards/${dashboardId}/panels/${panelId}`);
  },

  // UserPreferences（T-0164 收尾 G6-Gap-3：按制式分键持久化）
  async getUserPrefs(technology: Technology = 'lte'): Promise<UserDashboardPreferences> {
    const { data } = await http.get<BackendUserPreferences>('/pm/user-preferences/dashboard', {
      params: { technology },
    });
    return mapBackendPreferences(data);
  },

  async setUserPrefs(input: UpsertUserPreferencesInput): Promise<void> {
    await http.put('/pm/user-preferences/dashboard', {
      technology: input.technology,
      kpi_card_layout: input.kpiCardLayout ?? {},
      current_dashboard_id: input.currentDashboardId,
      shared_filters: input.sharedFilters ?? {},
    });
  },

  // G6 Phase 4: 走 G5 aggregator → 按粒度路由聚合表（hourly+ 直查物化表，15min 退回 pm_metrics 原表）。
  async queryAggregated(params: AggregatedQueryParams): Promise<AggregatedRow[]> {
    // metricPaths 后端期望 comma-separated；其它 snake_case 参数手工拼，避免被 http 拦截器误转。
    const qp: Record<string, string | number | undefined> = {
      granularity: params.granularity,
      dimension: params.dimension,
      device_oui: params.deviceOui,
      device_sn: params.deviceSn,
      device_group_id: params.deviceGroupId,
      metric_paths: params.metricPaths && params.metricPaths.length > 0
        ? params.metricPaths.join(',')
        : undefined,
      metric_type: params.metricType,
      start_time: params.startTime,
      end_time: params.endTime,
      limit: params.limit,
      offset: params.offset,
      fill_empty: params.fillEmpty ? 'true' : undefined,
    };
    const { data } = await http.get<{ items: BackendAggregatedRow[] | null; total: number }>(
      '/pm/metrics/aggregated',
      { params: qp },
    );
    return (data.items ?? []).map(mapBackendAggregatedRow);
  },
};

// ── Mock 服务（VITE_USE_MOCK=true 启用）────────────────────────────────

import { mockDashboards, mockPanels, mockUserPreferences } from '../../mock/data/pmDashboard';

let mockDashboardsState = [...mockDashboards];
let mockPanelsState = [...mockPanels];
let mockUserPrefsState = { ...mockUserPreferences };

function genId(prefix: string) {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

export const pmDashboardMock: typeof pmDashboardApi = {
  async list() {
    return [...mockDashboardsState];
  },

  async get(id) {
    const dashboard = mockDashboardsState.find((d) => d.id === id);
    if (!dashboard) throw new Error('dashboard not found');
    const panels = mockPanelsState.filter((p) => p.dashboardId === id);
    return { dashboard, panels };
  },

  async create(input) {
    const d: Dashboard = {
      id: genId('dash'),
      name: input.name,
      description: input.description,
      ownerId: 'mock-user-owner-0001',
      sharedWith: [],
      technology: input.technology,
      layout: input.layout ?? { panels: [] },
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    mockDashboardsState = [d, ...mockDashboardsState];
    return d;
  },

  async update(id, input) {
    mockDashboardsState = mockDashboardsState.map((d) =>
      d.id === id
        ? {
            ...d,
            name: input.name ?? d.name,
            description: input.description ?? d.description,
            technology: input.technology ?? d.technology,
            layout: input.layout ?? d.layout,
            updatedAt: new Date().toISOString(),
          }
        : d,
    );
  },

  async remove(id) {
    mockDashboardsState = mockDashboardsState.filter((d) => d.id !== id);
    mockPanelsState = mockPanelsState.filter((p) => p.dashboardId !== id);
  },

  async fork(input) {
    const src = mockDashboardsState.find((d) => d.id === input.sourceId);
    if (!src) throw new Error('source not found');
    const newDash: Dashboard = {
      ...src,
      id: genId('dash'),
      name: input.newName,
      parentDashboardId: src.id,
      sharedWith: [],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    mockDashboardsState = [newDash, ...mockDashboardsState];
    // 复制 panels
    const srcPanels = mockPanelsState.filter((p) => p.dashboardId === src.id);
    const newPanels = srcPanels.map((p) => ({ ...p, id: genId('panel'), dashboardId: newDash.id }));
    mockPanelsState = [...newPanels, ...mockPanelsState];
    return newDash;
  },

  async share(input) {
    mockDashboardsState = mockDashboardsState.map((d) =>
      d.id === input.dashboardId
        ? { ...d, sharedWith: Array.from(new Set([...d.sharedWith, ...input.userIds])) }
        : d,
    );
  },

  async unshare(dashboardId, userId) {
    mockDashboardsState = mockDashboardsState.map((d) =>
      d.id === dashboardId ? { ...d, sharedWith: d.sharedWith.filter((u) => u !== userId) } : d,
    );
  },

  async createPanel(input) {
    const p: Panel = {
      id: genId('panel'),
      dashboardId: input.dashboardId,
      panelType: input.panelType as PanelType,
      title: input.title,
      metricPaths: input.metricPaths,
      granularities: input.granularities as Granularity[],
      dimension: input.dimension as Dimension,
      deviceSns: input.deviceSns,
      deviceGroupIds: input.deviceGroupIds,
      timeRange: input.timeRange,
      compareMode: input.compareMode as CompareMode | undefined,
      adhocTaskId: input.adhocTaskId,
      config: input.config ?? {},
    };
    mockPanelsState = [...mockPanelsState, p];
    return p;
  },

  async updatePanel(input) {
    mockPanelsState = mockPanelsState.map((p) => (p.id === input.panelId ? { ...p, ...input } : p));
  },

  async deletePanel(_dashboardId, panelId) {
    mockPanelsState = mockPanelsState.filter((p) => p.id !== panelId);
  },

  async getUserPrefs(technology: Technology = 'lte') {
    // mock 一份按 technology 切的实现：同一 user 不同 technology 返不同 prefs
    return { ...mockUserPrefsState, technology };
  },

  async setUserPrefs(input) {
    mockUserPrefsState = {
      ...mockUserPrefsState,
      technology: input.technology,
      kpiCardLayout: input.kpiCardLayout ?? {},
      currentDashboardId: input.currentDashboardId,
      sharedFilters: input.sharedFilters ?? {},
    };
  },

  // Mock：每个 metric 拉出 deterministic 序列。粒度按入参 8 桶。
  async queryAggregated(params): Promise<AggregatedRow[]> {
    const buckets = mockBuckets(params.granularity);
    const paths = params.metricPaths && params.metricPaths.length > 0
      ? params.metricPaths
      : ['MOCK.Counter'];
    const out: AggregatedRow[] = [];
    paths.forEach((path) => {
      buckets.forEach((iso, i) => {
        out.push({
          deviceOui: params.deviceOui ?? 'MOCK',
          deviceSn: params.deviceSn ?? 'MOCK-0001',
          deviceGroupId: undefined,
          metricPath: path,
          metricType: (params.metricType ?? 'counter') as 'counter' | 'kpi',
          metricValue: Math.round((90 + ((i * 3) % 11) + Math.sin(i) * 2) * 100) / 100,
          statisType: 'sum',
          granularity: params.granularity,
          time: iso,
          startTime: iso,
          endTime: iso,
          ingestTime: iso,
          objectLdn: null,
          extra: {},
        });
      });
    });
    return out;
  },
};

// ── Helpers ────────────────────────────────────────────────────────────

function mockBuckets(g: Granularity): string[] {
  const now = Date.now();
  const stepMs = g === '15min'
    ? 15 * 60_000
    : g === 'hourly'
    ? 3_600_000
    : g === 'daily'
    ? 86_400_000
    : g === 'weekly'
    ? 7 * 86_400_000
    : 30 * 86_400_000; // monthly approx
  return Array.from({ length: 8 }, (_, i) => new Date(now - (7 - i) * stepMs).toISOString());
}
