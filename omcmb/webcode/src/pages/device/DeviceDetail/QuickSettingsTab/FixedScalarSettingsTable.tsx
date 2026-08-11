import { useMemo, useState } from 'react';
import { Button, Card, Input, Modal, Select, Space, Table, Typography, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EditOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useParameterSchema, useUpdateParameters } from '@core/hooks/api/useDeviceParameters';
import type { ParameterSchemaItem, ParameterType, ParameterUpdateRequest } from '@core/types/deviceParameter';
import type { QuickSettingsGroup, QuickSettingsParam } from '@core/types/quicksettings';
import { formatEnumDisplayValue, resolveQuickSettingsParameterType, validateValue } from './validators';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

type FixedScalarTableKind = 'wan' | 'static-route';

interface FixedScalarSettingsTableProps {
  deviceId: string;
  active?: boolean;
  groups: QuickSettingsGroup[];
  locale: 'zh-CN' | 'en-US';
  kind: FixedScalarTableKind;
  embedded?: boolean;
}

interface TableRow {
  key: string;
  index: number;
  group: QuickSettingsGroup;
}

interface EditState {
  row: TableRow;
  values: Record<string, string>;
  errors: Record<string, string>;
}

function groupMatchesRow(group: QuickSettingsGroup, kind: FixedScalarTableKind, rowIndex: number): boolean {
  if (group.id === `${kind === 'wan' ? 'device-wan' : 'device-static-route'}-${rowIndex}`) return true;
  const prefix = kind === 'wan' ? 'WAN_CONFIG' : 'ROUTE_CONFIG';
  return group.params.some((param) => param.standardPath?.includes(`Device.DeviceInfo.${prefix}${rowIndex}_`));
}

function sortedRows(groups: QuickSettingsGroup[], kind: FixedScalarTableKind): TableRow[] {
  const prefix = kind === 'wan' ? 'device-wan-' : 'device-static-route-';
  return Array.from({ length: 4 }, (_, index) => {
    const rowIndex = index + 1;
    const group = groups.find((candidate) => groupMatchesRow(candidate, kind, rowIndex));
    return {
      key: `${prefix}${rowIndex}`,
      index: rowIndex,
      group: group ?? {
        id: `${prefix}${rowIndex}`,
        titleZh: `${prefix}${rowIndex}`,
        titleEn: `${prefix}${rowIndex}`,
        multiInstance: false,
        params: [],
      },
    };
  });
}

function parameterByName(group: QuickSettingsGroup, name: string): QuickSettingsParam | undefined {
  const suffixes: Record<string, string[]> = {
    Enable: ['_ENABLE', '_ONBOOTENB'],
    IPMode: ['_IPMODE'],
    IPAddress: ['_IPADDR'],
    Netmask: ['_NETMASK'],
    Gateway: ['_DEFAULTGW', '_GW'],
    DestinationNetwork: ['_NETADDR'],
    VLAN: ['_VLAN'],
    Option60: ['_OPTION60'],
  };
  return group.params.find((param) => {
    if (param.name === name) return true;
    const path = param.standardPath ?? '';
    return suffixes[name]?.some((suffix) => path.endsWith(suffix)) ?? false;
  });
}

function editableParams(group: QuickSettingsGroup, kind: FixedScalarTableKind): QuickSettingsParam[] {
  const names = kind === 'wan'
    ? ['Enable', 'IPMode', 'IPAddress', 'Netmask', 'Gateway', 'VLAN', 'Option60']
    : ['Enable', 'DestinationNetwork', 'Netmask', 'Gateway'];
  return names
    .map((name) => parameterByName(group, name))
    .filter((param): param is QuickSettingsParam => Boolean(param));
}

function isStaticIpMode(value: string): boolean {
  const normalized = value.trim().toLowerCase();
  return normalized === '1' || normalized === 'static' || normalized === 'static ip';
}

