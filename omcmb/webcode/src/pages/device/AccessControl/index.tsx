import { useState } from 'react';
import { Select, Tabs } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import { usePermission } from '@core/hooks/usePermission';
import { useDeviceAccessRuntimeSettings } from '@core/hooks/api/useDeviceAccess';
import AccessStatesPanel from './AccessStatesPanel';
import AccessActionsPanel from './AccessActionsPanel';
import { AccessListPanel, CandidatePanel, PolicyPanel } from './GovernancePanels';
import RuntimeSwitch from './RuntimeSwitch';

const PERM_MANAGE_LIST = 'device:access-control:manage-list';
const PERM_REVIEW = 'device:access-control:review';
const PERM_ACTION = 'device:access-control:action';
const PERM_PUBLISH = 'device:access-control:publish';
const PERM_SETTINGS = 'device:access-control:settings';

export default function AccessControl() {
  const t = useT();
  const [operatorCode, setOperatorCode] = useState('cmcc');
  const canManageList = usePermission(PERM_MANAGE_LIST);
  const canReview = usePermission(PERM_REVIEW);
  const canAction = usePermission(PERM_ACTION);
  const canPublish = usePermission(PERM_PUBLISH);
  const canConfigureSettings = usePermission(PERM_SETTINGS);
  const runtimeSettings = useDeviceAccessRuntimeSettings(operatorCode);
  const operators = ['cmcc', 'ctcc', 'cucc'].map((value) => ({ value, label: t(`deviceAccess.operator.${value}`) }));

  return (
    <ListPageLayout
      title={t('nav.device.accessControl')}
      subtitle={t('deviceAccess.subtitle')}
      extra={<Select value={operatorCode} options={operators} onChange={setOperatorCode} style={{ width: 210 }} aria-label={t('deviceAccess.operator')} />}
    >
      <RuntimeSwitch operatorCode={operatorCode} allowed={canConfigureSettings} t={t} />
      <Tabs destroyOnHidden items={[
        { key: 'states', label: t('deviceAccess.tabs.states'), children: <AccessStatesPanel operatorCode={operatorCode} t={t} canReevaluate={canPublish} businessEnabled={runtimeSettings.data?.enabled ?? false} /> },
        { key: 'policies', label: t('deviceAccess.tabs.policies'), children: <PolicyPanel operatorCode={operatorCode} t={t} allowed={canPublish} /> },
        { key: 'lists', label: t('deviceAccess.tabs.lists'), children: <AccessListPanel operatorCode={operatorCode} t={t} allowed={canManageList} /> },
        { key: 'candidates', label: t('deviceAccess.tabs.candidates'), children: <CandidatePanel operatorCode={operatorCode} t={t} allowed={canReview} /> },
        { key: 'actions', label: t('deviceAccess.tabs.actions'), children: <AccessActionsPanel operatorCode={operatorCode} t={t} allowed={canAction} /> },
      ]} />
    </ListPageLayout>
  );
}
