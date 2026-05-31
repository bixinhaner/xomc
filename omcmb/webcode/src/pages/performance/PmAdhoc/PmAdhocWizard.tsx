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
 * T-0193：第2步自选设备（device/aggregate_group）分支接下钻勾选小区/PLMN；
 *   提交时把各设备选中 objectLdn 汇成 objectLdns 随 create 落库（默认全勾→不传，保持现状语义）。
 */

import { useMemo, useState } from 'react';
import { useIntl } from 'react-intl';
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
import { useMetricObjectsByDevices } from '@core/hooks/api/usePmQuery';
import { useIndicatorList } from '@core/hooks/api/useIndicatorsLibrary';
import type { AdhocDimension, AdhocMode } from '@core/types/pmAdhoc';
import type { DeviceType } from '@core/types/indicatorLibrary';
import CellDrilldownSelector from '../PmDashboard/CellDrilldownSelector';
import { getEffectiveLdns, type CellSelection } from '../PmDashboard/cellDrilldownUtils';

// 制式（含 GSM，networkType 过滤直接用小写值）
type WizardTech = 'lte' | 'nr' | 'gsm';

// 制式 → 指标库 deviceType（大写枚举）。
const TECH_TO_DEVICE_TYPE: Record<WizardTech, DeviceType> = {
  lte: 'ENB',
  nr: 'GNB',
  gsm: 'GSM',
};

interface DeviceTransferItem {
  key: string; // SN
  title: string; // 显示名
  technology: string;
}

