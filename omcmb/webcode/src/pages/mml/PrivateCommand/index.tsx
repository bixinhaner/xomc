import { useState, useMemo, useCallback } from 'react';
import { App, Button, Modal, Space, Tag } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import type { MMLCustomCommand } from '@core/types/mml';
import {
  useMMLTemplates,
  useCreateMMLTemplate,
  useUpdateMMLTemplate,
  useDeleteMMLTemplate,
} from '@core/hooks/api/useMML';
import { useT } from '@/hooks/useT';
import AddTemplateModal from '../Console/components/AddTemplateModal';

// T-0090-d：MML 私有命令独立列表页
//   - 后端 GET /mml/templates?command_scope=private 由 T-0090-c RBAC 服务端
//     自动过滤（creator self-fallback OR group-share）
//   - 复用 AddTemplateModal (scope='private')，间接使用 T-0090-a 抽出的
//     CommandCodeTextarea + OperationTypeWithModify 公共组件 → R-NEW-3
//     mitigation 第二步闭环
//   - 行为与公有命令页面（后续 sub-task 可镜像新建 PublicCommand 页）一致

function formatTime(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

export default function PrivateCommand() {
  const t = useT();
  const { message } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [commandCodeFilter, setCommandCodeFilter] = useState('');
  const [addModalOpen, setAddModalOpen] = useState(false);

  const { data, isLoading, refetch } = useMMLTemplates({
    templateScope: 'private',
    commandCode: commandCodeFilter || undefined,
    page,
    pageSize,
  });
  const deleteMutation = useDeleteMMLTemplate();

  // create+update hooks 由 AddTemplateModal 内部消费，本页面不直接使用；保留
  // 类型 import 让未来扩 inline edit modal 时不必再翻 hooks 文件。
  void useCreateMMLTemplate;
  void useUpdateMMLTemplate;

  const items = useMemo(() => data?.items ?? [], [data]);

  const handleDelete = useCallback(
    (record: MMLCustomCommand) => {
      Modal.confirm({
        title: t('common.confirmDelete'),
        content: t('mml.confirmDeleteCustomCommand', { name: record.commandName }),
        okText: t('common.delete'),
        okButtonProps: { danger: true },
        cancelText: t('common.cancel'),
        onOk: () =>
          new Promise<void>((resolve, reject) => {
            deleteMutation.mutate(record.id, {
              onSuccess: () => {
                void message.success(t('common.deleteSuccess'));
                resolve();
              },
              onError: (err) => {
                void message.error(
                  err instanceof Error ? err.message : String(err ?? 'Unknown'),
                );
                reject(err);
              },
            });
          }),
      });
    },
    [deleteMutation, message, t],
  );

  const columns: DataTableColumn<MMLCustomCommand>[] = useMemo(
    () => [
      {
        key: 'operation',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 120,
        fixed: 'right',
        render: (_, record) => (
          <Space size={4}>
            <Button type="link" size="small" danger onClick={() => handleDelete(record)}>
              {t('common.delete')}
            </Button>
          </Space>
        ),
      },
      { key: 'commandName', title: t('mml.console.commandName'), dataIndex: 'commandName', ellipsis: true },
      {
        key: 'commandCode',
        title: t('mml.console.commandCode'),
        dataIndex: 'commandCode',
        width: 240,
        ellipsis: true,
        render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</span>,
      },
      {
        key: 'operationType',
        title: t('mml.console.operationType'),
        dataIndex: 'operationType',
        width: 100,
        render: (v: string) => <Tag color={v === 'MOD' ? 'orange' : 'blue'}>{v}</Tag>,
      },
      { key: 'description', title: t('common.description'), dataIndex: 'description', ellipsis: true, render: (v: string) => v || '-' },
      { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
      { key: 'updatedAt', title: t('mml.updateTime'), dataIndex: 'updatedAt', width: 160, render: formatTime },
    ],
    [t, handleDelete],
  );

  const filterFields: FilterField[] = useMemo(
    () => [
      {
        name: 'commandCode',
        label: t('mml.console.commandCode'),
        type: 'input',
        placeholder: t('mml.console.commandCode'),
      },
    ],
    [t],
  );

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setCommandCodeFilter(typeof values.commandCode === 'string' ? values.commandCode : '');
    setPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setCommandCodeFilter('');
    setPage(1);
  }, []);

  return (
    <ListPageLayout
      title={t('mml.privateCommand.pageTitle')}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
            {t('common.refresh')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddModalOpen(true)}>
            {t('mml.console.addPrivateTemplate')}
          </Button>
        </Space>
      }
    >
      <FilterBar
        filterId="mml-private-command"
        fields={filterFields}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <DataTable<MMLCustomCommand>
        tableId="mml-private-commands"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1000 }}
      />

      <AddTemplateModal
        open={addModalOpen}
        scope="private"
        onClose={() => setAddModalOpen(false)}
        onSuccess={() => void refetch()}
      />
    </ListPageLayout>
  );
}
