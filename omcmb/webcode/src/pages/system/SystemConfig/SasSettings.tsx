import { useState } from 'react';
import { Form, Switch, Space, Table, Button, Modal, Input, Radio, Card, Upload, Dropdown, App } from 'antd';
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
  const { modal, message } = App.useApp();
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
    modal.confirm({
      title: t('common.confirm'),
      content: t('system.sas.confirmDeleteProvider'),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: () => {
        setProviders(providers.filter(p => p.id !== id));
        message.success(t('common.deleteSuccess'));
      },
    });
  };

  const getOperationMenu = (record: SasProvider): MenuProps['items'] => [
    {
      key: 'edit',
      label: t('common.edit'),
      onClick: () => handleEditProvider(record),
    },
    {
      key: 'delete',
      label: t('common.delete'),
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
        void message.success(t('common.updateSuccess'));
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
        void message.success(t('common.addSuccess'));
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
      fixed: 'left',
      render: (_: unknown, record: SasProvider) => (
        <Dropdown
          menu={{ items: getOperationMenu(record) }}
          trigger={['click']}
          destroyPopupOnHide
        >
          <Button size="small" type="text" icon={<MoreOutlined />} />
        </Dropdown>
      ),
    },
    {
      title: t('system.sas.providerName'),
      dataIndex: 'providerName',
      key: 'providerName',
      width: 150,
      ellipsis: true,
    },
    {
      title: t('system.sas.serverUrl'),
      dataIndex: 'url',
      key: 'url',
      width: 250,
      ellipsis: true,
    },
    {
      title: t('system.sas.tlsCert'),
      key: 'cert',
      width: 260,
      ellipsis: true,
      render: (_: unknown, record: SasProvider) => (
        <Space style={{ whiteSpace: 'nowrap' }}>
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
      title: t('common.updateTime'),
      dataIndex: 'updateTimeStr',
      key: 'updateTimeStr',
      width: 160,
      ellipsis: true,
    },
  ];

  return (
    <>
      <Form form={form} layout="vertical" size="small" initialValues={{
        mainLogHbEnable: false,
      }}>
        {/* SAS */}
        <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.sas.title')}</span>} style={{ marginBottom: 16 }}>
          <Space style={{ marginBottom: 16 }}>
            <span>{t('system.sas.heartbeatLog')}</span>
            <Form.Item name="mainLogHbEnable" valuePropName="checked" noStyle>
              <Switch size="small" />
            </Form.Item>
          </Space>

          <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>{t('system.sas.providerList')}</span>
            <Button type="primary" icon={<PlusOutlined />} size="small" onClick={handleAddProvider}>
              {t('common.add')}
            </Button>
          </div>

          <Table
            dataSource={providers}
            columns={columns}
            rowKey="id"
            pagination={false}
            size="small"
            bordered
            scroll={{ x: 'max-content' }}
            style={{ height: 300 }}
          />
        </Card>
      </Form>

      {/* 添加/编辑Provider弹窗 */}
      <Modal
        title={editingProvider ? t('system.sas.editProvider') : t('system.sas.addProvider')}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={600}
      >
        <Form form={providerForm} layout="vertical" initialValues={{ type: 'pem' }}>
          <Form.Item
            name="provider"
            label="SAS Provider"
            rules={[{ required: true, message: t('system.sas.providerRequired') }]}
          >
            <Input
              placeholder={t('system.sas.pleaseInputProvider')}
              maxLength={50}
              disabled={!!editingProvider}
            />
          </Form.Item>

          <Form.Item
            name="serverUrl"
            label="SAS Server URL"
            rules={[{ required: true, message: t('system.sas.serverUrlRequired') }]}
          >
            <Input placeholder={t('system.sas.pleaseInputServerUrl')} maxLength={200} />
          </Form.Item>

          {/* TLS证书 */}
          <Card size="small" title={t('system.sas.tlsCert')} style={{ marginBottom: 16 }}>
            <Form.Item name="type" label={t('system.sas.certFileType')}>
              <Radio.Group onChange={handleCertTypeChange}>
                <Radio value="pem">.PEM</Radio>
                <Radio value="p12">.P12</Radio>
              </Radio.Group>
            </Form.Item>

            <Form.Item label={t('system.sas.certFile')} required={!editingProvider}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <Input
                  readOnly
                  value={certFileName}
                  placeholder={t('system.sas.pleaseSelectFileFirst')}
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
              <Form.Item label={t('system.sas.privateKeyFile')} required={!editingProvider}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  <Input
                    readOnly
                    value={privateKeyFileName}
                    placeholder={t('system.sas.pleaseSelectFileFirst')}
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

            <Form.Item name="password" label={t('common.password')}>
              <Input.Password placeholder={t('system.sas.inputPasswordIfCertHas')} maxLength={50} />
            </Form.Item>
            <span style={{ color: '#bbb', fontSize: 12 }}>ℹ {t('system.sas.inputPasswordIfCertHas')}</span>
          </Card>
        </Form>
      </Modal>
    </>
  );
}
