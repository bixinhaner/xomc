/**
 * 运维管理：PM 聚合手动触发页面。
 *
 * 表单（粒度 + 维度 + 时间范围）→ POST /pm/aggregation/recompute → 显示 job_id。
 * 后端 INSERT async_jobs 入队，worker 异步执行。
 * D3-2 决策：暂无 async_jobs LIST API，QA / 运维需 SQL 查状态。
 */

import { useState } from 'react';
import { Alert, Button, Card, DatePicker, Form, message, Result, Select, Space, Typography } from 'antd';
import { ThunderboltOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useTriggerRecompute } from '@core/hooks/api/usePmAggregation';
import ListPageLayout from '@/components/Layout/ListPageLayout';

const { Text, Paragraph } = Typography;

interface FormValues {
  granularity: 'hourly' | 'daily' | 'weekly' | 'monthly';
  dimension: 'device' | 'device_group';
  window: [dayjs.Dayjs, dayjs.Dayjs];
}

const GRANULARITY_OPTIONS = [
  { label: '小时 (hourly)', value: 'hourly' },
  { label: '日 (daily)', value: 'daily' },
  { label: '周 (weekly)', value: 'weekly' },
  { label: '月 (monthly)', value: 'monthly' },
];

const DIMENSION_OPTIONS = [
  { label: '设备 (device)', value: 'device' },
  { label: '设备组 (device_group)', value: 'device_group' },
];

export default function AggregationTrigger() {
  const [form] = Form.useForm<FormValues>();
  const trigger = useTriggerRecompute();
  const [lastJobId, setLastJobId] = useState<string | null>(null);

  const handleSubmit = async () => {
    const v = await form.validateFields();
    const resp = await trigger.mutateAsync({
      granularity: v.granularity,
      dimension: v.dimension,
      start: v.window[0].toISOString(),
      end: v.window[1].toISOString(),
    });
    const jobId = (resp.job_id ?? resp.id ?? '') as string;
    if (jobId) {
      setLastJobId(jobId);
      message.success(`聚合任务已入队，job_id: ${jobId}`);
    } else {
      message.warning('提交成功但响应未返回 job_id，请用 SQL 查 async_jobs 最新行');
    }
  };

  return (
    <ListPageLayout title="PM 聚合手动触发" subtitle="按粒度 + 维度 + 时间窗触发 recompute（后端 worker 异步执行）">
      <Card>
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message="使用场景"
          description={
            <ul style={{ paddingLeft: 18, margin: 0 }}>
              <li>历史时段补算（如某段时间 worker 故障未跑聚合）</li>
              <li>QA 验证聚合结果一致性</li>
              <li>本工具不直接跑聚合，而是入队 async_jobs；worker 进程会抢任务执行</li>
              <li>暂无状态查询页（计划独立任务），返回 job_id 后请用 SQL 查 <Text code>async_jobs</Text> 表</li>
            </ul>
          }
        />

        <Form<FormValues>
          form={form}
          layout="vertical"
          style={{ maxWidth: 600 }}
          initialValues={{
            granularity: 'hourly',
            dimension: 'device',
            window: [dayjs().subtract(1, 'day'), dayjs()],
          }}
        >
          <Form.Item label="聚合粒度" name="granularity" rules={[{ required: true }]}>
            <Select options={GRANULARITY_OPTIONS} />
          </Form.Item>
          <Form.Item label="维度" name="dimension" rules={[{ required: true }]}>
            <Select options={DIMENSION_OPTIONS} />
          </Form.Item>
          <Form.Item label="时间窗" name="window" rules={[{ required: true, message: '请选择时间窗' }]}>
            <DatePicker.RangePicker showTime style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item>
            <Button
              type="primary"
              icon={<ThunderboltOutlined />}
              loading={trigger.isPending}
              onClick={handleSubmit}
            >
              入队聚合任务
            </Button>
          </Form.Item>
        </Form>

        {lastJobId && (
          <Result
            status="success"
            title="任务已入队"
            subTitle={
              <Space orientation="vertical" size={4} style={{ textAlign: 'left' }}>
                <div>
                  <Text strong>job_id: </Text>
                  <Text copyable code>{lastJobId}</Text>
                </div>
                <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  状态查询 SQL（无 UI 暂用）：
                </Paragraph>
                <pre style={{ background: '#f6f8fa', padding: 12, borderRadius: 4, marginBottom: 0 }}>
                  {`docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
SELECT id, status, started_at, finished_at, error_msg
FROM async_jobs WHERE id='${lastJobId}'"`}
                </pre>
              </Space>
            }
          />
        )}
      </Card>
    </ListPageLayout>
  );
}
