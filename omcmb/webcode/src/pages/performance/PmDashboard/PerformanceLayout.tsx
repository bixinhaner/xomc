/**
 * G6-Gap-1：左右栏布局 — 左侧 240px 仪表盘列表 + 右侧仪表盘编辑器。
 *
 * 路由：`/performance?dashboard=:id`（query param 替代旧 `/performance/pm-dashboard/:id` path 风格）。
 *
 * 老路由 `/performance/pm-dashboard` / `/performance/pm-dashboard/:id` 由 `routes.tsx` 兼容重定向
 * 到本页（带 `?dashboard=` query），避免历史链接打不开。
 *
 * 左侧分组：
 *   - 系统内置（is_builtin=true，readonly）
 *   - 我的（owner_id = currentUserId）
 *   - 来自分享（其他）
 */

import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Button, Card, Empty, List, Modal, Form, Input, Popconfirm, Select, Space, Tag, Tooltip, message, theme } from 'antd';
import { PlusOutlined, AppstoreAddOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  usePmDashboardList,
  useCreatePmDashboard,
  useDeletePmDashboard,
} from '@core/hooks/api/usePmDashboard';
import { useUserStore } from '@core/store/userStore';
import { usePmDashboardStore } from '@core/store/pmDashboardStore';
import type { Dashboard, Technology } from '@core/types/pmDashboard';
import DashboardEditorPane from './DashboardEditorPane';
import { KpiCardManager } from './KpiCardManager';

