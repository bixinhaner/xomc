/**
 * T-0185：自定义聚合任务「新建向导」整页 5 步。
 *
 * 顶部横向 antd Steps，5 步：
 *   ① 基本信息   — 任务名 / 制式（LTE/NR/GSM）/ 持续性（持续·非持续）/ 过期天数（仅非持续）
 *   ② 聚合范围   — 维度 5 选；自选设备按制式过滤多选到 SN；network/band 无需选；
 *                  device_group/product 全量聚合不给子选（给说明文案）
 *   ③ 指标选择   — 按制式 → deviceType 列指标库多选（收集指标 id = K/C 码，min 1）
 *   ④ 聚合设置   — 粒度单选；oneshot 额外给时间范围（RangePicker），continuous 不给（滚动）
 *   ⑤ 确认       — 预览汇总 → 提交
 *
 * 用户拍板不做（独立待办）：自选设备下钻到小区/PLMN；设备组/产品子集选择。
 */

import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Button,
  Card,
  DatePicker,
  Descriptions,
  Input,
  Radio,
  Select,
  Space,
  Spin,
  Steps,
  Tag,
  Transfer,
  message,
} from 'antd';
import dayjs from 'dayjs';
import { useCreatePmAdhoc } from '@core/hooks/api/usePmAdhoc';
import { useDeviceList } from '@core/hooks/api/useDevices';
import { useIndicatorList } from '@core/hooks/api/useIndicatorsLibrary';
import type { AdhocDimension, AdhocMode } from '@core/types/pmAdhoc';
import type { DeviceType } from '@core/types/indicatorLibrary';

// 制式（含 GSM，networkType 过滤直接用小写值）
type WizardTech = 'lte' | 'nr' | 'gsm';

const TECH_OPTIONS: { label: string; value: WizardTech }[] = [
  { label: 'LTE (4G)', value: 'lte' },
  { label: '5G NR', value: 'nr' },
  { label: 'GSM (2G)', value: 'gsm' },
];

// 制式 → 指标库 deviceType（大写枚举）。
const TECH_TO_DEVICE_TYPE: Record<WizardTech, DeviceType> = {
  lte: 'ENB',
  nr: 'GNB',
  gsm: 'GSM',
};

const DIMENSION_OPTIONS: { label: string; value: AdhocDimension; hint: string }[] = [
  { label: '全网汇总', value: 'network', hint: '全网所有设备汇总成一条总线，无需选范围。' },
  { label: '设备组', value: 'device_group', hint: '按设备组分组，每组一条聚合线（同制式全量聚合，无需选子集）。' },
  { label: '产品', value: 'product', hint: '按产品分组，每产品一条聚合线（同制式全量聚合，无需选子集）。' },
  { label: '频段', value: 'band', hint: '按频段自动分组，每频段一条聚合线，无需手选。' },
  { label: '自选设备', value: 'device', hint: '多选具体设备（受制式过滤），每设备一条结果。' },
];

const GRANULARITY_OPTIONS = [
  { label: '15 分钟', value: '15min' },
  { label: '小时', value: 'hourly' },
  { label: '天', value: 'daily' },
  { label: '周', value: 'weekly' },
  { label: '月', value: 'monthly' },
];

interface DeviceTransferItem {
  key: string; // SN
  title: string; // 显示名
  technology: string;
}

