import { useEffect, useMemo, useState } from 'react';
import {
  Drawer,
  Form,
  Input,
  Modal,
  Select,
  DatePicker,
  Checkbox,
  Button,
  Space,
  Divider,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd';
import { PlusOutlined } from '@ant-design/icons';
const { Text } = Typography;
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { useT } from '@/hooks/useT';
import { useCreateMRTask, useMRTasks } from '@core/hooks/api/useMrTasks';
import { useDeviceList } from '@core/hooks/api/useDevices';
import { useUserStore } from '@core/store/userStore';
import { useAppStore } from '@core/store/appStore';
import type { CreateMRTaskRequest } from '@core/types/mrTask';

/**
 * 新建 MR 任务抽屉。
 *
 * 简化版（2026-05-25）：
 *   - 移除"任务信息"段（creator / operator / note 都由后端自动处理）
 *   - 移除"目标小站"段（后端 scheduler 开启时从 mr_device_mappings 动态枚举
 *     所有 enabled=true 的 cell，无需用户在此选择）
 *
 * 表单只剩 MR 参数：task_name + measureType + period + start/end_time。
 */
interface CreateDrawerProps {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
}

interface FormShape {
  taskName: string;
  measureType: string[]; // 多选，强制含 MRS/MRE/MRO
  statisPeriod: string;
  reportPeriod: string;
  startTime: Dayjs;
  endTime?: Dayjs;
  unlimited: boolean;
}

const REQUIRED_MEASURE_TYPES = ['MRS', 'MRE', 'MRO'];

/**
 * buildDefaultMRTaskName 生成 MR 任务的默认名，格式与 UFTE buildDefaultUfteTaskName 同款：
 *   <prefix>_<username>_<YYYY-MM-DD HH:mm:ss>
 * 例：MR_admin_2026-05-26 14:35:12 / MR_user_2026-05-26 14:35:12
 */
function buildDefaultMRTaskName(username: string | undefined, locale: string): string {
  const useZh = locale.startsWith('zh');
  const prefix = useZh ? 'MR测量' : 'MR';
  const user = (username && username.trim()) || (useZh ? '用户' : 'user');
  const nowText = dayjs().format('YYYY-MM-DD HH:mm:ss');
  return `${prefix}_${user}_${nowText}`;
}

export default function CreateDrawer({ open, onClose, onCreated }: CreateDrawerProps) {
  const t = useT();
  const [form] = Form.useForm<FormShape>();
  const createMutation = useCreateMRTask();
  // 默认任务名自动填 — 与 UFTE 一致风格：抽屉打开时按 username + 时间戳生成
  const currentUser = useUserStore((s) => s.currentUser);
  const appLocale = useAppStore((s) => s.locale);

  // 时间选择步长跟 Report Period 联动（spec §3.4）：
  //   15min → 15 分钟步长，30min → 30 分钟，60min → 60 分钟（仅 :00）
  // disabledTime 兜底拦截手动输入的非整步分钟 / 非 0 秒。
  const reportPeriodValue = Form.useWatch('reportPeriod', form);
  const minuteStep = useMemo(() => {
    const n = parseInt(reportPeriodValue ?? '15', 10);
    return n === 30 || n === 60 ? n : 15;
  }, [reportPeriodValue]);
  const disabledTime = useMemo(
    () => () => ({
      disabledMinutes: () =>
        Array.from({ length: 60 }, (_, i) => i).filter((m) => m % minuteStep !== 0),
      disabledSeconds: () =>
        Array.from({ length: 60 }, (_, i) => i).filter((s) => s !== 0),
    }),
    [minuteStep],
  );

  // 目标设备选择：与 UFTE 升级任务同款 UX —— 搜索框 + 多选 Table + 已选计数 + 批量 SN 输入。
  const [deviceKeywordInput, setDeviceKeywordInput] = useState('');
  const [deviceKeyword, setDeviceKeyword] = useState('');
  // 仅查 4G 设备（networkType='lte'）— MR 支持的 4 个平台 BLQ/MLQ/MLN/BM 全是 4G，
  // 5G 设备（BaiBNQ 等）不在支持范围；列表层先过滤掉，避免用户白选。
  const { data: deviceData, isLoading: devicesLoading } = useDeviceList({
    page: 1,
    pageSize: 1000,
    networkType: 'lte',
    searchText: deviceKeyword || undefined,
  });
  const [selectedSns, setSelectedSns] = useState<string[]>([]);

  // 批量 SN 输入弹窗
  const [batchSNOpen, setBatchSNOpen] = useState(false);
  const [batchSNText, setBatchSNText] = useState('');

  // 冲突检测：查所有 MR 任务，过滤"未结束"（task_status ∈ {waitting, on}）的，
  // 建 SN → taskName 映射；用户选中的设备若命中映射 → 冲突。
  const { data: allTasksData } = useMRTasks({ page: 1, pageSize: 1000 });
  const activeDeviceToTask = useMemo(() => {
    const map = new Map<string, string>(); // sn → first conflicting taskName
    for (const t of allTasksData?.items ?? []) {
      if (t.taskStatus !== 'waitting' && t.taskStatus !== 'on') continue;
      for (const sn of t.targetDeviceSns ?? []) {
        if (!map.has(sn)) map.set(sn, t.taskName);
      }
    }
    return map;
  }, [allTasksData]);
  const conflictSns = useMemo(
    () => selectedSns.filter((sn) => activeDeviceToTask.has(sn)),
    [selectedSns, activeDeviceToTask],
  );

  // 抽屉打开瞬间填默认 taskName。Drawer destroyOnHidden 保证下次重开 form 已 reset，
  // 这时 taskName 是空，正好被此处填上；用户已输入则不覆盖。
  useEffect(() => {
    if (!open) return;
    const cur = form.getFieldValue('taskName');
    if (!cur || String(cur).trim() === '') {
      form.setFieldValue(
        'taskName',
        buildDefaultMRTaskName(
          currentUser?.username || currentUser?.displayName,
          appLocale,
        ),
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const handleClose = () => {
    form.resetFields();
    setSelectedSns([]);
    onClose();
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      const missingType = REQUIRED_MEASURE_TYPES.find((x) => !values.measureType.includes(x));
      if (missingType) {
        void message.error(t('mrTask.create.measureTypeHint'));
        return;
      }
      if (selectedSns.length === 0) {
        void message.error(t('mrTask.create.targetDeviceRequired'));
        return;
      }
      // 冲突阻止：若选中的设备已在其它"未结束"任务中，停止提交并提示
      if (conflictSns.length > 0) {
        const samples = conflictSns.slice(0, 3).map((sn) => {
          const name = activeDeviceToTask.get(sn) ?? '';
          return `${sn}（→ ${name}）`;
        }).join('; ');
        void message.error(
          t('mrTask.create.conflictHasActive', {
            count: conflictSns.length,
            samples: samples + (conflictSns.length > 3 ? '...' : ''),
          }),
          8,
        );
        return;
      }

      // dayjs.toISOString() 默认就是 UTC + ISO 8601（内部 new Date().toISOString()），
      // 无需 utc 插件 — 之前用 .utc() 会报 "utc is not a function"（未 extend utc plugin）。
      const req: CreateMRTaskRequest = {
        taskName: values.taskName,
        mrType: values.measureType.join(','),
        statisPeriod: values.statisPeriod as CreateMRTaskRequest['statisPeriod'],
        reportPeriod: values.reportPeriod as CreateMRTaskRequest['reportPeriod'],
        startTime: values.startTime.toISOString(),
        endTime: values.unlimited ? undefined : values.endTime?.toISOString(),
        targetDeviceSns: selectedSns,
      };
      await createMutation.mutateAsync(req);
      void message.success(t('mrTask.create.success'));
      form.resetFields();
      setSelectedSns([]);
      onCreated();
    } catch (e) {
      if (e instanceof Error) {
        void message.error(`${t('mrTask.create.failed')}: ${e.message}`);
      }
    }
  };

  // 设备列表（取后端 items；字段按 device 类型推断）
  const deviceRows = useMemo(() => {
    type DRow = { sn: string; name?: string; productClass?: string };
    const items = (deviceData?.items ?? []) as DRow[];
    return items.filter((d) => Boolean(d.sn));
  }, [deviceData]);

  // 批量 SN 输入：解析（分隔符 ; , 空格 \n \t \r）→ 与当前 deviceRows 匹配 →
  // 命中的并入 selectedSns；未命中的弹 warning 让用户校对。
  const applyBatchSNs = () => {
    const raw = batchSNText.split(/[\s,;]+/).map((s) => s.trim()).filter(Boolean);
    if (raw.length === 0) {
      void message.warning(t('mrTask.batchSN.emptyInput'));
      return;
    }
    const all = new Set(deviceRows.map((d) => d.sn));
    const hits: string[] = [];
    const misses: string[] = [];
    for (const sn of raw) {
      if (all.has(sn)) hits.push(sn);
      else misses.push(sn);
    }
    if (hits.length > 0) {
      setSelectedSns((prev) => Array.from(new Set([...prev, ...hits])));
    }
    if (misses.length > 0) {
      const samples = misses.slice(0, 3).join(', ') + (misses.length > 3 ? '...' : '');
      void message.warning(
        t('mrTask.batchSN.partialHits', {
          hits: hits.length,
          misses: misses.length,
          samples,
        }),
        6,
      );
    } else {
      void message.success(t('mrTask.batchSN.allHits', { count: hits.length }));
    }
    setBatchSNOpen(false);
    setBatchSNText('');
  };

  return (
    <Drawer
      title={t('mrTask.create.title')}
      width={780}
      open={open}
      onClose={handleClose}
      destroyOnHidden
      extra={
        <Space>
          <Button onClick={handleClose}>{t('common.cancel')}</Button>
          <Button type="primary" loading={createMutation.isPending} onClick={() => void handleSubmit()}>
            {t('common.confirm')}
          </Button>
        </Space>
      }
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          measureType: [...REQUIRED_MEASURE_TYPES],
          statisPeriod: '5120',
          reportPeriod: '15',
          // 默认起始时间向上对齐到下一个 15 分钟边界（默认 reportPeriod=15）
          startTime: dayjs()
            .add(15, 'minute')
            .second(0)
            .millisecond(0)
            .minute(Math.ceil(dayjs().add(15, 'minute').minute() / 15) * 15),
          unlimited: false,
        }}
      >
        <Form.Item
          name="taskName"
          label={t('mrTask.field.taskName')}
          rules={[{ required: true, message: t('mrTask.field.taskNamePlaceholder') }]}
        >
          <Input placeholder={t('mrTask.field.taskNamePlaceholder')} maxLength={128} />
        </Form.Item>

        <Divider orientation="left">{t('mrTask.create.section.params')}</Divider>
        <Form.Item name="measureType" label={t('mrTask.field.measureType')}>
          {/* MRS/MRE/MRO 强制选中（文档 §3.1）— disabled 防止用户取消勾选；
              MDT 自由选填（不强制）。 */}
          <Checkbox.Group
            options={[
              { label: 'MRS', value: 'MRS', disabled: true },
              { label: 'MRE', value: 'MRE', disabled: true },
              { label: 'MRO', value: 'MRO', disabled: true },
              { label: 'MDT', value: 'MDT' },
            ]}
          />
        </Form.Item>
        <Space.Compact block>
          <Form.Item
            name="statisPeriod"
            label={t('mrTask.field.statisPeriod')}
            style={{ flex: 1, marginRight: 8 }}
          >
            <Select
              options={[
                { label: 'ms2048', value: '2048' },
                { label: 'ms5120', value: '5120' },
                { label: 'ms10240', value: '10240' },
                { label: 'min1', value: '1' },
                { label: 'min6', value: '6' },
                { label: 'min12', value: '12' },
                { label: 'min30', value: '30' },
                { label: 'min60', value: '60' },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="reportPeriod"
            label={t('mrTask.field.reportPeriod')}
            style={{ flex: 1 }}
          >
            <Select
              options={[
                { label: '15 min', value: '15' },
                { label: '30 min', value: '30' },
                { label: '60 min', value: '60' },
              ]}
            />
          </Form.Item>
        </Space.Compact>
        <Form.Item
          name="startTime"
          label={t('mrTask.field.startTime')}
          rules={[{ required: true }]}
        >
          <DatePicker
            showTime={{ format: 'HH:mm', minuteStep: minuteStep as 15 | 30 | 60 }}
            format="YYYY-MM-DD HH:mm"
            disabledTime={disabledTime}
            style={{ width: '100%' }}
          />
        </Form.Item>
        <Form.Item name="unlimited" valuePropName="checked">
          <Checkbox>{t('mrTask.create.endTimeUnlimited')}</Checkbox>
        </Form.Item>
        <Form.Item
          noStyle
          shouldUpdate={(prev: FormShape, curr: FormShape) => prev.unlimited !== curr.unlimited}
        >
          {({ getFieldValue }) =>
            !getFieldValue('unlimited') ? (
              <Form.Item
                name="endTime"
                label={t('mrTask.field.endTime')}
                rules={[{ required: true }]}
              >
                <DatePicker
                  showTime={{ format: 'HH:mm', minuteStep: minuteStep as 15 | 30 | 60 }}
                  format="YYYY-MM-DD HH:mm"
                  disabledTime={disabledTime}
                  style={{ width: '100%' }}
                />
              </Form.Item>
            ) : null
          }
        </Form.Item>

        <Form.Item label={t('mrTask.create.section.targetDevice')} required style={{ marginBottom: 0 }}>
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Space>
              <Input.Search
                placeholder={t('mrTask.placeholder.deviceSearch')}
                allowClear
                style={{ width: 240 }}
                value={deviceKeywordInput}
                onChange={(e) => setDeviceKeywordInput(e.target.value)}
                onSearch={(value) => setDeviceKeyword(value.trim())}
              />
              <Button
                icon={<PlusOutlined />}
                onClick={() => { setBatchSNOpen(true); setBatchSNText(''); }}
              >
                {t('mrTask.action.batchSnInput')}
              </Button>
              <Text type="secondary">
                {t('mrTask.create.selectedDeviceCount', { count: selectedSns.length })}
              </Text>
            </Space>
            <Table
              size="small"
              rowKey="sn"
              loading={devicesLoading}
              columns={[
                { title: t('mrTask.field.deviceSn'), dataIndex: 'sn', key: 'sn', width: 200 },
                { title: t('mrTask.field.deviceName'), dataIndex: 'name', key: 'name', ellipsis: true,
                  render: (v?: string) => v ?? '-' },
                { title: t('mrTask.field.productClass'), dataIndex: 'productClass', key: 'productClass', width: 140,
                  render: (v?: string) => v ?? '-' },
                {
                  // 冲突列：显示该 SN 当前所在的活跃任务名（无则空）
                  title: t('mrTask.create.conflictColumn'),
                  key: 'conflict',
                  width: 200,
                  ellipsis: true,
                  render: (_v: unknown, record: { sn: string }) => {
                    const name = activeDeviceToTask.get(record.sn);
                    return name ? (
                      <Tooltip title={t('mrTask.create.conflictTooltip', { name })}>
                        <Tag color="orange">{name}</Tag>
                      </Tooltip>
                    ) : '-';
                  },
                },
              ]}
              dataSource={deviceRows}
              pagination={false}
              rowSelection={{
                selectedRowKeys: selectedSns,
                onChange: (keys) => setSelectedSns(keys.map(String)),
              }}
              scroll={{ x: 760, y: 240 }}
            />
          </Space>
        </Form.Item>
      </Form>

      {/* 批量 SN 输入弹窗 */}
      <Modal
        title={t('mrTask.batchSN.title')}
        open={batchSNOpen}
        onCancel={() => setBatchSNOpen(false)}
        onOk={applyBatchSNs}
        okText={t('mrTask.batchSN.applyOk')}
        cancelText={t('common.cancel')}
        destroyOnHidden
      >
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          <Text type="secondary">{t('mrTask.batchSN.hint')}</Text>
          <Input.TextArea
            rows={10}
            placeholder={t('mrTask.batchSN.placeholder')}
            value={batchSNText}
            onChange={(e) => setBatchSNText(e.target.value)}
          />
        </Space>
      </Modal>
    </Drawer>
  );
}
