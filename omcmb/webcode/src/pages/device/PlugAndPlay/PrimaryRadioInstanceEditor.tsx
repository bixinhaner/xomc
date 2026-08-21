import { CopyOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Collapse, Form, Input, InputNumber, Space, Typography } from 'antd';
import { useMemo, useState } from 'react';
import { useT } from '@/hooks/useT';
import { getParamConfigTemplateSheets } from './paramConfigTemplate';
import { primaryInstanceHeader, type ParamConfigDeviceType } from './paramConfigWorkbook';
import { ENB_SHEET_FIELD_MAPPINGS, GNB_SHEET_FIELD_MAPPINGS } from './paramConfigFieldMappings';
import {
  GNB_QUICK_SETTING_GROUPS,
  GNB_TEMPLATE_EXTRA_FIELDS,
  type GnbQuickSettingField,
} from './gnbQuickSettingsFields';
import { GnbQuickSettingFieldGrid } from './GnbQuickSettingsCards';
import { ENB_QUICK_SETTING_GROUPS } from './enbQuickSettingsFields';
import { GSM_GROUPED_TEMPLATE_FIELDS } from './paramConfigGroupedFields';

const { Text } = Typography;

function canonicalHeader(header: string): string {
  return header.replace(/^\*/, '').replace(/[\s_]+/g, '').toUpperCase();
}

function isSerialHeader(header: string): boolean {
  return canonicalHeader(header) === 'SERIALNUMBER';
}

function isDeviceLevelHeader(deviceType: ParamConfigDeviceType, header: string): boolean {
  if (deviceType !== 'gNB') return false;
  return ['GNBNAME', 'GNBID', 'GNBLENTH', 'GNBLENGTH'].includes(canonicalHeader(header));
}

function nextInstanceIndex(rows: Array<Record<string, unknown>>, header: string): number {
  return Math.max(0, ...rows.map((row) => Number(row[header]) || 0)) + 1;
}

function blankRow(headers: readonly string[], instanceHeader: string, instanceIndex: number) {
  return {
    ...Object.fromEntries(headers.filter((header) => !isSerialHeader(header)).map((header) => [header, ''])),
    [instanceHeader]: instanceIndex,
  };
}

export interface PrimaryRadioInstanceEditorProps {
  deviceType: ParamConfigDeviceType;
  productClass?: string;
  excludedFieldIds?: readonly string[];
  readOnly?: boolean;
}

interface RadioInstanceLabelProps {
  form: ReturnType<typeof Form.useFormInstance>;
  sheetName: string;
  fieldName: number;
  instanceHeader: string;
  rowIndex: number;
  itemName: string;
}

function RadioInstanceLabel({
  form,
  sheetName,
  fieldName,
  instanceHeader,
  rowIndex,
  itemName,
}: RadioInstanceLabelProps) {
  return (
    <Form.Item noStyle shouldUpdate>
      {() => {
        const instanceValue = form.getFieldValue(
          ['sheetParameters', sheetName, fieldName, instanceHeader],
        ) ?? rowIndex + 1;
        return (
          <Space>
            <Text strong>{`${itemName} ${rowIndex + 1}`}</Text>
            <Text type="secondary">{`${instanceHeader}: ${instanceValue}`}</Text>
          </Space>
        );
      }}
    </Form.Item>
  );
}

function displayHeaderLabel(header: string): string {
  const clean = header.replace(/^\*/, '').replace(/\s+/g, ' ').trim();
  return clean
    .replace(/^DL ULTransmissionPeriodicity([12])$/, 'DL/UL Transmission Periodicity $1')
    .replace(/^Nrof DownlinkSlots([12])$/, 'Nrof Downlink Slots $1')
    .replace(/^Nrof DownlinkSymbols([12])$/, 'Nrof Downlink Symbols $1')
    .replace(/^Nrof UplinkSlots([12])$/, 'Nrof Uplink Slots $1')
    .replace(/^Nrof UplinkSymbols([12])$/, 'Nrof Uplink Symbols $1')
    .replace(/^Prach RootSequenceIndex$/, 'Prach Root Sequence Index')
    .replace(/^Prach RootSequenceValue$/, 'Prach Root Sequence Value');
}

function lastNamePathPart(name: GnbQuickSettingField['name']): string | undefined {
  if (!Array.isArray(name)) return undefined;
  const last = name.at(-1);
  return last == null ? undefined : String(last);
}

