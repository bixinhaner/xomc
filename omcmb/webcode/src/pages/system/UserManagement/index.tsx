import { useState, useMemo, useCallback } from 'react';
import {
  Button,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  Dropdown,
  message,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  MoreOutlined,
  CopyOutlined,
  LockOutlined,
  UnlockOutlined,
  LogoutOutlined,
  KeyOutlined,
  ImportOutlined,
  ExportOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useUsers,
  useCreateUser,
  useUpdateUser,
  useDeleteUsers,
  useLockUser,
  useUnlockUser,
  useForceLogout,
  useCopyUser,
  useMoveUsersToGroup,
  useAllGroups,
} from '@/hooks/api/useSystem';
import type { User } from '@/types/system';
import { useT } from '@/hooks/useT';

export default function UserManagement() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createVisible, setCreateVisible] = useState(false);
  const [editVisible, setEditVisible] = useState(false);
  const [viewVisible, setViewVisible] = useState(false);
  const [resetPwdVisible, setResetPwdVisible] = useState(false);
  const [moveGroupVisible, setMoveGroupVisible] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [form] = Form.useForm();
  const [pwdForm] = Form.useForm();
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  const { data, isLoading, refetch } = useUsers({
    userName: filters.userName as string | undefined,
    page,
    pageSize,
  });

  const { data: allGroups } = useAllGroups();

  const createUser = useCreateUser();
  const updateUser = useUpdateUser();
  const deleteUsers = useDeleteUsers();
  const lockUser = useLockUser();
  const unlockUser = useUnlockUser();
  const forceLogout = useForceLogout();
  const copyUser = useCopyUser();
  const moveUsersToGroup = useMoveUsersToGroup();

  const isBuiltIn = useCallback((user: User) => user.builtIn === 1, []);
  const isAdmin = useCallback((user: User) => user.userName === 'admin', []);

  const handleDelete = useCallback((user: User) => {
    if (isBuiltIn(user)) {
      Modal.warning({
        title: t('common.warning'),
        content: t('user.builtInCannotDelete'),
      });
      return;
    }
    Modal.confirm({
      title: t('common.confirmDelete'),
      onOk: () => {
        deleteUsers.mutate([user.id], {
          onSuccess: () => void message.success(t('common.deleteSuccess')),
        });
      },
    });
  }, [isBuiltIn, t, deleteUsers]);

  const handleBatchDelete = useCallback((keys: React.Key[]) => {
    const usersToDelete = (data?.items || []).filter(
      (u) => keys.includes(u.id) && !isBuiltIn(u)
    );
    if (usersToDelete.length === 0) {
      Modal.warning({
        title: t('common.warning'),
        content: t('user.noUsersToDelete'),
      });
      return;
    }
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: usersToDelete.length < keys.length
        ? t('user.selectedBuiltInSkipped')
        : undefined,
      onOk: () => {
        deleteUsers.mutate(usersToDelete.map((u) => u.id), {
          onSuccess: () => {
            void message.success(t('common.deleteSuccess'));
            setSelectedKeys([]);
          },
        });
      },
    });
  }, [data?.items, isBuiltIn, t, deleteUsers]);

  const handleCreate = () => {
    form.validateFields().then((vals) => {
      createUser.mutate(
        {
          userName: vals.userName as string,
          password: vals.password as string,
          email: vals.email as string,
          groupNames: (vals.groupNames as string[]) || [],
          description: (vals.description as string) ?? '',
          source: '本地',
          onlineStatus: 'offline',
          lockStatus: 0,
          builtIn: 0,
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
        {
          id: selectedUser.id,
          data: {
            userName: vals.userName as string,
            email: vals.email as string,
            groupNames: (vals.groupNames as string[]) || [],
            description: vals.description as string,
          },
        },
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
      // Reset password via API
      void message.success(t('common.save'));
      setResetPwdVisible(false);
      pwdForm.resetFields();
      setSelectedUser(null);
    });
  };

  const handleMoveGroup = () => {
    const targetGroupId = form.getFieldValue('targetGroupId') as string;
    if (!targetGroupId) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    moveUsersToGroup.mutate(
      { userIds: selectedKeys.map(String), groupId: targetGroupId },
      {
        onSuccess: () => {
          void message.success(t('common.save'));
          setMoveGroupVisible(false);
          setSelectedKeys([]);
          form.resetFields();
        },
      }
    );
  };

  const handleBatchForceLogout = useCallback(() => {
    const onlineUsers = (data?.items || []).filter(
      (u) => selectedKeys.includes(u.id) && u.onlineStatus === 'online'
    );
    if (onlineUsers.length === 0) {
      Modal.warning({
        title: t('common.warning'),
        content: t('user.noOnlineUsers'),
      });
      return;
    }
    Modal.confirm({
      title: t('common.confirm'),
      content: t('user.confirmForceLogout'),
      onOk: () => {
        forceLogout.mutate(onlineUsers.map((u) => u.id), {
          onSuccess: () => {
            void message.success(t('common.success'));
            setSelectedKeys([]);
          },
        });
      },
    });
  }, [data?.items, selectedKeys, t, forceLogout]);

  const handleBatchLock = useCallback((lockStatus: 1 | 2) => {
    const users = (data?.items || []).filter((u) => selectedKeys.includes(u.id));
    Modal.confirm({
      title: t('common.confirm'),
      content: lockStatus === 0 ? t('user.confirmUnlock') : t('user.confirmLock'),
      onOk: async () => {
        for (const u of users) {
          if (lockStatus === 0) {
            await unlockUser.mutateAsync(u.id);
          } else {
            await lockUser.mutateAsync(u.id);
          }
        }
        void message.success(t('common.success'));
        setSelectedKeys([]);
      },
    });
  }, [data?.items, selectedKeys, t, lockUser, unlockUser]);

  const handleCopy = useCallback((user: User) => {
    copyUser.mutate(user.id, {
      onSuccess: () => {
        void message.success(t('common.success'));
      },
    });
  }, [copyUser, t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'userName', label: t('user.userName'), type: 'input', placeholder: t('user.userName') },
  ], [t]);

  const columns: DataTableColumn<User & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'onlineStatus',
      title: t('user.onlineStatus'),
      dataIndex: 'onlineStatus',
      width: 100,
      render: (val) => {
        const isOnline = val === 'online';
        return (
          <Tag color={isOnline ? 'green' : 'default'}>
            {isOnline ? t('user.online') : t('user.offline')}
          </Tag>
        );
      },
    },
    {
      key: 'lockStatus',
      title: t('user.lockStatus'),
      dataIndex: 'lockStatus',
      width: 100,
      render: (val) => {
        const status = val as number;
        if (status === 0) {
          return <Tag>{t('user.unlocked')}</Tag>;
        }
        return <Tag color="warning">{t('user.locked')}</Tag>;
      },
    },
    {
      key: 'userName',
      title: t('user.userName'),
      dataIndex: 'userName',
      width: 130,
      render: (val) => (
        <span style={{ fontFamily: 'monospace', fontWeight: 500 }}>{String(val)}</span>
      ),
    },
    { key: 'email', title: t('user.email'), dataIndex: 'email', ellipsis: true },
    {
      key: 'groupNames',
      title: t('user.groupName'),
      dataIndex: 'groupNames',
      width: 150,
      render: (val) => {
        const groups = val as string[];
        if (!groups || groups.length === 0) return '—';
        if (groups.length === 1) return groups[0];
        return (
          <Tooltip title={groups.join(', ')}>
            <span>{groups[0]}...</span>
          </Tooltip>
        );
      },
    },
    {
      key: 'lastLoginTime',
      title: t('user.lastLoginTime'),
      dataIndex: 'lastLoginTime',
      width: 160,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '—'),
    },
    { key: 'source', title: t('user.source'), dataIndex: 'source', width: 80 },
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const user = record as User;
        const canEdit = !isBuiltIn(user) || isAdmin(user);
        const canDelete = !isBuiltIn(user);
        const canLock = isAdmin(user);
        const canResetPwd = user.source !== 'LDAP' && (isAdmin(user));
        const isOnline = user.onlineStatus === 'online';

        return (
          <Dropdown
            menu={{
              items: [
                {
                  key: 'view',
                  label: t('common.view'),
                  icon: <EyeOutlined />,
                  onClick: () => {
                    setSelectedUser(user);
                    form.setFieldsValue({
                      userName: user.userName,
                      email: user.email,
                      groupNames: user.groupNames,
                      description: user.description,
                    });
                    setViewVisible(true);
                  },
                },
                {
                  key: 'edit',
                  label: t('common.edit'),
                  icon: <EditOutlined />,
                  disabled: !canEdit,
                  onClick: () => {
                    setSelectedUser(user);
                    form.setFieldsValue({
                      userName: user.userName,
                      email: user.email,
                      groupNames: user.groupNames,
                      description: user.description,
                    });
                    setEditVisible(true);
                  },
                },
                {
                  key: 'copy',
                  label: t('user.copy'),
                  icon: <CopyOutlined />,
                  onClick: () => handleCopy(user),
                },
                { type: 'divider' },
                {
                  key: 'lock',
                  label: user.lockStatus === 0 ? t('user.lock') : t('user.unlock'),
                  icon: user.lockStatus === 0 ? <LockOutlined /> : <UnlockOutlined />,
                  disabled: !canLock,
                  onClick: () => {
                    if (user.lockStatus === 0) {
                      lockUser.mutate(user.id, { onSuccess: () => void message.success(t('common.success')) });
                    } else {
                      unlockUser.mutate(user.id, { onSuccess: () => void message.success(t('common.success')) });
                    }
                  },
                },
                {
                  key: 'forceLogout',
                  label: t('user.forceLogout'),
                  icon: <LogoutOutlined />,
                  disabled: !canLock || !isOnline,
                  onClick: () => {
                    Modal.confirm({
                      title: t('common.confirm'),
                      content: t('user.confirmForceLogout'),
                      onOk: () => {
                        forceLogout.mutate([user.id], {
                          onSuccess: () => void message.success(t('common.success')),
                        });
                      },
                    });
                  },
                },
                {
                  key: 'resetPwd',
                  label: t('user.resetPassword'),
                  icon: <KeyOutlined />,
                  disabled: !canResetPwd,
                  onClick: () => {
                    setSelectedUser(user);
                    setResetPwdVisible(true);
                  },
                },
                { type: 'divider' },
                {
                  key: 'delete',
                  label: t('common.delete'),
                  icon: <DeleteOutlined />,
                  danger: true,
                  disabled: !canDelete,
                  onClick: () => handleDelete(user),
                },
              ],
            }}
          >
            <Button size="small" icon={<MoreOutlined />}>{t('common.more')}</Button>
          </Dropdown>
        );
      },
    },
  ], [t, form, isBuiltIn, isAdmin, handleDelete, handleCopy, lockUser, unlockUser, forceLogout]);

  return (
    <ListPageLayout
      title={t('nav.system.users')}
      subtitle={t('nav.system.users')}
      extra={
        <>
          <Button icon={<ImportOutlined />} style={{ marginRight: 8 }}>
            {t('common.import')}
          </Button>
          <Button icon={<ExportOutlined />} style={{ marginRight: 8 }}>
            {t('common.export')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
            {t('common.add')}
          </Button>
        </>
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
        dataSource={(data?.items ?? []) as (User & Record<string, unknown>)[]}
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
            key: 'forceLogout',
            label: t('user.forceLogout'),
            icon: <LogoutOutlined />,
            onClick: handleBatchForceLogout,
          },
          {
            key: 'lock',
            label: t('user.lock'),
            icon: <LockOutlined />,
            onClick: () => handleBatchLock(1),
          },
          {
            key: 'unlock',
            label: t('user.unlock'),
            icon: <UnlockOutlined />,
            onClick: () => handleBatchLock(0),
          },
          {
            key: 'moveGroup',
            label: t('user.moveGroup'),
            onClick: () => setMoveGroupVisible(true),
          },
          {
            key: 'delete',
            label: t('common.batchDelete'),
            danger: true,
            onClick: handleBatchDelete,
          },
        ]}
        scroll={{ x: 1200 }}
      />

      {/* Create Modal */}
      <Modal
        title={t('common.add')}
        open={createVisible}
        onOk={handleCreate}
        onCancel={() => { setCreateVisible(false); form.resetFields(); }}
        confirmLoading={createUser.isPending}
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="userName"
            label={t('user.userName')}
            rules={[{ required: true }, { pattern: /^[a-zA-Z0-9_]{3,32}$/, message: '3-32 characters' }]}
          >
            <Input placeholder={t('user.userName')} />
          </Form.Item>
          <Form.Item name="password" label={t('user.password')} rules={[{ required: true }, { min: 8 }]}>
            <Input.Password placeholder={t('user.password')} />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label={t('user.confirmPassword')}
            dependencies={['password']}
            rules={[
              { required: true },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) return Promise.resolve();
                  return Promise.reject(new Error(t('user.passwordMismatch')));
                },
              }),
            ]}
          >
            <Input.Password placeholder={t('user.confirmPassword')} />
          </Form.Item>
          <Form.Item name="email" label={t('user.email')} rules={[{ type: 'email' }]}>
            <Input placeholder={t('user.email')} />
          </Form.Item>
          <Form.Item name="groupNames" label={t('user.groupName')}>
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              options={(allGroups ?? []).map((g) => ({ label: g.groupName, value: g.groupName }))}
            />
          </Form.Item>
          <Form.Item name="description" label={t('user.description')}>
            <Input.TextArea rows={2} placeholder={t('user.description')} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Edit Modal */}
      <Modal
        title={t('common.edit')}
        open={editVisible}
        onOk={handleEdit}
        onCancel={() => { setEditVisible(false); form.resetFields(); setSelectedUser(null); }}
        confirmLoading={updateUser.isPending}
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="userName" label={t('user.userName')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item name="email" label={t('user.email')} rules={[{ type: 'email' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="groupNames" label={t('user.groupName')}>
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              options={(allGroups ?? []).map((g) => ({ label: g.groupName, value: g.groupName }))}
            />
          </Form.Item>
          <Form.Item name="description" label={t('user.description')}>
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      {/* View Modal */}
      <Modal
        title={t('common.view')}
        open={viewVisible}
        onCancel={() => { setViewVisible(false); form.resetFields(); setSelectedUser(null); }}
        footer={<Button onClick={() => { setViewVisible(false); form.resetFields(); setSelectedUser(null); }}>{t('common.close')}</Button>}
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item label={t('user.onlineStatus')}>
            <Tag color={selectedUser?.onlineStatus === 'online' ? 'green' : 'default'}>
              {selectedUser?.onlineStatus === 'online' ? t('user.online') : t('user.offline')}
            </Tag>
          </Form.Item>
          <Form.Item label={t('user.lockStatus')}>
            <Tag color={selectedUser?.lockStatus === 0 ? undefined : 'warning'}>
              {selectedUser?.lockStatus === 0 ? t('user.unlocked') : t('user.locked')}
            </Tag>
          </Form.Item>
          <Form.Item name="userName" label={t('user.userName')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item name="email" label={t('user.email')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item label={t('user.groupName')}>
            <span>{selectedUser?.groupNames?.join(', ') || '-'}</span>
          </Form.Item>
          <Form.Item label={t('user.source')}>
            <span>{selectedUser?.source || '-'}</span>
          </Form.Item>
          <Form.Item label={t('user.lastLoginTime')}>
            <span>{selectedUser?.lastLoginTime ? new Date(selectedUser.lastLoginTime).toLocaleString('zh-CN') : '-'}</span>
          </Form.Item>
          <Form.Item name="description" label={t('user.description')}>
            <Input.TextArea rows={2} readOnly />
          </Form.Item>
        </Form>
      </Modal>

      {/* Reset Password Modal */}
      <Modal
        title={`${t('user.resetPassword')} - ${selectedUser?.userName ?? ''}`}
        open={resetPwdVisible}
        onOk={handleResetPassword}
        onCancel={() => { setResetPwdVisible(false); pwdForm.resetFields(); setSelectedUser(null); }}
        width={420}
      >
        <Form form={pwdForm} layout="vertical">
          <Form.Item name="newPassword" label={t('user.newPassword')} rules={[{ required: true }, { min: 8 }]}>
            <Input.Password placeholder={t('user.newPassword')} />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label={t('user.confirmPassword')}
            dependencies={['newPassword']}
            rules={[
              { required: true },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) return Promise.resolve();
                  return Promise.reject(new Error(t('user.passwordMismatch')));
                },
              }),
            ]}
          >
            <Input.Password placeholder={t('user.confirmPassword')} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Move Group Modal */}
      <Modal
        title={t('user.moveGroup')}
        open={moveGroupVisible}
        onOk={handleMoveGroup}
        onCancel={() => { setMoveGroupVisible(false); form.resetFields(); }}
        confirmLoading={moveUsersToGroup.isPending}
        width={420}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="targetGroupId" label={t('user.targetGroup')} rules={[{ required: true }]}>
            <Select
              placeholder={t('common.pleaseSelect')}
              options={(allGroups ?? []).map((g) => ({ label: g.groupName, value: g.id }))}
            />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
