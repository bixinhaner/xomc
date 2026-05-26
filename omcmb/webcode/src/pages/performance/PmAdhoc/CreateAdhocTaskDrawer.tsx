/**
 * G7-Gap-1：自定义聚合任务创建抽屉 — 公共组件。
 *
 * 抽自原 PmAdhoc/index.tsx 内嵌的 Drawer + Form；同时被 PmAdhoc 列表页和
 * DashboardEditorPane 的"+ 自定义聚合"入口复用。
 *
 * preset 入参支持从 panel 配置预填（G6-Gap-12 跳转场景）。
 */

import { useEffect } from 'react';
import { Alert, Button, DatePicker, Drawer, Form, Input, Select, Switch, message } from 'antd';
import dayjs from 'dayjs';
import { useCreatePmAdhoc } from '@core/hooks/api/usePmAdhoc';
import type { AdhocMode } from '@core/types/pmAdhoc';

const MODE_OPTIONS: { label: string; value: AdhocMode }[] = [
  { label: '单次执行', value: 'oneshot' },
  { label: '持续执行', value: 'continuous' },
];

const GRANULARITY_OPTIONS = ['hourly', 'daily', 'weekly', 'monthly'].map((g) => ({
  label: g,
  value: g,
}));

export interface CreateAdhocPreset {
  name?: string;
  deviceSns?: string[];
  metricPaths?: string[];
  granularities?: string[];
}

interface CreateForm {
  name: string;
  mode: AdhocMode;
  cronExpr?: string;
  deviceSns: string; // csv
  metricPaths: string; // csv
  granularities: string[];
  window: [dayjs.Dayjs, dayjs.Dayjs];
  aggregateGroup: boolean;
}

interface Props {
  open: boolean;
  preset?: CreateAdhocPreset;
  onClose: () => void;
  onCreated?: (taskId: string) => void;
}

export function CreateAdhocTaskDrawer({ open, preset, onClose, onCreated }: Props) {
  const [form] = Form.useForm<CreateForm>();
  const createMut = useCreatePmAdhoc();

  useEffect(() => {
    if (!open) return;
    form.setFieldsValue({
      name: preset?.name ?? '',
      mode: 'oneshot',
      deviceSns: (preset?.deviceSns ?? []).join(', '),
      metricPaths: (preset?.metricPaths ?? []).join(', '),
      granularities: preset?.granularities && preset.granularities.length > 0 ? preset.granularities : ['hourly'],
      window: [dayjs().subtract(1, 'day'), dayjs()],
      aggregateGroup: false,
    });
  }, [open, preset, form]);

  const handleCreate = async () => {
    const v = await form.validateFields();
    const task = await createMut.mutateAsync({
      name: v.name,
      mode: v.mode,
      cronExpr: v.cronExpr,
      deviceSns: v.deviceSns
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      metricPaths: v.metricPaths
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      granularities: v.granularities,
      windowStart: v.window[0].toISOString(),
      windowEnd: v.window[1].toISOString(),
      dimension: v.aggregateGroup ? 'aggregate_group' : 'device',
    });
    message.success('任务已创建，worker 将开始执行');
    onClose();
    form.resetFields();
    onCreated?.(task.id);
  };

  return (
    <Drawer
      title="新建自定义聚合任务"
      width={600}
      open={open}
      onClose={onClose}
      extra={
        <Button type="primary" loading={createMut.isPending} onClick={handleCreate}>
          创建
        </Button>
      }
    >
      <Form form={form} layout="vertical">
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message="按设备查询 vs 聚合到组"
          description={
            <ul style={{ paddingLeft: 18, margin: 0 }}>
              <li><b>按设备查询</b>（默认）：每设备保留一条结果（既有行为）</li>
              <li><b>聚合到组</b>：所有选中设备按时间桶聚合成一条（结果 device_sn=AGGREGATED）。仅支持 counter 类指标 (sum/avg/max/min)，KPI 类 (statis_type=pct) 暂不支持</li>
            </ul>
          }
        />
        <Form.Item label="任务名称" name="name" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item label="模式" name="mode" rules={[{ required: true }]}>
          <Select options={MODE_OPTIONS} />
        </Form.Item>
        <Form.Item shouldUpdate={(p, c) => p.mode !== c.mode}>
          {() =>
            form.getFieldValue('mode') === 'continuous' ? (
              <Form.Item
                label="Cron 表达式（5 字段：m h dom mon dow）"
                name="cronExpr"
                rules={[{ required: true, message: 'continuous 必须填 cron_expr' }]}
              >
                <Input placeholder="0 * * * *（每小时整点）" />
              </Form.Item>
            ) : null
          }
        </Form.Item>
        <Form.Item label="设备 SN（逗号分隔）" name="deviceSns" rules={[{ required: true }]}>
          <Input placeholder="BLQ-001, BLQ-002" />
        </Form.Item>
        <Form.Item label="指标路径（逗号分隔）" name="metricPaths" rules={[{ required: true }]}>
          <Input placeholder="L.RRC.SuccRate, L.ERAB.SuccRate" />
        </Form.Item>
        <Form.Item label="粒度（多选）" name="granularities" rules={[{ required: true }]}>
          <Select mode="multiple" options={GRANULARITY_OPTIONS} />
        </Form.Item>
        <Form.Item label="时间窗" name="window" rules={[{ required: true }]}>
          <DatePicker.RangePicker showTime style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item
          label="聚合到组"
          name="aggregateGroup"
          valuePropName="checked"
          tooltip="勾选：所有选中 SN 临时组成一组聚合（按时间桶 + LDN GROUP BY），结果 device_sn=AGGREGATED；不勾：每设备一条结果"
        >
          <Switch checkedChildren="聚合到组" unCheckedChildren="按设备查询" />
        </Form.Item>
      </Form>
    </Drawer>
  );
}
