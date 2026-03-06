import { useState, useEffect } from 'react';
import { Button, Tree, Card, Tag, message, Spin, Empty, Space, Badge } from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useAllRoles, useRoleById, useAllPermissions, useUpdateRole } from '@/hooks/api/useSystem';
import type { Role } from '@/types/system';
import { useT } from '@/hooks/useT';

const permissionModules: DataNode[] = [
  {
    key: 'device', title: '设备管理',
    children: [
      { key: 'device:view', title: '查看设备列表' },
      { key: 'device:create', title: '创建设备' },
      { key: 'device:update', title: '编辑设备' },
      { key: 'device:delete', title: '删除设备' },
      { key: 'device:config', title: '配置管理' },
    ],
  },
  {
    key: 'alarm', title: '告警管理',
    children: [
      { key: 'alarm:view', title: '查看告警' },
      { key: 'alarm:confirm', title: '确认告警' },
      { key: 'alarm:clear', title: '清除告警' },
      { key: 'alarm:config', title: '配置告警规则' },
    ],
  },
  {
    key: 'performance', title: '性能管理',
    children: [
      { key: 'perf:view', title: '查看性能数据' },
      { key: 'perf:export', title: '导出性能数据' },
      { key: 'perf:config', title: '配置采集策略' },
    ],
  },
  {
    key: 'software', title: '软件版本',
    children: [
      { key: 'software:view', title: '查看版本列表' },
      { key: 'software:upload', title: '上传固件' },
      { key: 'software:upgrade', title: '发起升级' },
      { key: 'software:delete', title: '删除版本' },
    ],
  },
  {
    key: 'file', title: '文件管理',
    children: [
      { key: 'file:view', title: '查看文件' },
      { key: 'file:download', title: '下载文件' },
      { key: 'file:upload', title: '上传文件' },
      { key: 'file:delete', title: '删除文件' },
    ],
  },
  {
    key: 'log', title: '日志管理',
    children: [
      { key: 'log:view', title: '查看日志' },
      { key: 'log:export', title: '导出日志' },
      { key: 'log:config', title: '配置日志策略' },
    ],
  },
  {
    key: 'system', title: '系统管理',
    children: [
      { key: 'system:user', title: '用户管理' },
      { key: 'system:role', title: '角色权限管理' },
      { key: 'system:config', title: '系统配置' },
      { key: 'system:dict', title: '数据字典' },
    ],
  },
  {
    key: 'report', title: '报表管理',
    children: [
      { key: 'report:view', title: '查看报表' },
      { key: 'report:generate', title: '生成报表' },
      { key: 'report:download', title: '下载报表' },
    ],
  },
  {
    key: 'ops', title: '运维工具',
    children: [
      { key: 'ops:template', title: '模板管理' },
      { key: 'ops:command', title: '命令执行' },
      { key: 'ops:diagnosis', title: '网络诊断' },
    ],
  },
];

const rolePermissions: Record<string, string[]> = {
  admin: [
    'device', 'device:view', 'device:create', 'device:update', 'device:delete', 'device:config',
    'alarm', 'alarm:view', 'alarm:confirm', 'alarm:clear', 'alarm:config',
    'performance', 'perf:view', 'perf:export', 'perf:config',
    'software', 'software:view', 'software:upload', 'software:upgrade', 'software:delete',
    'file', 'file:view', 'file:download', 'file:upload', 'file:delete',
    'log', 'log:view', 'log:export', 'log:config',
    'system', 'system:user', 'system:role', 'system:config', 'system:dict',
    'report', 'report:view', 'report:generate', 'report:download',
    'ops', 'ops:template', 'ops:command', 'ops:diagnosis',
  ],
  operator: [
    'device', 'device:view', 'device:update', 'device:config',
    'alarm', 'alarm:view', 'alarm:confirm', 'alarm:clear',
    'performance', 'perf:view', 'perf:export',
    'software', 'software:view', 'software:upgrade',
    'file', 'file:view', 'file:download', 'file:upload',
    'log', 'log:view',
    'report', 'report:view', 'report:generate',
    'ops', 'ops:template', 'ops:command', 'ops:diagnosis',
  ],
  viewer: [
    'device', 'device:view',
    'alarm', 'alarm:view',
    'performance', 'perf:view',
    'software', 'software:view',
    'file', 'file:view', 'file:download',
    'log', 'log:view',
    'report', 'report:view',
  ],
  auditor: [
    'device', 'device:view',
    'alarm', 'alarm:view',
    'performance', 'perf:view', 'perf:export',
    'log', 'log:view', 'log:export',
    'report', 'report:view', 'report:download',
  ],
};