const GNB_CELL_HEADER_ALIASES: Record<string, string[]> = {
  pci: ['*PCI', 'PCI'],
  dlbandwidth: ['DLBandwidth', 'DL Carrier Bandwidth'],
  ulbandwidth: ['ULBandwidth', 'UL Carrier Bandwidth'],
  RFEnable: ['RFEnable', 'RF Enable'],
  totalTxPower: ['PowerModify', 'Power Level'],
  offsetToPointA: ['OffsetToPointA', 'Offset To Point A'],
  kssb: ['SsbSubcarrierOffset', 'SSB Subcarrier Offset'],
  'Prach RootSequenceIndex': ['Prach RootSequenceIndex', 'Prach Root Sequence Index'],
  'Prach RootSequenceValue': ['Prach RootSequenceValue', 'Prach Root Sequence Value'],
};

const GNB_CELL_FIELD_HEADERS = new Map(
  GNB_SHEET_FIELD_MAPPINGS
    .filter((mapping) => mapping.sheet === 'CELL')
    .map((mapping) => [mapping.field, mapping.header]),
);

const ENB_CELL_HEADER_ALIASES: Record<string, string[]> = {
  BandSupport: ['*BAND', 'BAND'],
  bandsSupport: ['*BAND', 'BAND'],
  bandWidth: ['*BANDWIDTH_DL', 'BANDWIDTH_DL'],
  MaxTxPower: ['MaxTxPower'],
  phycellid: ['*PCI', 'PCI'],
  rootSequenceIndex: ['*ROOT_SEQUENCE_INDEX', 'ROOT_SEQUENCE_INDEX'],
  tac: ['*TAC', 'TAC'],
  totalTxPower: ['X_COM_MaxTxPowerExpanded', 'ReferenceSignalPower', 'PowerClass', 'Transmit Power', 'Tx Power'],
  TxPower: ['X_COM_MaxTxPowerExpanded', 'ReferenceSignalPower', 'PowerClass', 'Transmit Power', 'Tx Power'],
};

const ENB_CELL_FIELD_HEADERS = new Map(
  ENB_SHEET_FIELD_MAPPINGS
    .filter((mapping) => mapping.sheet === 'CELL')
    .map((mapping) => [mapping.field, mapping.header]),
);

function uniqueHeaders(headers: string[]): string[] {
  return [...new Set(headers.filter(Boolean))];
}

function gnbCellFieldHeaderCandidates(field: GnbQuickSettingField): string[] {
  const pathHeader = lastNamePathPart(field.name);
  const fieldName = typeof field.name === 'string' ? field.name : pathHeader ?? '';
  return uniqueHeaders([
    ...(GNB_CELL_HEADER_ALIASES[fieldName] ?? []),
    GNB_CELL_FIELD_HEADERS.get(fieldName) ?? '',
    pathHeader ?? '',
    fieldName,
  ]);
}

function enbCellFieldHeaderCandidates(field: GnbQuickSettingField): string[] {
  const pathHeader = lastNamePathPart(field.name);
  const fieldName = typeof field.name === 'string' ? field.name : pathHeader ?? '';
  return uniqueHeaders([
    ...(ENB_CELL_HEADER_ALIASES[field.id] ?? []),
    ...(ENB_CELL_HEADER_ALIASES[fieldName] ?? []),
    ENB_CELL_FIELD_HEADERS.get(fieldName) ?? '',
    pathHeader ?? '',
    fieldName,
  ]);
}

function resolveAvailableHeader(
  candidates: string[],
  availableHeaders: readonly string[],
): string {
  for (const candidate of candidates) {
    const normalizedCandidate = canonicalHeader(candidate);
    const available = availableHeaders.find((header) => canonicalHeader(header) === normalizedCandidate);
    if (available) return available;
  }
  return candidates[0] ?? '';
}

function hasAvailableHeader(candidates: string[], availableHeaders: readonly string[]): boolean {
  return candidates.some((candidate) => (
    availableHeaders.some((header) => canonicalHeader(header) === canonicalHeader(candidate))
  ));
}

const GNB_CELL_GROUPS = GNB_QUICK_SETTING_GROUPS
  .filter((group) => group.id === 'gnb-cell' || group.id === 'gnb-tdd');

const GNB_CELL_EXTRA_FIELDS = GNB_TEMPLATE_EXTRA_FIELDS.filter((field) => {
  const name = field.name;
  return Array.isArray(name) && name[0] === 'sheetParameters' && name[1] === 'CELL';
});

