import { useState } from 'react';
import { Table, Tag, Tooltip, Button, Space, Modal, Spin, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  useAdminSubFieldList,
  useDeleteSubField,
} from '@core/hooks/api/useMmlAdmin';
import type { AdminSubFieldEnriched } from '@core/types/mmlAdmin';
import type { GroupTreeCommand } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';
import AddSubFieldsModal from './AddSubFieldsModal';
import EditSubFieldModal from './EditSubFieldModal';

// 2026-05-27 mml-admin-catalog-redesign-20260527 §4.4：右栏下方 path 列表。
// 列定义参考原 CommandDetailDrawer。
//
// 2026-05-27 用户决策:
//   - 显示名拆为"中文"+"英文"两列,各占独立列体现 i18n 全貌
//   - 取消 catalog_protected 在 UI 上的所有锁定逻辑(行 / 表头按钮),
//     权限交给路由层 RBAC(withAdminRole) + 后端 API 校验

export interface PathListSectionProps {
  command: GroupTreeCommand;
}

export default function PathListSection({
  command,
}: PathListSectionProps): React.ReactElement {
  const t = useT();
  const { data: subFields = [], isLoading, refetch } = useAdminSubFieldList(command.id);
  const deleteMut = useDeleteSubField();

  const [addOpen, setAddOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<AdminSubFieldEnriched | undefined>();

  const handleDelete = (sf: AdminSubFieldEnriched) => {
    Modal.confirm({
      title: `${t('mml.admin.catalog.common.delete')} ${sf.mmlCode}?`,
      okButtonProps: { danger: true },
      okText: t('mml.admin.catalog.common.confirm'),
      cancelText: t('mml.admin.catalog.common.cancel'),
      onOk: async () => {
        try {
          await deleteMut.mutateAsync({
            commandId: command.id,
            subFieldId: sf.id,
          });
          message.success(t('mml.admin.catalog.common.deleteSuccess'));
          void refetch();
        } catch (e) {
          if (e instanceof Error) message.error(e.message);
        }
      },
    });
  };

  const columns: ColumnsType<AdminSubFieldEnriched> = [
    {
      title: t('mml.admin.catalog.subField.mmlCode'),
      dataIndex: 'mmlCode',
      key: 'mmlCode',
      width: 160,
      ellipsis: true,
    },
    {
      title: t('mml.admin.catalog.subField.labelZh'),
      key: 'labelZh',
      width: 140,
      ellipsis: true,
      render: (_, sf) =>
        sf.labelI18n?.['zh-CN'] || sf.labelI18n?.zh || '-',
    },
    {
      title: t('mml.admin.catalog.subField.labelEn'),
      key: 'labelEn',
      width: 140,
      ellipsis: true,
      render: (_, sf) =>
        sf.labelI18n?.['en-US'] || sf.labelI18n?.en || '-',
    },
    {
      title: t('mml.admin.catalog.subField.standardPath'),
      dataIndex: 'tr069Path',
      key: 'standardPath',
      ellipsis: { showTitle: true },
      render: (p: string) => (
        <code style={{ fontSize: 12 }} title={p}>
          {p}
        </code>
      ),
    },
    {
      title: t('mml.admin.catalog.subField.defaultSelected'),
      dataIndex: 'defaultSelected',
      key: 'defaultSelected',
      width: 80,
      align: 'center',
      render: (v: boolean) => (v ? <Tag color="green">✓</Tag> : '-'),
    },
    {
      title: t('mml.admin.catalog.subField.supported'),
      dataIndex: 'isSupported',
      key: 'isSupported',
      width: 130,
      align: 'center',
      render: (_v: boolean, sf) => {
        if (sf.totalModelCount === 0) {
          return (
            <Tooltip title={t('mml.admin.catalog.subField.supportedUnknownTip')}>
              <Tag color="default">
                {t('mml.admin.catalog.subField.supportedUnknown')}
              </Tag>
            </Tooltip>
          );
        }
        const allOk = sf.supportedModelCount === sf.totalModelCount;
        const noneOk = sf.supportedModelCount === 0;
        const color = allOk ? 'green' : noneOk ? 'red' : 'orange';
        return (
          <Tooltip title={t('mml.admin.catalog.subField.supportedTip')}>
            <Tag color={color}>{`${sf.supportedModelCount}/${sf.totalModelCount}`}</Tag>
          </Tooltip>
        );
      },
    },
    {
      title: 'Op',
      key: 'op',
      width: 120,
      align: 'center',
      render: (_, sf) => (
        <Space size={0} split="·">
          <a onClick={() => setEditTarget(sf)}>
            {t('mml.admin.catalog.common.edit')}
          </a>
          <a onClick={() => handleDelete(sf)} style={{ color: '#ff4d4f' }}>
            {t('mml.admin.catalog.common.delete')}
          </a>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Space
        style={{ width: '100%', justifyContent: 'space-between', marginBottom: 8 }}
      >
        <h4 style={{ margin: 0 }}>
          {t('mml.admin.catalog.commands.subFieldsTitle')}
        </h4>
        <Button
          type="primary"
          size="small"
          icon={<PlusOutlined />}
          onClick={() => setAddOpen(true)}
        >
          {t('mml.admin.catalog.subField.batchAddBtn')}
        </Button>
      </Space>
      {isLoading ? (
        <Spin />
      ) : (
        <Table<AdminSubFieldEnriched>
          rowKey="id"
          columns={columns}
          dataSource={subFields}
          pagination={false}
          size="small"
        />
      )}
      <AddSubFieldsModal
        open={addOpen}
        commandId={command.id}
        existingPathIds={subFields.map((sf) => sf.paramId)}
        onClose={() => setAddOpen(false)}
        onSuccess={() => void refetch()}
      />
      <EditSubFieldModal
        open={!!editTarget}
        commandId={command.id}
        subField={editTarget}
        onClose={() => setEditTarget(undefined)}
        onSuccess={() => void refetch()}
      />
    </div>
  );
}
