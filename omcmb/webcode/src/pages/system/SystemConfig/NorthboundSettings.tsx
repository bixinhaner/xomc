import { useEffect, useState } from 'react';
import { Form, Input, InputNumber, Switch, Table, Button, Modal, Tag, Card, Spin, Tooltip, message } from 'antd';
import { PlusOutlined, MoreOutlined, SwapOutlined, EditOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';
import {
  useNorthboundServers,
  useSwitchActiveNorthboundServer,
  useUpdateNorthboundServer,
} from '@core/hooks/api/useNorthbound';
import { usePermission } from '@core/hooks/usePermission';
import type {
  NorthboundServer,
  NorthboundServerRole,
} from '@core/services/api/northboundApi';

// 北向服务器编辑权限 key（与 menus.permission_key 严格对齐）。
// seed/000071 注入 menu button 节点 + admin/operator role_menus 绑定；
// 后端 PUT /northbound/servers/:role 由 seed/000070 的 role_api_permissions
// 兜底 — 前端预测 + 后端兜底双层防护。
const PERM_NORTHBOUND_EDIT = 'system:config:northbound:edit';

interface NorthboundUser {
  id: string;
  userName: string;
  userPwd: string;
  userEnable: string;
  responseTime: string;
}

interface NorthboundSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// Mock 数据（用户管理仍 mock；本次范围聚焦主备服务器切换）
const mockUsers: NorthboundUser[] = [
  { id: '1', userName: 'northuser1', userPwd: '******', userEnable: '1', responseTime: '2026-01-15 10:30:00' },
  { id: '2', userName: 'northuser2', userPwd: '******', userEnable: '0', responseTime: '2026-02-20 14:15:00' },
];

// 设置行样式
const _settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};
void _settingRowStyle;

// 信息项样式
const infoItemStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
};

// 主/备服务器单卡片
interface ServerInfoBlockProps {
  server: NorthboundServer;
  onSwitch: () => void;
  onEdit: () => void;
  switching: boolean;
  canEdit: boolean;
  t: ReturnType<typeof useT>;
}

function ServerInfoBlock({ server, onSwitch, onEdit, switching, canEdit, t }: ServerInfoBlockProps) {
  const isActive = server.isActive;
  const titleKey =
    server.role === 'primary'
      ? 'system.northbound.primaryServer'
      : 'system.northbound.standbyServer';
  return (
    <Card
      size="small"
      type="inner"
      style={{
        flex: '1 1 320px',
        minWidth: 320,
        borderColor: isActive ? '#1677ff' : undefined,
        background: isActive ? '#e6f4ff' : undefined,
      }}
      styles={{ body: { padding: '12px 16px' } }}
      title={
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8, fontSize: 13, fontWeight: 600 }}>
          {t(titleKey)}
          {isActive && <Tag color="green" style={{ margin: 0 }}>{t('system.northbound.currentActive')}</Tag>}
        </span>
      }
      extra={
        <div style={{ display: 'inline-flex', gap: 8 }}>
          <Tooltip
            title={canEdit ? undefined : t('system.northbound.editNoPermission')}
          >
            {/* 无权限时按钮 disabled，hover 提示 — 与 PRD §4.3.6 "disabled 而非隐藏" 对齐 */}
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={onEdit}
              disabled={!canEdit}
            >
              {t('common.edit')}
            </Button>
          </Tooltip>
          {!isActive && (
            <Button
              type="primary"
              size="small"
              icon={<SwapOutlined />}
              loading={switching}
              onClick={onSwitch}
            >
              {t('system.northbound.switchToActive')}
            </Button>
          )}
        </div>
      }
    >
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 24 }}>
        <div style={infoItemStyle}>
          <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>{t('common.ipAddress')}：</span>
          <span style={{ fontWeight: 500 }}>{server.host}</span>
        </div>
        <div style={infoItemStyle}>
          <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>{t('common.port')}：</span>
          <span style={{ fontWeight: 500 }}>{server.port}</span>
        </div>
        <div style={infoItemStyle}>
          <span style={{ color: 'rgba(0, 0, 0, 0.65)' }}>{t('system.northbound.serviceStatus')}：</span>
          <Tag color={isActive ? 'green' : 'default'}>
            {isActive
              ? t('system.northbound.statusExecuting')
              : t('system.northbound.statusInactive')}
          </Tag>
        </div>
      </div>
    </Card>
  );
}

