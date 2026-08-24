import http from '../http';
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
