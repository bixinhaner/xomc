/**
 * 首页 KPI 配置页（issue #213 S3，管理员）
 *
 * 系统管理下的管理员编辑页：制式分 tab（LTE / NR / GSM），每套布局互不影响。
 * 主体一块拖拽网格画布（react-grid-layout）：每张图可拖动、可拉伸。
 * 「新增图」出空卡，起标题 + 从指标库勾指标（多选 = 一图多指标）；指标选择器按当前
 * 制式过滤（LTE tab 只列 4G/ENB 指标），库内不可用的置灰。「保存」把当前制式整套布局
 * 写回 S1 存盘接口（最后写入生效），存完对所有用户生效。
 *
 * 后端不动（S1 已就位）；不做实时预览、不做版本锁/冲突合并。
 */

import { useEffect, useMemo, useState } from 'react';
import {
  App,
  Button,
  Card,
  Empty,
  Input,
  Select,
  Space,
  Tabs,
  Tooltip,
  Typography,
} from 'antd';
import { DeleteOutlined, HolderOutlined, PlusOutlined, SaveOutlined } from '@ant-design/icons';
import GridLayout, { type Layout } from 'react-grid-layout';
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';

import { useT } from '@/hooks/useT';
import { useKPILayout, useKPIDefinitions, useSaveKPILayout } from '@core/hooks/api/useDashboard';
import { resolveLayout } from '@/pages/dashboard/layoutMapping';
import type { TechnologyType } from '@/pages/dashboard/kpi-config';
import { TECH_LABELS } from '@/pages/dashboard/kpi-config';
import {
  addPanel,
  applyGridLayout,
  definitionsForTech,
  removePanel,
  toGridLayout,
  toMetricOptions,
  toSavePanels,
  toWorkingPanels,
  updatePanelMetrics,
  updatePanelTitle,
  type GridLayoutItem,
  type WorkingPanel,
} from '@/pages/dashboard/kpiConfigLogic';

const { Text } = Typography;

/** 制式 tab 顺序（与首页一致）。 */
const TECHS: TechnologyType[] = ['lte', 'nr', 'gsm'];

/** 12 列网格、每行高度（px）；与首页渲染口径相近，纯编辑期视觉用。 */
const GRID_COLS = 12;
const ROW_HEIGHT = 30;
const GRID_WIDTH = 1100;

/**
 * 单个制式的编辑面板（拖拽画布 + 加图 + 保存）。
 * 每个制式 tab 独立挂载一个，状态互不影响。
 */