export default function PmAdhocWizard() {
  const intl = useIntl();
  const navigate = useNavigate();
  const createMut = useCreatePmAdhoc();

  // 制式 / 维度 / 粒度选项：value 不变，仅 label / hint 走 i18n。
  const TECH_OPTIONS = useMemo<{ label: string; value: WizardTech }[]>(
    () => [
      { label: intl.formatMessage({ id: 'perf.adhoc.techLte' }), value: 'lte' },
      { label: intl.formatMessage({ id: 'perf.adhoc.techNr' }), value: 'nr' },
      { label: intl.formatMessage({ id: 'perf.adhoc.techGsm' }), value: 'gsm' },
    ],
    [intl],
  );

  const DIMENSION_OPTIONS = useMemo<{ label: string; value: AdhocDimension; hint: string }[]>(
    () => [
      {
        label: intl.formatMessage({ id: 'perf.adhoc.dimNetworkLabel' }),
        value: 'network',
        hint: intl.formatMessage({ id: 'perf.adhoc.dimNetworkHint' }),
      },
      {
        label: intl.formatMessage({ id: 'perf.adhoc.dimDeviceGroupLabel' }),
        value: 'device_group',
        hint: intl.formatMessage({ id: 'perf.adhoc.dimDeviceGroupHint' }),
      },
      {
        label: intl.formatMessage({ id: 'perf.adhoc.dimProductLabel' }),
        value: 'product',
        hint: intl.formatMessage({ id: 'perf.adhoc.dimProductHint' }),
      },
      {
        label: intl.formatMessage({ id: 'perf.adhoc.dimBandLabel' }),
        value: 'band',
        hint: intl.formatMessage({ id: 'perf.adhoc.dimBandHint' }),
      },
      {
        label: intl.formatMessage({ id: 'perf.adhoc.dimDeviceLabel' }),
        value: 'device',
        hint: intl.formatMessage({ id: 'perf.adhoc.dimDeviceHint' }),
      },
    ],
    [intl],
  );

  const GRANULARITY_OPTIONS = useMemo(
    () => [
      { label: intl.formatMessage({ id: 'perf.adhoc.granular15min' }), value: '15min' },
      { label: intl.formatMessage({ id: 'perf.adhoc.granularHourly' }), value: 'hourly' },
      { label: intl.formatMessage({ id: 'perf.adhoc.granularDaily' }), value: 'daily' },
      { label: intl.formatMessage({ id: 'perf.adhoc.granularWeekly' }), value: 'weekly' },
      { label: intl.formatMessage({ id: 'perf.adhoc.granularMonthly' }), value: 'monthly' },
    ],
    [intl],
  );

  const [current, setCurrent] = useState(0);

  // ① 基本信息
  const [name, setName] = useState('');
  const [technology, setTechnology] = useState<WizardTech>('lte');
  const [mode, setMode] = useState<AdhocMode>('oneshot');
  const [expireDays, setExpireDays] = useState<number>(60);

  // ② 聚合范围
  const [dimension, setDimension] = useState<AdhocDimension>('network');
  const [selectedSns, setSelectedSns] = useState<string[]>([]);
  // T-0193 下钻：每设备选中的小区/PLMN 子集（缺席=全选不过滤）。
  const [cellSel, setCellSel] = useState<CellSelection>({});

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

  // T-0193：下钻白名单计算用的「按设备小区清单」（与选择器内部同 query key 去重，无额外请求）。
  const { byDevice: objectsByDevice } = useMetricObjectsByDevices(
    needsDevicePick ? selectedSns : [],
    technology,
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
      message.error(intl.formatMessage({ id: 'perf.adhoc.checkStepsIncomplete' }));
      return;
    }
    try {
      // T-0193：自选设备分支下钻白名单——全勾→空数组→api 层不传（保持现状语义）。
      const objectLdns = needsDevicePick ? getEffectiveLdns(cellSel, objectsByDevice) : [];
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
        // 小区/PLMN 白名单（空=不传）
        objectLdns: objectLdns.length > 0 ? objectLdns : undefined,
      });
      message.success(intl.formatMessage({ id: 'perf.adhoc.taskCreated' }));
      navigate('/performance/pm-adhoc');
    } catch (e) {
      message.error(intl.formatMessage({ id: 'perf.adhoc.createFailed' }, { msg: (e as Error).message }));
    }
  };

  // ── 各步内容 ──────────────────────────────────────────────────────────────
  const renderStep1 = () => (
    <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 560 }}>
      <div>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldTaskName' })}</div>
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={intl.formatMessage({ id: 'perf.adhoc.taskNamePlaceholder' })}
        />
      </div>
      <div>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldTechReq' })}</div>
        <Radio.Group
          optionType="button"
          buttonStyle="solid"
          options={TECH_OPTIONS}
          value={technology}
          onChange={(e) => {
            setTechnology(e.target.value);
            // 制式切换后清空已选设备 / 指标（避免跨制式残留）
            setSelectedSns([]);
            setCellSel({}); // 下钻选择重置（全选）
            setMetricPaths([]);
          }}
        />
      </div>
      <div>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldModeReq' })}</div>
        <Radio.Group
          optionType="button"
          buttonStyle="solid"
          value={mode}
          onChange={(e) => setMode(e.target.value as AdhocMode)}
          options={[
            { label: intl.formatMessage({ id: 'perf.adhoc.modeOneshotFull' }), value: 'oneshot' },
            { label: intl.formatMessage({ id: 'perf.adhoc.modeContinuousFull' }), value: 'continuous' },
          ]}
        />
      </div>
      {mode === 'oneshot' && (
        <div>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldExpireDays' })}</div>
          <Input
            type="number"
            min={1}
            style={{ width: 160 }}
            value={expireDays}
            onChange={(e) => setExpireDays(Number(e.target.value) || 60)}
            addonAfter={intl.formatMessage({ id: 'perf.adhoc.daySuffix' })}
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
          <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldDimensionReq' })}</div>
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
              {intl.formatMessage(
                { id: 'perf.adhoc.devicePickLabel' },
                { tech: technology.toUpperCase(), count: selectedSns.length },
              )}
            </div>
            <Spin spinning={devicesLoading}>
              <Transfer<DeviceTransferItem>
                dataSource={deviceItems}
                targetKeys={selectedSns}
                onChange={(keys: React.Key[]) => {
                  setSelectedSns(keys.map(String));
                  setCellSel({}); // 设备集变更 → 下钻选择重置（全选）。
                }}
                render={(item) => item.title}
                showSearch
                filterOption={(inputValue, item) =>
                  item.title.toLowerCase().includes(inputValue.toLowerCase())
                }
                titles={[
                  intl.formatMessage({ id: 'perf.adhoc.transferAvailable' }),
                  intl.formatMessage({ id: 'perf.adhoc.transferSelected' }),
                ]}
                listStyle={{ width: 320, height: 360 }}
              />
            </Spin>
            {selectedSns.length > 0 && (
              <div style={{ marginTop: 16 }}>
                <div style={{ marginBottom: 8, fontWeight: 500 }}>
                  {intl.formatMessage({ id: 'perf.drilldown.label' })}
                </div>
                <CellDrilldownSelector
                  deviceSns={selectedSns}
                  technology={technology}
                  value={cellSel}
                  onChange={setCellSel}
                />
              </div>
            )}
          </div>
        ) : (
          <Alert
            type="success"
            showIcon
            message={intl.formatMessage({ id: 'perf.adhoc.scopeAutoMessage' })}
            description={intl.formatMessage({ id: 'perf.adhoc.scopeAutoDesc' })}
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
        message={intl.formatMessage(
          { id: 'perf.adhoc.metricHint' },
          { tech: technology.toUpperCase(), deviceType },
        )}
      />
      <Select
        mode="multiple"
        style={{ width: '100%' }}
        placeholder={intl.formatMessage({ id: 'perf.adhoc.metricSelectPlaceholder' })}
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
      <div style={{ color: '#888' }}>
        {intl.formatMessage({ id: 'perf.adhoc.metricSelectedCount' }, { count: metricPaths.length })}
      </div>
    </Space>
  );

  const renderStep4 = () => (
    <Space direction="vertical" size="large" style={{ width: '100%', maxWidth: 560 }}>
      <div>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldGranReq' })}</div>
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
          <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldTimeRangeReq' })}</div>
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
          message={intl.formatMessage({ id: 'perf.adhoc.continuousNoRange' })}
          description={intl.formatMessage({ id: 'perf.adhoc.continuousNoRangeDesc' })}
        />
      )}
    </Space>
  );

  const renderStep5 = () => {
    const effectiveLdns = needsDevicePick ? getEffectiveLdns(cellSel, objectsByDevice) : [];
    const metricNames = metricPaths.map((id) => {
      const ind = indicatorItems.find((x) => x.id === id);
      return ind ? `${ind.id} ${ind.name ?? ''}`.trim() : id;
    });
    return (
      <Descriptions bordered column={1} size="middle">
        <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.confirmTaskName' })}>
          {name || <Tag>{intl.formatMessage({ id: 'perf.adhoc.confirmNotFilled' })}</Tag>}
        </Descriptions.Item>
        <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.confirmTech' })}>{technology.toUpperCase()}</Descriptions.Item>
        <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.confirmMode' })}>
          {mode === 'continuous' ? (
            <Tag color="purple">{intl.formatMessage({ id: 'perf.adhoc.modeContinuousFull' })}</Tag>
          ) : (
            <Tag>{intl.formatMessage({ id: 'perf.adhoc.modeOneshotFull' })}</Tag>
          )}
        </Descriptions.Item>
        {mode === 'oneshot' && (
          <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.confirmExpireDays' })}>
            {expireDays} {intl.formatMessage({ id: 'perf.adhoc.daySuffix' })}
          </Descriptions.Item>
        )}
        <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.confirmDimension' })}>
          {DIMENSION_OPTIONS.find((d) => d.value === dimension)?.label ?? dimension}
        </Descriptions.Item>
        {needsDevicePick && (
          <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.confirmSelectedDevices' })}>
            {selectedSns.length > 0 ? (
              intl.formatMessage(
                { id: 'perf.adhoc.confirmDeviceCount' },
                { count: selectedSns.length, sns: selectedSns.join(', ') },
              )
            ) : (
              <Tag>{intl.formatMessage({ id: 'perf.adhoc.confirmNotSelected' })}</Tag>
            )}
          </Descriptions.Item>
        )}
        {needsDevicePick && (
          <Descriptions.Item label={intl.formatMessage({ id: 'perf.drilldown.confirmCells' })}>
            {effectiveLdns.length > 0 ? (
              intl.formatMessage(
                { id: 'perf.drilldown.confirmCellSubset' },
                { count: effectiveLdns.length },
              )
            ) : (
              <Tag>{intl.formatMessage({ id: 'perf.drilldown.confirmCellAll' })}</Tag>
            )}
          </Descriptions.Item>
        )}
        <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.confirmMetric' })}>
          {metricNames.length > 0 ? (
            <Space size={[4, 4]} wrap>
              {metricNames.map((m) => (
                <Tag key={m}>{m}</Tag>
              ))}
            </Space>
          ) : (
            <Tag>{intl.formatMessage({ id: 'perf.adhoc.confirmNotSelected' })}</Tag>
          )}
        </Descriptions.Item>
        <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.confirmGranularity' })}>
          {GRANULARITY_OPTIONS.find((g) => g.value === granularity)?.label ?? granularity}
        </Descriptions.Item>
        {mode === 'oneshot' && (
          <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.confirmTimeRange' })}>
            {window[0]?.format('YYYY-MM-DD HH:mm')} ~ {window[1]?.format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
        )}
      </Descriptions>
    );
  };

  const steps = [
    { title: intl.formatMessage({ id: 'perf.adhoc.stepBasic' }), content: renderStep1 },
    { title: intl.formatMessage({ id: 'perf.adhoc.stepScope' }), content: renderStep2 },
    { title: intl.formatMessage({ id: 'perf.adhoc.stepMetric' }), content: renderStep3 },
    { title: intl.formatMessage({ id: 'perf.adhoc.stepSetting' }), content: renderStep4 },
    { title: intl.formatMessage({ id: 'perf.adhoc.stepConfirm' }), content: renderStep5 },
  ];

  return (
    <Card
      title={intl.formatMessage({ id: 'perf.adhoc.wizardTitle' })}
      extra={
        <Button onClick={() => navigate('/performance/pm-adhoc')}>
          {intl.formatMessage({ id: 'perf.adhoc.backToList' })}
        </Button>
      }
    >
      <Steps current={current} items={steps.map((s) => ({ title: s.title }))} style={{ marginBottom: 24 }} />

      <Card type="inner" style={{ minHeight: 360 }}>
        {steps[current].content()}
      </Card>

      <div style={{ marginTop: 24, display: 'flex', justifyContent: 'space-between' }}>
        <Button disabled={current === 0} onClick={() => setCurrent((c) => c - 1)}>
          {intl.formatMessage({ id: 'perf.adhoc.btnPrev' })}
        </Button>
        {current < steps.length - 1 ? (
          <Button type="primary" disabled={!canNext} onClick={() => setCurrent((c) => c + 1)}>
            {intl.formatMessage({ id: 'perf.adhoc.btnNext' })}
          </Button>
        ) : (
          <Button type="primary" loading={createMut.isPending} onClick={handleSubmit}>
            {intl.formatMessage({ id: 'perf.adhoc.btnSubmit' })}
          </Button>
        )}
      </div>
    </Card>
  );
}
