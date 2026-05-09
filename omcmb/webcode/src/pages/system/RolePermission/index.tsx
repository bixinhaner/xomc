import { useState, useMemo, useCallback, useEffect } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Card,
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
import type { MenuProps, TreeDataNode, TreeProps } from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  EyeOutlined,
  MoreOutlined,
  CopyOutlined,
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
} from '@core/hooks/api/useSystem';
import type { Role, ApiEndpoint } from '@core/types/system';
import type { DeviceGroup } from '@core/types/device';
import { useT } from '@/hooks/useT';
import { apiPermissionApi } from '@core/services/api/apiPermissionApi';
import { adminApi } from '@core/services/api/adminApi';
import { useMenuTree } from '@core/hooks/api/useMenus';
import { fetchRoleMenuIds, setRoleMenus as apiSetRoleMenus } from '@core/services/api/menuApi';
import type { Menu } from '@core/types/menu';

// PERMISSION_MODULES 已删除（B3-Phase2-B + 菜单动态加载 P3）。
// 角色菜单权限的唯一权威源是后端 menus 表（GET /admin/menus/tree）；
// 树形数据由 buildMenuPermissionTree(menuTree) 构建，节点 key 即 menu.id (UUID)。
//
// 历史 PERMISSION_MODULES 是把"模块/二级/三级操作"硬编码成字符串 (`device.list.add`)
// 的写法，与后端 permissions 表三元组 1:1 对应；该表已 DROP，常量同步删除。