export default function NorthboundSettings({ form }: NorthboundSettingsProps) {
  const t = useT();
  const [users, setUsers] = useState<NorthboundUser[]>(mockUsers);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState<NorthboundUser | null>(null);
  const [userForm] = Form.useForm();

  // 主备服务器：从后端 northbound_servers 表拉取，切换走 PUT /northbound/servers/active。
  // 后端事务保证 partial unique index `is_active=true` 全表只一行；切换成功后 query
  // 自动 invalidate refetch，UI 高亮自动跟随。
  const { data: servers, isLoading: serversLoading, error: serversError } = useNorthboundServers();
  const switchMutation = useSwitchActiveNorthboundServer();

  // 主备服务器编辑（独立编辑权限：默认仅 admin / operator 角色可调，
  // viewer 调用会收到 403，由 axios 拦截器统一提示）。
  // canEditServer：前端预测，根据 menuStore.permissionKeys 的 button 节点判定。
  // - admin / operator：seed/000071 已绑定 → true
  // - viewer：未绑定 → false → 编辑按钮 disabled + Tooltip
  // 后端 PUT 仍由 RequireAPIPermission 中间件兜底（即使前端绕过）。
  const canEditServer = usePermission(PERM_NORTHBOUND_EDIT);
  const updateMutation = useUpdateNorthboundServer();
  const [editForm] = Form.useForm();
  const [editVisible, setEditVisible] = useState(false);
  const [editing, setEditing] = useState<NorthboundServer | null>(null);

  const primary = servers?.find((s) => s.role === 'primary');
  const standby = servers?.find((s) => s.role === 'standby');
  const activeRole = servers?.find((s) => s.isActive)?.role;

  // editing 切换时同步表单字段（避免上次编辑残留）
  useEffect(() => {
    if (editVisible && editing) {
      editForm.setFieldsValue({
        host: editing.host,
        port: editing.port,
        description: editing.description,
      });
    }
  }, [editVisible, editing, editForm]);

  const openEditModal = (server: NorthboundServer) => {
    setEditing(server);
    setEditVisible(true);
  };

  const handleEditSubmit = async () => {
    if (!editing) return;
    try {
      const values = await editForm.validateFields();
      await updateMutation.mutateAsync({
        role: editing.role,
        host: String(values.host).trim(),
        port: Number(values.port),
        description: values.description ?? '',
      });
      void message.success(t('common.updateSuccess'));
      setEditVisible(false);
      setEditing(null);
      editForm.resetFields();
    } catch (err: unknown) {
      // antd validateFields 会抛 errorInfo（无 message 字段），这种情况下不显示错误 toast；
      // mutation 错误（含 403 无权限）通过 message.error 显示给用户。
      if (err instanceof Error && err.message) {
        void message.error(err.message);
      }
    }
  };

  const handleSwitchActive = (target: NorthboundServerRole) => {
    if (target === activeRole) return;
    const targetLabel =
      target === 'primary'
        ? t('system.northbound.primaryServer')
        : t('system.northbound.standbyServer');
    Modal.confirm({
      title: t('common.confirm'),
      content: t('system.northbound.switchConfirm', { server: targetLabel }),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        try {
          await switchMutation.mutateAsync(target);
          void message.success(t('system.northbound.switchSuccess'));
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : 'switch failed';
          void message.error(msg);
        }
      },
    });
  };

  const handleAddUser = () => {
    setEditingUser(null);
    userForm.resetFields();
    setModalVisible(true);
  };

  const handleEditUser = (user: NorthboundUser) => {
    setEditingUser(user);
    userForm.setFieldsValue({
      userName: user.userName,
      userPwd: '',
      userEnable: user.userEnable === '1',
    });
    setModalVisible(true);
  };

  const _handleDeleteUser = (id: string) => {
    setUsers(users.filter(u => u.id !== id));
    void message.success(t('common.deleteSuccess'));
  };
  void _handleDeleteUser;

  const handleToggleEnable = (user: NorthboundUser) => {
    const newEnable = user.userEnable === '1' ? '0' : '1';
    setUsers(users.map(u => u.id === user.id ? { ...u, userEnable: newEnable } : u));
    void message.success(newEnable === '1' ? t('system.northbound.enabled') : t('system.northbound.disabled'));
  };

  const handleModalOk = () => {
    userForm.validateFields().then((values) => {
      if (editingUser) {
        setUsers(users.map(u => u.id === editingUser.id ? {
          ...u,
          ...values,
          userEnable: values.userEnable ? '1' : '0'
        } : u));
        void message.success(t('common.updateSuccess'));
      } else {
        const newUser: NorthboundUser = {
          id: Date.now().toString(),
          userName: values.userName,
          userPwd: values.userPwd || '******',
          userEnable: values.userEnable ? '1' : '0',
          responseTime: new Date().toLocaleString(),
        };
        setUsers([...users, newUser]);
        void message.success(t('common.addSuccess'));
      }
      setModalVisible(false);
    });
  };

  const columns: ColumnsType<NorthboundUser> = [
    {
      title: '',
      key: 'operation',
      width: 50,
      render: (_: unknown, record: NorthboundUser) => (
        <Button size="small" icon={<MoreOutlined />} onClick={() => handleEditUser(record)} />
      ),
    },
    {
      title: t('system.northbound.isEnabled'),
      dataIndex: 'userEnable',
      key: 'userEnable',
      width: 100,
      render: (val: string, record: NorthboundUser) => (
        <Switch
          size="small"
          checked={val === '1'}
          onChange={() => handleToggleEnable(record)}
        />
      ),
    },
    {
      title: t('system.northbound.username'),
      dataIndex: 'userName',
      key: 'userName',
    },
    {
      title: t('system.northbound.password'),
      dataIndex: 'userPwd',
      key: 'userPwd',
    },
    {
      title: t('table.createTime'),
      dataIndex: 'responseTime',
      key: 'responseTime',
    },
  ];

  return (
    <>
      <Form form={form} layout="vertical" size="small" initialValues={{
        northboundIp: '192.168.1.100',
        northboundPort: 8081,
        northboundServiceStatus: '1',
      }}>
        {/* 服务信息：主用 / 备用 双服务器，支持主备切换 */}
        <Card
          size="small"
          title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.northbound.serviceInfo')}</span>}
          style={{ marginBottom: 16 }}
        >
          {serversLoading ? (
            <div style={{ display: 'flex', justifyContent: 'center', padding: '24px 0' }}>
              <Spin />
            </div>
          ) : serversError ? (
            <div style={{ color: 'var(--color-error)' }}>
              {t('common.failed')}
            </div>
          ) : (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
              {primary && (
                <ServerInfoBlock
                  server={primary}
                  switching={switchMutation.isPending && switchMutation.variables === 'primary'}
                  onSwitch={() => handleSwitchActive('primary')}
                  onEdit={() => openEditModal(primary)}
                  canEdit={canEditServer}
                  t={t}
                />
              )}
              {standby && (
                <ServerInfoBlock
                  server={standby}
                  switching={switchMutation.isPending && switchMutation.variables === 'standby'}
                  onSwitch={() => handleSwitchActive('standby')}
                  onEdit={() => openEditModal(standby)}
                  canEdit={canEditServer}
                  t={t}
                />
              )}
            </div>
          )}
        </Card>

        {/* 用户管理 */}
        <Card
          size="small"
          title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.northbound.userManagement')}</span>}
          extra={
            <Button type="primary" icon={<PlusOutlined />} size="small" onClick={handleAddUser}>
              {t('common.add')}
            </Button>
          }
        >
          <Table
            dataSource={users}
            columns={columns}
            rowKey="id"
            pagination={false}
            size="small"
            bordered
          />
        </Card>
      </Form>

      {/* 编辑主备服务器配置弹窗（独立编辑权限：viewer 调用会被后端 403） */}
      <Modal
        title={
          editing
            ? `${t('common.edit')}: ${
                editing.role === 'primary'
                  ? t('system.northbound.primaryServer')
                  : t('system.northbound.standbyServer')
              }`
            : t('common.edit')
        }
        open={editVisible}
        onOk={handleEditSubmit}
        onCancel={() => {
          setEditVisible(false);
          setEditing(null);
          editForm.resetFields();
        }}
        confirmLoading={updateMutation.isPending}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        destroyOnHidden
      >
        <Form form={editForm} layout="vertical">
          <Form.Item
            name="host"
            label={t('common.ipAddress')}
            rules={[
              { required: true, message: t('common.pleaseInput') },
              { max: 255, message: 'host length must be ≤ 255' },
            ]}
          >
            <Input placeholder="e.g. 192.168.1.100" />
          </Form.Item>
          <Form.Item
            name="port"
            label={t('common.port')}
            rules={[{ required: true, message: t('common.pleaseInput') }]}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} placeholder="1-65535" />
          </Form.Item>
          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea rows={2} maxLength={255} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 添加/编辑用户弹窗 */}
      <Modal
        title={editingUser ? t('system.northbound.editUser') : t('system.northbound.addUser')}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <Form form={userForm} layout="vertical">
          <Form.Item name="userName" label={t('system.northbound.username')} rules={[{ required: true, message: t('system.northbound.pleaseInputUsername') }]}>
            <Input placeholder={t('system.northbound.pleaseInputUsername')} disabled={!!editingUser} />
          </Form.Item>
          <Form.Item name="userPwd" label={t('system.northbound.password')} rules={editingUser ? [] : [{ required: true, message: t('system.northbound.pleaseInputPassword') }]}>
            <Input.Password placeholder={editingUser ? t('system.northbound.leaveEmptyToKeep') : t('system.northbound.pleaseInputPassword')} />
          </Form.Item>
          <Form.Item name="userEnable" label={t('system.northbound.isEnabled')} valuePropName="checked">
            <Switch checkedChildren={t('common.enable')} unCheckedChildren={t('common.disable')} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
