import { useEffect, useMemo } from 'react';
import { Alert, Button } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useStorageProtectionPolicies, useStorageProtectionTargets } from '@core/hooks/api/useStorageProtection';
import type {
  StorageProtectionPolicy,
  StorageProtectionTarget,
} from '@core/services/api/storageProtectionApi';
import { useTabStore } from '@core/store/tabStore';
import { useT } from '@/hooks/useT';
import styles from './StorageProtectionBlockingBanner.module.css';

const DETAIL_PATH = '/system/config?tab=retention_bp';

interface BlockingSummary {
  policy: StorageProtectionPolicy;
  target?: StorageProtectionTarget;
  targetLabel: string;
  usedPercent: string;
  blockPercent: string;
  recoverPercent: string;
}

interface StorageProtectionBlockingBannerProps {
  collapsed: boolean;
  onCollapsedChange: (collapsed: boolean) => void;
  onBlockedChange?: (blocked: boolean) => void;
}

function formatPercent(value: number | undefined, fallback: string) {
  if (value === undefined || !Number.isFinite(value)) return fallback;
  const percent = value <= 1 ? value * 100 : value;
  return `${percent.toFixed(1)}%`;
}

function matchTarget(policy: StorageProtectionPolicy, targets: StorageProtectionTarget[]) {
  const blockedTargets = targets.filter((target) => target.currentState === 'blocked');
  if (blockedTargets.length > 0) {
    return blockedTargets.reduce((worst, current) => (
      current.usedRatio > worst.usedRatio ? current : worst
    ));
  }
  return targets.find((target) => (
    target.targetType === policy.targetType && target.targetId === policy.targetId
  ));
}

function targetLabel(target: StorageProtectionTarget | undefined, policy: StorageProtectionPolicy, fallback: string) {
  if (!target) return policy.targetId || fallback;
  const paths = target.protectedPaths.length > 0 ? target.protectedPaths : target.sourcePaths;
  return target.mountpoint || target.mountPath || paths[0] || target.label || target.targetId || fallback;
}

function buildStorageProtectionBlockingSummary(
  policies: StorageProtectionPolicy[],
  targets: StorageProtectionTarget[],
  fallback: string,
): BlockingSummary | undefined {
  const enabledPolicies = policies.filter((item) => item.enabled);
  const hasBlockedTarget = targets.some((target) => target.currentState === 'blocked');
  const policy = enabledPolicies.find((item) => item.currentState === 'blocked')
    ?? (hasBlockedTarget ? enabledPolicies[0] : undefined);
  if (!policy) return undefined;

  const target = matchTarget(policy, targets);
  return {
    policy,
    target,
    targetLabel: targetLabel(target, policy, fallback),
    usedPercent: formatPercent(target?.usedRatio ?? policy.lastObservedRatio, fallback),
    blockPercent: formatPercent(policy.blockUsedPercent, fallback),
    recoverPercent: formatPercent(policy.recoverUsedPercent, fallback),
  };
}

export default function StorageProtectionBlockingBanner({
  collapsed,
  onCollapsedChange,
  onBlockedChange,
}: StorageProtectionBlockingBannerProps) {
  const t = useT();
  const navigate = useNavigate();
  const openTab = useTabStore((s) => s.openTab);
  const { data: policies = [] } = useStorageProtectionPolicies();
  const { data: targets = [] } = useStorageProtectionTargets();
  const unavailable = t('common.notAvailable');

  const summary = useMemo(
    () => buildStorageProtectionBlockingSummary(policies, targets, unavailable),
    [policies, targets, unavailable],
  );

  useEffect(() => {
    const blocked = summary !== undefined;
    onBlockedChange?.(blocked);
    if (!blocked) {
      onCollapsedChange(false);
    }
  }, [summary, onBlockedChange, onCollapsedChange]);

  if (!summary || collapsed) return null;

  const handleViewDetail = () => {
    openTab({
      key: 'system-config-retention-bp',
      label: 'system.config.retentionBp',
      labelRaw: false,
      path: DETAIL_PATH,
      closable: true,
    });
    void navigate(DETAIL_PATH);
  };

  return (
    <Alert
      className={styles.banner}
      type="error"
      showIcon
      icon={<ExclamationCircleOutlined />}
      message={(
        <div className={styles.content}>
          <div className={styles.message}>
            <span className={styles.title}>{t('system.storageProtection.globalBlock.title')}</span>
            <div className={styles.description}>
              {t('system.storageProtection.globalBlock.description', {
                target: summary.targetLabel,
                usedPercent: summary.usedPercent,
                blockPercent: summary.blockPercent,
                recoverPercent: summary.recoverPercent,
              })}
            </div>
          </div>
          <div className={styles.actions}>
            <Button danger type="primary" size="small" onClick={handleViewDetail}>
              {t('system.storageProtection.globalBlock.viewDetail')}
            </Button>
            <Button size="small" onClick={() => onCollapsedChange(true)}>
              {t('system.storageProtection.globalBlock.collapse')}
            </Button>
          </div>
        </div>
      )}
    />
  );
}
