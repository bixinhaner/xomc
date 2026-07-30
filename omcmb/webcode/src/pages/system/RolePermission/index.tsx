import { useState, useMemo, useCallback, useRef } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useIntl } from 'react-intl';
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
import type { ApiEndpoint, Role } from '@core/types/system';
import type { DeviceGroup } from '@core/types/device';
import { useT } from '@/hooks/useT';
import { adminApi } from '@core/services/api/adminApi';
import { apiPermissionApi } from '@core/services/api/apiPermissionApi';
import { useMenuTree, useInvalidateUserMenus } from '@core/hooks/api/useMenus';
import { fetchRoleMenuIds, setRoleMenus as apiSetRoleMenus } from '@core/services/api/menuApi';
import { resolveMenuLabel, type Menu } from '@core/types/menu';
import { getI18nText, type Locale } from '@core/utils/i18nText';
import { formatSystemTime } from '@core/utils/systemTime';
import { UNASSIGNED_GROUP_ID } from '@core/utils/deviceGroupTargets';
import {
  applyNewApiSuggestions,
  inferReadApiEndpointIds,
} from './roleApiPermissionModel';
import { saveRolePermissionBindings } from './rolePermissionSave';

// 菜单权限的唯一权威源是后端 menus 表（GET /admin/menus/tree）；
// 树形数据由 buildMenuPermissionTree(menuTree) 构建，节点 key 即 menu.id (UUID)。

// 网络类型权限选项
const DATA_NETWORK_TYPE_OPTIONS = [
  { label: 'eNB (LTE)', value: 'lte' },
  { label: 'gNB (5G NR)', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
  { label: 'CPE', value: 'cpe' },
  { label: 'eGW', value: 'egw' },
];

function getRoleDeleteErrorMessage(err: unknown, fallback: string, inUseMessage: string): string {
  const error = err as { bizCode?: number; userMessage?: string; message?: string } | undefined;
  if (error?.bizCode === 7008) return inUseMessage;
  return error?.userMessage || error?.message || fallback;
}

function renderRoleOperator(value: unknown, role: Role, builtInText: string): string {
  if (value) return String(value);
  return role.builtIn > 0 ? builtInText : '-';
}

// 基站制式选项
const buildNetworkTypeOptions = (t: (id: string) => string) => [
  { label: t('role.all'), value: '' },
  { label: 'eNB', value: 'eNB' },
  { label: 'gNB', value: 'gNB' },
  { label: 'GSM', value: 'GSM' },
];

// 产品类型选项
const buildProductTypeOptions = (t: (id: string) => string) => [
  { label: t('role.all'), value: '' },
  { label: t('role.macroBaseStation'), value: 'Macro' },
  { label: t('role.smallCell'), value: 'Small Cell' },
  { label: t('role.picoBaseStation'), value: 'Pico' },
];

// 构建设备组树形数据（带筛选）
const buildDeviceGroupTreeData = (
  groups: DeviceGroup[],
  locale: Locale,
  networkTypeFilter?: string,
  productClassFilter?: string
): TreeDataNode[] => {
  // 筛选二级节点
  let filteredChildGroups = groups.filter((g) => g.parentId);

  // 应用筛选条件
  if (networkTypeFilter) {
    filteredChildGroups = filteredChildGroups.filter(
      (g) => g.networkType === networkTypeFilter || !g.networkType
    );
  }
  if (productClassFilter) {
    filteredChildGroups = filteredChildGroups.filter(
      (g) => g.productClass === productClassFilter || !g.productClass
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
          title: getI18nText(child.nameI18n, locale, child.name),
        }));

        // 如果没有子节点且不是「未分组设备」虚拟节点，则不显示。
        if (children.length === 0 && root.builtIn === 1 && root.id !== UNASSIGNED_GROUP_ID) {
        return null;
      }

      return {
        key: root.id,
        title: getI18nText(root.nameI18n, locale, root.name),
        children,
      };
    })
    .filter((node) => node !== null) as TreeDataNode[];
};

// 菜单树驱动的辅助函数（替代 PERMISSION_MODULES 派生常量）。
//
// 节点 key = menu.id (UUID)。这些函数运行时依赖 menuTree（异步加载），故不再是
// 模块级常量；调用方在 useMemo([menuTree]) 内引用即可。
function isMenuVisible(m: Menu): boolean {
  return m.status === 'normal' && m.showStatus !== 'hide';
}

/** 菜单权限树只展示「目录 / 菜单」节点。按钮(type='button')由「角色 API」页签管理，
 * 不再混在菜单权限树里。若树里含 button 节点，旧 `expandMenuAncestors` 会在后端
 * 把 button 的祖先 menu 自动补回 role_menus，导致用户取消父菜单后回显复活。 */
function isMenuNode(m: Menu): boolean {
  return m.type !== 'button';
}

/** 把后端菜单树转 antd TreeDataNode 树。叶子节点显式 isLeaf=true。 */
function buildMenuPermissionTree(menus: Menu[], locale: string): TreeDataNode[] {
  const sortAndFilter = (list: Menu[]) =>
    [...list].filter(isMenuVisible).filter(isMenuNode).sort((a, b) => a.sortOrder - b.sortOrder);
  const walk = (list: Menu[]): TreeDataNode[] =>
    sortAndFilter(list).map((m) => {
      const childNodes = m.children?.length ? walk(m.children) : [];
      return childNodes.length > 0
        ? { key: m.id, title: resolveMenuLabel(m, locale), children: childNodes }
        : { key: m.id, title: resolveMenuLabel(m, locale), isLeaf: true };
    });
  return walk(menus);
}

/** 仅目录 + 菜单节点 ID（不含按钮），用于"全选/全不选"以及 `permissionsToMenuIds`
 * 的白名单——按钮 ID 由 [[buttonIdsUnderMenus]] 在保存时单独合并回去。 */
