import { useEffect, useMemo, useState } from 'react';
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Collapse, ConfigProvider, Form, Input, Modal, Select, Spin, Table, Typography } from 'antd';
import { useIntl } from 'react-intl';
import type { QuickSettingsGroup, QuickSettingsParam } from '@core/types/quicksettings';
import { quicksettingsApi } from '@core/services/api/quicksettingsApi';
import { useT } from '@/hooks/useT';

const { Text } = Typography;
const FIXED_NETWORK_TABLE_MODELS = new Set(['BLN', 'BLQ', 'MLN', 'MLQ']);

interface NetworkGroupCacheEntry {
  groups?: QuickSettingsGroup[];
  promise?: Promise<QuickSettingsGroup[]>;
}

const networkGroupCache = new Map<string, NetworkGroupCacheEntry>();

function isNetworkGroup(group: QuickSettingsGroup): boolean {
  return /(?:network|interface|wan|lan|route|dscp)/i.test(group.id)
    && !/(?:ipsec|dscp|static-route)/i.test(group.id);
}

function paramModelCacheKey(paramModelName: string | undefined): string | undefined {
  const key = paramModelName?.trim();
  return key ? key : undefined;
}

function cachedNetworkGroups(paramModelName: string | undefined): QuickSettingsGroup[] | undefined {
  const key = paramModelCacheKey(paramModelName);
  return key ? networkGroupCache.get(key)?.groups : undefined;
}

function loadNetworkGroups(paramModelName: string): Promise<QuickSettingsGroup[]> {
  const cached = networkGroupCache.get(paramModelName);
  if (cached?.groups) return Promise.resolve(cached.groups);
  if (cached?.promise) return cached.promise;
  const promise = quicksettingsApi.getGroupsByParamModel(paramModelName)
    .then((response) => response.groups.filter(isNetworkGroup));
  networkGroupCache.set(paramModelName, { promise });
  void promise.then(
    (groups) => networkGroupCache.set(paramModelName, { groups }),
    () => networkGroupCache.delete(paramModelName),
  );
  return promise;
}

function concretePath(template: string, instances: number[]): string {
  let index = 0;
  return template.replace(/\{[ij]\}/g, () => String(instances[index++] ?? 1));
}

