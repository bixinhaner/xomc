import { useEffect, useMemo, useState } from 'react';
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Collapse, ConfigProvider, Form, Input, Modal, Select, Spin, Table, Typography } from 'antd';
import { useIntl } from 'react-intl';
import type { QuickSettingsGroup, QuickSettingsParam } from '@core/types/quicksettings';
import { quicksettingsApi } from '@core/services/api/quicksettingsApi';
import { useT } from '@/hooks/useT';

const { Text } = Typography;
const FIXED_NETWORK_TABLE_MODELS = new Set(['BLN', 'BLQ', 'MLN', 'MLQ']);

function isNetworkGroup(group: QuickSettingsGroup): boolean {
  return /(?:network|interface|wan|lan|route|dscp)/i.test(group.id)
    && !/(?:ipsec|dscp|static-route)/i.test(group.id);
}

function concretePath(template: string, instances: number[]): string {
  let index = 0;
  return template.replace(/\{[ij]\}/g, () => String(instances[index++] ?? 1));
}

function fieldPath(group: QuickSettingsGroup, param: QuickSettingsParam, instances: number[]): string {
  if (param.standardPath) return concretePath(param.standardPath, instances);
  return `${concretePath(group.objectPath ?? '', instances)}${param.leaf ?? param.name}`;
}

function ParamControl({ param }: { param: QuickSettingsParam }) {
  const options = param.enumOptions?.map((option) => ({ value: option.value, label: option.label }));
  return options?.length ? <Select options={options} /> : <Input />;
}

function ParamGrid({ group, instances, includeInterfaceName = false }: {
  group: QuickSettingsGroup;
  instances: number[];
  includeInterfaceName?: boolean;
}) {
  const t = useT();
  const intl = useIntl();
  const writable = group.params.filter((param) => !param.readonly);
  const params = includeInterfaceName && !writable.some((param) => param.name === 'Name')
    ? [{ name: 'Name', titleZh: t('provision.network.interfaceName'), titleEn: t('provision.network.interfaceName'), leaf: 'Name' } as QuickSettingsParam, ...writable]
    : writable;
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 16 }}>
      {params.map((param) => {
        const path = fieldPath(group, param, instances);
        return (
          <Form.Item
            key={path}
            name={['networkParameterValues', path]}
            label={intl.locale === 'en-US' ? (param.titleEn || param.name) : (param.titleZh || param.name)}
            preserve={false}
            rules={param.name === 'Name' && includeInterfaceName ? [{ required: true }] : undefined}
          >
            <ParamControl param={param} />
          </Form.Item>
        );
      })}
    </div>
  );
}

function FixedNetworkTable({ groups, kind, locale }: {
  groups: QuickSettingsGroup[];
  kind: 'wan' | 'static-route';
  locale: string;
}) {
  const t = useT();
  const [editing, setEditing] = useState<QuickSettingsGroup>();
  const parameterValues = Form.useWatch('networkParameterValues') as Record<string, unknown> | undefined;
  const prefix = kind === 'wan' ? 'device-wan-' : 'device-static-route-';
  const rows = groups
    .filter((group) => group.id.startsWith(prefix))
    .sort((left, right) => Number(left.id.slice(prefix.length)) - Number(right.id.slice(prefix.length)));
  if (rows.length === 0) return null;
  const title = kind === 'wan'
    ? t('provision.network.wanConfig')
    : t('provision.network.staticRouting');
  const representative = rows[0].params.filter((param) => !param.readonly).slice(0, kind === 'wan' ? 4 : 3);
  return (
    <Card size="small" title={title} style={{ marginBottom: 16 }}>
      <Table
        rowKey="id"
        size="small"
        pagination={false}
        scroll={{ x: 'max-content', y: kind === 'wan' ? 240 : undefined }}
        dataSource={rows}
        columns={[
          { title: t('provision.network.instance'), width: 90, render: (_value, row) => Number(row.id.slice(prefix.length)) },
          ...representative.map((param) => ({
            title: locale.startsWith('zh') ? (param.titleZh || param.name) : (param.titleEn || param.name),
            key: param.name,
            width: 180,
            render: (_value: unknown, row: QuickSettingsGroup) => {
              const rowParam = row.params.find((candidate) => candidate.name === param.name);
              if (!rowParam) return '-';
              const value = parameterValues?.[fieldPath(row, rowParam, [])];
              return value == null || value === '' ? '-' : String(value);
            },
          })),
          {
            title: t('common.action'),
            key: 'action',
            fixed: 'right' as const,
            width: 100,
            render: (_value: unknown, row: QuickSettingsGroup) => (
              <Button type="link" icon={<EditOutlined />} onClick={() => setEditing(row)}>
                {t('common.edit')}
              </Button>
            ),
          },
        ]}
      />
      <Modal
        open={Boolean(editing)}
        title={t('provision.network.editInstance', { index: editing ? Number(editing.id.slice(prefix.length)) : '' })}
        footer={null}
        onCancel={() => setEditing(undefined)}
        width={900}
        destroyOnHidden
      >
        {editing && <ParamGrid group={editing} instances={[]} />}
      </Modal>
    </Card>
  );
}

