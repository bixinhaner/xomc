/**
 * T-0194：内置任务「编辑指标」页面内编辑层。
 *
 * 仅含指标穿梭框（预填当前 metric_paths、按任务制式加载候选、含 all/kpi/counter 类型筛选），
 * 与向导第 3 步同口径。提交只改 metric_paths（调 PUT，后端守门只取 metric_paths）。
 * 不涉及设备/粒度/时窗——内置任务结构性字段不可改。
 */

import type { CSSProperties } from 'react';
import { useMemo, useState, useEffect } from 'react';
import { useIntl } from 'react-intl';
import { ImportOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Select, Space, Spin, Tag, Tooltip, Transfer, message } from 'antd';
import { PM_QUERY_SELECTION_LIMIT } from '@/constants/pmQueryLimits';
import { useUpdatePmAdhoc } from '@core/hooks/api/usePmAdhoc';
import { useIndicatorCandidates } from '@core/hooks/api/usePerformance';
import type { IndicatorCandidate } from '@core/services/api/pmApi';
import type { AdhocTask } from '@core/types/pmAdhoc';
import { formatIndicatorLevel, shouldShowIndicatorLevel } from '@core/utils/indicatorLevelDisplay';
import {
  isKnownTechnology,
  technologyToDeviceType,
  useTechnologyDictionary,
} from '@core/hooks/api/useTechnologyDictionary';
import {
  MetricBatchInputModal,
  formatMetricIdSamples,
  type MetricBatchSelectionResult,
} from '@/components/MetricPickerModal';
import { resolveLimitedTransferSelection } from './selectionLimit';
import { displayAdhocTaskName } from '../adhocTaskDisplay';
import {
  clearBuiltinMetricDraft,
  getBuiltinMetricDraft,
  saveBuiltinMetricDraft,
} from './pmAdhocDraftState';

const METRIC_TRANSFER_SECTION_STYLE: CSSProperties = {
  flex: '1 1 0',
  minWidth: 0,
  height: 480,
};
const METRIC_TRANSFER_LIST_STYLE: CSSProperties = {
  overflowX: 'auto',
};
const METRIC_TRANSFER_ITEM_STYLE: CSSProperties = {
  minWidth: '100%',
  width: 'max-content',
};
const METRIC_TRANSFER_ITEM_CONTENT_STYLE: CSSProperties = {
  flex: '0 0 auto',
  minWidth: 'max-content',
  overflow: 'visible',
  textOverflow: 'clip',
  whiteSpace: 'nowrap',
};
const METRIC_ROW_STYLE: CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 6,
  minWidth: '100%',
  width: 'max-content',
};
const METRIC_TEXT_STYLE: CSSProperties = {
  flex: '0 0 auto',
  whiteSpace: 'nowrap',
};

interface MetricTransferItem {
  key: string;
  id: string;
  name: string;
  isCounter: boolean;
  indicatorLevel?: string;
  searchText: string;
}

interface Props {
  task: AdhocTask | null;
  open: boolean;
  onClose: () => void;
}

