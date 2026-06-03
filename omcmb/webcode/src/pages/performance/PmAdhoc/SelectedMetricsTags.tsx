/**
 * T-0194：任务详情「已选指标」平铺 Tag 列表。
 *
 * 按任务制式加载指标库候选（复用向导/结果面板的查法），把 metric_paths（K/C 编号）解析为
 * 「编号 + 中/英名」；查不到名时回退显编号（不报错、不留空）。数量与 metric_paths 一致。
 */

import { useMemo } from 'react';
import { useIntl } from 'react-intl';
import { Space, Spin, Tag, Typography } from 'antd';
import { useIndicatorCandidates } from '@core/hooks/api/usePerformance';
import type { IndicatorCandidate } from '@core/services/api/pmApi';
import type { DeviceType } from '@core/types/indicatorLibrary';

const TECH_TO_DEVICE_TYPE: Record<string, DeviceType> = {
  lte: 'ENB',
  nr: 'GNB',
  gsm: 'GSM',
};

interface Props {
  metricPaths: string[];
  technology?: string;
}

export default function SelectedMetricsTags({ metricPaths, technology }: Props) {
  const intl = useIntl();
  // 无制式（不限）时无法定位单一指标库 → 直接显编号（回退语义，不报错）。
  const deviceType = technology ? TECH_TO_DEVICE_TYPE[technology] : undefined;
  const { data: candidates, isLoading } = useIndicatorCandidates(deviceType, {
    includeCounters: true,
  });
  const nameById = useMemo(() => {
    const m = new Map<string, IndicatorCandidate>();
    for (const ind of candidates ?? []) m.set(ind.id, ind);
    return m;
  }, [candidates]);

  if (!metricPaths || metricPaths.length === 0) {
    return <Typography.Text type="secondary">—</Typography.Text>;
  }

  return (
    <Spin spinning={isLoading} size="small">
      <Space size={[4, 4]} wrap>
        {metricPaths.map((code) => {
          const ind = nameById.get(code);
          // 命中 → 「编号 + 中/英名」；缺失 → 回退显编号。
          const label = ind ? `${ind.id} ${ind.cnName || ind.name}`.trim() : code;
          return <Tag key={code}>{label}</Tag>;
        })}
      </Space>
    </Spin>
  );
}
