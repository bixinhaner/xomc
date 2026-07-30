/**
 * SummaryTab — PM 指标库一级列表(2026-06-02:"一个平台一条")。
 *
 * 用户决策:一个 XML 文件即一个平台,一级列表按平台聚合,每平台唯一一行。
 * 列:序号 / 平台(link → 二级) / 加载源 / 指标数 / 描述 / 操作(编辑描述)。
 * 2026-06-04 用户决策:取消「制式」列(一文件一平台,制式已隐含于平台/描述,单列冗余)。
 * 2026-06-03 用户决策:去掉"来源(builtin/custom)"列,保留"加载源(loaded_from)"列;
 *   文件删除仍留在 XMLFilesModal。
 */
import { useState } from 'react';
import { Card, Table, Button, Space, Tooltip, message, Input, Modal, Popconfirm } from 'antd';
import { EditOutlined, DownloadOutlined, DeleteOutlined } from '@ant-design/icons';
import type { AxiosError } from 'axios';
import type { IndicatorPlatformSummary, TechLower } from '@core/types/indicatorLibrary';
import {
  useIndicatorSummary,
  useUpdateIndicatorFileDescription,
  useIndicatorDownloadXml,
  useIndicatorDeleteFile,
} from '@core/hooks/api/useIndicatorsLibrary';
import { makeSeqColumn } from '@/components/Table/seqColumn';
import { useT } from '@/hooks/useT';

interface Props {
  onSelect: (tech: TechLower, platform: string) => void;
  // 2026-06-03:搜索框上提到页面 toolbar(与「导入 XML」同行),query 由父组件受控传入。
  query?: string;
}

// 2026-06-02 用户决策:制式列只显设备制式本身(ENB/GSM/GNB),不带 "(LTE)/(5G NR)" 括号注解。
const TECH_LABEL: Record<TechLower, string> = {
  enb: 'ENB',
  gsm: 'GSM',
  gnb: 'GNB',
};

