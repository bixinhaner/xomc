import { readdirSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createIntl, createIntlCache } from 'react-intl';

import { enUS, zhCN } from '@core/i18n';

const screenshotMessages = {
  'deviceAccess.allDecisions': ['全部决定结果', 'All decision results'],
  'deviceAccess.dimension': ['规则维度', 'Rule Dimension'],
  'deviceAccess.policyVersionId': ['策略版本 ID', 'Policy Version ID'],
  'deviceAccess.matchedRuleId': ['命中规则 ID', 'Matched Rule ID'],
  'deviceAccess.summary.total': ['筛选总数', 'Filtered Total'],
  'deviceAccess.summary.accepted': ['接受', 'Accepted'],
  'deviceAccess.summary.rejected': ['拒绝', 'Rejected'],
  'deviceAccess.summary.reviewRequired': ['未知设备待审核', 'Unknown Devices Pending Review'],
  'deviceAccess.summary.revoked': ['吊销', 'Revoked'],
  'deviceAccess.addToAllowlist': ['加入白名单', 'Add to Allowlist'],
  'deviceAccess.addToDenylist': ['加入黑名单', 'Add to Denylist'],
  'deviceAccess.nextAttemptAt': ['下次重试', 'Next retry'],
  'deviceAccess.lastFailureCode': ['失败码', 'Failure code'],
  'deviceAccess.policyDifference': ['版本差异', 'Version Difference'],
  'deviceAccess.priority': ['优先级（数值越小越先）', 'Priority (lower runs first)'],
  'deviceAccess.conditionAdvancedSettings': ['高级设置（证据采集与缺失处理）', 'Advanced settings (evidence collection and missing data)'],
} as const;

const dynamicMessageValues = {
  actionStatus: ['pending_dispatch', 'dispatching', 'verifying', 'retry_wait', 'succeeded', 'failed', 'dead', 'cancelled'],
  actionType: ['rf_off', 'rf_on'],
  attemptPhase: ['baseline_gpv', 'spv', 'readback_gpv'],
  attemptStatus: ['queued', 'sent', 'succeeded', 'failed', 'timeout', 'cancelled'],
  conditionType: ['tac', 'ecgi', 'observed_ip', 'gps'],
  decision: ['accept', 'reject', 'review', 'revoke', 'bypass'],
  defaultAction: ['reject', 'review'],
  dimension: ['sn', 'tac', 'ecgi', 'ip', 'gps'],
  direction: ['contain', 'release'],
  failureMode: ['fail_closed', 'review_hold'],
  failurePolicy: ['strict', 'valid_only'],
  importMode: ['append', 'replace'],
  importRowStatus: ['valid', 'invalid', 'duplicate', 'no_change'],
  importStatus: ['uploaded', 'validated', 'committing', 'committed', 'failed', 'rolled_back'],
  listStatus: ['active', 'disabled'],
  listType: ['allow', 'deny', 'revoked'],
  notificationStatus: ['pending', 'sent', 'failed', 'dead_letter', 'not_configured'],
  operator: ['cmcc', 'ctcc', 'cucc', 'equal', 'in', 'cidr', 'ip_range', 'within_radius', 'within_bounds'],
  policyStatus: ['draft', 'published', 'retired'],
  review: ['allow', 'deny'],
  reviewStatus: ['pending', 'approved', 'rejected', 'expired'],
  reason: [
    'access_control_disabled', 'authentication_failed', 'asset_retired', 'revoked_list_matched',
    'denylist_matched', 'identity_mismatch', 'identity_unverified', 'device_code_missing',
    'cloud_key_missing', 'ownership_mismatch', 'ownership_unverified', 'allowlist_matched',
    'bypass_profile_matched', 'rule_matched', 'no_applicable_rule', 'evidence_missing',
    'evidence_stale', 'evidence_mismatch', 'evidence_collection_failed', 'evidence_system_error',
    'rule_mismatch', 'rule_mismatch_pending_confirmation', 'rule_mismatch_confirmed',
  ],
  serialScope: ['all', 'list', 'prefix', 'range'],
  state: ['review_required', 'collecting', 'accepted', 'rejected', 'revalidating', 'revoked'],
} as const;

function productionSourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) return productionSourceFiles(path);
    if (!/\.(ts|tsx)$/.test(entry.name) || /\.(test|spec)\.(ts|tsx)$/.test(entry.name)) return [];
    return [path];
  });
}

function staticMessageKeys(): string[] {
  const sourceDirectory = dirname(fileURLToPath(import.meta.url));
  const keys = new Set<string>();
  for (const path of productionSourceFiles(sourceDirectory)) {
    const matches = readFileSync(path, 'utf8').match(/deviceAccess\.[A-Za-z0-9_.]+/g) ?? [];
    for (const key of matches) {
      if (!key.endsWith('.')) keys.add(key);
    }
  }
  return [...keys].sort();
}

describe('access-control locale resources', () => {
  it('keeps the complete Chinese and English key sets aligned and non-empty', () => {
    const zhKeys = Object.keys(zhCN).filter((key) => key.startsWith('deviceAccess.')).sort();
    const enKeys = Object.keys(enUS).filter((key) => key.startsWith('deviceAccess.')).sort();

    expect(zhKeys).toEqual(enKeys);
    expect(zhKeys.length).toBeGreaterThan(300);
    for (const key of zhKeys) {
      expect(zhCN[key]?.trim(), `empty zh-CN message: ${key}`).toBeTruthy();
      expect(enUS[key]?.trim(), `empty en-US message: ${key}`).toBeTruthy();
      expect(zhCN[key], `raw zh-CN key: ${key}`).not.toBe(key);
      expect(enUS[key], `raw en-US key: ${key}`).not.toBe(key);
    }
  });

  it('covers every static key referenced by the production access-control page', () => {
    for (const key of staticMessageKeys()) {
      expect(zhCN, `missing zh-CN message: ${key}`).toHaveProperty(key);
      expect(enUS, `missing en-US message: ${key}`).toHaveProperty(key);
    }
  });

  it.each([
    ['zh-CN', zhCN],
    ['en-US', enUS],
  ] as const)('formats every access-control key through the runtime provider for %s', (locale, messages) => {
    const intl = createIntl({ locale, defaultLocale: 'zh-CN', messages }, createIntlCache());
    const keys = new Set([
      ...staticMessageKeys(),
      ...Object.entries(dynamicMessageValues).flatMap(([prefix, values]) => values.map((value) => `deviceAccess.${prefix}.${value}`)),
    ]);

    for (const key of keys) {
      const template = messages[key] ?? '';
      const values = Object.fromEntries([...template.matchAll(/\{\s*([A-Za-z0-9_]+)(?:\s*,|\s*})/g)].map((match) => [match[1], 1]));
      expect(intl.formatMessage({ id: key }, values), `runtime returned raw key for ${locale}: ${key}`).not.toBe(key);
    }
  });

  it('covers every value rendered through a dynamic access-control key', () => {
    for (const [prefix, values] of Object.entries(dynamicMessageValues)) {
      for (const value of values) {
        const key = `deviceAccess.${prefix}.${value}`;
        expect(zhCN, `missing zh-CN dynamic message: ${key}`).toHaveProperty(key);
        expect(enUS, `missing en-US dynamic message: ${key}`).toHaveProperty(key);
      }
    }
  });

  it.each(Object.entries(screenshotMessages))(
    'provides the expected Chinese and English messages for %s',
    (key, [expectedZhCN, expectedEnUS]) => {
      expect(zhCN[key]).toBe(expectedZhCN);
      expect(enUS[key]).toBe(expectedEnUS);
    },
  );
});
