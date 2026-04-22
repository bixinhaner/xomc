import { useState } from 'react';
import { Form, Input, Switch, Table, Button, Modal, Tag, Card, message } from 'antd';
import { PlusOutlined, MoreOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';

interface NorthboundUser {
  id: string;
  userName: string;
  userPwd: string;
  userEnable: string;
  responseTime: string;
}

interface NorthboundSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// Mock 数据
const mockUsers: NorthboundUser[] = [
  { id: '1', userName: 'northuser1', userPwd: '******', userEnable: '1', responseTime: '2026-01-15 10:30:00' },
  { id: '2', userName: 'northuser2', userPwd: '******', userEnable: '0', responseTime: '2026-02-20 14:15:00' },
];

// 设置行样式
const _settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

// 信息项样式
const infoItemStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
};

export default function NorthboundSettings({ form }: NorthboundSettingsProps) {
  const t = useT();
  const [users, setUsers] = useState<NorthboundUser[]>(mockUsers);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState<NorthboundUser | null>(null);
  const [userForm] = Form.useForm();

  const handleAddUser = () => {
    setEditingUser(null);
    userForm.resetFields();
    setModalVisible(true);
  };

  const handleEditUser = (user: NorthboundUser) => {
    setEditingUser(user);
    userForm.setFieldsValue({
      userName: user.userName,
      userPwd: '',
      userEnable: user.userEnable === '1',
    });
    setModalVisible(true);
  };

  const _handleDeleteUser = (id: string) => {
    setUsers(users.filter(u => u.id !== id));
    void message.success(t('common.deleteSuccess'));
  };

  const handleToggleEnable = (user: NorthboundUser) => {
    const newEnable = user.userEnable === '1' ? '0' : '1';
    setUsers(users.map(u => u.id === user.id ? { ...u, userEnable: newEnable } : u));
    void message.success(newEnable === '1' ? t('system.northbound.enabled') : t('system.northbound.disabled'));
  };

  const handleModalOk = () => {
    userForm.validateFields().then((values) => {
      if (editingUser) {
        setUsers(users.map(u => u.id === editingUser.id ? {
          ...u,
          ...values,
          userEnable: values.userEnable ? '1' : '0'
        } : u));
        void message.success(t('common.updateSuccess'));
      } else {
        const newUser: NorthboundUser = {
          id: Date.now().toString(),
          userName: values.userName,
          userPwd: values.userPwd || '******',
          userEnable: values.userEnable ? '1' : '0',
          responseTime: new Date().toLocaleString(),
        };
        setUsers([...users, newUser]);
        void message.success(t('common.addSuccess'));
      }
      setModalVisible(false);
    });
  };

  const columns: ColumnsType<NorthboundUser> = [
    {
      title: '',
      key: 'operation',
      width: 50,
      render: (_: unknown, record: NorthboundUser) => (
        <Button size="small" icon={<MoreOutlined />} onClick={() => handleEditUser(record)} />
      ),
    },
    {
      title: t('system.northbound.isEnabled'),
      dataIndex: 'userEnable',
      key: 'userEnable',
      width: 100,
      render: (val: string, record: NorthboundUser) => (
        <Switch
          size="small"
          checked={val === '1'}
          onChange={() => handleToggleEnable(record)}
        />
      ),
    },
    {
      title: t('system.northbound.username'),
      dataIndex: 'userName',
      key: 'userName',
    },
    {
      title: t('system.northbound.password'),
      dataIndex: 'userPwd',
      key: 'userPwd',
    },
    {
      title: t('table.createTime'),
      dataIndex: 'responseTime',
      key: 'responseTime',
    },
  ];

  return (
    <>
      <Form form={form} layout="vertical" size="small" initialValues={{
        northboundIp: '192.168.1.100',
        northboundPort: 8081,
        northboundServiceStatus: '1',
      }}>
        {/* 服务信息 */}
        <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.northbound.serviceInfo')}</span>} style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 32 }}>
            <div style={infoItemStyle}>
              <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>{t('common.ipAddress')}：</span>
              <span style={{ fontWeight: 500 }}>192.168.1.100</span>
            </div>
            <div style={infoItemStyle}>
              <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>{t('common.port')}：</span>
              <span style={{ fontWeight: 500 }}>8081</span>
            </div>
            <div style={infoItemStyle}>
              <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>{t('system.northbound.serviceStatus')}：</span>
              <Tag color="green">{t('status.running')}</Tag>
            </div>
          </div>
        </Card>

        {/* 用户管理 */}
        <Card
          size="small"
          title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.northbound.userManagement')}</span>}
          extra={
            <Button type="primary" icon={<PlusOutlined />} size="small" onClick={handleAddUser}>
              {t('common.add')}
            </Button>
          }
        >
          <Table
            dataSource={users}
            columns={columns}
            rowKey="id"
            pagination={false}
            size="small"
            bordered
          />
        </Card>
      </Form>

      {/* 添加/编辑用户弹窗 */}
      <Modal
        title={editingUser ? t('system.northbound.editUser') : t('system.northbound.addUser')}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <Form form={userForm} layout="vertical">
          <Form.Item name="userName" label={t('system.northbound.username')} rules={[{ required: true, message: t('system.northbound.pleaseInputUsername') }]}>
            <Input placeholder={t('system.northbound.pleaseInputUsername')} disabled={!!editingUser} />
          </Form.Item>
          <Form.Item name="userPwd" label={t('system.northbound.password')} rules={editingUser ? [] : [{ required: true, message: t('system.northbound.pleaseInputPassword') }]}>
            <Input.Password placeholder={editingUser ? t('system.northbound.leaveEmptyToKeep') : t('system.northbound.pleaseInputPassword')} />
          </Form.Item>
          <Form.Item name="userEnable" label={t('system.northbound.isEnabled')} valuePropName="checked">
            <Switch checkedChildren={t('common.enable')} unCheckedChildren={t('common.disable')} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
