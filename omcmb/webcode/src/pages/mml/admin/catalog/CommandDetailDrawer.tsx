import { useState } from 'react';
import { Drawer, Descriptions, Table, Tag, Spin, Alert, Modal, Button, Space, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  useDeleteSubField,
  useAdminSubFieldList,
} from '@core/hooks/api/useMmlAdmin';
import type { AdminSubFieldEnriched } from '@core/types/mmlAdmin';
import type { GroupTreeCommand } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';
import AddSubFieldsModal from './AddSubFieldsModal';

export interface CommandDetailDrawerProps {
  open: boolean;
  command?: GroupTreeCommand;
  onClose: () => void;
}

export default function CommandDetailDrawer({ open, command, onClose }: CommandDetailDrawerProps) {
  const t = useT();
  // T-Mml-Admin: 使用 admin 视角 list（含 is_supported=false 行），与 console 端区分
  const { data: subFields = [], isLoading, refetch } = useAdminSubFieldList(command?.id);
  const deleteMut = useDeleteSubField();
  const [addOpen, setAddOpen] = useState(false);

  const handleDelete = (sf: AdminSubFieldEnriched) => {
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

  const subFieldColumns: ColumnsType<AdminSubFieldEnriched> = [
    { title: t('mml.admin.catalog.subField.mmlCode'), dataIndex: 'mmlCode', key: 'mmlCode', width: 160 },
    {
      title: t('mml.admin.catalog.subField.label'),
      dataIndex: 'label',
      key: 'label',
      ellipsis: true,
      render: (_, sf) => sf.labelI18n?.['zh-CN'] || sf.labelI18n?.['en-US'] || sf.label || sf.mmlCode,
    },
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
      // T-Mml-Admin: admin 视角必须显示 is_supported 标记，让维护人员能识别
      // auto-learn / 手工关闭的 path（console 视图直接过滤不展示）。
      title: t('mml.admin.catalog.subField.supported'),
      dataIndex: 'isSupported',
      key: 'isSupported',
      width: 90,
      render: (v: boolean) =>
        v ? <Tag color="green">{t('mml.admin.catalog.subField.supportedYes')}</Tag>
          : <Tag color="orange">{t('mml.admin.catalog.subField.supportedNo')}</Tag>,
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

          <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 8 }}>
            <h4 style={{ margin: 0 }}>{t('mml.admin.catalog.commands.subFieldsTitle')}</h4>
            <Button
              type="primary"
              size="small"
              icon={<PlusOutlined />}
              onClick={() => setAddOpen(true)}
              disabled={!command || command.catalogProtected}
              title={command?.catalogProtected ? t('mml.admin.catalog.common.lockedTip') : undefined}
            >
              {t('mml.admin.catalog.subField.batchAddBtn')}
            </Button>
          </Space>
          {isLoading ? (
            <Spin />
          ) : (
            <Table<AdminSubFieldEnriched>
              rowKey="id"
              columns={subFieldColumns}
              dataSource={subFields}
              pagination={false}
              size="small"
            />
          )}
          {command && (
            <AddSubFieldsModal
              open={addOpen}
              commandId={command.id}
              existingPathIds={subFields.map((sf) => sf.paramId)}
              onClose={() => setAddOpen(false)}
              onSuccess={() => void refetch()}
            />
          )}
        </>
      )}
    </Drawer>
  );
}
