import React, { useCallback, useMemo, useState } from 'react';
import { Button, Form, Input, Modal, Select, Space, Tag, Typography, message } from 'antd';
import {
  CheckOutlined,
  EyeOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

const { Text, TextArea } = Typography;

interface HandoverTask {
  id: string;
  taskId: string;
  deviceName: string;
  deviceSn: string;
  handoverStatus: 'pending' | 'in-progress' | 'completed' | 'rejected';
  handoverTime: string;
  operator: string;
  remarks: string;
  fromTeam: string;
  toTeam: string;
}

const MOCK_HANDOVER_TASKS: HandoverTask[] = [
  {
    id: '1', taskId: 'HO-2024-001', deviceName: '北京朝阳基站-001', deviceSn: 'SN-BJ001',
    handoverStatus: 'completed', handoverTime: '2024-03-01 10:00:00', operator: '张工',
    remarks: '设备运行正常，完成交维', fromTeam: '建设组', toTeam: '运维组',
  },
  {
    id: '2', taskId: 'HO-2024-002', deviceName: '上海浦东基站-002', deviceSn: 'SN-SH002',
    handoverStatus: 'in-progress', handoverTime: '2024-03-01 14:00:00', operator: '李工',
    remarks: '正在进行现场检查', fromTeam: '建设组', toTeam: '运维组',
  },
  {
    id: '3', taskId: 'HO-2024-003', deviceName: '广州天河基站-003', deviceSn: 'SN-GZ003',
    handoverStatus: 'pending', handoverTime: '2024-03-02 09:00:00', operator: '王工',
    remarks: '', fromTeam: '建设组', toTeam: '运维组',
  },
  {
    id: '4', taskId: 'HO-2024-004', deviceName: '深圳南山基站-004', deviceSn: 'SN-SZ004',
    handoverStatus: 'rejected', handoverTime: '2024-02-28 15:00:00', operator: '赵工',
    remarks: '设备存在天线故障，退回整改', fromTeam: '建设组', toTeam: '运维组',
  },
  {
    id: '5', taskId: 'HO-2024-005', deviceName: '成都武侯基站-005', deviceSn: 'SN-CD005',
    handoverStatus: 'completed', handoverTime: '2024-02-29 11:00:00', operator: '陈工',
    remarks: '顺利完成交维，签署验收单', fromTeam: '建设组', toTeam: '运维组',
  },
];

export default function HandoverManagement() {
  const t = useT();
  const [tasks, setTasks] = useState<HandoverTask[]>(MOCK_HANDOVER_TASKS);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [detailTask, setDetailTask] = useState<HandoverTask | null>(null);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createForm] = Form.useForm();

  const STATUS_CONFIG: Record<HandoverTask['handoverStatus'], { label: string; color: string }> = useMemo(() => ({
    pending: { label: t('status.pending'), color: 'default' },
    'in-progress': { label: t('status.running'), color: 'processing' },
    completed: { label: t('status.success'), color: 'success' },
    rejected: { label: t('status.failed'), color: 'error' },
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'taskId', label: t('table.index'), type: 'input' },
    { name: 'deviceName', label: t('device.name'), type: 'input' },
    { name: 'deviceSn', label: 'SN', type: 'input' },
    {
      name: 'handoverStatus',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.pending'), value: 'pending' },
        { label: t('status.running'), value: 'in-progress' },
        { label: t('status.success'), value: 'completed' },
        { label: t('status.failed'), value: 'rejected' },
      ],
    },
    { name: 'timeRange', label: t('table.time'), type: 'date-range' },
  ], [t]);

  const filteredTasks = useMemo(() => {
    return tasks.filter((tk) => {
      if (filterParams.handoverStatus && tk.handoverStatus !== filterParams.handoverStatus) return false;
      if (filterParams.taskId && !tk.taskId.toLowerCase().includes(String(filterParams.taskId).toLowerCase())) return false;
      if (filterParams.deviceSn && !tk.deviceSn.toLowerCase().includes(String(filterParams.deviceSn).toLowerCase())) return false;
      return true;
    });
  }, [tasks, filterParams]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const handleComplete = useCallback((id: string) => {
    Modal.confirm({
      title: t('common.confirm'),
      content: t('common.confirm'),
      okText: t('common.confirm'),
      onOk: () => {
        setTasks((prev) =>
          prev.map((tk) =>
            tk.id === id ? { ...tk, handoverStatus: 'completed' } : tk
          )
        );
        void message.success(t('status.success'));
      },
    });
  }, [t]);

  const handleCreateHandover = useCallback(async () => {
    try {
      const values = await createForm.validateFields() as { deviceSn: string; fromTeam: string; toTeam: string; operator: string; remarks: string };
      const newTask: HandoverTask = {
        id: String(Date.now()),
        taskId: `HO-2024-${String(tasks.length + 1).padStart(3, '0')}`,
        deviceName: `Device-${values.deviceSn}`,
        deviceSn: values.deviceSn,
        handoverStatus: 'pending',
        handoverTime: new Date().toLocaleString('zh-CN'),
        operator: values.operator,
        fromTeam: values.fromTeam,
        toTeam: values.toTeam,
        remarks: values.remarks ?? '',
      };
      setTasks((prev) => [newTask, ...prev]);
      setCreateModalOpen(false);
      createForm.resetFields();
      void message.success(t('status.success'));
    } catch {
      // validation error
    }
  }, [createForm, tasks.length, t]);

  const columns = useMemo(
    (): DataTableColumn<HandoverTask>[] => [
      {
        key: 'taskId',
        title: t('table.index'),
        dataIndex: 'taskId',
        width: 140,
        render: (v) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(v)}</Text>,
      },
      { key: 'deviceName', title: t('device.name'), dataIndex: 'deviceName', width: 180, ellipsis: true },
      {
        key: 'deviceSn',
        title: 'SN',
        dataIndex: 'deviceSn',
        width: 140,
        mono: true,
        copyable: true,
      },
      {
        key: 'handoverStatus',
        title: t('table.status'),
        dataIndex: 'handoverStatus',
        width: 100,
        render: (_val, record) => {
          const cfg = STATUS_CONFIG[record.handoverStatus];
          return <Tag color={cfg.color}>{cfg.label}</Tag>;
        },
      },
      { key: 'handoverTime', title: t('table.time'), dataIndex: 'handoverTime', width: 160 },
      { key: 'fromTeam', title: t('table.vendor'), dataIndex: 'fromTeam', width: 100 },
      { key: 'toTeam', title: t('table.operator'), dataIndex: 'toTeam', width: 100 },
      { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 90 },
      {
        key: 'remarks',
        title: t('table.description'),
        dataIndex: 'remarks',
        width: 200,
        ellipsis: true,
        render: (v) => v ? String(v) : <Text type="secondary">-</Text>,
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 120,
        fixed: 'right',
        render: (_val, record) => (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => setDetailTask(record)}
            >
              {t('common.detail')}
            </Button>
            {record.handoverStatus === 'in-progress' && (
              <Button
                type="link"
                size="small"
                icon={<CheckOutlined />}
                onClick={() => handleComplete(record.id)}
              >
                {t('common.finish')}
              </Button>
            )}
            {record.handoverStatus === 'pending' && (
              <Button
                type="link"
                size="small"
                onClick={() => {
                  setTasks((prev) =>
                    prev.map((tk) =>
                      tk.id === record.id ? { ...tk, handoverStatus: 'in-progress' } : tk
                    )
                  );
                  void message.info(t('status.running'));
                }}
              >
                {t('common.execute')}
              </Button>
            )}
          </Space>
        ),
      },
    ],
    [handleComplete, t, STATUS_CONFIG]
  );

  return (
    <>
      <ListPageLayout
        title={t('nav.device.handover')}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />}>{t('common.refresh')}</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
              {t('common.add')}
            </Button>
          </Space>
        }
      >
        <FilterBar
          filterId="handover-management"
          fields={FILTER_FIELDS}
          onSearch={handleSearch}
          onReset={handleReset}
          collapsedRows={1}
        />

        <DataTable<HandoverTask>
          tableId="handover-management-table"
          columns={columns}
          dataSource={filteredTasks}
          loading={false}
          rowKey="id"
          total={filteredTasks.length}
          pageSize={20}
          currentPage={currentPage}
          onPageChange={(page) => setCurrentPage(page)}
          defaultDensity="default"
        />
      </ListPageLayout>

      {/* Detail Modal */}
      <Modal
        title={`${t('common.detail')} - ${detailTask?.taskId ?? ''}`}
        open={Boolean(detailTask)}
        onCancel={() => setDetailTask(null)}
        footer={<Button onClick={() => setDetailTask(null)}>{t('common.close')}</Button>}
        width={520}
      >
        {detailTask && (
          <table style={{ width: '100%', fontSize: 14, marginTop: 16 }}>
            <tbody>
              {[
                { label: t('table.index'), value: detailTask.taskId },
                { label: t('device.name'), value: detailTask.deviceName },
                { label: t('device.sn'), value: detailTask.deviceSn },
                { label: t('table.status'), value: STATUS_CONFIG[detailTask.handoverStatus].label },
                { label: t('table.time'), value: detailTask.handoverTime },
                { label: t('table.vendor'), value: detailTask.fromTeam },
                { label: t('table.operator'), value: detailTask.toTeam },
                { label: t('table.operator'), value: detailTask.operator },
                { label: t('table.description'), value: detailTask.remarks || '-' },
              ].map(({ label, value }, idx) => (
                <tr key={idx} style={{ borderBottom: '1px solid #f5f5f5' }}>
                  <td style={{ padding: '8px 0', color: '#8c8c8c', width: 100 }}>{label}</td>
                  <td style={{ padding: '8px 0', fontWeight: 500 }}>{value}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Modal>

      {/* Create Modal */}
      <Modal
        title={t('common.add')}
        open={createModalOpen}
        onOk={() => void handleCreateHandover()}
        onCancel={() => setCreateModalOpen(false)}
        okText={t('common.confirm')}
        width={520}
      >
        <Form form={createForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="deviceSn" label={t('device.sn')} rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="fromTeam" label={t('table.vendor')} rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="toTeam" label={t('table.operator')} rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="operator" label={t('table.operator')} rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="remarks" label={t('table.description')}>
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
