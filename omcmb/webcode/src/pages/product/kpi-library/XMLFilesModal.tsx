/**
 * XMLFilesModal — T-0180 P4 "管理 XML 文件"弹窗(per tech)。
 *
 * 列出该制式下所有 XML 文件:DB GROUP BY loaded_from + 物理盘 uploaded-but-not-loaded 合并。
 * 每行:文件名 / 来源 Tag(内置/自定义/未知)/ 指标数 / 删除按钮(内置置灰)。
 *
 * 删除走 IsDeletable 守门(后端 file_handler.go):仅 source=custom 行可点;
 * onConfirm 调 useIndicatorDeleteFile,成功后 refetch 同步刷新 Summary + 二级表。
 */
import { Modal, Table, Tag, Button, Popconfirm, Tooltip, message, Space } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import {
  useIndicatorFiles,
  useIndicatorDeleteFile,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { IndicatorFile, TechLower } from '@core/types/indicatorLibrary';

interface Props {
  open: boolean;
  tech: TechLower | undefined;
  onClose: () => void;
}

const SOURCE_TAG: Record<IndicatorFile['source'], { color: string; label: string }> = {
  builtin: { color: 'default', label: '内置' },
  custom: { color: 'blue', label: '自定义' },
  unknown: { color: 'warning', label: '未知' },
};

const TECH_LABEL: Record<TechLower, string> = {
  enb: 'ENB (LTE)',
  gsm: 'GSM',
  gnb: 'GNB (5G NR)',
};

export default function XMLFilesModal({ open, tech, onClose }: Props) {
  const { data, isLoading, refetch } = useIndicatorFiles(open ? tech : undefined);
  const deleteMut = useIndicatorDeleteFile();
  const items = data?.items || [];

  const columns = [
    {
      // 2026-05-29 用户决策:文件名列显示后端 API 给的完整路径
      // (如 "indicator-library/enb/ALL.xml"),不再截 basename。
      title: '文件名',
      dataIndex: 'loadedFrom',
      render: (v: string) => <code>{v}</code>,
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 100,
      render: (v: IndicatorFile['source']) => (
        <Tag color={SOURCE_TAG[v].color}>{SOURCE_TAG[v].label}</Tag>
      ),
    },
    {
      title: '指标数',
      dataIndex: 'count',
      width: 90,
    },
    {
      title: '在盘',
      dataIndex: 'onDisk',
      width: 80,
      render: (v: boolean) =>
        v ? <Tag color="success">{t('common.yes')}</Tag> : <Tag color="warning">{t('common.no')}</Tag>,
    },
    {
      title: '操作',
      width: 100,
      render: (_: unknown, row: IndicatorFile) => {
        if (!row.deletable) {
          return (
            <Tooltip title={t('product.kpi.xml.builtinTip')}>
              <Button size="small" danger disabled icon={<DeleteOutlined />} />
            </Tooltip>
          );
        }
        return (
          <Popconfirm
            title={t('product.kpi.xml.delTitle')}
            description={
              <div style={{ maxWidth: 280 }}>
                文件备份为 <code>.deleted.&lt;ts&gt;</code>;
                <br />
                关联 {row.count} 个指标 + 公式 + 启用记录会一并清理。
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
                    r.backup ? `已删除(备份 ${r.backup})` : '已删除'
                  );
                  void refetch();
                })
                .catch((e) => message.error((e as Error).message))
            }
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        );
      },
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
