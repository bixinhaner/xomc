import React, { useCallback, useMemo, useState } from 'react';
import {
  App,
  Button,
  Divider,
  Drawer,
  Dropdown,
  Form,
  Input,
  InputNumber,
  Modal,
  Radio,
  Select,
  Switch,
  Tag,
  Tree,
  Typography,
} from 'antd';
import {
  CloseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  FolderAddOutlined,
  FolderOutlined,
  FolderOutlined as MoveToGroupIcon,
  MoreOutlined,
  PlusOutlined,
  RestOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import type { DataNode } from 'antd/es/tree';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import { useDeviceGroups, useDeviceList } from '@/hooks/api/useDevices';
import { useT } from '@/hooks/useT';
import type { Device, EngStatus } from '@/types/device';

const { Title, Text } = Typography;

// 名称过滤条件（复用设备规则的逻辑）
interface NameFilterItem {
  id: string;
  condition: 'contain' | 'notContain' | 'startWith' | 'endWith';
  value: string;
  andOr?: 'and' | 'or';
}

// 过滤条件选项
const getFilterConditionOptions = (t: (key: string) => string) => [
  { label: t('filter.contain'), value: 'contain' },
  { label: t('filter.notContain'), value: 'notContain' },
  { label: t('filter.startWith'), value: 'startWith' },
  { label: t('filter.endWith'), value: 'endWith' },
];

// And/Or 选项
const getAndOrOptions = (t: (key: string) => string) => [
  { label: t('filter.and'), value: 'and' },
  { label: t('filter.or'), value: 'or' },
];

// 生成唯一ID
const generateId = () => Math.random().toString(36).substring(2, 9);

// 生成操作描述
function generateOperators(
  rule: { matchingMode?: string; nameRuleList?: NameFilterItem[]; tacRag?: string },
  t: (key: string) => string
): string {
  if (rule.matchingMode === 'deviceName' && rule.nameRuleList?.length) {
    const orGroups: NameFilterItem[][] = [[]];

    rule.nameRuleList.forEach((filter, index) => {
      if (filter.value && filter.value.trim() !== '') {
        if (index > 0 && filter.andOr === 'or') {
          orGroups.push([]);
        }
        orGroups[orGroups.length - 1].push(filter);
      }
    });

    const filteredGroups = orGroups.filter((g) => g.length > 0);
    if (filteredGroups.length === 0) return '';

    const verbMap: Record<string, string> = {
      contain: t('filter.contain'),
      notContain: t('filter.notContain'),
      startWith: t('filter.startWith'),
      endWith: t('filter.endWith'),
    };

    const groupParts = filteredGroups.map((group) => {
      const conditionParts = group.map((item) => `${verbMap[item.condition]} "${item.value}"`);
      const groupText = conditionParts.join(` ${t('filter.and')} `);
      return filteredGroups.length > 1 || group.length > 1 ? `(${groupText})` : groupText;
    });

    return groupParts.join(` ${t('filter.or')} `);
  } else if (rule.matchingMode === 'tac') {
    return `TAC: ${rule.tacRag || ''}`;
  } else if (rule.matchingMode === 'lac') {
    return `LAC: ${rule.tacRag || ''}`;
  }
  return '';
}

function buildTreeData(
  groups: Array<{ id: string; name: string; parentId: string | null; deviceCount: number; description: string }>,
  selectedId: string | null,
  onContextMenu: (groupId: string) => void,
  t: (id: string, values?: Record<string, unknown>) => string
): DataNode[] {
  const rootGroups = groups.filter((g) => g.parentId === null);

  function buildNode(group: typeof groups[0], isRootLevel: boolean): DataNode {
    const children = groups.filter((g) => g.parentId === group.id);
    // 一级节点（parentId === null）不可选择，二级节点可选择
    const isLevel1 = group.parentId === null;

    // 默认设备组只显示添加，其他一级节点显示完整菜单
    const isDefaultGroup = group.id === 'grp-default';

    let menuItems: MenuProps['items'];
    if (isRootLevel && isDefaultGroup) {
      // 默认设备组：只显示添加
      menuItems = [
        {
          key: 'add-child',
          label: t('common.add'),
          icon: <FolderAddOutlined />,
          onClick: (info) => {
            info.domEvent.stopPropagation();
            onContextMenu(`add-child:${group.id}`);
          },
        },
      ];
    } else if (isRootLevel) {
      // 其他一级节点：显示添加、编辑、删除
      menuItems = [
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
      ];
    } else {
      // 二级节点：显示完整菜单
      menuItems = [
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
      ];
    }

    return {
      key: group.id,
      selectable: !isLevel1,
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
            menu={{ items: menuItems }}
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
      children: children.length > 0 ? children.map((child) => buildNode(child, false)) : undefined,
    };
  }

  return rootGroups.map((group) => buildNode(group, true));
}

export default function DeviceGrouping() {
  const t = useT();
  const { modal, message } = App.useApp();
  const { data: groupsData, refetch: refetchGroups } = useDeviceGroups();
  const groups = groupsData ?? [];

  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null);
  const [groupSearchText, setGroupSearchText] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [moveToGroupModalOpen, setMoveToGroupModalOpen] = useState(false);
  const [editDeviceModalOpen, setEditDeviceModalOpen] = useState(false);
  const [editingDevice, setEditingDevice] = useState<Device | null>(null);
  const [selectedDeviceIds, setSelectedDeviceIds] = useState<React.Key[]>([]);
  const [targetGroupId, setTargetGroupId] = useState<string | null>(null);
  const [addForm] = Form.useForm<{ name: string; description: string }>();
  const [editForm] = Form.useForm<{ name: string; description: string }>();
  const [editDeviceForm] = Form.useForm<{
    engStatus: EngStatus;
    longitude: number;
    latitude: number;
    gpsHeight: number;
  }>();

  // 新增子分组弹窗相关状态
  const [addChildDrawerOpen, setAddChildDrawerOpen] = useState(false);
  const [parentGroupId, setParentGroupId] = useState<string | null>(null);
  const [addChildForm] = Form.useForm<{
    name: string;
    autoMatch: boolean;
    matchingMode: 'deviceName' | 'lac' | 'tac';
    tacRag: string;
  }>();
  const [nameFilters, setNameFilters] = useState<NameFilterItem[]>([
    { id: generateId(), condition: 'contain', value: '' },
  ]);

  // 监听匹配模式变化
  const matchingMode = Form.useWatch('matchingMode', addChildForm);

  // 设备安装状态选项
  const ENG_STATUS_OPTIONS = useMemo(() => [
    { label: t('device.engStatus.commissioned'), value: 'commissioned' },
    { label: t('device.engStatus.uncommissioned'), value: 'uncommissioned' },
    { label: t('device.engStatus.decommissioned'), value: 'decommissioned' },
  ], [t]);

  // 获取设备列表
  const queryParams = useMemo(
    () => ({ page: currentPage, pageSize, groupId: selectedGroupId ?? undefined } as Parameters<typeof useDeviceList>[0]),
    [currentPage, pageSize, selectedGroupId]
  );

  const { data: deviceData, isLoading, refetch } = useDeviceList(queryParams);
  const devices: Device[] = deviceData?.items ?? [];
  const total = deviceData?.total ?? 0;

  // 打开编辑设备弹窗
  const handleEditDevice = useCallback((device: Device) => {
    setEditingDevice(device);
    editDeviceForm.setFieldsValue({
      engStatus: device.engStatus,
      longitude: device.longitude,
      latitude: device.latitude,
      gpsHeight: device.gpsHeight,
    });
    setEditDeviceModalOpen(true);
  }, [editDeviceForm]);

  // 保存设备修改
  const handleSaveDevice = useCallback(async () => {
    await editDeviceForm.validateFields();
    // TODO: 调用 API 更新设备
    message.success(t('common.operationSuccess'));
    setEditDeviceModalOpen(false);
    await refetch();
  }, [editDeviceForm, message, t, refetch]);

  const selectedGroup = useMemo(
    () => groups.find((g) => g.id === selectedGroupId),
    [groups, selectedGroupId]
  );

  const handleContextMenu = useCallback(
    (action: string) => {
      const [cmd, groupId] = action.split(':');
      if (cmd === 'add-child') {
        // 打开新增子分组弹窗
        setParentGroupId(groupId);
        addChildForm.resetFields();
        addChildForm.setFieldsValue({
          autoMatch: false,
          matchingMode: 'deviceName',
          tacRag: '',
        });
        setNameFilters([{ id: generateId(), condition: 'contain', value: '' }]);
        setAddChildDrawerOpen(true);
      } else if (cmd === 'edit') {
        const grp = groups.find((g) => g.id === groupId);
        if (grp) {
          editForm.setFieldsValue({ name: grp.name, description: grp.description });
          setEditModalOpen(true);
        }
      } else if (cmd === 'delete') {
        modal.confirm({
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
    [groups, addChildForm, editForm, refetchGroups, t, modal, message]
  );

  const treeData = useMemo(
    () => buildTreeData(groups, selectedGroupId, handleContextMenu, t),
    [groups, selectedGroupId, handleContextMenu, t]
  );

  // 根据搜索文本过滤分组
  const filteredGroups = useMemo(() => {
    if (!groupSearchText.trim()) return groups;
    const searchLower = groupSearchText.toLowerCase();
    return groups.filter((g) => g.name.toLowerCase().includes(searchLower));
  }, [groups, groupSearchText]);

  const filteredTreeData = useMemo(
    () => buildTreeData(filteredGroups, selectedGroupId, handleContextMenu, t),
    [filteredGroups, selectedGroupId, handleContextMenu, t]
  );

  // 获取父级分组名称
  const getParentName = useCallback((parentId: string | null): string => {
    if (!parentId) return '';
    const parent = groups.find((g) => g.id === parentId);
    return parent?.name ?? '';
  }, [groups]);

  // 目标设备组选项（显示父级名称）
  const targetGroupOptions = useMemo(() => {
    return groups
      .filter((g) => g.parentId !== null)
      .map((g) => {
        const parentName = getParentName(g.parentId);
        return {
          label: parentName ? `${parentName} / ${g.name}` : g.name,
          value: g.id,
        };
      });
  }, [groups, getParentName]);

  const handleAddGroup = useCallback(async () => {
    const values = await addForm.validateFields();
    // In a real app, call API to create group
    message.success(`${values.name}`);
    setAddModalOpen(false);
    await refetchGroups();
  }, [addForm, refetchGroups, message]);

  const handleEditGroup = useCallback(async () => {
    const values = await editForm.validateFields();
    message.success(`${values.name}`);
    setEditModalOpen(false);
    await refetchGroups();
  }, [editForm, refetchGroups, message]);

  // 新增子分组相关处理函数
  const handleAddFilter = useCallback(() => {
    if (nameFilters.length >= 10) {
      void message.warning(t('device.rules.maxConditions', { max: 10 }));
      return;
    }
    const hasOr = nameFilters.some((f, index) => index > 0 && f.andOr === 'or');
    setNameFilters((prev) => [
      ...prev,
      {
        id: generateId(),
        condition: 'contain',
        value: '',
        andOr: hasOr ? 'or' : 'and',
      },
    ]);
  }, [nameFilters, message, t]);

  const handleRemoveFilter = useCallback((id: string) => {
    setNameFilters((prev) => {
      if (prev.length <= 1) return prev;
      const newFilters = prev.filter((f) => f.id !== id);
      if (newFilters.length > 0 && newFilters[0].andOr !== undefined) {
        const { andOr: _, ...rest } = newFilters[0];
        newFilters[0] = rest as NameFilterItem;
      }
      return newFilters;
    });
  }, []);

  const handleUpdateFilter = useCallback((id: string, field: keyof NameFilterItem, value: string) => {
    setNameFilters((prev) => prev.map((f) => (f.id === id ? { ...f, [field]: value } : f)));
  }, []);

  const handleMatchingModeChange = useCallback(() => {
    setNameFilters([{ id: generateId(), condition: 'contain', value: '' }]);
    addChildForm.setFieldsValue({ tacRag: '' });
  }, [addChildForm]);

  const handleSaveChildGroup = useCallback(async () => {
    try {
      const values = await addChildForm.validateFields();

      // 如果开启了自动匹配且是设备名称模式，验证过滤条件
      if (values.autoMatch && values.matchingMode === 'deviceName') {
        const validFilters = nameFilters.filter((f) => f.value?.trim());
        if (validFilters.length === 0) {
          void message.error(t('device.rules.atLeastOneFilter'));
          return;
        }
      }

      // 如果开启了自动匹配且是 TAC/LAC 模式，验证范围输入
      if (values.autoMatch && (values.matchingMode === 'tac' || values.matchingMode === 'lac')) {
        if (!values.tacRag?.trim()) {
          void message.error(t('device.rules.inputRange', { type: values.matchingMode === 'tac' ? 'TAC' : 'LAC' }));
          return;
        }
      }

      // TODO: 调用 API 创建子分组
      const operators = values.autoMatch
        ? generateOperators(
            {
              matchingMode: values.matchingMode,
              nameRuleList: values.matchingMode === 'deviceName' ? nameFilters : undefined,
              tacRag: values.matchingMode !== 'deviceName' ? values.tacRag : undefined,
            },
            t
          )
        : '';

      console.log('创建子分组:', {
        parentGroupId,
        name: values.name,
        autoMatch: values.autoMatch,
        matchingMode: values.matchingMode,
        nameFilters: values.autoMatch && values.matchingMode === 'deviceName' ? nameFilters : [],
        tacRag: values.autoMatch && values.matchingMode !== 'deviceName' ? values.tacRag : '',
        operators,
      });

      void message.success(t('common.success'));
      setAddChildDrawerOpen(false);
      await refetchGroups();
    } catch {
      // validation error
    }
  }, [addChildForm, nameFilters, parentGroupId, refetchGroups, message, t]);

  // 预览条件描述
  const previewText = useMemo(() => {
    if (matchingMode === 'deviceName') {
      return generateOperators({ matchingMode: 'deviceName', nameRuleList: nameFilters }, t);
    }
    return '';
  }, [matchingMode, nameFilters, t]);

  // 计算离线天数
  const calculateOfflineDays = useCallback((lastOnlineTime: string): number => {
    if (!lastOnlineTime) return 0;
    const lastOnline = new Date(lastOnlineTime);
    const now = new Date();
    const diffMs = now.getTime() - lastOnline.getTime();
    return Math.max(0, Math.floor(diffMs / (1000 * 60 * 60 * 24)));
  }, []);

  const columns = useMemo(
    (): DataTableColumn<Device>[] => [
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 80,
        fixed: 'left',
        render: (_val, record) => (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEditDevice(record)}
          >
            {t('common.edit')}
          </Button>
        ),
      },
      {
        key: 'engStatus',
        title: t('device.installStatus'),
        dataIndex: 'engStatus',
        width: 100,
        render: (val: EngStatus) => {
          const statusMap: Record<EngStatus, { label: string; color: string }> = {
            commissioned: { label: t('device.engStatus.commissioned'), color: 'green' },
            uncommissioned: { label: t('device.engStatus.uncommissioned'), color: 'orange' },
            decommissioned: { label: t('device.engStatus.decommissioned'), color: 'red' },
          };
          const { label, color } = statusMap[val] || { label: val, color: 'default' };
          return <Tag color={color}>{label}</Tag>;
        },
      },
      {
        key: 'sn',
        title: t('device.serialNumber'),
        dataIndex: 'sn',
        width: 150,
        mono: true,
        copyable: true,
      },
      { key: 'name', title: t('device.stationName'), dataIndex: 'name', width: 160, ellipsis: true },
      { key: 'macAddress', title: t('device.macAddress'), dataIndex: 'macAddress', width: 150, mono: true },
      { key: 'groupName', title: t('device.groupName'), dataIndex: 'groupName', width: 140, ellipsis: true },
      { key: 'longitude', title: t('device.longitude'), dataIndex: 'longitude', width: 100 },
      { key: 'latitude', title: t('device.latitude'), dataIndex: 'latitude', width: 100 },
      { key: 'gpsHeight', title: t('device.height'), dataIndex: 'gpsHeight', width: 80 },
      {
        key: 'offlineDays',
        title: t('device.offlineDays'),
        width: 100,
        render: (_val, record) => {
          if (record.connStatus === 'online') return '-';
          return calculateOfflineDays(record.lastOnlineTime);
        },
      },
    ],
    [t, calculateOfflineDays, handleEditDevice]
  );

  // 批量操作
  const batchActions = useMemo(() => [
    {
      key: 'moveToGroup',
      label: t('device.batch.moveToGroup'),
      icon: <MoveToGroupIcon />,
      onClick: (selectedKeys: React.Key[]) => {
        setSelectedDeviceIds(selectedKeys);
        setTargetGroupId(null);
        setMoveToGroupModalOpen(true);
      },
    },
    {
      key: 'recycle',
      label: t('device.batch.recycle'),
      icon: <RestOutlined />,
      onClick: (selectedKeys: React.Key[]) => {
        modal.confirm({
          title: t('device.batch.recycleConfirm'),
          content: t('device.batch.recycleMsg', { count: selectedKeys.length }),
          okText: t('common.confirm'),
          okType: 'danger',
          onOk: async () => {
            // TODO: 调用 API 批量回收
            message.success(t('common.success'));
            await refetch();
          },
        });
      },
    },
    {
      key: 'delete',
      label: t('common.delete'),
      icon: <DeleteOutlined />,
      danger: true,
      onClick: (selectedKeys: React.Key[]) => {
        modal.confirm({
          title: t('common.confirmDelete'),
          content: t('common.deleteConfirmMsg', { count: selectedKeys.length }),
          okText: t('common.confirmDelete'),
          okType: 'danger',
          onOk: async () => {
            // TODO: 调用 API 批量删除
            message.success(t('common.deleteSuccess'));
            await refetch();
          },
        });
      },
    },
  ], [t, modal, message, refetch]);

  const handleExport = useCallback((format: 'xlsx' | 'csv') => {
    // TODO: 实现导出功能
    message.success(`Export as ${format.toUpperCase()}`);
  }, [message]);

  const handleMoveToGroup = useCallback(async () => {
    if (!targetGroupId) {
      message.warning(t('device.batch.selectGroup'));
      return;
    }
    // TODO: 调用 API 移动设备到设备组
    message.success(t('common.success'));
    setMoveToGroupModalOpen(false);
    await refetch();
  }, [targetGroupId, message, t, refetch]);

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
            addForm.resetFields();
            setAddModalOpen(true);
          }}
        >
          {t('common.add')}
        </Button>
      </div>
      {/* 搜索框 */}
      <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
        <Input
          placeholder={t('device.searchGroup')}
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          value={groupSearchText}
          onChange={(e) => setGroupSearchText(e.target.value)}
          allowClear
          size="small"
        />
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
              children: filteredTreeData,
              selectable: false,
            },
          ]}
          defaultExpandAll
          selectedKeys={selectedGroupId ? [selectedGroupId] : []}
          onSelect={(keys) => {
            const key = keys[0] as string | undefined;
            if (!key) return;
            // 只有二级节点（有 parentId 的组）才能点击
            const clickedGroup = groups.find((g) => g.id === key);
            if (clickedGroup && clickedGroup.parentId !== null) {
              setSelectedGroupId(key);
              setCurrentPage(1);
            }
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
            batchActions={batchActions}
            onExport={handleExport}
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

      {/* Move to Group Modal */}
      <Modal
        title={t('device.batch.moveToGroup')}
        open={moveToGroupModalOpen}
        onOk={() => void handleMoveToGroup()}
        onCancel={() => setMoveToGroupModalOpen(false)}
        okText={t('common.confirm')}
      >
        <div style={{ marginTop: 16 }}>
          <Text type="secondary">
            {t('device.batch.selectedDevices', { count: selectedDeviceIds.length })}
          </Text>
          <Form.Item label={t('device.batch.targetGroup')} style={{ marginTop: 16 }}>
            <Select
              style={{ width: '100%' }}
              placeholder={t('device.batch.selectGroupPlaceholder')}
              value={targetGroupId}
              onChange={setTargetGroupId}
              options={targetGroupOptions}
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>
        </div>
      </Modal>

      {/* Edit Device Modal */}
      <Modal
        title={`${t('common.edit')} - ${editingDevice?.name ?? ''}`}
        open={editDeviceModalOpen}
        onOk={() => void handleSaveDevice()}
        onCancel={() => setEditDeviceModalOpen(false)}
        okText={t('common.save')}
      >
        <Form form={editDeviceForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="engStatus"
            label={t('device.installStatus')}
            rules={[{ required: true, message: t('common.pleaseSelect') }]}
          >
            <Select options={ENG_STATUS_OPTIONS} />
          </Form.Item>
          <Form.Item
            name="longitude"
            label={t('device.longitude')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <InputNumber style={{ width: '100%' }} precision={6} />
          </Form.Item>
          <Form.Item
            name="latitude"
            label={t('device.latitude')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <InputNumber style={{ width: '100%' }} precision={6} />
          </Form.Item>
          <Form.Item
            name="gpsHeight"
            label={t('device.height')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <InputNumber style={{ width: '100%' }} precision={1} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Add Child Group Drawer */}
      <Drawer
        title={t('device.addChildGroup')}
        open={addChildDrawerOpen}
        onClose={() => setAddChildDrawerOpen(false)}
        width={520}
        destroyOnClose
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={() => setAddChildDrawerOpen(false)}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={() => void handleSaveChildGroup()}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={addChildForm} layout="vertical">
          {/* 子分组名称 */}
          <Form.Item
            name="name"
            label={t('device.childGroupName')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <Input placeholder={t('common.placeholder')} maxLength={50} />
          </Form.Item>

          <Divider style={{ margin: '16px 0' }} />

          {/* 自动匹配到组开关 */}
          <Form.Item
            name="autoMatch"
            label={t('device.autoMatch')}
            valuePropName="checked"
            extra={t('device.autoMatchDesc')}
          >
            <Switch checkedChildren={t('common.enable')} unCheckedChildren={t('common.disable')} />
          </Form.Item>

          {/* 匹配规则（仅当开启自动匹配时显示） */}
          {addChildForm.getFieldValue('autoMatch') && (
            <>
              <div style={{ marginBottom: 8, fontWeight: 500, color: 'var(--color-text)' }}>
                {t('device.matchRule')}
              </div>
              <Form.Item name="matchingMode" label={t('device.rules.matchingMode')} rules={[{ required: true }]}>
                <Radio.Group onChange={handleMatchingModeChange}>
                  <Radio value="deviceName">{t('device.rules.deviceName')}</Radio>
                  <Radio value="lac">LAC</Radio>
                  <Radio value="tac">TAC</Radio>
                </Radio.Group>
              </Form.Item>

              {/* 设备名称过滤条件 */}
              {matchingMode === 'deviceName' && (
                <>
                  <Form.Item
                    label={
                      <span>
                        {t('device.rules.filterCondition')}
                        <Text type="secondary" style={{ fontSize: 12, marginLeft: 4 }}>
                          {t('device.rules.conditionLimit', { max: 10 })}
                        </Text>
                      </span>
                    }
                  >
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                      {nameFilters.map((filter, index) => (
                        <div key={filter.id} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                          {index === 0 ? (
                            <>
                              <Select
                                value={filter.condition}
                                style={{ width: 120 }}
                                options={getFilterConditionOptions(t)}
                                onChange={(v) => handleUpdateFilter(filter.id, 'condition', v)}
                              />
                              <Input
                                value={filter.value}
                                style={{ flex: 1 }}
                                maxLength={64}
                                placeholder={t('common.placeholder')}
                                onChange={(e) => handleUpdateFilter(filter.id, 'value', e.target.value)}
                              />
                            </>
                          ) : (
                            <>
                              <Select
                                value={filter.andOr || 'and'}
                                style={{ width: 70 }}
                                options={getAndOrOptions(t)}
                                onChange={(v) => handleUpdateFilter(filter.id, 'andOr', v)}
                              />
                              <Select
                                value={filter.condition}
                                style={{ width: 120 }}
                                options={getFilterConditionOptions(t)}
                                onChange={(v) => handleUpdateFilter(filter.id, 'condition', v)}
                              />
                              <Input
                                value={filter.value}
                                style={{ flex: 1 }}
                                maxLength={64}
                                placeholder={t('common.placeholder')}
                                onChange={(e) => handleUpdateFilter(filter.id, 'value', e.target.value)}
                              />
                              <Button
                                type="text"
                                size="small"
                                icon={<CloseCircleOutlined />}
                                onClick={() => handleRemoveFilter(filter.id)}
                                style={{ color: 'var(--color-text-quaternary)' }}
                              />
                            </>
                          )}
                        </div>
                      ))}
                    </div>
                    {nameFilters.length < 10 && (
                      <Button type="dashed" icon={<PlusOutlined />} onClick={handleAddFilter} style={{ marginTop: 8 }}>
                        {t('device.rules.addCondition')}
                      </Button>
                    )}
                  </Form.Item>

                  {/* 预览条件描述 */}
                  {previewText && (
                    <div
                      style={{
                        color: 'var(--color-text-tertiary)',
                        fontSize: 12,
                        marginBottom: 16,
                        padding: '8px 12px',
                        background: 'var(--color-fill-quaternary)',
                        borderRadius: 4,
                        wordBreak: 'break-all',
                      }}
                    >
                      {previewText}
                    </div>
                  )}
                </>
              )}

              {/* TAC/LAC 输入 */}
              {(matchingMode === 'tac' || matchingMode === 'lac') && (
                <Form.Item
                  name="tacRag"
                  label={matchingMode === 'tac' ? 'TAC' : 'LAC'}
                  rules={[{ required: true, message: t('device.rules.inputRange', { type: matchingMode === 'tac' ? 'TAC' : 'LAC' }) }]}
                  extra={
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {t('device.rules.formatRange', { range: '0-65535' })}
                    </Text>
                  }
                >
                  <Input placeholder="eg: 1,2,3,1-3" maxLength={50} />
                </Form.Item>
              )}
            </>
          )}
        </Form>
      </Drawer>
    </>
  );
}
