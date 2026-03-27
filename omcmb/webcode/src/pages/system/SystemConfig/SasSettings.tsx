import { useState } from 'react';
import { Form, Switch, Space, Table, Button, Modal, Input, Card, message } from 'antd';
import { PlusOutlined, MoreOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';

interface SasProvider {
  id: string;
  providerName: string;
  url: string;
  certName: string;
  validTimeStr: string;
  updateTimeStr: string;
  uploadSuccess: number;
}

interface SasSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// Mock 数据
const mockProviders: SasProvider[] = [
  { id: '1', providerName: 'SAS-Provider-1', url: 'https://sas.example.com/api', certName: 'cert_sas_1.pem', validTimeStr: '2027-06-15', updateTimeStr: '2026-03-20 10:00:00', uploadSuccess: 1 },
  { id: '2', providerName: 'SAS-Provider-2', url: 'https://sas2.example.com/api', certName: 'cert_sas_2.pem', validTimeStr: '2027-12-31', updateTimeStr: '2026-02-10 14:30:00', uploadSuccess: 0 },
];

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

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
          uploadSuccess: 1,
        };
        setProviders([...providers, newProvider]);
        void message.success('添加成功');
      }
      setModalVisible(false);
    });
  };

  const columns: ColumnsType<SasProvider> = [
    {
      title: '',
      key: 'operation',
      width: 50,
      render: (_: unknown, record: SasProvider) => (
        <Button size="small" type="text" icon={<MoreOutlined />} onClick={() => handleEditProvider(record)} />
      ),
    },
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
        <Space>
          {record.uploadSuccess === 1 ? (
            <span style={{ color: '#67D972' }}>●</span>
          ) : (
            <span style={{ color: '#E88282' }}>●</span>
          )}
          <span>{record.certName}({record.validTimeStr})</span>
        </Space>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updateTimeStr',
      key: 'updateTimeStr',
    },
  ];

  return (
    <>
      <Form form={form} layout="vertical" size="small" initialValues={{
        mainLogHbEnable: false,
      }}>
        {/* SAS心跳日志 */}
        <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>SAS心跳日志</span>} style={{ marginBottom: 16 }}>
          <Space>
            <span>SAS心跳日志开关</span>
            <Form.Item name="mainLogHbEnable" valuePropName="checked" noStyle>
              <Switch size="small" />
            </Form.Item>
          </Space>
        </Card>

        {/* SAS Provider列表 */}
        <Card
          size="small"
          title={<span style={{ fontSize: 14, fontWeight: 600 }}>SAS Provider列表</span>}
          extra={
            <Button type="primary" icon={<PlusOutlined />} size="small" onClick={handleAddProvider}>
              添加
            </Button>
          }
        >
          <Table
            dataSource={providers}
            columns={columns}
            rowKey="id"
            pagination={false}
            size="small"
            bordered
          />
        </Card>
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
