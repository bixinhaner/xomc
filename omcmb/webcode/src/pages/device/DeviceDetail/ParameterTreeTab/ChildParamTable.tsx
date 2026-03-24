import React, { useState } from 'react';
import { Table, Tag, Typography, Empty, Tooltip, Space } from 'antd';
import { EditOutlined, FolderOpenOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type {
  ChildParameter,
  ParameterType,
  ParameterConstraints,
  DirectChildrenResponse,
  SubObjectSummary,
} from '@/types/deviceParameter';
import ParameterEditModal from './ParameterEditModal';

const { Text, Link } = Typography;

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
  constraints?: ParameterConstraints;
  description?: string;
  changeApplies?: string;
  defaultValue?: string;
}

interface ChildParamTableProps {
  deviceId: string;
  pathPrefix: string;
  data: DirectChildrenResponse | undefined;
  loading: boolean;
  page: number;
  pageSize: number;
  onPageChange: (page: number, pageSize: number) => void;
  onNavigate: (path: string) => void;
}

/** Extract the last segment of a dotted parameter path as the display name */
function getParamName(fullPath: string): string {
  const parts = fullPath.split('.');
  return parts[parts.length - 1] || parts[parts.length - 2] || fullPath;
}

export default function ChildParamTable({
  deviceId,
  pathPrefix,
  data,
  loading,
  page,
  pageSize,
  onPageChange,
  onNavigate,
}: ChildParamTableProps) {
  const [editTarget, setEditTarget] = useState<EditTarget | null>(null);

  const columns: ColumnsType<ChildParameter> = [
    {
      title: '参数名',
      dataIndex: 'parameterPath',
      key: 'name',
      width: 250,
      ellipsis: true,
      render: (v: string) => (
        <Tooltip title={v}>
          <Text code style={{ fontSize: 12 }}>
            {getParamName(v)}
          </Text>
        </Tooltip>
      ),
    },
    {
      title: '值',
      dataIndex: 'parameterValue',
      key: 'value',
      width: 200,
      ellipsis: true,
      render: (v: string) => (
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {v || '(空)'}
        </Text>
      ),
    },
    {
      title: '类型',
      dataIndex: 'parameterType',
      key: 'type',
      width: 100,
      render: (v: string) => (
        <Tag color={TYPE_COLOR[v] ?? 'default'} style={{ fontSize: 11 }}>
          {v}
        </Tag>
      ),
    },
    {
      title: '可写',
      dataIndex: 'writable',
      key: 'writable',
      width: 70,
      render: (v: boolean) =>
        v ? (
          <Tag color="success">可写</Tag>
        ) : (
          <Tag color="default">只读</Tag>
        ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 150,
      ellipsis: true,
      render: (v: string) =>
        v ? (
          <Tooltip title={v}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {v}
            </Text>
          </Tooltip>
        ) : (
          '-'
        ),
    },
    {
      title: '操作',
      key: 'action',
      width: 60,
      fixed: 'right',
      render: (_: unknown, record: ChildParameter) =>
        record.writable ? (
          <EditOutlined
            style={{ color: '#1677ff', cursor: 'pointer' }}
            onClick={() =>
              setEditTarget({
                parameterPath: record.parameterPath,
                currentValue: record.parameterValue,
                parameterType: record.parameterType,
                constraints: record.constraints,
                description: record.description,
                changeApplies: record.changeApplies,
                defaultValue: record.defaultValue,
              })
            }
          />
        ) : null,
    },
  ];

  if (!pathPrefix) {
    return (
      <Empty
        description="请在左侧选择一个参数节点"
        style={{ padding: 48 }}
      />
    );
  }

  const subObjects = data?.subObjects ?? [];
  const hasSubObjects = subObjects.length > 0;
  const hasLeaves = (data?.items?.length ?? 0) > 0;

  return (
    <>
      <div style={{ marginBottom: 8 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>
          路径:{' '}
        </Text>
        <Text code style={{ fontSize: 12 }}>
          {pathPrefix}
        </Text>
        {hasSubObjects && (
          <Text type="secondary" style={{ fontSize: 12, marginLeft: 12 }}>
            {subObjects.length} 个子对象, {data?.total ?? 0} 个参数
          </Text>
        )}
      </div>

      {/* Sub-object navigation cards */}
      {hasSubObjects && (
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: 8,
            marginBottom: 12,
            padding: '8px 0',
            borderBottom: hasLeaves ? '1px solid #f0f0f0' : undefined,
          }}
        >
          {subObjects.map((obj: SubObjectSummary) => (
            <Tag
              key={obj.fullPath}
              style={{
                cursor: 'pointer',
                padding: '4px 10px',
                fontSize: 13,
              }}
              color="processing"
              onClick={() => onNavigate(obj.fullPath + '.')}
            >
              <Space size={4}>
                <FolderOpenOutlined />
                <span>{obj.name}</span>
                <Text type="secondary" style={{ fontSize: 11 }}>
                  ({obj.childCount})
                </Text>
              </Space>
            </Tag>
          ))}
        </div>
      )}

      {/* Leaf parameter table */}
      {hasLeaves ? (
        <Table<ChildParameter>
          columns={columns}
          dataSource={data?.items ?? []}
          loading={loading}
          rowKey="parameterPath"
          size="small"
          scroll={{ x: 900 }}
          pagination={{
            current: page,
            pageSize,
            total: data?.total ?? 0,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条参数`,
            pageSizeOptions: ['20', '50', '100'],
            onChange: onPageChange,
          }}
        />
      ) : !hasSubObjects && !loading ? (
        <Empty description="该节点下没有直接参数" style={{ padding: 24 }} />
      ) : null}

      {editTarget && (
        <ParameterEditModal
          open={Boolean(editTarget)}
          deviceId={deviceId}
          parameterPath={editTarget.parameterPath}
          currentValue={editTarget.currentValue}
          parameterType={editTarget.parameterType}
          constraints={editTarget.constraints}
          description={editTarget.description}
          changeApplies={editTarget.changeApplies}
          defaultValue={editTarget.defaultValue}
          onClose={() => setEditTarget(null)}
        />
      )}
    </>
  );
}