export default function BuiltinMetricEditModal({ task, open, onClose }: Props) {
  const intl = useIntl();
  const { labelForTechnology } = useTechnologyDictionary();
  const updateMut = useUpdatePmAdhoc();

  const [metricPaths, setMetricPaths] = useState<string[]>([]);
  const [metricTypeFilter, setMetricTypeFilter] = useState<'all' | 'kpi' | 'counter'>('all');
  const [metricBatchOpen, setMetricBatchOpen] = useState(false);
  const [hydratedTaskId, setHydratedTaskId] = useState<string | null>(null);
  const [restoredFromDraft, setRestoredFromDraft] = useState(false);

  // 打开/切换任务时预填当前指标集。
  useEffect(() => {
    if (open && task) {
      const restoredDraft = getBuiltinMetricDraft(task.id);
      setMetricPaths(restoredDraft ? restoredDraft.metricPaths : [...task.metricPaths]);
      setMetricTypeFilter(restoredDraft?.metricTypeFilter ?? 'all');
      setMetricBatchOpen(false);
      setHydratedTaskId(task.id);
      setRestoredFromDraft(Boolean(restoredDraft));
      return;
    }
    setHydratedTaskId(null);
    setRestoredFromDraft(false);
  }, [open, task]);

  useEffect(() => {
    if (!open || !task || hydratedTaskId !== task.id) return;
    saveBuiltinMetricDraft({
      taskId: task.id,
      metricPaths,
      metricTypeFilter,
    });
  }, [open, task, hydratedTaskId, metricPaths, metricTypeFilter]);

  // 内置任务 technology 决定候选 deviceType；无制式（理论上内置都有）回退 ENB。
  const taskTechnology = task?.technology;
  const deviceType = isKnownTechnology(taskTechnology)
    ? technologyToDeviceType(taskTechnology)
    : 'ENB';
  const { data: candidates, isLoading } = useIndicatorCandidates(deviceType, {
    includeCounters: true,
    enabledOnly: true,
  });
  const indicatorItems = useMemo<IndicatorCandidate[]>(() => candidates ?? [], [candidates]);
  const indicatorIdSet = useMemo(
    () => new Set(indicatorItems.map((indicator) => indicator.id)),
    [indicatorItems],
  );

  useEffect(() => {
    if (!open || !task || !restoredFromDraft || isLoading) return;
    const nextMetricPaths = metricPaths.filter((metricPath) => indicatorIdSet.has(metricPath));
    if (nextMetricPaths.length === metricPaths.length) return;
    setMetricPaths(nextMetricPaths);
  }, [indicatorIdSet, isLoading, metricPaths, open, restoredFromDraft, task]);

  const selectedKeySet = useMemo(() => new Set(metricPaths), [metricPaths]);
  const transferItems = useMemo<MetricTransferItem[]>(
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
          isCounter: ind.isCounter,
          indicatorLevel: ind.indicatorLevel,
          searchText: `${ind.id} ${ind.name} ${ind.cnName}`.toLowerCase(),
        })),
    [indicatorItems, metricTypeFilter, selectedKeySet],
  );

  const renderItem = (item: MetricTransferItem) => {
    const typeLabel = item.isCounter
      ? intl.formatMessage({ id: 'perf.adhoc.metricTagCounter' })
      : intl.formatMessage({ id: 'perf.adhoc.metricTagKpi' });
    const levelLabel = shouldShowIndicatorLevel(deviceType)
      ? formatIndicatorLevel(item.indicatorLevel, (id) => intl.formatMessage({ id }))
      : '';
    const tooltipTitle = [typeLabel, levelLabel, item.id, item.name].filter(Boolean).join(' ');

    return {
      label: (
        <Tooltip title={tooltipTitle} placement="topLeft">
          <span style={METRIC_ROW_STYLE}>
            <Tag color={item.isCounter ? 'blue' : 'orange'} style={{ flex: '0 0 auto', marginInlineEnd: 0 }}>
              {typeLabel}
            </Tag>
            {shouldShowIndicatorLevel(deviceType) && (
              <Tag color="default" style={{ flex: '0 0 auto', marginInlineEnd: 0 }}>
                {levelLabel}
              </Tag>
            )}
            <span style={{ flex: '0 0 auto', color: '#999' }}>{item.id}</span>
            <span style={METRIC_TEXT_STYLE}>{item.name}</span>
          </span>
        </Tooltip>
      ),
      value: tooltipTitle,
    };
  };

  const warnMetricLimitExceeded = (count: number) => {
    if (count <= PM_QUERY_SELECTION_LIMIT) return false;
    message.warning(intl.formatMessage(
      { id: 'perf.picker.metricLimitExceeded' },
      { max: PM_QUERY_SELECTION_LIMIT, count },
    ));
    return true;
  };

  const limitMetricKeys = (keys: React.Key[]) => {
    const result = resolveLimitedTransferSelection(metricPaths, keys, PM_QUERY_SELECTION_LIMIT);
    if (result.exceeded) warnMetricLimitExceeded(result.count);
    return result.next;
  };

  const handleOk = async () => {
    if (!task) return;
    if (metricPaths.length < 1) {
      message.error(intl.formatMessage({ id: 'perf.adhoc.editMetricEmpty' }));
      return;
    }
    if (warnMetricLimitExceeded(metricPaths.length)) return;
    try {
      await updateMut.mutateAsync({ id: task.id, input: { metricPaths } });
      message.success(intl.formatMessage({ id: 'perf.adhoc.editMetricSaved' }));
      clearBuiltinMetricDraft(task.id);
      onClose();
    } catch (e) {
      message.error(
        intl.formatMessage({ id: 'perf.adhoc.editMetricFailed' }, { msg: (e as Error).message }),
      );
    }
  };

  if (!open || !task) return null;

  return (
    <Card
      title={intl.formatMessage(
        { id: 'perf.adhoc.editMetricTitle' },
        { name: displayAdhocTaskName(task, labelForTechnology) },
      )}
      extra={
        <Space>
          <Button onClick={onClose}>
            {intl.formatMessage({ id: 'perf.adhoc.backToList' })}
          </Button>
          <Button type="primary" loading={updateMut.isPending} onClick={handleOk}>
            {intl.formatMessage({ id: 'perf.adhoc.btnSaveEdit' })}
          </Button>
        </Space>
      }
    >
      <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
        <Alert
          type="info"
          showIcon
          message={intl.formatMessage(
            { id: 'perf.adhoc.editMetricHint' },
            { tech: labelForTechnology(task?.technology), deviceType },
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
          <Button
            icon={<ImportOutlined />}
            onClick={() => setMetricBatchOpen(true)}
            disabled={isLoading}
          >
            {intl.formatMessage({ id: 'perf.metricBatchInput.title' })}
          </Button>
        </Space>
        <Spin spinning={isLoading}>
          <Transfer<MetricTransferItem>
            dataSource={transferItems}
            targetKeys={metricPaths}
            onChange={(keys: React.Key[]) => setMetricPaths(limitMetricKeys(keys))}
            render={renderItem}
            showSearch
            filterOption={(inputValue, item) => {
              const kw = inputValue.trim().toLowerCase();
              return kw === '' || item.searchText.includes(kw);
            }}
            titles={[
              intl.formatMessage({ id: 'perf.adhoc.transferAvailableMetric' }),
              intl.formatMessage({ id: 'perf.adhoc.transferSelectedMetric' }),
            ]}
            style={{ width: '100%' }}
            styles={{
              section: METRIC_TRANSFER_SECTION_STYLE,
              list: METRIC_TRANSFER_LIST_STYLE,
              item: METRIC_TRANSFER_ITEM_STYLE,
              itemContent: METRIC_TRANSFER_ITEM_CONTENT_STYLE,
            }}
          />
        </Spin>
        <div style={{ color: '#888' }}>
          {intl.formatMessage({ id: 'perf.adhoc.metricSelectedCount' }, { count: metricPaths.length })}
        </div>
        <MetricBatchInputModal
          open={metricBatchOpen}
          candidates={indicatorItems}
          currentSelected={metricPaths}
          maxSelected={PM_QUERY_SELECTION_LIMIT}
          loading={isLoading}
          onCancel={() => setMetricBatchOpen(false)}
          onApply={(result: MetricBatchSelectionResult) => {
            setMetricPaths(result.nextSelected);
            setMetricBatchOpen(false);
            message.success(intl.formatMessage(
              { id: 'perf.metricBatchInput.importSuccess' },
              { added: result.addedIds.length, total: result.nextSelected.length },
            ));
            if (result.invalidIds.length > 0) {
              message.warning(intl.formatMessage(
                { id: 'perf.metricBatchInput.notFound' },
                { count: result.invalidIds.length, ids: formatMetricIdSamples(result.invalidIds) },
              ));
            }
            if (result.limitExceeded) {
              message.warning(intl.formatMessage(
                { id: 'perf.metricBatchInput.truncated' },
                {
                  max: PM_QUERY_SELECTION_LIMIT,
                  count: result.omittedValidIds.length,
                },
              ));
            }
          }}
        />
      </Space>
    </Card>
  );
}