// 网络类型权限选项
const DATA_NETWORK_TYPE_OPTIONS = [
  { label: 'eNB (LTE)', value: 'lte' },
  { label: 'gNB (5G NR)', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
  { label: 'CPE', value: 'cpe' },
  { label: 'eGW', value: 'egw' },
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

// 菜单树驱动的辅助函数（替代 PERMISSION_MODULES 派生常量）。
//
// 节点 key = menu.id (UUID)。这些函数运行时依赖 menuTree（异步加载），故不再是
// 模块级常量；调用方在 useMemo([menuTree]) 内引用即可。
function isMenuVisible(m: Menu): boolean {
  return m.status === 'normal' && m.showStatus !== 'hide';
}

/** 把后端菜单树转 antd TreeDataNode 树。叶子节点显式 isLeaf=true。 */
function buildMenuPermissionTree(menus: Menu[]): TreeDataNode[] {
  const sortAndFilter = (list: Menu[]) =>
    [...list].filter(isMenuVisible).sort((a, b) => a.sortOrder - b.sortOrder);
  const walk = (list: Menu[]): TreeDataNode[] =>
    sortAndFilter(list).map((m) => {
      const childNodes = m.children?.length ? walk(m.children) : [];
      return childNodes.length > 0
        ? { key: m.id, title: m.name, children: childNodes }
        : { key: m.id, title: m.name, isLeaf: true };
    });
  return walk(menus);
}

/** 全部节点 ID（含目录/菜单/按钮），用于"全选/全不选"。 */
function collectAllMenuIds(menus: Menu[]): string[] {
  const ids: string[] = [];
  const walk = (list: Menu[]) => {
    for (const m of list.filter(isMenuVisible)) {
      ids.push(m.id);
      if (m.children?.length) walk(m.children);
    }
  };
  walk(menus);
  return ids;
}

/** 一级目录 + 二级菜单 ID（用于"展开/折叠"复选框 + 回显默认展开集）。 */
function collectExpandableMenuIds(menus: Menu[]): string[] {
  const ids: string[] = [];
  for (const top of menus.filter(isMenuVisible)) {
    ids.push(top.id);
    if (top.children?.length) {
      for (const child of top.children.filter(isMenuVisible)) {
        if (child.children?.length) ids.push(child.id);
      }
    }
  }
  return ids;
}

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
  // 默认开启父子联动（checkStrictly=false）：编辑面板回显时由 antd Tree 自动从已勾选叶子
  // 推导出二级父节点的 halfChecked 状态。否则父节点收起时视觉上像"没勾选"，与 PRD §6
  // "已分配权限可见"诉求不符。详见 docs/prd/system/roles.md §6 / commit message。
  const [permissionCheckStrictly, setPermissionCheckStrictly] = useState(false);
  // 设备组选择
  const [selectedDeviceGroupIds, setSelectedDeviceGroupIds] = useState<string[]>([]);
  // 网络类型权限选择
  const [selectedNetworkTypes, setSelectedNetworkTypes] = useState<string[]>([]);
  // 设备组筛选条件
  const [deviceGroupNetworkType, setDeviceGroupNetworkType] = useState<string>('');
  const [deviceGroupProductType, setDeviceGroupProductType] = useState<string>('');
  // API权限选择
  const [selectedApiEndpointIds, setSelectedApiEndpointIds] = useState<string[]>([]);
  // API 树展开状态 + 父子联动（与菜单权限 UI 保持一致）
  const [expandedApiGroupKeys, setExpandedApiGroupKeys] = useState<React.Key[]>([]);
  const [apiCheckStrictly, setApiCheckStrictly] = useState(false);


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

  // API权限数据（全部端点列表）
  const { data: apiEndpoints, isLoading: isLoadingApiEndpoints } = useQuery({
    queryKey: ['apiEndpoints', 'all'],
    queryFn: apiPermissionApi.listEndpoints,
    staleTime: 5 * 60 * 1000,
  });

  // 按 apiGroup 分组
  const apiGroupMap = useMemo(() => {
    const map = new Map<string, ApiEndpoint[]>();
    for (const ep of apiEndpoints ?? []) {
      const group = ep.apiGroup || 'other';
      const list = map.get(group) || [];
      list.push(ep);
      map.set(group, list);
    }
    return map;
  }, [apiEndpoints]);

  const apiGroupNames = useMemo(() => Array.from(apiGroupMap.keys()).sort(), [apiGroupMap]);

  // 父级（分组）节点 key 用 'group:<name>' 前缀避免与端点 UUID 冲突。
  const apiGroupParentKeys = useMemo<string[]>(
    () => apiGroupNames.map((g) => `group:${g}`),
    [apiGroupNames],
  );
  // 所有 API 端点 ID（叶子，用于"全选"）。
  const allApiEndpointIds = useMemo<string[]>(
    () => (apiEndpoints ?? []).map((ep) => ep.id),
    [apiEndpoints],
  );

  // antd Tree 渲染颜色 helper（GET=blue / POST=green / ...）。
  const apiMethodColor = (method: string): string => {
    switch (method) {
      case 'GET': return 'blue';
      case 'POST': return 'green';
      case 'PUT': return 'orange';
      case 'DELETE': return 'red';
      case 'PATCH': return 'purple';
      default: return 'default';
    }
  };

  // 构建 API 权限树：分组（父）→ 端点（叶）。
  const apiTreeData = useMemo<TreeDataNode[]>(() => {
    return apiGroupNames.map((groupName) => {
      const group = apiGroupMap.get(groupName) || [];
      return {
        key: `group:${groupName}`,
        title: (
          <span>
            <span style={{ fontWeight: 500 }}>{groupName}</span>
            <span style={{ marginLeft: 8, color: 'var(--color-text-secondary)', fontSize: 12 }}>
              ({group.length})
            </span>
          </span>
        ),
        children: group.map((ep) => ({
          key: ep.id,
          title: (
            <span style={{ display: 'inline-flex', alignItems: 'center' }}>
              <Tag color={apiMethodColor(ep.method)} style={{ marginRight: 8, minWidth: 56, textAlign: 'center' }}>
                {ep.method}
              </Tag>
              <span style={{ fontFamily: 'monospace', fontSize: 13 }}>{ep.path}</span>
              {ep.name ? (
                <span style={{ marginLeft: 8, color: 'var(--color-text-secondary)', fontSize: 12 }}>
                  {ep.name}
                </span>
              ) : null}
            </span>
          ),
          isLeaf: true,
        })),
      };
    });
  }, [apiGroupNames, apiGroupMap]);

  // 数据加载到位后默认展开全部分组（与菜单权限"打开编辑面板自动展开二级"对齐）。
  useEffect(() => {
    if (apiGroupParentKeys.length > 0 && expandedApiGroupKeys.length === 0) {
      setExpandedApiGroupKeys(apiGroupParentKeys);
    }
    // 仅在分组首次出现时触发；后续手动展开/折叠由用户控制。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [apiGroupParentKeys.length]);

  const getRoleApiPermissions = useMutation({
    mutationFn: (roleId: string) => apiPermissionApi.getRolePermissions(roleId),
  });

  const setRoleApiPermissions = useMutation({
    mutationFn: ({ roleId, endpointIds }: { roleId: string; endpointIds: string[] }) =>
      apiPermissionApi.setRolePermissions(roleId, endpointIds),
  });

  // v0.5：与 API 权限对齐的专项端点回显（参 docs/prd/system/roles.md）。
  // 列表 / GetByID 接口虽已带 device_group_ids，但 network_types 仍由专项端点返回；
  // 同时 GetByID(id) 返回完整 permissions（菜单权限），避免列表接口的回显空洞。
  const getRoleDetail = useMutation({
    mutationFn: (roleId: string) => adminApi.getRoleById(roleId),
  });
  const getRoleDeviceGroupsMut = useMutation({
    mutationFn: (roleId: string) => adminApi.getRoleDeviceGroups(roleId),
  });
  const setRoleDeviceGroupsMut = useMutation({
    mutationFn: ({ roleId, deviceGroupIds, networkTypes }: { roleId: string; deviceGroupIds: string[]; networkTypes: string[] }) =>
      adminApi.setRoleDeviceGroups(roleId, { deviceGroupIds, networkTypes }),
  });

  // P3：菜单权限改由 role_menus 表驱动 — 拉/写当前角色的 menu_id 数组。
  const getRoleMenuIdsMut = useMutation({
    mutationFn: (roleId: string) => fetchRoleMenuIds(roleId),
  });
  const setRoleMenusMut = useMutation({
    mutationFn: ({ roleId, menuIds }: { roleId: string; menuIds: string[] }) =>
      apiSetRoleMenus(roleId, { menu_ids: menuIds }),
  });

  // 全量菜单树（GET /admin/menus/tree）。useMenuTree 内部 staleTime=60s。
  const { data: menuTree = [] } = useMenuTree();

  const permissionTreeData = useMemo(() => buildMenuPermissionTree(menuTree), [menuTree]);
  const allMenuIds = useMemo(() => collectAllMenuIds(menuTree), [menuTree]);
  const expandableMenuIds = useMemo(() => collectExpandableMenuIds(menuTree), [menuTree]);

  const isBuiltIn = useCallback((role: Role) => role.builtIn === 1 || role.builtIn === 2, []);

  // v0.5：编辑/查看面板打开时的统一回显逻辑（修复菜单权限 + 数据权限回显空白 bug）。
  //
  // 回显数据来源：
  //   - 菜单权限（permissions） → adminApi.getRoleById(id) 的 permissions 字段
  //     列表接口 ListWithPagination 不填充 permissions（性能考虑），用单角色详情兜底
  //   - 数据权限（deviceGroupIds + networkTypes） → adminApi.getRoleDeviceGroups(id)
  //     network_types 只有专项端点返回；列表接口的 device_group_ids 不带制式
  //   - API 权限（apiPermissions） → 沿用 getRoleApiPermissions.mutate（已有，不动）
  //
  // 列表数据中的 role.deviceGroupIds（v0.5 起后端已填充）作为「⚠️ 未绑分组」标识用，
  // 不参与编辑面板回显，避免与专项端点结果冲突。
  const loadRoleDetailToForm = useCallback((role: Role) => {
    setSelectedRole(role);
    form.setFieldsValue({
      roleName: role.roleName,
      description: role.description,
    });
    // 默认展开一级 + 二级目录（菜单树异步加载，expandableMenuIds 此时已就位 / 否则下一帧 useMemo 重算）。
    setExpandedPermissionKeys(expandableMenuIds);
    // 清空快速回显，避免上一个角色的菜单 ID 残留
    setCheckedPermissionKeys([]);
    setSelectedDeviceGroupIds(role.deviceGroupIds || []);
    setSelectedNetworkTypes(role.networkTypes || []);

    // 1) 拉角色已绑定 menu_ids（GET /admin/roles/:id/menus）
    getRoleMenuIdsMut.mutate(role.id, {
      onSuccess: (menuIds) => setCheckedPermissionKeys(menuIds),
    });
    // 2) 拉设备分组 + 网络制式专项端点
    getRoleDeviceGroupsMut.mutate(role.id, {
      onSuccess: ({ deviceGroupIds, networkTypes }) => {
        setSelectedDeviceGroupIds(deviceGroupIds);
        setSelectedNetworkTypes(networkTypes);
      },
    });
    // 3) 拉 API 权限专项端点（保持原有逻辑）
    getRoleApiPermissions.mutate(role.id, {
      onSuccess: (ids) => setSelectedApiEndpointIds(ids),
    });
  }, [form, expandableMenuIds, getRoleMenuIdsMut, getRoleDeviceGroupsMut, getRoleApiPermissions]);

  // 已有的角色名称列表（用于重复检查）
  const existingRoleNames = useMemo(
    () => (data?.items ?? []).map((r) => r.roleName.toLowerCase()),
    [data?.items]
  );

  // permissionTreeData / allMenuIds / expandableMenuIds 已在上方根据 menuTree 计算。

  /**
   * 把当前 checkedPermissionKeys（菜单 ID 数组）转成提交给后端的 menu_ids 数组。
   * 仅过滤出"实际存在于当前菜单树"的 ID（防止旧 cache 的脏 ID 漏到 PUT）。
   */
  const permissionsToMenuIds = useCallback((keys: React.Key[]): string[] => {
    const valid = new Set(allMenuIds);
    return (keys as string[]).filter((k) => valid.has(k));
  }, [allMenuIds]);

  // 至少选了一个菜单（无论目录/菜单/按钮）即视为有权限。
  const hasAnyPermission = useMemo(
    () => permissionsToMenuIds(checkedPermissionKeys).length > 0,
    [checkedPermissionKeys, permissionsToMenuIds]
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

  // v0.6（roles.md §7 P2 #9）：一键复制角色。
  // 后端会自动生成 base_copy / base_copy_2 ... 副本名，副本 is_system=false，
  // 并复制 permissions / role_menus / role_device_groups / role_api_permissions 全部绑定。
  const copyRoleMut = useMutation({
    mutationFn: (id: string) => adminApi.copyRole(id),
  });
  const handleCopy = useCallback((role: Role) => {
    modal.confirm({
      title: '确认复制角色',
      content: `将复制角色「${role.roleName}」的菜单/数据/API 权限到新副本（副本名自动生成 ${role.roleName}_copy）。`,
      onOk: () => {
        copyRoleMut.mutate(role.id, {
          onSuccess: (newRole) => {
            message.success(`已复制：${newRole.roleName}`);
            // 强制刷新列表
            void refetch();
          },
          onError: (err: unknown) => {
            const msg = err instanceof Error ? err.message : '复制失败';
            message.error(msg);
          },
        });
      },
    });
  }, [copyRoleMut, modal, message, refetch]);

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
      const menuIds = permissionsToMenuIds(checkedPermissionKeys);
      createRole.mutate(
        {
          roleName: vals.roleName as string,
          description: (vals.description as string) ?? '',
          // P3：permissions 字段已无意义（permissions 表 DROP），传空数组兼容 Role 类型签名。
          // 真正的菜单权限由下方 setRoleMenusMut 写入 role_menus。
          permissions: [],
          deviceGroupIds: selectedDeviceGroupIds,
          networkTypes: selectedNetworkTypes,
          builtIn: 0,
        },
        {
          onSuccess: (newRole) => {
            if (newRole?.id) {
              // 写菜单绑定（role_menus）
              setRoleMenusMut.mutate({ roleId: newRole.id, menuIds });
              // 通过专项端点保存设备分组 + 网络制式
              setRoleDeviceGroupsMut.mutate({
                roleId: newRole.id,
                deviceGroupIds: selectedDeviceGroupIds,
                networkTypes: selectedNetworkTypes,
              });
              // 保存 API 权限
              if (selectedApiEndpointIds.length > 0) {
                setRoleApiPermissions.mutate({ roleId: newRole.id, endpointIds: selectedApiEndpointIds });
              }
            }
            message.success(t('common.save'));
            setCreateVisible(false);
            form.resetFields();
            setCheckedPermissionKeys([]);
            setExpandedPermissionKeys([]);
            setSelectedDeviceGroupIds([]);
            setSelectedNetworkTypes([]);
            setSelectedApiEndpointIds([]);
          },
        },
      );
    });
  }, [form, createRole, checkedPermissionKeys, selectedDeviceGroupIds, selectedNetworkTypes, selectedApiEndpointIds, hasAnyPermission, allSecondLevelIds, permissionsToMenuIds, setRoleMenusMut, setRoleDeviceGroupsMut, setRoleApiPermissions, message, t]);

  // doEditSubmit 拆出实际提交逻辑，配合下方"清空设备分组二次确认"复用。
  // 必须先于 handleEdit 声明，否则 React 的 useCallback 会触发 react-hooks/refs：
  // "Cannot access doEditSubmit before it is declared"。
  const doEditSubmit = useCallback(() => {
    if (!selectedRole) return;
    form.validateFields().then((vals) => {
      const menuIds = permissionsToMenuIds(checkedPermissionKeys);
      updateRole.mutate(
        {
          id: selectedRole.id,
          data: {
            roleName: vals.roleName as string,
            description: vals.description as string,
            // P3：permissions 字段已无意义（permissions 表 DROP），传空数组。
            // 菜单权限改由下方 setRoleMenusMut 写入 role_menus。
            permissions: [],
            deviceGroupIds: selectedDeviceGroupIds,
            networkTypes: selectedNetworkTypes,
          },
        },
        {
          onSuccess: () => {
            // 写菜单绑定（role_menus）
            setRoleMenusMut.mutate({ roleId: selectedRole.id, menuIds });
            // 通过专项端点保存设备分组 + 网络制式
            setRoleDeviceGroupsMut.mutate({
              roleId: selectedRole.id,
              deviceGroupIds: selectedDeviceGroupIds,
              networkTypes: selectedNetworkTypes,
            });
            // 保存 API 权限
            setRoleApiPermissions.mutate({ roleId: selectedRole.id, endpointIds: selectedApiEndpointIds });
            message.success(t('common.save'));
            setEditVisible(false);
            form.resetFields();
            setSelectedRole(null);
            setCheckedPermissionKeys([]);
            setExpandedPermissionKeys([]);
            setSelectedDeviceGroupIds([]);
            setSelectedNetworkTypes([]);
            setSelectedApiEndpointIds([]);
          },
        },
      );
    });
  }, [selectedRole, form, updateRole, checkedPermissionKeys, selectedDeviceGroupIds, selectedNetworkTypes, selectedApiEndpointIds, permissionsToMenuIds, setRoleMenusMut, setRoleDeviceGroupsMut, setRoleApiPermissions, message, t]);

  // 校验并提交编辑
  const handleEdit = useCallback(() => {
    if (!selectedRole) return;

    // 校验权限
    if (!hasAnyPermission) {
      message.warning(t('role.pleaseSelectPermission'));
      return;
    }

    // 中层：清空设备分组二次确认 + 实际保存。
    // §11.2 决议 ① 触点 5：选 0 个二级节点不再硬阻止，改为弹 Modal 警告 → OK 才继续。
    const proceed = () => {
      const selectedSecondLevel = selectedDeviceGroupIds.filter((id) => allSecondLevelIds.includes(id));
      if (selectedSecondLevel.length === 0) {
        modal.confirm({
          title: '确认清空设备分组绑定',
          content: '该角色下的用户将立即失去设备数据可见权限。是否继续？',
          okText: '继续保存',
          okButtonProps: { danger: true },
          cancelText: '取消',
          onOk: () => doEditSubmit(),
        });
        return;
      }
      doEditSubmit();
    };

    // 外层：内置角色额外加一道"权限即时生效"二次确认。
    // 内置角色的权限改动会立即影响所有分配此角色的用户，需显式确认才能继续。
    if (isBuiltIn(selectedRole)) {
      modal.confirm({
        title: '确认修改内置角色权限',
        content: `内置角色「${selectedRole.roleName}」的权限调整将立即影响所有分配此角色的用户。是否继续？`,
        okText: '继续保存',
        okButtonProps: { danger: true },
        cancelText: '取消',
        onOk: proceed,
      });
      return;
    }

    proceed();
  }, [selectedRole, hasAnyPermission, isBuiltIn, selectedDeviceGroupIds, allSecondLevelIds, doEditSubmit, message, modal, t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'roleName', label: t('role.roleName'), type: 'input', placeholder: t('role.roleName') },
  ], [t]);

  const columns: DataTableColumn<Role & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const role = record as Role;
        const moreItems: MenuProps['items'] = [
          {
            key: 'view',
            label: t('common.view'),
            icon: <EyeOutlined />,
            onClick: () => {
              loadRoleDetailToForm(role);
              setViewVisible(true);
            },
          },
          {
            key: 'copy',
            label: '复制',
            icon: <CopyOutlined />,
            onClick: () => handleCopy(role),
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
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small"
              onClick={() => {
                loadRoleDetailToForm(role);
                setEditVisible(true);
              }}>
              {t('common.edit')}
            </Button>
            <Dropdown menu={{ items: moreItems }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
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
        // §11.2 决议 ① 触点 1：未绑设备分组的角色，列表行加 ⚠️ Tag
        // 用 list 接口返回的 deviceGroupIds（v0.5 已填充）判定
        const noGroups = !role.deviceGroupIds || role.deviceGroupIds.length === 0;
        return (
          <span>
            {String(val)}
            {isBuiltIn(role) && <Tag color="blue" style={{ marginLeft: 8 }}>{t('role.builtIn')}</Tag>}
            {!isBuiltIn(role) && noGroups && (
              <Tag color="warning" style={{ marginLeft: 8 }}>⚠️ 未绑分组</Tag>
            )}
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
  ], [t, isBuiltIn, handleDelete, handleCopy, loadRoleDetailToForm]);

  // 渲染菜单权限配置（树形结构 - 按图片样式）
  const renderPermissionConfig = (readOnly = false) => {
    // 是否全部展开（一级目录 + 二级菜单都已展开）
    const isAllExpanded =
      expandableMenuIds.length > 0 &&
      expandedPermissionKeys.length >= expandableMenuIds.length;

    // 展开/折叠所有（复选框）
    const handleExpandChange = (checked: boolean) => {
      if (checked) {
        setExpandedPermissionKeys(expandableMenuIds);
      } else {
        setExpandedPermissionKeys([]);
      }
    };

    // 全选/全不选（复选框）- 控制所有节点（含按钮）
    const handleSelectAllChange = (checked: boolean) => {
      setCheckedPermissionKeys(checked ? allMenuIds : []);
    };

    // 父子联动（复选框）- 勾选表示联动，不勾选表示不联动
    const handleLinkageChange = (checked: boolean) => {
      setPermissionCheckStrictly(!checked); // checkStrictly=false 表示联动
    };

    // 当前选中的菜单 ID 中"实际存在于树"的数量（去掉旧 cache 的脏 ID 影响）
    const validCheckedSet = new Set(allMenuIds);
    const checkedCount = checkedPermissionKeys.filter((k) =>
      validCheckedSet.has(k as string),
    ).length;

    // 是否全选
    const isAllSelected = allMenuIds.length > 0 && checkedCount === allMenuIds.length;
    // 是否部分选中
    const isIndeterminate = checkedCount > 0 && checkedCount < allMenuIds.length;

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

  // 渲染设备组树形选择（包含网络类型权限）
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
      <div>
        {/* 网络类型权限区域 */}
        <Form.Item label={t('role.networkType')}>
          {readOnly ? (
            <Space wrap>
              {selectedNetworkTypes.length > 0 ? (
                DATA_NETWORK_TYPE_OPTIONS
                  .filter((opt) => selectedNetworkTypes.includes(opt.value))
                  .map((opt) => <Tag key={opt.value}>{opt.label}</Tag>)
              ) : (
                <span style={{ color: 'var(--color-text-secondary)' }}>{t('role.networkTypeHint')}</span>
              )}
            </Space>
          ) : (
            <div style={{ border: '1px solid var(--color-border)', borderRadius: 6, padding: '8px 12px' }}>
              <Checkbox.Group
                options={DATA_NETWORK_TYPE_OPTIONS}
                value={selectedNetworkTypes}
                onChange={(vals) => setSelectedNetworkTypes(vals as string[])}
              />
              <div style={{ marginTop: 6, fontSize: 12, color: 'var(--color-text-secondary)' }}>
                {t('role.networkTypeHint')}
              </div>
            </div>
          )}
        </Form.Item>

        {/* 设备分组权限区域 */}
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
      </div>
    );
  };

  // 渲染API权限配置（按分组展示，支持分组批量勾选）
  // 渲染 API 权限：父=API分组、叶=端点（method+path），UI 与菜单权限 Tree 一致。
  const renderApiPermissionConfig = (readOnly = false) => {
    const isAllExpanded =
      apiGroupParentKeys.length > 0 && expandedApiGroupKeys.length >= apiGroupParentKeys.length;
    const isAllSelected =
      allApiEndpointIds.length > 0 && selectedApiEndpointIds.length === allApiEndpointIds.length;
    const isIndeterminate =
      selectedApiEndpointIds.length > 0 && selectedApiEndpointIds.length < allApiEndpointIds.length;

    const handleExpandAll = (checked: boolean) => {
      setExpandedApiGroupKeys(checked ? apiGroupParentKeys : []);
    };
    const handleSelectAll = (checked: boolean) => {
      setSelectedApiEndpointIds(checked ? allApiEndpointIds : []);
    };
    const handleLinkage = (checked: boolean) => {
      // 勾选"父子联动" => checkStrictly=false（一致于菜单权限的语义）。
      setApiCheckStrictly(!checked);
    };

    const handleCheck: TreeProps['onCheck'] = (checked) => {
      // 联动模式 checked: Key[]；strict 模式 checked: { checked, halfChecked }。
      const all = Array.isArray(checked) ? checked : checked.checked;
      // 仅保留叶子（端点 UUID），过滤掉 'group:xxx' 父节点 key。
      const leaves = (all as React.Key[]).filter(
        (k): k is string => typeof k === 'string' && !k.startsWith('group:'),
      );
      setSelectedApiEndpointIds(leaves);
    };

    const handleExpand: TreeProps['onExpand'] = (expanded) => {
      setExpandedApiGroupKeys(expanded as React.Key[]);
    };

    // checkStrictly=true 时 antd Tree 要求对象形态；strict=false 直接传叶子数组即可，
    // antd 会自动从子节点推导父节点 halfChecked。
    const treeCheckedKeys: TreeProps['checkedKeys'] = apiCheckStrictly
      ? { checked: selectedApiEndpointIds, halfChecked: [] }
      : selectedApiEndpointIds;

    return (
      <Form.Item label={t('role.apiPermission')}>
        <div style={{ border: '1px solid var(--color-border)', borderRadius: 6 }}>
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
              onChange={(e) => handleExpandAll(e.target.checked)}
              disabled={readOnly || apiGroupParentKeys.length === 0}
            >
              {t('role.expandCollapse')}
            </Checkbox>
            <Checkbox
              checked={isAllSelected}
              indeterminate={isIndeterminate}
              onChange={(e) => handleSelectAll(e.target.checked)}
              disabled={readOnly || allApiEndpointIds.length === 0}
            >
              {t('role.selectAllOrNone')}
            </Checkbox>
            <Checkbox
              checked={!apiCheckStrictly}
              onChange={(e) => handleLinkage(e.target.checked)}
              disabled={readOnly}
            >
              {t('role.parentChildLinkage')}
            </Checkbox>
            <span style={{ marginLeft: 'auto', color: 'var(--color-text-secondary)', fontSize: 12 }}>
              {t('role.selectedApiCount', { count: selectedApiEndpointIds.length })}
              {' / '}
              {allApiEndpointIds.length}
            </span>
          </div>
          <div style={{ padding: 8, maxHeight: 500, overflow: 'auto' }}>
            {isLoadingApiEndpoints ? (
              <div style={{ textAlign: 'center', padding: 24 }}><Spin /></div>
            ) : (
              <Tree
                treeData={apiTreeData}
                expandedKeys={expandedApiGroupKeys}
                checkedKeys={treeCheckedKeys}
                selectable={false}
                checkable
                disabled={readOnly}
                checkStrictly={apiCheckStrictly}
                onCheck={readOnly ? undefined : handleCheck}
                onExpand={handleExpand}
              />
            )}
          </div>
        </div>
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
    setSelectedNetworkTypes([]);
    setSelectedApiEndpointIds([]);
    setActiveTab('menu');
  }, [form]);

  const handleCloseEdit = useCallback(() => {
    setEditVisible(false);
    form.resetFields();
    setSelectedRole(null);
    setCheckedPermissionKeys([]);
    setExpandedPermissionKeys([]);
    setSelectedDeviceGroupIds([]);
    setSelectedNetworkTypes([]);
    setSelectedApiEndpointIds([]);
    setActiveTab('menu');
  }, [form]);

  const handleCloseView = useCallback(() => {
    setViewVisible(false);
    form.resetFields();
    setSelectedRole(null);
    setCheckedPermissionKeys([]);
    setExpandedPermissionKeys([]);
    setSelectedDeviceGroupIds([]);
    setSelectedNetworkTypes([]);
    setSelectedApiEndpointIds([]);
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
      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
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
      </Card>

      {/* Create Drawer */}
      <Drawer
        title={t('common.add')}
        open={createVisible}
        onClose={handleCloseCreate}
        width={800}
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
              children: renderApiPermissionConfig(false),
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
        {/* 内置角色提示：允许调整权限，但不允许改名/描述/删除 */}
        {selectedRole && isBuiltIn(selectedRole) && (
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message="内置角色"
            description="仅可调整菜单/API/数据权限，不允许修改名称、描述或删除。修改将立即影响所有分配此角色的用户。"
          />
        )}
        {/* §11.2 决议 ① 触点 2：未绑设备分组的角色 banner 提示 */}
        {selectedDeviceGroupIds.length === 0 && (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
            message="该角色未绑定任何设备分组"
            description="分配该角色的用户将无法查看任何设备数据。请在「数据权限」标签页选择至少一个二级设备分组。"
          />
        )}
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
              readOnly={selectedRole ? isBuiltIn(selectedRole) : false}
              style={selectedRole && isBuiltIn(selectedRole) ? { color: 'var(--color-text-secondary)' } : undefined}
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
              children: renderApiPermissionConfig(false),
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
        {/* §11.2 决议 ① 触点 3：未绑设备分组的角色 banner 提示（查看也展示）*/}
        {selectedDeviceGroupIds.length === 0 && (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
            message="该角色未绑定任何设备分组"
            description="分配该角色的用户将无法查看任何设备数据。"
          />
        )}
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
              children: renderApiPermissionConfig(true),
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