function escapeRegExp(input: string): string {
  return input.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function templatePrefixMatcher(template: string): RegExp {
  const parts = template.split(/\{[ij]\}/g);
  const expression = parts.map(escapeRegExp).join('(\\d+)');
  return new RegExp(`^${expression}`);
}

function fieldPath(group: QuickSettingsGroup, param: QuickSettingsParam, instances: number[]): string {
  if (param.standardPath) return concretePath(param.standardPath, instances);
  return `${concretePath(group.objectPath ?? '', instances)}${param.leaf ?? param.name}`;
}

function interfaceNameSheetFieldName(
  group: QuickSettingsGroup,
  param: QuickSettingsParam,
  instances: number[],
): Array<string | number> | undefined {
  if (group.id !== 'gnb-network-interface' || param.name !== 'Name') return undefined;
  const rowIndex = (instances[0] ?? 1) - 1;
  return ['sheetParameters', 'INTERFACE', rowIndex, 'Interface Name'];
}

function canonicalHeader(input: unknown): string {
  return String(input ?? '').trim().replace(/^\*/, '').replace(/[\s_]+/g, '').toUpperCase();
}

function leafFromPath(path: string): string {
  return path.split('.').filter(Boolean).at(-1) ?? '';
}

function firstSheetValueByHeader(
  sheets: Record<string, Array<Record<string, unknown>>> | undefined,
  headers: readonly string[],
): unknown {
  const expected = new Set(headers.map(canonicalHeader).filter(Boolean));
  if (expected.size === 0) return undefined;
  for (const rows of Object.values(sheets ?? {})) {
    for (const row of rows) {
      for (const [header, value] of Object.entries(row)) {
        if (!expected.has(canonicalHeader(header))) continue;
        if (value !== undefined && value !== null && String(value).trim() !== '') return value;
      }
    }
  }
  return undefined;
}

function sheetFieldNameByHeader(
  sheets: Record<string, Array<Record<string, unknown>>> | undefined,
  headers: readonly string[],
): Array<string | number> | undefined {
  const expected = new Set(headers.map(canonicalHeader).filter(Boolean));
  if (expected.size === 0) return undefined;
  for (const [sheetName, rows] of Object.entries(sheets ?? {})) {
    for (let rowIndex = 0; rowIndex < rows.length; rowIndex += 1) {
      const header = Object.keys(rows[rowIndex]).find((candidate) => expected.has(canonicalHeader(candidate)));
      if (header) return ['sheetParameters', sheetName, rowIndex, header];
    }
  }
  return undefined;
}

function candidateHeaders(group: QuickSettingsGroup, param: QuickSettingsParam, path: string): string[] {
  return [
    param.name,
    param.leaf ?? '',
    param.titleZh,
    param.titleEn,
    leafFromPath(path),
    group.titleZh && param.titleZh ? `${group.titleZh} ${param.titleZh}` : '',
    group.titleEn && param.titleEn ? `${group.titleEn} ${param.titleEn}` : '',
  ].filter(Boolean);
}

function inferredNetworkParameterValues(
  groups: readonly QuickSettingsGroup[],
  sheets: Record<string, Array<Record<string, unknown>>> | undefined,
): Record<string, unknown> {
  const values: Record<string, unknown> = {};
  for (const group of groups) {
    if (group.multiInstance) continue;
    for (const param of group.params) {
      if (param.readonly) continue;
      const path = fieldPath(group, param, []);
      const imported = firstSheetValueByHeader(sheets, candidateHeaders(group, param, path));
      if (imported !== undefined) values[path] = imported;
    }
  }
  return values;
}

function inferredNetworkObjectInstances(
  groups: readonly QuickSettingsGroup[],
  parameterValues: Record<string, unknown> | undefined,
  current: Record<string, unknown[]> | undefined,
): Record<string, unknown[]> {
  const next = { ...(current ?? {}) };
  let changed = false;
  const paths = Object.entries(parameterValues ?? {})
    .filter(([, value]) => value !== undefined && value !== null && String(value).trim() !== '')
    .map(([path]) => path);

  for (const group of groups) {
    if (!group.multiInstance || !group.objectPath) continue;
    const placeholderCount = (group.objectPath.match(/\{[ij]\}/g) ?? []).length;
    if (placeholderCount === 0) continue;
    const matcher = templatePrefixMatcher(group.objectPath);
    const countsByKey = new Map<string, number>();
    for (const path of paths) {
      const match = matcher.exec(path);
      if (!match) continue;
      const instances = match.slice(1).map((item) => Number(item)).filter((item) => Number.isFinite(item) && item > 0);
      if (instances.length !== placeholderCount) continue;
      const ancestors = instances.slice(0, -1);
      const ownIndex = instances.at(-1) ?? 0;
      const instanceKey = `${group.id}@${ancestors.join('.') || 'root'}`;
      countsByKey.set(instanceKey, Math.max(countsByKey.get(instanceKey) ?? 0, ownIndex));
    }
    for (const [instanceKey, count] of countsByKey) {
      const existing = Array.isArray(next[instanceKey]) ? next[instanceKey] : [];
      if (existing.length >= count) continue;
      next[instanceKey] = [
        ...existing,
        ...Array.from({ length: count - existing.length }, () => ({})),
      ];
      changed = true;
    }
  }

  return changed ? next : (current ?? {});
}

function ParamControl({
  param,
  readOnly = false,
  value,
  onChange,
  onValueChange,
  onRequestEdit,
}: {
  param: QuickSettingsParam;
  readOnly?: boolean;
  value?: unknown;
  onChange?: (value: unknown) => void;
  onValueChange?: (value: unknown) => void;
  onRequestEdit?: () => void;
}) {
  const options = param.enumOptions?.map((option) => ({ value: option.value, label: option.label }));
  if (readOnly && options?.length) {
    return <Input value={options.find((option) => option.value === value)?.label ?? String(value ?? '')} readOnly onFocus={onRequestEdit} />;
  }
  return options?.length
    ? <Select options={options} value={value as string | undefined} onChange={(nextValue) => { onRequestEdit?.(); onChange?.(nextValue); onValueChange?.(nextValue); }} />
    : (
      <Input
        readOnly={readOnly}
        value={value == null ? '' : String(value)}
        onFocus={readOnly ? onRequestEdit : undefined}
        onChange={(event) => {
          const nextValue = event.target.value;
          if (!readOnly) onRequestEdit?.();
          onChange?.(nextValue);
          onValueChange?.(nextValue);
        }}
      />
    );
}

function AbsoluteNetworkParameterItem({
  path,
  label,
  param,
  readOnly,
  onRequestEdit,
}: {
  path: string;
  label: string;
  param: QuickSettingsParam;
  readOnly: boolean;
  onRequestEdit?: () => void;
}) {
  const form = Form.useFormInstance();
  const namePath = ['networkParameterValues', path];
  const watchedValue = Form.useWatch(namePath, { form, preserve: true });
  const value = watchedValue ?? form.getFieldValue(namePath);
  const options = param.enumOptions?.map((option) => ({ value: option.value, label: option.label }));
  const control = options?.length
    ? (
      readOnly
        ? <Input value={options.find((option) => option.value === value)?.label ?? String(value ?? '')} readOnly onFocus={onRequestEdit} />
        : (
          <Select
            options={options}
            value={value as string | undefined}
            onChange={(nextValue) => { onRequestEdit?.(); form.setFieldValue(namePath, nextValue); }}
          />
        )
    )
    : (
      <Input
        readOnly={readOnly}
        value={value == null ? '' : String(value)}
        onFocus={readOnly ? onRequestEdit : undefined}
        onChange={(event) => {
          if (!readOnly) onRequestEdit?.();
          form.setFieldValue(namePath, event.target.value);
        }}
      />
    );
  return (
    <Form.Item
      label={label}
      preserve
    >
      {control}
    </Form.Item>
  );
}

function SheetParameterItem({
  namePath,
  label,
  param,
  readOnly,
  onRequestEdit,
}: {
  namePath: Array<string | number>;
  label: string;
  param: QuickSettingsParam;
  readOnly: boolean;
  onRequestEdit?: () => void;
}) {
  const form = Form.useFormInstance();
  const watchedValue = Form.useWatch(namePath, { form, preserve: true });
  const value = watchedValue ?? form.getFieldValue(namePath);
  const options = param.enumOptions?.map((option) => ({ value: option.value, label: option.label }));
  const setValue = (nextValue: unknown) => {
    form.setFieldValue(namePath, nextValue);
  };
  const control = options?.length
    ? (
      readOnly
        ? <Input value={options.find((option) => option.value === value)?.label ?? String(value ?? '')} readOnly onFocus={onRequestEdit} />
        : (
          <Select
            options={options}
            value={value as string | undefined}
            onChange={(nextValue) => { onRequestEdit?.(); setValue(nextValue); }}
          />
        )
    )
    : (
      <Input
        readOnly={readOnly}
        value={value == null ? '' : String(value)}
        onFocus={readOnly ? onRequestEdit : undefined}
        onChange={(event) => {
          if (!readOnly) onRequestEdit?.();
          setValue(event.target.value);
        }}
        onInput={(event) => {
          if (!readOnly) setValue((event.target as HTMLInputElement).value);
        }}
        onBlur={(event) => {
          if (!readOnly) setValue(event.target.value);
        }}
      />
    );
  return (
    <Form.Item
      label={label}
      preserve
    >
      {control}
    </Form.Item>
  );
}

function ParamGrid({ group, instances, includeInterfaceName = false, readOnly = false, onRequestEdit }: {
  group: QuickSettingsGroup;
  instances: number[];
  includeInterfaceName?: boolean;
  readOnly?: boolean;
  onRequestEdit?: () => void;
}) {
  const t = useT();
  const intl = useIntl();
  const form = Form.useFormInstance();
  const sheetParameters = Form.useWatch('sheetParameters', form) as
    | Record<string, Array<Record<string, unknown>>>
    | undefined;
  const writable = group.params.filter((param) => !param.readonly);
  const params = includeInterfaceName && !writable.some((param) => param.name === 'Name')
    ? [{ name: 'Name', titleZh: t('provision.network.interfaceName'), titleEn: t('provision.network.interfaceName'), leaf: 'Name' } as QuickSettingsParam, ...writable]
    : writable;
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 16 }}>
      {params.map((param) => {
        const path = fieldPath(group, param, instances);
        const interfaceNameFieldName = interfaceNameSheetFieldName(group, param, instances);
        const importedName = group.multiInstance
          ? undefined
          : sheetFieldNameByHeader(sheetParameters, candidateHeaders(group, param, path));
        const label = intl.locale === 'en-US' ? (param.titleEn || param.name) : (param.titleZh || param.name);
        if (interfaceNameFieldName) {
          return (
            <SheetParameterItem
              key={path}
              namePath={interfaceNameFieldName}
              label={label}
              param={param}
              readOnly={readOnly}
              onRequestEdit={onRequestEdit}
            />
          );
        }
        if (group.multiInstance) {
          return (
            <AbsoluteNetworkParameterItem
              key={path}
              path={path}
              label={label}
              param={param}
              readOnly={readOnly}
              onRequestEdit={onRequestEdit}
            />
          );
        }
        return (
          <Form.Item
            key={path}
            name={importedName ?? ['networkParameterValues', path]}
            label={label}
            preserve={false}
          >
            <ParamControl param={param} readOnly={readOnly} onRequestEdit={onRequestEdit} />
          </Form.Item>
        );
      })}
    </div>
  );
}

