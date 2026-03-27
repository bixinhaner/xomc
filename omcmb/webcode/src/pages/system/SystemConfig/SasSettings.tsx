import { useState } from 'react';
import { Form, Switch, Divider, Space, Table, Button, Modal, Input, Tag, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';

interface SasProvider {
  id: string;
  providerName: string;
  url: string;
  certName: string;
  validTimeStr: string;
  updateTimeStr: string;
}

interface SasSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// Mock 数据
const mockProviders: SasProvider[] = [
  { id: '1', providerName: 'SAS-Provider-1', url: 'https://sas.example.com/api', certName: 'cert_sas_1.pem', validTimeStr: '2027-06-15', updateTimeStr: '2026-03-20 10:00:00' },
  { id: '2', providerName: 'SAS-Provider-2', url: 'https://sas2.example.com/api', certName: 'cert_sas_2.pem', validTimeStr: '2027-12-31', updateTimeStr: '2026-02-10 14:30:00' },
];

export default function SasSettings({ form }: SasSettingsProps) {
  const t = useT();
  const [providers, setProviders] = useState<SasProvider[]>(mockProviders);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingProvider, setEditingProvider] = useState<SasProvider | null>(null);
  const [providerForm] = Form.useForm();

  const handleAddProvider = () => {
    setEditingProvider(null);
    providerForm.resetFields();
    setModalVisible(true);
  };

  const handleEditProvider = (provider: SasProvider) => {
    setEditingProvider(provider);
    providerForm.setFieldsValue({
      providerName: provider.providerName,
      url: provider.url,
      certName: provider.certName,
    });
    setModalVisible(true);
  };

  const handleDeleteProvider = (id: string) => {
    setProviders(providers.filter(p => p.id !== id));
    void message.success('删除成功');
  };

  const handleModalOk = () => {
    providerForm.validateFields().then((values) => {
      if (editingProvider) {
        setProviders(providers.map(p => p.id === editingProvider.id ? {
          ...p,
          ...values,
          updateTimeStr: new Date().toLocaleString(),
        } : p));
        void message.success('修改成功');
      } else {
        const newProvider: SasProvider = {
          id: Date.now().toString(),
          ...values,
          validTimeStr: '2027-12-31',
          updateTimeStr: new Date().toLocaleString(),
        };
        setProviders([...providers, newProvider]);
        void message.success('添加成功');
      }
      setModalVisible(false);
    });
  };

  const columns: ColumnsType<SasProvider> = [
    {
      title: 'Provider名称',
      dataIndex: 'providerName',
      key: 'providerName',
    },
    {
      title: '服务器URL',
      dataIndex: 'url',
      key: 'url',
      ellipsis: true,
    },
    {
      title: 'TLS证书',
      key: 'cert',
      render: (_: unknown, record: SasProvider) => (
        <Space direction="vertical" size={0}>
          <span>{record.certName}</span>
          <Tag color="green">有效期至 {record.validTimeStr}</Tag>
        </Space>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updateTimeStr',
      key: 'updateTimeStr',
    },
    {
      title: '操作',
      key: 'operation',
      width: 120,
      render: (_: unknown, record: SasProvider) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => handleEditProvider(record)}>
            编辑
          </Button>
          <Button size="small" danger icon={<DeleteOutlined />} onClick={() => handleDeleteProvider(record.id)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Form form={form} layout="vertical" size="small" initialValues={{
        mainLogHbEnable: false,
      }}>
        {/* SAS心跳日志 */}
        <Divider orientation="left" plain>SAS心跳日志</Divider>
        <Form.Item name="mainLogHbEnable" label="SAS心跳日志" valuePropName="checked">
          <Switch checkedChildren="开启" unCheckedChildren="关闭" />
        </Form.Item>

        {/* SAS Provider列表 */}
        <Divider orientation="left" plain>SAS Provider列表</Divider>
        <div style={{ marginBottom: 16 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddProvider}>
            添加Provider
          </Button>
        </div>
        <Table
          dataSource={providers}
          columns={columns}
          rowKey="id"
          pagination={false}
          size="small"
        />
      </Form>

      {/* 添加/编辑Provider弹窗 */}
      <Modal
        title={editingProvider ? '编辑Provider' : '添加Provider'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        okText="确定"
        cancelText="取消"
      >
        <Form form={providerForm} layout="vertical">
          <Form.Item name="providerName" label="Provider名称" rules={[{ required: true, message: '请输入Provider名称' }]}>
            <Input placeholder="请输入Provider名称" />
          </Form.Item>
          <Form.Item name="url" label="服务器URL" rules={[{ required: true, message: '请输入服务器URL' }]}>
            <Input placeholder="https://sas.example.com/api" />
          </Form.Item>
          <Form.Item name="certName" label="TLS证书名称">
            <Input placeholder="请输入证书名称" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
