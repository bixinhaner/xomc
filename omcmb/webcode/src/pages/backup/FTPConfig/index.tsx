import { useState, useMemo } from 'react';
import { Button, Dropdown, Form, Input, InputNumber, Modal, Select, Space, Switch, Tag, message } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ApiOutlined, MoreOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useFTPConfigs, useCreateFTPConfig, useUpdateFTPConfig, useDeleteFTPConfigs, useTestFTPConnection } from '@/hooks/api/useBackup';
import type { FTPConfig } from '@/mock/data/backup';
import { useT } from '@/hooks/useT';

interface FTPRow extends Record<string, unknown> {
  id: string;
  configName: string;
  host: string;
  port: number;
  username: string;
  protocol: 'FTP' | 'SFTP' | 'FTPS';
  remotePath: string;
  passive: boolean;
  enabled: boolean;
  createTime: string;
}

const PROTOCOL_COLORS: Record<string, string> = {
  FTP: 'blue',
  SFTP: 'green',
  FTPS: 'purple',
};

const DEFAULT_PORTS: Record<string, number> = {
  FTP: 21,
  SFTP: 22,
  FTPS: 990,
};

const mockData: FTPRow[] = [
  { id: '1', configName: '主备份FTP服务器', host: '192.168.100.10', port: 21, username: 'ftpuser', protocol: 'FTP', remotePath: '/backup/omc', passive: true, enabled: true, createTime: '2026-01-10 10:00:00' },
  { id: '2', configName: '异地备份SFTP', host: '10.200.1.50', port: 22, username: 'sftpuser', protocol: 'SFTP', remotePath: '/data/backup', passive: false, enabled: true, createTime: '2026-01-15 14:00:00' },
  { id: '3', configName: '云存储FTPS', host: 'ftps.cloud.example.com', port: 990, username: 'clouduser', protocol: 'FTPS', remotePath: '/omc/backup', passive: true, enabled: false, createTime: '2026-02-01 09:00:00' },
];

export default function FTPConfigPage() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRow, setEditingRow] = useState<FTPRow | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [form] = Form.useForm();

  const { data, isLoading, refetch } = useFTPConfigs({ page, pageSize });
  const createConfig = useCreateFTPConfig();
  const updateConfig = useUpdateFTPConfig();
  const deleteConfigs = useDeleteFTPConfigs();
  const testConnection = useTestFTPConnection();

  const tableSource = (data?.items ?? mockData) as unknown as FTPRow[];

  const openEdit = (record: FTPRow) => {
    setEditingRow(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleProtocolChange = (protocol: string) => {
    const defaultPort = DEFAULT_PORTS[protocol];
    if (defaultPort) {
      form.setFieldValue('port', defaultPort);
    }
  };

  const handleSave = () => {
    form.validateFields().then((vals: Partial<FTPConfig>) => {
      if (editingRow) {
        updateConfig.mutate({ id: editingRow.id, data: vals }, {
          onSuccess: () => { void message.success(t('common.save')); setModalVisible(false); },
        });
      } else {
        createConfig.mutate(vals as Omit<FTPConfig, 'id' | 'createTime'>, {
          onSuccess: () => { void message.success(t('common.save')); setModalVisible(false); },
        });
      }
      form.resetFields();
    }).catch(() => undefined);
  };

  const handleTestConnection = (id: string) => {
    setTestingId(id);
    testConnection.mutate(id, {
      onSuccess: (result: unknown) => {
        const r = result as { success: boolean; message?: string };
        if (r.success) {
          void message.success(t('status.success'));
        } else {
          void message.error(t('status.failed'));
        }
        setTestingId(null);
      },
      onError: () => {
        void message.error(t('status.failed'));
        setTestingId(null);
      },
    });
  };

  const columns: DataTableColumn<FTPRow>[] = useMemo(() => [
    { key: 'configName', title: t('table.name'), dataIndex: 'configName', width: 180 },
    { key: 'host', title: t('table.ip'), dataIndex: 'host', width: 180, mono: true, copyable: true },
    { key: 'port', title: t('table.ip'), dataIndex: 'port', width: 80 },
    {
      key: 'protocol',
      title: t('table.type'),
      dataIndex: 'protocol',
      width: 80,
      render: (val) => <Tag color={PROTOCOL_COLORS[val as string]}>{val as string}</Tag>,
    },
    { key: 'username', title: t('table.operator'), dataIndex: 'username', width: 120 },
    { key: 'remotePath', title: t('table.description'), dataIndex: 'remotePath', width: 160, mono: true, ellipsis: true },
    {
      key: 'passive',
      title: t('table.type'),
      dataIndex: 'passive',
      width: 90,
      render: (val) => <Tag color={val ? 'blue' : 'default'}>{val ? t('status.enabled') : t('status.disabled')}</Tag>,
    },
    {
      key: 'enabled',
      title: t('table.status'),
      dataIndex: 'enabled',
      width: 90,
      render: (val, record) => (
        <Switch
          size="small"
          checked={val as boolean}
          checkedChildren={t('common.enable')}
          unCheckedChildren={t('common.disable')}
          onChange={() => updateConfig.mutate({ id: record.id, data: { enabled: !val } })}
        />
      ),
    },
    { key: 'createTime', title: t('table.createTime'), dataIndex: 'createTime', width: 160 },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const items: MenuProps['items'] = [
          {
            key: 'test',
            label: t('common.execute'),
            icon: <ApiOutlined />,
            onClick: () => handleTestConnection(record.id),
          },
          {
            key: 'delete',
            label: t('common.delete'),
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => deleteConfigs.mutate([record.id]),
          },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>{t('common.edit')}</Button>
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
    },
  ], [t, testingId]);

  return (
    <ListPageLayout
      title={t('nav.backup.ftp')}
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => { setEditingRow(null); form.resetFields(); setModalVisible(true); }}
        >
          {t('common.add')}
        </Button>
      }
    >
      <DataTable<FTPRow>
        tableId="ftp-config"
        columns={columns}
        dataSource={tableSource}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? tableSource.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1500 }}
      />

      <Modal
        title={editingRow ? t('common.edit') : t('common.add')}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        okText={t('common.save')}
        width={560}
        confirmLoading={createConfig.isPending || updateConfig.isPending}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label={t('table.name')} name="configName" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.type')} name="protocol" rules={[{ required: true }]} initialValue="FTP">
            <Select
              options={[
                { label: 'FTP (端口 21)', value: 'FTP' },
                { label: 'SFTP (端口 22)', value: 'SFTP' },
                { label: 'FTPS (端口 990)', value: 'FTPS' },
              ]}
              onChange={handleProtocolChange}
            />
          </Form.Item>
          <Form.Item label={t('table.ip')} name="host" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.ip')} name="port" rules={[{ required: true }]} initialValue={21}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('table.operator')} name="username" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.description')} name="password" rules={[{ required: !editingRow }]}>
            <Input.Password placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.description')} name="remotePath" rules={[{ required: true }]}>
            <Input placeholder="/backup/omc" />
          </Form.Item>
          <Form.Item label={t('table.type')} name="passive" valuePropName="checked" initialValue>
            <Switch checkedChildren={t('status.enabled')} unCheckedChildren={t('status.disabled')} />
          </Form.Item>
          <Form.Item label={t('common.enable')} name="enabled" valuePropName="checked" initialValue>
            <Switch checkedChildren={t('common.enable')} unCheckedChildren={t('common.disable')} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
