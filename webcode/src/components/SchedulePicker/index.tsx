import React, { useState } from 'react';
import { Checkbox, Col, Radio, Row, Select, TimePicker, Typography } from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

export type ScheduleMode = 'daily' | 'weekly' | 'monthly';

export interface ScheduleValue {
  mode: ScheduleMode;
  time?: Dayjs | null;
  weekdays?: number[];
  monthDay?: number;
}

export interface SchedulePickerProps {
  value?: ScheduleValue;
  onChange?: (value: ScheduleValue) => void;
}

const WEEKDAY_OPTIONS = [
  { label: '周一', value: 1 },
  { label: '周二', value: 2 },
  { label: '周三', value: 3 },
  { label: '周四', value: 4 },
  { label: '周五', value: 5 },
  { label: '周六', value: 6 },
  { label: '周日', value: 0 },
];

const MONTH_DAY_OPTIONS = Array.from({ length: 31 }, (_, i) => ({
  label: `${i + 1} 日`,
  value: i + 1,
}));

const SchedulePicker: React.FC<SchedulePickerProps> = ({ value, onChange }) => {
  const [internal, setInternal] = useState<ScheduleValue>(
    value ?? { mode: 'daily', time: dayjs('08:00', 'HH:mm') }
  );

  const current = value ?? internal;

  const update = (patch: Partial<ScheduleValue>) => {
    const next = { ...current, ...patch };
    setInternal(next);
    onChange?.(next);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Mode selector */}
      <Radio.Group
        value={current.mode}
        onChange={(e) => update({ mode: e.target.value as ScheduleMode })}
      >
        <Radio value="daily">按天</Radio>
        <Radio value="weekly">按周</Radio>
        <Radio value="monthly">按月</Radio>
      </Radio.Group>

      {/* Daily */}
      {current.mode === 'daily' && (
        <Row align="middle" gutter={8}>
          <Col>
            <Typography.Text type="secondary" style={{ fontSize: 13 }}>
              每天执行时间：
            </Typography.Text>
          </Col>
          <Col>
            <TimePicker
              format="HH:mm"
              value={current.time}
              onChange={(t) => update({ time: t })}
              minuteStep={5}
              placeholder="选择时间"
            />
          </Col>
        </Row>
      )}

      {/* Weekly */}
      {current.mode === 'weekly' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div>
            <Typography.Text type="secondary" style={{ fontSize: 13, display: 'block', marginBottom: 6 }}>
              选择星期：
            </Typography.Text>
            <Checkbox.Group
              options={WEEKDAY_OPTIONS}
              value={current.weekdays ?? []}
              onChange={(vals) => update({ weekdays: vals as number[] })}
            />
          </div>
          <Row align="middle" gutter={8}>
            <Col>
              <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                执行时间：
              </Typography.Text>
            </Col>
            <Col>
              <TimePicker
                format="HH:mm"
                value={current.time}
                onChange={(t) => update({ time: t })}
                minuteStep={5}
                placeholder="选择时间"
              />
            </Col>
          </Row>
        </div>
      )}

      {/* Monthly */}
      {current.mode === 'monthly' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <Row align="middle" gutter={8}>
            <Col>
              <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                每月第：
              </Typography.Text>
            </Col>
            <Col>
              <Select
                style={{ width: 100 }}
                value={current.monthDay ?? 1}
                onChange={(v: number) => update({ monthDay: v })}
                options={MONTH_DAY_OPTIONS}
              />
            </Col>
            <Col>
              <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                天执行
              </Typography.Text>
            </Col>
          </Row>
          <Row align="middle" gutter={8}>
            <Col>
              <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                执行时间：
              </Typography.Text>
            </Col>
            <Col>
              <TimePicker
                format="HH:mm"
                value={current.time}
                onChange={(t) => update({ time: t })}
                minuteStep={5}
                placeholder="选择时间"
              />
            </Col>
          </Row>
        </div>
      )}
    </div>
  );
};

export default SchedulePicker;