function visibleEditableParams(
  group: QuickSettingsGroup,
  kind: FixedScalarTableKind,
  values: Record<string, string>,
): QuickSettingsParam[] {
  const params = editableParams(group, kind);
  if (kind !== 'wan') return params;

  const names = isStaticIpMode(values[parameterByName(group, 'IPMode')?.name ?? ''])
    ? ['Enable', 'IPMode', 'IPAddress', 'Netmask', 'Gateway', 'VLAN']
    : ['Enable', 'IPMode', 'Option60', 'VLAN'];
  const visibleNames = new Set(
    names
      .map((name) => parameterByName(group, name)?.name)
      .filter((name): name is string => Boolean(name)),
  );
  return params.filter((param) => visibleNames.has(param.name));
}

function parameterValue(
  param: QuickSettingsParam | undefined,
  schemaByPath: Map<string, ParameterSchemaItem>,
): string {
  if (!param?.standardPath) return '';
  return String(schemaByPath.get(param.standardPath)?.currentValue ?? param.defaultValue ?? '');
}

function parameterType(param: QuickSettingsParam | undefined, schemaByPath: Map<string, ParameterSchemaItem>): ParameterType {
  return resolveQuickSettingsParameterType(param?.type, param?.standardPath ? schemaByPath.get(param.standardPath)?.type : undefined);
}

function parameterLabel(param: QuickSettingsParam, locale: 'zh-CN' | 'en-US'): string {
  return locale === 'zh-CN' ? (param.titleZh || param.titleEn) : param.titleEn;
}

function displayValue(
  param: QuickSettingsParam | undefined,
  value: string,
  schemaByPath: Map<string, ParameterSchemaItem>,
  locale: 'zh-CN' | 'en-US',
): string {
  if (!param) return value || '-';
  const schema = param.standardPath ? schemaByPath.get(param.standardPath) : undefined;
  const constraints = schema?.constraints ?? (param.enumOptions?.length
    ? {
        enumValues: param.enumOptions.map((option) => option.value),
        enumLabels: param.enumOptions.map((option) => option.label),
      }
    : undefined);
  return formatEnumDisplayValue(value, constraints, schema?.path ?? param.standardPath, locale) || '-';
}

