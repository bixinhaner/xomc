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

import { useEffect, useMemo, useState } from 'react';
import { useIntl } from 'react-intl';
import { useNavigate, useParams } from 'react-router-dom';
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
import { InputAddon } from '@/components/common/InputAddon';
import { useCreatePmAdhoc, useUpdatePmAdhoc, usePmAdhocDetail } from '@core/hooks/api/usePmAdhoc';
import { useDeviceList } from '@core/hooks/api/useDevices';
import { useMetricObjectsByDevices } from '@core/hooks/api/usePmQuery';
import { useIndicatorCandidates } from '@core/hooks/api/usePerformance';
import type { IndicatorCandidate } from '@core/services/api/pmApi';
import type { AdhocDimension, AdhocMode } from '@core/types/pmAdhoc';
import type { DeviceType } from '@core/types/indicatorLibrary';
import { isGranularityDimensionSupported } from '@core/utils/pmAdhocConstraints';
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

// 指标穿梭框条目：key=指标 id（K/C 编号，也是提交标识），其余字段供渲染与过滤。
interface MetricTransferItem {
  key: string; // 指标 id（K/C 编号）
  id: string;
  name: string;
  cnName: string;
  isCounter: boolean;
  // 搜索用拼接串（编号 + 中英文名），小写。
  searchText: string;
}