function collectAllMenuIds(menus: Menu[]): string[] {
  const ids: string[] = [];
  const walk = (list: Menu[]) => {
    for (const m of list.filter(isMenuVisible).filter(isMenuNode)) {
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
  for (const top of menus.filter(isMenuVisible).filter(isMenuNode)) {
    ids.push(top.id);
    if (top.children?.length) {
      for (const child of top.children.filter(isMenuVisible).filter(isMenuNode)) {
        if (child.children?.length) ids.push(child.id);
      }
    }
  }
  return ids;
}

/** 「可视树叶子」判定：节点过滤完 button + isMenuVisible 后没有任何可视子节点。
 * 用于回显时把目录/菜单 ID 拆成「叶子（要进 checkedPermissionKeys）」与「目录
 * （让 antd Tree 在 checkStrictly=false 模式下自动派生）」两份——否则把父目录
 * 喂给 antd 会触发自动级联：父目录 checked → 所有子节点（含用户刚取消的）也
 * 被展示为 checked，从而出现"取消子菜单保存重开还在勾选"的回归。 */
function isVisibleLeaf(m: Menu): boolean {
  const visibleChildren = (m.children ?? []).filter(isMenuVisible).filter(isMenuNode);
  return visibleChildren.length === 0;
}

/** 取「祖先菜单仍被勾选」的 button ID 集合：保存时把这些 button 写回 role_menus，
 * 避免角色编辑面板把所有按钮级 RBAC 一次性清空。祖先（任意层级）的判定按 menus
 * 树的 parentId 链向上回溯；任一祖先不在 checkedMenuIds 即丢弃。 */
function buttonIdsUnderMenus(menus: Menu[], checkedMenuIds: Set<string>): string[] {
  const flat: Menu[] = [];
  const walk = (list: Menu[]) => {
    for (const m of list) {
      flat.push(m);
      if (m.children?.length) walk(m.children);
    }
  };
  walk(menus);
  const byId = new Map<string, Menu>();
  for (const m of flat) byId.set(m.id, m);

  const out: string[] = [];
  for (const m of flat) {
    if (!isMenuVisible(m) || m.type !== 'button') continue;
    // 按钮自身不在 menu tree 的勾选集合里；只要它「最近的菜单祖先」仍勾选即保留。
    let cur: Menu | undefined = m.parentId ? byId.get(m.parentId) : undefined;
    let keep = false;
    while (cur) {
      if (cur.type !== 'button') {
        keep = checkedMenuIds.has(cur.id);
        break;
      }
      cur = cur.parentId ? byId.get(cur.parentId) : undefined;
    }
    if (keep) out.push(m.id);
  }
  return out;
}

export default function RoleManagement() {
  const t = useT();
  const { locale: intlLocale } = useIntl();
  const locale: Locale = intlLocale === 'en-US' ? 'en-US' : 'zh-CN';
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
  const [deviceGroupProductClass, setDeviceGroupProductClass] = useState<string>('');
  // API 端点权限：编辑/查看严格回显 role_api_permissions；创建时只安全建议 GET。
  const [selectedApiEndpointIds, setSelectedApiEndpointIds] = useState<string[]>([]);
  const [suggestedApiEndpointIds, setSuggestedApiEndpointIds] = useState<string[]>([]);
  const [autoSelectedApiEndpointIds, setAutoSelectedApiEndpointIds] = useState<string[]>([]);
  const [expandedApiGroupKeys, setExpandedApiGroupKeys] = useState<React.Key[]>([]);
  const [apiCheckStrictly, setApiCheckStrictly] = useState(false);
  const [isRoleBindingsLoading, setIsRoleBindingsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const roleBindingLoadGeneration = useRef(0);
  const savingRef = useRef(false);
  // role_menus 中属于 button 类型的原始 ID（从后端读回的角色绑定里拆出来）。
  // 菜单权限树只展示 directory/menu，按钮不在树里；保存时把"祖先菜单仍勾选"的
  // 按钮 ID 回填进 PUT 请求，避免按钮级 RBAC 被本面板顺手清空。
  const [originalButtonIds, setOriginalButtonIds] = useState<string[]>([]);

  // 用于编辑/新增保存后立即让侧边栏（useUserMenus）失效重拉，
  // 修复"改完菜单权限后侧边栏要等 5 分钟或重新登录才更新"的体验问题。
  // （source='builtIn' 的内置超管走后端旁路返回全量菜单，不受此影响。）
  const invalidateUserMenus = useInvalidateUserMenus();


  const { data, isLoading, refetch } = useRoles({
    roleName: filters.roleName as string | undefined,
    page,
    pageSize,
  });

  const { data: allDeviceGroups, isLoading: isLoadingDeviceGroups } = useAllDeviceGroups();

  // 设备组树形数据（一级+二级节点，支持筛选）
  const deviceGroupTreeData = useMemo(
    () => buildDeviceGroupTreeData(allDeviceGroups ?? [], locale, deviceGroupNetworkType, deviceGroupProductClass),
    [allDeviceGroups, locale, deviceGroupNetworkType, deviceGroupProductClass]
  );

  // 筛选后的二级节点ID列表
  const filteredSecondLevelIds = useMemo(() => {
    let filtered = (allDeviceGroups ?? []).filter((g) => g.parentId);
    if (deviceGroupNetworkType) {
      filtered = filtered.filter((g) => g.networkType === deviceGroupNetworkType || !g.networkType);
    }
    if (deviceGroupProductClass) {
      filtered = filtered.filter((g) => g.productClass === deviceGroupProductClass || !g.productClass);
    }
    return filtered.map((g) => g.id);
  }, [allDeviceGroups, deviceGroupNetworkType, deviceGroupProductClass]);

  // 筛选后的一级节点ID列表（用于全选时同时选中父节点）
  const filteredFirstLevelIds = useMemo(() => {
    const secondLevelNodes = (allDeviceGroups ?? []).filter((g) =>
      filteredSecondLevelIds.includes(g.id)
    );
    const parentIds = new Set(secondLevelNodes.map((g) => g.parentId).filter(Boolean));
    return Array.from(parentIds) as string[];
  }, [allDeviceGroups, filteredSecondLevelIds]);

  // 所有可写入数据权限的 group ID：真实二级组 + 「未分组设备」伪节点。
  const allDataPermissionGroupIds = useMemo(
    () => (allDeviceGroups ?? []).filter((g) => g.parentId || g.id === UNASSIGNED_GROUP_ID).map((g) => g.id),
    [allDeviceGroups]
  );

  // 当前筛选条件下可被一键全选的 group ID：真实二级组 + 「未分组设备」伪节点。
  const filteredSelectableGroupIds = useMemo(
    () => [
      ...filteredSecondLevelIds,
      ...(allDeviceGroups?.some((g) => g.id === UNASSIGNED_GROUP_ID) ? [UNASSIGNED_GROUP_ID] : []),
    ],
    [allDeviceGroups, filteredSecondLevelIds]
  );

  const createRole = useCreateRole();
  const updateRole = useUpdateRole();
  const deleteRoles = useDeleteRoles();

  const { data: apiEndpoints = [], isLoading: isLoadingApiEndpoints } = useQuery({
    queryKey: ['apiEndpoints', 'all'],
    queryFn: apiPermissionApi.listEndpoints,
    staleTime: 5 * 60 * 1000,
  });

  const apiGroupMap = useMemo(() => {
    const grouped = new Map<string, ApiEndpoint[]>();
    for (const endpoint of apiEndpoints) {
      const group = endpoint.apiGroup || 'other';
      const items = grouped.get(group) ?? [];
      items.push(endpoint);
      grouped.set(group, items);
    }
    return grouped;
  }, [apiEndpoints]);
  const apiGroupNames = useMemo(
    () => Array.from(apiGroupMap.keys()).sort(),
    [apiGroupMap],
  );
  const apiGroupParentKeys = useMemo(
    () => apiGroupNames.map((group) => `group:${group}`),
    [apiGroupNames],
  );
  const allApiEndpointIds = useMemo(
    () => apiEndpoints.map((endpoint) => endpoint.id),
    [apiEndpoints],
  );

  const apiTreeData = useMemo<TreeDataNode[]>(() => {
    const methodColor = (method: string): string => {
      switch (method.toUpperCase()) {
        case 'GET': return 'blue';
        case 'POST': return 'green';
        case 'PUT': return 'orange';
        case 'DELETE': return 'red';
        case 'PATCH': return 'purple';
        default: return 'default';
      }
    };
    return apiGroupNames.map((groupName) => {
      const group = apiGroupMap.get(groupName) ?? [];
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
        children: group.map((endpoint) => ({
          key: endpoint.id,
          title: (
            <span style={{ display: 'inline-flex', alignItems: 'center' }}>
              <Tag
                color={methodColor(endpoint.method)}
                style={{ marginRight: 8, minWidth: 56, textAlign: 'center' }}
              >
                {endpoint.method.toUpperCase()}
              </Tag>
              <span style={{ fontFamily: 'monospace', fontSize: 13 }}>{endpoint.path}</span>
              {endpoint.name ? (
                <span style={{ marginLeft: 8, color: 'var(--color-text-secondary)', fontSize: 12 }}>
                  {endpoint.name}
                </span>
              ) : null}
            </span>
          ),
          isLeaf: true,
        })),
      };
    });
  }, [apiGroupMap, apiGroupNames]);

  const setRoleApiPermissionsMut = useMutation({
    mutationFn: ({ roleId, endpointIds }: { roleId: string; endpointIds: string[] }) =>
      apiPermissionApi.setRolePermissions(roleId, endpointIds),
  });

  // 菜单 / API / 设备组专项端点回显（参 docs/prd/system/roles.md）。
  // 列表 / GetByID 接口虽已带 device_group_ids，但 network_types 仍由专项端点返回；
  // 同时 GetByID(id) 返回完整 permissions（菜单权限），避免列表接口的回显空洞。
  const setRoleDeviceGroupsMut = useMutation({
    mutationFn: ({ roleId, deviceGroupIds, networkTypes }: { roleId: string; deviceGroupIds: string[]; networkTypes: string[] }) =>
      adminApi.setRoleDeviceGroups(roleId, { deviceGroupIds, networkTypes }),
  });

  // P3：菜单权限改由 role_menus 表驱动 — 拉/写当前角色的 menu_id 数组。
  const setRoleMenusMut = useMutation({
    mutationFn: ({ roleId, menuIds }: { roleId: string; menuIds: string[] }) =>
      apiSetRoleMenus(roleId, { menu_ids: menuIds }),
  });

  // 全量菜单树（GET /admin/menus/tree）。useMenuTree 内部 staleTime=60s。
  const { data: menuTree = [] } = useMenuTree();

  const permissionTreeData = useMemo(() => buildMenuPermissionTree(menuTree, locale), [menuTree, locale]);
  const allMenuIds = useMemo(() => collectAllMenuIds(menuTree), [menuTree]);
  const expandableMenuIds = useMemo(() => collectExpandableMenuIds(menuTree), [menuTree]);

  const isBuiltIn = useCallback((role: Role) => role.builtIn === 1 || role.builtIn === 2, []);

  // 编辑/查看面板打开时的统一回显逻辑（修复菜单权限 + 数据权限回显空白 bug）。
  //
  // 回显数据来源：
  //   - 菜单权限（permissions） → adminApi.getRoleById(id) 的 permissions 字段
  //     列表接口 ListWithPagination 不填充 permissions（性能考虑），用单角色详情兜底
  //   - 数据权限（deviceGroupIds + networkTypes） → adminApi.getRoleDeviceGroups(id)
  //     network_types 只有专项端点返回；列表接口的 device_group_ids 不带制式
  //
  // 列表数据中的 role.deviceGroupIds（v0.5 起后端已填充）作为「⚠️ 未绑分组」标识用，
  // 不参与编辑面板回显，避免与专项端点结果冲突。
  const loadRoleDetailToForm = useCallback((role: Role) => {
    const generation = ++roleBindingLoadGeneration.current;
    setIsRoleBindingsLoading(true);
    setSelectedRole(role);
    form.setFieldsValue({
      roleName: role.roleName,
      description: role.description,
    });
    // 默认折叠：菜单 / API 权限打开编辑面板时不预展开。
    // 用户需要展开时手动点击复选框或单个目录节点。
    setExpandedPermissionKeys([]);
    setExpandedApiGroupKeys([]);
    // 清空快速回显，避免上一个角色的菜单 ID 残留
    setCheckedPermissionKeys([]);
    setOriginalButtonIds([]);
    setSelectedApiEndpointIds([]);
    setSuggestedApiEndpointIds([]);
    setAutoSelectedApiEndpointIds([]);
    setSelectedDeviceGroupIds(role.deviceGroupIds || []);
    setSelectedNetworkTypes(role.networkTypes || []);

    void (async () => {
      try {
        const [allIds, deviceGroupData, endpointIds] = await Promise.all([
          fetchRoleMenuIds(role.id),
          adminApi.getRoleDeviceGroups(role.id),
          apiPermissionApi.getRolePermissions(role.id),
        ]);
        // 抽屉已关闭或用户已切换到另一个角色时，丢弃迟到响应，避免把角色 A
        // 的权限写进角色 B / 新增抽屉的共享 state。
        if (generation !== roleBindingLoadGeneration.current) return;

        // 把菜单 ID 拆成可视叶子和隐藏按钮。目录 ID 不直接回喂给 antd，
        // 否则联动模式会把用户取消的兄弟菜单重新展示为勾选。
        const menuSet = new Set(allMenuIds);
        const flat: Menu[] = [];
        const walk = (list: Menu[]) => {
          for (const m of list) {
            flat.push(m);
            if (m.children?.length) walk(m.children);
          }
        };
        walk(menuTree);
        const byId = new Map<string, Menu>();
        for (const m of flat) byId.set(m.id, m);
        const leafIds: string[] = [];
        for (const id of allIds) {
          if (!menuSet.has(id)) continue;
          const m = byId.get(id);
          if (m && isVisibleLeaf(m)) leafIds.push(id);
        }
        setCheckedPermissionKeys(leafIds);
        setOriginalButtonIds(allIds.filter((id) => !menuSet.has(id)));
        setSelectedDeviceGroupIds(deviceGroupData.deviceGroupIds);
        setSelectedNetworkTypes(deviceGroupData.networkTypes);
        // 编辑/查看严格以数据库值回显，不执行创建态建议。
        setSelectedApiEndpointIds(endpointIds);
      } catch (err) {
        if (generation !== roleBindingLoadGeneration.current) return;
        const msg = err instanceof Error ? err.message : t('empty.loadFailed');
        message.error(msg);
      } finally {
        if (generation === roleBindingLoadGeneration.current) {
          setIsRoleBindingsLoading(false);
        }
      }
    })();
  }, [
    form,
    allMenuIds,
    menuTree,
    message,
    t,
  ]);

  // 已有的角色名称列表（用于重复检查）
  const existingRoleNames = useMemo(
    () => (data?.items ?? []).map((r) => r.roleName.toLowerCase()),
    [data?.items]
  );

  // permissionTreeData / allMenuIds / expandableMenuIds 已在上方根据 menuTree 计算。

  /**
   * 把当前 checkedPermissionKeys（菜单 ID 数组）转成提交给后端的 menu_ids 数组。
   *
   * 组成两部分：
   *   1. 树里勾选且仍存在于菜单树的 directory/menu ID（防止旧 cache 脏 ID 漏到 PUT）
   *   2. 原 role_menus 中的 button ID 里「祖先菜单仍勾选」的子集
   *
   * 第 2 部分是保留按钮级 RBAC 的关键：菜单权限树不展示 button 节点，但 role_menus
   * 表的按钮绑定要保留下去——否则用户每次保存角色都会把所有按钮 RBAC 清掉。同时
   * 用户取消某个父菜单时，其下的按钮会被一起摘出 PUT，避免后端 expandMenuAncestors
   * 把父菜单从按钮反向补回（这正是"取消设备规则保存后又冒出来"的成因）。
   */
  const permissionsToMenuIds = useCallback((keys: React.Key[]): string[] => {
    const valid = new Set(allMenuIds);
    const menuIds = (keys as string[]).filter((k) => valid.has(k));
    const checkedMenuSet = new Set(menuIds);
    const liveButtons = new Set(buttonIdsUnderMenus(menuTree, checkedMenuSet));
    const preservedButtons = originalButtonIds.filter((id) => liveButtons.has(id));
    return [...menuIds, ...preservedButtons];
  }, [allMenuIds, originalButtonIds, menuTree]);

  // 至少选了一个菜单（无论目录/菜单/按钮）即视为有权限。
  const hasAnyPermission = useMemo(
    () => permissionsToMenuIds(checkedPermissionKeys).length > 0,
    [checkedPermissionKeys, permissionsToMenuIds]
  );

  const applyCreateApiSuggestions = useCallback((keys: React.Key[]) => {
    if (!createVisible) return;
    const menuIds = permissionsToMenuIds(keys);
    const suggestions = inferReadApiEndpointIds(menuIds, menuTree, apiEndpoints);
    const next = applyNewApiSuggestions(
      selectedApiEndpointIds,
      suggestions,
      suggestedApiEndpointIds,
      autoSelectedApiEndpointIds,
    );
    setSelectedApiEndpointIds(next.selectedIds);
    setSuggestedApiEndpointIds(next.suggestedIds);
    setAutoSelectedApiEndpointIds(next.autoSelectedIds);
  }, [
    createVisible,
    permissionsToMenuIds,
    menuTree,
    apiEndpoints,
    selectedApiEndpointIds,
    suggestedApiEndpointIds,
    autoSelectedApiEndpointIds,
  ]);

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
          onError: (err) => {
            message.error(
              getRoleDeleteErrorMessage(err, t('common.deleteFailed'), t('role.deleteInUse'))
            );
          },
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
      title: t('role.copyConfirmTitle'),
      content: t('role.copyConfirmContent', { roleName: role.roleName }),
      onOk: () => {
        copyRoleMut.mutate(role.id, {
          onSuccess: (newRole) => {
            message.success(t('role.copySuccess', { roleName: newRole.roleName }));
            // 强制刷新列表
            void refetch();
          },
          onError: (err: unknown) => {
            const msg = err instanceof Error ? err.message : t('role.copyFailed');
            message.error(msg);
          },
        });
      },
    });
  }, [copyRoleMut, modal, message, refetch, t]);

  const handleBatchDelete = useCallback((keys: React.Key[]) => {
    const selectedRoles = (data?.items || []).filter((r) => keys.includes(r.id));
    const builtInRoles = selectedRoles.filter((r) => isBuiltIn(r));
    const inUseRoles = selectedRoles.filter((r) => !isBuiltIn(r) && (r.userCount ?? 0) > 0);
    const rolesToDelete = selectedRoles.filter(
      (r) => !isBuiltIn(r) && (r.userCount ?? 0) === 0
    );
    if (rolesToDelete.length === 0) {
      const messages = [
        builtInRoles.length > 0
          ? `${t('role.selectedBuiltIn')} ${builtInRoles.length} ${t('role.builtInSkipped')}`
          : '',
        inUseRoles.length > 0
          ? t('role.deleteInUseNamed', { names: inUseRoles.map((r) => r.roleName).join('、') })
          : '',
      ].filter(Boolean);
      modal.warning({
        title: t('common.warning'),
        content: messages.length > 0 ? messages.join('；') : t('role.noRolesToDelete'),
      });
      return;
    }
    const builtInCount = builtInRoles.length;
    modal.confirm({
      title: t('common.confirmDelete'),
      content: (
        <div>
          {builtInCount > 0 ? (
            <div>{`${t('role.selectedBuiltIn')} ${builtInCount} ${t('role.builtInSkipped')}`}</div>
          ) : null}
          {inUseRoles.length > 0 ? (
            <div>{t('role.deleteWillSkipInUse', { names: inUseRoles.map((r) => r.roleName).join('、') })}</div>
          ) : null}
        </div>
      ),
      onOk: () => {
        deleteRoles.mutate(rolesToDelete.map((r) => r.id), {
          onSuccess: () => {
            message.success(t('common.deleteSuccess'));
            setSelectedKeys([]);
          },
          onError: (err) => {
            message.error(
              getRoleDeleteErrorMessage(err, t('common.deleteFailed'), t('role.deleteInUse'))
            );
          },
        });
      },
    });
  }, [data?.items, isBuiltIn, t, deleteRoles, modal, message]);

  // 校验并提交创建
  const handleCreate = useCallback(() => {
    if (savingRef.current) return;
    // 校验权限
    if (!hasAnyPermission) {
      message.warning(t('role.pleaseSelectPermission'));
      return;
    }
    if (selectedApiEndpointIds.length === 0) {
      message.warning(t('role.pleaseSelectApiPermission'));
      return;
    }
    // 校验设备组（至少选择一个二级节点）
      const selectedDataPermissions = selectedDeviceGroupIds.filter((id) =>
        allDataPermissionGroupIds.includes(id)
      );
      if (selectedDataPermissions.length === 0) {
      message.warning(t('role.pleaseSelectDeviceGroup'));
      return;
    }

    savingRef.current = true;
    setIsSaving(true);
    void (async () => {
      try {
        const vals = await form.validateFields();
        const menuIds = permissionsToMenuIds(checkedPermissionKeys);
        const newRole = await createRole.mutateAsync(
          {
            roleName: vals.roleName as string,
            description: (vals.description as string) ?? '',
            // P3：permissions 字段已无意义（permissions 表 DROP），传空数组兼容 Role 类型签名。
            // 真正的菜单权限由下方 setRoleMenusMut 写入 role_menus。
            permissions: [],
            deviceGroupIds: selectedDeviceGroupIds,
            networkTypes: selectedNetworkTypes,
            builtIn: 0,
          } as unknown as Parameters<typeof createRole.mutateAsync>[0],
        );
        if (newRole?.id) {
          // 串行 await 三个专项端点：
          //   - 任一失败立即 throw，下面 catch 弹真实 message
          //   - 不再写"成功 toast 已弹，setRoleMenus 静默 500"这种状态错位
          await saveRolePermissionBindings({
            setMenus: setRoleMenusMut.mutateAsync,
            setApiPermissions: setRoleApiPermissionsMut.mutateAsync,
            setDeviceGroups: setRoleDeviceGroupsMut.mutateAsync,
          }, {
            roleId: newRole.id,
            menuIds,
            endpointIds: selectedApiEndpointIds,
            deviceGroupIds: selectedDeviceGroupIds,
            networkTypes: selectedNetworkTypes,
          });
        }
        // 让当前用户的侧边栏菜单立即刷新（非 builtIn 用户）。
        await invalidateUserMenus();
        // 刷新角色列表：createRole 的 invalidate 在专项端点（设备分组/菜单/API）写入
        // 之前就触发了，列表里的 deviceGroupIds / ⚠️ 标记会是旧值；这里在整条流程结束
        // 后再 refetch 一次，确保当前页数据为最新。
        void refetch();
        message.success(t('common.save'));
        setCreateVisible(false);
        form.resetFields();
        setCheckedPermissionKeys([]);
        setOriginalButtonIds([]);
        setExpandedPermissionKeys([]);
        setSelectedApiEndpointIds([]);
        setSuggestedApiEndpointIds([]);
        setAutoSelectedApiEndpointIds([]);
        setExpandedApiGroupKeys([]);
        setSelectedDeviceGroupIds([]);
        setSelectedNetworkTypes([]);
      } catch (err) {
        const msg = err instanceof Error ? err.message : t('common.saveFailed');
        message.error(msg);
      } finally {
        savingRef.current = false;
        setIsSaving(false);
      }
    })();
    }, [form, createRole, checkedPermissionKeys, selectedDeviceGroupIds, selectedNetworkTypes, selectedApiEndpointIds, hasAnyPermission, allDataPermissionGroupIds, permissionsToMenuIds, setRoleMenusMut, setRoleApiPermissionsMut, setRoleDeviceGroupsMut, invalidateUserMenus, refetch, message, t]);

  // doEditSubmit 拆出实际提交逻辑，配合下方"清空设备分组二次确认"复用。
  // 必须先于 handleEdit 声明，否则 React 的 useCallback 会触发 react-hooks/refs：
  // "Cannot access doEditSubmit before it is declared"。
  const doEditSubmit = useCallback(() => {
    if (!selectedRole || savingRef.current || isRoleBindingsLoading) return;
    savingRef.current = true;
    setIsSaving(true);
    void (async () => {
      try {
        const vals = await form.validateFields();
        const menuIds = permissionsToMenuIds(checkedPermissionKeys);
        await updateRole.mutateAsync({
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
        });
        // 串行 await：任一专项端点失败立即 throw，下面 catch 弹真实 message。
        // 避免「updateRole 成功 toast 已弹，setRoleMenusMut 后台 500」的状态错位
        // ——这正是"取消设备规则保存后却没生效"的另一支可能源。
        await saveRolePermissionBindings({
          setMenus: setRoleMenusMut.mutateAsync,
          setApiPermissions: setRoleApiPermissionsMut.mutateAsync,
          setDeviceGroups: setRoleDeviceGroupsMut.mutateAsync,
        }, {
          roleId: selectedRole.id,
          menuIds,
          endpointIds: selectedApiEndpointIds,
          deviceGroupIds: selectedDeviceGroupIds,
          networkTypes: selectedNetworkTypes,
        });
        // 让当前用户的侧边栏菜单立即刷新（非 builtIn 用户）。
        await invalidateUserMenus();
        // 刷新角色列表：updateRole 的 invalidate 在专项端点（设备分组/菜单/API）写入
        // 之前就触发了，列表里的 deviceGroupIds / ⚠️ 标记会是旧值；这里在整条流程结束
        // 后再 refetch 一次，确保当前页数据为最新。
        void refetch();
        message.success(t('common.save'));
        setEditVisible(false);
        form.resetFields();
        setSelectedRole(null);
        setCheckedPermissionKeys([]);
        setOriginalButtonIds([]);
        setExpandedPermissionKeys([]);
        setSelectedApiEndpointIds([]);
        setSuggestedApiEndpointIds([]);
        setAutoSelectedApiEndpointIds([]);
        setExpandedApiGroupKeys([]);
        setSelectedDeviceGroupIds([]);
        setSelectedNetworkTypes([]);
      } catch (err) {
        const msg = err instanceof Error ? err.message : t('common.saveFailed');
        message.error(msg);
      } finally {
        savingRef.current = false;
        setIsSaving(false);
      }
    })();
  }, [selectedRole, isRoleBindingsLoading, form, updateRole, checkedPermissionKeys, selectedDeviceGroupIds, selectedNetworkTypes, selectedApiEndpointIds, permissionsToMenuIds, setRoleMenusMut, setRoleApiPermissionsMut, setRoleDeviceGroupsMut, invalidateUserMenus, refetch, message, t]);

  // 校验并提交编辑
  const handleEdit = useCallback(() => {
    if (!selectedRole || savingRef.current || isRoleBindingsLoading) return;

    // 校验权限
    if (!hasAnyPermission) {
      message.warning(t('role.pleaseSelectPermission'));
      return;
    }
    if (selectedApiEndpointIds.length === 0) {
      message.warning(t('role.pleaseSelectApiPermission'));
      return;
    }

    // 中层：清空设备分组二次确认 + 实际保存。
    // §11.2 决议 ① 触点 5：选 0 个二级节点不再硬阻止，改为弹 Modal 警告 → OK 才继续。
    const proceed = () => {
        const selectedDataPermissions = selectedDeviceGroupIds.filter((id) =>
          allDataPermissionGroupIds.includes(id)
        );
        if (selectedDataPermissions.length === 0) {
        modal.confirm({
          title: t('role.clearGroupBindingTitle'),
          content: t('role.clearGroupBindingContent'),
          okText: t('role.continueSave'),
          okButtonProps: { danger: true },
          cancelText: t('common.cancel'),
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
        title: t('role.editBuiltinTitle'),
        content: t('role.editBuiltinContent', { roleName: selectedRole.roleName }),
        okText: t('role.continueSave'),
        okButtonProps: { danger: true },
        cancelText: t('common.cancel'),
        onOk: proceed,
      });
      return;
    }

    proceed();
    }, [selectedRole, isRoleBindingsLoading, hasAnyPermission, selectedApiEndpointIds, isBuiltIn, selectedDeviceGroupIds, allDataPermissionGroupIds, doEditSubmit, message, modal, t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'roleName', label: t('role.roleName'), type: 'input', placeholder: t('role.roleName'), width: 240 },
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
            label: t('common.copy'),
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
        const realDeviceGroupIds = (role.deviceGroupIds ?? []).filter((id) => id !== UNASSIGNED_GROUP_ID);
        const noGroups = realDeviceGroupIds.length === 0 && !(role.deviceGroupIds ?? []).includes(UNASSIGNED_GROUP_ID);
        return (
          <span>
            {String(val)}
            {isBuiltIn(role) && <Tag color="blue" style={{ marginLeft: 8 }}>{t('role.builtIn')}</Tag>}
            {!isBuiltIn(role) && noGroups && (
              <Tag color="warning" style={{ marginLeft: 8 }}>{t('role.noGroupBinding.tag')}</Tag>
            )}
          </span>
        );
      },
    },
    { key: 'description', title: t('role.roleDescription'), dataIndex: 'description', width: 150, ellipsis: true, render: (v) => (v as string) || '-' },
    {
      key: 'userCount',
      title: t('role.userCount'),
      dataIndex: 'userCount',
      width: 100,
      render: (val) => String((val as number | undefined) ?? 0),
    },
    { key: 'createUser', title: t('role.createUser'), dataIndex: 'createUser', width: 100, render: (v, record) => renderRoleOperator(v, record, t('role.builtIn')) },
    {
      key: 'createTime',
      title: t('role.createTime'),
      dataIndex: 'createTime',
      width: 160,
      render: (val) => (val ? formatSystemTime(String(val)) : '-') as string,
    },
    { key: 'updateUser', title: t('role.updateUser'), dataIndex: 'updateUser', width: 100, render: (v, record) => renderRoleOperator(v, record, t('role.builtIn')) },
    {
      key: 'updateTime',
      title: t('role.updateTime'),
      dataIndex: 'updateTime',
      width: 160,
      render: (val) => (val ? formatSystemTime(String(val)) : '-') as string,
    },
  ], [t, isBuiltIn, handleDelete, handleCopy, loadRoleDetailToForm]);

  // 渲染菜单权限配置（树形结构 - 按图片样式）
  const renderPermissionConfig = (readOnly = false) => {
    const controlsDisabled = readOnly || isSaving || isRoleBindingsLoading;
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
      const keys = checked ? allMenuIds : [];
      setCheckedPermissionKeys(keys);
      applyCreateApiSuggestions(keys);
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
      const keys = Array.isArray(checked) ? checked : checked.checked;
      setCheckedPermissionKeys(keys);
      applyCreateApiSuggestions(keys);
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
        help={!readOnly && !isRoleBindingsLoading && !hasAnyPermission ? t('role.pleaseSelectPermission') : undefined}
        validateStatus={!readOnly && !isRoleBindingsLoading && !hasAnyPermission ? 'warning' : undefined}
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
              disabled={controlsDisabled}
            >
              {t('role.expandCollapse')}
            </Checkbox>
            <Checkbox
              checked={isAllSelected}
              indeterminate={isIndeterminate}
              onChange={(e) => handleSelectAllChange(e.target.checked)}
              disabled={controlsDisabled}
            >
              {t('role.selectAllOrNone')}
            </Checkbox>
            <Checkbox
              checked={!permissionCheckStrictly}
              onChange={(e) => handleLinkageChange(e.target.checked)}
              disabled={controlsDisabled}
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
                disabled={controlsDisabled}
                onExpand={handleExpand}
                onCheck={controlsDisabled ? undefined : handleCheck}
              />
            )}
          </div>
        </div>
      </Form.Item>
    );
  };

  const renderApiPermissionConfig = (readOnly = false) => {
    const controlsDisabled = readOnly || isSaving || isRoleBindingsLoading;
    const isAllExpanded =
      apiGroupParentKeys.length > 0 &&
      expandedApiGroupKeys.length >= apiGroupParentKeys.length;
    const isAllSelected =
      allApiEndpointIds.length > 0 &&
      selectedApiEndpointIds.length === allApiEndpointIds.length;
    const isIndeterminate =
      selectedApiEndpointIds.length > 0 &&
      selectedApiEndpointIds.length < allApiEndpointIds.length;

    const handleCheck: TreeProps['onCheck'] = (checked) => {
      const keys = Array.isArray(checked) ? checked : checked.checked;
      const endpointIds = (keys as React.Key[]).filter(
        (key): key is string => typeof key === 'string' && !key.startsWith('group:'),
      );
      setSelectedApiEndpointIds(endpointIds);
      // 用户取消自动建议后，把它移出 automatic 集合但保留在 suggested
      // 历史中，后续菜单变化不会静默重新勾选；用户手动重新勾选则视为显式授权。
      setAutoSelectedApiEndpointIds((previous) =>
        previous.filter((id) => endpointIds.includes(id)),
      );
    };

    const treeCheckedKeys: TreeProps['checkedKeys'] = apiCheckStrictly
      ? { checked: selectedApiEndpointIds, halfChecked: [] }
      : selectedApiEndpointIds;

    return (
      <Form.Item
        label={t('role.apiPermission')}
        required={!readOnly}
        help={
          !readOnly && !isRoleBindingsLoading && selectedApiEndpointIds.length === 0
            ? t('role.pleaseSelectApiPermission')
            : undefined
        }
        validateStatus={
          !readOnly && !isRoleBindingsLoading && selectedApiEndpointIds.length === 0 ? 'warning' : undefined
        }
      >
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
              onChange={(event) => {
                setExpandedApiGroupKeys(event.target.checked ? apiGroupParentKeys : []);
              }}
              disabled={controlsDisabled || apiGroupParentKeys.length === 0}
            >
              {t('role.expandCollapse')}
            </Checkbox>
            <Checkbox
              checked={isAllSelected}
              indeterminate={isIndeterminate}
              onChange={(event) => {
                setSelectedApiEndpointIds(event.target.checked ? allApiEndpointIds : []);
                // 全选/全不选是管理员显式操作，不再把任何端点视作自动授权。
                setAutoSelectedApiEndpointIds([]);
              }}
              disabled={controlsDisabled || allApiEndpointIds.length === 0}
            >
              {t('role.selectAllOrNone')}
            </Checkbox>
            <Checkbox
              checked={!apiCheckStrictly}
              onChange={(event) => setApiCheckStrictly(!event.target.checked)}
              disabled={controlsDisabled}
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
              <div style={{ textAlign: 'center', padding: 24 }}>
                <Spin />
              </div>
            ) : (
              <Tree
                treeData={apiTreeData}
                expandedKeys={expandedApiGroupKeys}
                checkedKeys={treeCheckedKeys}
                selectable={false}
                checkable
                disabled={controlsDisabled}
                checkStrictly={apiCheckStrictly}
                onCheck={controlsDisabled ? undefined : handleCheck}
                onExpand={(expanded) => setExpandedApiGroupKeys(expanded as React.Key[])}
              />
            )}
          </div>
        </div>
      </Form.Item>
    );
  };

  // 渲染设备组树形选择（包含网络类型权限）
  const renderDeviceGroupTree = (readOnly = false) => {
    const controlsDisabled = readOnly || isSaving || isRoleBindingsLoading;
    // 只检查真正承载设备的数据权限项是否被选中（基于筛选后的数据）
    const selectedDataPermissionCount = selectedDeviceGroupIds.filter((id) =>
      allDataPermissionGroupIds.includes(id)
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
          filteredSelectableGroupIds.forEach((id) => newIds.add(id));
          return Array.from(newIds);
        });
      } else {
        // 取消选中所有筛选后的节点（包括一级和二级）
        setSelectedDeviceGroupIds((prev) =>
          prev.filter((id) => !filteredSelectableGroupIds.includes(id) && !filteredFirstLevelIds.includes(id))
        );
      }
    };

    // 是否全选（基于筛选后的数据）
    const isAllSelected =
      filteredSelectableGroupIds.length > 0 &&
      filteredSelectableGroupIds.every((id) => selectedDeviceGroupIds.includes(id));

    // 是否部分选中
    const isIndeterminate =
      selectedDataPermissionCount > 0 && selectedDataPermissionCount < filteredSelectableGroupIds.length;

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
                disabled={controlsDisabled}
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
          help={!readOnly && !isRoleBindingsLoading && selectedDataPermissionCount === 0 ? t('role.pleaseSelectDeviceGroup') : undefined}
          validateStatus={!readOnly && !isRoleBindingsLoading && selectedDataPermissionCount === 0 ? 'warning' : undefined}
        >
          {readOnly ? (
            <Space wrap>
              {selectedDeviceGroupIds.length > 0 ? (
                (allDeviceGroups ?? [])
                  .filter((g) => selectedDeviceGroupIds.includes(g.id))
                  .map((g) => <Tag key={g.id}>{getI18nText(g.nameI18n, locale, g.name)}</Tag>)
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
                  options={buildNetworkTypeOptions(t)}
                  placeholder={t('role.baseStationType')}
                  disabled={controlsDisabled}
                />
                <Select
                  size="small"
                  style={{ width: 120 }}
                  value={deviceGroupProductClass}
                  onChange={(val) => setDeviceGroupProductClass(val)}
                  options={buildProductTypeOptions(t)}
                  placeholder={t('role.productClass')}
                  disabled={controlsDisabled}
                />
                <Divider orientation="vertical" style={{ height: 20, margin: 0 }} />
                <Checkbox
                  checked={isAllSelected}
                  indeterminate={isIndeterminate}
                  onChange={(e) => handleSelectAll(e.target.checked)}
                  disabled={controlsDisabled}
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
                    disabled={controlsDisabled}
                    onCheck={controlsDisabled ? undefined : (checked) => {
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

  // 关闭抽屉时重置状态
  const handleOpenCreate = useCallback(() => {
    // 新增/编辑/查看共用一组表单状态；使任何迟到的角色详情响应立即失效。
    roleBindingLoadGeneration.current += 1;
    setIsRoleBindingsLoading(false);
    setSelectedRole(null);
    form.resetFields();
    setCheckedPermissionKeys([]);
    setOriginalButtonIds([]);
    setExpandedPermissionKeys([]);
    setSelectedApiEndpointIds([]);
    setSuggestedApiEndpointIds([]);
    setAutoSelectedApiEndpointIds([]);
    setExpandedApiGroupKeys([]);
    setSelectedDeviceGroupIds([]);
    setSelectedNetworkTypes([]);
    setActiveTab('menu');
    setCreateVisible(true);
  }, [form]);

  const handleCloseCreate = useCallback(() => {
    if (savingRef.current) return;
    roleBindingLoadGeneration.current += 1;
    setIsRoleBindingsLoading(false);
    setCreateVisible(false);
    form.resetFields();
    setCheckedPermissionKeys([]);
    setOriginalButtonIds([]);
    setExpandedPermissionKeys([]);
    setSelectedApiEndpointIds([]);
    setSuggestedApiEndpointIds([]);
    setAutoSelectedApiEndpointIds([]);
    setExpandedApiGroupKeys([]);
    setSelectedDeviceGroupIds([]);
    setSelectedNetworkTypes([]);
    setActiveTab('menu');
  }, [form]);

  const handleCloseEdit = useCallback(() => {
    if (savingRef.current) return;
    roleBindingLoadGeneration.current += 1;
    setIsRoleBindingsLoading(false);
    setEditVisible(false);
    form.resetFields();
    setSelectedRole(null);
    setCheckedPermissionKeys([]);
    setOriginalButtonIds([]);
    setExpandedPermissionKeys([]);
    setSelectedApiEndpointIds([]);
    setSuggestedApiEndpointIds([]);
    setAutoSelectedApiEndpointIds([]);
    setExpandedApiGroupKeys([]);
    setSelectedDeviceGroupIds([]);
    setSelectedNetworkTypes([]);
    setActiveTab('menu');
  }, [form]);

  const handleCloseView = useCallback(() => {
    roleBindingLoadGeneration.current += 1;
    setIsRoleBindingsLoading(false);
    setViewVisible(false);
    form.resetFields();
    setSelectedRole(null);
    setCheckedPermissionKeys([]);
    setOriginalButtonIds([]);
    setExpandedPermissionKeys([]);
    setSelectedApiEndpointIds([]);
    setSuggestedApiEndpointIds([]);
    setAutoSelectedApiEndpointIds([]);
    setExpandedApiGroupKeys([]);
    setSelectedDeviceGroupIds([]);
    setSelectedNetworkTypes([]);
    setActiveTab('menu');
  }, [form]);

  return (
    <ListPageLayout>
      {/* 2026-06-03 用户决策:去"角色管理"标题;筛选与操作按钮同一行(筛选靠左、按钮靠右)。 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 12 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <FilterBar
            filterId="role-management-filter"
            fields={filterFields}
            onSearch={(vals) => { setFilters(vals); setPage(1); }}
            onReset={() => { setFilters({}); setPage(1); }}
          />
        </div>
        <Space style={{ flexShrink: 0 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleOpenCreate}>
            {t('common.add')}
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
          hideRealtime
          hideColumnSettings
          hideDensity
          hideRefresh
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
        closable={!isSaving}
        keyboard={!isSaving}
        maskClosable={!isSaving}
        size={800}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button style={{ marginRight: 8 }} onClick={handleCloseCreate} disabled={isSaving}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" loading={isSaving} onClick={handleCreate}>
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
            <Input placeholder={t('role.roleNamePlaceholder')} maxLength={200} showCount disabled={isSaving} />
          </Form.Item>
          <Form.Item name="description" label={t('role.description')}>
            <Input.TextArea
              rows={2}
              placeholder={t('role.descriptionPlaceholder')}
              maxLength={500}
              showCount
              disabled={isSaving}
            />
          </Form.Item>
        </Form>

        {/* 标签页：角色菜单、API 权限、资源权限 */}
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
        closable={!isSaving}
        keyboard={!isSaving}
        maskClosable={!isSaving}
        size={600}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button style={{ marginRight: 8 }} onClick={handleCloseEdit} disabled={isSaving}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" loading={isSaving} disabled={isRoleBindingsLoading} onClick={handleEdit}>
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
            message={t('role.builtinAlert.title')}
            description={t('role.builtinAlert.desc')}
          />
        )}
        {/* §11.2 决议 ① 触点 2：未绑设备分组的角色 banner 提示 */}
        {selectedDeviceGroupIds.length === 0 && (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
            message={t('role.noGroupBinding.title')}
            description={t('role.noGroupBinding.editDesc')}
          />
        )}
        <Form form={form} layout="vertical">
          <Form.Item
            name="roleName"
            label={t('role.roleName')}
          >
            <Input readOnly disabled={isSaving || isRoleBindingsLoading} style={{ color: 'var(--color-text-secondary)' }} />
          </Form.Item>
          <Form.Item name="description" label={t('role.description')}>
            <Input.TextArea
              rows={2}
              placeholder={t('role.descriptionPlaceholder')}
              maxLength={500}
              showCount
              readOnly={selectedRole ? isBuiltIn(selectedRole) : false}
              disabled={isSaving || isRoleBindingsLoading}
              style={selectedRole && isBuiltIn(selectedRole) ? { color: 'var(--color-text-secondary)' } : undefined}
            />
          </Form.Item>
        </Form>

        {/* 标签页：角色菜单、API 权限、资源权限 */}
        <Spin spinning={isRoleBindingsLoading}>
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
        </Spin>
      </Drawer>

      {/* View Drawer */}
      <Drawer
        title={t('common.view')}
        open={viewVisible}
        onClose={handleCloseView}
        size={600}
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
            message={t('role.noGroupBinding.title')}
            description={t('role.noGroupBinding.viewDesc')}
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

        {/* 标签页：角色菜单、API 权限、资源权限 */}
        <Spin spinning={isRoleBindingsLoading}>
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
        </Spin>

        <Divider />
        <Form.Item label={t('role.userCount')}>
          <span>{selectedRole?.userCount ?? 0}</span>
        </Form.Item>
        <Form.Item label={t('role.createUser')}>
          <span>{selectedRole ? renderRoleOperator(selectedRole.createUser, selectedRole, t('role.builtIn')) : '-'}</span>
        </Form.Item>
        <Form.Item label={t('role.createTime')}>
          <span>{selectedRole?.createTime ? formatSystemTime(selectedRole.createTime) : '-'}</span>
        </Form.Item>
        <Form.Item label={t('role.updateUser')}>
          <span>{selectedRole ? renderRoleOperator(selectedRole.updateUser, selectedRole, t('role.builtIn')) : '-'}</span>
        </Form.Item>
        <Form.Item label={t('role.updateTime')}>
          <span>{selectedRole?.updateTime ? formatSystemTime(selectedRole.updateTime) : '-'}</span>
        </Form.Item>
      </Drawer>
    </ListPageLayout>
  );
}
