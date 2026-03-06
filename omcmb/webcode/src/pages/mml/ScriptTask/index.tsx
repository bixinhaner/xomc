import { useState, useMemo } from 'react';
import { Button, Form, Input, Modal, Popconfirm, Select, Space, Tag, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useMMLScripts, useCreateMMLScript, useUpdateMMLScript, useDeleteMMLScripts, useExecuteMMLScript } from '@/hooks/api/useMML';
import type { MMLScript } from '@/types/mml';
import { useT } from '@/hooks/useT';

interface ScriptRow extends Record<string, unknown> {
  id: string;
  scriptName: string;
  description: string;
  deviceType: string;
  status: 'idle' | 'running' | 'success' | 'failed';
  lastExecuteTime: string;
  creator: string;
  tags: string[];
}

const DEVICE_TYPE_OPTIONS = [
  { label: '4G eNB', value: 'enb' },
  { label: '5G gNB', value: 'gnb' },
  { label: 'CPE', value: 'cpe' },
  { label: 'universal', value: 'universal' },
];

const mockData: ScriptRow[] = [
  {
    id: '1',
    scriptName: '批量激活小区脚本',
    description: '批量激活指定设备的所有小区',
    deviceType: 'enb',
    status: 'success',
    lastExecuteTime: '2026-03-01 14:30:00',
    creator: 'admin',
    tags: ['小区管理', '批量操作'],
  },
  {
    id: '2',
    scriptName: '5G频率配置脚本',
    description: '配置5G基站NR频率参数',
    deviceType: 'gnb',
    status: 'idle',
    lastExecuteTime: '2026-02-28 10:00:00',
    creator: 'operator1',
    tags: ['频率配置', '5G'],
  },
  {
    id: '3',
    scriptName: '告警清除脚本',
    description: '清除指定类型告警',
    deviceType: 'universal',
    status: 'running',
    lastExecuteTime: '2026-03-02 09:00:00',
    creator: 'operator2',
    tags: ['告警管理'],
  },
  {
    id: '4',
    scriptName: '邻区关系批量添加',
    description: '根据规划表批量添加邻区关系',
    deviceType: 'enb',
    status: 'failed',
    lastExecuteTime: '2026-03-01 16:45:00',
    creator: 'admin',
    tags: ['邻区管理'],
  },
];

