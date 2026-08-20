import { Fragment, type ReactNode, useEffect, useMemo, useState } from 'react';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Divider, Form, Input } from 'antd';
import { useT } from '@/hooks/useT';
import { quicksettingsApi } from '@core/services/api/quicksettingsApi';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import { GnbQuickSettingFieldGrid } from './GnbQuickSettingsCards';
import {
  ENB_IPSEC_TEMPLATE_EXTRA_FIELDS,
  ENB_QUICK_SETTING_GROUPS,
  ENB_TEMPLATE_EXTRA_FIELDS,
} from './enbQuickSettingsFields';
import { getEnbProductSyncConfig } from './enbProductSyncFields';
import { isIpsecParametersVisible } from './quickSettingsVisibility';

function isLteBandwidthPath(path: string | undefined): boolean {
  return /\.LTE\.RAN\.RF\.(?:DL|UL)Bandwidth$/i.test(path ?? '');
}

function isLteBandwidthField(name: string | undefined): boolean {
  return /^(?:DL|UL)?BandWidth$/i.test(name ?? '');
}

function normalizeLteBandwidthOptionValue(value: string): string {
  const trimmed = value.trim();
  const match = trimmed.match(/^n?(25|50|75|100)$/i);
  return match ? `n${match[1]}` : value;
}

export function withProductEnumOptions(groups: typeof ENB_QUICK_SETTING_GROUPS, metadata: QuickSettingsGroup[]) {
  return groups.map((group) => {
    const productGroup = metadata.find((item) => item.id === group.id);
    if (!productGroup) return group;
    return {
      ...group,
      fields: group.fields.map((field) => {
        const productParam = productGroup.params.find((item) => item.name === field.id);
        if (!productParam?.enumOptions?.length) return field;
        const lteBandwidth = isLteBandwidthPath(productParam.standardPath)
          || isLteBandwidthField(field.id)
          || isLteBandwidthField(productParam.name);
        return {
          ...field,
          control: 'select' as const,
          options: productParam.enumOptions.map((option) => ({
            value: lteBandwidth ? normalizeLteBandwidthOptionValue(option.value) : option.value,
            label: option.label,
          })),
        };
      }),
    };
  });
}

function MmeListCard() {
  const t = useT();
  return (
    <Form.List name="mmeList">
      {(fields, { add, remove }) => (
        <>
          {fields.map(({ key, name, ...restField }) => (
            <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.mmeItem')} ${name + 1}`} extra={
              <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
            }>
              <Form.Item {...restField} name={[name, 'mmeIp']} label={t('provision.lteQuick.mmeIp')}>
                <Input placeholder={t('provision.mmePlaceholder')} />
              </Form.Item>
            </Card>
          ))}
          <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
            {t('provision.addMme')}
          </Button>
        </>
      )}
    </Form.List>
  );
}

function ServingPlmnListCard() {
  const t = useT();
  const group = ENB_QUICK_SETTING_GROUPS.find(({ id }) => id === 'enb-plmn');
  return (
    <Form.List name="plmnConfigList">
      {(fields, { add, remove }) => (
        <>
          {fields.map(({ key, name }) => (
            <Card
              key={key}
              size="small"
              style={{ marginBottom: 12 }}
              title={`${t('provision.lteQuick.plmn')} ${name + 1}`}
              extra={<Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />}
            >
              <GnbQuickSettingFieldGrid fields={group?.fields ?? []} namePrefix={[name]} />
            </Card>
          ))}
          <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
            {t('provision.addPlmnConfig')}
          </Button>
        </>
      )}
    </Form.List>
  );
}

function IpsecListCard() {
  const t = useT();
  const group = ENB_QUICK_SETTING_GROUPS.find(({ id }) => id === 'device-ipsec');
  return (
    <Form.List name="ipsecList">
      {(fields, { add, remove }) => (
        <>
          {fields.map(({ key, name }) => (
            <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.ipsecTunnel')} ${name + 1}`} extra={
              <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
            }>
              <GnbQuickSettingFieldGrid fields={group?.fields ?? []} namePrefix={[name]} />
              <Divider titlePlacement="left">{t('provision.lteQuick.tunnelTemplateExtras')}</Divider>
              <GnbQuickSettingFieldGrid fields={ENB_IPSEC_TEMPLATE_EXTRA_FIELDS} namePrefix={[name]} />
            </Card>
          ))}
          <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
            {t('provision.addTunnel')}
          </Button>
        </>
      )}
    </Form.List>
  );
}

export default function EnbQuickSettingsCards({
  paramModelName,
  excludedGroupIds = [],
  beforeIpsec,
}: {
  paramModelName?: string;
  excludedGroupIds?: string[];
  beforeIpsec?: ReactNode;
}) {
  const t = useT();
  const form = Form.useFormInstance();
  const [productGroups, setProductGroups] = useState<QuickSettingsGroup[]>([]);
  useEffect(() => {
    let active = true;
    if (!paramModelName) {
      setProductGroups([]);
      return () => { active = false; };
    }
    void quicksettingsApi.getGroupsByParamModel(paramModelName)
      .then((response) => { if (active) setProductGroups(response.groups); })
      .catch(() => { if (active) setProductGroups([]); });
    return () => { active = false; };
  }, [paramModelName]);
  const resolvedGroups = useMemo(
    () => withProductEnumOptions(ENB_QUICK_SETTING_GROUPS, productGroups),
    [productGroups],
  );
  const ipsecEnable = Form.useWatch('ipsecEnable', form);
  const syncConfig = getEnbProductSyncConfig(paramModelName);
  const syncMode = Form.useWatch(syncConfig.modeFieldName ?? '__unsupportedSyncMode', form);
  const showSyncDetails = syncConfig.ptpModeValues.includes(String(syncMode ?? ''));
  const visibleSyncFields = showSyncDetails
    ? syncConfig.fields
    : syncConfig.fields.slice(0, syncConfig.collapsedFieldCount);

  return (
    <>
      {resolvedGroups
        .filter((group) => !excludedGroupIds.includes(group.id))
        .filter((group) => group.id !== 'device-ipsec' || isIpsecParametersVisible(ipsecEnable))
        .map((group) => (
        <Fragment key={group.id}>
          {group.id === 'device-ipsec-control' && beforeIpsec}
          <Card size="small" title={t(group.titleKey)} style={{ marginBottom: 16 }}>
            {group.id === 'enb-plmn'
              ? <ServingPlmnListCard />
              : group.id === 'enb-mme'
              ? <MmeListCard />
              : group.id === 'device-ipsec'
                ? <IpsecListCard />
                : <GnbQuickSettingFieldGrid fields={group.fields} />}
          </Card>
          {group.id === 'device-time' && visibleSyncFields.length > 0 && (
            <Card
              size="small"
              title={t('provision.syncSourceConfig')}
              style={{ marginBottom: 16 }}
            >
              <GnbQuickSettingFieldGrid fields={visibleSyncFields} />
            </Card>
          )}
        </Fragment>
        ))}
    </>
  );
}

export function EnbTemplateExtraFieldGrid({
  excludedFieldIds = [],
}: {
  excludedFieldIds?: readonly string[];
}) {
  return <GnbQuickSettingFieldGrid fields={ENB_TEMPLATE_EXTRA_FIELDS.filter((field) => !excludedFieldIds.includes(field.id))} />;
}
