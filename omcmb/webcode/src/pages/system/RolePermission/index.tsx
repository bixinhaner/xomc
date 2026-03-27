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
} from '@/hooks/api/useSystem';
import type { Role } from '@/types/system';
import { useT } from '@/hooks/useT';

// Permission tree data
const permissionModules = [
  {
    key: 'device',
    title: '设备管理',
    children: [
      { key: 'device:view', title: '查看设备列表' },
      { key: 'device:create', title: '创建设备' },
      { key: 'device:update', title: '编辑设备' },
      { key: 'device:delete', title: '删除设备' },
      { key: 'device:config', title: '配置管理' },
    ],
  },
  {
    key: 'alarm',
    title: '告警管理',
    children: [
      { key: 'alarm:view', title: '查看告警' },
      { key: 'alarm:confirm', title: '确认告警' },
      { key: 'alarm:clear', title: '清除告警' },
      { key: 'alarm:config', title: '配置告警规则' },
    ],
  },
  {
    key: 'performance',
    title: '性能管理',
    children: [
      { key: 'performance:view', title: '查看性能数据' },
      { key: 'performance:export', title: '导出性能数据' },
      { key: 'performance:config', title: '配置采集策略' },
    ],
  },
  {
    key: 'software',
    title: '软件版本',
    children: [
      { key: 'software:view', title: '查看版本列表' },
      { key: 'software:upload', title: '上传固件' },
      { key: 'software:upgrade', title: '发起升级' },
      { key: 'software:delete', title: '删除版本' },
    ],
  },
  {
    key: 'file',
    title: '文件管理',
    children: [
      { key: 'file:view', title: '查看文件' },
      { key: 'file:download', title: '下载文件' },
      { key: 'file:upload', title: '上传文件' },
      { key: 'file:delete', title: '删除文件' },
    ],
  },
  {
    key: 'log',
    title: '日志管理',
    children: [
      { key: 'log:view', title: '查看日志' },
      { key: 'log:export', title: '导出日志' },
      { key: 'log:config', title: '配置日志策略' },
    ],
  },
  {
    key: 'system',
    title: '系统管理',
    children: [
      { key: 'system:user', title: '用户管理' },
      { key: 'system:role', title: '角色权限管理' },
      { key: 'system:group', title: '用户组管理' },
      { key: 'system:config', title: '系统配置' },
      { key: 'system:dict', title: '数据字典' },
    ],
  },
  {
    key: 'report',
    title: '报表管理',
    children: [
      { key: 'report:view', title: '查看报表' },
      { key: 'report:generate', title: '生成报表' },
      { key: 'report:download', title: '下载报表' },
    ],
  },
  {
    key: 'ops',
    title: '运维工具',
    children: [
      { key: 'ops:template', title: '模板管理' },
      { key: 'ops:command', title: '命令执行' },
      { key: 'ops:diagnosis', title: '网络诊断' },
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

  const { data, isLoading, refetch } = useRoles({
    roleName: filters.roleName as string | undefined,
    page,
    pageSize,
  });

  const createRole = useCreateRole();
  const updateRole = useUpdateRole();
  const deleteRoles = useDeleteRoles();

  const isBuiltIn = useCallback((role: Role) => role.builtIn === 1 || role.builtIn === 2, []);

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

  const handleCreate = useCallback(() => {
    form.validateFields().then((vals) => {
      createRole.mutate(
        {
          roleName: vals.roleName as string,
          batchOperation: vals.batchOperation ? 1 : 0,
          description: (vals.description as string) ?? '',
          permissions: checkedPermissions,
          builtIn: 0,
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setCreateVisible(false);
            form.resetFields();
            setCheckedPermissions([]);
          },
        },
      );
    });
  }, [form, createRole, checkedPermissions, message, t]);

  const handleEdit = useCallback(() => {
    if (!selectedRole) return;
    form.validateFields().then((vals) => {
      updateRole.mutate(
        {
          id: selectedRole.id,
          data: {
            roleName: vals.roleName as string,
            batchOperation: vals.batchOperation ? 1 : 0,
            description: vals.description as string,
            permissions: checkedPermissions,
          },
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setEditVisible(false);
            form.resetFields();
            setSelectedRole(null);
            setCheckedPermissions([]);
          },
        },
      );
    });
  }, [selectedRole, form, updateRole, checkedPermissions, message, t]);

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
    <Card title={t('role.permissionConfig')} size="small" style={{ marginTop: 16 }}>
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
        onClose={() => {
          setCreateVisible(false);
          form.resetFields();
          setCheckedPermissions([]);
        }}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button
              style={{ marginRight: 8 }}
              onClick={() => {
                setCreateVisible(false);
                form.resetFields();
                setCheckedPermissions([]);
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type="primary"
              loading={createRole.isPending}
              onClick={handleCreate}
            >
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="roleName"
            label={t('role.roleName')}
            rules={[{ required: true, message: t('common.pleaseInput') }]}
          >
            <Input placeholder={t('role.roleName')} />
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
            <Input.TextArea rows={2} placeholder={t('role.description')} />
          </Form.Item>
          {renderPermissionTree(false)}
        </Form>
      </Drawer>

      {/* Edit Drawer */}
      <Drawer
        title={t('common.edit')}
        open={editVisible}
        onClose={() => {
          setEditVisible(false);
          form.resetFields();
          setSelectedRole(null);
          setCheckedPermissions([]);
        }}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button
              style={{ marginRight: 8 }}
              onClick={() => {
                setEditVisible(false);
                form.resetFields();
                setSelectedRole(null);
                setCheckedPermissions([]);
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type="primary"
              loading={updateRole.isPending}
              onClick={handleEdit}
            >
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="roleName"
            label={t('role.roleName')}
            rules={[{ required: true, message: t('common.pleaseInput') }]}
          >
            <Input placeholder={t('role.roleName')} />
          </Form.Item>
          <Form.Item
            name="batchOperation"
            label={t('role.batchOperation')}
            valuePropName="checked"
          >
            <Switch checkedChildren={t('common.yes')} unCheckedChildren={t('common.no')} />
          </Form.Item>
          <Form.Item name="description" label={t('role.description')}>
            <Input.TextArea rows={2} placeholder={t('role.description')} />
          </Form.Item>
          {renderPermissionTree(false)}
        </Form>
      </Drawer>

      {/* View Drawer */}
      <Drawer
        title={t('common.view')}
        open={viewVisible}
        onClose={() => {
          setViewVisible(false);
          form.resetFields();
          setSelectedRole(null);
          setCheckedPermissions([]);
        }}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button onClick={() => {
              setViewVisible(false);
              form.resetFields();
              setSelectedRole(null);
              setCheckedPermissions([]);
            }}>
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