export default function FixedScalarSettingsTable({
  deviceId,
  active = true,
  groups,
  locale,
  kind,
  embedded = false,
}: FixedScalarSettingsTableProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const updateMutation = useUpdateParameters();
  const [editState, setEditState] = useState<EditState | null>(null);

  const rows = useMemo(
    () => sortedRows(groups, kind),
    [groups, kind],
  );
  const { data: deviceInfoSchema, refetch: refetchDeviceInfo } = useParameterSchema(
    deviceId,
    'Device.DeviceInfo.',
    active,
  );
  const schemaByPath = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    for (const item of deviceInfoSchema?.parameters ?? []) {
      map.set(item.path, item);
    }
    return map;
  }, [deviceInfoSchema]);

  const openEdit = (row: TableRow) => {
    const values = Object.fromEntries(
      editableParams(row.group, kind).map((param) => [param.name, parameterValue(param, schemaByPath)]),
    );
    setEditState({ row, values, errors: {} });
  };

  const updateEditValue = (name: string, value: string) => {
    setEditState((previous) => previous
      ? { ...previous, values: { ...previous.values, [name]: value }, errors: { ...previous.errors, [name]: '' } }
      : previous);
  };

  const handleSave = async () => {
    if (!editState) return;
    const { row, values } = editState;
    const errors: Record<string, string> = {};
    const updates: ParameterUpdateRequest[] = [];

    for (const param of visibleEditableParams(row.group, kind, values)) {
      const value = values[param.name] ?? '';
      const schema = param.standardPath ? schemaByPath.get(param.standardPath) : undefined;
      const type = parameterType(param, schemaByPath);
      const error = validateValue(value, type, schema?.constraints);
      if (error) errors[param.name] = error;
      if (param.standardPath && value !== parameterValue(param, schemaByPath)) {
        updates.push({
          parameterPath: param.standardPath,
          parameterValue: value,
          parameterType: type,
        });
      }
    }

    if (Object.keys(errors).length > 0) {
      setEditState({ ...editState, errors });
      return;
    }
    if (updates.length === 0) {
      setEditState(null);
      return;
    }

    try {
      await updateMutation.mutateAsync({ deviceId, parameters: updates });
      await refetchDeviceInfo();
      await queryClient.invalidateQueries({ queryKey: ['devices', 'parameters', deviceId] });
      setEditState(null);
      message.success(locale === 'zh-CN' ? '配置已提交' : 'Configuration submitted');
    } catch (error) {
      message.error(String(error));
    }
  };

  const renderField = (param: QuickSettingsParam, value: string, error?: string) => {
    const schema = param.standardPath ? schemaByPath.get(param.standardPath) : undefined;
    const options = param.enumOptions?.map((option) => ({
      value: option.value,
      label: option.label,
    })) ?? schema?.constraints?.enumValues?.map((option, index) => ({
      value: option,
      label: schema.constraints?.enumLabels?.[index] ?? option,
    }));
    const label = parameterLabel(param, locale);
    return (
      <div key={param.name}>
        <div style={{ marginBottom: 6, fontWeight: 500 }}>{label}</div>
        {options && options.length > 0 ? (
          <Select
            aria-label={label}
            value={value || undefined}
            options={options}
            onChange={(next) => updateEditValue(param.name, String(next))}
            status={error ? 'error' : undefined}
            style={{ width: '100%' }}
          />
        ) : (
          <Input
            aria-label={label}
            value={value}
            onChange={(event) => updateEditValue(param.name, event.target.value)}
            status={error ? 'error' : undefined}
          />
        )}
        {error && <Text type="danger" style={{ display: 'block', marginTop: 4 }}>{error}</Text>}
      </div>
    );
  };

  const columns = useMemo<ColumnsType<TableRow>>(() => {
    const effectiveColumn = {
      title: locale === 'zh-CN' ? '是否生效' : 'Effective Or Not',
      key: 'effective',
      width: 145,
      render: (_value: unknown, row: TableRow) => {
        const param = parameterByName(row.group, 'Enable');
        return <Text>{param ? displayValue(param, parameterValue(param, schemaByPath), schemaByPath, locale) : '-'}</Text>;
      },
    };
    const operationColumn = {
      title: t('table.operation'),
      key: 'operation',
      width: 92,
      render: (_value: unknown, row: TableRow) => (
        <Button
          type="link"
          size="small"
          icon={<EditOutlined />}
          aria-label={locale === 'zh-CN' ? `编辑第 ${row.index} 行` : `Edit row ${row.index}`}
          onClick={() => openEdit(row)}
          disabled={updateMutation.isPending}
        />
      ),
    };
    const indexColumn = {
      title: locale === 'zh-CN' ? '索引' : 'Index',
      dataIndex: 'index',
      key: 'index',
      width: 72,
    };

    if (kind === 'wan') {
      return [
        indexColumn,
        operationColumn,
        effectiveColumn,
        {
          title: locale === 'zh-CN' ? 'WAN 名称' : 'WAN Name',
          key: 'wanName',
          width: 140,
          render: (_value: unknown, row: TableRow) => `wanConfig${row.index}`,
        },
        {
          title: locale === 'zh-CN' ? 'IP 接入模式' : 'IP Access Mode',
          key: 'ipMode',
          width: 155,
          render: (_value: unknown, row: TableRow) => {
            const param = parameterByName(row.group, 'IPMode');
            return displayValue(param, parameterValue(param, schemaByPath), schemaByPath, locale);
          },
        },
        {
          title: locale === 'zh-CN' ? 'IP 地址' : 'IP Address',
          key: 'ipAddress',
          width: 150,
          render: (_value: unknown, row: TableRow) => parameterValue(parameterByName(row.group, 'IPAddress'), schemaByPath) || '-',
        },
        {
          title: locale === 'zh-CN' ? '子网掩码' : 'Netmask',
          key: 'netmask',
          width: 150,
          render: (_value: unknown, row: TableRow) => parameterValue(parameterByName(row.group, 'Netmask'), schemaByPath) || '-',
        },
        {
          title: locale === 'zh-CN' ? '网关' : 'Gateway',
          key: 'gateway',
          width: 150,
          render: (_value: unknown, row: TableRow) => parameterValue(parameterByName(row.group, 'Gateway'), schemaByPath) || '-',
        },
      ];
    }

    return [
      indexColumn,
      operationColumn,
      effectiveColumn,
      {
        title: locale === 'zh-CN' ? '目的网络' : 'Destination Network',
        key: 'destinationNetwork',
        width: 220,
        render: (_value: unknown, row: TableRow) => parameterValue(parameterByName(row.group, 'DestinationNetwork'), schemaByPath) || '-',
      },
      {
        title: locale === 'zh-CN' ? '子网掩码' : 'Netmask',
        key: 'netmask',
        width: 180,
        render: (_value: unknown, row: TableRow) => parameterValue(parameterByName(row.group, 'Netmask'), schemaByPath) || '-',
      },
      {
        title: locale === 'zh-CN' ? '隧道网关' : 'Tunnel Gateway',
        key: 'gateway',
        width: 180,
        render: (_value: unknown, row: TableRow) => parameterValue(parameterByName(row.group, 'Gateway'), schemaByPath) || '-',
      },
    ];
  }, [kind, locale, schemaByPath, t, updateMutation.isPending]);

  const title = kind === 'wan'
    ? (locale === 'zh-CN' ? 'WAN 配置' : 'WAN Config')
    : (locale === 'zh-CN' ? '静态路由' : 'Static Routing');
  const note = locale === 'zh-CN'
    ? '修改后需重启，使配置生效。'
    : 'Reboot after changing, to let the configuration to take effect.';
  const tableTitle = (
    <Space size={10}>
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
        <span style={{ color: '#596579', fontSize: 11 }}>▾</span>
        <EditOutlined style={{ color: '#3e638b', fontSize: 14 }} />
        <span>{title}</span>
      </span>
      {kind === 'wan' && <Text type="danger" style={{ marginLeft: 12, fontSize: 12, fontWeight: 400 }}>{note}</Text>}
    </Space>
  );
  const tableContent = (
    <>
      <Table<TableRow>
        rowKey={(row) => row.key}
        dataSource={rows}
        columns={columns}
        loading={!deviceInfoSchema}
        size="small"
        pagination={false}
        scroll={{ x: 'max-content' }}
      />
      <Modal
        title={locale === 'zh-CN' ? `编辑第 ${editState?.row.index ?? ''} 行` : `Edit row ${editState?.row.index ?? ''}`}
        open={Boolean(editState)}
        onOk={() => void handleSave()}
        onCancel={() => setEditState(null)}
        confirmLoading={updateMutation.isPending}
        okText={locale === 'zh-CN' ? '提交' : 'Submit'}
        cancelText={locale === 'zh-CN' ? '取消' : 'Cancel'}
        destroyOnHidden
      >
        {editState && (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 16 }}>
            {visibleEditableParams(editState.row.group, kind, editState.values).map((param) => renderField(param, editState.values[param.name] ?? '', editState.errors[param.name]))}
          </div>
        )}
      </Modal>
    </>
  );

  if (embedded) {
    return (
      <div style={{ width: '100%', marginBottom: 10 }}>
        <div style={{ padding: '6px 0', color: '#273246', fontSize: 12, fontWeight: 600 }}>{tableTitle}</div>
        {tableContent}
      </div>
    );
  }

  return (
    <Card
      title={tableTitle}
      size="small"
      style={{ marginBottom: 10, border: '1px solid #d7dce5', borderRadius: 0, boxShadow: 'none', background: '#fff' }}
    >
      {tableContent}
    </Card>
  );
}
