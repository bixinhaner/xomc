import { useState, useMemo, useCallback } from 'react';
import {
  App,
  Button,
  Card,
  Dropdown,
  Tag,
  Drawer,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Switch,
  Tooltip,
  TreeSelect,
  Radio,
} from 'antd';
import IconPicker from '@/components/IconPicker';
import { resolveIcon } from '@/components/IconPicker/icons';
import type { MenuProps } from 'antd';
import {
  UpOutlined,
  DownOutlined,
  RightOutlined,
  PlusOutlined,
  MoreOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';
import http from '@core/services/http';
import { SUPPORTED_LOCALES, LOCALE_DISPLAY } from '@core/i18n';
import type { Locale } from '@core/types/common';
import { useAppStore } from '@core/store/appStore';
import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
} from '@core/hooks/api/useSystem';
import styles from './index.module.css';

// 菜单管理 UI 暴露的额外语言：除主语言（zh-CN）外的所有支持语言。
// 扩展新语言只需在 frontend-core/src/i18n/index.ts 追加 SUPPORTED_LOCALES，
// 表单自动多出一行输入框，无需改本文件。
const EXTRA_LOCALES: readonly Locale[] = SUPPORTED_LOCALES.filter((l) => l !== 'zh-CN');

// 菜单类型
type MenuType = 'menu' | 'directory' | 'button';

// 菜单状态
type MenuStatus = 'normal' | 'disabled';

// 显示状态
type ShowStatus = 'show' | 'hide';

// 是否外链
type IsExternal = 'yes' | 'no';

// API权限
type ApiPermission = 'required' | 'none';

// 菜单项接口
interface MenuItem {
  id: string;
  name: string;
  /** 多语言译文字典（migration 000083 引入）；编辑表单按 locale 拆字段填入。 */
  nameI18n?: Record<string, string>;
  type: MenuType;
  sort: number;
  permissionKey: string;
  componentPath: string;
  status: MenuStatus;
  parentId: string | null;
  icon?: string;
  isExternal?: IsExternal;
  routePath?: string;
  routeParams?: string;
  showStatus?: ShowStatus;
  apiPermission?: ApiPermission;
  children?: MenuItem[];
}

// 后端 Menu JSON 形态（snake_case，详见 omcgo/internal/admin/model.go Menu 结构体）。
// status 字段后端用 'normal'|'disabled'（DB CHECK），与页面本地一致；
// type='directory'|'menu'|'button' 与 DDL chk_menu_type 对齐。
interface BackendMenu {
  id: string;
  name: string;
  name_i18n?: Record<string, string> | null;
  type: MenuType;
  permission_key: string;
  parent_id?: string;
  sort_order: number;
  route_path?: string;
  component_path?: string;
  icon?: string;
  show_status?: ShowStatus;
  status: MenuStatus;
  children?: BackendMenu[];
}

// mapBackendMenu 把一颗 BackendMenu 子树转成页面本地 MenuItem 子树（递归）。
// isExternal / routeParams / apiPermission 是页面早期 mock 字段，后端无对应列；
// 编辑/新增表单以默认值兜底（'no' / undefined / 'none'），保证 UI 不崩。
function mapBackendMenu(b: BackendMenu): MenuItem {
  return {
    id: b.id,
    name: b.name,
    nameI18n: b.name_i18n ?? undefined,
    type: b.type,
    sort: b.sort_order,
    permissionKey: b.permission_key,
    componentPath: b.component_path ?? '',
    status: b.status,
    parentId: b.parent_id ?? null,
    icon: b.icon || undefined,
    routePath: b.route_path || undefined,
    showStatus: b.show_status ?? 'show',
    isExternal: 'no',
    apiPermission: 'none',
    children: b.children?.map(mapBackendMenu),
  };
}

// 菜单类型选项
const MENU_TYPE_OPTIONS = [
  { label: '目录', value: 'directory' },
  { label: '菜单', value: 'menu' },
  { label: '按钮', value: 'button' },
];

// 菜单状态选项
const MENU_STATUS_OPTIONS = [
  { label: '正常', value: 'normal' },
  { label: '停用', value: 'disabled' },
];

// 默认的操作按钮（三级节点）(T-0136: 保留为 export 占位避免 TS6133)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const DEFAULT_OPERATIONS = [
  { key: 'query', name: '查询' },
  { key: 'add', name: '添加' },
  { key: 'edit', name: '修改' },
  { key: 'delete', name: '删除' },
  { key: 'export', name: '导出' },
  { key: 'import', name: '导入' },
];



