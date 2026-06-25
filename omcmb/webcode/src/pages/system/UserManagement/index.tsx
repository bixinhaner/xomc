import { useState, useMemo, useCallback } from 'react';
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
  Switch,
} from 'antd';
import type { MenuProps } from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
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
import dayjs from 'dayjs';
import {
  useUsers,
  useCreateUser,
  useUpdateUser,
  useDeleteUsers,
  useLockUser,
  useUnlockUser,
  useForceLogout,
  useCopyUser,
  useAllRoles,
  useResetPassword,
  useBatchAssignRoles,
} from '@core/hooks/api/useSystem';
import { useSecuritySettings } from '@core/hooks/api/useSecuritySettings';
import type { User, UserRole, UserStatus } from '@core/types/system';
import { isBuiltInUser, isLdapUser } from '@core/types/system';
import { useT } from '@/hooks/useT';
import { toast } from '@/utils/toast';
import { formatSystemTime } from '@core/utils/systemTime';

export default function UserManagement() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createVisible, setCreateVisible] = useState(false);
  const [editVisible, setEditVisible] = useState(false);
  const [viewVisible, setViewVisible] = useState(false);
  const [resetPwdVisible, setResetPwdVisible] = useState(false);
  const [moveGroupVisible, setMoveGroupVisible] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  // Issue #649：「使用系统默认密码」开关用 React state 控制，靠父组件 re-render
  // 驱动密码 Form.Item rules / disabled / placeholder 切换。曾试过 Form.Item
  // noStyle + shouldUpdate render-prop，但在 Drawer / Modal 内嵌 Form 场景下
  // shouldUpdate 不能可靠触发 children 重渲染（bundle 已部署但 disabled / rules
  // 仍是初值），故改走最朴素的父组件 useState。
  const [createUseDefault, setCreateUseDefault] = useState(false);
  const [resetUseDefault, setResetUseDefault] = useState(false);
  const [form] = Form.useForm();
  const [pwdForm] = Form.useForm();
  const [moveGroupForm] = Form.useForm();
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  const { data, isLoading, refetch } = useUsers({
    userName: filters.userName as string | undefined,
    page,
    pageSize,
  });

  const { data: allRoles } = useAllRoles();

  // 创建人 / 更新人列：后端 ListUsers 反查 users.username 注入 creator_username /
  // updater_username（参 omcgo/internal/admin/service.go ListUsers），前端直接读。
  // 历史的 useAllUsers 全量映射方案已下线，避免重复请求 /admin/users。
  // PRD §11.9 v0.8：空值显示"内置"（覆盖 builtIn / LDAP / 历史三种无 operator 场景）。
  const renderOperator = useCallback((username: unknown) => {
    if (!username) return t('user.builtIn');
    return String(username);
  }, [t]);

  // PRD §11.7 决议 ①：未绑定任何设备分组的角色，下拉 option 追加 ⚠️ 标记，
  // 防止管理员误以为"分配了角色就能看到设备"。
  //
  // 与 RolePermission 列表保持一致（同一判据 + 同一文案 role.noGroupBinding.tag）：
  // 内置角色（admin 等，builtIn=1/2）有隐含的全设备访问权，不算"未绑定设备分组"，
  // 故内置角色不打该标记 —— 否则会出现"用户页 admin 提示无设备权限、角色页 admin
  // 却不提示"的两处判断不一致问题。builtIn 由 mapBackendRole(role.is_system) 填充。
  const roleOptions = useMemo(() =>
    (allRoles ?? []).map((r) => {
      const isBuiltInRole = r.builtIn === 1 || r.builtIn === 2;
      const noGroups = !isBuiltInRole && (!r.deviceGroupIds || r.deviceGroupIds.length === 0);
      return {
        value: r.id,
        label: noGroups ? (
          <span>
            {r.roleName}
            <Tag color="warning" style={{ marginLeft: 4 }}>{t('role.noGroupBinding.tag')}</Tag>
          </span>
        ) : (
          r.roleName
        ),
      };
    }), [allRoles, t]);

  const createUser = useCreateUser();
  const updateUser = useUpdateUser();
  const deleteUsers = useDeleteUsers();
  const lockUser = useLockUser();
  const unlockUser = useUnlockUser();
  const forceLogout = useForceLogout();
  const copyUser = useCopyUser();
  const batchAssignRoles = useBatchAssignRoles();
  const resetPassword = useResetPassword();

  // Issue #649：拉安全设置取 defaultPasswd，决定「使用系统默认密码」开关
  // 是否可用 + 占位文本。useSecuritySettings 走 30s 缓存（多组件共用）。
  const { settings: securitySettings, refetch: refetchSecurity } = useSecuritySettings();
  const defaultPasswd = securitySettings?.raw.get('defaultPasswd') ?? '';
  const hasDefaultPasswd = defaultPasswd !== '';

  // Issue #649：useDefaultPassword 联动改走 createUseDefault / resetUseDefault
  // 两个父组件 useState，Switch 受控 + 父组件 re-render 自然驱动密码字段 props 刷新。

  // 内置用户判定：后端 users.source === 'builtIn'（迁移 000053 / PRD §11.3）。
  const isBuiltIn = useCallback((user: User) => isBuiltInUser(user), []);

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
      // 与后端 admin.CreateUserRequest 对齐，仅传后端实际接收的字段。
      // role 用占位值满足 hook 类型；真实角色分配走 roleIds（→ 后端 role_ids）。
      const roleIds = (vals.roleIds as string[]) ?? [];
      const expire = vals.expireTime as dayjs.Dayjs | undefined;
      // Issue #649：开启「使用系统默认密码」时不传 password，由后端从
      // sys_configs.security.defaultPasswd 取值；硬规则要求 must_change_password=true。
      const useDefault = createUseDefault;
      const userData: Omit<User, 'id' | 'createTime' | 'lastLoginTime'> & {
        password?: string;
        useDefaultPassword?: boolean;
        roleIds?: string[];
      } = {
        username: vals.username as string,
        displayName: ((vals.displayName as string) || (vals.username as string)) ?? '',
        email: (vals.email as string) || '',
        phone: (vals.phone as string) || undefined,
        description: (vals.description as string) || undefined,
        expireTime: expire ? expire.toISOString() : undefined,
        role: 'viewer' as UserRole,
        status: (vals.status as UserStatus) ?? 'active',
        roleIds: roleIds.length > 0 ? roleIds : undefined,
        ...(useDefault
          ? { useDefaultPassword: true }
          : { password: vals.password as string }),
      };
      createUser.mutate(userData, {
        onSuccess: () => {
          toast.success(t('common.save'));
          setCreateVisible(false);
          setCreateUseDefault(false);
          form.resetFields();
        },
        onError: (err) => toast.error(err, t('user.createFailed')),
      });
    });
  };

  const handleEdit = () => {
    if (!selectedUser) return;
    form.validateFields().then((vals) => {
      // 仅传后端 UpdateUserRequest 接收的字段：display_name / email / phone / status / role_ids（v1.0：carrier 已删除）。
      // role_ids 非 undefined 时由后端做差量同步。
      const expire = vals.expireTime as dayjs.Dayjs | undefined;
      const data: Partial<User> = {
        displayName: (vals.displayName as string) || selectedUser.displayName,
        email: (vals.email as string) || '',
        phone: (vals.phone as string) || undefined,
        description: (vals.description as string) ?? '',
        expireTime: expire ? expire.toISOString() : undefined,
        status: vals.status as UserStatus,
        roleIds: (vals.roleIds as string[]) ?? [],
      };
      updateUser.mutate(
        { id: selectedUser.id, data },
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
    pwdForm.validateFields().then((vals) => {
      // Issue #649：根据 Switch 分两路 — ON 走 {useDefaultPassword:true}，
      // OFF 走 {newPassword}。后端会调 revoker.Revoke 立即吊销旧 token。
      const useDefault = resetUseDefault;
      const payload = useDefault
        ? { id: selectedUser.id, useDefaultPassword: true }
        : { id: selectedUser.id, newPassword: vals.newPassword as string };
      resetPassword.mutate(payload, {
        onSuccess: () => {
          message.success(t('common.save'));
          setResetPwdVisible(false);
          setResetUseDefault(false);
          pwdForm.resetFields();
          setSelectedUser(null);
        },
        onError: (err) => toast.error(err, t('common.error')),
      });
    });
  };

  // PRD §5.5 / §11.5：批量分配角色，整体替换语义。
  const handleMoveGroup = () => {
    const targetRoleIds = moveGroupForm.getFieldValue('targetRoleIds') as string[] | undefined;
    if (!targetRoleIds || targetRoleIds.length === 0) {
      message.warning(t('common.pleaseSelect'));
      return;
    }
    batchAssignRoles.mutate(
      { userIds: selectedKeys.map(String), roleIds: targetRoleIds },
      {
        onSuccess: () => {
          message.success(t('common.save'));
          setMoveGroupVisible(false);
          setSelectedKeys([]);
          moveGroupForm.resetFields();
        },
      },
    );
  };

  const handleBatchForceLogout = useCallback(() => {
    if (hasBuiltInSelected) {
      modal.warning({
        title: t('common.warning'),
        content: t('user.builtInCannotBatchOp'),
      });
      return;
    }
    // 后端 ForceLogout 通过 Redis 撤销 token，不依赖在线状态——选中即可下线。
    const targets = (data?.items || []).filter((u) => selectedKeys.includes(u.id));
    if (targets.length === 0) return;
    modal.confirm({
      title: t('common.confirm'),
      content: t('user.confirmForceLogout'),
      onOk: () => {
        forceLogout.mutate(targets.map((u) => u.id), {
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
      onSuccess: (result) => {
        modal.success({
          title: t('common.success'),
          content: t('user.copySuccessContent', { username: result.user.username, tempPassword: result.tempPassword }),
        });
      },
    });
  }, [copyUser, t, modal]);

  const openCreateDrawer = () => {
    setCreateUseDefault(false);
    setCreateVisible(true);
    form.resetFields();
  };

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'userName', label: t('user.userName'), type: 'input', placeholder: t('user.userName'), width: 240 },
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
        // PRD §4.1：内置（builtIn）禁止删除/禁用；LDAP 禁止重置密码。
        // 编辑 / 强制下线 在内置用户上 v0.2 起已放开（紧急通道）。
        // Issue #649：内置用户改密走 omcctl CLI，UI 改密一律拦截（不接默认密码也不开手动输入）。
        const builtIn = isBuiltIn(user);
        const ldap = isLdapUser(user);
        const canEdit = true;
        const canDelete = !builtIn;
        const canChangeStatus = !builtIn;
        const canForceLogout = true;
        const canResetPwd = !ldap && !builtIn;

        const moreItems: MenuProps['items'] = [
          {
            key: 'edit',
            label: t('common.edit'),
            icon: <EditOutlined />,
            disabled: !canEdit,
            onClick: () => {
              setSelectedUser(user);
              form.setFieldsValue({
                username: user.username,
                displayName: user.displayName,
                email: user.email,
                phone: user.phone,
                status: user.status,
                roleIds: user.roleIds ?? [],
                expireTime: user.expireTime ? dayjs(user.expireTime) : undefined,
                description: user.description ?? '',
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
            label: !canChangeStatus ? (
              <Tooltip title={t('user.tooltip.builtinNoDisable')} placement="left">
                <span>{user.status === 'active' ? t('common.disable') : t('common.enable')}</span>
              </Tooltip>
            ) : (
              user.status === 'active' ? t('common.disable') : t('common.enable')
            ),
            icon: user.status === 'active' ? <StopOutlined /> : <CheckCircleOutlined />,
            disabled: !canChangeStatus,
            onClick: () => {
              const targetStatus: UserStatus = user.status === 'active' ? 'disabled' : 'active';
              modal.confirm({
                title: t('common.confirm'),
                content: user.status === 'active' ? t('user.confirmDisableUser') : t('user.confirmEnableUser'),
                onOk: () => {
                  updateUser.mutate(
                    { id: user.id, data: { status: targetStatus } },
                    {
                      onSuccess: () => message.success(t('common.success')),
                    },
                  );
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
            label: !canResetPwd ? (
              <Tooltip
                title={
                  builtIn
                    ? t('system.user.builtInResetDisabledTip')
                    : t('user.tooltip.ldapNoReset')
                }
                placement="left"
              >
                <span>{t('user.resetPassword')}</span>
              </Tooltip>
            ) : (
              t('user.resetPassword')
            ),
            icon: <KeyOutlined />,
            disabled: !canResetPwd,
            onClick: () => {
              setSelectedUser(user);
              // Issue #649：弹窗打开前 refetch 一次安全设置，避免 30s 缓存窗口内
              // 默认密码刚改完拿到旧值。
              void refetchSecurity();
              // Issue #649：Switch 初值依赖当前 hasDefaultPasswd（有默认密码就默认
              // 走「重置为默认」一键路径）。
              setResetUseDefault(hasDefaultPasswd);
              setResetPwdVisible(true);
            },
          },
          { type: 'divider' },
          {
            key: 'delete',
            label: !canDelete ? (
              <Tooltip title={t('user.tooltip.builtinNoDelete')} placement="left">
                <span>{t('common.delete')}</span>
              </Tooltip>
            ) : (
              t('common.delete')
            ),
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
                  username: user.username,
                  email: user.email,
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
      key: 'displayName',
      title: t('user.form.displayName'),
      dataIndex: 'displayName',
      width: 150,
      // PRD §11.8 v0.7：不再追加"内置"Tag；内置/管理员/LDAP 来源由"来源"列承担。
      render: (val, record) => {
        const user = record as User;
        const text = (val as string) || user.username;
        return <span style={{ fontWeight: 500 }}>{text}</span>;
      },
    },
    {
      key: 'username',
      title: t('user.account'),
      dataIndex: 'username',
      width: 130,
      render: (val) => (
        <span style={{ fontFamily: 'monospace' }}>{String(val ?? '-')}</span>
      ),
    },
    {
      key: 'status',
      title: t('user.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const isActive = val === 'active';
        return (
          <Tag color={isActive ? 'success' : 'error'}>
            {isActive ? t('user.form.statusActive') : t('user.form.statusDisabled')}
          </Tag>
        );
      },
    },
    { key: 'email', title: t('user.email'), dataIndex: 'email', ellipsis: true },
    { key: 'phone', title: t('user.phone'), dataIndex: 'phone', width: 120, render: (v) => (v as string) || '-' },
    {
      key: 'roles',
      title: t('user.role'),
      dataIndex: 'roles',
      width: 150,
      render: (val) => {
        const roles = (val as string[]) ?? [];
        if (roles.length === 0) return '—';
        if (roles.length === 1) return roles[0];
        return (
          <Tooltip title={roles.join(', ')}>
            <span>{roles[0]}...</span>
          </Tooltip>
        );
      },
    },
    {
      key: 'source',
      title: t('user.source'),
      dataIndex: 'source',
      width: 110,
      render: (val) => {
        // PRD §3.1：source 列文本映射 + 颜色。
        switch (val) {
          case 'builtIn':
            return <Tag color="blue">{t('user.source.builtIn')}</Tag>;
          case 'LDAP':
            return <Tag color="purple">LDAP</Tag>;
          case 'admin':
            return <Tag>{t('user.source.admin')}</Tag>;
          default:
            return '—';
        }
      },
    },
    {
      key: 'expireTime',
      title: t('user.expireTime'),
      dataIndex: 'expireTime',
      width: 160,
      render: (val) => (val ? formatSystemTime(String(val)) : t('user.permanent')) as string,
    },
    {
      key: 'lastLoginTime',
      title: t('user.lastLoginTime'),
      dataIndex: 'lastLoginTime',
      width: 160,
      render: (val) => (val ? formatSystemTime(String(val)) : '—') as string,
    },
    {
      key: 'createTime',
      title: t('table.createTime'),
      dataIndex: 'createTime',
      width: 160,
      render: (val) => (val ? formatSystemTime(String(val)) : '—'),
    },
    {
      key: 'updateTime',
      title: t('table.updateTime'),
      dataIndex: 'updateTime',
      width: 160,
      render: (val) => (val ? formatSystemTime(String(val)) : '—'),
    },
    {
      key: 'creatorUsername',
      title: t('user.form.createdBy'),
      dataIndex: 'creatorUsername',
      width: 110,
      render: renderOperator,
    },
    {
      key: 'updaterUsername',
      title: t('user.form.updatedBy'),
      dataIndex: 'updaterUsername',
      width: 110,
      render: renderOperator,
    },
    {
      key: 'description',
      title: t('common.remark'),
      dataIndex: 'description',
      width: 160,
      ellipsis: true,
      render: (val) => {
        const desc = val as string | undefined;
        if (!desc) return '-';
        if (desc.length > 20) {
          return (
            <Tooltip title={desc}>
              <span>{desc.slice(0, 20)}…</span>
            </Tooltip>
          );
        }
        return desc;
      },
    },
  ], [t, form, isBuiltIn, handleDelete, handleCopy, forceLogout, updateUser, modal, message, renderOperator]);

  return (
    <ListPageLayout>
      {/* 2026-06-03 用户决策:去"用户管理"标题;筛选条件与操作按钮(新增/导出)同一行,
          筛选靠左、按钮两端对齐推到最右(与产品中心/回收站范式一致)。 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 12 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <FilterBar
            filterId="user-management-filter"
            fields={filterFields}
            onSearch={(vals) => { setFilters(vals); setPage(1); }}
            onReset={() => { setFilters({}); setPage(1); }}
          />
        </div>
        <Space style={{ flexShrink: 0 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateDrawer}>
            {t('common.add')}
          </Button>
          <Button icon={<ExportOutlined />}>
            {t('common.export')}
          </Button>
        </Space>
      </div>
      <Card
        size="small"
        variant="outlined"
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
              label: t('common.disable'),
              icon: <StopOutlined />,
              onClick: handleBatchLock,
              disabled: hasBuiltInSelected,
            },
            {
              key: 'enable',
              label: t('common.enable'),
              icon: <CheckCircleOutlined />,
              onClick: handleBatchUnlock,
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

      {/* 添加用户 Drawer (PRD §5.1 / §11.12)：v1.1 起取消"导入用户"模式切换；
          字段顺序与 §5.2 编辑表单完全对齐（创建独有的 password / confirmPassword 紧随 username 之后）。 */}
      <Drawer
        title={t('user.modal.add')}
        open={createVisible}
        onClose={() => {
          setCreateVisible(false);
          setCreateUseDefault(false);
          form.resetFields();
        }}
        size={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button
              style={{ marginRight: 8 }}
              onClick={() => {
                setCreateVisible(false);
                setCreateUseDefault(false);
                form.resetFields();
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type="primary"
              loading={createUser.isPending}
              onClick={handleCreate}
            >
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        {/* 添加用户表单 (PRD §5.1 / §11.12)：v1.1 起字段顺序与编辑表单同步，
            标签统一为"用户账号 / 用户昵称"。 */}
        <Form form={form} layout="vertical">
            <Form.Item
              name="username"
              label={t('user.form.username')}
              rules={[
                { required: true, message: t('user.pleaseInputUserName') },
                { pattern: /^[a-zA-Z0-9_-]{3,32}$/, message: t('user.userNameRule') },
              ]}
            >
              <Input placeholder={t('user.form.username')} maxLength={32} />
            </Form.Item>
            {/* Issue #649：使用系统默认密码开关。默认关；ON 时下方两个密码框 disabled
                + 不校验 rules；defaultPasswd 为空时开关 disabled + tooltip 引导。
                Switch 用 React useState 控制（不放进 Form.Item.name），靠父组件
                re-render 驱动下方密码 Form.Item rules / disabled / placeholder 切换。 */}
            <Form.Item
              label={t('system.user.useDefaultPassword')}
              extra={
                hasDefaultPasswd
                  ? undefined
                  : t('system.user.defaultPasswordNotSet')
              }
            >
              <Tooltip
                title={
                  hasDefaultPasswd
                    ? undefined
                    : t('system.user.defaultPasswordNotSet')
                }
                placement="right"
              >
                <Switch
                  checked={createUseDefault}
                  onChange={setCreateUseDefault}
                  disabled={!hasDefaultPasswd}
                />
              </Tooltip>
            </Form.Item>
            <Form.Item
              name="password"
              label={t('user.password')}
              rules={
                createUseDefault
                  ? []
                  : [
                      { required: true, message: t('user.pleaseInputPassword') },
                      { min: 8, message: t('user.passwordMinLength') },
                    ]
              }
            >
              <Input.Password
                placeholder={
                  createUseDefault
                    ? defaultPasswd || t('user.password')
                    : t('user.password')
                }
                maxLength={20}
                disabled={createUseDefault}
              />
            </Form.Item>
            <Form.Item
              name="confirmPassword"
              label={t('user.confirmPassword')}
              dependencies={['password']}
              rules={
                createUseDefault
                  ? []
                  : [
                      { required: true, message: t('user.pleaseConfirmPassword') },
                      ({ getFieldValue: gfv }) => ({
                        validator(_, value) {
                          if (!value || gfv('password') === value) return Promise.resolve();
                          return Promise.reject(new Error(t('user.passwordMismatch')));
                        },
                      }),
                    ]
              }
            >
              <Input.Password
                placeholder={
                  createUseDefault
                    ? defaultPasswd || t('user.confirmPassword')
                    : t('user.confirmPassword')
                }
                maxLength={20}
                disabled={createUseDefault}
              />
            </Form.Item>
            <Form.Item name="displayName" label={t('user.form.displayName')}>
              <Input placeholder={t('user.form.displayNamePlaceholder')} maxLength={64} />
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
              name="roleIds"
              label={t('user.form.role')}
              rules={[{ required: true, message: t('user.pleaseSelectGroup') }]}
            >
              <Select
                mode="multiple"
                placeholder={t('common.pleaseSelect')}
                options={roleOptions}
              />
            </Form.Item>
            <Form.Item name="status" label={t('user.form.status')} initialValue="active">
              <Radio.Group>
                <Radio value="active">{t('user.form.statusActive')}</Radio>
                <Radio value="disabled">{t('user.form.statusDisabled')}</Radio>
              </Radio.Group>
            </Form.Item>
            <Form.Item name="expireTime" label={t('user.form.expireTime')} extra={t('user.form.expireExtra')}>
              <DatePicker
                showTime
                format="YYYY-MM-DD HH:mm:ss"
                disabledDate={(current) => current && current.isBefore(dayjs().startOf('day'))}
                style={{ width: '100%' }}
              />
            </Form.Item>
            <Form.Item name="description" label={t('user.form.description')}>
              <Input.TextArea rows={3} placeholder={t('user.form.descriptionPlaceholder')} maxLength={500} showCount />
            </Form.Item>
          </Form>
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
        size={520}
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
          {/* 编辑表单字段与后端 admin.UpdateUserRequest 对齐：
              display_name / email / phone / status / role_ids（v1.0：carrier 已删除）。
              role_ids 由后端 service.syncUserRoles 做差量同步。 */}
          <Form.Item name="username" label={t('user.form.username')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item name="displayName" label={t('user.form.displayName')}>
            <Input placeholder={t('user.form.displayNameInputPlaceholder')} maxLength={64} />
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
          <Form.Item name="roleIds" label={t('user.form.role')}>
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              options={roleOptions}
              allowClear
            />
          </Form.Item>
          <Form.Item name="status" label={t('user.form.status')}>
            <Radio.Group>
              <Radio value="active">{t('user.form.statusActive')}</Radio>
              <Radio value="disabled">{t('user.form.statusDisabled')}</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item name="expireTime" label={t('user.form.expireTime')} extra={t('user.form.expireExtra')}>
            <DatePicker
              showTime
              format="YYYY-MM-DD HH:mm:ss"
              disabledDate={(current) => current && current.isBefore(dayjs().startOf('day'))}
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item name="description" label={t('user.form.description')}>
            <Input.TextArea rows={3} placeholder={t('user.form.descriptionEditPlaceholder')} maxLength={500} showCount />
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
        size={520}
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
          <Form.Item label={t('user.form.status')}>
            <Tag color={selectedUser?.status === 'active' ? 'success' : 'error'}>
              {selectedUser?.status === 'active' ? t('user.form.statusActive') : t('user.form.statusDisabled')}
            </Tag>
          </Form.Item>
          <Form.Item label={t('user.source')}>
            {selectedUser?.source === 'builtIn' && <Tag color="blue">{t('user.source.builtIn')}</Tag>}
            {selectedUser?.source === 'LDAP' && <Tag color="purple">LDAP</Tag>}
            {selectedUser?.source === 'admin' && <Tag>{t('user.source.admin')}</Tag>}
            {!selectedUser?.source && <span>—</span>}
          </Form.Item>
          <Form.Item name="username" label={t('user.form.username')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item label={t('user.form.displayName')}>
            <span>{selectedUser?.displayName || '-'}</span>
          </Form.Item>
          <Form.Item name="email" label={t('user.email')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item label={t('user.phone')}>
            <span>{selectedUser?.phone || '-'}</span>
          </Form.Item>
          <Form.Item label={t('user.form.role')}>
            <span>{selectedUser?.roles?.join(', ') || '-'}</span>
          </Form.Item>
          <Form.Item label={t('user.form.expireTime')}>
            <span>{selectedUser?.expireTime ? formatSystemTime(selectedUser.expireTime) : t('user.permanent')}</span>
          </Form.Item>
          <Form.Item label={t('user.lastLoginTime')}>
            <span>{selectedUser?.lastLoginTime ? formatSystemTime(selectedUser.lastLoginTime) : '-'}</span>
          </Form.Item>
          <Form.Item label={t('user.form.createdAt')}>
            <span>{selectedUser?.createTime ? formatSystemTime(selectedUser.createTime) : '-'}</span>
          </Form.Item>
          <Form.Item label={t('user.form.updatedAt')}>
            <span>{selectedUser?.updateTime ? formatSystemTime(selectedUser.updateTime) : '-'}</span>
          </Form.Item>
          <Form.Item label={t('user.form.createdBy')}>
            <span>{renderOperator(selectedUser?.creatorUsername)}</span>
          </Form.Item>
          <Form.Item label={t('user.form.updatedBy')}>
            <span>{renderOperator(selectedUser?.updaterUsername)}</span>
          </Form.Item>
          <Form.Item label={t('user.form.description')}>
            <span>{selectedUser?.description || '-'}</span>
          </Form.Item>
        </Form>
      </Drawer>

      {/* Reset Password Modal */}
      <Modal
        title={`${t('user.resetPassword')} - ${selectedUser?.username ?? ''}`}
        open={resetPwdVisible}
        onOk={handleResetPassword}
        confirmLoading={resetPassword.isPending}
        onCancel={() => {
          setResetPwdVisible(false);
          setResetUseDefault(false);
          pwdForm.resetFields();
          setSelectedUser(null);
        }}
        width={460}
      >
        <Form
          form={pwdForm}
          layout="vertical"
        >
          {/* Issue #649：Switch 用 React state 控制，
              靠父组件 re-render 驱动下方密码 Form.Item rules / disabled / placeholder 切换。 */}
          <Form.Item
            label={t('system.user.resetToDefault')}
            extra={
              hasDefaultPasswd
                ? t('system.user.resetToDefaultConfirm')
                : t('system.user.defaultPasswordNotSet')
            }
          >
            <Tooltip
              title={
                hasDefaultPasswd
                  ? undefined
                  : t('system.user.defaultPasswordNotSet')
              }
              placement="right"
            >
              <Switch
                checked={resetUseDefault}
                onChange={setResetUseDefault}
                disabled={!hasDefaultPasswd}
              />
            </Tooltip>
          </Form.Item>
          <Form.Item
            name="newPassword"
            label={t('user.newPassword')}
            rules={
              resetUseDefault
                ? []
                : [
                    { required: true, message: t('user.pleaseInputPassword') },
                    { min: 8, message: t('user.passwordMinLength') },
                  ]
            }
          >
            <Input.Password
              placeholder={
                resetUseDefault
                  ? defaultPasswd || t('user.newPassword')
                  : t('user.newPassword')
              }
              maxLength={20}
              disabled={resetUseDefault}
            />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label={t('user.confirmPassword')}
            dependencies={['newPassword']}
            rules={
              resetUseDefault
                ? []
                : [
                    { required: true, message: t('user.pleaseConfirmPassword') },
                    ({ getFieldValue: gfv }) => ({
                      validator(_, value) {
                        if (!value || gfv('newPassword') === value) return Promise.resolve();
                        return Promise.reject(new Error(t('user.passwordMismatch')));
                      },
                    }),
                  ]
            }
          >
            <Input.Password
              placeholder={
                resetUseDefault
                  ? defaultPasswd || t('user.confirmPassword')
                  : t('user.confirmPassword')
              }
              maxLength={20}
              disabled={resetUseDefault}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 批量分配角色 Modal（PRD §5.5 / §11.5：原"移动到组"重命名） */}
      <Modal
        title={t('user.modal.batchAssignRole')}
        open={moveGroupVisible}
        onOk={handleMoveGroup}
        onCancel={() => {
          setMoveGroupVisible(false);
          moveGroupForm.resetFields();
        }}
        confirmLoading={batchAssignRoles.isPending}
        width={420}
      >
        <Form form={moveGroupForm} layout="vertical">
          <Form.Item
            name="targetRoleIds"
            label={t('user.form.targetRole')}
            rules={[{ required: true, message: t('common.pleaseSelect') }]}
            extra={t('user.batchReplaceRoleHint')}
          >
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              options={roleOptions}
            />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
