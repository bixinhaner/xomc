/**
 * PM 指标库新建保存回归测试（issue #63）。
 *
 * HTTP 非 secure context 下浏览器可能没有 crypto.randomUUID；新建指标/分组保存路径
 * 不能依赖前端生成 ID，应直接调用 create mutation，由后端创建接口生成真实 ID。
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { message } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { createIndicatorMock, createGroupMock } = vi.hoisted(() => ({
  createIndicatorMock: vi.fn(),
  createGroupMock: vi.fn(),
}));

vi.mock('@/hooks/useT', () => ({
  useT:
    () =>
    (id: string) =>
      id,
}));

vi.mock('@core/hooks/api/useIndicatorsLibrary', () => ({
  useCreateIndicator: () => ({ mutateAsync: createIndicatorMock, isPending: false }),
  useUpdateIndicator: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useIndicatorGroups: () => ({
    data: { items: [{ id: 'gsm-call', name: '呼叫类', deviceType: 'GSM' }] },
    isLoading: false,
  }),
  useCreateGroup: () => ({ mutateAsync: createGroupMock, isPending: false }),
  useUpdateGroup: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteGroup: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock('../GroupTreeSelect', () => ({
  default: ({ value, onChange }: { value?: string; onChange?: (next: string) => void }) => (
    <input
      aria-label="group-tree-select"
      value={value ?? ''}
      onChange={(event) => onChange?.(event.target.value)}
    />
  ),
}));

vi.mock('../PlatformFormulasSection', () => ({
  default: () => <div data-testid="platform-formulas-section" />,
}));

import IndicatorFormModal from '../IndicatorFormModal';
import GroupsManageModal from '../GroupsManageModal';

function hideRandomUUID(): void {
  const cryptoWithoutRandomUUID = { ...globalThis.crypto, randomUUID: undefined };
  Object.defineProperty(globalThis, 'crypto', {
    configurable: true,
    value: cryptoWithoutRandomUUID,
  });
}

beforeEach(() => {
  createIndicatorMock.mockReset();
  createGroupMock.mockReset();
  vi.spyOn(message, 'error').mockImplementation(vi.fn());
  createIndicatorMock.mockResolvedValue({
    id: 'K900000001',
    name: '接入成功率',
    cnName: '接入成功率',
    enName: 'Attach success rate',
    groupId: 'gsm-call',
    deviceType: 'GSM',
  });
  createGroupMock.mockResolvedValue({
    id: 'backend-group-id',
    name: '自定义分组',
    parentId: '0',
    deviceType: 'GSM',
  });
  hideRandomUUID();
});

describe('PM 指标库 create 保存路径', () => {
  it('新建指标在 randomUUID 不可用时仍调用 create mutation 且不传 id', async () => {
    render(
      <IndicatorFormModal
        open
        onClose={vi.fn()}
        deviceType="GSM"
        operatorCode="cmcc"
        indicator={null}
      />,
    );

    fireEvent.change(screen.getByLabelText('product.kpi.indicator.cnNameLabel'), {
      target: { value: '接入成功率' },
    });
    fireEvent.change(screen.getByLabelText('product.kpi.indicator.enNameLabel'), {
      target: { value: 'Attach success rate' },
    });
    fireEvent.change(screen.getByLabelText('group-tree-select'), {
      target: { value: 'gsm-call' },
    });
    fireEvent.click(screen.getByText('product.kpi.indicator.typeCounter'));
    fireEvent.click(screen.getByText('common.save'));

    await waitFor(() => expect(createIndicatorMock).toHaveBeenCalledOnce());
    const payload = createIndicatorMock.mock.calls[0][0];
    expect(payload.input).not.toHaveProperty('id');
    expect(payload.input).toMatchObject({
      name: '接入成功率',
      cnName: '接入成功率',
      enName: 'Attach success rate',
      groupId: 'gsm-call',
      isCounter: '1',
      operatorCode: 'cmcc',
    });
  });

  it('新建指标失败时展示错误且不依赖 randomUUID', async () => {
    createIndicatorMock.mockRejectedValueOnce(new Error('create indicator failed'));
    render(
      <IndicatorFormModal
        open
        onClose={vi.fn()}
        deviceType="GSM"
        operatorCode="cmcc"
        indicator={null}
      />,
    );

    fireEvent.change(screen.getByLabelText('product.kpi.indicator.cnNameLabel'), {
      target: { value: '接入成功率' },
    });
    fireEvent.change(screen.getByLabelText('product.kpi.indicator.enNameLabel'), {
      target: { value: 'Attach success rate' },
    });
    fireEvent.change(screen.getByLabelText('group-tree-select'), {
      target: { value: 'gsm-call' },
    });
    fireEvent.click(screen.getByText('product.kpi.indicator.typeCounter'));
    fireEvent.click(screen.getByText('common.save'));

    await waitFor(() => expect(createIndicatorMock).toHaveBeenCalledOnce());
    expect(createIndicatorMock.mock.calls[0][0].input).not.toHaveProperty('id');
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('create indicator failed'));
  });

  it('新建分组在 randomUUID 不可用时仍调用 create mutation 且不传 id', async () => {
    render(
      <GroupsManageModal
        open
        onClose={vi.fn()}
        deviceType="GSM"
        operatorCode="cmcc"
      />,
    );

    fireEvent.click(screen.getAllByText('product.kpi.group.createTitle')[0]);
    fireEvent.change(screen.getByLabelText('product.kpi.group.nameLabel'), {
      target: { value: '自定义分组' },
    });
    fireEvent.click(screen.getByText('common.save'));

    await waitFor(() => expect(createGroupMock).toHaveBeenCalledOnce());
    const payload = createGroupMock.mock.calls[0][0];
    expect(payload.input).not.toHaveProperty('id');
    expect(payload.input).toMatchObject({
      name: '自定义分组',
      parentId: '0',
      operatorCode: 'cmcc',
    });
  });

  it('新建分组失败时展示错误且不依赖 randomUUID', async () => {
    createGroupMock.mockRejectedValueOnce(new Error('create group failed'));
    render(
      <GroupsManageModal
        open
        onClose={vi.fn()}
        deviceType="GSM"
        operatorCode="cmcc"
      />,
    );

    fireEvent.click(screen.getAllByText('product.kpi.group.createTitle')[0]);
    fireEvent.change(screen.getByLabelText('product.kpi.group.nameLabel'), {
      target: { value: '自定义分组' },
    });
    fireEvent.click(screen.getByText('common.save'));

    await waitFor(() => expect(createGroupMock).toHaveBeenCalledOnce());
    expect(createGroupMock.mock.calls[0][0].input).not.toHaveProperty('id');
    await waitFor(() => expect(message.error).toHaveBeenCalledWith('create group failed'));
  });
});
