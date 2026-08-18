import { App, Alert, Space, Switch, Tag, Tooltip } from 'antd';
import { useDeviceAccessRuntimeSettings, useUpdateDeviceAccessRuntimeSettings } from '@core/hooks/api/useDeviceAccess';
import { QueryError } from './shared';

export default function RuntimeSwitch({
  operatorCode,
  allowed,
  t,
}: {
  operatorCode: string;
  allowed: boolean;
  t: (key: string) => string;
}) {
  const { message, modal } = App.useApp();
  const query = useDeviceAccessRuntimeSettings(operatorCode);
  const update = useUpdateDeviceAccessRuntimeSettings();
  const settings = query.data;

  const confirmUpdate = (enabled: boolean) => {
    modal.confirm({
      title: t(enabled ? 'deviceAccess.enableConfirmTitle' : 'deviceAccess.disableConfirmTitle'),
      content: t(enabled ? 'deviceAccess.enableConfirmDescription' : 'deviceAccess.disableConfirmDescription'),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: !enabled },
      onOk: async () => {
        await update.mutateAsync({ operatorCode, enabled });
        void message.success(t(enabled ? 'deviceAccess.enabledSuccess' : 'deviceAccess.disabledSuccess'));
      },
    });
  };

  return (
    <>
      <QueryError error={query.error} t={t} />
      <Alert
        showIcon
        type={settings?.enabled ? 'warning' : 'info'}
        title={(
          <Space wrap>
            <span>{t('deviceAccess.businessSwitch')}</span>
            <Tag color={settings?.enabled ? 'green' : 'default'}>
              {t(settings?.enabled ? 'deviceAccess.switchEnabled' : 'deviceAccess.switchDisabled')}
            </Tag>
            <Tooltip title={allowed ? undefined : t('common.noPermission')}>
              <span>
                <Switch
                  aria-label={t('deviceAccess.businessSwitch')}
                  checked={settings?.enabled ?? false}
                  loading={query.isLoading || update.isPending}
                  disabled={!allowed || !settings}
                  onChange={confirmUpdate}
                />
              </span>
            </Tooltip>
          </Space>
        )}
        description={settings ? t(settings.enabled ? 'deviceAccess.enabledDescription' : 'deviceAccess.disabledDescription') : t('common.loading')}
        style={{ marginBottom: 16 }}
      />
    </>
  );
}
