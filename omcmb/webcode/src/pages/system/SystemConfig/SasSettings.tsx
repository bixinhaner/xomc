import { useState } from 'react';
import { Form, Switch, Space, Table, Button, Modal, Input, Radio, Card, message, Upload, Dropdown } from 'antd';
import { PlusOutlined, MoreOutlined, UploadOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { MenuProps } from 'antd';
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
  { id: '2', providerName: 'SAS-Provider-2', url: 'https://sas2.example.com/api', certName: 'cert_sas_2.p12', validTimeStr: '2027-12-31', updateTimeStr: '2026-02-10 14:30:00', uploadSuccess: 0 },
];

export default function SasSettings({ form }: SasSettingsProps) {
  const t = useT();
  const [providers, setProviders] = useState<SasProvider[]>(mockProviders);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingProvider, setEditingProvider] = useState<SasProvider | null>(null);
  const [providerForm] = Form.useForm();
  const [certType, setCertType] = useState<'pem' | 'p12'>('pem');
  const [certFileName, setCertFileName] = useState('');
  const [privateKeyFileName, setPrivateKeyFileName] = useState('');

  const handleAddProvider = () => {
    setEditingProvider(null);
    providerForm.resetFields();
    setCertType('pem');
    setCertFileName('');
    setPrivateKeyFileName('');
    setModalVisible(true);
  };

  const handleEditProvider = (provider: SasProvider) => {
    setEditingProvider(provider);
    providerForm.setFieldsValue({
      provider: provider.providerName,
      serverUrl: provider.url,
      type: provider.certName.endsWith('.p12') ? 'p12' : 'pem',
      password: '',
    });
    setCertType(provider.certName.endsWith('.p12') ? 'p12' : 'pem');
    setCertFileName(provider.certName);
    setPrivateKeyFileName('');
    setModalVisible(true);
  };

  const handleDeleteProvider = (id: string) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除此SAS Provider吗？',
      okText: '确定',
      cancelText: '取消',
      onOk: () => {
        setProviders(providers.filter(p => p.id !== id));
        void message.success('删除成功');
      },
    });
  };

  const getOperationMenu = (record: SasProvider): MenuProps['items'] => [
    {
      key: 'edit',
      label: '修改',
      onClick: () => handleEditProvider(record),
    },
    {
      key: 'delete',
      label: '删除',
      danger: true,
      onClick: () => handleDeleteProvider(record.id),
    },
  ];

  const handleModalOk = () => {
    providerForm.validateFields().then((values) => {
      if (editingProvider) {
        setProviders(providers.map(p => p.id === editingProvider.id ? {
          ...p,
          providerName: values.provider,
          url: values.serverUrl,
          certName: certFileName || p.certName,
          updateTimeStr: new Date().toLocaleString(),
        } : p));
        void message.success('修改成功');
      } else {
        const newProvider: SasProvider = {
          id: Date.now().toString(),
          providerName: values.provider,
          url: values.serverUrl,
          certName: certFileName || 'cert.pem',
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

  const handleCertTypeChange = (e: any) => {
    const type = e.target.value;
    setCertType(type);
    setCertFileName('');
    setPrivateKeyFileName('');
    providerForm.setFieldValue('password', '');
  };

  const handleCertFileChange = (info: any) => {
    if (info.file) {
      setCertFileName(info.file.name);
    }
  };

  const handlePrivateKeyFileChange = (info: any) => {
    if (info.file) {
      setPrivateKeyFileName(info.file.name);
    }
  };

  const columns: ColumnsType<SasProvider> = [
    {
      title: '',
      key: 'operation',
      width: 50,
      render: (_: unknown, record: SasProvider) => (
        <Dropdown menu={{ items: getOperationMenu(record) }} trigger={['click']}>
          <Button size="small" type="text" icon={<MoreOutlined />} />
        </Dropdown>
      ),
    },
    {
      title: 'Provider名称',
      dataIndex: 'providerName',
      key: 'providerName',
      width: 150,
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
      width: 280,
      render: (_: unknown, record: SasProvider) => (
        <Space>
          {record.uploadSuccess === 1 ? (
            <span style={{ color: '#52c41a' }}>●</span>
          ) : (
            <span style={{ color: '#ff4d4f' }}>●</span>
          )}
          <span>{record.certName}({record.validTimeStr})</span>
        </Space>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updateTimeStr',
      key: 'updateTimeStr',
      width: 160,
    },
  ];

  return (
    <>
      <Form form={form} layout="vertical" size="small" initialValues={{
        mainLogHbEnable: false,
      }}>
        {/* SAS */}
        <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>SAS</span>} style={{ marginBottom: 16 }}>
          <Space style={{ marginBottom: 16 }}>
            <span>SAS心跳日志</span>
            <Form.Item name="mainLogHbEnable" valuePropName="checked" noStyle>
              <Switch size="small" />
            </Form.Item>
          </Space>

          <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>SAS Provider列表</span>
            <Button type="primary" icon={<PlusOutlined />} size="small" onClick={handleAddProvider}>
              添加
            </Button>
          </div>

          <Table
            dataSource={providers}
            columns={columns}
            rowKey="id"
            pagination={false}
            size="small"
            bordered
            style={{ height: 300 }}
          />
        </Card>
      </Form>

      {/* 添加/编辑Provider弹窗 */}
      <Modal
        title={editingProvider ? 'Update SAS Provider' : 'New SAS Provider'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        okText="确定"
        cancelText="取消"
        width={600}
      >
        <Form form={providerForm} layout="vertical" initialValues={{ type: 'pem' }}>
          <Form.Item
            name="provider"
            label="SAS Provider"
            rules={[{ required: true, message: 'SAS Provider不能为空' }]}
          >
            <Input
              placeholder="请输入SAS Provider"
              maxLength={50}
              disabled={!!editingProvider}
            />
          </Form.Item>

          <Form.Item
            name="serverUrl"
            label="SAS Server URL"
            rules={[{ required: true, message: 'SAS Server URL不能为空' }]}
          >
            <Input placeholder="请输入SAS Server URL" maxLength={200} />
          </Form.Item>

          {/* TLS证书 */}
          <Card size="small" title="TLS证书" style={{ marginBottom: 16 }}>
            <Form.Item name="type" label="证书文件类型">
              <Radio.Group onChange={handleCertTypeChange}>
                <Radio value="pem">.PEM</Radio>
                <Radio value="p12">.P12</Radio>
              </Radio.Group>
            </Form.Item>

            <Form.Item label="证书文件" required={!editingProvider}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <Input
                  readOnly
                  value={certFileName}
                  placeholder="请先选择文件"
                  addonAfter={
                    <Upload
                      showUploadList={false}
                      beforeUpload={() => false}
                      onChange={handleCertFileChange}
                      accept={certType === 'pem' ? '.pem,.crt' : '.p12'}
                    >
                      <UploadOutlined style={{ cursor: 'pointer' }} />
                    </Upload>
                  }
                />
                <span style={{ color: '#999', fontSize: 12 }}>
                  {certType === 'pem' ? '支持 .pem 或 .crt 格式' : '支持 .p12 格式'}
                </span>
              </Space>
            </Form.Item>

            {certType === 'pem' && (
              <Form.Item label="私钥文件" required={!editingProvider}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  <Input
                    readOnly
                    value={privateKeyFileName}
                    placeholder="请先选择文件"
                    addonAfter={
                      <Upload
                        showUploadList={false}
                        beforeUpload={() => false}
                        onChange={handlePrivateKeyFileChange}
                        accept=".pem"
                      >
                        <UploadOutlined style={{ cursor: 'pointer' }} />
                      </Upload>
                    }
                  />
                  <span style={{ color: '#999', fontSize: 12 }}>支持 .pem 格式</span>
                </Space>
              </Form.Item>
            )}

            <Form.Item name="password" label="密码">
              <Input.Password placeholder="证书存在密码时输入" maxLength={50} />
            </Form.Item>
            <span style={{ color: '#bbb', fontSize: 12 }}>ℹ 证书存在密码时输入密码，否则无需填写</span>
          </Card>
        </Form>
      </Modal>
    </>
  );
}