export default function RolePermission() {
  const t = useT();
  const [selectedRoleId, setSelectedRoleId] = useState<string | null>(null);
  const [checkedKeys, setCheckedKeys] = useState<React.Key[]>([]);
  const [modified, setModified] = useState(false);

  const { data: rolesData, isLoading: rolesLoading } = useAllRoles();
  const { data: roleDetail } = useRoleById(selectedRoleId ?? '');
  const updateRole = useUpdateRole();

  useEffect(() => {
    if (selectedRoleId && rolesData) {
      const role = rolesData.find((r) => r.id === selectedRoleId);
      if (role) {
        const perms = rolePermissions[role.roleName.toLowerCase()] ?? rolePermissions.viewer;
        setCheckedKeys(perms);
        setModified(false);
      }
    }
  }, [selectedRoleId, rolesData]);

  const handleSave = () => {
    if (!selectedRoleId || !roleDetail) return;
    updateRole.mutate(
      { id: selectedRoleId, data: { permissions: checkedKeys.map(String) } },
      {
        onSuccess: () => {
          void message.success(t('common.save'));
          setModified(false);
        },
      },
    );
  };

  const roleColorMap: Record<string, string> = { admin: 'red', operator: 'blue', viewer: 'default', auditor: 'purple' };

  return (
    <ListPageLayout title={t('nav.system.roles')} subtitle={t('nav.system.roles')}>
      <div style={{ display: 'flex', gap: 16, height: 'calc(100vh - 200px)', minHeight: 500 }}>
        <div style={{ width: 220, flexShrink: 0 }}>
          <Card title={t('user.role')} size="small" style={{ height: '100%', overflow: 'auto' }}>
            {rolesLoading ? (
              <Spin />
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                {(rolesData ?? []).map((role: Role) => {
                  const roleName = role.roleName.toLowerCase();
                  const isSelected = role.id === selectedRoleId;
                  return (
                    <div
                      key={role.id}
                      onClick={() => setSelectedRoleId(role.id)}
                      style={{
                        padding: '8px 12px',
                        borderRadius: 6,
                        cursor: 'pointer',
                        background: isSelected ? '#e6f4ff' : 'transparent',
                        border: isSelected ? '1px solid #91caff' : '1px solid transparent',
                        transition: 'all 150ms ease',
                      }}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <span style={{ fontWeight: isSelected ? 600 : 400 }}>{role.roleName}</span>
                        <Tag color={roleColorMap[roleName] ?? 'default'} style={{ fontSize: 11 }}>
                          {role.userCount}
                        </Tag>
                      </div>
                      <div style={{ fontSize: 12, color: '#999', marginTop: 2 }}>{role.description}</div>
                    </div>
                  );
                })}
              </div>
            )}
          </Card>
        </div>

        <div style={{ flex: 1, minWidth: 0 }}>
          <Card
            title={
              <Space>
                <span>{t('nav.system.roles')}</span>
                {selectedRoleId && rolesData && (
                  <Badge
                    count={checkedKeys.filter((k) => !String(k).includes(':')).length + checkedKeys.filter((k) => String(k).includes(':')).length}
                    style={{ background: 'var(--color-primary-600)' }}
                    overflowCount={999}
                  />
                )}
                {modified && <Tag color="orange">{t('common.save')}</Tag>}
              </Space>
            }
            extra={
              <Button
                type="primary"
                icon={<SaveOutlined />}
                disabled={!selectedRoleId || !modified}
                loading={updateRole.isPending}
                onClick={handleSave}
              >
                {t('common.save')}
              </Button>
            }
            size="small"
            style={{ height: '100%', overflow: 'auto' }}
          >
            {!selectedRoleId ? (
              <Empty description={t('common.pleaseSelect')} />
            ) : (
              <Tree
                checkable
                checkedKeys={checkedKeys}
                treeData={permissionModules}
                defaultExpandAll
                onCheck={(checked) => {
                  setCheckedKeys(checked as React.Key[]);
                  setModified(true);
                }}
              />
            )}
          </Card>
        </div>
      </div>
    </ListPageLayout>
  );
}
