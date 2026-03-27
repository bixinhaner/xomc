import { useState, useMemo, useCallback } from 'react';
import {
  App,
  Button,
  Drawer,
  Form,
  Input,
  Dropdown,
  Tag,
  Select,
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
  useGroups,
  useCreateGroup,
  useUpdateGroup,
  useDeleteGroups,
  useAllRoles,
  useAllUsers,
} from '@/hooks/api/useSystem';
import type { Group } from '@/types/system';
import { useT } from '@/hooks/useT';

export default function GroupManagement() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createVisible, setCreateVisible] = useState(false);
  const [editVisible, setEditVisible] = useState(false);
  const [viewVisible, setViewVisible] = useState(false);
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null);
  const [form] = Form.useForm();
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [selectedRoleIds, setSelectedRoleIds] = useState<string[]>([]);
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);

  const { data, isLoading, refetch } = useGroups({
    groupName: filters.groupName as string | undefined,
    page,
    pageSize,
  });

  const { data: allRoles } = useAllRoles();
  const { data: allUsers } = useAllUsers();

  const createGroup = useCreateGroup();
  const updateGroup = useUpdateGroup();
  const deleteGroups = useDeleteGroups();

  const isBuiltIn = useCallback((group: Group) => group.builtIn === 1 || group.builtIn === 2, []);

  const handleDelete = useCallback((group: Group) => {
    if (isBuiltIn(group)) {
      modal.warning({
        title: t('common.warning'),
        content: t('group.builtInCannotDelete'),
      });
      return;
    }
    modal.confirm({
      title: t('common.confirmDelete'),
      onOk: () => {
        deleteGroups.mutate([group.id], {
          onSuccess: () => message.success(t('common.deleteSuccess')),
        });
      },
    });
  }, [isBuiltIn, t, deleteGroups, modal, message]);

  const handleCreate = useCallback(() => {
    form.validateFields().then((vals) => {
      createGroup.mutate(
        {
          groupName: vals.groupName as string,
          description: (vals.description as string) ?? '',
          builtIn: 0,
          roleIds: selectedRoleIds,
          userIds: selectedUserIds,
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setCreateVisible(false);
            form.resetFields();
            setSelectedRoleIds([]);
            setSelectedUserIds([]);
          },
        },
      );
    });
  }, [form, createGroup, selectedRoleIds, selectedUserIds, message, t]);

  const handleEdit = useCallback(() => {
    if (!selectedGroup) return;
    form.validateFields().then((vals) => {
      updateGroup.mutate(
        {
          id: selectedGroup.id,
          data: {
            groupName: vals.groupName as string,
            description: vals.description as string,
            roleIds: selectedRoleIds,
            userIds: selectedUserIds,
          },
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setEditVisible(false);
            form.resetFields();
            setSelectedGroup(null);
            setSelectedRoleIds([]);
            setSelectedUserIds([]);
          },
        },
      );
    });
  }, [selectedGroup, form, updateGroup, selectedRoleIds, selectedUserIds, message, t]);

  const handleBatchDelete = useCallback((keys: React.Key[]) => {
    const groupsToDelete = (data?.items || []).filter(
      (g) => keys.includes(g.id) && !isBuiltIn(g)
    );
    if (groupsToDelete.length === 0) {
      modal.warning({
        title: t('common.warning'),
        content: t('group.noGroupsToDelete'),
      });
      return;
    }
    const builtInCount = keys.length - groupsToDelete.length;
    modal.confirm({
      title: t('common.confirmDelete'),
      content: builtInCount > 0
        ? `${t('group.selectedBuiltIn')} ${builtInCount} ${t('group.builtInSkipped')}`
        : undefined,
      onOk: () => {
        deleteGroups.mutate(groupsToDelete.map((g) => g.id), {
          onSuccess: () => {
            message.success(t('common.deleteSuccess'));
            setSelectedKeys([]);
          },
        });
      },
    });
  }, [data?.items, isBuiltIn, t, deleteGroups, modal, message]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'groupName', label: t('group.groupName'), type: 'input', placeholder: t('group.groupName') },
  ], [t]);

  const columns: DataTableColumn<Group & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 100,
      fixed: 'left',
      render: (_, record) => {
        const group = record as Group;
        return (
          <Dropdown
            menu={{
              items: [
                {
                  key: 'view',
                  label: t('common.view'),
                  icon: <EyeOutlined />,
                  onClick: () => {
                    setSelectedGroup(group);
                    form.setFieldsValue({
                      groupName: group.groupName,
                      description: group.description,
                    });
                    setViewVisible(true);
                  },
                },
                {
                  key: 'edit',
                  label: t('common.edit'),
                  icon: <EditOutlined />,
                  disabled: isBuiltIn(group),
                  onClick: () => {
                    setSelectedGroup(group);
                    form.setFieldsValue({
                      groupName: group.groupName,
                      description: group.description,
                    });
                    setSelectedRoleIds([]);
                    setSelectedUserIds([]);
                    setEditVisible(true);
                  },
                },
                { type: 'divider' },
                {
                  key: 'delete',
                  label: t('common.delete'),
                  icon: <DeleteOutlined />,
                  danger: true,
                  disabled: isBuiltIn(group),
                  onClick: () => handleDelete(group),
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
      key: 'groupName',
      title: t('group.groupName'),
      dataIndex: 'groupName',
      width: 200,
      render: (val, record) => {
        const group = record as Group;
        return (
          <span>
            {String(val)}
            {isBuiltIn(group) && <Tag color="blue" style={{ marginLeft: 8 }}>{t('group.builtIn')}</Tag>}
          </span>
        );
      },
    },
    { key: 'userCount', title: t('group.userCount'), dataIndex: 'userCount', width: 200 },
    { key: 'roleCount', title: t('group.roleCount'), dataIndex: 'roleCount', width: 200 },
    { key: 'updUser', title: t('group.updUser'), dataIndex: 'updUser', width: 200 },
    {
      key: 'updTime',
      title: t('group.updTime'),
      dataIndex: 'updTime',
      ellipsis: true,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '—'),
    },
  ], [t, form, isBuiltIn, handleDelete]);

  return (
    <ListPageLayout
      title={t('nav.system.groups')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
          {t('common.add')}
        </Button>
      }
    >
      <FilterBar
        filterId="group-management-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="group-management-list"
        columns={columns}
        dataSource={(data?.items ?? []) as (Group & Record<string, unknown>)[]}
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
        scroll={{ x: 1000 }}
      />

      {/* Create Drawer */}
      <Drawer
        title={t('common.add')}
        open={createVisible}
        onClose={() => {
          setCreateVisible(false);
          form.resetFields();
          setSelectedRoleIds([]);
          setSelectedUserIds([]);
        }}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button
              style={{ marginRight: 8 }}
              onClick={() => {
                setCreateVisible(false);
                form.resetFields();
                setSelectedRoleIds([]);
                setSelectedUserIds([]);
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type="primary"
              loading={createGroup.isPending}
              onClick={handleCreate}
            >
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="groupName"
            label={t('group.groupName')}
            rules={[{ required: true, message: t('common.pleaseInput') }]}
          >
            <Input placeholder={t('group.groupName')} />
          </Form.Item>
          <Form.Item name="description" label={t('group.description')}>
            <Input.TextArea rows={2} placeholder={t('group.description')} />
          </Form.Item>
          <Form.Item label={t('group.associatedRoles')}>
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              value={selectedRoleIds}
              onChange={setSelectedRoleIds}
              options={(allRoles ?? []).map((r) => ({ label: r.roleName, value: r.id }))}
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item label={t('group.associatedUsers')}>
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              value={selectedUserIds}
              onChange={setSelectedUserIds}
              options={(allUsers ?? []).map((u) => ({ label: u.userName, value: u.id }))}
              style={{ width: '100%' }}
            />
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
          setSelectedGroup(null);
          setSelectedRoleIds([]);
          setSelectedUserIds([]);
        }}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button
              style={{ marginRight: 8 }}
              onClick={() => {
                setEditVisible(false);
                form.resetFields();
                setSelectedGroup(null);
                setSelectedRoleIds([]);
                setSelectedUserIds([]);
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type="primary"
              loading={updateGroup.isPending}
              onClick={handleEdit}
            >
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="groupName"
            label={t('group.groupName')}
            rules={[{ required: true, message: t('common.pleaseInput') }]}
          >
            <Input placeholder={t('group.groupName')} />
          </Form.Item>
          <Form.Item name="description" label={t('group.description')}>
            <Input.TextArea rows={2} placeholder={t('group.description')} />
          </Form.Item>
          <Form.Item label={t('group.associatedRoles')}>
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              value={selectedRoleIds}
              onChange={setSelectedRoleIds}
              options={(allRoles ?? []).map((r) => ({ label: r.roleName, value: r.id }))}
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item label={t('group.associatedUsers')}>
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              value={selectedUserIds}
              onChange={setSelectedUserIds}
              options={(allUsers ?? []).map((u) => ({ label: u.userName, value: u.id }))}
              style={{ width: '100%' }}
            />
          </Form.Item>
        </Form>
      </Drawer>

      {/* View Drawer */}
      <Drawer
        title={t('common.view')}
        open={viewVisible}
        onClose={() => { setViewVisible(false); form.resetFields(); setSelectedGroup(null); }}
        width={520}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button onClick={() => { setViewVisible(false); form.resetFields(); setSelectedGroup(null); }}>
              {t('common.close')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="groupName" label={t('group.groupName')}>
            <Input readOnly />
          </Form.Item>
          <Form.Item name="description" label={t('group.description')}>
            <Input.TextArea rows={2} readOnly />
          </Form.Item>
          <Form.Item label={t('group.userCount')}>
            <span>{selectedGroup?.userCount ?? 0}</span>
          </Form.Item>
          <Form.Item label={t('group.roleCount')}>
            <span>{selectedGroup?.roleCount ?? 0}</span>
          </Form.Item>
          <Form.Item label={t('group.updUser')}>
            <span>{selectedGroup?.updUser ?? '-'}</span>
          </Form.Item>
          <Form.Item label={t('group.updTime')}>
            <span>{selectedGroup?.updTime ? new Date(selectedGroup.updTime).toLocaleString('zh-CN') : '-'}</span>
          </Form.Item>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
