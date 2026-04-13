import { useState, useMemo, useCallback } from 'react';
import {
  App,
  Button,
  Tag,
  Drawer,
  Form,
  Input,
  Dropdown,
  Tree,
  Space,
  Select,
  Checkbox,
  Divider,
  Spin,
  Empty,
  Tabs,
} from 'antd';
import type { TreeDataNode, TreeProps } from 'antd';
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
import type { DeviceGroup } from '@/types/device';
import { useT } from '@/hooks/useT';

// 操作权限类型
interface OperationItem {
  key: string;
  titleKey: string;
}

// 权限子菜单项定义（二级菜单 + 三级操作）
interface PermissionSubItem {
  key: string;
  titleKey: string;
  operations?: OperationItem[]; // 三级操作权限
}

// 权限模块定义（一级菜单 + 二级菜单）
interface PermissionModule {
  key: string;
  titleKey: string;
  children: PermissionSubItem[];
}

// 默认的操作权限（查询、新增、修改、删除、导出、批量）
const DEFAULT_OPERATIONS: OperationItem[] = [
  { key: 'query', titleKey: 'role.operation.query' },
  { key: 'add', titleKey: 'role.operation.add' },
  { key: 'edit', titleKey: 'role.operation.edit' },
  { key: 'delete', titleKey: 'role.operation.delete' },
  { key: 'export', titleKey: 'role.operation.export' },
  { key: 'batch', titleKey: 'role.operation.batch' },
];

// 完整的权限模块结构（一级 + 二级菜单）
const PERMISSION_MODULES: PermissionModule[] = [
  {
    key: 'device',
    titleKey: 'role.modules.device',
    children: [
      { key: 'list', titleKey: 'nav.device.list' },
      { key: 'register', titleKey: 'nav.device.register' },
      { key: 'group', titleKey: 'nav.device.group' },
      { key: 'detail', titleKey: 'nav.device.detail' },
      { key: 'ne', titleKey: 'nav.device.ne' },
      { key: 'monitor', titleKey: 'nav.device.monitor' },
      { key: 'commission', titleKey: 'nav.device.commission' },
      { key: 'stats', titleKey: 'nav.device.stats' },
      { key: 'import', titleKey: 'nav.device.import' },
      { key: 'rules', titleKey: 'nav.device.rules' },
    ],
  },
  {
    key: 'alarm',
    titleKey: 'role.modules.alarm',
    children: [
      { key: 'current', titleKey: 'nav.alarm.current' },
      { key: 'history', titleKey: 'nav.alarm.history' },
      { key: 'statistics', titleKey: 'nav.alarm.statistics' },
      { key: 'rules', titleKey: 'nav.alarm.rules' },
      { key: 'library', titleKey: 'nav.alarm.library' },
      { key: 'sync', titleKey: 'nav.alarm.sync' },
    ],
  },
  {
    key: 'performance',
    titleKey: 'role.modules.performance',
    children: [
      { key: 'kpiStandard', titleKey: 'nav.performance.kpiStandard' },
      { key: 'kpiStation', titleKey: 'nav.performance.kpiStation' },
      { key: 'extraction', titleKey: 'nav.performance.extraction' },
      { key: 'charts', titleKey: 'nav.performance.charts' },
      { key: 'threshold', titleKey: 'nav.performance.threshold' },
      { key: 'files', titleKey: 'nav.performance.files' },
      { key: 'taskConfig', titleKey: 'nav.performance.taskConfig' },
    ],
  },
  {
    key: 'software',
    titleKey: 'role.modules.software',
    children: [
      { key: 'version', titleKey: 'nav.software.version' },
      { key: 'upgradePlan', titleKey: 'nav.software.upgradePlan' },
      { key: 'activation', titleKey: 'nav.software.activation' },
      { key: 'firmware', titleKey: 'nav.software.firmware' },
    ],
  },
  {
    key: 'file',
    titleKey: 'role.modules.file',
    children: [
      { key: 'configRetrieval', titleKey: 'nav.file.configRetrieval' },
      { key: 'configDistribution', titleKey: 'nav.file.configDistribution' },
      { key: 'logRetrieval', titleKey: 'nav.file.logRetrieval' },
      { key: 'perfRetrieval', titleKey: 'nav.file.perfRetrieval' },
      { key: 'mrRetrieval', titleKey: 'nav.file.mrRetrieval' },
      { key: 'userFiles', titleKey: 'nav.file.userFiles' },
      { key: 'deviceFiles', titleKey: 'nav.file.deviceFiles' },
    ],
  },
  {
    key: 'log',
    titleKey: 'role.modules.log',
    children: [
      { key: 'device', titleKey: 'nav.log.device' },
      { key: 'exception', titleKey: 'nav.log.exception' },
      { key: 'event', titleKey: 'nav.log.event' },
      { key: 'operation', titleKey: 'nav.log.operation' },
      { key: 'system', titleKey: 'nav.log.system' },
      { key: 'config', titleKey: 'nav.log.config' },
    ],
  },
  {
    key: 'system',
    titleKey: 'role.modules.system',
    children: [
      { key: 'deviceClass', titleKey: 'nav.system.deviceClass' },
      { key: 'users', titleKey: 'nav.system.users' },
      { key: 'groups', titleKey: 'nav.system.groups' },
      { key: 'roles', titleKey: 'nav.system.roles' },
      { key: 'menus', titleKey: 'nav.system.menus' },
      { key: 'operationLog', titleKey: 'nav.system.operationLog' },
      { key: 'config', titleKey: 'nav.system.config' },
      { key: 'dataDict', titleKey: 'nav.system.dataDict' },
    ],
  },
  {
    key: 'report',
    titleKey: 'role.modules.report',
    children: [
      { key: 'lteStandard', titleKey: 'nav.report.lteStandard' },
      { key: 'station', titleKey: 'nav.report.station' },
      { key: 'historicalKpi', titleKey: 'nav.report.historicalKpi' },
      { key: 'pollStats', titleKey: 'nav.report.pollStats' },
    ],
  },
  {
    key: 'ops',
    titleKey: 'role.modules.ops',
    children: [
      { key: 'templates', titleKey: 'nav.ops.templates' },
      { key: 'commands', titleKey: 'nav.ops.commands' },
      { key: 'tasks', titleKey: 'nav.ops.tasks' },
      { key: 'networkDiagnosis', titleKey: 'nav.ops.networkDiagnosis' },
      { key: 'downloads', titleKey: 'nav.ops.downloads' },
    ],
  },
];

