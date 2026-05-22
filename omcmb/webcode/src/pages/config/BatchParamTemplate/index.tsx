import { useState, useMemo } from 'react';
import { Button, Dropdown, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, message } from 'antd';
import { PlusOutlined, DeleteOutlined, CopyOutlined, MoreOutlined, SendOutlined, CheckCircleTwoTone, CloseCircleTwoTone } from '@ant-design/icons';
import type { MenuProps } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useConfigTemplates, useCreateConfigTemplate, useUpdateConfigTemplate, useDeleteConfigTemplates } from '@core/hooks/api/useConfig';
import { useDispatchTemplate } from '@core/hooks/api/useTemplate';
import { useDeviceList } from '@core/hooks/api/useDevices';
import type { ConfigTemplate } from '@core/types/config';
import type { DispatchResult, DispatchTemplateResponse } from '@core/services/api/templateApi';
import { useT } from '@/hooks/useT';

interface TemplateRow extends Record<string, unknown> {
  id: string;
  templateName: string;
  description: string;
  paramCount: number;
  deviceType: string;
  creator: string;
  createTime: string;
}

const DEVICE_TYPE_OPTIONS = [
  { label: '全部', value: 'all' },
  { label: '4G eNB', value: 'enb' },
  { label: '5G gNB', value: 'gnb' },
  { label: 'CPE', value: 'cpe' },
];

const DEVICE_TYPE_COLORS: Record<string, string> = {
  enb: 'blue',
  gnb: 'purple',
  cpe: 'cyan',
  all: 'default',
};

const mockTemplates: TemplateRow[] = [
  { id: '1', templateName: '4G标准基础参数模板', description: '适用于标准4G eNB的基础参数配置', paramCount: 42, deviceType: 'enb', creator: 'admin', createTime: '2026-01-15 10:00:00' },
  { id: '2', templateName: '5G高性能参数模板', description: '5G gNB高性能场景参数配置', paramCount: 68, deviceType: 'gnb', creator: 'operator1', createTime: '2026-02-01 14:30:00' },
  { id: '3', templateName: 'CPE室内覆盖模板', description: '室内CPE覆盖优化参数', paramCount: 23, deviceType: 'cpe', creator: 'operator2', createTime: '2026-02-10 09:00:00' },
];

