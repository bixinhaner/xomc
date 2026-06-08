import { Button, Descriptions, Empty, Popconfirm, Space, Tag, Typography } from 'antd';
import { EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { MMLCustomCommand } from '@core/types/mml';
import { useT } from '@/hooks/useT';

// mml-console-redesign-20260603：admin catalog 选中「自定义命令」叶子后，
// 在右栏 RightDetailPanel 渲染该 MMLCustomCommand 的只读详情 + 编辑/删除入口
// （编辑复用 Console 同款 AddTemplateModal，由本面板「编辑」按钮触发）。

const OP_TAG_COLOR: Record<string, string> = {
  LST: 'blue',
  MOD: 'orange',
  ADD: 'green',
  RMV: 'red',
};

export interface CustomCommandDetailPanelProps {
  command: MMLCustomCommand;
  /** 是否允许编辑/删除（isSuperAdmin || creator===currentUsername）。 */
  canModify: boolean;
  onEdit: (cc: MMLCustomCommand) => void;
  onDelete: (cc: MMLCustomCommand) => void;
}

export default function CustomCommandDetailPanel({
  command,
  canModify,
  onEdit,
  onDelete,
}: CustomCommandDetailPanelProps): React.ReactElement {
  const t = useT();

  const paramEntries = Object.entries(command.parameters ?? {});
  const paths = command.paramPaths ?? [];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <Space size="small">
          <Tag color={OP_TAG_COLOR[command.operationType] ?? 'default'}>
            {command.operationType}
          </Tag>
          <Typography.Title level={5} style={{ margin: 0 }}>
            {command.commandName}
          </Typography.Title>
          <Tag color={command.commandScope === 'public' ? 'green' : 'orange'}>
            {command.commandScope === 'public'
              ? t('mml.admin.catalog.customized.publicTemplate')
              : t('mml.admin.catalog.customized.privateTemplate')}
          </Tag>
        </Space>
        {canModify && (
          <Space size="small">
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => onEdit(command)}
            >
              {t('mml.admin.catalog.common.edit')}
            </Button>
            <Popconfirm
              title={t('mml.template.deleteConfirm', { name: command.commandName })}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
              okButtonProps={{ danger: true }}
              onConfirm={() => onDelete(command)}
            >
              <Button size="small" danger icon={<DeleteOutlined />}>
                {t('mml.admin.catalog.common.delete')}
              </Button>
            </Popconfirm>
          </Space>
        )}
      </div>

      <Descriptions
        column={1}
        size="small"
        bordered
        styles={{ label: { width: 160 } }}
      >
        <Descriptions.Item label={t('mml.console.commandName')}>
          {command.commandName}
        </Descriptions.Item>
        <Descriptions.Item label={t('mml.console.commandCode')}>
          <Typography.Text code>{command.commandCode}</Typography.Text>
        </Descriptions.Item>
        <Descriptions.Item label={t('mml.console.operationType')}>
          {command.operationType}
        </Descriptions.Item>
        <Descriptions.Item label={t('mml.admin.catalog.customized.scope')}>
          {command.commandScope === 'public'
            ? t('mml.admin.catalog.customized.publicTemplate')
            : t('mml.admin.catalog.customized.privateTemplate')}
        </Descriptions.Item>
        <Descriptions.Item label={t('mml.admin.catalog.customized.creator')}>
          {command.creator || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('common.description')}>
          {command.description || '-'}
        </Descriptions.Item>
      </Descriptions>

      <div>
        <Typography.Text strong>
          {t('mml.console.pathPicker.label')} ({paths.length})
        </Typography.Text>
        {paths.length > 0 ? (
          <ul style={{ margin: '8px 0 0', paddingLeft: 20 }}>
            {paths.map((p) => (
              <li key={p}>
                <Typography.Text code>{p}</Typography.Text>
              </li>
            ))}
          </ul>
        ) : (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={t('mml.admin.catalog.customized.noPaths')}
            style={{ margin: '8px 0' }}
          />
        )}
      </div>

      {command.operationType === 'MOD' && (
        <div>
          <Typography.Text strong>
            {t('mml.admin.catalog.customized.parameters')} ({paramEntries.length})
          </Typography.Text>
          {paramEntries.length > 0 ? (
            <Descriptions
              column={1}
              size="small"
              bordered
              style={{ marginTop: 8 }}
              styles={{ label: { width: 240 } }}
            >
              {paramEntries.map(([k, v]) => (
                <Descriptions.Item key={k} label={k}>
                  {String(v)}
                </Descriptions.Item>
              ))}
            </Descriptions>
          ) : (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={t('mml.admin.catalog.customized.noParameters')}
              style={{ margin: '8px 0' }}
            />
          )}
        </div>
      )}
    </div>
  );
}
