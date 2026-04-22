import { useState, useMemo, useCallback, useRef } from 'react';
import {
  App,
  Button,
  Card,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  Dropdown,
  Tooltip,
  Drawer,
  Radio,
  DatePicker,
  Space,
} from 'antd';
import type { MenuProps } from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  MoreOutlined,
  CopyOutlined,
  StopOutlined,
  CheckCircleOutlined,
  LogoutOutlined,
  KeyOutlined,
  ExportOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import ImportPanel, { type ImportPanelRef } from '@/components/ImportPanel';
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
} from '@core/hooks/api/useSystem';
import type { User } from '@core/types/system';
import { useT } from '@/hooks/useT';
import dayjs from 'dayjs';

type CreateMode = 'add' | 'import';

export default function UserManagement() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createVisible, setCreateVisible] = useState(false);
  const [createMode, setCreateMode] = useState<CreateMode>('add');
  const [editVisible, setEditVisible] = useState(false);
  const [viewVisible, setViewVisible] = useState(false);
  const [resetPwdVisible, setResetPwdVisible] = useState(false);
  const [moveGroupVisible, setMoveGroupVisible] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [form] = Form.useForm();
  const [pwdForm] = Form.useForm();
  const [moveGroupForm] = Form.useForm();
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const importPanelRef = useRef<ImportPanelRef>(null);

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

  // 检查选中的用户中是否有内置用户
  const selectedUsers = useMemo(() => {
    return (data?.items || []).filter((u) => selectedKeys.includes(u.id));
  }, [data?.items, selectedKeys]);

  const hasBuiltInSelected = useMemo(() => {
    return selectedUsers.some((u) => isBuiltIn(u));
  }, [selectedUsers, isBuiltIn]);

  const handleDelete = useCallback((user: User) => {
    if (isBuiltIn(user)) {
      modal.warning({
        title: t('common.warning'),
        content: t('user.builtInCannotDelete'),
      });
      return;
    }
    modal.confirm({
      title: t('common.confirmDelete'),
      onOk: () => {
        deleteUsers.mutate([user.id], {
          onSuccess: () => message.success(t('common.deleteSuccess')),
        });
      },
    });
  }, [isBuiltIn, t, deleteUsers, modal, message]);

  const handleBatchDelete = useCallback(() => {
    const keys = selectedKeys;
    const usersToDelete = (data?.items || []).filter(
      (u) => keys.includes(u.id) && !isBuiltIn(u)
    );
    if (usersToDelete.length === 0) {
      modal.warning({
        title: t('common.warning'),
        content: t('user.noUsersToDelete'),
      });
      return;
    }
    modal.confirm({
      title: t('common.confirmDelete'),
      content: usersToDelete.length < keys.length
        ? t('user.selectedBuiltInSkipped')
        : undefined,
      onOk: () => {
        deleteUsers.mutate(usersToDelete.map((u) => u.id), {
          onSuccess: () => {
            message.success(t('common.deleteSuccess'));
            setSelectedKeys([]);
          },
        });
      },
    });
  }, [data?.items, selectedKeys, isBuiltIn, t, deleteUsers, modal, message]);

  const handleCreate = () => {
    form.validateFields().then((vals) => {
      const userData = {
        userName: vals.userName as string,
        password: vals.password as string,
        email: vals.email as string,
        phone: vals.phone as string,
        groupNames: (vals.groupNames as string[]) || [],
        status: vals.status as string,
        expireTime: vals.expireTime?.format('YYYY-MM-DD HH:mm:ss'),
        description: (vals.description as string) ?? '',
        source: '本地',
        onlineStatus: 'offline',
        builtIn: 0,
      };
      createUser.mutate(userData, {
        onSuccess: () => {
          message.success(t('common.save'));
          setCreateVisible(false);
          form.resetFields();
        },
      });
    });
  };

  const handleImport = useCallback(async (file: File) => {
    // TODO: 实现导入API调用
    console.log('Import file:', file.name);
    // 模拟API调用
    await new Promise((resolve) => setTimeout(resolve, 1000));
    return { success: true };
  }, []);

  const handleImportSuccess = useCallback(() => {
    message.success(t('common.success'));
    setCreateVisible(false);
    void refetch();
  }, [message, t, refetch]);

  const handleDownloadTemplate = useCallback(() => {
    // TODO: 实现下载模板功能
    message.info(t('user.downloadingTemplate'));
  }, [message, t]);

  const handleEdit = () => {
    if (!selectedUser) return;
    form.validateFields().then((vals) => {
      updateUser.mutate(
        {
          id: selectedUser.id,
          data: {
            userName: vals.userName as string,
            email: vals.email as string,
            phone: vals.phone as string,
            groupNames: (vals.groupNames as string[]) || [],
            status: vals.status as string,
            expireTime: vals.expireTime?.format('YYYY-MM-DD HH:mm:ss'),
            description: vals.description as string,
          },
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
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
    pwdForm.validateFields().then(() => {
      message.success(t('common.save'));
      setResetPwdVisible(false);
      pwdForm.resetFields();
      setSelectedUser(null);
    });
  };

  const handleMoveGroup = () => {
    const targetGroupId = moveGroupForm.getFieldValue('targetGroupId') as string;
    if (!targetGroupId) {
      message.warning(t('common.pleaseSelect'));
      return;
    }
    moveUsersToGroup.mutate(
      { userIds: selectedKeys.map(String), groupId: targetGroupId },
      {
        onSuccess: () => {
          message.success(t('common.save'));
          setMoveGroupVisible(false);
          setSelectedKeys([]);
          moveGroupForm.resetFields();
        },
      }
    );
  };

  const handleBatchForceLogout = useCallback(() => {
    // 内置用户不允许操作
    if (hasBuiltInSelected) {
      modal.warning({
        title: t('common.warning'),
        content: t('user.builtInCannotBatchOp'),
      });
      return;
    }
    const onlineUsers = (data?.items || []).filter(
      (u) => selectedKeys.includes(u.id) && u.onlineStatus === 'online'
    );
    if (onlineUsers.length === 0) {
      modal.warning({
        title: t('common.warning'),
        content: t('user.noOnlineUsers'),
      });
      return;
    }
    modal.confirm({
      title: t('common.confirm'),
      content: t('user.confirmForceLogout'),
      onOk: () => {
        forceLogout.mutate(onlineUsers.map((u) => u.id), {
          onSuccess: () => {
            message.success(t('common.success'));
            setSelectedKeys([]);
          },
        });
      },
    });
  }, [data?.items, selectedKeys, hasBuiltInSelected, t, forceLogout, modal, message]);

  const handleBatchLock = useCallback(() => {
    // 内置用户不允许操作
    if (hasBuiltInSelected) {
      modal.warning({
        title: t('common.warning'),
        content: t('user.builtInCannotBatchOp'),
      });
      return;
    }
    const users = (data?.items || []).filter((u) => selectedKeys.includes(u.id));
    if (users.length === 0) return;
    modal.confirm({
      title: t('common.confirm'),
      content: t('user.confirmLock'),
      onOk: async () => {
        for (const u of users) {
          await lockUser.mutateAsync(u.id);
        }
        message.success(t('common.success'));
        setSelectedKeys([]);
      },
    });
  }, [data?.items, selectedKeys, hasBuiltInSelected, t, lockUser, modal, message]);

  const handleBatchUnlock = useCallback(() => {
    // 内置用户不允许操作
    if (hasBuiltInSelected) {
      modal.warning({
        title: t('common.warning'),
        content: t('user.builtInCannotBatchOp'),
      });
      return;
    }
    const users = (data?.items || []).filter((u) => selectedKeys.includes(u.id));
    if (users.length === 0) return;
    modal.confirm({
      title: t('common.confirm'),
      content: t('user.confirmUnlock'),
      onOk: async () => {
        for (const u of users) {
          await unlockUser.mutateAsync(u.id);
        }
        message.success(t('common.success'));
        setSelectedKeys([]);
      },
    });
  }, [data?.items, selectedKeys, hasBuiltInSelected, t, unlockUser, modal, message]);

  const handleBatchResetPassword = useCallback(() => {
    // 内置用户不允许操作
    if (hasBuiltInSelected) {
      modal.warning({
        title: t('common.warning'),
        content: t('user.builtInCannotBatchOp'),
      });
      return;
    }
    const users = (data?.items || []).filter((u) => selectedKeys.includes(u.id));
    if (users.length === 0) return;
    modal.confirm({
      title: t('common.confirm'),
      content: t('user.confirmBatchResetPassword'),
      onOk: () => {
        message.success(t('common.success'));
        setSelectedKeys([]);
      },
    });
  }, [data?.items, selectedKeys, hasBuiltInSelected, t, modal, message]);

  const handleBatchMoveGroup = useCallback(() => {
    // 内置用户不允许操作
    if (hasBuiltInSelected) {
      modal.warning({
        title: t('common.warning'),
        content: t('user.builtInCannotBatchOp'),
      });
      return;
    }
    setMoveGroupVisible(true);
  }, [hasBuiltInSelected, modal, t]);

  const handleCopy = useCallback((user: User) => {
    copyUser.mutate(user.id, {
      onSuccess: () => {
        message.success(t('common.success'));
      },
    });
  }, [copyUser, t, message]);

  const openCreateDrawer = () => {
    setCreateMode('add');
    setCreateVisible(true);
    form.resetFields();
  };

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'userName', label: t('user.userName'), type: 'input', placeholder: t('user.userName') },
  ], [t]);

  const columns: DataTableColumn<User & Record<string, unknown>>[] = useMemo(() => [
    // 操作列放在最前面
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const user = record as User;
        const isBuiltInUser = isBuiltIn(user);
        const isOnline = user.onlineStatus === 'online';
        const canEdit = !isBuiltInUser;
        const canDelete = !isBuiltInUser;
        const canChangeStatus = !isBuiltInUser;
        const canForceLogout = !isBuiltInUser && isOnline;
        const canResetPwd = !isBuiltInUser && user.source !== 'LDAP';

        const moreItems: MenuProps['items'] = [
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
                phone: user.phone,
                groupNames: user.groupNames,
                description: user.description,
                status: user.status,
                expireTime: user.expireTime ? dayjs(user.expireTime) : undefined,
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
            key: 'status',
            label: user.status === 'enabled' ? '禁用' : '启用',
            icon: user.status === 'enabled' ? <StopOutlined /> : <CheckCircleOutlined />,
            disabled: !canChangeStatus,
            onClick: () => {
              modal.confirm({
                title: t('common.confirm'),
                content: user.status === 'enabled' ? '确定要禁用该用户吗？禁用后用户将无法登录系统。' : '确定要启用该用户吗？',
                onOk: () => {
                  message.success(t('common.success'));
                  void refetch();
                },
              });
            },
          },
          {
            key: 'forceLogout',
            label: t('user.forceLogout'),
            icon: <LogoutOutlined />,
            disabled: !canForceLogout,
            onClick: () => {
              modal.confirm({
                title: t('common.confirm'),
                content: t('user.confirmForceLogout'),
                onOk: () => {
                  forceLogout.mutate([user.id], {
                    onSuccess: () => message.success(t('common.success')),
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
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small"
              onClick={() => {
                setSelectedUser(user);
                form.setFieldsValue({
                  userName: user.userName,
                  email: user.email,
                  phone: user.phone,
                  groupNames: user.groupNames,
                  description: user.description,
                });
                setViewVisible(true);
              }}>
              {t('common.view')}
            </Button>
            <Dropdown menu={{ items: moreItems }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
    },
    {
      key: 'userName',
      title: t('user.userName'),
      dataIndex: 'userName',
      width: 130,
      render: (val, record) => {
        const user = record as User;
        return (
          <span>
            <span style={{ fontFamily: 'monospace', fontWeight: 500 }}>{String(val)}</span>
            {isBuiltIn(user) && <Tag color="blue" style={{ marginLeft: 8 }}>{t('user.builtIn')}</Tag>}
          </span>
        );
      },
    },
    {
      key: 'status',
      title: t('user.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const isEnabled = val === 'enabled';
        return (
          <Tag color={isEnabled ? 'success' : 'error'}>
            {isEnabled ? '启用' : '禁用'}
          </Tag>
        );
      },
    },
    {
      key: 'onlineStatus',
      title: t('user.onlineStatus'),
      dataIndex: 'onlineStatus',
      width: 90,
      render: (val) => {
        const isOnline = val === 'online';
        return (
          <Tag color={isOnline ? 'green' : 'default'}>
            {isOnline ? t('user.online') : t('user.offline')}
          </Tag>
        );
      },
    },
    { key: 'email', title: t('user.email'), dataIndex: 'email', ellipsis: true },
    { key: 'phone', title: t('user.phone'), dataIndex: 'phone', width: 120, render: (v) => v || '-' },
    {
      key: 'groupNames',
      title: '角色',
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
    { key: 'source', title: t('user.source'), dataIndex: 'source', width: 80 },
    {
      key: 'expireTime',
      title: '过期时间',
      dataIndex: 'expireTime',
      width: 160,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '永久'),
    },
    {
      key: 'lastLoginTime',
      title: t('user.lastLoginTime'),
      dataIndex: 'lastLoginTime',
      width: 160,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '—'),
    },
    {
      key: 'createTime',
      title: t('table.createTime'),
      dataIndex: 'createTime',
      width: 160,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '—'),
    },
    {
      key: 'updateTime',
      title: t('table.updateTime'),
      dataIndex: 'updateTime',
      width: 160,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '—'),
    },
    { key: 'createUser', title: '创建人', dataIndex: 'createUser', width: 100, render: (v) => v || '-' },
    { key: 'updateUser', title: '更新人', dataIndex: 'updateUser', width: 100, render: (v) => v || '-' },
    {
      key: 'description',
      title: '备注',
      dataIndex: 'description',
      width: 150,
      ellipsis: true,
      render: (val) => {
        const desc = val as string;
        if (!desc) return '-';
        if (desc.length > 20) {
          return (
            <Tooltip title={desc}>
              <span>{desc.substring(0, 20)}...</span>
            </Tooltip>
          );
        }
        return desc;
      },
    },
  ], [t, form, isBuiltIn, isAdmin, handleDelete, handleCopy, lockUser, unlockUser, forceLogout, modal, message]);

  return (
    <ListPageLayout
      title={t('nav.system.users')}
      extra={
        <>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateDrawer} style={{ marginRight: 8 }}>
            {t('common.add')}
          </Button>
          <Button icon={<ExportOutlined />} style={{ marginRight: 8 }}>
            {t('common.export')}
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
      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
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
              disabled: hasBuiltInSelected,
            },
            {
              key: 'disable',
              label: '禁用',
              icon: <StopOutlined />,
              onClick: handleBatchLock,
              disabled: hasBuiltInSelected,
            },
            {
              key: 'enable',
              label: '启用',
              icon: <CheckCircleOutlined />,
              onClick: handleBatchUnlock,
              disabled: hasBuiltInSelected,
            },
            {
              key: 'resetPwd',
              label: t('user.resetPassword'),
              icon: <KeyOutlined />,
              onClick: handleBatchResetPassword,
              disabled: hasBuiltInSelected,
            },
            {
              key: 'moveGroup',
              label: t('user.moveGroup'),
              onClick: handleBatchMoveGroup,
              disabled: hasBuiltInSelected,
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
      </Card>

      {/* Create Drawer - 包含添加和导入两种模式 */}
      <Drawer
        title={t('common.add')}
        open={createVisible}
        onClose={() => {
          setCreateVisible(false);
          form.resetFields();
        }}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button
              style={{ marginRight: 8 }}
              onClick={() => {
                setCreateVisible(false);
                form.resetFields();
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type="primary"
              loading={createMode === 'add' ? createUser.isPending : importPanelRef.current?.loading}
              onClick={createMode === 'add' ? handleCreate : () => importPanelRef.current?.handleImport()}
            >
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        {/* 模式切换 */}
        <Radio.Group
          value={createMode}
          onChange={(e) => setCreateMode(e.target.value)}
          style={{ marginBottom: 24 }}
        >
          <Radio value="add">{t('user.addUser')}</Radio>
          <Radio value="import">{t('user.importUser')}</Radio>
        </Radio.Group>

        {/* 添加用户表单 */}
        {createMode === 'add' && (
          <Form form={form} layout="vertical">
            <Form.Item
              name="userName"
              label={t('user.userName')}
              rules={[
                { required: true, message: t('user.pleaseInputUserName') },
                { pattern: /^[a-zA-Z0-9_\-]{3,32}$/, message: t('user.userNameRule') },
              ]}
            >
              <Input placeholder={t('user.userName')} maxLength={32} />
            </Form.Item>
            <Form.Item
              name="password"
              label={t('user.password')}
              rules={[
                { required: true, message: t('user.pleaseInputPassword') },
                { min: 8, message: t('user.passwordMinLength') },
              ]}
            >
              <Input.Password placeholder={t('user.password')} maxLength={20} />
            </Form.Item>
            <Form.Item
              name="confirmPassword"
              label={t('user.confirmPassword')}
              dependencies={['password']}
              rules={[
                { required: true, message: t('user.pleaseConfirmPassword') },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue('password') === value) return Promise.resolve();
                    return Promise.reject(new Error(t('user.passwordMismatch')));
                  },
                }),
              ]}
            >
              <Input.Password placeholder={t('user.confirmPassword')} maxLength={20} />
            </Form.Item>
            <Form.Item name="status" label="状态" initialValue="enabled">
              <Radio.Group>
                <Radio value="enabled">启用</Radio>
                <Radio value="disabled">禁用</Radio>
              </Radio.Group>
            </Form.Item>
            <Form.Item
              name="email"
              label={t('user.email')}
              rules={[{ type: 'email', message: t('user.emailFormatError') }]}
            >
              <Input placeholder={t('user.email')} maxLength={50} />
            </Form.Item>
            <Form.Item
              name="phone"
              label={t('user.phone')}
              rules={[
                { pattern: /^1\d{10}$/, message: t('user.phoneFormatError') },
              ]}
            >
              <Input placeholder={t('user.phone')} maxLength={11} />
            </Form.Item>
            <Form.Item
              name="groupNames"
              label="角色"
              rules={[{ required: true, message: t('user.pleaseSelectGroup') }]}
            >
              <Select
                mode="multiple"
                placeholder={t('common.pleaseSelect')}
                options={(allGroups ?? []).map((g) => ({ label: g.groupName, value: g.groupName }))}
              />
            </Form.Item>
            <Form.Item
              name="expireTime"
              label={t('user.expireTime')}
            >
              <DatePicker
                showTime
                format="YYYY-MM-DD HH:mm:ss"
                disabledDate={(current) => current && current < dayjs().startOf('day')}
                style={{ width: '100%' }}
              />
            </Form.Item>
            <Form.Item name="description" label={t('user.description')}>
              <Input.TextArea rows={3} placeholder={t('user.description')} maxLength={500} showCount />
            </Form.Item>
          </Form>
        )}

        {/* 导入用户 */}
        {createMode === 'import' && (
          <ImportPanel
            ref={importPanelRef}
            accept=".xlsx,.xls"
            maxSizeMB={10}
            onImport={handleImport}
            onSuccess={handleImportSuccess}
            onDownloadTemplate={handleDownloadTemplate}
          />
        )}
      </Drawer>

      {/* Edit Drawer */}
      <Drawer
        title={t('common.edit')}
        open={editVisible}
        onClose={() => {
          setEditVisible(false);
          form.resetFields();
          setSelectedUser(null);
        }}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button
              style={{ marginRight: 8 }}
              onClick={() => {
                setEditVisible(false);
                form.resetFields();
                setSelectedUser(null);
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button type="primary" loading={updateUser.isPending} onClick={handleEdit}>
              {t('common.save')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="userName" label={t('user.userName')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item
            name="email"
            label={t('user.email')}
            rules={[{ type: 'email', message: t('user.emailFormatError') }]}
          >
            <Input placeholder={t('user.email')} maxLength={50} />
          </Form.Item>
          <Form.Item
            name="phone"
            label={t('user.phone')}
            rules={[
              { pattern: /^1\d{10}$/, message: t('user.phoneFormatError') },
            ]}
          >
            <Input placeholder={t('user.phone')} maxLength={11} />
          </Form.Item>
          <Form.Item
            name="groupNames"
            label="角色"
            rules={[{ required: true, message: t('user.pleaseSelectGroup') }]}
          >
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              options={(allGroups ?? []).map((g) => ({ label: g.groupName, value: g.groupName }))}
            />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Radio.Group>
              <Radio value="enabled">启用</Radio>
              <Radio value="disabled">禁用</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item
            name="expireTime"
            label={t('user.expireTime')}
          >
            <DatePicker
              showTime
              format="YYYY-MM-DD HH:mm:ss"
              disabledDate={(current) => current && current < dayjs().startOf('day')}
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item name="description" label={t('user.description')}>
            <Input.TextArea rows={3} maxLength={500} showCount />
          </Form.Item>
        </Form>
      </Drawer>

      {/* View Drawer */}
      <Drawer
        title={t('common.view')}
        open={viewVisible}
        onClose={() => {
          setViewVisible(false);
          form.resetFields();
          setSelectedUser(null);
        }}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button onClick={() => {
              setViewVisible(false);
              form.resetFields();
              setSelectedUser(null);
            }}>
              {t('common.close')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item label={t('user.onlineStatus')}>
            <Tag color={selectedUser?.onlineStatus === 'online' ? 'green' : 'default'}>
              {selectedUser?.onlineStatus === 'online' ? t('user.online') : t('user.offline')}
            </Tag>
          </Form.Item>
          <Form.Item label="状态">
            <Tag color={selectedUser?.status === 'enabled' ? 'success' : 'error'}>
              {selectedUser?.status === 'enabled' ? '启用' : '禁用'}
            </Tag>
          </Form.Item>
          <Form.Item name="userName" label={t('user.userName')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item name="email" label={t('user.email')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item label={t('user.phone')}>
            <span>{selectedUser?.phone || '-'}</span>
          </Form.Item>
          <Form.Item label={t('user.groupName')}>
            <span>{selectedUser?.groupNames?.join(', ') || '-'}</span>
          </Form.Item>
          <Form.Item label={t('user.source')}>
            <span>{selectedUser?.source || '-'}</span>
          </Form.Item>
          <Form.Item label={t('user.expireTime')}>
            <span>{selectedUser?.expireTime || t('user.noTimeLimit')}</span>
          </Form.Item>
          <Form.Item label={t('user.lastLoginTime')}>
            <span>{selectedUser?.lastLoginTime ? new Date(selectedUser.lastLoginTime).toLocaleString('zh-CN') : '-'}</span>
          </Form.Item>
          <Form.Item label="创建时间">
            <span>{selectedUser?.createTime ? new Date(selectedUser.createTime).toLocaleString('zh-CN') : '-'}</span>
          </Form.Item>
          <Form.Item label="更新时间">
            <span>{selectedUser?.updateTime ? new Date(selectedUser.updateTime).toLocaleString('zh-CN') : '-'}</span>
          </Form.Item>
          <Form.Item label="创建人">
            <span>{selectedUser?.createUser || '-'}</span>
          </Form.Item>
          <Form.Item label="更新人">
            <span>{selectedUser?.updateUser || '-'}</span>
          </Form.Item>
          <Form.Item label="备注">
            <span>{selectedUser?.description || '-'}</span>
          </Form.Item>
        </Form>
      </Drawer>

      {/* Reset Password Modal */}
      <Modal
        title={`${t('user.resetPassword')} - ${selectedUser?.userName ?? ''}`}
        open={resetPwdVisible}
        onOk={handleResetPassword}
        onCancel={() => {
          setResetPwdVisible(false);
          pwdForm.resetFields();
          setSelectedUser(null);
        }}
        width={420}
      >
        <Form form={pwdForm} layout="vertical">
          <Form.Item
            name="newPassword"
            label={t('user.newPassword')}
            rules={[
              { required: true, message: t('user.pleaseInputPassword') },
              { min: 8, message: t('user.passwordMinLength') },
            ]}
          >
            <Input.Password placeholder={t('user.newPassword')} maxLength={20} />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label={t('user.confirmPassword')}
            dependencies={['newPassword']}
            rules={[
              { required: true, message: t('user.pleaseConfirmPassword') },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) return Promise.resolve();
                  return Promise.reject(new Error(t('user.passwordMismatch')));
                },
              }),
            ]}
          >
            <Input.Password placeholder={t('user.confirmPassword')} maxLength={20} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Move Group Modal */}
      <Modal
        title={t('user.moveGroup')}
        open={moveGroupVisible}
        onOk={handleMoveGroup}
        onCancel={() => {
          setMoveGroupVisible(false);
          moveGroupForm.resetFields();
        }}
        confirmLoading={moveUsersToGroup.isPending}
        width={420}
      >
        <Form form={moveGroupForm} layout="vertical">
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
