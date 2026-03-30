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
  Radio,
  Space,
  Menu,
  Select,
  Checkbox,
  Divider,
} from 'antd';
import type { TreeDataNode, MenuProps } from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  MoreOutlined,
  DownOutlined,
  RightOutlined,
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

// 权限类型（分别跟踪只读和读写）
interface PermissionState {
  read: boolean;   // 只读权限
  write: boolean;  // 读写权限
}

// 权限子菜单项定义
interface PermissionSubItem {
  key: string;
  titleKey: string;
}

// 权限模块定义（一级菜单 + 二级菜单）
interface PermissionModule {
  key: string;
  titleKey: string;
  children: PermissionSubItem[];
}

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
      { key: 'operationLog', titleKey: 'nav.system.operationLog' },
      { key: 'config', titleKey: 'nav.system.config' },
      { key: 'dataDict', titleKey: 'nav.system.dataDict' },
      { key: 'notifications', titleKey: 'nav.system.notifications' },
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
  { label: 'LTE (4G)', value: 'LTE' },
  { label: '5G NR', value: '5G' },
  { label: 'GSM (2G)', value: 'GSM' },
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
          title: (
            <span>
              {child.name}
              {child.networkType && (
                <Tag color="blue" style={{ marginLeft: 4, fontSize: 10 }}>
                  {child.networkType}
                </Tag>
              )}
              {child.productType && (
                <Tag color="green" style={{ marginLeft: 4, fontSize: 10 }}>
                  {child.productType}
                </Tag>
              )}
            </span>
          ),
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
  // 权限配置：Record<模块key, { read: boolean, write: boolean }>
  const [permissionLevels, setPermissionLevels] = useState<Record<string, PermissionState>>({});
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

  const { data: allDeviceGroups } = useAllDeviceGroups();

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

  // 获取所有权限项的 key（格式：module.subItem）
  const allPermissionKeys = useMemo(() => {
    const keys: string[] = [];
    for (const module of PERMISSION_MODULES) {
      for (const child of module.children) {
        keys.push(`${module.key}.${child.key}`);
      }
    }
    return keys;
  }, []);

  // 将 permissionLevels 转换为 permissions 数组（用于提交）
  // 格式：["device.list:read", "device.list:write", ...]
  const permissionsToArray = useCallback((levels: Record<string, PermissionState>): string[] => {
    const result: string[] = [];
    for (const [key, state] of Object.entries(levels)) {
      if (state.read) {
        result.push(`${key}:read`);
      }
      if (state.write) {
        result.push(`${key}:write`);
      }
    }
    return result;
  }, []);

  // 将 permissions 数组转换为 permissionLevels
  const arrayToPermissionLevels = useCallback((permissions: string[]): Record<string, PermissionState> => {
    const result: Record<string, PermissionState> = {};

    for (const key of allPermissionKeys) {
      const hasRead = permissions.includes(`${key}:read`);
      const hasWrite = permissions.includes(`${key}:write`);
      result[key] = { read: hasRead, write: hasWrite };
    }
    return result;
  }, [allPermissionKeys]);

  // 检查是否至少选择了一个权限
  const hasAnyPermission = useMemo(
    () => Object.values(permissionLevels).some((level) => level.read || level.write),
    [permissionLevels]
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
          batchOperation: vals.batchOperation ? 1 : 0,
          description: (vals.description as string) ?? '',
          permissions: permissionsToArray(permissionLevels),
          deviceGroupIds: selectedDeviceGroupIds,
          builtIn: 0,
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setCreateVisible(false);
            form.resetFields();
            setPermissionLevels({});
            setSelectedDeviceGroupIds([]);
          },
        },
      );
    });
  }, [form, createRole, permissionLevels, selectedDeviceGroupIds, hasAnyPermission, allSecondLevelIds, permissionsToArray, message, t]);

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
            batchOperation: vals.batchOperation ? 1 : 0,
            description: vals.description as string,
            permissions: permissionsToArray(permissionLevels),
            deviceGroupIds: selectedDeviceGroupIds,
          },
        },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setEditVisible(false);
            form.resetFields();
            setSelectedRole(null);
            setPermissionLevels({});
            setSelectedDeviceGroupIds([]);
          },
        },
      );
    });
  }, [selectedRole, form, updateRole, permissionLevels, selectedDeviceGroupIds, hasAnyPermission, allSecondLevelIds, permissionsToArray, message, t]);

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
                    setPermissionLevels(arrayToPermissionLevels(role.permissions || []));
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
                    setPermissionLevels(arrayToPermissionLevels(role.permissions || []));
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
    {
      key: 'dataPermission',
      title: '数据权限',
      dataIndex: 'deviceGroupIds',
      width: 100,
      render: (val) => {
        const ids = val as string[];
        const count = ids?.length || 0;
        return <Tag color={count > 0 ? 'blue' : 'default'}>{count} 个设备组</Tag>;
      },
    },
    {
      key: 'batchOperation',
      title: '批量操作权限',
      dataIndex: 'batchOperation',
      width: 110,
      render: (val) => (
        <Tag color={val === 1 ? 'green' : 'default'}>
          {val === 1 ? '批量操作' : '单一操作'}
        </Tag>
      ),
    },
    { key: 'description', title: '角色描述', dataIndex: 'description', width: 150, ellipsis: true, render: (v) => v || '-' },
    { key: 'createUser', title: '创建人', dataIndex: 'createUser', width: 100, render: (v) => v || '-' },
    {
      key: 'createTime',
      title: '创建时间',
      dataIndex: 'createTime',
      width: 160,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '-'),
    },
    { key: 'updateUser', title: '更新人', dataIndex: 'updateUser', width: 100, render: (v) => v || '-' },
    {
      key: 'updateTime',
      title: '更新时间',
      dataIndex: 'updateTime',
      width: 160,
      render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '-'),
    },
  ], [t, form, isBuiltIn, handleDelete]);

  // 构建菜单权限树形数据（带只读/读写复选框）
  const buildPermissionTreeData = useCallback((
    readOnly: boolean,
    onReadChange: (fullKey: string, checked: boolean, isParent: boolean, moduleKey?: string) => void,
    onWriteChange: (fullKey: string, checked: boolean, isParent: boolean, moduleKey?: string) => void
  ): TreeDataNode[] => {
    return PERMISSION_MODULES.map((module) => {
      const parentKey = module.key;
      const allChildKeys = module.children.map((child) => `${module.key}.${child.key}`);

      // 计算一级菜单的复选框状态
      const childReadStates = allChildKeys.map((key) => permissionLevels[key]?.read ?? false);
      const childWriteStates = allChildKeys.map((key) => permissionLevels[key]?.write ?? false);

      const allReadChecked = childReadStates.every(Boolean) && allChildKeys.length > 0;
      const someReadChecked = childReadStates.some(Boolean) && !allReadChecked;
      const allWriteChecked = childWriteStates.every(Boolean) && allChildKeys.length > 0;
      const someWriteChecked = childWriteStates.some(Boolean) && !allWriteChecked;

      // 构建二级菜单节点
      const childNodes = module.children.map((child) => {
        const fullKey = `${module.key}.${child.key}`;
        const readChecked = permissionLevels[fullKey]?.read ?? false;
        const writeChecked = permissionLevels[fullKey]?.write ?? false;

        return {
          key: fullKey,
          title: (
            <div style={{ display: 'flex', alignItems: 'center', gap: 16, width: '100%' }}>
              <span style={{ minWidth: 100 }}>{t(child.titleKey)}</span>
              <Checkbox
                checked={readChecked}
                onChange={(e) => onReadChange(fullKey, e.target.checked, false)}
                disabled={readOnly}
                style={{ marginLeft: 'auto' }}
              >
                <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>{t('role.readOnly')}</span>
              </Checkbox>
              <Checkbox
                checked={writeChecked}
                onChange={(e) => onWriteChange(fullKey, e.target.checked, false)}
                disabled={readOnly}
              >
                <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>{t('role.readWrite')}</span>
              </Checkbox>
            </div>
          ),
          isLeaf: true,
        };
      });

      return {
        key: parentKey,
        title: (
          <div style={{ display: 'flex', alignItems: 'center', gap: 16, width: '100%' }}>
            <span style={{ fontWeight: 500, minWidth: 100 }}>{t(module.titleKey)}</span>
            <Checkbox
              checked={allReadChecked}
              indeterminate={someReadChecked}
              onChange={(e) => onReadChange(parentKey, e.target.checked, true, module.key)}
              disabled={readOnly}
              style={{ marginLeft: 'auto' }}
            >
              <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>{t('role.readOnly')}</span>
            </Checkbox>
            <Checkbox
              checked={allWriteChecked}
              indeterminate={someWriteChecked}
              onChange={(e) => onWriteChange(parentKey, e.target.checked, true, module.key)}
              disabled={readOnly}
            >
              <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>{t('role.readWrite')}</span>
            </Checkbox>
          </div>
        ),
        children: childNodes,
      };
    });
  }, [permissionLevels, t]);

  // 处理只读复选框变化
  const handleReadChange = useCallback((fullKey: string, checked: boolean, isParent: boolean, moduleKey?: string) => {
    setPermissionLevels((prev) => {
      const newLevels = { ...prev };

      if (isParent && moduleKey) {
        // 一级菜单：级联到所有子菜单
        const module = PERMISSION_MODULES.find((m) => m.key === moduleKey);
        if (module) {
          for (const child of module.children) {
            const childKey = `${moduleKey}.${child.key}`;
            newLevels[childKey] = {
              ...(newLevels[childKey] ?? { read: false, write: false }),
              read: checked,
            };
          }
        }
      } else {
        // 二级菜单：只更新当前项
        newLevels[fullKey] = {
          ...(newLevels[fullKey] ?? { read: false, write: false }),
          read: checked,
        };
      }

      return newLevels;
    });
  }, []);

  // 处理读写复选框变化
  const handleWriteChange = useCallback((fullKey: string, checked: boolean, isParent: boolean, moduleKey?: string) => {
    setPermissionLevels((prev) => {
      const newLevels = { ...prev };

      if (isParent && moduleKey) {
        // 一级菜单：级联到所有子菜单
        const module = PERMISSION_MODULES.find((m) => m.key === moduleKey);
        if (module) {
          for (const child of module.children) {
            const childKey = `${moduleKey}.${child.key}`;
            newLevels[childKey] = {
              ...(newLevels[childKey] ?? { read: false, write: false }),
              write: checked,
            };
          }
        }
      } else {
        // 二级菜单：只更新当前项
        newLevels[fullKey] = {
          ...(newLevels[fullKey] ?? { read: false, write: false }),
          write: checked,
        };
      }

      return newLevels;
    });
  }, []);

  // 渲染菜单权限配置（树形结构）
  const renderPermissionConfig = (readOnly = false) => {
    // 全选所有权限（只读）
    const handleSelectAllRead = () => {
      setPermissionLevels((prev) => {
        const newLevels = { ...prev };
        for (const module of PERMISSION_MODULES) {
          for (const child of module.children) {
            const fullKey = `${module.key}.${child.key}`;
            newLevels[fullKey] = {
              ...(newLevels[fullKey] ?? { read: false, write: false }),
              read: true,
            };
          }
        }
        return newLevels;
      });
    };

    // 全选所有权限（读写）
    const handleSelectAllWrite = () => {
      setPermissionLevels((prev) => {
        const newLevels = { ...prev };
        for (const module of PERMISSION_MODULES) {
          for (const child of module.children) {
            const fullKey = `${module.key}.${child.key}`;
            newLevels[fullKey] = {
              ...(newLevels[fullKey] ?? { read: false, write: false }),
              write: true,
            };
          }
        }
        return newLevels;
      });
    };

    // 取消全选所有权限
    const handleDeselectAll = () => {
      setPermissionLevels((prev) => {
        const newLevels = { ...prev };
        for (const module of PERMISSION_MODULES) {
          for (const child of module.children) {
            const fullKey = `${module.key}.${child.key}`;
            newLevels[fullKey] = { read: false, write: false };
          }
        }
        return newLevels;
      });
    };

    // 构建树形数据
    const treeData = buildPermissionTreeData(
      readOnly,
      handleReadChange,
      handleWriteChange
    );

    return (
      <Card
        title={t('role.menuPermission')}
        size="small"
        style={{ marginTop: 16 }}
        extra={
          !readOnly ? (
            <Space>
              <Button size="small" onClick={handleSelectAllRead}>
                {t('role.selectAllRead')}
              </Button>
              <Button size="small" onClick={handleSelectAllWrite}>
                {t('role.selectAllWrite')}
              </Button>
              <Button size="small" onClick={handleDeselectAll}>
                {t('role.deselectAll')}
              </Button>
            </Space>
          ) : null
        }
      >
        <div style={{ border: '1px solid var(--color-border)', borderRadius: 6, maxHeight: 400, overflow: 'auto' }}>
          <Tree
            treeData={treeData}
            defaultExpandAll
            selectable={false}
            showIcon={false}
            blockNode
          />
        </div>
      </Card>
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
              <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>筛选：</span>
              <Select
                size="small"
                style={{ width: 120 }}
                value={deviceGroupNetworkType}
                onChange={(val) => setDeviceGroupNetworkType(val)}
                options={NETWORK_TYPE_OPTIONS}
                placeholder="基站制式"
              />
              <Select
                size="small"
                style={{ width: 120 }}
                value={deviceGroupProductType}
                onChange={(val) => setDeviceGroupProductType(val)}
                options={PRODUCT_TYPE_OPTIONS}
                placeholder="产品类型"
              />
              <Divider type="vertical" style={{ height: 20, margin: 0 }} />
              <Checkbox
                checked={isAllSelected}
                indeterminate={isIndeterminate}
                onChange={(e) => handleSelectAll(e.target.checked)}
              >
                全选
              </Checkbox>
            </div>
            {/* 树形选择区域 */}
            <div style={{ padding: 8, maxHeight: 280, overflow: 'auto' }}>
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
    setPermissionLevels({});
    setSelectedDeviceGroupIds([]);
  }, [form]);

  const handleCloseEdit = useCallback(() => {
    setEditVisible(false);
    form.resetFields();
    setSelectedRole(null);
    setPermissionLevels({});
    setSelectedDeviceGroupIds([]);
  }, [form]);

  const handleCloseView = useCallback(() => {
    setViewVisible(false);
    form.resetFields();
    setSelectedRole(null);
    setPermissionLevels({});
    setSelectedDeviceGroupIds([]);
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
          {renderDeviceGroupTree(false)}
          {renderPermissionConfig(false)}
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
          {renderDeviceGroupTree(false)}
          {renderPermissionConfig(false)}
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
          {renderDeviceGroupTree(true)}
          <Form.Item label={t('role.userCount')}>
            <span>{selectedRole?.userCount ?? 0}</span>
          </Form.Item>
          <Form.Item label="创建人">
            <span>{selectedRole?.createUser ?? '-'}</span>
          </Form.Item>
          <Form.Item label="创建时间">
            <span>{selectedRole?.createTime ? new Date(selectedRole.createTime).toLocaleString('zh-CN') : '-'}</span>
          </Form.Item>
          <Form.Item label="更新人">
            <span>{selectedRole?.updateUser ?? '-'}</span>
          </Form.Item>
          <Form.Item label="更新时间">
            <span>{selectedRole?.updateTime ? new Date(selectedRole.updateTime).toLocaleString('zh-CN') : '-'}</span>
          </Form.Item>
          {renderPermissionConfig(true)}
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
