import { Fragment } from 'react';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Divider, Form, Input } from 'antd';
import { useT } from '@/hooks/useT';
import { GnbQuickSettingFieldGrid } from './GnbQuickSettingsCards';
import {
  ENB_1588_TEMPLATE_FIELDS,
  ENB_IPSEC_TEMPLATE_EXTRA_FIELDS,
  ENB_QUICK_SETTING_GROUPS,
  ENB_TEMPLATE_EXTRA_FIELDS,
} from './enbQuickSettingsFields';
import { isIpsecParametersVisible, isPtpDetailsVisible } from './quickSettingsVisibility';

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

export default function EnbQuickSettingsCards() {
  const t = useT();
  const form = Form.useFormInstance();
  const ipsecEnable = Form.useWatch('ipsecEnable', form);
  const syncMode = Form.useWatch('PpsTimeMode', form);
  const show1588Settings = isPtpDetailsVisible(syncMode);

  return (
    <>
      {ENB_QUICK_SETTING_GROUPS
        .filter((group) => group.id !== 'device-ipsec' || isIpsecParametersVisible(ipsecEnable))
        .map((group) => (
        <Fragment key={group.id}>
          <Card size="small" title={t(group.titleKey)} style={{ marginBottom: 16 }}>
            {group.id === 'enb-plmn'
              ? <ServingPlmnListCard />
              : group.id === 'enb-mme'
              ? <MmeListCard />
              : group.id === 'device-ipsec'
                ? <IpsecListCard />
                : <GnbQuickSettingFieldGrid fields={group.fields} />}
          </Card>
          {group.id === 'device-time' && (
            <Card
              size="small"
              title={t('provision.syncSourceConfig')}
              style={{ marginBottom: 16 }}
            >
              <GnbQuickSettingFieldGrid fields={
                show1588Settings
                  ? ENB_1588_TEMPLATE_FIELDS
                  : ENB_1588_TEMPLATE_FIELDS.slice(0, 2)
              } />
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
