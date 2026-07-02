import { useCallback, useState, useMemo, useRef } from 'react';
import {
  Table,
  Tag,
  Typography,
  Empty,
  Tooltip,
  Space,
  Input,
  Select,
  Button,
  message,
} from 'antd';
import {
  CheckOutlined,
  CloseOutlined,
  FolderOpenOutlined,
  LoadingOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TableProps } from 'antd/es/table';
import type {
  ChildParameter,
  ParameterType,
  ParameterConstraints,
  DirectChildrenResponse,
  SubObjectSummary,
} from '@core/types/deviceParameter';
import { useUpdateParameters } from '@core/hooks/api/useDeviceParameters';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

type TFn = (id: string, values?: Record<string, string | number>) => string;

const TYPE_COLOR: Record<string, string> = {
  string: 'blue',
  int: 'green',
  unsignedInt: 'green',
  boolean: 'orange',
  dateTime: 'purple',
  base64: 'cyan',
  hexBinary: 'cyan',
};

// ---- Validation (extracted from ParameterEditModal) ----

function validateValue(
  value: string,
  parameterType: ParameterType,
  t: TFn,
  constraints?: ParameterConstraints
): string | null {
  if (!value && parameterType !== 'string') return t('device.paramEdit.errRequired');

  if (parameterType === 'int') {
    const num = Number(value);
    if (!Number.isInteger(num)) return t('device.paramEdit.errInteger');
    if (constraints?.minValue !== undefined && num < constraints.minValue)
      return t('device.paramCell.errMin', { min: constraints.minValue });
    if (constraints?.maxValue !== undefined && num > constraints.maxValue)
      return t('device.paramCell.errMax', { max: constraints.maxValue });
  }

  if (parameterType === 'unsignedInt') {
    const num = Number(value);
    if (!Number.isInteger(num) || num < 0) return t('device.paramEdit.errNonNegative');
    if (constraints?.minValue !== undefined && num < constraints.minValue)
      return t('device.paramCell.errMin', { min: constraints.minValue });
    if (constraints?.maxValue !== undefined && num > constraints.maxValue)
      return t('device.paramCell.errMax', { max: constraints.maxValue });
  }

  if (parameterType === 'string' && constraints) {
    if (constraints.maxLength && value.length > constraints.maxLength)
      return t('device.paramCell.errMaxLen', { max: constraints.maxLength });
    if (constraints.minLength && value.length < constraints.minLength)
      return t('device.paramCell.errMinLen', { min: constraints.minLength });
    if (constraints.pattern) {
      try {
        if (!new RegExp(constraints.pattern).test(value))
          return t('device.paramCell.errPattern');
      } catch {
        /* ignore */
      }
    }
  }

  if (constraints?.enumValues?.length && !constraints.enumValues.includes(value))
    return t('device.paramCell.errEnum');

  return null;
}

// ---- Constraints display ----

function formatConstraints(c: ParameterConstraints | undefined, t: TFn): string {
  if (!c) return '-';
  const parts: string[] = [];
  if (c.minValue !== undefined || c.maxValue !== undefined) {
    parts.push(`${c.minValue ?? ''} ~ ${c.maxValue ?? ''}`);
  }
  if (c.minLength !== undefined || c.maxLength !== undefined) {
    if (c.minLength && c.maxLength) parts.push(t('device.paramCell.lenRange', { min: c.minLength, max: c.maxLength }));
    else if (c.maxLength) parts.push(t('device.paramCell.lenMax', { max: c.maxLength }));
    else if (c.minLength) parts.push(t('device.paramCell.lenMin', { min: c.minLength }));
  }
  if (c.enumValues?.length) {
    parts.push(c.enumValues.join(' | '));
  }
  if (c.pattern) parts.push(t('device.paramCell.patternLabel', { pattern: c.pattern }));
  return parts.length > 0 ? parts.join('; ') : '-';
}

// ---- Component ----

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

function getParamName(fullPath: string): string {
  const parts = fullPath.split('.');
  return parts[parts.length - 1] || parts[parts.length - 2] || fullPath;
}