export default function PmAdhocWizard() {
  const navigate = useNavigate();
  const createMut = useCreatePmAdhoc();

  const [current, setCurrent] = useState(0);

  // ① 基本信息
  const [name, setName] = useState('');
  const [technology, setTechnology] = useState<WizardTech>('lte');
  const [mode, setMode] = useState<AdhocMode>('oneshot');
  const [expireDays, setExpireDays] = useState<number>(60);

  // ② 聚合范围
  const [dimension, setDimension] = useState<AdhocDimension>('network');
  const [selectedSns, setSelectedSns] = useState<string[]>([]);

  // ③ 指标选择（收集指标 id = K/C 码）
  const [metricPaths, setMetricPaths] = useState<string[]>([]);

  // ④ 聚合设置
  const [granularity, setGranularity] = useState<string>('hourly');
  const [window, setWindow] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(1, 'day'),
    dayjs(),
  ]);

  const needsDevicePick = dimension === 'device' || dimension === 'aggregate_group';

  // 设备列表（自选设备步用，按制式过滤）。仅在需要选设备时才发请求。
  const { data: deviceResp, isLoading: devicesLoading } = useDeviceList(
    { networkType: technology, page: 1, pageSize: 500 },
    { enabled: needsDevicePick },
  );
  const deviceItems: DeviceTransferItem[] = useMemo(
    () =>
      (deviceResp?.items ?? []).map((d) => ({
        key: d.sn,
        title: d.name ? `${d.name} (${d.sn})` : d.sn,
        technology: d.networkType,
      })),
    [deviceResp],
  );

  // 指标库（按制式 → deviceType）。
  const deviceType = TECH_TO_DEVICE_TYPE[technology];
  const { data: indicatorResp, isLoading: indicatorsLoading } = useIndicatorList(deviceType, {
    pageSize: 1000,
  });
  const indicatorItems = useMemo(() => indicatorResp?.items ?? [], [indicatorResp]);

  // ── 步骤校验（决定"下一步"是否可点 / 提交是否可点）──────────────────────
  const step1Valid = name.trim().length > 0;
  const step2Valid = needsDevicePick ? selectedSns.length > 0 : true;
  const step3Valid = metricPaths.length >= 1;
  const step4Valid =
    granularity.length > 0 &&
    (mode === 'continuous' || (window[0] && window[1] && window[1].isAfter(window[0])));

  const canNext = [step1Valid, step2Valid, step3Valid, step4Valid][current];

  const handleSubmit = async () => {
    if (!step1Valid || !step2Valid || !step3Valid || !step4Valid) {
      message.error('请检查各步骤填写是否完整');
      return;
    }
    try {
      await createMut.mutateAsync({
        name: name.trim(),
        mode,
        dimension,
        technology,
        deviceSns: needsDevicePick ? selectedSns : [],
        metricPaths,
        granularities: [granularity],
        // oneshot 带 window；continuous 不带（后端开窗滚动）
        windowStart: mode === 'oneshot' ? window[0].toISOString() : undefined,
        windowEnd: mode === 'oneshot' ? window[1].toISOString() : undefined,
        // 过期天数仅非持续型生效
        expireDays: mode === 'oneshot' ? expireDays : undefined,
      });
      message.success('任务已创建，worker 将开始执行');
      navigate('/performance/pm-adhoc');
    } catch (e) {
      message.error(`创建失败：${(e as Error).message}`);
    }
  };

  // ── 各步内容 ──────────────────────────────────────────────────────────────
  const renderStep1 = () => (
    <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 560 }}>
      <div>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>任务名称 *</div>
        <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="如：小时级 RRC 成功率" />
      </div>
      <div>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>制式 *</div>
        <Radio.Group
          optionType="button"
          buttonStyle="solid"
          options={TECH_OPTIONS}
          value={technology}
          onChange={(e) => {
            setTechnology(e.target.value);
            // 制式切换后清空已选设备 / 指标（避免跨制式残留）
            setSelectedSns([]);
            setMetricPaths([]);
          }}
        />
      </div>
      <div>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>持续性 *</div>
        <Radio.Group
          optionType="button"
          buttonStyle="solid"
          value={mode}
          onChange={(e) => setMode(e.target.value as AdhocMode)}
          options={[
            { label: '非持续（单次）', value: 'oneshot' },
            { label: '持续（滚动）', value: 'continuous' },
          ]}
        />
      </div>
      {mode === 'oneshot' && (
        <div>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>过期天数</div>
          <Input
            type="number"
            min={1}
            style={{ width: 160 }}
            value={expireDays}
            onChange={(e) => setExpireDays(Number(e.target.value) || 60)}
            addonAfter="天"
          />
        </div>
      )}
    </Space>
  );

  const renderStep2 = () => {
    const dimHint = DIMENSION_OPTIONS.find((d) => d.value === dimension)?.hint ?? '';
    return (
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <div>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>聚合维度 *</div>
          <Radio.Group
            value={dimension}
            onChange={(e) => setDimension(e.target.value as AdhocDimension)}
          >
            <Space direction="vertical">
              {DIMENSION_OPTIONS.map((d) => (
                <Radio key={d.value} value={d.value}>
                  {d.label}
                </Radio>
              ))}
            </Space>
          </Radio.Group>
        </div>
        <Alert type="info" showIcon message={dimHint} />
        {needsDevicePick ? (
          <div>
            <div style={{ marginBottom: 8, fontWeight: 500 }}>
              自选设备（{technology.toUpperCase()}，已选 {selectedSns.length}）*
            </div>
            <Spin spinning={devicesLoading}>
              <Transfer<DeviceTransferItem>
                dataSource={deviceItems}
                targetKeys={selectedSns}
                onChange={(keys: React.Key[]) => setSelectedSns(keys.map(String))}
                render={(item) => item.title}
                showSearch
                filterOption={(inputValue, item) =>
                  item.title.toLowerCase().includes(inputValue.toLowerCase())
                }
                titles={['可选设备', '已选设备']}
                listStyle={{ width: 320, height: 360 }}
              />
            </Spin>
          </div>
        ) : (
          <Alert
            type="success"
            showIcon
            message="该维度按制式全量聚合，无需手选设备子集"
            description="后端会对该制式范围内的设备自动分组聚合，每组/每产品/每频段一条结果线。"
          />
        )}
      </Space>
    );
  };

  const renderStep3 = () => (
    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
      <Alert
        type="info"
        showIcon
        message={`按 ${technology.toUpperCase()} 制式列出指标库（${deviceType}），至少选 1 个`}
      />
      <Select
        mode="multiple"
        style={{ width: '100%' }}
        placeholder="选择指标（可搜索编号 / 名称）"
        loading={indicatorsLoading}
        value={metricPaths}
        onChange={(v: string[]) => setMetricPaths(v)}
        optionFilterProp="label"
        options={indicatorItems.map((ind) => ({
          label: `${ind.id} ${ind.name ?? ind.cnName ?? ''}`,
          value: ind.id,
        }))}
        maxTagCount="responsive"
      />
      <div style={{ color: '#888' }}>已选 {metricPaths.length} 个指标</div>
    </Space>
  );

  const renderStep4 = () => (
    <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 560 }}>
      <div>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>聚合粒度 *</div>
        <Radio.Group
          optionType="button"
          buttonStyle="solid"
          options={GRANULARITY_OPTIONS}
          value={granularity}
          onChange={(e) => setGranularity(e.target.value)}
        />
      </div>
      {mode === 'oneshot' ? (
        <div>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>时间范围 *</div>
          <DatePicker.RangePicker
            showTime
            style={{ width: '100%' }}
            value={window}
            onChange={(v) => {
              if (v && v[0] && v[1]) setWindow([v[0], v[1]]);
            }}
          />
        </div>
      ) : (
        <Alert
          type="info"
          showIcon
          message="持续型任务无需时间范围"
          description="后端按粒度自动滚动聚合最新数据（不固定时间窗）。"
        />
      )}
    </Space>
  );

  const renderStep5 = () => {
    const metricNames = metricPaths.map((id) => {
      const ind = indicatorItems.find((x) => x.id === id);
      return ind ? `${ind.id} ${ind.name ?? ''}`.trim() : id;
    });
    return (
      <Descriptions bordered column={1} size="middle">
        <Descriptions.Item label="任务名称">{name || <Tag>未填</Tag>}</Descriptions.Item>
        <Descriptions.Item label="制式">{technology.toUpperCase()}</Descriptions.Item>
        <Descriptions.Item label="持续性">
          {mode === 'continuous' ? <Tag color="purple">持续（滚动）</Tag> : <Tag>非持续（单次）</Tag>}
        </Descriptions.Item>
        {mode === 'oneshot' && (
          <Descriptions.Item label="过期天数">{expireDays} 天</Descriptions.Item>
        )}
        <Descriptions.Item label="聚合维度">
          {DIMENSION_OPTIONS.find((d) => d.value === dimension)?.label ?? dimension}
        </Descriptions.Item>
        {needsDevicePick && (
          <Descriptions.Item label="选中设备">
            {selectedSns.length > 0 ? `${selectedSns.length} 台：${selectedSns.join(', ')}` : <Tag>未选</Tag>}
          </Descriptions.Item>
        )}
        <Descriptions.Item label="指标">
          {metricNames.length > 0 ? (
            <Space size={[4, 4]} wrap>
              {metricNames.map((m) => (
                <Tag key={m}>{m}</Tag>
              ))}
            </Space>
          ) : (
            <Tag>未选</Tag>
          )}
        </Descriptions.Item>
        <Descriptions.Item label="粒度">
          {GRANULARITY_OPTIONS.find((g) => g.value === granularity)?.label ?? granularity}
        </Descriptions.Item>
        {mode === 'oneshot' && (
          <Descriptions.Item label="时间范围">
            {window[0]?.format('YYYY-MM-DD HH:mm')} ~ {window[1]?.format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
        )}
      </Descriptions>
    );
  };

  const steps = [
    { title: '基本信息', content: renderStep1 },
    { title: '聚合范围', content: renderStep2 },
    { title: '指标选择', content: renderStep3 },
    { title: '聚合设置', content: renderStep4 },
    { title: '确认', content: renderStep5 },
  ];

  return (
    <Card
      title="新建自定义聚合任务"
      extra={<Button onClick={() => navigate('/performance/pm-adhoc')}>返回列表</Button>}
    >
      <Steps current={current} items={steps.map((s) => ({ title: s.title }))} style={{ marginBottom: 24 }} />

      <Card type="inner" style={{ minHeight: 360 }}>
        {steps[current].content()}
      </Card>

      <div style={{ marginTop: 24, display: 'flex', justifyContent: 'space-between' }}>
        <Button disabled={current === 0} onClick={() => setCurrent((c) => c - 1)}>
          上一步
        </Button>
        {current < steps.length - 1 ? (
          <Button type="primary" disabled={!canNext} onClick={() => setCurrent((c) => c + 1)}>
            下一步
          </Button>
        ) : (
          <Button type="primary" loading={createMut.isPending} onClick={handleSubmit}>
            提交
          </Button>
        )}
      </div>
    </Card>
  );
}
