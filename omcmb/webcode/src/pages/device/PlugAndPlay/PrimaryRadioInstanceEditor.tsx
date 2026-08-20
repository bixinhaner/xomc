import { CopyOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Collapse, Form, Input, InputNumber, Space, Typography } from 'antd';
import { useMemo, useState } from 'react';
import { useT } from '@/hooks/useT';
import { getParamConfigTemplateSheets } from './paramConfigTemplate';
import { primaryInstanceHeader, type ParamConfigDeviceType } from './paramConfigWorkbook';

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

export default function PrimaryRadioInstanceEditor({ deviceType, productClass, readOnly = false }: PrimaryRadioInstanceEditorProps) {
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
                  children: (
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
                            if (rows.some((row, index) => index !== rowIndex && Number(row?.[instanceHeader]) === Number(candidate))) {
                              throw new Error(t('provision.instanceIndexDuplicate'));
                            }
                          },
                        },
                      ]}
                    >
                      <InputNumber min={1} precision={0} readOnly={readOnly} style={{ width: '100%' }} />
                    </Form.Item>
                    {headers.map((header) => (
                      <Form.Item key={header} name={[field.name, header]} label={displayHeaderLabel(header)}>
                        <Input readOnly={readOnly} />
                      </Form.Item>
                    ))}
                  </div>
                  ),
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
