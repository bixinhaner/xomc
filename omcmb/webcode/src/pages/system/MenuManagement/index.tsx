import { useState, useMemo, useCallback } from 'react';
import {
  App,
  Button,
  Tag,
  Drawer,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  TreeSelect,
  Radio,
  Checkbox,
} from 'antd';
import {
  UpOutlined,
  DownOutlined,
  RightOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';
import styles from './index.module.css';

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

// 显示状态选项
const SHOW_STATUS_OPTIONS = [
  { label: '显示', value: 'show' },
  { label: '隐藏', value: 'hide' },
];

// 是否外链选项
const IS_EXTERNAL_OPTIONS = [
  { label: '是', value: 'yes' },
  { label: '否', value: 'no' },
];

// API权限选项
const API_PERMISSION_OPTIONS = [
  { label: '需要', value: 'required' },
  { label: '无需', value: 'none' },
];

// 默认的操作按钮（三级节点）
const DEFAULT_OPERATIONS = [
  { key: 'query', name: '查询' },
  { key: 'add', name: '添加' },
  { key: 'edit', name: '修改' },
  { key: 'delete', name: '删除' },
  { key: 'export', name: '导出' },
  { key: 'import', name: '导入' },
];

// Mock数据 - 菜单列表
const MOCK_MENUS: MenuItem[] = [
  {
    id: '1',
    name: '设备管理',
    type: 'directory',
    sort: 1,
    permissionKey: 'device',
    componentPath: '',
    status: 'normal',
    parentId: null,
    children: [
      {
        id: '1-1',
        name: '设备列表',
        type: 'menu',
        sort: 1,
        permissionKey: 'device:list',
        componentPath: '/device/list',
        status: 'normal',
        parentId: '1',
        children: DEFAULT_OPERATIONS.map((op, idx) => ({
          id: `1-1-${op.key}`,
          name: op.name,
          type: 'button' as MenuType,
          sort: idx + 1,
          permissionKey: `device:list:${op.key}`,
          componentPath: '',
          status: 'normal' as MenuStatus,
          parentId: '1-1',
        })),
      },
      {
        id: '1-2',
        name: '设备分组',
        type: 'menu',
        sort: 2,
        permissionKey: 'device:group',
        componentPath: '/device/group',
        status: 'normal',
        parentId: '1',
        children: DEFAULT_OPERATIONS.map((op, idx) => ({
          id: `1-2-${op.key}`,
          name: op.name,
          type: 'button' as MenuType,
          sort: idx + 1,
          permissionKey: `device:group:${op.key}`,
          componentPath: '',
          status: 'normal' as MenuStatus,
          parentId: '1-2',
        })),
      },
    ],
  },
  {
    id: '2',
    name: '告警管理',
    type: 'directory',
    sort: 2,
    permissionKey: 'alarm',
    componentPath: '',
    status: 'normal',
    parentId: null,
    children: [
      {
        id: '2-1',
        name: '当前告警',
        type: 'menu',
        sort: 1,
        permissionKey: 'alarm:current',
        componentPath: '/alarm/current',
        status: 'normal',
        parentId: '2',
        children: DEFAULT_OPERATIONS.map((op, idx) => ({
          id: `2-1-${op.key}`,
          name: op.name,
          type: 'button' as MenuType,
          sort: idx + 1,
          permissionKey: `alarm:current:${op.key}`,
          componentPath: '',
          status: 'normal' as MenuStatus,
          parentId: '2-1',
        })),
      },
      {
        id: '2-2',
        name: '历史告警',
        type: 'menu',
        sort: 2,
        permissionKey: 'alarm:history',
        componentPath: '/alarm/history',
        status: 'disabled',
        parentId: '2',
        children: DEFAULT_OPERATIONS.map((op, idx) => ({
          id: `2-2-${op.key}`,
          name: op.name,
          type: 'button' as MenuType,
          sort: idx + 1,
          permissionKey: `alarm:history:${op.key}`,
          componentPath: '',
          status: 'disabled' as MenuStatus,
          parentId: '2-2',
        })),
      },
    ],
  },
  {
    id: '3',
    name: '系统管理',
    type: 'directory',
    sort: 3,
    permissionKey: 'system',
    componentPath: '',
    status: 'normal',
    parentId: null,
    children: [
      {
        id: '3-1',
        name: '用户管理',
        type: 'menu',
        sort: 1,
        permissionKey: 'system:users',
        componentPath: '/system/users',
        status: 'normal',
        parentId: '3',
        children: DEFAULT_OPERATIONS.map((op, idx) => ({
          id: `3-1-${op.key}`,
          name: op.name,
          type: 'button' as MenuType,
          sort: idx + 1,
          permissionKey: `system:users:${op.key}`,
          componentPath: '',
          status: 'normal' as MenuStatus,
          parentId: '3-1',
        })),
      },
      {
        id: '3-2',
        name: '角色管理',
        type: 'menu',
        sort: 2,
        permissionKey: 'system:roles',
        componentPath: '/system/roles',
        status: 'normal',
        parentId: '3',
        children: DEFAULT_OPERATIONS.map((op, idx) => ({
          id: `3-2-${op.key}`,
          name: op.name,
          type: 'button' as MenuType,
          sort: idx + 1,
          permissionKey: `system:roles:${op.key}`,
          componentPath: '',
          status: 'normal' as MenuStatus,
          parentId: '3-2',
        })),
      },
      {
        id: '3-3',
        name: '菜单管理',
        type: 'menu',
        sort: 3,
        permissionKey: 'system:menus',
        componentPath: '/system/menus',
        status: 'normal',
        parentId: '3',
        children: DEFAULT_OPERATIONS.map((op, idx) => ({
          id: `3-3-${op.key}`,
          name: op.name,
          type: 'button' as MenuType,
          sort: idx + 1,
          permissionKey: `system:menus:${op.key}`,
          componentPath: '',
          status: 'normal' as MenuStatus,
          parentId: '3-3',
        })),
      },
    ],
  },
];

export default function MenuManagement() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [menus, setMenus] = useState<MenuItem[]>(MOCK_MENUS);
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

  // 处理删除
  const handleDelete = useCallback((menu: MenuItem) => {
    modal.confirm({
      title: t('common.confirmDelete'),
      content: `确定要删除菜单「${menu.name}」吗？${menu.children && menu.children.length > 0 ? '该菜单下的子菜单也将被删除。' : ''}`,
      onOk: () => {
        // 递归删除菜单
        const deleteFromList = (items: MenuItem[], id: string): MenuItem[] => {
          return items.filter((item) => {
            if (item.id === id) return false;
            if (item.children) {
              item.children = deleteFromList(item.children, id);
            }
            return true;
          });
        };
        setMenus((prev) => deleteFromList([...prev], menu.id));
        message.success(t('common.deleteSuccess'));
      },
    });
  }, [modal, message, t]);

  // 处理编辑
  const handleEdit = useCallback((menu: MenuItem) => {
    setSelectedMenu(menu);
    form.setFieldsValue({
      name: menu.name,
      type: menu.type,
      sort: menu.sort,
      permissionKey: menu.permissionKey,
      componentPath: menu.componentPath,
      status: menu.status,
    });
    setEditVisible(true);
  }, [form]);

  // 处理保存
  const handleSave = useCallback(() => {
    form.validateFields().then((vals) => {
      if (!selectedMenu) return;

      // 更新菜单数据
      const updateInList = (items: MenuItem[]): MenuItem[] => {
        return items.map((item) => {
          if (item.id === selectedMenu.id) {
            return {
              ...item,
              name: vals.name,
              type: vals.type,
              sort: vals.sort,
              permissionKey: vals.permissionKey,
              componentPath: vals.componentPath,
              status: vals.status,
            };
          }
          if (item.children) {
            return { ...item, children: updateInList(item.children) };
          }
          return item;
        });
      };

      setMenus((prev) => updateInList([...prev]));
      message.success(t('common.save'));
      setEditVisible(false);
      form.resetFields();
      setSelectedMenu(null);
    });
  }, [form, selectedMenu, message, t]);

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
      showIcon: true,
    });
    setAddVisible(true);
  }, [addForm]);

  // 保存新增菜单
  const handleAddSave = useCallback(() => {
    addForm.validateFields().then((vals) => {
      const newMenu: MenuItem = {
        id: `menu_${Date.now()}`,
        name: vals.name,
        type: vals.type,
        sort: vals.sort,
        permissionKey: vals.permissionKey || '',
        componentPath: vals.componentPath || '',
        status: vals.status,
        parentId: vals.parentId === '0' ? null : vals.parentId,
        // 新增字段
        icon: vals.showIcon ? 'menu-icon' : undefined,
        isExternal: vals.isExternal,
        routePath: vals.routePath,
        routeParams: vals.routeParams,
        showStatus: vals.showStatus,
        apiPermission: vals.apiPermission,
      };

      // 添加到对应的位置
      if (vals.parentId === '0') {
        // 添加到根级别
        setMenus((prev) => [...prev, newMenu]);
      } else {
        // 添加到指定父菜单下
        const addToParent = (items: MenuItem[], parentId: string): MenuItem[] => {
          return items.map((item) => {
            if (item.id === parentId) {
              return {
                ...item,
                children: [...(item.children || []), newMenu],
              };
            }
            if (item.children) {
              return { ...item, children: addToParent(item.children, parentId) };
            }
            return item;
          });
        };
        setMenus((prev) => addToParent([...prev], vals.parentId));
      }

      message.success(t('common.save'));
      setAddVisible(false);
      addForm.resetFields();
    });
  }, [addForm, message, t]);

  // 排序值+1
  const handleMoveDown = useCallback((record: MenuItem & { level: number }) => {
    const updateSort = (items: MenuItem[], id: string): MenuItem[] => {
      return items.map((item) => {
        if (item.id === id) {
          return { ...item, sort: (item.sort || 0) + 1 };
        }
        if (item.children) {
          return { ...item, children: updateSort(item.children, id) };
        }
        return item;
      });
    };

    setMenus((prev) => updateSort(prev, record.id));
  }, []);

  // 排序值-1
  const handleMoveUp = useCallback((record: MenuItem & { level: number }) => {
    const updateSort = (items: MenuItem[], id: string): MenuItem[] => {
      return items.map((item) => {
        if (item.id === id) {
          return { ...item, sort: Math.max(1, (item.sort || 1) - 1) };
        }
        if (item.children) {
          return { ...item, children: updateSort(item.children, id) };
        }
        return item;
      });
    };

    setMenus((prev) => updateSort(prev, record.id));
  }, []);

  // 获取同级菜单列表（用于判断是否可以上移/下移）
  const getSiblingIds = useCallback((targetId: string): string[] => {
    const findSiblings = (items: MenuItem[], parentId: string | null): string[] => {
      for (const item of items) {
        if (item.id === targetId) {
          // 找到了，返回同级ID列表
          if (parentId === null) {
            return items.map((i) => i.id);
          } else {
            const parent = items.find((i) => i.id === parentId);
            return parent?.children?.map((i) => i.id) || [];
          }
        }
        if (item.children) {
          const result = findSiblings(item.children, item.id);
          if (result.length > 0) return result;
        }
      }
      return [];
    };
    return findSiblings(menus, null);
  }, [menus]);

  // 表格列定义
  const columns: DataTableColumn<MenuItem & { level: number; hasChildren: boolean }>[] = useMemo(() => [
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'left',
      render: (_, record) => (
        <Space size={4}>
          <Button
            size="small"
            type="link"
            onClick={() => handleEdit(record)}
            style={{ padding: '0 4px' }}
          >
            编辑
          </Button>
          <Button
            size="small"
            type="link"
            danger
            onClick={() => handleDelete(record)}
            style={{ padding: '0 4px' }}
          >
            删除
          </Button>
        </Space>
      ),
    },
    {
      key: 'name',
      title: '菜单名称',
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
      key: 'type',
      title: '类型',
      dataIndex: 'type',
      width: 100,
      render: (val: MenuType) => {
        const typeMap: Record<MenuType, { color: string; text: string }> = {
          menu: { color: 'blue', text: '菜单' },
          directory: { color: 'green', text: '目录' },
          button: { color: 'orange', text: '按钮' },
        };
        const { color, text } = typeMap[val] || { color: 'default', text: val };
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
            value={val}
            size="small"
            className={styles.sortInput}
            style={{ width: 80 }}
            controls={{
              upIcon: <UpOutlined style={{ fontSize: 10 }} />,
              downIcon: <DownOutlined style={{ fontSize: 10 }} />,
            }}
            onStep={(value, info) => {
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
      render: (val) => <code style={{ fontSize: 12 }}>{val || '-'}</code>,
    },
    {
      key: 'componentPath',
      title: '组件路径',
      dataIndex: 'componentPath',
      width: 180,
      ellipsis: true,
      render: (val) => val || '-',
    },
    {
      key: 'status',
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (val: MenuStatus) => (
        <Tag color={val === 'normal' ? 'success' : 'error'}>
          {val === 'normal' ? '正常' : '停用'}
        </Tag>
      ),
    },
  ], [t, handleEdit, handleDelete, expandedKeys, toggleExpand, handleMoveUp, handleMoveDown, getSiblingIds]);

  return (
    <ListPageLayout
      title="菜单管理"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          {t('common.add')}
        </Button>
      }
    >
      <FilterBar
        filterId="menu-management-filter"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />
      <div className={styles.menuTable}>
        <DataTable
          tableId="menu-management-list"
          columns={columns}
          dataSource={filteredMenus}
          rowKey="id"
          scroll={{ x: 1000 }}
        />
      </div>

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
            label="菜单名称"
            rules={[{ required: true, message: '请输入菜单名称' }]}
          >
            <Input placeholder="请输入菜单名称" maxLength={50} />
          </Form.Item>
          <Form.Item
            name="type"
            label="类型"
            rules={[{ required: true, message: '请选择类型' }]}
          >
            <Select options={MENU_TYPE_OPTIONS} disabled />
          </Form.Item>
          <Form.Item
            name="sort"
            label="排序"
            rules={[{ required: true, message: '请输入排序' }]}
          >
            <InputNumber min={1} max={999} style={{ width: '100%' }} placeholder="请输入排序" />
          </Form.Item>
          <Form.Item
            name="permissionKey"
            label="权限标识"
            rules={[{ required: true, message: '请输入权限标识' }]}
          >
            <Input placeholder="请输入权限标识，如：system:menus" maxLength={100} />
          </Form.Item>
          <Form.Item
            name="componentPath"
            label="组件路径"
          >
            <Input placeholder="请输入组件路径，如：/system/menus" maxLength={200} />
          </Form.Item>
          <Form.Item
            name="status"
            label="状态"
            rules={[{ required: true, message: '请选择状态' }]}
          >
            <Select options={MENU_STATUS_OPTIONS} />
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
                      label="菜单名称"
                      rules={[{ required: true, message: '请输入菜单名称' }]}
                    >
                      <Input placeholder="请输入菜单名称" maxLength={50} />
                    </Form.Item>
                    <Form.Item
                      name="showIcon"
                      label="菜单图标"
                      valuePropName="checked"
                      initialValue={true}
                    >
                      <Checkbox>显示菜单图标</Checkbox>
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
                      label="菜单名称"
                      rules={[{ required: true, message: '请输入菜单名称' }]}
                    >
                      <Input placeholder="请输入菜单名称" maxLength={50} />
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
                      label="菜单名称"
                      rules={[{ required: true, message: '请输入菜单名称' }]}
                    >
                      <Input placeholder="请输入菜单名称" maxLength={50} />
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
