/**
 * SummaryTab — T-0180 一级 (制式, 平台) 聚合表(2026-05-29 第四轮调整)。
 *
 * 本轮(用户决策,对齐 T-0178 param-model 范式):
 *   1. 新增"来源"列:Tag 内置 / 自定义(后端 source.go::ClassifySource 派生)
 *   2. "XML 文件"列改名"加载源",显示完整 loaded_from(原来只显 basename)
 *   3. 新增"操作"列:仅 deletable=true(custom)显红色删除;builtin 置灰 + Tooltip
 *      "{t('product.kpi.summary.builtinHint')}"
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
import { makeSeqColumn } from '@/components/Table/seqColumn';
import { useT } from '@/hooks/useT';

interface Props {
  onSelect: (tech: TechLower, platform: string) => void;
}

const TECH_LABEL: Record<TechLower, string> = {
  enb: 'ENB (LTE)',
  gsm: 'GSM',
  gnb: 'GNB (5G NR)',
};

export default function SummaryTab({ onSelect }: Props) {
  const t = useT();
  const { data, isLoading } = useIndicatorSummary();
  const deleteMut = useIndicatorDeleteFile();
  const items = data?.items || [];

  const columns = [
    {
      title: t('common.name'),
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
      // 后端 source.go::ClassifySource 派生,前端只渲染
      title: t('common.source'),
      dataIndex: 'source',
      width: 90,
      filters: [
        { text: t('common.builtin'), value: 'builtin' as IndicatorSource },
        { text: t('common.custom'), value: 'custom' as IndicatorSource },
      ],
      onFilter: (val: boolean | React.Key, row: IndicatorPlatformSummary) =>
        row.source === val,
      render: (s: IndicatorSource | undefined) =>
        s === 'custom' ? <Tag color="blue">{t('common.custom')}</Tag> : <Tag>{t('common.builtin')}</Tag>,
    },
    {
      // 2026-05-29:从"XML 文件"(只显 basename)改为"加载源"(全路径,ellipsis)
      // 2026-05-31:列宽 260→420,容纳 `indicator-library-custom/enb/<basename>.xml`
      //            等 50+ 字符的全路径,典型 builtin `indicator-library/enb/ALL.xml`
      //            (32 char) 也保留余量。
      title: t('common.loadedFrom'),
      dataIndex: 'loadedFrom',
      width: 420,
      ellipsis: true,
      render: (v: string) => <Tooltip title={v}><code>{v}</code></Tooltip>,
    },
    {
      title: t('common.tech'),
      dataIndex: 'tech',
      width: 130,
      render: (v: TechLower) => <Tag color="geekblue">{TECH_LABEL[v]}</Tag>,
    },
    { title: t('common.indicators'), dataIndex: 'indicators', width: 90 },
    {
      title: t('common.note'),
      ellipsis: true,
      render: (_: unknown, row: IndicatorPlatformSummary) => (
        <span style={{ color: 'rgba(0, 0, 0, 0.65)', fontSize: 12 }}>
          {t(`product.kpi.summary.tech.${row.tech}`)} · {t('common.platform')} {row.platform}
        </span>
      ),
    },
    {
      title: t('common.action'),
      width: 80,
      render: (_: unknown, row: IndicatorPlatformSummary) =>
        row.deletable ? (
          <Popconfirm
            title={t('product.kpi.summary.confirmDeleteXml')}
            description={
              <div style={{ maxWidth: 320 }}>
                · {t('product.kpi.summary.deleteBullet1Pre')} <code>{row.loadedFrom}</code> {t('product.kpi.summary.deleteBullet1Post')}
                <code>.deleted.&lt;ts&gt;</code> {t('product.kpi.summary.backupSuffix')}
                <br />· {t('product.kpi.summary.deleteBullet2', { count: row.indicators })}
                <br />· {t('product.kpi.summary.deleteBullet3')}
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
                {t('product.kpi.summary.builtinHint')}
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
        columns={[makeSeqColumn<IndicatorPlatformSummary>({ title: t('table.rowNumber'), dataSource: items }), ...columns]}
        dataSource={items}
        size="small"
        pagination={false}
      />
    </Card>
  );
}
