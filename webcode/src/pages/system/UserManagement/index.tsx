import { useState, useMemo } from 'react';
import {
  Button,
  Tag,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  message,
  Dropdown,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  LockOutlined,
  UnlockOutlined,
  KeyOutlined,
  MoreOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useUsers, useCreateUser, useUpdateUser, useDeleteUsers, useLockUser, useUnlockUser, useResetPassword } from '@/hooks/api/useSystem';
import type { User, UserRole, UserStatus } from '@/types/system';
import { useT } from '@/hooks/useT';

const roleColorMap: Record<UserRole, string> = {
  admin: 'red',
  operator: 'blue',
  viewer: 'default',
  auditor: 'purple',
};

const statusColorMap: Record<UserStatus, string> = {
  active: 'green',
  locked: 'red',
  inactive: 'default',
};

export default function UserManagement() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createVisible, setCreateVisible] = useState(false);
  const [editVisible, setEditVisible] = useState(false);
  const [resetPwdVisible, setResetPwdVisible] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [form] = Form.useForm();
  const [pwdForm] = Form.useForm();
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  const { data, isLoading, refetch } = useUsers({
    role: filters.role as UserRole | undefined,
    status: filters.status as UserStatus | undefined,
    keyword: filters.keyword as string | undefined,
    page,
    pageSize,
  });

  const createUser = useCreateUser();
  const updateUser = useUpdateUser();
  const deleteUsers = useDeleteUsers();
  const lockUser = useLockUser();
  const unlockUser = useUnlockUser();
  const resetPassword = useResetPassword();

  const handleCreate = () => {
    form.validateFields().then((vals) => {
      createUser.mutate(
        {
          username: vals.username as string,
          displayName: vals.displayName as string,
          email: vals.email as string,
          phone: (vals.phone as string) ?? '',
          role: vals.role as UserRole,
          status: 'active',
        },
        {
          onSuccess: () => {
            void message.success(t('common.save'));
            setCreateVisible(false);
            form.resetFields();
          },
        },
      );
    });
  };

  const handleEdit = () => {
    if (!selectedUser) return;
    form.validateFields().then((vals) => {
      updateUser.mutate(
        { id: selectedUser.id, data: { displayName: vals.displayName as string, email: vals.email as string, phone: vals.phone as string, role: vals.role as UserRole } },
        {
          onSuccess: () => {
            void message.success(t('common.save'));
            setEditVisible(false);
            form.resetFields();
            setSelectedUser(null);
          },
        },
      );
    });
  };

  const handleResetPassword = () => {
    if (!selectedUser) return;
    pwdForm.validateFields().then((vals) => {
      resetPassword.mutate(
        { id: selectedUser.id, newPassword: vals.newPassword as string },
        {
          onSuccess: () => {
            void message.success(t('common.save'));
            setResetPwdVisible(false);
            pwdForm.resetFields();
            setSelectedUser(null);
          },
        },
      );
    });
  };

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('user.username'), type: 'input', placeholder: t('user.username') },
    {
      name: 'role',
      label: t('user.role'),
      type: 'select',
      options: [
        { label: t('user.role.admin'), value: 'admin' },
        { label: t('user.role.operator'), value: 'operator' },
        { label: t('user.role.viewer'), value: 'viewer' },
        { label: t('user.role.auditor'), value: 'auditor' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.active'), value: 'active' },
        { label: t('status.disabled'), value: 'locked' },
        { label: t('status.inactive'), value: 'inactive' },
      ],
    },
  ], [t]);

  const columns: DataTableColumn<User & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'username',
      title: t('user.username'),
      dataIndex: 'username',
      width: 130,
      render: (val) => <span style={{ fontFamily: 'monospace', fontWeight: 500 }}>{String(val)}</span>,
    },
    { key: 'displayName', title: t('user.displayName'), dataIndex: 'displayName', width: 120 },
    { key: 'email', title: t('user.email'), dataIndex: 'email', ellipsis: true },
    { key: 'phone', title: t('user.phone'), dataIndex: 'phone', width: 130 },
    {
      key: 'role', title: t('user.role'), dataIndex: 'role', width: 100,
      render: (val) => {
        const r = val as UserRole;
        return <Tag color={roleColorMap[r] ?? 'default'}>{t(`user.role.${r}`)}</Tag>;
      },
    },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 90,
      render: (val) => {
        const s = val as UserStatus;
        return <Tag color={statusColorMap[s] ?? 'default'}>{s === 'active' ? t('status.active') : s === 'locked' ? t('status.disabled') : t('status.inactive')}</Tag>;
      },
    },
    {
      key: 'lastLoginTime', title: t('user.lastLogin'), dataIndex: 'lastLoginTime', width: 160,
      render: (val) => val ? new Date(String(val)).toLocaleString('zh-CN') : '—',
    },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 100, fixed: 'right',
      render: (_, record) => {
        const user = record as User;
        return (
          <Dropdown
            menu={{
              items: [
                {
                  key: 'edit',
                  label: t('common.edit'),
                  icon: <EditOutlined />,
                  onClick: () => {
                    setSelectedUser(user);
                    form.setFieldsValue({ displayName: user.displayName, email: user.email, phone: user.phone, role: user.role });
                    setEditVisible(true);
                  },
                },
                {
                  key: 'lock',
                  label: user.status === 'locked' ? t('common.enable') : t('common.disable'),
                  icon: user.status === 'locked' ? <UnlockOutlined /> : <LockOutlined />,
                  onClick: () => {
                    if (user.status === 'locked') {
                      unlockUser.mutate(user.id, { onSuccess: () => void message.success(t('common.save')) });
                    } else {
                      lockUser.mutate(user.id, { onSuccess: () => void message.success(t('common.save')) });
                    }
                  },
                },
                {
                  key: 'resetPwd',
                  label: t('user.newPassword'),
                  icon: <KeyOutlined />,
                  onClick: () => { setSelectedUser(user); setResetPwdVisible(true); },
                },
                { type: 'divider' },
                {
                  key: 'delete',
                  label: t('common.delete'),
                  icon: <DeleteOutlined />,
                  danger: true,
                  onClick: () => {
                    Modal.confirm({
                      title: t('common.confirmDelete'),
                      onOk: () => deleteUsers.mutate([user.id], { onSuccess: () => void message.success(t('common.deleteSuccess')) }),
                    });
                  },
                },
              ],
            }}
          >
            <Button size="small" icon={<MoreOutlined />}>{t('common.more')}</Button>
          </Dropdown>
        );
      },
    },
  ], [t, form, deleteUsers, lockUser, unlockUser]);

  return (
    <ListPageLayout
      title={t('nav.system.users')}
      subtitle={t('nav.system.users')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
          {t('common.add')}
        </Button>
      }
    >
      <FilterBar
        filterId="user-management-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="user-management-list"
        columns={columns}
        dataSource={(data?.list ?? []) as (User & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={(keys) => setSelectedKeys(keys)}
        batchActions={[
          {
            key: 'delete',
            label: t('common.batchDelete'),
            danger: true,
            onClick: (keys) => {
              Modal.confirm({
                title: t('common.confirmDelete'),
                onOk: () => deleteUsers.mutate(keys.map(String), {
                  onSuccess: () => { void message.success(t('common.deleteSuccess')); setSelectedKeys([]); },
                }),
              });
            },
          },
        ]}
        scroll={{ x: 1000 }}
      />

      <Modal title={t('common.add')} open={createVisible} onOk={handleCreate} onCancel={() => { setCreateVisible(false); form.resetFields(); }} confirmLoading={createUser.isPending} width={520}>
        <Form form={form} layout="vertical">
          <Form.Item name="username" label={t('user.username')} rules={[{ required: true }, { pattern: /^[a-zA-Z0-9_]{3,32}$/, message: '3-32 characters' }]}>
            <Input placeholder={t('user.username')} />
          </Form.Item>
          <Form.Item name="displayName" label={t('user.displayName')} rules={[{ required: true }]}>
            <Input placeholder={t('user.displayName')} />
          </Form.Item>
          <Form.Item name="email" label={t('user.email')} rules={[{ required: true }, { type: 'email' }]}>
            <Input placeholder={t('user.email')} />
          </Form.Item>
          <Form.Item name="phone" label={t('user.phone')}>
            <Input placeholder={t('user.phone')} />
          </Form.Item>
          <Form.Item name="role" label={t('user.role')} rules={[{ required: true }]}>
            <Select options={[
              { label: t('user.role.admin'), value: 'admin' },
              { label: t('user.role.operator'), value: 'operator' },
              { label: t('user.role.viewer'), value: 'viewer' },
              { label: t('user.role.auditor'), value: 'auditor' },
            ]} placeholder={t('common.pleaseSelect')} />
          </Form.Item>
          <Form.Item name="password" label={t('user.password')} rules={[{ required: true }, { min: 8 }]}>
            <Input.Password placeholder={t('user.password')} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('common.edit')} open={editVisible} onOk={handleEdit} onCancel={() => { setEditVisible(false); form.resetFields(); setSelectedUser(null); }} confirmLoading={updateUser.isPending} width={520}>
        <Form form={form} layout="vertical">
          <Form.Item name="displayName" label={t('user.displayName')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="email" label={t('user.email')} rules={[{ required: true }, { type: 'email' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="phone" label={t('user.phone')}>
            <Input />
          </Form.Item>
          <Form.Item name="role" label={t('user.role')} rules={[{ required: true }]}>
            <Select options={[
              { label: t('user.role.admin'), value: 'admin' },
              { label: t('user.role.operator'), value: 'operator' },
              { label: t('user.role.viewer'), value: 'viewer' },
              { label: t('user.role.auditor'), value: 'auditor' },
            ]} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={`${t('user.newPassword')} - ${selectedUser?.username ?? ''}`} open={resetPwdVisible} onOk={handleResetPassword} onCancel={() => { setResetPwdVisible(false); pwdForm.resetFields(); setSelectedUser(null); }} confirmLoading={resetPassword.isPending} width={420}>
        <Form form={pwdForm} layout="vertical">
          <Form.Item name="newPassword" label={t('user.newPassword')} rules={[{ required: true }, { min: 8 }]}>
            <Input.Password placeholder={t('user.newPassword')} />
          </Form.Item>
          <Form.Item name="confirmPassword" label={t('user.confirmPwd')} dependencies={['newPassword']}
            rules={[
              { required: true },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) return Promise.resolve();
                  return Promise.reject(new Error(t('user.confirmPwd')));
                },
              }),
            ]}>
            <Input.Password placeholder={t('user.confirmPwd')} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
