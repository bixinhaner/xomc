/**
 * T-0193 下钻选择器：两层结构「设备 → 该设备的小区+PLMN 勾选」。
 *
 * 语义：
 *   - 每台设备一个折叠面板，展开后列出该设备实际出现过的「小区+PLMN」（来自列小区接口）。
 *   - 默认全勾 = 该设备全部小区（= 不过滤，向后兼容）。
 *     #241 后 5G 默认只勾 Type=gNB 与 Type=Cell+PLMNID；用户仍可手动全选查全部 job。
 *   - value 是 Record<deviceSn, objectLdn[]>：承载用户在某设备下的显式勾选。
 *     某设备缺席 = 该设备未动过（使用共享默认勾选规则）。
 *   - label 直接显示原始 object_ldn，避免格式化后 CU/DU/NBRCID 等字段丢失导致重名。
 *
 * 输出最终白名单走 cellDrilldownUtils.getEffectiveLdns（拍平+去重，全选设备不贡献过滤项）。
 * 列小区接口本身不返回 device_sn，故按设备分别取（useMetricObjectsByDevices）。
 */

import { useIntl } from 'react-intl';
import { Alert, Checkbox, Collapse, Empty, Space, Spin, Tag, Typography } from 'antd';
import { useMetricObjectsByDevices } from '@core/hooks/api/usePmQuery';
import type { MetricObjectsTimeRange } from '@core/services/api/pmObjectsApi';
import { getNrRecommendedDefaultSelectedObjectLdns, type CellSelection } from './cellDrilldownUtils';

interface CellDrilldownSelectorProps {
  deviceSns: string[];
  technology?: string;
  timeRange?: MetricObjectsTimeRange;
  useNrRecommendedDefault?: boolean;
  value: CellSelection;
  onChange: (next: CellSelection) => void;
}

export default function CellDrilldownSelector({
  deviceSns,
  technology,
  timeRange,
  useNrRecommendedDefault = false,
  value,
  onChange,
}: CellDrilldownSelectorProps) {
  const intl = useIntl();
  const { byDevice, isLoading } = useMetricObjectsByDevices(deviceSns, technology, timeRange);

  if (deviceSns.length === 0) {
    return (
      <Alert
        type="info"
        showIcon
        title={intl.formatMessage({ id: 'perf.drilldown.pickDeviceFirst' })}
      />
    );
  }

  const items = deviceSns.map((sn) => {
    const objects = byDevice[sn] ?? [];
    const allLdns = objects.map((o) => o.objectLdn);
    const sel = value[sn];
    // sel===undefined → 默认全选；#241 指定入口的 5G 建议默认态使用推荐子集。
    const checkedLdns = sel ?? (
      useNrRecommendedDefault ? getNrRecommendedDefaultSelectedObjectLdns(objects) : allLdns
    );
    const allChecked = allLdns.length > 0 && checkedLdns.length >= allLdns.length;
    const indeterminate = checkedLdns.length > 0 && !allChecked;

    const allSummary = intl.formatMessage({
      id: useNrRecommendedDefault ? 'perf.drilldown.headerAllObjects' : 'perf.drilldown.headerAll',
    });
    const summary =
      checkedLdns.length === 0 || checkedLdns.length >= allLdns.length
        ? allSummary
        : intl.formatMessage({
          id: useNrRecommendedDefault ? 'perf.drilldown.headerSubsetObjects' : 'perf.drilldown.headerSubset',
        }, { count: checkedLdns.length });

    const body =
      isLoading && objects.length === 0 ? (
        <Spin size="small" />
      ) : objects.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={intl.formatMessage({ id: 'perf.drilldown.noCells' })}
        />
      ) : (
        <Space orientation="vertical" size={4} style={{ width: '100%' }}>
          <Checkbox
            checked={allChecked}
            indeterminate={indeterminate}
            onChange={(e) => onChange({ ...value, [sn]: e.target.checked ? [...allLdns] : [] })}
          >
            <Typography.Text strong>
              {intl.formatMessage({ id: 'perf.drilldown.selectAll' })}
            </Typography.Text>
          </Checkbox>
          <Checkbox.Group
            value={checkedLdns}
            onChange={(vals) => onChange({ ...value, [sn]: vals as string[] })}
            style={{ display: 'flex', flexDirection: 'column', gap: 4, paddingLeft: 8 }}
            options={objects.map((o) => ({
              label: o.objectLdn,
              value: o.objectLdn,
            }))}
          />
        </Space>
      );

    return {
      key: sn,
      label: (
        <Space>
          <Typography.Text>{sn}</Typography.Text>
          <Tag color={summary === allSummary ? 'default' : 'blue'}>
            {summary}
          </Tag>
        </Space>
      ),
      children: body,
    };
  });

  return (
    <Space orientation="vertical" size="small" style={{ width: '100%' }}>
      <Alert
        type="info"
        showIcon
        title={intl.formatMessage({
          id: useNrRecommendedDefault ? 'perf.drilldown.recommendedHint' : 'perf.drilldown.hint',
        })}
      />
      <Collapse items={items} />
    </Space>
  );
}
