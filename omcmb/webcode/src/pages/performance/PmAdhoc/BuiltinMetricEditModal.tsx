/**
 * T-0194：内置任务「编辑指标」轻量弹窗。
 *
 * 仅含指标穿梭框（预填当前 metric_paths、按任务制式加载候选、含 all/kpi/counter 类型筛选），
 * 与向导第 3 步同口径。提交只改 metric_paths（调 PATCH，后端守门只取 metric_paths）。
 * 不涉及设备/粒度/时窗——内置任务结构性字段不可改。
 */

import { useMemo, useState, useEffect } from 'react';
import { useIntl } from 'react-intl';
import { Alert, Modal, Select, Space, Spin, Tag, Transfer, message } from 'antd';
import { useUpdatePmAdhoc } from '@core/hooks/api/usePmAdhoc';
import { useIndicatorCandidates } from '@core/hooks/api/usePerformance';
import type { IndicatorCandidate } from '@core/services/api/pmApi';
import type { AdhocTask } from '@core/types/pmAdhoc';
import type { DeviceType } from '@core/types/indicatorLibrary';

// 制式 → 指标库 deviceType（与向导一致）。
const TECH_TO_DEVICE_TYPE: Record<string, DeviceType> = {
  lte: 'ENB',
  nr: 'GNB',
  gsm: 'GSM',
};

interface MetricTransferItem {
  key: string;
  id: string;
  name: string;
  isCounter: boolean;
  searchText: string;
}

interface Props {
  task: AdhocTask | null;
  open: boolean;
  onClose: () => void;
}

export default function BuiltinMetricEditModal({ task, open, onClose }: Props) {
  const intl = useIntl();
  const updateMut = useUpdatePmAdhoc();

  const [metricPaths, setMetricPaths] = useState<string[]>([]);
  const [metricTypeFilter, setMetricTypeFilter] = useState<'all' | 'kpi' | 'counter'>('all');

  // 打开/切换任务时预填当前指标集。
  useEffect(() => {
    if (open && task) {
      setMetricPaths([...task.metricPaths]);
      setMetricTypeFilter('all');
    }
  }, [open, task]);

  // 内置任务 technology 决定候选 deviceType；无制式（理论上内置都有）回退 ENB。
  const deviceType = TECH_TO_DEVICE_TYPE[task?.technology ?? ''] ?? 'ENB';
  const { data: candidates, isLoading } = useIndicatorCandidates(deviceType, {
    includeCounters: true,
  });
  const indicatorItems = useMemo<IndicatorCandidate[]>(() => candidates ?? [], [candidates]);

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
          searchText: `${ind.id} ${ind.name} ${ind.cnName}`.toLowerCase(),
        })),
    [indicatorItems, metricTypeFilter, selectedKeySet],
  );

  const renderItem = (item: MetricTransferItem) => (
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

  const handleOk = async () => {
    if (!task) return;
    if (metricPaths.length < 1) {
      message.error(intl.formatMessage({ id: 'perf.adhoc.editMetricEmpty' }));
      return;
    }
    try {
      await updateMut.mutateAsync({ id: task.id, input: { metricPaths } });
      message.success(intl.formatMessage({ id: 'perf.adhoc.editMetricSaved' }));
      onClose();
    } catch (e) {
      message.error(
        intl.formatMessage({ id: 'perf.adhoc.editMetricFailed' }, { msg: (e as Error).message }),
      );
    }
  };

  return (
    <Modal
      title={
        task
          ? intl.formatMessage({ id: 'perf.adhoc.editMetricTitle' }, { name: task.name })
          : intl.formatMessage({ id: 'perf.adhoc.editMetricTitleDefault' })
      }
      open={open}
      onCancel={onClose}
      onOk={handleOk}
      confirmLoading={updateMut.isPending}
      width={820}
      destroyOnClose
    >
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Alert
          type="info"
          showIcon
          message={intl.formatMessage(
            { id: 'perf.adhoc.editMetricHint' },
            { tech: (task?.technology ?? '').toUpperCase(), deviceType },
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
        <Spin spinning={isLoading}>
          <Transfer<MetricTransferItem>
            dataSource={transferItems}
            targetKeys={metricPaths}
            onChange={(keys: React.Key[]) => setMetricPaths(keys.map(String))}
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
            listStyle={{ width: 360, height: 380 }}
          />
        </Spin>
        <div style={{ color: '#888' }}>
          {intl.formatMessage({ id: 'perf.adhoc.metricSelectedCount' }, { count: metricPaths.length })}
        </div>
      </Space>
    </Modal>
  );
}
