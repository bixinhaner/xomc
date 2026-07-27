import { useMemo, useState } from 'react';
import type { Key } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Descriptions, Drawer, Dropdown, Empty, Form, Input, Modal, Space, Spin, Tooltip, Typography, message } from 'antd';
import type { MenuProps } from 'antd';
import { DeleteOutlined, DownloadOutlined, EditOutlined, EyeOutlined, MoreOutlined, PlayCircleOutlined, PlusOutlined, ReloadOutlined, UploadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import SearchInput from '@/components/SearchInput';
import { useT } from '@/hooks/useT';
import type { MMLScript } from '@core/types/mml';
import { useMMLScripts, useMMLScriptById, useUpdateMMLScript, useDeleteMMLScripts } from '@core/hooks/api/useMML';
import { formatSystemTime } from '@core/utils/systemTime';
import ScriptImportModal from './ScriptImportModal';
import ScriptExecutionDrawer from './ScriptExecutionDrawer';
import ScriptImportPreview from './ScriptImportPreview';

function formatTime(iso?: string | null): string {
  return formatSystemTime(iso);
}

function downloadScript(script: MMLScript) {
  const url = URL.createObjectURL(new Blob([script.content || ''], { type: 'text/plain;charset=utf-8' }));
  const anchor = document.createElement('a'); anchor.href = url; anchor.download = script.originalFilename || `${script.scriptName || 'mml-script'}.txt`; anchor.click(); URL.revokeObjectURL(url);
}

interface BasicForm { scriptName: string; description: string; }

export default function ScriptTask() {
  const t = useT();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [search, setSearch] = useState('');
  const [importOpen, setImportOpen] = useState(false);
  const [reimporting, setReimporting] = useState<MMLScript | null>(null);
  const [viewing, setViewing] = useState<MMLScript | null>(null);
  const [execScript, setExecScript] = useState<MMLScript | null>(null);
  const [editing, setEditing] = useState<MMLScript | null>(null);
  const [selectedScriptIds, setSelectedScriptIds] = useState<Key[]>([]);
  const [basicForm] = Form.useForm<BasicForm>();
  const { data, isLoading, refetch } = useMMLScripts({ page, pageSize, search: search.trim() || undefined });
  const { data: detail, isFetching } = useMMLScriptById(viewing?.id ?? '');
  const updateMutation = useUpdateMMLScript();
  const deleteMutation = useDeleteMMLScripts();
  const scripts = useMemo(() => data?.items ?? [], [data]);
  const detailScript = detail ?? viewing;

  const closeBasic = () => { setEditing(null); basicForm.resetFields(); };
  const saveBasic = async () => {
    if (!editing) return;
    const values = await basicForm.validateFields();
    updateMutation.mutate({ id: editing.id, data: { scriptName: values.scriptName.trim(), description: values.description ?? '' } }, { onSuccess: () => { void refetch(); closeBasic(); void message.success(t('common.saveSuccess')); } });
  };

  const confirmBatchDelete = () => {
    const ids = selectedScriptIds.map(String);
    if (ids.length === 0) return;
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: t('mml.confirmBatchDeleteScripts', { count: ids.length }),
      okText: t('common.delete'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: () => new Promise<void>((resolve, reject) => {
        deleteMutation.mutate(ids, {
          onSuccess: () => {
            setSelectedScriptIds([]);
            void refetch();
            void message.success(t('common.deleteSuccess'));
            resolve();
          },
          onError: (error) => {
            void message.error(t('common.deleteFailed'));
            reject(error);
          },
        });
      }),
    });
  };

  const columns: DataTableColumn<MMLScript>[] = [
    {
      key: 'operation',
      title: t('table.operation'),
      width: 120,
      render: (_, record) => {
        const openEdit = () => {
          setEditing(record);
          basicForm.setFieldsValue({ scriptName: record.scriptName, description: record.description });
        };
        const confirmDelete = () => Modal.confirm({
          title: t('common.confirmDelete'),
          content: t('mml.confirmDeleteScript', { name: record.scriptName }),
          okText: t('common.delete'),
          cancelText: t('common.cancel'),
          okButtonProps: { danger: true },
          onOk: () => new Promise<void>((resolve, reject) => deleteMutation.mutate([record.id], {
            onSuccess: () => {
              setSelectedScriptIds((prev) => prev.filter((id) => id !== record.id));
              void refetch();
              resolve();
            },
            onError: reject,
          })),
        });
        const items: MenuProps['items'] = [
          { key: 'view', label: t('mml.script.action.viewDetail'), icon: <EyeOutlined />, onClick: () => setViewing(record) },
          { key: 'reimport', label: t('mml.script.action.reimport'), icon: <UploadOutlined />, onClick: () => setReimporting(record) },
          { key: 'download', label: t('mml.script.action.downloadTxt'), icon: <DownloadOutlined />, onClick: () => downloadScript(record) },
          { key: 'edit', label: t('common.edit'), icon: <EditOutlined />, onClick: openEdit },
          { type: 'divider' },
          { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, onClick: confirmDelete },
        ];
        return <Space size={4}>
          <Tooltip title={t('mml.script.action.execute')}>
            <Button type="link" size="small" aria-label={t('mml.script.action.execute')} icon={<PlayCircleOutlined />} onClick={(event) => { event.stopPropagation(); setExecScript(record); }} />
          </Tooltip>
          <Dropdown menu={{ items }} trigger={['click']}>
            <Button type="text" size="small" aria-label={t('mml.script.action.more')} icon={<MoreOutlined />} onClick={(event) => event.stopPropagation()} />
          </Dropdown>
        </Space>;
      },
    },
    { key: 'scriptName', title: t('mml.scriptName'), dataIndex: 'scriptName', ellipsis: true },
    { key: 'description', title: t('mml.description'), dataIndex: 'description', ellipsis: true, render: (value) => String(value || '-') },
    { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
    { key: 'updatedAt', title: t('mml.updateTime'), dataIndex: 'updateTime', width: 170, render: (value) => formatTime(value as string) },
  ];

  return <ListPageLayout title={t('nav.mml.script')} extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setImportOpen(true)}>{t('mml.script.action.importTxt')}</Button>}>
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
      <SearchInput placeholder={t('mml.scriptName')} allowClear style={{ width: 300 }} onSearch={(value) => { setSearch(value); setPage(1); }} />
      <Space size={8} wrap>
        {selectedScriptIds.length > 0 ? (
          <Typography.Text type="secondary">
            {t('table.selected', { count: selectedScriptIds.length })}
          </Typography.Text>
        ) : null}
        <Button
          danger
          disabled={selectedScriptIds.length === 0}
          icon={<DeleteOutlined />}
          loading={deleteMutation.isPending}
          onClick={confirmBatchDelete}
        >
          {t('common.batchDelete')}
        </Button>
        <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
          {t('common.refresh')}
        </Button>
      </Space>
    </div>
    <DataTable<MMLScript>
      tableId="mml-scripts"
      columns={columns}
      dataSource={scripts}
      loading={isLoading}
      rowKey="id"
      selectable
      selectedRowKeys={selectedScriptIds}
      onSelectionChange={(keys) => setSelectedScriptIds(keys)}
      preserveSelectedRowKeys
      total={data?.total ?? 0}
      currentPage={page}
      pageSize={pageSize}
      onPageChange={(nextPage, nextSize) => { setPage(nextPage); setPageSize(nextSize); }}
      hideToolbar
      scroll={{ x: 1200 }}
    />
    <ScriptImportModal open={importOpen || Boolean(reimporting)} script={reimporting} onClose={() => { setImportOpen(false); setReimporting(null); }} onSaved={() => { setImportOpen(false); setReimporting(null); void refetch(); }} />
    <Drawer title={detailScript?.scriptName || t('mml.scriptDetail')} open={Boolean(viewing)} onClose={() => setViewing(null)} width={820} destroyOnHidden>
      {detailScript ? <Spin spinning={isFetching}><Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Descriptions bordered size="small" column={2}><Descriptions.Item label={t('mml.scriptName')} span={2}>{detailScript.scriptName}</Descriptions.Item><Descriptions.Item label={t('mml.description')} span={2}>{detailScript.description || '-'}</Descriptions.Item><Descriptions.Item label={t('mml.creator')}>{detailScript.creator || '-'}</Descriptions.Item><Descriptions.Item label={t('mml.updateTime')}>{formatTime(detailScript.updateTime)}</Descriptions.Item><Descriptions.Item label={t('mml.script.originalFile')}>{detailScript.originalFilename || '-'}</Descriptions.Item><Descriptions.Item label={t('mml.script.validationVersion')}>{detailScript.validationVersion || '-'}</Descriptions.Item></Descriptions>
        <Typography.Text strong>{t('mml.script.readOnlyTxtContent')}</Typography.Text>
        {detailScript.content ? <pre style={{ whiteSpace: 'pre-wrap', maxHeight: 280, overflow: 'auto' }}>{detailScript.content}</pre> : <Empty description={t('mml.script.noContent')} />}
        {detailScript.validationSummary ? <ScriptImportPreview validation={{ planItems: detailScript.planItems ?? [], issues: detailScript.validationIssues ?? [], summary: detailScript.validationSummary, originalFilename: detailScript.originalFilename }} /> : null}
      </Space></Spin> : null}
    </Drawer>
    <ScriptExecutionDrawer
      open={Boolean(execScript)}
      script={execScript}
      onClose={() => setExecScript(null)}
      onSuccess={() => {
        void refetch();
        void navigate('/mml/task-records');
      }}
    />
    <Modal title={t('mml.script.editBasicInfo')} open={Boolean(editing)} onCancel={closeBasic} onOk={() => void saveBasic()} confirmLoading={updateMutation.isPending} okText={t('common.save')} cancelText={t('common.cancel')}>
      <Form form={basicForm} layout="vertical"><Form.Item label={t('mml.scriptName')} name="scriptName" rules={[{ required: true, message: t('mml.inputScriptName') }]}><Input /></Form.Item><Form.Item label={t('mml.description')} name="description"><Input /></Form.Item></Form>
    </Modal>
  </ListPageLayout>;
}
