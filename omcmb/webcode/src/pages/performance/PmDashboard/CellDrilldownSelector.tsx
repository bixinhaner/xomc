/**
 * T-0193 下钻选择器：两层结构「设备 → 该设备的小区+PLMN 勾选」。
 *
 * 语义：
 *   - 每台设备一个折叠面板，展开后列出该设备实际出现过的「小区+PLMN」（来自列小区接口）。
 *   - 默认全勾 = 该设备全部小区（= 不过滤，向后兼容）。
 *   - value 是 Record<deviceSn, objectLdn[]>：承载用户在某设备下的显式勾选。
 *     某设备缺席 = 该设备未动过（默认全选，不过滤）。
 *   - label 直接显示原始 object_ldn，避免格式化后 CU/DU/NBRCID 等字段丢失导致重名。
 *
 * 输出最终白名单走 cellDrilldownUtils.getEffectiveLdns（拍平+去重，全选设备不贡献过滤项）。
 * 列小区接口本身不返回 device_sn，故按设备分别取（useMetricObjectsByDevices）。
 */

import { useIntl } from 'react-intl';
import { Alert, Checkbox, Collapse, Empty, Space, Spin, Tag, Typography } from 'antd';
import { useMetricObjectsByDevices } from '@core/hooks/api/usePmQuery';
import type { CellSelection } from './cellDrilldownUtils';

interface CellDrilldownSelectorProps {
  deviceSns: string[];
  technology?: string;
  value: CellSelection;
  onChange: (next: CellSelection) => void;
}

export default function CellDrilldownSelector({
  deviceSns,
  technology,
  value,
  onChange,
}: CellDrilldownSelectorProps) {
  const intl = useIntl();
  const { byDevice, isLoading } = useMetricObjectsByDevices(deviceSns, technology);

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
    // sel===undefined → 默认全选；否则用显式数组。
    const checkedLdns = sel ?? allLdns;
    const allChecked = allLdns.length > 0 && checkedLdns.length >= allLdns.length;
    const indeterminate = checkedLdns.length > 0 && !allChecked;

    const summary =
      sel === undefined || sel.length === 0 || sel.length >= allLdns.length
        ? intl.formatMessage({ id: 'perf.drilldown.headerAll' })
        : intl.formatMessage({ id: 'perf.drilldown.headerSubset' }, { count: sel.length });

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
          <Tag color={summary === intl.formatMessage({ id: 'perf.drilldown.headerAll' }) ? 'default' : 'blue'}>
            {summary}
          </Tag>
        </Space>
      ),
      children: body,
    };
  });

  return (
    <Space orientation="vertical" size="small" style={{ width: '100%' }}>
      <Alert type="info" showIcon title={intl.formatMessage({ id: 'perf.drilldown.hint' })} />
      <Collapse items={items} />
    </Space>
  );
}
