// T-0164-P2 / G2 PM 保留策略管理界面（系统配置页 — PM 保留策略卡片）。
//
// 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.2 + §4.6
// 后端：internal/pm/retention（sys_configs category='pm.retention' 5 键）
// 接入方式：复用现有 useSysConfigsByCategory + useBatchUpdateSysConfigs
// （sys_configs.changed 通过 admin.SysConfigService.RegisterSavedHook 触发后端 retention.Service.Reload）
//
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Card, Form, message, Space, Spin } from 'antd';
import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
  useSysConfigApplyBatch,
} from '@core/hooks/api/useSystem';
import { AddonInputNumber } from '@/components/common/InputAddon';
import { useT } from '@/hooks/useT';
import type { ConfigApplyBatch } from '@core/types/system';

// 与后端 internal/pm/retention/policies.go 的 PolicyKey 严格对齐。
type PolicyKey =
  | 'raw_15min_days'
  | 'hourly_days'
  | 'daily_days'
  | 'weekly_days'
  | 'monthly_days';

const PM_RETENTION_CATEGORY = 'pm.retention';

// 默认值与后端 DefaultDays（金字塔保留）对齐。
const DEFAULTS: Record<PolicyKey, number> = {
  raw_15min_days: 30,
  hourly_days: 180,
  daily_days: 730,
  weekly_days: 730,
  monthly_days: 1825,
};

const KEYS: PolicyKey[] = [
  'raw_15min_days',
  'hourly_days',
  'daily_days',
  'weekly_days',
  'monthly_days',
];

const MIN_DAYS = 1;
const MAX_DAYS = 3650;

export default function PmRetentionSection() {
  const t = useT();
  const [form] = Form.useForm<Record<PolicyKey, number>>();
  const [submitting, setSubmitting] = useState(false);
  const [submittedBatch, setSubmittedBatch] = useState<ConfigApplyBatch | null>(null);
  const { data: refreshedBatch } = useSysConfigApplyBatch(submittedBatch?.id);
  const applyBatch = refreshedBatch ?? submittedBatch;
  const applyFailureDetail = applyBatch?.status === 'failed'
    ? applyBatch.targets.find((target) => target.lastError)?.lastError?.trim()
    : undefined;

  const {
    data: configs,
    isLoading,
    isFetching,
    isError,
    isSuccess,
    refetch,
  } = useSysConfigsByCategory(PM_RETENTION_CATEGORY);
  const { mutateAsync: batchUpdate } = useBatchUpdateSysConfigs();
  const canEdit = isSuccess && !isFetching;
  const canEditRef = useRef(canEdit);
  useLayoutEffect(() => {
    canEditRef.current = canEdit;
  }, [canEdit]);

  // 把后端返回的 sys_configs 行映射为 { key: int }
  const initialValues = useMemo<Record<PolicyKey, number>>(() => {
    const out: Record<PolicyKey, number> = { ...DEFAULTS };
    if (!configs) {
      return out;
    }
    for (const cfg of configs) {
      if ((KEYS as string[]).includes(cfg.key)) {
        const n = Number(cfg.value);
        if (Number.isFinite(n) && n >= MIN_DAYS && n <= MAX_DAYS) {
          out[cfg.key as PolicyKey] = n;
        }
      }
    }
    return out;
  }, [configs]);

  useEffect(() => {
    if (!isSuccess) return;
    form.setFields(
      KEYS.map((name) => ({ name, value: initialValues[name], touched: false })),
    );
  }, [form, initialValues, isSuccess]);

  const handleSave = async () => {
    if (!canEditRef.current) {
      message.error(t('empty.loadFailed'));
      return;
    }
    try {
      const values = await form.validateFields();
      if (!canEditRef.current) {
        message.error(t('empty.loadFailed'));
        return;
      }
      const changedKeys = KEYS.filter((key) => form.isFieldTouched(key));
      if (changedKeys.length === 0) {
        message.warning(t('sysconfig.warn.noSaveable'));
        return;
      }
      setSubmitting(true);
      const result = await batchUpdate({
        category: PM_RETENTION_CATEGORY,
        items: changedKeys.map((k) => ({
          key: k,
          value: String(values[k]),
          value_type: 'int',
        })),
      });
      setSubmittedBatch(result.batch);
      message.success(t('pmRetention.save.success'));
    } catch {
      // validateFields 失败或网络错误：交给 antd 自身的 UI 提示，不再额外弹错
    } finally {
      setSubmitting(false);
    }
  };

  const handleReset = () => {
    if (!canEdit) return;
    form.setFields(
      KEYS.map((name) => ({ name, value: DEFAULTS[name], touched: true })),
    );
  };

  return (
    <Card
      size="small"
      title={
        <span style={{ fontSize: 14, fontWeight: 600 }}>{t('pmRetention.title')}</span>
      }
      style={{ marginBottom: 16 }}
      extra={
        <Space>
          <Button size="small" onClick={handleReset} disabled={submitting || !canEdit}>
            {t('pmRetention.reset')}
          </Button>
          <Button size="small" type="primary" onClick={handleSave} loading={submitting} disabled={!canEdit}>
            {t('pmRetention.save')}
          </Button>
        </Space>
      }
    >
      <div style={{ color: '#888', fontSize: 12, marginBottom: 12 }}>
        {t('pmRetention.description')}
      </div>
      {isError && (
        <Alert
          type="error"
          showIcon
          title={t('empty.loadFailed')}
          description={t('empty.loadFailedDesc')}
          action={<Button size="small" onClick={() => void refetch()}>{t('common.retry')}</Button>}
          style={{ marginBottom: 12 }}
        />
      )}
      <Spin spinning={isLoading || isFetching}>
        <Form form={form} layout="vertical" size="small" initialValues={DEFAULTS}>
          {KEYS.map((k) => (
            <Form.Item
              key={k}
              name={k}
              label={t(`pmRetention.${k}`)}
              rules={[
                { required: true },
                {
                  type: 'integer',
                  min: MIN_DAYS,
                  message: t('pmRetention.validate.tooShort'),
                },
                {
                  type: 'integer',
                  max: MAX_DAYS,
                  message: t('pmRetention.validate.tooLong'),
                },
              ]}
              style={{ marginBottom: 12, maxWidth: 320 }}
            >
              {/* antd6 addonAfter 已废弃：改用受控 AddonInputNumber（Space.Compact +
                  InputAddon）复刻"天"后缀盒子，保留 Form.Item 受控绑定。 */}
              <AddonInputNumber
                addonAfter={t('pmRetention.unit.days')}
                compactStyle={{ width: 'auto' }}
                min={MIN_DAYS}
                max={MAX_DAYS}
                step={1}
                style={{ width: 200 }}
              />
            </Form.Item>
          ))}
        </Form>
      </Spin>
      {applyBatch && (
        <Alert
          style={{ marginTop: 12 }}
          type={applyBatch.status === 'failed' ? 'error' : applyBatch.status === 'applied' ? 'success' : 'info'}
          showIcon
          message={t(`sysconfig.apply.${applyBatch.status}`)}
          description={applyBatch.status === 'failed'
            ? (
                <div>
                  <div>{t('sysconfig.apply.failed.description')}</div>
                  {applyFailureDetail && (
                    <details style={{ marginTop: 8 }}>
                      <summary style={{ cursor: 'pointer' }}>
                        {t('sysconfig.apply.failed.detail')}
                      </summary>
                      <pre style={{ margin: '8px 0 0', whiteSpace: 'pre-wrap' }}>{applyFailureDetail}</pre>
                    </details>
                  )}
                </div>
              )
            : undefined}
        />
      )}
    </Card>
  );
}