// Virtual table row height constant
const ROW_HEIGHT = 40;
const _HEADER_HEIGHT = 39;
void _HEADER_HEIGHT;
// Max visible rows for virtual scroll area
const MAX_VISIBLE_ROWS = 15;
const VIRTUAL_SCROLL_HEIGHT = ROW_HEIGHT * MAX_VISIBLE_ROWS;

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
  const t = useT();
  const [editingPath, setEditingPath] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);
  const updateMutation = useUpdateParameters();
  const tableContainerRef = useRef<HTMLDivElement>(null);

  // Calculate dynamic scroll height based on data size
  const scrollY = useMemo(() => {
    const itemCount = data?.items?.length ?? 0;
    // If items less than max visible, use actual height; otherwise use max
    const actualHeight = Math.max(ROW_HEIGHT * itemCount, ROW_HEIGHT * 3); // minimum 3 rows
    return Math.min(actualHeight, VIRTUAL_SCROLL_HEIGHT);
  }, [data?.items?.length]);

  // Check if virtual scroll should be enabled (more than threshold items)
  const shouldVirtualize = useMemo(() => {
    return (data?.items?.length ?? 0) > MAX_VISIBLE_ROWS;
  }, [data?.items?.length]);

  const startEdit = useCallback((record: ChildParameter) => {
    setEditingPath(record.parameterPath);
    setEditingValue(record.parameterValue);
    setValidationError(null);
  }, []);

  const cancelEdit = useCallback(() => {
    setEditingPath(null);
    setEditingValue('');
    setValidationError(null);
  }, []);

  const handleSave = useCallback(
    (record: ChildParameter) => {
      const error = validateValue(editingValue, record.parameterType, t, record.constraints);
      if (error) {
        setValidationError(error);
        return;
      }
      if (editingValue === record.parameterValue) {
        cancelEdit();
        return;
      }
      updateMutation.mutate(
        {
          deviceId,
          parameters: [
            {
              parameterPath: record.parameterPath,
              parameterValue: editingValue,
              parameterType: record.parameterType,
            },
          ],
        },
        {
          onSuccess: () => {
            message.success(t('device.paramCell.paramDispatched', { name: getParamName(record.parameterPath) }));
            cancelEdit();
          },
          onError: () => {
            message.error(t('device.paramCell.dispatchFailed'));
          },
        }
      );
    },
    [deviceId, editingValue, updateMutation, cancelEdit, t]
  );

  const handleValueChange = useCallback(
    (value: string, parameterType: ParameterType, constraints?: ParameterConstraints) => {
      setEditingValue(value);
      setValidationError(validateValue(value, parameterType, t, constraints));
    },
    [t]
  );

  const columns: ColumnsType<ChildParameter> = useMemo(() => [
    {
      title: t('device.paramCell.colName'),
      dataIndex: 'parameterPath',
      key: 'name',
      width: 200,
      ellipsis: true,
      render: (v: string, record: ChildParameter) => (
        <Tooltip title={record.description ? `${v}\n${record.description}` : v}>
          <Text code style={{ fontSize: 12 }}>
            {getParamName(v)}
          </Text>
        </Tooltip>
      ),
    },
    {
      title: t('device.paramEdit.currentValue'),
      dataIndex: 'parameterValue',
      key: 'value',
      width: 220,
      render: (_: string, record: ChildParameter) => {
        const isEditing = editingPath === record.parameterPath;

        if (isEditing) {
          if (record.parameterType === 'boolean') {
            return (
              <div>
                <Select
                  size="small"
                  value={editingValue}
                  onChange={(v) => handleValueChange(v, record.parameterType, record.constraints)}
                  options={[
                    { label: 'true', value: 'true' },
                    { label: 'false', value: 'false' },
                  ]}
                  style={{ width: '100%' }}
                  autoFocus
                />
                {validationError && (
                  <Text type="danger" style={{ fontSize: 11 }}>
                    {validationError}
                  </Text>
                )}
              </div>
            );
          }
          if (record.constraints?.enumValues?.length) {
            return (
              <div>
                <Select
                  size="small"
                  value={editingValue}
                  onChange={(v) => handleValueChange(v, record.parameterType, record.constraints)}
                  options={record.constraints.enumValues.map((v, i) => ({ label: record.constraints!.enumLabels?.[i] ?? v, value: v }))}
                  style={{ width: '100%' }}
                  showSearch
                  autoFocus
                />
                {validationError && (
                  <Text type="danger" style={{ fontSize: 11 }}>
                    {validationError}
                  </Text>
                )}
              </div>
            );
          }
          return (
            <div>
              <Input
                size="small"
                value={editingValue}
                onChange={(e) =>
                  handleValueChange(e.target.value, record.parameterType, record.constraints)
                }
                onPressEnter={() => handleSave(record)}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') cancelEdit();
                }}
                status={validationError ? 'error' : undefined}
                autoFocus
              />
              {validationError && (
                <Text type="danger" style={{ fontSize: 11 }}>
                  {validationError}
                </Text>
              )}
            </div>
          );
        }

        return (
          <Text
            style={{
              fontFamily: 'monospace',
              fontSize: 12,
              cursor: record.writable ? 'pointer' : 'default',
              color: record.writable ? '#1677ff' : undefined,
            }}
            onClick={() => record.writable && startEdit(record)}
          >
            {record.parameterValue || t('device.paramTree.emptyValue')}
          </Text>
        );
      },
    },
    {
      title: t('table.type'),
      dataIndex: 'parameterType',
      key: 'type',
      width: 90,
      render: (v: string, record: ChildParameter) => (
        <Space size={2}>
          <Tag color={TYPE_COLOR[v] ?? 'default'} style={{ fontSize: 11, margin: 0 }}>
            {v}
          </Tag>
          {record.writable && (
            <Tag color="success" style={{ fontSize: 10, margin: 0 }}>
              W
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: t('device.paramEdit.defaultValue'),
      dataIndex: 'defaultValue',
      key: 'defaultValue',
      width: 120,
      ellipsis: true,
      render: (v: string) =>
        v ? (
          <Text type="secondary" style={{ fontFamily: 'monospace', fontSize: 11 }}>
            {v}
          </Text>
        ) : (
          <Text type="secondary" style={{ fontSize: 11 }}>
            -
          </Text>
        ),
    },
    {
      title: t('device.paramCell.colRange'),
      key: 'constraints',
      width: 160,
      ellipsis: true,
      render: (_: unknown, record: ChildParameter) => {
        const text = formatConstraints(record.constraints, t);
        return text !== '-' ? (
          <Tooltip title={text}>
            <Text type="secondary" style={{ fontSize: 11 }}>
              {text}
            </Text>
          </Tooltip>
        ) : (
          <Text type="secondary" style={{ fontSize: 11 }}>
            -
          </Text>
        );
      },
    },
    {
      title: t('table.operation'),
      key: 'action',
      width: 80,
      fixed: 'right',
      render: (_: unknown, record: ChildParameter) => {
        const isEditing = editingPath === record.parameterPath;
        if (isEditing) {
          return (
            <Space size={4}>
              <Button
                type="primary"
                size="small"
                icon={<CheckOutlined />}
                loading={updateMutation.isPending}
                disabled={Boolean(validationError)}
                onClick={() => handleSave(record)}
              />
              <Button
                size="small"
                icon={<CloseOutlined />}
                onClick={cancelEdit}
              />
            </Space>
          );
        }
        if (!record.writable) return null;
        return (
          <Button
            type="link"
            size="small"
            style={{ padding: 0 }}
            onClick={() => startEdit(record)}
          >
            {t('common.edit')}
          </Button>
        );
      },
    },
  ], [editingPath, editingValue, validationError, updateMutation.isPending, startEdit, cancelEdit, handleSave, handleValueChange, t]);

  if (!pathPrefix) {
    return <Empty description={t('device.paramCell.selectNode')} style={{ padding: 48 }} />;
  }

  const subObjects = data?.subObjects ?? [];
  const hasSubObjects = subObjects.length > 0;
  const hasLeaves = (data?.items?.length ?? 0) > 0;
  const totalItems = data?.total ?? 0;

  // Table props for virtual scroll
  const tableProps: TableProps<ChildParameter> = {
    columns,
    dataSource: data?.items ?? [],
    loading: {
      spinning: loading,
      indicator: <LoadingOutlined spin />,
    },
    rowKey: 'parameterPath',
    size: 'small',
    scroll: { x: 950, y: scrollY },
    pagination: {
      current: page,
      pageSize,
      total: totalItems,
      showSizeChanger: true,
      showTotal: (total) => t('device.paramTree.totalParams', { total }),
      pageSizeOptions: ['20', '50', '100', '200'],
      onChange: onPageChange,
    },
    // Enable virtual scroll when data exceeds threshold
    virtual: shouldVirtualize,
    // Row props for virtual scroll - fixed height is required
    ...(shouldVirtualize && {
      rowProps: () => ({
        style: { height: ROW_HEIGHT },
      }),
    }),
  };

  return (
    <div ref={tableContainerRef}>
      {/* Path header */}
      <div style={{ marginBottom: 8 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>
          {t('device.paramCell.pathLabel')}:{' '}
        </Text>
        <Text code style={{ fontSize: 12 }}>
          {pathPrefix}
        </Text>
        {(hasSubObjects || hasLeaves) && (
          <Text type="secondary" style={{ fontSize: 12, marginLeft: 12 }}>
            {hasSubObjects ? t('device.paramCell.subObjectCount', { count: subObjects.length }) : ''}
            {hasSubObjects && hasLeaves ? ', ' : ''}
            {hasLeaves ? t('device.paramCell.paramCount', { count: totalItems }) : ''}
          </Text>
        )}
      </div>

      {/* Sub-object navigation */}
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
              style={{ cursor: 'pointer', padding: '4px 10px', fontSize: 13 }}
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

      {/* Leaf parameter table with virtual scroll */}
      {hasLeaves ? (
        <Table<ChildParameter> {...tableProps} />
      ) : !hasSubObjects && !loading ? (
        <Empty description={t('device.paramCell.noDirectParams')} style={{ padding: 24 }} />
      ) : null}

      {/* Performance hint for large datasets */}
      {shouldVirtualize && (
        <div style={{ marginTop: 8, textAlign: 'right' }}>
          <Text type="secondary" style={{ fontSize: 11 }}>
            {t('device.paramCell.virtualScrollEnabled')}
          </Text>
        </div>
      )}
    </div>
  );
}
