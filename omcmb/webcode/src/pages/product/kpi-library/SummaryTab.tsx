/**
 * SummaryTab — T-0180 一级 (制式, 平台) 聚合表(2026-05-29 第四轮调整)。
 *
 * 本轮(用户决策,对齐 T-0178 param-model 范式):
 *   1. 新增"来源"列:Tag 内置 / 自定义(后端 source.go::ClassifySource 派生)
 *   2. "XML 文件"列改名"加载源",显示完整 loaded_from(原来只显 basename)
 *   3. 新增"操作"列:仅 deletable=true(custom)显红色删除;builtin 置灰 + Tooltip
 *      "内置 KPI XML 不可在线删除。如需移除,请联系管理员"
 *
 * 历史决策(沿用):
 *   - 列粒度:每个 (制式, 平台) 唯一一行
 *   - 平台名是 type=link Button,点击 → onSelect(tech, platform)
 */
import { Card, Table, Tag, Button, Tooltip, Popconfirm, message } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import type { AxiosError } from 'axios';
import type {
  IndicatorPlatformSummary,
  IndicatorSource,
  TechLower,
} from '@core/types/indicatorLibrary';
import {
  useIndicatorSummary,
  useIndicatorDeleteFile,
} from '@core/hooks/api/useIndicatorsLibrary';
import { useT } from '@/hooks/useT';

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

export default function SummaryTab({ onSelect }: Props) {
  const t = useT();
  const { data, isLoading } = useIndicatorSummary();
  const deleteMut = useIndicatorDeleteFile();
  const items = data?.items || [];

  const columns = [
    {
      title: '平台',
      dataIndex: 'platform',
      width: 160,
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
      title: '制式',
      dataIndex: 'tech',
      width: 130,
      render: (v: TechLower) => <Tag color="geekblue">{TECH_LABEL[v]}</Tag>,
    },
    {
      // 后端 source.go::ClassifySource 派生,前端只渲染
      title: '来源',
      dataIndex: 'source',
      width: 90,
      filters: [
        { text: '内置', value: 'builtin' as IndicatorSource },
        { text: '自定义', value: 'custom' as IndicatorSource },
      ],
      onFilter: (val: boolean | React.Key, row: IndicatorPlatformSummary) =>
        row.source === val,
      render: (s: IndicatorSource | undefined) =>
        s === 'custom' ? <Tag color="blue">{t('common.custom')}</Tag> : <Tag>{t('common.builtin')}</Tag>,
    },
    {
      // 2026-05-29:从"XML 文件"(只显 basename)改为"加载源"(全路径,ellipsis)
      title: '加载源',
      dataIndex: 'loadedFrom',
      width: 260,
      ellipsis: true,
      render: (v: string) => <Tooltip title={v}><code>{v}</code></Tooltip>,
    },
    { title: '指标数', dataIndex: 'indicators', width: 90 },
    {
      title: '说明',
      ellipsis: true,
      render: (_: unknown, row: IndicatorPlatformSummary) => (
        <span style={{ color: 'rgba(0, 0, 0, 0.65)', fontSize: 12 }}>
          {TECH_DESC[row.tech]} · 平台 {row.platform}
        </span>
      ),
    },
    {
      title: '操作',
      width: 80,
      render: (_: unknown, row: IndicatorPlatformSummary) =>
        row.deletable ? (
          <Popconfirm
            title={`确认删除自定义 KPI XML?`}
            description={
              <div style={{ maxWidth: 320 }}>
                · 文件 <code>{row.loadedFrom}</code> 将 rename 为
                <code>.deleted.&lt;ts&gt;</code> 备份
                <br />· 关联的 {row.indicators} 条指标 + 公式级联删除
                <br />· 若存在同名内置 XML,删除后将自动回退到内置版本
              </div>
            }
            okButtonProps={{ danger: true }}
            okText={t('product.paramModel.delConfirm')}
            onConfirm={() =>
              deleteMut
                .mutateAsync(row.loadedFrom)
                .then(() => message.success(t('common.deleted')))
                .catch((e) => {
                  const ax = e as AxiosError<{ msg?: string; message?: string }>;
                  const msg =
                    ax.response?.data?.msg ??
                    ax.response?.data?.message ??
                    (e instanceof Error ? e.message : String(e));
                  message.error(msg);
                })
            }
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        ) : (
          <Tooltip
            title={
              <div style={{ maxWidth: 240 }}>
                内置 KPI XML 不可在线删除。如需移除,请联系管理员
              </div>
            }
            placement="topRight"
          >
            <Button
              size="small"
              icon={<DeleteOutlined />}
              disabled
              aria-label="builtin XML not deletable"
            />
          </Tooltip>
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
