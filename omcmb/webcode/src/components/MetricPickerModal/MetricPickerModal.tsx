/**
 * T-0174 指标选择 Modal。
 *
 * - 复用 frontend-core useIndicatorList（GET /api/v1/indicators）。
 * - 关键词搜索 + 服务端分页 + 已选面板。
 * - 设备类型由调用方传入（按选定设备 technology 推断）。
 */

import { useMemo, useState } from 'react';
import { useIntl } from 'react-intl';
import {
  Modal,
  Input,
  Table,
  Space,
  Button,
  Tag,
  Tooltip,
  Typography,
  Pagination,
  Divider,
  Select,
  theme,
  App,
} from 'antd';
import { SearchOutlined, ClearOutlined, ImportOutlined, CloseOutlined } from '@ant-design/icons';
import { useAllIndicators, useIndicatorList } from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import { useAppStore } from '@core/store/appStore';
import { formatIndicatorLevel, shouldShowIndicatorLevel } from '@core/utils/indicatorLevelDisplay';
import MetricBatchInputModal from './MetricBatchInputModal';
import {
  formatMetricIdSamples,
  type MetricBatchSelectionResult,
} from './metricBatchSelection';
const { Text } = Typography;

// 选中值 = 该指标在 pm_metrics 里的 metric_path：
//   原始计数指标落库编号化后（路线一·落库即编号），counter 与 KPI 落库键统一为指标编号（id）。
//   这样选中值直接作为查询过滤条件即可命中，无需再做映射。
//   （历史：编号化前 counter 落库键曾是点分上报名 en_name，故此处旧实现对 counter 取 en_name；
//    P2/P3 落库改编号后随之改为统一取 id，否则按点分名查 counter 一行不出。）
function metricValueOf(r: IndicatorInfo): string {
  return r.id;
}

// 选中标签的友好显示名：英文态缺英文名时显示编号，避免英文界面混入中文名；
// 中文态仍可回退英文名。
function metricLabelOf(r: IndicatorInfo, en: boolean): string {
  return en ? r.enName || r.id : r.cnName || r.enName || r.id;
}

function formatMetricDisplay(id: string, label?: string): string {
  return label && label !== id ? `${id} ${label}` : id;
}

interface MetricPickerModalProps {
  open: boolean;
  onClose: () => void;
  // labels: 选中值 → 友好名（KPI 友好名带 PLMN 标记）。调用方可用于摘要展示，不需要时可忽略。
  onConfirm: (selectedPaths: string[], labels: Record<string, string>) => void;
  initialSelected?: string[];
  /** 初始友好名映射（调用方从指标库预加载）。弹窗首次打开时合并进 labelMap，
   *  避免预选但未出现在当前页的指标显示原始编号。 */
  initialLabels?: Record<string, string>;
  initialDeviceType?: DeviceType;
  // 制式联动锁定（T-0188）：true 时隐藏内部「设备类型」下拉，deviceType 固定为
  // initialDeviceType 不可手动切换；不传/false 保持原下拉可切换行为（向后兼容 KPIQuery）。
  lockDeviceType?: boolean;
  maxSelected?: number;
  maxSelectedMessageId?: string;
  enableBatchInput?: boolean;
  onlyEnabledIndicators?: boolean;
}

const DEVICE_TYPE_OPTIONS: { label: string; value: DeviceType }[] = [
  { label: 'eNB (LTE)', value: 'ENB' },
  { label: 'gNB (5G NR)', value: 'GNB' },
  { label: 'GSM (2G)', value: 'GSM' },
];