function FixedNetworkTable({ groups, kind, locale, readOnly = false }: {
  groups: QuickSettingsGroup[];
  kind: 'wan' | 'static-route';
  locale: string;
  readOnly?: boolean;
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
              <Button type="link" icon={<EditOutlined />} disabled={readOnly} onClick={() => setEditing(row)}>
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
        {editing && <ParamGrid group={editing} instances={[]} readOnly={readOnly} />}
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
  readOnly = false,
}: {
  group: QuickSettingsGroup;
  groups: QuickSettingsGroup[];
  ancestors: number[];
  locale: string;
  onRequestEdit?: () => void;
  readOnly?: boolean;
}) {
  const t = useT();
  const form = Form.useFormInstance();
  const instanceKey = `${group.id}@${ancestors.join('.') || 'root'}`;
  const maximum = group.maxInstances && group.maxInstances > 0 ? group.maxInstances : 1;
  const children = groups.filter((candidate) => candidate.parentSelector === group.id);
  const title = locale.startsWith('zh') ? group.titleZh : group.titleEn;
  const instancesValue = Form.useWatch(['networkObjectInstances', instanceKey], { form, preserve: true }) as unknown[] | undefined;
  const fields = Array.from({ length: Array.isArray(instancesValue) ? instancesValue.length : 0 }, (_, index) => ({
    key: `${instanceKey}-${index}`,
    name: index,
  }));
  const add = () => {
    onRequestEdit?.();
    form.setFieldValue(['networkObjectInstances', instanceKey], [...(instancesValue ?? []), {}]);
  };
  const remove = (index: number) => {
    onRequestEdit?.();
    form.setFieldValue(
      ['networkObjectInstances', instanceKey],
      (instancesValue ?? []).filter((_item, itemIndex) => itemIndex !== index),
    );
  };
  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <Text type="secondary">{fields.length}/{maximum}</Text>
        {!readOnly && (
          <Button icon={<PlusOutlined />} disabled={fields.length >= maximum} onClick={add}>
            {t('common.add')}
          </Button>
        )}
      </div>
      {fields.map(({ key, name }) => {
        const instances = [...ancestors, name + 1];
        return (
          <Card
            key={key}
            size="small"
            title={`${title} ${name + 1}`}
            extra={readOnly ? undefined : (
              <ConfigProvider componentDisabled={onRequestEdit ? false : undefined}>
                <Button
                  type="text"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => remove(name)}
                />
              </ConfigProvider>
            )}
            style={{ marginBottom: 12 }}
          >
            <ParamGrid
              group={group}
              instances={instances}
              includeInterfaceName={group.id === 'gnb-network-interface'}
              readOnly={readOnly}
              onRequestEdit={onRequestEdit}
            />
            {children.length > 0 && (
              <Collapse items={children.map((child) => ({
                key: child.id,
                label: locale.startsWith('zh') ? child.titleZh : child.titleEn,
                children: child.multiInstance
                  ? <ObjectGroup group={child} groups={groups} ancestors={instances} locale={locale} onRequestEdit={onRequestEdit} readOnly={readOnly} />
                  : <ParamGrid group={child} instances={instances} readOnly={readOnly} onRequestEdit={onRequestEdit} />,
              }))} />
            )}
          </Card>
        );
      })}
    </>
  );
}