const ENB_CELL_GROUP = ENB_QUICK_SETTING_GROUPS.find((group) => group.id === 'enb-cell');
const ENB_CELL_HIDDEN_READONLY_FIELD_IDS = new Set(['MaxTxPower']);
const ENB_CELL_TX_POWER_FIELD: GnbQuickSettingField = {
  id: 'TxPower',
  name: 'totalTxPower',
  labelKey: 'provision.totalTxPower',
  range: '0 ~ 46',
};

function enbCellEditFields(): GnbQuickSettingField[] {
  return (ENB_CELL_GROUP?.fields ?? []).flatMap((field) => (
    field.id === 'MaxTxPower' ? [field, ENB_CELL_TX_POWER_FIELD] : [field]
  ));
}

function isHiddenReadOnlyEnbCellField(field: GnbQuickSettingField): boolean {
  return field.control === 'readonly' || ENB_CELL_HIDDEN_READONLY_FIELD_IDS.has(field.id);
}

const GSM_RADIO_GROUPS = [
  {
    id: 'gsm-abis',
    titleKey: 'provision.gsmQuick.abisParameters',
    fields: GSM_GROUPED_TEMPLATE_FIELDS.quickAbis,
  },
  {
    id: 'gsm-other',
    titleKey: 'provision.otherTemplateParams',
    fields: GSM_GROUPED_TEMPLATE_FIELDS.other,
  },
];

interface GnbCellGroupedFieldsProps {
  fieldName: number;
  availableHeaders: readonly string[];
  excludedFieldIds?: readonly string[];
  readOnly: boolean;
}

function GnbCellGroupedFields({
  fieldName,
  availableHeaders,
  excludedFieldIds = [],
  readOnly,
}: GnbCellGroupedFieldsProps) {
  const t = useT();
  const excludedFieldIdSet = new Set(excludedFieldIds);
  const resolveName = (field: GnbQuickSettingField) => [
    fieldName,
    resolveAvailableHeader(gnbCellFieldHeaderCandidates(field), availableHeaders),
  ];
  const sectionStyle = {
    borderTop: '1px solid #f0f0f0',
    paddingTop: 12,
    marginTop: 12,
  };

  return (
    <>
      {GNB_CELL_GROUPS.map((group, index) => {
        const fields = group.fields.filter((field) => !excludedFieldIdSet.has(field.id));
        if (fields.length === 0) return null;
        return (
        <div key={group.id} style={index === 0 ? undefined : sectionStyle}>
          <Text strong>{t(group.titleKey)}</Text>
          <div style={{ marginTop: 12 }}>
            <GnbQuickSettingFieldGrid
              fields={fields}
              nameResolver={resolveName}
              dlScsName={['sheetParameters', 'CELL', fieldName, resolveAvailableHeader(['SubcarrierSpacing(DL)'], availableHeaders)]}
              ulScsName={['sheetParameters', 'CELL', fieldName, resolveAvailableHeader(['SubcarrierSpacing(UL)'], availableHeaders)]}
              readOnly={readOnly}
            />
          </div>
        </div>
        );
      })}
      {GNB_CELL_EXTRA_FIELDS.length > 0 && (
        <div style={sectionStyle}>
          <Text strong>{t('provision.otherTemplateParams')}</Text>
          <div style={{ marginTop: 12 }}>
            <GnbQuickSettingFieldGrid
              fields={GNB_CELL_EXTRA_FIELDS}
              nameResolver={resolveName}
              dlScsName={['sheetParameters', 'CELL', fieldName, resolveAvailableHeader(['SubcarrierSpacing(DL)'], availableHeaders)]}
              ulScsName={['sheetParameters', 'CELL', fieldName, resolveAvailableHeader(['SubcarrierSpacing(UL)'], availableHeaders)]}
              readOnly={readOnly}
            />
          </div>
        </div>
      )}
    </>
  );
}

function EnbCellGroupedFields({
  fieldName,
  availableHeaders,
  readOnly,
}: GnbCellGroupedFieldsProps) {
  const t = useT();
  const fields = enbCellEditFields().filter((field) => (
    !isHiddenReadOnlyEnbCellField(field)
      && hasAvailableHeader(enbCellFieldHeaderCandidates(field), availableHeaders)
  ));
  if (fields.length === 0) return null;
  return (
    <div style={{ borderTop: '1px solid #f0f0f0', paddingTop: 12, marginTop: 12 }}>
      <Text strong>{t(ENB_CELL_GROUP?.titleKey ?? 'provision.lteQuick.cellParameters')}</Text>
      <div style={{ marginTop: 12 }}>
        <GnbQuickSettingFieldGrid
          fields={fields}
          nameResolver={(field) => [
            fieldName,
            resolveAvailableHeader(enbCellFieldHeaderCandidates(field), availableHeaders),
          ]}
          readOnly={readOnly}
        />
      </div>
    </div>
  );
}