function ObjectGroup({
  group,
  groups,
  ancestors,
  locale,
  onRequestEdit,
}: {
  group: QuickSettingsGroup;
  groups: QuickSettingsGroup[];
  ancestors: number[];
  locale: string;
  onRequestEdit?: () => void;
}) {
  const t = useT();
  const instanceKey = `${group.id}@${ancestors.join('.') || 'root'}`;
  const maximum = group.maxInstances && group.maxInstances > 0 ? group.maxInstances : 1;
  const children = groups.filter((candidate) => candidate.parentSelector === group.id);
  const title = locale.startsWith('zh') ? group.titleZh : group.titleEn;
  return (
    <Form.List name={['networkObjectInstances', instanceKey]}>
      {(fields, { add, remove }) => (
        <>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <Text type="secondary">{fields.length}/{maximum}</Text>
            <Button icon={<PlusOutlined />} disabled={fields.length >= maximum} onClick={() => add({})}>
              {t('common.add')}
            </Button>
          </div>
          {fields.map(({ key, name }) => {
            const instances = [...ancestors, name + 1];
            return (
              <Card
                key={key}
                size="small"
                title={`${title} ${name + 1}`}
                extra={(
                  <ConfigProvider componentDisabled={onRequestEdit ? false : undefined}>
                    <Button
                      type="text"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={() => {
                        onRequestEdit?.();
                        remove(name);
                      }}
                    />
                  </ConfigProvider>
                )}
                style={{ marginBottom: 12 }}
              >
                <ParamGrid group={group} instances={instances} includeInterfaceName={group.id === 'gnb-network-interface'} />
                {children.length > 0 && (
                  <Collapse items={children.map((child) => ({
                    key: child.id,
                    label: locale.startsWith('zh') ? child.titleZh : child.titleEn,
                    children: child.multiInstance
                      ? <ObjectGroup group={child} groups={groups} ancestors={instances} locale={locale} onRequestEdit={onRequestEdit} />
                      : <ParamGrid group={child} instances={instances} />,
                  }))} />
                )}
              </Card>
            );
          })}
        </>
      )}
    </Form.List>
  );
}

export default function CommonQuickSettingsNetworkCards({
  paramModelName,
  onRequestEdit,
}: {
  paramModelName?: string;
  onRequestEdit?: () => void;
}) {
  const t = useT();
  const intl = useIntl();
  const [groups, setGroups] = useState<QuickSettingsGroup[]>([]);
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);
  const locale = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';
  useEffect(() => {
    let active = true;
    if (!paramModelName) {
      setGroups([]);
      return undefined;
    }
    setLoading(true);
    setFailed(false);
    void quicksettingsApi.getGroupsByParamModel(paramModelName)
      .then((response) => { if (active) setGroups(response.groups.filter(isNetworkGroup)); })
      .catch(() => { if (active) setFailed(true); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [paramModelName]);
  const roots = useMemo(() => groups.filter((group) => !group.parentSelector), [groups]);
  const fixedTableMode = FIXED_NETWORK_TABLE_MODELS.has((paramModelName ?? '').toUpperCase());
  const displayedRoots = fixedTableMode
    ? roots.filter((group) => !/^device-(?:wan|static-route)-\d+$/.test(group.id))
    : roots;
  if (!paramModelName || (!loading && !failed && roots.length === 0)) return null;
  if (loading) return <Card size="small" style={{ marginBottom: 16 }}><Spin /></Card>;
  if (failed) return <Alert type="warning" showIcon title={t('empty.loadFailed')} style={{ marginBottom: 16 }} />;
  return (
    <>
      {displayedRoots.map((group) => (
        <Card
          key={group.id}
          size="small"
          title={locale.startsWith('zh') ? group.titleZh : group.titleEn}
          style={{ marginBottom: 16 }}
        >
          {group.multiInstance
            ? <ObjectGroup group={group} groups={groups} ancestors={[]} locale={locale} onRequestEdit={onRequestEdit} />
            : <ParamGrid group={group} instances={[]} />}
        </Card>
      ))}
      {fixedTableMode && <FixedNetworkTable groups={groups} kind="wan" locale={locale} />}
      {fixedTableMode && <FixedNetworkTable groups={groups} kind="static-route" locale={locale} />}
    </>
  );
}
