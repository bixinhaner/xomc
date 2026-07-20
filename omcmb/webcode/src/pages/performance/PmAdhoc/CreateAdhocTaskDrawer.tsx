/**
 * G7-Gap-1：自定义聚合任务创建抽屉 — 公共组件。
 *
 * 抽自原 PmAdhoc/index.tsx 内嵌的 Drawer + Form；同时被 PmAdhoc 列表页和
 * DashboardEditorPane 的"+ 自定义聚合"入口复用。
 *
 * preset 入参支持从 panel 配置预填（G6-Gap-12 跳转场景）。
 */

import { useEffect, useMemo } from 'react';
import { useIntl } from 'react-intl';
import { Alert, Button, DatePicker, Drawer, Form, Input, Select, Switch, message } from 'antd';
import dayjs from 'dayjs';
import { useCreatePmAdhoc } from '@core/hooks/api/usePmAdhoc';
import type { AdhocMode } from '@core/types/pmAdhoc';
import { PM_QUERY_SELECTION_LIMIT } from '@/constants/pmQueryLimits';

// 纯枚举（label = value），不译，保持原样。
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

function parseCsvList(value: string): string[] {
  return Array.from(new Set(
    value
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean),
  ));
}

export function CreateAdhocTaskDrawer({ open, preset, onClose, onCreated }: Props) {
  const intl = useIntl();
  const [form] = Form.useForm<CreateForm>();
  const createMut = useCreatePmAdhoc();

  const MODE_OPTIONS = useMemo<{ label: string; value: AdhocMode }[]>(
    () => [
      { label: intl.formatMessage({ id: 'perf.adhoc.modeOneshotOpt' }), value: 'oneshot' },
      { label: intl.formatMessage({ id: 'perf.adhoc.modeContinuousOpt' }), value: 'continuous' },
    ],
    [intl],
  );

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
    const deviceSns = parseCsvList(v.deviceSns);
    const metricPaths = parseCsvList(v.metricPaths);

    if (deviceSns.length > PM_QUERY_SELECTION_LIMIT) {
      message.warning(intl.formatMessage(
        { id: 'perf.picker.deviceLimitExceeded' },
        { max: PM_QUERY_SELECTION_LIMIT, count: deviceSns.length },
      ));
      return;
    }
    if (metricPaths.length > PM_QUERY_SELECTION_LIMIT) {
      message.warning(intl.formatMessage(
        { id: 'perf.picker.metricLimitExceeded' },
        { max: PM_QUERY_SELECTION_LIMIT, count: metricPaths.length },
      ));
      return;
    }

    const task = await createMut.mutateAsync({
      name: v.name,
      mode: v.mode,
      cronExpr: v.cronExpr,
      deviceSns,
      metricPaths,
      granularities: v.granularities,
      windowStart: v.window[0].toISOString(),
      windowEnd: v.window[1].toISOString(),
      dimension: v.aggregateGroup ? 'aggregate_group' : 'device',
    });
    message.success(intl.formatMessage({ id: 'perf.adhoc.taskCreated' }));
    onClose();
    form.resetFields();
    onCreated?.(task.id);
  };

  return (
    <Drawer
      title={intl.formatMessage({ id: 'perf.adhoc.wizardTitle' })}
      size={600}
      open={open}
      onClose={onClose}
      extra={
        <Button type="primary" loading={createMut.isPending} onClick={handleCreate}>
          {intl.formatMessage({ id: 'perf.adhoc.btnCreate' })}
        </Button>
      }
    >
      <Form form={form} layout="vertical">
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message={intl.formatMessage({ id: 'perf.adhoc.byDeviceVsGroup' })}
          description={
            <ul style={{ paddingLeft: 18, margin: 0 }}>
              <li>
                <b>{intl.formatMessage({ id: 'perf.adhoc.byDeviceTitle' })}</b>
                {intl.formatMessage({ id: 'perf.adhoc.byDeviceDesc' })}
              </li>
              <li>
                <b>{intl.formatMessage({ id: 'perf.adhoc.toGroupTitle' })}</b>
                {intl.formatMessage({ id: 'perf.adhoc.toGroupDesc' })}
              </li>
            </ul>
          }
        />
        <Form.Item label={intl.formatMessage({ id: 'perf.adhoc.drawerTaskName' })} name="name" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item label={intl.formatMessage({ id: 'perf.adhoc.drawerMode' })} name="mode" rules={[{ required: true }]}>
          <Select options={MODE_OPTIONS} />
        </Form.Item>
        <Form.Item shouldUpdate={(p, c) => p.mode !== c.mode}>
          {() =>
            form.getFieldValue('mode') === 'continuous' ? (
              <Form.Item
                label={intl.formatMessage({ id: 'perf.adhoc.cronLabel' })}
                name="cronExpr"
                rules={[{ required: true, message: intl.formatMessage({ id: 'perf.adhoc.cronRequired' }) }]}
              >
                <Input placeholder={intl.formatMessage({ id: 'perf.adhoc.cronPlaceholder' })} />
              </Form.Item>
            ) : null
          }
        </Form.Item>
        <Form.Item
          label={intl.formatMessage({ id: 'perf.adhoc.deviceSnsLabel' })}
          name="deviceSns"
          rules={[{ required: true }]}
        >
          <Input placeholder={intl.formatMessage({ id: 'perf.adhoc.deviceSnsPlaceholder' })} />
        </Form.Item>
        <Form.Item
          label={intl.formatMessage({ id: 'perf.adhoc.metricPathsLabel' })}
          name="metricPaths"
          rules={[{ required: true }]}
        >
          <Input placeholder={intl.formatMessage({ id: 'perf.adhoc.metricPathsPlaceholder' })} />
        </Form.Item>
        <Form.Item
          label={intl.formatMessage({ id: 'perf.adhoc.granMultiLabel' })}
          name="granularities"
          rules={[{ required: true }]}
        >
          <Select mode="multiple" options={GRANULARITY_OPTIONS} />
        </Form.Item>
        <Form.Item label={intl.formatMessage({ id: 'perf.adhoc.windowLabel' })} name="window" rules={[{ required: true }]}>
          <DatePicker.RangePicker showTime style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item
          label={intl.formatMessage({ id: 'perf.adhoc.aggregateGroupLabel' })}
          name="aggregateGroup"
          valuePropName="checked"
          tooltip={intl.formatMessage({ id: 'perf.adhoc.aggregateGroupTooltip' })}
        >
          <Switch
            checkedChildren={intl.formatMessage({ id: 'perf.adhoc.switchToGroup' })}
            unCheckedChildren={intl.formatMessage({ id: 'perf.adhoc.switchByDevice' })}
          />
        </Form.Item>
      </Form>
    </Drawer>
  );
}