export default function BatchParamTemplate() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<TemplateRow | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [form] = Form.useForm();

  // T-0120-b dispatch modal state
  const [dispatchTarget, setDispatchTarget] = useState<TemplateRow | null>(null);
  const [dispatchDeviceIds, setDispatchDeviceIds] = useState<string[]>([]);
  const [dispatchResult, setDispatchResult] = useState<DispatchTemplateResponse | null>(null);

  const { data, isLoading, refetch } = useConfigTemplates({ page, pageSize });
  const createTemplate = useCreateConfigTemplate();
  const updateTemplate = useUpdateConfigTemplate();
  const deleteTemplates = useDeleteConfigTemplates();
  const dispatchTemplate = useDispatchTemplate();
  // 设备下拉源：最多 200，按 SN 升序，足够普通场景手选；规模化筛选留后续
  const { data: devicePage, isLoading: devLoading } = useDeviceList(
    { page: 1, pageSize: 200 },
  );

  const tableSource = (data?.items ?? mockTemplates) as unknown as TemplateRow[];

  const openCreate = () => {
    setEditingTemplate(null);
    form.resetFields();
    setModalVisible(true);
  };

  const openEdit = (record: TemplateRow) => {
    setEditingTemplate(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleSave = () => {
    form.validateFields().then((vals: Partial<ConfigTemplate>) => {
      if (editingTemplate) {
        updateTemplate.mutate(
          { id: editingTemplate.id, data: vals },
          {
            onSuccess: () => {
              void message.success(t('common.save'));
              setModalVisible(false);
            },
          },
        );
      } else {
        createTemplate.mutate(vals as Omit<ConfigTemplate, 'id' | 'createTime'>, {
          onSuccess: () => {
            void message.success(t('common.save'));
            setModalVisible(false);
          },
        });
      }
    }).catch(() => undefined);
  };

  const handleDelete = (ids: string[]) => {
    deleteTemplates.mutate(ids, {
      onSuccess: () => {
        void message.success(t('common.deleteSuccess'));
        setSelectedKeys([]);
      },
    });
  };

  const openDispatch = (record: TemplateRow) => {
    setDispatchTarget(record);
    setDispatchDeviceIds([]);
    setDispatchResult(null);
  };

  const closeDispatch = () => {
    setDispatchTarget(null);
    setDispatchDeviceIds([]);
    setDispatchResult(null);
  };

  const handleDispatch = () => {
    if (!dispatchTarget || dispatchDeviceIds.length === 0) {
      void message.warning('请至少选择一台设备');
      return;
    }
    dispatchTemplate.mutate(
      { templateId: dispatchTarget.id, deviceIds: dispatchDeviceIds },
      {
        onSuccess: (resp) => {
          setDispatchResult(resp);
          if (resp.failed.length === 0) {
            void message.success(`已成功下发到 ${resp.dispatched.length} 台设备`);
          } else if (resp.dispatched.length === 0) {
            void message.error(`全部 ${resp.totalDevices} 台设备下发失败`);
          } else {
            void message.warning(
              `部分成功：${resp.dispatched.length} 成功 / ${resp.failed.length} 失败`,
            );
          }
        },
        onError: (err: Error) => {
          void message.error(`下发失败：${err.message || '未知错误'}`);
        },
      },
    );
  };

  const columns: DataTableColumn<TemplateRow>[] = useMemo(() => [
    { key: 'templateName', title: t('config.template'), dataIndex: 'templateName', width: 220, ellipsis: true },
    { key: 'description', title: t('table.description'), dataIndex: 'description', width: 260, ellipsis: true },
    { key: 'paramCount', title: t('table.total'), dataIndex: 'paramCount', width: 100 },
    {
      key: 'deviceType',
      title: t('device.productClass'),
      dataIndex: 'deviceType',
      width: 130,
      render: (val) => <Tag color={DEVICE_TYPE_COLORS[val as string] ?? 'default'}>{val as string}</Tag>,
    },
    { key: 'creator', title: t('table.operator'), dataIndex: 'creator', width: 100 },
    { key: 'createTime', title: t('table.createTime'), dataIndex: 'createTime', width: 160 },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 180,
      fixed: 'right',
      render: (_, record) => {
        const menuItems: MenuProps['items'] = [
          { key: 'copy', label: t('common.copy'), icon: <CopyOutlined />, onClick: () => void message.info(t('common.copy')) },
          { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, onClick: () => handleDelete([record.id]) },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => openEdit(record)}>{t('common.edit')}</Button>
            <Button
              type="link"
              size="small"
              icon={<SendOutlined />}
              onClick={() => openDispatch(record)}
            >
              下发
            </Button>
            <Dropdown menu={{ items: menuItems }} trigger={['click']}>
              <Button type="link" size="small" icon={<MoreOutlined />} />
            </Dropdown>
          </Space>
        );
      },
    },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.config.batchTemplate')}
      extra={
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>{t('common.add')}</Button>
          {selectedKeys.length > 0 && (
            <Popconfirm title={t('common.deleteConfirmMsg')} onConfirm={() => handleDelete(selectedKeys as string[])}>
              <Button danger>{t('common.batchDelete')}</Button>
            </Popconfirm>
          )}
        </Space>
      }
    >
      <DataTable<TemplateRow>
        tableId="batch-param-template"
        columns={columns}
        dataSource={tableSource}
        loading={isLoading}
        rowKey="id"
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={(keys) => setSelectedKeys(keys)}
        total={data?.total ?? tableSource.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        onExport={() => void message.info(t('common.exportInProgress'))}
        scroll={{ x: 1200 }}
      />

      <Modal
        title={editingTemplate ? t('common.edit') : t('common.add')}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        okText={t('common.save')}
        width={560}
        confirmLoading={createTemplate.isPending || updateTemplate.isPending}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('config.template')} name="templateName" rules={[{ required: true, message: t('common.placeholder') }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('device.productClass')} name="deviceType" rules={[{ required: true, message: t('common.pleaseSelect') }]}>
            <Select options={DEVICE_TYPE_OPTIONS} placeholder={t('common.pleaseSelect')} />
          </Form.Item>
          <Form.Item label={t('table.description')} name="description">
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>

      {/* T-0120-b 下发 Modal：选设备 → 调 dispatch API → 展示结果 */}
      <Modal
        title={`下发模板：${dispatchTarget?.templateName ?? ''}`}
        open={!!dispatchTarget}
        onCancel={closeDispatch}
        width={680}
        footer={
          dispatchResult ? (
            <Button type="primary" onClick={closeDispatch}>关闭</Button>
          ) : (
            <Space>
              <Button onClick={closeDispatch}>{t('common.cancel') ?? '取消'}</Button>
              <Button
                type="primary"
                icon={<SendOutlined />}
                loading={dispatchTemplate.isPending}
                onClick={handleDispatch}
                disabled={dispatchDeviceIds.length === 0}
              >
                确认下发到 {dispatchDeviceIds.length} 台设备
              </Button>
            </Space>
          )
        }
      >
        {!dispatchResult ? (
          <Form layout="vertical" style={{ marginTop: 8 }}>
            <Form.Item label="目标设备" required>
              <Select
                mode="multiple"
                placeholder="选择要下发的设备（可多选；按 SN 或名称搜索）"
                loading={devLoading}
                value={dispatchDeviceIds}
                onChange={setDispatchDeviceIds}
                optionFilterProp="label"
                showSearch
                style={{ width: '100%' }}
                options={(devicePage?.items ?? []).map((d) => ({
                  label: `${d.sn}${d.name && d.name !== d.sn ? ` (${d.name})` : ''}`,
                  value: d.id,
                }))}
              />
            </Form.Item>
            <div style={{ color: '#999', fontSize: 12 }}>
              说明：本次下发将强制走 Path A（模板 standardPath → privatePath 翻译 → SetParameterValues），
              不受全局 auto_configure 开关影响。每台设备独立成败。
            </div>
          </Form>
        ) : (
          <DispatchResultPanel result={dispatchResult} />
        )}
      </Modal>
    </ListPageLayout>
  );
}

// ---------------------------------------------------------------------------
// T-0120-b：dispatch 结果面板
// ---------------------------------------------------------------------------
function DispatchResultPanel({ result }: { result: DispatchTemplateResponse }) {
  const summary = (
    <Space size="large" style={{ marginBottom: 12 }}>
      <Tag color="blue">总计 {result.totalDevices}</Tag>
      <Tag color="green">成功 {result.dispatched.length}</Tag>
      <Tag color={result.failed.length > 0 ? 'red' : 'default'}>
        失败 {result.failed.length}
      </Tag>
    </Space>
  );

  const rows: Array<DispatchResult & { ok: boolean; key: string }> = [
    ...result.dispatched.map((r) => ({ ...r, ok: true, key: `s-${r.deviceId}` })),
    ...result.failed.map((r) => ({ ...r, ok: false, key: `f-${r.deviceId}` })),
  ];

  return (
    <div>
      {summary}
      <Table
        size="small"
        rowKey="key"
        pagination={false}
        dataSource={rows}
        columns={[
          {
            title: '状态',
            dataIndex: 'ok',
            width: 80,
            render: (ok: boolean) =>
              ok ? (
                <CheckCircleTwoTone twoToneColor="#52c41a" />
              ) : (
                <CloseCircleTwoTone twoToneColor="#ff4d4f" />
              ),
          },
          { title: '设备 ID', dataIndex: 'deviceId', ellipsis: true },
          {
            title: 'Task ID / 错误',
            dataIndex: 'taskId',
            render: (taskId: string | undefined, r) =>
              r.ok ? <code style={{ fontSize: 12 }}>{taskId}</code> : <span style={{ color: '#ff4d4f' }}>{r.error}</span>,
          },
        ]}
      />
    </div>
  );
}
