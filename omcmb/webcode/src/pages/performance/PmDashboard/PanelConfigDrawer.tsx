/**
 * Panel 配置抽屉 — 创建 / 编辑单 panel 的所有属性。
 *
 * Mode：
 *   - 'create' panel=null → 新建（POST）
 *   - 'edit'   panel=existing → 更新（PUT）
 */

import { useEffect } from 'react';
import { Button, Drawer, Form, Input, InputNumber, Select, Space, Tag, message } from 'antd';
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
import { usePmDashboardStore } from '@core/store/pmDashboardStore';

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
  granularity: Granularity;
  dimension: Dimension;
  deviceSns?: string;           // 逗号分隔
  deviceGroupIds?: string;      // 逗号分隔
  windowOffset: string;         // 如 '-1h' '-24h' '-7d'
  compareMode?: CompareMode;
  adhocTaskId?: string;
  unit?: string;
  threshold?: number;
}

const PANEL_TYPE_OPTIONS: { label: string; value: PanelType }[] = [
  { label: 'KPI 卡片', value: 'kpi_card' },
  { label: '曲线图', value: 'line_chart' },
  { label: '柱状图', value: 'bar_chart' },
  { label: '表格', value: 'table' },
  { label: '仪表盘', value: 'gauge' },
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
  { label: '同时段对比其它设备', value: 'same_window_other_devices' },
  { label: '与上一周期对比', value: 'previous_window' },
];

function splitCsv(s?: string): string[] {
  if (!s) return [];
  return s
    .split(',')
    .map((x) => x.trim())
    .filter(Boolean);
}

export function PanelConfigDrawer({ open, mode, panel, dashboardId, technology, onClose }: Props) {
  const [form] = Form.useForm<FormValues>();
  const createMut = useCreatePmPanel();
  const updateMut = useUpdatePmPanel();
  const addPanel = usePmDashboardStore((s) => s.addPanel);

  useEffect(() => {
    if (!open) return;
    if (panel) {
      const offset = 'start_offset' in panel.timeRange ? (panel.timeRange.start_offset as string) : '-24h';
      form.setFieldsValue({
        panelType: panel.panelType,
        title: panel.title,
        metricPaths: panel.metricPaths.join(', '),
        granularity: panel.granularity,
        dimension: panel.dimension,
        deviceSns: (panel.deviceSns ?? []).join(', '),
        deviceGroupIds: (panel.deviceGroupIds ?? []).join(', '),
        windowOffset: offset,
        compareMode: panel.compareMode,
        adhocTaskId: panel.adhocTaskId,
        unit: panel.config?.unit as string | undefined,
        threshold: panel.config?.threshold as number | undefined,
      });
    } else {
      form.setFieldsValue({
        panelType: 'kpi_card',
        title: '',
        metricPaths: '',
        granularity: 'hourly',
        dimension: 'device',
        windowOffset: '-24h',
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
      granularity: values.granularity,
      dimension: values.dimension,
      deviceSns: values.dimension === 'device' ? splitCsv(values.deviceSns) : undefined,
      deviceGroupIds: values.dimension === 'device_group' ? splitCsv(values.deviceGroupIds) : undefined,
      timeRange: { start_offset: values.windowOffset },
      compareMode,
      adhocTaskId: values.adhocTaskId,
      config: { unit: values.unit, threshold: values.threshold },
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
          label="指标路径（逗号分隔）"
          name="metricPaths"
          rules={[{ required: true, message: '至少一个 metric_path' }]}
          tooltip="如 L.RRC.SuccRate, L.ERAB.SuccRate；与后端 perf_indicators_*.en_name 对齐"
        >
          <Input placeholder="L.RRC.SuccRate" />
        </Form.Item>
        <Form.Item label="粒度" name="granularity" rules={[{ required: true }]}>
          <Select options={GRANULARITY_OPTIONS} />
        </Form.Item>
        <Form.Item label="维度" name="dimension" rules={[{ required: true }]}>
          <Select options={DIMENSION_OPTIONS} />
        </Form.Item>
        <Form.Item shouldUpdate={(p, c) => p.dimension !== c.dimension}>
          {() =>
            form.getFieldValue('dimension') === 'device' ? (
              <Form.Item label="设备 SN（逗号分隔）" name="deviceSns">
                <Input placeholder="SN-001, SN-002" />
              </Form.Item>
            ) : (
              <Form.Item label="设备组 ID（逗号分隔）" name="deviceGroupIds">
                <Input placeholder="uuid-1, uuid-2" />
              </Form.Item>
            )
          }
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
        <Form.Item label="阈值（仅 KPI 卡片用）" name="threshold">
          <InputNumber style={{ width: '100%' }} />
        </Form.Item>
      </Form>
    </Drawer>
  );
}
