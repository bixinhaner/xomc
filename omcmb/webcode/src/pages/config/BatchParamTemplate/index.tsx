import { useState, useMemo } from 'react';
import { Button, Dropdown, Form, Input, Modal, Popconfirm, Select, Space, Tag, message } from 'antd';
import { PlusOutlined, DeleteOutlined, CopyOutlined, MoreOutlined } from '@ant-design/icons';
import type { MenuProps } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useConfigTemplates, useCreateConfigTemplate, useUpdateConfigTemplate, useDeleteConfigTemplates } from '@/hooks/api/useConfig';
import type { ConfigTemplate } from '@/types/config';
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

  const { data, isLoading, refetch } = useConfigTemplates({ page, pageSize });
  const createTemplate = useCreateConfigTemplate();
  const updateTemplate = useUpdateConfigTemplate();
  const deleteTemplates = useDeleteConfigTemplates();

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

  const columns: DataTableColumn<TemplateRow>[] = useMemo(() => [
    { key: 'templateName', title: t('config.template'), dataIndex: 'templateName', width: 220, ellipsis: true },
    { key: 'description', title: t('table.description'), dataIndex: 'description', width: 260, ellipsis: true },
    { key: 'paramCount', title: t('table.total'), dataIndex: 'paramCount', width: 100 },
    {
      key: 'deviceType',
      title: t('device.productType'),
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
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const menuItems: MenuProps['items'] = [
          { key: 'copy', label: t('common.copy'), icon: <CopyOutlined />, onClick: () => void message.info(t('common.copy')) },
          { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, onClick: () => handleDelete([record.id]) },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => openEdit(record)}>{t('common.edit')}</Button>
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
        scroll={{ x: 1100 }}
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
          <Form.Item label={t('device.productType')} name="deviceType" rules={[{ required: true, message: t('common.pleaseSelect') }]}>
            <Select options={DEVICE_TYPE_OPTIONS} placeholder={t('common.pleaseSelect')} />
          </Form.Item>
          <Form.Item label={t('table.description')} name="description">
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
