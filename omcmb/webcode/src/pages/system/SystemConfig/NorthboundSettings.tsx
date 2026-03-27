import { useState } from 'react';
import { Form, Input, Switch, Divider, Space, Table, Button, Modal, Tag, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';

interface NorthboundUser {
  id: string;
  userName: string;
  userPwd: string;
  userEnable: boolean;
  responseTime: string;
}

interface NorthboundSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// Mock 数据
const mockUsers: NorthboundUser[] = [
  { id: '1', userName: 'northuser1', userPwd: '******', userEnable: true, responseTime: '2026-01-15 10:30:00' },
  { id: '2', userName: 'northuser2', userPwd: '******', userEnable: false, responseTime: '2026-02-20 14:15:00' },
];

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
      userEnable: user.userEnable,
    });
    setModalVisible(true);
  };

  const handleDeleteUser = (id: string) => {
    setUsers(users.filter(u => u.id !== id));
    void message.success('删除成功');
  };

  const handleModalOk = () => {
    userForm.validateFields().then((values) => {
      if (editingUser) {
        setUsers(users.map(u => u.id === editingUser.id ? { ...u, ...values } : u));
        void message.success('修改成功');
      } else {
        const newUser: NorthboundUser = {
          id: Date.now().toString(),
          ...values,
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
      title: '是否启用',
      dataIndex: 'userEnable',
      key: 'userEnable',
      width: 100,
      render: (val: boolean) => (
        <Tag color={val ? 'green' : 'default'}>{val ? '启用' : '禁用'}</Tag>
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
      render: () => '******',
    },
    {
      title: '创建时间',
      dataIndex: 'responseTime',
      key: 'responseTime',
    },
    {
      title: '操作',
      key: 'operation',
      width: 120,
      render: (_: unknown, record: NorthboundUser) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => handleEditUser(record)}>
            编辑
          </Button>
          <Button size="small" danger icon={<DeleteOutlined />} onClick={() => handleDeleteUser(record.id)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Form form={form} layout="vertical" size="small" initialValues={{
        northboundIp: '192.168.1.100',
        northboundPort: 8081,
        northboundServiceStatus: 'running',
      }}>
        {/* 基本信息 */}
        <Divider orientation="left" plain>基本信息</Divider>
        <Space>
          <Form.Item name="northboundIp" label="IP地址">
            <Input disabled style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="northboundPort" label="端口">
            <Input disabled style={{ width: 100 }} />
          </Form.Item>
          <Form.Item name="northboundServiceStatus" label="服务状态">
            <Tag color="green">运行中</Tag>
          </Form.Item>
        </Space>

        {/* 用户列表 */}
        <Divider orientation="left" plain>用户列表</Divider>
        <div style={{ marginBottom: 16 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddUser}>
            添加用户
          </Button>
        </div>
        <Table
          dataSource={users}
          columns={columns}
          rowKey="id"
          pagination={false}
          size="small"
        />
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
