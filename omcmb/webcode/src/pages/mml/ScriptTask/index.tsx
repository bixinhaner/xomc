import { useMemo, useState } from 'react';
import { Button, Descriptions, Drawer, Empty, Form, Input, Modal, Space, Spin, Tag, Typography, message } from 'antd';
import { DownloadOutlined, PlusOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import SearchInput from '@/components/SearchInput';
import { useT } from '@/hooks/useT';
import type { MMLScript } from '@core/types/mml';
import { useMMLScripts, useMMLScriptById, useUpdateMMLScript, useDeleteMMLScripts } from '@core/hooks/api/useMML';
import ScriptImportModal from './ScriptImportModal';
import ScriptExecutionDrawer from './ScriptExecutionDrawer';
import ScriptImportPreview from './ScriptImportPreview';

function formatTime(iso?: string | null): string {
  if (!iso) return '-';
  const value = dayjs(iso);
  return value.isValid() ? value.format('YYYY-MM-DD HH:mm:ss') : '-';
}

function downloadScript(script: MMLScript) {
  const url = URL.createObjectURL(new Blob([script.content || ''], { type: 'text/plain;charset=utf-8' }));
  const anchor = document.createElement('a'); anchor.href = url; anchor.download = script.originalFilename || `${script.scriptName || 'mml-script'}.txt`; anchor.click(); URL.revokeObjectURL(url);
}

interface BasicForm { scriptName: string; description: string; }

export default function ScriptTask() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [search, setSearch] = useState('');
  const [importOpen, setImportOpen] = useState(false);
  const [reimporting, setReimporting] = useState<MMLScript | null>(null);
  const [viewing, setViewing] = useState<MMLScript | null>(null);
  const [execScript, setExecScript] = useState<MMLScript | null>(null);
  const [editing, setEditing] = useState<MMLScript | null>(null);
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
    updateMutation.mutate({ id: editing.id, data: { scriptName: values.scriptName.trim(), description: values.description ?? '' } }, { onSuccess: () => { void refetch(); closeBasic(); void message.success('保存成功'); } });
  };
  const columns: DataTableColumn<MMLScript>[] = [
    { key: 'operation', title: t('table.operation'), width: 360, render: (_, record) => <Space size={2} wrap>
      <Button type="link" size="small" onClick={() => setViewing(record)}>查看</Button>
      <Button type="link" size="small" onClick={() => setExecScript(record)}>执行</Button>
      <Button type="link" size="small" onClick={() => setReimporting(record)}>重新导入</Button>
      <Button type="link" size="small" icon={<DownloadOutlined />} onClick={() => downloadScript(record)}>下载 TXT</Button>
      <Button type="link" size="small" onClick={() => { setEditing(record); basicForm.setFieldsValue({ scriptName: record.scriptName, description: record.description }); }}>编辑</Button>
      <Button type="link" size="small" danger onClick={() => Modal.confirm({ title: '确认删除？', okText: '删除', okButtonProps: { danger: true }, onOk: () => new Promise<void>((resolve, reject) => deleteMutation.mutate([record.id], { onSuccess: () => { void refetch(); resolve(); }, onError: reject })) })}>删除</Button>
    </Space> },
    { key: 'scriptName', title: t('mml.scriptName'), dataIndex: 'scriptName', ellipsis: true },
    { key: 'description', title: t('mml.description'), dataIndex: 'description', ellipsis: true, render: (value) => String(value || '-') },
    { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
    { key: 'updatedAt', title: t('mml.updateTime'), dataIndex: 'updateTime', width: 170, render: (value) => formatTime(value as string) },
    { key: 'status', title: '状态', dataIndex: 'status', width: 100, render: (value) => <Tag>{String(value || '-')}</Tag> },
  ];

  return <ListPageLayout title={t('nav.mml.script')} extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setImportOpen(true)}>导入 TXT</Button>}>
    <SearchInput placeholder={t('mml.scriptName')} allowClear style={{ width: 300, marginBottom: 16 }} onSearch={(value) => { setSearch(value); setPage(1); }} />
    <DataTable<MMLScript> tableId="mml-scripts" columns={columns} dataSource={scripts} loading={isLoading} rowKey="id" total={data?.total ?? 0} currentPage={page} pageSize={pageSize} onPageChange={(nextPage, nextSize) => { setPage(nextPage); setPageSize(nextSize); }} onRefresh={() => void refetch()} scroll={{ x: 1200 }} />
    <ScriptImportModal open={importOpen || Boolean(reimporting)} script={reimporting} onClose={() => { setImportOpen(false); setReimporting(null); }} onSaved={() => { setImportOpen(false); setReimporting(null); void refetch(); }} />
    <Drawer title={detailScript?.scriptName || '脚本详情'} open={Boolean(viewing)} onClose={() => setViewing(null)} width={820} destroyOnHidden>
      {detailScript ? <Spin spinning={isFetching}><Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Descriptions bordered size="small" column={2}><Descriptions.Item label="脚本名称" span={2}>{detailScript.scriptName}</Descriptions.Item><Descriptions.Item label="描述" span={2}>{detailScript.description || '-'}</Descriptions.Item><Descriptions.Item label="创建者">{detailScript.creator || '-'}</Descriptions.Item><Descriptions.Item label="更新时间">{formatTime(detailScript.updateTime)}</Descriptions.Item><Descriptions.Item label="原始文件">{detailScript.originalFilename || '-'}</Descriptions.Item><Descriptions.Item label="校验版本">{detailScript.validationVersion || '-'}</Descriptions.Item></Descriptions>
        <Typography.Text strong>只读 TXT 内容</Typography.Text>
        {detailScript.content ? <pre style={{ whiteSpace: 'pre-wrap', maxHeight: 280, overflow: 'auto' }}>{detailScript.content}</pre> : <Empty description="暂无脚本内容" />}
        {detailScript.validationSummary ? <ScriptImportPreview validation={{ planItems: detailScript.planItems ?? [], issues: detailScript.validationIssues ?? [], summary: detailScript.validationSummary, originalFilename: detailScript.originalFilename }} /> : null}
      </Space></Spin> : null}
    </Drawer>
    <ScriptExecutionDrawer open={Boolean(execScript)} script={execScript} onClose={() => setExecScript(null)} onSuccess={() => void refetch()} />
    <Modal title="编辑脚本基本信息" open={Boolean(editing)} onCancel={closeBasic} onOk={() => void saveBasic()} confirmLoading={updateMutation.isPending}>
      <Form form={basicForm} layout="vertical"><Form.Item label="脚本名称" name="scriptName" rules={[{ required: true, message: '请输入脚本名称' }]}><Input /></Form.Item><Form.Item label="描述" name="description"><Input /></Form.Item></Form>
    </Modal>
  </ListPageLayout>;
}
