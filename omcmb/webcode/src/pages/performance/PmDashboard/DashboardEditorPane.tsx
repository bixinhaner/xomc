/**
 * G6-Gap-1：DashboardEditor 的"右侧 pane"形态。
 *
 * 与原 DashboardEditor.tsx 的区别：dashboardId 通过 props 传入（替代 useParams），
 * 便于嵌入 PerformanceLayout 的左右栏布局。
 *
 * 原 DashboardEditor.tsx 保留为向后兼容（旧路由 `/performance/pm-dashboard/:id` 仍可用），
 * 内部委托给本组件。
 */

import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Card, Modal, Segmented, Space, Spin, Tag, message } from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  SaveOutlined,
  ForkOutlined,
  ShareAltOutlined,
  ThunderboltOutlined,
  PrinterOutlined,
  FileExcelOutlined,
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
import { GlobalFilterBar } from './GlobalFilterBar';
import { CreateAdhocTaskDrawer } from '../PmAdhoc/CreateAdhocTaskDrawer';
import { exportWorkbook, printAsPDF } from '@core/utils/excelExport';

const TECH_OPTIONS = [
  { label: 'LTE', value: 'lte' as Technology },
  { label: '5G NR', value: 'nr' as Technology },
  { label: 'GSM', value: 'gsm' as Technology },
];

interface Props {
  dashboardId: string;
}

export default function DashboardEditorPane({ dashboardId }: Props) {
  const navigate = useNavigate();
  const { data, isLoading } = usePmDashboardDetail(dashboardId);
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
  const [adhocOpen, setAdhocOpen] = useState(false);

  useEffect(() => {
    if (data?.dashboard) {
      loadDashboard(data.dashboard, data.panels);
    }
    return () => {
      reset();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dashboardId, data?.dashboard.id]);

  if (isLoading || !currentDashboard) {
    return <Spin tip="加载仪表盘..." />;
  }

  const isOwner = currentUserId === currentDashboard.ownerId;
  const isBuiltin = currentDashboard.isBuiltin;
  const canEdit = isOwner && !isBuiltin;

  const handleTechChange = (tech: Technology) => {
    if (!canEdit) return;
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
        navigate(`/performance?dashboard=${d.id}`);
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
          {isBuiltin && <Tag color="blue">系统内置</Tag>}
          {!isOwner && !isBuiltin && <Tag color="orange">只读（来自分享）</Tag>}
          {unsavedChanges && <Tag color="warning">未保存</Tag>}
        </Space>
      }
      extra={
        <Space>
          <Segmented
            options={TECH_OPTIONS}
            value={currentDashboard.technology}
            onChange={(v) => handleTechChange(v as Technology)}
            disabled={!canEdit || updateMut.isPending}
          />
          {canEdit && (
            <Button
              icon={<EditOutlined />}
              type={editMode ? 'primary' : 'default'}
              onClick={toggleEditMode}
            >
              {editMode ? '退出编辑' : '编辑'}
            </Button>
          )}
          {canEdit && editMode && (
            <Button icon={<PlusOutlined />} onClick={handleAddPanel}>
              添加 Panel
            </Button>
          )}
          {canEdit && (
            <Button
              icon={<SaveOutlined />}
              onClick={handleSave}
              disabled={!unsavedChanges}
              loading={updateMut.isPending}
            >
              保存布局
            </Button>
          )}
          {/* G7-Gap-1：从仪表盘工具栏进入自定义聚合（结果可在 /performance/pm-adhoc 查看） */}
          <Button icon={<ThunderboltOutlined />} onClick={() => setAdhocOpen(true)}>
            + 自定义聚合
          </Button>
          {/* G6-Gap-10：导出仪表盘配置 Excel（panel 元数据；单 panel 数据由 PanelCard 的导出按钮承担）+ PDF 打印 */}
          <Button
            icon={<FileExcelOutlined />}
            onClick={() => {
              exportWorkbook(`dashboard_${currentDashboard.name}_${currentDashboard.id.slice(0, 8)}_config`, [
                {
                  name: 'dashboard',
                  rows: [
                    { id: currentDashboard.id, name: currentDashboard.name, technology: currentDashboard.technology, is_builtin: currentDashboard.isBuiltin ? 1 : 0, panels_count: currentPanels.length },
                  ],
                },
                {
                  name: 'panels',
                  rows: currentPanels.map((p) => ({
                    id: p.id,
                    title: p.title,
                    panel_type: p.panelType,
                    metric_paths: p.metricPaths.join(' | '),
                    granularities: (p.granularities ?? []).join(' | '),
                    dimension: p.dimension,
                    device_sns: (p.deviceSns ?? []).join(' | '),
                    device_group_ids: (p.deviceGroupIds ?? []).join(' | '),
                    compare_mode: p.compareMode ?? '',
                    adhoc_task_id: p.adhocTaskId ?? '',
                  })),
                },
              ]);
            }}
          >
            导出配置
          </Button>
          <Button icon={<PrinterOutlined />} onClick={() => printAsPDF(`dashboard_${currentDashboard.name}`)}>
            打印 / PDF
          </Button>
          <Button icon={<ForkOutlined />} onClick={handleFork}>
            派生
          </Button>
          {canEdit && (
            <Button icon={<ShareAltOutlined />} onClick={() => setShareOpen(true)}>
              分享
            </Button>
          )}
        </Space>
      }
    >
      <GlobalFilterBar />

      <PanelGrid
        panels={currentPanels}
        editMode={editMode && canEdit}
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

      <CreateAdhocTaskDrawer
        open={adhocOpen}
        onClose={() => setAdhocOpen(false)}
        onCreated={(taskId) => {
          message.success(`已创建任务 ${taskId.slice(0, 8)}…`);
          navigate(`/performance/pm-adhoc`);
        }}
      />
    </Card>
  );
}