// 基站制式选项
const NETWORK_TYPE_OPTIONS = [
  { label: '全部', value: '' },
  { label: 'eNB', value: 'eNB' },
  { label: 'gNB', value: 'gNB' },
  { label: 'GSM', value: 'GSM' },
];

// 产品类型选项
const PRODUCT_TYPE_OPTIONS = [
  { label: '全部', value: '' },
  { label: '宏基站', value: 'Macro' },
  { label: '小基站', value: 'Small Cell' },
  { label: '皮基站', value: 'Pico' },
];

// 构建设备组树形数据（带筛选）
const buildDeviceGroupTreeData = (
  groups: DeviceGroup[],
  networkTypeFilter?: string,
  productTypeFilter?: string
): TreeDataNode[] => {
  // 筛选二级节点
  let filteredChildGroups = groups.filter((g) => g.parentId);

  // 应用筛选条件
  if (networkTypeFilter) {
    filteredChildGroups = filteredChildGroups.filter(
      (g) => g.networkType === networkTypeFilter || !g.networkType
    );
  }
  if (productTypeFilter) {
    filteredChildGroups = filteredChildGroups.filter(
      (g) => g.productType === productTypeFilter || !g.productType
    );
  }

  // 一级节点
  const rootGroups = groups.filter((g) => !g.parentId);

  return rootGroups
    .map((root) => {
      const children = filteredChildGroups
        .filter((child) => child.parentId === root.id)
        .map((child) => ({
          key: child.id,
          title: child.name,
        }));

      // 如果没有子节点且不是自定义组，则不显示
      if (children.length === 0 && root.builtIn === 1) {
        return null;
      }

      return {
        key: root.id,
        title: root.name,
        children,
      };
    })
    .filter((node): node is TreeDataNode => node !== null);
};

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
  // 当前激活的标签页
  const [activeTab, setActiveTab] = useState<string>('menu');
  // 权限配置：选中的权限key列表
  const [checkedPermissionKeys, setCheckedPermissionKeys] = useState<React.Key[]>([]);
  // 权限树展开的节点
  const [expandedPermissionKeys, setExpandedPermissionKeys] = useState<React.Key[]>([]);
  // 父子联动开关
  const [permissionCheckStrictly, setPermissionCheckStrictly] = useState(true);
  // 设备组选择
  const [selectedDeviceGroupIds, setSelectedDeviceGroupIds] = useState<string[]>([]);
  // 设备组筛选条件
  const [deviceGroupNetworkType, setDeviceGroupNetworkType] = useState<string>('');
  const [deviceGroupProductType, setDeviceGroupProductType] = useState<string>('');


  const { data, isLoading, refetch } = useRoles({
    roleName: filters.roleName as string | undefined,
    page,
    pageSize,
  });

  const { data: allDeviceGroups, isLoading: isLoadingDeviceGroups } = useAllDeviceGroups();

  // 设备组树形数据（一级+二级节点，支持筛选）
  const deviceGroupTreeData = useMemo(
    () => buildDeviceGroupTreeData(allDeviceGroups ?? [], deviceGroupNetworkType, deviceGroupProductType),
    [allDeviceGroups, deviceGroupNetworkType, deviceGroupProductType]
  );

  // 筛选后的二级节点ID列表
  const filteredSecondLevelIds = useMemo(() => {
    let filtered = (allDeviceGroups ?? []).filter((g) => g.parentId);
    if (deviceGroupNetworkType) {
      filtered = filtered.filter((g) => g.networkType === deviceGroupNetworkType || !g.networkType);
    }
    if (deviceGroupProductType) {
      filtered = filtered.filter((g) => g.productType === deviceGroupProductType || !g.productType);
    }
    return filtered.map((g) => g.id);
  }, [allDeviceGroups, deviceGroupNetworkType, deviceGroupProductType]);

  // 筛选后的一级节点ID列表（用于全选时同时选中父节点）
  const filteredFirstLevelIds = useMemo(() => {
    const secondLevelNodes = (allDeviceGroups ?? []).filter((g) =>
      filteredSecondLevelIds.includes(g.id)
    );
    const parentIds = new Set(secondLevelNodes.map((g) => g.parentId).filter(Boolean));
    return Array.from(parentIds) as string[];
  }, [allDeviceGroups, filteredSecondLevelIds]);

  // 所有二级节点 ID（用于验证是否至少选择了一个）
  const allSecondLevelIds = useMemo(
    () => (allDeviceGroups ?? []).filter((g) => g.parentId).map((g) => g.id),
    [allDeviceGroups]
  );

  const createRole = useCreateRole();
  const updateRole = useUpdateRole();
  const deleteRoles = useDeleteRoles();

  const isBuiltIn = useCallback((role: Role) => role.builtIn === 1 || role.builtIn === 2, []);

  // 已有的角色名称列表（用于重复检查）
  const existingRoleNames = useMemo(
    () => (data?.items ?? []).map((r) => r.roleName.toLowerCase()),
    [data?.items]
  );

  // 获取所有权限项的 key（格式：module.subItem.operation）- 仅叶子节点（三级操作）
  const allPermissionLeafKeys = useMemo(() => {
    const keys: string[] = [];
    for (const module of PERMISSION_MODULES) {
      for (const child of module.children) {
        const operations = child.operations || DEFAULT_OPERATIONS;
        for (const op of operations) {
          keys.push(`${module.key}.${child.key}.${op.key}`);
        }
      }
    }
    return keys;
  }, []);

  // 获取所有二级菜单的 key（用于展开）
  const allSecondLevelKeys = useMemo(() => {
    const keys: string[] = [];
    for (const module of PERMISSION_MODULES) {
      for (const child of module.children) {
        keys.push(`${module.key}.${child.key}`);
      }
    }
    return keys;
  }, []);

  // 获取所有一级模块的 key
  const allModuleKeys = useMemo(() => {
    return PERMISSION_MODULES.map((m) => m.key);
  }, []);

  // 获取所有节点的 key（一级 + 二级 + 三级）- 用于全选
  const allPermissionKeys = useMemo(() => {
    const keys: string[] = [];
    for (const module of PERMISSION_MODULES) {
      // 一级节点
      keys.push(module.key);
      for (const child of module.children) {
        // 二级节点
        keys.push(`${module.key}.${child.key}`);
        // 三级节点（操作）
        const operations = child.operations || DEFAULT_OPERATIONS;
        for (const op of operations) {
          keys.push(`${module.key}.${child.key}.${op.key}`);
        }
      }
    }
    return keys;
  }, []);

  // 构建菜单权限树形数据（三级结构）
  const permissionTreeData = useMemo((): TreeDataNode[] => {
    return PERMISSION_MODULES.map((module) => ({
      key: module.key,
      title: t(module.titleKey),
      children: module.children.map((child) => ({
        key: `${module.key}.${child.key}`,
        title: t(child.titleKey),
        children: (child.operations || DEFAULT_OPERATIONS).map((op) => ({
          key: `${module.key}.${child.key}.${op.key}`,
          title: t(op.titleKey),
          isLeaf: true,
        })),
      })),
    }));
  }, [t]);

  // 将 checkedPermissionKeys 转换为 permissions 数组（用于提交）
  // 格式：["device.list", "alarm.current", ...]
  const permissionsToArray = useCallback((keys: React.Key[]): string[] => {
    // 只返回叶子节点的 key
    return keys.filter((k) => allPermissionLeafKeys.includes(k as string)) as string[];
  }, [allPermissionLeafKeys]);

  // 将 permissions 数组转换为 checkedPermissionKeys
  const arrayToCheckedKeys = useCallback((permissions: string[]): React.Key[] => {
    // 直接返回权限列表作为选中的 key
    return permissions.filter((p) => allPermissionLeafKeys.includes(p));
  }, [allPermissionLeafKeys]);

  // 检查是否至少选择了一个权限
  const hasAnyPermission = useMemo(
    () => checkedPermissionKeys.some((k) => allPermissionLeafKeys.includes(k as string)),
    [checkedPermissionKeys, allPermissionLeafKeys]
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
    if (!hasAnyPermission) {
      message.warning(t('role.pleaseSelectPermission'));
      return;
    }
    // 校验设备组（至少选择一个二级节点）
    const selectedSecondLevel = selectedDeviceGroupIds.filter((id) => allSecondLevelIds.includes(id));
    if (selectedSecondLevel.length === 0) {
      message.warning(t('role.pleaseSelectDeviceGroup'));
      return;
    }

    form.validateFields().then((vals) => {
      createRole.mutate(
        {
          roleName: vals.roleName as string,
          description: (vals.description as string) ?? '',
          permissions: permissionsToArray(checkedPermissionKeys),
          deviceGroupIds: selectedDeviceGroupIds,
          builtIn: 0,
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setCreateVisible(false);
            form.resetFields();
            setCheckedPermissionKeys([]);
            setExpandedPermissionKeys([]);
            setSelectedDeviceGroupIds([]);
          },
        },
      );
    });
  }, [form, createRole, checkedPermissionKeys, selectedDeviceGroupIds, hasAnyPermission, allSecondLevelIds, permissionsToArray, message, t]);

  // 校验并提交编辑
  const handleEdit = useCallback(() => {
    if (!selectedRole) return;

    // 校验权限
    if (!hasAnyPermission) {
      message.warning(t('role.pleaseSelectPermission'));
      return;
    }
    // 校验设备组（至少选择一个二级节点）
    const selectedSecondLevel = selectedDeviceGroupIds.filter((id) => allSecondLevelIds.includes(id));
    if (selectedSecondLevel.length === 0) {
      message.warning(t('role.pleaseSelectDeviceGroup'));
      return;
    }

    form.validateFields().then((vals) => {
      updateRole.mutate(
        {
          id: selectedRole.id,
          data: {
            roleName: vals.roleName as string,
            description: vals.description as string,
            permissions: permissionsToArray(checkedPermissionKeys),
            deviceGroupIds: selectedDeviceGroupIds,
          },
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setEditVisible(false);
            form.resetFields();
            setSelectedRole(null);
            setCheckedPermissionKeys([]);
            setExpandedPermissionKeys([]);
            setSelectedDeviceGroupIds([]);
          },
        },
      );
    });
  }, [selectedRole, form, updateRole, checkedPermissionKeys, selectedDeviceGroupIds, hasAnyPermission, allSecondLevelIds, permissionsToArray, message, t]);

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
                      description: role.description,
                    });
                    setCheckedPermissionKeys(arrayToCheckedKeys(role.permissions || []));
                    setExpandedPermissionKeys(allModuleKeys);
                    setSelectedDeviceGroupIds(role.deviceGroupIds || []);
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
                      description: role.description,
                    });
                    setCheckedPermissionKeys(arrayToCheckedKeys(role.permissions || []));
                    setExpandedPermissionKeys(allModuleKeys);
                    setSelectedDeviceGroupIds(role.deviceGroupIds || []);
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
      width: 150,
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
    { key: 'description', title: t('role.roleDescription'), dataIndex: 'description', width: 150, ellipsis: true, render: (v) => v || '-' },
    { key: 'createUser', title: t('role.createUser'), dataIndex: 'createUser', width: 100, render: (v) => v || '-' },
    {
      key: 'createTime',
      title: t('role.createTime'),
      dataIndex: 'createTime',
      width: 160,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '-'),
    },
    { key: 'updateUser', title: t('role.updateUser'), dataIndex: 'updateUser', width: 100, render: (v) => v || '-' },
    {
      key: 'updateTime',
      title: t('role.updateTime'),
      dataIndex: 'updateTime',
      width: 160,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '-'),
    },
  ], [t, form, isBuiltIn, handleDelete, allModuleKeys, arrayToCheckedKeys]);

  // 渲染菜单权限配置（树形结构 - 按图片样式）
  const renderPermissionConfig = (readOnly = false) => {
    // 是否全部展开（一级 + 二级都要展开）
    const isAllExpanded = expandedPermissionKeys.length >= allModuleKeys.length + allSecondLevelKeys.length;

    // 展开/折叠所有（复选框）
    const handleExpandChange = (checked: boolean) => {
      if (checked) {
        // 展开所有一级和二级节点
        setExpandedPermissionKeys([...allModuleKeys, ...allSecondLevelKeys]);
      } else {
        setExpandedPermissionKeys([]);
      }
    };

    // 全选/全不选（复选框）- 控制所有节点
    const handleSelectAllChange = (checked: boolean) => {
      if (checked) {
        setCheckedPermissionKeys(allPermissionKeys);
      } else {
        setCheckedPermissionKeys([]);
      }
    };

    // 父子联动（复选框）- 勾选表示联动，不勾选表示不联动
    const handleLinkageChange = (checked: boolean) => {
      setPermissionCheckStrictly(!checked); // checkStrictly=false 表示联动
    };

    // 获取当前选中的节点数量（用于全选状态计算）- 统计所有节点
    const checkedCount = checkedPermissionKeys.filter((k) =>
      allPermissionKeys.includes(k as string)
    ).length;

    // 是否全选
    const isAllSelected = checkedCount === allPermissionKeys.length && allPermissionKeys.length > 0;
    // 是否部分选中
    const isIndeterminate = checkedCount > 0 && checkedCount < allPermissionKeys.length;

    // 处理树节点选中
    const handleCheck: TreeProps['onCheck'] = (checked) => {
      // 当 checkStrictly 为 true 时，checked 是 { checked: [], halfChecked: [] } 对象
      // 当 checkStrictly 为 false 时，checked 是数组
      if (Array.isArray(checked)) {
        setCheckedPermissionKeys(checked);
      } else {
        setCheckedPermissionKeys(checked.checked);
      }
    };

    // 处理展开/折叠
    const handleExpand: TreeProps['onExpand'] = (expanded) => {
      setExpandedPermissionKeys(expanded as React.Key[]);
    };

    // checkedKeys 格式：当 checkStrictly 为 true 时需要传对象，否则传数组
    const treeCheckedKeys = permissionCheckStrictly
      ? { checked: checkedPermissionKeys, halfChecked: [] }
      : checkedPermissionKeys;

    return (
      <Form.Item
        label={t('role.menuPermission')}
        required={!readOnly}
        help={!readOnly && !hasAnyPermission ? t('role.pleaseSelectPermission') : undefined}
        validateStatus={!readOnly && !hasAnyPermission ? 'warning' : undefined}
      >
        <div style={{ border: '1px solid var(--color-border)', borderRadius: 6 }}>
          {/* 顶部操作按钮区域 */}
          <div
            style={{
              padding: '8px 12px',
              borderBottom: '1px solid var(--color-border)',
              background: 'var(--color-fill-quaternary)',
              display: 'flex',
              alignItems: 'center',
              gap: 16,
              flexWrap: 'wrap',
            }}
          >
            <Checkbox
              checked={isAllExpanded}
              onChange={(e) => handleExpandChange(e.target.checked)}
              disabled={readOnly}
            >
              {t('role.expandCollapse')}
            </Checkbox>
            <Checkbox
              checked={isAllSelected}
              indeterminate={isIndeterminate}
              onChange={(e) => handleSelectAllChange(e.target.checked)}
              disabled={readOnly}
            >
              {t('role.selectAllOrNone')}
            </Checkbox>
            <Checkbox
              checked={!permissionCheckStrictly}
              onChange={(e) => handleLinkageChange(e.target.checked)}
              disabled={readOnly}
            >
              {t('role.parentChildLinkage')}
            </Checkbox>
          </div>
          {/* 树形选择区域 */}
          <div style={{ padding: 8, maxHeight: 400, overflow: 'auto' }}>
            {readOnly ? (
              <Tree
                treeData={permissionTreeData}
                expandedKeys={expandedPermissionKeys}
                checkedKeys={treeCheckedKeys}
                selectable={false}
                checkable
                disabled
                onExpand={handleExpand}
              />
            ) : (
              <Tree
                treeData={permissionTreeData}
                expandedKeys={expandedPermissionKeys}
                checkedKeys={treeCheckedKeys}
                selectable={false}
                checkable
                checkStrictly={permissionCheckStrictly}
                onExpand={handleExpand}
                onCheck={handleCheck}
              />
            )}
          </div>
        </div>
      </Form.Item>
    );
  };

  // 渲染设备组树形选择
  const renderDeviceGroupTree = (readOnly = false) => {
    // 只检查二级节点是否被选中（基于筛选后的数据）
    const selectedSecondLevelCount = selectedDeviceGroupIds.filter((id) =>
      filteredSecondLevelIds.includes(id)
    ).length;

    // 全选/取消全选
    const handleSelectAll = (checked: boolean) => {
      if (checked) {
        // 选中所有筛选后的节点（包括一级和二级）
        setSelectedDeviceGroupIds((prev) => {
          const newIds = new Set(prev);
          // 添加一级节点
          filteredFirstLevelIds.forEach((id) => newIds.add(id));
          // 添加二级节点
          filteredSecondLevelIds.forEach((id) => newIds.add(id));
          return Array.from(newIds);
        });
      } else {
        // 取消选中所有筛选后的节点（包括一级和二级）
        setSelectedDeviceGroupIds((prev) =>
          prev.filter((id) => !filteredSecondLevelIds.includes(id) && !filteredFirstLevelIds.includes(id))
        );
      }
    };

    // 是否全选（基于筛选后的数据）
    const isAllSelected =
      filteredSecondLevelIds.length > 0 &&
      filteredSecondLevelIds.every((id) => selectedDeviceGroupIds.includes(id));

    // 是否部分选中
    const isIndeterminate =
      selectedSecondLevelCount > 0 && selectedSecondLevelCount < filteredSecondLevelIds.length;

    return (
      <Form.Item
        label={t('role.dataPermission')}
        required={!readOnly}
        help={!readOnly && selectedSecondLevelCount === 0 ? t('role.pleaseSelectDeviceGroup') : undefined}
        validateStatus={!readOnly && selectedSecondLevelCount === 0 ? 'warning' : undefined}
      >
        {readOnly ? (
          <Space wrap>
            {selectedDeviceGroupIds.length > 0 ? (
              (allDeviceGroups ?? [])
                .filter((g) => selectedDeviceGroupIds.includes(g.id))
                .map((g) => <Tag key={g.id}>{g.name}</Tag>)
            ) : (
              <span style={{ color: 'var(--color-text-secondary)' }}>-</span>
            )}
          </Space>
        ) : (
          <div style={{ border: '1px solid var(--color-border)', borderRadius: 6 }}>
            {/* 筛选区域 */}
            <div
              style={{
                padding: '8px 12px',
                borderBottom: '1px solid var(--color-border)',
                background: 'var(--color-fill-quaternary)',
                display: 'flex',
                alignItems: 'center',
                gap: 12,
                flexWrap: 'wrap',
              }}
            >
              <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>{t('role.filter')}</span>
              <Select
                size="small"
                style={{ width: 120 }}
                value={deviceGroupNetworkType}
                onChange={(val) => setDeviceGroupNetworkType(val)}
                options={NETWORK_TYPE_OPTIONS}
                placeholder={t('role.baseStationType')}
              />
              <Select
                size="small"
                style={{ width: 120 }}
                value={deviceGroupProductType}
                onChange={(val) => setDeviceGroupProductType(val)}
                options={PRODUCT_TYPE_OPTIONS}
                placeholder={t('role.productType')}
              />
              <Divider type="vertical" style={{ height: 20, margin: 0 }} />
              <Checkbox
                checked={isAllSelected}
                indeterminate={isIndeterminate}
                onChange={(e) => handleSelectAll(e.target.checked)}
              >
                {t('role.selectAll')}
              </Checkbox>
            </div>
            {/* 树形选择区域 */}
            <div style={{ padding: 8, maxHeight: 280, overflow: 'auto' }}>
              {isLoadingDeviceGroups ? (
                <div style={{ textAlign: 'center', padding: 24 }}>
                  <Spin />
                </div>
              ) : deviceGroupTreeData.length === 0 ? (
                <Empty description={t('role.noDeviceGroupData')} />
              ) : (
                <Tree
                  checkable
                  checkedKeys={selectedDeviceGroupIds}
                  treeData={deviceGroupTreeData}
                  defaultExpandAll
                  onCheck={(checked) => {
                    setSelectedDeviceGroupIds(checked as string[]);
                  }}
                  selectable={false}
                />
              )}
            </div>
          </div>
        )}
      </Form.Item>
    );
  };

  // 关闭抽屉时重置状态
  const handleCloseCreate = useCallback(() => {
    setCreateVisible(false);
    form.resetFields();
    setCheckedPermissionKeys([]);
    setExpandedPermissionKeys([]);
    setSelectedDeviceGroupIds([]);
    setActiveTab('menu');
  }, [form]);

  const handleCloseEdit = useCallback(() => {
    setEditVisible(false);
    form.resetFields();
    setSelectedRole(null);
    setCheckedPermissionKeys([]);
    setExpandedPermissionKeys([]);
    setSelectedDeviceGroupIds([]);
    setActiveTab('menu');
  }, [form]);

  const handleCloseView = useCallback(() => {
    setViewVisible(false);
    form.resetFields();
    setSelectedRole(null);
    setCheckedPermissionKeys([]);
    setExpandedPermissionKeys([]);
    setSelectedDeviceGroupIds([]);
    setActiveTab('menu');
  }, [form]);

  return (
    <ListPageLayout
      title={t('nav.system.roles')}
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
        width={600}
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
          <Form.Item name="description" label={t('role.description')}>
            <Input.TextArea
              rows={2}
              placeholder={t('role.descriptionPlaceholder')}
              maxLength={500}
              showCount
            />
          </Form.Item>
        </Form>

        {/* 标签页：角色菜单、角色API、资源权限 */}
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'menu',
              label: t('role.menuPermission'),
              children: renderPermissionConfig(false),
            },
            {
              key: 'api',
              label: t('role.apiPermission'),
              children: (
                <div style={{ padding: '24px 0', textAlign: 'center', color: 'var(--color-text-secondary)' }}>
                  <Empty description={t('role.apiPermissionComingSoon')} />
                </div>
              ),
            },
            {
              key: 'resource',
              label: t('role.resourcePermission'),
              children: renderDeviceGroupTree(false),
            },
          ]}
        />
      </Drawer>

      {/* Edit Drawer */}
      <Drawer
        title={t('common.edit')}
        open={editVisible}
        onClose={handleCloseEdit}
        width={600}
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
          <Form.Item name="description" label={t('role.description')}>
            <Input.TextArea
              rows={2}
              placeholder={t('role.descriptionPlaceholder')}
              maxLength={500}
              showCount
            />
          </Form.Item>
        </Form>

        {/* 标签页：角色菜单、角色API、资源权限 */}
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'menu',
              label: t('role.menuPermission'),
              children: renderPermissionConfig(false),
            },
            {
              key: 'api',
              label: t('role.apiPermission'),
              children: (
                <div style={{ padding: '24px 0', textAlign: 'center', color: 'var(--color-text-secondary)' }}>
                  <Empty description={t('role.apiPermissionComingSoon')} />
                </div>
              ),
            },
            {
              key: 'resource',
              label: t('role.resourcePermission'),
              children: renderDeviceGroupTree(false),
            },
          ]}
        />
      </Drawer>

      {/* View Drawer */}
      <Drawer
        title={t('common.view')}
        open={viewVisible}
        onClose={handleCloseView}
        width={600}
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
          <Form.Item name="description" label={t('role.description')}>
            <Input.TextArea rows={2} readOnly />
          </Form.Item>
        </Form>

        {/* 标签页：角色菜单、角色API、资源权限 */}
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'menu',
              label: t('role.menuPermission'),
              children: renderPermissionConfig(true),
            },
            {
              key: 'api',
              label: t('role.apiPermission'),
              children: (
                <div style={{ padding: '24px 0', textAlign: 'center', color: 'var(--color-text-secondary)' }}>
                  <Empty description={t('role.apiPermissionComingSoon')} />
                </div>
              ),
            },
            {
              key: 'resource',
              label: t('role.resourcePermission'),
              children: renderDeviceGroupTree(true),
            },
          ]}
        />

        <Divider />
        <Form.Item label={t('role.userCount')}>
          <span>{selectedRole?.userCount ?? 0}</span>
        </Form.Item>
        <Form.Item label={t('role.createUser')}>
          <span>{selectedRole?.createUser ?? '-'}</span>
        </Form.Item>
        <Form.Item label={t('role.createTime')}>
          <span>{selectedRole?.createTime ? new Date(selectedRole.createTime).toLocaleString('zh-CN') : '-'}</span>
        </Form.Item>
        <Form.Item label={t('role.updateUser')}>
          <span>{selectedRole?.updateUser ?? '-'}</span>
        </Form.Item>
        <Form.Item label={t('role.updateTime')}>
          <span>{selectedRole?.updateTime ? new Date(selectedRole.updateTime).toLocaleString('zh-CN') : '-'}</span>
        </Form.Item>
      </Drawer>
    </ListPageLayout>
  );
}
