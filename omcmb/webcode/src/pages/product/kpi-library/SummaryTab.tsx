/**
 * SummaryTab — T-0180 一级 (制式, 平台) 聚合表(2026-05-29 粒度调整)。
 *
 * 用户决策:
 *   - 每个 (制式, 平台) 唯一一行(替代原"每制式一行")
 *   - 点击平台 → 进入 platform 的 KPI 详情页面(?tech=X&platform=Y URL)
 *   - 列顺序:制式 / 平台(可点) / XML 文件名 / 指标数 / 说明(最后,简洁)
 *   - 删除"操作"列(平台名 button.link 已可点击 drill-down)
 *   - 删除"分组数 / builtin/custom 计数 / 平台 distinct" 等冗余列
 */
import { Card, Table, Tag, Button, Tooltip } from 'antd';
import type { IndicatorPlatformSummary, TechLower } from '@core/types/indicatorLibrary';
import { useIndicatorSummary } from '@core/hooks/api/useIndicatorsLibrary';

interface Props {
  onSelect: (tech: TechLower, platform: string) => void;
}

const TECH_LABEL: Record<TechLower, string> = {
  enb: 'ENB (LTE)',
  gsm: 'GSM',
  gnb: 'GNB (5G NR)',
};

const TECH_DESC: Record<TechLower, string> = {
  enb: '4G LTE 基站(eNodeB)',
  gsm: '2G GSM 基站控制器(BSC)',
  gnb: '5G NR 基站(gNodeB)',
};

// 按 (loaded_from, platform) 拼一个文件名展示串(尾段 basename)
function basenameOf(loadedFrom: string): string {
  const idx = loadedFrom.lastIndexOf('/');
  return idx >= 0 ? loadedFrom.slice(idx + 1) : loadedFrom;
}

export default function SummaryTab({ onSelect }: Props) {
  const { data, isLoading } = useIndicatorSummary();
  const items = data?.items || [];

  const columns = [
    {
      title: '制式',
      dataIndex: 'tech',
      width: 140,
      render: (v: TechLower) => <Tag color="geekblue">{TECH_LABEL[v]}</Tag>,
    },
    {
      title: '平台',
      dataIndex: 'platform',
      width: 180,
      render: (v: string, row: IndicatorPlatformSummary) => (
        <Button
          type="link"
          size="small"
          onClick={() => onSelect(row.tech, v)}
          style={{ padding: 0, fontWeight: 600 }}
        >
          {v}
        </Button>
      ),
    },
    {
      title: 'XML文件',
      dataIndex: 'loadedFrom',
      width: 220,
      render: (v: string) => (
        <Tooltip title={v}>
          <Tag>{basenameOf(v)}</Tag>
        </Tooltip>
      ),
    },
    { title: '指标数', dataIndex: 'indicators', width: 100 },
    {
      title: '说明',
      ellipsis: true,
      render: (_: unknown, row: IndicatorPlatformSummary) => (
        <span style={{ color: 'rgba(0, 0, 0, 0.65)', fontSize: 12 }}>
          {TECH_DESC[row.tech]} · 平台 {row.platform}
        </span>
      ),
    },
  ];

  return (
    <Card size="small">
      <Table<IndicatorPlatformSummary>
        rowKey={(r) => `${r.tech}__${r.platform}__${r.loadedFrom}`}
        loading={isLoading}
        columns={columns}
        dataSource={items}
        size="small"
        pagination={false}
      />
    </Card>
  );
}
