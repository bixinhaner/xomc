import { useState, useMemo, useCallback } from 'react';
import {
  Button,
  Modal,
  Space,
  Tag,
  message,
} from 'antd';
import {
  DownloadOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import type { MMLScript } from '@/types/mml';
import { useMMLScripts, useDeleteMMLScripts } from '@/hooks/api/useMML';
import { useDictionary } from '@/hooks/api/useSystem';
import { useT } from '@/hooks/useT';

function formatTime(iso?: string): string {
  if (!iso) return '';
  const d = iso ? new Date(iso) : null;
  if (!d || isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

export default function ScriptLibrary() {
  const t = useT();
  const [scriptPage, setScriptPage] = useState(1);
  const [scriptPageSize, setScriptPageSize] = useState(50);
  const [scriptSearch, setScriptSearch] = useState('');
  const [scriptViewVisible, setScriptViewVisible] = useState(false);
  const [viewingScript, setViewingScript] = useState<MMLScript | null>(null);

  const { data: productTypeDict } = useDictionary('product_type');
  const productTypeOptions = useMemo(() => {
    const details = productTypeDict?.sysDictionaryDetails;
    return details?.length ? details.map((d) => ({ label: d.label, value: d.value })) : [];
  }, [productTypeDict]);

  const { data: scriptsData, isLoading: scriptsLoading, refetch: refetchScripts } = useMMLScripts({
    page: scriptPage,
    pageSize: scriptPageSize,
    search: scriptSearch.trim() || undefined,
  });
  const deleteScriptsMutation = useDeleteMMLScripts();

  const scripts = useMemo(() => scriptsData?.items ?? [], [scriptsData]);

  const handleDeleteScripts = useCallback((record: MMLScript) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: t('mml.confirmDeleteScript', { name: record.scriptName }),
      okText: t('common.delete'),
      okButtonProps: { danger: true },
      cancelText: t('common.cancel'),
      onOk: () =>
        new Promise<void>((resolve, reject) => {
          deleteScriptsMutation.mutate([record.id], {
            onSuccess: () => {
              void message.success(t('common.deleteSuccess'));
              resolve();
            },
            onError: (err) => {
              void message.error(
                err instanceof Error ? err.message : String(err ?? 'Unknown')
              );
              reject(err);
            },
          });
        }),
    });
  }, [deleteScriptsMutation, t]);

  const scriptColumns: DataTableColumn<MMLScript>[] = useMemo(() => [
    {
      key: 'operation', title: t('table.operation'), dataIndex: 'id', width: 120, fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => { setViewingScript(record); setScriptViewVisible(true); }}>{t('mml.info')}</Button>
          <Button type="link" size="small" danger onClick={() => handleDeleteScripts(record)}>{t('common.delete')}</Button>
        </Space>
      ),
    },
    { key: 'scriptName', title: t('mml.scriptName'), dataIndex: 'scriptName', ellipsis: true },
    { key: 'description', title: t('mml.description'), dataIndex: 'description', ellipsis: true, render: (v: string) => v || '-' },
    { key: 'deviceType', title: t('mml.deviceType'), dataIndex: 'deviceType', width: 120, render: (v: string) => v || '-' },
    { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
    { key: 'tags', title: t('mml.tags'), dataIndex: 'tags', width: 180, render: (tags: string[]) => tags?.length ? tags.map((tag) => <Tag key={tag}>{tag}</Tag>) : '-' },
    { key: 'updatedAt', title: t('mml.updateTime'), dataIndex: 'updateTime', width: 160, render: (val: string) => formatTime(val) },
  ], [t, handleDeleteScripts]);

  const scriptFilterFields: FilterField[] = useMemo(() => [
    { name: 'scriptName', label: t('mml.scriptName'), type: 'input', placeholder: t('mml.scriptName') },
    { name: 'deviceType', label: t('mml.deviceType'), type: 'select', placeholder: t('mml.deviceType'), options: [
      { label: t('common.all'), value: 'all' },
      ...productTypeOptions,
    ] },
    { name: 'creator', label: t('mml.creator'), type: 'input', placeholder: t('mml.creator') },
  ], [t, productTypeOptions]);

  const handleScriptSearch = useCallback((values: Record<string, unknown>) => {
    setScriptPage(1);
    const keyword = (values.scriptName as string) || '';
    setScriptSearch(keyword);
  }, []);

  const handleScriptReset = useCallback(() => {
    setScriptSearch('');
    setScriptPage(1);
  }, []);

  return (
    <ListPageLayout
      title={t('mml.scriptLibrary')}
      extra={
        <Button icon={<DownloadOutlined />} onClick={() => void refetchScripts()}>{t('common.refresh')}</Button>
      }
    >
      <FilterBar filterId="mml-script-library" fields={scriptFilterFields} onSearch={handleScriptSearch} onReset={handleScriptReset} />
      <DataTable<MMLScript>
        tableId="mml-scripts" columns={scriptColumns} dataSource={scripts} loading={scriptsLoading} rowKey="id"
        total={scriptsData?.total ?? 0} currentPage={scriptPage} pageSize={scriptPageSize}
        onPageChange={(p, s) => { setScriptPage(p); setScriptPageSize(s); }} onRefresh={() => void refetchScripts()} scroll={{ x: 1000 }}
      />

      <Modal title={t('mml.scriptDetail')} open={scriptViewVisible} onCancel={() => setScriptViewVisible(false)} footer={null} width={600}>
        {viewingScript && (
          <div style={{ padding: '16px 0' }}>
            <p><strong>{t('mml.scriptNameLabel')}</strong>{viewingScript.scriptName}</p>
            <p><strong>{t('mml.description')}</strong>{viewingScript.description || '-'}</p>
            <p><strong>{t('mml.deviceType')}</strong>{viewingScript.deviceType || '-'}</p>
            <p><strong>{t('mml.creatorLabel')}</strong>{viewingScript.creator}</p>
            <p><strong>{t('mml.updateTime')}</strong>{formatTime(viewingScript.updateTime)}</p>
            <div style={{ marginTop: 12 }}>
              <strong>{t('mml.scriptContent')}</strong>
              <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, maxHeight: 300, overflow: 'auto', fontSize: 13, fontFamily: 'monospace' }}>
                {viewingScript.content}
              </pre>
            </div>
          </div>
        )}
      </Modal>
    </ListPageLayout>
  );
}
