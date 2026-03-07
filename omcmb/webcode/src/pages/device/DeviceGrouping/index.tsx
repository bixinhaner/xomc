import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Button,
  Dropdown,
  Form,
  Input,
  Modal,
  Space,
  Tag,
  Tree,
  Typography,
  message,
} from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  FolderAddOutlined,
  FolderOutlined,
  MoreOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import type { DataNode } from 'antd/es/tree';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import StatusIndicator from '@/components/StatusIndicator';
import { useDeviceGroups, useDeviceList } from '@/hooks/api/useDevices';
import { useT } from '@/hooks/useT';
import type { Device } from '@/types/device';

const { Title, Text } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
  none: 'default',
};

function buildTreeData(
  groups: Array<{ id: string; name: string; parentId: string | null; deviceCount: number; description: string }>,
  selectedId: string | null,
  onContextMenu: (groupId: string) => void,
  t: (id: string, values?: Record<string, unknown>) => string
): DataNode[] {
  const rootGroups = groups.filter((g) => g.parentId === null);

  function buildNode(group: typeof groups[0]): DataNode {
    const children = groups.filter((g) => g.parentId === group.id);
    return {
      key: group.id,
      title: (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            width: '100%',
            padding: '2px 0',
          }}
        >
          <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            <FolderOutlined style={{ marginRight: 6, color: '#FA8C16' }} />
            {group.name}
            <Text type="secondary" style={{ fontSize: 11, marginLeft: 6 }}>
              ({group.deviceCount})
            </Text>
          </span>
          <Dropdown
            menu={{
              items: [
                {
                  key: 'add-child',
                  label: t('common.add'),
                  icon: <FolderAddOutlined />,
                  onClick: (info) => {
                    info.domEvent.stopPropagation();
                    onContextMenu(`add-child:${group.id}`);
                  },
                },
                {
                  key: 'edit',
                  label: t('common.edit'),
                  icon: <EditOutlined />,
                  onClick: (info) => {
                    info.domEvent.stopPropagation();
                    onContextMenu(`edit:${group.id}`);
                  },
                },
                { type: 'divider' },
                {
                  key: 'delete',
                  label: t('common.delete'),
                  icon: <DeleteOutlined />,
                  danger: true,
                  onClick: (info) => {
                    info.domEvent.stopPropagation();
                    onContextMenu(`delete:${group.id}`);
                  },
                },
              ] as MenuProps['items'],
            }}
            trigger={['click']}
          >
            <Button
              type="text"
              size="small"
              icon={<MoreOutlined />}
              onClick={(e) => e.stopPropagation()}
              style={{ flexShrink: 0 }}
            />
          </Dropdown>
        </div>
      ),
      icon: null,
      children: children.length > 0 ? children.map(buildNode) : undefined,
    };
  }

  return rootGroups.map(buildNode);
}

