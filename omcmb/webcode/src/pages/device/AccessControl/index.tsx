import { useMemo, useState } from 'react';
import { Select, Tabs } from 'antd';
import { useSearchParams } from 'react-router-dom';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import { usePermission } from '@core/hooks/usePermission';
import { useDeviceAccessRuntimeSettings } from '@core/hooks/api/useDeviceAccess';
import AccessStatesPanel from './AccessStatesPanel';
import AccessActionsPanel from './AccessActionsPanel';
import { AccessListPanel, CandidatePanel, PolicyPanel } from './GovernancePanels';
import RuntimeSwitch from './RuntimeSwitch';
import styles from './AccessControl.module.css';
import { parseAccessControlDeepLink, updateAccessControlSearch } from './deepLink';

const PERM_MANAGE_LIST = 'device:access-control:manage-list';
const PERM_REVIEW = 'device:access-control:review';
const PERM_ACTION = 'device:access-control:action';
const PERM_PUBLISH = 'device:access-control:publish';
const PERM_SETTINGS = 'device:access-control:settings';
const PERM_AUDIT = 'device:access-control:audit';

export default function AccessControl() {
  const t = useT();
  const [searchParams, setSearchParams] = useSearchParams();
  const deepLink = useMemo(() => parseAccessControlDeepLink(searchParams), [searchParams]);
  const operatorCode = deepLink.operator;
  const activeTab = deepLink.tab;
  const [ruleDrilldown, setRuleDrilldown] = useState<{ policyVersionId: string; matchedRuleId: string }>();
  const [candidateSerialNumber, setCandidateSerialNumber] = useState<string>();
  const canManageList = usePermission(PERM_MANAGE_LIST);
  const canReview = usePermission(PERM_REVIEW);
  const canAction = usePermission(PERM_ACTION);
  const canPublish = usePermission(PERM_PUBLISH);
  const canConfigureSettings = usePermission(PERM_SETTINGS);
  const canArchiveAudit = usePermission(PERM_AUDIT);
  const runtimeSettings = useDeviceAccessRuntimeSettings(operatorCode);
  const operators = ['cmcc', 'ctcc', 'cucc'].map((value) => ({ value, label: t(`deviceAccess.operator.${value}`) }));
  const updateSearch = (patch: Parameters<typeof updateAccessControlSearch>[1]) => {
    setSearchParams(updateAccessControlSearch(searchParams, patch), { replace: true });
  };

  return (
    <ListPageLayout
      title={t('nav.device.accessControl')}
      subtitle={t('deviceAccess.subtitle')}
      extra={<Select value={operatorCode} options={operators} onChange={(operator) => updateSearch({ operator, candidateId: undefined })} style={{ width: 210 }} aria-label={t('deviceAccess.operator')} />}
    >
      <RuntimeSwitch operatorCode={operatorCode} allowed={canConfigureSettings} t={t} />
      <Tabs className={styles.pageTabs} activeKey={activeTab} onChange={(tab) => updateSearch({ tab, ...(tab === 'candidates' ? {} : { candidateId: undefined }) })} destroyOnHidden items={[
        { key: 'states', label: t('deviceAccess.tabs.states'), children: <AccessStatesPanel operatorCode={operatorCode} t={t} canReevaluate={canPublish} canManageList={canManageList} canArchive={canArchiveAudit} businessEnabled={runtimeSettings.data?.enabled ?? false} ruleDrilldown={ruleDrilldown} onReviewCandidate={(serialNumber) => { setCandidateSerialNumber(serialNumber); updateSearch({ tab: 'candidates', candidateId: undefined }); }} /> },
        { key: 'policies', label: t('deviceAccess.tabs.policies'), children: <PolicyPanel operatorCode={operatorCode} t={t} allowed={canPublish} onDrilldownRule={(policyVersionId, matchedRuleId) => { setRuleDrilldown({ policyVersionId, matchedRuleId }); updateSearch({ tab: 'states' }); }} /> },
        { key: 'lists', label: t('deviceAccess.tabs.lists'), children: <AccessListPanel operatorCode={operatorCode} t={t} allowed={canManageList} businessEnabled={runtimeSettings.data?.enabled ?? false} /> },
        { key: 'candidates', label: t('deviceAccess.tabs.candidates'), children: <CandidatePanel key={`${operatorCode}:${deepLink.reviewStatus}:${deepLink.candidateId ?? ''}:${candidateSerialNumber ?? ''}`} operatorCode={operatorCode} t={t} allowed={canReview} businessEnabled={runtimeSettings.data?.enabled ?? false} initialSerialNumber={candidateSerialNumber} candidateId={deepLink.candidateId} reviewStatus={deepLink.reviewStatus} onReviewStatusChange={(reviewStatus) => updateSearch({ reviewStatus, candidateId: undefined })} onCandidateHandled={() => updateSearch({ candidateId: undefined })} /> },
        { key: 'actions', label: t('deviceAccess.tabs.actions'), children: <AccessActionsPanel operatorCode={operatorCode} t={t} allowed={canAction} /> },
      ]} />
    </ListPageLayout>
  );
}
