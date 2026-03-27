import { useState } from 'react';
import { Form, Input, Switch, Space, Table, Button, Modal, Tag, Card, message } from 'antd';
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
const settingRowStyle: React.CSSProperties = {
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

  const handleDeleteUser = (id: string) => {
    setUsers(users.filter(u => u.id !== id));
    void message.success('删除成功');
  };

  const handleToggleEnable = (user: NorthboundUser) => {
    const newEnable = user.userEnable === '1' ? '0' : '1';
    setUsers(users.map(u => u.id === user.id ? { ...u, userEnable: newEnable } : u));
    void message.success(newEnable === '1' ? '已启用' : '已禁用');
  };

  const handleModalOk = () => {
    userForm.validateFields().then((values) => {
      if (editingUser) {
        setUsers(users.map(u => u.id === editingUser.id ? {
          ...u,
          ...values,
          userEnable: values.userEnable ? '1' : '0'
        } : u));
        void message.success('修改成功');
      } else {
        const newUser: NorthboundUser = {
          id: Date.now().toString(),
          userName: values.userName,
          userPwd: values.userPwd || '******',
          userEnable: values.userEnable ? '1' : '0',
          responseTime: new Date().toLocaleString(),
        };
        setUsers([...users, newUser]);
        void message.success('添加成功');
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
      title: '是否启用',
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
      title: '用户名称',
      dataIndex: 'userName',
      key: 'userName',
    },
    {
      title: '密码',
      dataIndex: 'userPwd',
      key: 'userPwd',
    },
    {
      title: '创建时间',
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
        <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>服务信息</span>} style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 32 }}>
            <div style={infoItemStyle}>
              <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>IP地址：</span>
              <span style={{ fontWeight: 500 }}>192.168.1.100</span>
            </div>
            <div style={infoItemStyle}>
              <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>端口：</span>
              <span style={{ fontWeight: 500 }}>8081</span>
            </div>
            <div style={infoItemStyle}>
              <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>服务状态：</span>
              <Tag color="green">运行中</Tag>
            </div>
          </div>
        </Card>

        {/* 用户管理 */}
        <Card
          size="small"
          title={<span style={{ fontSize: 14, fontWeight: 600 }}>用户管理</span>}
          extra={
            <Button type="primary" icon={<PlusOutlined />} size="small" onClick={handleAddUser}>
              添加
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
        title={editingUser ? '编辑用户' : '添加用户'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        okText="确定"
        cancelText="取消"
      >
        <Form form={userForm} layout="vertical">
          <Form.Item name="userName" label="用户名称" rules={[{ required: true, message: '请输入用户名称' }]}>
            <Input placeholder="请输入用户名称" disabled={!!editingUser} />
          </Form.Item>
          <Form.Item name="userPwd" label="密码" rules={editingUser ? [] : [{ required: true, message: '请输入密码' }]}>
            <Input.Password placeholder={editingUser ? '不修改请留空' : '请输入密码'} />
          </Form.Item>
          <Form.Item name="userEnable" label="是否启用" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
