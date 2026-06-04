/**
 * T-0189 仪表盘共用筛选区（受控 / props 驱动），挂两页签（任务仪表盘 + 设备列表）顶部。
 *
 * 控件：① 大时间段 RangePicker（showTime，默认近 7 天，allowClear=false）
 *       ② 星期多选（默认全选；0=周日..6=周六，dayjs().day() 口径）
 *       ③ 小时段多选（默认全选 0..23）
 *       ④ 周期对比 Switch
 *
 * 纯展示 + 回调，自己不持状态（state 在各 Pane 持有，便于驱动取数 + 二拉对比）。
 * 不复用旧 GlobalFilterBar（绑死旧拖拽编辑器 store，T-0190 清理）。中英文先不做，硬编码中文。
 */

import { useMemo } from 'react';
import { useIntl } from 'react-intl';
import { DatePicker, Select, Space, Switch, Typography } from 'antd';
import type { Dayjs } from 'dayjs';
import type { AdhocDimension, AdhocFilterOption } from '@core/types/pmAdhoc';
import {
  ALL_HOURS,
  ALL_WEEKDAYS,
  dimensionFilterMeta,
  parseBandNumber,
} from './dashboardFilterUtils';

const { RangePicker } = DatePicker;

// 星期 0..6 对应的语料键（dayjs().day() 口径：0=周日..6=周六）。
const WEEKDAY_MSG_IDS = [
  'perf.dashboard.weekdaySun',
  'perf.dashboard.weekdayMon',
  'perf.dashboard.weekdayTue',
  'perf.dashboard.weekdayWed',
  'perf.dashboard.weekdayThu',
  'perf.dashboard.weekdayFri',
  'perf.dashboard.weekdaySat',
];

export interface DashboardFilterValue {
  range: [Dayjs, Dayjs];
  weekdays: number[];
  hours: number[];
  compare: boolean;
}

interface Props {
  value: DashboardFilterValue;
  onChange: (next: DashboardFilterValue) => void;
  // PM-DASH-DIMFILTER：任务聚合维度 + 该维度可筛选子集选项 + 当前选中子集（受控）。
  dimension?: AdhocDimension;
  dimOptions?: AdhocFilterOption[];
  dimSelected?: string[];
  onDimChange?: (next: string[]) => void;
}

export default function DashboardFilterBar({
  value,
  onChange,
  dimension,
  dimOptions,
  dimSelected,
  onDimChange,
}: Props) {
  const intl = useIntl();
  const patch = (p: Partial<DashboardFilterValue>) => onChange({ ...value, ...p });

  const dimMeta = dimensionFilterMeta(dimension);
  // 频段维度把后端原值 label（'Band=42' 或后端给的名）可读化为「频段 42」；其余维度直接用后端 label。
  const dimSelectOptions = useMemo(
    () =>
      (dimOptions ?? []).map((o) => ({
        value: o.value,
        label:
          dimension === 'band'
            ? intl.formatMessage(
                { id: 'perf.dashboard.bandLabel' },
                { n: parseBandNumber(o.value) },
              )
            : o.label,
      })),
    [dimOptions, dimension, intl],
  );

  const weekdayOptions = useMemo(
    () =>
      WEEKDAY_MSG_IDS.map((id, value) => ({
        label: intl.formatMessage({ id }),
        value,
      })),
    [intl],
  );

  const hourOptions = useMemo(
    () =>
      ALL_HOURS.map((h) => ({
        label: intl.formatMessage(
          { id: 'perf.dashboard.hourSuffix' },
          { hour: String(h).padStart(2, '0') },
        ),
        value: h,
      })),
    [intl],
  );

  return (
    <Space wrap size="middle" align="center">
      <Space size={4} align="center">
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {intl.formatMessage({ id: 'perf.dashboard.filterTimeRange' })}
        </Typography.Text>
        <RangePicker
          showTime
          allowClear={false}
          value={value.range}
          onChange={(v) => {
            if (v && v[0] && v[1]) patch({ range: [v[0], v[1]] });
          }}
        />
      </Space>

      <Space size={4} align="center">
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {intl.formatMessage({ id: 'perf.dashboard.filterWeekday' })}
        </Typography.Text>
        <Select
          mode="multiple"
          allowClear
          maxTagCount="responsive"
          style={{ minWidth: 180 }}
          placeholder={intl.formatMessage({ id: 'perf.dashboard.allWeekdays' })}
          value={value.weekdays}
          options={weekdayOptions}
          onChange={(v) => patch({ weekdays: v.length === 0 ? [...ALL_WEEKDAYS] : v })}
        />
      </Space>

      <Space size={4} align="center">
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {intl.formatMessage({ id: 'perf.dashboard.filterHour' })}
        </Typography.Text>
        <Select
          mode="multiple"
          allowClear
          maxTagCount="responsive"
          style={{ minWidth: 200 }}
          placeholder={intl.formatMessage({ id: 'perf.dashboard.allHours' })}
          value={value.hours}
          options={hourOptions}
          onChange={(v) => patch({ hours: v.length === 0 ? [...ALL_HOURS] : v })}
        />
      </Space>

      {dimMeta && (
        <Space size={4} align="center">
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {intl.formatMessage({ id: dimMeta.titleId })}
          </Typography.Text>
          <Select
            mode="multiple"
            allowClear
            maxTagCount="responsive"
            style={{ minWidth: 200 }}
            placeholder={intl.formatMessage({ id: dimMeta.placeholderId })}
            value={dimSelected ?? []}
            options={dimSelectOptions}
            // 默认全不选 = 不过滤 = 显示全部（无回归）；选了若干项即只看这几项。
            onChange={(v) => onDimChange?.(v as string[])}
          />
        </Space>
      )}

      <Space size={4} align="center">
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {intl.formatMessage({ id: 'perf.dashboard.filterCompare' })}
        </Typography.Text>
        <Switch checked={value.compare} onChange={(c) => patch({ compare: c })} />
      </Space>
    </Space>
  );
}
