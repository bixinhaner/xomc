import { useState, useMemo, useCallback } from 'react';
import {
  App,
  Button,
  Tag,
  Drawer,
  Form,
  Input,
  Switch,
  Dropdown,
  Tree,
  Card,
  Select,
  Space,
  Alert,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  MoreOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useRoles,
  useCreateRole,
  useUpdateRole,
  useDeleteRoles,
  useAllDeviceGroups,
} from '@/hooks/api/useSystem';
import type { Role } from '@/types/system';
import { useT } from '@/hooks/useT';

// Permission tree data - 每个模块包含只读/读写两种权限
const getPermissionModules = (t: (id: string, values?: Record<string, unknown>) => string) => [
  {
    key: 'device',
    title: t('role.modules.device'),
    children: [
      { key: 'device:read', title: t('role.permission.read') },
      { key: 'device:write', title: t('role.permission.write') },
    ],
  },
  {
    key: 'alarm',
    title: t('role.modules.alarm'),
    children: [
      { key: 'alarm:read', title: t('role.permission.read') },
      { key: 'alarm:write', title: t('role.permission.write') },
    ],
  },
  {
    key: 'performance',
    title: t('role.modules.performance'),
    children: [
      { key: 'performance:read', title: t('role.permission.read') },
      { key: 'performance:write', title: t('role.permission.write') },
    ],
  },
  {
    key: 'software',
    title: t('role.modules.software'),
    children: [
      { key: 'software:read', title: t('role.permission.read') },
      { key: 'software:write', title: t('role.permission.write') },
    ],
  },
  {
    key: 'file',
    title: t('role.modules.file'),
    children: [
      { key: 'file:read', title: t('role.permission.read') },
      { key: 'file:write', title: t('role.permission.write') },
    ],
  },
  {
    key: 'log',
    title: t('role.modules.log'),
    children: [
      { key: 'log:read', title: t('role.permission.read') },
      { key: 'log:write', title: t('role.permission.write') },
    ],
  },
  {
    key: 'system',
    title: t('role.modules.system'),
    children: [
      { key: 'system:read', title: t('role.permission.read') },
      { key: 'system:write', title: t('role.permission.write') },
    ],
  },
  {
    key: 'report',
    title: t('role.modules.report'),
    children: [
      { key: 'report:read', title: t('role.permission.read') },
      { key: 'report:write', title: t('role.permission.write') },
    ],
  },
  {
    key: 'ops',
    title: t('role.modules.ops'),
    children: [
      { key: 'ops:read', title: t('role.permission.read') },
      { key: 'ops:write', title: t('role.permission.write') },
    ],
  },
];