export default function PmAdhocWizard() {
  const intl = useIntl();
  const navigate = useNavigate();
  const createMut = useCreatePmAdhoc();
  // T-0194：编辑模式 —— 路由带 :id 即编辑（复用本向导），无 id 则为新建。
  const { id: editId } = useParams<{ id: string }>();
  const isEdit = Boolean(editId);
  const updateMut = useUpdatePmAdhoc();
  const { data: editTask } = usePmAdhocDetail(editId);

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
        value: 'aggregate_group',
        hint: intl.formatMessage({ id: 'perf.adhoc.dimDeviceHint' }),
      },
    ],
    [intl],
  );

  // #669：自定义聚合任务下线 15min，最细粒度限定 hourly（详见 pmAdhocConstraints.ts）。
  const GRANULARITY_OPTIONS = useMemo(
    () => [
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
  // 穿梭框候选侧类型筛选：all=全部 / kpi=只看 KPI(K) / counter=只看计数(C)。
  const [metricTypeFilter, setMetricTypeFilter] = useState<'all' | 'kpi' | 'counter'>('all');

  // ④ 聚合设置
  const [granularity, setGranularity] = useState<string>('hourly');
  const [window, setWindow] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(1, 'day'),
    dayjs(),
  ]);

  // T-0194 编辑模式：预填守卫（只预填一次，避免覆盖用户后续编辑）+ 下钻是否被用户改动。
  const [prefilled, setPrefilled] = useState(false);
  const [drilldownTouched, setDrilldownTouched] = useState(false);
  const [originalObjectLdns, setOriginalObjectLdns] = useState<string[]>([]);

  // 编辑模式：详情到手后按各步初值预填（名称/制式/模式/维度/设备/指标/粒度/时窗）。
  // 制式与模式预填后在 UI 锁定只读（结构性字段不可改）。
  useEffect(() => {
    if (!isEdit || prefilled || !editTask) return;
    setName(editTask.name);
    if (editTask.technology === 'lte' || editTask.technology === 'nr' || editTask.technology === 'gsm') {
      setTechnology(editTask.technology);
    }
    setMode(editTask.mode);
    setExpireDays(editTask.expireDays || 60);
    setDimension(editTask.dimension);
    setSelectedSns(editTask.deviceSns ?? []);
    setMetricPaths(editTask.metricPaths ?? []);
    if (editTask.granularities && editTask.granularities.length > 0) {
      setGranularity(editTask.granularities[0]);
    }
    if (editTask.mode === 'oneshot' && editTask.windowStart && editTask.windowEnd) {
      const ws = dayjs(editTask.windowStart);
      const we = dayjs(editTask.windowEnd);
      if (ws.isValid() && we.isValid()) setWindow([ws, we]);
    }
    setOriginalObjectLdns(editTask.objectLdns ?? []);
    setPrefilled(true);
  }, [isEdit, prefilled, editTask]);

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

  // 指标候选（按制式 → deviceType）。走 pm 权限的 /pm/kpi/definitions：运维可访问、
  // 全量加载无截断、含计数器。替代原 super_admin 的 /indicators（截断 + 403）。
  const deviceType = TECH_TO_DEVICE_TYPE[technology];
  const { data: candidates, isLoading: indicatorsLoading } = useIndicatorCandidates(deviceType, {
    includeCounters: true,
  });
  const indicatorItems = useMemo<IndicatorCandidate[]>(() => candidates ?? [], [candidates]);
  // 完整 lookup（id → 候选项），用于已选侧渲染与确认页指标名解析——不受类型筛选影响（D5）。
  const indicatorById = useMemo(() => {
    const m = new Map<string, IndicatorCandidate>();
    for (const ind of indicatorItems) m.set(ind.id, ind);
    return m;
  }, [indicatorItems]);
  // Transfer dataSource：类型筛选在 dataSource 层预过滤（不靠 antd 的 filterOption——
  // 后者只在搜索框有输入时才被调用，空搜索下类型筛选会失效）。
  // 规则：保留「类型匹配」或「已被选中」的候选——已选项无论类型都进 dataSource，
  // 保证右侧已选侧按 targetKeys 完整渲染、不被类型筛选隐藏（D5）。
  const selectedKeySet = useMemo(() => new Set(metricPaths), [metricPaths]);
  const metricTransferItems = useMemo<MetricTransferItem[]>(
    () =>
      indicatorItems
        .filter((ind) => {
          const matchType =
            metricTypeFilter === 'all' ||
            (metricTypeFilter === 'kpi' && !ind.isCounter) ||
            (metricTypeFilter === 'counter' && ind.isCounter);
          return matchType || selectedKeySet.has(ind.id);
        })
        .map((ind) => ({
          key: ind.id,
          id: ind.id,
          name: ind.cnName || ind.name,
          cnName: ind.cnName,
          isCounter: ind.isCounter,
          searchText: `${ind.id} ${ind.name} ${ind.cnName}`.toLowerCase(),
        })),
    [indicatorItems, metricTypeFilter, selectedKeySet],
  );

  // ── 步骤校验（决定"下一步"是否可点 / 提交是否可点）──────────────────────
  const step1Valid = name.trim().length > 0;
  const step2Valid = needsDevicePick ? selectedSns.length > 0 : true;
  const step3Valid = metricPaths.length >= 1;
  const step4Valid =
    granularity.length > 0 &&
    // #669：兜底——15min 已从粒度选项删除，但编辑模式遇旧任务仍可能传入 15min，由此拦截。
    isGranularityDimensionSupported(granularity, dimension) &&
    (mode === 'continuous' || (window[0] && window[1] && window[1].isAfter(window[0])));

  const canNext = [step1Valid, step2Valid, step3Valid, step4Valid][current];

  const handleSubmit = async () => {
    if (!step1Valid || !step2Valid || !step3Valid || !step4Valid) {
      message.error(intl.formatMessage({ id: 'perf.adhoc.checkStepsIncomplete' }));
      return;
    }
    try {
      // T-0193：自选设备分支下钻白名单——全勾→空数组→api 层不传（保持现状语义）。
      // 编辑模式：用户未动下钻时沿用任务原白名单（避免编辑指标顺带清掉小区过滤）。
      let objectLdns = needsDevicePick ? getEffectiveLdns(cellSel, objectsByDevice) : [];
      if (isEdit && !drilldownTouched && objectLdns.length === 0) {
        objectLdns = originalObjectLdns;
      }
      if (isEdit && editId) {
        // 编辑：mode/technology/dimension 锁定不可改，只发可改字段；后端按既有任务校验。
        await updateMut.mutateAsync({
          id: editId,
          input: {
            name: name.trim(),
            deviceSns: needsDevicePick ? selectedSns : [],
            metricPaths,
            granularities: [granularity],
            windowStart: mode === 'oneshot' ? window[0].toISOString() : undefined,
            windowEnd: mode === 'oneshot' ? window[1].toISOString() : undefined,
            objectLdns: objectLdns.length > 0 ? objectLdns : undefined,
          },
        });
        message.success(intl.formatMessage({ id: 'perf.adhoc.taskUpdated' }));
        navigate('/performance/pm-adhoc');
        return;
      }
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
      message.error(
        intl.formatMessage(
          { id: isEdit ? 'perf.adhoc.updateFailed' : 'perf.adhoc.createFailed' },
          { msg: (e as Error).message },
        ),
      );
    }
  };

  // ── 各步内容 ──────────────────────────────────────────────────────────────
  const renderStep1 = () => (
    <Space orientation="vertical" size="large" style={{ width: '100%', maxWidth: 560 }}>
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
        {/* T-0194 编辑模式：制式锁定只读（建后不可改） */}
        <Radio.Group
          optionType="button"
          buttonStyle="solid"
          options={TECH_OPTIONS}
          value={technology}
          disabled={isEdit}
          onChange={(e) => {
            setTechnology(e.target.value);
            // 制式切换后清空已选设备 / 指标（避免跨制式残留）
            setSelectedSns([]);
            setCellSel({}); // 下钻选择重置（全选）
            setMetricPaths([]);
          }}
        />
        {isEdit && (
          <div style={{ marginTop: 6, color: '#999', fontSize: 12 }}>
            {intl.formatMessage({ id: 'perf.adhoc.editLockedTech' })}
          </div>
        )}
      </div>
      <div>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldModeReq' })}</div>
        {/* T-0194 编辑模式：模式锁定只读（变更会翻转时窗↔cron 语义） */}
        <Radio.Group
          optionType="button"
          buttonStyle="solid"
          value={mode}
          disabled={isEdit}
          onChange={(e) => setMode(e.target.value as AdhocMode)}
          options={[
            { label: intl.formatMessage({ id: 'perf.adhoc.modeOneshotFull' }), value: 'oneshot' },
            { label: intl.formatMessage({ id: 'perf.adhoc.modeContinuousFull' }), value: 'continuous' },
          ]}
        />
        {isEdit && (
          <div style={{ marginTop: 6, color: '#999', fontSize: 12 }}>
            {intl.formatMessage({ id: 'perf.adhoc.editLockedMode' })}
          </div>
        )}
      </div>
      {mode === 'oneshot' && (
        <div>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldExpireDays' })}</div>
          {/* T-0194 编辑模式：过期天数锁定只读（建后不可改，提交不发该字段）。
              antd6 addonAfter 已废弃：Space.Compact + InputAddon 复刻天数后缀盒子。 */}
          <Space.Compact style={{ width: 160 }}>
            <Input
              type="number"
              min={1}
              value={expireDays}
              disabled={isEdit}
              onChange={(e) => setExpireDays(Number(e.target.value) || 60)}
            />
            <InputAddon>{intl.formatMessage({ id: 'perf.adhoc.daySuffix' })}</InputAddon>
          </Space.Compact>
          {isEdit && (
            <div style={{ marginTop: 6, color: '#999', fontSize: 12 }}>
              {intl.formatMessage({ id: 'perf.adhoc.editLockedExpire' })}
            </div>
          )}
        </div>
      )}
    </Space>
  );

  const renderStep2 = () => {
    const dimHint = DIMENSION_OPTIONS.find((d) => d.value === dimension)?.hint ?? '';
    return (
      <Space orientation="vertical" size="large" style={{ width: '100%' }}>
        <div>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldDimensionReq' })}</div>
          <Radio.Group
            value={dimension}
            onChange={(e) => {
              const next = e.target.value as AdhocDimension;
              setDimension(next);
              // #669：编辑模式遇旧 15min 任务，切维度时一并回落 hourly（新建路径已无 15min 选项）。
              if (!isGranularityDimensionSupported(granularity, next)) {
                setGranularity('hourly');
              }
            }}
          >
            <Space orientation="vertical">
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
                  onChange={(v) => {
                    setCellSel(v);
                    setDrilldownTouched(true); // T-0194：用户改了下钻 → 编辑提交用新白名单而非原值
                  }}
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

  // 穿梭框单行渲染：编号 + 指标名 + KPI/计数 小标签。
  const renderMetricItem = (item: MetricTransferItem) => (
    <Space size={6}>
      <Tag color={item.isCounter ? 'blue' : 'orange'} style={{ marginInlineEnd: 0 }}>
        {item.isCounter
          ? intl.formatMessage({ id: 'perf.adhoc.metricTagCounter' })
          : intl.formatMessage({ id: 'perf.adhoc.metricTagKpi' })}
      </Tag>
      <span style={{ color: '#999' }}>{item.id}</span>
      <span>{item.name}</span>
    </Space>
  );

  const renderStep3 = () => (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Alert
        type="info"
        showIcon
        message={intl.formatMessage(
          { id: 'perf.adhoc.metricHint' },
          { tech: technology.toUpperCase(), deviceType },
        )}
      />
      <Space size="middle" align="center">
        <span style={{ fontWeight: 500 }}>
          {intl.formatMessage({ id: 'perf.adhoc.metricTypeFilterLabel' })}
        </span>
        <Select
          style={{ width: 160 }}
          value={metricTypeFilter}
          onChange={(v: 'all' | 'kpi' | 'counter') => setMetricTypeFilter(v)}
          options={[
            { label: intl.formatMessage({ id: 'perf.adhoc.metricTypeAll' }), value: 'all' },
            { label: intl.formatMessage({ id: 'perf.adhoc.metricTypeKpi' }), value: 'kpi' },
            { label: intl.formatMessage({ id: 'perf.adhoc.metricTypeCounter' }), value: 'counter' },
          ]}
        />
      </Space>
      <Spin spinning={indicatorsLoading}>
        <Transfer<MetricTransferItem>
          dataSource={metricTransferItems}
          targetKeys={metricPaths}
          onChange={(keys: React.Key[]) => setMetricPaths(keys.map(String))}
          render={renderMetricItem}
          showSearch
          // 类型筛选已下移到 dataSource 层预过滤（见 metricTransferItems）；此处 filterOption
          // 只负责搜索词匹配，左右两栏一致。空搜索时 antd 不调本函数也无妨——类型筛选不依赖它。
          filterOption={(inputValue, item) => {
            const kw = inputValue.trim().toLowerCase();
            return kw === '' || item.searchText.includes(kw);
          }}
          titles={[
            intl.formatMessage({ id: 'perf.adhoc.transferAvailableMetric' }),
            intl.formatMessage({ id: 'perf.adhoc.transferSelectedMetric' }),
          ]}
          // #665：让两个列表横向撑满父容器、左右等分（antd Transfer 内部 flex 布局）。
          // 原 360×380 写死宽度导致长指标名挤掉/换行；改为 flex:1 + minWidth:0
          // 大屏看到完整名称、小屏自动收缩不出横向滚动条。高度保持 480 让一屏多看几行。
          style={{ width: '100%' }}
          listStyle={{ flex: '1 1 0', minWidth: 0, height: 480 }}
        />
      </Spin>
      <div style={{ color: '#888' }}>
        {intl.formatMessage({ id: 'perf.adhoc.metricSelectedCount' }, { count: metricPaths.length })}
      </div>
    </Space>
  );

  const renderStep4 = () => (
    <Space orientation="vertical" size="large" style={{ width: '100%', maxWidth: 560 }}>
      <div>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>{intl.formatMessage({ id: 'perf.adhoc.fieldGranReq' })}</div>
        {/* #669：自定义聚合任务最细粒度限定 hourly，15min 选项已从数组中移除。 */}
        <Radio.Group
          optionType="button"
          buttonStyle="solid"
          value={granularity}
          onChange={(e) => setGranularity(e.target.value)}
        >
          {GRANULARITY_OPTIONS.map((g) => (
            <Radio.Button key={g.value} value={g.value}>
              {g.label}
            </Radio.Button>
          ))}
        </Radio.Group>
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
      const ind = indicatorById.get(id);
      return ind ? `${ind.id} ${ind.cnName || ind.name}`.trim() : id;
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
      title={intl.formatMessage({
        id: isEdit ? 'perf.adhoc.wizardEditTitle' : 'perf.adhoc.wizardTitle',
      })}
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
          <Button
            type="primary"
            loading={createMut.isPending || updateMut.isPending}
            onClick={handleSubmit}
          >
            {intl.formatMessage({
              id: isEdit ? 'perf.adhoc.btnSaveEdit' : 'perf.adhoc.btnSubmit',
            })}
          </Button>
        )}
      </div>
    </Card>
  );
}
