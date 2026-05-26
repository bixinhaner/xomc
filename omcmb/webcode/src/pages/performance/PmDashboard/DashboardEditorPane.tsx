/**
 * G6-Gap-1：DashboardEditor 的"右侧 pane"形态。
 *
 * 与原 DashboardEditor.tsx 的区别：dashboardId 通过 props 传入（替代 useParams），
 * 便于嵌入 PerformanceLayout 的左右栏布局。
 *
 * 原 DashboardEditor.tsx 保留为向后兼容（旧路由 `/performance/pm-dashboard/:id` 仍可用），
 * 内部委托给本组件。
 */

import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Card, Modal, Segmented, Space, Spin, Tag, message } from 'antd';
import {
  PlusOutlined,
  ForkOutlined,
  ShareAltOutlined,
  ThunderboltOutlined,
  PrinterOutlined,
  FileExcelOutlined,
  CloudSyncOutlined,
  CheckCircleOutlined,
  WarningOutlined,
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

  const { currentDashboard, currentPanels, unsavedChanges } = usePmDashboardStore();
  const loadDashboard = usePmDashboardStore((s) => s.loadDashboard);
  const markClean = usePmDashboardStore((s) => s.markClean);
  const removePanel = usePmDashboardStore((s) => s.removePanel);
  const reset = usePmDashboardStore((s) => s.reset);

  const [configPanel, setConfigPanel] = useState<Panel | null>(null);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('edit');
  const [shareOpen, setShareOpen] = useState(false);
  const [adhocOpen, setAdhocOpen] = useState(false);

  // 自动保存状态指示器（D4）：idle / saving / saved / error
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const savedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (data?.dashboard) {
      loadDashboard(data.dashboard, data.panels);
    }
    return () => {
      reset();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dashboardId, data?.dashboard.id]);

  // 自动保存（D3）：unsavedChanges 为 true 时 debounce 600ms 调 update，仅写 layout。
  // 失败不 markClean，UI 显示「保存失败」，下一次 dirty 变化触发新的 debounce 自动重试。
  const currentLayout = usePmDashboardStore((s) => s.currentDashboard?.layout);
  const dashboardOwnerId = data?.dashboard?.ownerId;
  const dashboardIsBuiltin = !!data?.dashboard?.isBuiltin;
  const dashboardIdSafe = data?.dashboard?.id;
  const canEditForAutoSave = currentUserId === dashboardOwnerId && !dashboardIsBuiltin;
  useEffect(() => {
    if (!unsavedChanges || !canEditForAutoSave || !dashboardIdSafe || !currentLayout) return;
    setSaveState('saving');
    const timer = setTimeout(() => {
      updateMut.mutate(
        { id: dashboardIdSafe, input: { layout: currentLayout } },
        {
          onSuccess: () => {
            markClean();
            setSaveState('saved');
            if (savedTimerRef.current) clearTimeout(savedTimerRef.current);
            savedTimerRef.current = setTimeout(() => setSaveState('idle'), 3000);
          },
          onError: () => {
            setSaveState('error');
          },
        },
      );
    }, 600);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [unsavedChanges, canEditForAutoSave, dashboardIdSafe, currentLayout]);

  useEffect(() => {
    return () => {
      if (savedTimerRef.current) clearTimeout(savedTimerRef.current);
    };
  }, []);

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
          {canEdit && saveState === 'saving' && (
            <Tag icon={<CloudSyncOutlined spin />} color="processing">
              自动保存中...
            </Tag>
          )}
          {canEdit && saveState === 'saved' && (
            <Tag icon={<CheckCircleOutlined />} color="success">
              已自动保存
            </Tag>
          )}
          {canEdit && saveState === 'error' && (
            <Tag icon={<WarningOutlined />} color="error">
              保存失败
            </Tag>
          )}
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
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddPanel}>
              添加 Panel
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
        canEdit={canEdit}
        isBuiltin={isBuiltin}
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
