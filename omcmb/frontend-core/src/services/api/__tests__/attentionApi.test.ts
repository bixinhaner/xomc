import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }));

vi.mock('../../http', () => ({
  default: { get: getMock },
}));

import { attentionApi } from '../attentionApi';
import { attentionKeys } from '../../../hooks/api/useAttention';

beforeEach(() => getMock.mockReset());

const backendItem = {
  id: 'todo:device_access_review:1',
  kind: 'device_access_review',
  source: 'device_access',
  source_id: 'candidate-1',
  title: 'SN-1',
  summary: 'CMCC',
  priority: 'normal',
  target: { type: 'device_candidate', id: 'candidate-1', serial_number: 'SN-1' },
  created_at: '2026-08-24T09:00:00Z',
  detail_route: '/device/access-control?tab=candidates&candidateId=candidate-1',
  allowed_actions: ['review_device_candidate'],
};

describe('attentionApi', () => {
  it('maps summary fields and requests only the compact limit', async () => {
    getMock.mockResolvedValue({
      data: {
        abnormalities: { status: 'ok', total: 0, items: [] },
        todos: { status: 'partial', total: 1, items: [backendItem] },
        generated_at: '2026-08-24T09:01:00Z',
      },
    });

    const result = await attentionApi.getSummary();

    expect(getMock).toHaveBeenCalledWith('/dashboard/attention', { params: { abnormal_limit: 1, todo_limit: 1 } });
    expect(result.todos.items[0]).toMatchObject({
      sourceId: 'candidate-1', detailRoute: '/device/access-control?tab=candidates&candidateId=candidate-1',
      target: { serialNumber: 'SN-1' }, allowedActions: ['review_device_candidate'],
    });
  });

  it('uses the section endpoint and snake_case pagination parameters', async () => {
    getMock.mockResolvedValue({
      data: { status: 'ok', total: 21, items: [backendItem], page: 2, page_size: 20 },
    });

    const result = await attentionApi.getPage('todos', 2, 20);

    expect(getMock).toHaveBeenCalledWith('/dashboard/attention/todos', {
      params: { page: 2, page_size: 20 },
    });
    expect(result).toMatchObject({ page: 2, pageSize: 20, total: 21 });
  });

  it('keeps summary and drawer query keys separate', () => {
    expect(attentionKeys.summary()).toEqual(['dashboard', 'attention', 'summary']);
    expect(attentionKeys.drawer('todos', 2, 20)).toEqual(['dashboard', 'attention', 'drawer', 'todos', 2, 20]);
  });
});
