import { useState, useMemo, useCallback } from 'react';
import { Button, Input, Modal, Space, Tag, message } from 'antd';
import dayjs from 'dayjs';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

import type { MMLScript } from '@core/types/mml';
import { useMMLScripts, useDeleteMMLScripts } from '@core/hooks/api/useMML';

function formatTime(iso?: string | null): string {
  if (!iso) return '-';
  const d = dayjs(iso);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : '-';
}

// 脚本任务（mml/script）：脚本库列表，读 mml_scripts。
// 任务执行记录（mml_tasks）由独立页面 mml/task-records 承载。
export default function ScriptTask() {
  const t = useT();

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [search, setSearch] = useState('');

  const { data, isLoading, refetch } = useMMLScripts({
    page,
    pageSize,
    search: search.trim() || undefined,
  });
  const deleteScriptsMutation = useDeleteMMLScripts();

  const scripts = useMemo(() => data?.items ?? [], [data]);

  const [viewing, setViewing] = useState<MMLScript | null>(null);

  // 删除脚本：敏感操作，走 Modal.confirm 二次确认。
  const handleDelete = useCallback((script: MMLScript) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      okText: t('common.delete'),
      okButtonProps: { danger: true },
      cancelText: t('common.cancel'),
      onOk: () =>
        new Promise<void>((resolve, reject) => {
          deleteScriptsMutation.mutate([script.id], {
            onSuccess: () => {
              void message.success(t('common.deleteSuccess'));
              resolve();
            },
            onError: (err) => {
              void message.error(err instanceof Error ? err.message : 'Unknown');
              reject(err);
            },
          });
        }),
    });
  }, [deleteScriptsMutation, t]);

  const columns: DataTableColumn<MMLScript>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => setViewing(record)}>{t('mml.info')}</Button>
          <Button type="link" size="small" danger onClick={() => handleDelete(record)}>{t('common.delete')}</Button>
        </Space>
      ),
    },
    { key: 'scriptName', title: t('mml.scriptName'), dataIndex: 'scriptName', ellipsis: true },
    { key: 'description', title: t('mml.description'), dataIndex: 'description', ellipsis: true, render: (v: unknown) => (v as string) || '-' },
    { key: 'deviceType', title: t('mml.deviceType'), dataIndex: 'deviceType', width: 120, render: (v: unknown) => (v as string) || '-' },
    { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
    { key: 'tags', title: t('mml.tags'), dataIndex: 'tags', width: 180, render: (tags: unknown) => Array.isArray(tags) && tags.length ? (tags as string[]).map((tag) => <Tag key={tag}>{tag}</Tag>) : '-' },
    { key: 'updatedAt', title: t('mml.updateTime'), dataIndex: 'updateTime', width: 160, render: (val: unknown) => formatTime(val as string) },
  ], [t, handleDelete]);

  return (
    <ListPageLayout title={t('nav.mml.script')}>
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder={t('mml.scriptName')}
          allowClear
          style={{ width: 300 }}
          onSearch={(val) => { setSearch(val); setPage(1); }}
        />
      </div>
      <DataTable<MMLScript>
        tableId="mml-scripts"
        columns={columns}
        dataSource={scripts}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1000 }}
      />

      <Modal title={t('mml.scriptDetail')} open={Boolean(viewing)} onCancel={() => setViewing(null)} footer={null} width={600}>
        {viewing && (
          <div style={{ padding: '16px 0' }}>
            <p><strong>{t('mml.scriptNameLabel')}</strong>{viewing.scriptName}</p>
            <p><strong>{t('mml.description')}</strong>{viewing.description || '-'}</p>
            <p><strong>{t('mml.creatorLabel')}</strong>{viewing.creator}</p>
            <p><strong>{t('mml.updateTime')}</strong>{formatTime(viewing.updateTime)}</p>
            <div style={{ marginTop: 12 }}>
              <strong>{t('mml.scriptContent')}</strong>
              <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, maxHeight: 300, overflow: 'auto', fontSize: 13, fontFamily: 'monospace' }}>
                {viewing.content}
              </pre>
            </div>
          </div>
        )}
      </Modal>
    </ListPageLayout>
  );
}
