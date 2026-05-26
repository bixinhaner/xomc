/**
 * Panel 配置抽屉 — 创建 / 编辑单 panel 的所有属性。
 *
 * Mode：
 *   - 'create' panel=null → 新建（POST）
 *   - 'edit'   panel=existing → 更新（PUT）
 */

import { useEffect, useState } from 'react';
import { Alert, Button, Drawer, Form, Input, InputNumber, Select, Space, Switch, Tag, Tooltip, message } from 'antd';
import { ThunderboltOutlined, AppstoreOutlined, MobileOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import {
  useCreatePmPanel,
  useUpdatePmPanel,
} from '@core/hooks/api/usePmDashboard';
import type {
  CompareMode,
  Dimension,
  Granularity,
  Panel,
  PanelType,
  Technology,
} from '@core/types/pmDashboard';
import type { DeviceType } from '@core/types/indicatorLibrary';
import { usePmDashboardStore } from '@core/store/pmDashboardStore';
import MetricPickerModal from '../KPIQuery/components/MetricPickerModal';
import DevicePickerModal from '../KPIQuery/components/DevicePickerModal';
import { csvToArray, arrayToCsv, techToDeviceType } from './dashboardUtils';

interface Props {
  open: boolean;
  mode: 'create' | 'edit';
  panel: Panel | null;
  dashboardId: string;
  technology: Technology;
  onClose: () => void;
}

interface FormValues {
  panelType: PanelType;
  title: string;
  metricPaths: string;          // 逗号分隔 → 入参时拆数组
  // G6-Gap-6：粒度多选，前端 PanelHeader 用 Tab 切换浏览
  granularities: Granularity[];
  dimension: Dimension;
  deviceSns?: string;           // 逗号分隔
  deviceGroupIds?: string;      // 逗号分隔
  windowOffset: string;         // 如 '-1h' '-24h' '-7d'
  compareMode?: CompareMode;
  adhocTaskId?: string;
  unit?: string;
  threshold?: number;
  // G6-Gap-5 TopN
  topnN?: number;
  topnSortDesc?: boolean;
  // G6-Gap-5 BigNumber
  bigNumberFontSize?: number;
  warningThreshold?: number;
  // G6-Gap-9: 时间轴切换（end_time = 采集时间；ingest_time = 入库时间）
  timeAxis?: 'end_time' | 'ingest_time';
  // G6-Gap-2: 是否继承全局筛选（默认 true）。false 时 panel 用自己的 timeRange / deviceSns / deviceGroupIds。
  inheritGlobal?: boolean;
}

const PANEL_TYPE_OPTIONS: { label: string; value: PanelType }[] = [
  { label: 'KPI 卡片', value: 'kpi_card' },
  { label: '曲线图', value: 'line_chart' },
  { label: '柱状图', value: 'bar_chart' },
  { label: '表格', value: 'table' },
  { label: '仪表盘', value: 'gauge' },
  // G6-Gap-5
  { label: '排行榜 (TopN)', value: 'topn' },
  { label: '数值大屏 (BigNumber)', value: 'big_number' },
];

const GRANULARITY_OPTIONS: { label: string; value: Granularity }[] = [
  { label: '15 分钟', value: '15min' },
  { label: '小时', value: 'hourly' },
  { label: '日', value: 'daily' },
  { label: '周', value: 'weekly' },
  { label: '月', value: 'monthly' },
];

const DIMENSION_OPTIONS: { label: string; value: Dimension }[] = [
  { label: '设备', value: 'device' },
  { label: '设备组', value: 'device_group' },
];

const COMPARE_OPTIONS: { label: string; value: CompareMode | '' }[] = [
  { label: '不对比', value: '' },
  { label: '与上一周期对比', value: 'previous_window' },
];

function splitCsv(s?: string): string[] {
  return csvToArray(s);
}

/**
 * 指标路径选择字段 — antd Form 子组件契约：自动接收 value/onChange。
 * value 为逗号分隔字符串（与表单 schema 兼容），picker 内部用 string[]，提交时回写 csv。
 */
function MetricPickerField({
  value,
  onChange,
  deviceType,
}: {
  value?: string;
  onChange?: (v: string) => void;
  deviceType: DeviceType;
}) {
  const [open, setOpen] = useState(false);
  const arr = csvToArray(value);
  return (
    <>
      <Space wrap>
        <Button icon={<AppstoreOutlined />} onClick={() => setOpen(true)}>
          选择指标（已选 {arr.length}）
        </Button>
        {arr.length > 0 && (
          <Tooltip title={<div style={{ whiteSpace: 'pre-wrap' }}>{arr.join('\n')}</div>}>
            <Tag>
              {arr.slice(0, 3).join(', ')}
              {arr.length > 3 ? ` 等 ${arr.length} 个` : ''}
            </Tag>
          </Tooltip>
        )}
      </Space>
      <MetricPickerModal
        open={open}
        onClose={() => setOpen(false)}
        onConfirm={(paths) => {
          onChange?.(arrayToCsv(paths));
          setOpen(false);
        }}
        initialSelected={arr}
        initialDeviceType={deviceType}
      />
    </>
  );
}

/**
 * 设备 SN 选择字段 — DevicePickerModal 已含搜索/分页/批量粘贴。
 */
function DeviceSnPickerField({
  value,
  onChange,
}: {
  value?: string;
  onChange?: (v: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const arr = csvToArray(value);
  return (
    <>
      <Space wrap>
        <Button icon={<MobileOutlined />} onClick={() => setOpen(true)}>
          选择设备（已选 {arr.length}）
        </Button>
        {arr.length > 0 && (
          <Tooltip title={<div style={{ whiteSpace: 'pre-wrap' }}>{arr.join('\n')}</div>}>
            <Tag>
              {arr.slice(0, 3).join(', ')}
              {arr.length > 3 ? ` 等 ${arr.length} 个` : ''}
            </Tag>
          </Tooltip>
        )}
      </Space>
      <DevicePickerModal
        open={open}
        onClose={() => setOpen(false)}
        onConfirm={(sns) => {
          onChange?.(arrayToCsv(sns));
          setOpen(false);
        }}
        initialSelected={arr}
      />
    </>
  );
}

// G6-Gap-12：deviceSns 超过此阈值时弹"建议改用自定义聚合任务"提示
const ADHOC_SUGGESTION_THRESHOLD = 10;

export function PanelConfigDrawer({ open, mode, panel, dashboardId, technology, onClose }: Props) {
  const [form] = Form.useForm<FormValues>();
  const createMut = useCreatePmPanel();
  const updateMut = useUpdatePmPanel();
  const addPanel = usePmDashboardStore((s) => s.addPanel);
  const navigate = useNavigate();

  useEffect(() => {
    if (!open) return;
    if (panel) {
      const offset = 'start_offset' in panel.timeRange ? (panel.timeRange.start_offset as string) : '-24h';
      form.setFieldsValue({
        panelType: panel.panelType,
        title: panel.title,
        metricPaths: panel.metricPaths.join(', '),
        granularities: panel.granularities,
        dimension: panel.dimension,
        deviceSns: (panel.deviceSns ?? []).join(', '),
        deviceGroupIds: (panel.deviceGroupIds ?? []).join(', '),
        windowOffset: offset,
        compareMode: panel.compareMode,
        adhocTaskId: panel.adhocTaskId,
        unit: panel.config?.unit as string | undefined,
        threshold: panel.config?.threshold as number | undefined,
        topnN: panel.config?.n as number | undefined,
        topnSortDesc: panel.config?.sort_desc as boolean | undefined,
        bigNumberFontSize: panel.config?.font_size as number | undefined,
        warningThreshold: panel.config?.warning_threshold as number | undefined,
        timeAxis: (panel.config?.time_axis as 'end_time' | 'ingest_time' | undefined) ?? 'end_time',
        inheritGlobal: (panel.config?.inherit_global as boolean | undefined) ?? true,
      });
    } else {
      form.setFieldsValue({
        panelType: 'kpi_card',
        title: '',
        metricPaths: '',
        granularities: ['hourly'],
        dimension: 'device',
        windowOffset: '-24h',
        inheritGlobal: true,
      });
    }
  }, [open, panel, form]);

  const handleSubmit = async () => {
    const values = await form.validateFields();
    const compareMode = values.compareMode || undefined;
    const baseInput = {
      dashboardId,
      panelType: values.panelType,
      title: values.title,
      metricPaths: splitCsv(values.metricPaths),
      granularities: values.granularities,
      dimension: values.dimension,
      deviceSns: values.dimension === 'device' ? splitCsv(values.deviceSns) : undefined,
      deviceGroupIds: values.dimension === 'device_group' ? splitCsv(values.deviceGroupIds) : undefined,
      timeRange: { start_offset: values.windowOffset },
      compareMode,
      adhocTaskId: values.adhocTaskId,
      config: {
        unit: values.unit,
        threshold: values.threshold,
        // G6-Gap-9 时间轴
        time_axis: values.timeAxis ?? 'end_time',
        // G6-Gap-2 是否继承全局筛选
        inherit_global: values.inheritGlobal ?? true,
        // G6-Gap-5 类型化 config
        ...(values.panelType === 'topn'
          ? { n: values.topnN ?? 10, sort_desc: values.topnSortDesc ?? true }
          : {}),
        ...(values.panelType === 'big_number'
          ? { font_size: values.bigNumberFontSize ?? 56, warning_threshold: values.warningThreshold }
          : {}),
      },
    };

    if (mode === 'create' || !panel) {
      const created = await createMut.mutateAsync(baseInput);
      addPanel(created);
      message.success('已添加');
    } else {
      await updateMut.mutateAsync({ ...baseInput, panelId: panel.id });
      message.success('已更新');
    }
    onClose();
  };

  return (
    <Drawer
      title={mode === 'create' ? '添加 Panel' : `编辑：${panel?.title ?? ''}`}
      width={520}
      open={open}
      onClose={onClose}
      extra={
        <Space>
          <Tag>{technology.toUpperCase()}</Tag>
          <Button onClick={onClose}>取消</Button>
          <Button type="primary" loading={createMut.isPending || updateMut.isPending} onClick={handleSubmit}>
            保存
          </Button>
        </Space>
      }
    >
      <Form form={form} layout="vertical">
        <Form.Item label="Panel 类型" name="panelType" rules={[{ required: true }]}>
          <Select options={PANEL_TYPE_OPTIONS} />
        </Form.Item>
        <Form.Item label="标题" name="title" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item
          label="指标路径"
          name="metricPaths"
          rules={[
            {
              validator: (_r, v) => (csvToArray(v).length > 0 ? Promise.resolve() : Promise.reject(new Error('至少选择一个指标'))),
            },
          ]}
          tooltip="点按钮弹出指标选择器（支持搜索 / 多选 / 分页）"
        >
          <MetricPickerField deviceType={techToDeviceType(technology)} />
        </Form.Item>
        <Form.Item
          label="粒度（多选 — 前端 Tab 切换浏览）"
          name="granularities"
          rules={[{ required: true, message: '至少一个粒度' }]}
          tooltip="可同时绑定多个粒度，PanelHeader 会显示 Tab，切换 Tab 时不重新请求 panel CRUD，只切换数据源"
        >
          <Select mode="multiple" options={GRANULARITY_OPTIONS} placeholder="hourly / daily / weekly / monthly" />
        </Form.Item>
        <Form.Item label="维度" name="dimension" rules={[{ required: true }]}>
          <Select options={DIMENSION_OPTIONS} />
        </Form.Item>
        <Form.Item shouldUpdate={(p, c) => p.dimension !== c.dimension || p.deviceSns !== c.deviceSns}>
          {() => {
            const dim = form.getFieldValue('dimension');
            if (dim === 'device') {
              const sns = splitCsv(form.getFieldValue('deviceSns'));
              const overflow = sns.length > ADHOC_SUGGESTION_THRESHOLD;
              return (
                <>
                  <Form.Item label="设备 SN" name="deviceSns" tooltip="点按钮弹出设备选择器（支持 SN/OUI/站点名搜索 / 批量粘贴）">
                    <DeviceSnPickerField />
                  </Form.Item>
                  {/* G6-Gap-12: deviceSns 超阈值时建议改用 adhoc */}
                  {overflow && (
                    <Alert
                      style={{ marginBottom: 16 }}
                      type="warning"
                      showIcon
                      message={`已输入 ${sns.length} 个设备 SN（超过 ${ADHOC_SUGGESTION_THRESHOLD} 个）`}
                      description="该数量级建议改用「自定义聚合任务」(G7) — 后端会异步并发执行 + 进度反馈 + 可持续化定时任务，避免 panel 实时拉数据时压力。"
                      action={
                        <Space direction="vertical">
                          <Button
                            size="small"
                            type="primary"
                            icon={<ThunderboltOutlined />}
                            onClick={() => {
                              // 关闭抽屉 → 跳转 adhoc 页（带预填的 SN/指标/粒度）
                              const params = new URLSearchParams({
                                preset: 'panel',
                                device_sns: sns.join(','),
                                metric_paths: form.getFieldValue('metricPaths') ?? '',
                                granularities: (form.getFieldValue('granularities') ?? []).join(','),
                              });
                              onClose();
                              navigate(`/performance/pm-adhoc?${params.toString()}`);
                            }}
                          >
                            去创建自定义聚合任务
                          </Button>
                        </Space>
                      }
                    />
                  )}
                </>
              );
            }
            return (
              <Form.Item label="设备组 ID（逗号分隔）" name="deviceGroupIds">
                <Input placeholder="uuid-1, uuid-2" />
              </Form.Item>
            );
          }}
        </Form.Item>
        <Form.Item
          label="继承全局筛选"
          name="inheritGlobal"
          valuePropName="checked"
          tooltip="开启时本 panel 使用 Dashboard 顶部的 GlobalFilterBar 时间窗 / 设备组 / 设备多选；关闭时用下面 panel 自己的设置"
          initialValue={true}
        >
          <Switch />
        </Form.Item>
        <Form.Item label="时间窗（offset）" name="windowOffset" rules={[{ required: true }]}>
          <Input placeholder="-24h" />
        </Form.Item>
        <Form.Item label="对比模式" name="compareMode">
          <Select options={COMPARE_OPTIONS} allowClear />
        </Form.Item>
        <Form.Item label="关联自定义聚合任务 ID（可选）" name="adhocTaskId">
          <Input placeholder="adhoc-task uuid" />
        </Form.Item>
        <Form.Item label="单位" name="unit">
          <Input placeholder="%, Mbps..." />
        </Form.Item>
        <Form.Item
          label="阈值（KPI 卡片低于此值红色 / 曲线 + 柱状图画阈值线）"
          name="threshold"
        >
          <InputNumber style={{ width: '100%' }} />
        </Form.Item>

        {/* G6-Gap-9: 时间轴切换 */}
        <Form.Item
          label="时间轴"
          name="timeAxis"
          tooltip="end_time 是基站采集窗口的止点；ingest_time 是 OMC 入库时刻。两者差即上报延迟。"
          initialValue="end_time"
        >
          <Select
            options={[
              { label: '采集时间 (end_time)', value: 'end_time' },
              { label: '入库时间 (ingest_time)', value: 'ingest_time' },
            ]}
          />
        </Form.Item>

        {/* G6-Gap-5: TopN 类型专用 */}
        <Form.Item shouldUpdate={(p, c) => p.panelType !== c.panelType} noStyle>
          {() =>
            form.getFieldValue('panelType') === 'topn' ? (
              <>
                <Form.Item label="TopN — N（取前 N 名）" name="topnN" initialValue={10}>
                  <InputNumber style={{ width: '100%' }} min={1} max={100} />
                </Form.Item>
                <Form.Item label="TopN — 排序" name="topnSortDesc" initialValue={true}>
                  <Select
                    options={[
                      { label: '降序（默认）', value: true },
                      { label: '升序', value: false },
                    ]}
                  />
                </Form.Item>
              </>
            ) : null
          }
        </Form.Item>

        {/* G6-Gap-5: BigNumber 类型专用 */}
        <Form.Item shouldUpdate={(p, c) => p.panelType !== c.panelType} noStyle>
          {() =>
            form.getFieldValue('panelType') === 'big_number' ? (
              <>
                <Form.Item label="字号 (px)" name="bigNumberFontSize" initialValue={56}>
                  <InputNumber style={{ width: '100%' }} min={24} max={160} />
                </Form.Item>
                <Form.Item label="警戒阈值（低于此值红色）" name="warningThreshold">
                  <InputNumber style={{ width: '100%' }} />
                </Form.Item>
              </>
            ) : null
          }
        </Form.Item>
      </Form>
    </Drawer>
  );
}