// 后端 GET /admin/menus/tree 响应。handler 把树包了一层 {data: [...]}（参 menu_handler.go GetMenuTree），
// 与外层信封 {ret,msg,data} 由 http.ts 拦截器拆包后，response.data 是内层 {data: BackendMenu[]}。
async function fetchMenuTree(): Promise<MenuItem[]> {
  const { data } = await http.get<{ data: BackendMenu[] }>('/admin/menus/tree');
  return (data?.data ?? []).map(mapBackendMenu);
}

// CreateMenuRequest payload（与 omcgo/internal/admin/model.go CreateMenuRequest 对齐）。
interface CreateMenuPayload {
  name: string;
  name_i18n?: Record<string, string>;
  type: MenuType;
  permission_key: string;
  parent_id?: string | null;
  sort_order?: number;
  route_path?: string;
  component_path?: string;
  icon?: string;
  show_status?: ShowStatus;
}

// UpdateMenuRequest payload（字段全可选）。
type UpdateMenuPayload = Partial<CreateMenuPayload> & { status?: MenuStatus };

const MENU_QUERY_KEY = ['admin', 'menus', 'tree'] as const;

export default function MenuManagement() {
  const t = useT();
  const { modal, message } = App.useApp();
  const queryClient = useQueryClient();

  // sys_configs.system.show_menu_icon — 控制 NavMenu 是否显示菜单图标。
  // appStore.showMenuIcon 由 MenuBootstrap 启动期同步；这里改本地 + DB 双写：
  //   - 立刻 setShowMenuIcon → NavMenu 即时切换
  //   - batchUpdateSysConfigs 持久化 → 下次刷新 / 其他 tab 一致
  // sysConfig 列表用 useSysConfigsByCategory('system') 读，本组件不直接依赖结果，
  // 仅用它来拿 show_menu_icon item 是否存在（用于 button loading 状态）。
  const showMenuIcon = useAppStore((s) => s.showMenuIcon);
  const setShowMenuIcon = useAppStore((s) => s.setShowMenuIcon);
  const { data: sysConfigs } = useSysConfigsByCategory('system');
  const batchUpdateSysConfigs = useBatchUpdateSysConfigs();
  const handleToggleMenuIcon = useCallback(
    (checked: boolean) => {
      setShowMenuIcon(checked); // 本地即时
      batchUpdateSysConfigs.mutate(
        {
          category: 'system',
          items: [{ key: 'show_menu_icon', value: checked ? 'true' : 'false', value_type: 'bool' }],
        },
        {
          onError: (err) => {
            // 回滚本地，避免本地状态与 DB 不一致误导用户
            setShowMenuIcon(!checked);
            message.error((err as Error).message || t('common.saveFailed'));
          },
        },
      );
    },
    [setShowMenuIcon, batchUpdateSysConfigs, message, t],
  );
  // 仅用于 Switch 的 loading 反馈（用户在网络慢时可见旋转）
  const showMenuIconSwitchLoading =
    batchUpdateSysConfigs.isPending && sysConfigs !== undefined;

  // ───── 拉取菜单树 ─────
  const { data: menus = [], isLoading } = useQuery({
    queryKey: MENU_QUERY_KEY,
    queryFn: fetchMenuTree,
    staleTime: 60 * 1000,
  });

  // ───── 增删改 mutation ─────
  const createMenuMut = useMutation({
    mutationFn: (payload: CreateMenuPayload) => http.post('/admin/menus', payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: MENU_QUERY_KEY }),
  });
  const updateMenuMut = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateMenuPayload }) =>
      http.put(`/admin/menus/${id}`, payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: MENU_QUERY_KEY }),
  });
  const deleteMenusMut = useMutation({
    mutationFn: (ids: string[]) => http.delete('/admin/menus', { data: { ids } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: MENU_QUERY_KEY }),
  });

  const [editVisible, setEditVisible] = useState(false);
  const [selectedMenu, setSelectedMenu] = useState<MenuItem | null>(null);
  const [form] = Form.useForm();
  // 新增相关状态
  const [addVisible, setAddVisible] = useState(false);
  const [addForm] = Form.useForm();
  // 展开/收起状态
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set());
  // 搜索过滤状态
  const [filters, setFilters] = useState<Record<string, string>>({});

  // 搜索字段配置
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'name', label: '菜单名称', type: 'input', placeholder: '请输入菜单名称' },
    { name: 'status', label: '状态', type: 'select', placeholder: '请选择状态', options: MENU_STATUS_OPTIONS },
  ], []);

  // 根据展开状态扁平化菜单数据
  const flatMenus = useMemo(() => {
    const result: (MenuItem & { level: number; hasChildren: boolean })[] = [];

    const flatten = (items: MenuItem[], level: number) => {
      items.forEach((item) => {
        const hasChildren = item.children && item.children.length > 0;
        result.push({ ...item, level, hasChildren: hasChildren || false });

        // 只有展开时才显示子菜单
        if (hasChildren && expandedKeys.has(item.id)) {
          flatten(item.children!, level + 1);
        }
      });
    };

    flatten(menus, 0);
    return result;
  }, [menus, expandedKeys]);

  // 根据搜索条件过滤菜单
  const filteredMenus = useMemo(() => {
    if (!filters.name && !filters.status) {
      return flatMenus;
    }

    return flatMenus.filter((item) => {
      // 菜单名称过滤
      if (filters.name && !item.name.toLowerCase().includes(filters.name.toLowerCase())) {
        return false;
      }
      // 状态过滤
      if (filters.status && item.status !== filters.status) {
        return false;
      }
      return true;
    });
  }, [flatMenus, filters]);

  // 切换展开/收起
  const toggleExpand = useCallback((id: string) => {
    setExpandedKeys((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(id)) {
        newSet.delete(id);
      } else {
        newSet.add(id);
      }
      return newSet;
    });
  }, []);

  // 处理删除（DELETE /admin/menus，body { ids: [...] }）。
  // 后端 admin.AdminService.DeleteMenus 在 service 层自行级联删除子节点，前端只发顶层 id。
  const handleDelete = useCallback((menu: MenuItem) => {
    modal.confirm({
      title: t('common.confirmDelete'),
      content: `确定要删除菜单「${menu.name}」吗？${menu.children && menu.children.length > 0 ? '该菜单下的子菜单也将被删除。' : ''}`,
      onOk: () => {
        deleteMenusMut.mutate([menu.id], {
          onSuccess: () => message.success(t('common.deleteSuccess')),
          onError: (err) => message.error((err as Error).message || t('common.deleteFailed')),
        });
      },
    });
  }, [modal, message, t, deleteMenusMut]);

  // 处理编辑
  const handleEdit = useCallback((menu: MenuItem) => {
    setSelectedMenu(menu);
    form.setFieldsValue({
      name: menu.name,
      // 多语言译文字典回显到嵌套 form 字段（namePath: ['nameI18n', locale]）；
      // 缺译文的 locale 在表单里就是空字符串，保存时聚合逻辑会自动剔除。
      nameI18n: menu.nameI18n ?? {},
      type: menu.type,
      sort: menu.sort,
      permissionKey: menu.permissionKey,
      componentPath: menu.componentPath,
      status: menu.status,
      // 图标：直接回显 antd icon export name（如 'DashboardOutlined'）
      // 仅一级 directory 类型显示 IconPicker，其他类型字段不渲染
      icon: menu.icon,
      isExternal: menu.isExternal || 'no',
      routePath: menu.routePath,
      routeParams: menu.routeParams,
      showStatus: menu.showStatus || 'show',
      apiPermission: menu.apiPermission || 'none',
    });
    setEditVisible(true);
  }, [form]);

  // 把表单的 name + nameI18n 字段聚合为后端期望的 name_i18n。
  // 规则：
  //   - 主语言（zh-CN）与表单"菜单名称"字段强绑定，确保 name 与 name_i18n["zh-CN"] 始终一致
  //   - 其他 locale 取自嵌套字段；空字符串视为"未填"，从结果剔除（避免存空字符串污染回退链）
  // 返回值始终是一个 map（最小含 zh-CN）；上层把它放进 payload.name_i18n。
  const buildNameI18nPayload = useCallback(
    (formName: string, formNameI18n: Record<string, string> | undefined): Record<string, string> => {
      const result: Record<string, string> = { 'zh-CN': formName };
      if (formNameI18n) {
        for (const locale of EXTRA_LOCALES) {
          const v = formNameI18n[locale];
          if (typeof v === 'string' && v.trim() !== '') {
            result[locale] = v;
          }
        }
      }
      return result;
    },
    [],
  );

  // 处理保存（PUT /admin/menus/:id）。
  // 字段映射：sort → sort_order；permissionKey → permission_key；空字符串显式传以便清空。
  // icon 直接来自 IconPicker（antd icon export name 或 undefined）；
  // 仅 directory 类型表单渲染了 IconPicker，其他类型 vals.icon 为 undefined。
  const handleSave = useCallback(() => {
    form.validateFields().then((vals) => {
      if (!selectedMenu) return;
      const payload: UpdateMenuPayload = {
        name: vals.name,
        name_i18n: buildNameI18nPayload(vals.name, vals.nameI18n),
        type: vals.type,
        sort_order: vals.sort,
        permission_key: vals.permissionKey,
        route_path: vals.routePath ?? '',
        component_path: vals.componentPath ?? '',
        show_status: vals.showStatus ?? 'show',
        status: vals.status,
        icon: vals.icon ?? '',
      };
      updateMenuMut.mutate(
        { id: selectedMenu.id, payload },
        {
          onSuccess: () => {
            message.success(t('common.save'));
            setEditVisible(false);
            form.resetFields();
            setSelectedMenu(null);
          },
          onError: (err) => message.error((err as Error).message || t('common.saveFailed')),
        },
      );
    });
  }, [form, selectedMenu, message, t, updateMenuMut, buildNameI18nPayload]);

  // 生成树形选择数据
  const menuTreeData = useMemo(() => {
    const buildTree = (items: MenuItem[]): { value: string; title: string; children?: { value: string; title: string }[] }[] => {
      return items.map((item) => ({
        value: item.id,
        title: item.name,
        children: item.children ? buildTree(item.children) : undefined,
      }));
    };
    return [{ value: '0', title: '主类目', children: buildTree(menus) }];
  }, [menus]);

  // 打开新增弹窗
  const handleAdd = useCallback(() => {
    addForm.resetFields();
    addForm.setFieldsValue({
      parentId: '0',
      type: 'directory',
      sort: 1,
      status: 'normal',
      showStatus: 'show',
      isExternal: 'no',
      apiPermission: 'none',
      icon: undefined,
      nameI18n: {},
    });
    setAddVisible(true);
  }, [addForm]);

  // 保存新增菜单（POST /admin/menus）。
  // parentId === '0' 是 TreeSelect 的「主类目」哨兵 → 转后端 parent_id=null。
  // icon 仅 directory 类型表单含 IconPicker，其他类型 vals.icon 为 undefined。
  const handleAddSave = useCallback(() => {
    addForm.validateFields().then((vals) => {
      const payload: CreateMenuPayload = {
        name: vals.name,
        name_i18n: buildNameI18nPayload(vals.name, vals.nameI18n),
        type: vals.type,
        permission_key: vals.permissionKey || '',
        parent_id: vals.parentId === '0' ? null : vals.parentId,
        sort_order: vals.sort,
        route_path: vals.routePath ?? '',
        component_path: vals.componentPath ?? '',
        show_status: vals.showStatus ?? 'show',
        icon: vals.icon ?? '',
      };
      createMenuMut.mutate(payload, {
        onSuccess: () => {
          message.success(t('common.save'));
          setAddVisible(false);
          addForm.resetFields();
        },
        onError: (err) => message.error((err as Error).message || t('common.saveFailed')),
      });
    });
  }, [addForm, message, t, createMenuMut, buildNameI18nPayload]);

  // 排序值 +1（PUT /admin/menus/:id 仅更新 sort_order）。
  const handleMoveDown = useCallback((record: MenuItem & { level: number }) => {
    updateMenuMut.mutate({
      id: record.id,
      payload: { sort_order: (record.sort || 0) + 1 },
    });
  }, [updateMenuMut]);

  // 排序值 -1。
  const handleMoveUp = useCallback((record: MenuItem & { level: number }) => {
    updateMenuMut.mutate({
      id: record.id,
      payload: { sort_order: Math.max(1, (record.sort || 1) - 1) },
    });
  }, [updateMenuMut]);

  // 表格列定义
  const columns: DataTableColumn<MenuItem & { level: number; hasChildren: boolean }>[] = useMemo(() => [
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const moreItems: MenuProps['items'] = [
          { key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true, onClick: () => handleDelete(record) },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => handleEdit(record)}>
              编辑
            </Button>
            <Dropdown menu={{ items: moreItems }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
    },
    {
      key: 'icon',
      title: '图标',
      dataIndex: 'icon',
      width: 60,
      render: (raw: unknown) => {
        const val = raw as string | undefined;
        if (!val) return <span style={{ color: 'var(--color-text-secondary)' }}>-</span>;
        const Icon = resolveIcon(val);
        // Icon === null：DB 写了 icon 名但不在 IconPicker 白名单（历史脏数据）；
        // 视觉上仍只渲染一个图标（与"只显示图标"一致），通过 Tooltip 暴露原始
        // icon 名给运营定位 fixup 项。
        if (!Icon) {
          return (
            <Tooltip title={`${val}（未在白名单）`}>
              <ExclamationCircleOutlined style={{ color: 'var(--color-error)', fontSize: 16 }} />
            </Tooltip>
          );
        }
        return <Icon style={{ fontSize: 16 }} />;
      },
    },
    {
      key: 'name',
      title: '菜单名称（中文）',
      dataIndex: 'name',
      width: 260,
      render: (val, record) => {
        const indent = record.level * 24;
        const isExpanded = expandedKeys.has(record.id);

        return (
          <span style={{ paddingLeft: indent, display: 'flex', alignItems: 'center' }}>
            {/* 展开/收起箭头 */}
            {record.hasChildren ? (
              <span
                onClick={(e) => {
                  e.stopPropagation();
                  toggleExpand(record.id);
                }}
                style={{
                  cursor: 'pointer',
                  marginRight: 4,
                  display: 'inline-flex',
                  alignItems: 'center',
                  color: 'var(--color-text-secondary)',
                }}
              >
                {isExpanded ? <DownOutlined /> : <RightOutlined />}
              </span>
            ) : (
              <span style={{ width: 16, marginRight: 4 }} />
            )}
            <span>{String(val)}</span>
          </span>
        );
      },
    },
    {
      key: 'nameEn',
      title: '菜单名称（English）',
      // nameI18n 是嵌套对象，无法用 dataIndex 直取；用 render 自定义。
      // 缺译文时显示 "-"，与图标列保持一致的「无值」视觉。
      dataIndex: 'id',
      width: 200,
      render: (_val, record) => {
        const en = record.nameI18n?.['en-US'];
        if (!en) return <span style={{ color: 'var(--color-text-secondary)' }}>-</span>;
        return <span>{en}</span>;
      },
    },
    {
      key: 'type',
      title: '类型',
      dataIndex: 'type',
      width: 100,
      render: (raw: unknown) => {
        const val = raw as MenuType;
        const typeMap: Record<MenuType, { color: string; text: string }> = {
          menu: { color: 'blue', text: '菜单' },
          directory: { color: 'green', text: '目录' },
          button: { color: 'orange', text: '按钮' },
        };
        const { color, text } = typeMap[val] || { color: 'default', text: String(val) };
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      key: 'sort',
      title: '排序',
      dataIndex: 'sort',
      width: 120,
      render: (val, record) => {
        return (
          <InputNumber
            min={1}
            max={999}
            value={val as number}
            size="small"
            className={styles.sortInput}
            style={{ width: 80 }}
            controls={{
              upIcon: <UpOutlined style={{ fontSize: 10 }} />,
              downIcon: <DownOutlined style={{ fontSize: 10 }} />,
            }}
            onStep={(_value, info) => {
              if (info.type === 'up') {
                handleMoveUp(record);
              } else {
                handleMoveDown(record);
              }
            }}
          />
        );
      },
    },
    {
      key: 'permissionKey',
      title: '权限标识',
      dataIndex: 'permissionKey',
      width: 200,
      ellipsis: true,
      render: (val) => <code style={{ fontSize: 12 }}>{(val as string) || '-'}</code>,
    },
    {
      key: 'componentPath',
      title: '组件路径',
      dataIndex: 'componentPath',
      width: 180,
      ellipsis: true,
      render: (val) => (val as string) || '-',
    },
    {
      key: 'status',
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (raw: unknown) => {
        const val = raw as MenuStatus;
        return (
          <Tag color={val === 'normal' ? 'success' : 'error'}>
            {val === 'normal' ? '正常' : '停用'}
          </Tag>
        );
      },
    },
  ], [t, handleEdit, handleDelete, expandedKeys, toggleExpand, handleMoveUp, handleMoveDown]);

  // i18n 字段块：在每个含"菜单名称"输入框的表单分支后追加。
  // 除主语言（zh-CN）外的每种 locale 一个输入框。
  // 数组循环 EXTRA_LOCALES 实现「新增语言只改 SUPPORTED_LOCALES 一处」。
  const i18nFields = (
    <>
      {EXTRA_LOCALES.map((locale) => (
        <Form.Item
          key={locale}
          name={['nameI18n', locale]}
          label={`菜单名称（${LOCALE_DISPLAY[locale]}）`}
          extra={`可选；未填时在 ${LOCALE_DISPLAY[locale]} 语境下回退到中文`}
        >
          <Input placeholder={`${LOCALE_DISPLAY[locale]} 名称`} maxLength={64} />
        </Form.Item>
      ))}
    </>
  );

  return (
    <ListPageLayout
      title="菜单管理"
      extra={
        <Space>
          <span style={{ color: 'var(--color-text-secondary)' }}>显示菜单图标</span>
          <Switch
            checked={showMenuIcon}
            onChange={handleToggleMenuIcon}
            loading={showMenuIconSwitchLoading}
          />
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            {t('common.add')}
          </Button>
        </Space>
      }
    >
      <FilterBar
        filterId="menu-management-filter"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals as Record<string, string>)}
        onReset={() => setFilters({})}
      />
      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable
          tableId="menu-management-list"
          columns={columns}
          dataSource={filteredMenus}
          loading={isLoading}
          rowKey="id"
          scroll={{ x: 1000 }}
        />
      </Card>

      {/* Edit Drawer */}
      <Drawer
        title={t('common.edit')}
        open={editVisible}
        onClose={() => {
          setEditVisible(false);
          form.resetFields();
          setSelectedMenu(null);
        }}
        width={480}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button
              style={{ marginRight: 8 }}
              onClick={() => {
                setEditVisible(false);
                form.resetFields();
                setSelectedMenu(null);
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button type="primary" onClick={handleSave}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="菜单名称（中文）"
            rules={[{ required: true, message: '请输入菜单名称' }]}
          >
            <Input placeholder="请输入菜单名称" maxLength={50} />
          </Form.Item>
          {i18nFields}
          <Form.Item
            name="type"
            label="类型"
            rules={[{ required: true, message: '请选择类型' }]}
          >
            <Select options={MENU_TYPE_OPTIONS} disabled />
          </Form.Item>

          {/* 根据类型动态显示字段 - 与添加页面保持一致 */}
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.type !== cur.type}>
            {({ getFieldValue }) => {
              const type = getFieldValue('type');

              // 目录类型字段
              if (type === 'directory') {
                return (
                  <>
                    <Form.Item
                      name="sort"
                      label="显示排序"
                      rules={[{ required: true, message: '请输入显示排序' }]}
                    >
                      <InputNumber min={1} max={999} style={{ width: '100%' }} placeholder="请输入显示排序" />
                    </Form.Item>
                    <Form.Item
                      name="icon"
                      label={t('system.menu.icon')}
                      extra={t('system.menu.iconHint')}
                    >
                      <IconPicker placeholder={t('system.menu.iconPlaceholder')} />
                    </Form.Item>
                    <Form.Item
                      name="isExternal"
                      label="是否外链"
                      rules={[{ required: true, message: '请选择是否外链' }]}
                    >
                      <Radio.Group>
                        <Radio value="yes">是</Radio>
                        <Radio value="no">否</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="routePath"
                      label="路由地址"
                      rules={[{ required: true, message: '请输入路由地址' }]}
                    >
                      <Input placeholder="请输入路由地址" maxLength={200} />
                    </Form.Item>
                    <Form.Item
                      name="showStatus"
                      label="显示状态"
                      rules={[{ required: true, message: '请选择显示状态' }]}
                    >
                      <Radio.Group>
                        <Radio value="show">显示</Radio>
                        <Radio value="hide">隐藏</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="status"
                      label="菜单状态"
                      rules={[{ required: true, message: '请选择菜单状态' }]}
                    >
                      <Radio.Group>
                        <Radio value="normal">正常</Radio>
                        <Radio value="disabled">停用</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="apiPermission"
                      label="API权限"
                      rules={[{ required: true, message: '请选择API权限' }]}
                    >
                      <Radio.Group>
                        <Radio value="required">需要</Radio>
                        <Radio value="none">无需</Radio>
                      </Radio.Group>
                    </Form.Item>
                  </>
                );
              }

              // 菜单类型字段
              if (type === 'menu') {
                return (
                  <>
                    <Form.Item
                      name="sort"
                      label="显示排序"
                      rules={[{ required: true, message: '请输入显示排序' }]}
                    >
                      <InputNumber min={1} max={999} style={{ width: '100%' }} placeholder="请输入显示排序" />
                    </Form.Item>
                    <Form.Item
                      name="icon"
                      label={t('system.menu.icon')}
                      extra={t('system.menu.iconHint')}
                    >
                      <IconPicker placeholder={t('system.menu.iconPlaceholder')} />
                    </Form.Item>
                    <Form.Item
                      name="isExternal"
                      label="是否外链"
                      rules={[{ required: true, message: '请选择是否外链' }]}
                    >
                      <Radio.Group>
                        <Radio value="yes">是</Radio>
                        <Radio value="no">否</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="routePath"
                      label="路由地址"
                      rules={[{ required: true, message: '请输入路由地址' }]}
                    >
                      <Input placeholder="请输入路由地址" maxLength={200} />
                    </Form.Item>
                    <Form.Item
                      name="componentPath"
                      label="组件路径"
                    >
                      <Input placeholder="请输入组件路径" maxLength={200} />
                    </Form.Item>
                    <Form.Item
                      name="permissionKey"
                      label="权限字符"
                    >
                      <Input placeholder="请输入权限字符" maxLength={100} />
                    </Form.Item>
                    <Form.Item
                      name="routeParams"
                      label="路由参数"
                    >
                      <Input placeholder="请输入路由参数" maxLength={200} />
                    </Form.Item>
                    <Form.Item
                      name="showStatus"
                      label="显示状态"
                      rules={[{ required: true, message: '请选择显示状态' }]}
                    >
                      <Radio.Group>
                        <Radio value="show">显示</Radio>
                        <Radio value="hide">隐藏</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="status"
                      label="菜单状态"
                      rules={[{ required: true, message: '请选择菜单状态' }]}
                    >
                      <Radio.Group>
                        <Radio value="normal">正常</Radio>
                        <Radio value="disabled">停用</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="apiPermission"
                      label="API权限"
                      rules={[{ required: true, message: '请选择API权限' }]}
                    >
                      <Radio.Group>
                        <Radio value="required">需要</Radio>
                        <Radio value="none">无需</Radio>
                      </Radio.Group>
                    </Form.Item>
                  </>
                );
              }

              // 按钮类型字段
              if (type === 'button') {
                return (
                  <>
                    <Form.Item
                      name="sort"
                      label="显示排序"
                      rules={[{ required: true, message: '请输入显示排序' }]}
                    >
                      <InputNumber min={1} max={999} style={{ width: '100%' }} placeholder="请输入显示排序" />
                    </Form.Item>
                    <Form.Item
                      name="permissionKey"
                      label="权限字符"
                    >
                      <Input placeholder="请输入权限字符" maxLength={100} />
                    </Form.Item>
                    <Form.Item
                      name="status"
                      label="菜单状态"
                      rules={[{ required: true, message: '请选择菜单状态' }]}
                    >
                      <Radio.Group>
                        <Radio value="normal">正常</Radio>
                        <Radio value="disabled">停用</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="apiPermission"
                      label="API权限"
                      rules={[{ required: true, message: '请选择API权限' }]}
                    >
                      <Radio.Group>
                        <Radio value="required">需要</Radio>
                        <Radio value="none">无需</Radio>
                      </Radio.Group>
                    </Form.Item>
                  </>
                );
              }

              return null;
            }}
          </Form.Item>
        </Form>
      </Drawer>

      {/* Add Drawer */}
      <Drawer
        title={t('common.add')}
        open={addVisible}
        onClose={() => {
          setAddVisible(false);
          addForm.resetFields();
        }}
        width={480}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button
              style={{ marginRight: 8 }}
              onClick={() => {
                setAddVisible(false);
                addForm.resetFields();
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button type="primary" onClick={handleAddSave}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={addForm} layout="vertical">
          <Form.Item
            name="parentId"
            label="上级菜单"
            rules={[{ required: true, message: '请选择上级菜单' }]}
          >
            <TreeSelect
              treeData={menuTreeData}
              placeholder="请选择上级菜单"
              treeDefaultExpandAll
            />
          </Form.Item>
          <Form.Item
            name="type"
            label="菜单类型"
            rules={[{ required: true, message: '请选择菜单类型' }]}
          >
            <Radio.Group>
              <Radio value="directory">目录</Radio>
              <Radio value="menu">菜单</Radio>
              <Radio value="button">按钮</Radio>
            </Radio.Group>
          </Form.Item>

          {/* 根据类型动态显示字段 */}
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.type !== cur.type}>
            {({ getFieldValue }) => {
              const type = getFieldValue('type');

              // 目录类型字段
              if (type === 'directory') {
                return (
                  <>
                    <Form.Item
                      name="sort"
                      label="显示排序"
                      rules={[{ required: true, message: '请输入显示排序' }]}
                    >
                      <InputNumber min={1} max={999} style={{ width: '100%' }} placeholder="请输入显示排序" />
                    </Form.Item>
                    <Form.Item
                      name="name"
                      label="菜单名称（中文）"
                      rules={[{ required: true, message: '请输入菜单名称' }]}
                    >
                      <Input placeholder="请输入菜单名称" maxLength={50} />
                    </Form.Item>
                    {i18nFields}
                    <Form.Item
                      name="icon"
                      label={t('system.menu.icon')}
                      extra={t('system.menu.iconHint')}
                    >
                      <IconPicker placeholder={t('system.menu.iconPlaceholder')} />
                    </Form.Item>
                    <Form.Item
                      name="isExternal"
                      label="是否外链"
                      rules={[{ required: true, message: '请选择是否外链' }]}
                      initialValue="no"
                    >
                      <Radio.Group>
                        <Radio value="yes">是</Radio>
                        <Radio value="no">否</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="routePath"
                      label="路由地址"
                      rules={[{ required: true, message: '请输入路由地址' }]}
                    >
                      <Input placeholder="请输入路由地址" maxLength={200} />
                    </Form.Item>
                    <Form.Item
                      name="showStatus"
                      label="显示状态"
                      rules={[{ required: true, message: '请选择显示状态' }]}
                      initialValue="show"
                    >
                      <Radio.Group>
                        <Radio value="show">显示</Radio>
                        <Radio value="hide">隐藏</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="status"
                      label="菜单状态"
                      rules={[{ required: true, message: '请选择菜单状态' }]}
                      initialValue="normal"
                    >
                      <Radio.Group>
                        <Radio value="normal">正常</Radio>
                        <Radio value="disabled">停用</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="apiPermission"
                      label="API权限"
                      rules={[{ required: true, message: '请选择API权限' }]}
                      initialValue="none"
                    >
                      <Radio.Group>
                        <Radio value="required">需要</Radio>
                        <Radio value="none">无需</Radio>
                      </Radio.Group>
                    </Form.Item>
                  </>
                );
              }

              // 菜单类型字段
              if (type === 'menu') {
                return (
                  <>
                    <Form.Item
                      name="sort"
                      label="显示排序"
                      rules={[{ required: true, message: '请输入显示排序' }]}
                    >
                      <InputNumber min={1} max={999} style={{ width: '100%' }} placeholder="请输入显示排序" />
                    </Form.Item>
                    <Form.Item
                      name="name"
                      label="菜单名称（中文）"
                      rules={[{ required: true, message: '请输入菜单名称' }]}
                    >
                      <Input placeholder="请输入菜单名称" maxLength={50} />
                    </Form.Item>
                    {i18nFields}
                    <Form.Item
                      name="icon"
                      label={t('system.menu.icon')}
                      extra={t('system.menu.iconHint')}
                    >
                      <IconPicker placeholder={t('system.menu.iconPlaceholder')} />
                    </Form.Item>
                    <Form.Item
                      name="isExternal"
                      label="是否外链"
                      rules={[{ required: true, message: '请选择是否外链' }]}
                      initialValue="no"
                    >
                      <Radio.Group>
                        <Radio value="yes">是</Radio>
                        <Radio value="no">否</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="routePath"
                      label="路由地址"
                      rules={[{ required: true, message: '请输入路由地址' }]}
                    >
                      <Input placeholder="请输入路由地址" maxLength={200} />
                    </Form.Item>
                    <Form.Item
                      name="componentPath"
                      label="组件路径"
                    >
                      <Input placeholder="请输入组件路径" maxLength={200} />
                    </Form.Item>
                    <Form.Item
                      name="permissionKey"
                      label="权限字符"
                    >
                      <Input placeholder="请输入权限字符" maxLength={100} />
                    </Form.Item>
                    <Form.Item
                      name="routeParams"
                      label="路由参数"
                    >
                      <Input placeholder="请输入路由参数" maxLength={200} />
                    </Form.Item>
                    <Form.Item
                      name="showStatus"
                      label="显示状态"
                      rules={[{ required: true, message: '请选择显示状态' }]}
                      initialValue="show"
                    >
                      <Radio.Group>
                        <Radio value="show">显示</Radio>
                        <Radio value="hide">隐藏</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="status"
                      label="菜单状态"
                      rules={[{ required: true, message: '请选择菜单状态' }]}
                      initialValue="normal"
                    >
                      <Radio.Group>
                        <Radio value="normal">正常</Radio>
                        <Radio value="disabled">停用</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="apiPermission"
                      label="API权限"
                      rules={[{ required: true, message: '请选择API权限' }]}
                      initialValue="none"
                    >
                      <Radio.Group>
                        <Radio value="required">需要</Radio>
                        <Radio value="none">无需</Radio>
                      </Radio.Group>
                    </Form.Item>
                  </>
                );
              }

              // 按钮类型字段
              if (type === 'button') {
                return (
                  <>
                    <Form.Item
                      name="sort"
                      label="显示排序"
                      rules={[{ required: true, message: '请输入显示排序' }]}
                    >
                      <InputNumber min={1} max={999} style={{ width: '100%' }} placeholder="请输入显示排序" />
                    </Form.Item>
                    <Form.Item
                      name="name"
                      label="菜单名称（中文）"
                      rules={[{ required: true, message: '请输入菜单名称' }]}
                    >
                      <Input placeholder="请输入菜单名称" maxLength={50} />
                    </Form.Item>
                    {i18nFields}
                    <Form.Item
                      name="permissionKey"
                      label="权限字符"
                    >
                      <Input placeholder="请输入权限字符" maxLength={100} />
                    </Form.Item>
                    <Form.Item
                      name="status"
                      label="菜单状态"
                      rules={[{ required: true, message: '请选择菜单状态' }]}
                      initialValue="normal"
                    >
                      <Radio.Group>
                        <Radio value="normal">正常</Radio>
                        <Radio value="disabled">停用</Radio>
                      </Radio.Group>
                    </Form.Item>
                    <Form.Item
                      name="apiPermission"
                      label="API权限"
                      rules={[{ required: true, message: '请选择API权限' }]}
                      initialValue="none"
                    >
                      <Radio.Group>
                        <Radio value="required">需要</Radio>
                        <Radio value="none">无需</Radio>
                      </Radio.Group>
                    </Form.Item>
                  </>
                );
              }

              return null;
            }}
          </Form.Item>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
