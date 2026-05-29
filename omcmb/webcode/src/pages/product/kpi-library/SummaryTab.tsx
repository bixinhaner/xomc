/**
 * SummaryTab — T-0180 P4 一级页面三制式聚合表。
 *
 * 数据来源:GET /indicators/summary,后端 PgFileRepository.SummaryByTech 单次扫表。
 * 每行包含:制式 / 指标数 / 分组数 / "M 内置 / N 自定义"(loaded_from 前缀分类)/ 平台 distinct。
 *
 * 点击"制式名"或"详情"按钮 → onSelect 触发 drill-down(由 index.tsx 同步 URL ?tech=)。
 */
import { Card, Table, Tag, Button, Space } from 'antd';
import type { IndicatorTechSummary, TechLower } from '@core/types/indicatorLibrary';
import { useIndicatorSummary } from '@core/hooks/api/useIndicatorsLibrary';

interface Props {
  onSelect: (tech: TechLower) => void;
}

const TECH_LABEL: Record<TechLower, string> = {
  enb: 'ENB (LTE)',
  gsm: 'GSM',
  gnb: 'GNB (5G NR)',
};

export default function SummaryTab({ onSelect }: Props) {
  const { data, isLoading } = useIndicatorSummary();
  const items = data?.items || [];

  const columns = [
    {
      title: '制式',
      dataIndex: 'tech',
      width: 140,
      render: (v: TechLower) => (
        <Button
          type="link"
          size="small"
          onClick={() => onSelect(v)}
          style={{ padding: 0, fontWeight: 600 }}
        >
          {TECH_LABEL[v]}
        </Button>
      ),
    },
    { title: '指标数', dataIndex: 'indicators', width: 100 },
    { title: '分组数', dataIndex: 'groups', width: 100 },
    {
      title: 'XML 文件',
      width: 180,
      render: (_: unknown, row: IndicatorTechSummary) => (
        <Space size={4}>
          <Tag color="default">{row.builtinCount} 内置</Tag>
          <Tag color={row.customCount > 0 ? 'blue' : 'default'}>{row.customCount} 自定义</Tag>
          {row.unknownCount > 0 && <Tag color="warning">{row.unknownCount} 未知</Tag>}
        </Space>
      ),
    },
    {
      title: '平台',
      dataIndex: 'platforms',
      ellipsis: true,
      render: (platforms: string[]) =>
        platforms.length > 0 ? (
          <Space size={4} wrap>
            {platforms.map((p) => (
              <Tag key={p}>{p}</Tag>
            ))}
          </Space>
        ) : (
          <span>—</span>
        ),
    },
    {
      title: '操作',
      width: 100,
      render: (_: unknown, row: IndicatorTechSummary) => (
        <Button size="small" type="link" onClick={() => onSelect(row.tech)}>
          详情
        </Button>
      ),
    },
  ];

  return (
    <Card size="small">
      <Table<IndicatorTechSummary>
        rowKey="tech"
        loading={isLoading}
        columns={columns}
        dataSource={items}
        size="small"
        pagination={false}
      />
    </Card>
  );
}
