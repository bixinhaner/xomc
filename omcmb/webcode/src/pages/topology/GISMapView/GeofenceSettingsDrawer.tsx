import { useEffect, useState } from 'react';
import {
  App,
  Alert,
  Button,
  Descriptions,
  Drawer,
  Form,
  InputNumber,
  Select,
  Space,
  Spin,
  Tag,
  message as staticMessage,
} from 'antd';
import { useIntl } from 'react-intl';
import {
  useGeofenceSettings,
  usePreviewGeofenceSettings,
  useUpdateGeofenceSettings,
} from '@core/hooks/api/useGeofence';
import type {
  GeofenceRuntimeMode,
  GeofenceSettingsPreview,
  UpdateGeofenceSettingsInput,
} from '@core/types/geofence';
import {
  effectiveGeofenceMode,
  geofenceSettingsToInput,
  validateGeofenceSettingsInput,
} from '@core/utils/geofenceSettings';

interface GeofenceSettingsDrawerProps {
  open: boolean;
  onClose: () => void;
}

const MODE_OPTIONS: GeofenceRuntimeMode[] = [
  'off',
  'observe',
  'enforce',
];

export default function GeofenceSettingsDrawer({
  open,
  onClose,
}: GeofenceSettingsDrawerProps) {
  const intl = useIntl();
  const appContext = App.useApp();
  const message = typeof (appContext.message as { success?: unknown }).success === 'function'
    ? appContext.message
    : staticMessage;
  const [form] = Form.useForm<UpdateGeofenceSettingsInput>();
  const [preview, setPreview] =
    useState<GeofenceSettingsPreview>();
  const settingsQuery = useGeofenceSettings();
  const previewMutation = usePreviewGeofenceSettings();
  const updateMutation = useUpdateGeofenceSettings();
  const systemMode =
    Form.useWatch('systemMode', form) ?? 'off';
  const carriers = Form.useWatch('carriers', form) ?? [];

  useEffect(() => {
    if (!open || !settingsQuery.data) return;
    form.setFieldsValue(
      geofenceSettingsToInput(settingsQuery.data),
    );
  }, [form, open, settingsQuery.data]);

  const modeOptions = MODE_OPTIONS.map((mode) => ({
    value: mode,
    label: intl.formatMessage({ id: `geofence.mode.${mode}` }),
  }));

  const resetPreview = () => {
    setPreview(undefined);
    previewMutation.reset();
  };

  const validate = async () => {
    const input = await form.validateFields();
    const errors = validateGeofenceSettingsInput(input);
    if (errors.length === 0) return input;
    const first = errors[0];
    const messageId =
      first.code === 'radius_out_of_range'
        ? 'geofence.validation.radiusRange'
        : 'geofence.validation.carriersRequired';
    void message.error(intl.formatMessage({ id: messageId }));
    return undefined;
  };

  const handlePreview = async () => {
    try {
      const input = await validate();
      if (!input) return;
      setPreview(await previewMutation.mutateAsync(input));
    } catch (error) {
      if (error instanceof Error) {
        void message.error(error.message);
      }
    }
  };

  const handleSave = async () => {
    if (!preview) return;
    try {
      const input = await validate();
      if (!input) return;
      await updateMutation.mutateAsync(input);
      void message.success(
        intl.formatMessage({ id: 'geofence.message.saved' }),
      );
      onClose();
    } catch (error) {
      void message.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({
              id: 'geofence.message.operationFailed',
            }),
      );
    }
  };

  const busy =
    settingsQuery.isFetching ||
    previewMutation.isPending ||
    updateMutation.isPending;

  return (
    <Drawer
      rootClassName="geofence-drawer"
      title={intl.formatMessage({
        id: 'geofence.settings.title',
      })}
      open={open}
      size={560}
      onClose={onClose}
      footer={
        <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
          <Button onClick={onClose}>
            {intl.formatMessage({ id: 'common.cancel' })}
          </Button>
          <Button
            onClick={() => void handlePreview()}
            disabled={
              settingsQuery.isError ||
              !settingsQuery.data ||
              busy
            }
          >
            {intl.formatMessage({
              id: 'geofence.settings.preview',
            })}
          </Button>
          <Button
            type="primary"
            onClick={() => void handleSave()}
            disabled={!preview || busy}
          >
            {intl.formatMessage({
              id: 'geofence.settings.confirmSave',
            })}
          </Button>
        </Space>
      }
    >
      {settingsQuery.isLoading ? (
        <div style={{ padding: 48, textAlign: 'center' }}>
          <Spin />
        </div>
      ) : settingsQuery.isError || !settingsQuery.data ? (
        <Alert
          type="error"
          showIcon
          title={intl.formatMessage({
            id: 'geofence.message.loadFailed',
          })}
          action={
            <Button
              size="small"
              onClick={() => void settingsQuery.refetch()}
            >
              {intl.formatMessage({ id: 'common.retry' })}
            </Button>
          }
        />
      ) : (
        <Form
          form={form}
          layout="vertical"
          className="geofence-drawer-form"
          onValuesChange={resetPreview}
        >
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            title={intl.formatMessage({
              id: 'geofence.settings.observeNotice',
            })}
          />

          <Form.Item name="systemMode" hidden>
            <input />
          </Form.Item>

          <Alert
            type="success"
            showIcon
            style={{ marginBottom: 16 }}
            title={intl.formatMessage({
              id: 'geofence.settings.masterEnabled',
            })}
            description={intl.formatMessage({
              id: 'geofence.settings.systemPlacementHint',
            })}
          />

          {settingsQuery.data.carriers.map((setting, index) => {
            const carrierSetting =
              carriers[index] ?? setting;
            const carrierName = intl.formatMessage({
              id: `geofence.carrier.${carrierSetting.carrier}`,
            });
            const carrierMode =
              carrierSetting.mode ?? 'off';
            const effectiveMode = effectiveGeofenceMode(
              systemMode,
              carrierMode,
            );
            return (
              <div
                key={carrierSetting.carrier}
                className="geofence-settings-carrier"
              >
                <Space
                  className="geofence-settings-carrier-header"
                >
                  <strong>{carrierName}</strong>
                  <Tag color={effectiveMode === 'observe' ? 'blue' : 'default'}>
                    {intl.formatMessage({
                      id: 'geofence.settings.effectiveModeValue',
                    }, {
                      mode: intl.formatMessage({
                        id: `geofence.mode.${effectiveMode}`,
                      }),
                    })}
                  </Tag>
                </Space>
                <Form.Item
                  name={['carriers', index, 'carrier']}
                  hidden
                >
                  <input />
                </Form.Item>
                <Form.Item
                  name={['carriers', index, 'mode']}
                  label={intl.formatMessage({
                    id: 'geofence.settings.carrierMode',
                  })}
                  rules={[{ required: true }]}
                >
                  <Select options={modeOptions} />
                </Form.Item>
                <Form.Item
                  name={[
                    'carriers',
                    index,
                    'defaultBaselineRadiusMeters',
                  ]}
                  label={intl.formatMessage({
                    id: 'geofence.settings.defaultRadius',
                  })}
                  rules={[
                    {
                      required: true,
                      type: 'number',
                      min: 0.000001,
                      max: 50000,
                    },
                  ]}
                >
                  <InputNumber
                    min={0.000001}
                    max={50000}
                    style={{ width: '100%' }}
                    addonAfter="m"
                  />
                </Form.Item>
              </div>
            );
          })}

          {preview && (
            <Descriptions
              bordered
              size="small"
              column={1}
              title={intl.formatMessage({
                id: 'geofence.settings.previewResult',
              })}
              items={[
                {
                  key: 'enabled',
                  label: intl.formatMessage({
                    id: 'geofence.settings.enabledGeofences',
                  }),
                  children: preview.enabledGeofences,
                },
                {
                  key: 'bindings',
                  label: intl.formatMessage({
                    id: 'geofence.settings.activeBindings',
                  }),
                  children: preview.activeBindings,
                },
                {
                  key: 'observed',
                  label: intl.formatMessage({
                    id: 'geofence.settings.newlyObservedDevices',
                  }),
                  children: preview.newlyObservedDevices,
                },
              ]}
            />
          )}
        </Form>
      )}
    </Drawer>
  );
}
