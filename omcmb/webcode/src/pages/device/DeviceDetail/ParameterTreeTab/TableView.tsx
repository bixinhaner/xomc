import { useState } from 'react';
import { Table, Tag, Typography, Button, Empty } from 'antd';
import { EditOutlined, LoadingOutlined } from '@ant-design/icons';
import type { ColumnsType, TableProps } from 'antd/es/table';
import type { DeviceParameter, ParameterType } from '@core/types/deviceParameter';
import type { PageResponse } from '@core/types/pagination';
import ParameterEditModal from './ParameterEditModal';
import { useT } from '@/hooks/useT';
import { formatSystemTime } from '@core/utils/systemTime';

const { Text } = Typography;

// Virtual table row height constant
const ROW_HEIGHT = 40;
const MAX_VISIBLE_ROWS = 15;
const VIRTUAL_SCROLL_HEIGHT = ROW_HEIGHT * MAX_VISIBLE_ROWS;

interface TableViewProps {
  deviceId: string;
  data: PageResponse<DeviceParameter> | undefined;
  loading: boolean;
  page: number;
  pageSize: number;
  onPageChange: (page: number, pageSize: number) => void;
}

const TYPE_COLOR: Record<string, string> = {
  string: 'blue',
  int: 'green',
  unsignedInt: 'green',
  boolean: 'orange',
  dateTime: 'purple',
  base64: 'cyan',
  hexBinary: 'cyan',
};

interface EditTarget {
  parameterPath: string;
  currentValue: string;
  parameterType: ParameterType;
}

export default function TableView({
  deviceId,
  data,
  loading,
  page,
  pageSize,
  onPageChange,
}: TableViewProps) {
  const t = useT();
  const [editTarget, setEditTarget] = useState<EditTarget | null>(null);

  const columns: ColumnsType<DeviceParameter> = [
    {
      title: t('device.paramTree.colPath'),
      dataIndex: 'parameterPath',
      key: 'parameterPath',
      width: 400,
      ellipsis: true,
      sorter: true,
      render: (v: string) => (
        <Text
          code
          copyable={{ text: v }}
          style={{ fontSize: 12, wordBreak: 'break-all' }}
        >
          {v}
        </Text>
      ),
    },
    {
      title: t('device.paramTree.colValue'),
      dataIndex: 'parameterValue',
      key: 'parameterValue',
      width: 200,
      ellipsis: true,
      render: (v: string) => (
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {v || t('device.paramTree.emptyValue')}
        </Text>
      ),
    },
    {
      title: t('table.type'),
      dataIndex: 'parameterType',
      key: 'parameterType',
      width: 120,
      filters: [
        { text: 'string', value: 'string' },
        { text: 'int', value: 'int' },
        { text: 'unsignedInt', value: 'unsignedInt' },
        { text: 'boolean', value: 'boolean' },
        { text: 'dateTime', value: 'dateTime' },
      ],
      render: (v: string) => (
        <Tag color={TYPE_COLOR[v] ?? 'default'} style={{ fontSize: 11 }}>
          {v}
        </Tag>
      ),
    },
    {
      title: t('device.paramTree.writable'),
      dataIndex: 'writable',
      key: 'writable',
      width: 80,
      filters: [
        { text: t('device.paramTree.writable'), value: true },
        { text: t('device.paramTree.readonly'), value: false },
      ],
      render: (v: boolean) =>
        v ? (
          <Tag color="success">{t('device.paramTree.writable')}</Tag>
        ) : (
          <Tag color="default">{t('device.paramTree.readonly')}</Tag>
        ),
    },
    {
      title: t('device.paramTree.colLastUpdate'),
      dataIndex: 'lastUpdatedAt',
      key: 'lastUpdatedAt',
      width: 170,
      sorter: true,
      render: (v: string) =>
        v ? (
          <Text type="secondary" style={{ fontSize: 12 }}>
            {formatSystemTime(v)}
          </Text>
        ) : (
          '-'
        ),
    },
    {
      title: t('table.operation'),
      key: 'action',
      width: 80,
      fixed: 'right',
      render: (_: unknown, record: DeviceParameter) =>
        record.writable ? (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() =>
              setEditTarget({
                parameterPath: record.parameterPath,
                currentValue: record.parameterValue,
                parameterType: record.parameterType,
              })
            }
          />
        ) : null,
    },
  ];

  if (!data && !loading) {
    return <Empty description={t('device.parameter.noDataHint')} />;
  }

  // Calculate dynamic scroll height and virtual scroll threshold
  const itemCount = data?.items?.length ?? 0;
  const scrollY = Math.min(Math.max(ROW_HEIGHT * itemCount, ROW_HEIGHT * 3), VIRTUAL_SCROLL_HEIGHT);
  const shouldVirtualize = itemCount > MAX_VISIBLE_ROWS;

  const tableProps: TableProps<DeviceParameter> = {
    columns,
    dataSource: data?.items ?? [],
    loading: {
      spinning: loading,
      indicator: <LoadingOutlined spin />,
    },
    rowKey: 'id',
    size: 'small',
    scroll: { x: 1100, y: scrollY },
    pagination: {
      current: page,
      pageSize,
      total: data?.total ?? 0,
      showSizeChanger: true,
      showTotal: (total) => t('device.paramTree.totalParams', { total }),
      pageSizeOptions: ['20', '50', '100', '200'],
      onChange: onPageChange,
    },
    // Enable virtual scroll when data exceeds threshold
    virtual: shouldVirtualize,
    ...(shouldVirtualize && {
      rowProps: () => ({
        style: { height: ROW_HEIGHT },
      }),
    }),
  };

  return (
    <>
      <Table<DeviceParameter> {...tableProps} />

      {editTarget && (
        <ParameterEditModal
          open={Boolean(editTarget)}
          deviceId={deviceId}
          parameterPath={editTarget.parameterPath}
          currentValue={editTarget.currentValue}
          parameterType={editTarget.parameterType}
          onClose={() => setEditTarget(null)}
        />
      )}
    </>
  );
}