export default function SummaryTab({ onSelect, query = '' }: Props) {
  const t = useT();
  const { data, isLoading } = useIndicatorSummary();
  const updateDescMut = useUpdateIndicatorFileDescription();
  const downloadMut = useIndicatorDownloadXml();
  const deleteFileMut = useIndicatorDeleteFile();
  const allItems = data?.items || [];

  // 编辑描述(按 (制式, 平台) 维度)
  const [editRow, setEditRow] = useState<IndicatorPlatformSummary | null>(null);
  const [draft, setDraft] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // 搜索词变化时回到第一页——渲染期重置,避免 set-state-in-effect。
  const [prevQuery, setPrevQuery] = useState(query);
  if (query !== prevQuery) {
    setPrevQuery(query);
    setPage(1);
  }

  // 自动文本兜底:未填自定义描述时展示"制式说明 · 平台 X"。
  const autoDesc = (row: IndicatorPlatformSummary) =>
    `${t(`product.kpi.summary.tech.${row.tech}`)} · ${t('common.platform')} ${row.platform}`;
  // 展示用描述:有存储描述则用之,否则用自动文本。列渲染与搜索统一走这里。
  const shownDesc = (row: IndicatorPlatformSummary) =>
    row.description && row.description.trim() ? row.description : autoDesc(row);

  // 客户端模糊筛选:名称(平台)+ 描述(summary 一次性全量返回,client 过滤即可)。
  const q = query.trim().toLowerCase();
  const items = q
    ? allItems.filter(
        (r) => r.platform.toLowerCase().includes(q) || shownDesc(r).toLowerCase().includes(q),
      )
    : allItems;

  const saveDesc = () => {
    if (!editRow) return;
    updateDescMut
      .mutateAsync({ tech: editRow.tech, platform: editRow.platform, description: draft })
      .then(() => {
        message.success(t('common.saved'));
        setEditRow(null);
      })
      .catch((e) => {
        const ax = e as AxiosError<{ msg?: string; message?: string }>;
        message.error(
          ax.response?.data?.msg ?? ax.response?.data?.message ?? (e instanceof Error ? e.message : String(e)),
        );
      });
  };

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
      // 该平台对应的 XML 文件(后端 MAX(formula.loaded_from));一文件一平台故单值
      title: t('common.loadedFrom'),
      dataIndex: 'loadedFrom',
      width: 320,
      ellipsis: true,
      render: (v: string) => (v ? <Tooltip title={v}><code>{v}</code></Tooltip> : <span>—</span>),
    },
    { title: t('common.indicators'), dataIndex: 'indicators', width: 90 },
    {
      title: t('common.description'),
      ellipsis: true,
      render: (_: unknown, row: IndicatorPlatformSummary) => (
        // 字体大小/颜色对齐「指标数」列(默认主题文本色),避免硬编码亮色在暗黑模式下不可见。
        <Tooltip title={shownDesc(row)}>
          <span>{shownDesc(row)}</span>
        </Tooltip>
      ),
    },
    {
      title: t('common.action'),
      width: 150,
      render: (_: unknown, row: IndicatorPlatformSummary) => (
        <Space>
          {/* 2026-06-05:下载 XML 原文件(builtin / custom 均可;无加载源禁用) */}
          <Tooltip title={t('product.upload.downloadXml')}>
            <Button
              size="small"
              icon={<DownloadOutlined />}
              disabled={!row.loadedFrom}
              onClick={() =>
                downloadMut
                  .mutateAsync({ loadedFrom: row.loadedFrom })
                  .catch((e) => message.error((e as Error).message))
              }
            />
          </Tooltip>
          <Tooltip title={t('product.kpi.summary.editDescTitle')}>
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => {
                setEditRow(row);
                setDraft(row.description ?? '');
              }}
            />
          </Tooltip>
          {/* 2026-06-05:一级列表删除(对齐 param-model / alarm-library):
              仅 custom 可删,内置置灰 + Tooltip;真值源 = 后端 deletable。 */}
          {row.deletable ? (
            <Popconfirm
              title={t('product.kpi.xml.delTitle')}
              description={
                <div style={{ maxWidth: 280 }}>
                  {t('product.kpi.xml.deleteBullet1Pre')}<code>.deleted.&lt;ts&gt;</code>{t('product.kpi.xml.deleteBullet1Post')}
                  <br />
                  {t('product.kpi.xml.deleteBullet2', { count: row.indicators })}
                </div>
              }
              okText={t('common.delete')}
              cancelText={t('common.cancel')}
              okButtonProps={{ danger: true }}
              onConfirm={() =>
                deleteFileMut
                  .mutateAsync(row.loadedFrom)
                  .then((r) =>
                    message.success(
                      r.backup
                        ? t('product.kpi.xml.deleteSuccessWithBackup', { backup: r.backup })
                        : t('common.deleted'),
                    ),
                  )
                  .catch((e) => message.error((e as Error).message))
              }
            >
              <Button size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          ) : (
            <Tooltip title={t('common.builtinNoDelete')}>
              <Button size="small" danger icon={<DeleteOutlined />} disabled />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  return (
    <Card size="small">
      <Table<IndicatorPlatformSummary>
        rowKey={(r) => `${r.tech}__${r.platform}`}
        loading={isLoading}
        columns={[makeSeqColumn<IndicatorPlatformSummary>({ title: t('table.rowNumber'), dataSource: items }), ...columns]}
        dataSource={items}
        size="small"
        pagination={{
          current: page,
          pageSize,
          total: items.length,
          showSizeChanger: true,
          pageSizeOptions: ['10', '20', '50', '1000'],
          showTotal: (n) => t('common.totalCount', { count: n }),
          onChange: (p, ps) => {
            setPage(p);
            if (ps !== pageSize) setPageSize(ps);
          },
        }}
      />

      <Modal
        title={t('product.kpi.summary.editDescTitle')}
        open={Boolean(editRow)}
        onOk={saveDesc}
        confirmLoading={updateDescMut.isPending}
        onCancel={() => setEditRow(null)}
        okText={t('common.save')}
        destroyOnHidden
      >
        {editRow && (
          <>
            {/* 描述按 (制式, 平台) 维度 */}
            <div style={{ color: 'rgba(0,0,0,0.45)', fontSize: 12, marginBottom: 8 }}>
              {t('common.platform')}：<code>{TECH_LABEL[editRow.tech]} / {editRow.platform}</code>
            </div>
            <div style={{ color: 'rgba(0,0,0,0.45)', fontSize: 12, marginBottom: 8 }}>
              {t('product.kpi.summary.editDescHint')}
            </div>
            <Input.TextArea
              rows={4}
              maxLength={500}
              showCount
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder={t('product.kpi.summary.editDescPh')}
            />
          </>
        )}
      </Modal>
    </Card>
  );
}