export default function CommonQuickSettingsNetworkCards({
  paramModelName,
  onRequestEdit,
  readOnly = false,
}: {
  paramModelName?: string;
  onRequestEdit?: () => void;
  readOnly?: boolean;
}) {
  const t = useT();
  const intl = useIntl();
  const [groups, setGroups] = useState<QuickSettingsGroup[]>(() => cachedNetworkGroups(paramModelName) ?? []);
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);
  const form = Form.useFormInstance();
  const sheetParameters = Form.useWatch('sheetParameters', form) as
    | Record<string, Array<Record<string, unknown>>>
    | undefined;
  const networkParameterValues = Form.useWatch('networkParameterValues', { form, preserve: true }) as
    | Record<string, unknown>
    | undefined;
  const locale = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';
  useEffect(() => {
    let active = true;
    const cacheKey = paramModelCacheKey(paramModelName);
    if (!cacheKey) {
      setGroups([]);
      return undefined;
    }
    const cached = cachedNetworkGroups(cacheKey);
    if (cached) {
      setGroups(cached);
      setLoading(false);
      setFailed(false);
      return undefined;
    }
    setLoading(true);
    setFailed(false);
    void loadNetworkGroups(cacheKey)
      .then((nextGroups) => { if (active) setGroups(nextGroups); })
      .catch(() => { if (active) setFailed(true); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [paramModelName]);
  useEffect(() => {
    if (groups.length === 0) return;
    const inferred = inferredNetworkParameterValues(groups, sheetParameters);
    if (Object.keys(inferred).length === 0) return;
    const current = form.getFieldValue('networkParameterValues') as Record<string, unknown> | undefined;
    const next = { ...(current ?? {}) };
    let changed = false;
    for (const [path, value] of Object.entries(inferred)) {
      if (next[path] !== undefined && next[path] !== null && String(next[path]).trim() !== '') continue;
      next[path] = value;
      changed = true;
    }
    if (changed) form.setFieldsValue({ networkParameterValues: next });
  }, [form, groups, sheetParameters]);
  useEffect(() => {
    if (groups.length === 0) return;
    const values = networkParameterValues
      ?? (form.getFieldsValue(true) as { networkParameterValues?: Record<string, unknown> }).networkParameterValues;
    if (!values) return;
    const current = form.getFieldValue('networkObjectInstances') as Record<string, unknown[]> | undefined;
    const base = current ?? {};
    const next = inferredNetworkObjectInstances(groups, values, base);
    if (next !== base) {
      for (const [instanceKey, instances] of Object.entries(next)) {
        if (base[instanceKey]?.length === instances.length) continue;
        form.setFieldValue(['networkObjectInstances', instanceKey], instances);
      }
    }
  }, [form, groups, networkParameterValues]);
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
            ? <ObjectGroup group={group} groups={groups} ancestors={[]} locale={locale} onRequestEdit={onRequestEdit} readOnly={readOnly} />
            : <ParamGrid group={group} instances={[]} readOnly={readOnly} onRequestEdit={onRequestEdit} />}
        </Card>
      ))}
      {fixedTableMode && <FixedNetworkTable groups={groups} kind="wan" locale={locale} readOnly={readOnly} />}
      {fixedTableMode && <FixedNetworkTable groups={groups} kind="static-route" locale={locale} readOnly={readOnly} />}
    </>
  );
}
