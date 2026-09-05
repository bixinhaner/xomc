import { describe, expect, it } from 'vitest';
import type { Assistant, AssistantRun } from '@core/types/assistant';
import { assistantEnUS, assistantZhCN } from '@core/i18n/assistantMessages';
import { canPublish, errorText, isRunning, trialFor } from './helpers';
const a = { revision: 2, readiness: 'ready', definition: { name: 'Independent assistant' }, publishedRevision: 1 } as Assistant;
const run = { id: 'current', revision: 2, kind: 'trial', status: 'COMPLETED', output: { outcome: 'finding' } } as AssistantRun;
describe('assistant publish gates and UI copy', () => {
  it('requires a completed real trial for the current revision', () => { expect(canPublish(a, [])).toBe(false); expect(canPublish(a, [{ ...run, revision: 1 }])).toBe(false); expect(canPublish(a, [run])).toBe(true); });
  it('blocks a newer failed trial even after an earlier success', () => { const failed = { ...run, id: 'latest', status: 'FAILED' as const }; expect(trialFor(a, [failed, run])?.id).toBe('latest'); expect(canPublish(a, [failed, run])).toBe(false); });
  it('does not treat missing evidence as no anomaly', () => expect(canPublish(a, [{ ...run, output: { ...run.output!, outcome: 'insufficient_data' } }])).toBe(false));
  it('does not publish while cancelling or repeat an existing version', () => { expect(canPublish(a, [run, { ...run, status: 'CANCELLING' }])).toBe(false); expect(canPublish({ ...a, publishedRevision: 2 }, [run])).toBe(false); });
  it('explains conflicts without leaking raw upstream errors', () => { expect(errorText(new Error('HTTP 409 ASSISTANT_REVISION_CONFLICT'), (key) => key)).toBe('assistant.error.ASSISTANT_REVISION_CONFLICT'); expect(errorText('postgres secret: unknown', (key) => key)).toBe('assistant.error.generic'); });
  it('has complete English/Chinese keys, no silent language gaps', () => { expect(Object.keys(assistantEnUS).sort()).toEqual(Object.keys(assistantZhCN).sort()); });
  it('tracks all in-flight states', () => { for (const status of ['QUEUED', 'RUNNING', 'CANCELLING'] as const) expect(isRunning({ ...run, status })).toBe(true); expect(isRunning(run)).toBe(false); });
});