export default function RoleManagement() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createVisible, setCreateVisible] = useState(false);
  const [editVisible, setEditVisible] = useState(false);
  const [viewVisible, setViewVisible] = useState(false);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [form] = Form.useForm();
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [checkedPermissions, setCheckedPermissions] = useState<string[]>([]);
  const [selectedDeviceGroupIds, setSelectedDeviceGroupIds] = useState<string[]>([]);

  const { data, isLoading, refetch } = useRoles({
    roleName: filters.roleName as string | undefined,
    page,
    pageSize,
  });

  const { data: allDeviceGroups } = useAllDeviceGroups();

  // 设备组选项（所有设备组都可批量选择）
  const deviceGroupOptions = useMemo(
    () => (allDeviceGroups ?? [])
      .filter((g) => g.parentId !== null) // 只显示二级节点
      .map((g) => ({ label: g.name, value: g.id })),
    [allDeviceGroups]
  );

  const createRole = useCreateRole();
  const updateRole = useUpdateRole();
  const deleteRoles = useDeleteRoles();

  const isBuiltIn = useCallback((role: Role) => role.builtIn === 1 || role.builtIn === 2, []);

  // 权限模块数据
  const permissionModules = useMemo(() => getPermissionModules(t as (id: string, values?: Record<string, unknown>) => string), [t]);

  // 已有的角色名称列表（用于重复检查）
  const existingRoleNames = useMemo(
    () => (data?.items ?? []).map((r) => r.roleName.toLowerCase()),
    [data?.items]
  );

  // 校验角色名称
  const validateRoleName = useCallback((_: unknown, value: string) => {
    if (!value || !value.trim()) {
      return Promise.reject(new Error(t('common.pleaseInput')));
    }
    if (value.length > 200) {
      return Promise.reject(new Error(t('role.roleNameMaxLength')));
    }
    // 编辑模式下，如果名称没有变化，则跳过重复检查
    const currentName = selectedRole?.roleName?.toLowerCase();
    if (editVisible && currentName === value.toLowerCase()) {
      return Promise.resolve();
    }
    // 检查是否重复
    if (existingRoleNames.includes(value.toLowerCase())) {
      return Promise.reject(new Error(t('role.roleNameExists')));
    }
    return Promise.resolve();
  }, [t, existingRoleNames, selectedRole, editVisible]);

  const handleDelete = useCallback((role: Role) => {
    if (isBuiltIn(role)) {
      modal.warning({
        title: t('common.warning'),
        content: t('role.builtInCannotDelete'),
      });
      return;
    }
    modal.confirm({
      title: t('common.confirmDelete'),
      onOk: () => {
        deleteRoles.mutate([role.id], {
          onSuccess: () => message.success(t('common.deleteSuccess')),
        });
      },
    });
  }, [isBuiltIn, t, deleteRoles, modal, message]);

  const handleBatchDelete = useCallback((keys: React.Key[]) => {
    const rolesToDelete = (data?.items || []).filter(
      (r) => keys.includes(r.id) && !isBuiltIn(r)
    );
    if (rolesToDelete.length === 0) {
      modal.warning({
        title: t('common.warning'),
        content: t('role.noRolesToDelete'),
      });
      return;
    }
    const builtInCount = keys.length - rolesToDelete.length;
    modal.confirm({
      title: t('common.confirmDelete'),
      content: builtInCount > 0
        ? `${t('role.selectedBuiltIn')} ${builtInCount} ${t('role.builtInSkipped')}`
        : undefined,
      onOk: () => {
        deleteRoles.mutate(rolesToDelete.map((r) => r.id), {
          onSuccess: () => {
            message.success(t('common.deleteSuccess'));
            setSelectedKeys([]);
          },
        });
      },
    });
  }, [data?.items, isBuiltIn, t, deleteRoles, modal, message]);

  // 校验并提交创建
  const handleCreate = useCallback(() => {
    // 校验权限
    if (checkedPermissions.length === 0) {
      message.warning(t('role.pleaseSelectPermission'));
      return;
    }
    // 校验设备组
    if (selectedDeviceGroupIds.length === 0) {
      message.warning(t('role.pleaseSelectDeviceGroup'));
      return;
    }

    form.validateFields().then((vals) => {
      createRole.mutate(
        {
          roleName: vals.roleName as string,
          batchOperation: vals.batchOperation ? 1 : 0,
          description: (vals.description as string) ?? '',
          permissions: checkedPermissions,
          deviceGroupIds: selectedDeviceGroupIds,
          builtIn: 0,
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setCreateVisible(false);
            form.resetFields();
            setCheckedPermissions([]);
            setSelectedDeviceGroupIds([]);
          },
        },
      );
    });
  }, [form, createRole, checkedPermissions, selectedDeviceGroupIds, message, t]);

  // 校验并提交编辑
  const handleEdit = useCallback(() => {
    if (!selectedRole) return;

    // 校验权限
    if (checkedPermissions.length === 0) {
      message.warning(t('role.pleaseSelectPermission'));
      return;
    }
    // 校验设备组
    if (selectedDeviceGroupIds.length === 0) {
      message.warning(t('role.pleaseSelectDeviceGroup'));
      return;
    }

    form.validateFields().then((vals) => {
      updateRole.mutate(
        {
          id: selectedRole.id,
          data: {
            roleName: vals.roleName as string,
            batchOperation: vals.batchOperation ? 1 : 0,
            description: vals.description as string,
            permissions: checkedPermissions,
            deviceGroupIds: selectedDeviceGroupIds,
          },
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setEditVisible(false);
            form.resetFields();
            setSelectedRole(null);
            setCheckedPermissions([]);
            setSelectedDeviceGroupIds([]);
          },
        },
      );
    });
  }, [selectedRole, form, updateRole, checkedPermissions, selectedDeviceGroupIds, message, t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'roleName', label: t('role.roleName'), type: 'input', placeholder: t('role.roleName') },
  ], [t]);

  const columns: DataTableColumn<Role & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 100,
      fixed: 'left',
      render: (_, record) => {
        const role = record as Role;
        return (
          <Dropdown
            menu={{
              items: [
                {
                  key: 'view',
                  label: t('common.view'),
                  icon: <EyeOutlined />,
                  onClick: () => {
                    setSelectedRole(role);
                    form.setFieldsValue({
                      roleName: role.roleName,
                      batchOperation: role.batchOperation === 1,
                      description: role.description,
                    });
                    setCheckedPermissions(role.permissions || []);
                    setSelectedDeviceGroupIds((role as Role & { deviceGroupIds?: string[] }).deviceGroupIds || []);
                    setViewVisible(true);
                  },
                },
                {
                  key: 'edit',
                  label: t('common.edit'),
                  icon: <EditOutlined />,
                  disabled: isBuiltIn(role),
                  onClick: () => {
                    setSelectedRole(role);
                    form.setFieldsValue({
                      roleName: role.roleName,
                      batchOperation: role.batchOperation === 1,
                      description: role.description,
                    });
                    setCheckedPermissions(role.permissions || []);
                    setSelectedDeviceGroupIds((role as Role & { deviceGroupIds?: string[] }).deviceGroupIds || []);
                    setEditVisible(true);
                  },
                },
                { type: 'divider' },
                {
                  key: 'delete',
                  label: t('common.delete'),
                  icon: <DeleteOutlined />,
                  danger: true,
                  disabled: isBuiltIn(role),
                  onClick: () => handleDelete(role),
                },
              ],
            }}
          >
            <Button size="small" icon={<MoreOutlined />}>{t('common.more')}</Button>
          </Dropdown>
        );
      },
    },
    {
      key: 'roleName',
      title: t('role.roleName'),
      dataIndex: 'roleName',
      width: 200,
      render: (val, record) => {
        const role = record as Role;
        return (
          <span>
            {String(val)}
            {isBuiltIn(role) && <Tag color="blue" style={{ marginLeft: 8 }}>{t('role.builtIn')}</Tag>}
          </span>
        );
      },
    },
    {
      key: 'batchOperation',
      title: t('role.batchOperation'),
      dataIndex: 'batchOperation',
      width: 120,
      render: (val) => (
        <Tag color={val === 1 ? 'green' : 'default'}>
          {val === 1 ? t('common.yes') : t('common.no')}
        </Tag>
      ),
    },
    { key: 'updUser', title: t('role.updUser'), dataIndex: 'updUser', width: 150 },
    {
      key: 'updTime',
      title: t('role.updTime'),
      dataIndex: 'updTime',
      width: 180,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '—'),
    },
    { key: 'description', title: t('role.description'), dataIndex: 'description', ellipsis: true },
  ], [t, form, isBuiltIn, handleDelete]);

  const renderPermissionTree = (readOnly = false) => (
    <Card
      title={t('role.permissionConfig')}
      size="small"
      style={{ marginTop: 16 }}
      extra={!readOnly && checkedPermissions.length === 0 ? (
        <span style={{ color: 'var(--color-error)', fontSize: 12 }}>
          {t('role.pleaseSelectPermission')}
        </span>
      ) : null}
    >
      <Tree
        checkable
        checkedKeys={checkedPermissions}
        treeData={permissionModules}
        defaultExpandAll
        onCheck={(checked) => {
          if (!readOnly) {
            setCheckedPermissions(checked as string[]);
          }
        }}
        selectable={false}
        disabled={readOnly}
      />
    </Card>
  );

  const renderDeviceGroupSelect = (readOnly = false) => (
    <Form.Item
      label={t('role.deviceGroups')}
      required={!readOnly}
      help={!readOnly && selectedDeviceGroupIds.length === 0 ? t('role.pleaseSelectDeviceGroup') : undefined}
      validateStatus={!readOnly && selectedDeviceGroupIds.length === 0 ? 'warning' : undefined}
    >
      {readOnly ? (
        <Space wrap>
          {selectedDeviceGroupIds.length > 0 ? (
            deviceGroupOptions
              .filter((opt) => selectedDeviceGroupIds.includes(opt.value))
              .map((opt) => <Tag key={opt.value}>{opt.label}</Tag>)
          ) : (
            <span style={{ color: 'var(--color-text-secondary)' }}>-</span>
          )}
        </Space>
      ) : (
        <Select
          mode="multiple"
          placeholder={t('role.pleaseSelectDeviceGroups')}
          value={selectedDeviceGroupIds}
          onChange={setSelectedDeviceGroupIds}
          options={deviceGroupOptions}
          style={{ width: '100%' }}
          showSearch
          filterOption={(input, option) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
          }
        />
      )}
    </Form.Item>
  );

  // 关闭抽屉时重置状态
  const handleCloseCreate = useCallback(() => {
    setCreateVisible(false);
    form.resetFields();
    setCheckedPermissions([]);
    setSelectedDeviceGroupIds([]);
  }, [form]);

  const handleCloseEdit = useCallback(() => {
    setEditVisible(false);
    form.resetFields();
    setSelectedRole(null);
    setCheckedPermissions([]);
    setSelectedDeviceGroupIds([]);
  }, [form]);

  const handleCloseView = useCallback(() => {
    setViewVisible(false);
    form.resetFields();
    setSelectedRole(null);
    setCheckedPermissions([]);
    setSelectedDeviceGroupIds([]);
  }, [form]);

  return (
    <ListPageLayout
      title={t('nav.system.roles')}
      subtitle={t('nav.system.roles')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
          {t('common.add')}
        </Button>
      }
    >
      <FilterBar
        filterId="role-management-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="role-management-list"
        columns={columns}
        dataSource={(data?.items ?? []) as (Role & Record<string, unknown>)[]}
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
            onClick: handleBatchDelete,
          },
        ]}
        scroll={{ x: 1100 }}
      />

      {/* Create Drawer */}
      <Drawer
        title={t('common.add')}
        open={createVisible}
        onClose={handleCloseCreate}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button style={{ marginRight: 8 }} onClick={handleCloseCreate}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" loading={createRole.isPending} onClick={handleCreate}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="roleName"
            label={t('role.roleName')}
            rules={[{ required: true, validator: validateRoleName }]}
          >
            <Input placeholder={t('role.roleNamePlaceholder')} maxLength={200} showCount />
          </Form.Item>
          <Form.Item
            name="batchOperation"
            label={t('role.batchOperation')}
            valuePropName="checked"
            initialValue={false}
          >
            <Switch checkedChildren={t('common.yes')} unCheckedChildren={t('common.no')} />
          </Form.Item>
          <Form.Item name="description" label={t('role.description')}>
            <Input.TextArea
              rows={2}
              placeholder={t('role.descriptionPlaceholder')}
              maxLength={500}
              showCount
            />
          </Form.Item>
          {renderDeviceGroupSelect(false)}
          {renderPermissionTree(false)}
        </Form>
      </Drawer>

      {/* Edit Drawer */}
      <Drawer
        title={t('common.edit')}
        open={editVisible}
        onClose={handleCloseEdit}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button style={{ marginRight: 8 }} onClick={handleCloseEdit}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" loading={updateRole.isPending} onClick={handleEdit}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="roleName"
            label={t('role.roleName')}
          >
            <Input readOnly style={{ color: 'var(--color-text-secondary)' }} />
          </Form.Item>
          <Form.Item
            name="batchOperation"
            label={t('role.batchOperation')}
            valuePropName="checked"
          >
            <Switch checkedChildren={t('common.yes')} unCheckedChildren={t('common.no')} />
          </Form.Item>
          <Form.Item name="description" label={t('role.description')}>
            <Input.TextArea
              rows={2}
              placeholder={t('role.descriptionPlaceholder')}
              maxLength={500}
              showCount
            />
          </Form.Item>
          {renderDeviceGroupSelect(false)}
          {renderPermissionTree(false)}
        </Form>
      </Drawer>

      {/* View Drawer */}
      <Drawer
        title={t('common.view')}
        open={viewVisible}
        onClose={handleCloseView}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button onClick={handleCloseView}>
              {t('common.close')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="roleName" label={t('role.roleName')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item label={t('role.batchOperation')}>
            <Tag color={selectedRole?.batchOperation === 1 ? 'green' : 'default'}>
              {selectedRole?.batchOperation === 1 ? t('common.yes') : t('common.no')}
            </Tag>
          </Form.Item>
          <Form.Item name="description" label={t('role.description')}>
            <Input.TextArea rows={2} readOnly />
          </Form.Item>
          {renderDeviceGroupSelect(true)}
          <Form.Item label={t('role.userCount')}>
            <span>{selectedRole?.userCount ?? 0}</span>
          </Form.Item>
          <Form.Item label={t('role.updUser')}>
            <span>{selectedRole?.updUser ?? '-'}</span>
          </Form.Item>
          <Form.Item label={t('role.updTime')}>
            <span>{selectedRole?.updTime ? new Date(selectedRole.updTime).toLocaleString('zh-CN') : '-'}</span>
          </Form.Item>
          {renderPermissionTree(true)}
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
