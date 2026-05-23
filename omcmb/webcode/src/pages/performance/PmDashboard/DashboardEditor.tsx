/**
 * T-0164-P6 / G6 PM 仪表盘编辑器主页。
 *
 * 路由：/performance/pm-dashboard/:id
 *
 * 顶部 toolbar：制式切换 lte/nr/gsm + 添加 Panel + 保存 + 编辑模式 + Fork + Share
 * 主区：PanelGrid 渲染所有 Panel；编辑模式下每个 Panel 有 设置 / 删除 按钮
 */

import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Button, Card, Modal, Segmented, Space, Spin, Tag, message } from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  SaveOutlined,
  ForkOutlined,
  ShareAltOutlined,
} from '@ant-design/icons';
import {
  usePmDashboardDetail,
  useUpdatePmDashboard,
  useDeletePmPanel,
  useForkPmDashboard,
} from '@core/hooks/api/usePmDashboard';
import { usePmDashboardStore } from '@core/store/pmDashboardStore';
import { useUserStore } from '@core/store/userStore';
import type { Panel, Technology } from '@core/types/pmDashboard';
import { PanelGrid } from './PanelGrid';
import { PanelConfigDrawer } from './PanelConfigDrawer';
import { ShareDialog } from './ShareDialog';

const TECH_OPTIONS = [
  { label: 'LTE', value: 'lte' as Technology },
  { label: '5G NR', value: 'nr' as Technology },
  { label: 'GSM', value: 'gsm' as Technology },
];

export default function DashboardEditor() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data, isLoading } = usePmDashboardDetail(id);
  const updateMut = useUpdatePmDashboard();
  const deletePanelMut = useDeletePmPanel();
  const forkMut = useForkPmDashboard();
  const currentUserId = useUserStore((s) => s.currentUser?.id);

  const { currentDashboard, currentPanels, editMode, unsavedChanges } = usePmDashboardStore();
  const loadDashboard = usePmDashboardStore((s) => s.loadDashboard);
  const toggleEditMode = usePmDashboardStore((s) => s.toggleEditMode);
  const markClean = usePmDashboardStore((s) => s.markClean);
  const removePanel = usePmDashboardStore((s) => s.removePanel);
  const reset = usePmDashboardStore((s) => s.reset);

  const [configPanel, setConfigPanel] = useState<Panel | null>(null);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('edit');
  const [shareOpen, setShareOpen] = useState(false);

  // 数据变化时同步到 store
  useEffect(() => {
    if (data?.dashboard) {
      loadDashboard(data.dashboard, data.panels);
    }
    return () => {
      reset();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, data?.dashboard.id]);

  if (isLoading || !currentDashboard) {
    return <Spin tip="加载仪表盘..." />;
  }

  const isOwner = currentUserId === currentDashboard.ownerId;

  const handleTechChange = (tech: Technology) => {
    if (!isOwner) return;
    updateMut.mutate({ id: currentDashboard.id, input: { technology: tech } });
  };

  const handleSave = async () => {
    if (!currentDashboard) return;
    await updateMut.mutateAsync({
      id: currentDashboard.id,
      input: { layout: currentDashboard.layout },
    });
    markClean();
    message.success('保存成功');
  };

  const handleAddPanel = () => {
    setConfigPanel(null);
    setDrawerMode('create');
  };

  const handleConfigPanel = (panel: Panel) => {
    setConfigPanel(panel);
    setDrawerMode('edit');
  };

  const handleDeletePanel = (panel: Panel) => {
    Modal.confirm({
      title: '确认删除 panel',
      onOk: async () => {
        await deletePanelMut.mutateAsync({ dashboardId: currentDashboard.id, panelId: panel.id });
        removePanel(panel.id);
      },
    });
  };

  const handleFork = () => {
    Modal.confirm({
      title: '派生此仪表盘',
      content: '将创建一个新的可独立编辑的副本（指向你自己）。',
      onOk: async () => {
        const newName = `${currentDashboard.name}（副本）`;
        const d = await forkMut.mutateAsync({ sourceId: currentDashboard.id, newName });
        message.success('派生成功');
        navigate(`/performance/pm-dashboard/${d.id}`);
      },
    });
  };

  return (
    <Card
      title={
        <Space>
          {currentDashboard.name}
          <Tag color={currentDashboard.technology === 'nr' ? 'geekblue' : 'green'}>
            {currentDashboard.technology.toUpperCase()}
          </Tag>
          {!isOwner && <Tag color="orange">只读（来自分享）</Tag>}
          {unsavedChanges && <Tag color="warning">未保存</Tag>}
        </Space>
      }
      extra={
        <Space>
          <Segmented
            options={TECH_OPTIONS}
            value={currentDashboard.technology}
            onChange={(v) => handleTechChange(v as Technology)}
            disabled={!isOwner || updateMut.isPending}
          />
          {isOwner && (
            <Button
              icon={<EditOutlined />}
              type={editMode ? 'primary' : 'default'}
              onClick={toggleEditMode}
            >
              {editMode ? '退出编辑' : '编辑'}
            </Button>
          )}
          {isOwner && editMode && (
            <Button icon={<PlusOutlined />} onClick={handleAddPanel}>
              添加 Panel
            </Button>
          )}
          {isOwner && (
            <Button
              icon={<SaveOutlined />}
              onClick={handleSave}
              disabled={!unsavedChanges}
              loading={updateMut.isPending}
            >
              保存布局
            </Button>
          )}
          <Button icon={<ForkOutlined />} onClick={handleFork}>
            派生
          </Button>
          {isOwner && (
            <Button icon={<ShareAltOutlined />} onClick={() => setShareOpen(true)}>
              分享
            </Button>
          )}
        </Space>
      }
    >
      <PanelGrid
        panels={currentPanels}
        editMode={editMode && isOwner}
        onConfigPanel={handleConfigPanel}
        onDeletePanel={handleDeletePanel}
      />

      <PanelConfigDrawer
        open={drawerMode === 'create' || Boolean(configPanel)}
        mode={drawerMode}
        panel={configPanel}
        dashboardId={currentDashboard.id}
        technology={currentDashboard.technology}
        onClose={() => {
          setConfigPanel(null);
          setDrawerMode('edit');
        }}
      />

      <ShareDialog open={shareOpen} dashboard={currentDashboard} onClose={() => setShareOpen(false)} />
    </Card>
  );
}