function GsmGroupedFields({
  fieldName,
  availableHeaders,
  readOnly,
}: GnbCellGroupedFieldsProps) {
  const t = useT();
  return (
    <>
      {GSM_RADIO_GROUPS.map((group) => {
        const headers = group.fields
          .map((field) => resolveAvailableHeader([field.header], availableHeaders))
          .filter((header) => hasAvailableHeader([header], availableHeaders));
        if (headers.length === 0) return null;
        return (
          <div key={group.id} style={{ borderTop: '1px solid #f0f0f0', paddingTop: 12, marginTop: 12 }}>
            <Text strong>{t(group.titleKey)}</Text>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', columnGap: 16, rowGap: 6, marginTop: 12 }}>
              {headers.map((header) => (
                <Form.Item key={header} name={[fieldName, header]} label={displayHeaderLabel(header)}>
                  <Input readOnly={readOnly} />
                </Form.Item>
              ))}
            </div>
          </div>
        );
      })}
    </>
  );
}

export default function PrimaryRadioInstanceEditor({
  deviceType,
  productClass,
  excludedFieldIds = [],
  readOnly = false,
}: PrimaryRadioInstanceEditorProps) {
  const t = useT();
  const form = Form.useFormInstance();
  const [activeKeys, setActiveKeys] = useState<string[]>([]);
  const sheetName = deviceType === 'GSM' ? 'GSM' : 'CELL';
  const instanceHeader = primaryInstanceHeader(deviceType, sheetName, productClass) ?? 'Cell Index';
  const watchedRows = Form.useWatch(['sheetParameters', sheetName], form) as Array<Record<string, unknown>> | undefined;
  const templateHeaders = getParamConfigTemplateSheets(deviceType)?.[sheetName] ?? [];
  const headers = useMemo(() => {
    const result = new Set<string>([instanceHeader, ...templateHeaders]);
    for (const row of watchedRows ?? []) Object.keys(row ?? {}).forEach((header) => result.add(header));
    return [...result].filter((header) => !isSerialHeader(header)
      && !isDeviceLevelHeader(deviceType, header)
      && canonicalHeader(header) !== canonicalHeader(instanceHeader));
  }, [deviceType, instanceHeader, templateHeaders, watchedRows]);
  const isBts = instanceHeader === 'BTS Index';
  const itemName = isBts ? t('provision.bts') : t('provision.cell');
  const useGnbCellLayout = deviceType === 'gNB' && sheetName === 'CELL';
  const useEnbCellLayout = deviceType === 'eNB' && sheetName === 'CELL';
  const useGsmLayout = deviceType === 'GSM' && sheetName === 'GSM';

  return (
    <Card size="small" title={t('provision.radioInstanceList', { item: itemName })} style={{ marginBottom: 16 }}>
      <Text type="secondary">{t('provision.radioInstanceListHint', { item: itemName })}</Text>
      <Form.List name={['sheetParameters', sheetName]}>
        {(fields, { add, remove }) => (
          <>
            <Collapse
              activeKey={activeKeys}
              onChange={(keys) => setActiveKeys(Array.isArray(keys) ? keys.map(String) : [String(keys)])}
              style={{ margin: '12px 0' }}
              items={fields.map((field, rowIndex) => {
                const panelKey = String(field.key);
                const toggleEditor = () => setActiveKeys((current) => (
                  current.includes(panelKey)
                    ? current.filter((key) => key !== panelKey)
                    : [...current, panelKey]
                ));
                return {
                  key: panelKey,
                  label: (
                    <RadioInstanceLabel
                      form={form}
                      sheetName={sheetName}
                      fieldName={field.name}
                      instanceHeader={instanceHeader}
                      rowIndex={rowIndex}
                      itemName={itemName}
                    />
                  ),
                  extra: readOnly ? undefined : (
                    <Space onClick={(event) => event.stopPropagation()}>
                      <Button type="link" onClick={toggleEditor}>{t('common.edit')}</Button>
                      <Button
                        type="link"
                        icon={<CopyOutlined />}
                        onClick={() => {
                          const rows = (form.getFieldValue(['sheetParameters', sheetName]) ?? []) as Array<Record<string, unknown>>;
                          const source = rows[rowIndex] ?? {};
                          add({ ...source, [instanceHeader]: nextInstanceIndex(rows, instanceHeader) });
                        }}
                      >
                        {t('common.copy')}
                      </Button>
                      <Button
                        type="link"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => {
                          setActiveKeys((current) => current.filter((key) => key !== panelKey));
                          remove(field.name);
                        }}
                      >
                        {t('common.delete')}
                      </Button>
                    </Space>
                  ),
                  children: (() => {
                    const row = watchedRows?.[rowIndex] ?? {};
                    const availableHeaders = uniqueHeaders([...Object.keys(row), ...templateHeaders]);
                    const coveredHeaders = new Set<string>();
                    if (useGnbCellLayout) {
                      [...GNB_CELL_GROUPS.flatMap((group) => group.fields), ...GNB_CELL_EXTRA_FIELDS].forEach((quickField) => {
                        coveredHeaders.add(canonicalHeader(resolveAvailableHeader(
                          gnbCellFieldHeaderCandidates(quickField),
                          availableHeaders,
                        )));
                      });
                    }
                    if (useEnbCellLayout) {
                      enbCellEditFields().forEach((quickField) => {
                        const candidates = enbCellFieldHeaderCandidates(quickField);
                        if (hasAvailableHeader(candidates, availableHeaders)) {
                          coveredHeaders.add(canonicalHeader(resolveAvailableHeader(candidates, availableHeaders)));
                        }
                      });
                    }
                    if (useGsmLayout) {
                      GSM_RADIO_GROUPS.flatMap((group) => group.fields).forEach((fieldRef) => {
                        if (hasAvailableHeader([fieldRef.header], availableHeaders)) {
                          coveredHeaders.add(canonicalHeader(resolveAvailableHeader([fieldRef.header], availableHeaders)));
                        }
                      });
                    }
                    const fallbackHeaders = headers.filter((header) => !coveredHeaders.has(canonicalHeader(header)));
                    return (
                      <>
                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', columnGap: 16, rowGap: 6 }}>
                          <Form.Item
                            name={[field.name, instanceHeader]}
                            label={displayHeaderLabel(instanceHeader)}
                            rules={[
                              { required: true, message: t('provision.instanceIndexRequired') },
                              {
                                validator: async (_, candidate) => {
                                  if (!Number.isInteger(Number(candidate)) || Number(candidate) <= 0) {
                                    throw new Error(t('provision.instanceIndexInvalid'));
                                  }
                                  const rows = (form.getFieldValue(['sheetParameters', sheetName]) ?? []) as Array<Record<string, unknown>>;
                                  if (rows.some((rowItem, index) => index !== rowIndex && Number(rowItem?.[instanceHeader]) === Number(candidate))) {
                                    throw new Error(t('provision.instanceIndexDuplicate'));
                                  }
                                },
                              },
                            ]}
                          >
                            <InputNumber min={1} precision={0} readOnly={readOnly} style={{ width: '100%' }} />
                          </Form.Item>
                        </div>
                        {useGnbCellLayout && (
                          <GnbCellGroupedFields
                            fieldName={field.name}
                            availableHeaders={availableHeaders}
                            excludedFieldIds={excludedFieldIds}
                            readOnly={readOnly}
                          />
                        )}
                        {useEnbCellLayout && (
                          <EnbCellGroupedFields
                            fieldName={field.name}
                            availableHeaders={availableHeaders}
                            readOnly={readOnly}
                          />
                        )}
                        {useGsmLayout && (
                          <GsmGroupedFields
                            fieldName={field.name}
                            availableHeaders={availableHeaders}
                            readOnly={readOnly}
                          />
                        )}
                        {fallbackHeaders.length > 0 && (
                          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', columnGap: 16, rowGap: 6, marginTop: useGnbCellLayout ? 12 : 0 }}>
                            {fallbackHeaders.map((header) => (
                              <Form.Item key={header} name={[field.name, header]} label={displayHeaderLabel(header)}>
                                <Input readOnly={readOnly} />
                              </Form.Item>
                            ))}
                          </div>
                        )}
                      </>
                    );
                  })(),
                };
              })}
            />
            {!readOnly && (
              <Button
                type="dashed"
                block
                icon={<PlusOutlined />}
                onClick={() => {
                  const rows = (form.getFieldValue(['sheetParameters', sheetName]) ?? []) as Array<Record<string, unknown>>;
                  add(blankRow(templateHeaders, instanceHeader, nextInstanceIndex(rows, instanceHeader)));
                }}
              >
                {t('provision.addRadioInstance', { item: itemName })}
              </Button>
            )}
          </>
        )}
      </Form.List>
    </Card>
  );
}