const TECH_OPTIONS: { label: string; value: Technology }[] = [
  { label: 'LTE', value: 'lte' },
  { label: '5G NR', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
];

interface Group {
  key: 'builtin' | 'owned' | 'shared';
  title: string;
  color: string;
  items: Dashboard[];
}

export default function PerformanceLayout() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const dashboardId = searchParams.get('dashboard') ?? undefined;
  const { token } = theme.useToken();

  const { data: dashboards = [], isLoading } = usePmDashboardList();
  const createMut = useCreatePmDashboard();
  const deleteMut = useDeletePmDashboard();
  const currentUserId = useUserStore((s) => s.currentUser?.id);

  const [form] = Form.useForm<{ name: string; technology: Technology; description?: string }>();
  const [createOpen, setCreateOpen] = useState(false);
  const [kpiMgrOpen, setKpiMgrOpen] = useState(false);
  const [hoveredId, setHoveredId] = useState<string | null>(null);

  // G6-Gap-11 URL 复现：把 globalFilter 同步进 URL，反之亦然
  const globalFilter = usePmDashboardStore((s) => s.globalFilter);
  const setGlobalFilter = usePmDashboardStore((s) => s.setGlobalFilter);

  // URL → store（首次 + 用户 paste 链接时）
  useEffect(() => {
    const start = searchParams.get('start');
    const absStart = searchParams.get('abs_start');
    const absEnd = searchParams.get('abs_end');
    const devs = searchParams.get('devices');
    const groups = searchParams.get('groups');
    const next: Parameters<typeof setGlobalFilter>[0] = {};
    if (start) next.startOffset = start;
    if (absStart) next.absoluteStart = absStart;
    if (absEnd) next.absoluteEnd = absEnd;
    if (devs !== null) next.deviceSns = devs ? devs.split(',') : [];
    if (groups !== null) next.deviceGroupIds = groups ? groups.split(',') : [];
    if (Object.keys(next).length > 0) setGlobalFilter(next);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dashboardId]); // 仅在切 dashboard 时从 URL 重读，避免循环

  // store → URL（用户在 GlobalFilterBar 上交互时）
  useEffect(() => {
    const next = new URLSearchParams(searchParams);
    if (dashboardId) next.set('dashboard', dashboardId);
    // 时间窗：absoluteStart 优先
    if (globalFilter.absoluteStart && globalFilter.absoluteEnd) {
      next.set('abs_start', globalFilter.absoluteStart);
      next.set('abs_end', globalFilter.absoluteEnd);
      next.delete('start');
    } else if (globalFilter.startOffset) {
      next.set('start', globalFilter.startOffset);
      next.delete('abs_start');
      next.delete('abs_end');
    }
    // 设备 / 设备组：空数组不写
    if (globalFilter.deviceSns && globalFilter.deviceSns.length > 0) {
      next.set('devices', globalFilter.deviceSns.join(','));
    } else {
      next.delete('devices');
    }
    if (globalFilter.deviceGroupIds && globalFilter.deviceGroupIds.length > 0) {
      next.set('groups', globalFilter.deviceGroupIds.join(','));
    } else {
      next.delete('groups');
    }
    // 仅在 URL 真正变化时 setSearchParams，避免重渲染循环
    if (next.toString() !== searchParams.toString()) {
      setSearchParams(next, { replace: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [globalFilter, dashboardId]);
  // KPI 卡片管理弹窗用的当前制式：取选中 dashboard 的 technology；否则默认 lte
  const selectedDashboard = useMemo(
    () => dashboards.find((d) => d.id === dashboardId),
    [dashboards, dashboardId],
  );
  const kpiMgrTech: Technology = (selectedDashboard?.technology ?? 'lte') as Technology;

  const groups: Group[] = useMemo(() => {
    const builtin: Dashboard[] = [];
    const owned: Dashboard[] = [];
    const shared: Dashboard[] = [];
    dashboards.forEach((d) => {
      if (d.isBuiltin) builtin.push(d);
      else if (currentUserId && d.ownerId === currentUserId) owned.push(d);
      else shared.push(d);
    });
    return [
      { key: 'builtin', title: '系统内置', color: 'blue', items: builtin },
      { key: 'owned', title: '我的', color: 'green', items: owned },
      { key: 'shared', title: '来自分享', color: 'orange', items: shared },
    ];
  }, [dashboards, currentUserId]);

  // 默认选中：URL 没指定、或指向的仪表盘已不存在（被删除 / 失效链接）时，
  // 回退到首个 builtin（再退化为 owned 第一个），避免编辑区加载已删 dashboard 报错。
  const selectedMissing = Boolean(dashboardId) && !dashboards.some((d) => d.id === dashboardId);
  if (!isLoading && dashboards.length > 0 && (!dashboardId || selectedMissing)) {
    const first =
      groups.find((g) => g.key === 'builtin')?.items[0] ??
      groups.find((g) => g.key === 'owned')?.items[0] ??
      dashboards[0];
    if (first && first.id !== dashboardId) {
      setSearchParams({ dashboard: first.id }, { replace: true });
    }
  }

  const handleSelect = (d: Dashboard) => {
    setSearchParams({ dashboard: d.id });
  };

  const handleDelete = async (d: Dashboard) => {
    try {
      await deleteMut.mutateAsync(d.id);
      message.success('已删除');
      // 删的是当前打开的仪表盘：清掉选中，让默认选中逻辑回退到第一个
      if (d.id === dashboardId) {
        const next = new URLSearchParams(searchParams);
        next.delete('dashboard');
        setSearchParams(next, { replace: true });
      }
    } catch {
      message.error('删除失败');
    }
  };

  const handleCreate = async () => {
    const values = await form.validateFields();
    const d = await createMut.mutateAsync(values);
    message.success('创建成功');
    setCreateOpen(false);
    form.resetFields();
    navigate(`/performance?dashboard=${d.id}`);
  };

  return (
    <div style={{ display: 'flex', gap: 12, height: 'calc(100vh - 140px)' }}>
      <Card
        size="small"
        title="仪表盘"
        extra={
          <Space size={4}>
            <Tooltip title="KPI 卡片管理">
              <Button size="small" icon={<AppstoreAddOutlined />} onClick={() => setKpiMgrOpen(true)} />
            </Tooltip>
            <Tooltip title="新建仪表盘">
              <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)} />
            </Tooltip>
          </Space>
        }
        style={{ width: 240, flexShrink: 0, overflow: 'auto' }}
        styles={{ body: { padding: 8 } }}
        loading={isLoading}
      >
        {groups.map((g) =>
          g.items.length === 0 ? null : (
            <div key={g.key} style={{ marginBottom: 12 }}>
              <div style={{ fontSize: 12, color: '#888', marginBottom: 4, paddingLeft: 4 }}>
                <Tag color={g.color}>{g.title}</Tag>
                <span style={{ marginLeft: 4 }}>{g.items.length}</span>
              </div>
              <List
                size="small"
                dataSource={g.items}
                renderItem={(d) => {
                  const active = d.id === dashboardId;
                  const canDelete = g.key === 'owned'; // 仅自有仪表盘可删（内置只读、分享非 owner）
                  return (
                    <List.Item
                      onClick={() => handleSelect(d)}
                      onMouseEnter={() => setHoveredId(d.id)}
                      onMouseLeave={() => setHoveredId((prev) => (prev === d.id ? null : prev))}
                      style={{
                        cursor: 'pointer',
                        padding: '6px 8px',
                        background: active ? token.controlItemBgActive : undefined,
                        color: active ? token.colorPrimary : undefined,
                        borderRadius: 4,
                      }}
                    >
                      <Space size={4} style={{ width: '100%' }}>
                        <Tag style={{ marginRight: 0 }}>{d.technology.toUpperCase()}</Tag>
                        <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {d.name}
                        </span>
                        {canDelete && (
                          <Popconfirm
                            title="删除该仪表盘？"
                            description="删除后不可恢复（含全部 Panel 和分享关系）"
                            okText="删除"
                            okButtonProps={{ danger: true, loading: deleteMut.isPending }}
                            cancelText="取消"
                            onConfirm={() => handleDelete(d)}
                          >
                            <DeleteOutlined
                              onClick={(e) => e.stopPropagation()}
                              style={{
                                color: token.colorError,
                                opacity: hoveredId === d.id || active ? 1 : 0.3,
                              }}
                            />
                          </Popconfirm>
                        )}
                      </Space>
                    </List.Item>
                  );
                }}
              />
            </div>
          ),
        )}
        {dashboards.length === 0 && !isLoading && <Empty description="没有仪表盘" />}
      </Card>

      <div style={{ flex: 1, overflow: 'auto' }}>
        {dashboardId ? (
          <DashboardEditorPane dashboardId={dashboardId} />
        ) : (
          <Card>
            <Empty description='左侧选择仪表盘，或点"+"新建' />
          </Card>
        )}
      </div>

      <KpiCardManager
        open={kpiMgrOpen}
        technology={kpiMgrTech}
        onClose={() => setKpiMgrOpen(false)}
      />

      <Modal
        title="新建仪表盘"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={handleCreate}
        confirmLoading={createMut.isPending}
      >
        <Form form={form} layout="vertical" initialValues={{ technology: 'lte' as Technology }}>
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入仪表盘名称' }]}>
            <Input placeholder="如：基础 LTE 性能概览" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item label="制式" name="technology" rules={[{ required: true }]}>
            <Select options={TECH_OPTIONS} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
