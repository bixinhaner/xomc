import http from '../http';
import { useMock } from '../apiSwitch';
import type {
  AttentionAction,
  AttentionItem,
  AttentionKind,
  AttentionPage,
  AttentionSection,
  AttentionSectionKey,
  AttentionSectionStatus,
  AttentionSummary,
  AttentionTarget,
} from '../../types/attention';

interface BackendAttentionTarget {
  type: string;
  id?: string;
  name?: string;
  serial_number?: string;
}

interface BackendAttentionItem {
  id: string;
  kind: AttentionKind;
  source: string;
  source_id: string;
  title: string;
  summary: string;
  severity?: string;
  priority: string;
  risk?: string;
  target?: BackendAttentionTarget;
  occurred_at?: string;
  created_at?: string;
  updated_at?: string;
  detail_route: string;
  allowed_actions?: AttentionAction[];
}

interface BackendAttentionSection {
  status: AttentionSectionStatus;
  total: number;
  items?: BackendAttentionItem[];
}

interface BackendAttentionSummary {
  abnormalities: BackendAttentionSection;
  todos: BackendAttentionSection;
  generated_at: string;
}

interface BackendAttentionPage extends BackendAttentionSection {
  page: number;
  page_size: number;
}

function mapTarget(target: BackendAttentionTarget | undefined): AttentionTarget | undefined {
  return target ? {
    type: target.type,
    id: target.id,
    name: target.name,
    serialNumber: target.serial_number,
  } : undefined;
}

function mapItem(item: BackendAttentionItem): AttentionItem {
  return {
    id: item.id,
    kind: item.kind,
    source: item.source,
    sourceId: item.source_id,
    title: item.title,
    summary: item.summary,
    severity: item.severity,
    priority: item.priority,
    risk: item.risk,
    target: mapTarget(item.target),
    occurredAt: item.occurred_at,
    createdAt: item.created_at,
    updatedAt: item.updated_at,
    detailRoute: item.detail_route,
    allowedActions: item.allowed_actions ?? [],
  };
}

function mapSection(section: BackendAttentionSection): AttentionSection {
  return {
    status: section.status,
    total: section.total,
    items: (section.items ?? []).map(mapItem),
  };
}

export const attentionApi = {
  async getSummary(limit = 1): Promise<AttentionSummary> {
    if (useMock) {
      const finding: AttentionItem = {
        id: 'abnormal:agent_finding:mock-finding-task-failure', kind: 'agent_finding', source: 'agent',
        sourceId: 'mock-finding-task-failure', title: '任务失败主要由设备离线导致',
        summary: '任务执行窗口内设备 SN001 处于离线状态，建议先恢复设备连通性。',
        severity: 'high', priority: 'high', target: { type: 'device', id: 'device-456', name: 'SN001' },
        createdAt: new Date().toISOString(), detailRoute: '/dashboard?agentFinding=mock-finding-task-failure',
        allowedActions: ['view_agent_finding'],
      };
      return {
        abnormalities: { status: 'ok', total: 1, items: limit > 0 ? [finding] : [] },
        todos: { status: 'ok', total: 0, items: [] }, generatedAt: new Date().toISOString(),
      };
    }
    const { data } = await http.get<BackendAttentionSummary>('/dashboard/attention', {
      params: { abnormal_limit: limit, todo_limit: limit },
    });
    return {
      abnormalities: mapSection(data.abnormalities),
      todos: mapSection(data.todos),
      generatedAt: data.generated_at,
    };
  },

  async getPage(section: AttentionSectionKey, page: number, pageSize: number): Promise<AttentionPage> {
    if (useMock) {
      const summary = await this.getSummary(pageSize);
      return { ...summary[section], page, pageSize };
    }
    const { data } = await http.get<BackendAttentionPage>(`/dashboard/attention/${section}`, {
      params: { page, page_size: pageSize },
    });
    return {
      ...mapSection(data),
      page: data.page,
      pageSize: data.page_size,
    };
  },
};
