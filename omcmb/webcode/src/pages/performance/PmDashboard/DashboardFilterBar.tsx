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

import { DatePicker, Select, Space, Switch, Typography } from 'antd';
import type { Dayjs } from 'dayjs';
import { ALL_HOURS, ALL_WEEKDAYS } from './dashboardFilterUtils';

const { RangePicker } = DatePicker;

const WEEKDAY_OPTIONS = [
  { label: '周日', value: 0 },
  { label: '周一', value: 1 },
  { label: '周二', value: 2 },
  { label: '周三', value: 3 },
  { label: '周四', value: 4 },
  { label: '周五', value: 5 },
  { label: '周六', value: 6 },
];

const HOUR_OPTIONS = ALL_HOURS.map((h) => ({
  label: `${String(h).padStart(2, '0')} 点`,
  value: h,
}));

export interface DashboardFilterValue {
  range: [Dayjs, Dayjs];
  weekdays: number[];
  hours: number[];
  compare: boolean;
}

interface Props {
  value: DashboardFilterValue;
  onChange: (next: DashboardFilterValue) => void;
}

export default function DashboardFilterBar({ value, onChange }: Props) {
  const patch = (p: Partial<DashboardFilterValue>) => onChange({ ...value, ...p });

  return (
    <Space wrap size="middle" align="center">
      <Space size={4} align="center">
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          时间段
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
          星期
        </Typography.Text>
        <Select
          mode="multiple"
          allowClear
          maxTagCount="responsive"
          style={{ minWidth: 180 }}
          placeholder="全部星期"
          value={value.weekdays}
          options={WEEKDAY_OPTIONS}
          onChange={(v) => patch({ weekdays: v.length === 0 ? [...ALL_WEEKDAYS] : v })}
        />
      </Space>

      <Space size={4} align="center">
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          小时段
        </Typography.Text>
        <Select
          mode="multiple"
          allowClear
          maxTagCount="responsive"
          style={{ minWidth: 200 }}
          placeholder="全部小时"
          value={value.hours}
          options={HOUR_OPTIONS}
          onChange={(v) => patch({ hours: v.length === 0 ? [...ALL_HOURS] : v })}
        />
      </Space>

      <Space size={4} align="center">
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          周期对比
        </Typography.Text>
        <Switch checked={value.compare} onChange={(c) => patch({ compare: c })} />
      </Space>
    </Space>
  );
}