export default function ScriptTask() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRow, setEditingRow] = useState<ScriptRow | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [executeModalVisible, setExecuteModalVisible] = useState(false);
  const [executingScript, setExecutingScript] = useState<ScriptRow | null>(null);
  const [execDevices, setExecDevices] = useState<string[]>([]);
  const [form] = Form.useForm();

  const { data, isLoading, refetch } = useMMLScripts({ page, pageSize });
  const createScript = useCreateMMLScript();
  const updateScript = useUpdateMMLScript();
  const deleteScripts = useDeleteMMLScripts();
  const executeScript = useExecuteMMLScript();

  const tableSource = (data?.items ?? mockData) as unknown as ScriptRow[];

  const STATUS_MAP: Record<string, { color: string; text: string }> = useMemo(() => ({
    idle: { color: 'default', text: t('status.pending') },
    running: { color: 'processing', text: t('status.running') },
    success: { color: 'success', text: t('status.success') },
    failed: { color: 'error', text: t('status.failed') },
  }), [t]);

  const openEdit = (record: ScriptRow) => {
    setEditingRow(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const openExecute = (record: ScriptRow) => {
    setExecutingScript(record);
    setExecDevices([]);
    setExecuteModalVisible(true);
  };

  const handleSave = () => {
    form.validateFields().then((vals: Partial<MMLScript>) => {
      if (editingRow) {
        updateScript.mutate({ id: editingRow.id, data: vals }, {
          onSuccess: () => { void message.success(t('common.save')); setModalVisible(false); },
        });
      } else {
        createScript.mutate(vals as Omit<MMLScript, 'id' | 'createTime' | 'updateTime'>, {
          onSuccess: () => { void message.success(t('common.save')); setModalVisible(false); },
        });
      }
      form.resetFields();
    }).catch(() => undefined);
  };

  const handleExecute = () => {
    if (!executingScript || execDevices.length === 0) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    executeScript.mutate(
      { scriptId: executingScript.id, deviceSns: execDevices },
      {
        onSuccess: () => {
          void message.success(t('common.execute'));
          setExecuteModalVisible(false);
          void refetch();
        },
      },
    );
  };

  const columns: DataTableColumn<ScriptRow>[] = useMemo(() => [
    { key: 'scriptName', title: t('table.name'), dataIndex: 'scriptName', width: 200, ellipsis: true },
    { key: 'description', title: t('table.description'), dataIndex: 'description', width: 260, ellipsis: true },
    {
      key: 'deviceType',
      title: t('device.productType'),
      dataIndex: 'deviceType',
      width: 100,
      render: (val) => {
        const opt = DEVICE_TYPE_OPTIONS.find((o) => o.value === val);
        return <Tag color="blue">{opt?.label ?? (val as string)}</Tag>;
      },
    },
    {
      key: 'tags',
      title: t('table.type'),
      dataIndex: 'tags',
      width: 160,
      render: (val) =>
        (val as string[]).map((tag) => (
          <Tag key={tag} style={{ marginBottom: 2 }}>{tag}</Tag>
        )),
    },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const cfg = STATUS_MAP[val as string] ?? STATUS_MAP.idle;
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'lastExecuteTime', title: t('table.updateTime'), dataIndex: 'lastExecuteTime', width: 160 },
    { key: 'creator', title: t('table.operator'), dataIndex: 'creator', width: 90 },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 160,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<PlayCircleOutlined />}
            disabled={record.status === 'running'}
            onClick={() => openExecute(record)}
          >
            {t('common.execute')}
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>{t('common.edit')}</Button>
          <Popconfirm title={t('common.confirmDelete')} onConfirm={() => deleteScripts.mutate([record.id])}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>{t('common.delete')}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ], [t, STATUS_MAP]);

  return (
    <ListPageLayout
      title={t('nav.mml.script')}
      extra={
        <Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { setEditingRow(null); form.resetFields(); setModalVisible(true); }}
          >
            {t('common.add')}
          </Button>
          {selectedKeys.length > 0 && (
            <Popconfirm title={t('common.deleteConfirmMsg')} onConfirm={() => deleteScripts.mutate(selectedKeys as string[])}>
              <Button danger>{t('common.batchDelete')}</Button>
            </Popconfirm>
          )}
        </Space>
      }
    >
      <DataTable<ScriptRow>
        tableId="script-task"
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
        scroll={{ x: 1300 }}
      />

      {/* Create/Edit Script Modal */}
      <Modal
        title={editingRow ? t('common.edit') : t('common.add')}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        okText={t('common.save')}
        width={680}
        confirmLoading={createScript.isPending || updateScript.isPending}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('table.name')} name="scriptName" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('device.productType')} name="deviceType" rules={[{ required: true }]}>
            <Select options={DEVICE_TYPE_OPTIONS} placeholder={t('common.pleaseSelect')} />
          </Form.Item>
          <Form.Item label={t('table.description')} name="description">
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('nav.mml.script')} name="content" rules={[{ required: true, message: t('common.placeholder') }]}>
            <Input.TextArea
              rows={10}
              placeholder={`// MML脚本示例\nLST CELL;\nACT CELL:CELLID=0;\nACT CELL:CELLID=1;`}
              style={{ fontFamily: 'SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace', fontSize: 12 }}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Execute Script Modal */}
      <Modal
        title={`${t('common.execute')}: ${executingScript?.scriptName ?? ''}`}
        open={executeModalVisible}
        onOk={handleExecute}
        onCancel={() => setExecuteModalVisible(false)}
        okText={t('common.execute')}
        confirmLoading={executeScript.isPending}
      >
        <Form layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('device.name')} required>
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              value={execDevices}
              onChange={setExecDevices}
              options={[
                { label: 'ENB00001 - 北京朝阳基站01', value: 'ENB00001' },
                { label: 'ENB00002 - 北京海淀基站01', value: 'ENB00002' },
                { label: 'ENB00003 - 上海浦东基站01', value: 'ENB00003' },
                { label: 'GNB00001 - 北京5G基站01', value: 'GNB00001' },
              ]}
              style={{ width: '100%' }}
            />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
