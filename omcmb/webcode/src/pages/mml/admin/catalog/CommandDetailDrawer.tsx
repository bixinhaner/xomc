import { Drawer, Descriptions, Table, Tag, Spin, Alert, Modal, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  useCommandSubFields,
} from '@core/hooks/api/useMmlConsole';
import { useDeleteSubField } from '@core/hooks/api/useMmlAdmin';
import type { SubFieldDef, GroupTreeCommand } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

export interface CommandDetailDrawerProps {
  open: boolean;
  command?: GroupTreeCommand;
  onClose: () => void;
}

export default function CommandDetailDrawer({ open, command, onClose }: CommandDetailDrawerProps) {
  const t = useT();
  const { data: subFields = [], isLoading, refetch } = useCommandSubFields(command?.id, 'zh-CN');
  const deleteMut = useDeleteSubField();

  const handleDelete = (sf: SubFieldDef) => {
    if (!command) return;
    Modal.confirm({
      title: `${t('mml.admin.catalog.common.delete')} ${sf.mmlCode}?`,
      okButtonProps: { danger: true },
      okText: t('mml.admin.catalog.common.confirm'),
      cancelText: t('mml.admin.catalog.common.cancel'),
      onOk: async () => {
        try {
          await deleteMut.mutateAsync({ commandId: command.id, subFieldId: sf.id });
          message.success(t('mml.admin.catalog.common.deleteSuccess'));
          void refetch();
        } catch (e) {
          if (e instanceof Error) message.error(e.message);
        }
      },
    });
  };

  const subFieldColumns: ColumnsType<SubFieldDef> = [
    { title: t('mml.admin.catalog.subField.mmlCode'), dataIndex: 'mmlCode', key: 'mmlCode', width: 160 },
    { title: t('mml.admin.catalog.subField.label'), dataIndex: 'label', key: 'label', ellipsis: true },
    { title: 'TR-069 Path', dataIndex: 'tr069Path', key: 'tr069Path', ellipsis: true },
    {
      title: t('mml.admin.catalog.subField.defaultSelected'),
      dataIndex: 'defaultSelected',
      key: 'defaultSelected',
      width: 90,
      render: (v: boolean) => (v ? <Tag color="green">✓</Tag> : '-'),
    },
    {
      title: t('mml.admin.catalog.subField.isRequired'),
      dataIndex: 'isRequired',
      key: 'isRequired',
      width: 70,
      render: (v: boolean) => (v ? <Tag color="red">*</Tag> : '-'),
    },
    {
      title: t('mml.admin.catalog.subField.sortOrder'),
      dataIndex: 'sortOrder',
      key: 'sortOrder',
      width: 70,
    },
    {
      title: t('mml.admin.catalog.common.delete'),
      key: 'op',
      width: 80,
      render: (_, sf) => (
        <a onClick={() => handleDelete(sf)}>{t('mml.admin.catalog.common.delete')}</a>
      ),
    },
  ];

  return (
    <Drawer open={open} onClose={onClose} width={720} title={command?.displayName ?? ''}>
      {!command ? (
        <Alert type="info" message="Select a command" />
      ) : (
        <>
          <Descriptions column={2} size="small" bordered style={{ marginBottom: 16 }}>
            <Descriptions.Item label={t('mml.admin.catalog.commands.code')}>
              {command.commandCode}
            </Descriptions.Item>
            <Descriptions.Item label={t('mml.admin.catalog.commands.logicalCode')}>
              {command.logicalCode}
            </Descriptions.Item>
            <Descriptions.Item label={t('mml.admin.catalog.commands.op')}>
              <Tag>{command.operationType}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('mml.admin.catalog.commands.displayName')}>
              {command.displayName}
            </Descriptions.Item>
            {command.targetObject && (
              <Descriptions.Item label="target_object" span={2}>
                {command.targetObject}
              </Descriptions.Item>
            )}
            <Descriptions.Item label="require_confirm" span={2}>
              {command.requireConfirm ? <Tag color="warning">true</Tag> : 'false'}
            </Descriptions.Item>
            <Descriptions.Item label="source">
              <Tag color={command.source === 'admin' ? 'blue' : 'default'}>
                {command.source ?? 'standard'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="catalog_protected">
              {command.catalogProtected ? <Tag color="default">locked</Tag> : '-'}
            </Descriptions.Item>
          </Descriptions>

          <h4>{t('mml.admin.catalog.commands.subFieldsTitle')}</h4>
          {isLoading ? (
            <Spin />
          ) : (
            <Table<SubFieldDef>
              rowKey="id"
              columns={subFieldColumns}
              dataSource={subFields}
              pagination={false}
              size="small"
            />
          )}
        </>
      )}
    </Drawer>
  );
}
