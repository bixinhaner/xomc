import { useState } from 'react';
import { Table, Tag, Tooltip, Button, Space, Modal, Spin, Typography, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
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

type ObjectPathRow = {
  id: string;
  rowKind: 'object';
  mmlCode: string;
  label: string;
  tr069Path: string;
  defaultSelected: boolean;
};

type PathRow = AdminSubFieldEnriched | ObjectPathRow;

function isObjectPathRow(row: PathRow): row is ObjectPathRow {
  return 'rowKind' in row && row.rowKind === 'object';
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

  const isObjectOperation = command.operationType === 'ADD' || command.operationType === 'RMV';
  const objectPath = command.targetObject?.trim();
  const rows: PathRow[] =
    subFields.length === 0 && isObjectOperation && objectPath
      ? [
          {
            id: `${command.id}:target-object`,
            rowKind: 'object',
            mmlCode: command.operationType,
            label: t('mml.admin.catalog.commands.targetObject'),
            tr069Path: objectPath,
            defaultSelected: true,
          },
        ]
      : subFields;

  const columns: ColumnsType<PathRow> = [
    {
      title: t('mml.admin.catalog.subField.mmlCode'),
      dataIndex: 'mmlCode',
      key: 'mmlCode',
      width: 160,
      ellipsis: true,
    },
    {
      title: t('mml.admin.catalog.commands.displayName'),
      key: 'label',
      width: 160,
      ellipsis: true,
      render: (_, sf) => {
        if (isObjectPathRow(sf)) return sf.label;
        return (
          sf.labelI18n?.['zh-CN'] ||
          sf.labelI18n?.['en-US'] ||
          sf.labelI18n?.zh ||
          sf.labelI18n?.en ||
          sf.label ||
          '-'
        );
      },
    },
    {
      title: t('mml.admin.catalog.subField.standardPath'),
      dataIndex: 'tr069Path',
      key: 'standardPath',
      // 定宽列：超长 path 在列内截断，复制图标固定在右侧同一位置（不被长路径顶飞）。
      width: 360,
      render: (p: string) => (
        // flex 布局：路径占满剩余宽度并截断(minWidth:0 让其可收缩)，复制图标 flex 不收缩、固定右侧。
        // 复用 antd Typography.copyable（自带剪贴板 + 反馈，tooltip 走 i18n）。
        <div style={{ display: 'flex', alignItems: 'center', gap: 4, minWidth: 0 }}>
          <Typography.Text
            code
            style={{ fontSize: 12, flex: 1, minWidth: 0 }}
            ellipsis={{ tooltip: p }}
          >
            {p}
          </Typography.Text>
          <Typography.Text
            style={{ flex: '0 0 auto' }}
            copyable={{ text: p, tooltips: [t('common.copy'), t('table.copied')] }}
          />
        </div>
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
      title: t('mml.admin.catalog.common.actions'),
      key: 'op',
      width: 120,
      align: 'center',
      render: (_, sf) =>
        isObjectPathRow(sf) ? null : (
          <Space size={4}>
            <Tooltip title={t('mml.admin.catalog.common.edit')}>
              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => setEditTarget(sf)}
              />
            </Tooltip>
            <Tooltip title={t('mml.admin.catalog.common.delete')}>
              <Button
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                onClick={() => handleDelete(sf)}
              />
            </Tooltip>
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
          disabled={isObjectOperation}
        >
          {t('mml.admin.catalog.subField.batchAddBtn')}
        </Button>
      </Space>
      {isLoading ? (
        <Spin />
      ) : (
        <Table<PathRow>
          rowKey="id"
          columns={columns}
          dataSource={rows}
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