export default function DeviceGrouping() {
  const t = useT();
  const navigate = useNavigate();
  const { data: groupsData, refetch: refetchGroups } = useDeviceGroups();
  const groups = groupsData ?? [];

  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [parentIdForAdd, setParentIdForAdd] = useState<string | null>(null);
  const [editGroupId, setEditGroupId] = useState<string | null>(null);
  const [addForm] = Form.useForm<{ name: string; description: string }>();
  const [editForm] = Form.useForm<{ name: string; description: string }>();

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
    none: t('alarm.severity.none'),
  }), [t]);

  const queryParams = useMemo(
    () => ({ page: currentPage, pageSize } as Parameters<typeof useDeviceList>[0]),
    [currentPage, pageSize]
  );

  const { data: deviceData, isLoading, refetch } = useDeviceList(queryParams);
  const devices: Device[] = deviceData?.items ?? [];
  const total = deviceData?.total ?? 0;

  const selectedGroup = useMemo(
    () => groups.find((g) => g.id === selectedGroupId),
    [groups, selectedGroupId]
  );

  const handleContextMenu = useCallback(
    (action: string) => {
      const [cmd, groupId] = action.split(':');
      if (cmd === 'add-child') {
        setParentIdForAdd(groupId ?? null);
        addForm.resetFields();
        setAddModalOpen(true);
      } else if (cmd === 'edit') {
        const grp = groups.find((g) => g.id === groupId);
        if (grp) {
          setEditGroupId(groupId ?? null);
          editForm.setFieldsValue({ name: grp.name, description: grp.description });
          setEditModalOpen(true);
        }
      } else if (cmd === 'delete') {
        Modal.confirm({
          title: t('common.confirmDelete'),
          content: t('common.deleteConfirmMsg'),
          okText: t('common.confirmDelete'),
          okType: 'danger',
          onOk: async () => {
            await refetchGroups();
            void message.success(t('common.deleteSuccess'));
          },
        });
      }
    },
    [groups, addForm, editForm, refetchGroups, t]
  );

  const treeData = useMemo(
    () => buildTreeData(groups, selectedGroupId, handleContextMenu, t),
    [groups, selectedGroupId, handleContextMenu, t]
  );

  const handleAddGroup = useCallback(async () => {
    const values = await addForm.validateFields();
    // In a real app, call API to create group
    void message.success(`${values.name}`);
    setAddModalOpen(false);
    await refetchGroups();
  }, [addForm, refetchGroups]);

  const handleEditGroup = useCallback(async () => {
    const values = await editForm.validateFields();
    void message.success(`${values.name}`);
    setEditModalOpen(false);
    await refetchGroups();
  }, [editForm, refetchGroups]);

  const columns = useMemo(
    (): DataTableColumn<Device>[] => [
      {
        key: 'sn',
        title: 'SN',
        dataIndex: 'sn',
        width: 150,
        mono: true,
        copyable: true,
        render: (_val, record) => (
          <Typography.Link
            style={{ fontFamily: 'monospace', fontSize: 12 }}
            onClick={() => void navigate(`/device/detail/${record.sn}`)}
          >
            {record.sn}
          </Typography.Link>
        ),
      },
      { key: 'name', title: t('device.name'), dataIndex: 'name', width: 160, ellipsis: true },
      { key: 'vendor', title: t('device.vendor'), dataIndex: 'vendor', width: 90 },
      { key: 'productType', title: t('device.productType'), dataIndex: 'productType', width: 90 },
      {
        key: 'connStatus',
        title: t('device.connStatus'),
        dataIndex: 'connStatus',
        width: 100,
        render: (_val, record) => (
          <StatusIndicator status={record.connStatus === 'online' ? 'online' : 'offline'} />
        ),
      },
      {
        key: 'alarmLevel',
        title: t('device.alarmLevel'),
        dataIndex: 'alarmLevel',
        width: 90,
        render: (_val, record) => (
          <Tag color={SEVERITY_COLOR[record.alarmLevel] ?? 'default'}>
            {SEVERITY_LABEL[record.alarmLevel] ?? record.alarmLevel}
          </Tag>
        ),
      },
      { key: 'ipAddress', title: t('device.ipAddress'), dataIndex: 'ipAddress', width: 130, mono: true },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 80,
        fixed: 'right',
        render: (_val, record) => (
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => void navigate(`/device/detail/${record.sn}`)}
          >
            {t('common.detail')}
          </Button>
        ),
      },
    ],
    [navigate, t, SEVERITY_LABEL]
  );

  const treePanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div
        style={{
          padding: '12px 12px 8px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <Title level={5} style={{ margin: 0, fontSize: 14 }}>
          {t('nav.device.group')}
        </Title>
        <Button
          type="text"
          size="small"
          icon={<PlusOutlined />}
          onClick={() => {
            setParentIdForAdd(null);
            addForm.resetFields();
            setAddModalOpen(true);
          }}
        >
          {t('common.add')}
        </Button>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '8px 4px' }}>
        <Tree
          treeData={[
            {
              key: '__all__',
              title: (
                <span>
                  <FolderOutlined style={{ marginRight: 6, color: 'var(--color-primary-600)' }} />
                  {t('common.all')}
                  <Text type="secondary" style={{ fontSize: 11, marginLeft: 6 }}>
                    ({total})
                  </Text>
                </span>
              ),
              children: treeData,
            },
          ]}
          defaultExpandAll
          selectedKeys={selectedGroupId ? [selectedGroupId] : ['__all__']}
          onSelect={(keys) => {
            const key = keys[0] as string | undefined;
            if (key === '__all__' || !key) {
              setSelectedGroupId(null);
            } else {
              setSelectedGroupId(key);
            }
            setCurrentPage(1);
          }}
          blockNode
          style={{ fontSize: 13 }}
        />
      </div>
    </div>
  );

  return (
    <>
      <TreeListPageLayout tree={treePanel} defaultTreeWidth={260}>
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%', padding: 16, gap: 12 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Title level={5} style={{ margin: 0 }}>
              {selectedGroup ? selectedGroup.name : t('common.all')}
              <Text type="secondary" style={{ fontSize: 13, marginLeft: 8, fontWeight: 400 }}>
                {t('table.total')} {total}
              </Text>
            </Title>
          </div>

          <DataTable<Device>
            tableId="device-grouping-table"
            columns={columns}
            dataSource={devices}
            loading={isLoading}
            rowKey="id"
            selectable
            total={total}
            pageSize={pageSize}
            currentPage={currentPage}
            onPageChange={(page, size) => {
              setCurrentPage(page);
              setPageSize(size);
            }}
            onRefresh={() => void refetch()}
            defaultDensity="compact"
          />
        </div>
      </TreeListPageLayout>

      {/* Add Group Modal */}
      <Modal
        title={t('common.add')}
        open={addModalOpen}
        onOk={() => void handleAddGroup()}
        onCancel={() => setAddModalOpen(false)}
        okText={t('common.confirm')}
      >
        <Form form={addForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label={t('table.name')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="description" label={t('table.description')}>
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Edit Group Modal */}
      <Modal
        title={t('common.edit')}
        open={editModalOpen}
        onOk={() => void handleEditGroup()}
        onCancel={() => setEditModalOpen(false)}
        okText={t('common.save')}
      >
        <Form form={editForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label={t('table.name')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="description" label={t('table.description')}>
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
