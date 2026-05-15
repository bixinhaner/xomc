import { Drawer, Table, Tag, Spin, Empty, Descriptions } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useParamReferences } from '@core/hooks/api/useMmlAdmin';
import type { ParamAdmin, ParamReference } from '@core/types/mmlAdmin';
import { useT } from '@/hooks/useT';

export interface ParamReferencesDrawerProps {
  open: boolean;
  param?: ParamAdmin;
  onClose: () => void;
}

export default function ParamReferencesDrawer({
  open,
  param,
  onClose,
}: ParamReferencesDrawerProps) {
  const t = useT();
  const { data: refs = [], isLoading } = useParamReferences(param?.id);

  const columns: ColumnsType<ParamReference> = [
    {
      title: t('mml.admin.catalog.params.references.commandCode'),
      dataIndex: 'commandCode',
      key: 'commandCode',
      width: 220,
    },
    {
      title: t('mml.admin.catalog.commands.logicalCode'),
      dataIndex: 'logicalCode',
      key: 'logicalCode',
      width: 160,
    },
    {
      title: t('mml.admin.catalog.commands.op'),
      dataIndex: 'operationType',
      key: 'operationType',
      width: 80,
      render: (op: string) => <Tag>{op}</Tag>,
    },
    {
      title: t('mml.admin.catalog.params.references.groupPath'),
      dataIndex: 'groupPath',
      key: 'groupPath',
      ellipsis: true,
    },
  ];

  return (
    <Drawer
      open={open}
      onClose={onClose}
      width={720}
      title={t('mml.admin.catalog.params.references.title')}
    >
      {param && (
        <Descriptions column={2} size="small" bordered style={{ marginBottom: 16 }}>
          <Descriptions.Item label="param_code">{param.paramCode}</Descriptions.Item>
          <Descriptions.Item label="access_type">
            <Tag>{param.accessType}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="TR-069 path" span={2}>
            {param.tr069Path}
          </Descriptions.Item>
        </Descriptions>
      )}

      {isLoading ? (
        <Spin tip={t('mml.admin.catalog.params.references.loading')} />
      ) : refs.length === 0 ? (
        <Empty description={t('mml.admin.catalog.params.references.empty')} />
      ) : (
        <Table<ParamReference>
          rowKey="commandId"
          columns={columns}
          dataSource={refs}
          size="small"
          pagination={false}
        />
      )}
    </Drawer>
  );
}
