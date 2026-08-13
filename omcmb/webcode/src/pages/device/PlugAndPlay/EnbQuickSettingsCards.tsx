import { Fragment, type ReactNode } from 'react';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Divider, Form, Input } from 'antd';
import { useT } from '@/hooks/useT';
import { GnbQuickSettingFieldGrid } from './GnbQuickSettingsCards';
import {
  ENB_IPSEC_TEMPLATE_EXTRA_FIELDS,
  ENB_QUICK_SETTING_GROUPS,
  ENB_TEMPLATE_EXTRA_FIELDS,
} from './enbQuickSettingsFields';
import { getEnbProductSyncConfig } from './enbProductSyncFields';
import { isIpsecParametersVisible } from './quickSettingsVisibility';

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
  beforeIpsec,
}: {
  paramModelName?: string;
  beforeIpsec?: ReactNode;
}) {
  const t = useT();
  const form = Form.useFormInstance();
  const ipsecEnable = Form.useWatch('ipsecEnable', form);
  const syncConfig = getEnbProductSyncConfig(paramModelName);
  const syncMode = Form.useWatch(syncConfig.modeFieldName ?? '__unsupportedSyncMode', form);
  const showSyncDetails = syncConfig.ptpModeValues.includes(String(syncMode ?? ''));
  const visibleSyncFields = showSyncDetails
    ? syncConfig.fields
    : syncConfig.fields.slice(0, syncConfig.collapsedFieldCount);

  return (
    <>
      {ENB_QUICK_SETTING_GROUPS
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

export function EnbTemplateExtraFieldGrid() {
  return <GnbQuickSettingFieldGrid fields={ENB_TEMPLATE_EXTRA_FIELDS} />;
}
