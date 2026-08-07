import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Form, Input } from 'antd';
import { useT } from '@/hooks/useT';
import { GNB_NETWORK_CONFIG_FIELDS } from './gnbNetworkConfigFields';

export default function GnbNetworkConfigCards({ readOnly = false }: { readOnly?: boolean }) {
  const t = useT();
  return (
    <Card size="small" title={t('provision.networkConfig')} style={{ marginBottom: 16 }}>
      <Form.List name="networkConfigList">
        {(fields, { add, remove }) => (
          <>
            {fields.map(({ key, name, ...restField }) => (
              <Card
                key={key}
                size="small"
                title={`${t('provision.networkConfigItem')} ${name + 1}`}
                extra={!readOnly && (
                  <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                )}
                style={{ marginBottom: 12 }}
              >
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', columnGap: 16 }}>
                  {GNB_NETWORK_CONFIG_FIELDS.map((field) => (
                    <Form.Item
                      {...restField}
                      key={field.header}
                      name={[name, field.header]}
                      label={t(field.labelKey)}
                    >
                      <Input />
                    </Form.Item>
                  ))}
                </div>
              </Card>
            ))}
            {!readOnly && (
              <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                {t('provision.addNetworkConfig')}
              </Button>
            )}
          </>
        )}
      </Form.List>
    </Card>
  );
}
