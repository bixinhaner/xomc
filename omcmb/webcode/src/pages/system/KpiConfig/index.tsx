/**
 * 首页 KPI 配置页（issue #213 S3，管理员）
 *
 * 系统管理下的管理员编辑页：制式 tab 由字典 `network_type` 动态生成（仅保留前端已知
 * KNOWN_TECHS = LTE/NR/GSM 之交集），每套布局互不影响。
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
  Space,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  DeleteOutlined,
  HolderOutlined,
  PlusOutlined,
  SaveOutlined,
  TableOutlined,
} from '@ant-design/icons';
import GridLayout, { type Layout } from 'react-grid-layout';
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';

import { useT } from '@/hooks/useT';
import MetricPickerModal from '@/components/MetricPickerModal';
import type { DeviceType } from '@core/types/indicatorLibrary';
import { useKPILayout, useSaveKPILayout } from '@core/hooks/api/useDashboard';
import {
  validateKpiPanels,
  formatKpiPanelViolations,
} from '@core/utils/kpiPanelValidation';
import { resolveLayout } from '@/pages/dashboard/layoutMapping';
import type { TechnologyType } from '@/pages/dashboard/kpi-config';
import { useTechnologyDictionary } from '@/components/dashboard/useTechnologyDictionary';
import {
  addPanel,
  applyGridLayout,
  removePanel,
  toGridLayout,
  toSavePanels,
  toWorkingPanels,
  updatePanelMetrics,
  updatePanelTitle,
  type GridLayoutItem,
  type WorkingPanel,
} from '@/pages/dashboard/kpiConfigLogic';

const { Text } = Typography;

/**
 * 制式 → 指标库设备类型（弹窗按制式锁定取数）。
 * LTE→ENB（4G），NR→GNB（5G），GSM→GSM（2G）。
 */