export default function MetricPickerModal({
  open,
  onClose,
  onConfirm,
  initialSelected = [],
  initialLabels,
  initialDeviceType = 'ENB',
  lockDeviceType = false,
  maxSelected,
  maxSelectedMessageId = 'perf.picker.metricLimitExceeded',
  enableBatchInput = false,
  onlyEnabledIndicators = false,
}: MetricPickerModalProps) {
  const intl = useIntl();
  const { message } = App.useApp();
  const { token } = theme.useToken();
  const isEn = useAppStore((s) => s.locale) === 'en-US';
  // 非锁定态：用户可在弹窗内自行切换设备类型（KPIQuery 用法），用内部 state。
  const [deviceTypeState, setDeviceType] = useState<DeviceType>(initialDeviceType);
  const [keyword, setKeyword] = useState('');
  const [keywordDraft, setKeywordDraft] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selected, setSelected] = useState<string[]>(initialSelected);
  const [pasteOpen, setPasteOpen] = useState(false);
  // initialLabels 预加载的指标名 map；声明须在 prevOpen 块之前（该块内调用 setLabelMap）。
  const [labelMap, setLabelMap] = useState<Record<string, string>>(initialLabels ?? {});

  // 已选回显同步：与下方制式同款「组件常驻不卸载」问题——内部 selected 仅首挂载赋值一次，
  // 关闭后用新 initialSelected 重开（如切换编辑不同模板）时不会自动刷新，会残留上次打开的选择。
  // 故用渲染期幂等同步（与 labelMap/seenItems 同套路、不用 useEffect）：监听 open 由关变开，
  // 把 selected 重置为最新入参；open 期间用户的勾选不受影响（仅在打开瞬间同步一次）。
  const [prevOpen, setPrevOpen] = useState(open);
  if (prevOpen !== open) {
    setPrevOpen(open);
    if (open) {
      setSelected(initialSelected);
      setKeyword('');
      setKeywordDraft('');
      setPage(1);
      setPasteOpen(false);
      setLabelMap(initialLabels ?? {});
    }
  }

  // 制式联动锁定（T-0188）：锁定态 deviceType 恒等于外部制式入参 initialDeviceType。
  // 本组件在调用页常驻不卸载（Modal 的 destroyOnHidden 只销毁弹窗 DOM 内容、不重挂载本组件），
  // 内部 state 仅首挂载赋值一次、不随外部制式切换更新；故锁定态直接取 prop，避免停留在首挂载制式。
  const deviceType = lockDeviceType ? initialDeviceType : deviceTypeState;

  const { data, isLoading } = useIndicatorList(deviceType, {
    keyword: keyword || undefined,
    isEnabled: onlyEnabledIndicators ? true : undefined,
    page,
    pageSize,
  });
  const { data: allIndicatorsData, isLoading: isAllIndicatorsLoading } = useAllIndicators(deviceType, {
    enabled: open && (enableBatchInput || onlyEnabledIndicators),
    isEnabled: onlyEnabledIndicators ? true : undefined,
  });

  const items = useMemo(() => data?.items ?? [], [data]);
  const allItems = useMemo(() => allIndicatorsData?.items ?? [], [allIndicatorsData]);
  const allItemById = useMemo(
    () => new Map(allItems.map((item) => [metricValueOf(item), item])),
    [allItems],
  );
  const total = data?.total ?? 0;

  // 选中值 → 友好名 映射：items 变化时在渲染期幂等累积（与 PivotTable 列宽同款 render-phase sync，
  // 不用 useEffect），供「已选」面板标签显示友好名，避免露出 K 编号。
  // 初始值：优先用调用方预加载的 initialLabels（指标库全量名），否则为空 map。
  // （声明已提前至 prevOpen 块之前，此处保留注释供追溯）

  // initialLabels 异步加载完成后（调用方 useAllIndicators 返回数据），同步更新 labelMap。
  // render-phase sync：与 seenItems 同套路，幂等、不依赖 useEffect。
  // 优先级：用户在本次弹窗内看到并确认的名称（已在 labelMap 中）优先级高于 initialLabels。
  const [seenInitialLabels, setSeenInitialLabels] = useState(initialLabels);
  if (seenInitialLabels !== initialLabels) {
    setSeenInitialLabels(initialLabels);
    if (initialLabels && Object.keys(initialLabels).length > 0) {
      // initialLabels 作为底层兜底，prev 中已有的条目（用户本次确认的）不被覆盖。
      setLabelMap((prev) => ({ ...initialLabels, ...prev }));
    }
  }

  const [seenItems, setSeenItems] = useState(items);
  // 语言切换时也要刷新已选标签名（pm-name-i18n）：把 isEn 并入 sync 触发条件。
  const [seenIsEn, setSeenIsEn] = useState(isEn);
  if (seenItems !== items || seenIsEn !== isEn) {
    setSeenItems(items);
    setSeenIsEn(isEn);
    if (items.length > 0) {
      setLabelMap((prev) => {
        const next = { ...prev };
        for (const it of items) next[metricValueOf(it)] = metricLabelOf(it, isEn);
        return next;
      });
    }
  }

  const columns = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'perf.picker.colCode' }),
        dataIndex: 'id',
        key: 'code',
        width: 220,
        ellipsis: true,
        render: (v: string) => <Text code>{v}</Text>,
      },
      {
        title: intl.formatMessage({ id: 'perf.picker.colName' }),
        key: 'name',
        width: 280,
        ellipsis: true,
        render: (_: unknown, r: IndicatorInfo) => metricLabelOf(r, isEn),
      },
      {
        title: intl.formatMessage({ id: 'perf.picker.colType' }),
        key: 'type',
        width: 80,
        render: (_: unknown, r: IndicatorInfo) =>
          r.isCounter ? <Tag color="blue">Counter</Tag> : <Tag color="orange">KPI</Tag>,
      },
      ...(shouldShowIndicatorLevel(deviceType)
        ? [
            {
              title: intl.formatMessage({ id: 'perf.picker.colIndicatorLevel' }),
              key: 'indicatorLevel',
              dataIndex: 'indicatorLevel',
              width: 130,
              render: (v: string | undefined) => formatIndicatorLevel(v, (id) => intl.formatMessage({ id })),
            },
          ]
        : []),
    ],
    [intl, isEn, deviceType],
  );

  const handleSearch = () => {
    setKeyword(keywordDraft.trim());
    setPage(1);
  };

  const handleBatchApply = (result: MetricBatchSelectionResult) => {
    setSelected(result.nextSelected);
    setLabelMap((prev) => {
      const next = { ...prev };
      for (const id of result.validIds) {
        const item = allItemById.get(id);
        if (item) next[id] = metricLabelOf(item, isEn);
      }
      return next;
    });
    setPasteOpen(false);
    message.success(
      intl.formatMessage(
        { id: 'perf.metricBatchInput.importSuccess' },
        { added: result.addedIds.length, total: result.nextSelected.length },
      ),
    );
    if (result.invalidIds.length > 0) {
      message.warning(
        intl.formatMessage(
          { id: 'perf.metricBatchInput.notFound' },
          { count: result.invalidIds.length, ids: formatMetricIdSamples(result.invalidIds) },
        ),
      );
    }
    if (result.omittedValidIds.length > 0 && maxSelected !== undefined) {
      message.warning(
        intl.formatMessage(
          { id: 'perf.metricBatchInput.truncated' },
          { max: maxSelected, count: result.omittedValidIds.length },
        ),
      );
    }
  };

  const warnIfTooManySelected = (count: number) => {
    if (maxSelected === undefined || count <= maxSelected) return false;
    message.warning(
      intl.formatMessage(
        { id: maxSelectedMessageId },
        { max: maxSelected, count },
      ),
    );
    return true;
  };

  const handleConfirm = () => {
    if (warnIfTooManySelected(selected.length)) return;
    if (onlyEnabledIndicators) {
      const unavailableIds = selected.filter((id) => !allItemById.has(id));
      if (unavailableIds.length > 0) {
        message.warning(
          intl.formatMessage(
            { id: 'perf.picker.enabledOnlyUnavailable' },
            { count: unavailableIds.length, ids: formatMetricIdSamples(unavailableIds) },
          ),
        );
        return;
      }
    }
    const labels: Record<string, string> = {};
    selected.forEach((v) => {
      labels[v] = labelMap[v] ?? v;
    });
    onConfirm(selected, labels);
    onClose();
  };

  const rowSelection = {
    selectedRowKeys: selected,
    preserveSelectedRowKeys: true,
    onChange: (keys: React.Key[]) => {
      if (warnIfTooManySelected(keys.length)) return;
      const nextSelected = keys as string[];
      const nextSelectedSet = new Set(nextSelected);
      setSelected(nextSelected);
      setLabelMap((prev) => {
        const next = { ...prev };
        for (const item of items) {
          const id = metricValueOf(item);
          if (nextSelectedSet.has(id)) next[id] = metricLabelOf(item, isEn);
        }
        return next;
      });
    },
  };

  return (
    <>
      <Modal
        title={intl.formatMessage({ id: 'perf.picker.metricTitle' }, { count: selected.length })}
        open={open}
        onCancel={onClose}
        onOk={handleConfirm}
        okText={intl.formatMessage({ id: 'common.confirm' })}
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        confirmLoading={onlyEnabledIndicators && isAllIndicatorsLoading}
        okButtonProps={{
          disabled: onlyEnabledIndicators && isAllIndicatorsLoading,
        }}
        width={920}
        destroyOnHidden
      >
        <Space orientation="vertical" style={{ width: '100%' }} size="middle">
          <Space>
            {lockDeviceType ? null : (
              <>
                <Text>{intl.formatMessage({ id: 'perf.picker.deviceTypeLabel' })}</Text>
                <Select<DeviceType>
                  value={deviceType}
                  onChange={(v) => {
                    setDeviceType(v);
                    setPage(1);
                  }}
                  options={DEVICE_TYPE_OPTIONS}
                  style={{ width: 140 }}
                />
              </>
            )}
            <Input
              placeholder={intl.formatMessage({ id: 'perf.picker.searchMetricPlaceholder' })}
              prefix={<SearchOutlined />}
              value={keywordDraft}
              onChange={(e) => setKeywordDraft(e.target.value)}
              onPressEnter={handleSearch}
              style={{ width: 280 }}
              allowClear
              onClear={() => {
                setKeywordDraft('');
                setKeyword('');
                setPage(1);
              }}
            />
            <Button onClick={handleSearch} type="primary">
              {intl.formatMessage({ id: 'common.search' })}
            </Button>
            {enableBatchInput ? (
              <Button
                icon={<ImportOutlined />}
                onClick={() => setPasteOpen(true)}
                loading={isAllIndicatorsLoading}
              >
                {intl.formatMessage({ id: 'perf.metricBatchInput.title' })}
              </Button>
            ) : null}
          </Space>

          <Table<IndicatorInfo>
            rowKey={metricValueOf}
            size="small"
            loading={isLoading}
            columns={columns}
            dataSource={items}
            rowSelection={rowSelection}
            pagination={false}
            scroll={{ x: shouldShowIndicatorLevel(deviceType) ? 760 : undefined, y: 320 }}
          />

          <Pagination
            current={page}
            pageSize={pageSize}
            total={total}
            showSizeChanger
            showTotal={(t) => intl.formatMessage({ id: 'perf.picker.totalCount' }, { total: t })}
            onChange={(p, s) => {
              setPage(p);
              setPageSize(s);
            }}
          />

          <Divider style={{ margin: '4px 0' }} />

          <div>
            <Space style={{ marginBottom: 6 }}>
              <Text strong>{intl.formatMessage({ id: 'perf.picker.selectedMetric' })}</Text>
              <Button
                size="small"
                icon={<ClearOutlined />}
                onClick={() => setSelected([])}
                disabled={selected.length === 0}
              >
                {intl.formatMessage({ id: 'common.clear' })}
              </Button>
            </Space>
            <div
              style={{
                minHeight: 40,
                background: token.colorFillAlter,
                border: `1px solid ${token.colorBorderSecondary}`,
                padding: '6px 8px',
                borderRadius: token.borderRadiusSM,
              }}
            >
              {selected.length === 0 ? (
                <Text type="secondary" style={{ lineHeight: '28px' }}>
                  {intl.formatMessage({ id: 'perf.picker.noSelectedMetric' })}
                </Text>
              ) : (
                <div style={{ maxHeight: 160, overflowY: 'auto' }}>
                  {selected.map((path) => {
                    const name = labelMap[path];
                    const hasName = Boolean(name && name !== path);
                    return (
                      <div
                        key={path}
                        title={formatMetricDisplay(path, name)}
                        aria-label={formatMetricDisplay(path, name)}
                        style={{
                          display: 'flex',
                          alignItems: 'flex-start',
                          gap: 8,
                          minHeight: 30,
                          padding: '3px 0',
                          borderBottom: `1px solid ${token.colorBorderSecondary}`,
                        }}
                      >
                        <Text code style={{ flex: '0 0 116px', lineHeight: '24px' }}>
                          {path}
                        </Text>
                        {hasName ? (
                          <Text style={{ flex: 1, minWidth: 0, lineHeight: '24px', wordBreak: 'break-word' }}>
                            {name}
                          </Text>
                        ) : null}
                        <Tooltip title={intl.formatMessage({ id: 'common.delete' })}>
                          <Button
                            type="text"
                            size="small"
                            icon={<CloseOutlined />}
                            onClick={() => setSelected(selected.filter((p) => p !== path))}
                            style={{ flex: '0 0 auto' }}
                          />
                        </Tooltip>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </Space>
      </Modal>

      {enableBatchInput ? (
        <MetricBatchInputModal
          open={pasteOpen}
          candidates={allItems}
          currentSelected={selected}
          maxSelected={maxSelected}
          loading={isAllIndicatorsLoading}
          onApply={handleBatchApply}
          onCancel={() => setPasteOpen(false)}
        />
      ) : null}
    </>
  );
}