function TechEditor({ tech }: { tech: TechnologyType }) {
  const t = useT();
  const { message } = App.useApp();

  const { data: remoteLayout, isLoading: layoutLoading } = useKPILayout(tech);
  const { data: definitions } = useKPIDefinitions();
  const saveMutation = useSaveKPILayout();

  // 工作态：本地可编辑的 panels（带前端临时 id）。读到布局 / 回退默认后初始化一次。
  const [panels, setPanels] = useState<WorkingPanel[]>([]);
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (layoutLoading || initialized) return;
    const resolved = resolveLayout(tech, remoteLayout);
    /* eslint-disable react-hooks/set-state-in-effect -- 读到布局后一次性初始化本地工作态 */
    setPanels(toWorkingPanels(resolved.panels));
    setInitialized(true);
    /* eslint-enable react-hooks/set-state-in-effect */
  }, [tech, remoteLayout, layoutLoading, initialized]);

  // 指标选择器选项：按当前制式过滤指标库，不可用项置灰。
  const metricOptions = useMemo(
    () => toMetricOptions(definitionsForTech(definitions?.technologies, tech)),
    [definitions, tech],
  );

  const gridLayout = useMemo<GridLayoutItem[]>(() => toGridLayout(panels), [panels]);

  const handleLayoutChange = (layout: Layout) => {
    setPanels((prev) => applyGridLayout(prev, layout as unknown as GridLayoutItem[]));
  };

  const handleAdd = () => {
    setPanels((prev) => addPanel(prev, t('dashboard.kpiConfig.newPanelTitle')));
  };

  const handleSave = () => {
    saveMutation.mutate(
      { tech, panels: toSavePanels(panels) },
      {
        onSuccess: () => message.success(t('dashboard.kpiConfig.saveSuccess')),
        onError: () => message.error(t('dashboard.kpiConfig.saveFailed')),
      },
    );
  };

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<PlusOutlined />} onClick={handleAdd}>
          {t('dashboard.kpiConfig.addPanel')}
        </Button>
        <Button
          type="primary"
          icon={<SaveOutlined />}
          loading={saveMutation.isPending}
          onClick={handleSave}
        >
          {t('dashboard.kpiConfig.save')}
        </Button>
        <Text type="secondary">{t('dashboard.kpiConfig.hint')}</Text>
      </Space>

      {panels.length === 0 ? (
        <Empty description={t('dashboard.kpiConfig.emptyTip')} />
      ) : (
        <GridLayout
          layout={gridLayout as Layout}
          width={GRID_WIDTH}
          gridConfig={{ cols: GRID_COLS, rowHeight: ROW_HEIGHT }}
          dragConfig={{ handle: '.kpi-config-drag-grip' }}
          onLayoutChange={handleLayoutChange}
        >
          {panels.map((panel) => (
            <div key={panel.id}>
              <Card
                size="small"
                style={{ height: '100%', overflow: 'hidden' }}
                styles={{ body: { padding: 12 } }}
                title={
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    {/* 专用拖拽手柄：grip 图标负责拖动整张图卡；标题输入框拦截 mousedown 只供编辑，
                        故二者分离——避免「输入框填满标题栏导致无处可抓、拖不动」。 */}
                    <Tooltip title={t('dashboard.kpiConfig.dragHint')}>
                      <HolderOutlined
                        className="kpi-config-drag-grip"
                        style={{ cursor: 'move', color: '#999', flex: 'none' }}
                      />
                    </Tooltip>
                    <Input
                      size="small"
                      placeholder={t('dashboard.kpiConfig.titlePlaceholder')}
                      // 默认图的 title 是 i18n key（如 dashboard.panel.traffic），显示译文「流量」而非代号；
                      // 不编辑则 panel.title 仍是 key（存盘保持双语），一旦编辑即变成管理员输入的纯文本。
                      value={t(panel.title)}
                      onMouseDown={(e) => e.stopPropagation()}
                      onChange={(e) =>
                        setPanels((prev) => updatePanelTitle(prev, panel.id, e.target.value))
                      }
                    />
                  </div>
                }
                extra={
                  <Tooltip title={t('dashboard.kpiConfig.deletePanel')}>
                    <Button
                      type="text"
                      danger
                      size="small"
                      icon={<DeleteOutlined />}
                      onMouseDown={(e) => e.stopPropagation()}
                      onClick={() => setPanels((prev) => removePanel(prev, panel.id))}
                    />
                  </Tooltip>
                }
              >
                <div onMouseDown={(e) => e.stopPropagation()}>
                  <Select
                    mode="multiple"
                    allowClear
                    style={{ width: '100%' }}
                    placeholder={t('dashboard.kpiConfig.metricsPlaceholder')}
                    value={panel.metrics}
                    options={metricOptions}
                    maxTagCount="responsive"
                    onChange={(values: string[]) =>
                      setPanels((prev) => updatePanelMetrics(prev, panel.id, values))
                    }
                  />
                </div>
              </Card>
            </div>
          ))}
        </GridLayout>
      )}
    </div>
  );
}

/**
 * 首页 KPI 配置页（管理员）：制式分 tab。
 */
export default function KpiConfigPage() {
  const t = useT();
  const [activeTech, setActiveTech] = useState<TechnologyType>('lte');

  const items = TECHS.map((tech) => ({
    key: tech,
    label: TECH_LABELS[tech],
    // 仅在选中的 tab 挂载编辑器，避免一次性读三套布局 / 三套状态互相干扰。
    children: activeTech === tech ? <TechEditor tech={tech} /> : null,
  }));

  return (
    <Card title={t('dashboard.kpiConfig.pageTitle')}>
      <Tabs
        activeKey={activeTech}
        items={items}
        onChange={(key) => setActiveTech(key as TechnologyType)}
      />
    </Card>
  );
}