const TECH_TO_DEVICE_TYPE: Record<TechnologyType, DeviceType> = {
  lte: 'ENB',
  nr: 'GNB',
  gsm: 'GSM',
};

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
  const saveMutation = useSaveKPILayout();

  // 工作态：本地可编辑的 panels（带前端临时 id）。读到布局 / 回退默认后初始化一次。
  const [panels, setPanels] = useState<WorkingPanel[]>([]);
  const [initialized, setInitialized] = useState(false);

  // 指标表格弹窗（共享件 MetricPickerModal）：记录当前正在编辑指标的 panel id；
  // null = 弹窗关闭。弹窗按当前制式锁定（lockDeviceType），选回的是指标编号（K/C 编号）。
  const [pickerPanelId, setPickerPanelId] = useState<string | null>(null);
  // 编号 → 友好名映射：弹窗确认时累积，仅供编辑期摘要标签显示，不持久化（面板存的是编号）。
  const [metricLabels, setMetricLabels] = useState<Record<string, string>>({});

  useEffect(() => {
    if (layoutLoading || initialized) return;
    const resolved = resolveLayout(tech, remoteLayout);
    /* eslint-disable react-hooks/set-state-in-effect -- 读到布局后一次性初始化本地工作态 */
    setPanels(toWorkingPanels(resolved.panels));
    setInitialized(true);
    /* eslint-enable react-hooks/set-state-in-effect */
  }, [tech, remoteLayout, layoutLoading, initialized]);

  const gridLayout = useMemo<GridLayoutItem[]>(() => toGridLayout(panels), [panels]);

  // 正在编辑指标的 panel（弹窗以它的已选指标为初始选中）。
  const pickerPanel = pickerPanelId ? panels.find((p) => p.id === pickerPanelId) : undefined;

  const handleLayoutChange = (layout: Layout) => {
    setPanels((prev) => applyGridLayout(prev, layout as unknown as GridLayoutItem[]));
  };

  const handleAdd = () => {
    setPanels((prev) => addPanel(prev, t('dashboard.kpiConfig.newPanelTitle')));
  };

  const handleSave = () => {
    const savePanels = toSavePanels(panels);
    // 保存前校验：每张图必须「标题非空 + 至少 1 指标」，否则拦截不发 PUT，弹明确提示。
    // 默认图标题是 i18n key（非空），管理员清空后才为空串——按当前 tab 译文判定可见标题是否真空。
    const result = validateKpiPanels(
      savePanels.map((p) => ({ title: t(p.title), metrics: p.metrics })),
    );
    if (!result.valid) {
      message.error(formatKpiPanelViolations(result.violations));
      return;
    }
    saveMutation.mutate(
      { tech, panels: savePanels },
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
                  <Space orientation="vertical" style={{ width: '100%' }} size={4}>
                    <Button
                      icon={<TableOutlined />}
                      style={{ width: '100%' }}
                      onClick={() => setPickerPanelId(panel.id)}
                    >
                      {t('dashboard.kpiConfig.pickMetrics')}
                      {panel.metrics.length > 0
                        ? `（${t('dashboard.kpiConfig.metricsCount', { count: panel.metrics.length })}）`
                        : ''}
                    </Button>
                    <div style={{ maxHeight: 64, overflowY: 'auto' }}>
                      {panel.metrics.length === 0 ? (
                        <Text type="secondary">{t('dashboard.kpiConfig.noMetrics')}</Text>
                      ) : (
                        panel.metrics.map((code) => (
                          <Tag
                            key={code}
                            closable
                            onClose={() =>
                              setPanels((prev) =>
                                updatePanelMetrics(
                                  prev,
                                  panel.id,
                                  panel.metrics.filter((m) => m !== code),
                                ),
                              )
                            }
                            style={{ marginBottom: 4 }}
                          >
                            {metricLabels[code] ?? code}
                          </Tag>
                        ))
                      )}
                    </div>
                  </Space>
                </div>
              </Card>
            </div>
          ))}
        </GridLayout>
      )}

      {/* 指标表格弹窗（共享件）：按当前制式锁定取数，可搜索 / 跨页多选全库（counter+KPI）。
          确认后把选回的指标编号写进目标 panel.metrics，并累积友好名供摘要标签显示。 */}
      <MetricPickerModal
        // 按 panel id 重挂载：弹窗内部「已选」state 仅首挂载读 initialSelected，
        // 不随后续 prop 变化重置；给每个 panel 一个独立实例，避免开 B 图却带出 A 图的旧选中。
        key={pickerPanelId ?? 'none'}
        open={pickerPanelId !== null}
        onClose={() => setPickerPanelId(null)}
        onConfirm={(codes, labels) => {
          setMetricLabels((prev) => ({ ...prev, ...labels }));
          if (pickerPanelId) {
            setPanels((prev) => updatePanelMetrics(prev, pickerPanelId, codes));
          }
        }}
        initialSelected={pickerPanel?.metrics ?? []}
        initialDeviceType={TECH_TO_DEVICE_TYPE[tech]}
        lockDeviceType
      />
    </div>
  );
}

/**
 * 首页 KPI 配置页（管理员）：制式分 tab。
 */
export default function KpiConfigPage() {
  const t = useT();
  const [activeTech, setActiveTech] = useState<TechnologyType>('lte');
  const { options: techOptions } = useTechnologyDictionary();

  // 字典禁用了当前 tab → 回退首项（防止 children 取不到布局）
  useEffect(() => {
    if (techOptions.length && !techOptions.some((o) => o.value === activeTech)) {
      setActiveTech(techOptions[0].value);
    }
  }, [techOptions, activeTech]);

  const items = techOptions.map((opt) => ({
    key: opt.value,
    label: opt.label,
    // 仅在选中的 tab 挂载编辑器，避免一次性读三套布局 / 三套状态互相干扰。
    children: activeTech === opt.value ? <TechEditor tech={opt.value} /> : null,
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
