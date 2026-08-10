import { useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Descriptions,
  Space,
  Spin,
  Switch,
  message as staticMessage,
} from 'antd';
import { SafetyCertificateOutlined } from '@ant-design/icons';
import { useIntl } from 'react-intl';
import {
  useGeofenceSettings,
  usePreviewGeofenceSettings,
  useUpdateGeofenceSettings,
} from '@core/hooks/api/useGeofence';
import type {
  GeofenceSettingsPreview,
  UpdateGeofenceSettingsInput,
} from '@core/types/geofence';
import { geofenceSettingsToInput } from '@core/utils/geofenceSettings';

interface PendingSystemToggle {
  enabled: boolean;
  input: UpdateGeofenceSettingsInput;
  preview: GeofenceSettingsPreview;
}

export default function GeofenceSystemSettings() {
  const intl = useIntl();
  const appContext = App.useApp();
  const message = typeof (appContext.message as { success?: unknown }).success === 'function'
    ? appContext.message
    : staticMessage;
  const settingsQuery = useGeofenceSettings();
  const previewMutation = usePreviewGeofenceSettings();
  const updateMutation = useUpdateGeofenceSettings();
  const [pending, setPending] = useState<PendingSystemToggle>();
  const enabled = settingsQuery.data?.systemMode !== 'off';

  const previewToggle = async (nextEnabled: boolean) => {
    if (!settingsQuery.data) return;
    const input = {
      ...geofenceSettingsToInput(settingsQuery.data),
      systemMode: nextEnabled ? 'enforce' : 'off',
    } satisfies UpdateGeofenceSettingsInput;
    try {
      const preview = await previewMutation.mutateAsync(input);
      setPending({ enabled: nextEnabled, input, preview });
    } catch (error) {
      void message.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'geofence.message.operationFailed' }),
      );
    }
  };

  const saveToggle = async () => {
    if (!pending) return;
    try {
      await updateMutation.mutateAsync(pending.input);
      setPending(undefined);
      void message.success(
        intl.formatMessage({ id: 'geofence.settings.masterUpdated' }),
      );
    } catch (error) {
      void message.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({ id: 'geofence.message.operationFailed' }),
      );
    }
  };

  if (settingsQuery.isLoading) {
    return <Spin />;
  }
  if (settingsQuery.isError || !settingsQuery.data) {
    return (
      <Alert
        type="error"
        showIcon
        title={intl.formatMessage({ id: 'geofence.message.loadFailed' })}
        action={(
          <Button size="small" onClick={() => void settingsQuery.refetch()}>
            {intl.formatMessage({ id: 'common.retry' })}
          </Button>
        )}
      />
    );
  }

  return (
    <Card
      title={(
        <Space>
          <SafetyCertificateOutlined />
          {intl.formatMessage({ id: 'geofence.settings.masterSwitch' })}
        </Space>
      )}
      extra={(
        <Switch
          aria-label={intl.formatMessage({ id: 'geofence.settings.masterSwitch' })}
          checked={enabled}
          loading={previewMutation.isPending || updateMutation.isPending}
          onChange={(checked) => void previewToggle(checked)}
        />
      )}
    >
      <Alert
        type={enabled ? 'info' : 'warning'}
        showIcon
        title={intl.formatMessage({
          id: enabled
            ? 'geofence.settings.masterEnabled'
            : 'geofence.settings.masterDisabled',
        })}
        description={intl.formatMessage({ id: 'geofence.settings.systemPlacementHint' })}
      />

      {pending && (
        <Card size="small" style={{ marginTop: 16 }}>
          <Alert
            type={pending.enabled ? 'info' : 'warning'}
            showIcon
            title={intl.formatMessage({
              id: pending.enabled
                ? 'geofence.settings.masterEnableNotice'
                : 'geofence.settings.masterDisableNotice',
            })}
          />
          <Descriptions
            size="small"
            column={3}
            style={{ marginTop: 16 }}
            items={[
              {
                key: 'fences',
                label: intl.formatMessage({ id: 'geofence.settings.enabledGeofences' }),
                children: pending.preview.enabledGeofences,
              },
              {
                key: 'bindings',
                label: intl.formatMessage({ id: 'geofence.settings.activeBindings' }),
                children: pending.preview.activeBindings,
              },
              {
                key: 'observed',
                label: intl.formatMessage({ id: 'geofence.settings.newlyObservedDevices' }),
                children: pending.preview.newlyObservedDevices,
              },
            ]}
          />
          <Space style={{ marginTop: 16 }}>
            <Button onClick={() => setPending(undefined)}>
              {intl.formatMessage({ id: 'common.cancel' })}
            </Button>
            <Button
              type="primary"
              loading={updateMutation.isPending}
              onClick={() => void saveToggle()}
            >
              {intl.formatMessage({ id: 'common.confirm' })}
            </Button>
          </Space>
        </Card>
      )}
    </Card>
  );
}
