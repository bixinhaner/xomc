/**
 * XMLFilesModal — T-0180 P4 "管理 XML 文件"弹窗(per tech)。
 *
 * 列出该制式下所有 XML 文件:DB GROUP BY loaded_from + 物理盘 uploaded-but-not-loaded 合并。
 * 每行:文件名 / 来源 Tag(内置/自定义/未知)/ 指标数 / 删除按钮。
 *
 * 2026-06-03 用户决策:所有文件均可删除,去掉 deletable 守门(后端 deletable 恒为 true)。
 * onConfirm 调 useIndicatorDeleteFile,成功后 refetch 同步刷新 Summary + 二级表。
 */
import { Modal, Table, Tag, Button, Popconfirm, message, Space } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import {
  useIndicatorFiles,
  useIndicatorDeleteFile,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { IndicatorFile, TechLower } from '@core/types/indicatorLibrary';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  tech: TechLower | undefined;
  onClose: () => void;
}

const SOURCE_TAG: Record<IndicatorFile['source'], { color: string }> = {
  builtin: { color: 'default' },
  custom: { color: 'blue' },
  unknown: { color: 'warning' },
};

const TECH_LABEL: Record<TechLower, string> = {
  enb: 'ENB (LTE)',
  gsm: 'GSM',
  gnb: 'GNB (5G NR)',
};

export default function XMLFilesModal({ open, tech, onClose }: Props) {
  const t = useT();
  const { data, isLoading, refetch } = useIndicatorFiles(open ? tech : undefined);
  const deleteMut = useIndicatorDeleteFile();
  const items = data?.items || [];

  const columns = [
    {
      // 2026-05-29 用户决策:文件名列显示后端 API 给的完整路径
      // (如 "indicator-library/enb/ALL.xml"),不再截 basename。
      title: t('common.fileName'),
      dataIndex: 'loadedFrom',
      render: (v: string) => <code>{v}</code>,
    },
    {
      title: t('common.source'),
      dataIndex: 'source',
      width: 100,
      render: (v: IndicatorFile['source']) => (
        <Tag color={SOURCE_TAG[v].color}>{t(`common.${v}`)}</Tag>
      ),
    },
    {
      title: t('common.indicators'),
      dataIndex: 'count',
      width: 90,
    },
    {
      title: t('product.kpi.xml.colInDisk'),
      dataIndex: 'onDisk',
      width: 80,
      render: (v: boolean) =>
        v ? <Tag color="success">{t('common.yes')}</Tag> : <Tag color="warning">{t('common.no')}</Tag>,
    },
    {
      title: t('common.action'),
      width: 100,
      // 2026-06-03:所有文件均可删除,去掉 deletable 置灰守门。
      render: (_: unknown, row: IndicatorFile) => (
        <Popconfirm
          title={t('product.kpi.xml.delTitle')}
          description={
            <div style={{ maxWidth: 280 }}>
              {t('product.kpi.xml.deleteBullet1Pre')}<code>.deleted.&lt;ts&gt;</code>{t('product.kpi.xml.deleteBullet1Post')}
              <br />
              {t('product.kpi.xml.deleteBullet2', { count: row.count })}
            </div>
          }
          okText={t('common.delete')}
          cancelText={t('common.cancel')}
          okButtonProps={{ danger: true }}
          onConfirm={() =>
            deleteMut
              .mutateAsync(row.loadedFrom)
              .then((r) => {
                message.success(
                  r.backup ? t('product.kpi.xml.deleteSuccessWithBackup', { backup: r.backup }) : t('common.deleted')
                );
                void refetch();
              })
              .catch((e) => message.error((e as Error).message))
          }
        >
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <Modal
      title={
        <Space>
          <span>{t('product.kpi.xml.manageTitle')}</span>
          {tech && <Tag color="blue">{TECH_LABEL[tech]}</Tag>}
        </Space>
      }
      open={open}
      onCancel={onClose}
      footer={null}
      width={760}
      destroyOnHidden
    >
      <Table<IndicatorFile>
        rowKey="loadedFrom"
        loading={isLoading}
        columns={columns}
        dataSource={items}
        size="small"
        pagination={false}
      />
    </Modal>
  );
}
