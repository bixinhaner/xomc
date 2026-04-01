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
  Dropdown,
  Switch,
} from 'antd';
import type { MenuProps } from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  MoreOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

// 菜单类型
type MenuType = 'menu' | 'directory' | 'button';

// 菜单状态
type MenuStatus = 'normal' | 'disabled';

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
  children?: MenuItem[];
}

// 菜单类型选项
const MENU_TYPE_OPTIONS = [
  { label: '菜单', value: 'menu' },
  { label: '目录', value: 'directory' },
  { label: '按钮', value: 'button' },
];

// 菜单状态选项
const MENU_STATUS_OPTIONS = [
  { label: '正常', value: 'normal' },
  { label: '停用', value: 'disabled' },
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
      },
      {
        id: '1-3',
        name: '新增设备',
        type: 'button',
        sort: 3,
        permissionKey: 'device:add',
        componentPath: '',
        status: 'normal',
        parentId: '1',
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
      },
    ],
  },
];

// 将树形数据扁平化为表格数据
const flattenMenus = (menus: MenuItem[], level = 0): (MenuItem & { level: number })[] => {
  const result: (MenuItem & { level: number })[] = [];
  menus.forEach((menu) => {
    result.push({ ...menu, level });
    if (menu.children && menu.children.length > 0) {
      result.push(...flattenMenus(menu.children, level + 1));
    }
  });
  return result;
};

export default function MenuManagement() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [menus, setMenus] = useState<MenuItem[]>(MOCK_MENUS);
  const [editVisible, setEditVisible] = useState(false);
  const [selectedMenu, setSelectedMenu] = useState<MenuItem | null>(null);
  const [form] = Form.useForm();
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  // 扁平化后的菜单数据
  const flatMenus = useMemo(() => flattenMenus(menus), [menus]);

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

  // 表格列定义
  const columns: DataTableColumn<MenuItem & { level: number }>[] = useMemo(() => [
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'left',
      render: (_, record) => {
        const items: MenuProps['items'] = [
          {
            key: 'edit',
            label: t('common.edit'),
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
        ];
        // 按钮类型不能有子菜单，所以可以删除
        if (record.type === 'button' || !record.children || record.children.length === 0) {
          items.push({
            key: 'delete',
            label: t('common.delete'),
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => handleDelete(record),
          });
        }
        return (
          <Dropdown menu={{ items }} trigger={['click']}>
            <Button size="small" icon={<MoreOutlined />}>{t('common.more')}</Button>
          </Dropdown>
        );
      },
    },
    {
      key: 'name',
      title: '菜单名称',
      dataIndex: 'name',
      width: 200,
      render: (val, record) => {
        const indent = record.level * 24;
        return (
          <span style={{ paddingLeft: indent }}>
            {record.level > 0 && '└ '}
            {String(val)}
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
      width: 80,
      render: (val) => <span>{val}</span>,
    },
    {
      key: 'permissionKey',
      title: '权限标识',
      dataIndex: 'permissionKey',
      width: 180,
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
  ], [t, handleEdit, handleDelete]);

  return (
    <ListPageLayout title="菜单管理">
      <DataTable
        tableId="menu-management-list"
        columns={columns}
        dataSource={flatMenus}
        rowKey="id"
        scroll={{ x: 1000 }}
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={(keys) => setSelectedKeys(keys)}
      />

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
    </ListPageLayout>
  );
}
